package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Log registra uso de skill e referencia em .agents/telemetry.log apenas quando
// a variavel de ambiente GOVERNANCE_TELEMETRY estiver definida como "1".
func Log(rootDir, skill, ref string) error {
	if os.Getenv("GOVERNANCE_TELEMETRY") != "1" {
		return nil
	}

	logDir := filepath.Join(rootDir, ".agents")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("criar diretorio de telemetria: %w", err)
	}

	logPath := filepath.Join(logDir, "telemetry.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("abrir log de telemetria: %w", err)
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s skill=%s ref=%s\n", ts, skill, ref)
	_, err = f.WriteString(line)
	return err
}

// GeminiSessionMetrics é o conjunto de métricas Gemini-2026 de uma sessão ACP.
// Passado para LogGeminiMetrics após o encerramento da sessão.
type GeminiSessionMetrics struct {
	CacheReadTokens        int
	EffectiveContextTokens int
	PromptTokensBilled     int
	ThoughtsTokens         int
}

// LogGeminiMetrics registra entries de métricas Gemini-2026 em .agents/telemetry.log
// apenas quando GOVERNANCE_TELEMETRY=1 (opt-in, ADR-006) e cada valor > 0.
//
// Formato canônico por linha: "<ts> <chave>=<valor>" (sem JSON, ADR-006).
// Entries registradas quando valor > 0:
//   - gemini.cache_read=N
//   - gemini.effective_context=N
//   - gemini.prompt_billed=N
//   - gemini.thoughts=N
//
// Sem GOVERNANCE_TELEMETRY=1 ou quando todos os valores são zero: operação é no-op.
func LogGeminiMetrics(rootDir string, m GeminiSessionMetrics) error {
	if os.Getenv("GOVERNANCE_TELEMETRY") != "1" {
		return nil
	}

	// No-op quando todos valores são zero (evita poluição em relatórios de sessões não-Gemini).
	if m.CacheReadTokens == 0 &&
		m.EffectiveContextTokens == 0 &&
		m.PromptTokensBilled == 0 &&
		m.ThoughtsTokens == 0 {
		return nil
	}

	logDir := filepath.Join(rootDir, ".agents")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("criar diretorio de telemetria: %w", err)
	}

	logPath := filepath.Join(logDir, "telemetry.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("abrir log de telemetria (gemini metrics): %w", err)
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	type entry struct {
		key string
		val int
	}
	entries := []entry{
		{"gemini.cache_read", m.CacheReadTokens},
		{"gemini.effective_context", m.EffectiveContextTokens},
		{"gemini.prompt_billed", m.PromptTokensBilled},
		{"gemini.thoughts", m.ThoughtsTokens},
	}
	for _, e := range entries {
		if e.val > 0 {
			line := fmt.Sprintf("%s %s=%d\n", ts, e.key, e.val)
			if _, werr := f.WriteString(line); werr != nil {
				return fmt.Errorf("escrever entry telemetria gemini: %w", werr)
			}
		}
	}
	return nil
}

// ClaudeSessionMetrics é o conjunto de métricas Claude-2026 de uma sessão ACP.
// Passado para LogClaudeMetrics após o encerramento da sessão.
type ClaudeSessionMetrics struct {
	CacheReadTokens          int
	CacheCreationTokens      int
	ThinkingTokens           int
	ToolCallsNormalizedCount int
}

// LogClaudeMetrics registra 4 entries de métricas Claude-2026 em .agents/telemetry.log
// apenas quando GOVERNANCE_TELEMETRY=1 (opt-in, ADR-006).
//
// Formato canônico por linha: "<ts> <chave>=<valor>" (sem JSON, ADR-006).
// Entries registradas:
//   - claude.cache_read=N
//   - claude.cache_creation=N
//   - claude.thinking=N
//   - claude.normalized_tools=N
//
// Sem GOVERNANCE_TELEMETRY=1: operação é no-op; nenhum arquivo criado.
func LogClaudeMetrics(rootDir string, m ClaudeSessionMetrics) error {
	if os.Getenv("GOVERNANCE_TELEMETRY") != "1" {
		return nil
	}

	logDir := filepath.Join(rootDir, ".agents")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("criar diretorio de telemetria: %w", err)
	}

	logPath := filepath.Join(logDir, "telemetry.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("abrir log de telemetria (claude metrics): %w", err)
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	entries := []string{
		fmt.Sprintf("%s claude.cache_read=%d\n", ts, m.CacheReadTokens),
		fmt.Sprintf("%s claude.cache_creation=%d\n", ts, m.CacheCreationTokens),
		fmt.Sprintf("%s claude.thinking=%d\n", ts, m.ThinkingTokens),
		fmt.Sprintf("%s claude.normalized_tools=%d\n", ts, m.ToolCallsNormalizedCount),
	}
	for _, entry := range entries {
		if _, werr := f.WriteString(entry); werr != nil {
			return fmt.Errorf("escrever entry telemetria claude: %w", werr)
		}
	}
	return nil
}
