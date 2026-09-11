package handshake_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/handshake"
)

func TestNewWaiterEmptySessionDirCreatesOwnUniqueDir(t *testing.T) {
	t.Parallel()

	wa, err := handshake.NewCatalog().NewWaiter("", time.Second)
	if err != nil {
		t.Fatalf("NewWaiter(\"\"): %v", err)
	}
	wb, err := handshake.NewCatalog().NewWaiter("", time.Second)
	if err != nil {
		t.Fatalf("NewWaiter(\"\"): %v", err)
	}
	if wa.Path() == wb.Path() {
		t.Fatalf("own-dir sentinel path is not unique per session: %q == %q", wa.Path(), wb.Path())
	}
	dir := filepath.Dir(wa.Path())
	if err := wa.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Fatalf("Close() did not remove own session dir %q: stat err = %v", dir, statErr)
	}
	_ = wb.Close()
}

func TestNewWaiterPathIsUniquePerSessionDir(t *testing.T) {
	t.Parallel()

	dirA := t.TempDir()
	dirB := t.TempDir()

	wa, err := handshake.NewCatalog().NewWaiter(dirA, time.Second)
	if err != nil {
		t.Fatalf("NewWaiter(dirA): %v", err)
	}
	wb, err := handshake.NewCatalog().NewWaiter(dirB, time.Second)
	if err != nil {
		t.Fatalf("NewWaiter(dirB): %v", err)
	}
	if wa.Path() == wb.Path() {
		t.Fatalf("sentinel path is not unique per session dir: %q == %q", wa.Path(), wb.Path())
	}
	if filepath.Dir(wa.Path()) != dirA {
		t.Fatalf("sentinel path %q not under session dir %q", wa.Path(), dirA)
	}
}

func TestNewWaiterRemovesStaleSentinelBeforeSpawn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stale := filepath.Join(dir, handshake.SentinelFileName)
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatalf("seed stale sentinel: %v", err)
	}

	w, err := handshake.NewCatalog().NewWaiter(dir, 50*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWaiter: %v", err)
	}
	if _, statErr := os.Stat(w.Path()); !os.IsNotExist(statErr) {
		t.Fatalf("stale sentinel not removed before spawn: stat err = %v", statErr)
	}

	err = w.Wait(context.Background())
	if !errors.Is(err, handshake.ErrSignalNotReceived) {
		t.Fatalf("Wait() after removing stale sentinel = %v; want ErrSignalNotReceived (residual sentinel must not pass the handshake)", err)
	}
}

func TestWaiterSucceedsWhenSignalArrives(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w, err := handshake.NewCatalog().NewWaiter(dir, time.Second)
	if err != nil {
		t.Fatalf("NewWaiter: %v", err)
	}

	go func() {
		time.Sleep(30 * time.Millisecond)
		_ = os.WriteFile(w.Path(), []byte{}, 0o644)
	}()

	if err := w.Wait(context.Background()); err != nil {
		t.Fatalf("Wait() = %v; want nil once sentinel is written", err)
	}
}

func TestWaiterAbortsWithoutSignalBeforeTimeout(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w, err := handshake.NewCatalog().NewWaiter(dir, 40*time.Millisecond)
	if err != nil {
		t.Fatalf("NewWaiter: %v", err)
	}

	start := time.Now()
	err = w.Wait(context.Background())
	elapsed := time.Since(start)

	if !errors.Is(err, handshake.ErrSignalNotReceived) {
		t.Fatalf("Wait() = %v; want ErrSignalNotReceived", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("Wait() took %v; want bounded by timeout", elapsed)
	}
}

func TestWaiterHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	w, err := handshake.NewCatalog().NewWaiter(dir, time.Minute)
	if err != nil {
		t.Fatalf("NewWaiter: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err = w.Wait(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait() = %v; want context.Canceled", err)
	}
}
