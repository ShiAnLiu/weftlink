#include "tui/tui_app.h"

#include <QCoreApplication>
#include <QDateTime>
#include <QEventLoop>
#include <QThread>
#include <QTime>

#include <cstdio>

namespace {

QString paint(const char *color, const QString &text)
{
    return QStringLiteral("\x1b[%1m%2\x1b[0m")
        .arg(QString::fromLatin1(color), text);
}

int visibleLength(const QString &text)
{
    int count = 0;
    bool escape = false;
    for (const auto ch : text) {
        if (escape) {
            if ((ch >= QLatin1Char('A') && ch <= QLatin1Char('Z'))
                || (ch >= QLatin1Char('a') && ch <= QLatin1Char('z'))) {
                escape = false;
            }
            continue;
        }
        if (ch == QChar(0x1b)) {
            escape = true;
            continue;
        }
        ++count;
    }
    return count;
}

QString fit(const QString &text, int width)
{
    if (width <= 0)
        return {};
    QString out;
    int visible = 0;
    bool escape = false;
    bool hasAnsi = false;
    for (int i = 0; i < text.size() && visible < width; ++i) {
        const auto ch = text.at(i);
        if (escape) {
            out += ch;
            if ((ch >= QLatin1Char('A') && ch <= QLatin1Char('Z'))
                || (ch >= QLatin1Char('a') && ch <= QLatin1Char('z'))) {
                escape = false;
            }
            continue;
        }
        if (ch == QChar(0x1b)) {
            escape = true;
            hasAnsi = true;
            out += ch;
            continue;
        }
        out += ch;
        ++visible;
    }
    if (escape)
        out += QLatin1Char('m');
    if (hasAnsi)
        out += QStringLiteral("\x1b[0m");
    if (visible < width)
        out += QString(width - visible, QLatin1Char(' '));
    return out;
}

QString separator(const QString &title, int width)
{
    if (width < 2)
        return {};
    if (title.isEmpty())
        return QStringLiteral("+") + QString(width - 2, QLatin1Char('-')) + QStringLiteral("+");
    const auto label = QStringLiteral(" ") + title + QStringLiteral(" ");
    auto remaining = width - 2 - visibleLength(label);
    if (remaining < 0)
        remaining = 0;
    return QStringLiteral("+") + label + QString(remaining, QLatin1Char('-')) + QStringLiteral("+");
}

QString border(int width)
{
    return separator(QString(), width);
}

QString shortFingerprint(const QString &fingerprint)
{
    if (fingerprint.size() <= 32)
        return fingerprint;
    return fingerprint.left(16) + QStringLiteral("...") + fingerprint.right(8);
}

} // namespace

TuiApp::TuiApp(QString host, quint16 port)
    : m_host(std::move(host))
    , m_port(port)
{
}

int TuiApp::run()
{
    QString error;
    if (!m_term.init(&error)) {
        std::fprintf(stderr, "weftlink-tui: %s\n", error.toUtf8().constData());
        return 1;
    }

    QObject::connect(&m_client, &weft::Client::connectionStateChanged, &m_client,
                     [this](bool connected) {
                         m_connected = connected;
                         if (connected)
                             m_lastRefresh.restart();
                         m_dirty = true;
                     });
    QObject::connect(&m_client, &weft::Client::statusReceived, &m_client,
                     [this](const weft::Status &status) {
                         m_status = status;
                         m_peers = status.peers;
                         m_dirty = true;
                     });
    QObject::connect(&m_client, &weft::Client::pongReceived, &m_client,
                     [this](qint64 rtt) {
                         if (rtt >= 0)
                             log(QStringLiteral("Ping: %1 ms").arg(rtt));
                         else
                             log(QStringLiteral("Pong received"));
                     });
    QObject::connect(&m_client, &weft::Client::pairCodeReceived, &m_client,
                     [this](const QString &code, int expires) {
                         m_pairCode = code;
                         m_pairExpires = expires;
                         m_pairTimer.start();
                         log(QStringLiteral("Pair code: %1 (expires in %2s)")
                                 .arg(code)
                                 .arg(expires));
                         m_dirty = true;
                     });
    QObject::connect(&m_client, &weft::Client::pairResultReceived, &m_client,
                     [this](const weft::PairResult &) { m_dirty = true; });
    QObject::connect(&m_client, &weft::Client::errorOccurred, &m_client,
                     [this](const QString &) { m_dirty = true; });
    QObject::connect(&m_client, &weft::Client::logMessage, &m_client,
                     [this](const QString &message) { log(message); });

    log(QStringLiteral("Weftlink TUI %1").arg(QStringLiteral(WEFTLINK_CLIENT_VERSION)));
    log(QStringLiteral("Press c to connect to %1:%2").arg(m_host).arg(m_port));
    m_lastRefresh.start();
    render();
    m_dirty = false;

    while (m_running) {
        QCoreApplication::processEvents(QEventLoop::AllEvents, 10);

        for (auto key = m_term.pollKey(); !key.isEmpty(); key = m_term.pollKey()) {
            handleKey(key);
            if (!m_running)
                break;
        }
        if (!m_running)
            break;

        if (m_connected && m_lastRefresh.elapsed() >= 2000) {
            m_client.requestStatus();
            m_lastRefresh.restart();
        }
        if (m_dirty) {
            render();
            m_dirty = false;
        }
        QThread::msleep(20);
    }

    m_client.disconnectFromDaemon();
    QCoreApplication::processEvents(QEventLoop::AllEvents, 100);
    m_term.shutdown();
    return 0;
}

