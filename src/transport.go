// Transport: TCP listener + length-prefixed JSON framing (Weft Protocol)
package main

import "crypto/tls"
import "encoding/json"
import "fmt"
import "io"
import "log"
import "net"
import "path/filepath"
import "time"

import "weftlink/internal/proto"

// -- TLS config -------------------------------------------------------------

// loadTLSConfig loads (or generates) the self-signed server cert.
func loadTLSConfig(cfg **tls.Config) error {
	var certPath = filepath.Join(tlsDir(), "weftlinkd-cert.pem")
	var keyPath = filepath.Join(tlsDir(), "weftlinkd-key.pem")
	var genErr error
	genErr = ensureTLSCert(certPath, keyPath)
	if genErr != nil {
		return genErr
	}
	var cert tls.Certificate
	var err error
	cert, err = tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return err
	}
	var c = new(tls.Config)
	c.Certificates = []tls.Certificate{cert}
	*cfg = c
	return nil
}

// newTestTLSConfig creates a client TLS config that skips verification (test).
func newTestTLSConfig(cfg **tls.Config) {
	var c = new(tls.Config)
	c.InsecureSkipVerify = true
	*cfg = c
}

// -- TLS Listener -----------------------------------------------------------

func listenTLSAndServe(port int) error {
	var cfg *tls.Config
	var err error
	err = loadTLSConfig(&cfg)
	if err != nil {
		return err
	}
	var addr = fmt.Sprintf(":%d", port)
	var ln net.Listener
	ln, err = tls.Listen("tcp", addr, cfg)
	if err != nil {
		return err
	}
	log.Println("weftlinkd TLS listening on", addr)
	serveLoop(ln)
	return nil
}

// -- TCP Listener (for tests / ephemeral port) ------------------------------

// listenAndServe starts the daemon TCP listener on :port and blocks forever.
func listenAndServe(port int) error {
	var ln net.Listener
	var err error
	err = listenTCP(port, &ln)
	if err != nil {
		return err
	}
	log.Println("weftlinkd listening on", ln.Addr())
	serveLoop(ln)
	return nil // unreachable
}

// listenTCP creates the listener (port 0 = ephemeral, for tests).
func listenTCP(port int, ln *net.Listener) error {
	var addr = fmt.Sprintf(":%d", port)
	var err error
	*ln, err = net.Listen("tcp", addr)
	return err
}

// serveLoop accepts connections forever.
func serveLoop(ln net.Listener) {
	for {
		var conn net.Conn
		var err error
		conn, err = ln.Accept()
		if err != nil {
			log.Println("accept error:", err)
			continue
		}
		log.Println("peer connected:", conn.RemoteAddr())
		go handleConn(conn)
	}
}

// -- Connection handler ------------------------------------------------------

func handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		var payload []byte
		var err error
		payload, err = readFrame(conn)
		if err != nil {
			if err != io.EOF {
				log.Println("read error:", err)
			}
			return
		}
		var resp []byte
		resp, err = handleRequest(conn.RemoteAddr(), payload)
		if err != nil {
			log.Println("handle error:", err)
			return
		}
		var wErr error
		wErr = writeFrame(conn, resp)
		if wErr != nil {
			log.Println("write error:", wErr)
			return
		}
	}
}

// -- Request dispatch --------------------------------------------------------

// request is the union of all daemon control verbs. Keeping one struct keeps
// the dispatcher small; unknown fields are ignored by encoding/json.
type request struct {
	Type  string       `json:"type"`
	Code  string       `json:"code"`
	Host  string       `json:"host"`
	Nonce string       `json:"nonce"`
	TS    int64        `json:"ts"`
	Node  pairNodeWire `json:"node"`
}

