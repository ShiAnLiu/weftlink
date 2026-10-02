# Weftlink HarmonyOS Thin Client

> M6 milestone — pre-validated via M0.5 spike on Windows daemon.

## Files

| File | Purpose |
|------|---------|
| `weftclient.ets` | TCP client with Weft Protocol framing (4-byte BE length + JSON). Uses `@ohos.net.socket`. Implements connect/sendFrame/roundTrip. |
| `pages/Index.ets` | Main page — connect to daemon, send `hello`, display status. Configurable host:port. |
| `oh-package.json5` | Package manifest. |

## How to use

1. **Enable USB debugging on your Mate X7**  
   Settings → About phone → tap build number 7× → Settings → System & updates → Developer options → USB debugging ON  
   Connect USB, select "Transfer files" mode, confirm popup.

2. **Open project in DevEco Studio**  
   Click "New ArkTS project" → create a blank project → then copy `.ets` files into `src/main/ets/`  
   Or: Create new project from existing files pointing to `clients/harmony/`.

3. **Build & install**  
   DevEco Studio → build → run on connected device.

4. **Verify**  
   The app will connect to `192.168.43.34:7801` (your Windows daemon's Wi-Fi IP).  
   You should see `"node_id": "weftlinkd-windows"` in the response.

## Protocol messages supported

| Request | Response |
|---------|----------|
| `{"type":"hello"}` | `{"type":"hello_ack","node_id":"...","version":"0.1.0","role":"warp"}` |
| `{"type":"status"}` | `{"type":"status","peers":0,"version":"0.1.0"}` |
| `{"type":"ping"}` | `{"type":"pong"}` |