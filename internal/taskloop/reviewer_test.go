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
		TaskFile:       "tasks/prd-feat/task-1.0.md",
		PRDFolder:      "tasks/prd-feat",
		TechSpec:       "tasks/prd-feat/techspec.md",
		TasksFile:      "tasks/prd-feat/tasks.md",
		Diff:           "diff --git a/main.go b/main.go\n+// change",
		CompletedTasks: "1.0 (task anterior)",
		RiskAreas:      "contratos, seguranca",
	}

	customTemplate := `Review task: {{.TaskFile}}
PRD: {{.PRDFolder}}
Spec: {{.TechSpec}}
Tasks: {{.TasksFile}}
Completed: {{.CompletedTasks}}
Risk: {{.RiskAreas}}
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
				"Tasks executadas: 1.0 (task anterior)",
				"Areas de risco: contratos, seguranca",
				"corretude:",
				"regressao:",
				"divida tecnica introduzida:",
				"aprovado / aprovado com ressalvas / reprovado",
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
				"1.0 (task anterior)",
				"contratos, seguranca",
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

// TestDetectRiskAreas valida deteccao de areas de risco a partir de techspec e diff.
func TestDetectRiskAreas(t *testing.T) {
	tests := []struct {
		name     string
		techspec string
		diff     string
		want     []string
		notWant  []string
	}{
		{
			name:     "detecta performance e concorrencia",
			techspec: "O sistema usa goroutine pool com buffer para latencia minima.",
			diff:     "",
			want:     []string{"performance", "concorrencia"},
		},
		{
			name:     "detecta seguranca no diff",
			techspec: "",
			diff:     "+func validateToken(credential string) error {",
			want:     []string{"seguranca"},
		},
		{
			name:     "detecta contratos",
			techspec: "A interface AgentInvoker e o contrato publico.",
			diff:     "",
			want:     []string{"contratos"},
		},
		{
			name:     "detecta persistencia",
			techspec: "Migrar schema do database para incluir nova coluna.",
			diff:     "",
			want:     []string{"persistencia"},
		},
		{
			name:     "fallback quando nenhuma area detectada",
			techspec: "Atualizar documentacao do projeto.",
			diff:     "+// comentario simples",
			want:     []string{"contratos", "seguranca"},
		},
		{
			name:     "techspec e diff vazios retorna fallback",
			techspec: "",
			diff:     "",
			want:     []string{"contratos", "seguranca"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := fs.NewFakeFileSystem()
			if tt.techspec != "" {
				_ = fsys.WriteFile("/work/tasks/prd-test/techspec.md", []byte(tt.techspec))
			}

			got := detectRiskAreas("tasks/prd-test", "/work", tt.diff, fsys)
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("detectRiskAreas() deveria conter %q, obteve: %q", w, got)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(got, nw) {
					t.Errorf("detectRiskAreas() nao deveria conter %q, obteve: %q", nw, got)
				}
			}
		})
	}
}

// TestBugfixBuildPrompt valida os cenarios de construcao do prompt de bugfix.
func TestBugfixBuildPrompt(t *testing.T) {
	data := BugfixTemplateData{
		TaskFile:       "tasks/prd-feat/task-1.0.md",
		PRDFolder:      "tasks/prd-feat",
		TechSpec:       "tasks/prd-feat/techspec.md",
		TasksFile:      "tasks/prd-feat/tasks.md",
		ReviewFindings: "- [main.go:42] Achado 1: variavel nao inicializada\n- [handler.go:10] Achado 2: erro nao tratado",
		Diff:           "diff --git a/main.go b/main.go\n+// change",
	}

	tests := []struct {
		name         string
		data         BugfixTemplateData
		wantContains []string
	}{
		{
			name: "template default resolve todos os placeholders",
			data: data,
			wantContains: []string{
				"tasks/prd-feat/task-1.0.md",
				"tasks/prd-feat",
				"tasks/prd-feat/techspec.md",
				"tasks/prd-feat/tasks.md",
				"variavel nao inicializada",
				"erro nao tratado",
				"diff --git a/main.go b/main.go",
				"AGENTS.md",
				".agents/skills/bugfix/SKILL.md",
				"causa raiz",
				"testes de regressao",
				"go test ./...",
				"go vet ./...",
				"Do NOT modify any other task file.",
				"contratos publicos",
				"tipos de erro",
			},
		},
		{
			name: "template com review findings vazio",
			data: BugfixTemplateData{
				TaskFile:       "tasks/prd-foo/task-2.0.md",
				PRDFolder:      "tasks/prd-foo",
				TechSpec:       "tasks/prd-foo/techspec.md",
				TasksFile:      "tasks/prd-foo/tasks.md",
				ReviewFindings: "",
				Diff:           "(diff indisponivel)",
			},
			wantContains: []string{
				"tasks/prd-foo/task-2.0.md",
				"(diff indisponivel)",
				"skill bugfix",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildBugfixPrompt(tt.data)
			if err != nil {
				t.Fatalf("BuildBugfixPrompt retornou erro inesperado: %v", err)
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("prompt nao contem %q\nprompt:\n%s", want, got)
				}
			}
		})
	}
}

// TestBugfixErrBugfixTemplateInvalidoSentinel verifica que ErrBugfixTemplateInvalido pode ser detectado via errors.Is.
func TestBugfixErrBugfixTemplateInvalidoSentinel(t *testing.T) {
	if !errors.Is(ErrBugfixTemplateInvalido, ErrBugfixTemplateInvalido) {
		t.Error("ErrBugfixTemplateInvalido nao e detectavel via errors.Is")
	}
}

// TestBugfixPromptAgentsMdBeforeSkillMd verifica que AGENTS.md aparece antes de SKILL.md
// no prompt de bugfix (contrato de carga base exigido por RF-04).
func TestBugfixPromptAgentsMdBeforeSkillMd(t *testing.T) {
	data := BugfixTemplateData{
		TaskFile:       "tasks/prd-feat/task-1.0.md",
		PRDFolder:      "tasks/prd-feat",
		TechSpec:       "tasks/prd-feat/techspec.md",
		TasksFile:      "tasks/prd-feat/tasks.md",
		ReviewFindings: "achado critico",
		Diff:           "diff content",
	}

	prompt, err := BuildBugfixPrompt(data)
	if err != nil {
		t.Fatalf("BuildBugfixPrompt retornou erro inesperado: %v", err)
	}

	agentsIdx := strings.Index(prompt, "AGENTS.md")
	skillIdx := strings.Index(prompt, ".agents/skills/bugfix/SKILL.md")
	if agentsIdx < 0 {
		t.Fatal("prompt de bugfix nao contem referencia a AGENTS.md")
	}
	if skillIdx < 0 {
		t.Fatal("prompt de bugfix nao contem referencia a .agents/skills/bugfix/SKILL.md")
	}
	if agentsIdx >= skillIdx {
		t.Errorf("AGENTS.md (pos=%d) deve aparecer antes de SKILL.md (pos=%d) no prompt de bugfix", agentsIdx, skillIdx)
	}
}

// TestBugfixPromptAllMandatorySections verifica que o prompt de bugfix contem
// todas as secoes obrigatorias do template.
func TestBugfixPromptAllMandatorySections(t *testing.T) {
	data := BugfixTemplateData{
		TaskFile:       "tasks/prd-feat/task-1.0.md",
		PRDFolder:      "tasks/prd-feat",
		TechSpec:       "tasks/prd-feat/techspec.md",
		TasksFile:      "tasks/prd-feat/tasks.md",
		ReviewFindings: "- [handler.go:15] null pointer dereference",
		Diff:           "diff --git a/handler.go",
	}

	prompt, err := BuildBugfixPrompt(data)
	if err != nil {
		t.Fatalf("BuildBugfixPrompt retornou erro inesperado: %v", err)
	}

	sections := []string{
		"Contexto da implementacao:",
		"Achados a corrigir (da saida da skill review):",
		"Comportamento esperado apos a correcao:",
		"Invariantes que nao podem mudar:",
		"Regras de execucao nao negociaveis:",
		"Saidas esperadas:",
		"Diff original da implementacao:",
		"Do NOT modify any other task file.",
		"Do NOT modify any row in tasks.md except the current task row.",
	}

	for _, section := range sections {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt de bugfix nao contem secao obrigatoria %q", section)
		}
	}
}

// TestBugfixPromptPreservesMultilineFindings verifica que achados multilinhas
// do reviewer sao preservados integralmente no prompt de bugfix.
func TestBugfixPromptPreservesMultilineFindings(t *testing.T) {
	findings := `Achados criticos:
