/// Weftlink UI — home page.
///
/// Talks to the core only through [CoreClient], so it works identically
/// for the desktop (loopback) and mobile (gomobile) implementations.
library;

import 'dart:async';

import 'package:core_client/core_client.dart';
import 'package:flutter/material.dart';

class HomePage extends StatefulWidget {
  final CoreClient client;

  const HomePage({super.key, required this.client});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  NodeStatus? _status;
  String? _error;
  bool _busy = false;
  List<Peer> _peers = [];
  StreamSubscription<List<Peer>>? _peerSub;

  @override
  void initState() {
    super.initState();
    _connect();
    // Verbs beyond `status` may not be wired in every core implementation
    // yet (e.g. loopback watchPeers lands in M1). Stay resilient.
    try {
      _peerSub = widget.client.watchPeers().listen((p) {
        if (mounted) setState(() => _peers = p);
      });
    } catch (_) {
      _peerSub = null;
    }
  }

  @override
  void dispose() {
    _peerSub?.cancel();
    widget.client.down();
    super.dispose();
  }

  Future<void> _connect() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final s = await widget.client.up();
      if (!mounted) return;
      setState(() {
        _status = s;
        _busy = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = '$e';
        _busy = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Weftlink'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Reconnect',
            onPressed: _busy ? null : _connect,
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (_busy) const LinearProgressIndicator(),
            if (_error != null)
              Card(
                color: Colors.red.shade50,
                child: ListTile(
                  leading: const Icon(Icons.error_outline),
                  title: const Text('Not connected'),
                  subtitle: Text(_error!),
                ),
              ),
            if (_status != null)
              Card(
                child: Column(
                  children: [
                    _row('Status', _status!.connected ? 'Connected' : 'Down'),
                    _row('Node ID', _status!.nodeId ?? '-'),
                    _row('Version', _status!.version),
                  ],
                ),
              ),
            const SizedBox(height: 16),
            Text('Peers (${_peers.length})',
                style: Theme.of(context).textTheme.titleMedium),
            Expanded(
              child: _peers.isEmpty
                  ? const Center(child: Text('No peers woven in'))
                  : ListView.builder(
                      itemCount: _peers.length,
                      itemBuilder: (c, i) {
                        final p = _peers[i];
                        return ListTile(
                          leading: Icon(p.role == 'warp'
                              ? Icons.dns
                              : Icons.smartphone),
                          title: Text(p.displayName ?? p.id.id),
                          subtitle: Text('${p.role} · ${p.isWarped ? "paired" : "unpaired"}'),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _row(String label, String value) {
    return ListTile(
      dense: true,
      title: Text(label),
      trailing: Text(value, style: const TextStyle(fontWeight: FontWeight.bold)),
    );
  }
}