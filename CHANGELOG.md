# Changelog

## [Unreleased]

### Added
- **MSIX packaging (CI/CD)**: `packaging/msix/build-msix.ps1` builds a signed
  full-trust MSIX with Qt runtime, CLI, daemon, GUI and TUI; CI validates the
  package and CD attaches `weftlink-<version>-x64.msix` plus the signing cert.
- **Firewall integration**: the GUI has a `Firewall rules` button (UAC
  elevation); `weftlink up` also applies the rules automatically when running
  elevated. `weftlink firewall allow` remains available explicitly.
- **Unpair** (`weftlink untie` / GUI Unpair): removes the local knot and
  best-effort notifies the peer (`unpair_request` / `unpair_ack`).
- **Firewall helper**: `weftlink firewall allow` (Windows, administrator)
  adds inbound TCP 7801 / UDP 7802 rules for `profile=any`.
- **Daemon auto-start**: GUI Connect and TUI `c` try `weftlink up` when the
  daemon is not running, then retry the connection.
- **Quiet remote execution**: approved commands run with a hidden console on
  Windows; denials/timeouts show a warning on the caller.
- **GUI M2 panel** (`weftlink-gui`): file send/receive, remote status,
  run-request approval (requests table + approve/deny), and clipboard
  enable/push/pull, sharing the same Qt client as the CLI.
- **Remote shell selection**: `weftlink run --shell auto|powershell|cmd|sh`
  (Windows `auto` = PowerShell with UTF-8 output; Unix `auto` = `/bin/sh`).
- **UTF-8 console handling**: CLI sets the Windows console to UTF-8, and remote
  commands force UTF-8 child output so Chinese stdout/stderr stays readable.
- **`weftlink stop`**: stops the local daemon through a loopback-only
  `shutdown` control verb (remote clients cannot stop a daemon).
- **File transfer (M2)**: `weftlink send <peer> <file>` and
  `weftlink recv --from <peer> --dir <dir>` with explicit receiver consent.
  Paired-only TLS transfer, `FileOffer` / `FileAccept` / `FileChunk` /
  `FileDone`, 64 KiB chunks, SHA-256 + size verification, sanitized/unique
  file names, `.part` temp file + atomic rename.
- **Remote status**: `weftlink status <peer>` queries a paired peer's daemon
  through the `peer_status` control verb.
- **Remote execution (M2)**: `weftlink run <peer> "<command>"` queues an
  `ExecRequest` on the peer; it only runs after a local `weftlink requests` /
  `weftlink approve <id>` (2-minute approval window, per-request timeout,
  capped stdout/stderr, explicit deny/timeout results). `requests`, `approve`
  and `deny` control verbs are included.
- **Clipboard (M2)**: `weftlink clipboard on|off|status|push|pull` with a
  time-limited receiver opt-in, paired-only TLS transfer, and Windows clipboard
  support via `golang.design/x/clipboard` (other platforms are stubs for now).
- **C++/Qt desktop clients** (`clients/cpp`): shared Qt 6 `weftclient`
  (`QSslSocket` TLS + Weft framing), a Qt Widgets GUI and a full-screen ANSI
  TUI. Both show node identity, discovery counters, peers, pairing and Ping
  RTT; one-time pair codes can be generated or joined. `weftlink-tui --check`
  is a headless smoke test used by CI.
- **Discovery diagnostics**: `status` now reports the discovery port, whether
  the beacon listener is active, and beacon sent/received counters. Beacons are
  emitted from the listener socket (source port 7802) to the multicast group,
  limited broadcast, each interface's directed broadcast, and known peers. The
  CLI prints the same counters, making firewall/Public-network issues visible.
- CMake/CTest build (`weftclient_tests` covers framing), Windows deploy script
  (`windeployqt` + OpenSSL/MinGW runtime), and CI jobs for Linux/Windows.
- ADR-0002 records the desktop UI stack change from Flutter to C++/Qt; Flutter
  remains for mobile.

## [0.4.0] — 2026-10-31

### Added
- **Desktop UI** (`clients/flutter/app`): two-pane shell — left rail with
  daemon endpoint (host/port), Connect / Disconnect, Ping with live latency,
  node identity and an error card; right pane with the peer list and a
  timestamped activity log. Material 3, light + dark themes.
- `CoreClient.ping()` verb (backend by the existing Weft `Ping` message),
  implemented for loopback and fake clients.
- `internal/proto`: Weft Protocol framing extracted into a shared package
  (`ReadFrame` / `WriteFrame`) used by the daemon and available to the CLI.
- **Release archives**: `weftlink-<version>-<os>-<arch>.zip` (Windows) and
  `.tar.gz` (Linux/macOS) containing correctly named `weftlink` + `weftlinkd`
  side by side plus a usage guide, so `weftlink up` works straight out of the
  box. Raw per-binary assets and `SHA256SUMS` are still published.
