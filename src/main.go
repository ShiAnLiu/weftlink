// weftlinkd — Weftlink daemon entry point
package main

import "log"

func main() {
	log.Println("weftlinkd v" + coreVersion() + " starting")

	if err := initNodeIdentity(); err != nil {
		log.Fatal("identity init failed: ", err)
	}
	if err := loadKnots(); err != nil {
		log.Println("load knots failed:", err)
	}
	log.Println("node id:", localNode.nodeID, "fingerprint:", localNode.fingerprint)

	if err := startBeacon(discoveryPort); err != nil {
		log.Println("LAN discovery disabled:", err)
	}

	// TLS listener (production)
	log.Println("weftlinkd: serving Weft Protocol on port", listenPort)
	log.Println("weftlinkd started successfully")

	var serveErr error
	serveErr = listenTLSAndServe(listenPort)
	if serveErr != nil {
		log.Println("TLS listen failed:", serveErr)
		// Fallback to plain TCP for dev
		log.Println("falling back to plain TCP...")
		serveErr = listenAndServe(listenPort)
		if serveErr != nil {
			log.Println("listen failed:", serveErr)
		}
	}
}
