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

## Daemon Control Verbs (CLI / local client)

These verbs are sent by `weftlink` (and desktop clients) to a running daemon
over the same framed TLS transport. They are control-plane requests, distinct
from the node-to-node message types above.

| Request                    | Response (key fields)                                      |
|----------------------------|------------------------------------------------------------|
| `{"type":"hello"}`         | `hello_ack`: `node_id`, `display_name`, `role`, `version`, `fingerprint` |
| `{"type":"status"}`        | `status`: node identity + `peers[]`, `peer_count`, `paired_count` |
| `{"type":"peers"}`         | `peers`: `peers[]`                                         |
| `{"type":"ping"}`          | `pong`: echoes `nonce`, `ts`                               |
| `{"type":"knot_init"}`     | `knot_code`: `code`, `expires_in_seconds`                  |
| `{"type":"knot_join","code":"...","host":"..."}` | `knot_result`: `ok`, `peer`, `error` |

A `Peer` has: `peer_id`, `display_name`, `role`, `version`, `address`,
`fingerprint`, `last_seen` (unix ms), `paired`.

## Pairing (Knot)

1. The waiting node runs `weftlink knot`; its daemon generates a random
   single-use `^[A-Z0-9]{6}$` code valid for 2 minutes (max 5 failures).
2. The joining node runs `weftlink knot <code>`. Its daemon sends a
   `pair_request` over TLS to each discovered peer (or to `--peer`):

   ```json
   {"type":"pair_request","code":"ABC123",
    "node":{"node_id":"...","display_name":"...","role":"warp",
            "version":"0.2.1","fingerprint":"<hex>",
            "public_key":"<hex>","port":7801}}
   ```

3. A peer holding a matching pending code stores the knot and replies
   `pair_confirm` with its own node identity.
4. Both sides persist the peer fingerprint and address under
   `~/.weftlink/knots/<peer_id>.json`. Pairing is mutual: generating the code
   confirms the waiting node, entering it confirms the joining node.

## LAN Discovery

Daemons emit a JSON beacon every 3 seconds over UDP 7802 (multicast group
`224.0.0.167`, plus limited broadcast as a fallback). Beacons carry only
public metadata (`node_id`, `display_name`, `role`, `version`, `fingerprint`,
`port`); no secrets. Unpaired peers expire after 12 seconds; paired knots are
kept.


## Schema

Canonical type definitions in `schema/` (YAML). Generated code in `gen/`.
All implementations must conform to vectors in `conformance/`.