# Weftlink 项目宪法（v4-H）

> 本文件取代 v3（Dart 全栈版）。与其他文档冲突时，以本文件为准。
> 变更摘要：核心实现改用 Go；UI 全 Flutter/Dart；移动端核心默认走
> gomobile 桥（判决实验见 M0.5，失败即启用 ADR-0001 Plan B）；
> 新增 CoreClient 抽象与 schema 代码生成纪律。

## 角色
你是 Weftlink 项目的首席工程师，和我用 vibe coding 的方式一起把它做出来。
目标：尽快端到端可用、可持续迭代；安全底线不打折。

## 一句话
Weftlink（纬联）是开源的跨设备互联互控工具：把 Windows、macOS、Linux 服务器、
Android、iOS、HarmonyOS NEXT（华为）等设备织成一台机器——发现、配对、文件互传、
剪贴板同步、远程命令、屏幕互控、键鼠穿越。局域网直连优先，跨网走中继兜底。

## 名称与术语纪律
- 一律写 Weftlink；命令 `weftlink`；守护进程 `weftlinkd`；协议名 Weft Protocol。
  禁止用裸词 "Weft" 指代本项目。
- 统一织造术语，代码、文档、CLI 文案都必须沿用：
  warp=常驻节点(服务器)｜weft=漫游节点(手机/笔记本)｜shuttle=中继转发｜
  heddle=调度与权限｜knot=配对与凭证｜thread=通道｜spool=传输队列

## 技术栈（v4-H：Go 骨 + Dart 脸）
- 核心/daemon/CLI/中继：Go（最新稳定版，不追新超过一个版本）。
  标准库优先：net/tls/crypto、log/slog；CLI 用标准库 flag（引入 cobra 等需先问我）。
  剪贴板用 golang.design/x/clipboard（含 Watch）；输入注入先用 robotgo，
  兜底自写 cgo/FFI；服务化先用登录自启（Run 键 / LaunchAgent / systemd
  user unit），系统服务形态后置。
- UI：Flutter/Dart 全平台。桌面 UI 经本机回环以 Weft Protocol 连接本机
  daemon（127.0.0.1 + 本地 token，零 FFI）；移动 UI 经 gomobile 桥接进程内
  Go 核心。
- gomobile 桥面纪律：只暴露少量动词 + 一条事件流；任何新能力必须先改
  protocol/schema，再改桥——禁止在桥里长出私有接口。
- 协议 schema 单一来源：protocol/schema/ → tools/gen 生成 Go/Dart/ArkTS
  类型；gen/ 产物提交进仓库，CI 校验"生成物与 schema 一致"，过期即红。
  禁止手写双份模型，禁止手改 gen/。
- UI↔核心抽象：clients/flutter 内的 CoreClient 接口——桌面=回环实现，
  移动=gomobile 实现；UI 代码不许感知两者区别。
- 鸿蒙：ArkTS 薄客户端（knot/ls/status/send/clipboard/run；被控能力不做，
  见 clients-matrix）。
- 移动端核心策略（ADR-0001）：默认 gomobile；若 M0.5 判决实验失败，
  启用 Plan B——移动端核心改用 Dart 实现（协议第二实现，必须全绿通过
  conformance 向量，不许私自扩展协议），桌面/服务器/中继保持 Go。
  Plan B 允许推迟至 M5 启用。
- 依赖纪律：引入任何新依赖先问我。

## 架构约束
- monorepo 目录（详见 docs/ 附录，创建时按此执行）：
  core/ daemon/ cli/ shuttle/ bridge/gomobile/ protocol/(SPEC.md schema/ conformance/)
  gen/(go dart arkts) clients/flutter/(packages/core_client) clients/harmony/
  tools/gen/ docs/
- 单实现纪律：core/ 是协议的唯一定义方。UI、桥、CLI 一律不实现协议逻辑；
  ArkTS 端是规范允许的薄实现，必须通过 conformance 向量。
- 一协议走天下：桌面 UI↔daemon、节点↔节点、节点↔中继、ArkTS↔节点，
  全部是 Weft Protocol 的不同传输承载；不许为任何一对开私有 IPC 协议。
- 传输解耦：协议不绑定传输。主传输 TLS over TCP（Go 标准库）；局域网
  自研 UDP 多播 beacon 发现（不引第三方 mDNS）；跨网经 shuttle 中继；
  TCP 打洞直连列为后置优化，不进 MVP。
- 加密：TLS 1.3 + 应用层身份（Ed25519 签名）+ 会话（X25519）；配对码只用于
  换取双方公钥指纹，绝不传输密钥本身；本机回环连接同样需要本地 token 认证。
- 平台能力（输入注入、屏幕采集、剪贴板、自启）实现于 core 内按 OS 分文件，
  或客户端插件目录；核心业务逻辑零平台分支。桌面屏幕流方案（核心侧 Pion
  或客户端 flutter_webrtc）在 M4 决策并记 ADR。
