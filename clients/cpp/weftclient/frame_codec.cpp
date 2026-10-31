#include "weftclient/frame_codec.h"

#include <QJsonDocument>
#include <QJsonParseError>
#include <QtEndian>

namespace weft {

QByteArray FrameCodec::encode(const QJsonObject &message, QString *error)
{
    const auto payload = QJsonDocument(message).toJson(QJsonDocument::Compact);
    if (payload.size() <= 0 || quint32(payload.size()) > MaxFrameSize) {
        if (error)
            *error = QStringLiteral("frame too large: %1 bytes").arg(payload.size());
        return {};
    }

    QByteArray out;
    out.resize(4);
    qToBigEndian<quint32>(quint32(payload.size()), reinterpret_cast<uchar *>(out.data()));
    out.append(payload);
    return out;
}

void FrameCodec::append(const QByteArray &data)
{
    m_buffer.append(data);
}

QList<QJsonObject> FrameCodec::takeMessages(QString *error)
{
    QList<QJsonObject> messages;
    while (m_buffer.size() >= 4) {
        const auto length = qFromBigEndian<quint32>(
            reinterpret_cast<const uchar *>(m_buffer.constData()));
        if (length > MaxFrameSize) {
            if (error)
                *error = QStringLiteral("frame too large: %1 bytes").arg(length);
            m_buffer.clear();
            break;
        }
        const auto needed = qsizetype(4) + qsizetype(length);
        if (m_buffer.size() < needed)
            break;

        const auto payload = m_buffer.mid(4, int(length));
        m_buffer.remove(0, needed);

        QJsonParseError parseError;
        const auto doc = QJsonDocument::fromJson(payload, &parseError);
        if (parseError.error != QJsonParseError::NoError || !doc.isObject()) {
            if (error) {
                *error = QStringLiteral("invalid JSON frame: %1")
                             .arg(parseError.errorString());
            }
            continue;
        }
        messages.append(doc.object());
    }
    return messages;
}

void FrameCodec::reset()
{
    m_buffer.clear();
}

} // namespace weft
