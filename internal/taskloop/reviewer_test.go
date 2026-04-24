package taskloop

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// TestReviewBuildPrompt valida os cenarios de construcao do prompt de revisao.
func TestReviewBuildPrompt(t *testing.T) {
	data := ReviewTemplateData{
		TaskFile:  "tasks/prd-feat/task-1.0.md",
		PRDFolder: "tasks/prd-feat",
		TechSpec:  "tasks/prd-feat/techspec.md",
		TasksFile: "tasks/prd-feat/tasks.md",
		Diff:      "diff --git a/main.go b/main.go\n+// change",
	}

	customTemplate := `Review task: {{.TaskFile}}
PRD: {{.PRDFolder}}
Spec: {{.TechSpec}}
Tasks: {{.TasksFile}}
Diff: {{.Diff}}`

	invalidTemplate := `{{.Unclosed`

	tests := []struct {
		name         string
		templatePath string
		setupFsys    func() fs.FileSystem
		data         ReviewTemplateData
		wantErr      bool
		wantSentinel error
		wantContains []string
	}{
		{
			name:         "template default resolve todos os placeholders",
			templatePath: "",
			setupFsys:    func() fs.FileSystem { return fs.NewFakeFileSystem() },
			data:         data,
			wantErr:      false,
			wantContains: []string{
				"tasks/prd-feat/task-1.0.md",
				"tasks/prd-feat",
				"tasks/prd-feat/techspec.md",
				"tasks/prd-feat/tasks.md",
				"diff --git a/main.go b/main.go",
				".agents/skills/review/SKILL.md",
				"AGENTS.md",
				"Do NOT modify any task file or any row in tasks.md.",
			},
		},
		{
			name:         "template custom valido carregado do filesystem",
			templatePath: "/custom/review.tmpl",
			setupFsys: func() fs.FileSystem {
				fsys := fs.NewFakeFileSystem()
				_ = fsys.WriteFile("/custom/review.tmpl", []byte(customTemplate))
				return fsys
			},
			data:    data,
			wantErr: false,
			wantContains: []string{
				"tasks/prd-feat/task-1.0.md",
				"tasks/prd-feat",
				"tasks/prd-feat/techspec.md",
				"tasks/prd-feat/tasks.md",
				"diff --git a/main.go b/main.go",
			},
		},
		{
			name:         "template com sintaxe invalida retorna ErrTemplateInvalido",
			templatePath: "/invalid/template.tmpl",
			setupFsys: func() fs.FileSystem {
				fsys := fs.NewFakeFileSystem()
				_ = fsys.WriteFile("/invalid/template.tmpl", []byte(invalidTemplate))
				return fsys
			},
			data:         data,
			wantErr:      true,
			wantSentinel: ErrTemplateInvalido,
		},
		{
			name:         "arquivo de template inexistente retorna ErrTemplateInvalido",
			templatePath: "/nao/existe.tmpl",
			setupFsys:    func() fs.FileSystem { return fs.NewFakeFileSystem() },
			data:         data,
			wantErr:      true,
			wantSentinel: ErrTemplateInvalido,
		},
		{
			name:         "template default com diff vazio ainda resolve placeholders",
			templatePath: "",
			setupFsys:    func() fs.FileSystem { return fs.NewFakeFileSystem() },
			data: ReviewTemplateData{
				TaskFile:  "tasks/prd-foo/task-2.0.md",
				PRDFolder: "tasks/prd-foo",
				TechSpec:  "tasks/prd-foo/techspec.md",
				TasksFile: "tasks/prd-foo/tasks.md",
				Diff:      "(diff indisponivel)",
			},
			wantErr: false,
			wantContains: []string{
				"tasks/prd-foo/task-2.0.md",
				"(diff indisponivel)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := tt.setupFsys()
			got, err := BuildReviewPrompt(tt.templatePath, tt.data, fsys)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("esperado erro, mas BuildReviewPrompt retornou nil")
				}
				if tt.wantSentinel != nil && !errors.Is(err, tt.wantSentinel) {
					t.Errorf("errors.Is(err, %v) = false; err = %v", tt.wantSentinel, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildReviewPrompt retornou erro inesperado: %v", err)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("prompt nao contem %q\nprompt:\n%s", want, got)
				}
			}
		})
	}
}

// TestReviewErrTemplateInvalidoSentinel verifica que ErrTemplateInvalido pode ser detectado via errors.Is.
func TestReviewErrTemplateInvalidoSentinel(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	_ = fsys.WriteFile("/bad.tmpl", []byte("{{unclosed"))
	_, err := BuildReviewPrompt("/bad.tmpl", ReviewTemplateData{}, fsys)
	if err == nil {
		t.Fatal("esperado ErrTemplateInvalido, mas nenhum erro")
	}
	if !errors.Is(err, ErrTemplateInvalido) {
		t.Errorf("errors.Is(err, ErrTemplateInvalido) = false; err = %v", err)
	}
}

