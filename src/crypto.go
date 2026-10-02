// Crypto: Ed25519 signing, key generation
package main

import "crypto/ed25519"
import "crypto/rand"
import "encoding/hex"

func generateNodeKeypair(pubKey *[]byte, privKey *[]byte) error {
	var pub ed25519.PublicKey
	var priv ed25519.PrivateKey
	var genErr error
	pub, priv, genErr = ed25519.GenerateKey(rand.Reader)
	if genErr != nil {
		return genErr
	}
	*pubKey = pub
	*privKey = priv.Seed()
	return nil
}

func signData(privKey []byte, data []byte, sig *[]byte) error {
	var priv = ed25519.NewKeyFromSeed(privKey)
	var signature = ed25519.Sign(priv, data)
	*sig = signature
	return nil
}

func fingerprint(pubKey []byte) string {
	return hex.EncodeToString(pubKey)
}
