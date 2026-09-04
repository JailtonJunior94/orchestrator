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
	"sort"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

// ErrWriterLocked indica que outro processo ja possui o escritor do PRD.
var ErrWriterLocked = errors.New("taskloop: escritor do PRD ja esta em uso")

// ErrParallelExecutionUnsafe indica uma tentativa de escrita paralela sem as
// garantias minimas de isolamento e ownership disjunto.
var ErrParallelExecutionUnsafe = errors.New("taskloop: execucao paralela insegura")

var ErrNoReadyTask = errors.New("taskloop: nenhuma task pronta para executar")

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

// StartNext seleciona e inicia atomicamente a próxima tentativa determinística.
func (o *Orchestrator) StartNext(prdDir, runID string, snapshot Snapshot) (state sdd.State, taskID string, attempt int, err error) {
	release, err := o.AcquireWriter(prdDir)
	if err != nil {
		return sdd.State{}, "", 0, fmt.Errorf("taskloop: adquirir escritor para orquestrar: %w", err)
	}
	defer func() {
		if releaseErr := release(); releaseErr != nil && err == nil {
			err = fmt.Errorf("taskloop: liberar escritor apos orquestrar: %w", releaseErr)
		}
	}()

	current, err := o.store.Load(prdDir)
	if err != nil {
		return sdd.State{}, "", 0, fmt.Errorf("taskloop: carregar estado para orquestrar: %w", err)
	}
	if current.RunID != runID {
		return sdd.State{}, "", 0, fmt.Errorf("taskloop: run_id %q diverge do estado %q", runID, current.RunID)
	}
	ids := make([]string, 0, len(current.Tasks))
	for id := range current.Tasks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		task := current.Tasks[id]
		if task.Status == sdd.StatusExecuting {
			persisted := Snapshot{BaseSHA: task.Snapshot.BaseSHA, PatchSHA256: task.Snapshot.PatchSHA256, FinalStateSHA256: task.Snapshot.FinalStateSHA256, Dirty: task.Snapshot.Dirty}
			state, err = o.startLocked(prdDir, runID, id, task.Attempt, persisted)
			return state, id, task.Attempt, err
		}
		if task.Status != sdd.StatusDraft || !o.dependenciesDone(current, task) {
			continue
		}
		state, err = o.startLocked(prdDir, runID, id, task.Attempt+1, snapshot)
		return state, id, task.Attempt + 1, err
	}
	return sdd.State{}, "", 0, ErrNoReadyTask
}

func (o *Orchestrator) dependenciesDone(state sdd.State, task sdd.TaskState) bool {
	for _, dependency := range task.Dependencies {
		if strings.Contains(dependency, ":") {
			return false
		}
		if state.Tasks[dependency].Status != sdd.StatusDone {
			return false
		}
	}
	return true
}

