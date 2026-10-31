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
| `{"type":"peer_status","peer":"..."}` | `peer_status`: `ok`, remote `status`, `error` |
| `{"type":"send_file","peer":"...","path":"..."}` | `send_result`: `ok`, `checksum`, `size`, `error` |
| `{"type":"recv_begin","dir":"...","from":"...","timeout_seconds":300}` | `recv_begin`: `ok`, `dir`, `timeout_seconds`, `error` |
| `{"type":"recv_wait","timeout_seconds":300}` | `recv_result`: `ok`, `path`, `name`, `size`, `checksum`, `from_node_id`, `error` |
| `{"type":"recv_cancel"}`   | `recv_cancelled`                                            |
| `{"type":"run_command","peer":"...","command":"...","timeout_seconds":30}` | `run_result`: `ok`, `exit_code`, `stdout`, `stderr`, `error` |
| `{"type":"exec_requests"}` | `exec_requests`: pending request list                       |
| `{"type":"exec_approve","exec_id":"..."}` / `exec_deny` | `exec_action`: `ok`, `error` |
| `{"type":"clipboard_begin","from":"...","timeout_seconds":600}` | `clipboard_begin`: `ok`, `from`, `expires_in_seconds` |
| `{"type":"clipboard_end"}` / `clipboard_state` | off / current policy                   |
| `{"type":"clipboard_push","peer":"..."}` / `clipboard_pull` | `clipboard_result`: `ok`, `length`, `error` |

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

## File Transfer (M2)

A receiver must explicitly open a one-file receive window first:

```sh
weftlink recv --dir ~/Downloads/Weftlink --from <peer> --timeout 300
```

The sender then targets the peer by node id or display name:

```sh
weftlink send <peer> /path/to/file
```

Flow (all on one TLS connection, pairs only):

1. Sender dials the peer's Weft port and sends `FileOffer` (file id, name,
   size, mime, SHA-256, sender node id + fingerprint).
2. Receiver verifies the sender is a paired knot and a receive window is
   active, then replies `FileAccept`. Otherwise it replies with
   `accept:false` and a reason; nothing is downloaded automatically.
3. Sender streams 64 KiB `FileChunk` frames with monotonically increasing
   offsets, ending with an empty `final:true` chunk.
4. Receiver writes to a `.part` file, checks size and SHA-256, then renames
   it into place (sanitized file name, unique path) and replies `FileDone`.
5. The `weftlink send` call returns the checksum; `weftlink recv` prints the
   saved path.

Remote status uses the control verb `peer_status` (paired peers only) and
returns the remote daemon's normal `status` payload.

## Remote Execution (M2)

Execution is never automatic. A paired sender runs:

```sh
weftlink run <peer> "command"
```

The sender daemon sends `ExecRequest` (id, command, timeout, sender identity)
over TLS. The receiver daemon verifies the sender is a paired knot, replies
`exec_pending`, and keeps the connection open for up to 2 minutes. The peer
user lists and approves it locally:

```sh
weftlink requests
weftlink approve <exec-id>   # or: weftlink deny <exec-id>
```

On approval the receiver runs the command with a per-request timeout (default
30s, max 5m), captures stdout/stderr (capped at 64 KiB each), and sends
`ExecResult` (exit code, output, error) back on the same connection. Denials
and approval timeouts return an error result. Only approved commands execute.

## Clipboard (M2)

The receiver opts in for a time window:

```sh
weftlink clipboard on --from <peer> --timeout 600
weftlink clipboard status
weftlink clipboard off
```

Then either side can push or pull text:

```sh
weftlink clipboard push <peer>   # local -> peer
weftlink clipboard pull <peer>   # peer -> local
```

`ClipboardUpdate` / `ClipboardAck` and `ClipboardRequest` / `ClipboardContent`
carry the text over TLS. Only paired peers are accepted, the receiver's policy
must be active (and match `--from`), and no clipboard text is written to logs.
The current build implements clipboard read/write on Windows; other platforms
return "not implemented" until their backends land.

## LAN Discovery

Daemons emit a JSON beacon every 3 seconds over UDP 7802 from the same socket
that receives beacons (source port 7802): multicast group `224.0.0.167`,
limited broadcast, each interface's directed broadcast, and unicast to known
peers. Beacons carry only public metadata (`node_id`, `display_name`, `role`,
`version`, `fingerprint`, `port`); no secrets. `status` reports the discovery
port, listener state, and beacon sent/received counters. Unpaired peers expire
after 12 seconds; paired knots are kept.


## Schema

Canonical type definitions in `schema/` (YAML). Generated code in `gen/`.
All implementations must conform to vectors in `conformance/`.