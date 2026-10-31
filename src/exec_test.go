package main

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLimitedBuffer(t *testing.T) {
	var buf = &limitedBuffer{limit: 5}
	if n, err := buf.Write([]byte("hello world")); err != nil || n != 11 {
		t.Fatalf("Write = %d, %v", n, err)
	}
	if !buf.truncated {
		t.Error("buffer should report truncation")
	}
	if got := buf.String(); !strings.HasPrefix(got, "hello") || !strings.Contains(got, "truncated") {
		t.Errorf("unexpected buffer: %q", got)
	}
}

func TestExecuteCommandEcho(t *testing.T) {
	var result = executeCommand("test", "echo hello", "", 5*time.Second)
	if !result.OK {
		t.Fatalf("command failed: %s", result.Error)
	}
	if result.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("stdout = %q, want hello", result.Stdout)
	}
}

func TestExecuteCommandExitCode(t *testing.T) {
	var command = "exit 3"
	if runtime.GOOS == "windows" {
		command = "exit /b 3"
	}
	var result = executeCommand("test", command, "", 5*time.Second)
	if !result.OK {
		t.Fatalf("command should run: %s", result.Error)
	}
	if result.ExitCode != 3 {
		t.Errorf("exit code = %d, want 3", result.ExitCode)
	}
}

func TestPendingExecListAndDecision(t *testing.T) {
	execMu.Lock()
	pendingExecs = map[string]*pendingExec{}
	execMu.Unlock()
	t.Cleanup(func() {
		execMu.Lock()
		pendingExecs = map[string]*pendingExec{}
		execMu.Unlock()
	})

	var pending = &pendingExec{
		id:       "exec-1",
		peerID:   "peer-a",
		command:  "echo hi",
		created:  time.Now(),
		decision: make(chan bool, 1),
	}
	execMu.Lock()
	pendingExecs[pending.id] = pending
	execMu.Unlock()

	var list = listPendingExecs()
	if len(list) != 1 || list[0].ExecID != "exec-1" || list[0].Command != "echo hi" {
		t.Fatalf("unexpected list: %+v", list)
	}
	if err := decideExec("exec-1", true); err != nil {
		t.Fatal(err)
	}
	select {
	case approved := <-pending.decision:
		if !approved {
			t.Error("expected approval")
		}
	case <-time.After(time.Second):
		t.Fatal("approval was not delivered")
	}
	if err := decideExec("exec-1", true); err == nil {
		t.Error("second decision should fail")
	}
	if len(listPendingExecs()) != 0 {
		t.Error("pending request should be removed")
	}
}
