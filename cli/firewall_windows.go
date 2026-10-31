//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func configureFirewall() error {
	var exe, err = os.Executable()
	if err != nil {
		return err
	}
	var daemon = filepath.Join(filepath.Dir(exe), "weftlinkd.exe")
	if _, statErr := os.Stat(daemon); statErr != nil {
		return fmt.Errorf("weftlinkd.exe not found next to weftlink.exe")
	}

	for _, name := range []string{"Weftlink TCP", "Weftlink LAN"} {
		_ = runNetsh("advfirewall", "firewall", "delete", "rule", "name="+name)
	}
	if err = runNetsh("advfirewall", "firewall", "add", "rule",
		"name=Weftlink TCP", "dir=in", "action=allow",
		"program="+daemon, "protocol=TCP", "localport=7801",
		"profile=any", "enable=yes"); err != nil {
		return fmt.Errorf("adding TCP rule failed (run as Administrator): %w", err)
	}
	if err = runNetsh("advfirewall", "firewall", "add", "rule",
		"name=Weftlink LAN", "dir=in", "action=allow",
		"program="+daemon, "protocol=UDP", "localport=7802",
		"profile=any", "enable=yes"); err != nil {
		return fmt.Errorf("adding UDP rule failed (run as Administrator): %w", err)
	}
	return nil
}

// tryConfigureFirewall runs automatically during `weftlink up` when elevated.
func tryConfigureFirewall() {
	if !isAdmin() {
		return
	}
	_ = configureFirewall()
}

func isAdmin() bool {
	var cmd = exec.Command("net", "session")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	return cmd.Run() == nil
}

func runNetsh(args ...string) error {
	var cmd = exec.Command("netsh", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	return cmd.Run()
}
