package main

import (
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"weftlink/internal/proto"
)

func TestGeneratePairCode(t *testing.T) {
	var seen = map[string]bool{}
	for i := 0; i < 50; i++ {
		var code, err = generatePairCode()
		if err != nil {
			t.Fatalf("generatePairCode: %v", err)
		}
		if !validPairCode(code) {
			t.Fatalf("generated invalid code %q", code)
		}
		seen[code] = true
	}
	if len(seen) < 45 {
		t.Errorf("only %d unique codes out of 50", len(seen))
	}
}

func TestValidPairCode(t *testing.T) {
	var cases = map[string]bool{
		"ABC123":  true,
		"abcdef":  false,
		"ABC12":   false,
		"ABC1234": false,
		"ABC-12":  false,
		"":        false,
	}
	for code, want := range cases {
		if got := validPairCode(code); got != want {
			t.Errorf("validPairCode(%q) = %v, want %v", code, got, want)
		}
	}
}

func TestConsumePairCode(t *testing.T) {
	resetPendingCode()
	t.Cleanup(resetPendingCode)

	var code, until, err = startPairCode()
	if err != nil {
		t.Fatalf("startPairCode: %v", err)
	}
	if time.Until(until) <= 0 {
		t.Error("pair code expiry should be in the future")
	}
	var wrong = "AAAAAA"
	if code == wrong {
		wrong = "BBBBBB"
	}
	for i := 0; i < pairCodeMaxTries; i++ {
		if consumePairCode(wrong) {
			t.Fatal("wrong code must not be accepted")
		}
	}
	if consumePairCode(code) {
		t.Error("code should be cleared after too many wrong attempts")
	}
}

func TestConsumePairCodeExpires(t *testing.T) {
	resetPendingCode()
	t.Cleanup(resetPendingCode)

	var code, _, err = startPairCode()
	if err != nil {
		t.Fatalf("startPairCode: %v", err)
	}
	pendingMu.Lock()
	pendingUntil = time.Now().Add(-time.Second)
	pendingMu.Unlock()
	if consumePairCode(code) {
		t.Error("expired code must not be accepted")
	}
}

func TestSaveLoadKnots(t *testing.T) {
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
		PeerID:      "peer1",
		DisplayName: "Peer One",
		Role:        "warp",
		Fingerprint: strings.Repeat("a", 64),
		Address:     "127.0.0.1:7801",
		PairedAt:    time.Now().UnixMilli(),
	}
	if err := saveKnot(k); err != nil {
		t.Fatalf("saveKnot: %v", err)
	}
	if !hasKnot("peer1") {
		t.Fatal("knot not in memory after save")
	}
	if p, ok := findPeer("peer1"); !ok || !p.Paired {
		t.Fatalf("peer not marked paired after save: %+v", p)
	}
	if _, err := os.Stat(knotsDir() + string(os.PathSeparator) + "peer1.json"); err != nil {
		t.Fatalf("knot file missing: %v", err)
	}

	resetKnotTable()
	resetPeerTable()
	if err := loadKnots(); err != nil {
		t.Fatalf("loadKnots: %v", err)
	}
	if !hasKnot("peer1") {
		t.Error("knot not restored from disk")
	}
	if p, ok := findPeer("peer1"); !ok || !p.Paired || p.Address != "127.0.0.1:7801" {
		t.Errorf("peer not restored: %+v", p)
	}
}

func TestValidNodeIdentity(t *testing.T) {
	var id = makeTestIdentity("node-x", "X")
	var pubHex = hex.EncodeToString(id.publicKey)
	if !validNodeIdentity(id.fingerprint, pubHex) {
		t.Error("valid identity rejected")
	}
	if validNodeIdentity(strings.Repeat("0", 64), pubHex) {
		t.Error("mismatched fingerprint accepted")
	}
	if validNodeIdentity(id.fingerprint, "not-hex") {
		t.Error("invalid public key accepted")
	}
}

