abstract class CoreClient {
  Future<NodeStatus> up();                        // 启动/连接核心
  Future<void> down();
  Stream<List<Peer>> watchPeers();                // 已织入设备（warp/weft 标注）
  Future<KnotInfo> knot(String pairCode);         // 配对（打结）
  Future<void> untie(PeerId id);
  Future<void> sendFile(PeerId id, String path);  // send
  Stream<ClipboardEvent> watchClipboard();        // clipboard
  Future<ExecResult> run(PeerId id, String cmd);  // run
  Stream<CoreEvent> watchEvents();                // 统一事件流（状态/错误/进度）
}
