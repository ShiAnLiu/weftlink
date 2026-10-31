#pragma once

#include <QElapsedTimer>
#include <QHash>
#include <QJsonObject>
#include <QObject>
#include <QSslSocket>
#include <QString>

#include "weftclient/frame_codec.h"
#include "weftclient/weft_types.h"

namespace weft {

// Client is an async Weft Protocol client for the local daemon.
// It speaks TLS + length-prefixed JSON and emits typed signals.
class Client : public QObject
{
    Q_OBJECT

public:
    explicit Client(QObject *parent = nullptr);
    ~Client() override;

    void connectToDaemon(const QString &host, quint16 port);
    void disconnectFromDaemon();
    bool isConnected() const;
    QString endpoint() const;
    QString lastError() const { return m_lastError; }

    void requestStatus();
    void requestPeers();
    void ping();
    void requestPairCode();
    void joinPair(const QString &code, const QString &peerHost = {});

signals:
    void connectionStateChanged(bool connected);
    void statusReceived(const weft::Status &status);
    void peersReceived(const QList<weft::Peer> &peers);
    void pongReceived(qint64 rttMs);
    void pairCodeReceived(const QString &code, int expiresInSeconds);
    void pairResultReceived(const weft::PairResult &result);
    void errorOccurred(const QString &message);
    void logMessage(const QString &message);

private:
    void sendRequest(const QJsonObject &request);
    void handleMessage(const QJsonObject &message);
    void setError(const QString &message);

    static Peer parsePeer(const QJsonObject &object);
    static Status parseStatus(const QJsonObject &object);

    QSslSocket *m_socket = nullptr;
    FrameCodec m_codec;
    QHash<QString, QElapsedTimer> m_pingTimers;
    QString m_lastError;
    QString m_host;
    quint16 m_port = 0;
};

} // namespace weft
