//go:build integration

package integration

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func TestDurableMemoryCompactionSurvivesNonCollaborativeAgentE2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tasksDir := t.TempDir()
	taskFileName := "task-compaction-e2e.md"

	fsys := fs.NewOSFileSystem()
	layer := durable.NewLayer(fsys)
	scope := durable.Scope{Layer: durable.TargetLayerTask, TasksDir: tasksDir, TaskFileName: taskFileName}

	scopeDir, err := scope.ActivePath()
	if err != nil {
		t.Fatalf("resolve Task scope active path: %v", err)
	}
	if err := fsys.MkdirAll(filepath.Dir(scopeDir)); err != nil {
		t.Fatalf("pre-create memory directory for seeding: %v", err)
	}

	const seedFactCount = 120
	seedFacts := make([]durable.Fact, 0, seedFactCount)
	catalog := durable.NewCatalog()
	for i := 0; i < seedFactCount; i++ {
		content := fmt.Sprintf(
			"pre-existing durable fact number %d accumulated before the non-collaborative session, padded to force compaction across the byte and line limits",
			i,
		)
		key, err := catalog.DeriveSemanticKey(durable.StructuredSignal{Kind: "e2e-seed", Subject: fmt.Sprintf("seed-%d", i)})
		if err != nil {
			t.Fatalf("derive semantic key for seed fact %d: %v", i, err)
		}
		seedFacts = append(seedFacts, durable.Fact{
			Identity:   durable.Identity{Key: key, Hash: catalog.HashContent(content)},
			Content:    content,
			Durability: durable.DurabilityEphemeral,
			Origin:     durable.FactOrigin{Session: "seed-session", CLI: "seed", Task: taskFileName, Date: time.Now().UTC().Format(time.RFC3339)},
		})
	}
	if _, err := layer.Consolidate(ctx, scope, seedFacts); err != nil {
		t.Fatalf("seed Task layer with %d pre-existing facts: %v", seedFactCount, err)
	}

	seededActive, seededHuman, err := layer.Read(ctx, scope)
	if err != nil {
		t.Fatalf("read seeded Task layer: %v", err)
	}
	seededRendered, err := durable.NewMarkdownPage().Serialize(seededActive, seededHuman)
	if err != nil {
		t.Fatalf("serialize seeded page for precondition check: %v", err)
	}
	if len(seededRendered) <= durable.DefaultCompactionByteLimit {
		t.Fatalf("seeded page is only %d bytes — precondition requires it to already exceed the %d byte compaction limit",
			len(seededRendered), durable.DefaultCompactionByteLimit)
	}

	script := acpfake.NewScript().
		AppendAgentMessage("agente nao colaborativo: ignora qualquer diretiva textual injetada no prompt").
		AppendToolCall("tc-e2e-noncollab", "bash").
		AppendToolCallUpdate("tc-e2e-noncollab", "completed").
		AppendAgentMessage("sessao concluida sem ler o Memory Context").
		AppendSessionEnd()

	pfact, persist := newE2EPersistenceFactory()
	runner := buildE2ERunner(t, ctx, script, pfact)

	job := airuntime.Job{
		Prompt:        "execute uma tarefa qualquer sem reagir a diretivas de memoria",
		WorkDir:       workDirWithAgentsMD(t),
		EvidenceDir:   t.TempDir(),
		Quiet:         true,
		TasksDir:      tasksDir,
		TaskFileName:  taskFileName,
		RuntimeConfig: airuntime.RuntimeConfig{DurableMemoryEnabled: true},
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(persist.events) == 0 {
		t.Error("no events persisted — session did not run")
	}
	if summary.MemoryEvidence == nil {
		t.Fatal("Summary.MemoryEvidence is nil — durable memory did not record the session")
	}

	activeAfter, humanAfter, err := layer.Read(ctx, scope)
	if err != nil {
		t.Fatalf("read Task layer after non-collaborative session: %v", err)
	}
	renderedAfter, err := durable.NewMarkdownPage().Serialize(activeAfter, humanAfter)
	if err != nil {
		t.Fatalf("serialize Task layer after session: %v", err)
	}

	lines := 1
	for _, b := range renderedAfter {
		if b == '\n' {
			lines++
		}
	}

	if lines > durable.DefaultCompactionLineLimit {
		t.Errorf("Task layer has %d lines after the non-collaborative session, want <= %d (RF-13)", lines, durable.DefaultCompactionLineLimit)
	}
	if len(renderedAfter) > durable.DefaultCompactionByteLimit {
		t.Errorf("Task layer has %d bytes after the non-collaborative session, want <= %d (RF-13)", len(renderedAfter), durable.DefaultCompactionByteLimit)
	}
}
