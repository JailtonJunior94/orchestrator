package runtime_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type depthProbeHook struct {
	mu       sync.Mutex
	observed []string
}

func (h *depthProbeHook) Name() string { return "depth_probe" }

func (h *depthProbeHook) Run(_ context.Context, evt hooks.Event) error {
	promptEvt, ok := evt.(hooks.PromptBuildEvent)
	if !ok || promptEvt.Prompt == nil {
		return nil
	}
	if !strings.Contains(*promptEvt.Prompt, "## Findings to fix") {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.observed = append(h.observed, os.Getenv("AI_INVOCATION_DEPTH"))
	return nil
}

func TestACPRunnerCycleFixSessionResetsInvocationDepth(t *testing.T) {
	t.Setenv("AI_INVOCATION_DEPTH", "3")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	probe := &depthProbeHook{}

	var mu sync.Mutex
	calls := 0
	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		return "Verdict: REJECTED\n[HIGH] fix.go:1 problema\n", nil
	}

	runner := airuntime.NewACPRunner(specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(&fakeProberForReview{}),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactoryForReview{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(&fakePersistenceFactoryForReview{}),
		airuntime.NewCatalog().WithRenderer(&fakeRendererForReview{}),
		airuntime.NewCatalog().WithReviewOutputFn(reviewFn),
		airuntime.NewCatalog().WithPromptPostBuildTestHook(probe),
	)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")

	if _, err := runner.Run(ctx, airuntime.Job{
		Prompt:       "implementar tarefa x",
		WorkDir:      workDir,
		EvidenceDir:  t.TempDir(),
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	}); err != nil {
		t.Fatalf("Run falhou: %v", err)
	}

	probe.mu.Lock()
	observed := append([]string(nil), probe.observed...)
	probe.mu.Unlock()

	if len(observed) == 0 {
		t.Fatal("nenhuma sessao de correcao observada — sonda nao exercitou o FixerAdapter")
	}
	for i, depth := range observed {
		if depth != "0" {
			t.Errorf("sessao de correcao %d abriu com AI_INVOCATION_DEPTH=%q, quero \"0\" (RF-38)", i+1, depth)
		}
	}
	if got := os.Getenv("AI_INVOCATION_DEPTH"); got != "3" {
		t.Errorf("profundidade nao restaurada apos o ciclo: %q, quero \"3\"", got)
	}
}
