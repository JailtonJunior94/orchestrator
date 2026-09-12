//go:build integration

package durable_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestBootstrapOnEmptyRepositoryDoesNotFail(t *testing.T) {
	projectDir := t.TempDir()
	tasksDir := t.TempDir() + string(os.PathSeparator) + "prd-nonexistent"

	facade := durable.NewFacade(fs.NewOSFileSystem(), durable.FacadeConfig{
		ProjectDir: projectDir,
		TasksDir:   tasksDir,
	})

	memCtx, err := facade.BuildContext(context.Background(), durable.MemoryScope{TaskFileName: "task-1.0-example.md"})
	if err != nil {
		t.Fatalf("BuildContext on empty repository must not fail (RF-21), got: %v", err)
	}
	if memCtx.Block != "" {
		t.Fatalf("BuildContext on empty repository must not emit a memory block, got: %q", memCtx.Block)
	}
	if memCtx.Omitted != 0 || memCtx.Contradicted != 0 {
		t.Fatalf("BuildContext on empty repository must report zero omissions and contradictions, got omitted=%d contradicted=%d",
			memCtx.Omitted, memCtx.Contradicted)
	}
}

func TestHandoffCrossCLIDeliversAtLeast90PercentOfFacts(t *testing.T) {
	projectDir := t.TempDir()
	tasksDir := t.TempDir()

	writerFacade := durable.NewFacade(fs.NewOSFileSystem(), durable.FacadeConfig{
		ProjectDir: projectDir,
		TasksDir:   tasksDir,
	})

	const factCount = 8
	writerCLIs := []string{"claude", "codex", "opencode", "copilot"}
	ctx := context.Background()
	for i := 0; i < factCount; i++ {
		cli := writerCLIs[i%len(writerCLIs)]
		_, err := writerFacade.RecordSession(ctx, durable.SessionFacts{
			TaskFileName:    fmt.Sprintf("task-%d.0-cross-cli.md", i),
			ExitStatus:      "none",
			EventsCount:     i,
			DeclaredSection: fmt.Sprintf("durable fact %d emitted by session N under CLI %s", i, cli),
			SessionID:       fmt.Sprintf("session-N-%d", i),
			CLI:             cli,
		})
		if err != nil {
			t.Fatalf("record session %d under CLI %s: %v", i, cli, err)
		}
	}

	readerFacade := durable.NewFacade(fs.NewOSFileSystem(), durable.FacadeConfig{
		ProjectDir: projectDir,
		TasksDir:   tasksDir,
	})

	memCtx, err := readerFacade.BuildContext(ctx, durable.MemoryScope{})
	if err != nil {
		t.Fatalf("BuildContext for session N+1: %v", err)
	}

	delivered := 0
	for i := 0; i < factCount; i++ {
		cli := writerCLIs[i%len(writerCLIs)]
		needle := fmt.Sprintf("durable fact %d emitted by session N under CLI %s", i, cli)
		if strings.Contains(memCtx.Block, needle) {
			delivered++
		}
	}

	recall := float64(delivered) / float64(factCount)
	if recall < 0.9 {
		t.Fatalf("cross-CLI handoff recall = %.2f (%d/%d), want >= 0.90 (RF-25, objective O-1)", recall, delivered, factCount)
	}
}
