#pragma once

#include <QDialog>

class QDialogButtonBox;
class QLineEdit;

// JoinPairDialog collects a one-time pair code and an optional peer address.
class JoinPairDialog : public QDialog
{
    Q_OBJECT

public:
    explicit JoinPairDialog(QWidget *parent = nullptr);

    QString code() const;
    QString peerHost() const;

private:
    void updateAcceptState();

    QLineEdit *m_codeEdit = nullptr;
    QLineEdit *m_peerEdit = nullptr;
    QDialogButtonBox *m_buttons = nullptr;
};
