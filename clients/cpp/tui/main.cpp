#include <QCommandLineOption>
#include <QCommandLineParser>
#include <QCoreApplication>
#include <QEventLoop>
#include <QTextStream>
#include <QTimer>

#include <cstdio>

#include "tui/tui_app.h"

namespace {

int runCheck(const QString &host, quint16 port)
{
    weft::Client client;
    QEventLoop loop;
    QTimer timeout;
    timeout.setSingleShot(true);
    timeout.start(5000);
    int result = 1;

    QObject::connect(&timeout, &QTimer::timeout, &loop, [&loop, &result] {
        std::fprintf(stderr, "weftlink-tui: timeout waiting for daemon\n");
        result = 1;
        loop.quit();
    });
    QObject::connect(&client, &weft::Client::statusReceived, &loop,
                     [&client, &loop, &result](const weft::Status &status) {
                         QTextStream out(stdout);
                         out << "node_id=" << status.nodeId << '\n';
                         out << "display_name=" << status.displayName << '\n';
                         out << "version=" << status.version << '\n';
                         out << "role=" << status.role << '\n';
                         out << "peers=" << status.peerCount << '\n';
                         out << "paired=" << status.pairedCount << '\n';
                         out << "discovery_port=" << status.discoveryPort << '\n';
                         out << "discovery_active=" << (status.discoveryActive ? "true" : "false")
                             << '\n';
                         out << "beacons_sent=" << status.beaconsSent << '\n';
                         out << "beacons_received=" << status.beaconsReceived << '\n';
                         for (const auto &peer : status.peers) {
                             out << "peer=" << peer.peerId << '|' << peer.displayName << '|'
                                 << peer.address << '|' << (peer.paired ? "paired" : "unpaired")
                                 << '\n';
                         }
                         client.requestExecRequests();
                     });
    QObject::connect(&client, &weft::Client::execRequestsReceived, &loop,
                     [&client, &loop, &result](const QList<weft::ExecRequest> &requests) {
                         QTextStream out(stdout);
                         out << "exec_requests=" << requests.size() << '\n';
                         client.requestClipboardState();
                     });
    QObject::connect(&client, &weft::Client::clipboardStateReceived, &loop,
                     [&loop, &result](const weft::ClipboardState &state) {
                         QTextStream out(stdout);
                         out << "clipboard_active=" << (state.active ? "true" : "false") << '\n';
                         out << "clipboard_expires=" << state.expiresInSeconds << '\n';
                         result = 0;
                         loop.quit();
                     });
    QObject::connect(&client, &weft::Client::errorOccurred, &loop,
                     [&loop, &result](const QString &message) {
                         std::fprintf(stderr, "weftlink-tui: %s\n",
                                      message.toUtf8().constData());
                         result = 1;
                         loop.quit();
                     });

    client.connectToDaemon(host, port);
    loop.exec();
    client.disconnectFromDaemon();
    return result;
}

} // namespace

int main(int argc, char *argv[])
{
    QCoreApplication app(argc, argv);
    QCoreApplication::setApplicationName(QStringLiteral("weftlink-tui"));
    QCoreApplication::setApplicationVersion(QStringLiteral(WEFTLINK_CLIENT_VERSION));

    QCommandLineParser parser;
    parser.setApplicationDescription(
        QStringLiteral("Weftlink terminal UI (Weft Protocol over TLS)"));
    parser.addHelpOption();
    parser.addVersionOption();

    QCommandLineOption hostOption({QStringLiteral("H"), QStringLiteral("host")},
                                  QStringLiteral("Daemon host"),
                                  QStringLiteral("host"),
                                  qEnvironmentVariable("WEFTLINK_HOST", "127.0.0.1"));
    QCommandLineOption portOption({QStringLiteral("p"), QStringLiteral("port")},
                                  QStringLiteral("Daemon port"),
                                  QStringLiteral("port"),
                                  qEnvironmentVariable("WEFTLINK_PORT", "7801"));
    QCommandLineOption checkOption(
        QStringLiteral("check"),
        QStringLiteral("Connect, print daemon status, and exit (no TUI)"));
    parser.addOptions({hostOption, portOption, checkOption});
    parser.process(app);

    bool ok = false;
    auto port = parser.value(portOption).toInt(&ok);
    if (!ok || port <= 0 || port > 65535)
        port = 7801;

    if (parser.isSet(checkOption))
        return runCheck(parser.value(hostOption), quint16(port));

    TuiApp tui(parser.value(hostOption), quint16(port));
    return tui.run();
}
