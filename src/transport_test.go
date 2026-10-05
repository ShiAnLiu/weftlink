// Integration test: loopback Weft Protocol echo (hello/status/ping)
package main

import "crypto/tls"
import "encoding/json"
import "net"
import "testing"

// -- Helpers -----------------------------------------------------------------

var tlsCertPEM = `-----BEGIN CERTIFICATE-----
MIIBPDCB76ADAgECAhRK+x8obN7IgM4C6bXCJ69kWb1D6zAFBgMrZXAwFDESMBAG
A1UEAwwJd2VmdGxpbmtkMB4XDTI2MTAwMjA5MTg1MloXDTM2MDkyOTA5MTg1Mlow
FDESMBAGA1UEAwwJd2VmdGxpbmtkMCowBQYDK2VwAyEAo0wlWFso9JUIpA5s5qv9
RQgfFGrffVVXyimJKgthDD2jUzBRMB0GA1UdDgQWBBQLJpdGyq2t18Xk8p2r8pSh
ofLCljAfBgNVHSMEGDAWgBQLJpdGyq2t18Xk8p2r8pShofLCljAPBgNVHRMBAf8E
BTADAQH/MAUGAytlcANBAB8UcuAw0aSk8BqLDqlcDelex88/4L1nTNOrsLJTZDK8
07swe0i/SCLJkUrAj9ADUe9YPM3o+xV+a8rwAjBC9Q8=
-----END CERTIFICATE-----
`

var tlsKeyPEM = `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIFsR0OMq5JUcs6OLArPd+Bwb4YphLFHsfj8NPIb4CvUl
-----END PRIVATE KEY-----
`

// startTestListener spins up a TCP listener on an ephemeral port.
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

// makeTLSTestListener spins up a TLS-wrapped listener on ephemeral port.
func makeTLSTestListener(t *testing.T) string {
	var cert tls.Certificate
	var certErr error
	cert, certErr = tls.X509KeyPair([]byte(tlsCertPEM), []byte(tlsKeyPEM))
	if certErr != nil {
		t.Skipf("no server cert: %v", certErr)
		return ""
	}
	var srvCfg = new(tls.Config)
	srvCfg.Certificates = []tls.Certificate{cert}

	var ln net.Listener
	var lnErr error
	ln, lnErr = tls.Listen("tcp", ":0", srvCfg)
	if lnErr != nil {
		t.Fatalf("TLS listen failed: %v", lnErr)
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

// -- TCP tests ---------------------------------------------------------------

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
}

// -- TLS tests ---------------------------------------------------------------

func TestTLSLoopbackHello(t *testing.T) {
	var addr = makeTLSTestListener(t)
	if addr == "" {
		return
	}

	var cliCfg = new(tls.Config)
	cliCfg.InsecureSkipVerify = true

	var conn *tls.Conn
	var dialErr error
	conn, dialErr = tls.Dial("tcp", addr, cliCfg)
	if dialErr != nil {
		t.Fatalf("TLS dial failed: %v", dialErr)
	}
	defer conn.Close()

	var resp = sendRecv(t, conn, `{"type":"hello"}`)
	if resp["type"] != "hello_ack" {
		t.Errorf("type = %v, want hello_ack", resp["type"])
	}
	if resp["version"] != coreVersion() {
		t.Errorf("version = %v, want %v", resp["version"], coreVersion())
	}
}
