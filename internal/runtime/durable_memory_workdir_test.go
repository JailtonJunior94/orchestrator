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
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestDurableMemoryWiring_ProjectLayerFollowsWorkDirWhenWorkDirIsNotTheRepositoryRoot(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	repositoryRoot := t.TempDir()
	ensureAgentsMD(t, repositoryRoot)
	workDir := filepath.Join(repositoryRoot, "services", "billing")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("create nested work dir: %v", err)
	}
	ensureAgentsMD(t, workDir)

	runFor := func(prompt string) *promptCaptureHook {
		script := acpfake.NewScript().
			AppendAgentMessage("## Memory Declared\ndecision: usar backoff exponencial").
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
			Prompt:      prompt,
			WorkDir:     workDir,
			EvidenceDir: t.TempDir(),
			Quiet:       true,
			RuntimeConfig: airuntime.RuntimeConfig{
				DurableMemoryEnabled: true,
			},
		}
		if _, err := runner.Run(ctx, job); err != nil {
			t.Fatalf("Run: %v", err)
		}
		return capture
	}

	runFor("session one")

	nestedPage := filepath.Join(workDir, ".aispec", "memory", "PROJECT.md")
	if _, err := os.Stat(nestedPage); err != nil {
		t.Fatalf("project layer page must live under Job.WorkDir (%s): %v", nestedPage, err)
	}
	rootPage := filepath.Join(repositoryRoot, ".aispec", "memory", "PROJECT.md")
	if _, err := os.Stat(rootPage); !os.IsNotExist(err) {
		t.Fatalf("project layer must not leak to the repository root (%s), stat err=%v", rootPage, err)
	}

	second := runFor("session two")
	if !strings.Contains(second.captured, "## Durable Memory") {
		t.Fatalf("second session must read back the project layer written under Job.WorkDir; captured=%q", second.captured)
	}
	if !strings.Contains(second.captured, "usar backoff exponencial") {
		t.Errorf("second session must carry the fact declared by the first one; captured=%q", second.captured)
	}
}
