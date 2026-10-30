package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func makeTestIdentity(nodeID string, name string) nodeIdentity {
	var sum = sha256.Sum256([]byte(nodeID))
	return nodeIdentity{
		nodeID:      nodeID,
		publicKey:   sum[:],
		fingerprint: hex.EncodeToString(sum[:]),
		displayName: name,
		role:        RoleWarp,
		version:     coreVersion(),
	}
}

func useIdentity(t *testing.T, id nodeIdentity) {
	t.Helper()
	var old = localNode
	localNode = id
	t.Cleanup(func() { localNode = old })
}

func resetPeerTable() {
	peersMu.Lock()
	peers = map[string]Peer{}
	peersMu.Unlock()
}

func resetKnotTable() {
	knotMu.Lock()
	knots = map[string]Knot{}
	knotMu.Unlock()
}

func resetPendingCode() {
	pendingMu.Lock()
	pendingCode = ""
	pendingUntil = time.Time{}
	pendingTries = 0
	pendingMu.Unlock()
}