func (o *Orchestrator) startLocked(prdDir, runID, taskID string, attempt int, snapshot Snapshot) (sdd.State, error) {
	state, err := o.store.Load(prdDir)
	if err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: carregar estado para iniciar tentativa: %w", err)
	}
	if state.RunID != runID {
		return sdd.State{}, fmt.Errorf("taskloop: run_id %q diverge do estado %q", runID, state.RunID)
	}
	if !o.validSnapshot(snapshot) {
		return sdd.State{}, fmt.Errorf("taskloop: snapshot da tentativa invalido")
	}
	if o.hasTerminalEvent(state, taskID, attempt) {
		return state, nil
	}
	if o.hasStartedEvent(state, taskID, attempt) {
		// A mesma identidade representa a mesma tentativa. Repetir Start após uma
		// queda devolve o estado pendente sem acrescentar um segundo evento.
		task := state.Tasks[taskID]
		if task.Snapshot == nil || !o.snapshotMatches(snapshot, *task.Snapshot) {
			return sdd.State{}, fmt.Errorf("taskloop: snapshot diverge da tentativa persistida")
		}
		return state, nil
	}
	task, exists := state.Tasks[taskID]
	if len(state.Tasks) > 0 && !exists {
		return sdd.State{}, fmt.Errorf("taskloop: task %s nao existe no modelo operacional", taskID)
	}
	for _, dependency := range task.Dependencies {
		if strings.Contains(dependency, ":") {
			continue
		}
		if state.Tasks[dependency].Status != sdd.StatusDone {
			return sdd.State{}, fmt.Errorf("taskloop: task %s aguarda dependencia %s", taskID, dependency)
		}
	}
	task.ID = taskID
	task.Status = sdd.StatusExecuting
	task.Attempt = attempt
	task.Snapshot = &sdd.AttemptSnapshot{BaseSHA: snapshot.BaseSHA, PatchSHA256: snapshot.PatchSHA256, FinalStateSHA256: snapshot.FinalStateSHA256, Dirty: snapshot.Dirty}
	if state.Tasks == nil {
		state.Tasks = make(map[string]sdd.TaskState)
	}
	state.Tasks[taskID] = task
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
	task, exists := state.Tasks[result.TaskID]
	if !exists || task.Attempt != result.Attempt || task.Snapshot == nil || !o.hasStartedEvent(state, result.TaskID, result.Attempt) {
		return sdd.State{}, fmt.Errorf("taskloop: tentativa nao possui snapshot persistido")
	}
	if result.BaseSHA != task.Snapshot.BaseSHA {
		return sdd.State{}, fmt.Errorf("taskloop: base_sha diverge do snapshot inicial")
	}
	if result.Status == sdd.StatusDone {
		if err := o.ValidateExecutionEvidence(prdDir, result); err != nil {
			return sdd.State{}, err
		}
	}
	if o.hasTerminalEvent(state, result.TaskID, result.Attempt) {
		return state, nil
	}
	task.Status = result.Status
	task.EvidenceRefs = append([]string(nil), result.Evidence...)
	state.Tasks[result.TaskID] = task
	state.EvidenceRefs = append(state.EvidenceRefs, result.Evidence...)
	state.Events = append(state.Events, sdd.Event{At: time.Now().UTC(), RunID: result.RunID, TaskID: result.TaskID, Attempt: result.Attempt,
		Action: "attempt_" + string(result.Status), Detail: "resultado validado"})
	if err := o.store.Save(prdDir, state); err != nil {
		return sdd.State{}, fmt.Errorf("taskloop: persistir fim da tentativa: %w", err)
	}
	return state, nil
}

// ValidateExecutionEvidence recompõe o patch Git canônico e o estado final,
// além de confrontá-los com o artefato e as evidências físicas declaradas.
// Exclusões adicionais destinam-se somente ao envelope operacional que chama
// o validador, como o próprio JSON de resultado e o relatório Markdown.
func (o *Orchestrator) ValidateExecutionEvidence(prdDir string, result sdd.ExecutionResult, additionalExclusions ...string) error {
	if err := o.store.ValidateExecutionResult(result); err != nil {
		return fmt.Errorf("taskloop: validar contrato do resultado: %w", err)
	}
	finalSnapshot, patch, err := o.captureFinalSnapshot(prdDir, result, additionalExclusions...)
	if err != nil {
		return err
	}
	if result.BaseSHA != finalSnapshot.BaseSHA || result.PatchSHA256 != finalSnapshot.PatchSHA256 ||
		result.FinalStateSHA256 != finalSnapshot.FinalStateSHA256 {
		return fmt.Errorf("taskloop: resultado diverge do estado final recomputado")
	}
	if err := o.validatePatchArtifact(prdDir, result.PatchRef, patch, result.PatchSHA256); err != nil {
		return err
	}
	if err := o.validatePhysicalEvidence(prdDir, result); err != nil {
		return err
	}
	return nil
}

