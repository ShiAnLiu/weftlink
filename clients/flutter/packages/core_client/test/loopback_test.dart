/// Tests for LoopbackCoreClient against a running weftlinkd daemon.
///
/// Requires the daemon listening on 127.0.0.1:7801. If not reachable,
/// the tests are skipped (so `dart test` works without a daemon).
library;

import 'dart:io';

import 'package:core_client/src/loopback.dart';
import 'package:test/test.dart';

Future<bool> _daemonUp() async {
  try {
    final s = await Socket.connect('127.0.0.1', 7801,
        timeout: const Duration(milliseconds: 500));
    s.destroy();
    return true;
  } catch (_) {
    return false;
  }
}

void main() {
  late bool up;

  setUpAll(() async {
    up = await _daemonUp();
  });

  test('up() returns NodeStatus from daemon', () async {
    if (!up) {
      markTestSkipped('weftlinkd not running on 127.0.0.1:7801');
      return;
    }
    final client = LoopbackCoreClient();
    final status = await client.up();
    expect(status.connected, isTrue);
    expect(status.nodeId, 'weftlinkd-windows');
    expect(status.version, isNotEmpty);
    await client.down();
  });

  test('multiple sequential hellos work (framing)', () async {
    if (!up) {
      markTestSkipped('weftlinkd not running on 127.0.0.1:7801');
      return;
    }
    for (var i = 0; i < 3; i++) {
      final client = LoopbackCoreClient();
      final status = await client.up();
      expect(status.connected, isTrue);
      await client.down();
    }
  });
}