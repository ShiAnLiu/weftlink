/**
 * Weftlink HarmonyOS thin client — main page.
 *
 * Connects to a weftlinkd daemon over TLS and shows node status.
 * Scope (per docs/clients-matrix.md): knot / ls / status / send /
 * clipboard / run. This page implements the `status` slice for M0.5.
 */
import { WeftClient } from '../weftclient';

@Entry
@Component
struct Index {
  @State host: string = '192.168.43.34';
  @State port: number = 7801;
  @State status: string = 'Idle';
  @State nodeId: string = '-';
  @State version: string = '-';
  @State role: string = '-';
  @State errorMsg: string = '';

  async doHello(): Promise<void> {
    this.status = 'Connecting…';
    this.errorMsg = '';
    const client = new WeftClient(this.host, this.port);
    try {
      await client.connect();
      const req: Record<string, Object> = { 'type': 'hello' };
      const res: Record<string, Object> = await client.roundTrip(req);
      this.nodeId = res['node_id'] as string;
      this.version = res['version'] as string;
      this.role = res['role'] as string;
      this.status = 'Connected';
      client.close();
    } catch (e) {
      this.status = 'Failed';
      this.errorMsg = '' + e;
    }
  }

  @Builder
  row(label: string, value: string) {
    Row() {
      Text(label).fontSize(14).fontColor('#666').width(100)
      Text(value).fontSize(16).fontWeight(FontWeight.Medium)
    }.width('100%').padding({ top: 6, bottom: 6 })
  }

  build() {
    Column({ space: 8 }) {
      Text('Weftlink').fontSize(30).fontWeight(FontWeight.Bold).margin({ top: 32 })
      Text('HarmonyOS thin client').fontSize(13).fontColor('#999').margin({ bottom: 16 })

      TextInput({ text: this.host, placeholder: 'daemon host' })
        .width('90%')
        .onChange((v: string) => { this.host = v; })
      TextInput({ text: '' + this.port, placeholder: 'port' })
        .width('90%')
        .type(InputType.Number)
        .onChange((v: string) => {
          const n: number = parseInt(v);
          if (!isNaN(n)) { this.port = n; }
        })

      Button('Connect (status)')
        .width('90%')
        .margin({ top: 12 })
        .onClick(() => { this.doHello(); })

      Column() {
        this.row('Status', this.status)
        this.row('Node ID', this.nodeId)
        this.row('Version', this.version)
        this.row('Role', this.role)
        this.row('Error', this.errorMsg)
      }
      .width('90%')
      .padding(12)
      .margin({ top: 16 })
      .backgroundColor('#F5F5F5')
      .borderRadius(8)
    }
    .width('100%')
    .height('100%')
  }
}