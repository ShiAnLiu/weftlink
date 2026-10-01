/// Weftlink CoreClient abstraction.
///
/// Desktop UI uses [LoopbackCoreClient] (connects to local daemon via loopback).
/// Mobile UI uses [GomobileCoreClient] (in-process Go core via gomobile bridge).
/// UI code must NOT know which implementation it uses.

abstract class CoreClient {
  /// Start / connect to core.
  Future<NodeStatus> up();

  /// Shut down core connection.
  Future<void> down();

  /// Stream of peers currently woven in (warp/weft annotated).
  Stream<List<Peer>> watchPeers();

  /// Knot (pair) with a device using a 6-character pair code.
  Future<KnotInfo> knot(String pairCode);

  /// Untie (unpair) a device.
  Future<void> untie(PeerId id);

  /// Send a file to a peer.
  Future<void> sendFile(PeerId id, String path);

  /// Clipboard event stream (bidirectional sync).
  Stream<ClipboardEvent> watchClipboard();

  /// Execute a remote command on a peer.
  Future<ExecResult> run(PeerId id, String cmd);

  /// Unified event stream (status / error / progress).
  Stream<CoreEvent> watchEvents();
}

// ---------------------------------------------------------------------------
// Data types (stub — M0; will be generated from protocol/schema in M1)
// ---------------------------------------------------------------------------

class NodeStatus {
  final bool connected;
  final String? nodeId;
  final String version;

  NodeStatus(this.connected, this.nodeId, this.version);
}

class Peer {
  final PeerId id;
  final String? displayName;
  final String role; // "warp" | "weft"
  final bool isWarped; // paired
  final bool isWefted; // active session
  final DateTime? lastSeen;

  Peer(this.id, this.displayName, this.role, this.isWarped, this.isWefted, this.lastSeen);
}

class PeerId {
  final String id;
  PeerId(this.id);
}

class KnotInfo {
  final PeerId peerId;
  final String fingerprint;
  KnotInfo(this.peerId, this.fingerprint);
}

class ClipboardEvent {
  final String text;
  final String mime;
  final DateTime timestamp;
  ClipboardEvent(this.text, this.mime, this.timestamp);
}

class ExecResult {
  final int exitCode;
  final String stdout;
  final String stderr;
  ExecResult(this.exitCode, this.stdout, this.stderr);
}

class CoreEvent {
  final String type; // "connected" | "disconnected" | "error" | "progress"
  final String? message;
  final int? progress; // 0-100
  CoreEvent(this.type, this.message, this.progress);
}