# Weftlink HarmonyOS Thin Client

ArkTS client for HarmonyOS NEXT (Huawei Mate X7). Talks Weft Protocol over
**TLS** to a `weftlinkd` daemon (on a Windows/macOS/Linux warp node).

## Files

| File | Purpose |
|------|---------|
| `weftclient.ets` | TLS transport (`@ohos.net.socket` TLSSocket) + Weft Protocol framing (`[4-byte BE length][JSON]`). `connect()` / `roundTrip()` / `sendFrame()` / `close()`. |
| `pages/Index.ets` | ArkUI page: host/port inputs, Connect button, status display (node_id / version / role). |
| `oh-package.json5` | Package manifest. |

## Protocol

| Request | Response |
|---------|----------|
| `{"type":"hello"}` | `{"type":"hello_ack","node_id":"...","version":"0.1.0","role":"warp"}` |
| `{"type":"status"}` | `{"type":"status","peers":0,"version":"0.1.0"}` |
| `{"type":"ping"}` | `{"type":"pong"}` |

## Build & run

1. **Enable USB debugging on the Mate X7**
   Settings → About phone → tap *Build number* 7× → Settings → System & updates
   → Developer options → **USB debugging ON**. Connect USB, pick *Transfer files*,
   confirm the "Allow USB debugging?" popup.

2. **Verify the device is visible** (DevEco toolchain):
   ```sh
   "<DevEco>/26.0.0/toolchains/hdc.exe" list targets
   ```

3. **Create a project in DevEco Studio** (File → New → New Project → Empty Ability,
   ArkTS, Stage model), then copy `weftclient.ets` and `pages/Index.ets` into
   `entry/src/main/ets/`.

4. **Set the daemon address** to your PC's Wi-Fi IP (`index.ets` default
   `192.168.43.34:7801`) and Run.

## TLS note

The daemon uses a self-signed cert (auto-generated on first run). The client
uses `skipRemoteValidation: true` for the dev/spike path. For production, pin
the cert via `secureOptions.ca` (see `etc/weftlinkd-cert.pem` on the daemon).
