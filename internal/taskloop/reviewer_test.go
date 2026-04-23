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

// TestReviewCaptureGitDiff cobre o cenario sem git (dir nao-git) e o cenario com repo git real.
func TestReviewCaptureGitDiff(t *testing.T) {
	t.Run("dir sem git retorna fallback", func(t *testing.T) {
		dir := t.TempDir()
		got := captureGitDiff(context.Background(), dir)
		if got != "(diff indisponivel)" {
			t.Errorf("esperado fallback, got %q", got)
		}
	})

	t.Run("repo git com dois commits retorna diff ou fallback sem panic", func(t *testing.T) {
		// Verificar se git esta disponivel
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git nao disponivel no PATH")
		}

		dir := t.TempDir()

		// Inicializar repo e criar dois commits com arquivo real para gerar diff
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

		// Primeiro commit: criar arquivo
		file := filepath.Join(dir, "main.go")
		if err := os.WriteFile(file, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel criar arquivo: %v", err)
		}
		addCmd := exec.Command("git", "add", ".")
		addCmd.Dir = dir
		if out, err := addCmd.CombinedOutput(); err != nil {
			t.Skipf("git add falhou: %v — %s", err, out)
		}
		commitCmd := exec.Command("git", "commit", "-m", "initial")
		commitCmd.Dir = dir
		if out, err := commitCmd.CombinedOutput(); err != nil {
			t.Skipf("primeiro commit falhou: %v — %s", err, out)
		}

		// Segundo commit: modificar arquivo
		if err := os.WriteFile(file, []byte("package main\n\n// changed\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel modificar arquivo: %v", err)
		}
		addCmd2 := exec.Command("git", "add", ".")
		addCmd2.Dir = dir
		if out, err := addCmd2.CombinedOutput(); err != nil {
			t.Skipf("git add (2) falhou: %v — %s", err, out)
		}
		commitCmd2 := exec.Command("git", "commit", "-m", "second")
		commitCmd2.Dir = dir
		if out, err := commitCmd2.CombinedOutput(); err != nil {
			t.Skipf("segundo commit falhou: %v — %s", err, out)
		}

		got := captureGitDiff(context.Background(), dir)
		// Com dois commits e arquivo alterado, diff deve conter conteudo real
		if got == "(diff indisponivel)" {
			t.Errorf("esperado diff real mas obteve fallback")
		}
		if !strings.Contains(got, "main.go") {
			t.Errorf("diff nao contem main.go:\n%s", got)
		}
	})

	t.Run("repo git com apenas um commit retorna fallback", func(t *testing.T) {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("git nao disponivel no PATH")
		}

		dir := t.TempDir()
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

		// Apenas um commit — HEAD~1 nao existe
		commitCmd := exec.Command("git", "commit", "--allow-empty", "-m", "initial")
		commitCmd.Dir = dir
		if out, err := commitCmd.CombinedOutput(); err != nil {
			t.Skipf("commit falhou: %v — %s", err, out)
		}

		got := captureGitDiff(context.Background(), dir)
		if got != "(diff indisponivel)" {
			t.Errorf("esperado fallback com apenas um commit, got %q", got)
		}
	})
}