void TuiApp::handleKey(const QString &key)
{
    if (key.isEmpty())
        return;

    if (m_mode == InputMode::JoinCode) {
        if (key == QLatin1String("ESC")) {
            m_mode = InputMode::Normal;
            m_input.clear();
            log(QStringLiteral("Join cancelled"));
        } else if (key == QLatin1String("ENTER")) {
            if (m_input.size() == 6) {
                m_joinCode = m_input;
                m_input.clear();
                m_mode = InputMode::JoinHost;
                log(QStringLiteral("Enter peer host (optional), Enter to join"));
            } else {
                log(QStringLiteral("Pair code must be 6 characters"));
            }
        } else if (key == QLatin1String("BACKSPACE")) {
            m_input.chop(1);
        } else if (key.size() == 1 && key.at(0).isLetterOrNumber() && m_input.size() < 6) {
            m_input += key.toUpper();
        }
        m_dirty = true;
        return;
    }

    if (m_mode == InputMode::JoinHost) {
        if (key == QLatin1String("ESC")) {
            m_mode = InputMode::Normal;
            m_input.clear();
            m_joinCode.clear();
            log(QStringLiteral("Join cancelled"));
        } else if (key == QLatin1String("ENTER")) {
            commitJoin();
        } else if (key == QLatin1String("BACKSPACE")) {
            m_input.chop(1);
        } else if (key.size() == 1 && key.at(0).isPrint() && m_input.size() < 128) {
            m_input += key;
        }
        m_dirty = true;
        return;
    }

    if (key == QLatin1String("q") || key == QLatin1String("CTRL_C")) {
        m_running = false;
        return;
    }
    if (key == QLatin1String("c")) {
        if (m_connected) {
            log(QStringLiteral("Already connected"));
        } else {
            log(QStringLiteral("Connecting to %1:%2 ...").arg(m_host).arg(m_port));
            m_client.connectToDaemon(m_host, m_port);
        }
        m_dirty = true;
        return;
    }
    if (key == QLatin1String("d")) {
        m_client.disconnectFromDaemon();
        m_dirty = true;
        return;
    }
    if (key == QLatin1String("p")) {
        if (m_connected)
            m_client.ping();
        else
            log(QStringLiteral("Not connected (press c)"));
        m_dirty = true;
        return;
    }
    if (key == QLatin1String("r")) {
        if (m_connected) {
            m_client.requestStatus();
            log(QStringLiteral("Refresh requested"));
        } else {
            log(QStringLiteral("Not connected (press c)"));
        }
        m_dirty = true;
        return;
    }
    if (key == QLatin1String("n")) {
        if (m_connected)
            m_client.requestPairCode();
        else
            log(QStringLiteral("Not connected (press c)"));
        m_dirty = true;
        return;
    }
    if (key == QLatin1String("j")) {
        startJoin();
        return;
    }
    if (key == QLatin1String("h") || key == QLatin1String("?")) {
        log(QStringLiteral("c connect  d disconnect  p ping  r refresh  n new code  j join  q quit"));
        m_dirty = true;
    }
}

void TuiApp::startJoin()
{
    if (!m_connected) {
        log(QStringLiteral("Not connected (press c first)"));
        m_dirty = true;
        return;
    }
    m_mode = InputMode::JoinCode;
    m_input.clear();
    m_joinCode.clear();
    log(QStringLiteral("Enter the 6-character pair code"));
    m_dirty = true;
}

