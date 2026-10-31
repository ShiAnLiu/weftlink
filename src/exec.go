// Remote command execution with explicit per-request approval.
//
// A paired sender asks a peer to run a command. The receiver daemon queues the
// request and only executes it after a local `weftlink approve <id>`; denials
// and timeouts are reported back to the sender.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"weftlink/internal/proto"
)

const (
	execApprovalTimeout = 2 * time.Minute
	execDefaultTimeout  = 30 * time.Second
	execMaxTimeout      = 5 * time.Minute
	execMaxOutput       = 64 * 1024
	execMaxCommand      = 8 * 1024
)

type execRequestMessage struct {
	Type            string `json:"type"`
	ExecID          string `json:"exec_id"`
	Command         string `json:"command"`
	Cwd             string `json:"cwd,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds,omitempty"`
	Shell           string `json:"shell,omitempty"`
	FromNodeID      string `json:"from_node_id"`
	FromFingerprint string `json:"from_fingerprint"`
}

type execPendingMessage struct {
	Type    string `json:"type"`
	ExecID  string `json:"exec_id"`
	Message string `json:"message,omitempty"`
}

type execResultMessage struct {
	Type     string `json:"type"`
	ExecID   string `json:"exec_id"`
	OK       bool   `json:"ok"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	Error    string `json:"error,omitempty"`
}

type execRequestInfo struct {
	ExecID     string `json:"exec_id"`
	PeerID     string `json:"peer_id"`
	Command    string `json:"command"`
	Shell      string `json:"shell,omitempty"`
	Cwd        string `json:"cwd,omitempty"`
	CreatedMs  int64  `json:"created_ms"`
	AgeSeconds int    `json:"age_seconds"`
}

type pendingExec struct {
	id       string
	peerID   string
	command  string
	shell    string
	cwd      string
	timeout  time.Duration
	created  time.Time
	decision chan bool
}

var (
	execMu       sync.Mutex
	pendingExecs = map[string]*pendingExec{}
)

func newExecID() (string, error) {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func clampExecTimeout(seconds int) time.Duration {
	var timeout = time.Duration(seconds) * time.Second
	if timeout <= 0 {
		timeout = execDefaultTimeout
	}
	if timeout > execMaxTimeout {
		timeout = execMaxTimeout
	}
	return timeout
}

func handleExecRequest(conn net.Conn, payload []byte) error {
	var req execRequestMessage
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}
	if req.ExecID == "" || strings.TrimSpace(req.Command) == "" {
		return writeWire(conn, execResultMessage{
			Type: "exec_result", OK: false, Error: "invalid exec request",
		}, 30*time.Second)
	}
	if len(req.Command) > execMaxCommand {
		return writeWire(conn, execResultMessage{
			Type: "exec_result", ExecID: req.ExecID, OK: false, Error: "command too long",
		}, 30*time.Second)
	}
	var shell, shellErr = normalizeShellName(req.Shell)
	if shellErr != nil {
		return writeWire(conn, execResultMessage{
			Type: "exec_result", ExecID: req.ExecID, OK: false, Error: shellErr.Error(),
		}, 30*time.Second)
	}
	if k, ok := getKnot(req.FromNodeID); !ok ||
		!strings.EqualFold(k.Fingerprint, req.FromFingerprint) {
		return writeWire(conn, execResultMessage{
			Type: "exec_result", ExecID: req.ExecID, OK: false, Error: "unknown or unpaired sender",
		}, 30*time.Second)
	}

	var pending = &pendingExec{
		id:       req.ExecID,
		peerID:   req.FromNodeID,
		command:  req.Command,
		shell:    shell,
		cwd:      req.Cwd,
		timeout:  clampExecTimeout(req.TimeoutSeconds),
		created:  time.Now(),
		decision: make(chan bool, 1),
	}
	execMu.Lock()
	if _, exists := pendingExecs[req.ExecID]; exists {
		execMu.Unlock()
		return writeWire(conn, execResultMessage{
			Type: "exec_result", ExecID: req.ExecID, OK: false, Error: "duplicate request id",
		}, 30*time.Second)
	}
	pendingExecs[req.ExecID] = pending
	execMu.Unlock()

	var remove = func() {
		execMu.Lock()
		if current, ok := pendingExecs[req.ExecID]; ok && current == pending {
			delete(pendingExecs, req.ExecID)
		}
		execMu.Unlock()
	}
	defer remove()

	if err := writeWire(conn, execPendingMessage{
		Type:    "exec_pending",
		ExecID:  req.ExecID,
		Message: "waiting for local approval",
	}, 30*time.Second); err != nil {
		return err
	}
	log.Println("exec request", req.ExecID, "from", req.FromNodeID, ":", req.Command)

	var timer = time.NewTimer(execApprovalTimeout)
	defer timer.Stop()

	select {
	case approved := <-pending.decision:
		if !approved {
			return writeWire(conn, execResultMessage{
				Type: "exec_result", ExecID: req.ExecID, OK: false, Error: "denied by peer",
			}, 30*time.Second)
		}
	case <-timer.C:
		return writeWire(conn, execResultMessage{
			Type: "exec_result", ExecID: req.ExecID, OK: false, Error: "approval timed out",
		}, 30*time.Second)
	}

	var result = executeCommand(req.ExecID, req.Command, req.Cwd, pending.timeout, shell)
	return writeWire(conn, result, 30*time.Second)
}

func defaultShellName() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "sh"
}

func normalizeShellName(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "", "auto":
		return defaultShellName(), nil
	case "pwsh", "powershell":
		return "powershell", nil
	case "cmd", "cmd.exe":
		return "cmd", nil
	case "sh", "bash":
		return "sh", nil
	default:
		return "", fmt.Errorf("unsupported shell %q (use auto|cmd|powershell|sh)", shell)
	}
}

func shellInvocation(shell string, command string) (string, []string, error) {
	switch shell {
	case "powershell":
		if runtime.GOOS != "windows" {
			return "", nil, fmt.Errorf("powershell is only available on Windows")
		}
		var exe, err = exec.LookPath("powershell.exe")
		if err != nil {
			return "", nil, fmt.Errorf("powershell not found: %w", err)
		}
		return exe, []string{"-NoProfile", "-NonInteractive", "-Command",
			"[Console]::OutputEncoding=[System.Text.Encoding]::UTF8; " +
				"$OutputEncoding=[System.Text.Encoding]::UTF8; " + command}, nil
	case "cmd":
		if runtime.GOOS != "windows" {
			return "", nil, fmt.Errorf("cmd is only available on Windows")
		}
		var exe = os.Getenv("ComSpec")
		if exe == "" {
			exe = "cmd.exe"
		}
		return exe, []string{"/C", "chcp 65001 >nul & " + command}, nil
	case "sh":
		if runtime.GOOS == "windows" {
			return "", nil, fmt.Errorf("sh is not available on Windows")
		}
		return "/bin/sh", []string{"-c", command}, nil
	default:
		return "", nil, fmt.Errorf("unsupported shell %q", shell)
	}
}

func executeCommand(execID string, command string, cwd string, timeout time.Duration, shell string) execResultMessage {
	var result = execResultMessage{Type: "exec_result", ExecID: execID}
	var normalized, shellErr = normalizeShellName(shell)
	if shellErr != nil {
		result.Error = shellErr.Error()
		return result
	}
	var name, args, invokeErr = shellInvocation(normalized, command)
	if invokeErr != nil {
		result.Error = invokeErr.Error()
		return result
	}
	var ctx, cancel = context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd = exec.CommandContext(ctx, name, args...)
	configureExecProcess(cmd)
	if cwd != "" {
		var info, err = os.Stat(cwd)
		if err != nil || !info.IsDir() {
			result.Error = "working directory not found"
			return result
		}
		cmd.Dir = cwd
	}

	var stdout = &limitedBuffer{limit: execMaxOutput}
	var stderr = &limitedBuffer{limit: execMaxOutput}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	var err = cmd.Run()
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		result.ExitCode = -1
	}
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		result.OK = false
		result.Error = "command timed out"
		return result
	}
	var exitErr *exec.ExitError
	if err == nil || errors.As(err, &exitErr) {
		result.OK = true
		return result
	}
	result.OK = false
	result.Error = err.Error()
	return result
}

type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len() >= b.limit {
		b.truncated = true
		return len(p), nil
	}
	var remaining = b.limit - b.buf.Len()
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) String() string {
	var out = b.buf.String()
	if b.truncated {
		out += "\n[output truncated]"
	}
	return out
}

func listPendingExecs() []execRequestInfo {
	execMu.Lock()
	var out = make([]execRequestInfo, 0, len(pendingExecs))
	var now = time.Now()
	for _, pending := range pendingExecs {
		out = append(out, execRequestInfo{
			ExecID:     pending.id,
			PeerID:     pending.peerID,
			Command:    pending.command,
			Shell:      pending.shell,
			Cwd:        pending.cwd,
			CreatedMs:  pending.created.UnixMilli(),
			AgeSeconds: int(now.Sub(pending.created).Seconds()),
		})
	}
	execMu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedMs < out[j].CreatedMs })
	return out
}

func decideExec(execID string, approve bool) error {
	execMu.Lock()
	var pending, ok = pendingExecs[execID]
	if ok {
		delete(pendingExecs, execID)
	}
	execMu.Unlock()
	if !ok {
		return fmt.Errorf("unknown or expired exec request %q", execID)
	}
	select {
	case pending.decision <- approve:
	default:
	}
	return nil
}

func runRemoteCommand(peerQuery string, command string, timeoutSeconds int, shell string) (execResultMessage, error) {
	var k, err = resolvePairedPeer(peerQuery)
	if err != nil {
		return execResultMessage{}, err
	}
	var address string
	address, err = peerAddress(k)
	if err != nil {
		return execResultMessage{}, err
	}
	var execID string
	execID, err = newExecID()
	if err != nil {
		return execResultMessage{}, err
	}
	var timeout = clampExecTimeout(timeoutSeconds)

	var client *proto.Client
	client, err = proto.DialTLS(address, true, 10*time.Second)
	if err != nil {
		return execResultMessage{}, fmt.Errorf("%s: %w", address, err)
	}
	defer client.Close()

	if err = client.SendTimeout(execRequestMessage{
		Type:            "exec_request",
		ExecID:          execID,
		Command:         command,
		TimeoutSeconds:  int(timeout.Seconds()),
		Shell:           shell,
		FromNodeID:      localNode.nodeID,
		FromFingerprint: localNode.fingerprint,
	}, 15*time.Second); err != nil {
		return execResultMessage{}, err
	}

	var first execResultMessage
	if err = client.ReceiveTimeout(&first, 15*time.Second); err != nil {
		return execResultMessage{}, err
	}
	if first.Type == "exec_result" {
		return first, nil
	}
	if first.Type != "exec_pending" {
		return execResultMessage{}, fmt.Errorf("unexpected peer response: %s", first.Type)
	}

	var wait = execApprovalTimeout + timeout + 30*time.Second
	var result execResultMessage
	if err = client.ReceiveTimeout(&result, wait); err != nil {
		return execResultMessage{}, err
	}
	if result.Type != "exec_result" {
		return execResultMessage{}, fmt.Errorf("unexpected peer response: %s", result.Type)
	}
	return result, nil
}
