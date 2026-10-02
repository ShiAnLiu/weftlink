// weftlinkd — Weftlink daemon entry point
package main

import "fmt"
import "log"

func main() {
	log.Println("weftlinkd v" + coreVersion() + " starting")

	var port = defaultPort
	var lnErr error
	lnErr = listenTLS(port)
	if lnErr != nil {
		log.Println("listen failed:", lnErr)
		return
	}

	var discErr error
	discErr = startBeacon(port)
	if discErr != nil {
		log.Println("beacon init failed:", discErr)
		return
	}

	fmt.Println("weftlinkd: ready on port", port)
	log.Println("weftlinkd started successfully")
}
