#include "tui/terminal.h"

#include <QStringList>

#include <cstdio>

#ifdef _WIN32
#  include <conio.h>
#  include <windows.h>
#  ifndef ENABLE_VIRTUAL_TERMINAL_PROCESSING
#    define ENABLE_VIRTUAL_TERMINAL_PROCESSING 0x0004
#  endif
#else
#  include <sys/ioctl.h>
#  include <sys/select.h>
#  include <termios.h>
#  include <unistd.h>
#endif

struct Terminal::Impl
{
    bool active = false;

#ifdef _WIN32
    HANDLE input = INVALID_HANDLE_VALUE;
    HANDLE output = INVALID_HANDLE_VALUE;
    DWORD oldInputMode = 0;
    DWORD oldOutputMode = 0;
#else
    termios oldMode {};
#endif
};

Terminal::Terminal()
    : m_impl(std::make_unique<Impl>())
{
}

Terminal::~Terminal()
{
    shutdown();
}

bool Terminal::init(QString *error)
{
    if (m_impl->active)
        return true;

#ifdef _WIN32
    m_impl->input = GetStdHandle(STD_INPUT_HANDLE);
    m_impl->output = GetStdHandle(STD_OUTPUT_HANDLE);
    if (m_impl->input == INVALID_HANDLE_VALUE || m_impl->output == INVALID_HANDLE_VALUE) {
        if (error)
            *error = QStringLiteral("no Windows console attached");
        return false;
    }

    DWORD outMode = 0;
    if (!GetConsoleMode(m_impl->output, &outMode)) {
        if (error)
            *error = QStringLiteral("stdout is not a console");
        return false;
    }
    m_impl->oldOutputMode = outMode;
    SetConsoleMode(m_impl->output,
                   outMode | ENABLE_PROCESSED_OUTPUT | ENABLE_VIRTUAL_TERMINAL_PROCESSING);

    DWORD inMode = 0;
    if (!GetConsoleMode(m_impl->input, &inMode)) {
        if (error)
            *error = QStringLiteral("stdin is not a console");
        return false;
    }
    m_impl->oldInputMode = inMode;
    const DWORD rawInput = (inMode
                            & ~(ENABLE_LINE_INPUT | ENABLE_ECHO_INPUT | ENABLE_PROCESSED_INPUT));
    SetConsoleMode(m_impl->input, rawInput | ENABLE_EXTENDED_FLAGS);
#else
    if (!isatty(STDIN_FILENO) || !isatty(STDOUT_FILENO)) {
        if (error)
            *error = QStringLiteral("an interactive terminal is required");
        return false;
    }
    if (tcgetattr(STDIN_FILENO, &m_impl->oldMode) != 0) {
        if (error)
            *error = QStringLiteral("cannot read terminal attributes");
        return false;
    }
    termios raw = m_impl->oldMode;
    raw.c_lflag &= ~(ICANON | ECHO);
    raw.c_iflag &= ~(IXON | ICRNL);
    raw.c_cc[VMIN] = 0;
    raw.c_cc[VTIME] = 0;
    tcsetattr(STDIN_FILENO, TCSANOW, &raw);
#endif

    m_impl->active = true;
    write(QStringLiteral("\x1b[?1049h\x1b[?25l"));
    flush();
    return true;
}

void Terminal::shutdown()
{
    if (!m_impl || !m_impl->active)
        return;
    write(QStringLiteral("\x1b[?25h\x1b[?1049l"));
    flush();

#ifdef _WIN32
    SetConsoleMode(m_impl->input, m_impl->oldInputMode);
    SetConsoleMode(m_impl->output, m_impl->oldOutputMode);
#else
    tcsetattr(STDIN_FILENO, TCSANOW, &m_impl->oldMode);
#endif
    m_impl->active = false;
}

bool Terminal::isActive() const
{
    return m_impl && m_impl->active;
}

QSize Terminal::size() const
{
#ifdef _WIN32
    CONSOLE_SCREEN_BUFFER_INFO info {};
    if (GetConsoleScreenBufferInfo(m_impl->output, &info)) {
        const auto width = int(info.srWindow.Right - info.srWindow.Left + 1);
        const auto height = int(info.srWindow.Bottom - info.srWindow.Top + 1);
        return QSize(qMax(width, 40), qMax(height, 12));
    }
#else
    winsize ws {};
    if (ioctl(STDOUT_FILENO, TIOCGWINSZ, &ws) == 0 && ws.ws_col > 0 && ws.ws_row > 0)
        return QSize(qMax(int(ws.ws_col), 40), qMax(int(ws.ws_row), 12));
#endif
    return QSize(100, 30);
}

QString Terminal::pollKey()
{
#ifdef _WIN32
    if (!_kbhit())
        return {};
    const int ch = _getch();
    if (ch == 0 || ch == 224) {
        const int second = _getch();
        switch (second) {
        case 72:
            return QStringLiteral("UP");
        case 80:
            return QStringLiteral("DOWN");
        case 75:
            return QStringLiteral("LEFT");
        case 77:
            return QStringLiteral("RIGHT");
        case 71:
            return QStringLiteral("HOME");
        case 79:
            return QStringLiteral("END");
        case 83:
            return QStringLiteral("DELETE");
        default:
            return QStringLiteral("KEY");
        }
    }
    if (ch == 3)
        return QStringLiteral("CTRL_C");
    if (ch == 13)
        return QStringLiteral("ENTER");
    if (ch == 27)
        return QStringLiteral("ESC");
    if (ch == 8 || ch == 127)
        return QStringLiteral("BACKSPACE");
    if (ch == 9)
        return QStringLiteral("TAB");
    if (ch >= 32 && ch < 127)
        return QString(QChar(ch));
    return {};
#else
    fd_set set;
    FD_ZERO(&set);
    FD_SET(STDIN_FILENO, &set);
    timeval timeout {0, 0};
    if (select(STDIN_FILENO + 1, &set, nullptr, nullptr, &timeout) <= 0)
        return {};

    unsigned char buffer[16] = {};
    const auto count = read(STDIN_FILENO, buffer, sizeof(buffer));
    if (count <= 0)
        return {};

    if (buffer[0] == 27) {
        if (count >= 3 && buffer[1] == '[') {
            switch (buffer[2]) {
            case 'A':
                return QStringLiteral("UP");
            case 'B':
                return QStringLiteral("DOWN");
            case 'C':
                return QStringLiteral("RIGHT");
            case 'D':
                return QStringLiteral("LEFT");
            case 'H':
                return QStringLiteral("HOME");
            case 'F':
                return QStringLiteral("END");
            default:
                return QStringLiteral("KEY");
            }
        }
        return QStringLiteral("ESC");
    }
    if (buffer[0] == 3)
        return QStringLiteral("CTRL_C");
    if (buffer[0] == 13 || buffer[0] == 10)
        return QStringLiteral("ENTER");
    if (buffer[0] == 8 || buffer[0] == 127)
        return QStringLiteral("BACKSPACE");
    if (buffer[0] == 9)
        return QStringLiteral("TAB");
    if (buffer[0] >= 32)
        return QString::fromUtf8(reinterpret_cast<const char *>(buffer), int(count));
    return {};
#endif
}

void Terminal::write(const QString &text)
{
    const auto utf8 = text.toUtf8();
    std::fwrite(utf8.constData(), 1, size_t(utf8.size()), stdout);
}

void Terminal::flush()
{
    std::fflush(stdout);
}
