/**
 * Weftlink HarmonyOS — main page: connect, status display, actions.
 */
import { constructTCPSocketInstance } from '@ohos.net.socket';
import { taskpool } from '@ohos.lang.taskpool';
import { WeftClient } from '../weftclient';

// Default daemon address (configurable via input field)
const DEFAULT_HOST = '192.168.43.34';
const DEFAULT_PORT = 7801;

let client: WeftClient | null = null;

export default {
  data: {
    host: DEFAULT_HOST,
    port: DEFAULT_PORT as int,
    status: 'Idle',
    nodeId: '',
    version: '',
    role: '',
    errorMsg: '',
  },

  async onConnect(): Promise<void> {
    this.data.status = 'Connecting...';
    this.data.errorMsg = '';

    try {
      // Run in worker thread to avoid blocking UI
      const result: Record<string, object> = await taskpool.invoke(
        async () => {
          const c = new WeftClient(/*@*/ this.data.host, /*@*/this.data.port /* the * /);
          await c.connect();
          const res = await c.roundTrip({ type: 'hello' });
          c.close();
          return res;
        }
      );

      this.data.nodeId = result.get('node_id') as string;
      this.data.version = result.get('version') as string;
      this.data.role = result.get('role') as string;
      this.data.status = 'Connected';
    } catch (e) {
      this.data.status = 'Failed';
      this.data.errorMsg = String(e);
    }
  },
};