// TestReviewCaptureGitDiff cobre o cenario sem git e o escopo seguro derivado da iteracao atual.
func TestReviewCaptureGitDiff(t *testing.T) {
	t.Run("dir sem git retorna fallback", func(t *testing.T) {
		dir := t.TempDir()
		got := captureGitDiff(context.Background(), dir, []string{"main.go"})
		if got != gitDiffUnavailable {
			t.Errorf("esperado fallback, got %q", got)
		}
	})

	t.Run("repo git retorna apenas alteracoes novas da iteracao atual", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git nao disponivel no PATH")
		}

		dir := t.TempDir()
		initGitRepo(t, dir)

		unrelated := filepath.Join(dir, "unrelated.go")
		if err := os.WriteFile(unrelated, []byte("package main\n\n// preexistente\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel criar arquivo preexistente: %v", err)
		}

		before, err := captureGitStatusSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("captureGitStatusSnapshot() retornou erro inesperado: %v", err)
		}

		safe := filepath.Join(dir, "safe.go")
		if err := os.WriteFile(safe, []byte("package main\n\n// iteracao atual\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel criar arquivo seguro: %v", err)
		}

		after, err := captureGitStatusSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("captureGitStatusSnapshot() retornou erro inesperado: %v", err)
		}

		paths := changedGitPathsSince(before, after)
		if len(paths) != 1 || paths[0] != "safe.go" {
			t.Fatalf("changedGitPathsSince() = %v, esperado [safe.go]", paths)
		}

		got := captureGitDiff(context.Background(), dir, paths)
		if !strings.Contains(got, "safe.go") {
			t.Fatalf("diff nao contem safe.go:\n%s", got)
		}
		if strings.Contains(got, "unrelated.go") {
			t.Fatalf("diff nao deveria expor unrelated.go:\n%s", got)
		}
	})

	t.Run("arquivo ja sujo entra no diff quando recebe alteracao nova", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git nao disponivel no PATH")
		}

		dir := t.TempDir()
		initGitRepo(t, dir)

		mainFile := filepath.Join(dir, "main.go")
		if err := os.WriteFile(mainFile, []byte("package main\n\n// alteracao preexistente\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel criar alteracao preexistente: %v", err)
		}

		before, err := captureGitStatusSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("captureGitStatusSnapshot() retornou erro inesperado: %v", err)
		}

		if err := os.WriteFile(mainFile, []byte("package main\n\n// alteracao preexistente\n// alteracao da iteracao atual\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel aplicar alteracao da iteracao: %v", err)
		}

		after, err := captureGitStatusSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("captureGitStatusSnapshot() retornou erro inesperado: %v", err)
		}

		paths := changedGitPathsSince(before, after)
		if len(paths) != 1 || paths[0] != "main.go" {
			t.Fatalf("changedGitPathsSince() = %v, esperado [main.go]", paths)
		}

		got := captureGitDiff(context.Background(), dir, paths)
		if !strings.Contains(got, "alteracao da iteracao atual") {
			t.Fatalf("diff deveria refletir alteracao nova em arquivo ja sujo:\n%s", got)
		}
	})

	t.Run("sem alteracao segura nova retorna fallback seguro", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git nao disponivel no PATH")
		}

		dir := t.TempDir()
		initGitRepo(t, dir)

		before, err := captureGitStatusSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("captureGitStatusSnapshot() retornou erro inesperado: %v", err)
		}
		after, err := captureGitStatusSnapshot(context.Background(), dir)
		if err != nil {
			t.Fatalf("captureGitStatusSnapshot() retornou erro inesperado: %v", err)
		}

		got := captureGitDiff(context.Background(), dir, changedGitPathsSince(before, after))
		if got != gitDiffUnavailableSafe {
			t.Errorf("esperado fallback seguro, got %q", got)
		}
	})
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	setupCmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Tester"},
		{"git", "config", "commit.gpgsign", "false"},
	}
	for _, args := range setupCmds {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("falha ao configurar repo git: %v — %s", err, out)
		}
	}

	mainFile := filepath.Join(dir, "main.go")
	if err := os.WriteFile(mainFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("nao foi possivel criar arquivo inicial: %v", err)
	}

	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = dir
	if out, err := addCmd.CombinedOutput(); err != nil {
		t.Skipf("git add falhou: %v — %s", err, out)
	}

	commitCmd := exec.Command("git", "commit", "-m", "initial")
	commitCmd.Dir = dir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		t.Skipf("commit inicial falhou: %v — %s", err, out)
	}
}
