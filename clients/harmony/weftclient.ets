/**
 * Weftlink HarmonyOS thin client — Weft Protocol TCP framing.
 *
 * Frame format (per protocol/SPEC.md):
 *   [4-byte big-endian length][JSON payload]
 *
 * Uses @ohos.net.socket (HarmonyOS NEXT native socket API).
 */
import { socket } from '@ohos.net.socket';

export class WeftClient {
  private tcp: socket.TCPSocket | null = null;
  private readBuffer: ArrayBuffer = new ArrayBuffer(0);
  private pendingFrames: Record<string, object>[] = [];
  private frameResolve: ((v: Record<string, object>) => void) | null = null;

  constructor(private host: string, private port: number) {
    this.tcp = socket.constructTCPSocketInstance();
  }

  /** Connect to the daemon. */
  async connect(): Promise<void> {
    const t = this.tcp!;
    t.on('connect', () => {});
    t.on('error', (err) => { throw err; });
    t.on('message', (msg) => { this.onData(msg.data); });

    await t.connect({
      address: { remoteIP: this.host, remotePort: this.port },
      timeout: 5000
    });
  }

  /** Send a JSON request and wait for a complete response frame. */
  async roundTrip(req: Record<string, object>): Promise<Record<string, object>> {
    this.sendFrame(req);
    return this.waitFrame();
  }

  /** Encode and send one frame. */
  sendFrame(req: Record<string, object>): void {
    const json = JSON.stringify(req);
    const encoder = new util.TextEncoder();
    const jsonBytes: ArrayBuffer = encoder.encode(json);
    const len = jsonBytes.length;
    const header = new ArrayBuffer(4);
    const dv = new DataView(header);
    dv.setUint32(0, len, false); // big-endian
    // Concatenate header + payload
    const frame = new Uint8Array(4 + len);
    frame.set(new Uint8Array(header), 0);
    frame.set(new Uint8Array(jsonBytes), 4);
    this.tcp!.send({ data: frame.buffer as ArrayBuffer });
  }

  /** Accumulate data; when a complete frame is received, resolve the promise. */
  private onData(chunk: ArrayBuffer): void {
    // Concatenate
    const merged = new Uint8Array(this.readBuffer.byteLength + chunk.byteLength);
    merged.set(new Uint8Array(this.readBuffer), 0);
    merged.set(new Uint8Array(chunk), this.readBuffer.byteLength);
    this.readBuffer = merged.buffer;

    while (this.readBuffer.byteLength >= 4) {
      const dv = new DataView(this.readBuffer);
      const frameLen = dv.getUint32(0, false); // big-endian
      const totalLen = 4 + frameLen;
      if (this.readBuffer.byteLength < totalLen) break; // wait for more
      // Extract frame payload
      const payload = new Uint8Array(this.readBuffer, 4, frameLen);
      const decoder = new util.TextDecoder();
      const json = decoder.decode(payload.buffer);
      const resp = JSON.parse(json) as Record<string, object>;
      // Remove frame from buffer
      this.readBuffer = (new Uint8Array(this.readBuffer, totalLen)).buffer;

      if (this.frameResolve) {
        const r = this.frameResolve;
        this.frameResolve = null;
        r(resp);
      } else {
        this.pendingFrames.push(resp);
      }
    }
  }

  /** Wait for a complete response frame. */
  private async waitFrame(): Promise<Record<string, object>> {
    if (this.pendingFrames.length > 0) {
      return this.pendingFrames.shift()!;
    }
    return new Promise((resolve) => { this.frameResolve = resolve; });
  }

  close(): void { this.tcp?.close(); }
}