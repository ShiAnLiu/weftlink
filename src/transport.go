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
		var head struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(payload, &head) == nil && head.Type == "file_offer" {
			if err = handleIncomingFile(conn, payload); err != nil {
				log.Println("file offer error:", err)
				return
			}
			continue
		}
		if json.Unmarshal(payload, &head) == nil && head.Type == "exec_request" {
			if err = handleExecRequest(conn, payload); err != nil {
				log.Println("exec request error:", err)
				return
			}
			continue
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
	Type            string       `json:"type"`
	Code            string       `json:"code"`
	Host            string       `json:"host"`
	Nonce           string       `json:"nonce"`
	TS              int64        `json:"ts"`
	Node            pairNodeWire `json:"node"`
	Peer            string       `json:"peer"`
	Path            string       `json:"path"`
	Dir             string       `json:"dir"`
	From            string       `json:"from"`
	TimeoutSeconds  int          `json:"timeout_seconds"`
	ExecID          string       `json:"exec_id"`
	Command         string       `json:"command"`
	Shell           string       `json:"shell"`
	Text            string       `json:"text"`
	Mime            string       `json:"mime"`
	FromNodeID      string       `json:"from_node_id"`
	FromFingerprint string       `json:"from_fingerprint"`
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

type peerStatusResultResponse struct {
	Type   string          `json:"type"`
	OK     bool            `json:"ok"`
	Status *statusResponse `json:"status,omitempty"`
	Error  string          `json:"error,omitempty"`
}

type sendFileResponse struct {
	Type     string `json:"type"`
	OK       bool   `json:"ok"`
	Checksum string `json:"checksum,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Error    string `json:"error,omitempty"`
}

type recvBeginResponse struct {
	Type           string `json:"type"`
	OK             bool   `json:"ok"`
	Dir            string `json:"dir,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
	Error          string `json:"error,omitempty"`
}

type recvWaitResponse struct {
	Type       string `json:"type"`
	OK         bool   `json:"ok"`
	Pending    bool   `json:"pending,omitempty"`
	Path       string `json:"path,omitempty"`
	Name       string `json:"name,omitempty"`
	Size       int64  `json:"size,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
	FromNodeID string `json:"from_node_id,omitempty"`
	Error      string `json:"error,omitempty"`
}

type recvCancelResponse struct {
	Type string `json:"type"`
	OK   bool   `json:"ok"`
}

type runCommandResponse struct {
	Type     string `json:"type"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Error    string `json:"error,omitempty"`
}

type execRequestsResponse struct {
	Type     string            `json:"type"`
	OK       bool              `json:"ok"`
	Requests []execRequestInfo `json:"requests"`
	Error    string            `json:"error,omitempty"`
}

type execActionResponse struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type shutdownResponse struct {
	Type string `json:"type"`
	OK   bool   `json:"ok"`
}

type untieResultResponse struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

type clipboardBeginResponse struct {
	Type             string `json:"type"`
	OK               bool   `json:"ok"`
	From             string `json:"from,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
	Error            string `json:"error,omitempty"`
}

type clipboardStateResponse struct {
	Type             string `json:"type"`
	OK               bool   `json:"ok"`
	Active           bool   `json:"active"`
	From             string `json:"from,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds,omitempty"`
}

type clipboardResultResponse struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Length int    `json:"length,omitempty"`
	Error  string `json:"error,omitempty"`
}

type clipboardAckResponse struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Length int    `json:"length,omitempty"`
	Error  string `json:"error,omitempty"`
}

type clipboardContentResponse struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Text  string `json:"text,omitempty"`
	Mime  string `json:"mime,omitempty"`
	Error string `json:"error,omitempty"`
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
	case "peer_status":
		var status statusResponse
		status, err = peerStatus(req.Peer)
		if err != nil {
			resp = peerStatusResultResponse{Type: "peer_status", OK: false, Error: err.Error()}
			break
		}
		resp = peerStatusResultResponse{Type: "peer_status", OK: true, Status: &status}
	case "send_file":
		var result sendFileResult
		result, err = sendFileToPeer(req.Peer, req.Path)
		if err != nil {
			resp = sendFileResponse{Type: "send_result", OK: false, Error: err.Error()}
			break
		}
		resp = sendFileResponse{
			Type:     "send_result",
			OK:       true,
			Checksum: result.Checksum,
			Size:     result.Size,
		}
	case "recv_begin":
		var fromID = req.From
		if fromID != "" {
			var k Knot
			k, err = resolvePairedPeer(fromID)
			if err != nil {
				resp = recvBeginResponse{Type: "recv_begin", OK: false, Error: err.Error()}
				break
			}
			fromID = k.PeerID
		}
		var dir string
		var timeout time.Duration
		dir, timeout, err = beginReceive(req.Dir, fromID,
			time.Duration(req.TimeoutSeconds)*time.Second)
		if err != nil {
			resp = recvBeginResponse{Type: "recv_begin", OK: false, Error: err.Error()}
			break
		}
		resp = recvBeginResponse{
			Type:           "recv_begin",
			OK:             true,
			Dir:            dir,
			TimeoutSeconds: int(timeout.Seconds()),
		}
	case "recv_wait":
		var result receiveResult
		result, err = waitReceive(time.Duration(req.TimeoutSeconds) * time.Second)
		if err != nil {
			resp = recvWaitResponse{Type: "recv_result", OK: false, Error: err.Error()}
			break
		}
		resp = recvWaitResponse{
			Type:       "recv_result",
			OK:         result.OK,
			Path:       result.Path,
			Name:       result.Name,
			Size:       result.Size,
			Checksum:   result.Checksum,
			FromNodeID: result.FromNodeID,
			Error:      result.Error,
		}
	case "recv_poll":
		var result receiveResult
		var done bool
		result, done = pollReceive()
		if !done {
			resp = recvWaitResponse{Type: "recv_result", OK: true, Pending: true}
			break
		}
		resp = recvWaitResponse{
			Type:       "recv_result",
			OK:         result.OK,
			Path:       result.Path,
			Name:       result.Name,
			Size:       result.Size,
			Checksum:   result.Checksum,
			FromNodeID: result.FromNodeID,
			Error:      result.Error,
		}
	case "recv_cancel":
		cancelReceive()
		resp = recvCancelResponse{Type: "recv_cancelled", OK: true}
	case "run_command":
		var result execResultMessage
		result, err = runRemoteCommand(req.Peer, req.Command, req.TimeoutSeconds, req.Shell)
		if err != nil {
			resp = runCommandResponse{Type: "run_result", OK: false, Error: err.Error()}
			break
		}
		resp = runCommandResponse{
			Type:     "run_result",
			OK:       result.OK,
			ExitCode: result.ExitCode,
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			Error:    result.Error,
		}
	case "exec_requests":
		resp = execRequestsResponse{
			Type:     "exec_requests",
			OK:       true,
			Requests: listPendingExecs(),
		}
	case "exec_approve":
		if err = decideExec(req.ExecID, true); err != nil {
			resp = execActionResponse{Type: "exec_action", OK: false, Error: err.Error()}
			break
		}
		resp = execActionResponse{Type: "exec_action", OK: true}
	case "exec_deny":
		if err = decideExec(req.ExecID, false); err != nil {
			resp = execActionResponse{Type: "exec_action", OK: false, Error: err.Error()}
			break
		}
		resp = execActionResponse{Type: "exec_action", OK: true}
	case "clipboard_update":
		var length int
		length, err = applyClipboardUpdate(req.Text, req.Mime, req.FromNodeID, req.FromFingerprint)
		if err != nil {
			resp = clipboardAckResponse{Type: "clipboard_ack", OK: false, Error: err.Error()}
			break
		}
		resp = clipboardAckResponse{Type: "clipboard_ack", OK: true, Length: length}
	case "clipboard_request":
		var text, mime string
		text, mime, err = handleClipboardRequest(req.FromNodeID, req.FromFingerprint)
		if err != nil {
			resp = clipboardContentResponse{Type: "clipboard_content", OK: false, Error: err.Error()}
			break
		}
		resp = clipboardContentResponse{Type: "clipboard_content", OK: true, Text: text, Mime: mime}
	case "clipboard_begin":
		var fromID = req.From
		if fromID != "" {
			var k Knot
			k, err = resolvePairedPeer(fromID)
			if err != nil {
				resp = clipboardBeginResponse{Type: "clipboard_begin", OK: false, Error: err.Error()}
				break
			}
			fromID = k.PeerID
		}
		var until time.Time
		until, err = beginClipboard(fromID, time.Duration(req.TimeoutSeconds)*time.Second)
		if err != nil {
			resp = clipboardBeginResponse{Type: "clipboard_begin", OK: false, Error: err.Error()}
			break
		}
		resp = clipboardBeginResponse{
			Type:             "clipboard_begin",
			OK:               true,
			From:             fromID,
			ExpiresInSeconds: int(time.Until(until).Seconds()),
		}
	case "clipboard_end":
		cancelClipboard()
		resp = clipboardBeginResponse{Type: "clipboard_end", OK: true}
	case "clipboard_state":
		var active bool
		var from string
		var left int
		active, from, left = clipboardStatus()
		resp = clipboardStateResponse{
			Type:             "clipboard_state",
			OK:               true,
			Active:           active,
			From:             from,
			ExpiresInSeconds: left,
		}
	case "clipboard_push":
		var length int
		length, err = pushClipboardToPeer(req.Peer)
		if err != nil {
			resp = clipboardResultResponse{Type: "clipboard_result", OK: false, Error: err.Error()}
			break
		}
		resp = clipboardResultResponse{Type: "clipboard_result", OK: true, Length: length}
	case "clipboard_pull":
		var length int
		length, err = pullClipboardFromPeer(req.Peer)
		if err != nil {
			resp = clipboardResultResponse{Type: "clipboard_result", OK: false, Error: err.Error()}
			break
		}
		resp = clipboardResultResponse{Type: "clipboard_result", OK: true, Length: length}
	case "unpair_request":
		if err = handleUnpairRequest(req.FromNodeID, req.FromFingerprint); err != nil {
			resp = unpairAck{Type: "unpair_ack", OK: false, Error: err.Error()}
			break
		}
		resp = unpairAck{Type: "unpair_ack", OK: true}
	case "untie":
		if err = untiePeer(req.Peer); err != nil {
			resp = untieResultResponse{Type: "untie_result", OK: false, Error: err.Error()}
			break
		}
		resp = untieResultResponse{Type: "untie_result", OK: true}
	case "shutdown":
		if !isLoopbackAddr(remote) {
			resp = errorResponse{Type: "error", Message: "shutdown is only allowed from loopback"}
			break
		}
		requestShutdown()
		resp = shutdownResponse{Type: "shutting_down", OK: true}
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
