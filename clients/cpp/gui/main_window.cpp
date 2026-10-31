#include "gui/main_window.h"

#include "gui/pair_dialog.h"

#include <QClipboard>
#include <QCloseEvent>
#include <QColor>
#include <QDateTime>
#include <QFontDatabase>
#include <QFormLayout>
#include <QGridLayout>
#include <QGroupBox>
#include <QGuiApplication>
#include <QHBoxLayout>
#include <QHeaderView>
#include <QLabel>
#include <QLineEdit>
#include <QMessageBox>
#include <QPlainTextEdit>
#include <QPushButton>
#include <QSpinBox>
#include <QStatusBar>
#include <QStringList>
#include <QTableWidget>
#include <QTimer>
#include <QVBoxLayout>

namespace {

QLabel *makeCaption(const QString &text, QWidget *parent)
{
    auto *label = new QLabel(text, parent);
    label->setObjectName(QStringLiteral("caption"));
    return label;
}

QLabel *makeValue(QWidget *parent)
{
    auto *label = new QLabel(QStringLiteral("-"), parent);
    label->setObjectName(QStringLiteral("value"));
    label->setTextInteractionFlags(Qt::TextSelectableByMouse);
    label->setWordWrap(true);
    return label;
}

} // namespace

MainWindow::MainWindow(QWidget *parent)
    : QMainWindow(parent)
    , m_client(new weft::Client(this))
{
    setWindowTitle(QStringLiteral("Weftlink"));
    resize(1180, 740);

    buildUi();
    applyStyle();
    clearStatus();
    setConnectedUi(false);

    m_refreshTimer = new QTimer(this);
    m_refreshTimer->setInterval(2000);
    connect(m_refreshTimer, &QTimer::timeout, this, &MainWindow::refresh);

    connect(m_client, &weft::Client::connectionStateChanged,
            this, &MainWindow::onConnectionStateChanged);
    connect(m_client, &weft::Client::statusReceived, this, &MainWindow::onStatus);
    connect(m_client, &weft::Client::pongReceived, this, &MainWindow::onPong);
    connect(m_client, &weft::Client::pairCodeReceived, this, &MainWindow::onPairCode);
    connect(m_client, &weft::Client::pairResultReceived, this, &MainWindow::onPairResult);
    connect(m_client, &weft::Client::errorOccurred, this, &MainWindow::onError);
    connect(m_client, &weft::Client::logMessage, this, &MainWindow::onLog);

    appendLog(QStringLiteral("Ready. Start weftlinkd with 'weftlink up' on this machine, then Connect."));
    statusBar()->showMessage(QStringLiteral("Disconnected"));
}

MainWindow::~MainWindow() = default;

