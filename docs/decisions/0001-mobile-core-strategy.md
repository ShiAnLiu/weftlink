# ADR-0001：移动端核心策略
- 日期：<M0.5 完成日> ｜ 状态：提议
## 背景
移动端（Android/iOS）核心必须进程内运行。方案一：gomobile 桥接 Go 核心
（官方标注 experimental，风险自担）；方案二：移动端核心用 Dart 实现
（协议第二实现，consistency 由 conformance 向量保证）。
## 决策
默认方案一。判决实验：M0.5 spike 第②项——半天内 gomobile 在安卓真机
完成 hello + TLS 握手。
- 通过 → 方案一成立，本 ADR 状态改"通过"。
- 失败 → 启用 Plan B（方案二）：移动端核心 Dart 实现，桌面/服务器/中继
  保持 Go；Plan B 允许推迟至 M5 启用。本 ADR 状态改"通过（触发 Plan B）"。
## 后果
方案一：单核心，无双份协议维护；代价是移动端依赖实验性工具链。
Plan B：摆脱实验性依赖；代价是协议双实现，所有特性双倍改动与测试。
## 替代方案
NDK 直接编 c-shared + dart:ffi（跳过 gomobile）：层数少，但 JNI 生命周期
与工具链全部自理，暂不作为首选。