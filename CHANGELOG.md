# Changelog

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
