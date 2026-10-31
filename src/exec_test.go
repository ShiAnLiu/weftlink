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
	var result = executeCommand("test", "echo hello", "", 5*time.Second, "auto")
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
	var result = executeCommand("test", "exit 3", "", 5*time.Second, "auto")
	if !result.OK {
		t.Fatalf("command should run: %s", result.Error)
	}
	if result.ExitCode != 3 {
		t.Errorf("exit code = %d, want 3", result.ExitCode)
	}
}

func TestExecuteCommandCmdShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("cmd.exe is Windows-only")
	}
	var result = executeCommand("test", "echo hello-cmd", "", 5*time.Second, "cmd")
	if !result.OK {
		t.Fatalf("cmd shell failed: %s", result.Error)
	}
	if !strings.Contains(result.Stdout, "hello-cmd") {
		t.Errorf("stdout = %q", result.Stdout)
	}
}

func TestExecuteCommandUTF8(t *testing.T) {
	var result = executeCommand("test", "echo ??", "", 5*time.Second, "auto")
	if !result.OK {
		t.Fatalf("command failed: %s", result.Error)
	}
	if !strings.Contains(result.Stdout, "??") {
		t.Errorf("stdout = %q, want UTF-8 Chinese text", result.Stdout)
	}
}

func TestExecuteCommandInvalidShell(t *testing.T) {
	var result = executeCommand("test", "echo hi", "", time.Second, "no-such-shell")
	if result.OK {
		t.Error("unsupported shell should fail")
	}
	if !strings.Contains(result.Error, "unsupported shell") {
		t.Errorf("unexpected error: %q", result.Error)
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
