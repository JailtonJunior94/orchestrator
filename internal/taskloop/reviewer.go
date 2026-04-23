package taskloop

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

const (
	gitDiffUnavailable     = "(diff indisponivel)"
	gitDiffUnavailableSafe = "(diff indisponivel: nenhuma alteracao segura da iteracao atual foi identificada)"
)

type gitPathSnapshot struct {
	Status       string
	UnstagedDiff string
	StagedDiff   string
	ContentHash  string
}

// captureGitStatusSnapshot captura o conjunto atual de paths alterados no workspace.
// O snapshot inclui a assinatura observavel de cada path alterado para detectar
// mudancas novas dentro de arquivos ja sujos antes da iteracao atual.
func captureGitStatusSnapshot(ctx context.Context, workDir string) (map[string]gitPathSnapshot, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain", "--untracked-files=all")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	snapshot := make(map[string]gitPathSnapshot)
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		if strings.TrimSpace(line) == "" || len(line) < 4 {
			continue
		}
		status := line[:2]
		path := strings.TrimSpace(line[3:])
		if renameParts := strings.Split(path, " -> "); len(renameParts) == 2 {
			path = strings.TrimSpace(renameParts[1])
		}
		if path == "" {
			continue
		}
		normalizedPath := filepath.ToSlash(filepath.Clean(path))
		snapshot[normalizedPath] = gitPathSnapshot{
			Status:       status,
			UnstagedDiff: gitDiffOutput(ctx, workDir, "diff", "--no-ext-diff", "--", normalizedPath),
			StagedDiff:   gitDiffOutput(ctx, workDir, "diff", "--no-ext-diff", "--cached", "--", normalizedPath),
			ContentHash:  fileContentHash(filepath.Join(workDir, filepath.FromSlash(normalizedPath))),
		}
	}
	return snapshot, nil
}

// changedGitPathsSince retorna apenas os paths cuja assinatura mudou durante a
// iteracao atual. Isso inclui arquivos ja sujos antes da execucao que receberam
// alteracoes adicionais, sem reencaminhar paths que permaneceram intactos.
func changedGitPathsSince(before, after map[string]gitPathSnapshot) []string {
	if len(after) == 0 {
		return nil
	}

	paths := make([]string, 0, len(after))
	for path, afterSnapshot := range after {
		if beforeSnapshot, existed := before[path]; existed && beforeSnapshot == afterSnapshot {
			continue
		}
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}

// captureGitDiff retorna um diff limitado aos paths seguros da iteracao atual.
// Se o escopo seguro nao puder ser garantido, o diff e omitido para evitar
// vazar alteracoes nao relacionadas do workspace ao reviewer.
func captureGitDiff(ctx context.Context, workDir string, paths []string) string {
	if len(paths) == 0 {
		return gitDiffUnavailableSafe
	}

	normalized := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		cleaned := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
		if cleaned == "" || cleaned == "." {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}
	if len(normalized) == 0 {
		return gitDiffUnavailableSafe
	}

	statuses, err := captureGitStatusSnapshot(ctx, workDir)
	if err != nil {
		return gitDiffUnavailable
	}

	unstaged := gitDiffOutput(ctx, workDir, append([]string{"diff", "--no-ext-diff", "--"}, normalized...)...)
	staged := gitDiffOutput(ctx, workDir, append([]string{"diff", "--no-ext-diff", "--cached", "--"}, normalized...)...)

	var untracked []string
	for _, path := range normalized {
		if statuses[path].Status == "??" {
			untracked = append(untracked, path)
		}
	}

	var sections []string
	if unstaged != "" {
		sections = append(sections, unstaged)
	}
	if staged != "" {
		sections = append(sections, staged)
	}
	if len(untracked) > 0 {
		sections = append(sections,
			"novos arquivos no escopo seguro da iteracao atual:\n"+strings.Join(untracked, "\n"))
	}
	if len(sections) == 0 {
		return gitDiffUnavailableSafe
	}

	return strings.Join(sections, "\n")
}

func gitDiffOutput(ctx context.Context, workDir string, args ...string) string {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func fileContentHash(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum)
}
