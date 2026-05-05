package taskloop

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ErrNoEligibleTask indica que nenhuma task elegivel foi encontrada em tasks.md.
var ErrNoEligibleTask = errors.New("taskloop: nenhuma task elegivel")

// ErrDependencyCycle indica que existe um ciclo nas dependencias de tasks.
var ErrDependencyCycle = errors.New("taskloop: ciclo detectado nas dependencias")

// TaskSelector descobre e retorna a proxima task elegivel em tasks.md.
type TaskSelector interface {
	Next(ctx context.Context, prdFolder string) (*TaskEntry, error)
}

// defaultTaskSelector implementa TaskSelector com politica deterministica:
// retomadas em in_progress tem prioridade, seguidas por pending; dentro de cada
// grupo, a ordenacao por ID e ascendente.
type defaultTaskSelector struct {
	fsys fs.FileSystem
}

// NewTaskSelector cria um TaskSelector com o filesystem fornecido.
func NewTaskSelector(fsys fs.FileSystem) TaskSelector {
	return &defaultTaskSelector{fsys: fsys}
}

// Next le tasks.md do prdFolder e retorna a proxima task elegivel.
// Retorna ErrNoEligibleTask se nenhuma task atender os criterios.
// Retorna ErrDependencyCycle se houver ciclo nas dependencias.
func (s *defaultTaskSelector) Next(ctx context.Context, prdFolder string) (*TaskEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("taskloop: contexto cancelado antes de ler tasks.md: %w", err)
	}

	tasksPath := filepath.Join(prdFolder, "tasks.md")
	data, err := s.fsys.ReadFile(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("taskloop: erro ao ler %s: %w", tasksPath, err)
	}

	tasks, err := ParseTasksFile(data)
	if err != nil {
		return nil, fmt.Errorf("taskloop: erro ao parsear tasks.md: %w", err)
	}

	if err := detectCycle(tasks); err != nil {
		return nil, err
	}
	tasks = reconcileTaskStatuses(tasks, prdFolder, s.fsys)
	eligible := FindEligible(tasks, nil)

	if len(eligible) == 0 {
		return nil, ErrNoEligibleTask
	}

	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].Status != eligible[j].Status {
			return eligible[i].Status == "in_progress"
		}
		return eligible[i].ID < eligible[j].ID
	})

	result := eligible[0]
	return &result, nil
}

// detectCycle usa DFS para detectar ciclos no grafo de dependencias.
func detectCycle(tasks []TaskEntry) error {
	adj := make(map[string][]string, len(tasks))
	ids := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		adj[t.ID] = t.Dependencies
		ids[t.ID] = true
	}

	// visited: 0=nao visitado, 1=em progresso, 2=concluido
	state := make(map[string]int, len(tasks))

	var dfs func(id string) bool
	dfs = func(id string) bool {
		if state[id] == 1 {
			return true // ciclo
		}
		if state[id] == 2 {
			return false
		}
		state[id] = 1
		for _, dep := range adj[id] {
			if dfs(dep) {
				return true
			}
		}
		state[id] = 2
		return false
	}

	for id := range ids {
		if state[id] == 0 {
			if dfs(id) {
				return fmt.Errorf("%w", ErrDependencyCycle)
			}
		}
	}
	return nil
}
