package events_test

import (
	"encoding/json"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// mustParseDriver cria DriverID ou falha o teste.
func mustParseDriver(t *testing.T, s string) specs.DriverID {
	t.Helper()
	d, err := specs.ParseDriverID(s)
	if err != nil {
		t.Fatalf("ParseDriverID(%q): %v", s, err)
	}
	return d
}

// ── ExtractorFor seleção ──────────────────────────────────────────────────────

func TestExtractorFor_ReturnsNullExtractor_ForCodex(t *testing.T) {
	d := mustParseDriver(t, "codex")
	ext := events.ExtractorFor(d)
	// nullExtractor deve retornar zero para qualquer payload.
	payload := json.RawMessage(`{"usage":{"cache_read_input_tokens":100}}`)
	m := ext.Extract(payload)
	if !m.IsZero() {
		t.Errorf("codex deve retornar MetricSet zero; got fields: %v", m.Fields())
	}
}

func TestExtractorFor_ReturnsNullExtractor_ForCopilot(t *testing.T) {
	d := mustParseDriver(t, "copilot")
	ext := events.ExtractorFor(d)
	payload := json.RawMessage(`{"usage":{"cache_read_input_tokens":100}}`)
	m := ext.Extract(payload)
	if !m.IsZero() {
		t.Errorf("copilot deve retornar MetricSet zero; got fields: %v", m.Fields())
	}
}

func TestExtractorFor_ReturnsDifferentExtractors_ForClaudeAndGemini(t *testing.T) {
	claude := events.ExtractorFor(mustParseDriver(t, "claude"))
	gemini := events.ExtractorFor(mustParseDriver(t, "gemini"))
	// Simplesmente garantir que são tipos distintos verificando comportamento diferente.
	claudePayload := json.RawMessage(`{"usage":{"cache_read_input_tokens":50,"thoughtTokens":10}}`)
	geminiPayload := json.RawMessage(`{"usage":{"cache_read_tokens":50,"thoughts_tokens":10}}`)

	cm := claude.Extract(claudePayload)
	gm := gemini.Extract(geminiPayload)

	if cm.IsZero() {
		t.Error("claude extractor deve extrair métricas do payload Claude")
	}
	if gm.IsZero() {
		t.Error("gemini extractor deve extrair métricas do payload Gemini")
	}
}

// ── claudeExtractor ───────────────────────────────────────────────────────────

func TestClaudeExtractor_FullPayload(t *testing.T) {
	d := mustParseDriver(t, "claude")
	ext := events.ExtractorFor(d)

	payload := json.RawMessage(`{
		"usage": {
			"cache_read_input_tokens": 150,
			"cache_creation_input_tokens": 300,
			"thoughtTokens": 42
		}
	}`)

	m := ext.Extract(payload)

	if m.IsZero() {
		t.Fatal("MetricSet não deve ser zero para payload com métricas")
	}
	if got := m.CacheReadTokens(); got != 150 {
		t.Errorf("CacheReadTokens: got %d, want 150", got)
	}
	if got := m.ThinkingTokens(); got != 42 {
		t.Errorf("ThinkingTokens: got %d, want 42", got)
	}
	extra := m.Extra()
	if got := extra["cache_creation_tokens"]; got != 300 {
		t.Errorf("extra[cache_creation_tokens]: got %d, want 300", got)
	}
}

func TestClaudeExtractor_PartialPayload_OnlyCacheRead(t *testing.T) {
	d := mustParseDriver(t, "claude")
	ext := events.ExtractorFor(d)

	payload := json.RawMessage(`{"usage":{"cache_read_input_tokens":75}}`)
	m := ext.Extract(payload)

	if m.IsZero() {
		t.Fatal("MetricSet não deve ser zero com apenas cache_read_input_tokens")
	}
	if got := m.CacheReadTokens(); got != 75 {
		t.Errorf("CacheReadTokens: got %d, want 75", got)
	}
}

func TestClaudeExtractor_AbsentUsage_ReturnsZero(t *testing.T) {
	d := mustParseDriver(t, "claude")
	ext := events.ExtractorFor(d)

	payload := json.RawMessage(`{"sessionUpdate":"agent_message"}`)
	m := ext.Extract(payload)

	if !m.IsZero() {
		t.Errorf("payload sem usage deve retornar MetricSet zero; got: %v", m.Fields())
	}
}

func TestClaudeExtractor_NilPayload_ReturnsZero(t *testing.T) {
	d := mustParseDriver(t, "claude")
	ext := events.ExtractorFor(d)

	m := ext.Extract(nil)
	if !m.IsZero() {
		t.Errorf("payload nil deve retornar MetricSet zero; got: %v", m.Fields())
	}
}

func TestClaudeExtractor_InvalidJSON_ReturnsZero(t *testing.T) {
	d := mustParseDriver(t, "claude")
	ext := events.ExtractorFor(d)

	m := ext.Extract(json.RawMessage(`{invalid json}`))
	if !m.IsZero() {
		t.Errorf("JSON inválido deve retornar MetricSet zero; got: %v", m.Fields())
	}
}

// ── geminiExtractor ───────────────────────────────────────────────────────────

func TestGeminiExtractor_FullPayload(t *testing.T) {
	d := mustParseDriver(t, "gemini")
	ext := events.ExtractorFor(d)

	payload := json.RawMessage(`{
		"usage": {
			"cache_read_tokens": 100,
			"effective_context_tokens": 200,
			"prompt_tokens_billed": 300,
			"thoughts_tokens": 40
		}
	}`)

	m := ext.Extract(payload)

	if m.IsZero() {
		t.Fatal("MetricSet não deve ser zero para payload Gemini com métricas")
	}
	if got := m.CacheReadTokens(); got != 100 {
		t.Errorf("CacheReadTokens: got %d, want 100", got)
	}
	if got := m.ThinkingTokens(); got != 40 {
		t.Errorf("ThinkingTokens (thoughts_tokens): got %d, want 40", got)
	}
	extra := m.Extra()
	if got := extra["effective_context_tokens"]; got != 200 {
		t.Errorf("extra[effective_context_tokens]: got %d, want 200", got)
	}
	if got := extra["prompt_tokens_billed"]; got != 300 {
		t.Errorf("extra[prompt_tokens_billed]: got %d, want 300", got)
	}
}

func TestGeminiExtractor_AbsentUsage_ReturnsZero(t *testing.T) {
	d := mustParseDriver(t, "gemini")
	ext := events.ExtractorFor(d)

	m := ext.Extract(json.RawMessage(`{"sessionUpdate":"agent_message"}`))
	if !m.IsZero() {
		t.Errorf("payload sem usage deve retornar MetricSet zero; got: %v", m.Fields())
	}
}

func TestGeminiExtractor_NilPayload_ReturnsZero(t *testing.T) {
	d := mustParseDriver(t, "gemini")
	ext := events.ExtractorFor(d)

	m := ext.Extract(nil)
	if !m.IsZero() {
		t.Errorf("payload nil deve retornar MetricSet zero; got: %v", m.Fields())
	}
}

// ── nullExtractor ─────────────────────────────────────────────────────────────

func TestNullExtractor_AlwaysReturnsZero(t *testing.T) {
	payloads := []json.RawMessage{
		nil,
		json.RawMessage(`{}`),
		json.RawMessage(`{"usage":{"cache_read_input_tokens":999}}`),
		json.RawMessage(`{"usage":{"cache_read_tokens":500}}`),
	}

	for _, driver := range []string{"codex", "copilot"} {
		d := mustParseDriver(t, driver)
		ext := events.ExtractorFor(d)
		for _, p := range payloads {
			m := ext.Extract(p)
			if !m.IsZero() {
				t.Errorf("nullExtractor(%s) deve retornar zero para qualquer payload; got: %v", driver, m.Fields())
			}
		}
	}
}

// ── Telemetria opt-in (RG-04/ADR-006) ─────────────────────────────────────────

// TestExtractor_ZeroIsZero valida que MetricSet zero-value satisfaz IsZero()==true.
// Garante que sessões sem métricas não disparam telemetria opt-in.
func TestExtractor_ZeroIsZero(t *testing.T) {
	var m events.MetricSet
	if !m.IsZero() {
		t.Error("zero-value MetricSet deve satisfazer IsZero()==true (RG-04/ADR-006)")
	}
}

// TestExtractor_MergeAssociative valida que Merge é associativo/acumulativo via extractor.
func TestExtractor_MergeAssociative(t *testing.T) {
	d := mustParseDriver(t, "claude")
	ext := events.ExtractorFor(d)

	p1 := json.RawMessage(`{"usage":{"cache_read_input_tokens":50}}`)
	p2 := json.RawMessage(`{"usage":{"cache_read_input_tokens":30,"thoughtTokens":10}}`)

	m1 := ext.Extract(p1)
	m2 := ext.Extract(p2)
	merged := m1.Merge(m2)

	if got := merged.CacheReadTokens(); got != 80 {
		t.Errorf("Merge CacheReadTokens: got %d, want 80", got)
	}
	if got := merged.ThinkingTokens(); got != 10 {
		t.Errorf("Merge ThinkingTokens: got %d, want 10", got)
	}
}
