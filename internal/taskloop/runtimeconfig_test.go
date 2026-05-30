package taskloop_test

import (
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

// TestBuildRuntimeConfig_TimeoutVazio verifica que Timeout vazio resulta em F1 (disabled).
func TestBuildRuntimeConfig_TimeoutVazio(t *testing.T) {
	t.Parallel()

	rc, err := taskloop.NewCatalog().BuildRuntimeConfig(config.Runtime{})
	if err != nil {
		t.Fatalf("erro inesperado com Runtime zero-value: %v", err)
	}
	if !rc.Timeout.Disabled() {
		t.Errorf("Timeout zero-value deveria ser disabled (F1), got %v", rc.Timeout)
	}
}

// TestBuildRuntimeConfig_TimeoutValido verifica mapeamento de string para ActivityTimeout.
func TestBuildRuntimeConfig_TimeoutValido(t *testing.T) {
	t.Parallel()

	rc, err := taskloop.NewCatalog().BuildRuntimeConfig(config.Runtime{Timeout: "30s"})
	if err != nil {
		t.Fatalf("erro inesperado com timeout=30s: %v", err)
	}
	if rc.Timeout.Disabled() {
		t.Error("Timeout 30s não deveria estar disabled")
	}
	if rc.Timeout.Duration() != 30*time.Second {
		t.Errorf("Timeout.Duration() = %v, want 30s", rc.Timeout.Duration())
	}
}

// TestBuildRuntimeConfig_TimeoutMalformado verifica fail-fast com mensagem descritiva.
func TestBuildRuntimeConfig_TimeoutMalformado(t *testing.T) {
	t.Parallel()

	_, err := taskloop.NewCatalog().BuildRuntimeConfig(config.Runtime{Timeout: "10x"})
	if err == nil {
		t.Fatal("esperava erro com timeout malformado, obteve nil")
	}
	got := err.Error()
	if len(got) == 0 {
		t.Fatal("mensagem de erro vazia")
	}
	// Deve conter "timeout inválido" para ser descritivo (R-ERR-001).
	if !contains(got, "timeout") && !contains(got, "invalid") {
		t.Errorf("mensagem de erro %q não é descritiva o suficiente", got)
	}
}

// TestBuildRuntimeConfig_NumericosMapeados1a1 verifica mapeamento 1:1 de numéricos.
func TestBuildRuntimeConfig_NumericosMapeados1a1(t *testing.T) {
	t.Parallel()

	rc, err := taskloop.NewCatalog().BuildRuntimeConfig(config.Runtime{
		MaxRetries:             3,
		RetryBackoffMultiplier: 1.5,
		Concurrent:             4,
		BatchSize:              2,
	})
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if rc.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", rc.MaxRetries)
	}
	if rc.RetryBackoffMultiplier != 1.5 {
		t.Errorf("RetryBackoffMultiplier = %f, want 1.5", rc.RetryBackoffMultiplier)
	}
	if rc.Concurrent != 4 {
		t.Errorf("Concurrent = %d, want 4", rc.Concurrent)
	}
	if rc.BatchSize != 2 {
		t.Errorf("BatchSize = %d, want 2", rc.BatchSize)
	}
}

// TestBuildRuntimeConfig_RegressaoF1 verifica que zero-value preserva comportamento F1.
// Concurrent e BatchSize são normalizados para 1 por ApplyDefaults (mínimo funcional).
func TestBuildRuntimeConfig_RegressaoF1(t *testing.T) {
	t.Parallel()

	rc, err := taskloop.NewCatalog().BuildRuntimeConfig(config.Runtime{})
	if err != nil {
		t.Fatalf("erro inesperado com Runtime zero-value: %v", err)
	}
	// ApplyDefaults normaliza <=0 para 1 (sequencial — F1).
	if rc.Concurrent != 1 {
		t.Errorf("Concurrent zero-value após ApplyDefaults = %d, want 1 (F1)", rc.Concurrent)
	}
	if rc.BatchSize != 1 {
		t.Errorf("BatchSize zero-value após ApplyDefaults = %d, want 1 (F1)", rc.BatchSize)
	}
	if rc.MaxRetries != 0 {
		t.Errorf("MaxRetries zero-value = %d, want 0 (F1)", rc.MaxRetries)
	}
	if rc.RetryBackoffMultiplier != 0 {
		t.Errorf("RetryBackoffMultiplier zero-value = %f, want 0 (F1)", rc.RetryBackoffMultiplier)
	}
	if !rc.Timeout.Disabled() {
		t.Errorf("Timeout zero-value deveria ser disabled (F1), got %v", rc.Timeout)
	}
}

