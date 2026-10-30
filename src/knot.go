// Knot: pairing (6-character code exchange) and paired-peer persistence.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"weftlink/internal/proto"
)

const (
	pairCodeTTL      = 2 * time.Minute
	pairCodeMaxTries = 5
	pairDialTimeout  = 3 * time.Second
)

var errPairDenied = errors.New("invalid or expired pair code")

// Knot is a paired peer persisted in ~/.weftlink/knots/<peer_id>.json.
type Knot struct {
	PeerID      string `json:"peer_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
	Address     string `json:"address"`
	PairedAt    int64  `json:"paired_at"`
}

var (
	knotMu sync.RWMutex
	knots  = map[string]Knot{}

	pendingMu    sync.Mutex
	pendingCode  string
	pendingUntil time.Time
	pendingTries int
)

// pairNodeWire is the node description exchanged during pairing.
type pairNodeWire struct {
	NodeID      string `json:"node_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
	Port        int    `json:"port"`
}

// pairRequest is sent by the joining node to the waiting node.
type pairRequest struct {
	Type string       `json:"type"`
	Code string       `json:"code"`
	Node pairNodeWire `json:"node"`
}

// pairResponse is the waiting node's answer.
type pairResponse struct {
	Type    string       `json:"type"`
	Node    pairNodeWire `json:"node"`
	Message string       `json:"message"`
}

func localPairNode() pairNodeWire {
	return pairNodeWire{
		NodeID:      localNode.nodeID,
		DisplayName: localNode.displayName,
		Role:        "warp",
		Version:     localNode.version,
		Fingerprint: localNode.fingerprint,
		PublicKey:   hex.EncodeToString(localNode.publicKey),
		Port:        listenPort,
	}
}

// startPairCode creates and stores a new single-use code.
func startPairCode() (string, time.Time, error) {
	var code, err = generatePairCode()
	if err != nil {
		return "", time.Time{}, err
	}
	var until = time.Now().Add(pairCodeTTL)
	pendingMu.Lock()
	pendingCode = code
	pendingUntil = until
	pendingTries = 0
	pendingMu.Unlock()
	return code, until, nil
}

// consumePairCode atomically validates and invalidates a pending code.
func consumePairCode(code string) bool {
	pendingMu.Lock()
	defer pendingMu.Unlock()
	if pendingCode == "" {
		return false
	}
	if time.Now().After(pendingUntil) {
		pendingCode = ""
		return false
	}
	if len(code) != len(pendingCode) {
		pendingTries++
		if pendingTries >= pairCodeMaxTries {
			pendingCode = ""
		}
		return false
	}
	if subtle.ConstantTimeCompare([]byte(code), []byte(pendingCode)) != 1 {
		pendingTries++
		if pendingTries >= pairCodeMaxTries {
			pendingCode = ""
		}
		return false
	}
	pendingCode = ""
	return true
}

// generatePairCode returns 6 unambiguous uppercase characters/digits.
func generatePairCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var code = make([]byte, pairCodeLen)
	var i = 0
	for i < pairCodeLen {
		var b [1]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", err
		}
		if int(b[0]) >= 256-256%len(alphabet) {
			continue
		}
		code[i] = alphabet[int(b[0])%len(alphabet)]
		i++
	}
	return string(code), nil
}

