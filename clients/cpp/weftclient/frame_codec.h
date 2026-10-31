#pragma once

#include <QByteArray>
#include <QJsonObject>
#include <QList>
#include <QString>
#include <QtGlobal>

namespace weft {

// FrameCodec implements the Weft Protocol wire framing shared with the Go
// daemon: [4-byte big-endian length][UTF-8 JSON payload].
class FrameCodec
{
public:
    static constexpr quint32 MaxFrameSize = 1u << 20; // 1 MiB

    static QByteArray encode(const QJsonObject &message, QString *error = nullptr);

    void append(const QByteArray &data);
    QList<QJsonObject> takeMessages(QString *error = nullptr);
    void reset();
    qsizetype bufferedBytes() const noexcept { return m_buffer.size(); }

private:
    QByteArray m_buffer;
};

} // namespace weft
