package handshake

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const SentinelFileName = ".opencode-governance.sentinel"

const DefaultTimeout = 10 * time.Second

const pollInterval = 25 * time.Millisecond

var ErrSignalNotReceived = errors.New("handshake: governance plugin load signal not received before first prompt")

type Waiter struct {
	path    string
	dir     string
	timeout time.Duration
}

func (c *Catalog) NewWaiter(sessionDir string, timeout time.Duration) (*Waiter, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	dir := sessionDir
	ownDir := false
	if dir == "" {
		tmp, err := os.MkdirTemp("", "aispec-opencode-handshake-*")
		if err != nil {
			return nil, fmt.Errorf("handshake: create session dir: %w", err)
		}
		dir = tmp
		ownDir = true
	}
	path := filepath.Join(dir, SentinelFileName)
	if err := removeStale(path); err != nil {
		if ownDir {
			_ = os.RemoveAll(dir)
		}
		return nil, err
	}
	w := &Waiter{path: path, timeout: timeout}
	if ownDir {
		w.dir = dir
	}
	return w, nil
}

func (w *Waiter) Close() error {
	if w.dir == "" {
		return nil
	}
	return os.RemoveAll(w.dir)
}

func removeStale(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("handshake: remove stale sentinel: %w", err)
	}
	return nil
}

func (w *Waiter) Path() string { return w.path }

func (w *Waiter) Wait(ctx context.Context) error {
	deadline := time.Now().Add(w.timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(w.path); err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return ErrSignalNotReceived
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
