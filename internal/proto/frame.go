// Package proto implements Weft Protocol framing shared by daemon and CLI.
//
// Frame format (per protocol/SPEC.md):
//
//	[4-byte big-endian length][JSON payload]
package proto

import "encoding/binary"
import "fmt"
import "io"
import "net"

// MaxFrameSize is the upper bound accepted for a single frame (1 MiB).
const MaxFrameSize = 1 << 20

// ReadFrame reads one length-prefixed frame into *payload.
func ReadFrame(conn net.Conn, payload *[]byte) error {
	var lenBuf [4]byte
	var n int
	var err error
	n, err = io.ReadFull(conn, lenBuf[:])
	if err != nil {
		return err
	}
	_ = n
	var length = binary.BigEndian.Uint32(lenBuf[:])
	if length > MaxFrameSize {
		return fmt.Errorf("frame too large: %d", length)
	}
	var buf = make([]byte, length)
	n, err = io.ReadFull(conn, buf)
	if err != nil {
		return err
	}
	_ = n
	*payload = buf
	return nil
}

// WriteFrame writes data as a single length-prefixed frame.
func WriteFrame(conn net.Conn, data []byte) error {
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

// Request is the minimal envelope every Weft verb shares.
type Request struct {
	Type string `json:"type"`
}
