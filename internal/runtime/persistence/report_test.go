package persistence_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
)

func makeSummary() runtime.Summary {
	return runtime.Summary{
		Launcher:           "binary",
		EventsCount:        42,
		UnknownEventsCount: 1,
		CancelReason:       events.CancelReasonNone,
	}
}

func TestEnrichReport_FreshAppend(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n\n## Resultado\n\n- status: done\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", makeSummary(), fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if !strings.Contains(content, "## Runtime ACP") {
		t.Error("seção '## Runtime ACP' não encontrada no relatório")
	}
	if !strings.Contains(content, "launcher: binary") {
		t.Error("campo 'launcher' não encontrado")
	}
	if !strings.Contains(content, "events_count: 42") {
		t.Error("campo 'events_count' não encontrado")
	}
	if !strings.Contains(content, "unknown_events_count: 1") {
		t.Error("campo 'unknown_events_count' não encontrado")
	}
	if !strings.Contains(content, "cancel_reason: none") {
		t.Error("campo 'cancel_reason' não encontrado")
	}
	// Conteúdo original preservado.
	if !strings.Contains(content, "## Resultado") {
		t.Error("seção original '## Resultado' foi removida")
	}
}

func TestEnrichReport_EmptyFile(t *testing.T) {
	fsys := fs.NewFakeFileSystem()

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", makeSummary(), fsys); err != nil {
		t.Fatalf("EnrichReport em arquivo inexistente: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	if !strings.Contains(string(data), "## Runtime ACP") {
		t.Error("seção '## Runtime ACP' não gerada em arquivo vazio")
	}
}

func TestEnrichReport_ReplaceExisting(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	initial := "# Relatório\n\n## Runtime ACP\n\n- runtime: acp\n- launcher: npx\n- events_count: 5\n- unknown_events_count: 0\n- cancel_reason: none\n\n## Outra Seção\n\n- conteúdo\n"
	if err := fsys.WriteFile("/report/execution_report.md", []byte(initial)); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := runtime.Summary{
		Launcher:           "binary",
		EventsCount:        99,
		UnknownEventsCount: 3,
		CancelReason:       events.CancelReasonActivityTimeout,
	}

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	// Verifica que o novo conteúdo foi aplicado.
	if !strings.Contains(content, "launcher: binary") {
		t.Error("campo 'launcher' não atualizado para binary")
	}
	if !strings.Contains(content, "events_count: 99") {
		t.Error("campo 'events_count' não atualizado")
	}
	if !strings.Contains(content, "cancel_reason: activity_timeout") {
		t.Error("campo 'cancel_reason' não atualizado")
	}
	// Verifica que lançador antigo foi removido.
	if strings.Contains(content, "launcher: npx") {
		t.Error("launcher antigo 'npx' ainda presente após substituição")
	}
	// Seção seguinte preservada.
	if !strings.Contains(content, "## Outra Seção") {
		t.Error("seção '## Outra Seção' foi removida indevidamente")
	}
}

func TestEnrichReport_Idempotency(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}
	summary := makeSummary()

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("primeira chamada: %v", err)
	}
	data1, _ := fsys.ReadFile("/report/execution_report.md")

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("segunda chamada: %v", err)
	}
	data2, _ := fsys.ReadFile("/report/execution_report.md")

	if string(data1) != string(data2) {
		t.Errorf("idempotência falhou:\napós 1ª chamada:\n%s\napós 2ª chamada:\n%s",
			string(data1), string(data2))
	}
}

func TestEnrichReport_WriteError(t *testing.T) {
	efs := &errFS{
		FakeFileSystem: fs.NewFakeFileSystem(),
		writeErr:       errors.New("disco cheio"),
	}
	err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", makeSummary(), efs)
	if err == nil {
		t.Fatal("esperava erro de WriteFile")
	}
}

// ── Métricas Claude-2026 ──────────────────────────────────────────────────────

func TestRenderClaudeMetricsSection_AllZero_ReturnsEmpty(t *testing.T) {
	summary := runtime.Summary{} // todos campos zero
	got := persistence.NewCatalog().RenderClaudeMetricsSection(summary)
	if got != "" {
		t.Errorf("esperado string vazia quando todos métricas zero; got: %q", got)
	}
}

func TestRenderClaudeMetricsSection_WithValues_ContainsAllFields(t *testing.T) {
	// ADR-021: RenderClaudeMetricsSection delega para RenderMetricsSection(summary.Metrics).
	// Testar via MetricSet com campos canônicos Claude (cache_read, thinking) + extra (cache_creation).
	m := events.NewMetricSet(0, 150, 42, map[string]int{"cache_creation_tokens": 300})
	summary := runtime.Summary{
		Metrics: m,
	}
	got := persistence.NewCatalog().RenderClaudeMetricsSection(summary)
	if got == "" {
		t.Fatal("esperado seção não-vazia quando métricas > 0")
	}
	for _, want := range []string{
		"Métricas Claude-2026",
		"cache_read_tokens",
		"cache_creation_tokens",
		"thinking_tokens",
		"150",
		"300",
		"42",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("seção não contém %q; got:\n%s", want, got)
		}
	}
}

