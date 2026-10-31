package main

import (
	"testing"
	"time"
)

func TestClipboardPolicy(t *testing.T) {
	cancelClipboard()
	t.Cleanup(cancelClipboard)

	if err := clipboardAllows("peer-a"); err == nil {
		t.Fatal("clipboard should be off by default")
	}
	if _, err := beginClipboard("peer-a", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := clipboardAllows("peer-a"); err != nil {
		t.Errorf("peer-a should be allowed: %v", err)
	}
	if err := clipboardAllows("peer-b"); err == nil {
		t.Error("peer-b should be rejected by the from filter")
	}
	active, from, left := clipboardStatus()
	if !active || from != "peer-a" || left <= 0 {
		t.Errorf("unexpected policy: active=%v from=%q left=%d", active, from, left)
	}
	cancelClipboard()
	if err := clipboardAllows("peer-a"); err == nil {
		t.Error("clipboard should be off after cancel")
	}
}

func TestClipboardPolicyExpires(t *testing.T) {
	cancelClipboard()
	t.Cleanup(cancelClipboard)

	if _, err := beginClipboard("", 30*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if err := clipboardAllows("peer-a"); err == nil {
		t.Error("expired policy should reject updates")
	}
	if active, _, _ := clipboardStatus(); active {
		t.Error("expired policy should report inactive")
	}
}
