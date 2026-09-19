package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRF44_RoundTripViaProductionParser(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	evt := ACPSessionEvent{
		Runtime:           "acp",
		Launcher:          "binary",
		CancelReason:      "none",
		RetryAttempts:     2,
		Provider:          "claude",
		Model:             "claude-sonnet-5",
		Duration:          1500 * time.Millisecond,
		ToolCallsTotal:    7,
		HasToolCallsTotal: true,
		FinalStatus:       "completed",
	}
	if err := NewCatalog().LogACPSession(dir, evt); err != nil {
		t.Fatalf("LogACPSession: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	entries, err := NewCatalog().parseLogEntries(logPath, 0)
	if err != nil {
		t.Fatalf("parseLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("esperava 1 entrada, obteve %d", len(entries))
	}
	e := entries[0]

	tests := []struct {
		metric string
		want   string
	}{
		{"provider", "claude"},
		{"model", "claude-sonnet-5"},
		{"duration", "1500"},
		{"tool_calls", "7"},
		{"final_status", "completed"},
		{"retries", "2"},
	}
	for _, tc := range tests {
		got, ok := e.Metric(tc.metric)
		if !ok {
			t.Errorf("metrica %q: esperava presente, esta ausente", tc.metric)
			continue
		}
		if got != tc.want {
			t.Errorf("metrica %q: quero %q, obteve %q", tc.metric, tc.want, got)
		}
	}
}

func TestRF44_AbsentMetricIsNeverZero(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	evt := ACPSessionEvent{
		Runtime:      "acp",
		Launcher:     "binary",
		CancelReason: "none",
	}
	if err := NewCatalog().LogACPSession(dir, evt); err != nil {
		t.Fatalf("LogACPSession: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	entries, err := NewCatalog().parseLogEntries(logPath, 0)
	if err != nil {
		t.Fatalf("parseLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("esperava 1 entrada, obteve %d", len(entries))
	}
	e := entries[0]

	for _, metric := range []string{"provider", "model", "duration", "tool_calls", "final_status", "retries"} {
		if v, ok := e.Metric(metric); ok {
			t.Errorf("metrica %q deveria estar ausente, mas retornou presente com valor %q", metric, v)
		}
	}

	report, err := NewCatalog().Report(dir, 0)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	for _, m := range report.Metrics {
		t.Errorf("ReportData.Metrics nao deveria conter nenhuma metrica RF-44 nesta janela; encontrou %q", m.Metric)
	}
}

func TestRF44_UnknownRawKeyNeverResolvesAsMetric(t *testing.T) {
	dir := t.TempDir()
	logDir := filepath.Join(dir, ".agents")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logPath := filepath.Join(logDir, "telemetry.log")
	line := time.Now().UTC().Format(time.RFC3339) +
		" skill=task-loop ref=acp-session tokens_in=999 tokens_out=111 context_loaded=42 tests_passed=true\n"
	if err := os.WriteFile(logPath, []byte(line), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	entries, err := NewCatalog().parseLogEntries(logPath, 0)
	if err != nil {
		t.Fatalf("parseLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("esperava 1 entrada, obteve %d", len(entries))
	}
	e := entries[0]

	for _, metric := range []string{"tokens_in", "tokens_out", "context_loaded", "tests_passed"} {
		if v, ok := e.Metric(metric); ok {
			t.Errorf("metrica %q sem escritor de producao identificado nunca deveria resolver; obteve %q", metric, v)
		}
	}
	if e.Skill != "task-loop" || e.Ref != "acp-session" {
		t.Errorf("skill/ref deveriam continuar reconhecidos: skill=%q ref=%q", e.Skill, e.Ref)
	}
}

func TestRF44_MetricsWithProductionWriterCoverage(t *testing.T) {
	all := RF44MetricNames()
	if len(all) != 15 {
		t.Fatalf("RF44MetricNames: esperava 15 nomes declarados no RF-44, obteve %d", len(all))
	}

	withWriter := RF44MetricsWithProductionWriter()
	wantWithWriter := map[string]bool{
		"provider": true, "model": true, "duration": true,
		"tool_calls": true, "final_status": true, "retries": true,
	}
	if len(withWriter) != len(wantWithWriter) {
		t.Fatalf("RF44MetricsWithProductionWriter: esperava %d nomes, obteve %d (%v)", len(wantWithWriter), len(withWriter), withWriter)
	}
	seen := make(map[string]bool, len(withWriter))
	for _, name := range withWriter {
		seen[name] = true
		if !wantWithWriter[name] {
			t.Errorf("metrica %q nao deveria ter escritor de producao identificado nesta entrega", name)
		}
	}
	for name := range wantWithWriter {
		if !seen[name] {
			t.Errorf("metrica %q deveria ter escritor de producao identificado", name)
		}
	}

	allNames := make(map[string]bool, len(all))
	for _, n := range all {
		allNames[n] = true
	}
	for _, n := range withWriter {
		if !allNames[n] {
			t.Errorf("metrica %q com escritor identificado nao pertence a lista canonica RF44MetricNames", n)
		}
	}
}

func TestRF44_ReportSummaryTrendSurfaceNewKey(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	evt := ACPSessionEvent{
		Runtime:           "acp",
		Launcher:          "binary",
		CancelReason:      "none",
		Provider:          "codex",
		Model:             "gpt-test",
		Duration:          2 * time.Second,
		ToolCallsTotal:    3,
		HasToolCallsTotal: true,
		FinalStatus:       "completed",
	}
	if err := NewCatalog().LogACPSession(dir, evt); err != nil {
		t.Fatalf("LogACPSession: %v", err)
	}

	report, err := NewCatalog().Report(dir, 0)
	if err != nil {
		t.Fatalf("Report: %v", err)
	}
	if !hasMetric(report.Metrics, "provider") {
		t.Errorf("Report.Metrics deveria conter 'provider'; obteve %+v", report.Metrics)
	}
	text := NewCatalog().FormatText(report)
	if !containsAll(text, "Métricas RF-44", "provider") {
		t.Errorf("FormatText deveria mencionar as metricas RF-44; texto:\n%s", text)
	}

	summaryText, err := NewCatalog().Summary(dir, 0)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if !containsAll(summaryText, "provider") {
		t.Errorf("Summary deveria mencionar 'provider'; texto:\n%s", summaryText)
	}

	trend, err := NewCatalog().Trend(dir)
	if err != nil {
		t.Fatalf("Trend: %v", err)
	}
	var totalObserved int
	for _, w := range trend.Weeks {
		totalObserved += w.MetricsObserved
	}
	if totalObserved == 0 {
		t.Errorf("Trend deveria refletir ao menos uma semana com MetricsObserved > 0; weeks=%+v", trend.Weeks)
	}
}

func hasMetric(metrics []MetricSummary, name string) bool {
	for _, m := range metrics {
		if m.Metric == name {
			return true
		}
	}
	return false
}

func containsAll(haystack string, needles ...string) bool {
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			return false
		}
	}
	return true
}
