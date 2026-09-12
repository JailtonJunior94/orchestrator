//go:build windows

package durable

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var layerLockReadRetryAttempts = 3

var layerLockReadRetryDelay = 5 * time.Millisecond

func acquireLayerLock(path string) (func() error, error) {
	if err := createLayerLockFile(path); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("durable: create layer lock: %w", err)
		}
		if takeoverErr := takeOverStaleLayerLock(path); takeoverErr != nil {
			return nil, takeoverErr
		}
		if err := createLayerLockFile(path); err != nil {
			return nil, ErrLayerLocked
		}
	}
	return func() error {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("durable: release layer lock: %w", err)
		}
		return nil
	}, nil
}

func createLayerLockFile(path string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".lock-tmp-*")
	if err != nil {
		return fmt.Errorf("durable: create layer lock temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	ref := CurrentProcessRef()
	content := fmt.Sprintf("%d\n%s\n%s\n", ref.PID, ref.Hostname, ref.StartedAt.Format(time.RFC3339Nano))
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("durable: write layer lock temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("durable: sync layer lock temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("durable: close layer lock temp file: %w", err)
	}

	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return err
		}
		return fmt.Errorf("durable: publish layer lock: %w", err)
	}
	return nil
}

func takeOverStaleLayerLock(path string) error {
	raw, ref, ok := readLayerLockRawWithRetry(path)
	if !ok {
		return removeLayerLockFileIfUnchanged(path, raw)
	}
	liveness := DefaultLivenessProbe.Probe(ref)
	if liveness.Reliable && liveness.Alive {
		return ErrLayerLocked
	}
	return removeLayerLockFileIfUnchanged(path, raw)
}

var readLayerLockRawHook = readLayerLockRaw

func readLayerLockRawWithRetry(path string) ([]byte, ProcessRef, bool) {
	var lastRaw []byte
	for attempt := 0; attempt < layerLockReadRetryAttempts; attempt++ {
		raw, ref, ok := readLayerLockRawHook(path)
		lastRaw = raw
		if ok {
			return raw, ref, true
		}
		if attempt < layerLockReadRetryAttempts-1 {
			time.Sleep(layerLockReadRetryDelay)
		}
	}
	return lastRaw, ProcessRef{}, false
}

func removeLayerLockFileIfUnchanged(path string, expected []byte) error {
	current, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("durable: read layer lock before takeover: %w", err)
	}
	if !bytes.Equal(current, expected) {
		return ErrLayerLocked
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("durable: remove stale layer lock: %w", err)
	}
	return nil
}

func readLayerLockRaw(path string) ([]byte, ProcessRef, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ProcessRef{}, false
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 3 {
		return data, ProcessRef{}, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return data, ProcessRef{}, false
	}
	startedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(lines[2]))
	if err != nil {
		return data, ProcessRef{}, false
	}
	return data, ProcessRef{PID: pid, Hostname: strings.TrimSpace(lines[1]), StartedAt: startedAt}, true
}
