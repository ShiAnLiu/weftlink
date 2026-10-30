package main

import (
	"os"
	"testing"
)

// TestMain gives the package a throwaway identity/state directory so tests
// never touch the developer's real ~/.weftlink.
func TestMain(m *testing.M) {
	var dir, err = os.MkdirTemp("", "weftlink-test-*")
	if err != nil {
		panic(err)
	}
	os.Setenv("WEFTLINK_HOME", dir)
	os.Setenv("WEFTLINK_NAME", "test-daemon")
	if err = initNodeIdentity(); err != nil {
		panic(err)
	}
	var code = m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
