#pragma once

#include <QList>
#include <QMetaType>
#include <QString>

namespace weft {

struct Peer
{
    QString peerId;
    QString displayName;
    QString role;
    QString version;
    QString address;
    QString fingerprint;
    qint64 lastSeen = 0;
    bool paired = false;
};

struct Status
{
    QString nodeId;
    QString displayName;
    QString role;
    QString version;
    QString fingerprint;
    QList<Peer> peers;
    int peerCount = 0;
    int pairedCount = 0;
    int discoveryPort = 0;
    bool discoveryActive = false;
    qint64 beaconsSent = 0;
    qint64 beaconsReceived = 0;
    qint64 lastBeaconMs = 0;
};

struct PairResult
{
    bool ok = false;
    QString error;
    Peer peer;
};

struct SendFileResult
{
    bool ok = false;
    QString checksum;
    qint64 size = 0;
    QString error;
};

struct ReceiveResult
{
    bool ok = false;
    bool pending = false;
    QString path;
    QString name;
    QString checksum;
    QString fromNodeId;
    qint64 size = 0;
    QString error;
};

struct RunResult
{
    bool ok = false;
    int exitCode = -1;
    QString stdoutText;
    QString stderrText;
    QString error;
};

struct ExecRequest
{
    QString execId;
    QString peerId;
    QString command;
    QString shell;
    QString cwd;
    qint64 createdMs = 0;
    int ageSeconds = 0;
};

struct ClipboardState
{
    bool active = false;
    QString from;
    int expiresInSeconds = 0;
};

struct ClipboardResult
{
    bool ok = false;
    int length = 0;
    QString error;
};

} // namespace weft

Q_DECLARE_METATYPE(weft::Peer)
Q_DECLARE_METATYPE(weft::Status)
Q_DECLARE_METATYPE(weft::PairResult)
Q_DECLARE_METATYPE(weft::SendFileResult)
Q_DECLARE_METATYPE(weft::ReceiveResult)
Q_DECLARE_METATYPE(weft::RunResult)
Q_DECLARE_METATYPE(weft::ExecRequest)
Q_DECLARE_METATYPE(weft::ClipboardState)
Q_DECLARE_METATYPE(weft::ClipboardResult)
