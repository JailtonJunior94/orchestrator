package taskloop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

// ErrWriterLocked indica que outro processo ja possui o escritor do PRD.
var ErrWriterLocked = errors.New("taskloop: escritor do PRD ja esta em uso")

// ErrParallelExecutionUnsafe indica uma tentativa de escrita paralela sem as
// garantias minimas de isolamento e ownership disjunto.
var ErrParallelExecutionUnsafe = errors.New("taskloop: execucao paralela insegura")

// RuntimeCapabilities descreve apenas capacidades detectadas localmente. Ela
// nao executa nem consulta runtimes externos.
type RuntimeCapabilities struct {
	SupportsWrite     bool `json:"supports_write"`
	SupportsWorktree  bool `json:"supports_worktree"`
	IsolatedWorktrees bool `json:"isolated_worktrees"`
}

// DetectRuntimeCapabilities consulta somente capacidades locais do Git e do
// filesystem. A deteccao nao cria worktrees: criar um worktree e uma acao de
// execucao, nao uma sondagem. Como este orquestrador ainda nao recebe um
// isolador explicitamente injetado, IsolatedWorktrees permanece falso; assim
// qualquer plano concorrente falha fechado em vez de escrever no worktree
// compartilhado.
func DetectRuntimeCapabilities(ctx context.Context, workDir string) (RuntimeCapabilities, error) {
	absolute, err := filepath.Abs(workDir)
	if err != nil {
		return RuntimeCapabilities{}, fmt.Errorf("taskloop: resolver worktree para detectar capacidades: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return RuntimeCapabilities{}, fmt.Errorf("taskloop: inspecionar worktree para detectar capacidades: %w", err)
	}
	if !info.IsDir() {
		return RuntimeCapabilities{}, fmt.Errorf("taskloop: worktree nao e diretorio: %s", absolute)
	}

	capabilities := RuntimeCapabilities{SupportsWrite: info.Mode().Perm()&0o222 != 0}
	command := exec.CommandContext(ctx, "git", "worktree", "list", "--porcelain")
	command.Dir = absolute
	if err := command.Run(); err == nil {
		capabilities.SupportsWorktree = true
	}
	return capabilities, nil
}

// TaskOwnership declara os paths que uma task pode alterar. Paths vazios
// significam ownership desconhecido e impedem paralelismo.
type TaskOwnership struct {
	TaskID     string
	OwnedPaths []string
}

// ExecutionPlan e a decisao deterministica de concorrencia antes de qualquer
// chamada a um executor.
type ExecutionPlan struct {
	Concurrent int
	Tasks      []TaskOwnership
	Runtime    RuntimeCapabilities
}

// Validate verifica que escrita concorrente possui worktrees isolados e que
// nenhuma task declara o mesmo path. Sem essas provas, o chamador deve usar o
// caminho sequencial.
func (p ExecutionPlan) Validate() error {
	if p.Concurrent <= 1 || len(p.Tasks) <= 1 {
		return nil
	}
	if !p.Runtime.SupportsWrite || !p.Runtime.SupportsWorktree || !p.Runtime.IsolatedWorktrees {
		return fmt.Errorf("%w: runtime nao oferece worktree isolado para escrita", ErrParallelExecutionUnsafe)
	}
	seen := make(map[string]string)
	for _, task := range p.Tasks {
		if strings.TrimSpace(task.TaskID) == "" || len(task.OwnedPaths) == 0 {
			return fmt.Errorf("%w: ownership desconhecido para task %q", ErrParallelExecutionUnsafe, task.TaskID)
		}
		for _, owned := range task.OwnedPaths {
			clean := filepath.Clean(owned)
			if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
				return fmt.Errorf("%w: path de ownership invalido %q", ErrParallelExecutionUnsafe, owned)
			}
			for path, owner := range seen {
				if owner != task.TaskID && pathsOverlap(path, clean) {
					return fmt.Errorf("%w: ownership sobreposto em %s (%s e %s)", ErrParallelExecutionUnsafe, clean, owner, task.TaskID)
				}
			}
			seen[clean] = task.TaskID
		}
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	separator := string(filepath.Separator)
	return left == right || strings.HasPrefix(left, right+separator) || strings.HasPrefix(right, left+separator)
}

// Snapshot vincula uma tentativa ao estado completo observavel do repositório.
type Snapshot struct {
	BaseSHA          string
	PatchSHA256      string
	FinalStateSHA256 string
	Dirty            bool
}

// NewSnapshot calcula digests SHA-256 a partir de entradas já capturadas. O
// patch deve incluir staged, unstaged e arquivos nao rastreados.
func NewSnapshot(baseSHA, patch, finalState string) Snapshot {
	patchDigest := sha256.Sum256([]byte(patch))
	stateDigest := sha256.Sum256([]byte(finalState))
	return Snapshot{
		BaseSHA:          baseSHA,
		PatchSHA256:      hex.EncodeToString(patchDigest[:]),
		FinalStateSHA256: hex.EncodeToString(stateDigest[:]),
		Dirty:            strings.TrimSpace(patch) != "",
	}
}

// Orchestrator persiste transicoes de tentativa no estado SDD. Nenhum metodo
// invoca runtime: a execucao efetiva continua uma dependencia explicita do
// task-loop.
type Orchestrator struct {
	store *sdd.Store
}

func NewOrchestrator(store *sdd.Store) *Orchestrator {
	if store == nil {
		store = sdd.NewStore()
	}
	return &Orchestrator{store: store}
}

// AcquireWriter obtem o lock global do PRD. O release deve sempre ser chamado.
func (o *Orchestrator) AcquireWriter(prdDir string) (func() error, error) {
	absolute, err := filepath.Abs(prdDir)
	if err != nil {
		return nil, fmt.Errorf("taskloop: resolver diretorio do PRD: %w", err)
	}
	return acquireOrchestratorLock(filepath.Join(absolute, ".sdd-orchestrate.lock"))
}

// Start registra uma tentativa nova ou sua retomada. Eventos anteriores nunca
// sao alterados, por isso uma queda entre transicoes permanece auditavel.
func (o *Orchestrator) Start(prdDir, runID, taskID string, attempt int, snapshot Snapshot) (state sdd.State, err error) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(taskID) == "" || attempt < 1 {
		return sdd.State{}, fmt.Errorf("taskloop: run_id, task_id e attempt valido sao obrigatorios")
	}
	release, err := o.AcquireWriter(prdDir)
	if err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: adquirir escritor para iniciar tentativa: %w", err)
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil && err == nil {
			err = fmt.Errorf("taskloop: liberar escritor apos iniciar tentativa: %w", releaseErr)
		}
	}()

	return o.startLocked(prdDir, runID, taskID, attempt, snapshot)
}

