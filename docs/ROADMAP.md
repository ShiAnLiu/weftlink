# Roadmap

> See also: STATUS.md for milestone completion status.

## Milestones

| Milestone | Theme                          | Target |
|-----------|--------------------------------|--------|
| M0        | Skeleton                       | NOW    |
| M0.5      | Platform spike                 | 2-3d   |
| M1        | LAN discover & pair (CLI)      | TBD    |
| M2        | Send / clipboard / run (CLI)   | TBD    |
| M3        | Shuttle relay (cross-network)  | TBD    |
| M4        | Screen control + KVM           | TBD    |
| M5        | Flutter client                 | TBD    |
| M6        | HarmonyOS ArkTS client         | TBD    |

## M0 — Skeleton

- [ ] Go module layout (core/daemon/cli/shuttle) with `go vet` passing
- [ ] Flutter project scaffold with `dart analyze` clean
- [ ] CI matrix (go vet/test, dart analyze/test, gen consistency check)
- [ ] LICENSE (Apache-2.0)
- [ ] README / ROADMAP / STATUS
- [ ] permissions.md, clients-matrix.md, platform-matrix.md templates
- [ ] docs/decisions/template.md
- [ ] protocol/schema first draft
- [ ] tools/gen stub

## M0.5 — Platform Spike (2-3 days)

- [ ] Go cross-compile daemon for 3 platforms + TLS echo
- [ ] gomobile: Go core on Android real device hello + TLS handshake
- [ ] Flutter desktop UI connects to local daemon via loopback
- [ ] Schema -> Go/Dart type generation test run
- [ ] Windows input injection POC (robotgo or custom FFI)