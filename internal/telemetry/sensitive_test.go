package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindSensitiveField_Mutation(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantFound bool
		wantKey   string
	}{
		{"linha_limpa", "2026-01-01T00:00:00Z skill=bugfix ref=testing.md provider=claude", false, ""},
		{"api_key", "2026-01-01T00:00:00Z skill=x api_key=abc123", true, "api_key"},
		{"token_generico", "2026-01-01T00:00:00Z skill=x auth_token=abc", true, "auth_token"},
		{"secret", "2026-01-01T00:00:00Z skill=x client_secret=abc", true, "client_secret"},
		{"password", "2026-01-01T00:00:00Z skill=x password=abc", true, "password"},
		{"cookie", "2026-01-01T00:00:00Z skill=x cookie=abc", true, "cookie"},
		{"case_insensitive", "2026-01-01T00:00:00Z skill=x API_KEY=abc", true, "API_KEY"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			key, found := FindSensitiveField(tc.line)
			if found != tc.wantFound {
				t.Fatalf("FindSensitiveField(%q) found=%v, want %v", tc.line, found, tc.wantFound)
			}
			if found && key != tc.wantKey {
				t.Errorf("FindSensitiveField(%q) key=%q, want %q", tc.line, key, tc.wantKey)
			}
		})
	}
}

func TestNoSensitiveFieldsEmittedByProductionWriters(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	if err := NewCatalog().Log(dir, "execute-task", "go-implementation"); err != nil {
		t.Fatalf("Log: %v", err)
	}
	if err := NewCatalog().LogClaudeMetrics(dir, ClaudeSessionMetrics{
		CacheReadTokens: 100, CacheCreationTokens: 50, ThinkingTokens: 10, ToolCallsNormalizedCount: 3,
	}); err != nil {
		t.Fatalf("LogClaudeMetrics: %v", err)
	}
	if err := NewCatalog().LogPreconditionRejection(dir, "codex", "handshake_signal_not_received"); err != nil {
		t.Fatalf("LogPreconditionRejection: %v", err)
	}
	if err := NewCatalog().LogACPSession(dir, ACPSessionEvent{
		Runtime: "acp", Launcher: "binary", EventsCount: 10, UnknownEventsCount: 1,
		CancelReason: "none", SlowPublishes: 2, DroppedUpdates: 1, RetryAttempts: 1,
		Provider: "claude", Model: "claude-sonnet-5", Duration: 1000000000,
		ToolCallsTotal: 4, HasToolCallsTotal: true, FinalStatus: "completed",
	}); err != nil {
		t.Fatalf("LogACPSession: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ler log: %v", err)
	}

	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) == 0 {
		t.Fatal("log vazio: nenhum escritor de producao gravou nada")
	}
	for _, line := range lines {
		if key, found := FindSensitiveField(line); found {
			t.Errorf("campo sensivel %q emitido pela producao na linha: %s", key, line)
		}
	}
}
