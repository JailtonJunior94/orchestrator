package telemetry

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Log registra uso de skill e referencia em .agents/telemetry.log apenas quando
// a variavel de ambiente GOVERNANCE_TELEMETRY estiver definida como "1".
func (c *Catalog) Log(rootDir, skill, ref string) error {
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
func (c *Catalog) LogClaudeMetrics(rootDir string, m ClaudeSessionMetrics) error {
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

func (c *Catalog) LogPreconditionRejection(rootDir, agentID, reason string) error {
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
		return fmt.Errorf("abrir log de telemetria (precondition rejection): %w", err)
	}
	defer f.Close()

	ts := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf("%s precondition.rejected agent=%s reason=%s\n", ts, agentID, reason)
	_, err = f.WriteString(line)
	return err
}