type helloAck struct {
	Type        string `json:"type"`
	NodeID      string `json:"node_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
}

type statusResponse struct {
	Type            string `json:"type"`
	NodeID          string `json:"node_id"`
	DisplayName     string `json:"display_name"`
	Role            string `json:"role"`
	Version         string `json:"version"`
	Fingerprint     string `json:"fingerprint"`
	Peers           []Peer `json:"peers"`
	PeerCount       int    `json:"peer_count"`
	PairedCount     int    `json:"paired_count"`
	DiscoveryPort   int    `json:"discovery_port"`
	DiscoveryActive bool   `json:"discovery_active"`
	BeaconsSent     int64  `json:"beacons_sent"`
	BeaconsReceived int64  `json:"beacons_received"`
	LastBeaconMs    int64  `json:"last_beacon_ms"`
}

type peersResponse struct {
	Type  string `json:"type"`
	Peers []Peer `json:"peers"`
}

type pongResponse struct {
	Type  string `json:"type"`
	Nonce string `json:"nonce,omitempty"`
	TS    int64  `json:"ts,omitempty"`
}

type knotCodeResponse struct {
	Type             string `json:"type"`
	Code             string `json:"code"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type knotResultResponse struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Peer  *Peer  `json:"peer,omitempty"`
	Error string `json:"error,omitempty"`
}

type pairConfirmResponse struct {
	Type string       `json:"type"`
	Node pairNodeWire `json:"node"`
}

type pairDeniedResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type errorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func handleRequest(remote net.Addr, payload []byte) ([]byte, error) {
	var req request
	var err error
	err = json.Unmarshal(payload, &req)
	if err != nil {
		return nil, fmt.Errorf("bad json: %w", err)
	}
	var resp any
	switch req.Type {
	case "hello":
		resp = helloAck{
			Type:        "hello_ack",
			NodeID:      localNode.nodeID,
			DisplayName: localNode.displayName,
			Role:        "warp",
			Version:     localNode.version,
			Fingerprint: localNode.fingerprint,
		}
	case "status":
		var peers = listPeers()
		var discovery = currentDiscoveryState()
		resp = statusResponse{
			Type:            "status",
			NodeID:          localNode.nodeID,
			DisplayName:     localNode.displayName,
			Role:            "warp",
			Version:         localNode.version,
			Fingerprint:     localNode.fingerprint,
			Peers:           peers,
			PeerCount:       len(peers),
			PairedCount:     pairedPeerCount(),
			DiscoveryPort:   discovery.Port,
			DiscoveryActive: discovery.Active,
			BeaconsSent:     discovery.Sent,
			BeaconsReceived: discovery.Received,
			LastBeaconMs:    discovery.LastReceivedMs,
		}
	case "peers":
		resp = peersResponse{Type: "peers", Peers: listPeers()}
	case "ping":
		resp = pongResponse{Type: "pong", Nonce: req.Nonce, TS: req.TS}
	case "knot_init":
		var code string
		var until time.Time
		code, until, err = startPairCode()
		if err != nil {
			resp = errorResponse{Type: "error", Message: "cannot generate pair code: " + err.Error()}
			break
		}
		resp = knotCodeResponse{
			Type:             "knot_code",
			Code:             code,
			ExpiresInSeconds: int(time.Until(until).Seconds()),
		}
	case "knot_join":
		var k Knot
		k, err = joinKnot(req.Code, req.Host)
		if err != nil {
			resp = knotResultResponse{Type: "knot_result", OK: false, Error: err.Error()}
			break
		}
		var p = knotToPeer(k)
		resp = knotResultResponse{Type: "knot_result", OK: true, Peer: &p}
	case "pair_request":
		var node pairNodeWire
		node, err = handlePairRequest(req.Code, req.Node, hostFromAddr(remote))
		if err != nil {
			resp = pairDeniedResponse{Type: "pair_denied", Message: err.Error()}
			break
		}
		resp = pairConfirmResponse{Type: "pair_confirm", Node: node}
	default:
		resp = errorResponse{Type: "error", Message: "unknown type: " + req.Type}
	}
	var data []byte
	data, err = json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func hostFromAddr(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	var host, _, err = net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}

// -- Frame I/O (per SPEC.md: [4-byte BE length][JSON payload]) ---------------

func readFrame(conn net.Conn) ([]byte, error) {
	var payload []byte
	var err = proto.ReadFrame(conn, &payload)
	return payload, err
}

func writeFrame(conn net.Conn, data []byte) error {
	return proto.WriteFrame(conn, data)
}