func TestEnrichReport_ClaudeMetricsAppended_WhenNonZero(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	m := events.NewMetricSet(0, 150, 42, map[string]int{"cache_creation_tokens": 300})
	summary := runtime.Summary{
		Launcher:    "binary",
		EventsCount: 10,
		Metrics:     m,
	}

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if !strings.Contains(content, "Métricas Claude-2026") {
		t.Error("seção Métricas Claude-2026 não encontrada quando métricas > 0")
	}
	if !strings.Contains(content, "cache_read_tokens") {
		t.Error("campo cache_read_tokens não encontrado")
	}
}

func TestEnrichReport_ClaudeMetrics_NotPresent_WhenAllZero(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := makeSummary() // Metrics é zero-value (IsZero()==true)

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if strings.Contains(content, "Métricas Claude-2026") {
		t.Error("seção Métricas Claude-2026 não deve aparecer quando MetricSet.IsZero()==true")
	}
}

// ── Métricas unificadas (ADR-021) ─────────────────────────────────────────────

// TestRenderMetricsSection_AllZero valida que MetricSet zero-value não gera seção.
func TestRenderMetricsSection_AllZero(t *testing.T) {
	got := persistence.NewCatalog().RenderMetricsSection(events.MetricSet{})
	if got != "" {
		t.Errorf("esperado string vazia para MetricSet zero; got: %q", got)
	}
}

// TestRenderMetricsSection_ExtraFields valida campos extra por driver via MetricSet.Extra.
func TestRenderMetricsSection_ExtraFields(t *testing.T) {
	m := events.NewMetricSet(0, 100, 0, map[string]int{
		"effective_context_tokens": 200,
		"prompt_tokens_billed":     300,
	})
	got := persistence.NewCatalog().RenderMetricsSection(m)
	if got == "" {
		t.Fatal("esperado seção não-vazia para MetricSet com campos extra > 0")
	}
	for _, want := range []string{
		"Métricas Claude-2026",
		"cache_read_tokens",
		"effective_context_tokens",
		"prompt_tokens_billed",
		"100", "200", "300",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("seção não contém %q; got:\n%s", want, got)
		}
	}
}

// TestEnrichReport_MetricsSection_Codex valida que Codex (MetricSet zero) não emite seção.
func TestEnrichReport_MetricsSection_Codex(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := runtime.Summary{
		Launcher:    "binary",
		EventsCount: 5,
		// Metrics zero-value → nullExtractor → sem seção de métricas (RP-02).
	}

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if strings.Contains(content, "Métricas") {
		t.Error("seção de métricas não deve aparecer quando MetricSet.IsZero()==true (Codex/Copilot)")
	}
}

func TestEnrichReport_SectionAtEndOfFile(t *testing.T) {
	// Seção ACP é a última do arquivo (sem seção seguinte).
	fsys := fs.NewFakeFileSystem()
	initial := "# Relatório\n\n## Runtime ACP\n\n- runtime: acp\n- launcher: npx\n- events_count: 5\n- unknown_events_count: 0\n- cancel_reason: none\n"
	if err := fsys.WriteFile("/report/execution_report.md", []byte(initial)); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := runtime.Summary{
		Launcher:           "binary",
		EventsCount:        10,
		UnknownEventsCount: 0,
		CancelReason:       events.CancelReasonNone,
	}

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if !strings.Contains(content, "launcher: binary") {
		t.Error("launcher não atualizado")
	}
	if strings.Contains(content, "launcher: npx") {
		t.Error("launcher antigo npx ainda presente")
	}
	// Seção não deve aparecer duas vezes.
	count := strings.Count(content, "## Runtime ACP")
	if count != 1 {
		t.Errorf("## Runtime ACP aparece %d vezes, esperava 1", count)
	}
}

func makeMemoryEvidenceSummary() runtime.Summary {
	s := makeSummary()
	s.Metrics = events.NewMetricSet(1, 0, 0, nil)
	s.MemoryEvidence = &runtime.MemoryEvidence{
		SessionID:         "20260910T101010.000000000-1234",
		CLI:               "claude",
		TaskFileName:      "task-8.0.md",
		FactsByLayer:      map[string]int{"task": 2, "prd": 1},
		FactsOmitted:      1,
		FactsContradicted: 1,
		PagesUnreadable:   0,
		BudgetByLayer:     map[string]int{"task": 40, "prd": 30},
		WritesByLayer:     map[string]int{"task": 2},
		ArchivedByLayer:   map[string]int{"task": 1},
		Redactions:        1,
		Compactions:       1,
		Contradictions:    1,
		BatonClaimed:      true,
	}
	return s
}

