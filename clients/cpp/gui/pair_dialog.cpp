#include "gui/pair_dialog.h"

#include <QDialogButtonBox>
#include <QFont>
#include <QFormLayout>
#include <QLabel>
#include <QLineEdit>
#include <QPushButton>
#include <QRegularExpression>
#include <QRegularExpressionValidator>
#include <QVBoxLayout>

JoinPairDialog::JoinPairDialog(QWidget *parent)
    : QDialog(parent)
{
    setWindowTitle(QStringLiteral("Join a device"));
    setMinimumWidth(420);

    auto *intro = new QLabel(
        QStringLiteral("Enter the 6-character code shown on the other device."),
        this);
    intro->setWordWrap(true);

    m_codeEdit = new QLineEdit(this);
    m_codeEdit->setMaxLength(6);
    m_codeEdit->setPlaceholderText(QStringLiteral("ABC123"));
    m_codeEdit->setAlignment(Qt::AlignCenter);
    m_codeEdit->setFont(QFont(QStringLiteral("Consolas"), 18, QFont::DemiBold));
    m_codeEdit->setValidator(new QRegularExpressionValidator(
        QRegularExpression(QStringLiteral("^[A-Za-z0-9]{0,6}$")), m_codeEdit));

    m_peerEdit = new QLineEdit(this);
    m_peerEdit->setPlaceholderText(QStringLiteral("optional: 192.168.3.107[:7801]"));

    auto *form = new QFormLayout;
    form->addRow(QStringLiteral("Pair code"), m_codeEdit);
    form->addRow(QStringLiteral("Peer host"), m_peerEdit);

    auto *hint = new QLabel(
        QStringLiteral("Leave Peer host empty to use LAN discovery. If discovery is "
                       "blocked by a firewall, enter the other machine's IP."),
        this);
    hint->setWordWrap(true);
    hint->setObjectName(QStringLiteral("hint"));

    m_buttons = new QDialogButtonBox(QDialogButtonBox::Ok | QDialogButtonBox::Cancel,
                                     this);
    m_buttons->button(QDialogButtonBox::Ok)->setText(QStringLiteral("Join"));

    auto *layout = new QVBoxLayout(this);
    layout->addWidget(intro);
    layout->addLayout(form);
    layout->addWidget(hint);
    layout->addWidget(m_buttons);

    connect(m_buttons, &QDialogButtonBox::accepted, this, &QDialog::accept);
    connect(m_buttons, &QDialogButtonBox::rejected, this, &QDialog::reject);
    connect(m_codeEdit, &QLineEdit::textChanged, this, [this](const QString &text) {
        const auto upper = text.toUpper();
        if (text != upper) {
            m_codeEdit->setText(upper);
            return;
        }
        updateAcceptState();
    });

    updateAcceptState();
}

QString JoinPairDialog::code() const
{
    return m_codeEdit->text().trimmed().toUpper();
}

QString JoinPairDialog::peerHost() const
{
    return m_peerEdit->text().trimmed();
}

void JoinPairDialog::updateAcceptState()
{
    m_buttons->button(QDialogButtonBox::Ok)
        ->setEnabled(m_codeEdit->text().trimmed().size() == 6);
}
