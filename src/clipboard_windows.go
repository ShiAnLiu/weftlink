//go:build windows

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.design/x/clipboard"
)

var (
	clipboardOnce sync.Once
	clipboardErr  error
	clipboardMu   sync.Mutex
)

func ensureClipboard() error {
	clipboardOnce.Do(func() {
		clipboardErr = clipboard.Init()
	})
	if clipboardErr != nil {
		return fmt.Errorf("clipboard init: %w", clipboardErr)
	}
	return nil
}

func readLocalClipboard() (string, error) {
	if err := ensureClipboard(); err != nil {
		return "", err
	}
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	var ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var data, err = clipboard.Read(ctx, clipboard.FmtText)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeLocalClipboard(text string) error {
	if err := ensureClipboard(); err != nil {
		return err
	}
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	var ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := clipboard.Write(ctx, clipboard.FmtText, []byte(text))
	return err
}
