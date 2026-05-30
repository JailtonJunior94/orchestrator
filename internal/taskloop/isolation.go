package taskloop

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type taskIsolationSnapshot struct {
	tasksContent []byte
	taskRows     map[string]string
	taskFiles    map[string][]byte
	prdFiles     map[string][]byte
}

type taskIsolationMode int

const (
	_taskIsolationModeExecutor taskIsolationMode = iota
	_taskIsolationModeReviewer
)

var _trackedTaskFileNameRe = regexp.MustCompile(`^(?i)(\d+(?:\.\d+)?[._-].+|task-\d+\.\d+[._-].+|task-\d+[._-].+|task-\d+_[^.]+|task-\d+-[^.]+|task-\d+\.\d+_[^.]+|task-\d+\.\d+-[^.]+|TASK-\d+[._-].+)\.md$`)

func (c *Catalog) captureTaskIsolationSnapshot(prdFolder string, fsys fs.FileSystem) (*taskIsolationSnapshot, error) {
	return NewCatalog().captureTaskIsolationSnapshotWithMode(prdFolder, _taskIsolationModeExecutor, fsys)
}

func (c *Catalog) captureTaskIsolationSnapshotWithMode(prdFolder string, mode taskIsolationMode, fsys fs.FileSystem) (*taskIsolationSnapshot, error) {
	tasksContent, err := fsys.ReadFile(filepath.Join(prdFolder, "tasks.md"))
	if err != nil {
		return nil, fmt.Errorf("ler tasks.md para snapshot: %w", err)
	}

	taskRows, err := NewCatalog().extractTaskRows(tasksContent)
	if err != nil {
		return nil, fmt.Errorf("snapshot das rows de tasks.md: %w", err)
	}

	taskFiles, err := NewCatalog().readTaskFiles(prdFolder, fsys)
	if err != nil {
		return nil, fmt.Errorf("snapshot dos arquivos de task: %w", err)
	}

	prdFiles, err := NewCatalog().readProtectedPRDFiles(prdFolder, mode, fsys)
	if err != nil {
		return nil, fmt.Errorf("snapshot dos arquivos protegidos do PRD: %w", err)
	}

	return &taskIsolationSnapshot{
		tasksContent: append([]byte(nil), tasksContent...),
		taskRows:     taskRows,
		taskFiles:    taskFiles,
		prdFiles:     prdFiles,
	}, nil
}

func (c *Catalog) validateTaskIsolation(snapshot *taskIsolationSnapshot, prdFolder, currentTaskID, currentTaskFile string, fsys fs.FileSystem) error {
	return NewCatalog().validateTaskIsolationWithMode(snapshot, prdFolder, currentTaskID, currentTaskFile, _taskIsolationModeExecutor, fsys)
}

func (c *Catalog) validateReviewerIsolation(snapshot *taskIsolationSnapshot, prdFolder, currentTaskID, currentTaskFile string, fsys fs.FileSystem) error {
	return NewCatalog().validateTaskIsolationWithMode(snapshot, prdFolder, currentTaskID, currentTaskFile, _taskIsolationModeReviewer, fsys)
}

func (c *Catalog) validateTaskIsolationWithMode(snapshot *taskIsolationSnapshot, prdFolder, currentTaskID, currentTaskFile string, mode taskIsolationMode, fsys fs.FileSystem) error {
	currentTasksContent, err := fsys.ReadFile(filepath.Join(prdFolder, "tasks.md"))
	if err != nil {
		return fmt.Errorf("nao foi possivel reler tasks.md apos execucao: %w", err)
	}

	currentRows, err := NewCatalog().extractTaskRows(currentTasksContent)
	if err != nil {
		return fmt.Errorf("tasks.md ficou invalido apos execucao: %w", err)
	}

	if _, ok := currentRows[currentTaskID]; !ok {
		return fmt.Errorf("row da task atual %s foi removida de tasks.md", currentTaskID)
	}

	if err := NewCatalog().validateTaskRowIsolation(snapshot.taskRows, currentRows, currentTaskID, mode == _taskIsolationModeExecutor); err != nil {
		return err
	}

	currentTaskFiles, err := NewCatalog().readTaskFiles(prdFolder, fsys)
	if err != nil {
		return fmt.Errorf("nao foi possivel reler arquivos de task apos execucao: %w", err)
	}

	if err := NewCatalog().validateTaskFileIsolation(snapshot.taskFiles, currentTaskFiles, currentTaskFile, mode == _taskIsolationModeExecutor); err != nil {
		return err
	}

	currentPRDFiles, err := NewCatalog().readProtectedPRDFiles(prdFolder, mode, fsys)
	if err != nil {
		return fmt.Errorf("nao foi possivel reler arquivos protegidos do PRD apos execucao: %w", err)
	}

	if err := NewCatalog().validateProtectedPRDFileIsolation(snapshot.prdFiles, currentPRDFiles); err != nil {
		return err
	}

	return nil
}

