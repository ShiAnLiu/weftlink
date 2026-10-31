#include <QApplication>

#include "gui/main_window.h"

int main(int argc, char *argv[])
{
    QApplication app(argc, argv);
    QApplication::setApplicationName(QStringLiteral("Weftlink"));
    QApplication::setApplicationVersion(QStringLiteral(WEFTLINK_CLIENT_VERSION));
    QApplication::setOrganizationName(QStringLiteral("Weftlink"));
    QApplication::setStyle(QStringLiteral("Fusion"));

    MainWindow window;
    window.show();
    return app.exec();
}