void TuiApp::commitJoin()
{
    const auto code = m_joinCode.trimmed().toUpper();
    const auto host = m_input.trimmed();
    m_mode = InputMode::Normal;
    m_input.clear();
    m_joinCode.clear();
    log(QStringLiteral("Joining with code %1 ...").arg(code));
    m_client.joinPair(code, host);
    m_dirty = true;
}

void TuiApp::refresh()
{
    if (m_connected)
        m_client.requestStatus();
}

void TuiApp::render()
{
    const auto terminalSize = m_term.size();
    const auto width = qMax(40, terminalSize.width());
    const auto height = qMax(12, terminalSize.height());
    const auto innerWidth = width - 4;

    auto box = [innerWidth](const QString &content) {
        return QStringLiteral("| ") + fit(content, innerWidth) + QStringLiteral(" |");
    };

    QStringList lines;
    lines << paint("36", border(width));

    const auto title = paint("1;36", QStringLiteral("WEFTLINK"))
        + QLatin1Char(' ') + paint("90", QStringLiteral("Weft Protocol terminal"));
    const auto endpoint = paint("90", QStringLiteral("%1:%2").arg(m_host).arg(m_port));
    auto gap = innerWidth - visibleLength(title) - visibleLength(endpoint);
    if (gap < 1)
        gap = 1;
    lines << QStringLiteral("| ") + fit(title + QString(gap, QLatin1Char(' ')) + endpoint,
                                       innerWidth)
                 + QStringLiteral(" |");

    lines << paint("36", separator(QStringLiteral("Status"), width));
    lines << box(paint("90", QStringLiteral("Endpoint  ")) + paint("1", endpoint));
    lines << box(paint("90", QStringLiteral("State     "))
                 + (m_connected ? paint("32", QStringLiteral("Connected"))
                                : paint("31", QStringLiteral("Disconnected"))));
    if (!m_status.nodeId.isEmpty()) {
        lines << box(paint("90", QStringLiteral("Node      "))
                     + QStringLiteral("%1  %2  %3  v%4")
                           .arg(m_status.nodeId, m_status.displayName, m_status.role,
                                m_status.version));
        lines << box(paint("90", QStringLiteral("Fingerprint  "))
                     + paint("90", shortFingerprint(m_status.fingerprint)));
    }
    if (m_connected) {
        if (m_status.discoveryPort <= 0) {
            lines << box(paint("90", QStringLiteral("Discovery  "))
                         + paint("90", QStringLiteral("not reported by daemon")));
        } else if (m_status.discoveryActive) {
            const auto last = m_status.lastBeaconMs > 0 ? formatAge(m_status.lastBeaconMs)
                                                        : QStringLiteral("-");
            lines << box(paint("90", QStringLiteral("Discovery  "))
                         + QStringLiteral("UDP/%1 active  sent %2  received %3  last %4")
                               .arg(m_status.discoveryPort)
                               .arg(m_status.beaconsSent)
                               .arg(m_status.beaconsReceived)
                               .arg(last));
        } else {
            lines << box(paint("90", QStringLiteral("Discovery  "))
                         + paint("33", QStringLiteral("UDP/%1 inactive")
                                           .arg(m_status.discoveryPort)));
        }
    } else {
        lines << box(paint("90", QStringLiteral("Discovery  -")));
    }
    if (!m_pairCode.isEmpty()) {
        const auto left = pairSecondsLeft();
        lines << box(paint("90", QStringLiteral("Pair code  "))
                     + paint("1;33", m_pairCode)
                     + (left > 0 ? paint("90", QStringLiteral("  expires in %1s").arg(left))
                                 : paint("31", QStringLiteral("  expired"))));
    }

    auto pairedCount = 0;
    for (const auto &peer : m_peers) {
        if (peer.paired)
            ++pairedCount;
    }

    const auto statusLines = 3 + (m_status.nodeId.isEmpty() ? 0 : 2)
        + (m_pairCode.isEmpty() ? 0 : 1);
    const auto fixedLines = 1 + 1 + 1 + statusLines + 1 + 1 + 1 + 1 + 1 + 1 + 1;
    auto available = qMax(2, height - fixedLines);
    auto peerRows = qMin(qMax(1, int(m_peers.size())), qMax(2, available / 2));
    auto activityRows = qMax(1, available - peerRows);

    lines << paint("36", separator(QStringLiteral("Peers (%1, %2 paired)")
                                        .arg(m_peers.size())
                                        .arg(pairedCount),
                                    width));
    lines << box(paint("90", QStringLiteral("%1 %2 %3 %4 %5")
                                    .arg(QStringLiteral("P"), -1)
                                    .arg(QStringLiteral("NODE ID"), -14)
                                    .arg(QStringLiteral("NAME"), -20)
                                    .arg(QStringLiteral("ADDRESS"), -22)
                                    .arg(QStringLiteral("LAST SEEN"))));
    if (m_peers.isEmpty()) {
        lines << box(paint("90", QStringLiteral("(no peers discovered)")));
    } else {
        for (int i = 0; i < qMin(peerRows, int(m_peers.size())); ++i) {
            const auto &peer = m_peers.at(i);
            auto name = peer.displayName.isEmpty() ? QStringLiteral("-") : peer.displayName;
            auto address = peer.address.isEmpty() ? QStringLiteral("-") : peer.address;
            auto marker = peer.paired ? paint("32", QStringLiteral("*"))
                                      : paint("90", QStringLiteral("-"));
            lines << box(marker + QLatin1Char(' ')
                         + paint("1", peer.peerId.left(14).leftJustified(14))
                         + QLatin1Char(' ') + name.left(20).leftJustified(20)
                         + QLatin1Char(' ') + address.left(22).leftJustified(22)
                         + QLatin1Char(' ') + paint("90", formatAge(peer.lastSeen)));
        }
    }

    lines << paint("36", separator(QStringLiteral("Activity"), width));
    const auto firstLog = qMax(0, int(m_log.size()) - activityRows);
    for (int i = firstLog; i < int(m_log.size()); ++i)
        lines << box(m_log.at(i));
    if (m_log.isEmpty())
        lines << box(paint("90", QStringLiteral("(no activity yet)")));

    lines << paint("36", border(width));
    lines << box(paint("1", QStringLiteral("c")) + QStringLiteral(" connect  ")
                 + paint("1", QStringLiteral("d")) + QStringLiteral(" disconnect  ")
                 + paint("1", QStringLiteral("p")) + QStringLiteral(" ping  ")
                 + paint("1", QStringLiteral("r")) + QStringLiteral(" refresh  ")
                 + paint("1", QStringLiteral("n")) + QStringLiteral(" new code  ")
                 + paint("1", QStringLiteral("j")) + QStringLiteral(" join  ")
                 + paint("1", QStringLiteral("q")) + QStringLiteral(" quit"));
    if (m_mode == InputMode::JoinCode) {
        lines << box(paint("33", QStringLiteral("Pair code: ")) + m_input
                     + QStringLiteral("_   Enter=next  Esc=cancel"));
    } else if (m_mode == InputMode::JoinHost) {
        lines << box(paint("33", QStringLiteral("Peer host (optional): ")) + m_input
                     + QStringLiteral("_   Enter=join  Esc=cancel"));
    } else {
        lines << box(paint("90", QStringLiteral("Auto-refresh every 2s  |  one-time pair codes  |  TLS loopback")));
    }
    lines << paint("36", border(width));

    while (int(lines.size()) > height)
        lines.removeLast();

    auto screen = QStringLiteral("\x1b[H\x1b[2J");
    screen += lines.join(QLatin1Char('\n'));
    m_term.write(screen);
    m_term.flush();
}