func TestEnrichReport_MemoryEvidenceSection_InjectedBeforeMetrics(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := makeMemoryEvidenceSummary()

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if !strings.Contains(content, "## Evidência de Memória Durável") {
		t.Fatal("seção '## Evidência de Memória Durável' não encontrada")
	}
	if !strings.Contains(content, "session: 20260910T101010.000000000-1234") {
		t.Error("campo session não encontrado")
	}
	if !strings.Contains(content, "cli: claude") {
		t.Error("campo cli não encontrado")
	}
	if !strings.Contains(content, "write_facts_by_layer: task=2") {
		t.Error("campo write_facts_by_layer não encontrado")
	}
	if !strings.Contains(content, "budget_consumed_by_layer: prd=30, task=40") {
		t.Error("campo budget_consumed_by_layer não encontrado ou fora de ordem determinística")
	}
	if !strings.Contains(content, "baton_claimed: true") {
		t.Error("campo baton_claimed não encontrado")
	}

	evidenceIdx := strings.Index(content, "## Evidência de Memória Durável")
	metricsIdx := strings.Index(content, "## Métricas Claude-2026")
	if evidenceIdx == -1 || metricsIdx == -1 {
		t.Fatal("seções esperadas ausentes")
	}
	if evidenceIdx >= metricsIdx {
		t.Error("seção de evidência de memória deve vir antes da seção de métricas")
	}
	if strings.LastIndex(content, "## ") != strings.Index(content, "## Métricas Claude-2026") {
		t.Error("seção de métricas deve permanecer a última do relatório")
	}
}

func TestEnrichReport_MetricsThenMemoryEvidence_BothSectionsSurvive(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	metricsOnly := runtime.Summary{
		Launcher:    "binary",
		EventsCount: 10,
		Metrics:     events.NewMetricSet(0, 150, 42, nil),
	}

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", metricsOnly, fsys); err != nil {
		t.Fatalf("EnrichReport (métricas): %v", err)
	}

	withEvidence := makeMemoryEvidenceSummary()
	withEvidence.Metrics = metricsOnly.Metrics

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", withEvidence, fsys); err != nil {
		t.Fatalf("EnrichReport (métricas + evidência): %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if count := strings.Count(content, "## Métricas Claude-2026"); count != 1 {
		t.Errorf("## Métricas Claude-2026 aparece %d vezes, esperava 1", count)
	}
	if count := strings.Count(content, "## Evidência de Memória Durável"); count != 1 {
		t.Errorf("## Evidência de Memória Durável aparece %d vezes, esperava 1 (não deve ser apagada pela reinjeção de métricas)", count)
	}
	if !strings.Contains(content, "cache_read_tokens") {
		t.Error("campo cache_read_tokens da 1ª chamada não sobreviveu à 2ª chamada")
	}
	if !strings.Contains(content, "session: 20260910T101010.000000000-1234") {
		t.Error("campo session da seção de evidência não encontrado após reinjeção de métricas")
	}

	evidenceIdx := strings.Index(content, "## Evidência de Memória Durável")
	metricsIdx := strings.Index(content, "## Métricas Claude-2026")
	if evidenceIdx == -1 || metricsIdx == -1 {
		t.Fatal("seções esperadas ausentes")
	}
	if evidenceIdx >= metricsIdx {
		t.Error("seção de evidência de memória deve vir antes da seção de métricas")
	}
}

func TestEnrichReport_MemoryEvidenceSection_AbsentWhenNil(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", makeSummary(), fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if strings.Contains(content, "Evidência de Memória Durável") {
		t.Error("seção de evidência de memória não deve aparecer quando MemoryEvidence é nil")
	}
}

func TestEnrichReport_MemoryEvidenceSection_IdempotentAndLastSectionPreserved(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n\n## Riscos Residuais\n\n- nenhum\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := makeMemoryEvidenceSummary()

	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport (1a): %v", err)
	}
	if err := persistence.NewCatalog().EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport (2a): %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if count := strings.Count(content, "## Evidência de Memória Durável"); count != 1 {
		t.Errorf("## Evidência de Memória Durável aparece %d vezes, esperava 1", count)
	}
	if count := strings.Count(content, "## Métricas Claude-2026"); count != 1 {
		t.Errorf("## Métricas Claude-2026 aparece %d vezes, esperava 1", count)
	}
	if !strings.Contains(content, "## Riscos Residuais") {
		t.Error("seção original '## Riscos Residuais' foi removida")
	}
	if strings.LastIndex(content, "## ") != strings.Index(content, "## Métricas Claude-2026") {
		t.Error("seção de métricas deve permanecer a última do relatório após reexecução idempotente")
	}
}
