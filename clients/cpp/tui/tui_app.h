#pragma once

#include <QElapsedTimer>
#include <QString>
#include <QStringList>

#include "tui/terminal.h"
#include "weftclient/weft_client.h"

class TuiApp
{
public:
    TuiApp(QString host, quint16 port);
    int run();

private:
    enum class InputMode { Normal, JoinCode, JoinHost };

    void handleKey(const QString &key);
    void startJoin();
    void commitJoin();
    void refresh();
    void render();
    void log(const QString &message);
    QString formatAge(qint64 millisecondsSinceEpoch) const;
    int pairSecondsLeft() const;

    weft::Client m_client;
    Terminal m_term;
    QString m_host;
    quint16 m_port = 7801;

    bool m_running = true;
    bool m_dirty = true;
    bool m_connected = false;
    weft::Status m_status;
    QList<weft::Peer> m_peers;
    QStringList m_log;
    QString m_pairCode;
    int m_pairExpires = 0;
    QElapsedTimer m_pairTimer;

    InputMode m_mode = InputMode::Normal;
    QString m_input;
    QString m_joinCode;

    QElapsedTimer m_lastRefresh;
};
