#pragma once

#include <QMainWindow>
#include <QList>

#include "weftclient/weft_client.h"

class QCloseEvent;
class QLabel;
class QLineEdit;
class QPlainTextEdit;
class QPushButton;
class QSpinBox;
class QTableWidget;
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

private:
    void buildUi();
    void applyStyle();
    void setConnectedUi(bool connected);
    void updatePeers(const QList<weft::Peer> &peers);
    void clearStatus();
    void appendLog(const QString &message);
    QString formatAge(qint64 millisecondsSinceEpoch) const;

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
    QTimer *m_refreshTimer = nullptr;

    QString m_lastPeerState;
    QString m_pairCode;
};
