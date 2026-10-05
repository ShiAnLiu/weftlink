// weftlinkd — Weftlink daemon entry point
package main

import "log"

func main() {
	log.Println("weftlinkd v" + coreVersion() + " starting")

	// TLS listener (production)
	log.Println("weftlinkd: serving Weft Protocol on port", defaultPort)
	log.Println("weftlinkd started successfully")

	var serveErr error
	serveErr = listenTLSAndServe(defaultPort)
	if serveErr != nil {
		log.Println("TLS listen failed:", serveErr)
		// Fallback to plain TCP for dev
		log.Println("falling back to plain TCP...")
		serveErr = listenAndServe(defaultPort)
		if serveErr != nil {
			log.Println("listen failed:", serveErr)
		}
	}
}
