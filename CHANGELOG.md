# Changelog

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
