/// Loopback CoreClient implementation.
///
/// Connects to a local (or LAN) weftlinkd daemon over TLS + Weft Protocol.
/// Zero FFI — pure Dart. Used by the Flutter desktop UI.
///
/// Frame format (per protocol/SPEC.md):
///   [4-byte big-endian length][JSON payload]
library;

import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import '../core_client.dart';

/// A Weft Protocol client over TLS.
class LoopbackCoreClient implements CoreClient {
  final String host;
  final int port;

  /// Set true to accept the daemon's self-signed cert (dev / LAN).
  final bool allowSelfSigned;

  SecureSocket? _socket;
  final _buffer = BytesBuilder(copy: false);
  final _pending = <Completer<Map<String, dynamic>>>[];
  late final StreamSubscription<Uint8List> _sub;
  bool _closed = false;

  LoopbackCoreClient({
    this.host = '127.0.0.1',
    this.port = 7801,
    this.allowSelfSigned = true,
  });

  @override
  Future<NodeStatus> up() async {
    _socket = await SecureSocket.connect(
      host,
      port,
      onBadCertificate: allowSelfSigned ? (cert) => true : null,
    );
    _sub = _socket!.listen(
      _onData,
      onError: _onError,
      onDone: _onDone,
      cancelOnError: false,
    );
    final resp = await _roundTrip({'type': 'hello'});
    return NodeStatus(
      true,
      resp['node_id'] as String?,
      (resp['version'] as String?) ?? '?',
    );
  }

  @override
  Future<void> down() async {
    if (_closed) return;
    _closed = true;
    await _sub.cancel();
    await _socket?.close();
    _socket = null;
  }

  // -- Weft Protocol round trip ----------------------------------------------

  Future<Map<String, dynamic>> _roundTrip(Map<String, dynamic> req) {
    if (_socket == null) {
      throw StateError('not connected — call up() first');
    }
    final completer = Completer<Map<String, dynamic>>();
    _pending.add(completer);
    _writeFrame(req);
    return completer.future.timeout(
      const Duration(seconds: 10),
      onTimeout: () {
        _pending.remove(completer);
        throw TimeoutException('no response for ${req['type']}');
      },
    );
  }

  void _writeFrame(Map<String, dynamic> msg) {
    final payload = utf8.encode(jsonEncode(msg));
    final header = ByteData(4)..setUint32(0, payload.length, Endian.big);
    final frame = BytesBuilder(copy: false)
      ..add(header.buffer.asUint8List())
      ..add(payload);
    _socket!.add(frame.takeBytes());
  }

  // -- Frame reading ---------------------------------------------------------

  void _onData(Uint8List chunk) {
    _buffer.add(chunk);
    _drainFrames();
  }

  void _drainFrames() {
    var bytes = _buffer.toBytes();
    var offset = 0;
    while (true) {
      if (bytes.length - offset < 4) break;
      final header = ByteData.sublistView(bytes, offset, offset + 4);
      final len = header.getUint32(0, Endian.big);
      if (bytes.length - offset - 4 < len) break;
      final payload = bytes.sublist(offset + 4, offset + 4 + len);
      offset += 4 + len;
      _deliver(payload);
    }
    // Keep the un-consumed remainder.
    _buffer.clear();
    if (offset < bytes.length) {
      _buffer.add(bytes.sublist(offset));
    }
  }

  void _deliver(Uint8List payload) {
    final decoded = jsonDecode(utf8.decode(payload)) as Map<String, dynamic>;
    if (_pending.isEmpty) return; // unexpected; drop
    final completer = _pending.removeAt(0);
    if (!completer.isCompleted) completer.complete(decoded);
  }

  void _onError(Object error) {
    for (final c in _pending) {
      if (!c.isCompleted) c.completeError(error);
    }
    _pending.clear();
  }

  void _onDone() {
    for (final c in _pending) {
      if (!c.isCompleted) {
        c.completeError(StateError('connection closed'));
      }
    }
    _pending.clear();
  }

  // -- CoreClient surface (M1/M2 verbs; unimplemented until daemon supports) --

  @override
  Stream<List<Peer>> watchPeers() {
    throw UnimplementedError('watchPeers — M1');
  }

  @override
  Future<KnotInfo> knot(String pairCode) {
    throw UnimplementedError('knot — M1');
  }

  @override
  Future<void> untie(PeerId id) {
    throw UnimplementedError('untie — M1');
  }

  @override
  Future<void> sendFile(PeerId id, String path) {
    throw UnimplementedError('sendFile — M2');
  }

  @override
  Stream<ClipboardEvent> watchClipboard() {
    throw UnimplementedError('watchClipboard — M2');
  }

  @override
  Future<ExecResult> run(PeerId id, String cmd) {
    throw UnimplementedError('run — M2');
  }

  @override
  Stream<CoreEvent> watchEvents() {
    throw UnimplementedError('watchEvents — M1');
  }
}