func (o *Orchestrator) startLocked(prdDir, runID, taskID string, attempt int, snapshot Snapshot) (sdd.State, error) {
	state, err := o.store.Load(prdDir)
	if err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: carregar estado para iniciar tentativa: %w", err)
	}
	if state.RunID != runID {
		return sdd.State{}, fmt.Errorf("taskloop: run_id %q diverge do estado %q", runID, state.RunID)
	}
	if o.hasTerminalEvent(state, taskID, attempt) {
		return state, nil
	}
	if o.hasStartedEvent(state, taskID, attempt) {
		// A mesma identidade representa a mesma tentativa. Repetir Start após uma
		// queda devolve o estado pendente sem acrescentar um segundo evento.
		return state, nil
	}
	state.Events = append(state.Events, sdd.Event{At: time.Now().UTC(), RunID: runID, TaskID: taskID, Attempt: attempt, Action: "attempt_started",
		Detail: fmt.Sprintf("base_sha=%s patch_sha256=%s final_state_sha256=%s dirty=%t", snapshot.BaseSHA, snapshot.PatchSHA256, snapshot.FinalStateSHA256, snapshot.Dirty)})
	if err := o.store.Save(prdDir, state); err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: persistir inicio da tentativa: %w", err)
	}
	return state, nil
}

