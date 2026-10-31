# ADR-0002：桌面客户端改用 C++/Qt（GUI + TUI）

- 日期：2026-10-31 · 状态：通过（用户指定）

## 背景

现有桌面 UI（`clients/flutter/app`）是 Flutter/Dart，M0.5 只做到状态页。用户明确要求
先做 **C++ + Qt** 的 TUI 和 GUI，作为桌面操作入口。

## 决策

新增 `clients/cpp/`：

- `weftclient`：共享 C++/Qt 客户端（`QSslSocket` TLS + Weft 帧 + JSON 模型）。
- `weftlink-gui`：Qt 6 Widgets 双栏控制台（节点/发现/配对/peers/活动日志）。
- `weftlink-tui`：全屏 ANSI TUI，同样的客户端与操作（c/d/p/r/n/j/q）。

Flutter 客户端保留（移动端 M5 与现有桌面状态页不立即删除）。桌面 GUI 的新功能以
C++/Qt 为准；Flutter 桌面是否退役另议。

## 后果

- 双 UI 栈：Go daemon/CLI + Flutter（移动端/现有桌面）+ C++/Qt（新桌面）。
- 新增 Qt 6 与 OpenSSL 运行时依赖；Windows 发布需要 `windeployqt` + MinGW/OpenSSL DLL。
- CI 增加 Linux 与 Windows 的 Qt 构建/CTest，以及 TUI 对真实 daemon 的 TLS `--check`。
- 协议仍只有 daemon 一个实现；C++/Qt 只是 Weft Protocol 的薄客户端，不复制业务逻辑。

## 替代方案

- 继续 Flutter 桌面：被用户否决（本轮目标）。
- TUI 使用 ncurses/ftxui：仍需要一套 TLS 栈，且多一个第三方依赖；直接用 Qt
  Core/Network + ANSI 更省依赖，GUI/TUI 共享同一客户端。