func (c *Catalog) restoreTaskIsolationSnapshotAt(snapshot *taskIsolationSnapshot, prdFolder string, fsys fs.FileSystem) error {
	currentTaskFiles, err := NewCatalog().readTaskFiles(prdFolder, fsys)
	if err != nil {
		return fmt.Errorf("listar arquivos atuais de task para restauracao: %w", err)
	}
	for path := range currentTaskFiles {
		if _, ok := snapshot.taskFiles[path]; ok {
			continue
		}
		if err := fsys.Remove(path); err != nil {
			return fmt.Errorf("remover arquivo de task adicionado indevidamente %s: %w", path, err)
		}
	}

	currentPRDFiles, err := NewCatalog().readProtectedPRDFiles(prdFolder, _taskIsolationModeReviewer, fsys)
	if err != nil {
		return fmt.Errorf("listar arquivos protegidos atuais do PRD para restauracao: %w", err)
	}
	for path := range currentPRDFiles {
		if _, ok := snapshot.prdFiles[path]; ok {
			continue
		}
		if err := fsys.Remove(path); err != nil {
			return fmt.Errorf("remover arquivo protegido do PRD adicionado indevidamente %s: %w", path, err)
		}
	}

	if err := fsys.WriteFile(filepath.Join(prdFolder, "tasks.md"), snapshot.tasksContent); err != nil {
		return fmt.Errorf("restaurar tasks.md: %w", err)
	}

	for path, content := range snapshot.taskFiles {
		if err := fsys.WriteFile(path, content); err != nil {
			return fmt.Errorf("restaurar arquivo de task %s: %w", path, err)
		}
	}

	for path, content := range snapshot.prdFiles {
		if err := fsys.WriteFile(path, content); err != nil {
			return fmt.Errorf("restaurar arquivo protegido do PRD %s: %w", path, err)
		}
	}

	return nil
}

func (c *Catalog) validateTaskRowIsolation(before, after map[string]string, currentTaskID string, allowCurrentTaskMutation bool) error {
	for id, rowBefore := range before {
		rowAfter, ok := after[id]
		if !ok {
			return fmt.Errorf("row da task %s foi removida de tasks.md", id)
		}
		if allowCurrentTaskMutation && id == currentTaskID {
			continue
		}
		if rowAfter != rowBefore {
			return fmt.Errorf("row da task %s foi alterada indevidamente em tasks.md", id)
		}
	}

	for id := range after {
		if _, ok := before[id]; !ok {
			return fmt.Errorf("nova row de task %s foi adicionada indevidamente em tasks.md", id)
		}
	}

	return nil
}

func (c *Catalog) validateTaskFileIsolation(before, after map[string][]byte, currentTaskFile string, allowCurrentTaskMutation bool) error {
	for path, contentBefore := range before {
		contentAfter, ok := after[path]
		if !ok {
			return fmt.Errorf("arquivo de task %s foi removido", filepath.Base(path))
		}
		if allowCurrentTaskMutation && path == currentTaskFile {
			continue
		}
		if !bytes.Equal(contentAfter, contentBefore) {
			return fmt.Errorf("arquivo de task %s foi alterado indevidamente", filepath.Base(path))
		}
	}

	for path := range after {
		if allowCurrentTaskMutation && path == currentTaskFile {
			continue
		}
		if _, ok := before[path]; !ok {
			return fmt.Errorf("novo arquivo de task %s foi adicionado indevidamente", filepath.Base(path))
		}
	}

	return nil
}

func (c *Catalog) validateProtectedPRDFileIsolation(before, after map[string][]byte) error {
	for path, contentBefore := range before {
		contentAfter, ok := after[path]
		if !ok {
			return fmt.Errorf("arquivo protegido do PRD %s foi removido", filepath.Base(path))
		}
		if !bytes.Equal(contentAfter, contentBefore) {
			return fmt.Errorf("arquivo protegido do PRD %s foi alterado indevidamente", filepath.Base(path))
		}
	}

	for path := range after {
		if _, ok := before[path]; !ok {
			return fmt.Errorf("novo arquivo protegido do PRD %s foi adicionado indevidamente", filepath.Base(path))
		}
	}

	return nil
}

