// File transfer: paired peer -> paired peer over a dedicated TLS connection.
//
// The receiver must explicitly opt in with `weftlink recv` before an offer is
// accepted. Nothing is written to the destination until the transfer verifies.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"weftlink/internal/proto"
)

const (
	fileChunkSize         = 64 * 1024
	maxFileSize           = int64(4) << 30 // 4 GiB
	offerTimeout          = 30 * time.Second
	transferFrameTimeout  = 2 * time.Minute
	defaultReceiveTimeout = 5 * time.Minute
	maxReceiveTimeout     = 30 * time.Minute
)

// -- Wire messages -----------------------------------------------------------

type fileOfferMessage struct {
	Type            string `json:"type"`
	FileID          string `json:"file_id"`
	Name            string `json:"name"`
	Size            int64  `json:"size"`
	Mime            string `json:"mime,omitempty"`
	Checksum        string `json:"checksum"`
	FromNodeID      string `json:"from_node_id"`
	FromFingerprint string `json:"from_fingerprint"`
}

type fileAcceptMessage struct {
	Type   string `json:"type"`
	FileID string `json:"file_id"`
	Accept bool   `json:"accept"`
	Error  string `json:"error,omitempty"`
}

type fileChunkMessage struct {
	Type   string `json:"type"`
	FileID string `json:"file_id"`
	Offset int64  `json:"offset"`
	Data   []byte `json:"data,omitempty"`
	Final  bool   `json:"final"`
}

