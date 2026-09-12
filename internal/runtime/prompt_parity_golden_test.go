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
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type promptCaptureHook struct {
	captured string
}

func (h *promptCaptureHook) Name() string { return "golden_prompt_capture" }

func (h *promptCaptureHook) Run(_ context.Context, evt hooks.Event) error {
	pb, ok := evt.(hooks.PromptBuildEvent)
	if !ok || pb.Prompt == nil {
		return nil
	}
	h.captured = *pb.Prompt
	return nil
}

func buildGoldenRunner(t *testing.T, ctx context.Context, capture *promptCaptureHook) *airuntime.ACPRunner {
	t.Helper()
	script := acpfake.NewScript().AppendAgentMessage("golden ack").AppendSessionEnd()
	pfact, _ := newFakePersistenceFactory()
	return airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(proberBinary()),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(pfact),
		airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
		airuntime.NewCatalog().WithPromptPostBuildTestHook(capture),
	)
}

func writeGoldenMemoryFile(t *testing.T, tasksDir, name, content string) {
	t.Helper()
	memDir := filepath.Join(tasksDir, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", memDir, err)
	}
	if err := os.WriteFile(filepath.Join(memDir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", name, err)
	}
}

const goldenBasePrompt = "execute golden parity task"

const goldenTaskFileName = "task-1.0.md"

func TestPromptParityGolden(t *testing.T) {
	t.Parallel()

	t.Run("case 1: total memory absence", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		capture := &promptCaptureHook{}
		runner := buildGoldenRunner(t, ctx, capture)

		job := airuntime.Job{
			Prompt:       goldenBasePrompt,
			WorkDir:      workDirWithAgentsMD(t),
			EvidenceDir:  t.TempDir(),
			TasksDir:     t.TempDir(),
			TaskFileName: goldenTaskFileName,
			Quiet:        true,
		}

		if _, err := runner.Run(ctx, job); err != nil {
			t.Fatalf("Run: %v", err)
		}

		want := goldenBasePrompt
		if capture.captured != want {
			t.Errorf("captured prompt = %q, want %q", capture.captured, want)
		}
	})

	t.Run("case 1b: zero-value TasksDir", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		capture := &promptCaptureHook{}
		runner := buildGoldenRunner(t, ctx, capture)

		job := airuntime.Job{
			Prompt:       goldenBasePrompt,
			WorkDir:      workDirWithAgentsMD(t),
			EvidenceDir:  t.TempDir(),
			TasksDir:     "",
			TaskFileName: goldenTaskFileName,
			Quiet:        true,
		}

		if _, err := runner.Run(ctx, job); err != nil {
			t.Fatalf("Run: %v", err)
		}

		want := goldenBasePrompt
		if capture.captured != want {
			t.Errorf("captured prompt = %q, want %q", capture.captured, want)
		}
	})

	t.Run("case 2: workflow only", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tasksDir := t.TempDir()
		const workflowContent = "durable workflow fact one\ndurable workflow fact two\n"
		writeGoldenMemoryFile(t, tasksDir, "MEMORY.md", workflowContent)

		capture := &promptCaptureHook{}
		runner := buildGoldenRunner(t, ctx, capture)

		job := airuntime.Job{
			Prompt:       goldenBasePrompt,
			WorkDir:      workDirWithAgentsMD(t),
			EvidenceDir:  t.TempDir(),
			TasksDir:     tasksDir,
			TaskFileName: goldenTaskFileName,
			Quiet:        true,
		}

		if _, err := runner.Run(ctx, job); err != nil {
			t.Fatalf("Run: %v", err)
		}

		var sb strings.Builder
		sb.WriteString(goldenBasePrompt)
		sb.WriteString("\n\n## Memory Context\n")
		sb.WriteString("\n### Workflow Memory\n\n")
		sb.WriteString(workflowContent)
		want := sb.String()

		if capture.captured != want {
			t.Errorf("captured prompt = %q, want %q", capture.captured, want)
		}
	})

	t.Run("case 3: workflow and task in correct order", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tasksDir := t.TempDir()
		const workflowContent = "durable workflow fact\n"
		const taskContent = "durable task fact\n"
		writeGoldenMemoryFile(t, tasksDir, "MEMORY.md", workflowContent)
		writeGoldenMemoryFile(t, tasksDir, goldenTaskFileName, taskContent)

		capture := &promptCaptureHook{}
		runner := buildGoldenRunner(t, ctx, capture)

		job := airuntime.Job{
			Prompt:       goldenBasePrompt,
			WorkDir:      workDirWithAgentsMD(t),
			EvidenceDir:  t.TempDir(),
			TasksDir:     tasksDir,
			TaskFileName: goldenTaskFileName,
			Quiet:        true,
		}

		if _, err := runner.Run(ctx, job); err != nil {
			t.Fatalf("Run: %v", err)
		}

		var sb strings.Builder
		sb.WriteString(goldenBasePrompt)
		sb.WriteString("\n\n## Memory Context\n")
		sb.WriteString("\n### Workflow Memory\n\n")
		sb.WriteString(workflowContent)
		sb.WriteString("\n### Task Memory\n\n")
		sb.WriteString(taskContent)
		want := sb.String()

		if capture.captured != want {
			t.Errorf("captured prompt = %q, want %q", capture.captured, want)
		}
	})

	t.Run("case 4: compaction directive appended", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tasksDir := t.TempDir()
		workflowContent := strings.Repeat("durable line\n", 160)
		writeGoldenMemoryFile(t, tasksDir, "MEMORY.md", workflowContent)

		capture := &promptCaptureHook{}
		runner := buildGoldenRunner(t, ctx, capture)

		job := airuntime.Job{
			Prompt:       goldenBasePrompt,
			WorkDir:      workDirWithAgentsMD(t),
			EvidenceDir:  t.TempDir(),
			TasksDir:     tasksDir,
			TaskFileName: goldenTaskFileName,
			Quiet:        true,
		}

		if _, err := runner.Run(ctx, job); err != nil {
			t.Fatalf("Run: %v", err)
		}

		var sb strings.Builder
		sb.WriteString(goldenBasePrompt)
		sb.WriteString("\n\n## Memory Context\n")
		sb.WriteString("\n### Workflow Memory\n\n")
		sb.WriteString(workflowContent)
		sb.WriteString("\ncompact the flagged memory files before proceeding\n")
		want := sb.String()

		if capture.captured != want {
			t.Errorf("captured prompt = %q, want %q", capture.captured, want)
		}
	})
}

func TestPromptParityGolden_AnchoredAtHookDispatch(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tasksDir := t.TempDir()
	writeGoldenMemoryFile(t, tasksDir, "MEMORY.md", "anchor probe content\n")

	capture := &promptCaptureHook{}
	runner := buildGoldenRunner(t, ctx, capture)

	job := airuntime.Job{
		Prompt:       goldenBasePrompt,
		WorkDir:      workDirWithAgentsMD(t),
		EvidenceDir:  t.TempDir(),
		TasksDir:     tasksDir,
		TaskFileName: goldenTaskFileName,
		Quiet:        true,
	}

	if _, err := runner.Run(ctx, job); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if capture.captured == "" {
		t.Fatal("capture hook never observed PointPromptPostBuild dispatch; golden is not anchored at hooks.PointPromptPostBuild")
	}
	if !strings.Contains(capture.captured, "anchor probe content") {
		t.Errorf("captured prompt = %q, want it to contain injected memory content dispatched at hooks.PointPromptPostBuild", capture.captured)
	}
}
