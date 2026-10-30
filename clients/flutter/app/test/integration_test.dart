/// Integration test: Flutter UI ← LoopbackCoreClient (TLS) → live weftlinkd.
///
/// M0.5 spike #3: the desktop UI pulls device status from the local daemon
/// over loopback TLS. Skipped automatically when no daemon is running.
library;

import 'dart:io';

import 'package:core_client/src/loopback.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:weftlink_app/home_page.dart';

/// Returns the daemon's node id, or null when no daemon is listening.
Future<String?> _probeDaemon() async {
  try {
    final s = await Socket.connect('127.0.0.1', 7801,
        timeout: const Duration(milliseconds: 500));
    s.destroy();
  } catch (_) {
    return null;
  }
  final probe = LoopbackCoreClient();
  try {
    final status = await probe.up();
    return status.nodeId;
  } finally {
    await probe.down();
  }
}

void main() {
  testWidgets('UI shows live daemon status over loopback TLS', (tester) async {
    String? expectedNodeId;
    await tester.runAsync(() async {
      expectedNodeId = await _probeDaemon();
    });
    if (expectedNodeId == null) {
      markTestSkipped('weftlinkd not running on 127.0.0.1:7801');
      return;
    }

    await tester.runAsync(() async {
      // pumpWidget must run inside runAsync: initState() starts real socket
      // I/O, which only progresses in the real-async zone.
      await tester.pumpWidget(MaterialApp(
        home: HomePage(
          factory: (host, port) => LoopbackCoreClient(host: host, port: port),
          initialHost: '127.0.0.1',
          initialPort: 7801,
        ),
      ));
      await Future<void>.delayed(const Duration(milliseconds: 1500));
      await tester.pump();
    });

    expect(find.text('Connected'), findsOneWidget);
    expect(find.text(expectedNodeId!), findsOneWidget);
  });
}