func (c *Catalog) readTaskFiles(prdFolder string, fsys fs.FileSystem) (map[string][]byte, error) {
	entries, err := fsys.ReadDir(prdFolder)
	if err != nil {
		return nil, err
	}

	files := make(map[string][]byte)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !NewCatalog().isTrackedTaskFile(name) {
			continue
		}

		fullPath := filepath.Join(prdFolder, name)
		content, err := fsys.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		files[fullPath] = append([]byte(nil), content...)
	}

	return files, nil
}

func (c *Catalog) readProtectedPRDFiles(prdFolder string, mode taskIsolationMode, fsys fs.FileSystem) (map[string][]byte, error) {
	files := make(map[string][]byte)
	if err := NewCatalog().walkFiles(prdFolder, fsys, func(path string) error {
		if !NewCatalog().isProtectedPRDFile(prdFolder, path, mode) {
			return nil
		}
		content, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}
		files[path] = append([]byte(nil), content...)
		return nil
	}); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *Catalog) walkFiles(root string, fsys fs.FileSystem, visit func(path string) error) error {
	entries, err := fsys.ReadDir(root)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			if err := NewCatalog().walkFiles(fullPath, fsys, visit); err != nil {
				return err
			}
			continue
		}
		if err := visit(fullPath); err != nil {
			return err
		}
	}

	return nil
}

func (c *Catalog) isProtectedPRDFile(prdFolder, fullPath string, mode taskIsolationMode) bool {
	relPath, err := filepath.Rel(prdFolder, fullPath)
	if err != nil {
		return false
	}
	name := filepath.Base(fullPath)

	if relPath == "tasks.md" || NewCatalog().isTrackedTaskFile(name) {
		return false
	}
	// Subdiretórios gerenciados pelo harness (não pelo agente) não são arquivos protegidos do PRD.
	// Sem isso, artefatos gravados pela própria stack disparam falso-positivo de isolamento e
	// abortam a task-loop ACP:
	//   - memory/      → hook memory_persist (F3) grava MEMORY.md.
	//   - .checkpoints/ → execute-task (F25) grava <num>.yaml antes de mutar tasks.md.
	//   - .partials/    → execute-task (F32) grava tasks.md.<num>.partial em wave paralela.
	if NewCatalog().isHarnessManagedPRDDir(relPath) {
		return false
	}
	if mode == _taskIsolationModeExecutor && NewCatalog().isAllowedExecutorArtifact(name) {
		return false
	}
	return true
}

// isHarnessManagedPRDDir indica se relPath está sob um subdiretório do PRD gerenciado pela própria
// stack (não pelo agente): memory/, .checkpoints/, .partials/. Esses artefatos são legítimos e não
// devem disparar violação de isolamento. Literais espelham memory.DirName e os paths de execute-task.
func (c *Catalog) isHarnessManagedPRDDir(relPath string) bool {
	for _, dir := range []string{"memory", ".checkpoints", ".partials"} {
		if relPath == dir || strings.HasPrefix(relPath, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (c *Catalog) isAllowedExecutorArtifact(name string) bool {
	return strings.Contains(name, "execution_report") ||
		strings.Contains(name, "bugfix_report") ||
		name == "report.md"
}

func (c *Catalog) isTrackedTaskFile(name string) bool {
	if !strings.HasSuffix(name, ".md") {
		return false
	}
	if name == "tasks.md" || name == "prd.md" || name == "techspec.md" {
		return false
	}
	if strings.Contains(name, "execution_report") || strings.Contains(name, "bugfix_report") || name == "report.md" {
		return false
	}
	return _trackedTaskFileNameRe.MatchString(name)
}

func (c *Catalog) extractTaskRows(content []byte) (map[string]string, error) {
	lines := strings.Split(string(content), "\n")
	rows := make(map[string]string)
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !_tableRowRe.MatchString(trimmed) {
			continue
		}
		found = true

		cols := strings.Split(trimmed, "|")
		if len(cols) < 5 {
			// tabelas auxiliares (ex: cobertura de requisitos) tem menos colunas — ignorar
			continue
		}

		id := strings.TrimSpace(cols[1])
		if id == "" {
			return nil, fmt.Errorf("row sem task id: %q", trimmed)
		}
		rows[id] = trimmed
	}

	if !found {
		return nil, fmt.Errorf("nenhuma task encontrada")
	}

	return rows, nil
}
