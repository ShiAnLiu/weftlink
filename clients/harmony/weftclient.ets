/**
 * Weftlink HarmonyOS thin client — TLS transport + Weft Protocol framing.
 *
 * Frame format (per protocol/SPEC.md):
 *   [4-byte big-endian length][JSON payload]
 *
 * Uses @ohos.net.socket TLSSocket (HarmonyOS NEXT native API).
 *
 * Dev note: `skipRemoteValidation` accepts the daemon's self-signed cert.
 * Production should pin the cert via secureOptions.ca instead.
 */
import { socket } from '@ohos.net.socket';
import { util } from '@ohos.util';

/** Response frames are delivered as they arrive; callers await complete ones. */
export class WeftClient {
  private tcp: socket.TLSSocket;
  private readBuffer: ArrayBuffer = new ArrayBuffer(0);
  private pendingFrames: Record<string, Object>[] = [];
  private frameResolve: ((v: Record<string, Object>) => void) | null = null;
  private closed: boolean = false;

  constructor(private host: string, private port: number, private allowSelfSigned: boolean = true) {
    this.tcp = socket.constructTLSSocketInstance();
  }

  /** Connect (TLS handshake) to the daemon. */
  async connect(): Promise<void> {
    this.tcp.on('message', (msg: socket.SocketMessageInfo) => {
      this.onData(msg.message as ArrayBuffer);
    });
    this.tcp.on('close', () => {
      this.closed = true;
    });

    await this.tcp.connect({
      address: { address: this.host, port: this.port, family: 1 },
      secureOptions: {},
      skipRemoteValidation: this.allowSelfSigned,
      timeout: 5000
    });
  }

  /** Send one request and await one complete response frame. */
  async roundTrip(req: Record<string, Object>): Promise<Record<string, Object>> {
    this.sendFrame(req);
    return this.waitFrame();
  }

  /** Encode [len][json] and send. */
  sendFrame(req: Record<string, Object>): void {
    const json: string = JSON.stringify(req);
    const encoder = new util.TextEncoder();
    const jsonBytes: Uint8Array = encoder.encodeInto(json);
    const frame = new Uint8Array(4 + jsonBytes.length);
    const dv = new DataView(frame.buffer);
    dv.setUint32(0, jsonBytes.length, false); // big-endian
    frame.set(jsonBytes, 4);
    this.tcp.send(frame.buffer as ArrayBuffer);
  }

  /** Accumulate bytes; resolve when a full frame is available. */
  private onData(chunk: ArrayBuffer): void {
    const prev = new Uint8Array(this.readBuffer);
    const next = new Uint8Array(chunk);
    const merged = new Uint8Array(prev.length + next.length);
    merged.set(prev, 0);
    merged.set(next, prev.length);
    this.readBuffer = merged.buffer;

    let offset = 0;
    while (this.readBuffer.byteLength - offset >= 4) {
      const dv = new DataView(this.readBuffer);
      const frameLen = dv.getUint32(offset, false);
      if (this.readBuffer.byteLength - offset - 4 < frameLen) {
        break;
      }
      const payload = new Uint8Array(this.readBuffer, offset + 4, frameLen);
      const text = util.TextDecoder.create('utf-8').decodeToString(new Uint8Array(payload));
      const resp = JSON.parse(text) as Record<string, Object>;
      offset += 4 + frameLen;
      if (this.frameResolve !== null) {
        const r = this.frameResolve;
        this.frameResolve = null;
        r(resp);
      } else {
        this.pendingFrames.push(resp);
      }
    }
    // keep remainder
    if (offset > 0) {
      const rest = new Uint8Array(this.readBuffer, offset);
      this.readBuffer = rest.slice().buffer;
    }
  }

  private async waitFrame(): Promise<Record<string, Object>> {
    if (this.pendingFrames.length > 0) {
      return this.pendingFrames.shift()!;
    }
    return new Promise<Record<string, Object>>((resolve) => {
      this.frameResolve = resolve;
    });
  }

  close(): void {
    this.tcp.close();
  }
}