void MainWindow::buildUi()
{
    auto *central = new QWidget(this);
    central->setObjectName(QStringLiteral("central"));
    setCentralWidget(central);

    auto host = qEnvironmentVariable("WEFTLINK_HOST");
    if (host.trimmed().isEmpty())
        host = QStringLiteral("127.0.0.1");
    auto port = qEnvironmentVariableIntValue("WEFTLINK_PORT");
    if (port <= 0)
        port = 7801;

    m_hostEdit = new QLineEdit(host, central);
    m_hostEdit->setPlaceholderText(QStringLiteral("127.0.0.1"));
    m_portSpin = new QSpinBox(central);
    m_portSpin->setRange(1, 65535);
    m_portSpin->setValue(port);

    m_connectButton = new QPushButton(QStringLiteral("Connect"), central);
    m_connectButton->setObjectName(QStringLiteral("primaryButton"));
    m_pingButton = new QPushButton(QStringLiteral("Ping"), central);

    auto *endpointButtons = new QHBoxLayout;
    endpointButtons->setContentsMargins(0, 0, 0, 0);
    endpointButtons->addWidget(m_connectButton, 2);
    endpointButtons->addWidget(m_pingButton, 1);

    auto *endpointForm = new QFormLayout;
    endpointForm->setLabelAlignment(Qt::AlignLeft);
    endpointForm->addRow(QStringLiteral("Host"), m_hostEdit);
    endpointForm->addRow(QStringLiteral("Port"), m_portSpin);
    endpointForm->addRow(endpointButtons);

    auto *endpointBox = new QGroupBox(QStringLiteral("Daemon"), central);
    endpointBox->setLayout(endpointForm);

    m_stateValue = makeValue(central);
    m_nodeValue = makeValue(central);
    m_nameValue = makeValue(central);
    m_roleValue = makeValue(central);
    m_versionValue = makeValue(central);
    m_fingerprintValue = makeValue(central);
    m_fingerprintValue->setFont(QFontDatabase::systemFont(QFontDatabase::FixedFont));
    m_fingerprintValue->setWordWrap(true);

    auto *nodeGrid = new QGridLayout;
    nodeGrid->setColumnStretch(1, 1);
    nodeGrid->addWidget(makeCaption(QStringLiteral("State"), central), 0, 0);
    nodeGrid->addWidget(m_stateValue, 0, 1);
    nodeGrid->addWidget(makeCaption(QStringLiteral("Node ID"), central), 1, 0);
    nodeGrid->addWidget(m_nodeValue, 1, 1);
    nodeGrid->addWidget(makeCaption(QStringLiteral("Name"), central), 2, 0);
    nodeGrid->addWidget(m_nameValue, 2, 1);
    nodeGrid->addWidget(makeCaption(QStringLiteral("Role"), central), 3, 0);
    nodeGrid->addWidget(m_roleValue, 3, 1);
    nodeGrid->addWidget(makeCaption(QStringLiteral("Version"), central), 4, 0);
    nodeGrid->addWidget(m_versionValue, 4, 1);
    nodeGrid->addWidget(makeCaption(QStringLiteral("Fingerprint"), central), 5, 0, Qt::AlignTop);
    nodeGrid->addWidget(m_fingerprintValue, 5, 1);

    auto *nodeBox = new QGroupBox(QStringLiteral("Node"), central);
    nodeBox->setLayout(nodeGrid);

    m_discoveryValue = new QLabel(QStringLiteral("-"), central);
    m_discoveryValue->setObjectName(QStringLiteral("value"));
    m_discoveryValue->setWordWrap(true);
    m_discoveryValue->setTextInteractionFlags(Qt::TextSelectableByMouse);
    auto *discoveryLayout = new QVBoxLayout;
    discoveryLayout->addWidget(m_discoveryValue);
    auto *discoveryBox = new QGroupBox(QStringLiteral("Discovery"), central);
    discoveryBox->setLayout(discoveryLayout);

    m_newCodeButton = new QPushButton(QStringLiteral("New pair code"), central);
    m_newCodeButton->setObjectName(QStringLiteral("primaryButton"));
    m_joinButton = new QPushButton(QStringLiteral("Join with code"), central);

    auto *pairButtons = new QHBoxLayout;
    pairButtons->setContentsMargins(0, 0, 0, 0);
    pairButtons->addWidget(m_newCodeButton, 1);
    pairButtons->addWidget(m_joinButton, 1);

    m_pairCodeValue = new QLabel(QStringLiteral("—"), central);
    m_pairCodeValue->setObjectName(QStringLiteral("pairCode"));
    m_pairCodeValue->setAlignment(Qt::AlignCenter);
    m_pairCodeValue->setTextInteractionFlags(Qt::TextSelectableByMouse);

    m_copyCodeButton = new QPushButton(QStringLiteral("Copy code"), central);
    m_pairHint = new QLabel(QStringLiteral("No active code"), central);
    m_pairHint->setObjectName(QStringLiteral("hint"));
    m_pairHint->setWordWrap(true);
    m_pairHint->setAlignment(Qt::AlignCenter);

    auto *pairBoxLayout = new QVBoxLayout;
    pairBoxLayout->addLayout(pairButtons);
    pairBoxLayout->addWidget(m_pairCodeValue);
    pairBoxLayout->addWidget(m_copyCodeButton);
    pairBoxLayout->addWidget(m_pairHint);
    auto *pairBox = new QGroupBox(QStringLiteral("Pair"), central);
    pairBox->setLayout(pairBoxLayout);

    auto *left = new QWidget(central);
    left->setFixedWidth(370);
    auto *leftLayout = new QVBoxLayout(left);
    leftLayout->setContentsMargins(0, 0, 0, 0);
    leftLayout->setSpacing(12);

    auto *title = new QLabel(QStringLiteral("Weftlink"), left);
    title->setObjectName(QStringLiteral("title"));
    auto *subtitle = new QLabel(QStringLiteral("Weft Protocol desktop console"), left);
    subtitle->setObjectName(QStringLiteral("subtitle"));
    leftLayout->addWidget(title);
    leftLayout->addWidget(subtitle);
    leftLayout->addWidget(endpointBox);
    leftLayout->addWidget(nodeBox);
    leftLayout->addWidget(discoveryBox);
    leftLayout->addWidget(pairBox);
    leftLayout->addStretch(1);

    m_peersHeader = new QLabel(QStringLiteral("Peers (0)"), central);
    m_peersHeader->setObjectName(QStringLiteral("sectionTitle"));

    m_peerTable = new QTableWidget(0, 5, central);
    m_peerTable->setHorizontalHeaderLabels(
        {QStringLiteral("Paired"), QStringLiteral("Node ID"), QStringLiteral("Name"),
         QStringLiteral("Address"), QStringLiteral("Last seen")});
    m_peerTable->verticalHeader()->setVisible(false);
    m_peerTable->setSelectionBehavior(QAbstractItemView::SelectRows);
    m_peerTable->setSelectionMode(QAbstractItemView::SingleSelection);
    m_peerTable->setEditTriggers(QAbstractItemView::NoEditTriggers);
    m_peerTable->setAlternatingRowColors(true);
    m_peerTable->setShowGrid(false);
    m_peerTable->setWordWrap(false);
    m_peerTable->horizontalHeader()->setSectionResizeMode(0, QHeaderView::ResizeToContents);
    m_peerTable->horizontalHeader()->setSectionResizeMode(1, QHeaderView::ResizeToContents);
    m_peerTable->horizontalHeader()->setSectionResizeMode(2, QHeaderView::Stretch);
    m_peerTable->horizontalHeader()->setSectionResizeMode(3, QHeaderView::ResizeToContents);
    m_peerTable->horizontalHeader()->setSectionResizeMode(4, QHeaderView::ResizeToContents);

    auto *activityHeader = new QLabel(QStringLiteral("Activity"), central);
    activityHeader->setObjectName(QStringLiteral("sectionTitle"));

    m_log = new QPlainTextEdit(central);
    m_log->setReadOnly(true);
    m_log->setMaximumBlockCount(500);
    m_log->setFont(QFontDatabase::systemFont(QFontDatabase::FixedFont));

    auto *right = new QWidget(central);
    auto *rightLayout = new QVBoxLayout(right);
    rightLayout->setContentsMargins(0, 0, 0, 0);
    rightLayout->setSpacing(10);
    rightLayout->addWidget(m_peersHeader);
    rightLayout->addWidget(m_peerTable, 3);
    rightLayout->addWidget(activityHeader);
    rightLayout->addWidget(m_log, 2);

    auto *root = new QHBoxLayout(central);
    root->setContentsMargins(20, 16, 20, 16);
    root->setSpacing(20);
    root->addWidget(left);
    root->addWidget(right, 1);

    connect(m_connectButton, &QPushButton::clicked, this, &MainWindow::toggleConnection);
    connect(m_pingButton, &QPushButton::clicked, this, &MainWindow::ping);
    connect(m_newCodeButton, &QPushButton::clicked, this, &MainWindow::newPairCode);
    connect(m_joinButton, &QPushButton::clicked, this, &MainWindow::joinPair);
    connect(m_copyCodeButton, &QPushButton::clicked, this, &MainWindow::copyPairCode);
}

