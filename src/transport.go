// Transport: TCP/TLS transport layer (stub for M0)
package main

func listenTLS(port int) error {
	_ = port
	return nil
}

func connectTLS(host string, port int, conn *int, err *error) {
	_ = host
	_ = port
	_ = conn
	*err = nil
}
