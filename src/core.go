// Core: version, constants, shared types
package main

func coreVersion() string {
	return "0.1.0"
}

const (
	defaultPort     = 7801
	protocolVersion = "0.1.0"
	pairCodeLen     = 6
	beaconPort      = 7802
)

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

type Peer struct {
	peerID   string
	name     string
	role     NodeRole
	isWarped bool
	isWefted bool
	lastSeen int64
}