void MainWindow::applyStyle()
{
    setStyleSheet(QStringLiteral(R"(
QWidget#central { background: #f6f7fb; }
QLabel#title { font-size: 24px; font-weight: 700; color: #1f2430; }
QLabel#subtitle { color: #6b7280; margin-bottom: 4px; }
QLabel#sectionTitle { font-size: 15px; font-weight: 600; color: #1f2430; }
QLabel#caption { color: #6b7280; }
QLabel#value { color: #111827; }
QLabel#hint { color: #6b7280; font-size: 12px; }
QLabel#pairCode { font-family: Consolas, Menlo, monospace; font-size: 26px;
                  font-weight: 700; letter-spacing: 5px; color: #4338ca; }
QGroupBox { background: #ffffff; border: 1px solid #e5e7eb; border-radius: 10px;
            margin-top: 12px; padding: 14px 12px 12px 12px; font-weight: 600;
            color: #374151; }
QGroupBox::title { subcontrol-origin: margin; left: 12px; padding: 0 4px; }
QLineEdit, QSpinBox { background: #ffffff; border: 1px solid #d1d5db;
                      border-radius: 6px; padding: 6px 8px; color: #111827; }
QLineEdit:focus, QSpinBox:focus { border-color: #6366f1; }
QPushButton { background: #ffffff; border: 1px solid #d1d5db; border-radius: 6px;
              padding: 7px 12px; color: #1f2430; }
QPushButton:hover { background: #f3f4f6; }
QPushButton:disabled { color: #9ca3af; background: #f9fafb; }
QPushButton#primaryButton { background: #4f46e5; border-color: #4f46e5;
                            color: white; font-weight: 600; }
QPushButton#primaryButton:hover { background: #4338ca; }
QPushButton#primaryButton:disabled { background: #c7d2fe; border-color: #c7d2fe;
                                     color: #eef2ff; }
QTableWidget { background: #ffffff; border: 1px solid #e5e7eb; border-radius: 10px;
               alternate-background-color: #fafafa; selection-background-color: #eef2ff;
               selection-color: #111827; }
QHeaderView::section { background: #f9fafb; border: none;
                       border-bottom: 1px solid #e5e7eb; padding: 7px 8px;
                       color: #6b7280; font-weight: 600; }
QPlainTextEdit { background: #0f172a; color: #e2e8f0; border: 1px solid #1e293b;
                 border-radius: 10px; padding: 8px; font-size: 12px; }
QStatusBar { color: #6b7280; }
)"));
}

void MainWindow::toggleConnection()
{
    if (m_client->isConnected()) {
        m_client->disconnectFromDaemon();
        return;
    }
    m_connectButton->setEnabled(false);
    m_stateValue->setText(QStringLiteral("Connecting..."));
    statusBar()->showMessage(QStringLiteral("Connecting to %1:%2 ...")
                                 .arg(m_hostEdit->text().trimmed())
                                 .arg(m_portSpin->value()));
    m_client->connectToDaemon(m_hostEdit->text().trimmed(),
                              quint16(m_portSpin->value()));
}

void MainWindow::onConnectionStateChanged(bool connected)
{
    setConnectedUi(connected);
    if (connected) {
        m_refreshTimer->start();
        refresh();
    } else {
        m_refreshTimer->stop();
    }
}

void MainWindow::onStatus(const weft::Status &status)
{
    m_stateValue->setText(QStringLiteral("Connected"));
    m_nodeValue->setText(status.nodeId.isEmpty() ? QStringLiteral("-") : status.nodeId);
    m_nameValue->setText(status.displayName.isEmpty() ? QStringLiteral("-") : status.displayName);
    m_roleValue->setText(status.role.isEmpty() ? QStringLiteral("-") : status.role);
    m_versionValue->setText(status.version.isEmpty() ? QStringLiteral("-") : status.version);
    m_fingerprintValue->setText(status.fingerprint.isEmpty()
                                    ? QStringLiteral("-")
                                    : status.fingerprint);
    m_fingerprintValue->setToolTip(status.fingerprint);

    if (status.discoveryPort <= 0) {
        m_discoveryValue->setText(
            QStringLiteral("Discovery telemetry not reported by this daemon"));
    } else if (status.discoveryActive) {
        const auto last = status.lastBeaconMs > 0 ? formatAge(status.lastBeaconMs)
                                                  : QStringLiteral("-");
        m_discoveryValue->setText(QStringLiteral("UDP/%1 active · sent %2 · received %3 · last %4")
                                      .arg(status.discoveryPort)
                                      .arg(status.beaconsSent)
                                      .arg(status.beaconsReceived)
                                      .arg(last));
    } else {
        m_discoveryValue->setText(QStringLiteral("UDP/%1 inactive · check firewall or port")
                                      .arg(status.discoveryPort));
    }
    updatePeers(status.peers);
}

void MainWindow::updatePeers(const QList<weft::Peer> &peers)
{
    auto pairedCount = 0;
    QStringList state;
    for (const auto &peer : peers) {
        if (peer.paired)
            ++pairedCount;
        state << peer.peerId + (peer.paired ? QStringLiteral("*") : QString());
    }
    const auto stateString = state.join(QLatin1Char('|'));
    if (stateString != m_lastPeerState) {
        m_lastPeerState = stateString;
        appendLog(QStringLiteral("Peers: %1 (%2 paired)").arg(peers.size()).arg(pairedCount));
    }

    m_peersHeader->setText(QStringLiteral("Peers (%1)").arg(peers.size()));
    m_peerTable->setRowCount(int(peers.size()));
    for (int row = 0; row < int(peers.size()); ++row) {
        const auto &peer = peers.at(row);
        auto *paired = new QTableWidgetItem(peer.paired ? QStringLiteral("*")
                                                        : QStringLiteral("-"));
        paired->setTextAlignment(Qt::AlignCenter);
        if (peer.paired)
            paired->setForeground(QColor(QStringLiteral("#047857")));

        auto *id = new QTableWidgetItem(peer.peerId);
        id->setToolTip(peer.fingerprint);
        id->setFont(QFontDatabase::systemFont(QFontDatabase::FixedFont));

        auto *name = new QTableWidgetItem(peer.displayName.isEmpty()
                                              ? QStringLiteral("-")
                                              : peer.displayName);
        auto *address = new QTableWidgetItem(peer.address.isEmpty()
                                                 ? QStringLiteral("-")
                                                 : peer.address);
        auto *lastSeen = new QTableWidgetItem(formatAge(peer.lastSeen));

        m_peerTable->setItem(row, 0, paired);
        m_peerTable->setItem(row, 1, id);
        m_peerTable->setItem(row, 2, name);
        m_peerTable->setItem(row, 3, address);
        m_peerTable->setItem(row, 4, lastSeen);
    }
}

void MainWindow::onPong(qint64 rttMs)
{
    if (rttMs < 0) {
        appendLog(QStringLiteral("Pong received"));
        statusBar()->showMessage(QStringLiteral("Pong received"), 4000);
        return;
    }
    appendLog(QStringLiteral("Ping: %1 ms").arg(rttMs));
    statusBar()->showMessage(QStringLiteral("Ping: %1 ms").arg(rttMs), 4000);
}

void MainWindow::onPairCode(const QString &code, int expiresInSeconds)
{
    m_pairCode = code;
    m_pairCodeValue->setText(code);
    m_copyCodeButton->setEnabled(true);
    m_pairHint->setText(QStringLiteral("Single use · expires in %1s · enter it on the other device")
                            .arg(expiresInSeconds));
    appendLog(QStringLiteral("Pair code: %1 (expires in %2s)").arg(code).arg(expiresInSeconds));
    statusBar()->showMessage(QStringLiteral("Pair code ready: %1").arg(code), 6000);
}

void MainWindow::onPairResult(const weft::PairResult &result)
{
    if (result.ok) {
        m_pairHint->setText(QStringLiteral("Paired with %1").arg(result.peer.displayName));
        statusBar()->showMessage(QStringLiteral("Paired with %1").arg(result.peer.peerId), 6000);
        refresh();
    } else {
        m_pairHint->setText(QStringLiteral("Pairing failed"));
        QMessageBox::warning(this, QStringLiteral("Pairing failed"), result.error);
    }
}

void MainWindow::onError(const QString &message)
{
    statusBar()->showMessage(message, 8000);
    if (!m_client->isConnected())
        setConnectedUi(false);
}

void MainWindow::onLog(const QString &message)
{
    appendLog(message);
}

void MainWindow::ping()
{
    if (!m_client->isConnected()) {
        appendLog(QStringLiteral("Not connected"));
        return;
    }
    m_client->ping();
}

void MainWindow::newPairCode()
{
    if (!m_client->isConnected()) {
        appendLog(QStringLiteral("Not connected"));
        return;
    }
    m_client->requestPairCode();
}

void MainWindow::joinPair()
{
    if (!m_client->isConnected()) {
        appendLog(QStringLiteral("Not connected"));
        return;
    }
    JoinPairDialog dialog(this);
    if (dialog.exec() != QDialog::Accepted)
        return;
    m_client->joinPair(dialog.code(), dialog.peerHost());
    appendLog(QStringLiteral("Joining with code %1 ...").arg(dialog.code()));
}

void MainWindow::refresh()
{
    if (m_client->isConnected())
        m_client->requestStatus();
}

void MainWindow::copyPairCode()
{
    if (m_pairCode.isEmpty())
        return;
    QGuiApplication::clipboard()->setText(m_pairCode);
    statusBar()->showMessage(QStringLiteral("Pair code copied"), 3000);
}

void MainWindow::setConnectedUi(bool connected)
{
    m_connectButton->setEnabled(true);
    m_connectButton->setText(connected ? QStringLiteral("Disconnect")
                                       : QStringLiteral("Connect"));
    m_hostEdit->setEnabled(!connected);
    m_portSpin->setEnabled(!connected);
    m_pingButton->setEnabled(connected);
    m_newCodeButton->setEnabled(connected);
    m_joinButton->setEnabled(connected);
    m_copyCodeButton->setEnabled(connected && !m_pairCode.isEmpty());
    if (!connected)
        m_stateValue->setText(QStringLiteral("Disconnected"));
}

void MainWindow::clearStatus()
{
    m_stateValue->setText(QStringLiteral("Disconnected"));
    m_nodeValue->setText(QStringLiteral("-"));
    m_nameValue->setText(QStringLiteral("-"));
    m_roleValue->setText(QStringLiteral("-"));
    m_versionValue->setText(QStringLiteral("-"));
    m_fingerprintValue->setText(QStringLiteral("-"));
    m_discoveryValue->setText(QStringLiteral("Not connected"));
    m_pairCodeValue->setText(QStringLiteral("—"));
    m_pairHint->setText(QStringLiteral("No active code"));
    m_copyCodeButton->setEnabled(false);
    m_peersHeader->setText(QStringLiteral("Peers (0)"));
    m_peerTable->setRowCount(0);
    m_lastPeerState.clear();
}

void MainWindow::appendLog(const QString &message)
{
    const auto stamp = QDateTime::currentDateTime().toString(QStringLiteral("HH:mm:ss"));
    m_log->appendPlainText(QStringLiteral("[%1] %2").arg(stamp, message));
}

QString MainWindow::formatAge(qint64 millisecondsSinceEpoch) const
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

void MainWindow::closeEvent(QCloseEvent *event)
{
    m_refreshTimer->stop();
    m_client->disconnectFromDaemon();
    event->accept();
}
