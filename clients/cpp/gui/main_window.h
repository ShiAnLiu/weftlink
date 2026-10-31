#pragma once

#include <QMainWindow>
#include <QList>

#include "weftclient/weft_client.h"

class QCheckBox;
class QCloseEvent;
class QComboBox;
class QLabel;
class QLineEdit;
class QPlainTextEdit;
class QPushButton;
class QSpinBox;
class QTableWidget;
class QTabWidget;
class QTimer;

class MainWindow : public QMainWindow
{
    Q_OBJECT

public:
    explicit MainWindow(QWidget *parent = nullptr);
    ~MainWindow() override;

protected:
    void closeEvent(QCloseEvent *event) override;

private slots:
    void toggleConnection();
    void onConnectionStateChanged(bool connected);
    void onStatus(const weft::Status &status);
    void onPong(qint64 rttMs);
    void onPairCode(const QString &code, int expiresInSeconds);
    void onPairResult(const weft::PairResult &result);
    void onError(const QString &message);
    void onLog(const QString &message);
    void ping();
    void newPairCode();
    void joinPair();
    void refresh();
    void copyPairCode();

    void remoteStatus();
    void browseSendFile();
    void sendFile();
    void browseReceiveDir();
    void beginReceive();
    void cancelReceive();
    void onFileSent(const weft::SendFileResult &result);
    void onFileReceiveStarted(const QString &dir, int timeoutSeconds);
    void onFileReceived(const weft::ReceiveResult &result);
    void runCommand();
    void onRunResult(const QString &peer, const weft::RunResult &result);
    void refreshExecRequests();
    void onExecRequests(const QList<weft::ExecRequest> &requests);
    void approveSelectedExec();
    void unpairSelectedPeer();
    void denySelectedExec();
    void onExecAction(bool ok, const QString &error);
    void enableClipboard();
    void disableClipboard();
    void pushClipboard();
    void pullClipboard();
    void onClipboardState(const weft::ClipboardState &state);
    void onClipboardResult(const weft::ClipboardResult &result);

private:
    void buildUi();
    void applyStyle();
    void setConnectedUi(bool connected);
    void updatePeers(const QList<weft::Peer> &peers);
    void updatePeerCombo(const QList<weft::Peer> &peers);
    void clearStatus();
    void appendLog(const QString &message);
    void appendRunOutput(const QString &text);
    QString formatAge(qint64 millisecondsSinceEpoch) const;
    QString selectedPeerId() const;
    QString findWeftlinkCli() const;
    bool startDaemon();
    QString selectedPeerLabel() const;
    void setM2Enabled(bool enabled);
    void setFileStatus(const QString &text);

    weft::Client *m_client = nullptr;

    QLineEdit *m_hostEdit = nullptr;
    QSpinBox *m_portSpin = nullptr;
    QPushButton *m_connectButton = nullptr;
    QPushButton *m_pingButton = nullptr;
    QPushButton *m_newCodeButton = nullptr;
    QPushButton *m_joinButton = nullptr;
    QPushButton *m_copyCodeButton = nullptr;

    QLabel *m_stateValue = nullptr;
    QLabel *m_nodeValue = nullptr;
    QLabel *m_nameValue = nullptr;
    QLabel *m_roleValue = nullptr;
    QLabel *m_versionValue = nullptr;
    QLabel *m_fingerprintValue = nullptr;
    QLabel *m_discoveryValue = nullptr;
    QLabel *m_pairCodeValue = nullptr;
    QLabel *m_pairHint = nullptr;
    QLabel *m_peersHeader = nullptr;

    QTableWidget *m_peerTable = nullptr;
    QPlainTextEdit *m_log = nullptr;
    QTabWidget *m_tabs = nullptr;
    QTimer *m_refreshTimer = nullptr;

    QComboBox *m_peerCombo = nullptr;
    QLineEdit *m_sendPathEdit = nullptr;
    QLineEdit *m_recvDirEdit = nullptr;
    QCheckBox *m_recvAnyCheck = nullptr;
    QLabel *m_fileStatus = nullptr;
    QPushButton *m_sendFileButton = nullptr;
    QPushButton *m_recvButton = nullptr;
    QPushButton *m_recvCancelButton = nullptr;
    QPushButton *m_unpairButton = nullptr;
    QTimer *m_recvPollTimer = nullptr;
    QTimer *m_execPollTimer = nullptr;

    QLineEdit *m_runCommandEdit = nullptr;
    QComboBox *m_runShellCombo = nullptr;
    QSpinBox *m_runTimeoutSpin = nullptr;
    QPushButton *m_runButton = nullptr;
    QTableWidget *m_execTable = nullptr;
    QPushButton *m_approveButton = nullptr;
    QPushButton *m_denyButton = nullptr;
    QPushButton *m_execRefreshButton = nullptr;
    QPlainTextEdit *m_runOutput = nullptr;

    QPushButton *m_clipboardOnButton = nullptr;
    QPushButton *m_clipboardOffButton = nullptr;
    QPushButton *m_clipboardPushButton = nullptr;
    QPushButton *m_clipboardPullButton = nullptr;
    QSpinBox *m_clipboardTimeoutSpin = nullptr;
    QLabel *m_clipboardStatus = nullptr;

    QString m_lastPeerState;
    bool m_autoStartTried = false;
    QString m_pairCode;
};
