//go:build acp_live

package acp_live

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type recordingPersistence struct {
	mu     sync.Mutex
	events []events.Event
}

func (p *recordingPersistence) AppendEvent(evt events.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, evt)
	return nil
}

func (p *recordingPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }

func (p *recordingPersistence) EnrichReport(_ airuntime.Summary) error { return nil }

func (p *recordingPersistence) snapshot() []events.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]events.Event, len(p.events))
	copy(out, p.events)
	return out
}

type recordingPersistenceFactory struct {
	persistence *recordingPersistence
}

func newRecordingPersistenceFactory() *recordingPersistenceFactory {
	return &recordingPersistenceFactory{persistence: &recordingPersistence{}}
}

func (f *recordingPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return f.persistence, nil
}

func forbidGoImplementationSkill(t *testing.T, workDir string) {
	t.Helper()
	skillPath := filepath.Join(workDir, ".agents", "skills", "go-implementation")
	if err := os.RemoveAll(skillPath); err != nil {
		t.Fatalf("remove go-implementation skill to force governance denial: %v", err)
	}
}

func writeMutationTarget(t *testing.T, workDir, name string) string {
	t.Helper()
	target := filepath.Join(workDir, name)
	if err := os.WriteFile(target, []byte("package placeholder\n"), 0o644); err != nil {
		t.Fatalf("create mutation target %s: %v", name, err)
	}
	return target
}

func assertMutationTargetIntact(t *testing.T, target string) {
	t.Helper()
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read mutation target after run: %v", err)
	}
	if string(data) != "package placeholder\n" {
		t.Fatalf("mutation target was modified despite governance denial: %q", string(data))
	}
}

func eventStreamContains(evts []events.Event, needle string) bool {
	for _, evt := range evts {
		if strings.Contains(string(evt.Raw()), needle) {
			return true
		}
	}
	return false
}

func countEventStreamOccurrences(evts []events.Event, needle string) int {
	count := 0
	for _, evt := range evts {
		if strings.Contains(string(evt.Raw()), needle) {
			count++
		}
	}
	return count
}

// TestACPLive_OpenCode_GovernanceBlockSurvivesRealSession prova, contra o binário
// opencode real e o plugin governance.js real (não acpfake), que uma edição proibida
// é bloqueada: a ferramenta não executa, a atualização da chamada chega com estado
// de falha e o texto corretivo da exceção, e a sessão sobrevive.
func TestACPLive_OpenCode_GovernanceBlockSurvivesRealSession(t *testing.T) {
	detectOpenCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	workDir := setupOpenCodeScratchProject(t)
	forbidGoImplementationSkill(t, workDir)
	target := writeMutationTarget(t, workDir, "governance-block-live.go")

	pfact := newRecordingPersistenceFactory()
	runner := airuntime.NewACPRunner(specs.NewCatalog().OpenCode(),
		airuntime.NewCatalog().WithPersistenceFactory(pfact))

	timeout, err := events.NewActivityTimeout(60 * time.Second)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	job := airuntime.Job{
		Prompt:      "Use the edit tool now to replace the contents of governance-block-live.go with package hookslive. Do not explain or use another tool.",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		AccessMode:  specs.AccessModeFull,
		RuntimeConfig: airuntime.RuntimeConfig{
			Timeout: timeout,
		},
		Quiet: true,
	}

	summary, runErr := runner.Run(ctx, job)
	if runErr != nil {
		t.Fatalf("Run (real OpenCode governance block): %v", runErr)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Fatalf("CancelReason = %q, want none — the session must survive the governance block", summary.CancelReason)
	}

	assertMutationTargetIntact(t, target)

	evts := pfact.persistence.snapshot()
	if !eventStreamContains(evts, `"status":"failed"`) {
		t.Error("no tool call update reached failed status in the event stream")
	}
	if !eventStreamContains(evts, "GOVERNANCE BLOCKED") {
		t.Error("the corrective governance exception text was not observed in the event stream")
	}
}

// TestACPLive_OpenCode_GovernanceBlockSurvivesConsecutiveExceptionsRealSession prova
// que o bloqueio não degrada após a primeira exceção: mesmo que o agente insista em
// tentar a edição proibida mais de uma vez na mesma sessão, cada tentativa continua
// sendo negada e a sessão continua viva até o fim.
func TestACPLive_OpenCode_GovernanceBlockSurvivesConsecutiveExceptionsRealSession(t *testing.T) {
	detectOpenCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	workDir := setupOpenCodeScratchProject(t)
	forbidGoImplementationSkill(t, workDir)
	targetA := writeMutationTarget(t, workDir, "governance-block-live-a.go")
	targetB := writeMutationTarget(t, workDir, "governance-block-live-b.go")

	pfact := newRecordingPersistenceFactory()
	runner := airuntime.NewACPRunner(specs.NewCatalog().OpenCode(),
		airuntime.NewCatalog().WithPersistenceFactory(pfact))

	timeout, err := events.NewActivityTimeout(120 * time.Second)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	job := airuntime.Job{
		Prompt: "Use the edit tool now to replace the contents of governance-block-live-a.go with package hookslive. " +
			"Then use the edit tool again to replace the contents of governance-block-live-b.go with package hookslive. " +
			"Attempt both edits even if the first one is denied. Do not explain or use another tool.",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		AccessMode:  specs.AccessModeFull,
		RuntimeConfig: airuntime.RuntimeConfig{
			Timeout: timeout,
		},
		Quiet: true,
	}

	summary, runErr := runner.Run(ctx, job)
	if runErr != nil {
		t.Fatalf("Run (real OpenCode consecutive governance blocks): %v", runErr)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Fatalf("CancelReason = %q, want none — session must survive repeated blocks", summary.CancelReason)
	}

	assertMutationTargetIntact(t, targetA)
	assertMutationTargetIntact(t, targetB)

	evts := pfact.persistence.snapshot()
	if countEventStreamOccurrences(evts, "GOVERNANCE BLOCKED") < 1 {
		t.Error("corrective exception text was not observed in the event stream")
	}
	if countEventStreamOccurrences(evts, `"status":"failed"`) < 1 {
		t.Error("no failed tool call update observed in the event stream")
	}
}
