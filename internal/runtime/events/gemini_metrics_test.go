package events_test

import (
	"encoding/json"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// T-36: TestExtractGeminiMetricsFromACPPayload — 5 cenários (RF-18, TD-02)

// cenário 1: payload vazio → zero-value silencioso
func TestExtractGeminiMetrics_EmptyPayload(t *testing.T) {
	m, err := events.NewCatalog().ExtractGeminiMetrics(json.RawMessage{})
	if err != nil {
		t.Fatalf("esperado err=nil para payload vazio; got: %v", err)
	}
	if m != (events.GeminiMetrics{}) {
		t.Errorf("esperado GeminiMetrics{} para payload vazio; got: %+v", m)
	}
}

// cenário 2: payload parcial (apenas cache_read_tokens)
func TestExtractGeminiMetrics_PartialPayload(t *testing.T) {
	raw := json.RawMessage(`{"usage":{"cache_read_tokens":50}}`)
	m, err := events.NewCatalog().ExtractGeminiMetrics(raw)
	if err != nil {
		t.Fatalf("esperado err=nil para payload parcial; got: %v", err)
	}
	if m.CacheReadTokens != 50 {
		t.Errorf("esperado CacheReadTokens=50; got: %d", m.CacheReadTokens)
	}
	if m.EffectiveContextTokens != 0 {
		t.Errorf("esperado EffectiveContextTokens=0; got: %d", m.EffectiveContextTokens)
	}
	if m.PromptTokensBilled != 0 {
		t.Errorf("esperado PromptTokensBilled=0; got: %d", m.PromptTokensBilled)
	}
	if m.ThoughtsTokens != 0 {
		t.Errorf("esperado ThoughtsTokens=0; got: %d", m.ThoughtsTokens)
	}
}

// cenário 3: payload completo com todos os campos
func TestExtractGeminiMetrics_FullPayload(t *testing.T) {
	raw := json.RawMessage(`{"usage":{"cache_read_tokens":100,"effective_context_tokens":200,"prompt_tokens_billed":300,"thoughts_tokens":40}}`)
	m, err := events.NewCatalog().ExtractGeminiMetrics(raw)
	if err != nil {
		t.Fatalf("esperado err=nil para payload completo; got: %v", err)
	}
	if m.CacheReadTokens != 100 {
		t.Errorf("CacheReadTokens: esperado 100, got %d", m.CacheReadTokens)
	}
	if m.EffectiveContextTokens != 200 {
		t.Errorf("EffectiveContextTokens: esperado 200, got %d", m.EffectiveContextTokens)
	}
	if m.PromptTokensBilled != 300 {
		t.Errorf("PromptTokensBilled: esperado 300, got %d", m.PromptTokensBilled)
	}
	if m.ThoughtsTokens != 40 {
		t.Errorf("ThoughtsTokens: esperado 40, got %d", m.ThoughtsTokens)
	}
}

// cenário 4: payload com chaves inesperadas — deve ser ignorado silenciosamente
func TestExtractGeminiMetrics_UnexpectedKeys(t *testing.T) {
	raw := json.RawMessage(`{"usage":{"cache_read_tokens":77,"unknown_future_field":999},"extra":"data"}`)
	m, err := events.NewCatalog().ExtractGeminiMetrics(raw)
	if err != nil {
		t.Fatalf("esperado err=nil para payload com chaves inesperadas; got: %v", err)
	}
	if m.CacheReadTokens != 77 {
		t.Errorf("CacheReadTokens: esperado 77, got %d", m.CacheReadTokens)
	}
	// Campos extras devem ser ignorados (zero-value nos campos conhecidos ausentes).
	if m.PromptTokensBilled != 0 {
		t.Errorf("PromptTokensBilled: esperado 0 (chave ausente), got %d", m.PromptTokensBilled)
	}
}

// cenário 5: JSON inválido → erro retornado
func TestExtractGeminiMetrics_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`{invalid json`)
	_, err := events.NewCatalog().ExtractGeminiMetrics(raw)
	if err == nil {
		t.Fatal("esperado err != nil para JSON inválido")
	}
}

// TestExtractGeminiMetricsForDriver — driver não-gemini retorna zero-value
func TestExtractGeminiMetricsForDriver_NonGeminiDriver(t *testing.T) {
	raw := json.RawMessage(`{"usage":{"cache_read_tokens":100,"effective_context_tokens":200}}`)
	for _, driver := range []string{"claude", "codex", "copilot", ""} {
		m := events.NewCatalog().ExtractGeminiMetricsForDriver(driver, raw)
		if m != (events.GeminiMetrics{}) {
			t.Errorf("driver=%q: esperado GeminiMetrics{} zero-value; got: %+v", driver, m)
		}
	}
}

// TestExtractGeminiMetricsForDriver — driver gemini extrai corretamente
func TestExtractGeminiMetricsForDriver_GeminiDriver(t *testing.T) {
	raw := json.RawMessage(`{"usage":{"cache_read_tokens":42,"prompt_tokens_billed":10}}`)
	m := events.NewCatalog().ExtractGeminiMetricsForDriver("gemini", raw)
	if m.CacheReadTokens != 42 {
		t.Errorf("CacheReadTokens: esperado 42, got %d", m.CacheReadTokens)
	}
	if m.PromptTokensBilled != 10 {
		t.Errorf("PromptTokensBilled: esperado 10, got %d", m.PromptTokensBilled)
	}
}
