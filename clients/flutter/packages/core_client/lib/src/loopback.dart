/// Loopback CoreClient implementation.
///
/// Connects to local weftlinkd daemon via 127.0.0.1 + local token.
/// Zero FFI — pure Dart over Weft Protocol on loopback TCP.

import 'package:core_client/core_client.dart';

class LoopbackCoreClient extends CoreClient {
  // TODO: M0.5 spike — implement loopback connection
  @override
  Future<NodeStatus> up() async {
    throw UnimplementedError();
  }

  @override
  Future<void> down() async {
    throw UnimplementedError();
  }

  @override
  Stream<List<Peer>> watchPeers() {
    throw UnimplementedError();
  }

  @override
  Future<KnotInfo> knot(String pairCode) async {
    throw UnimplementedError();
  }

  @override
  Future<void> untie(PeerId id) async {
    throw UnimplementedError();
  }

  @override
  Future<void> sendFile(PeerId id, String path) async {
    throw UnimplementedError();
  }

  @override
  Stream<ClipboardEvent> watchClipboard() {
    throw UnimplementedError();
  }

  @override
  Future<ExecResult> run(PeerId id, String cmd) async {
    throw UnimplementedError();
  }

  @override
  Stream<CoreEvent> watchEvents() {
    throw UnimplementedError();
  }
}