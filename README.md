# Weftlink (纬联)

Open-source cross-device interconnect & control tool. Weave your Windows, macOS,
Linux server, Android, iOS, and HarmonyOS NEXT devices into one machine —
discovery, pairing, file transfer, clipboard sync, remote command, screen control,
keyboard/mouse穿越.

LAN direct connect优先, cross-network via shuttle relay as fallback.

## Quick Start

_TODO: M0 after milestone_

## Directory Layout

```
weftlink/
├─ core/           # Go: protocol, discovery, transport, crypto, verbs
├─ daemon/         # Go: weftlinkd
├─ cli/            # Go: weftlink command-line tool
├─ shuttle/        # Go: relay server
├─ bridge/         # Go -> mobile export (gomobile)
├─ protocol/       # Spec, schema (YAML), conformance vectors (JSON)
├─ gen/            # Generated types (go/dart/arkts) — committed, CI-verified
├─ clients/
│  ├─ flutter/     # Dart/Flutter UI + core_client package
│  └─ harmony/     # ArkTS thin client
├─ tools/gen/      # Schema -> multi-language type generator
└─ docs/           # Roadmap, status, matrices, ADRs
```

## License

Apache 2.0