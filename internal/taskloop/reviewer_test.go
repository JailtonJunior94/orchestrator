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

	t.Run("repo git com working tree alterado retorna diff real", func(t *testing.T) {
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

		// Segundo commit: modificar arquivo e consolidar baseline limpa.
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

		// Alteracao ainda nao commitada deve aparecer no diff consolidado atual.
		if err := os.WriteFile(file, []byte("package main\n\n// changed again\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel alterar working tree: %v", err)
		}

		got := captureGitDiff(context.Background(), dir)
		// Com working tree alterado, diff deve conter conteudo real.
		if got == "(diff indisponivel)" {
			t.Errorf("esperado diff real mas obteve fallback")
		}
		if !strings.Contains(got, "main.go") {
			t.Errorf("diff nao contem main.go:\n%s", got)
		}
		if !strings.Contains(got, "changed again") {
			t.Errorf("diff nao contem alteracao atual do working tree:\n%s", got)
		}
	})

	t.Run("repo git inclui arquivo untracked no diff consolidado", func(t *testing.T) {
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

		commitCmd := exec.Command("git", "commit", "--allow-empty", "-m", "initial")
		commitCmd.Dir = dir
		if out, err := commitCmd.CombinedOutput(); err != nil {
			t.Skipf("commit falhou: %v — %s", err, out)
		}

		untracked := filepath.Join(dir, "novo.go")
		if err := os.WriteFile(untracked, []byte("package main\n"), 0o644); err != nil {
			t.Fatalf("nao foi possivel criar arquivo untracked: %v", err)
		}

		got := captureGitDiff(context.Background(), dir)
		if got == "(diff indisponivel)" {
			t.Fatalf("esperado diff real para arquivo untracked")
		}
		if !strings.Contains(got, "novo.go") {
			t.Errorf("diff nao contem arquivo untracked:\n%s", got)
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

// --- FinalReviewer tests ---

// stubInvoker implementa AgentInvoker retornando saidas fixas para testes.
type stubInvoker struct {
	stdout string
	err    error
	calls  int
}

func (s *stubInvoker) Invoke(_ context.Context, _, _, _ string) (string, string, int, error) {
	s.calls++
	return s.stdout, "", 0, s.err
}
func (s *stubInvoker) BinaryName() string { return "stub" }

// TestParseVerdict valida extracao de veredito a partir de saida bruta.
func TestParseVerdict(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want ReviewVerdict
	}{
		{"APPROVED literal", "APPROVED", VerdictApproved},
		{"aprovado PT-BR", "Veredicto: aprovado", VerdictApproved},
		{"APPROVED_WITH_REMARKS literal", "APPROVED_WITH_REMARKS", VerdictApprovedWithRemarks},
		{"aprovado com ressalvas PT-BR", "veredicto: aprovado com ressalvas", VerdictApprovedWithRemarks},
		{"approved with remarks EN", "approved with remarks: some issues", VerdictApprovedWithRemarks},
		{"BLOCKED literal", "BLOCKED", VerdictBlocked},
		{"bloqueado PT-BR", "veredicto final: bloqueado por falta de evidencias", VerdictBlocked},
		{"REJECTED literal", "REJECTED", VerdictRejected},
		{"reprovado PT-BR", "veredicto final: reprovado", VerdictRejected},
		{"rejeitado PT-BR", "resultado: rejeitado", VerdictRejected},
		{"ressalvas antes de aprovado prioridade correta", "aprovado com ressalvas — veja achados", VerdictApprovedWithRemarks},
		{"sem palavra chave padrao blocked", "sem informacao", VerdictBlocked},
		// Regressao Bug 2: ancora de linha dedicada evita falso positivo de
		// BLOCKED/REJECTED quando a palavra-chave aparece em texto livre.
		{
			"linha dedicada APPROVED ignora 'blocked' no corpo",
			"Findings:\n- the CI was blocked earlier but recovered\n- nothing else\n\nVerdict: APPROVED",
			VerdictApproved,
		},
		{
			"linha dedicada APPROVED_WITH_REMARKS ignora 'rejected' no corpo",
			"We rejected an alternative approach during analysis.\n\nVerdict: APPROVED_WITH_REMARKS",
			VerdictApprovedWithRemarks,
		},
		{
			"linha dedicada PT-BR ignora 'reprovado' incidental",
			"O CI reprovou uma execucao anterior.\nVeredito final: APROVADO",
			VerdictApproved,
		},
		{
			"linha dedicada com markdown bold",
			"resumo...\n**Veredito:** REJECTED\nfim",
			VerdictRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVerdict(tt.raw)
			if got != tt.want {
				t.Errorf("parseVerdict(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// TestParseFindings valida extracao de achados da saida bruta.
func TestParseFindings(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantLen   int
		wantFirst Finding
	}{
		{
			name:    "achado critical com arquivo e linha",
			raw:     "- [Critical] [handler.go:42] null pointer dereference",
			wantLen: 1,
			wantFirst: Finding{
				Severity: SeverityCritical,
				File:     "handler.go",
				Line:     42,
				Message:  "- [Critical] [handler.go:42] null pointer dereference",
			},
		},
		{
			name:    "achado important sem arquivo",
			raw:     "- [Important] consider adding error handling",
			wantLen: 1,
			wantFirst: Finding{
				Severity: SeverityImportant,
				File:     "",
				Line:     0,
				Message:  "- [Important] consider adding error handling",
			},
		},
		{
			name:    "achado suggestion",
			raw:     "- [Suggestion] rename variable for clarity",
			wantLen: 1,
			wantFirst: Finding{
				Severity: SeveritySuggestion,
				File:     "",
				Line:     0,
				Message:  "- [Suggestion] rename variable for clarity",
			},
		},
		{
			name:      "variante PT-BR critico",
			raw:       "- [critico] [service.go:10] race condition detectada",
			wantLen:   1,
			wantFirst: Finding{Severity: SeverityCritical},
		},
		{
			name:    "linha sem marcador ignorada",
			raw:     "Esta linha nao tem marcador de severidade",
			wantLen: 0,
		},
		{
			name: "multiplos achados em texto misto",
			raw: `Revisao:
- [Critical] [main.go:5] erro critico
- texto sem marcador
- [Important] aviso importante
- [Suggestion] sugestao opcional
Veredicto: APPROVED_WITH_REMARKS`,
			wantLen: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFindings(tt.raw)
			if len(got) != tt.wantLen {
				t.Fatalf("parseFindings() len = %d, want %d\nraw: %q", len(got), tt.wantLen, tt.raw)
			}
			if tt.wantLen > 0 && tt.wantFirst.Severity != "" {
				if got[0].Severity != tt.wantFirst.Severity {
					t.Errorf("got[0].Severity = %q, want %q", got[0].Severity, tt.wantFirst.Severity)
				}
				if tt.wantFirst.File != "" && got[0].File != tt.wantFirst.File {
					t.Errorf("got[0].File = %q, want %q", got[0].File, tt.wantFirst.File)
				}
				if tt.wantFirst.Line != 0 && got[0].Line != tt.wantFirst.Line {
					t.Errorf("got[0].Line = %d, want %d", got[0].Line, tt.wantFirst.Line)
				}
			}
		})
	}
}

// TestParseReviewOutput valida os 3 vereditos com fixtures completas.
func TestParseReviewOutput(t *testing.T) {
	fixtures := map[ReviewVerdict]string{
		VerdictApproved: `Revisao concluida.
Nenhum achado critico identificado.
Veredicto: APPROVED`,

		VerdictApprovedWithRemarks: `Achados:
- [Important] [parser.go:15] consider caching result
- [Suggestion] add more comments

Veredicto: APPROVED_WITH_REMARKS`,

		VerdictRejected: `Achados criticos:
- [Critical] [main.go:42] nil pointer dereference — variavel nao inicializada
- [Critical] [service.go:10] goroutine leak sem cancelamento

Veredicto: REJECTED`,
	}

	for verdict, raw := range fixtures {
		t.Run(string(verdict), func(t *testing.T) {
			result := parseReviewOutput(raw)
			if result.Verdict != verdict {
				t.Errorf("parseReviewOutput().Verdict = %q, want %q", result.Verdict, verdict)
			}
			if result.RawOutput != raw {
				t.Errorf("parseReviewOutput().RawOutput nao preserva entrada original")
			}
			switch verdict {
			case VerdictRejected:
				if len(result.Findings) < 2 {
					t.Errorf("esperado >= 2 achados para REJECTED, got %d", len(result.Findings))
				}
				for _, f := range result.Findings {
					if f.Severity != SeverityCritical {
						t.Errorf("achado inesperado nao-critico: %q", f.Severity)
					}
				}
			case VerdictApprovedWithRemarks:
				if len(result.Findings) < 1 {
					t.Errorf("esperado >= 1 achado para APPROVED_WITH_REMARKS, got %d", len(result.Findings))
				}
			case VerdictApproved:
				// zero achados eh valido para aprovado
			}
		})
	}
}

// TestReviewConsolidated_InvocacaoUnica valida que ReviewConsolidated chama o invoker exatamente 1 vez
// para diffs pequenos.
func TestReviewConsolidated_InvocacaoUnica(t *testing.T) {
	inv := &stubInvoker{stdout: "Veredicto: APPROVED"}
	fr := &defaultFinalReviewer{invoker: inv, workDir: t.TempDir(), model: "", maxDiff: maxDiffPartitionSize}

	diff := "diff --git a/main.go b/main.go\n+// change\n"
	result, err := fr.ReviewConsolidated(context.Background(), diff)
	if err != nil {
		t.Fatalf("ReviewConsolidated retornou erro inesperado: %v", err)
	}
	if inv.calls != 1 {
		t.Errorf("invoker chamado %d vez(es), esperado 1", inv.calls)
	}
	if result.Verdict != VerdictApproved {
		t.Errorf("veredito = %q, want APPROVED", result.Verdict)
	}
}

// TestReviewConsolidated_VerdictosAgregados valida que o pior veredito prevalece ao particionar.
func TestReviewConsolidated_VerdictosAgregados(t *testing.T) {
	// Diff grande o suficiente para ser particionado (maxDiff = 10)
	// Duas particoes: uma retorna APPROVED, outra REJECTED
	callCount := 0
	outputs := []string{
		"Veredicto: APPROVED",
		"- [Critical] [x.go:1] problema\nVeredicto: REJECTED",
	}
	inv := &stubInvokerMulti{outputs: outputs}

	fr := &defaultFinalReviewer{invoker: inv, workDir: t.TempDir(), model: "", maxDiff: 10}

	// Diff com dois arquivos que excedem limite de 10 bytes
	diff := "diff --git a/a.go b/a.go\n+// file a\ndiff --git a/b.go b/b.go\n+// file b\n"
	_ = callCount

	result, err := fr.ReviewConsolidated(context.Background(), diff)
	if err != nil {
		t.Fatalf("ReviewConsolidated retornou erro: %v", err)
	}
	if result.Verdict != VerdictRejected {
		t.Errorf("veredito agregado = %q, want REJECTED (pior deve prevalecer)", result.Verdict)
	}
}

// stubInvokerMulti retorna outputs diferentes a cada chamada.
type stubInvokerMulti struct {
	outputs []string
	calls   int
}

func (s *stubInvokerMulti) Invoke(_ context.Context, _, _, _ string) (string, string, int, error) {
	if s.calls < len(s.outputs) {
		out := s.outputs[s.calls]
		s.calls++
		return out, "", 0, nil
	}
	s.calls++
	return "", "", 0, nil
}
func (s *stubInvokerMulti) BinaryName() string { return "stub-multi" }

// TestPartitionDiff valida a estrategia de particionamento.
func TestPartitionDiff(t *testing.T) {
	tests := []struct {
		name      string
		diff      string
		maxSize   int
		wantParts int
	}{
		{
			name:      "diff pequeno retorna uma particao",
			diff:      "diff --git a/main.go b/main.go\n+// small change\n",
			maxSize:   10000,
			wantParts: 1,
		},
		{
			name:      "diff vazio retorna uma particao",
			diff:      "",
			maxSize:   100,
			wantParts: 1,
		},
		{
			name: "diff com dois arquivos grande particionado",
			diff: strings.Repeat("diff --git a/a.go b/a.go\n+// aaaa\n", 1) +
				strings.Repeat("diff --git a/b.go b/b.go\n+// bbbb\n", 1),
			maxSize:   10, // muito pequeno, forca particao
			wantParts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := partitionDiff(tt.diff, tt.maxSize)
			if len(parts) != tt.wantParts {
				t.Errorf("partitionDiff() len = %d, want %d", len(parts), tt.wantParts)
			}
			// Verifica que o conteudo original e preservado
			if tt.diff != "" {
				combined := strings.Join(parts, "")
				if combined != tt.diff {
					t.Errorf("conteudo reconstruido difere do original")
				}
			}
		})
	}
}

// TestPartitionDiffSingleFileOversize — regressao Bug 3: uma unica seccao
// "diff --git" maior que maxSize deve ser sub-particionada em hunks "@@",
// repetindo o cabecalho do arquivo, de forma que cada particao caiba em maxSize.
func TestPartitionDiffSingleFileOversize(t *testing.T) {
	header := "diff --git a/big.go b/big.go\nindex 1111..2222 100644\n--- a/big.go\n+++ b/big.go\n"
	hunk := func(n int) string {
		return "@@ -1,1 +1,1 @@\n" + strings.Repeat("+linha de conteudo de hunk\n", n)
	}
	diff := header + hunk(50) + hunk(50) + hunk(50)

	// maxSize forca subdivisao por hunks.
	maxSize := 800
	parts := partitionDiff(diff, maxSize)

	if len(parts) < 2 {
		t.Fatalf("len(parts)=%d, esperava >=2 sub-particoes para arquivo unico oversized", len(parts))
	}
	for i, p := range parts {
		if len(p) > maxSize {
			t.Errorf("particao %d tem %d bytes > maxSize=%d (sem truncamento esperado para hunks pequenos)",
				i, len(p), maxSize)
		}
		if !strings.HasPrefix(p, "diff --git a/big.go") {
			cap := 40
			if len(p) < cap {
				cap = len(p)
			}
			t.Errorf("particao %d nao replica cabecalho do arquivo: %q", i, p[:cap])
		}
		if !strings.Contains(p, "@@") {
			t.Errorf("particao %d nao contem hunk: %q", i, p)
		}
	}
}

// TestPartitionDiffOversizeHunkTruncates — quando um unico hunk excede maxSize,
// o particionador aplica truncamento explicito sinalizado por marcador.
func TestPartitionDiffOversizeHunkTruncates(t *testing.T) {
	header := "diff --git a/huge.go b/huge.go\n--- a/huge.go\n+++ b/huge.go\n"
	hunk := "@@ -1,1 +1,1 @@\n" + strings.Repeat("+x\n", 5000)
	diff := header + hunk
	maxSize := 1000
	parts := partitionDiff(diff, maxSize)

	if len(parts) == 0 {
		t.Fatal("nenhuma particao retornada")
	}
	gotMarker := false
	for _, p := range parts {
		if strings.Contains(p, "[truncado: secao excede maxDiffPartitionSize]") {
			gotMarker = true
		}
		if len(p) > maxSize {
			t.Errorf("particao excede maxSize=%d: tem %d bytes", maxSize, len(p))
		}
	}
	if !gotMarker {
		t.Error("marcador de truncamento ausente para hunk oversized")
	}
}

// TestVerdictConstants valida que as constantes estao definidas corretamente.
func TestVerdictConstants(t *testing.T) {
	if VerdictApproved != "APPROVED" {
		t.Errorf("VerdictApproved = %q, want APPROVED", VerdictApproved)
	}
	if VerdictApprovedWithRemarks != "APPROVED_WITH_REMARKS" {
		t.Errorf("VerdictApprovedWithRemarks = %q, want APPROVED_WITH_REMARKS", VerdictApprovedWithRemarks)
	}
	if VerdictBlocked != "BLOCKED" {
		t.Errorf("VerdictBlocked = %q, want BLOCKED", VerdictBlocked)
	}
	if VerdictRejected != "REJECTED" {
		t.Errorf("VerdictRejected = %q, want REJECTED", VerdictRejected)
	}
}

// TestNewFinalReviewer valida que o construtor retorna implementacao nao-nula.
func TestNewFinalReviewer(t *testing.T) {
	inv := &stubInvoker{stdout: "APPROVED"}
	fr := NewFinalReviewer(inv, t.TempDir(), "")
	if fr == nil {
		t.Fatal("NewFinalReviewer retornou nil")
	}
}

// TestErrReviewRejected valida que o sentinel pode ser detectado via errors.Is.
func TestErrReviewRejected(t *testing.T) {
	if !errors.Is(ErrReviewRejected, ErrReviewRejected) {
		t.Error("ErrReviewRejected nao e detectavel via errors.Is")
	}
}

func TestErrReviewBlocked(t *testing.T) {
	if !errors.Is(ErrReviewBlocked, ErrReviewBlocked) {
		t.Error("ErrReviewBlocked nao e detectavel via errors.Is")
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
