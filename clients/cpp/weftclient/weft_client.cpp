#include "weftclient/weft_client.h"

#include <QDateTime>
#include <QJsonArray>
#include <QJsonValue>
#include <QSslError>
#include <QUuid>

namespace weft {

namespace {

qint64 jsonInt64(const QJsonValue &value)
{
    if (value.isDouble())
        return qint64(value.toDouble());
    return value.toVariant().toLongLong();
}

QString jsonString(const QJsonObject &object, const char *key)
{
    return object.value(QLatin1String(key)).toString();
}

int jsonInt(const QJsonObject &object, const char *key, int fallback = 0)
{
    const auto value = object.value(QLatin1String(key));
    return value.isUndefined() ? fallback : value.toInt(fallback);
}

qint64 jsonInt64(const QJsonObject &object, const char *key, qint64 fallback = 0)
{
    const auto value = object.value(QLatin1String(key));
    return value.isUndefined() ? fallback : jsonInt64(value);
}

} // namespace

Client::Client(QObject *parent)
    : QObject(parent)
    , m_socket(new QSslSocket(this))
{
    qRegisterMetaType<weft::Peer>();
    qRegisterMetaType<weft::Status>();
    qRegisterMetaType<weft::PairResult>();

    // The daemon uses a self-signed certificate; identity is established by the
    // knot fingerprint at the application layer (same policy as `weftlink` CLI).
    m_socket->setPeerVerifyMode(QSslSocket::VerifyNone);

    connect(m_socket, &QSslSocket::encrypted, this, [this] {
        m_lastError.clear();
        emit connectionStateChanged(true);
        emit logMessage(QStringLiteral("Connected to %1").arg(endpoint()));
        requestStatus();
    });
    connect(m_socket, &QSslSocket::disconnected, this, [this] {
        emit connectionStateChanged(false);
        emit logMessage(QStringLiteral("Disconnected from %1").arg(endpoint()));
    });
    connect(m_socket, &QSslSocket::readyRead, this, [this] {
        m_codec.append(m_socket->readAll());
        QString error;
        const auto messages = m_codec.takeMessages(&error);
        if (!error.isEmpty())
            setError(error);
        for (const auto &message : messages)
            handleMessage(message);
    });
    connect(m_socket, &QSslSocket::errorOccurred, this,
            [this](QAbstractSocket::SocketError) {
                setError(m_socket->errorString());
                emit connectionStateChanged(false);
            });
    connect(m_socket, &QSslSocket::sslErrors, this,
            [this](const QList<QSslError> &errors) {
                for (const auto &error : errors)
                    emit logMessage(QStringLiteral("TLS: %1").arg(error.errorString()));
                m_socket->ignoreSslErrors();
            });
}

Client::~Client()
{
    if (m_socket)
        m_socket->abort();
}

void Client::connectToDaemon(const QString &host, quint16 port)
{
    if (!QSslSocket::supportsSsl()) {
        setError(QStringLiteral(
            "TLS backend unavailable (Qt/OpenSSL DLLs not found)"));
        return;
    }
    if (host.trimmed().isEmpty()) {
        setError(QStringLiteral("daemon host is empty"));
        return;
    }

    m_host = host.trimmed();
    m_port = port;
    m_codec.reset();
    m_pingTimers.clear();
    m_lastError.clear();
    if (m_socket->state() != QAbstractSocket::UnconnectedState)
        m_socket->abort();

    emit logMessage(QStringLiteral("Connecting to %1 ...").arg(endpoint()));
    m_socket->connectToHostEncrypted(m_host, m_port);
}

void Client::disconnectFromDaemon()
{
    if (m_socket->state() != QAbstractSocket::UnconnectedState) {
        emit logMessage(QStringLiteral("Disconnecting ..."));
        m_socket->disconnectFromHost();
        if (m_socket->state() != QAbstractSocket::UnconnectedState)
            m_socket->abort();
    }
}

bool Client::isConnected() const
{
    return m_socket->state() == QAbstractSocket::ConnectedState && m_socket->isEncrypted();
}

QString Client::endpoint() const
{
    return QStringLiteral("%1:%2").arg(m_host).arg(m_port);
}

void Client::requestStatus()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("status")}});
}

void Client::requestPeers()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("peers")}});
}

void Client::ping()
{
    if (!isConnected())
        return;
    const auto nonce = QUuid::createUuid().toString(QUuid::WithoutBraces);
    m_pingTimers.insert(nonce, QElapsedTimer());
    m_pingTimers[nonce].start();
    sendRequest({{QStringLiteral("type"), QStringLiteral("ping")},
                 {QStringLiteral("nonce"), nonce},
                 {QStringLiteral("ts"), double(QDateTime::currentMSecsSinceEpoch())}});
}

void Client::requestPairCode()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("knot_init")}});
}

