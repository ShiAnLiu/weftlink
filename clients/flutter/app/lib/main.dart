/// Weftlink UI entry point.
///
/// The UI only ever talks to [CoreClient]; which implementation backs it
/// (desktop loopback over TLS, or the M5 gomobile bridge) is decided here
/// and nowhere else.
library;

import 'dart:io' show Platform;

import 'package:core_client/core_client.dart';
import 'package:core_client/implementations.dart';
import 'package:flutter/material.dart';

import 'home_page.dart';

/// Builds a client for the given daemon endpoint.
typedef CoreClientFactory = CoreClient Function(String host, int port);

/// Desktop default: loopback TLS to the local daemon.
CoreClient desktopClient(String host, int port) =>
    LoopbackCoreClient(host: host, port: port);

void main() {
  final fake = Platform.environment.containsKey('WEFTLINK_FAKE');
  final host = Platform.environment['WEFTLINK_HOST'] ?? '127.0.0.1';
  final port = int.tryParse(Platform.environment['WEFTLINK_PORT'] ?? '') ?? 7801;

  runApp(WeftlinkApp(
    factory: fake ? (h, p) => FakeCoreClient() : desktopClient,
    initialHost: host,
    initialPort: port,
  ));
}

class WeftlinkApp extends StatelessWidget {
  final CoreClientFactory factory;
  final String initialHost;
  final int initialPort;

  const WeftlinkApp({
    super.key,
    required this.factory,
    required this.initialHost,
    required this.initialPort,
  });

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Weftlink',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF3F51B5),
          brightness: Brightness.light,
        ),
        useMaterial3: true,
        visualDensity: VisualDensity.compact,
      ),
      darkTheme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF3F51B5),
          brightness: Brightness.dark,
        ),
        useMaterial3: true,
        visualDensity: VisualDensity.compact,
      ),
      home: HomePage(
        factory: factory,
        initialHost: initialHost,
        initialPort: initialPort,
      ),
    );
  }
}
