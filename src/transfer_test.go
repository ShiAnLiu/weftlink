package main

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"weftlink/internal/proto"
)

func TestSafeFileName(t *testing.T) {
	var cases = map[string]string{
		"../../etc/passwd":   "passwd",
		`C:\temp\report.pdf`: "report.pdf",
		"normal.txt":         "normal.txt",
		"  spaced.txt  ":     "spaced.txt",
		"bad<>name?.txt":     "bad__name_.txt",
		"":                   "weftlink-file",
		"..":                 "weftlink-file",
		"CON":                "_CON",
		"what\x00ever.txt":   "whatever.txt",
	}
	for input, want := range cases {
		if got := safeFileName(input); got != want {
			t.Errorf("safeFileName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestUniquePath(t *testing.T) {
	var dir = t.TempDir()
	var first = filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(first, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	var got = uniquePath(dir, "photo.jpg")
	if got != filepath.Join(dir, "photo (1).jpg") {
		t.Errorf("uniquePath = %q, want photo (1).jpg", got)
	}
}

func TestFileChecksum(t *testing.T) {
	var path = filepath.Join(t.TempDir(), "data.bin")
	var content = []byte("weftlink transfer test")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	var sum = sha256.Sum256(content)
	var want = hex.EncodeToString(sum[:])
	var got, err = fileChecksum(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("checksum = %q, want %q", got, want)
	}
}

func TestReceiveState(t *testing.T) {
	cancelReceive()
	t.Cleanup(cancelReceive)

	var dir, timeout, err = beginReceive(t.TempDir(), "peer-a", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if timeout <= 0 || dir == "" {
		t.Fatalf("unexpected receive state: dir=%q timeout=%v", dir, timeout)
	}

	if _, _, err = takeReceive("peer-b"); err == nil {
		t.Fatal("takeReceive accepted the wrong peer")
	}
	var receiveDir string
	var ch chan receiveResult
	receiveDir, ch, err = takeReceive("peer-a")
	if err != nil {
		t.Fatal(err)
	}
	if receiveDir != dir {
		t.Errorf("receive dir = %q, want %q", receiveDir, dir)
	}
	finishReceive(ch, receiveResult{OK: true, Name: "a.txt", Path: filepath.Join(dir, "a.txt")})

	result, err := waitReceive(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Name != "a.txt" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestReceiveStateTimeout(t *testing.T) {
	cancelReceive()
	t.Cleanup(cancelReceive)

	if _, _, err := beginReceive(t.TempDir(), "", 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := waitReceive(2 * time.Second); err == nil {
		t.Fatal("waitReceive should time out")
	}
	if time.Since(start) > time.Second {
		t.Errorf("timeout took too long: %v", time.Since(start))
	}
}

func testTLSListener(t *testing.T) net.Listener {
	t.Helper()
	var cert, err = tls.X509KeyPair([]byte(tlsCertPEM), []byte(tlsKeyPEM))
	if err != nil {
		t.Skipf("no test cert: %v", err)
	}
	var ln net.Listener
	ln, err = tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cert}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return ln
}

func TestSendFileToPeer(t *testing.T) {
	resetKnotTable()
	t.Cleanup(resetKnotTable)

	var content = []byte(strings.Repeat("weftlink-file-chunk-", 5000))
	var source = filepath.Join(t.TempDir(), "payload.bin")
	if err := os.WriteFile(source, content, 0600); err != nil {
		t.Fatal(err)
	}
	var sum = sha256.Sum256(content)
	var wantChecksum = hex.EncodeToString(sum[:])

	var ln = testTLSListener(t)
	var received = make(chan []byte, 1)
	go func() {
		var conn net.Conn
		var err error
		conn, err = ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		var offer fileOfferMessage
		if readWire(conn, &offer, 2*time.Second) != nil {
			return
		}
		if offer.Checksum != wantChecksum || offer.Size != int64(len(content)) {
			return
		}
		if writeWire(conn, fileAcceptMessage{Type: "file_accept", FileID: offer.FileID, Accept: true}, 2*time.Second) != nil {
			return
		}
		var data []byte
		for {
			var chunk fileChunkMessage
			if readWire(conn, &chunk, 2*time.Second) != nil {
				return
			}
			data = append(data, chunk.Data...)
			if chunk.Final {
				break
			}
		}
		received <- data
		_ = writeWire(conn, fileDoneMessage{Type: "file_done", FileID: offer.FileID, OK: true}, 2*time.Second)
	}()

	knotMu.Lock()
	knots["receiver"] = Knot{
		PeerID:      "receiver",
		DisplayName: "Receiver",
		Address:     ln.Addr().String(),
		Fingerprint: strings.Repeat("a", 64),
	}
	knotMu.Unlock()

	result, err := sendFileToPeer("receiver", source)
	if err != nil {
		t.Fatalf("sendFileToPeer: %v", err)
	}
	if result.Checksum != wantChecksum || result.Size != int64(len(content)) {
		t.Errorf("unexpected result: %+v", result)
	}
	select {
	case got := <-received:
		if string(got) != string(content) {
			t.Errorf("receiver got %d bytes, want %d", len(got), len(content))
		}
	case <-time.After(3 * time.Second):
		t.Fatal("receiver did not get the file")
	}
}

func TestHandleIncomingFile(t *testing.T) {
	resetKnotTable()
	resetReceiver := func() { cancelReceive() }
	resetReceiver()
	t.Cleanup(func() {
		resetKnotTable()
		resetReceiver()
	})

	var sender = makeTestIdentity("sender", "Sender")
	useIdentity(t, makeTestIdentity("receiver", "Receiver"))

	knotMu.Lock()
	knots[sender.nodeID] = Knot{
		PeerID:      sender.nodeID,
		DisplayName: sender.displayName,
		Fingerprint: sender.fingerprint,
	}
	knotMu.Unlock()

	var targetDir = t.TempDir()
	var _, _, err = beginReceive(targetDir, sender.nodeID, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	var content = []byte(strings.Repeat("receiver-side-data-", 2000))
	var sum = sha256.Sum256(content)
	var checksum = hex.EncodeToString(sum[:])

	var ln = testTLSListener(t)
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
		_ = handleIncomingFile(conn, payload)
	}()

	var client *proto.Client
	client, err = proto.DialTLS(ln.Addr().String(), true, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	var offer = fileOfferMessage{
		Type:            "file_offer",
		FileID:          "file-1",
		Name:            "hello.txt",
		Size:            int64(len(content)),
		Checksum:        checksum,
		FromNodeID:      sender.nodeID,
		FromFingerprint: sender.fingerprint,
	}
	if err = client.SendTimeout(offer, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	var accept fileAcceptMessage
	if err = client.ReceiveTimeout(&accept, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if !accept.Accept {
		t.Fatalf("offer rejected: %s", accept.Error)
	}
	if err = client.SendTimeout(fileChunkMessage{
		Type: "file_chunk", FileID: "file-1", Offset: 0, Data: content,
	}, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if err = client.SendTimeout(fileChunkMessage{
		Type: "file_chunk", FileID: "file-1", Offset: int64(len(content)), Final: true,
	}, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	var done fileDoneMessage
	if err = client.ReceiveTimeout(&done, 2*time.Second); err != nil {
		t.Fatal(err)
	}
	if !done.OK {
		t.Fatalf("transfer failed: %s", done.Error)
	}

	result, err := waitReceive(3 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("receive failed: %s", result.Error)
	}
	if result.Checksum != checksum || result.Size != int64(len(content)) {
		t.Errorf("unexpected result: %+v", result)
	}
	got, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("saved content mismatch")
	}
}
