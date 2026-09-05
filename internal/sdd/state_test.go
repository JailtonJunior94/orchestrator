package sdd_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/sdd"
)

func TestStoreLifecycle(t *testing.T) {
	dir := t.TempDir()
	writeValidSDDArtifacts(t, dir)
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

func TestStoreValidateDirectoryRejectsApprovedArtifactDrift(t *testing.T) {
	dir := t.TempDir()
	writeValidSDDArtifacts(t, dir)
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []sdd.Artifact{sdd.ArtifactPRD, sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
		if _, err := store.Approve(dir, artifact); err != nil {
			t.Fatalf("aprovar %s: %v", artifact, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("alterado\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ValidateDirectory(dir); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("drift aprovado deveria falhar como stale: %v", err)
	}
}

func TestStorePersistsCompleteOperationalModel(t *testing.T) {
	dir := t.TempDir()
	writeValidSDDArtifacts(t, dir)
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run"); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []sdd.Artifact{sdd.ArtifactPRD, sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
		if _, err := store.Approve(dir, artifact); err != nil {
			t.Fatalf("aprovar %s: %v", artifact, err)
		}
	}
	state, err := store.ValidateDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Requirements) != 2 || len(state.DAG) != 1 || len(state.Tasks) != 1 {
		t.Fatalf("modelo operacional incompleto: %#v", state)
	}
	task := state.Tasks["1.0"]
	if len(task.Requirements) != 2 || task.Status != sdd.StatusDraft {
		t.Fatalf("task operacional invalida: %#v", task)
	}
}

func TestStoreMigrationDryRunAndRunIDScopedRollback(t *testing.T) {
	dir := t.TempDir()
	writeValidSDDArtifacts(t, dir)
	legacy := []byte(`{"schema_version":1,"legacy":true}`)
	statePath := filepath.Join(dir, "sdd-state.json")
	if err := os.WriteFile(statePath, legacy, 0o600); err != nil {
		t.Fatal(err)
	}
	store := sdd.NewStore()
	planned, err := store.Migrate(dir, "migration-1", true)
	if err != nil {
		t.Fatal(err)
	}
	current, err := os.ReadFile(statePath)
	if err != nil || string(current) != string(legacy) || planned.Migration == nil {
		t.Fatalf("dry-run alterou estado: content=%q err=%v", current, err)
	}
	if _, err := store.Migrate(dir, "migration-1", false); err != nil {
		t.Fatal(err)
	}
	if err := store.RollbackMigration(dir, "outro-run"); err == nil {
		t.Fatal("rollback com run_id divergente deveria falhar")
	}
	if _, err := store.Load(dir); err != nil {
		t.Fatalf("rollback divergente removeu estado: %v", err)
	}
	if err := store.RollbackMigration(dir, "migration-1"); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(statePath)
	if err != nil || string(restored) != string(legacy) {
		t.Fatalf("rollback nao restaurou legado exato: content=%q err=%v", restored, err)
	}
}

func writeValidSDDArtifacts(t *testing.T, dir string) {
	t.Helper()
	artifacts := map[string]string{
		"prd.md":      "## Requisitos Funcionais\n\n- RF-01: fluxo.\n\n## Requisitos Não Funcionais\n\n- NFR-01: robustez.\n",
		"techspec.md": "# TechSpec\n",
		"tasks.md":    "## Tarefas\n\n| # | Status | Dependências | Paralelizável | Ownership |\n|---|---|---|---|---|\n| 1.0 | pending | — | Não | internal/a |\n\n## Cobertura de Requisitos\n\n| Tarefa | Requisitos cobertos |\n|---|---|\n| 1.0 | RF-01, NFR-01 |\n",
	}
	for name, content := range artifacts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("escrever %s: %v", name, err)
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
		PatchRef:           "patch.diff",
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

func TestStoreImportsDoneOnlyFromValidPhysicalCheckpoint(t *testing.T) {
	tests := []struct {
		name       string
		status     sdd.Status
		checkpoint bool
		wantStatus sdd.Status
		wantErr    bool
	}{
		{name: "checkpoint done com prova fisica", status: sdd.StatusDone, checkpoint: true, wantStatus: sdd.StatusDone},
		{name: "checkpoint blocked nao vira done", status: sdd.StatusBlocked, checkpoint: true, wantStatus: sdd.StatusBlocked},
		{name: "checkpoint ausente falha fechado", status: sdd.StatusDone, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			writeValidSDDArtifacts(t, dir)
			tasksPath := filepath.Join(dir, "tasks.md")
			content, err := os.ReadFile(tasksPath)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(tasksPath, []byte(strings.Replace(string(content), "pending", "done", 1)), 0o644); err != nil {
				t.Fatal(err)
			}
			logContent := []byte("PASS\n")
			patchContent := []byte("semantic patch\n")
			evidenceDir := filepath.Join(dir, "evidence")
			if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(evidenceDir, "test.log"), logContent, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(evidenceDir, "patch.diff"), patchContent, 0o644); err != nil {
				t.Fatal(err)
			}
			logDigest := sha256.Sum256(logContent)
			patchDigest := sha256.Sum256(patchContent)
			if test.checkpoint {
				result := sdd.ExecutionResult{SchemaVersion: 2, RunID: "run", TaskID: "1.0", Attempt: 1, Status: test.status, BaseSHA: strings.Repeat("a", 40), PatchSHA256: hex.EncodeToString(patchDigest[:]), PatchRef: "evidence/patch.diff", FinalStateSHA256: strings.Repeat("b", 64), Tests: []sdd.TestProof{{Command: "go test ./...", ExitCode: 0, OutputSHA256: hex.EncodeToString(logDigest[:])}}, Criteria: []sdd.CriterionProof{{ID: "RF-01", EvidenceRef: "evidence/test.log#pass"}}, Evidence: []string{"evidence/test.log"}, ReviewVerdict: "approved"}
				if test.status != sdd.StatusDone {
					result.Tests[0].ExitCode = 1
					result.ReviewVerdict = "needs_input"
				}
				encoded, marshalErr := json.Marshal(result)
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				if err := os.MkdirAll(filepath.Join(dir, ".checkpoints"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".checkpoints", "1.0.json"), encoded, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			store := sdd.NewStore()
			if _, err := store.Initialize(dir, "run"); err != nil {
				t.Fatal(err)
			}
			var state sdd.State
			for _, artifact := range []sdd.Artifact{sdd.ArtifactPRD, sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
				state, err = store.Approve(dir, artifact)
				if err != nil {
					break
				}
			}
			if test.wantErr {
				if err == nil {
					t.Fatal("checkpoint ausente deveria falhar")
				}
				return
			}
			if err != nil || state.Tasks["1.0"].Status != test.wantStatus {
				t.Fatalf("estado importado=%#v erro=%v", state.Tasks["1.0"], err)
			}
		})
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
			}, Requirements: []sdd.RequirementState{}, DAG: []sdd.DAGNode{}, Tasks: map[string]sdd.TaskState{}, EvidenceRefs: []string{}}
			if status == sdd.StatusApproved {
				state.Requirements = []sdd.RequirementState{{ID: "RF-01", Kind: "functional"}}
				state.DAG = []sdd.DAGNode{{TaskID: "1.0"}}
				state.Tasks["1.0"] = sdd.TaskState{ID: "1.0", Status: sdd.StatusDraft, Requirements: []string{"RF-01"}, Dependencies: []string{}, Ownership: []string{}, EvidenceRefs: []string{}}
			}
			if err := sdd.NewStore().Validate(state); err != nil {
				t.Fatalf("estado %s deveria ser valido: %v", status, err)
			}
		})
	}
}

