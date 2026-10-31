#pragma once

#include <QSize>
#include <QString>

#include <memory>

// Terminal owns the console in raw/alternate-screen mode and yields simple key
// tokens (printable characters, ENTER, ESC, UP, DOWN, CTRL_C, ...).
class Terminal
{
public:
    Terminal();
    ~Terminal();

    Terminal(const Terminal &) = delete;
    Terminal &operator=(const Terminal &) = delete;

    bool init(QString *error);
    void shutdown();
    bool isActive() const;

    QSize size() const;
    QString pollKey();
    void write(const QString &text);
    void flush();

private:
    struct Impl;
    std::unique_ptr<Impl> m_impl;
};
