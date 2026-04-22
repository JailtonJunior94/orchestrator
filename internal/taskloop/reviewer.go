package taskloop

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os/exec"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ErrTemplateInvalido indica erro no parsing ou execucao do template de revisao.
var ErrTemplateInvalido = errors.New("template de revisao invalido")

// defaultReviewTemplate e o template embutido via go:embed.
//
//go:embed review_template.tmpl
var defaultReviewTemplate string

// ReviewTemplateData agrupa os placeholders do template de revisao.
type ReviewTemplateData struct {
	TaskFile  string
	PRDFolder string
	TechSpec  string
	TasksFile string
	Diff      string
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
