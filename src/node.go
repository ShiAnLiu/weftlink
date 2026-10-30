// Node: stable local identity (node id + Ed25519 key) persisted under
// ~/.weftlink (or $WEFTLINK_HOME for tests).
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dataDirEnv   = "WEFTLINK_HOME"
	nodeNameEnv  = "WEFTLINK_NAME"
	nodeKeyFile  = "node.key"
	knotsDirName = "knots"
)

// nodeIdentity is the daemon's long-lived application-layer identity.
type nodeIdentity struct {
	nodeID      string
	publicKey   []byte
	fingerprint string
	displayName string
	role        NodeRole
	version     string
}

var localNode nodeIdentity

// dataDir returns the Weftlink state directory.
func dataDir() string {
	if v := os.Getenv(dataDirEnv); v != "" {
		return v
	}
	var home, err = os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, ".weftlink")
}

func nodeKeyPath() string {
	return filepath.Join(dataDir(), nodeKeyFile)
}

func knotsDir() string {
	return filepath.Join(dataDir(), knotsDirName)
}

func tlsDir() string {
	return filepath.Join(dataDir(), "tls")
}

// initNodeIdentity loads (or creates) the node key and derives the node id.
func initNodeIdentity() error {
	var seed []byte
	var err error
	seed, err = loadOrCreateNodeKey()
	if err != nil {
		return err
	}
	var priv = ed25519.NewKeyFromSeed(seed)
	var pub = priv.Public().(ed25519.PublicKey)
	var sum = sha256.Sum256(pub)
	localNode = nodeIdentity{
		nodeID:      hex.EncodeToString(sum[:])[:12],
		publicKey:   append([]byte(nil), pub...),
		fingerprint: fingerprint(pub),
		displayName: localDisplayName(),
		role:        RoleWarp,
		version:     coreVersion(),
	}
	return nil
}

// loadOrCreateNodeKey returns the 32-byte Ed25519 seed, creating it on first run.
func loadOrCreateNodeKey() ([]byte, error) {
	var raw, err = os.ReadFile(nodeKeyPath())
	if err == nil {
		var seed []byte
		seed, err = hex.DecodeString(strings.TrimSpace(string(raw)))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", nodeKeyPath(), err)
		}
		if len(seed) != ed25519.SeedSize {
			return nil, fmt.Errorf("%s: invalid seed length %d", nodeKeyPath(), len(seed))
		}
		return seed, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	var seed = make([]byte, ed25519.SeedSize)
	if _, err = rand.Read(seed); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(dataDir(), 0700); err != nil {
		return nil, err
	}
	if err = os.WriteFile(nodeKeyPath(), []byte(hex.EncodeToString(seed)+"\n"), 0600); err != nil {
		return nil, err
	}
	return seed, nil
}

func localDisplayName() string {
	var name = strings.TrimSpace(os.Getenv(nodeNameEnv))
	if name == "" {
		name, _ = os.Hostname()
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "weftlink-node"
	}
	if len(name) > 64 {
		name = name[:64]
	}
	return name
}
