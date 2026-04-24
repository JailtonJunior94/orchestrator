package taskloop

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ErrTemplateInvalido indica erro no parsing ou execucao do template de revisao.
var ErrTemplateInvalido = errors.New("template de revisao invalido")

// defaultReviewTemplate e o template embutido via go:embed.
//
//go:embed review_template.tmpl
var defaultReviewTemplate string

// defaultBugfixTemplate e o template embutido via go:embed para o prompt de bugfix.
//
//go:embed bugfix_template.tmpl
var defaultBugfixTemplate string

// ReviewTemplateData agrupa os placeholders do template de revisao.
type ReviewTemplateData struct {
	TaskFile       string
	PRDFolder      string
	TechSpec       string
	TasksFile      string
	Diff           string
	CompletedTasks string // Lista de tasks executadas no bundle ate o momento
	RiskAreas      string // Areas de risco detectadas (performance, seguranca, contratos, concorrencia)
}

// BuildReviewPrompt constroi o prompt de revisao a partir do template.
// Se templatePath != "", carrega template customizado do disco via fsys.
// Se templatePath == "", usa defaultReviewTemplate embutido.
// Retorna erro wrappado com ErrTemplateInvalido em caso de falha de parsing ou execucao.
func BuildReviewPrompt(templatePath string, data ReviewTemplateData, fsys fs.FileSystem) (string, error) {
	var tmplContent string
	if templatePath != "" {
		raw, err := fsys.ReadFile(templatePath)
		if err != nil {
			return "", fmt.Errorf("%w: nao foi possivel ler arquivo %q: %v", ErrTemplateInvalido, templatePath, err)
		}
		tmplContent = string(raw)
	} else {
		tmplContent = defaultReviewTemplate
	}

	tmpl, err := template.New("review").Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateInvalido, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateInvalido, err)
	}

	return buf.String(), nil
}

// captureGitDiff executa `git diff HEAD~1` no workDir e retorna o diff.
// Se o comando falhar (repo nao-git, sem commits suficientes) ou o diff for vazio,
// retorna "(diff indisponivel)". Nao e erro bloqueante.
func captureGitDiff(ctx context.Context, workDir string) string {
	cmd := exec.CommandContext(ctx, "git", "diff", "HEAD~1")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return "(diff indisponivel)"
	}
	if len(out) == 0 {
		return "(diff indisponivel)"
	}
	return string(out)
}

// detectRiskAreas analisa o conteudo combinado de techspec e diff para detectar
// areas de risco relevantes para a revisao.
func detectRiskAreas(prdFolder, workDir string, diff string, fsys fs.FileSystem) string {
	techspecPath := filepath.Join(workDir, prdFolder, "techspec.md")
	techspec, _ := fsys.ReadFile(techspecPath)
	combined := strings.ToLower(string(techspec) + "\n" + diff)

	var areas []string

	if containsAnyPattern(combined, "performance", "latencia", "latency", "benchmark", "cache", "pool", "buffer") {
		areas = append(areas, "performance")
	}
	if containsAnyPattern(combined, "seguranca", "security", "auth", "credential", "token", "injection", "xss", "csrf") {
		areas = append(areas, "seguranca")
	}
	if containsAnyPattern(combined, "interface ", "contrato", "contract", "assinatura publica", "public api", "breaking change") {
		areas = append(areas, "contratos")
	}
	if containsAnyPattern(combined, "goroutine", "mutex", "channel", "sync.", "concurren", "race", "deadlock", "lock") {
		areas = append(areas, "concorrencia")
	}
	if containsAnyPattern(combined, "migra", "schema", "database", "sql", "query") {
		areas = append(areas, "persistencia")
	}

	if len(areas) == 0 {
		areas = append(areas, "contratos", "seguranca")
	}
	return strings.Join(areas, ", ")
}

// BugfixTemplateData agrupa os placeholders do template de bugfix.
type BugfixTemplateData struct {
	TaskFile       string
	PRDFolder      string
	TechSpec       string
	TasksFile      string
	ReviewFindings string // saida bruta do reviewer (achados criticos)
	Diff           string // diff original que disparou os achados
}

// ErrBugfixTemplateInvalido indica erro no parsing ou execucao do template de bugfix.
var ErrBugfixTemplateInvalido = errors.New("template de bugfix invalido")

// BuildBugfixPrompt constroi o prompt de bugfix a partir do template embutido.
// Retorna erro wrappado com ErrBugfixTemplateInvalido em caso de falha de parsing ou execucao.
func BuildBugfixPrompt(data BugfixTemplateData) (string, error) {
	tmpl, err := template.New("bugfix").Parse(defaultBugfixTemplate)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrBugfixTemplateInvalido, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: %v", ErrBugfixTemplateInvalido, err)
	}

	return buf.String(), nil
}

// formatCompletedTasks formata a lista de tasks ja executadas no bundle
// a partir das iteracoes anteriores do report.
func formatCompletedTasks(iterations []IterationResult, currentTaskID string) string {
	var completed []string
	seen := make(map[string]bool)

	for _, iter := range iterations {
		if iter.PostStatus == "done" && !seen[iter.TaskID] {
			seen[iter.TaskID] = true
			completed = append(completed, fmt.Sprintf("%s (%s)", iter.TaskID, iter.Title))
		}
	}

	if !seen[currentTaskID] {
		completed = append(completed, currentTaskID+" (atual)")
	}

	if len(completed) == 0 {
		return "(nenhuma task concluida anteriormente)"
	}
	return strings.Join(completed, ", ")
}
