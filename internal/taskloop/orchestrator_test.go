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

func TestNewSnapshotBindsBasePatchAndFinalState(t *testing.T) {
	snapshot := NewSnapshot("base", "patch", "estado-final")
	patchDigest := sha256.Sum256([]byte("patch"))
	stateDigest := sha256.Sum256([]byte("estado-final"))
	if snapshot.BaseSHA != "base" || snapshot.PatchSHA256 != hex.EncodeToString(patchDigest[:]) || snapshot.FinalStateSHA256 != hex.EncodeToString(stateDigest[:]) {
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
	if _, err := o.Start(dir, "run", "1.0", 1, NewSnapshot("base", "patch", "state")); err != nil {
		t.Fatal(err)
	}
	state, err := o.Start(dir, "run", "1.0", 1, NewSnapshot("base", "patch", "state"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Events) != 2 || state.Events[1].Action != "attempt_started" {
		t.Fatalf("retomada deveria preservar um unico evento de inicio: %#v", state.Events)
	}
}

func TestOrchestratorSerializesConcurrentStarts(t *testing.T) {
	dir := newOrchestratorStateDir(t)
	o := NewOrchestrator(sdd.NewStore())
	snapshot := NewSnapshot("base", "patch", "state")

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
	result := newDoneResult()
	if _, err := o.Finish(dir, result); err != nil {
		t.Fatal(err)
	}
	state, err := o.Finish(dir, result)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Events) != 2 || state.Events[1].Action != "attempt_done" {
		t.Fatalf("finalizacao deveria ter evento unico e terminal: %#v", state.Events)
	}
	if _, err := o.Start(dir, "run", "1.0", 1, NewSnapshot("base", "patch", "state")); err != nil {
		t.Fatal(err)
	}
	state, err = sdd.NewStore().Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Events) != 2 {
		t.Fatalf("tentativa terminal nao pode ser reiniciada: %#v", state.Events)
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
	return dir
}

func newDoneResult() sdd.ExecutionResult {
	digest := strings.Repeat("a", 64)
	return sdd.ExecutionResult{
		SchemaVersion:    sdd.SchemaVersion,
		RunID:            "run",
		TaskID:           "1.0",
		Attempt:          1,
		Status:           sdd.StatusDone,
		BaseSHA:          digest,
		PatchSHA256:      digest,
		FinalStateSHA256: digest,
		Tests:            []sdd.TestProof{{Command: "go test ./...", ExitCode: 0, OutputSHA256: digest}},
		Criteria:         []sdd.CriterionProof{{ID: "AC-01", EvidenceRef: "relatorio.md#criterio"}},
		Evidence:         []string{"relatorio.md"},
		ReviewVerdict:    "approved",
	}
}
