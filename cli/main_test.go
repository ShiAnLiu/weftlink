package main

import (
	"reflect"
	"testing"
	"time"
)

func TestReorderFlags(t *testing.T) {
	var fs, _ = newFlagSet("knot")
	fs.String("peer", "", "peer")
	fs.Duration("timeout", time.Second, "timeout")

	var args = []string{"CODE", "--peer", "192.168.1.5", "--json", "--timeout", "5s"}
	var got = reorderFlags(fs, args)
	var want = []string{"--peer", "192.168.1.5", "--json", "--timeout", "5s", "CODE"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reorderFlags = %v, want %v", got, want)
	}
}

func TestReorderFlagsBooleanDoesNotEatValue(t *testing.T) {
	var fs, _ = newFlagSet("ping")
	var args = []string{"--json", "extra"}
	var got = reorderFlags(fs, args)
	var want = []string{"--json", "extra"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reorderFlags = %v, want %v", got, want)
	}
}

func TestIsLocalHost(t *testing.T) {
	var cases = map[string]bool{
		"127.0.0.1":   true,
		"localhost":   true,
		"::1":         true,
		"0.0.0.0":     false,
		"192.168.1.2": false,
		"example.com": false,
	}
	for host, want := range cases {
		if got := isLocalHost(host); got != want {
			t.Errorf("isLocalHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestPeerAge(t *testing.T) {
	if got := peerAge(0); got != "-" {
		t.Errorf("peerAge(0) = %q, want -", got)
	}
	var now = time.Now().UnixMilli()
	if got := peerAge(now); got != "0s" {
		t.Errorf("peerAge(now) = %q, want 0s", got)
	}
	if got := peerAge(time.Now().Add(-3 * time.Second).UnixMilli()); got != "3s" {
		t.Errorf("peerAge(-3s) = %q, want 3s", got)
	}
}
