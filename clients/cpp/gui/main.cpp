#include <QApplication>
#include <QDateTime>
#include <QDebug>
#include <QDir>
#include <QFile>
#include <QMutex>
#include <QMutexLocker>
#include <QTextStream>

#include <cstdlib>

#ifdef Q_OS_WIN
#include <windows.h>
#endif

#include "gui/main_window.h"

namespace {

QFile *g_logFile = nullptr;
QMutex g_logMutex;

QString logDirectory()
{
    auto dir = qEnvironmentVariable("WEFTLINK_HOME");
    if (dir.trimmed().isEmpty())
        dir = QDir::homePath() + QStringLiteral("/.weftlink");
    QDir().mkpath(dir);
    return dir;
}

void guiMessageHandler(QtMsgType type, const QMessageLogContext &, const QString &message)
{
    QMutexLocker locker(&g_logMutex);
    if (!g_logFile)
        return;
    const char *level = "INFO";
    switch (type) {
    case QtDebugMsg:
        level = "DEBUG";
        break;
    case QtInfoMsg:
        level = "INFO";
        break;
    case QtWarningMsg:
        level = "WARN";
        break;
    case QtCriticalMsg:
        level = "ERROR";
        break;
    case QtFatalMsg:
        level = "FATAL";
        break;
    }
    QTextStream out(g_logFile);
    out << QDateTime::currentDateTime().toString(Qt::ISODate) << " [" << level << "] "
        << message << '\n';
    g_logFile->flush();
    if (type == QtFatalMsg)
        std::abort();
}

#ifdef Q_OS_WIN
LONG WINAPI guiCrashFilter(EXCEPTION_POINTERS *info)
{
    if (info && info->ExceptionRecord) {
        qCritical("unhandled exception code=0x%08lx address=%p",
                  static_cast<unsigned long>(info->ExceptionRecord->ExceptionCode),
                  info->ExceptionRecord->ExceptionAddress);
    }
    return EXCEPTION_CONTINUE_SEARCH;
}
#endif

} // namespace

int main(int argc, char *argv[])
{
    auto logPath = logDirectory() + QStringLiteral("/weftlink-gui.log");
    g_logFile = new QFile(logPath);
    if (!g_logFile->open(QIODevice::WriteOnly | QIODevice::Append | QIODevice::Text)) {
        delete g_logFile;
        g_logFile = nullptr;
    }
    qInstallMessageHandler(guiMessageHandler);

#ifdef Q_OS_WIN
    SetUnhandledExceptionFilter(guiCrashFilter);
#endif

    QApplication app(argc, argv);
    QApplication::setApplicationName(QStringLiteral("Weftlink"));
    QApplication::setApplicationVersion(QStringLiteral(WEFTLINK_CLIENT_VERSION));
    QApplication::setOrganizationName(QStringLiteral("Weftlink"));
    QApplication::setStyle(QStringLiteral("Fusion"));

    qInfo() << "GUI starting" << WEFTLINK_CLIENT_VERSION << "log" << logPath;
    MainWindow window;
    window.show();
    auto code = app.exec();
    qInfo() << "GUI exiting" << code;
    if (g_logFile) {
        g_logFile->close();
        delete g_logFile;
        g_logFile = nullptr;
    }
    return code;
}
