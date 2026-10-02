/**
 * Weftlink HarmonyOS thin client — Weft Protocol TCP framing.
 *
 * Frame format (per SPEC.md):
 *   [4-byte big-endian length][JSON payload]
 */
import { socket } from '@ohos.net.socket';

export class WeftClient {
  private conn: socket.Connection | null = null;
  private buf: DataBlob = new DataBlob(0);

  constructor(private host: string, private port: number) {}

  /** Connect to daemon. */
  async connect(): Promise<void> {
    const ctx = new socket.SocketContext(socket.SocketType.TCP);
    const res = await ctx.bind(this.host, this.port);
    if (res.err) throw new Error('bind failed: ' + res.err);
    this.conn = socket.Connection;
    // Wait for connection
    await new Promise<void>((resolve) => { this.conn!.on('connect', resolve); });
  }

  /** Send a request and read the JSON response. */
  async roundTrip(req: Record<string, object>): Promise<Record<string, object>> {
    if (!this.conn) throw new Error('not connected');
    const json = JSON.stringify(req);
    const data = new DataBlob(json.length + 4);
    // Write 4-byte big-endian length
    const view = new DataView(data);
    view.setUint32(0, json.length, true); // little-endian on HarmonyOS? No, big-endian per spec
    // Actually HarmonyOS DataView uses native endian? Need to check docs.
    // For now assume big-endian: (data, 0, false) for big-endian.
    // Actually DataView.setUint32(offset, value, littleEndian)
    // We want big-endian so littleEndian = false.
    view.setUint32(0, json.length, false); // big-endian
    // Write JSON after length
    const encoder = new util.TextEncoder();
    const jsonBytes = encoder.encode(json);
    data.append(jsonBytes);

    let written = await this.conn!.send(data);
    if (written !== data.size) throw new Error('short write');

    // Read response frame
    const lenHeader = await this.conn!.receive(4);
    const hdr = new DataView(lenHeader);
    const respLen = hdr.getUint32(0, false); // big-endian
    const respData = await this.conn!.receive(respLen);
    const decoder = new util.TextDecoder();
    const respJson = decoder.decode(respData);
    return JSON.parse(respJson) as Record<string, object>;
  }
}