package runtime_test

import (
	"bytes"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func TestACPRunner_SessionPostEndDispatchFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendAgentMessage("hook dispatch failure probe").AppendSessionEnd()
	pfact, _ := newFakePersistenceFactory()
	runner := airuntime.NewACPRunner(
		specs.NewCatalog().Claude(),
		airuntime.NewCatalog().WithProber(proberBinary()),
		airuntime.NewCatalog().WithClientFactory(&fakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.NewCatalog().WithPersistenceFactory(pfact),
		airuntime.NewCatalog().WithRenderer(&discardRenderer{}),
	)

	notADir := filepath.Join(t.TempDir(), "tasks-dir-is-a-file")
	if err := os.WriteFile(notADir, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	job := airuntime.Job{
		Prompt:         "prompt",
		WorkDir:        workDirWithAgentsMD(t),
		EvidenceDir:    t.TempDir(),
		TasksDir:       notADir,
		TaskFileName:   "task-1.0.md",
		SkipDriftGuard: true,
		Quiet:          true,
	}

	var logBuf bytes.Buffer
	origOutput := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(origOutput)

	summary, runErr := runner.Run(ctx, job)

	if runErr != nil {
		t.Fatalf("Run: session must not abort on memory dispatch failure; got err=%v", runErr)
	}

	if len(summary.HookDispatchErrors) == 0 {
		t.Fatal("Summary.HookDispatchErrors is empty; dispatch failure must be composed into the Summary (RF-30)")
	}
	if !strings.Contains(summary.HookDispatchErrors[0], "memory_persist") {
		t.Errorf("Summary.HookDispatchErrors[0] = %q, want it to reference the memory_persist hook failure", summary.HookDispatchErrors[0])
	}

	if !strings.Contains(logBuf.String(), "session.post_end hook dispatch failed") {
		t.Errorf("log output = %q, want explicit log of the session.post_end dispatch failure (RF-30, no silent degradation)", logBuf.String())
	}
}
