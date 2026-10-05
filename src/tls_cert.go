// TLS cert: self-signed certificate generation for the daemon.
//
// Certs are never committed (see .gitignore *.pem). On first start the
// daemon generates a local self-signed Ed25519 cert under etc/.
package main

import "crypto/ecdsa"
import "crypto/elliptic"
import "crypto/rand"
import "crypto/x509"
import "crypto/x509/pkix"
import "encoding/pem"
import "math/big"
import "os"
import "path/filepath"
import "time"

// ensureTLSCert generates a self-signed cert+key if they don't exist yet.
func ensureTLSCert(certPath string, keyPath string) error {
	var _, statErr = os.Stat(certPath)
	if statErr == nil {
		return nil // already present
	}

	var priv *ecdsa.PrivateKey
	var genErr error
	priv, genErr = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if genErr != nil {
		return genErr
	}

	var serial *big.Int
	serial, _ = rand.Int(rand.Reader, big.NewInt(1<<62))

	var tmpl = x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "weftlinkd"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(3650 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	var der []byte
	var certErr error
	der, certErr = x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if certErr != nil {
		return certErr
	}
	var certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	var keyDER []byte
	var keyErr error
	keyDER, keyErr = x509.MarshalPKCS8PrivateKey(priv)
	if keyErr != nil {
		return keyErr
	}
	var keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	var dir = filepath.Dir(certPath)
	var mkErr error
	mkErr = os.MkdirAll(dir, 0700)
	if mkErr != nil {
		return mkErr
	}

	var wErr error
	wErr = os.WriteFile(certPath, certPEM, 0600)
	if wErr != nil {
		return wErr
	}
	wErr = os.WriteFile(keyPath, keyPEM, 0600)
	return wErr
}
