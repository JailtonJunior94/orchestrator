package taskloop

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

func TestExecutionPlanRejectsOverlappingOwnership(t *testing.T) {
	plan := ExecutionPlan{Concurrent: 2, Runtime: RuntimeCapabilities{SupportsWrite: true, SupportsWorktree: true, IsolatedWorktrees: true}, Tasks: []TaskOwnership{{TaskID: "1.0", OwnedPaths: []string{"internal/a.go"}}, {TaskID: "2.0", OwnedPaths: []string{"internal/a.go"}}}}
	if err := plan.Validate(); err == nil {
		t.Fatal("ownership sobreposto deveria bloquear paralelismo")
	}
}

func TestExecutionPlanRejectsNestedOverlappingOwnership(t *testing.T) {
	plan := ExecutionPlan{Concurrent: 2, Runtime: RuntimeCapabilities{SupportsWrite: true, SupportsWorktree: true, IsolatedWorktrees: true}, Tasks: []TaskOwnership{{TaskID: "1.0", OwnedPaths: []string{"internal/sdd"}}, {TaskID: "2.0", OwnedPaths: []string{"internal/sdd/state.go"}}}}
	if err := plan.Validate(); !errors.Is(err, ErrParallelExecutionUnsafe) {
		t.Fatalf("ownership ancestral sobreposto = %v, quero ErrParallelExecutionUnsafe", err)
	}
}

func TestExecutionPlanRejectsCapabilitiesWithoutIsolation(t *testing.T) {
	plan := ExecutionPlan{Concurrent: 2, Runtime: RuntimeCapabilities{SupportsWrite: true, SupportsWorktree: true}, Tasks: []TaskOwnership{{TaskID: "1.0", OwnedPaths: []string{"internal/a.go"}}, {TaskID: "2.0", OwnedPaths: []string{"internal/b.go"}}}}
	if err := plan.Validate(); !errors.Is(err, ErrParallelExecutionUnsafe) {
		t.Fatalf("capacidade sem isolamento = %v, quero ErrParallelExecutionUnsafe", err)
	}
}

func TestDetectRuntimeCapabilitiesNeverClaimsIsolationWithoutIsolator(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	capabilities, err := DetectRuntimeCapabilities(t.Context(), dir)
	if err != nil {
		t.Fatalf("DetectRuntimeCapabilities retornou erro: %v", err)
	}
	if !capabilities.SupportsWrite || !capabilities.SupportsWorktree || capabilities.IsolatedWorktrees {
		t.Fatalf("capabilities inesperadas: %#v", capabilities)
	}
}

