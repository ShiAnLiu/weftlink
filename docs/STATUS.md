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

## CI/CD 管道（已验证）

- **CI**：push main → 三平台（win/ubuntu/mac）build/vet/test + gofmt + schema 校验，全绿
- **CD**：push tag v* → 交叉编译 weftlinkd（windows/linux/darwin amd64）→ GitHub Release 附二进制
- **端到端验证**：v0.1.0 release 产物下载后运行正常（weftlinkd v0.1.0 ready on port 7801）
- **M0.5 spike ① 预验证**：Windows 本机交叉编译 Linux ELF / macOS 二进制成功

## 待办（M0.5）

- [x] ① Go 交叉编译三平台 ✅（CI release 已验证；双机 TLS echo 互通待做）
- [ ] ② gomobile 安卓真机验证（ADR-0001 判决点）
- [ ] ③ Flutter 桌面回环连 daemon
- [ ] ④ schema → Go/Dart 类型生成试跑
- [ ] ⑤ Windows 输入注入预演