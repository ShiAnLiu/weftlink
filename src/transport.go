// Transport: TCP listener + length-prefixed JSON framing (Weft Protocol)
package main

import "crypto/tls"
import "encoding/binary"
import "encoding/json"
import "fmt"
import "io"
import "log"
import "net"

const maxFrameSize = 1 << 20 // 1 MiB

// -- TLS config -------------------------------------------------------------

// loadTLSConfig loads (or generates) the self-signed server cert.
func loadTLSConfig(cfg **tls.Config) error {
	var certPath = "etc/weftlinkd-cert.pem"
	var keyPath = "etc/weftlinkd-key.pem"
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
		resp, err = handleRequest(payload)
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

type wireMsg struct {
	Type string `json:"type"`
}

func handleRequest(payload []byte) ([]byte, error) {
	var req wireMsg
	var err error
	err = json.Unmarshal(payload, &req)
	if err != nil {
		return nil, fmt.Errorf("bad json: %w", err)
	}
	var resp any
	switch req.Type {
	case "hello":
		resp = map[string]any{
			"type":    "hello_ack",
			"node_id": "weftlinkd-windows",
			"version": coreVersion(),
			"role":    "warp",
		}
	case "status":
		resp = map[string]any{
			"type":    "status",
			"peers":   peerCount(),
			"version": coreVersion(),
		}
	case "ping":
		resp = map[string]any{
			"type": "pong",
		}
	default:
		resp = map[string]any{
			"type":    "error",
			"message": "unknown type: " + req.Type,
		}
	}
	var data []byte
	data, err = json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// -- Frame I/O (per SPEC.md: [4-byte BE length][JSON payload]) ---------------

func readFrame(conn net.Conn) ([]byte, error) {
	var lenBuf [4]byte
	var n int
	var err error
	n, err = io.ReadFull(conn, lenBuf[:])
	if err != nil {
		return nil, err
	}
	_ = n
	var length = binary.BigEndian.Uint32(lenBuf[:])
	if length > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", length)
	}
	var buf = make([]byte, length)
	n, err = io.ReadFull(conn, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(conn net.Conn, data []byte) error {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	var n int
	var err error
	n, err = conn.Write(lenBuf[:])
	if err != nil {
		return err
	}
	_ = n
	n, err = conn.Write(data)
	if err != nil {
		return err
	}
	_ = n
	return nil
}
