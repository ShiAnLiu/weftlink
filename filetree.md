weftlink/
├─ core/                     # Go：协议/发现/传输/加密/动词（唯一核心实现）
├─ daemon/                   # Go：weftlinkd
├─ cli/                      # Go：weftlink
├─ shuttle/                  # Go：中继
├─ bridge/gomobile/          # Go：移动端导出包（窄接口：动词 + 事件流）
├─ protocol/
│  ├─ SPEC.md                # Weft Protocol 规范（人读）
│  ├─ schema/                # 机器可读 schema（单一来源，YAML）
│  └─ conformance/           # 一致性测试向量（JSON）
├─ gen/  go/ dart/ arkts/    # 生成物（提交进仓库，CI 校验）
├─ tools/gen/                # schema → 三语言类型生成器
├─ clients/
│  ├─ flutter/
│  │  └─ packages/core_client/   # 抽象 + 回环实现 + gomobile 实现
│  └─ harmony/               # ArkTS 薄客户端
└─ docs/
   ├─ ROADMAP.md  STATUS.md
   ├─ clients-matrix.md  permissions.md  platform-matrix.md
   ├─ spike-platform.md
   └─ decisions/             # ADR
      ├─ template.md
      └─ 0001-mobile-core-strategy.md