func (o *Orchestrator) captureFinalSnapshot(prdDir string, result sdd.ExecutionResult, additionalExclusions ...string) (Snapshot, []byte, error) {
	root, err := o.repositoryRoot(prdDir)
	if err != nil {
		return Snapshot{}, nil, err
	}
	excluded := make(map[string]bool, len(result.Evidence)+3)
	for _, reference := range result.Evidence {
		path := filepath.ToSlash(strings.Split(reference, "#")[0])
		if strings.HasPrefix(path, "evidence/") {
			excluded[path] = true
		}
	}
	patchReference := filepath.ToSlash(strings.Split(result.PatchRef, "#")[0])
	if !strings.HasPrefix(patchReference, "evidence/") {
		return Snapshot{}, nil, fmt.Errorf("taskloop: artefato do patch deve estar em evidence/: %q", result.PatchRef)
	}
	excluded[patchReference] = true
	for _, reference := range additionalExclusions {
		path := strings.Split(reference, "#")[0]
		if filepath.IsAbs(path) {
			relative, relativeErr := filepath.Rel(root, path)
			if relativeErr != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				return Snapshot{}, nil, fmt.Errorf("taskloop: exclusao operacional escapa do repositorio: %q", reference)
			}
			path = relative
		}
		path = filepath.ToSlash(path)
		if path != "" {
			excluded[path] = true
		}
	}
	absolutePRDDir, err := filepath.Abs(prdDir)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("taskloop: resolver diretorio absoluto do PRD: %w", err)
	}
	resolvedPRDDir, err := filepath.EvalSymlinks(absolutePRDDir)
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("taskloop: resolver diretorio do PRD: %w", err)
	}
	for _, name := range []string{"sdd-state.json", ".sdd-orchestrate.lock"} {
		path, relativeErr := filepath.Rel(root, filepath.Join(resolvedPRDDir, name))
		if relativeErr != nil {
			return Snapshot{}, nil, fmt.Errorf("taskloop: resolver artefato operacional: %w", relativeErr)
		}
		excluded[filepath.ToSlash(path)] = true
	}
	migrationBackupPath, relativeErr := filepath.Rel(root, filepath.Join(resolvedPRDDir, ".sdd-state.*.legacy.json"))
	if relativeErr != nil {
		return Snapshot{}, nil, fmt.Errorf("taskloop: resolver backup operacional: %w", relativeErr)
	}
	excluded[filepath.ToSlash(migrationBackupPath)] = true
	checkpointsPath, relativeErr := filepath.Rel(root, filepath.Join(resolvedPRDDir, ".checkpoints"))
	if relativeErr != nil {
		return Snapshot{}, nil, fmt.Errorf("taskloop: resolver checkpoints operacionais: %w", relativeErr)
	}
	excluded[filepath.ToSlash(checkpointsPath)+"/**"] = true
	base, err := o.gitOutput(prdDir, "rev-parse", "HEAD")
	if err != nil {
		return Snapshot{}, nil, fmt.Errorf("taskloop: capturar SHA base final: %w", err)
	}
	patch, err := o.semanticPatch(root, excluded)
	if err != nil {
		return Snapshot{}, nil, err
	}
	snapshot := NewSnapshot(strings.TrimSpace(string(base)), string(patch), strings.TrimSpace(string(base))+"\n"+string(patch))
	return snapshot, patch, nil
}