void Client::joinPair(const QString &code, const QString &peerHost)
{
    const auto normalized = code.trimmed().toUpper();
    if (normalized.size() != 6) {
        setError(QStringLiteral("pair code must be 6 characters (A-Z, 0-9)"));
        return;
    }
    sendRequest({{QStringLiteral("type"), QStringLiteral("knot_join")},
                 {QStringLiteral("code"), normalized},
                 {QStringLiteral("host"), peerHost.trimmed()}});
}

void Client::sendRequest(const QJsonObject &request)
{
    if (!isConnected()) {
        setError(QStringLiteral("not connected to a daemon"));
        return;
    }
    QString error;
    const auto frame = FrameCodec::encode(request, &error);
    if (frame.isEmpty()) {
        setError(error.isEmpty() ? QStringLiteral("cannot encode request") : error);
        return;
    }
    if (m_socket->write(frame) < 0)
        setError(m_socket->errorString());
    m_socket->flush();
}

void Client::handleMessage(const QJsonObject &message)
{
    const auto type = jsonString(message, "type");
    if (type == QLatin1String("status")) {
        emit statusReceived(parseStatus(message));
    } else if (type == QLatin1String("peers")) {
        QList<Peer> peers;
        const auto array = message.value(QStringLiteral("peers")).toArray();
        for (const auto &item : array) {
            if (item.isObject())
                peers.append(parsePeer(item.toObject()));
        }
        emit peersReceived(peers);
    } else if (type == QLatin1String("pong")) {
        const auto nonce = jsonString(message, "nonce");
        auto it = m_pingTimers.find(nonce);
        if (it != m_pingTimers.end()) {
            emit pongReceived(it->elapsed());
            m_pingTimers.erase(it);
        } else {
            emit pongReceived(-1);
        }
    } else if (type == QLatin1String("knot_code")) {
        emit pairCodeReceived(jsonString(message, "code"),
                              jsonInt(message, "expires_in_seconds", 120));
    } else if (type == QLatin1String("knot_result")) {
        PairResult result;
        result.ok = message.value(QStringLiteral("ok")).toBool();
        result.error = jsonString(message, "error");
        if (message.value(QStringLiteral("peer")).isObject())
            result.peer = parsePeer(message.value(QStringLiteral("peer")).toObject());
        emit pairResultReceived(result);
        if (result.ok) {
            emit logMessage(QStringLiteral("Paired with %1 (%2)")
                                .arg(result.peer.displayName, result.peer.peerId));
        } else {
            emit logMessage(QStringLiteral("Pairing failed: %1").arg(result.error));
        }
        requestStatus();
    } else if (type == QLatin1String("hello_ack")) {
        emit logMessage(QStringLiteral("Daemon %1 (%2)")
                            .arg(jsonString(message, "node_id"),
                                 jsonString(message, "version")));
    } else if (type == QLatin1String("pair_confirm")) {
        emit logMessage(QStringLiteral("Pairing confirmed by peer"));
    } else if (type == QLatin1String("error")) {
        setError(jsonString(message, "message"));
    } else {
        emit logMessage(QStringLiteral("Unhandled message type: %1").arg(type));
    }
}

void Client::setError(const QString &message)
{
    if (message.isEmpty())
        return;
    m_lastError = message;
    emit errorOccurred(message);
    emit logMessage(QStringLiteral("ERROR: %1").arg(message));
}

Peer Client::parsePeer(const QJsonObject &object)
{
    Peer peer;
    peer.peerId = jsonString(object, "peer_id");
    peer.displayName = jsonString(object, "display_name");
    peer.role = jsonString(object, "role");
    peer.version = jsonString(object, "version");
    peer.address = jsonString(object, "address");
    peer.fingerprint = jsonString(object, "fingerprint");
    peer.lastSeen = jsonInt64(object, "last_seen");
    peer.paired = object.value(QStringLiteral("paired")).toBool();
    return peer;
}

Status Client::parseStatus(const QJsonObject &object)
{
    Status status;
    status.nodeId = jsonString(object, "node_id");
    status.displayName = jsonString(object, "display_name");
    status.role = jsonString(object, "role");
    status.version = jsonString(object, "version");
    status.fingerprint = jsonString(object, "fingerprint");
    status.discoveryPort = jsonInt(object, "discovery_port");
    status.discoveryActive = object.value(QStringLiteral("discovery_active")).toBool();
    status.beaconsSent = jsonInt64(object, "beacons_sent");
    status.beaconsReceived = jsonInt64(object, "beacons_received");
    status.lastBeaconMs = jsonInt64(object, "last_beacon_ms");

    const auto array = object.value(QStringLiteral("peers")).toArray();
    for (const auto &item : array) {
        if (item.isObject())
            status.peers.append(parsePeer(item.toObject()));
    }
    status.peerCount = jsonInt(object, "peer_count", int(status.peers.size()));
    status.pairedCount = jsonInt(object, "paired_count");
    return status;
}

} // namespace weft
