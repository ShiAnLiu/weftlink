// Unit tests for crypto module (Ed25519 keygen/sign/fingerprint)
package main

import "testing"

func TestGenerateNodeKeypair(t *testing.T) {
	var pub []byte
	var priv []byte
	err := generateNodeKeypair(&pub, &priv)
	if err != nil {
		t.Fatalf("generateNodeKeypair failed: %v", err)
	}
	if len(pub) != 32 {
		t.Errorf("public key length = %d, want 32 (ed25519)", len(pub))
	}
	if len(priv) != 32 {
		t.Errorf("private seed length = %d, want 32 (ed25519)", len(priv))
	}
}

func TestSignDataRoundTrip(t *testing.T) {
	var pub []byte
	var priv []byte
	if err := generateNodeKeypair(&pub, &priv); err != nil {
		t.Fatalf("keygen failed: %v", err)
	}
	data := []byte("weftlink test payload")
	var sig []byte
	if err := signData(priv, data, &sig); err != nil {
		t.Fatalf("signData failed: %v", err)
	}
	if len(sig) != 64 {
		t.Errorf("signature length = %d, want 64 (ed25519)", len(sig))
	}
}

func TestFingerprint(t *testing.T) {
	pub := []byte{0x01, 0x02, 0x03, 0x04}
	fp := fingerprint(pub)
	want := "01020304"
	if fp != want {
		t.Errorf("fingerprint = %q, want %q", fp, want)
	}
}
