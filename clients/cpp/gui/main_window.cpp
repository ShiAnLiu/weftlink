#include "gui/main_window.h"

#include "gui/pair_dialog.h"

#include <QCheckBox>
#include <QClipboard>
#include <QCloseEvent>
#include <QColor>
#include <QComboBox>
#include <QDateTime>
#include <QDir>
#include <QFileDialog>
#include <QFileInfo>
#include <QFontDatabase>
#include <QFormLayout>
#include <QFrame>
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
#include <QScrollArea>
#include <QScrollBar>
#include <QSpinBox>
#include <QSignalBlocker>
#include <QStatusBar>
#include <QStringList>
#include <QTabWidget>
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
    resize(1220, 860);

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
    connect(m_client, &weft::Client::peerStatusReceived, this,
            [this](const QString &peer, const weft::Status &status) {
                appendLog(QStringLiteral("Remote %1: %2 v%3, %4 peer(s), %5 paired")
                              .arg(peer, status.displayName, status.version)
                              .arg(status.peerCount)
                              .arg(status.pairedCount));
            });
    connect(m_client, &weft::Client::fileReceiveStarted,
            this, &MainWindow::onFileReceiveStarted);
    connect(m_client, &weft::Client::fileReceived, this, &MainWindow::onFileReceived);
    connect(m_client, &weft::Client::execRequestsReceived,
            this, &MainWindow::onExecRequests);
    connect(m_client, &weft::Client::execActionFinished,
            this, &MainWindow::onExecAction);
    connect(m_client, &weft::Client::clipboardStateReceived,
            this, &MainWindow::onClipboardState);
    connect(m_client, &weft::Client::clipboardResultReceived,
            this, &MainWindow::onClipboardResult);

    m_recvPollTimer = new QTimer(this);
    m_recvPollTimer->setInterval(1000);
    connect(m_recvPollTimer, &QTimer::timeout, this, [this] {
        if (m_client->isConnected())
            m_client->pollReceive();
    });
    m_execPollTimer = new QTimer(this);
    m_execPollTimer->setInterval(2500);
    connect(m_execPollTimer, &QTimer::timeout, this, [this] {
        if (m_client->isConnected())
            m_client->requestExecRequests();
    });

    if (qEnvironmentVariableIntValue("WEFTLINK_M2_TAB") != 0)
        m_tabs->setCurrentIndex(1);
    if (qEnvironmentVariableIntValue("WEFTLINK_AUTOCONNECT") != 0)
        QTimer::singleShot(300, this, &MainWindow::toggleConnection);

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

    m_pairCodeValue = new QLabel(QStringLiteral("-"), central);
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
    left->setFixedWidth(360);
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

    m_log = new QPlainTextEdit(central);
    m_log->setReadOnly(true);
    m_log->setMaximumBlockCount(500);
    m_log->setFont(QFontDatabase::systemFont(QFontDatabase::FixedFont));

    auto *m2Page = new QWidget(central);
    auto *m2Layout = new QVBoxLayout(m2Page);
    m2Layout->setContentsMargins(4, 4, 4, 4);
    m2Layout->setSpacing(10);

    auto *peerBox = new QGroupBox(QStringLiteral("Peer"), m2Page);
    auto *peerLayout = new QHBoxLayout(peerBox);
    m_peerCombo = new QComboBox(peerBox);
    m_peerCombo->setMinimumContentsLength(26);
    m_peerCombo->setSizeAdjustPolicy(QComboBox::AdjustToMinimumContentsLengthWithIcon);
    auto *remoteStatusButton = new QPushButton(QStringLiteral("Remote status"), peerBox);
    peerLayout->addWidget(new QLabel(QStringLiteral("Target"), peerBox));
    peerLayout->addWidget(m_peerCombo, 1);
    peerLayout->addWidget(remoteStatusButton);
    m2Layout->addWidget(peerBox);

    auto *filesBox = new QGroupBox(QStringLiteral("Files"), m2Page);
    auto *filesLayout = new QGridLayout(filesBox);
    filesLayout->setColumnStretch(1, 1);
    m_sendPathEdit = new QLineEdit(filesBox);
    m_sendPathEdit->setPlaceholderText(QStringLiteral("C:\\path\\to\\file.bin"));
    auto *browseSendButton = new QPushButton(QStringLiteral("Browse..."), filesBox);
    m_sendFileButton = new QPushButton(QStringLiteral("Send file"), filesBox);
    m_sendFileButton->setObjectName(QStringLiteral("primaryButton"));
    m_recvDirEdit = new QLineEdit(filesBox);
    m_recvDirEdit->setText(QDir::homePath() + QStringLiteral("/Downloads/Weftlink"));
    auto *browseRecvButton = new QPushButton(QStringLiteral("Browse..."), filesBox);
    m_recvButton = new QPushButton(QStringLiteral("Receive one file"), filesBox);
    m_recvButton->setObjectName(QStringLiteral("primaryButton"));
    m_recvCancelButton = new QPushButton(QStringLiteral("Cancel"), filesBox);
    m_recvAnyCheck = new QCheckBox(QStringLiteral("Accept from any paired peer"), filesBox);
    m_recvAnyCheck->setChecked(true);
    m_fileStatus = new QLabel(QStringLiteral("Idle"), filesBox);
    m_fileStatus->setObjectName(QStringLiteral("value"));
    m_fileStatus->setWordWrap(true);
    filesLayout->addWidget(new QLabel(QStringLiteral("Send"), filesBox), 0, 0);
    filesLayout->addWidget(m_sendPathEdit, 0, 1);
    filesLayout->addWidget(browseSendButton, 0, 2);
    filesLayout->addWidget(m_sendFileButton, 0, 3);
    filesLayout->addWidget(new QLabel(QStringLiteral("Receive to"), filesBox), 1, 0);
    filesLayout->addWidget(m_recvDirEdit, 1, 1);
    filesLayout->addWidget(browseRecvButton, 1, 2);
    filesLayout->addWidget(m_recvButton, 1, 3);
    filesLayout->addWidget(m_recvAnyCheck, 2, 0, 1, 3);
    filesLayout->addWidget(m_recvCancelButton, 2, 3);
    filesLayout->addWidget(m_fileStatus, 3, 0, 1, 4);
    m2Layout->addWidget(filesBox);

    auto *runBox = new QGroupBox(QStringLiteral("Run command (requires approval)"), m2Page);
    auto *runLayout = new QVBoxLayout(runBox);
    auto *runRow = new QHBoxLayout;
    m_runCommandEdit = new QLineEdit(runBox);
    m_runCommandEdit->setPlaceholderText(QStringLiteral("e.g. ipconfig  or  Get-Location"));
    m_runShellCombo = new QComboBox(runBox);
    m_runShellCombo->addItems({QStringLiteral("auto"), QStringLiteral("powershell"),
                               QStringLiteral("cmd"), QStringLiteral("sh")});
    m_runTimeoutSpin = new QSpinBox(runBox);
    m_runTimeoutSpin->setRange(1, 300);
    m_runTimeoutSpin->setValue(30);
    m_runTimeoutSpin->setSuffix(QStringLiteral(" s"));
    m_runButton = new QPushButton(QStringLiteral("Request run"), runBox);
    m_runButton->setObjectName(QStringLiteral("primaryButton"));
    runRow->addWidget(new QLabel(QStringLiteral("Command"), runBox));
    runRow->addWidget(m_runCommandEdit, 1);
    runRow->addWidget(m_runShellCombo);
    runRow->addWidget(m_runTimeoutSpin);
    runRow->addWidget(m_runButton);
    runLayout->addLayout(runRow);

    m_execTable = new QTableWidget(0, 4, runBox);
    m_execTable->setHorizontalHeaderLabels(
        {QStringLiteral("ID"), QStringLiteral("Peer"), QStringLiteral("Shell"),
         QStringLiteral("Command")});
    m_execTable->verticalHeader()->setVisible(false);
    m_execTable->setSelectionBehavior(QAbstractItemView::SelectRows);
    m_execTable->setSelectionMode(QAbstractItemView::SingleSelection);
    m_execTable->setEditTriggers(QAbstractItemView::NoEditTriggers);
    m_execTable->setShowGrid(false);
    m_execTable->setMinimumHeight(110);
    m_execTable->horizontalHeader()->setSectionResizeMode(0, QHeaderView::ResizeToContents);
    m_execTable->horizontalHeader()->setSectionResizeMode(1, QHeaderView::ResizeToContents);
    m_execTable->horizontalHeader()->setSectionResizeMode(2, QHeaderView::ResizeToContents);
    m_execTable->horizontalHeader()->setSectionResizeMode(3, QHeaderView::Stretch);
    runLayout->addWidget(m_execTable);

    auto *execButtons = new QHBoxLayout;
    m_approveButton = new QPushButton(QStringLiteral("Approve"), runBox);
    m_denyButton = new QPushButton(QStringLiteral("Deny"), runBox);
    m_execRefreshButton = new QPushButton(QStringLiteral("Refresh"), runBox);
    execButtons->addWidget(m_approveButton);
    execButtons->addWidget(m_denyButton);
    execButtons->addWidget(m_execRefreshButton);
    execButtons->addStretch(1);
    runLayout->addLayout(execButtons);

    m_runOutput = new QPlainTextEdit(runBox);
    m_runOutput->setReadOnly(true);
    m_runOutput->setMaximumBlockCount(400);
    m_runOutput->setMinimumHeight(120);
    m_runOutput->setFont(QFontDatabase::systemFont(QFontDatabase::FixedFont));
    m_runOutput->setPlaceholderText(QStringLiteral("Remote stdout/stderr appears here"));
    runLayout->addWidget(m_runOutput);
    m2Layout->addWidget(runBox);

    auto *clipboardBox = new QGroupBox(QStringLiteral("Clipboard"), m2Page);
    auto *clipboardLayout = new QVBoxLayout(clipboardBox);
    auto *clipboardRow = new QHBoxLayout;
    m_clipboardOnButton = new QPushButton(QStringLiteral("Enable receive"), clipboardBox);
    m_clipboardTimeoutSpin = new QSpinBox(clipboardBox);
    m_clipboardTimeoutSpin->setRange(30, 3600);
    m_clipboardTimeoutSpin->setValue(600);
    m_clipboardTimeoutSpin->setSuffix(QStringLiteral(" s"));
    m_clipboardOffButton = new QPushButton(QStringLiteral("Disable"), clipboardBox);
    m_clipboardStatus = new QLabel(QStringLiteral("off"), clipboardBox);
    m_clipboardStatus->setObjectName(QStringLiteral("value"));
    clipboardRow->addWidget(m_clipboardOnButton);
    clipboardRow->addWidget(m_clipboardTimeoutSpin);
    clipboardRow->addWidget(m_clipboardOffButton);
    clipboardRow->addWidget(new QLabel(QStringLiteral("Status:"), clipboardBox));
    clipboardRow->addWidget(m_clipboardStatus, 1);
    clipboardLayout->addLayout(clipboardRow);
    auto *clipboardActions = new QHBoxLayout;
    m_clipboardPushButton = new QPushButton(QStringLiteral("Push selected peer"), clipboardBox);
    m_clipboardPullButton = new QPushButton(QStringLiteral("Pull selected peer"), clipboardBox);
    clipboardActions->addWidget(m_clipboardPushButton);
    clipboardActions->addWidget(m_clipboardPullButton);
    clipboardActions->addStretch(1);
    clipboardLayout->addLayout(clipboardActions);
    m2Layout->addWidget(clipboardBox);
    m2Layout->addStretch(1);

    auto *m2Scroll = new QScrollArea(central);
    m2Scroll->setWidgetResizable(true);
    m2Scroll->setFrameShape(QFrame::NoFrame);
    m2Scroll->setWidget(m2Page);

    m_tabs = new QTabWidget(central);
    m_tabs->addTab(m_log, QStringLiteral("Activity"));
    m_tabs->addTab(m2Scroll, QStringLiteral("M2"));

    auto *right = new QWidget(central);
    auto *rightLayout = new QVBoxLayout(right);
    rightLayout->setContentsMargins(0, 0, 0, 0);
    rightLayout->setSpacing(10);
    rightLayout->addWidget(m_peersHeader);
    rightLayout->addWidget(m_peerTable, 2);
    rightLayout->addWidget(m_tabs, 3);

    auto *root = new QHBoxLayout(central);
    root->setContentsMargins(20, 16, 20, 16);
    root->setSpacing(20);
    auto *leftScroll = new QScrollArea(central);
    leftScroll->setWidgetResizable(true);
    leftScroll->setFrameShape(QFrame::NoFrame);
    leftScroll->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    leftScroll->setWidget(left);
    leftScroll->setFixedWidth(380);
    root->addWidget(leftScroll);
    root->addWidget(right, 1);

    connect(m_connectButton, &QPushButton::clicked, this, &MainWindow::toggleConnection);
    connect(m_pingButton, &QPushButton::clicked, this, &MainWindow::ping);
    connect(m_newCodeButton, &QPushButton::clicked, this, &MainWindow::newPairCode);
    connect(m_joinButton, &QPushButton::clicked, this, &MainWindow::joinPair);
    connect(m_copyCodeButton, &QPushButton::clicked, this, &MainWindow::copyPairCode);
    connect(m_peerCombo, QOverload<int>::of(&QComboBox::currentIndexChanged), this, [this] { setM2Enabled(m_client->isConnected()); });
    connect(remoteStatusButton, &QPushButton::clicked, this, &MainWindow::remoteStatus);
    connect(browseSendButton, &QPushButton::clicked, this, &MainWindow::browseSendFile);
    connect(browseRecvButton, &QPushButton::clicked, this, &MainWindow::browseReceiveDir);
    connect(m_sendFileButton, &QPushButton::clicked, this, &MainWindow::sendFile);
    connect(m_recvButton, &QPushButton::clicked, this, &MainWindow::beginReceive);
    connect(m_recvCancelButton, &QPushButton::clicked, this, &MainWindow::cancelReceive);
    connect(m_runButton, &QPushButton::clicked, this, &MainWindow::runCommand);
    connect(m_execRefreshButton, &QPushButton::clicked, this, &MainWindow::refreshExecRequests);
    connect(m_approveButton, &QPushButton::clicked, this, &MainWindow::approveSelectedExec);
    connect(m_denyButton, &QPushButton::clicked, this, &MainWindow::denySelectedExec);
    connect(m_clipboardOnButton, &QPushButton::clicked, this, &MainWindow::enableClipboard);
    connect(m_clipboardOffButton, &QPushButton::clicked, this, &MainWindow::disableClipboard);
    connect(m_clipboardPushButton, &QPushButton::clicked, this, &MainWindow::pushClipboard);
    connect(m_clipboardPullButton, &QPushButton::clicked, this, &MainWindow::pullClipboard);
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
        m_execPollTimer->start();
        refresh();
        m_client->requestExecRequests();
        m_client->requestClipboardState();
    } else {
        m_refreshTimer->stop();
        m_execPollTimer->stop();
        m_recvPollTimer->stop();
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
        m_discoveryValue->setText(QStringLiteral("UDP/%1 active | sent %2 | received %3 | last %4")
                                      .arg(status.discoveryPort)
                                      .arg(status.beaconsSent)
                                      .arg(status.beaconsReceived)
                                      .arg(last));
    } else {
        m_discoveryValue->setText(QStringLiteral("UDP/%1 inactive - check firewall or port")
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
    updatePeerCombo(peers);
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
    m_pairHint->setText(QStringLiteral("Single use - expires in %1s - enter it on the other device")
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


QString MainWindow::selectedPeerId() const
{
    if (!m_peerCombo)
        return {};
    return m_peerCombo->currentData().toString();
}

QString MainWindow::selectedPeerLabel() const
{
    return m_peerCombo ? m_peerCombo->currentText() : QString();
}

void MainWindow::updatePeerCombo(const QList<weft::Peer> &peers)
{
    if (!m_peerCombo)
        return;
    const auto current = m_peerCombo->currentData().toString();
    QSignalBlocker blocker(m_peerCombo);
    m_peerCombo->clear();
    for (const auto &peer : peers) {
        if (!peer.paired)
            continue;
        const auto name = peer.displayName.isEmpty() ? peer.peerId : peer.displayName;
        m_peerCombo->addItem(QStringLiteral("%1 (%2)").arg(name, peer.peerId), peer.peerId);
    }
    if (m_peerCombo->count() == 0)
        m_peerCombo->addItem(QStringLiteral("(no paired peers)"), QString());
    const auto index = m_peerCombo->findData(current);
    if (index >= 0)
        m_peerCombo->setCurrentIndex(index);
    setM2Enabled(m_client->isConnected());
}

void MainWindow::setM2Enabled(bool enabled)
{
    const auto hasPeer = !selectedPeerId().isEmpty();
    m_peerCombo->setEnabled(enabled);
    m_sendFileButton->setEnabled(enabled && hasPeer);
    m_recvButton->setEnabled(enabled);
    m_runButton->setEnabled(enabled && hasPeer);
    m_clipboardOnButton->setEnabled(enabled);
    m_clipboardOffButton->setEnabled(enabled);
    m_clipboardPushButton->setEnabled(enabled && hasPeer);
    m_clipboardPullButton->setEnabled(enabled && hasPeer);
    m_execRefreshButton->setEnabled(enabled);
    m_approveButton->setEnabled(enabled && m_execTable->rowCount() > 0);
    m_denyButton->setEnabled(enabled && m_execTable->rowCount() > 0);
}

void MainWindow::setFileStatus(const QString &text)
{
    if (m_fileStatus)
        m_fileStatus->setText(text);
}

void MainWindow::remoteStatus()
{
    if (!m_client->isConnected())
        return;
    const auto peer = selectedPeerId();
    if (peer.isEmpty()) {
        appendLog(QStringLiteral("Select a paired peer first"));
        return;
    }
    m_client->requestPeerStatus(peer);
    statusBar()->showMessage(QStringLiteral("Requesting remote status..."), 3000);
}

void MainWindow::browseSendFile()
{
    const auto path = QFileDialog::getOpenFileName(this, QStringLiteral("Choose a file to send"));
    if (!path.isEmpty())
        m_sendPathEdit->setText(path);
}

void MainWindow::sendFile()
{
    if (!m_client->isConnected())
        return;
    const auto peer = selectedPeerId();
    const auto path = m_sendPathEdit->text().trimmed();
    if (peer.isEmpty() || path.isEmpty()) {
        setFileStatus(QStringLiteral("Choose a paired peer and a file"));
        return;
    }
    setFileStatus(QStringLiteral("Sending %1 ...").arg(QFileInfo(path).fileName()));
    auto *client = new weft::Client(this);
    connect(client, &weft::Client::connectionStateChanged, this,
            [client, peer, path](bool connected) {
                if (connected)
                    client->sendFile(peer, path);
            });
    connect(client, &weft::Client::fileSent, this,
            [this, client](const weft::SendFileResult &result) {
                onFileSent(result);
                client->deleteLater();
            });
    connect(client, &weft::Client::errorOccurred, this,
            [this, client](const QString &message) {
                setFileStatus(QStringLiteral("Send failed: %1").arg(message));
                appendLog(QStringLiteral("Send failed: %1").arg(message));
                client->deleteLater();
            });
    connect(client, &weft::Client::logMessage, this, &MainWindow::onLog);
    client->connectToDaemon(m_client->host(), m_client->port());
}

void MainWindow::onFileSent(const weft::SendFileResult &result)
{
    if (result.ok) {
        setFileStatus(QStringLiteral("Sent %1 bytes | sha256 %2")
                          .arg(result.size)
                          .arg(result.checksum));
    } else {
        setFileStatus(QStringLiteral("Send failed: %1").arg(result.error));
    }
}

void MainWindow::browseReceiveDir()
{
    const auto dir = QFileDialog::getExistingDirectory(this, QStringLiteral("Choose receive directory"));
    if (!dir.isEmpty())
        m_recvDirEdit->setText(dir);
}

void MainWindow::beginReceive()
{
    if (!m_client->isConnected())
        return;
    auto from = QString();
    if (!m_recvAnyCheck->isChecked()) {
        from = selectedPeerId();
        if (from.isEmpty()) {
            setFileStatus(QStringLiteral("Select a peer or enable 'any paired peer'"));
            return;
        }
    }
    const auto dir = m_recvDirEdit->text().trimmed();
    if (dir.isEmpty()) {
        setFileStatus(QStringLiteral("Choose a receive directory"));
        return;
    }
    setFileStatus(QStringLiteral("Opening receive window..."));
    m_client->beginReceive(dir, from, 300);
}

void MainWindow::onFileReceiveStarted(const QString &dir, int timeoutSeconds)
{
    Q_UNUSED(timeoutSeconds);
    setFileStatus(QStringLiteral("Waiting for a file in %1 ...").arg(dir));
    m_recvPollTimer->start();
}

void MainWindow::onFileReceived(const weft::ReceiveResult &result)
{
    m_recvPollTimer->stop();
    if (result.ok) {
        setFileStatus(QStringLiteral("Received %1 (%2 bytes) -> %3")
                          .arg(result.name)
                          .arg(result.size)
                          .arg(result.path));
        appendLog(QStringLiteral("File received: %1").arg(result.path));
    } else {
        setFileStatus(QStringLiteral("Receive failed: %1").arg(result.error));
        appendLog(QStringLiteral("Receive failed: %1").arg(result.error));
        QMessageBox::warning(this, QStringLiteral("Receive failed"), result.error);
    }
}

void MainWindow::cancelReceive()
{
    if (m_client->isConnected())
        m_client->cancelReceive();
    m_recvPollTimer->stop();
    setFileStatus(QStringLiteral("Receive cancelled"));
}

void MainWindow::runCommand()
{
    if (!m_client->isConnected())
        return;
    const auto peer = selectedPeerId();
    const auto command = m_runCommandEdit->text().trimmed();
    if (peer.isEmpty() || command.isEmpty()) {
        appendRunOutput(QStringLiteral("[UI] Choose a paired peer and enter a command."));
        return;
    }
    const auto shell = m_runShellCombo->currentText();
    appendRunOutput(QStringLiteral("[UI] Waiting for %1 to approve: %2").arg(peer, command));
    auto *client = new weft::Client(this);
    connect(client, &weft::Client::connectionStateChanged, this,
            [client, peer, command, shell, this](bool connected) {
                if (connected)
                    client->runCommand(peer, command, m_runTimeoutSpin->value(), shell);
            });
    connect(client, &weft::Client::runResultReceived, this,
            [this, client](const QString &peerName, const weft::RunResult &result) {
                onRunResult(peerName, result);
                client->deleteLater();
            });
    connect(client, &weft::Client::errorOccurred, this,
            [this, client](const QString &message) {
                appendRunOutput(QStringLiteral("[UI] Run failed: %1").arg(message));
                client->deleteLater();
            });
    connect(client, &weft::Client::logMessage, this, &MainWindow::onLog);
    client->connectToDaemon(m_client->host(), m_client->port());
}

void MainWindow::onRunResult(const QString &peer, const weft::RunResult &result)
{
    appendRunOutput(QStringLiteral("=== %1 | exit code %2 ===").arg(peer).arg(result.exitCode));
    if (!result.stdoutText.isEmpty())
        appendRunOutput(result.stdoutText);
    if (!result.stderrText.isEmpty()) {
        appendRunOutput(QStringLiteral("--- stderr ---"));
        appendRunOutput(result.stderrText);
    }
    if (!result.ok)
        appendRunOutput(QStringLiteral("[error] %1").arg(result.error));
}

void MainWindow::appendRunOutput(const QString &text)
{
    if (!m_runOutput)
        return;
    m_runOutput->appendPlainText(text);
    m_runOutput->verticalScrollBar()->setValue(m_runOutput->verticalScrollBar()->maximum());
}

void MainWindow::refreshExecRequests()
{
    if (m_client->isConnected())
        m_client->requestExecRequests();
}

void MainWindow::onExecRequests(const QList<weft::ExecRequest> &requests)
{
    const auto currentId = [this]() {
        const auto row = m_execTable->currentRow();
        if (row < 0)
            return QString();
        const auto *item = m_execTable->item(row, 0);
        return item ? item->data(Qt::UserRole).toString() : QString();
    }();
    m_execTable->setRowCount(int(requests.size()));
    for (int row = 0; row < int(requests.size()); ++row) {
        const auto &request = requests.at(row);
        auto *id = new QTableWidgetItem(request.execId);
        id->setData(Qt::UserRole, request.execId);
        auto *peer = new QTableWidgetItem(request.peerId);
        auto *shell = new QTableWidgetItem(request.shell);
        auto *command = new QTableWidgetItem(request.command);
        m_execTable->setItem(row, 0, id);
        m_execTable->setItem(row, 1, peer);
        m_execTable->setItem(row, 2, shell);
        m_execTable->setItem(row, 3, command);
        if (request.execId == currentId)
            m_execTable->selectRow(row);
    }
    setM2Enabled(m_client->isConnected());
}

void MainWindow::approveSelectedExec()
{
    const auto row = m_execTable->currentRow();
    if (row < 0)
        return;
    const auto *item = m_execTable->item(row, 0);
    if (!item)
        return;
    const auto execId = item->data(Qt::UserRole).toString();
    m_client->approveExec(execId);
    appendRunOutput(QStringLiteral("[UI] Approved %1").arg(execId));
    QTimer::singleShot(500, this, &MainWindow::refreshExecRequests);
}

void MainWindow::denySelectedExec()
{
    const auto row = m_execTable->currentRow();
    if (row < 0)
        return;
    const auto *item = m_execTable->item(row, 0);
    if (!item)
        return;
    const auto execId = item->data(Qt::UserRole).toString();
    m_client->denyExec(execId);
    appendRunOutput(QStringLiteral("[UI] Denied %1").arg(execId));
    QTimer::singleShot(500, this, &MainWindow::refreshExecRequests);
}

void MainWindow::onExecAction(bool ok, const QString &error)
{
    if (!ok)
        appendRunOutput(QStringLiteral("[UI] Exec action failed: %1").arg(error));
}

void MainWindow::enableClipboard()
{
    if (!m_client->isConnected())
        return;
    m_client->clipboardOn(selectedPeerId(), m_clipboardTimeoutSpin->value());
}

void MainWindow::disableClipboard()
{
    if (m_client->isConnected())
        m_client->clipboardOff();
}

void MainWindow::pushClipboard()
{
    const auto peer = selectedPeerId();
    if (peer.isEmpty())
        return;
    m_client->clipboardPush(peer);
}

void MainWindow::pullClipboard()
{
    const auto peer = selectedPeerId();
    if (peer.isEmpty())
        return;
    m_client->clipboardPull(peer);
}

void MainWindow::onClipboardState(const weft::ClipboardState &state)
{
    if (!state.active) {
        m_clipboardStatus->setText(QStringLiteral("off"));
        return;
    }
    const auto from = state.from.isEmpty() ? QStringLiteral("any paired peer") : state.from;
    m_clipboardStatus->setText(QStringLiteral("on | %1 | %2s left")
                                   .arg(from)
                                   .arg(state.expiresInSeconds));
}

void MainWindow::onClipboardResult(const weft::ClipboardResult &result)
{
    if (result.ok)
        appendLog(QStringLiteral("Clipboard transferred (%1 chars)").arg(result.length));
    else
        appendLog(QStringLiteral("Clipboard failed: %1").arg(result.error));
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
    setM2Enabled(connected);
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
    m_pairCodeValue->setText(QStringLiteral("-"));
    m_pairHint->setText(QStringLiteral("No active code"));
    m_copyCodeButton->setEnabled(false);
    m_peersHeader->setText(QStringLiteral("Peers (0)"));
    m_peerTable->setRowCount(0);
    m_lastPeerState.clear();
    if (m_execTable)
        m_execTable->setRowCount(0);
    if (m_fileStatus)
        m_fileStatus->setText(QStringLiteral("Idle"));
    if (m_clipboardStatus)
        m_clipboardStatus->setText(QStringLiteral("off"));
    if (m_runOutput)
        m_runOutput->clear();
    if (m_peerCombo)
        updatePeerCombo({});
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
