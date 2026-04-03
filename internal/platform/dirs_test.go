package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDirResolverUserConfigDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		goos string
		env  map[string]string
		home string
		want string
	}{
		{
			name: "linux default",
			goos: "linux",
			home: filepath.Join("home", "tester"),
			want: filepath.Join("home", "tester", ".config"),
		},
		{
			name: "linux xdg override",
			goos: "linux",
			env:  map[string]string{"XDG_CONFIG_HOME": filepath.Join("data", "config")},
			home: filepath.Join("home", "tester"),
			want: filepath.Join("data", "config"),
		},
		{
			name: "darwin default",
			goos: "darwin",
			home: filepath.Join("Users", "tester"),
			want: filepath.Join("Users", "tester", "Library", "Application Support"),
		},
		{
			name: "windows appdata override",
			goos: "windows",
			env:  map[string]string{"APPDATA": filepath.Join("Users", "tester", "AppData", "Roaming")},
			home: filepath.Join("Users", "tester"),
			want: filepath.Join("Users", "tester", "AppData", "Roaming"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestDirResolver(tt.goos, tt.home, tt.env)
			got, err := resolver.UserConfigDir()
			if err != nil {
				t.Fatalf("UserConfigDir() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("UserConfigDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDirResolverUserStateDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		goos string
		env  map[string]string
		home string
		want string
	}{
		{
			name: "linux default",
			goos: "linux",
			home: filepath.Join("home", "tester"),
			want: filepath.Join("home", "tester", ".local", "state"),
		},
		{
			name: "linux xdg override",
			goos: "linux",
			env:  map[string]string{"XDG_STATE_HOME": filepath.Join("data", "state")},
			home: filepath.Join("home", "tester"),
			want: filepath.Join("data", "state"),
		},
		{
			name: "darwin default",
			goos: "darwin",
			home: filepath.Join("Users", "tester"),
			want: filepath.Join("Users", "tester", "Library", "Application Support"),
		},
		{
			name: "windows localappdata override",
			goos: "windows",
			env:  map[string]string{"LOCALAPPDATA": filepath.Join("Users", "tester", "AppData", "Local")},
			home: filepath.Join("Users", "tester"),
			want: filepath.Join("Users", "tester", "AppData", "Local"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := newTestDirResolver(tt.goos, tt.home, tt.env)
			got, err := resolver.UserStateDir()
			if err != nil {
				t.Fatalf("UserStateDir() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("UserStateDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDirResolverResolveProjectRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "internal", "install")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	resolver := NewDirResolver()
	got, err := resolver.ResolveProjectRoot(nested)
	if err != nil {
		t.Fatalf("ResolveProjectRoot() error = %v", err)
	}
	if got != root {
		t.Fatalf("ResolveProjectRoot() = %q, want %q", got, root)
	}
}

func TestDirResolverResolveProjectRootFallsBackToStart(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "pkg", "module")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	resolver := NewDirResolver()
	got, err := resolver.ResolveProjectRoot(nested)
	if err != nil {
		t.Fatalf("ResolveProjectRoot() error = %v", err)
	}
	if got != nested {
		t.Fatalf("ResolveProjectRoot() = %q, want %q", got, nested)
	}
}

func TestDirResolverUserStateDirPropagatesHomeErrors(t *testing.T) {
	t.Parallel()

	resolver := &DirResolver{
		goos: "linux",
		envLookup: func(string) string {
			return ""
		},
		userHomeDir: func() (string, error) {
			return "", errors.New("boom")
		},
	}

	if _, err := resolver.UserStateDir(); err == nil {
		t.Fatal("UserStateDir() expected error")
	}
}

func TestDirResolverUserHomeDir(t *testing.T) {
	t.Parallel()

	resolver := newTestDirResolver("linux", filepath.Join("home", "tester"), nil)
	got, err := resolver.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir() error = %v", err)
	}
	if want := filepath.Join("home", "tester"); got != want {
		t.Fatalf("UserHomeDir() = %q, want %q", got, want)
	}
}

func newTestDirResolver(goos string, home string, env map[string]string) *DirResolver {
	return &DirResolver{
		goos: goos,
		envLookup: func(key string) string {
			return env[key]
		},
		userHomeDir: func() (string, error) {
			return home, nil
		},
		userConfigDir: func() (string, error) {
			return "", nil
		},
	}
}
