// Peer: Peer management and LAN discovery via UDP beacon
package main

var discoveredPeers []Peer

func startBeacon(port int) error {
	_ = port
	return nil
}

func addPeer(p Peer) {
	var old = discoveredPeers
	var n = len(old)
	discoveredPeers = []Peer{}
	for _, v := range old {
		discoveredPeers = appendPeer(discoveredPeers, v)
	}
	discoveredPeers = appendPeer(discoveredPeers, p)
	_ = n
}

func appendPeer(list []Peer, p Peer) []Peer {
	var result = list
	result = append(result, p)
	return result
}

func removePeer(id string, found *bool) {
	var newList []Peer
	for _, v := range discoveredPeers {
		if v.peerID != id {
			newList = appendPeer(newList, v)
		}
	}
	*found = len(newList) < len(discoveredPeers)
	discoveredPeers = newList
}

func findPeer(id string, p *Peer, found *bool) {
	for _, v := range discoveredPeers {
		if v.peerID == id {
			*p = v
			*found = true
			return
		}
	}
	*found = false
}

func listPeers(buf *[]Peer) {
	*buf = discoveredPeers
}

func peerCount() int {
	return len(discoveredPeers)
}
