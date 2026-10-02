// Knot: Pairing logic — 6-character code exchange
package main

import "crypto/rand"

type KnotState struct {
	code       string
	localKey   []byte
	remoteKey  []byte
	remoteNode NodeInfo
	confirmed  bool
}

func generatePairCode(code *string) error {
	var buf []byte = []byte{0, 0, 0, 0}
	rand.Reader.Read(buf)
	// TODO: proper base32 encoding, return 6 chars
	// For now return a placeholder
	*code = "ABCDEF"
	return nil
}

func initiateKnot(pairCode string, state *KnotState) error {
	*state = KnotState{pairCode, nil, nil, NodeInfo{}, false}
	return nil
}

func acceptKnot(pairCode string, remote NodeInfo, remoteKey []byte,
	state *KnotState) error {
	*state = KnotState{pairCode, nil, remoteKey, remote, true}
	return nil
}
