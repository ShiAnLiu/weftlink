// Shutdown control verb: only the local loopback client may stop the daemon.
package main

import (
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var (
	shutdownOnce sync.Once
	exitProcess  = os.Exit
)

// isLoopbackAddr reports whether a connection comes from 127.0.0.0/8 or ::1.
func isLoopbackAddr(addr net.Addr) bool {
	if addr == nil {
		return false
	}
	var host, _, err = net.SplitHostPort(addr.String())
	if err != nil {
		return false
	}
	var ip = net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// requestShutdown lets the current response be written, then exits. The delay
// is deliberately small; the daemon has no state that needs flushing.
func requestShutdown() {
	shutdownOnce.Do(func() {
		go func() {
			time.Sleep(200 * time.Millisecond)
			log.Println("shutdown requested; exiting")
			exitProcess(0)
		}()
	})
}
