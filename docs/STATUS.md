# Status

Last updated: 2026-10-31

| Milestone | Status | Date   |
|-----------|--------|--------|
| M0        | 🟢     | 2026-10-02 |
| M0.5      | 🟢     | 2026-10-06 |
| M1        | 🟢     | 2026-10-31 |
| M2        | 🔴     | 2026-10-31 |
| M3        | ⚪     | —      |
| M4        | ⚪     | —      |
| M5        | ⚪     | —      |
| M6        | ⚪     | —      |

🔴 In progress  ⚪ Not started  🟢 Done  🟡 Blocked

## M0 — 骨架（完成）

- 单 module `weftlink`（src/）+ protocol schema（8 消息）+ CoreClient 抽象
- Go：build / vet / test / gofmt 全绿；CI 三平台矩阵
- CD：tag → 交叉编译三平台 → GitHub Release
- docs 全套（ROADMAP / STATUS / permissions / clients-matrix / platform-matrix / ADR-0001）

## M0.5 — 平台验证（完成）

| # | 项目 | 状态 | 证据 |
|---|------|------|------|
| ① | Go 交叉编译 + TLS echo 互通 | ✅ | CI Release 三平台二进制；TLS 1.3 握手（openssl/Go/Dart 客户端）；回环集成测试；Wi-Fi IP 直连 |
| ② | gomobile 安卓真机 | ✅（改道） | Mate X7 为 HarmonyOS NEXT → 走 ArkTS 路线（非 gomobile）。ADR-0001 记录改道理由 |
| ③ | Flutter 桌面 UI 回环连 daemon | ✅ | `clients/flutter/app`：HomePage 经 LoopbackCoreClient(TLS) 拉到本机 daemon 状态（节点 ID 动态）；集成测试通过；`flutter build web` 成功 |
| ④ | schema → Go/Dart 类型生成 | ✅ | `tools/gen`（stdlib）+ `gen/{go,dart,arkts}`；`-check` 模式接入 CI；Dart 关键字转义 |
| ⑤ | Windows 输入注入预演 | ⚪ | 未开始（M4 相关） |

## M1 — 局域网互见与配对（CLI，完成）

- ✅ `weftlink` CLI：`up` / `status` / `ls` / `ping` / `knot` / `version` / `help`；
  全局 `--host` / `--port` / `--json`，状态目录 `~/.weftlink`（`WEFTLINK_HOME` 可覆盖）。
- ✅ daemon：稳定 Ed25519 节点身份（`node.key`）、UDP 组播 + 广播 beacon 发现（3s 间隔、12s TTL）、
  `peers` / `knot_init` / `knot_join` 控制动词；TLS 证书移入状态目录。
- ✅ 配对：单次 6 位码（2 分钟过期、5 次错误上限），双方校验指纹/公钥一致性，
  knot 持久化到 `~/.weftlink/knots/`；本地双 daemon（7801/7811）端到端配对已跑通。
- ✅ **真机验证（2026-10-31）**：DESKTOP-JSRV3V1 (192.168.3.107) ↔ DESKTOP-HF8BFLA
  (192.168.3.140)，两台 Windows 同 Wi-Fi；放行防火墙 TCP 7801 / UDP 7802 后
  `ls` 互见，`knot` 配对成功，两端 `ls` 均显示 `*`。
- ⚠️ 已知边界：`ping` 目前只测本机 daemon RTT；节点间 ping 归入 M2。
- ⚠️ 防火墙坑：Windows Wi-Fi 常被标为 **公用网络**，规则必须 `-Profile Any`
  （或把网络改成专用）；否则 UDP 发现被拦，`ls` 为空，可用 `--peer` 兜底。

## 其他 CLI/桌面能力

- ✅ **取消配对**：`weftlink untie <peer>`（别名 `unpair`）/ GUI Unpair；删除本地
  knot，并 best-effort 通知对方 `unpair_request`（对方离线时本地仍成功）。
- ✅ **自动防火墙**：`weftlink firewall allow`（Windows 管理员）写入 TCP 7801 /
  UDP 7802、`profile=any` 入站规则；GUI 的 **Firewall rules** 按钮通过 UAC 提权，
  `weftlink up` 在管理员权限下也会自动应用。
- ✅ **MSIX 打包**：`packaging/msix/build-msix.ps1` 生成全信任 MSIX（GUI/TUI/CLI/
  daemon + Qt 运行时），CI 校验，CD 在 tag release 中附带 `.msix` 与签名证书。
- ✅ **GUI/TUI 自动拉起 daemon**：连接失败时尝试 `weftlink up` 后重试。
- ✅ **远程命令无控制台弹窗**：批准后子进程使用 `CREATE_NO_WINDOW`；deny/timeout
  会在调用方 GUI 弹出提示。
- ⏳ 待办：交互式命令（ssh 等）需要 PTY + 流式 stdin/stdout，单独设计。

## 本轮实现要点

- **TLS 加密**：daemon 首次启动自动生成自签名证书（ECDSA P-256），`*.pem` 永不入库；
  `listenTLSAndServe()` 默认 TLS 1.3；明文 TCP 仅作 fallback。
- **Dart CoreClient**：`LoopbackCoreClient` 真实 `SecureSocket` + Weft Protocol 帧；
  `FakeCoreClient` 供 widget 测试。
- **Flutter UI**：`clients/flutter/app`（Material 3），仅依赖 `CoreClient` 抽象；
  desktop 选 loopback，mobile 待 M5。平台脚手架 windows/linux/macos/web 齐备。
- **schema 生成器**：`go run ./tools/gen` 产出三语言类型；CI `-check` 防漂移。
- **ArkTS 客户端**：TLSSocket + 帧解析 + ArkUI 状态页（待真机验证）。

