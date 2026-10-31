//go:build !windows

package main

import "os/exec"

func configureExecProcess(cmd *exec.Cmd) {
	_ = cmd
}
