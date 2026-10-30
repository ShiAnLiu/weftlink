package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const daemonBinary = "weftlinkd"

// stateDir mirrors the daemon's data directory (~/.weftlink by default).
func stateDir() string {
	if v := os.Getenv("WEFTLINK_HOME"); v != "" {
		return v
	}
	var home, err = os.UserHomeDir()
	if err != nil || home == "" {
		home = "."
	}
	return filepath.Join(home, ".weftlink")
}

func daemonLogPath() string {
	return filepath.Join(stateDir(), "weftlinkd.log")
}

func findDaemon() (string, error) {
	var name = daemonBinary
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if exe, err := os.Executable(); err == nil {
		var candidate = filepath.Join(filepath.Dir(exe), name)
		if fi, statErr := os.Stat(candidate); statErr == nil && !fi.IsDir() {
			return candidate, nil
		}
	}
	if path, err := exec.LookPath(name); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("%s not found next to weftlink or in PATH (build: go build -o bin/weftlinkd ./src)", name)
}

func spawnDaemon(path string, port int) error {
	if err := os.MkdirAll(stateDir(), 0700); err != nil {
		return err
	}
	var logFile, err = os.OpenFile(daemonLogPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer logFile.Close()

	var cmd = exec.Command(path)
	cmd.Stdin = nil
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Env = envWith("WEFTLINK_PORT", strconv.Itoa(port))
	configureDetach(cmd)

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", path, err)
	}
	// The daemon outlives this process; release our handle to it.
	return cmd.Process.Release()
}

// envWith returns the current environment with key set to val (no duplicates).
func envWith(key string, val string) []string {
	var prefix = key + "="
	var out = make([]string, 0, len(os.Environ())+1)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		out = append(out, entry)
	}
	return append(out, prefix+val)
}
