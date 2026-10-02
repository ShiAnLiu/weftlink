// weftlinkd — Weftlink daemon entry point
package main

import "fmt"
import "log"

func main() {
	log.Println("weftlinkd v" + coreVersion() + " starting")

	var port = defaultPort

	// LAN beacon (UDP discovery) — stub until M1
	var discErr error
	discErr = startBeacon(port)
	if discErr != nil {
		log.Println("beacon init failed:", discErr)
	}

	fmt.Println("weftlinkd: serving Weft Protocol on port", port)
	log.Println("weftlinkd started successfully")

	// Blocks forever in the accept loop.
	var serveErr error
	serveErr = listenAndServe(port)
	if serveErr != nil {
		log.Println("listen failed:", serveErr)
	}
}
