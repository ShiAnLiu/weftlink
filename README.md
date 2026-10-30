# Weftlink (纬联)

Open-source cross-device interconnect & control tool. Weave your Windows, macOS,
Linux server, Android, iOS, and HarmonyOS NEXT devices into one machine —
discovery, pairing, file transfer, clipboard sync, remote command, screen control,
keyboard/mouse穿越.

LAN direct connect优先, cross-network via shuttle relay as fallback.

## Quick Start

Build the daemon and CLI:

```sh
go build -o bin/weftlinkd ./src
go build -o bin/weftlink ./cli
```

Use it:

```sh
./bin/weftlink up                 # start the local daemon (background)
./bin/weftlink status             # local node id / version / peer counts
./bin/weftlink ls                 # peers discovered on the LAN
./bin/weftlink ping               # round-trip time to the local daemon
./bin/weftlink knot               # print a one-time 6-character pair code
./bin/weftlink knot ABC123        # pair from the other device
```

State (node key, TLS cert, paired knots) lives in `~/.weftlink/`; set
`WEFTLINK_HOME` to isolate it. Default ports: 7801 (Weft Protocol), 7802
(discovery beacon). Pass `--json` for machine-readable output.

Pairing is mutual: one side runs `weftlink knot` to print a code, the other
runs `weftlink knot <code>`. The code expires after two minutes and is
single-use. If discovery is unavailable, pass `--peer <ip[:port]>`.

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