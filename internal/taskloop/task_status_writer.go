package taskloop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const (
	statusBlocked    = "blocked"
	statusNeedsInput = "needs_input"
)

func (s *Service) resolveCycleCriteria(taskFile string) ([]approval.AcceptanceCriterion, error) {
	content, err := s.fsys.ReadFile(taskFile)
	if err != nil {
		return nil, fmt.Errorf("mapa 1:1 nao confrontavel (RF-47): falha ao ler %s: %w", taskFile, err)
	}
	criteria, err := acceptanceCriteriaFromTaskFile(content)
	if err != nil {
		return nil, fmt.Errorf("mapa 1:1 nao confrontavel (RF-47): criterios invalidos em %s: %w", taskFile, err)
	}
	if len(criteria) == 0 {
		return nil, fmt.Errorf("mapa 1:1 nao confrontavel (RF-47/RF-51): %s nao declara criterios de aceite", taskFile)
	}
	return criteria, nil
}

func (c *Catalog) forceTaskStatus(prdFolder, taskFile, taskID, status string, fsys fs.FileSystem) error {
	if err := NewCatalog().writeTaskFileStatus(taskFile, status, fsys); err != nil {
		return err
	}
	return NewCatalog().writeTasksTableStatus(filepath.Join(prdFolder, "tasks.md"), taskID, status, fsys)
}

func (c *Catalog) writeTaskFileStatus(taskFile, status string, fsys fs.FileSystem) error {
	content, err := fsys.ReadFile(taskFile)
	if err != nil {
		return fmt.Errorf("erro ao forcar status %q em %s: %w", status, taskFile, err)
	}
	if !statusFieldRe.Match(content) {
		return fmt.Errorf("erro ao forcar status %q em %s: campo de status ausente", status, taskFile)
	}
	updated := statusFieldRe.ReplaceAll(content, []byte("**Status:** "+status))
	if err := fsys.WriteFileAtomic(taskFile, updated); err != nil {
		return fmt.Errorf("erro ao forcar status %q em %s: %w", status, taskFile, err)
	}
	return nil
}

func (c *Catalog) writeTasksTableStatus(tasksFile, taskID, status string, fsys fs.FileSystem) error {
	if strings.TrimSpace(taskID) == "" {
		return nil
	}
	lockErr := withTasksFileLock(fsys, tasksFile, func() error {
		return NewCatalog().replaceTaskStatusLine(fsys, tasksFile, taskID, status)
	})
	if lockErr != nil {
		return fmt.Errorf("erro ao forcar status %q da task %s em %s: %w", status, taskID, tasksFile, lockErr)
	}
	return nil
}

func (c *Catalog) replaceTaskStatusLine(fsys fs.FileSystem, tasksFile, taskID, status string) error {
	content, err := fsys.ReadFile(tasksFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	statusIdx, depsIdx := 3, 4
	for _, line := range lines {
		cols := strings.Split(strings.TrimSpace(line), "|")
		if NewCatalog().detectColumnIndices(cols, &statusIdx, &depsIdx) {
			break
		}
	}

	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !tableRowRe.MatchString(trimmed) {
			continue
		}
		cols := strings.Split(trimmed, "|")
		if len(cols) <= statusIdx || strings.TrimSpace(cols[1]) != taskID {
			continue
		}
		cols[statusIdx] = " " + status + " "
		lines[i] = strings.Join(cols, "|")
		changed = true
		break
	}
	if !changed {
		return fmt.Errorf("linha da task nao encontrada")
	}

	return fsys.WriteFileAtomic(tasksFile, []byte(strings.Join(lines, "\n")))
}

func (c *Catalog) reloadFinalTasks(prdFolder string, fallback []TaskEntry, fsys fs.FileSystem) []TaskEntry {
	content, err := fsys.ReadFile(filepath.Join(prdFolder, "tasks.md"))
	if err != nil {
		return fallback
	}
	tasks, err := NewCatalog().ParseTasksFile(content)
	if err != nil {
		return fallback
	}
	return tasks
}
