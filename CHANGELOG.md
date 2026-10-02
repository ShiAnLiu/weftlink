# Changelog

## [0.1.0] — 2026-10-02

### Added
- Project skeleton: single Go module `weftlink` with flat src/ package
- Crypto module: Ed25519 keygen / sign / fingerprint (crypto_test.go)
- Protocol message types: Hello / PairRequest / PairConfirm / Ping /
  FileOffer / FileChunk / ClipboardUpdate / ExecRequest
- Peer management + LAN beacon stub (UDP discovery placeholder)
- Knot (pairing) state machine stub
- Transport stub (TLS listener/connect placeholder)
- Daemon entry: weftlinkd v0.1.0 starts and reports ready on port 7801
- Protocol schema first draft (protocol/schema/weftlink.yaml)
- CoreClient Dart abstract + loopback/gomobile/fake implementations
- docs: ROADMAP / STATUS / permissions / clients-matrix / platform-matrix
- ADR-0001: mobile core strategy (gomobile default, Plan B = Dart core)
- License: Apache-2.0

### Changed
- Restructured from 5 separate Go modules to single module `weftlink`
  (flat `src/` package main; cross-module imports were impractical)

### Verified
- `go build ./...` green
- `go vet ./...` green
- `go test ./...` green (crypto unit tests)
- `gofmt -l .` clean

### CI/CD
- CI: 3-platform matrix (windows/ubuntu/macos) build/vet/test/gofmt/schema — all green
- CD: tag v* triggers cross-compile + GitHub Release with binaries
- v0.1.0 released: weftlinkd-{darwin,linux}-amd64 + weftlinkd-windows-amd64.exe
- Release artifact downloaded and verified running (M0.5 spike #1 cross-compile validated)
- Fixed CI: pin Go via go-version-file; build to bin/ (src dir name collision on linux/mac)