- `docs/dist/HOWTO-Windows.txt`, `docs/dist/README-other.txt`.

### Fixed
- Desktop UI integration test asserts the daemon's actual node id instead of
  a hard-coded one; `pumpWidget` runs inside `runAsync` so real socket I/O
  completes.

## [0.3.0] — 2026-10-30

### Added
- **`weftlink` CLI** (`cli/`): `up`, `status`, `ls`, `ping`, `knot`, `version`,
  `help`; stdlib `flag`, `--host` / `--port` / `--json`, flags may appear before
  or after positional arguments.
- **LAN discovery**: 3-second UDP beacons (multicast `224.0.0.167` plus
  limited broadcast fallback) advertising public node metadata; unpaired peers
  expire after 12 seconds.
- **Knot pairing**: one-time 6-character code (2-minute TTL, 5 failures),
  `pair_request` / `pair_confirm` over TLS, Ed25519 fingerprint/public-key
  consistency checks, knots persisted to `~/.weftlink/knots/`.
- **Stable node identity**: Ed25519 key persisted to `~/.weftlink/node.key`;
  node id and fingerprint are derived instead of hardcoded.
- Daemon control verbs `peers`, `knot_init`, `knot_join`; `status` now returns
  the full peer list and counts.
- `WEFTLINK_HOME`, `WEFTLINK_NAME`, `WEFTLINK_PORT`, `WEFTLINK_BEACON_PORT`
  overrides for isolated/multi-instance runs.

### Changed
- **CI**: explicit `permissions`, concurrency cancellation, Go module read-only
  mode, `-count=1`, and a Linux `-race` run; gofmt gate now covers the whole
  tree (including generated Go).
- **CI**: daemon-dependent jobs wait for `127.0.0.1:7801` readiness and fail
  instead of silently skipping Dart/Flutter tests
  (`WEFTLINK_REQUIRE_DAEMON=1`).
- **CI**: schema job validates conformance vectors as JSON; new CLI
  build/smoke job.
- **CD**: release is gated on vet/test/gen; builds `weftlinkd` and `weftlink`
  for linux/darwin/windows on amd64 and arm64; publishes `SHA256SUMS`; injects
  the tag version via `-X main.productVersion` / `-X main.version`.
- TLS cert/key now live in `~/.weftlink/tls/` instead of the working directory.

## [0.2.0] — 2026-10-06

### Added
- **TLS encryption (mandatory)**: daemon serves TLS 1.3; self-signed cert
  auto-generated on first run (`tls_cert.go`, ECDSA P-256); certs never
  committed (`*.pem` ignored). Plain TCP only as fallback.
- **Dart core client**: `LoopbackCoreClient` (real `SecureSocket` + Weft
  Protocol framing) and `FakeCoreClient`; public barrel `implementations.dart`.
- **Flutter UI** (`clients/flutter/app`): Material 3 status page driven purely
  by the `CoreClient` abstraction; windows/linux/macos/web scaffolding;
  widget tests + a live-daemon integration test.
- **Schema generator** (`tools/gen`): stdlib-only YAML subset parser emitting
  Go structs, Dart classes and ArkTS interfaces into `gen/`; `-check` mode.
- **ArkTS thin client**: TLS transport (`TLSSocket`) + ArkUI status page.
- Tests: crypto (3), TLS cert (2), TCP loopback (5), TLS loopback (1),
  Dart (2), Flutter (3).

### Changed
- TLS certificate algorithm switched Ed25519 → ECDSA P-256 (Dart/BoringSSL
  rejects Ed25519-signed certs in TLS 1.3). Application-layer node identity
  remains Ed25519.
- `.gitignore`: `gen/` products are now committed (single-source discipline).
- CI: added Dart and Flutter jobs; schema job verifies `tools/gen -check`.

### Fixed
- `go build ./...` name collision with `src/` on Linux/macOS → build to `bin/`.
- Dart reserved-word field names escaped by the generator (`final` → `final_`).

## [0.1.0] — 2026-10-02

### Added
- Project skeleton: single Go module `weftlink` with flat `src/` package
- Crypto module: Ed25519 keygen / sign / fingerprint
- Protocol message types: Hello / PairRequest / PairConfirm / Ping /
  FileOffer / FileChunk / ClipboardUpdate / ExecRequest
- Peer management + LAN beacon stub; Knot stub; Transport stub
- Daemon entry: `weftlinkd` v0.1.0
- Protocol schema first draft; CoreClient Dart abstract
- docs (ROADMAP/STATUS/permissions/clients-matrix/platform-matrix) + ADR-0001
- Apache-2.0 license
- CI: 3-platform matrix; CD: tag → cross-compile → GitHub Release
