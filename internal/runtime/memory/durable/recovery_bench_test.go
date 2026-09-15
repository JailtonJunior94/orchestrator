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

func BenchmarkRecoveryWith1000ActiveFactsInSinglePage(b *testing.B) {
	fsys := fs.NewFakeFileSystem()
	underlyingLayer := durable.NewLayerWithLocker(fsys, newInMemoryLayerLocker())
	facade := durable.NewFacadeWithLayer(fsys, underlyingLayer, durable.FacadeConfig{
		ProjectDir: "/bench-project",
		TasksDir:   "/bench-project/tasks",
	})

	const factCount = 1000
	scope := durable.Scope{Layer: durable.TargetLayerPRD, ProjectDir: "/bench-project", TasksDir: "/bench-project/tasks"}
	catalog := durable.NewCatalog()
	facts := make([]durable.Fact, 0, factCount)
	for i := 0; i < factCount; i++ {
		content := fmt.Sprintf("durable fact number %d accumulated for the recovery benchmark", i)
		key, err := catalog.DeriveSemanticKey(durable.StructuredSignal{Kind: "declared-section", Subject: fmt.Sprintf("task-%d.0.md", i)})
		if err != nil {
			b.Fatalf("derive semantic key %d: %v", i, err)
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
		b.Fatalf("seed %d facts into a single prd page: %v", factCount, err)
	}

	seeded, err := facade.BuildContext(ctx, durable.MemoryScope{})
	if err != nil {
		b.Fatalf("BuildContext after seeding: %v", err)
	}
	if seeded.FactsByLayer["prd"] != factCount {
		b.Fatalf("expected %d facts recovered from the single prd page, got %d (FactsByLayer=%v)", factCount, seeded.FactsByLayer["prd"], seeded.FactsByLayer)
	}

	durations := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		if _, err := facade.BuildContext(ctx, durable.MemoryScope{}); err != nil {
			b.Fatalf("BuildContext: %v", err)
		}
		durations = append(durations, time.Since(start))
	}
	b.StopTimer()

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p95Index := int(float64(len(durations)) * 0.95)
	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	}
	p95 := durations[p95Index]

	b.ReportMetric(float64(p95.Microseconds())/1000.0, "p95_ms")

	if p95 > 200*time.Millisecond {
		b.Fatalf("recovery p95 with %d active facts in a single page = %v, want < 200ms (RF-20)", factCount, p95)
	}
}