func TestHandlePairRequest(t *testing.T) {
	var oldHome = os.Getenv("WEFTLINK_HOME")
	os.Setenv("WEFTLINK_HOME", t.TempDir())
	resetKnotTable()
	resetPeerTable()
	resetPendingCode()
	t.Cleanup(func() {
		os.Setenv("WEFTLINK_HOME", oldHome)
		resetKnotTable()
		resetPeerTable()
		resetPendingCode()
	})

	useIdentity(t, makeTestIdentity("node-a", "A"))
	var code, _, err = startPairCode()
	if err != nil {
		t.Fatalf("startPairCode: %v", err)
	}
	var remote = makeTestIdentity("node-b", "B")
	var node = pairNodeWire{
		NodeID:      remote.nodeID,
		DisplayName: remote.displayName,
		Role:        "warp",
		Version:     remote.version,
		Fingerprint: remote.fingerprint,
		PublicKey:   hex.EncodeToString(remote.publicKey),
		Port:        listenPort,
	}

	var confirm pairNodeWire
	confirm, err = handlePairRequest(code, node, "192.168.1.9")
	if err != nil {
		t.Fatalf("handlePairRequest: %v", err)
	}
	if confirm.NodeID != "node-a" || confirm.Fingerprint != localNode.fingerprint {
		t.Errorf("unexpected confirm identity: %+v", confirm)
	}
	var k Knot
	var ks = listKnots()
	if len(ks) != 1 {
		t.Fatalf("len(knots) = %d, want 1", len(ks))
	}
	k = ks[0]
	if k.PeerID != "node-b" || k.Address != "192.168.1.9:7801" {
		t.Errorf("unexpected knot: %+v", k)
	}
	if _, err = handlePairRequest(code, node, "192.168.1.9"); err == nil {
		t.Error("replayed pair code must be rejected")
	}
}

func TestPairWithPeer(t *testing.T) {
	var oldHome = os.Getenv("WEFTLINK_HOME")
	os.Setenv("WEFTLINK_HOME", t.TempDir())
	resetKnotTable()
	resetPeerTable()
	t.Cleanup(func() {
		os.Setenv("WEFTLINK_HOME", oldHome)
		resetKnotTable()
		resetPeerTable()
	})

	var cert tls.Certificate
	var certErr error
	cert, certErr = tls.X509KeyPair([]byte(tlsCertPEM), []byte(tlsKeyPEM))
	if certErr != nil {
		t.Skipf("no test cert: %v", certErr)
	}
	var ln net.Listener
	var err error
	ln, err = tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatalf("TLS listen: %v", err)
	}
	defer ln.Close()

	var peer = makeTestIdentity("node-a", "A")
	var got = make(chan pairRequest, 1)
	go func() {
		var conn net.Conn
		var acceptErr error
		conn, acceptErr = ln.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()
		var payload []byte
		if proto.ReadFrame(conn, &payload) != nil {
			return
		}
		var req pairRequest
		if json.Unmarshal(payload, &req) != nil {
			return
		}
		got <- req
		var raw []byte
		raw, _ = json.Marshal(pairResponse{
			Type: "pair_confirm",
			Node: pairNodeWire{
				NodeID:      peer.nodeID,
				DisplayName: peer.displayName,
				Role:        "warp",
				Version:     peer.version,
				Fingerprint: peer.fingerprint,
				PublicKey:   hex.EncodeToString(peer.publicKey),
				Port:        listenPort,
			},
		})
		_ = proto.WriteFrame(conn, raw)
	}()

	useIdentity(t, makeTestIdentity("node-b", "B"))
	var k, pairErr = pairWithPeer(ln.Addr().String(), "ABC123")
	if pairErr != nil {
		t.Fatalf("pairWithPeer: %v", pairErr)
	}
	if k.PeerID != "node-a" {
		t.Errorf("peer id = %q, want node-a", k.PeerID)
	}
	if k.Address != ln.Addr().String() {
		t.Errorf("address = %q, want %q", k.Address, ln.Addr().String())
	}
	if k.PairedAt == 0 {
		t.Error("PairedAt should be set")
	}
	if k.Fingerprint != peer.fingerprint {
		t.Errorf("fingerprint = %q, want %q", k.Fingerprint, peer.fingerprint)
	}

	var req pairRequest
	select {
	case req = <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not receive pair request")
	}
	if req.Type != "pair_request" || req.Code != "ABC123" {
		t.Errorf("unexpected request: %+v", req)
	}
	if req.Node.NodeID != "node-b" {
		t.Errorf("request node id = %q, want node-b", req.Node.NodeID)
	}
}

func TestNormalizeAddress(t *testing.T) {
	var cases = map[string]string{
		"192.168.1.5":      "192.168.1.5:7801",
		"192.168.1.5:9000": "192.168.1.5:9000",
		"::1":              "[::1]:7801",
		"[::1]":            "[::1]:7801",
		"[::1]:9000":       "[::1]:9000",
	}
	for in, want := range cases {
		var got, err = normalizeAddress(in, listenPort)
		if err != nil {
			t.Errorf("normalizeAddress(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("normalizeAddress(%q) = %q, want %q", in, got, want)
		}
	}
}
