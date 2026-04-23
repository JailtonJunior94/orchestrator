package taskloop

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// TaskEntry representa uma linha da tabela de tasks em tasks.md.
type TaskEntry struct {
	ID           string
	Title        string
	Status       string
	Dependencies []string
}

var (
	tableRowRe    = regexp.MustCompile(`^\|\s*(\d+\.\d+)\s*\|`)
	statusFieldRe = regexp.MustCompile(`(?i)\*\*Status:\*\*\s*(.+)`)
)

// ParseTasksFile extrai entradas da tabela markdown em tasks.md.
func ParseTasksFile(content []byte) ([]TaskEntry, error) {
	lines := strings.Split(string(content), "\n")
	var entries []TaskEntry

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !tableRowRe.MatchString(line) {
			continue
		}

		cols := strings.Split(line, "|")
		// Esperado: vazio, ID, Title, Status, Deps, Paralelizavel, vazio
		if len(cols) < 5 {
			continue
		}

		id := strings.TrimSpace(cols[1])
		title := strings.TrimSpace(cols[2])
		status := normalizeStatus(strings.TrimSpace(cols[3]))
		deps := parseDependencies(strings.TrimSpace(cols[4]))

		entries = append(entries, TaskEntry{
			ID:           id,
			Title:        title,
			Status:       status,
			Dependencies: deps,
		})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("nenhuma task encontrada na tabela de tasks.md")
	}
	return entries, nil
}

// ReadTaskFileStatus extrai o campo **Status:** de um arquivo de task individual.
func ReadTaskFileStatus(content []byte) string {
	matches := statusFieldRe.FindSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	raw := strings.TrimSpace(string(matches[1]))
	// Extrair apenas a primeira palavra ou conteudo entre parenteses
	if idx := strings.Index(raw, "("); idx > 0 {
		inner := raw[idx+1:]
		if end := strings.Index(inner, ")"); end > 0 {
			return normalizeStatus(inner[:end])
		}
	}
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return ""
	}
	return normalizeStatus(fields[0])
}

// FindEligible retorna tasks elegiveis: status pending ou in_progress, todas deps done,
// nao no skipped set. Tasks em "in_progress" tem prioridade sobre "pending" para
// garantir retomada objetiva da sessao anterior antes de abrir uma nova task.
func FindEligible(tasks []TaskEntry, skipped map[string]bool) []TaskEntry {
	statusMap := make(map[string]string, len(tasks))
	for _, t := range tasks {
		statusMap[t.ID] = t.Status
	}

	var inProgress []TaskEntry
	var pending []TaskEntry
	for _, t := range tasks {
		if skipped[t.ID] || !isResumableStatus(t.Status) {
			continue
		}
		allDepsDone := true
		for _, dep := range t.Dependencies {
			if statusMap[dep] != "done" {
				allDepsDone = false
				break
			}
		}
		if allDepsDone {
			if t.Status == "in_progress" {
				inProgress = append(inProgress, t)
				continue
			}
			pending = append(pending, t)
		}
	}
	return append(inProgress, pending...)
}

func isResumableStatus(status string) bool {
	return status == "pending" || status == "in_progress"
}

func reconcileTaskStatuses(tasks []TaskEntry, prdFolder string, fsys fs.FileSystem) []TaskEntry {
	reconciled := append([]TaskEntry(nil), tasks...)
	for i := range reconciled {
		if !isResumableStatus(reconciled[i].Status) {
			continue
		}
		taskFile, err := ResolveTaskFile(prdFolder, reconciled[i], fsys)
		if err != nil {
			continue
		}
		if fileStatus := readTaskStatus(taskFile, fsys); fileStatus != "" {
			reconciled[i].Status = fileStatus
		}
	}
	return reconciled
}

// ResolveTaskFile encontra o arquivo de task individual pelo prefixo numerico do ID.
func ResolveTaskFile(prdFolder string, task TaskEntry, fsys fs.FileSystem) (string, error) {
	// ID eh algo como "1.0" — o prefixo numerico eh "1"
	prefix := strings.Split(task.ID, ".")[0]

	entries, err := fsys.ReadDir(prdFolder)
	if err != nil {
		return "", fmt.Errorf("erro ao ler diretorio %s: %w", prdFolder, err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		// Ignorar tasks.md, prd.md, techspec.md e relatorios
		if name == "tasks.md" || name == "prd.md" || name == "techspec.md" {
			continue
		}
		if strings.Contains(name, "execution_report") || strings.Contains(name, "bugfix_report") {
			continue
		}
		// Verificar convencoes de nome: "1-", "1.0-" ou "task-1.0-"
		if matchesTaskPrefix(name, prefix, task.ID) {
			return filepath.Join(prdFolder, name), nil
		}
	}

	return "", fmt.Errorf("arquivo de task nao encontrado para ID %s em %s", task.ID, prdFolder)
}

func matchesTaskPrefix(filename, prefix, fullID string) bool {
	// Separadores validos: _, -, .
	hasSep := func(s string) bool {
		return len(s) > 0 && (s[0] == '_' || s[0] == '-' || s[0] == '.')
	}

	// Convencao 1: "1-desc.md" ou "1_desc.md" (prefixo so com o primeiro segmento)
	if strings.HasPrefix(filename, prefix) && hasSep(filename[len(prefix):]) {
		return true
	}

	// Convencao 2: "1.0-desc.md" ou "1.0_desc.md" (ID completo)
	if strings.HasPrefix(filename, fullID) && hasSep(filename[len(fullID):]) {
		return true
	}

	// Convencao 3: "task-1.0-desc.md" ou "task-1.0_desc.md"
	taskFull := "task-" + fullID
	if strings.HasPrefix(filename, taskFull) && hasSep(filename[len(taskFull):]) {
		return true
	}

	// Convencao 4: "TASK-NNN-desc.md" (zero-padded, case-insensitive)
	upper := strings.ToUpper(filename)
	const taskDash = "TASK-"
	if strings.HasPrefix(upper, taskDash) {
		rest := filename[len(taskDash):]
		numEnd := 0
		for numEnd < len(rest) && rest[numEnd] >= '0' && rest[numEnd] <= '9' {
			numEnd++
		}
		if numEnd > 0 && hasSep(rest[numEnd:]) {
			numStr := strings.TrimLeft(rest[:numEnd], "0")
			if numStr == "" {
				numStr = "0"
			}
			if numStr == prefix {
				return true
			}
		}
	}

	return false
}

func parseDependencies(raw string) []string {
	if raw == "" || raw == "\u2014" || raw == "-" || strings.ToLower(raw) == "nenhuma" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var deps []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		deps = append(deps, p)
	}
	return deps
}

func normalizeStatus(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "concluido", "concluída", "concluído":
		return "done"
	case "em execucao", "em execução", "em andamento":
		return "in_progress"
	case "pendente":
		return "pending"
	case "bloqueado", "bloqueada":
		return "blocked"
	case "falhou", "falha":
		return "failed"
	case "aguardando input", "aguardando informacao":
		return "needs_input"
	}
	return s
}

// AllTerminal verifica se todas as tasks estao em estado terminal (done, failed, blocked).
func AllTerminal(tasks []TaskEntry) bool {
	for _, t := range tasks {
		switch t.Status {
		case "done", "failed", "blocked":
			continue
		default:
			return false
		}
	}
	return true
}

// readTaskStatus le o status de um arquivo de task via fs.FileSystem.
func readTaskStatus(path string, fsys fs.FileSystem) string {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return ""
	}
	return ReadTaskFileStatus(data)
}
