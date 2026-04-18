package git

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Repository abstrai deteccao e operacoes git.
type Repository interface {
	IsRepo(path string) bool
	Root(path string) (string, error)
	RemoteURL(path string) (string, error)
}

// CLIRepository usa o binario git do sistema.
type CLIRepository struct{}

func NewCLIRepository() *CLIRepository { return &CLIRepository{} }

func (r *CLIRepository) IsRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func (r *CLIRepository) Root(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

func (r *CLIRepository) RemoteURL(path string) (string, error) {
	cmd := exec.Command("git", "-C", path, "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
