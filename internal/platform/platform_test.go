package platform_test

import (
	"runtime"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/platform"
)

func TestCurrent_matchesRuntime(t *testing.T) {
	info := platform.Current()
	if info.OS != runtime.GOOS {
		t.Errorf("Current().OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Current().Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
}

func TestSupportsSymlinks_nonWindows(t *testing.T) {
	info := platform.Info{OS: "linux", Arch: "amd64"}
	if !info.SupportsSymlinks() {
		t.Error("linux should support symlinks")
	}
}

func TestSupportsSymlinks_darwin(t *testing.T) {
	info := platform.Info{OS: "darwin", Arch: "arm64"}
	if !info.SupportsSymlinks() {
		t.Error("darwin should support symlinks")
	}
}

func TestSupportsSymlinks_windows(t *testing.T) {
	info := platform.Info{OS: "windows", Arch: "amd64"}
	if info.SupportsSymlinks() {
		t.Error("windows should not report symlink support")
	}
}

func TestCurrent_supportsSymlinks(t *testing.T) {
	info := platform.Current()
	// On any non-Windows CI or dev machine this test runs, symlinks should be supported.
	if runtime.GOOS != "windows" && !info.SupportsSymlinks() {
		t.Errorf("Current().SupportsSymlinks() = false on %s, expected true", runtime.GOOS)
	}
}
