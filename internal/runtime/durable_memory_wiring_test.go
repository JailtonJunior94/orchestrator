package runtime_test

import (
	"context"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestDurableMemoryWiring_Disabled_PreservesLegacyPath verifica que, com
// Job.DurableMemoryEnabled==false (zero-value), o prompt final permanece
// identico ao caminho legado — a fachada nao e sequer construida. Este teste
// e complementar ao golden de paridade da tarefa 6.0, exercitando o wiring
// diretamente pelo campo do Job (tarefa 7.0, RF-28, RF-29).
func TestDurableMemoryWiring_Disabled_PreservesLegacyPath(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendAgentMessage("ack").AppendSessionEnd()
	pfact, _ := newFakePersistenceFactory()
	capture := &promptCaptureHook{}
	runner := airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(proberBinary()),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(pfact),
		airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
		airuntime.NewCatalog().WithPromptPostBuildTestHook(capture),
	)

	job := airuntime.Job{
		Prompt:      "execute durable memory wiring off",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
	}

	if _, err := runner.Run(ctx, job); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if capture.captured != job.Prompt {
		t.Errorf("captured prompt = %q, want %q (feature disabled must be a no-op)", capture.captured, job.Prompt)
	}
}

// TestDurableMemoryWiring_Enabled_InjectsDurableBlockOnSubsequentSession prova
// a fachada ativada ponta a ponta: uma primeira sessao registra o sinal
// estruturado via RecordSession (session.post_end) — independente do agente,
// RF-09 — e uma segunda sessao, no mesmo diretorio de tasks, o le de volta
// via BuildContext e injeta no prompt.
func TestDurableMemoryWiring_Enabled_InjectsDurableBlockOnSubsequentSession(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasksDir := t.TempDir()
	workDir := workDirWithAgentsMD(t)

	runFor := func(prompt string) *promptCaptureHook {
		script := acpfake.NewScript().AppendAgentMessage("agent output ignored by RF-09 structured part").AppendSessionEnd()
		pfact, _ := newFakePersistenceFactory()
		capture := &promptCaptureHook{}
		runner := airuntime.NewACPRunner(
			specs.NewCatalog().Claude(),
			airuntime.NewCatalog().WithProber(proberBinary()),
			airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
			airuntime.NewCatalog().WithPersistenceFactory(pfact),
			airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
			airuntime.NewCatalog().WithPromptPostBuildTestHook(capture),
		)

		job := airuntime.Job{
			Prompt:       prompt,
			WorkDir:      workDir,
			EvidenceDir:  t.TempDir(),
			TasksDir:     tasksDir,
			TaskFileName: "task-1.0.md",
			Quiet:        true,
			RuntimeConfig: airuntime.RuntimeConfig{
				DurableMemoryEnabled: true,
			},
		}
		if _, err := runner.Run(ctx, job); err != nil {
			t.Fatalf("Run: %v", err)
		}
		return capture
	}

	first := runFor("session one")
	if strings.Contains(first.captured, "## Durable Memory") {
		t.Errorf("primeira sessao nao deveria ter fatos previos para injetar: %q", first.captured)
	}

	second := runFor("session two")
	if !strings.Contains(second.captured, "## Durable Memory") {
		t.Fatalf("segunda sessao deveria injetar bloco de memoria duravel; captured=%q", second.captured)
	}
	if !strings.Contains(second.captured, "Exit Status:") {
		t.Errorf("segunda sessao deveria conter o sinal estruturado gravado na primeira sessao; captured=%q", second.captured)
	}
}

