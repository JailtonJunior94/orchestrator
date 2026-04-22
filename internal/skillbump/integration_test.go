//go:build integration

package skillbump_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillbump"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestIntegrationSkillBump(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git nao disponivel no PATH")
	}

	dir := t.TempDir()
	skillsDir := ".agents/skills"
	skillName := "execute-task"
	skillPath := filepath.Join(dir, skillsDir, skillName, "SKILL.md")

	// Inicializar repositorio git
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")

	// Criar skill inicial
	skillContent := "---\nname: execute-task\nversion: 1.0.0\ndescription: skill execute-task\n---\n\n# Corpo\n"
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "chore: initial commit")
	gitRun(t, dir, "tag", "v1.0.0")

	// Alterar skill apos a tag
	updatedContent := "---\nname: execute-task\nversion: 1.0.0\ndescription: skill execute-task\n---\n\n# Corpo atualizado\n"
	if err := os.WriteFile(skillPath, []byte(updatedContent), 0o644); err != nil {
		t.Fatalf("WriteFile update: %v", err)
	}
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "feat(execute-task): add retry logic")

	fsys := fs.NewOSFileSystem()
	printer := &output.Printer{Out: os.Stdout, Err: os.Stderr}
	svc := skillbump.NewService(fsys, printer)

	t.Run("dry-run nao altera arquivo", func(t *testing.T) {
		results, err := svc.Execute(dir, skillsDir, true)
		if err != nil && !errors.Is(err, skillbump.ErrNoChanges) {
			t.Fatalf("Execute(dry-run) error = %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Execute(dry-run) = %d results, want 1", len(results))
		}
		if results[0].SkillName != skillName {
			t.Errorf("SkillName = %q, want %q", results[0].SkillName, skillName)
		}
		if results[0].PreviousVersion != "1.0.0" {
			t.Errorf("PreviousVersion = %q, want 1.0.0", results[0].PreviousVersion)
		}
		if results[0].NewVersion != "1.1.0" {
			t.Errorf("NewVersion = %q, want 1.1.0", results[0].NewVersion)
		}
		// Arquivo nao deve ter sido alterado
		got, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if !strings.Contains(string(got), "version: 1.0.0") {
			t.Errorf("dry-run alterou o arquivo: %s", string(got))
		}
	})

	t.Run("execucao real aplica bump", func(t *testing.T) {
		results, err := svc.Execute(dir, skillsDir, false)
		if err != nil && !errors.Is(err, skillbump.ErrNoChanges) {
			t.Fatalf("Execute error = %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Execute() = %d results, want 1", len(results))
		}
		if results[0].NewVersion != "1.1.0" {
			t.Errorf("NewVersion = %q, want 1.1.0", results[0].NewVersion)
		}
		got, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("ReadFile: %v", err)
		}
		if !strings.Contains(string(got), "version: 1.1.0") {
			t.Errorf("arquivo nao foi atualizado para 1.1.0: %s", string(got))
		}
	})
}

func TestIntegrationSkillBumpNoTag(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git nao disponivel no PATH")
	}

	dir := t.TempDir()
	skillsDir := ".agents/skills"
	skillPath := filepath.Join(dir, skillsDir, "execute-task", "SKILL.md")

	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.email", "test@test.com")
	gitRun(t, dir, "config", "user.name", "Test")

	skillContent := "---\nname: execute-task\nversion: 1.0.0\ndescription: skill execute-task\n---\n\n# Corpo\n"
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "-c", "commit.gpgsign=false", "commit", "-m", "feat: initial skill")

	fsys := fs.NewOSFileSystem()
	printer := &output.Printer{Out: os.Stdout, Err: os.Stderr}
	svc := skillbump.NewService(fsys, printer)

	_, err := svc.Execute(dir, skillsDir, false)
	if !errors.Is(err, skillbump.ErrNoTagFound) {
		t.Fatalf("Execute() error = %v, want ErrNoTagFound", err)
	}
}
