#include <QtEndian>
#include <QtTest>

#include <QJsonDocument>

#include "weftclient/frame_codec.h"

class FrameCodecTest : public QObject
{
    Q_OBJECT

private slots:
    void encodesLengthPrefix();
    void roundTripsOneFrame();
    void waitsForPartialFrame();
    void decodesMultipleFrames();
    void rejectsOversize();
    void rejectsInvalidJson();
};

void FrameCodecTest::encodesLengthPrefix()
{
    const QJsonObject message{{QStringLiteral("type"), QStringLiteral("ping")}};
    const auto frame = weft::FrameCodec::encode(message);
    QVERIFY(!frame.isEmpty());
    QVERIFY(frame.size() > 4);

    const auto length = qFromBigEndian<quint32>(
        reinterpret_cast<const uchar *>(frame.constData()));
    const auto payload = QJsonDocument(message).toJson(QJsonDocument::Compact);
    QCOMPARE(length, quint32(payload.size()));
    QCOMPARE(frame.mid(4), payload);
}

void FrameCodecTest::roundTripsOneFrame()
{
    const QJsonObject message{{QStringLiteral("type"), QStringLiteral("status")}};
    const auto frame = weft::FrameCodec::encode(message);
    weft::FrameCodec codec;
    codec.append(frame);

    QString error;
    const auto messages = codec.takeMessages(&error);
    QVERIFY2(error.isEmpty(), qPrintable(error));
    QCOMPARE(messages.size(), 1);
    QCOMPARE(messages.first().value(QStringLiteral("type")).toString(),
             QStringLiteral("status"));
    QCOMPARE(codec.bufferedBytes(), qsizetype(0));
}

void FrameCodecTest::waitsForPartialFrame()
{
    const QJsonObject message{{QStringLiteral("type"), QStringLiteral("pong")}};
    const auto frame = weft::FrameCodec::encode(message);

    weft::FrameCodec codec;
    codec.append(frame.left(3));
    QVERIFY(codec.takeMessages().isEmpty());
    codec.append(frame.mid(3));

    const auto messages = codec.takeMessages();
    QCOMPARE(messages.size(), 1);
    QCOMPARE(messages.first().value(QStringLiteral("type")).toString(),
             QStringLiteral("pong"));
}

void FrameCodecTest::decodesMultipleFrames()
{
    const auto first = weft::FrameCodec::encode(
        QJsonObject{{QStringLiteral("type"), QStringLiteral("ping")}});
    const auto second = weft::FrameCodec::encode(
        QJsonObject{{QStringLiteral("type"), QStringLiteral("peers")}});

    weft::FrameCodec codec;
    codec.append(first + second.left(2));
    QCOMPARE(codec.takeMessages().size(), 1);
    codec.append(second.mid(2));

    const auto messages = codec.takeMessages();
    QCOMPARE(messages.size(), 1);
    QCOMPARE(messages.first().value(QStringLiteral("type")).toString(),
             QStringLiteral("peers"));
}

void FrameCodecTest::rejectsOversize()
{
    QByteArray header(4, '\0');
    const auto length = weft::FrameCodec::MaxFrameSize + 1;
    qToBigEndian<quint32>(length, reinterpret_cast<uchar *>(header.data()));

    weft::FrameCodec codec;
    codec.append(header);
    QString error;
    const auto messages = codec.takeMessages(&error);
    QVERIFY(messages.isEmpty());
    QVERIFY(error.contains(QStringLiteral("frame too large")));
    QCOMPARE(codec.bufferedBytes(), qsizetype(0));
}

void FrameCodecTest::rejectsInvalidJson()
{
    const QByteArray payload("{not json");
    QByteArray frame(4, '\0');
    qToBigEndian<quint32>(quint32(payload.size()),
                          reinterpret_cast<uchar *>(frame.data()));
    frame.append(payload);

    weft::FrameCodec codec;
    codec.append(frame);
    QString error;
    const auto messages = codec.takeMessages(&error);
    QVERIFY(messages.isEmpty());
    QVERIFY(error.contains(QStringLiteral("invalid JSON")));
}

QTEST_GUILESS_MAIN(FrameCodecTest)
#include "frame_codec_test.moc"
