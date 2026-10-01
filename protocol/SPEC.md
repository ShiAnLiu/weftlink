# Weft Protocol Specification

> Draft — M0 skeleton

## Transport

Weft Protocol is transport-agnostic. Primary transports:
- TLS 1.3 over TCP (LAN/WAN)
- UDP multicast beacon (LAN discovery)
- Shuttle relay (cross-network)

## Message Format

Messages are JSON over a framed length-prefixed stream.

```
[4-byte big-endian length][JSON payload]
```

## Initial Message Types (M1 scope)

| Message          | Direction          | Description                    |
|------------------|--------------------|--------------------------------|
| Hello            | bidirectional      | Handshake, node info           |
| PairRequest      | initiator→target   | Initiate pairing (6-char code) |
| PairConfirm      | target→initiator   | Confirm pairing                |
| Ping             | bidirectional      | Liveness check                 |
| FileOffer        | sender→receiver    | File transfer offer            |
| FileChunk        | sender→receiver    | File data chunk                |
| ClipboardUpdate  | bidirectional      | Clipboard content sync         |
| ExecRequest      | sender→receiver    | Remote command execution       |

## Schema

Canonical type definitions in `schema/` (YAML). Generated code in `gen/`.
All implementations must conform to vectors in `conformance/`.