void TuiApp::log(const QString &message)
{
    const auto stamp = QTime::currentTime().toString(QStringLiteral("HH:mm:ss"));
    m_log.append(QStringLiteral("[%1] %2").arg(stamp, message));
    while (int(m_log.size()) > 300)
        m_log.removeFirst();
    m_dirty = true;
}

QString TuiApp::formatAge(qint64 millisecondsSinceEpoch) const
{
    if (millisecondsSinceEpoch <= 0)
        return QStringLiteral("-");
    auto diff = QDateTime::currentMSecsSinceEpoch() - millisecondsSinceEpoch;
    if (diff < 0)
        diff = 0;
    if (diff < 1000)
        return QStringLiteral("now");
    if (diff < 60 * 1000)
        return QStringLiteral("%1s ago").arg(diff / 1000);
    if (diff < 60 * 60 * 1000)
        return QStringLiteral("%1m ago").arg(diff / (60 * 1000));
    return QStringLiteral("%1h ago").arg(diff / (60 * 60 * 1000));
}

int TuiApp::pairSecondsLeft() const
{
    if (m_pairCode.isEmpty() || !m_pairTimer.isValid())
        return 0;
    const auto left = m_pairExpires - int(m_pairTimer.elapsed() / 1000);
    return qMax(0, left);
}
