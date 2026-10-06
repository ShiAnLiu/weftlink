/// Integration test: Flutter UI ← LoopbackCoreClient (TLS) → live weftlinkd.
///
/// This is M0.5 spike #3: "Flutter desktop UI pulls device status from the
/// local daemon via loopback". Skipped automatically if no daemon is running.
///
/// Real socket I/O runs inside [WidgetTester.runAsync]; we avoid
/// pumpAndSettle because the busy indicator animates forever.
library;

import 'dart:io';

import 'package:core_client/src/loopback.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:weftlink_app/home_page.dart';

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
  testWidgets('UI shows live daemon status over loopback TLS', (tester) async {
    var up = false;
    await tester.runAsync(() async {
      up = await _daemonUp();
    });
    if (!up) {
      markTestSkipped('weftlinkd not running on 127.0.0.1:7801');
      return;
    }

    await tester.runAsync(() async {
      final client = LoopbackCoreClient();
      await tester.pumpWidget(MaterialApp(home: HomePage(client: client)));
      await Future<void>.delayed(const Duration(milliseconds: 1500));
      await tester.pump();
    });

    expect(find.text('Connected'), findsOneWidget);
    expect(find.text('weftlinkd-windows'), findsOneWidget);
  });
}