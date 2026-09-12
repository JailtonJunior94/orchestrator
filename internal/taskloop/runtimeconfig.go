package taskloop

import (
	"fmt"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

const defaultACPActivityTimeout = 2 * time.Minute

// BuildRuntimeConfig converte um config.Runtime já resolvido (via config.Resolver.Resolve)
// para um runtime.RuntimeConfig pronto para injeção em Job.
//
// Regras de mapeamento (ADR-025):
//   - Timeout string vazia → zero-value de events.ActivityTimeout (F1 preservado).
//   - Timeout não vazio → time.ParseDuration; erro descritivo em caso de malformação.
//   - MaxRetries, RetryBackoffMultiplier, Concurrent, BatchSize: mapeamento 1:1.
//   - ApplyDefaults() normaliza Concurrent/BatchSize ≤0 → 1 (F1 preservado).
//
// Zero-value de qualquer campo em resolved preserva o comportamento F1 (regressão zero).
func (c *Catalog) BuildRuntimeConfig(resolved config.Runtime) (airuntime.RuntimeConfig, error) {
	var timeout events.ActivityTimeout

	if resolved.Timeout != "" {
		d, err := time.ParseDuration(resolved.Timeout)
		if err != nil {
			return airuntime.RuntimeConfig{}, fmt.Errorf("timeout inválido: %w", err)
		}
		t, newErr := events.NewActivityTimeout(d)
		if newErr != nil {
			return airuntime.RuntimeConfig{}, fmt.Errorf("timeout inválido: %w", newErr)
		}
		timeout = t
	}

	var leaseTTL time.Duration
	if resolved.HandoffLeaseTTL != "" {
		d, err := time.ParseDuration(resolved.HandoffLeaseTTL)
		if err != nil {
			return airuntime.RuntimeConfig{}, fmt.Errorf("invalid handoff lease ttl: %w", err)
		}
		leaseTTL = d
	}

	rc := airuntime.RuntimeConfig{
		Timeout:                timeout,
		MaxRetries:             resolved.MaxRetries,
		RetryBackoffMultiplier: resolved.RetryBackoffMultiplier,
		Concurrent:             resolved.Concurrent,
		BatchSize:              resolved.BatchSize,
		MaxBugfixIterations:    resolved.MaxBugfixIterations,
		HandoffLeaseTTL:        leaseTTL,
		DurableMemoryEnabled:   resolved.DurableMemoryEnabled,
	}
	rc.ApplyDefaults()
	return rc, nil
}

// resolveRuntimeConfig resolve a configuração hierárquica e constrói o RuntimeConfig
// a ser injetado nos Jobs das 4 CLIs antes de runner.Run().
//
// Precedência (ADR-016 + ADR-025): flags CLI > workspace > global > defaults built-in.
// cwd é o diretório de trabalho para o upward-walk do resolver.
// flagsOverrides contém os valores de flags CLI; campos zero-value são ignorados.
//
// Zero-value em todas as camadas retorna o RuntimeConfig de DefaultRuntime (F1 exato).
func (c *Catalog) resolveRuntimeConfig(cwd string, flagsOverrides config.Runtime) (airuntime.RuntimeConfig, error) {
	resolver := config.NewDefaultResolver()
	resolved, err := resolver.Resolve(cwd, flagsOverrides)
	if err != nil {
		return airuntime.RuntimeConfig{}, fmt.Errorf("taskloop: resolver config hierárquica: %w", err)
	}
	// F1: aplicar o watchdog default (120s) quando nenhuma camada (flag/config) define timeout.
	// Mantém a precedência (flag/config vencem) e preserva "0s" explícito como desabilitado.
	if resolved.Timeout == "" {
		resolved.Timeout = defaultACPActivityTimeout.String()
	}
	rc, err := NewCatalog().BuildRuntimeConfig(resolved)
	if err != nil {
		return airuntime.RuntimeConfig{}, fmt.Errorf("taskloop: construir RuntimeConfig: %w", err)
	}
	return rc, nil
}

func (c *Catalog) applyResolvedMaxBugfixIterations(opts Options, resolved airuntime.RuntimeConfig) Options {
	if resolved.MaxBugfixIterations > 0 {
		opts.MaxBugfixIterations = resolved.MaxBugfixIterations
	}
	return opts
}

// optionsToConfigOverrides mapeia os campos de Options relevantes para config.Runtime,
// formando a camada de overrides das flags CLI (maior precedência na cascata ADR-016).
// Apenas campos não-zero são propagados; zero-value preserva a camada inferior (F1).
func (c *Catalog) optionsToConfigOverrides(opts Options) config.Runtime {
	var timeout string
	// Flag explícita vence (inclui --activity-timeout=0 → "0s" = desabilitar o watchdog).
	// Negativo é rejeitado na validação da CLI antes de chegar aqui.
	if opts.ActivityTimeoutSet && opts.ActivityTimeout >= 0 {
		timeout = opts.ActivityTimeout.String()
	}
	var maxBugfixIterations int
	if opts.MaxBugfixIterationsSet {
		maxBugfixIterations = opts.MaxBugfixIterations
	}
	var handoffLeaseTTL string
	if opts.HandoffLeaseTTLSet {
		handoffLeaseTTL = opts.HandoffLeaseTTL.String()
	}
	return config.Runtime{
		Timeout:                 timeout,
		Concurrent:              opts.Concurrent,
		BatchSize:               opts.BatchSize,
		MaxBugfixIterations:     maxBugfixIterations,
		HandoffLeaseTTL:         handoffLeaseTTL,
		DurableMemoryEnabled:    opts.DurableMemoryEnabled,
		DurableMemoryEnabledSet: opts.DurableMemoryEnabledSet,
	}
}