func (o *Orchestrator) semanticPatch(root string, excluded map[string]bool) ([]byte, error) {
	args := []string{"diff", "--binary", "HEAD", "--", "."}
	keys := make([]string, 0, len(excluded))
	for path := range excluded {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	for _, path := range keys {
		args = append(args, ":(exclude)"+path)
	}
	tracked, err := o.gitOutput(root, args...)
	if err != nil {
		return nil, fmt.Errorf("taskloop: capturar diff rastreado: %w", err)
	}
	untrackedOutput, err := o.gitOutput(root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("taskloop: listar arquivos nao rastreados: %w", err)
	}
	patch := append([]byte(nil), tracked...)
	untracked := strings.Split(strings.TrimSuffix(string(untrackedOutput), "\x00"), "\x00")
	sort.Strings(untracked)
	for _, path := range untracked {
		path = filepath.ToSlash(path)
		if path == "" || o.isExcluded(path, excluded) {
			continue
		}
		command := exec.Command("git", "diff", "--binary", "--no-index", "--", os.DevNull, path)
		command.Dir = root
		output, commandErr := command.CombinedOutput()
		if commandErr != nil {
			var exitErr *exec.ExitError
			if !errors.As(commandErr, &exitErr) || exitErr.ExitCode() != 1 {
				return nil, fmt.Errorf("taskloop: capturar arquivo novo %s: %w", path, commandErr)
			}
		}
		patch = append(patch, output...)
	}
	return patch, nil
}

func (o *Orchestrator) isExcluded(path string, excluded map[string]bool) bool {
	if excluded[path] {
		return true
	}
	for excludedPath := range excluded {
		if strings.HasSuffix(excludedPath, "/**") &&
			strings.HasPrefix(path, strings.TrimSuffix(excludedPath, "**")) {
			return true
		}
		if strings.HasSuffix(excludedPath, "*.legacy.json") &&
			strings.HasPrefix(path, strings.TrimSuffix(excludedPath, "*.legacy.json")) &&
			strings.HasSuffix(path, ".legacy.json") {
			return true
		}
	}
	return false
}

func (o *Orchestrator) gitOutput(workDir string, args ...string) ([]byte, error) {
	command := exec.Command("git", args...)
	command.Dir = workDir
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func (o *Orchestrator) validatePatchArtifact(prdDir, reference string, patch []byte, expectedDigest string) error {
	root, err := o.repositoryRoot(prdDir)
	if err != nil {
		return err
	}
	path, err := o.resolveEvidence(root, reference)
	if err != nil {
		return fmt.Errorf("taskloop: artefato do patch invalido: %w", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("taskloop: ler artefato do patch: %w", err)
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != expectedDigest || string(content) != string(patch) {
		return fmt.Errorf("taskloop: artefato do patch diverge do patch semantico")
	}
	return nil
}

func (o *Orchestrator) validatePhysicalEvidence(prdDir string, result sdd.ExecutionResult) error {
	root, err := o.repositoryRoot(prdDir)
	if err != nil {
		return err
	}
	digests := make(map[string]string, len(result.Evidence))
	for _, reference := range result.Evidence {
		path, resolveErr := o.resolveEvidence(root, reference)
		if resolveErr != nil {
			return fmt.Errorf("taskloop: validar evidencia fisica: %w", resolveErr)
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("taskloop: ler evidencia fisica: %w", readErr)
		}
		digest := sha256.Sum256(content)
		digests[filepath.ToSlash(strings.Split(reference, "#")[0])] = hex.EncodeToString(digest[:])
	}
	for _, criterion := range result.Criteria {
		reference := filepath.ToSlash(strings.Split(criterion.EvidenceRef, "#")[0])
		if _, exists := digests[reference]; !exists {
			return fmt.Errorf("taskloop: criterio %s nao referencia evidencia declarada", criterion.ID)
		}
	}
	for _, proof := range result.Tests {
		matched := false
		for _, digest := range digests {
			if proof.OutputSHA256 == digest {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("taskloop: digest do teste %q nao corresponde a evidencia fisica", proof.Command)
		}
	}
	return nil
}

func (o *Orchestrator) repositoryRoot(prdDir string) (string, error) {
	command := exec.Command("git", "rev-parse", "--show-toplevel")
	command.Dir = prdDir
	output, err := command.Output()
	if err == nil {
		return filepath.Abs(strings.TrimSpace(string(output)))
	}
	return filepath.Abs(prdDir)
}

func (o *Orchestrator) resolveEvidence(root, reference string) (string, error) {
	relative := strings.Split(reference, "#")[0]
	normalized := strings.ReplaceAll(relative, "\\", "/")
	if relative == "" || strings.HasPrefix(normalized, "/") ||
		(len(normalized) >= 2 && normalized[1] == ':' &&
			((normalized[0] >= 'A' && normalized[0] <= 'Z') ||
				(normalized[0] >= 'a' && normalized[0] <= 'z'))) {
		return "", fmt.Errorf("referencia de evidencia invalida %q", reference)
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", fmt.Errorf("referencia de evidencia invalida %q", reference)
		}
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolver raiz: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(resolvedRoot, relative))
	if err != nil {
		return "", fmt.Errorf("resolver %q: %w", reference, err)
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("evidencia escapa do repositorio: %q", reference)
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("evidencia nao e arquivo regular: %q", reference)
	}
	return resolved, nil
}

func (o *Orchestrator) validSnapshot(snapshot Snapshot) bool {
	if len(snapshot.BaseSHA) != 40 && len(snapshot.BaseSHA) != 64 {
		return false
	}
	if _, err := hex.DecodeString(snapshot.BaseSHA); err != nil {
		return false
	}
	for _, digest := range []string{snapshot.PatchSHA256, snapshot.FinalStateSHA256} {
		if len(digest) != 64 {
			return false
		}
		if _, err := hex.DecodeString(digest); err != nil {
			return false
		}
	}
	return true
}

func (o *Orchestrator) snapshotMatches(snapshot Snapshot, persisted sdd.AttemptSnapshot) bool {
	return snapshot.BaseSHA == persisted.BaseSHA && snapshot.PatchSHA256 == persisted.PatchSHA256 &&
		snapshot.FinalStateSHA256 == persisted.FinalStateSHA256 && snapshot.Dirty == persisted.Dirty
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
	root, err := o.repositoryRoot(workDir)
	if err != nil {
		return Snapshot{}, err
	}
	patch, err := o.semanticPatch(root, map[string]bool{})
	if err != nil {
		return Snapshot{}, err
	}
	return NewSnapshot(baseSHA, string(patch), baseSHA+"\n"+string(patch)), nil
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
