package sdd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

func TestStoreLifecycle(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("escrever %s: %v", name, err)
		}
	}
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run-1"); err != nil {
		t.Fatalf("inicializar: %v", err)
	}
	for _, artifact := range []sdd.Artifact{sdd.ArtifactPRD, sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
		if _, err := store.Approve(dir, artifact); err != nil {
			t.Fatalf("aprovar %s: %v", artifact, err)
		}
	}
	state, err := store.Invalidate(dir, sdd.ArtifactPRD)
	if err != nil {
		t.Fatalf("invalidar: %v", err)
	}
	if state.Artifacts[sdd.ArtifactPRD].Status != sdd.StatusApproved {
		t.Fatal("PRD aprovado nao deve ficar stale")
	}
	for _, artifact := range []sdd.Artifact{sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
		if state.Artifacts[artifact].Status != sdd.StatusStale || state.Artifacts[artifact].Approved {
			t.Fatalf("%s deveria estar stale", artifact)
		}
	}
}

func TestStoreRejectsInvalidApproval(t *testing.T) {
	state := sdd.State{SchemaVersion: sdd.SchemaVersion, RunID: "run", Artifacts: map[sdd.Artifact]sdd.ArtifactState{
		sdd.ArtifactPRD:      {Status: sdd.StatusApproved, Approved: true},
		sdd.ArtifactTechSpec: {Status: sdd.StatusDraft},
		sdd.ArtifactTasks:    {Status: sdd.StatusDraft},
	}}
	if err := sdd.NewStore().Validate(state); err == nil {
		t.Fatal("estado aprovado sem digest deveria falhar")
	}
}

func TestStoreWritesAtomicallyLoadableJSON(t *testing.T) {
	dir := t.TempDir()
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatalf("inicializar: %v", err)
	}
	if _, err := store.Load(dir); err != nil {
		t.Fatalf("reler estado: %v", err)
	}
}

func TestStoreRejectsDoneWithoutIndependentProof(t *testing.T) {
	result := sdd.ExecutionResult{SchemaVersion: sdd.SchemaVersion, RunID: "run", TaskID: "1.0", Attempt: 1, Status: sdd.StatusDone}
	if err := sdd.NewStore().ValidateExecutionResult(result); err == nil {
		t.Fatal("done sem prova deveria falhar")
	}
}

func TestStoreRejectsOutOfOrderApproval(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("escrever %s: %v", name, err)
		}
	}
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatalf("inicializar: %v", err)
	}
	if _, err := store.Approve(dir, sdd.ArtifactTechSpec); err == nil {
		t.Fatal("techspec sem PRD aprovado deveria falhar")
	}
}

func TestStoreRejectsDownstreamApprovalAfterUpstreamChange(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"prd.md", "techspec.md", "tasks.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("escrever %s: %v", name, err)
		}
	}
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatalf("inicializar: %v", err)
	}
	if _, err := store.Approve(dir, sdd.ArtifactPRD); err != nil {
		t.Fatalf("aprovar PRD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "prd.md"), []byte("alterado"), 0o644); err != nil {
		t.Fatalf("alterar PRD: %v", err)
	}
	if _, err := store.Approve(dir, sdd.ArtifactTechSpec); err == nil {
		t.Fatal("techspec com PRD alterado deveria falhar")
	}
}

func TestStoreRejectsInvalidArtifactTransitions(t *testing.T) {
	dir := t.TempDir()
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatalf("inicializar: %v", err)
	}
	if _, err := store.Invalidate(dir, sdd.ArtifactTasks); err == nil {
		t.Fatal("tasks sem descendentes deveria falhar")
	}
	if _, err := store.Approve(dir, sdd.Artifact("desconhecido")); err == nil {
		t.Fatal("artefato desconhecido deveria falhar")
	}
}

func TestStoreValidatesCompleteDoneResult(t *testing.T) {
	digest := strings.Repeat("a", 64)
	result := sdd.ExecutionResult{
		SchemaVersion:      sdd.SchemaVersion,
		RunID:              "run",
		TaskID:             "3.0",
		Attempt:            1,
		Status:             sdd.StatusDone,
		BaseSHA:            digest,
		PatchSHA256:        digest,
		FinalStateSHA256:   digest,
		CoverageRegression: false,
		Tests:              []sdd.TestProof{{Command: "go test ./...", ExitCode: 0, OutputSHA256: digest}},
		Criteria:           []sdd.CriterionProof{{ID: "AC-01", EvidenceRef: "report.md#criterio"}},
		Evidence:           []string{"report.md"},
		ReviewVerdict:      "approved",
	}
	if err := sdd.NewStore().ValidateExecutionResult(result); err != nil {
		t.Fatalf("resultado completo deveria ser valido: %v", err)
	}
}

func TestStoreRejectsNonHexApprovedDigest(t *testing.T) {
	state := sdd.State{SchemaVersion: sdd.SchemaVersion, RunID: "run", Artifacts: map[sdd.Artifact]sdd.ArtifactState{
		sdd.ArtifactPRD:      {Status: sdd.StatusApproved, SHA256: strings.Repeat("z", 64), Approved: true},
		sdd.ArtifactTechSpec: {Status: sdd.StatusDraft},
		sdd.ArtifactTasks:    {Status: sdd.StatusDraft},
	}}
	if err := sdd.NewStore().Validate(state); err == nil {
		t.Fatal("digest nao hexadecimal deveria falhar")
	}
}

func TestStoreRejectsInconsistentApprovalStatus(t *testing.T) {
	state := sdd.State{SchemaVersion: sdd.SchemaVersion, RunID: "run", Artifacts: map[sdd.Artifact]sdd.ArtifactState{
		sdd.ArtifactPRD:      {Status: sdd.StatusStale, SHA256: strings.Repeat("a", 64), Approved: true},
		sdd.ArtifactTechSpec: {Status: sdd.StatusDraft},
		sdd.ArtifactTasks:    {Status: sdd.StatusDraft},
	}}
	if err := sdd.NewStore().Validate(state); err == nil {
		t.Fatal("status stale com aprovacao deveria falhar")
	}
}

func TestStoreAcceptsCanonicalArtifactStates(t *testing.T) {
	for _, status := range []sdd.Status{
		sdd.StatusDraft,
		sdd.StatusApproved,
		sdd.StatusStale,
		sdd.StatusExecuting,
		sdd.StatusBlocked,
		sdd.StatusNeedsInput,
		sdd.StatusFailed,
		sdd.StatusDone,
	} {
		t.Run(string(status), func(t *testing.T) {
			entry := sdd.ArtifactState{Status: status}
			if status == sdd.StatusApproved {
				entry.SHA256 = strings.Repeat("a", 64)
				entry.Approved = true
			}
			state := sdd.State{SchemaVersion: sdd.SchemaVersion, RunID: "run", Artifacts: map[sdd.Artifact]sdd.ArtifactState{
				sdd.ArtifactPRD:      entry,
				sdd.ArtifactTechSpec: entry,
				sdd.ArtifactTasks:    entry,
			}}
			if err := sdd.NewStore().Validate(state); err != nil {
				t.Fatalf("estado %s deveria ser valido: %v", status, err)
			}
		})
	}
}