## M2 — 互控四件套（CLI，进行中）

- ✅ **文件发送**：接收端先 `weftlink recv --from <peer> --dir <dir>` 显式打开一次性接收窗口；
  发送端 `weftlink send <peer> <file>`。FileOffer/FileAccept/FileChunk/FileDone，
  仅限已配对 peer，64 KiB 分块，SHA-256 + 大小校验，`.part` 临时文件 + 原子改名，
  文件名消毒/去重。两台本地 daemon（7821/7831）端到端传输 + 校验已跑通。
- ✅ **远程 status**：`weftlink status <peer>` 经 `peer_status` 控制动词读取对方的
  节点/peers/发现计数；仅限已配对 peer。
- ✅ **run 远程命令**：`weftlink run <peer> "cmd"` 在对方排队，必须由对方
  `weftlink requests` + `weftlink approve <id>` 显式批准后才执行；2 分钟审批窗口，
  默认 30s / 最大 5m 执行超时，stdout/stderr 各限 64 KiB，deny/timeout 有明确回执。
- ✅ **clipboard 同步**：接收端 `weftlink clipboard on --from <peer>` 显式开启窗口；
  `clipboard push/pull <peer>` 经加密通道交换文本，仅限已配对 peer。Windows 使用
  `golang.design/x/clipboard`，其他平台暂为 stub（返回 not implemented）。
- ✅ 本地双 daemon 端到端已验证：文件 send/recv + SHA-256、远程 status、run 批准执行、
  clipboard push/pull（Windows）。
- ⏳ 待办：两台真机 Windows 复测 M2；TUI 的 M2 操作面板（GUI 已接入）。

## 桌面客户端（C++/Qt，新增）

- `clients/cpp`：共享 `weftclient`（Qt Network/`QSslSocket` + Weft 帧），
  `weftlink-gui`（Qt Widgets 双栏控制台）与 `weftlink-tui`（全屏 ANSI）。
  基础功能：节点/发现计数/peers/配对/Ping/日志；GUI 的 M2 页签已接入
  文件 send/recv、远程 status、run 审批、clipboard push/pull、Unpair；
  daemon 未运行时 GUI/TUI 会尝试自动 `weftlink up`。
- 本地验证（2026-10-31）：MinGW 13.1 + Qt 6.7.2 构建通过；CTest 6 项全绿；
  `weftlink-tui --check` 对运行中的 daemon 成功读取状态/peers；
  `windeployqt` 打包后的 GUI/TUI 在最小 PATH 下可独立运行。
- 决策记录：ADR-0002（桌面栈由 Flutter 改为 C++/Qt；Flutter 移动端保留）。

## 测试与检查（全绿）

| 套件 | 数量 | 命令 |
|------|------|------|
| Go 单元/集成 | 54 | `go test ./...` |
| Dart core_client | 2 | `dart test` |
| Flutter app | 3 | `flutter test` |
| C++/Qt | 6 | `ctest --test-dir build/qt` |
| 生成物一致性 | 3 | `go run ./tools/gen -check` |

## CI/CD

- **CI**：三平台 Go（build/vet/test/gofmt，Linux 额外 `-race`）+ CLI（build + up/status/ls/ping/knot 冒烟）+ M2 双 daemon 端到端（配对→recv→send→校验→远程 status→run 审批执行）+ C++/Qt（Linux/Windows 构建 + CTest + TUI `--check`）+ Windows MSIX 打包/签名 + Dart（analysis/test）+ Flutter（analyze/test）+ schema/gen 一致性 + conformance JSON 校验；daemon 依赖测试启动前做就绪探测，CI 中不再允许静默跳过
- **CD**：tag `v*` → 先跑 vet/test/gen 门禁 → 交叉编译 `weftlinkd` 与 `weftlink`（windows/linux/darwin × amd64/arm64）+ 构建签名 MSIX → SHA256SUMS → GitHub Release；tag 版本号通过 `-X main.productVersion` / `-X main.version` 注入
- 已发布：v0.1.0、v0.2.0、v0.3.0、v0.4.0（CLI + 发现/配对 + 桌面 UI + 开箱即用压缩包）

## 经验记录（踩坑）

- **Ed25519 证书与 Dart/BoringSSL 不兼容**：TLS 1.3 中 Ed25519 签名的证书会让
  Dart 报 `HANDSHAKE_FAILURE_ON_CLIENT_HELLO`。→ TLS 传输证书改用 **ECDSA P-256**；
  应用层节点身份仍用 Ed25519（符合宪法）。
- **Go 方言限制**：函数不支持多返回值签名（`func f() (int, error)` 编译失败）→
  用指针出参；标准库多返回值可用「预声明变量 + `a, b = f()`」接收。
- **`go build ./...` 与同名目录冲突**：main 包输出名若等于目录名（`src`、`gen`），
  Linux/macOS 报错。CI 统一用 `-o bin/...`。
- **Flutter widget 测试中的真实 I/O**：必须包在 `tester.runAsync()` 内；
  有无限动画（进度条）时不要 `pumpAndSettle`。
- **Windows 防火墙 / 公用网络**：两台真机首次联调时 `ls` 一直为空，原因是 Wi-Fi
  被标记为“公用网络”，而示例规则只放行了 Private。→ 防火墙规则统一用
  `-Profile Any`（TCP 7801、UDP 7802），或把网络改成专用；发现仍失败时用
  `knot ... --peer <ip>:7801` 绕过 UDP 发现直接配对。
- **SSH 22 端口被断**：改用 `ssh.github.com:443`（`~/.ssh/config`）。