// Finish somente aceita um resultado que satisfaz o contrato SDD, impedindo
// que uma tentativa seja marcada done sem provas e revisao aprovada.
func (o *Orchestrator) Finish(prdDir string, result sdd.ExecutionResult) (state sdd.State, err error) {
	if err := o.store.ValidateExecutionResult(result); err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: validar resultado da tentativa: %w", err)
	}
	release, err := o.AcquireWriter(prdDir)
	if err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: adquirir escritor para finalizar tentativa: %w", err)
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil && err == nil {
			err = fmt.Errorf("taskloop: liberar escritor apos finalizar tentativa: %w", releaseErr)
		}
	}()

	return o.finishLocked(prdDir, result)
}

func (o *Orchestrator) finishLocked(prdDir string, result sdd.ExecutionResult) (sdd.State, error) {
	state, err := o.store.Load(prdDir)
	if err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: carregar estado para finalizar tentativa: %w", err)
	}
	if state.RunID != result.RunID {
		return sdd.State{}, fmt.Errorf("taskloop: run_id do resultado diverge do estado")
	}
	if o.hasTerminalEvent(state, result.TaskID, result.Attempt) {
		return state, nil
	}
	state.Events = append(state.Events, sdd.Event{At: time.Now().UTC(), RunID: result.RunID, TaskID: result.TaskID, Attempt: result.Attempt,
		Action: "attempt_" + string(result.Status), Detail: "resultado validado"})
	if err := o.store.Save(prdDir, state); err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: persistir fim da tentativa: %w", err)
	}
	return state, nil
}

func (o *Orchestrator) hasStartedEvent(state sdd.State, taskID string, attempt int) bool {
	for _, event := range state.Events {
		if event.TaskID == taskID && event.Attempt == attempt && (event.Action == "attempt_started" || event.Action == "attempt_resumed") {
			return true
		}
	}
	return false
}

func (o *Orchestrator) hasTerminalEvent(state sdd.State, taskID string, attempt int) bool {
	for _, event := range state.Events {
		if event.TaskID == taskID && event.Attempt == attempt && (event.Action == "attempt_done" || event.Action == "attempt_failed" || event.Action == "attempt_blocked" || event.Action == "attempt_needs_input") {
			return true
		}
	}
	return false
}

// CaptureSnapshotFromGit captura uma prova cumulativa local e verificavel. O
// patch inclui staged, unstaged e arquivos nao rastreados; combinado com a
// base, ele representa o estado final sem depender de existir um commit novo.
func (o *Orchestrator) CaptureSnapshotFromGit(ctx context.Context, workDir string) (Snapshot, error) {
	if !NewCatalog().isGitWorkTree(ctx, workDir) {
		return Snapshot{}, fmt.Errorf("taskloop: capturar snapshot requer worktree git valido")
	}
	base, err := NewCatalog().commandOutput(ctx, workDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return Snapshot{}, fmt.Errorf("taskloop: capturar SHA base: %w", err)
	}
	baseSHA := strings.TrimSpace(string(base))
	if baseSHA == "" {
		return Snapshot{}, fmt.Errorf("taskloop: SHA base ausente")
	}
	patch := NewCatalog().captureGitDiff(ctx, workDir)
	if patch == "(diff indisponivel)" {
		return Snapshot{}, fmt.Errorf("taskloop: capturar patch cumulativo")
	}
	return NewSnapshot(baseSHA, patch, baseSHA+"\n"+patch), nil
}

// SnapshotFromGit preserva a API sem retorno de erro para consumidores legados.
// Novos fluxos devem usar CaptureSnapshotFromGit para manter o comportamento
// fail-closed quando a prova local nao puder ser produzida.
func (o *Orchestrator) SnapshotFromGit(ctx context.Context, workDir string) Snapshot {
	snapshot, err := o.CaptureSnapshotFromGit(ctx, workDir)
	if err != nil {
		return Snapshot{}
	}
	return snapshot
}