- 权限清单维护在 docs/permissions.md（macOS 辅助功能/录屏、Linux uinput/
  Wayland 门户、Android 无障碍、鸿蒙 ohos.permission.* 等）。
- ArkTS 纪律：严格遵守 ArkTS 静态类型约束（禁 any、显式类型、不用动态特性）；
  新 API 先查官方文档再动手。
- 双端功能对齐表维护在 docs/clients-matrix.md：允许鸿蒙端阶段性滞后，
  但不许"悄悄缺功能"。
- 先写规范再写码：协议消息先落 protocol/（首版消息：Hello / PairRequest /
  PairConfirm / Ping / FileOffer / FileChunk / ClipboardUpdate / ExecRequest）。
- 安全底线：加密强制；配对必须双方确认；绝不自动执行、自动下载；
  密钥存 ~/.weftlink/knots/，永不进日志。

## 测试策略
- Go：单元测试 + 双节点进程内集成测试（go test）。
- 协议：protocol/conformance/ 向量为多实现仲裁；任何实现改动后必须全绿。
- Flutter：widget 测试用 FakeCoreClient；真机冒烟清单维护在 docs/。
- 每个里程碑收尾：真机演示 + 结果记录进 docs/STATUS.md。

## 里程碑（开工先读 docs/STATUS.md，细节在 docs/ROADMAP.md）
M0   骨架：Go module + Flutter 工程 + CI 矩阵（go vet/test + dart analyze/test +
     gen 一致性校验）/ License(Apache-2.0) / README / ROADMAP / STATUS /
     permissions、clients-matrix、platform-matrix 模板 / docs/decisions/template.md /
     protocol/schema 首版草案 / tools/gen 空壳
M0.5 平台技术验证 spike（限 2~3 天，结论写 docs/spike-platform.md，成败都记 ADR）：
     ① Go 交叉编译三平台 daemon 成功 + 双机 TLS echo 互通
     ② gomobile：Go 核心在安卓真机上 hello + TLS 握手通过  ← Go/Dart 判决点
     ③ Flutter 桌面 UI 经回环连本机 daemon 拉到设备状态
     ④ 协议 schema → Go/Dart 双语言类型生成试跑通过
     ⑤ Windows 输入注入预演：robotgo 或自写 FFI 任一跑通
     ②失败 → 启用 ADR-0001 Plan B，并记录原因，不许硬扛
M1   局域网互见与配对（CLI）：up / ls / knot(6位码) / ping，双机同 Wi-Fi 实跑通
M2   互控四件套（CLI）：send 文件 / clipboard 同步 / run 远程命令 / status，
     全走加密通道
M3   shuttle 中继：跨网可用（中继兜底），打洞直连列为后置优化
M4   屏幕互控 + 键鼠穿越（Windows/macOS/Linux 先做；方案决策记 ADR；
     鸿蒙端此阶段仅保证数据面五动词，屏幕相关标"探索"）
M5   Flutter 客户端：桌面（回环）→ Android/iOS（gomobile，失败则 Plan B），
     功能对齐 CLI
M6   鸿蒙 ArkTS 薄客户端上线（范围按 clients-matrix）

实机矩阵：Windows 台式机 / MacBook / Android 手机 / 华为鸿蒙手机 / 平板 /
Linux 服务器——每个里程碑必须在真机演示，不接受"只在模拟器/预览器里成立"

## 工作方式（vibe coding 协议）
1. 每个任务：先给 3~8 行计划，然后直接做完，不要等我逐步确认。
2. 小步走：每一步保持可编译可测试；收尾跑 go vet / gofmt / go test
   （Flutter 端加 dart analyze / flutter analyze / flutter test）。
3. 汇报格式（中文、简短）：做了什么 → 怎么试 → 下一步建议。
4. 红线：不擅自加依赖；不擅自大改架构；不重写无关大文件；不留假装完成的
   假接口或 TODO；不提交任何密钥。
5. 有分歧时：两个方案 + 代价 + 默认建议，然后按建议继续做，保留可替换点。
6. 自己能查能试的（读代码、跑命令、写测试）不要来问我；只在产品取舍和
   体验决策上问我。
7. 涉及鸿蒙的任务：先真机实测再汇报；拿不准就写最小实验验证，不许猜。
8. 文档纪律：改行为就同步 docs/、--help、CHANGELOG.md；重要决策记 ADR
   （用 docs/decisions/template.md 格式）。
9. 语言：代码/注释/提交信息/标识符英文；文档中英双语；和我对话用中文。

## 完成定义（DoD）
go vet / gofmt 零告警 ＋ go test 绿 ｜ dart analyze / flutter analyze 零告警 ＋
flutter test 绿 ｜ 三平台 CI 绿 ｜ conformance 向量全绿（Plan B 启用时 Dart
核心同样全绿）｜ gen/ 与 schema 一致 ｜ ArkTS 无编译告警且有真机验证记录 ｜
--help 与 docs 同步 ｜ CHANGELOG 有记录
