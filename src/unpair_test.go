package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestRemoveKnot(t *testing.T) {
	var oldHome = os.Getenv("WEFTLINK_HOME")
	os.Setenv("WEFTLINK_HOME", t.TempDir())
	resetKnotTable()
	resetPeerTable()
	t.Cleanup(func() {
		os.Setenv("WEFTLINK_HOME", oldHome)
		resetKnotTable()
		resetPeerTable()
	})

	var k = Knot{
		PeerID:      "peer-x",
		DisplayName: "Peer X",
		Fingerprint: strings.Repeat("a", 64),
		Address:     "127.0.0.1:7801",
		PairedAt:    time.Now().UnixMilli(),
	}
	if err := saveKnot(k); err != nil {
		t.Fatal(err)
	}
	if !hasKnot("peer-x") {
		t.Fatal("knot missing after save")
	}
	if err := removeKnot("peer-x"); err != nil {
		t.Fatal(err)
	}
	if hasKnot("peer-x") {
		t.Error("knot still in memory")
	}
	if _, err := os.Stat(knotsDir() + string(os.PathSeparator) + "peer-x.json"); !os.IsNotExist(err) {
		t.Error("knot file still exists")
	}
	if p, ok := findPeer("peer-x"); ok && p.Paired {
		t.Error("peer should be unpaired")
	}
}

func TestHandleUnpairRequest(t *testing.T) {
	resetKnotTable()
	t.Cleanup(resetKnotTable)

	var fingerprint = strings.Repeat("b", 64)
	knotMu.Lock()
	knots["peer-y"] = Knot{PeerID: "peer-y", Fingerprint: fingerprint}
	knotMu.Unlock()

	if err := handleUnpairRequest("peer-y", strings.Repeat("c", 64)); err == nil {
		t.Error("wrong fingerprint should be rejected")
	}
	if !hasKnot("peer-y") {
		t.Error("rejected request must not remove the knot")
	}
	if err := handleUnpairRequest("peer-y", fingerprint); err != nil {
		t.Fatal(err)
	}
	if hasKnot("peer-y") {
		t.Error("knot should be removed")
	}
}
