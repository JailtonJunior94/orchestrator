//go:build !race

package durable_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

func TestBuildContextP95PreliminaryBenchmark_1000ActiveFactsInSinglePage(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	underlyingLayer := durable.NewLayerWithLocker(fsys, newInMemoryLayerLocker())
	facade := durable.NewFacadeWithLayer(fsys, underlyingLayer, durable.FacadeConfig{
		ProjectDir: "/project",
		TasksDir:   "/project/tasks",
	})

	const factCount = 1000
	scope := durable.Scope{Layer: durable.TargetLayerPRD, ProjectDir: "/project", TasksDir: "/project/tasks"}
	catalog := durable.NewCatalog()
	facts := make([]durable.Fact, 0, factCount)
	for i := 0; i < factCount; i++ {
		content := fmt.Sprintf("durable fact number %d for benchmark coverage", i)
		key, err := catalog.DeriveSemanticKey(durable.StructuredSignal{Kind: "declared-section", Subject: fmt.Sprintf("task-%d.0.md", i)})
		if err != nil {
			t.Fatalf("derive semantic key %d: %v", i, err)
		}
		facts = append(facts, durable.Fact{
			Identity:   durable.Identity{Key: key, Hash: catalog.HashContent(content)},
			Content:    content,
			Durability: durable.DurabilityPRD,
			State:      durable.FactStateActive,
		})
	}

	ctx := context.Background()
	if _, err := underlyingLayer.Consolidate(ctx, scope, facts); err != nil {
		t.Fatalf("seed %d facts into a single prd page: %v", factCount, err)
	}

	seeded, err := facade.BuildContext(ctx, durable.MemoryScope{})
	if err != nil {
		t.Fatalf("BuildContext after seeding: %v", err)
	}
	if seeded.FactsByLayer["prd"] != factCount {
		t.Fatalf("expected %d facts recovered from the single prd page, got %d (FactsByLayer=%v)", factCount, seeded.FactsByLayer["prd"], seeded.FactsByLayer)
	}

	const samples = 20
	durations := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		start := time.Now()
		if _, err := facade.BuildContext(ctx, durable.MemoryScope{}); err != nil {
			t.Fatalf("BuildContext: %v", err)
		}
		durations = append(durations, time.Since(start))
	}

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95Index := (samples*95+99)/100 - 1
	p95 := durations[p95Index]
	if p95 > buildContextP95Budget {
		t.Errorf("BuildContext p95 preliminary with %d active facts in a single page = %v, want < %v (RF-20)", factCount, p95, buildContextP95Budget)
	}
}
