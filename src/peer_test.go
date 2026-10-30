package main

import (
	"testing"
	"time"
)

func TestPeerTableUpsertAndSort(t *testing.T) {
	resetPeerTable()
	t.Cleanup(resetPeerTable)

	if !upsertPeer(Peer{PeerID: "b", DisplayName: "beta"}) {
		t.Error("first insert should report new")
	}
	if upsertPeer(Peer{PeerID: "b", DisplayName: "beta"}) {
		t.Error("second insert should report existing")
	}
	upsertPeer(Peer{PeerID: "a", DisplayName: "alpha", Paired: true})
	upsertPeer(Peer{PeerID: "c", DisplayName: "gamma"})

	var got = listPeers()
	if len(got) != 3 {
		t.Fatalf("len(peers) = %d, want 3", len(got))
	}
	if got[0].PeerID != "a" {
		t.Errorf("paired peer should sort first, got %q", got[0].PeerID)
	}
	if got[1].PeerID != "b" || got[2].PeerID != "c" {
		t.Errorf("unexpected order: %q, %q", got[1].PeerID, got[2].PeerID)
	}
	if peerCount() != 3 {
		t.Errorf("peerCount = %d, want 3", peerCount())
	}
	if pairedPeerCount() != 1 {
		t.Errorf("pairedPeerCount = %d, want 1", pairedPeerCount())
	}
}

func TestPeerPruneKeepsPaired(t *testing.T) {
	resetPeerTable()
	t.Cleanup(resetPeerTable)

	upsertPeer(Peer{PeerID: "stale", LastSeen: 0})
	upsertPeer(Peer{PeerID: "fresh", LastSeen: time.Now().UnixMilli()})
	upsertPeer(Peer{PeerID: "paired", LastSeen: 0, Paired: true})

	prunePeers(time.Second)

	if _, ok := findPeer("stale"); ok {
		t.Error("stale unpaired peer should be pruned")
	}
	if _, ok := findPeer("fresh"); !ok {
		t.Error("fresh peer should be kept")
	}
	if _, ok := findPeer("paired"); !ok {
		t.Error("paired peer should be kept regardless of age")
	}
}

func TestMarkPeerPaired(t *testing.T) {
	resetPeerTable()
	t.Cleanup(resetPeerTable)

	upsertPeer(Peer{PeerID: "p1"})
	markPeerPaired("p1")
	var p, ok = findPeer("p1")
	if !ok || !p.Paired {
		t.Fatalf("peer should be marked paired: %+v", p)
	}
}

func TestBeaconMessageRoundTrip(t *testing.T) {
	var msg = currentBeacon()
	if msg.Type != "beacon" || msg.NodeID == "" || msg.Port != listenPort {
		t.Errorf("invalid beacon: %+v", msg)
	}
	if msg.Fingerprint != localNode.fingerprint {
		t.Errorf("beacon fingerprint = %q, want %q", msg.Fingerprint, localNode.fingerprint)
	}
}
