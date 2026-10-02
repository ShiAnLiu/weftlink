# Status

Last updated: 2026-10-02

| Milestone | Status | Date   |
|-----------|--------|--------|
| M0        | 🟢     | 2026-10-02 |
| M0.5      | ⚪     | —      |
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

## 待办（M0.5）

- [ ] Go 交叉编译三平台 + 双机 TLS echo
- [ ] gomobile 安卓真机验证（ADR-0001 判决点）
- [ ] Flutter 桌面回环连 daemon
- [ ] schema → Go/Dart 类型生成试跑
- [ ] Windows 输入注入预演