/// Weftlink desktop home page.
///
/// Left rail: endpoint + connection controls + node identity.
/// Right pane: peers woven in and a live activity log.
library;

import 'dart:async';

import 'package:core_client/core_client.dart';
import 'package:flutter/material.dart';

import 'main.dart' show CoreClientFactory;

class HomePage extends StatefulWidget {
  final CoreClientFactory factory;
  final String initialHost;
  final int initialPort;

  const HomePage({
    super.key,
    required this.factory,
    this.initialHost = '127.0.0.1',
    this.initialPort = 7801,
  });

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  late final TextEditingController _hostCtrl;
  late final TextEditingController _portCtrl;

  CoreClient? _client;
  NodeStatus? _status;
  int? _latencyMs;
  bool _busy = false;
  String? _error;
  List<Peer> _peers = [];
  final List<String> _log = [];

  StreamSubscription<List<Peer>>? _peerSub;
  StreamSubscription<CoreEvent>? _eventSub;

  @override
  void initState() {
    super.initState();
    _hostCtrl = TextEditingController(text: widget.initialHost);
    _portCtrl = TextEditingController(text: '${widget.initialPort}');
    _connect();
  }

  @override
  void dispose() {
    _peerSub?.cancel();
    _eventSub?.cancel();
    _client?.down();
    _hostCtrl.dispose();
    _portCtrl.dispose();
    super.dispose();
  }

  void _logLine(String msg) {
    final t = DateTime.now().toIso8601String().substring(11, 19);
    if (!mounted) return;
    setState(() {
      _log.insert(0, '$t  $msg');
      if (_log.length > 200) _log.removeLast();
    });
  }

  Future<void> _connect() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    _logLine('connecting…');

    await _peerSub?.cancel();
    await _eventSub?.cancel();
    await _client?.down();

    final host = _hostCtrl.text.trim();
    final port = int.tryParse(_portCtrl.text.trim()) ?? 7801;
    final client = widget.factory(host, port);
    _client = client;

    // Optional verbs may be unimplemented in some core builds — stay alive.
    try {
      _peerSub = client.watchPeers().listen((p) {
        if (mounted) setState(() => _peers = p);
      });
    } catch (_) {
      _peerSub = null;
    }
    try {
      _eventSub = client.watchEvents().listen((e) {
        _logLine('${e.type}${e.message != null ? ": ${e.message}" : ""}');
      });
    } catch (_) {
      _eventSub = null;
    }

    try {
      final s = await client.up();
      if (!mounted) return;
      setState(() {
        _status = s;
        _busy = false;
      });
      _logLine('connected to ${s.nodeId ?? host} (v${s.version})');
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = '$e';
        _busy = false;
      });
      _logLine('connect failed');
    }
  }

  Future<void> _disconnect() async {
    await _client?.down();
    if (!mounted) return;
    setState(() => _status = null);
    _logLine('disconnected');
  }

  Future<void> _ping() async {
    if (_client == null) return;
    try {
      final ms = await _client!.ping();
      if (!mounted) return;
      setState(() => _latencyMs = ms);
      _logLine('ping ${ms}ms');
    } catch (e) {
      _logLine('ping failed: $e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Row(
          children: [
            Icon(Icons.hub_outlined),
            SizedBox(width: 8),
            Text('Weftlink'),
          ],
        ),
        actions: [
          if (_busy)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 16),
              child: Center(
                child: SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                ),
              ),
            ),
          IconButton(
            tooltip: 'Reconnect',
            onPressed: _busy ? null : _connect,
            icon: const Icon(Icons.refresh),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: Row(
        children: [
          SizedBox(width: 340, child: _leftRail(context)),
          const VerticalDivider(width: 1),
          Expanded(child: _rightPane(context)),
        ],
      ),
    );
  }

  // -- Left rail -------------------------------------------------------------

  Widget _leftRail(BuildContext context) {
    final theme = Theme.of(context);
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('DAEMON', style: theme.textTheme.labelSmall),
        const SizedBox(height: 8),
        TextField(
          controller: _hostCtrl,
          decoration: const InputDecoration(
            labelText: 'Host',
            isDense: true,
            border: OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: _portCtrl,
          decoration: const InputDecoration(
            labelText: 'Port',
            isDense: true,
            border: OutlineInputBorder(),
          ),
          keyboardType: TextInputType.number,
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: FilledButton.icon(
                onPressed: _busy ? null : _connect,
                icon: const Icon(Icons.link),
                label: const Text('Connect'),
              ),
            ),
            const SizedBox(width: 8),
            IconButton.outlined(
              tooltip: 'Disconnect',
              onPressed: _status == null ? null : _disconnect,
              icon: const Icon(Icons.link_off),
            ),
          ],
        ),
        const SizedBox(height: 8),
        OutlinedButton.icon(
          onPressed: _status == null ? null : _ping,
          icon: const Icon(Icons.speed),
          label: Text(_latencyMs == null ? 'Ping' : 'Ping · ${_latencyMs}ms'),
        ),
        const Divider(height: 32),
        Text('NODE', style: theme.textTheme.labelSmall),
        const SizedBox(height: 8),
        _kv('Status', _status == null ? 'Offline' : 'Connected'),
        _kv('Node ID', _status?.nodeId ?? '-'),
        _kv('Version', _status?.version ?? '-'),
        if (_error != null) ...[
          const SizedBox(height: 12),
          Card(
            color: theme.colorScheme.errorContainer,
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Text(
                _error!,
                style: TextStyle(color: theme.colorScheme.onErrorContainer),
              ),
            ),
          ),
        ],
      ],
    );
  }

  Widget _kv(String k, String v) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 90,
            child: Text(k, style: const TextStyle(fontSize: 12, color: Colors.grey)),
          ),
          Expanded(
            child: Text(
              v,
              style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
            ),
          ),
        ],
      ),
    );
  }

  // -- Right pane ------------------------------------------------------------

  Widget _rightPane(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: Text('Peers (${_peers.length})',
              style: theme.textTheme.titleMedium),
        ),
        Expanded(
          flex: 3,
          child: _peers.isEmpty
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.blur_on,
                          size: 40, color: theme.colorScheme.outline),
                      const SizedBox(height: 8),
                      Text('No peers woven in',
                          style: TextStyle(color: theme.colorScheme.outline)),
                      const SizedBox(height: 4),
                      Text('Pairing (knot) lands in M1',
                          style: theme.textTheme.bodySmall),
                    ],
                  ),
                )
              : ListView.separated(
                  itemCount: _peers.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (c, i) {
                    final p = _peers[i];
                    return ListTile(
                      leading: Icon(p.role == 'warp'
                          ? Icons.dns_outlined
                          : Icons.smartphone_outlined),
                      title: Text(p.displayName ?? p.id.id),
                      subtitle: Text(
                          '${p.role} · ${p.isWarped ? "paired" : "unpaired"}'),
                      trailing: p.isWefted
                          ? Icon(Icons.wifi_tethering,
                              color: theme.colorScheme.primary)
                          : const SizedBox.shrink(),
                    );
                  },
                ),
        ),
        const Divider(height: 1),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
          child: Text('Activity', style: theme.textTheme.titleMedium),
        ),
        Expanded(
          flex: 2,
          child: Container(
            color: theme.colorScheme.surfaceContainerLowest,
            child: ListView.builder(
              reverse: true,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              itemCount: _log.length,
              itemBuilder: (c, i) => Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Text(
                  _log[i],
                  style: const TextStyle(
                      fontFamily: 'monospace', fontSize: 12),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}
