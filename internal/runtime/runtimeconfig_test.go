package runtime_test

import (
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// TestRuntimeConfig_ApplyDefaults verifica que zero-values são normalizados
// para os valores funcionais mínimos preservando o comportamento F1.
func TestRuntimeConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  airuntime.RuntimeConfig
		wantC  int
		wantB  int
		wantTO bool // Timeout deve permanecer zero?
		wantMR int  // MaxRetries deve permanecer zero?
	}{
		{
			name:   "zero-value: Concurrent e BatchSize normalizados para 1",
			input:  airuntime.RuntimeConfig{},
			wantC:  1,
			wantB:  1,
			wantTO: true, // zero = desabilitado (F1)
			wantMR: 0,    // zero = uma tentativa (F1)
		},
		{
			name:   "valores positivos: preservados sem alteração",
			input:  airuntime.RuntimeConfig{Concurrent: 4, BatchSize: 2, MaxRetries: 3},
			wantC:  4,
			wantB:  2,
			wantTO: true,
			wantMR: 3,
		},
		{
			name:   "Concurrent negativo normalizado para 1",
			input:  airuntime.RuntimeConfig{Concurrent: -5, BatchSize: 1},
			wantC:  1,
			wantB:  1,
			wantTO: true,
			wantMR: 0,
		},
		{
			name:   "BatchSize negativo normalizado para 1",
			input:  airuntime.RuntimeConfig{Concurrent: 2, BatchSize: -3},
			wantC:  2,
			wantB:  1,
			wantTO: true,
			wantMR: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := tc.input
			cfg.ApplyDefaults()
			if cfg.Concurrent != tc.wantC {
				t.Errorf("Concurrent = %d, want %d", cfg.Concurrent, tc.wantC)
			}
			if cfg.BatchSize != tc.wantB {
				t.Errorf("BatchSize = %d, want %d", cfg.BatchSize, tc.wantB)
			}
			if tc.wantTO && cfg.Timeout != 0 {
				t.Errorf("Timeout = %v, want zero (disabled)", cfg.Timeout)
			}
			if cfg.MaxRetries != tc.wantMR {
				t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, tc.wantMR)
			}
		})
	}
}

// TestRuntimeConfig_TimeoutMapping verifica que RuntimeConfig.Timeout mapeia
// corretamente ActivityTimeout e que o zero-value = disabled (F1).
func TestRuntimeConfig_TimeoutMapping(t *testing.T) {
	t.Parallel()

	t.Run("zero-value Timeout = disabled", func(t *testing.T) {
		t.Parallel()
		cfg := airuntime.RuntimeConfig{}
		if !events.ActivityTimeout(cfg.Timeout).Disabled() {
			t.Errorf("Timeout zero-value deveria ser disabled, got %v", cfg.Timeout)
		}
	})

	t.Run("Timeout > 0 preservado no Job embutido", func(t *testing.T) {
		t.Parallel()
		to, err := events.NewActivityTimeout(30 * time.Second)
		if err != nil {
			t.Fatalf("NewActivityTimeout: %v", err)
		}
		job := airuntime.Job{
			RuntimeConfig: airuntime.RuntimeConfig{Timeout: to},
		}
		// O campo embutido Timeout é acessível diretamente.
		if job.Timeout != to {
			t.Errorf("job.Timeout = %v, want %v", job.Timeout, to)
		}
		if job.Timeout.Disabled() {
			t.Error("job.Timeout não deveria estar disabled com 30s")
		}
		if job.Timeout.Duration() != 30*time.Second {
			t.Errorf("job.Timeout.Duration() = %v, want 30s", job.Timeout.Duration())
		}
	})
}

// TestJob_RuntimeConfigEmbedded verifica que RuntimeConfig é embutido em Job
// e que os campos F2–F5 permanecem acessíveis e não são afetados pelo embedding.
func TestJob_RuntimeConfigEmbedded(t *testing.T) {
	t.Parallel()

	to, _ := events.NewActivityTimeout(10 * time.Second)
	job := airuntime.Job{
		Prompt:      "test",
		WorkDir:     "/tmp",
		EvidenceDir: "/tmp/ev",
		RuntimeConfig: airuntime.RuntimeConfig{
			Timeout:                to,
			MaxRetries:             2,
			RetryBackoffMultiplier: 1.5,
			Concurrent:             3,
			BatchSize:              5,
		},
		Quiet:    true,
		MCPNested: false,
	}

	// Campos de RuntimeConfig acessíveis via campo embutido.
	if job.Timeout != to {
		t.Errorf("job.Timeout = %v, want %v", job.Timeout, to)
	}
	if job.MaxRetries != 2 {
		t.Errorf("job.MaxRetries = %d, want 2", job.MaxRetries)
	}
	if job.Concurrent != 3 {
		t.Errorf("job.Concurrent = %d, want 3", job.Concurrent)
	}
	if job.BatchSize != 5 {
		t.Errorf("job.BatchSize = %d, want 5", job.BatchSize)
	}

	// Campos F2–F5 não afetados.
	if !job.Quiet {
		t.Error("job.Quiet deveria ser true")
	}
	if job.MCPNested {
		t.Error("job.MCPNested deveria ser false")
	}
	if job.Prompt != "test" {
		t.Errorf("job.Prompt = %q, want %q", job.Prompt, "test")
	}
}
