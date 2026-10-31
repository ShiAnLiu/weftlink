//go:build windows

package main

import "golang.org/x/sys/windows"

// Use UTF-8 on the console so Chinese output from the CLI/daemon matches the
// strings we emit (Go writes UTF-8).
func init() {
	_ = windows.SetConsoleOutputCP(65001)
	_ = windows.SetConsoleCP(65001)
}
