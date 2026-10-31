//go:build !windows

package main

import "fmt"

func readLocalClipboard() (string, error) {
	return "", fmt.Errorf("clipboard is only implemented on Windows in this build")
}

func writeLocalClipboard(text string) error {
	return fmt.Errorf("clipboard is only implemented on Windows in this build")
}
