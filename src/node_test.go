package main

import (
	"os"
	"testing"
)

func TestNodeIdentityStable(t *testing.T) {
	var oldHome = os.Getenv("WEFTLINK_HOME")
	var oldName = os.Getenv("WEFTLINK_NAME")
	var oldNode = localNode
	t.Cleanup(func() {
		os.Setenv("WEFTLINK_HOME", oldHome)
		os.Setenv("WEFTLINK_NAME", oldName)
		localNode = oldNode
	})

	var dir = t.TempDir()
	os.Setenv("WEFTLINK_HOME", dir)
	os.Setenv("WEFTLINK_NAME", "node-a")

	if err := initNodeIdentity(); err != nil {
		t.Fatalf("initNodeIdentity: %v", err)
	}
	var first = localNode
	if len(first.nodeID) != 12 {
		t.Errorf("node id length = %d, want 12", len(first.nodeID))
	}
	if !validFingerprint(first.fingerprint) {
		t.Errorf("fingerprint %q is not 64 lowercase hex chars", first.fingerprint)
	}
	if first.displayName != "node-a" {
		t.Errorf("display name = %q, want node-a", first.displayName)
	}
	if first.version != coreVersion() {
		t.Errorf("version = %q, want %q", first.version, coreVersion())
	}
	if _, err := os.Stat(nodeKeyPath()); err != nil {
		t.Fatalf("node key not persisted: %v", err)
	}

	if err := initNodeIdentity(); err != nil {
		t.Fatalf("second initNodeIdentity: %v", err)
	}
	if localNode.nodeID != first.nodeID {
		t.Errorf("node id changed across restarts: %q -> %q", first.nodeID, localNode.nodeID)
	}
	if localNode.fingerprint != first.fingerprint {
		t.Errorf("fingerprint changed across restarts")
	}
}
