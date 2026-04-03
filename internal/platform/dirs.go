package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var errHomeDirNotFound = errors.New("home directory not found")

// DirResolver resolves project and user directories in a cross-platform way.
type DirResolver struct {
	goos          string
	envLookup     func(string) string
	userHomeDir   func() (string, error)
	userConfigDir func() (string, error)
}

// UserHomeDir returns the normalized home directory for the current user.
func (r *DirResolver) UserHomeDir() (string, error) {
	return r.homeDir()
}

// NewDirResolver creates the production directory resolver.
func NewDirResolver() *DirResolver {
	return &DirResolver{
		goos:          runtime.GOOS,
		envLookup:     os.Getenv,
		userHomeDir:   os.UserHomeDir,
		userConfigDir: os.UserConfigDir,
	}
}

// ResolveProjectRoot returns the detected project root starting from the input path.
func (r *DirResolver) ResolveProjectRoot(start string) (string, error) {
	if start == "" {
		return "", errors.New("start path must not be empty")
	}

	absoluteStart, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve absolute start path: %w", err)
	}

	info, err := os.Stat(absoluteStart)
	if err != nil {
		return "", fmt.Errorf("stat start path %q: %w", absoluteStart, err)
	}

	current := absoluteStart
	if !info.IsDir() {
		current = filepath.Dir(absoluteStart)
	}
	fallback := current

	for {
		if r.isProjectRoot(current) {
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return fallback, nil
		}
		current = parent
	}
}

// UserConfigDir returns the provider configuration base directory for the current user.
func (r *DirResolver) UserConfigDir() (string, error) {
	homeDir, err := r.homeDir()
	if err != nil {
		return "", err
	}

	switch r.goos {
	case "windows":
		if value := r.envLookup("APPDATA"); value != "" {
			return filepath.Clean(value), nil
		}
		return filepath.Join(homeDir, "AppData", "Roaming"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support"), nil
	default:
		if value := r.envLookup("XDG_CONFIG_HOME"); value != "" {
			return filepath.Clean(value), nil
		}
		return filepath.Join(homeDir, ".config"), nil
	}
}

// UserStateDir returns the ORQ state base directory for the current user.
func (r *DirResolver) UserStateDir() (string, error) {
	homeDir, err := r.homeDir()
	if err != nil {
		return "", err
	}

	switch r.goos {
	case "windows":
		if value := r.envLookup("LOCALAPPDATA"); value != "" {
			return filepath.Clean(value), nil
		}
		return filepath.Join(homeDir, "AppData", "Local"), nil
	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support"), nil
	default:
		if value := r.envLookup("XDG_STATE_HOME"); value != "" {
			return filepath.Clean(value), nil
		}
		return filepath.Join(homeDir, ".local", "state"), nil
	}
}

func (r *DirResolver) homeDir() (string, error) {
	if r.userHomeDir == nil {
		return "", errHomeDirNotFound
	}

	homeDir, err := r.userHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	if homeDir == "" {
		return "", errHomeDirNotFound
	}
	return filepath.Clean(homeDir), nil
}

func (r *DirResolver) isProjectRoot(path string) bool {
	markers := []string{
		".git",
		"go.mod",
		"Taskfile.yml",
		"AGENTS.md",
	}

	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(path, marker)); err == nil {
			return true
		}
	}

	return false
}
