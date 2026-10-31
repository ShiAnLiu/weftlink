//go:build !windows

package main

import "fmt"

func configureFirewall() error {
	return fmt.Errorf("firewall helper is Windows-only; allow inbound TCP 7801 and UDP 7802 manually")
}