type fileDoneMessage struct {
	Type   string `json:"type"`
	FileID string `json:"file_id"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

type sendFileResult struct {
	Checksum string `json:"checksum"`
	Size     int64  `json:"size"`
}

// -- Receiver consent --------------------------------------------------------

type receiveResult struct {
	OK         bool   `json:"ok"`
	Path       string `json:"path,omitempty"`
	Name       string `json:"name,omitempty"`
	Size       int64  `json:"size,omitempty"`
	Checksum   string `json:"checksum,omitempty"`
	FromNodeID string `json:"from_node_id,omitempty"`
	Error      string `json:"error,omitempty"`
}

type receiveState struct {
	active bool
	dir    string
	from   string
	until  time.Time
	ch     chan receiveResult
}

var (
	recvMu sync.Mutex
	recv   *receiveState
)

func beginReceive(dir string, fromID string, timeout time.Duration) (string, time.Duration, error) {
	if dir == "" {
		dir = defaultReceiveDir()
	}
	var abs, err = filepath.Abs(dir)
	if err != nil {
		return "", 0, err
	}
	if timeout <= 0 {
		timeout = defaultReceiveTimeout
	}
	if timeout > maxReceiveTimeout {
		timeout = maxReceiveTimeout
	}
	if err = os.MkdirAll(abs, 0755); err != nil {
		return "", 0, err
	}

	recvMu.Lock()
	defer recvMu.Unlock()
	if recv != nil && recv.active {
		return "", 0, fmt.Errorf("already waiting for a file")
	}
	recv = &receiveState{
		active: true,
		dir:    abs,
		from:   fromID,
		until:  time.Now().Add(timeout),
		ch:     make(chan receiveResult, 1),
	}
	return abs, timeout, nil
}

func takeReceive(fromID string) (string, chan receiveResult, error) {
	recvMu.Lock()
	defer recvMu.Unlock()
	if recv == nil || !recv.active {
		return "", nil, fmt.Errorf("peer is not accepting files (run 'weftlink recv' on the target)")
	}
	if time.Now().After(recv.until) {
		recv.active = false
		return "", nil, fmt.Errorf("receive window expired (run 'weftlink recv' again)")
	}
	if recv.from != "" && recv.from != fromID {
		return "", nil, fmt.Errorf("not accepting files from this peer")
	}
	recv.active = false
	return recv.dir, recv.ch, nil
}

func waitReceive(timeout time.Duration) (receiveResult, error) {
	if timeout <= 0 {
		timeout = defaultReceiveTimeout
	}
	recvMu.Lock()
	if recv == nil || recv.ch == nil {
		recvMu.Unlock()
		return receiveResult{}, fmt.Errorf("no receive session (run 'weftlink recv' first)")
	}
	var ch = recv.ch
	var until = recv.until
	recvMu.Unlock()

	var wait = time.Until(until)
	if wait <= 0 || wait > timeout {
		wait = timeout
	}
	var timer = time.NewTimer(wait)
	defer timer.Stop()

	select {
	case result := <-ch:
		recvMu.Lock()
		if recv != nil && recv.ch == ch {
			recv = nil
		}
		recvMu.Unlock()
		return result, nil
	case <-timer.C:
		recvMu.Lock()
		if recv != nil && recv.ch == ch {
			recv = nil
		}
		recvMu.Unlock()
		return receiveResult{}, fmt.Errorf("timed out waiting for a file")
	}
}

func cancelReceive() {
	recvMu.Lock()
	recv = nil
	recvMu.Unlock()
}

func finishReceive(ch chan receiveResult, result receiveResult) {
	if ch == nil {
		return
	}
	select {
	case ch <- result:
	default:
	}
}

func defaultReceiveDir() string {
	var home, err = os.UserHomeDir()
	if err != nil || home == "" {
		return "weftlink-downloads"
	}
	return filepath.Join(home, "Downloads", "Weftlink")
}

// -- Helpers -----------------------------------------------------------------

func newTransferID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func safeFileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(strings.TrimSpace(name))
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 0x7f {
			return -1
		}
		switch r {
		case '<', '>', ':', '"', '|', '?', '*':
			return '_'
		}
		return r
	}, name)
	name = strings.Trim(name, ". ")
	if name == "" || name == "." || name == ".." {
		return "weftlink-file"
	}
	var runes = []rune(name)
	if len(runes) > 200 {
		name = string(runes[len(runes)-200:])
	}
	var base = strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4",
		"COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3",
		"LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	return name
}

func uniquePath(dir string, name string) string {
	var candidate = filepath.Join(dir, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	var ext = filepath.Ext(name)
	var base = strings.TrimSuffix(name, ext)
	for i := 1; ; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func fileChecksum(path string) (string, error) {
	var f, err = os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var h = sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func validChecksum(sum string) bool {
	if len(sum) != 64 {
		return false
	}
	for i := 0; i < len(sum); i++ {
		var c = sum[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}

func writeWire(conn net.Conn, value any, timeout time.Duration) error {
	var payload, err = json.Marshal(value)
	if err != nil {
		return err
	}
	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	}
	return proto.WriteFrame(conn, payload)
}

func readWire(conn net.Conn, value any, timeout time.Duration) error {
	if timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
	}
	var payload []byte
	if err := proto.ReadFrame(conn, &payload); err != nil {
		return err
	}
	return json.Unmarshal(payload, value)
}

func resolvePairedPeer(query string) (Knot, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return Knot{}, fmt.Errorf("peer is required")
	}
	if k, ok := getKnot(query); ok {
		return k, nil
	}
	for _, k := range listKnots() {
		if strings.EqualFold(k.DisplayName, query) || strings.EqualFold(k.PeerID, query) {
			return k, nil
		}
	}
	return Knot{}, fmt.Errorf("unknown paired peer %q (see 'weftlink ls')", query)
}

func peerAddress(k Knot) (string, error) {
	if k.Address != "" {
		return k.Address, nil
	}
	if p, ok := findPeer(k.PeerID); ok && p.Address != "" {
		return p.Address, nil
	}
	return "", fmt.Errorf("no known address for %q; run 'weftlink ls'", k.DisplayName)
}

// -- Inbound file ------------------------------------------------------------

func handleIncomingFile(conn net.Conn, payload []byte) error {
	var offer fileOfferMessage
	if err := json.Unmarshal(payload, &offer); err != nil {
		return err
	}
	var reject = func(message string) error {
		log.Println("file offer rejected from", offer.FromNodeID, ":", message)
		return writeWire(conn, fileAcceptMessage{
			Type:   "file_accept",
			FileID: offer.FileID,
			Accept: false,
			Error:  message,
		}, offerTimeout)
	}

	if offer.FileID == "" || offer.Name == "" {
		return reject("invalid offer")
	}
	if offer.Size < 0 || offer.Size > maxFileSize {
		return reject("file too large")
	}
	offer.Checksum = strings.ToLower(offer.Checksum)
	if !validChecksum(offer.Checksum) {
		return reject("invalid checksum")
	}
	if k, ok := getKnot(offer.FromNodeID); !ok ||
		!strings.EqualFold(k.Fingerprint, offer.FromFingerprint) {
		return reject("unknown or unpaired sender")
	}

	var name = safeFileName(offer.Name)
	if name == "" {
		return reject("invalid file name")
	}
	var dir, resultCh, err = takeReceive(offer.FromNodeID)
	if err != nil {
		return reject(err.Error())
	}

	var finalPath = uniquePath(dir, name)
	var tmp, tmpErr = os.CreateTemp(dir, ".weftlink-*.part")
	if tmpErr != nil {
		finishReceive(resultCh, receiveResult{OK: false, Error: tmpErr.Error()})
		return reject("cannot create destination file")
	}
	var cleanup = func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}

	if err = writeWire(conn, fileAcceptMessage{
		Type:   "file_accept",
		FileID: offer.FileID,
		Accept: true,
	}, offerTimeout); err != nil {
		cleanup()
		finishReceive(resultCh, receiveResult{OK: false, Error: err.Error()})
		return err
	}

	log.Println("receiving file", name, "from", offer.FromNodeID, "("+fmt.Sprint(offer.Size)+" bytes)")

	var fail = func(message string) error {
		cleanup()
		_ = writeWire(conn, fileDoneMessage{
			Type:   "file_done",
			FileID: offer.FileID,
			OK:     false,
			Error:  message,
		}, offerTimeout)
		finishReceive(resultCh, receiveResult{OK: false, Error: message, FromNodeID: offer.FromNodeID})
		return nil
	}

	var hasher = sha256.New()
	var written int64
	for {
		var chunk fileChunkMessage
		if err = readWire(conn, &chunk, transferFrameTimeout); err != nil {
			return fail("transfer interrupted: " + err.Error())
		}
		if chunk.FileID != offer.FileID {
			return fail("unexpected file id")
		}
		if chunk.Offset != written {
			return fail("unexpected chunk offset")
		}
		if int64(len(chunk.Data)) > fileChunkSize {
			return fail("chunk too large")
		}
		if written+int64(len(chunk.Data)) > offer.Size {
			return fail("file grew beyond the offer")
		}
		if len(chunk.Data) > 0 {
			if _, err = tmp.Write(chunk.Data); err != nil {
				return fail("cannot write destination: " + err.Error())
			}
			_, _ = hasher.Write(chunk.Data)
			written += int64(len(chunk.Data))
		}
		if chunk.Final {
			break
		}
	}

	if written != offer.Size {
		return fail(fmt.Sprintf("size mismatch: got %d, want %d", written, offer.Size))
	}
	var sum = hex.EncodeToString(hasher.Sum(nil))
	if sum != offer.Checksum {
		return fail("checksum mismatch")
	}
	if err = tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return fail("cannot finalize destination: " + err.Error())
	}
	if err = os.Rename(tmp.Name(), finalPath); err != nil {
		_ = os.Remove(tmp.Name())
		return fail("cannot move file into place: " + err.Error())
	}

	_ = writeWire(conn, fileDoneMessage{
		Type:   "file_done",
		FileID: offer.FileID,
		OK:     true,
	}, offerTimeout)
	finishReceive(resultCh, receiveResult{
		OK:         true,
		Path:       finalPath,
		Name:       name,
		Size:       written,
		Checksum:   sum,
		FromNodeID: offer.FromNodeID,
	})
	log.Println("received file", name, "->", finalPath)
	return nil
}

// -- Outbound file -----------------------------------------------------------

func sendFileToPeer(peerQuery string, sourcePath string) (sendFileResult, error) {
	var k, err = resolvePairedPeer(peerQuery)
	if err != nil {
		return sendFileResult{}, err
	}
	var address string
	address, err = peerAddress(k)
	if err != nil {
		return sendFileResult{}, err
	}

	var abs, absErr = filepath.Abs(sourcePath)
	if absErr != nil {
		return sendFileResult{}, absErr
	}
	var stat, statErr = os.Stat(abs)
	if statErr != nil {
		return sendFileResult{}, statErr
	}
	if !stat.Mode().IsRegular() {
		return sendFileResult{}, fmt.Errorf("not a regular file: %s", abs)
	}
	if stat.Size() > maxFileSize {
		return sendFileResult{}, fmt.Errorf("file too large (max %d bytes)", maxFileSize)
	}
	abs = filepath.Clean(abs)

	var checksum string
	checksum, err = fileChecksum(abs)
	if err != nil {
		return sendFileResult{}, err
	}
	var fileID string
	fileID, err = newTransferID()
	if err != nil {
		return sendFileResult{}, err
	}

	var client *proto.Client
	client, err = proto.DialTLS(address, true, 10*time.Second)
	if err != nil {
		return sendFileResult{}, fmt.Errorf("%s: %w", address, err)
	}
	defer client.Close()

	var offer = fileOfferMessage{
		Type:            "file_offer",
		FileID:          fileID,
		Name:            filepath.Base(abs),
		Size:            stat.Size(),
		Mime:            mime.TypeByExtension(filepath.Ext(abs)),
		Checksum:        checksum,
		FromNodeID:      localNode.nodeID,
		FromFingerprint: localNode.fingerprint,
	}
	if err = client.SendTimeout(offer, offerTimeout); err != nil {
		return sendFileResult{}, err
	}
	var accept fileAcceptMessage
	if err = client.ReceiveTimeout(&accept, offerTimeout); err != nil {
		return sendFileResult{}, err
	}
	if accept.Type != "file_accept" || !accept.Accept {
		var message = accept.Error
		if message == "" {
			message = "peer rejected the file offer"
		}
		return sendFileResult{}, fmt.Errorf("%s", message)
	}

	log.Println("sending file", offer.Name, "to", k.PeerID, "("+fmt.Sprint(stat.Size())+" bytes)")

	var f *os.File
	f, err = os.Open(abs)
	if err != nil {
		return sendFileResult{}, err
	}
	defer f.Close()

	var buf = make([]byte, fileChunkSize)
	var offset int64
	for {
		var n int
		n, err = f.Read(buf)
		if n > 0 {
			var chunk = fileChunkMessage{
				Type:   "file_chunk",
				FileID: fileID,
				Offset: offset,
				Data:   append([]byte(nil), buf[:n]...),
			}
			if err = client.SendTimeout(chunk, transferFrameTimeout); err != nil {
				return sendFileResult{}, err
			}
			offset += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return sendFileResult{}, err
		}
	}
	if err = client.SendTimeout(fileChunkMessage{
		Type:   "file_chunk",
		FileID: fileID,
		Offset: offset,
		Final:  true,
	}, transferFrameTimeout); err != nil {
		return sendFileResult{}, err
	}

	var done fileDoneMessage
	if err = client.ReceiveTimeout(&done, transferFrameTimeout); err != nil {
		return sendFileResult{}, err
	}
	if done.Type != "file_done" || !done.OK {
		var message = done.Error
		if message == "" {
			message = "peer did not confirm the file"
		}
		return sendFileResult{}, fmt.Errorf("%s", message)
	}
	log.Println("sent file", offer.Name, "to", k.PeerID)
	return sendFileResult{Checksum: checksum, Size: stat.Size()}, nil
}

// -- Remote status -----------------------------------------------------------

func peerStatus(peerQuery string) (statusResponse, error) {
	var k, err = resolvePairedPeer(peerQuery)
	if err != nil {
		return statusResponse{}, err
	}
	var address string
	address, err = peerAddress(k)
	if err != nil {
		return statusResponse{}, err
	}
	var client *proto.Client
	client, err = proto.DialTLS(address, true, 10*time.Second)
	if err != nil {
		return statusResponse{}, fmt.Errorf("%s: %w", address, err)
	}
	defer client.Close()

	if err = client.SendTimeout(map[string]string{"type": "status"}, 10*time.Second); err != nil {
		return statusResponse{}, err
	}
	var status statusResponse
	if err = client.ReceiveTimeout(&status, 10*time.Second); err != nil {
		return statusResponse{}, err
	}
	if status.Type != "status" || status.NodeID == "" {
		return statusResponse{}, fmt.Errorf("peer returned an unexpected status")
	}
	return status, nil
}