- [main.go:42] Variavel 'conn' nao inicializada antes do uso
  Sugestao: inicializar com db.Open() antes do loop
- [handler.go:15] Erro nao tratado no retorno de ParseRequest
  Sugestao: adicionar check de erro com return early
- [service.go:88] Race condition no acesso a cache compartilhado
  Sugestao: proteger com sync.Mutex

Veredicto: reprovado`

	data := BugfixTemplateData{
		TaskFile:       "tasks/prd-feat/task-1.0.md",
		PRDFolder:      "tasks/prd-feat",
		TechSpec:       "tasks/prd-feat/techspec.md",
		TasksFile:      "tasks/prd-feat/tasks.md",
		ReviewFindings: findings,
		Diff:           "diff content",
	}

	prompt, err := BuildBugfixPrompt(data)
	if err != nil {
		t.Fatalf("BuildBugfixPrompt retornou erro inesperado: %v", err)
	}

	// Cada linha dos achados deve estar presente no prompt
	for _, line := range strings.Split(findings, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.Contains(prompt, trimmed) {
			t.Errorf("prompt de bugfix nao preservou linha do reviewer: %q", trimmed)
		}
	}
}

// TestFormatCompletedTasks valida formatacao da lista de tasks concluidas.
func TestFormatCompletedTasks(t *testing.T) {
	tests := []struct {
		name          string
		iterations    []IterationResult
		currentTaskID string
		want          string
	}{
		{
			name:          "sem iteracoes anteriores",
			iterations:    nil,
			currentTaskID: "1.0",
			want:          "1.0 (atual)",
		},
		{
			name: "com tasks concluidas anteriormente",
			iterations: []IterationResult{
				{TaskID: "1.0", Title: "setup", PostStatus: "done"},
				{TaskID: "2.0", Title: "implementacao", PostStatus: "done"},
			},
			currentTaskID: "3.0",
			want:          "1.0 (setup), 2.0 (implementacao), 3.0 (atual)",
		},
		{
			name: "ignora tasks nao concluidas",
			iterations: []IterationResult{
				{TaskID: "1.0", Title: "setup", PostStatus: "done"},
				{TaskID: "2.0", Title: "falhou", PostStatus: "failed"},
			},
			currentTaskID: "3.0",
			want:          "1.0 (setup), 3.0 (atual)",
		},
		{
			name: "task atual ja concluida nao duplica",
			iterations: []IterationResult{
				{TaskID: "1.0", Title: "setup", PostStatus: "done"},
			},
			currentTaskID: "1.0",
			want:          "1.0 (setup)",
		},
		{
			name: "deduplica iteracoes repetidas da mesma task",
			iterations: []IterationResult{
				{TaskID: "1.0", Title: "setup", PostStatus: "done"},
				{TaskID: "1.0", Title: "setup", PostStatus: "done"},
			},
			currentTaskID: "2.0",
			want:          "1.0 (setup), 2.0 (atual)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCompletedTasks(tt.iterations, tt.currentTaskID)
			if got != tt.want {
				t.Errorf("formatCompletedTasks()\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