func TestDurableMemoryWiring_Enabled_CapturesDeclaredSectionWhenAgentCollaborates(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasksDir := t.TempDir()
	workDir := workDirWithAgentsMD(t)

	script := acpfake.NewScript().
		AppendAgentMessage("## Memory Declared\ndecision: use exponential backoff for retries").
		AppendSessionEnd()
	pfact, _ := newFakePersistenceFactory()
	capture := &promptCaptureHook{}
	runner := airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(proberBinary()),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(pfact),
		airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
		airuntime.NewCatalog().WithPromptPostBuildTestHook(capture),
	)

	job := airuntime.Job{
		Prompt:       "session declaring a section",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		TasksDir:     tasksDir,
		TaskFileName: "task-1.0.md",
		Quiet:        true,
		RuntimeConfig: airuntime.RuntimeConfig{
			DurableMemoryEnabled: true,
		},
	}
	if _, err := runner.Run(ctx, job); err != nil {
		t.Fatalf("Run: %v", err)
	}

	readScript := acpfake.NewScript().AppendAgentMessage("ack").AppendSessionEnd()
	readCapture := &promptCaptureHook{}
	readRunner := airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(proberBinary()),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: readScript, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(pfact),
		airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
		airuntime.NewCatalog().WithPromptPostBuildTestHook(readCapture),
	)
	readJob := airuntime.Job{
		Prompt:       "session reading declared section back",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		TasksDir:     tasksDir,
		TaskFileName: "task-1.0.md",
		Quiet:        true,
		RuntimeConfig: airuntime.RuntimeConfig{
			DurableMemoryEnabled: true,
		},
	}
	if _, err := readRunner.Run(ctx, readJob); err != nil {
		t.Fatalf("Run (read back): %v", err)
	}

	if !strings.Contains(readCapture.captured, "decision: use exponential backoff for retries") {
		t.Fatalf("declared section should have been captured and persisted as a fact (RF-09 second source); captured=%q", readCapture.captured)
	}
}

func TestDurableMemoryWiring_Enabled_LeavesDeclaredSectionEmptyWithoutFailingWhenAgentDoesNotCollaborate(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendAgentMessage("agent output without the declared-section marker").AppendSessionEnd()
	pfact, fp := newFakePersistenceFactory()
	capture := &promptCaptureHook{}
	runner := airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(proberBinary()),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(pfact),
		airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
		airuntime.NewCatalog().WithPromptPostBuildTestHook(capture),
	)

	job := airuntime.Job{
		Prompt:       "session without declared section",
		WorkDir:      workDirWithAgentsMD(t),
		EvidenceDir:  t.TempDir(),
		TasksDir:     t.TempDir(),
		TaskFileName: "task-1.0.md",
		Quiet:        true,
		RuntimeConfig: airuntime.RuntimeConfig{
			DurableMemoryEnabled: true,
		},
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.MemoryEvidence == nil {
		t.Fatal("Summary.MemoryEvidence must still be populated even without a declared section (RF-09: never empty on the structured side)")
	}
	for _, hookErr := range summary.HookDispatchErrors {
		t.Errorf("HookDispatchErrors deveria estar vazio quando o agente nao colabora; obtido: %s", hookErr)
	}
	if fp.summary == nil {
		t.Fatal("persistence.EnrichReport nao foi chamado")
	}
}

func TestDurableMemoryWiring_Enabled_PopulatesEvidenceMetricsAndTraceability(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendAgentMessage("ack").AppendSessionEnd()
	pfact, fp := newFakePersistenceFactory()
	capture := &promptCaptureHook{}
	runner := airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(proberBinary()),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(pfact),
		airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
		airuntime.NewCatalog().WithPromptPostBuildTestHook(capture),
	)

	job := airuntime.Job{
		Prompt:       "session with evidence",
		WorkDir:      workDirWithAgentsMD(t),
		EvidenceDir:  t.TempDir(),
		TasksDir:     t.TempDir(),
		TaskFileName: "task-8.0.md",
		Quiet:        true,
		RuntimeConfig: airuntime.RuntimeConfig{
			DurableMemoryEnabled: true,
		},
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if summary.MemoryEvidence == nil {
		t.Fatal("Summary.MemoryEvidence nao populado com durable memory ativada")
	}
	if summary.MemoryEvidence.SessionID == "" {
		t.Error("MemoryEvidence.SessionID vazio; fato nao seria rastreavel ate a sessao (RF-32)")
	}
	if summary.MemoryEvidence.CLI != "claude" {
		t.Errorf("MemoryEvidence.CLI = %q; want claude", summary.MemoryEvidence.CLI)
	}
	if summary.Metrics.IsZero() {
		t.Error("Summary.Metrics nao deveria ser zero-value com durable memory ativada e fatos gravados")
	}
	for _, hookErr := range summary.HookDispatchErrors {
		t.Errorf("HookDispatchErrors deveria estar vazio para eventos de memoria conhecidos; obtido: %s", hookErr)
	}

	if fp.summary == nil {
		t.Fatal("persistence.EnrichReport nao foi chamado")
	}
	if fp.summary.MemoryEvidence == nil {
		t.Fatal("summary persistido nao contem MemoryEvidence")
	}
}
