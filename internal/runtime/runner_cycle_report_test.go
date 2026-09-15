package runtime_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestACPRunnerCyclePersistsConsolidatedReport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessão").
		AppendSessionEnd()

	reviewFn := func(_ context.Context, _ airuntime.Job) (string, error) {
		return "Verdict: REJECTED\n[HIGH] internal/x/fix.go:10 mesmo achado\n\n## Mapa de Critérios de Aceite\n- [atendido] Faz X -> go test ./... -> PASS\n", nil
	}

	runner := airuntime.NewACPRunner(specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(&fakeProberForReview{}),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactoryForReview{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(persistence.NewSessionPersistenceFactory(fs.NewOSFileSystem())),
		airuntime.NewCatalog().WithRenderer(&fakeRendererForReview{}),
		airuntime.NewCatalog().WithReviewOutputFn(reviewFn),
	)

	workDir := gitWorkDirWithAgentsMDForCycle(t)
	tasksDir := t.TempDir()
	writeCycleTaskFile(t, tasksDir, "task-x.md")
	evidenceDir := t.TempDir()

	summary, err := runner.Run(ctx, airuntime.Job{
		Prompt:       "implementar tarefa x",
		WorkDir:      workDir,
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		AutoReview:   true,
		TasksDir:     tasksDir,
		TaskFileName: "task-x.md",
	})
	if err != nil {
		t.Fatalf("Run falhou: %v", err)
	}
	if len(summary.CycleRounds) < 2 {
		t.Fatalf("CycleRounds=%d, quero >= 2", len(summary.CycleRounds))
	}

	raw, readErr := os.ReadFile(filepath.Join(evidenceDir, "execution_report.md"))
	if readErr != nil {
		t.Fatalf("ler relatorio persistido: %v", readErr)
	}
	report := string(raw)

	fingerprint := summary.CycleRounds[0].Fingerprint
	if fingerprint == "" {
		t.Fatal("fingerprint ausente no summary — sonda invalida")
	}

	for _, want := range []string{
		"## Ciclo de Aprovação",
		"cycle_stop_reason: no_convergence",
		"cycle_rounds: 2",
		"REJECTED",
		fingerprint,
		"high=1",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("relatorio persistido nao contem %q (RF-42):\n%s", want, report)
		}
	}
}