func TestCaptureSnapshotFromGitIncludesCumulativePatch(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "teste@example.com")
	runGit(t, dir, "config", "user.name", "Teste")
	if err := os.WriteFile(filepath.Join(dir, "rastreado.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "rastreado.txt")
	runGit(t, dir, "-c", "commit.gpgSign=false", "commit", "-m", "base")
	if err := os.WriteFile(filepath.Join(dir, "rastreado.txt"), []byte("alterado\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "novo.txt"), []byte("novo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot, err := NewOrchestrator(sdd.NewStore()).CaptureSnapshotFromGit(t.Context(), dir)
	if err != nil {
		t.Fatalf("CaptureSnapshotFromGit retornou erro: %v", err)
	}
	if len(snapshot.BaseSHA) != 40 || len(snapshot.PatchSHA256) != 64 || len(snapshot.FinalStateSHA256) != 64 || !snapshot.Dirty {
		t.Fatalf("snapshot invalido: %#v", snapshot)
	}
}

func TestCaptureFinalSnapshotExcludesOperationalCheckpoints(t *testing.T) {
	dir := newOrchestratorStateDir(t)
	o := NewOrchestrator(sdd.NewStore())
	startSnapshot, err := o.CaptureSnapshotFromGit(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Start(dir, "run", "1.0", 1, startSnapshot); err != nil {
		t.Fatal(err)
	}
	checkpoints := filepath.Join(dir, ".checkpoints")
	if err := os.MkdirAll(checkpoints, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(checkpoints, "1.0.json"), []byte(`{"operacional":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sdd-state.run.legacy.json"), []byte(`{"operacional":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	result := newDoneResult(t, dir, o, startSnapshot)
	patch, err := os.ReadFile(filepath.Join(dir, result.PatchRef))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(patch), ".checkpoints/1.0.json") ||
		strings.Contains(string(patch), ".sdd-state.run.legacy.json") {
		t.Fatal("snapshot nao deve incluir artefato operacional")
	}
	if _, err := o.Finish(dir, result); err != nil {
		t.Fatalf("resultado com checkpoint operacional deveria finalizar: %v", err)
	}
}

func TestOrchestratorFinishesUsingRelativePRDDirectory(t *testing.T) {
	dir := newOrchestratorStateDir(t)
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cleanupErr := os.Chdir(original); cleanupErr != nil {
			t.Fatal(cleanupErr)
		}
	})

	o := NewOrchestrator(sdd.NewStore())
	startSnapshot, err := o.CaptureSnapshotFromGit(t.Context(), ".")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Start(".", "run", "1.0", 1, startSnapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := o.Finish(".", newDoneResult(t, ".", o, startSnapshot)); err != nil {
		t.Fatalf("diretorio relativo deveria finalizar: %v", err)
	}
}

func TestResolveEvidenceRejectsCrossPlatformUnsafePaths(t *testing.T) {
	dir := t.TempDir()
	o := NewOrchestrator(sdd.NewStore())
	for _, reference := range []string{"/etc/passwd", `\Windows\System32`, `C:\evidence\report.md`, `..\report.md`} {
		if _, err := o.resolveEvidence(dir, reference); err == nil {
			t.Fatalf("referencia insegura %q deveria falhar", reference)
		}
	}
}

func TestNewSnapshotBindsBasePatchAndFinalState(t *testing.T) {
	baseSHA := strings.Repeat("a", 40)
	snapshot := NewSnapshot(baseSHA, "patch", "estado-final")
	patchDigest := sha256.Sum256([]byte("patch"))
	stateDigest := sha256.Sum256([]byte("estado-final"))
	if snapshot.BaseSHA != baseSHA || snapshot.PatchSHA256 != hex.EncodeToString(patchDigest[:]) || snapshot.FinalStateSHA256 != hex.EncodeToString(stateDigest[:]) {
		t.Fatalf("snapshot nao vinculou os digests verificaveis: %#v", snapshot)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
	}
}

func TestOrchestratorStartsAttemptIdempotentlyAfterCrash(t *testing.T) {
	dir := t.TempDir()
	for _, file := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatal(err)
	}
	o := NewOrchestrator(store)
	snapshot := testSnapshot()
	if _, err := o.Start(dir, "run", "1.0", 1, snapshot); err != nil {
		t.Fatal(err)
	}
	state, err := o.Start(dir, "run", "1.0", 1, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Events) != 2 || state.Events[1].Action != "attempt_started" {
		t.Fatalf("retomada deveria preservar um unico evento de inicio: %#v", state.Events)
	}
}

func TestOrchestratorStartNextSelectsReadyTaskDeterministically(t *testing.T) {
	dir := newOrchestratorStateDir(t)
	store := sdd.NewStore()
	state, err := store.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	state.Tasks = map[string]sdd.TaskState{
		"2.0": {ID: "2.0", Status: sdd.StatusDraft, Dependencies: []string{"1.0"}, Requirements: []string{}, Ownership: []string{}, EvidenceRefs: []string{}},
		"1.0": {ID: "1.0", Status: sdd.StatusDraft, Dependencies: []string{}, Requirements: []string{}, Ownership: []string{}, EvidenceRefs: []string{}},
	}
	if err := store.Save(dir, state); err != nil {
		t.Fatal(err)
	}
	started, taskID, attempt, err := NewOrchestrator(store).StartNext(dir, "run", testSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if taskID != "1.0" || attempt != 1 || started.Tasks[taskID].Status != sdd.StatusExecuting {
		t.Fatalf("tentativa determinística inesperada: task=%s attempt=%d state=%#v", taskID, attempt, started.Tasks[taskID])
	}
}

func TestOrchestratorSerializesConcurrentStarts(t *testing.T) {
	dir := newOrchestratorStateDir(t)
	o := NewOrchestrator(sdd.NewStore())
	snapshot := testSnapshot()

	var group sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := o.Start(dir, "run", "1.0", 1, snapshot)
			errs <- err
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil && !errors.Is(err, ErrWriterLocked) {
			t.Fatalf("inicio concorrente deveria falhar apenas pelo lock: %v", err)
		}
	}

	state, err := sdd.NewStore().Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Events) != 2 {
		t.Fatalf("eventos = %d, quero initialized + attempt_started", len(state.Events))
	}
}

func TestOrchestratorRejectsSecondWriter(t *testing.T) {
	dir := newOrchestratorStateDir(t)
	o := NewOrchestrator(sdd.NewStore())
	release, err := o.AcquireWriter(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := release(); err != nil {
			t.Fatal(err)
		}
	}()

	_, err = o.AcquireWriter(dir)
	if !errors.Is(err, ErrWriterLocked) {
		t.Fatalf("segundo escritor = %v, quero ErrWriterLocked", err)
	}
}

func TestOrchestratorFinishesAttemptIdempotently(t *testing.T) {
	dir := newOrchestratorStateDir(t)
	o := NewOrchestrator(sdd.NewStore())
	startSnapshot, err := o.CaptureSnapshotFromGit(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Start(dir, "run", "1.0", 1, startSnapshot); err != nil {
		t.Fatal(err)
	}
	result := newDoneResult(t, dir, o, startSnapshot)
	if result.PatchSHA256 == startSnapshot.PatchSHA256 || result.FinalStateSHA256 == startSnapshot.FinalStateSHA256 {
		t.Fatal("resultado final deve refletir a alteração posterior ao Start")
	}
	if _, err := o.Finish(dir, result); err != nil {
		t.Fatal(err)
	}
	state, err := o.Finish(dir, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Events) != 3 || state.Events[2].Action != "attempt_done" {
		t.Fatalf("finalizacao deveria ter evento unico e terminal: %#v", state.Events)
	}
	if _, err := o.Start(dir, "run", "1.0", 1, startSnapshot); err != nil {
		t.Fatal(err)
	}
	state, err = sdd.NewStore().Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Events) != 3 {
		t.Fatalf("tentativa terminal nao pode ser reiniciada: %#v", state.Events)
	}
}

func TestOrchestratorFinishRejectsSnapshotAndPhysicalEvidenceDivergence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*sdd.ExecutionResult)
	}{
		{name: "patch artificial", mutate: func(result *sdd.ExecutionResult) { result.PatchSHA256 = strings.Repeat("b", 64) }},
		{name: "evidencia inexistente", mutate: func(result *sdd.ExecutionResult) {
			result.Evidence = []string{"missing.log"}
			result.Criteria[0].EvidenceRef = "missing.log#criterio"
		}},
		{name: "digest de teste artificial", mutate: func(result *sdd.ExecutionResult) { result.Tests[0].OutputSHA256 = strings.Repeat("c", 64) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := newOrchestratorStateDir(t)
			o := NewOrchestrator(sdd.NewStore())
			startSnapshot, err := o.CaptureSnapshotFromGit(t.Context(), dir)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := o.Start(dir, "run", "1.0", 1, startSnapshot); err != nil {
				t.Fatal(err)
			}
			result := newDoneResult(t, dir, o, startSnapshot)
			test.mutate(&result)
			if _, err := o.Finish(dir, result); err == nil {
				t.Fatal("resultado sem vínculo físico deveria ser rejeitado")
			}
		})
	}
}

func newOrchestratorStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, file := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sdd.NewStore().Initialize(dir, "run"); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "teste@example.com")
	runGit(t, dir, "config", "user.name", "Teste")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "-c", "commit.gpgSign=false", "commit", "-m", "base")
	return dir
}

func newDoneResult(t *testing.T, dir string, orchestrator *Orchestrator, startSnapshot Snapshot) sdd.ExecutionResult {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "change.go"), []byte("package change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	evidenceDir := filepath.Join(dir, "evidence")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	logContent := []byte("PASS\n")
	if err := os.WriteFile(filepath.Join(evidenceDir, "test.log"), logContent, 0o644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(logContent)
	result := sdd.ExecutionResult{
		SchemaVersion:    sdd.SchemaVersion,
		RunID:            "run",
		TaskID:           "1.0",
		Attempt:          1,
		Status:           sdd.StatusDone,
		BaseSHA:          startSnapshot.BaseSHA,
		PatchSHA256:      strings.Repeat("0", 64),
		PatchRef:         "evidence/patch.diff",
		FinalStateSHA256: strings.Repeat("0", 64),
		Tests:            []sdd.TestProof{{Command: "go test ./...", ExitCode: 0, OutputSHA256: hex.EncodeToString(digest[:])}},
		Criteria:         []sdd.CriterionProof{{ID: "AC-01", EvidenceRef: "evidence/test.log#criterio"}},
		Evidence:         []string{"evidence/test.log"},
		ReviewVerdict:    "approved",
	}
	finalSnapshot, patch, err := orchestrator.captureFinalSnapshot(dir, result)
	if err != nil {
		t.Fatal(err)
	}
	result.PatchSHA256 = finalSnapshot.PatchSHA256
	result.FinalStateSHA256 = finalSnapshot.FinalStateSHA256
	if err := os.WriteFile(filepath.Join(dir, result.PatchRef), patch, 0o644); err != nil {
		t.Fatal(err)
	}
	return result
}

func testSnapshot() Snapshot {
	return NewSnapshot(strings.Repeat("a", 40), "patch", "state")
}
