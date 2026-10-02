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

### ✅ ① Go 交叉编译三平台 + TCP echo 互通（~80%）

- 交叉编译：CI Release 已验证 → 三平台二进制 ✅
- TCP 回环集成测试：5 个测试全通（hello/ping/status/unknown/multi-frame）✅
- 外部客户端直连：Wi-Fi IP `192.168.43.34:7801` 验证 ok ✅
- Windows 防火墙规则已添加 ✅
- ⬜ TLS 加密（当前是明文 TCP）
- ⬜ 真双机（需手机端 hdc 确认 + ArkTS App 跑通）

### ⬜ ② gomobile 安卓真机验证

Mate X7 是 HarmonyOS NEXT，不走 gomobile 路线。焊死 → 改走 ArkTS 客户端路线。

### ⬜ ③ Flutter 桌面回环连 daemon

FakeCoreClient 已就绪；真回环实现待写。依赖 Dart SDK 可用。

### ⬜ ④ schema → Go/Dart 类型生成试跑

tools/gen 还是空壳。

### ⬜ ⑤ Windows 输入注入预演

未开始。

### 另：ArkTS 薄客户端（超前 M6）

- `weftclient.ets`：TCP 客户端 + Weft Protocol 帧解析 ✅
- `pages/Index.ets`：主页 → 连 daemon → 显示状态 ✅
- 待手机 USB 调试启用后 → 导入 DevEco Studio → 真机验证

## CI/CD 管道

- **CI**：push main → 三平台（win/ubuntu/macos）build/vet/test/gofmt/schema，全绿
- **CD**：push tag v* → 交叉编译 → GitHub Release 附二进制
- v0.1.0 released：weftlinkd-{darwin,linux,windows}-amd64