// Core: version, constants, shared types
package main

import "strconv"
import "os"

// productVersion is the Weftlink product version (not the protocol version).
// Release builds override it with:
//
//	-ldflags "-X main.productVersion=<tag without v>"
var productVersion = "0.5.0"

func coreVersion() string {
	return productVersion
}

const (
	defaultPort     = 7801
	protocolVersion = "0.2.0"
	pairCodeLen     = 6
	beaconPort      = 7802
)

// Ports are overridable so two daemons can run on one machine for development
// and pairing tests.
var (
	listenPort    = envPort("WEFTLINK_PORT", defaultPort)
	discoveryPort = envPort("WEFTLINK_BEACON_PORT", beaconPort)
)

func envPort(key string, def int) int {
	var raw = os.Getenv(key)
	if raw == "" {
		return def
	}
	var n, err = strconv.Atoi(raw)
	if err != nil || n <= 0 || n > 65535 {
		return def
	}
	return n
}

type NodeRole int

const (
	RoleWarp NodeRole = 0
	RoleWeft NodeRole = 1
)

type NodeAddr struct {
	transport string
	host      string
	port      int
}

type NodeInfo struct {
	nodeID      string
	displayName string
	role        NodeRole
	version     string
	addresses   []NodeAddr
}
