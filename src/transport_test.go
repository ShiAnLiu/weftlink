// Integration test: loopback Weft Protocol echo (hello/status/ping)
package main

import "encoding/json"
import "fmt"
import "net"
import "testing"

// startTestListener spins up a listener on an ephemeral port and returns
// the address to dial.
func startTestListener(t *testing.T) string {
	var ln net.Listener
	var err error
	err = listenTCP(0, &ln)
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	go serveLoop(ln)
	return ln.Addr().String()
}

// sendRecv performs one request/response round trip over framed JSON.
func sendRecv(t *testing.T, conn net.Conn, req string) map[string]any {
	t.Helper()
	var wErr error
	wErr = writeFrame(conn, []byte(req))
	if wErr != nil {
		t.Fatalf("writeFrame failed: %v", wErr)
	}
	var payload []byte
	var rErr error
	payload, rErr = readFrame(conn)
	if rErr != nil {
		t.Fatalf("readFrame failed: %v", rErr)
	}
	var resp map[string]any
	var jErr error
	jErr = json.Unmarshal(payload, &resp)
	if jErr != nil {
		t.Fatalf("bad json response %q: %v", string(payload), jErr)
	}
	return resp
}

func TestLoopbackHello(t *testing.T) {
	var addr = startTestListener(t)
	var conn net.Conn
	var err error
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var resp = sendRecv(t, conn, `{"type":"hello"}`)
	if resp["type"] != "hello_ack" {
		t.Errorf("type = %v, want hello_ack", resp["type"])
	}
	if resp["node_id"] != "weftlinkd-windows" {
		t.Errorf("node_id = %v, want weftlinkd-windows", resp["node_id"])
	}
	if resp["role"] != "warp" {
		t.Errorf("role = %v, want warp", resp["role"])
	}
	if resp["version"] != coreVersion() {
		t.Errorf("version = %v, want %v", resp["version"], coreVersion())
	}
}

func TestLoopbackPingPong(t *testing.T) {
	var addr = startTestListener(t)
	var conn net.Conn
	var err error
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var resp = sendRecv(t, conn, `{"type":"ping"}`)
	if resp["type"] != "pong" {
		t.Errorf("type = %v, want pong", resp["type"])
	}
}

func TestLoopbackStatus(t *testing.T) {
	var addr = startTestListener(t)
	var conn net.Conn
	var err error
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var resp = sendRecv(t, conn, `{"type":"status"}`)
	if resp["type"] != "status" {
		t.Errorf("type = %v, want status", resp["type"])
	}
	var peers, ok = resp["peers"].(float64)
	if !ok {
		t.Fatalf("peers not numeric: %v", resp["peers"])
	}
	_ = peers
}

func TestLoopbackUnknownType(t *testing.T) {
	var addr = startTestListener(t)
	var conn net.Conn
	var err error
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var resp = sendRecv(t, conn, `{"type":"no_such_verb"}`)
	if resp["type"] != "error" {
		t.Errorf("type = %v, want error", resp["type"])
	}
}

func TestFrameRoundTrip(t *testing.T) {
	// Framing sanity: two consecutive frames on one pipe.
	var addr = startTestListener(t)
	var conn net.Conn
	var err error
	conn, err = net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	var r1 = sendRecv(t, conn, `{"type":"ping"}`)
	var r2 = sendRecv(t, conn, `{"type":"hello"}`)
	if r1["type"] != "pong" {
		t.Errorf("frame 1 type = %v, want pong", r1["type"])
	}
	if r2["type"] != "hello_ack" {
		t.Errorf("frame 2 type = %v, want hello_ack", r2["type"])
	}
	_ = fmt.Sprint("done")
}
