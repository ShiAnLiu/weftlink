// Transport: TCP listener + length-prefixed JSON framing (Weft Protocol)
package main

import "encoding/binary"
import "encoding/json"
import "fmt"
import "io"
import "log"
import "net"

const maxFrameSize = 1 << 20 // 1 MiB

// -- Listener ----------------------------------------------------------------

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
