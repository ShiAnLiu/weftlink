// Unit tests for TLS cert auto-generation
package main

import "crypto/tls"
import "os"
import "path/filepath"
import "testing"

func TestEnsureTLSCertCreates(t *testing.T) {
	var dir = filepath.Join(os.TempDir(), "weftlink-cert-test")
	os.RemoveAll(dir)
	var certPath = filepath.Join(dir, "cert.pem")
	var keyPath = filepath.Join(dir, "key.pem")
	defer os.RemoveAll(dir)

	var err error
	err = ensureTLSCert(certPath, keyPath)
	if err != nil {
		t.Fatalf("ensureTLSCert failed: %v", err)
	}

	// Files must exist and load as a valid keypair.
	var cert tls.Certificate
	var loadErr error
	cert, loadErr = tls.LoadX509KeyPair(certPath, keyPath)
	if loadErr != nil {
		t.Fatalf("generated cert does not load: %v", loadErr)
	}
	if len(cert.Certificate) == 0 {
		t.Error("certificate chain empty")
	}
}

func TestEnsureTLSCertIdempotent(t *testing.T) {
	var dir = filepath.Join(os.TempDir(), "weftlink-cert-test2")
	os.RemoveAll(dir)
	var certPath = filepath.Join(dir, "cert.pem")
	var keyPath = filepath.Join(dir, "key.pem")
	defer os.RemoveAll(dir)

	var err error
	err = ensureTLSCert(certPath, keyPath)
	if err != nil {
		t.Fatalf("first ensureTLSCert failed: %v", err)
	}
	var first []byte
	first, _ = os.ReadFile(certPath)

	// Second call must NOT regenerate (file unchanged).
	err = ensureTLSCert(certPath, keyPath)
	if err != nil {
		t.Fatalf("second ensureTLSCert failed: %v", err)
	}
	var second []byte
	second, _ = os.ReadFile(certPath)
	if string(first) != string(second) {
		t.Error("cert was regenerated on second call (should be idempotent)")
	}
}
