/// Gomobile CoreClient implementation.
///
/// In-process Go core via gomobile bridge (FFI).
/// Android/iOS only. Exposes narrow verb + event-stream surface.

import 'package:core_client/core_client.dart';

class GomobileCoreClient extends CoreClient {
  // TODO: M5 — implement gomobile bridge integration
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