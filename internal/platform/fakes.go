package platform

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FakeCommandRunner is a programmable CommandRunner for tests.
type FakeCommandRunner struct {
	RunFunc          func(ctx context.Context, name string, args []string, stdin string) (CommandResult, error)
	RunStreamingFunc func(ctx context.Context, name string, args []string, stdin string) (*StreamResult, error)
}

// Run executes the programmed fake callback.
func (f FakeCommandRunner) Run(ctx context.Context, name string, args []string, stdin string) (CommandResult, error) {
	if f.RunFunc == nil {
		return CommandResult{}, nil
	}
	return f.RunFunc(ctx, name, args, stdin)
}

// RunStreaming executes the programmed streaming fake callback. If no callback
// is set, it returns an empty reader and a Wait that succeeds immediately.
func (f FakeCommandRunner) RunStreaming(ctx context.Context, name string, args []string, stdin string) (*StreamResult, error) {
	if f.RunStreamingFunc != nil {
		return f.RunStreamingFunc(ctx, name, args, stdin)
	}
	// Default: return empty stdout/stderr with no-op wait.
	_ = strings.NewReader(stdin) // suppress unused warning
	exitCode := 0
	return &StreamResult{
		Stdout:    bytes.NewReader(nil),
		Stderr:    bytes.NewReader(nil),
		StartedAt: time.Now(),
		Wait:      func() error { return nil },
		ExitCode:  func() int { return exitCode },
	}, nil
}

// Verify FakeCommandRunner implements CommandRunner at compile time.
var _ CommandRunner = FakeCommandRunner{}

// NewFakeStreamResult builds a StreamResult from fixed stdout/stderr strings,
// useful in tests that need controlled streaming output.
func NewFakeStreamResult(stdout, stderr string) *StreamResult {
	exitCode := 0
	return &StreamResult{
		Stdout:    strings.NewReader(stdout),
		Stderr:    strings.NewReader(stderr),
		StartedAt: time.Now(),
		Wait:      func() error { return nil },
		ExitCode:  func() int { return exitCode },
	}
}

// FakeEditor is a programmable Editor for tests.
type FakeEditor struct {
	EditFunc func(ctx context.Context, content string) (string, error)
}

// Edit executes the programmed fake callback.
func (f FakeEditor) Edit(ctx context.Context, content string) (string, error) {
	if f.EditFunc == nil {
		return content, nil
	}
	return f.EditFunc(ctx, content)
}

// FakeFileSystem is a lightweight filesystem backed by a temp directory.
type FakeFileSystem struct {
	root string
}

// NewFakeFileSystem creates a tempdir-backed filesystem fake.
func NewFakeFileSystem() (*FakeFileSystem, error) {
	root, err := os.MkdirTemp("", "orq-fs-*")
	if err != nil {
		return nil, err
	}
	return &FakeFileSystem{root: root}, nil
}

// Close removes the fake filesystem root.
func (f *FakeFileSystem) Close() error {
	if f.root == "" {
		return nil
	}
	return os.RemoveAll(f.root)
}

// Root returns the temp directory backing this fake filesystem.
func (f *FakeFileSystem) Root() string {
	return f.root
}

func (f *FakeFileSystem) resolve(path string) (string, error) {
	if path == "" {
		return "", errors.New("path must not be empty")
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	return filepath.Join(f.root, path), nil
}

// MkdirAll creates a directory hierarchy.
func (f *FakeFileSystem) MkdirAll(path string, perm os.FileMode) error {
	resolved, err := f.resolve(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(resolved, perm)
}

// ReadFile reads a file.
func (f *FakeFileSystem) ReadFile(path string) ([]byte, error) {
	resolved, err := f.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(resolved)
}

// WriteFile writes a file.
func (f *FakeFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	resolved, err := f.resolve(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return err
	}
	return os.WriteFile(resolved, data, perm)
}

// Remove removes a file.
func (f *FakeFileSystem) Remove(path string) error {
	resolved, err := f.resolve(path)
	if err != nil {
		return err
	}
	return os.Remove(resolved)
}

// RemoveAll removes a path recursively.
func (f *FakeFileSystem) RemoveAll(path string) error {
	resolved, err := f.resolve(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(resolved)
}

// Rename renames a file or directory.
func (f *FakeFileSystem) Rename(oldPath string, newPath string) error {
	resolvedOld, err := f.resolve(oldPath)
	if err != nil {
		return err
	}
	resolvedNew, err := f.resolve(newPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedNew), 0o755); err != nil {
		return err
	}
	return os.Rename(resolvedOld, resolvedNew)
}

// ReadDir reads a directory.
func (f *FakeFileSystem) ReadDir(path string) ([]fs.DirEntry, error) {
	resolved, err := f.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(resolved)
}

// Stat returns file metadata.
func (f *FakeFileSystem) Stat(path string) (fs.FileInfo, error) {
	resolved, err := f.resolve(path)
	if err != nil {
		return nil, err
	}
	return os.Stat(resolved)
}

// Advance moves the fake clock forward.
func (c *FakeClock) Advance(delta time.Duration) {
	c.current = c.current.Add(delta)
}