// validPairCode enforces the schema pattern ^[A-Z0-9]{6}$.
func validPairCode(code string) bool {
	if len(code) != pairCodeLen {
		return false
	}
	for i := 0; i < len(code); i++ {
		var c = code[i]
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

func validFingerprint(fp string) bool {
	if len(fp) != 64 {
		return false
	}
	for i := 0; i < len(fp); i++ {
		var c = fp[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}

// validNodeIdentity checks that the fingerprint is the hash of the supplied
// Ed25519 public key.
func validNodeIdentity(fingerprintHex string, publicKeyHex string) bool {
	if !validFingerprint(fingerprintHex) {
		return false
	}
	var raw, err = hex.DecodeString(publicKeyHex)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(fingerprint(raw)), []byte(fingerprintHex)) == 1
}

// -- Pairing -----------------------------------------------------------------

// handlePairRequest validates a joining node's code and returns our identity.
func handlePairRequest(code string, node pairNodeWire, remoteHost string) (pairNodeWire, error) {
	if node.NodeID == "" || node.NodeID == localNode.nodeID {
		return pairNodeWire{}, errPairDenied
	}
	if !validNodeIdentity(node.Fingerprint, node.PublicKey) {
		return pairNodeWire{}, errPairDenied
	}
	if !consumePairCode(code) {
		return pairNodeWire{}, errPairDenied
	}
	var address = remoteHost
	if node.Port > 0 && node.Port <= 65535 {
		address = net.JoinHostPort(remoteHost, strconv.Itoa(node.Port))
	}
	var k = Knot{
		PeerID:      node.NodeID,
		DisplayName: node.DisplayName,
		Role:        node.Role,
		Version:     node.Version,
		Fingerprint: node.Fingerprint,
		Address:     address,
		PairedAt:    time.Now().UnixMilli(),
	}
	if err := saveKnot(k); err != nil {
		return pairNodeWire{}, err
	}
	return localPairNode(), nil
}

// joinKnot pairs this daemon with a discovered peer (or an explicit address).
func joinKnot(code string, host string) (Knot, error) {
	if !validPairCode(code) {
		return Knot{}, fmt.Errorf("pair code must be 6 characters (A-Z, 0-9)")
	}
	var targets []string
	if host != "" {
		var addr, err = normalizeAddress(host, listenPort)
		if err != nil {
			return Knot{}, err
		}
		targets = []string{addr}
	} else {
		targets = peerAddresses()
	}
	if len(targets) == 0 {
		return Knot{}, fmt.Errorf("no peers discovered; run 'weftlink ls' or pass --peer <ip[:port]>")
	}

	var lastErr error
	for _, addr := range targets {
		var k Knot
		var err error
		k, err = pairWithPeer(addr, code)
		if err != nil {
			lastErr = err
			continue
		}
		if err = saveKnot(k); err != nil {
			return Knot{}, err
		}
		return k, nil
	}
	return Knot{}, fmt.Errorf("pairing failed: %v", lastErr)
}

func pairWithPeer(addr string, code string) (Knot, error) {
	var client *proto.Client
	var err error
	client, err = proto.DialTLS(addr, true, pairDialTimeout)
	if err != nil {
		return Knot{}, fmt.Errorf("%s: %w", addr, err)
	}
	defer client.Close()

	var resp pairResponse
	err = client.RoundTripTimeout(pairRequest{
		Type: "pair_request",
		Code: code,
		Node: localPairNode(),
	}, &resp, pairDialTimeout+time.Second)
	if err != nil {
		return Knot{}, fmt.Errorf("%s: %w", addr, err)
	}
	if resp.Type != "pair_confirm" {
		var msg = resp.Message
		if msg == "" {
			msg = "pairing rejected"
		}
		return Knot{}, fmt.Errorf("%s: %s", addr, msg)
	}
	if resp.Node.NodeID == "" || resp.Node.NodeID == localNode.nodeID {
		return Knot{}, fmt.Errorf("%s: invalid peer identity", addr)
	}
	if !validNodeIdentity(resp.Node.Fingerprint, resp.Node.PublicKey) {
		return Knot{}, fmt.Errorf("%s: invalid peer fingerprint", addr)
	}
	return Knot{
		PeerID:      resp.Node.NodeID,
		DisplayName: resp.Node.DisplayName,
		Role:        resp.Node.Role,
		Version:     resp.Node.Version,
		Fingerprint: resp.Node.Fingerprint,
		Address:     addr,
		PairedAt:    time.Now().UnixMilli(),
	}, nil
}

func normalizeAddress(raw string, defPort int) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty peer address")
	}
	var host, port, err = net.SplitHostPort(raw)
	if err == nil {
		if host == "" || port == "" {
			return "", fmt.Errorf("invalid peer address %q", raw)
		}
		return net.JoinHostPort(host, port), nil
	}
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		raw = raw[1 : len(raw)-1]
	}
	return net.JoinHostPort(raw, strconv.Itoa(defPort)), nil
}

// -- Knot persistence --------------------------------------------------------

func loadKnots() error {
	var dir = knotsDir()
	var entries []os.DirEntry
	var err error
	entries, err = os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var loaded = 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		var raw []byte
		raw, err = os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var k Knot
		if json.Unmarshal(raw, &k) != nil || k.PeerID == "" {
			continue
		}
		knotMu.Lock()
		knots[k.PeerID] = k
		knotMu.Unlock()
		upsertPeer(Peer{
			PeerID:      k.PeerID,
			DisplayName: k.DisplayName,
			Role:        k.Role,
			Version:     k.Version,
			Fingerprint: k.Fingerprint,
			Address:     k.Address,
			LastSeen:    k.PairedAt,
			Paired:      true,
		})
		loaded++
	}
	if loaded > 0 {
		log.Println("loaded", loaded, "knot(s)")
	}
	return nil
}

func knotToPeer(k Knot) Peer {
	return Peer{
		PeerID:      k.PeerID,
		DisplayName: k.DisplayName,
		Role:        k.Role,
		Version:     k.Version,
		Fingerprint: k.Fingerprint,
		Address:     k.Address,
		LastSeen:    k.PairedAt,
		Paired:      true,
	}
}

func saveKnot(k Knot) error {
	var dir = knotsDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	var raw, err = json.MarshalIndent(k, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	var path = filepath.Join(dir, k.PeerID+".json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		return err
	}
	knotMu.Lock()
	knots[k.PeerID] = k
	knotMu.Unlock()
	upsertPeer(knotToPeer(k))
	return nil
}

func hasKnot(peerID string) bool {
	knotMu.RLock()
	defer knotMu.RUnlock()
	var _, ok = knots[peerID]
	return ok
}

func getKnot(peerID string) (Knot, bool) {
	knotMu.RLock()
	defer knotMu.RUnlock()
	var k, ok = knots[peerID]
	return k, ok
}

func listKnots() []Knot {
	knotMu.RLock()
	defer knotMu.RUnlock()
	var out = make([]Knot, 0, len(knots))
	for _, k := range knots {
		out = append(out, k)
	}
	return out
}
