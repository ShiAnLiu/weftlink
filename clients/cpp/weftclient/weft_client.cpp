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
    qRegisterMetaType<weft::SendFileResult>();
    qRegisterMetaType<weft::ReceiveResult>();
    qRegisterMetaType<weft::RunResult>();
    qRegisterMetaType<weft::ExecRequest>();
    qRegisterMetaType<weft::ClipboardState>();
    qRegisterMetaType<weft::ClipboardResult>();

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

void Client::requestPeerStatus(const QString &peer)
{
    m_pendingStatusPeer = peer;
    sendRequest({{QStringLiteral("type"), QStringLiteral("peer_status")},
                 {QStringLiteral("peer"), peer}});
}

void Client::sendFile(const QString &peer, const QString &path)
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("send_file")},
                 {QStringLiteral("peer"), peer},
                 {QStringLiteral("path"), path}});
}

void Client::beginReceive(const QString &dir, const QString &from, int timeoutSeconds)
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("recv_begin")},
                 {QStringLiteral("dir"), dir},
                 {QStringLiteral("from"), from},
                 {QStringLiteral("timeout_seconds"), timeoutSeconds}});
}

void Client::pollReceive()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("recv_poll")}});
}

void Client::cancelReceive()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("recv_cancel")}});
}

void Client::runCommand(const QString &peer, const QString &command, int timeoutSeconds,
                        const QString &shell)
{
    m_pendingRunPeer = peer;
    sendRequest({{QStringLiteral("type"), QStringLiteral("run_command")},
                 {QStringLiteral("peer"), peer},
                 {QStringLiteral("command"), command},
                 {QStringLiteral("timeout_seconds"), timeoutSeconds},
                 {QStringLiteral("shell"), shell}});
}

void Client::requestExecRequests()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("exec_requests")}});
}

void Client::approveExec(const QString &execId)
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("exec_approve")},
                 {QStringLiteral("exec_id"), execId}});
}

void Client::denyExec(const QString &execId)
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("exec_deny")},
                 {QStringLiteral("exec_id"), execId}});
}

void Client::clipboardOn(const QString &from, int timeoutSeconds)
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("clipboard_begin")},
                 {QStringLiteral("from"), from},
                 {QStringLiteral("timeout_seconds"), timeoutSeconds}});
}

void Client::clipboardOff()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("clipboard_end")}});
}

void Client::requestClipboardState()
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("clipboard_state")}});
}

void Client::clipboardPush(const QString &peer)
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("clipboard_push")},
                 {QStringLiteral("peer"), peer}});
}

void Client::clipboardPull(const QString &peer)
{
    sendRequest({{QStringLiteral("type"), QStringLiteral("clipboard_pull")},
                 {QStringLiteral("peer"), peer}});
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
    } else if (type == QLatin1String("peer_status")) {
        if (!message.value(QStringLiteral("ok")).toBool()) {
            setError(jsonString(message, "error"));
        } else if (message.value(QStringLiteral("status")).isObject()) {
            emit peerStatusReceived(m_pendingStatusPeer,
                                    parseStatus(message.value(QStringLiteral("status")).toObject()));
        }
        m_pendingStatusPeer.clear();
    } else if (type == QLatin1String("send_result")) {
        SendFileResult result;
        result.ok = message.value(QStringLiteral("ok")).toBool();
        result.checksum = jsonString(message, "checksum");
        result.size = jsonInt64(message, "size");
        result.error = jsonString(message, "error");
        emit fileSent(result);
        emit logMessage(result.ok
                            ? QStringLiteral("File sent (%1 bytes)").arg(result.size)
                            : QStringLiteral("Send failed: %1").arg(result.error));
    } else if (type == QLatin1String("recv_begin")) {
        if (!message.value(QStringLiteral("ok")).toBool()) {
            setError(jsonString(message, "error"));
        } else {
            emit fileReceiveStarted(jsonString(message, "dir"),
                                    jsonInt(message, "timeout_seconds", 300));
            emit logMessage(QStringLiteral("Receiving into %1").arg(jsonString(message, "dir")));
        }
    } else if (type == QLatin1String("recv_result")) {
        ReceiveResult result;
        result.pending = message.value(QStringLiteral("pending")).toBool();
        result.ok = message.value(QStringLiteral("ok")).toBool();
        result.path = jsonString(message, "path");
        result.name = jsonString(message, "name");
        result.checksum = jsonString(message, "checksum");
        result.fromNodeId = jsonString(message, "from_node_id");
        result.size = jsonInt64(message, "size");
        result.error = jsonString(message, "error");
        if (!result.pending)
            emit fileReceived(result);
    } else if (type == QLatin1String("run_result")) {
        RunResult result;
        result.ok = message.value(QStringLiteral("ok")).toBool();
        result.exitCode = jsonInt(message, "exit_code", -1);
        result.stdoutText = jsonString(message, "stdout");
        result.stderrText = jsonString(message, "stderr");
        result.error = jsonString(message, "error");
        emit runResultReceived(m_pendingRunPeer, result);
        m_pendingRunPeer.clear();
    } else if (type == QLatin1String("exec_requests")) {
        QList<ExecRequest> requests;
        const auto array = message.value(QStringLiteral("requests")).toArray();
        for (const auto &item : array) {
            if (item.isObject())
                requests.append(parseExecRequest(item.toObject()));
        }
        emit execRequestsReceived(requests);
    } else if (type == QLatin1String("exec_action")) {
        emit execActionFinished(message.value(QStringLiteral("ok")).toBool(),
                                jsonString(message, "error"));
    } else if (type == QLatin1String("clipboard_begin")
               || type == QLatin1String("clipboard_end")
               || type == QLatin1String("clipboard_state")) {
        ClipboardState state;
        state.active = type != QLatin1String("clipboard_end")
            && message.value(QStringLiteral("active")).toBool(true);
        state.from = jsonString(message, "from");
        state.expiresInSeconds = jsonInt(message, "expires_in_seconds", 0);
        if (type == QLatin1String("clipboard_begin") && state.active)
            state.active = true;
        emit clipboardStateReceived(state);
    } else if (type == QLatin1String("clipboard_result")) {
        ClipboardResult result;
        result.ok = message.value(QStringLiteral("ok")).toBool();
        result.length = jsonInt(message, "length");
        result.error = jsonString(message, "error");
        emit clipboardResultReceived(result);
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

ExecRequest Client::parseExecRequest(const QJsonObject &object)
{
    ExecRequest request;
    request.execId = jsonString(object, "exec_id");
    request.peerId = jsonString(object, "peer_id");
    request.command = jsonString(object, "command");
    request.shell = jsonString(object, "shell");
    request.cwd = jsonString(object, "cwd");
    request.createdMs = jsonInt64(object, "created_ms");
    request.ageSeconds = jsonInt(object, "age_seconds");
    return request;
}

} // namespace weft