// TestBuildRuntimeConfig_ApplyDefaultsNormalization verifica que Concurrent/BatchSize <=0
// são normalizados para 1 após BuildRuntimeConfig.
func TestBuildRuntimeConfig_ApplyDefaultsNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		concurrent     int
		batchSize      int
		wantConcurrent int
		wantBatchSize  int
	}{
		{
			name:           "negativo normalizado para 1",
			concurrent:     -2,
			batchSize:      -5,
			wantConcurrent: 1,
			wantBatchSize:  1,
		},
		{
			name:           "zero normalizado para 1",
			concurrent:     0,
			batchSize:      0,
			wantConcurrent: 1,
			wantBatchSize:  1,
		},
		{
			name:           "positivos preservados",
			concurrent:     3,
			batchSize:      4,
			wantConcurrent: 3,
			wantBatchSize:  4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, err := taskloop.NewCatalog().BuildRuntimeConfig(config.Runtime{
				Concurrent: tc.concurrent,
				BatchSize:  tc.batchSize,
			})
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if rc.Concurrent != tc.wantConcurrent {
				t.Errorf("Concurrent = %d, want %d", rc.Concurrent, tc.wantConcurrent)
			}
			if rc.BatchSize != tc.wantBatchSize {
				t.Errorf("BatchSize = %d, want %d", rc.BatchSize, tc.wantBatchSize)
			}
		})
	}
}

// TestBuildRuntimeConfig_TimeoutVariedades verifica vários formatos de duração válidos.
func TestBuildRuntimeConfig_TimeoutVariedades(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		timeout      string
		wantErr      bool
		wantDisabled bool
		wantDuration time.Duration
	}{
		{name: "vazio", timeout: "", wantDisabled: true},
		{name: "30s", timeout: "30s", wantDuration: 30 * time.Second},
		{name: "2m", timeout: "2m", wantDuration: 2 * time.Minute},
		{name: "1h30m", timeout: "1h30m", wantDuration: 90 * time.Minute},
		{name: "malformado-10x", timeout: "10x", wantErr: true},
		{name: "malformado-abc", timeout: "abc", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, err := taskloop.NewCatalog().BuildRuntimeConfig(config.Runtime{Timeout: tc.timeout})
			if tc.wantErr {
				if err == nil {
					t.Fatal("esperava erro, obteve nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if tc.wantDisabled {
				if !rc.Timeout.Disabled() {
					t.Errorf("Timeout deveria estar disabled, got %v", rc.Timeout)
				}
				return
			}
			if rc.Timeout.Duration() != tc.wantDuration {
				t.Errorf("Timeout.Duration() = %v, want %v", rc.Timeout.Duration(), tc.wantDuration)
			}
		})
	}
}

// TestBuildRuntimeConfig_ParidadeCLIs verifica que a mesma config.Runtime produz
// o mesmo RuntimeConfig (base de paridade entre as 4 CLIs — RIN-01).
func TestBuildRuntimeConfig_ParidadeCLIs(t *testing.T) {
	t.Parallel()

	cfg := config.Runtime{
		Timeout:                "60s",
		MaxRetries:             2,
		RetryBackoffMultiplier: 1.5,
		Concurrent:             3,
		BatchSize:              5,
	}

	// Simula construção para cada uma das 4 CLIs a partir da mesma config resolvida.
	drivers := []string{"claude", "codex", "copilot", "gemini"}
	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			t.Parallel()
			rc, err := taskloop.NewCatalog().BuildRuntimeConfig(cfg)
			if err != nil {
				t.Fatalf("driver %s: erro inesperado: %v", driver, err)
			}
			if rc.Timeout.Duration() != 60*time.Second {
				t.Errorf("driver %s: Timeout = %v, want 60s", driver, rc.Timeout.Duration())
			}
			if rc.MaxRetries != 2 {
				t.Errorf("driver %s: MaxRetries = %d, want 2", driver, rc.MaxRetries)
			}
			if rc.RetryBackoffMultiplier != 1.5 {
				t.Errorf("driver %s: RetryBackoffMultiplier = %f, want 1.5", driver, rc.RetryBackoffMultiplier)
			}
			if rc.Concurrent != 3 {
				t.Errorf("driver %s: Concurrent = %d, want 3", driver, rc.Concurrent)
			}
			if rc.BatchSize != 5 {
				t.Errorf("driver %s: BatchSize = %d, want 5", driver, rc.BatchSize)
			}
		})
	}
}

// contains verifica se s contém substr (case-insensitive helper para mensagens de erro).
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
