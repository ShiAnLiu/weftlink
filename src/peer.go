// Peer: LAN discovery (UDP multicast beacon) and the in-memory peer table.
package main

import (
	"encoding/json"
	"log"
	"net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
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

	beaconActive    atomic.Bool
	beaconsSent     atomic.Int64
	beaconsReceived atomic.Int64
	lastBeaconMs    atomic.Int64
)

// discoveryState is exposed through the status verb for diagnostics.
type discoveryState struct {
	Active         bool  `json:"active"`
	Port           int   `json:"port"`
	Sent           int64 `json:"sent"`
	Received       int64 `json:"received"`
	LastReceivedMs int64 `json:"last_received_ms"`
}

func currentDiscoveryState() discoveryState {
	return discoveryState{
		Active:         beaconActive.Load(),
		Port:           discoveryPort,
		Sent:           beaconsSent.Load(),
		Received:       beaconsReceived.Load(),
		LastReceivedMs: lastBeaconMs.Load(),
	}
}

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
	beaconActive.Store(true)
	go readBeacons(conn)
	go sendBeacons(conn, port)
	go pruneLoop()
	return nil
}

// sendBeacons writes from the same socket that receives beacons. Sending from
// port 7802 keeps Windows Firewall's stateful mapping consistent and avoids a
// separate ephemeral socket per interface.
func sendBeacons(conn *net.UDPConn, port int) {
	var ticker = time.NewTicker(beaconInterval)
	defer ticker.Stop()
	for {
		var payload, err = json.Marshal(currentBeacon())
		if err == nil {
			for _, dst := range beaconDestinations(port) {
				if _, werr := conn.WriteToUDP(payload, dst); werr == nil {
					beaconsSent.Add(1)
				}
			}
			// Unicast beacons keep discovered peers warm even when broadcast
			// or multicast is filtered by the AP/driver.
			sendUnicastBeacons(conn, payload)
		}
		<-ticker.C
	}
}

// beaconDestinations returns the multicast group, limited broadcast, and the
// directed broadcast address of every usable IPv4 interface.
func beaconDestinations(port int) []*net.UDPAddr {
	var out []*net.UDPAddr
	var seen = map[string]bool{}
	var add = func(ip net.IP) {
		if ip == nil {
			return
		}
		var key = ip.String()
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, &net.UDPAddr{IP: ip, Port: port})
	}
	add(net.ParseIP(multicastGroup))
	add(net.IPv4bcast)
	for _, ifc := range ipv4Interfaces() {
		add(directedBroadcast(ifc.ip, ifc.mask))
	}
	return out
}

type iface4 struct {
	ip   net.IP
	mask net.IPMask
}

func ipv4Interfaces() []iface4 {
	var out []iface4
	var ifaces, err = net.Interfaces()
	if err != nil {
		return out
	}
	for _, ifi := range ifaces {
		if ifi.Flags&net.FlagUp == 0 || ifi.Flags&net.FlagLoopback != 0 {
			continue
		}
		var addrs, aerr = ifi.Addrs()
		if aerr != nil {
			continue
		}
		for _, a := range addrs {
			var ipnet, ok = a.(*net.IPNet)
			if !ok {
				continue
			}
			var ip4 = ipnet.IP.To4()
			if ip4 == nil || ip4.IsLinkLocalUnicast() || ip4.IsLoopback() {
				continue
			}
			out = append(out, iface4{ip: ip4, mask: ipnet.Mask})
		}
	}
	return out
}

func directedBroadcast(ip net.IP, mask net.IPMask) net.IP {
	var ip4 = ip.To4()
	if ip4 == nil {
		return nil
	}
	var m = mask
	if len(m) == 16 {
		m = m[12:]
	}
	if len(m) != 4 {
		return nil
	}
	var b = make(net.IP, 4)
	for i := 0; i < 4; i++ {
		b[i] = ip4[i] | ^m[i]
	}
	return b
}

func sendUnicastBeacons(conn *net.UDPConn, payload []byte) {
	for _, p := range listPeers() {
		if p.Address == "" {
			continue
		}
		var host, _, err = net.SplitHostPort(p.Address)
		if err != nil {
			continue
		}
		var ip = net.ParseIP(host)
		if ip == nil {
			continue
		}
		var dst = &net.UDPAddr{IP: ip, Port: discoveryPort}
		if _, werr := conn.WriteToUDP(payload, dst); werr == nil {
			beaconsSent.Add(1)
		}
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
		beaconsReceived.Add(1)
		lastBeaconMs.Store(time.Now().UnixMilli())
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
