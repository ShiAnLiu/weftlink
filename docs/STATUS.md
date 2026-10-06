# Status

Last updated: 2026-10-06

| Milestone | Status | Date   |
|-----------|--------|--------|
| M0        | 🟢     | 2026-10-02 |
| M0.5      | 🟢     | 2026-10-06 |
| M1        | ⚪     | —      |
| M2        | ⚪     | —      |
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
| ③ | Flutter 桌面 UI 回环连 daemon | ✅ | `clients/flutter/app`：HomePage 经 LoopbackCoreClient(TLS) 拉到 `weftlinkd-windows` 状态；集成测试通过；`flutter build web` 成功 |
| ④ | schema → Go/Dart 类型生成 | ✅ | `tools/gen`（stdlib）+ `gen/{go,dart,arkts}`；`-check` 模式接入 CI；Dart 关键字转义 |
| ⑤ | Windows 输入注入预演 | ⚪ | 未开始（M4 相关） |

## 本轮实现要点

- **TLS 加密**：daemon 首次启动自动生成自签名证书（ECDSA P-256），`*.pem` 永不入库；
  `listenTLSAndServe()` 默认 TLS 1.3；明文 TCP 仅作 fallback。
- **Dart CoreClient**：`LoopbackCoreClient` 真实 `SecureSocket` + Weft Protocol 帧；
  `FakeCoreClient` 供 widget 测试。
- **Flutter UI**：`clients/flutter/app`（Material 3），仅依赖 `CoreClient` 抽象；
  desktop 选 loopback，mobile 待 M5。平台脚手架 windows/linux/macos/web 齐备。
- **schema 生成器**：`go run ./tools/gen` 产出三语言类型；CI `-check` 防漂移。
- **ArkTS 客户端**：TLSSocket + 帧解析 + ArkUI 状态页（待真机验证）。

## 测试与检查（全绿）

| 套件 | 数量 | 命令 |
|------|------|------|
| Go 单元/集成 | 11 | `go test ./...` |
| Dart core_client | 2 | `dart test` |
| Flutter app | 3 | `flutter test` |
| 生成物一致性 | 3 | `go run ./tools/gen -check` |

## CI/CD

- **CI**：三平台 Go（build/vet/test/gofmt）+ Dart（analysis/test）+ Flutter（analyze/test）+ schema/gen 一致性
- **CD**：tag `v*` → 交叉编译 `weftlinkd`（windows/linux/darwin amd64）→ GitHub Release
- 已发布：v0.1.0、v0.2.0

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
- **SSH 22 端口被断**：改用 `ssh.github.com:443`（`~/.ssh/config`）。
