// Peer: LAN discovery (UDP multicast beacon) and the in-memory peer table.
package main

import (
	"encoding/json"
	"log"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Discovery constants. The multicast group is in the administratively scoped
// range and only carries node metadata.
const (
	multicastGroup = "224.0.0.167"
	beaconInterval = 3 * time.Second
	peerTTL        = 12 * time.Second
	pruneInterval  = 5 * time.Second
)

// Peer is the wire/display representation of a discovered node.
type Peer struct {
	PeerID      string `json:"peer_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Version     string `json:"version"`
	Address     string `json:"address"`
	Fingerprint string `json:"fingerprint"`
	LastSeen    int64  `json:"last_seen"`
	Paired      bool   `json:"paired"`
}

// beaconMessage is the UDP discovery payload.
type beaconMessage struct {
	Type        string `json:"type"`
	NodeID      string `json:"node_id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
	Port        int    `json:"port"`
	TS          int64  `json:"ts"`
}

var (
	peersMu sync.RWMutex
	peers   = map[string]Peer{}
)

// startBeacon joins the multicast group and starts sending/receiving beacons.
func startBeacon(port int) error {
	var group = &net.UDPAddr{IP: net.ParseIP(multicastGroup), Port: port}
	var conn *net.UDPConn
	var err error
	conn, err = net.ListenMulticastUDP("udp4", nil, group)
	if err != nil {
		return err
	}
	_ = conn.SetReadBuffer(1 << 16)
	go readBeacons(conn)
	go sendBeacons(port)
	go pruneLoop()
	return nil
}

func sendBeacons(port int) {
	// Multicast is the intended transport; limited broadcast is a pragmatic
	// fallback because some Windows setups drop looped-back multicast.
	var destinations = []*net.UDPAddr{
		{IP: net.ParseIP(multicastGroup), Port: port},
		{IP: net.IPv4bcast, Port: port},
	}
	var conns []*net.UDPConn
	for _, dst := range destinations {
		var conn, err = net.DialUDP("udp4", nil, dst)
		if err != nil {
			log.Println("beacon send disabled for", dst.IP, ":", err)
			continue
		}
		defer conn.Close()
		conns = append(conns, conn)
	}
	if len(conns) == 0 {
		return
	}

	var ticker = time.NewTicker(beaconInterval)
	defer ticker.Stop()
	for {
		var payload, err = json.Marshal(currentBeacon())
		if err == nil {
			for _, conn := range conns {
				if _, err = conn.Write(payload); err != nil {
					log.Println("beacon send error:", err)
				}
			}
		}
		<-ticker.C
	}
}

func currentBeacon() beaconMessage {
	return beaconMessage{
		Type:        "beacon",
		NodeID:      localNode.nodeID,
		DisplayName: localNode.displayName,
		Role:        "warp",
		Version:     localNode.version,
		Fingerprint: localNode.fingerprint,
		Port:        listenPort,
		TS:          time.Now().UnixMilli(),
	}
}

func readBeacons(conn *net.UDPConn) {
	var buf = make([]byte, 4096)
	for {
		var n int
		var src *net.UDPAddr
		var err error
		n, src, err = conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("beacon read error:", err)
			continue
		}
		var msg beaconMessage
		if json.Unmarshal(buf[:n], &msg) != nil {
			continue
		}
		if msg.Type != "beacon" || msg.NodeID == "" || msg.NodeID == localNode.nodeID {
			continue
		}
		if msg.Port <= 0 || msg.Port > 65535 {
			continue
		}
		var address = net.JoinHostPort(src.IP.String(), strconv.Itoa(msg.Port))
		var paired = false
		if k, ok := getKnot(msg.NodeID); ok {
			// A beacon cannot change the identity/address of a paired node.
			if k.Fingerprint != msg.Fingerprint {
				continue
			}
			paired = true
		}
		var p = Peer{
			PeerID:      msg.NodeID,
			DisplayName: msg.DisplayName,
			Role:        msg.Role,
			Version:     msg.Version,
			Address:     address,
			Fingerprint: msg.Fingerprint,
			LastSeen:    time.Now().UnixMilli(),
			Paired:      paired,
		}
		if upsertPeer(p) {
			log.Println("peer discovered:", p.DisplayName, "("+p.PeerID+")", p.Address)
		}
	}
}

func pruneLoop() {
	var ticker = time.NewTicker(pruneInterval)
	defer ticker.Stop()
	for range ticker.C {
		prunePeers(peerTTL)
	}
}

// upsertPeer inserts or refreshes a peer. It reports whether the peer is new.
func upsertPeer(p Peer) bool {
	peersMu.Lock()
	defer peersMu.Unlock()
	var old, existed = peers[p.PeerID]
	if existed {
		if old.Paired {
			p.Paired = true
		}
		if p.DisplayName == "" {
			p.DisplayName = old.DisplayName
		}
		if p.Address == "" {
			p.Address = old.Address
		}
		if p.Fingerprint == "" {
			p.Fingerprint = old.Fingerprint
		}
	}
	peers[p.PeerID] = p
	return !existed
}

// prunePeers drops unpaired peers not seen within ttl. Paired knots are kept.
func prunePeers(ttl time.Duration) {
	var cutoff = time.Now().Add(-ttl).UnixMilli()
	peersMu.Lock()
	defer peersMu.Unlock()
	for id, p := range peers {
		if !p.Paired && p.LastSeen < cutoff {
			delete(peers, id)
		}
	}
}

func markPeerPaired(peerID string) {
	peersMu.Lock()
	defer peersMu.Unlock()
	var p, ok = peers[peerID]
	if !ok {
		return
	}
	p.Paired = true
	peers[peerID] = p
}

// findPeer returns a copy of the peer with the given id.
func findPeer(id string) (Peer, bool) {
	peersMu.RLock()
	defer peersMu.RUnlock()
	var p, ok = peers[id]
	return p, ok
}

// listPeers returns a stable, paired-first snapshot.
func listPeers() []Peer {
	peersMu.RLock()
	var out = make([]Peer, 0, len(peers))
	for _, p := range peers {
		out = append(out, p)
	}
	peersMu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Paired != out[j].Paired {
			return out[i].Paired
		}
		if out[i].DisplayName != out[j].DisplayName {
			return out[i].DisplayName < out[j].DisplayName
		}
		return out[i].PeerID < out[j].PeerID
	})
	return out
}

// peerAddresses returns the dialable addresses of all known peers.
func peerAddresses() []string {
	var out []string
	for _, p := range listPeers() {
		if p.Address != "" {
			out = append(out, p.Address)
		}
	}
	return out
}

func peerCount() int {
	peersMu.RLock()
	defer peersMu.RUnlock()
	return len(peers)
}

func pairedPeerCount() int {
	peersMu.RLock()
	defer peersMu.RUnlock()
	var n = 0
	for _, p := range peers {
		if p.Paired {
			n++
		}
	}
	return n
}
