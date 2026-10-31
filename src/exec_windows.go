//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// configureExecProcess hides the child console so approved commands do not
// flash a window on the receiver.
func configureExecProcess(cmd *exec.Cmd) {
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}
