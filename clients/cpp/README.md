# Weftlink C++/Qt desktop clients

Two thin desktop clients for the local `weftlinkd` daemon, sharing one
C++/Qt implementation of the Weft Protocol:

| Target | UI | Stack |
|--------|----|-------|
| `weftlink-gui` | Qt Widgets two-pane console | Qt 6 Widgets |
| `weftlink-tui` | Full-screen ANSI terminal UI | Qt 6 Core/Network + terminal |
| `weftclient` | shared static library | `QSslSocket` + framed JSON |

Both talk TLS to `127.0.0.1:7801` (the same loopback path as `weftlink`), show
the daemon node identity, discovery counters, peers and pairing state, measure
Ping RTT, and generate/join one-time 6-character pair codes. The GUI also has
an **M2** tab:

- send a file to the selected paired peer;
- open a one-file receive window (or accept from any paired peer);
- request a remote command and approve/deny pending requests on this machine;
- enable clipboard receive and push/pull clipboard text;
- query the selected peer's remote status;
- unpair the selected peer.

If the daemon is not running, Connect (GUI) or `c` (TUI) tries `weftlink up`
and retries. On Windows `weftlink firewall allow` (administrator) adds the
inbound rules.

The daemon still enforces explicit consent: receive windows, run approval and
clipboard opt-in are all required. The TUI currently covers the M1 operations
(connect/status/ping/pair); its M2 panel is the next step.

## Build

Prerequisites: Qt 6.5+ (Core/Network/Widgets/Test), CMake 3.21+, Ninja, a C++17
compiler. On Windows the bundled MinGW build needs the matching MinGW runtime
and OpenSSL 3 DLLs at runtime.

Example using an aqt-installed Qt:

```powershell
$qt   = "D:\QtLocal\6.7.2\mingw_64"
$mingw = "D:\QtLocal\Tools\mingw1310_64\bin"
$ssl  = "D:\QtLocal\Tools\OpenSSLv3\Win_x64\bin"
$env:PATH = "$qt\bin;$mingw;$ssl;" + $env:PATH

cmake -S clients/cpp -B build/cpp -G Ninja -DCMAKE_BUILD_TYPE=Release `
  -DCMAKE_PREFIX_PATH="$qt" `
  -DCMAKE_C_COMPILER="$mingw\gcc.exe" `
  -DCMAKE_CXX_COMPILER="$mingw\g++.exe"
cmake --build build/cpp
ctest --test-dir build/cpp --output-on-failure
```

Run:

```powershell
.\build\cpp\weftlink-gui.exe
.\build\cpp\weftlink-tui.exe
.\build\cpp\weftlink-tui.exe --check      # connect, print status, exit
```

`--check` is headless and used by CI against a live daemon.

## Deploy a runnable Windows folder

`windeployqt` copies the Qt DLLs/plugins, then the script adds OpenSSL and the
MinGW runtime:

```powershell
.\clients\cpp\scripts\deploy.ps1 `
  -QtDir D:\QtLocal\6.7.2\mingw_64 `
  -OpenSslDir D:\QtLocal\Tools\OpenSSLv3\Win_x64\bin `
  -MinGwBin D:\QtLocal\Tools\mingw1310_64\bin
.\build\cpp\dist\weftlink-gui.exe
.\build\cpp\dist\weftlink-tui.exe
```

## TUI keys

| Key | Action |
|-----|--------|
| `c` | connect to the daemon |
| `d` | disconnect |
| `p` | ping (RTT) |
| `r` | refresh status/peers now |
| `n` | generate a new pair code |
| `j` | join with a pair code (then optional peer host) |
| `q` | quit |

The TUI auto-refreshes status every 2 seconds while connected. The optional
peer host lets you pair when UDP discovery is blocked by a firewall.

## Layout

```
clients/cpp/
├─ CMakeLists.txt
├─ weftclient/      # frame codec, TLS client, wire models
├─ gui/             # Qt Widgets GUI
├─ tui/             # ANSI TUI + terminal raw mode
├─ tests/           # FrameCodec tests (CTest)
└─ scripts/deploy.ps1
```

## Notes

- TLS uses the daemon's self-signed certificate with verification disabled;
  identity is pinned by the daemon's knot store/application layer (same
  tradeoff as the CLI at this stage).
- On Windows the Wi-Fi is often classified as **Public**; allow inbound TCP
  7801 and UDP 7802 with `-Profile Any`, or use `--peer`-style pairing from
  the CLI. The GUI/TUI pair dialog accepts an explicit peer host too.
- The GUI and TUI are read-mostly consoles: pairing changes state, but no file
  or command verbs exist yet.
