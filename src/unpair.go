// Unpair: remove a knot locally and best-effort notify the peer.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"weftlink/internal/proto"
)

type unpairRequest struct {
	Type            string `json:"type"`
	FromNodeID      string `json:"from_node_id"`
	FromFingerprint string `json:"from_fingerprint"`
}

type unpairAck struct {
	Type  string `json:"type"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// removeKnot deletes the persisted knot and marks the peer unpaired.
func removeKnot(peerID string) error {
	knotMu.Lock()
	delete(knots, peerID)
	knotMu.Unlock()

	var path = filepath.Join(knotsDir(), peerID+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	peersMu.Lock()
	if peer, ok := peers[peerID]; ok {
		peer.Paired = false
		peers[peerID] = peer
	}
	peersMu.Unlock()
	return nil
}

func handleUnpairRequest(fromID string, fingerprint string) error {
	var k, ok = getKnot(fromID)
	if !ok || !strings.EqualFold(k.Fingerprint, fingerprint) {
		return fmt.Errorf("unknown or unpaired sender")
	}
	return removeKnot(fromID)
}

// untiePeer removes the knot locally, then asks the peer to drop it too. The
// local removal succeeds even if the peer is offline.
func untiePeer(query string) error {
	var k, err = resolvePairedPeer(query)
	if err != nil {
		return err
	}
	if err = removeKnot(k.PeerID); err != nil {
		return err
	}
	if address, addrErr := peerAddress(k); addrErr == nil {
		if client, dialErr := proto.DialTLS(address, true, 5*time.Second); dialErr == nil {
			defer client.Close()
			var ack unpairAck
			_ = client.RoundTripTimeout(unpairRequest{
				Type:            "unpair_request",
				FromNodeID:      localNode.nodeID,
				FromFingerprint: localNode.fingerprint,
			}, &ack, 5*time.Second)
		}
	}
	return nil
}
