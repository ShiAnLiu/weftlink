// Clipboard sync: explicit opt-in policy plus manual push/pull between paired
// peers. Applying remote text requires `weftlink clipboard on` on the receiver.
package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"weftlink/internal/proto"
)

const (
	clipboardDefaultTimeout = 10 * time.Minute
	clipboardMaxTimeout     = 60 * time.Minute
	clipboardMaxText        = 128 * 1024
)

type clipboardUpdateMessage struct {
	Type            string `json:"type"`
	Text            string `json:"text"`
	Mime            string `json:"mime,omitempty"`
	TS              int64  `json:"ts"`
	FromNodeID      string `json:"from_node_id"`
	FromFingerprint string `json:"from_fingerprint"`
}

type clipboardAckMessage struct {
	Type   string `json:"type"`
	OK     bool   `json:"ok"`
	Length int    `json:"length,omitempty"`
	Error  string `json:"error,omitempty"`
}

type clipboardRequestMessage struct {
	Type            string `json:"type"`
	FromNodeID      string `json:"from_node_id"`
	FromFingerprint string `json:"from_fingerprint"`
}

type clipboardContentMessage struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Text  string `json:"text,omitempty"`
	Mime  string `json:"mime,omitempty"`
	Error string `json:"error,omitempty"`
}

type clipboardState struct {
	active bool
	from   string
	until  time.Time
}

var (
	clipMu    sync.Mutex
	clipState *clipboardState
)

func beginClipboard(fromID string, timeout time.Duration) (time.Time, error) {
	if timeout <= 0 {
		timeout = clipboardDefaultTimeout
	}
	if timeout > clipboardMaxTimeout {
		timeout = clipboardMaxTimeout
	}
	var until = time.Now().Add(timeout)
	clipMu.Lock()
	clipState = &clipboardState{active: true, from: fromID, until: until}
	clipMu.Unlock()
	return until, nil
}

func cancelClipboard() {
	clipMu.Lock()
	clipState = nil
	clipMu.Unlock()
}

func clipboardAllows(fromID string) error {
	clipMu.Lock()
	defer clipMu.Unlock()
	if clipState == nil || !clipState.active {
		return fmt.Errorf("clipboard sync is off (run 'weftlink clipboard on' on the target)")
	}
	if time.Now().After(clipState.until) {
		clipState = nil
		return fmt.Errorf("clipboard sync window expired")
	}
	if clipState.from != "" && clipState.from != fromID {
		return fmt.Errorf("not accepting clipboard from this peer")
	}
	return nil
}

func clipboardStatus() (bool, string, int) {
	clipMu.Lock()
	defer clipMu.Unlock()
	if clipState == nil || !clipState.active {
		return false, "", 0
	}
	if time.Now().After(clipState.until) {
		clipState = nil
		return false, "", 0
	}
	var left = int(time.Until(clipState.until).Seconds())
	if left < 0 {
		left = 0
	}
	return true, clipState.from, left
}

func applyClipboardUpdate(text string, mime string, fromID string, fingerprint string) (int, error) {
	if len(text) > clipboardMaxText {
		return 0, fmt.Errorf("clipboard text too large")
	}
	if k, ok := getKnot(fromID); !ok || !strings.EqualFold(k.Fingerprint, fingerprint) {
		return 0, fmt.Errorf("unknown or unpaired sender")
	}
	if err := clipboardAllows(fromID); err != nil {
		return 0, err
	}
	if err := writeLocalClipboard(text); err != nil {
		return 0, err
	}
	log.Println("clipboard updated from", fromID, "(", len(text), "chars )")
	return len(text), nil
}

func handleClipboardRequest(fromID string, fingerprint string) (string, string, error) {
	if k, ok := getKnot(fromID); !ok || !strings.EqualFold(k.Fingerprint, fingerprint) {
		return "", "", fmt.Errorf("unknown or unpaired sender")
	}
	if err := clipboardAllows(fromID); err != nil {
		return "", "", err
	}
	var text, err = readLocalClipboard()
	if err != nil {
		return "", "", err
	}
	if len(text) > clipboardMaxText {
		text = text[:clipboardMaxText]
	}
	return text, "text/plain", nil
}

func pushClipboardToPeer(peerQuery string) (int, error) {
	var k, err = resolvePairedPeer(peerQuery)
	if err != nil {
		return 0, err
	}
	var address string
	address, err = peerAddress(k)
	if err != nil {
		return 0, err
	}
	var text string
	text, err = readLocalClipboard()
	if err != nil {
		return 0, err
	}
	if len(text) > clipboardMaxText {
		return 0, fmt.Errorf("local clipboard is too large (max %d bytes)", clipboardMaxText)
	}

	var client *proto.Client
	client, err = proto.DialTLS(address, true, 10*time.Second)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", address, err)
	}
	defer client.Close()

	if err = client.SendTimeout(clipboardUpdateMessage{
		Type:            "clipboard_update",
		Text:            text,
		Mime:            "text/plain",
		TS:              time.Now().UnixMilli(),
		FromNodeID:      localNode.nodeID,
		FromFingerprint: localNode.fingerprint,
	}, 15*time.Second); err != nil {
		return 0, err
	}
	var ack clipboardAckMessage
	if err = client.ReceiveTimeout(&ack, 15*time.Second); err != nil {
		return 0, err
	}
	if ack.Type != "clipboard_ack" || !ack.OK {
		var message = ack.Error
		if message == "" {
			message = "peer rejected the clipboard update"
		}
		return 0, fmt.Errorf("%s", message)
	}
	return ack.Length, nil
}

func pullClipboardFromPeer(peerQuery string) (int, error) {
	var k, err = resolvePairedPeer(peerQuery)
	if err != nil {
		return 0, err
	}
	var address string
	address, err = peerAddress(k)
	if err != nil {
		return 0, err
	}

	var client *proto.Client
	client, err = proto.DialTLS(address, true, 10*time.Second)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", address, err)
	}
	defer client.Close()

	if err = client.SendTimeout(clipboardRequestMessage{
		Type:            "clipboard_request",
		FromNodeID:      localNode.nodeID,
		FromFingerprint: localNode.fingerprint,
	}, 15*time.Second); err != nil {
		return 0, err
	}
	var content clipboardContentMessage
	if err = client.ReceiveTimeout(&content, 15*time.Second); err != nil {
		return 0, err
	}
	if content.Type != "clipboard_content" || !content.OK {
		var message = content.Error
		if message == "" {
			message = "peer rejected the clipboard request"
		}
		return 0, fmt.Errorf("%s", message)
	}
	if err = writeLocalClipboard(content.Text); err != nil {
		return 0, err
	}
	return len(content.Text), nil
}
