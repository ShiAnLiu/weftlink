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

} // namespace weft

Q_DECLARE_METATYPE(weft::Peer)
Q_DECLARE_METATYPE(weft::Status)
Q_DECLARE_METATYPE(weft::PairResult)
