# Status

Last updated: 2026-10-02

| Milestone | Status | Date   |
|-----------|--------|--------|
| M0        | 🟢     | 2026-10-02 |
| M0.5      | 🔴     | 2026-10-02 |
| M1        | ⚪     | —      |
| M2        | ⚪     | —      |
| M3        | ⚪     | —      |
| M4        | ⚪     | —      |
| M5        | ⚪     | —      |
| M6        | ⚪     | —      |

🔴 In progress  ⚪ Not started  🟢 Done  🟡 Blocked

## M0 骨架完成记录

- ✅ Go module（单 module `weftlink`，src/ 扁平包）
- ✅ `go build` / `go vet` / `go test` 全绿；`gofmt` 无告警
- ✅ 单元测试：crypto（Ed25519 keygen/sign/fingerprint）
- ✅ daemon 入口可运行：`weftlinkd v0.1.0 ready on port 7801`
- ✅ 协议 schema 首版（protocol/schema/weftlink.yaml，8 个消息）
- ✅ CoreClient Dart 抽象（clients/flutter/packages/core_client）
- ✅ docs 全套（ROADMAP / STATUS / permissions / clients-matrix / platform-matrix / ADR-0001）
- ✅ Apache-2.0 LICENSE / README / CHANGELOG

## M0.5 进度

### ✅ ① Go 交叉编译三平台 + TLS echo 互通（完成）

- 交叉编译：CI Release 已验证 → 三平台二进制 ✅
- TCP 回环集成测试：5 个测试全通（hello/ping/status/unknown/multi-frame）✅
- **TLS 1.3 加密**：自签名证书自动生成（ECDSA P-256）；TLS 回环测试通过 ✅
- **真实 TLS 验证**：openssl + Go 客户端 + Dart 客户端均成功握手 ✅
- Windows 防火墙规则已添加 ✅
- 明文 TCP 仅作 fallback

### ⬜ ② gomobile 安卓真机验证

Mate X7 是 HarmonyOS NEXT，不走 gomobile 路线。改走 ArkTS 客户端路线。

### 🟡 ③ Flutter 桌面回环连 daemon

- **Dart LoopbackCoreClient 已实现**（TLS + Weft Protocol 帧），2 个 Dart 测试通过 ✅
- FakeCoreClient 已实现（widget 测试用）✅
- ⬜ 真 Flutter UI + Flutter SDK 未安装（Dart 有，Flutter 无）

### ⬜ ④ schema → Go/Dart 类型生成试跑

tools/gen 还是空壳。

### ⬜ ⑤ Windows 输入注入预演

未开始。

### 另：ArkTS 薄客户端（超前 M6）

- `weftclient.ets`：TCP 客户端 + Weft Protocol 帧解析 ✅
- `pages/Index.ets`：主页 → 连 daemon → 显示状态 ✅
- 待手机 USB 调试启用后 → 导入 DevEco Studio → 真机验证
- 注意：ArkTS 也需用 `constructTLSSocketInstance` 做 TLS（证书用 ECDSA）

## CI/CD 管道

- **CI**：三平台 build/vet/test/gofmt + schema + **Dart analyze/test（含起 daemon）**
- **CD**：tag v* → 交叉编译 → GitHub Release 附二进制
- v0.1.0 released：weftlinkd-{darwin,linux,windows}-amd64

## 经验记录

- **Ed25519 证书与 Dart 不兼容**：Dart 内置 BoringSSL 无法处理 TLS 1.3 中
  由 Ed25519 签名的证书（`HANDSHAKE_FAILURE_ON_CLIENT_HELLO`）。解决：TLS
  传输证书改用 ECDSA P-256；应用层节点身份仍用 Ed25519（符合宪法）。
- **Go 函数不支持多返回值签名**：本环境 Go 方言中 `func f() (int, error)`
  编译失败；改用指针出参或标准库调用+预声明变量。
- **`go build ./...` 与 src/ 目录冲突**：main 包输出名与目录同名，Linux/macOS
  报 “build output \"src\" already exists and is a directory”；CI 改用 `-o bin/weftlinkd`。