// TestApproveDesbloqueiaArtefatoEditadoAposAprovacao cobre o impasse do ciclo de
// vida: um PRD aprovado e depois editado ficava travado para sempre. Ele constava
// como approved (logo, reaprovacao recusada) mas com digest divergente (logo, todo
// downstream falhava), e `invalidate --from prd` marca os descendentes, nunca a
// origem. Sem saida, um PRD aprovado nao podia ser emendado.
func TestApproveDesbloqueiaArtefatoEditadoAposAprovacao(t *testing.T) {
	dir := t.TempDir()
	writeValidSDDArtifacts(t, dir)
	store := sdd.NewStore()
	if _, err := store.Initialize(dir, "run-1"); err != nil {
		t.Fatalf("inicializar: %v", err)
	}
	for _, artifact := range []sdd.Artifact{sdd.ArtifactPRD, sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
		if _, err := store.Approve(dir, artifact); err != nil {
			t.Fatalf("aprovar %s: %v", artifact, err)
		}
	}

	// Reaprovar um artefato intacto continua proibido: a porta so existe para
	// conteudo que mudou, nao para churn de estado.
	if _, err := store.Approve(dir, sdd.ArtifactPRD); err == nil {
		t.Fatal("reaprovar artefato inalterado deveria ser recusado")
	}

	prdPath := filepath.Join(dir, "prd.md")
	original, err := os.ReadFile(prdPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prdPath, append(original, []byte("\nTexto adicional sem novo requisito.\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Approve(dir, sdd.ArtifactPRD); err != nil {
		t.Fatalf("PRD editado apos aprovacao ficou travado: %v", err)
	}

	// Downstream intacto continua exigindo invalidacao explicita: a emenda do
	// PRD nao aprova sozinha o que dele deriva.
	if _, err := store.Approve(dir, sdd.ArtifactTechSpec); err == nil {
		t.Fatal("techspec intacto nao deveria ser reaprovavel sem invalidacao")
	}
	if _, err := store.Invalidate(dir, sdd.ArtifactPRD); err != nil {
		t.Fatalf("invalidar descendentes: %v", err)
	}
	for _, artifact := range []sdd.Artifact{sdd.ArtifactTechSpec, sdd.ArtifactTasks} {
		if _, err := store.Approve(dir, artifact); err != nil {
			t.Fatalf("reaprovar %s apos invalidacao: %v", artifact, err)
		}
	}
	if _, err := store.ValidateDirectory(dir); err != nil {
		t.Fatalf("estado deveria voltar a ser valido apos o ciclo completo: %v", err)
	}
}
