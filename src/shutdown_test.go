package main

import (
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

func TestIsLoopbackAddr(t *testing.T) {
	var cases = []struct {
		addr net.Addr
		want bool
	}{
		{&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1234}, true},
		{&net.TCPAddr{IP: net.ParseIP("::1"), Port: 1234}, true},
		{&net.TCPAddr{IP: net.ParseIP("192.168.1.5"), Port: 1234}, false},
		{nil, false},
	}
	for _, c := range cases {
		if got := isLoopbackAddr(c.addr); got != c.want {
			t.Errorf("isLoopbackAddr(%v) = %v, want %v", c.addr, got, c.want)
		}
	}
}

func TestShutdownLoopbackOnly(t *testing.T) {
	var oldExit = exitProcess
	var called = make(chan int, 1)
	exitProcess = func(code int) { called <- code }
	shutdownOnce = sync.Once{}
	t.Cleanup(func() {
		exitProcess = oldExit
		shutdownOnce = sync.Once{}
	})

	var payload = []byte(`{"type":"shutdown"}`)
	var remote = &net.TCPAddr{IP: net.ParseIP("192.168.1.5"), Port: 5555}
	var raw, err = handleRequest(remote, payload)
	if err != nil {
		t.Fatal(err)
	}
	var response map[string]any
	if err = json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	if response["type"] != "error" {
		t.Errorf("remote shutdown returned %v, want error", response["type"])
	}
	select {
	case <-called:
		t.Fatal("remote shutdown must not exit the daemon")
	case <-time.After(100 * time.Millisecond):
	}

	var local = &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5555}
	raw, err = handleRequest(local, payload)
	if err != nil {
		t.Fatal(err)
	}
	response = nil
	if err = json.Unmarshal(raw, &response); err != nil {
		t.Fatal(err)
	}
	if response["type"] != "shutting_down" {
		t.Errorf("loopback shutdown returned %v, want shutting_down", response["type"])
	}
	select {
	case code := <-called:
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
	case <-time.After(time.Second):
		t.Fatal("loopback shutdown did not invoke exit")
	}
}
