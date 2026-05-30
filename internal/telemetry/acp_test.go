package telemetry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTelemetry_LegacyUnchanged verifica que o caminho legacy não emite campos novos de ACP.
// A função LogACPSession não deve ser chamada no caminho legacy; este teste valida
// que Log() (usado pelo caminho legacy) não escreve campos ACP.
func TestTelemetry_LegacyUnchanged(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	// Caminho legacy: usa Log() — sem campos ACP.
	if err := NewCatalog().Log(dir, "execute-task", "go-implementation"); err != nil {
		t.Fatalf("Log: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}

	content := string(data)

	// Campos ACP não devem aparecer no log do caminho legacy.
	for _, field := range []string{"runtime=", "launcher=", "events_count=", "unknown_events_count=", "cancel_reason="} {
		if strings.Contains(content, field) {
			t.Errorf("campo ACP %q não deve aparecer no log do caminho legacy; log:\n%s", field, content)
		}
	}

	// Campos legacy devem estar presentes.
	if !strings.Contains(content, "skill=execute-task") {
		t.Errorf("campo skill ausente no log legacy; log:\n%s", content)
	}
	if !strings.Contains(content, "ref=go-implementation") {
		t.Errorf("campo ref ausente no log legacy; log:\n%s", content)
	}
}

// TestTelemetry_ACPFields valida que LogACPSession emite os 5 campos corretos.
func TestTelemetry_ACPFields(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")
	dir := t.TempDir()

	evt := ACPSessionEvent{
		Runtime:            "acp",
		Launcher:           "binary",
		EventsCount:        42,
		UnknownEventsCount: 3,
		CancelReason:       "none",
	}

	if err := NewCatalog().LogACPSession(dir, evt); err != nil {
		t.Fatalf("LogACPSession: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("log file not created: %v", err)
	}

	content := string(data)

	// Todos os 5 campos ACP devem estar presentes com os valores corretos.
	checks := []struct {
		field string
		want  string
	}{
		{"runtime", "runtime=acp"},
		{"launcher", "launcher=binary"},
		{"events_count", "events_count=42"},
		{"unknown_events_count", "unknown_events_count=3"},
		{"cancel_reason", "cancel_reason=none"},
	}
	for _, c := range checks {
		if !strings.Contains(content, c.want) {
			t.Errorf("campo %s ausente ou incorreto; quero %q; log:\n%s", c.field, c.want, content)
		}
	}
}

// TestLogACPSession_NoWriteWhenDisabled verifica que LogACPSession não escreve
// quando GOVERNANCE_TELEMETRY != 1.
func TestLogACPSession_NoWriteWhenDisabled(t *testing.T) {
	os.Unsetenv("GOVERNANCE_TELEMETRY")
	dir := t.TempDir()

	evt := ACPSessionEvent{Runtime: "acp", Launcher: "binary"}
	if err := NewCatalog().LogACPSession(dir, evt); err != nil {
		t.Fatalf("LogACPSession: %v", err)
	}

	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("log file should not exist when GOVERNANCE_TELEMETRY != 1")
	}
}

// TestLogACPSession_AllCancelReasons verifica os 5 valores de cancel_reason.
func TestLogACPSession_AllCancelReasons(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	reasons := []string{"none", "activity_timeout", "context_canceled", "tool_error", "permission_denied"}
	for _, reason := range reasons {
		reason := reason
		t.Run(reason, func(t *testing.T) {
			dir := t.TempDir()
			evt := ACPSessionEvent{
				Runtime:      "acp",
				Launcher:     "npx",
				CancelReason: reason,
			}
			if err := NewCatalog().LogACPSession(dir, evt); err != nil {
				t.Fatalf("LogACPSession: %v", err)
			}

			logPath := filepath.Join(dir, ".agents", "telemetry.log")
			data, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatalf("log file not created: %v", err)
			}
			if !strings.Contains(string(data), "cancel_reason="+reason) {
				t.Errorf("cancel_reason=%s ausente no log; log:\n%s", reason, data)
			}
		})
	}
}
