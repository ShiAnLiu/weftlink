/// Weftlink UI entry point.
///
/// Desktop uses [LoopbackCoreClient] (TLS to the local daemon, zero FFI).
/// Mobile will use [GomobileCoreClient] (in-process Go core) once M5 lands.
/// The UI never knows which one it is.
library;

import 'dart:io' show Platform;

import 'package:core_client/core_client.dart';
import 'package:core_client/implementations.dart';

import 'package:flutter/material.dart';

import 'home_page.dart';

/// Picks the CoreClient implementation for the current platform.
CoreClient makeCoreClient() {
  final isDesktop = Platform.isWindows || Platform.isMacOS || Platform.isLinux;
  if (isDesktop && Platform.environment.containsKey('WEFTLINK_FAKE')) {
    return FakeCoreClient();
  }
  if (isDesktop) {
    final host = Platform.environment['WEFTLINK_HOST'] ?? '127.0.0.1';
    final port = int.tryParse(Platform.environment['WEFTLINK_PORT'] ?? '') ?? 7801;
    return LoopbackCoreClient(host: host, port: port);
  }
  // Mobile (Android/iOS/HarmonyOS) — gomobile bridge lands in M5.
  throw UnsupportedError('mobile core not wired yet (M5)');
}

void main() {
  runApp(WeftlinkApp(client: makeCoreClient()));
}

class WeftlinkApp extends StatelessWidget {
  final CoreClient client;

  const WeftlinkApp({super.key, required this.client});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Weftlink',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
        useMaterial3: true,
      ),
      home: HomePage(client: client),
    );
  }
}