/// Fake CoreClient for widget testing.
///
/// Returns canned responses. Used in Flutter widget tests.
/// Never used in production.
library;

import 'dart:async';

import '../core_client.dart';

class FakeCoreClient implements CoreClient {
  final _peers = StreamController<List<Peer>>.broadcast();
  final _clipboard = StreamController<ClipboardEvent>.broadcast();
  final _events = StreamController<CoreEvent>.broadcast();

  bool _connected = false;

  /// Canned peer list returned by [watchPeers].
  List<Peer> peers = [];

  @override
  Future<NodeStatus> up() async {
    _connected = true;
    _events.add(CoreEvent('connected', 'fake core up', null));
    return NodeStatus(true, 'fake-node', '0.1.0');
  }

  @override
  Future<void> down() async {
    _connected = false;
    _events.add(CoreEvent('disconnected', 'fake core down', null));
  }

  bool get isConnected => _connected;

  @override
  Stream<List<Peer>> watchPeers() => _peers.stream;

  @override
  Future<KnotInfo> knot(String pairCode) async {
    final id = PeerId('peer-$pairCode');
    peers = [...peers, Peer(id, 'fake-$pairCode', 'weft', true, true, DateTime.now())];
    _peers.add(peers);
    return KnotInfo(id, 'fingerprint-$pairCode');
  }

  @override
  Future<void> untie(PeerId id) async {
    peers = peers.where((p) => p.id.id != id.id).toList();
    _peers.add(peers);
  }

  @override
  Future<void> sendFile(PeerId id, String path) async {
    _events.add(CoreEvent('progress', 'sent $path to ${id.id}', 100));
  }

  @override
  Stream<ClipboardEvent> watchClipboard() => _clipboard.stream;

  /// Push a clipboard event (for tests).
  void emitClipboard(String text) {
    _clipboard.add(ClipboardEvent(text, 'text/plain', DateTime.now()));
  }

  @override
  Future<ExecResult> run(PeerId id, String cmd) async {
    return ExecResult(0, 'ran "$cmd" on ${id.id}', '');
  }

  @override
  Stream<CoreEvent> watchEvents() => _events.stream;

  /// Push an event (for tests).
  void emitEvent(CoreEvent e) => _events.add(e);
}