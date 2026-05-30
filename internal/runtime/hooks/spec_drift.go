// spec_drift.go — hook de guard de governança: spec-hash/drift + PRD-first (ADR-022).
// Registrado em PointRuntimePreOpen ao lado de GovernanceHook.
// Aborta a sessão ACP antes de c.Open quando:
//   - O hash do PRD/techspec diverge do hash registrado em tasks.md (RG-01).
//   - tasks.md não contém hash de PRD rastreável (RG-02, PRD-first enforce).
//
// No-op quando:
//   - RuntimePreOpenEvent.TasksDir == "" (uso ad-hoc/F1 preservado).
//   - Job.SkipDriftGuard == true (bypass explícito; --disable-hooks desabilita tudo).
//   - tasks.md ausente no TasksDir (não é sessão PRD-tracked; ex.: TasksDir usado só pela
//     memória 2-tier F3). No fluxo real de task-loop o tasks.md sempre existe, preservando RG-02.
package hooks

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/JailtonJunior94/ai-spec-harness/internal/specdrift"
)

// ErrSpecDrift é retornado quando o hash do PRD/techspec diverge do registrado em tasks.md.
// Acionável: executar `ai-spec sync-spec-hash` ou `ai-spec check-spec-drift` para corrigir.
var ErrSpecDrift = errors.New("spec-hash divergente: atualize o hash em tasks.md com 'ai-spec sync-spec-hash'")

// ErrPRDUntracked é retornado quando tasks.md não contém hash de PRD rastreável (PRD-first).
// Acionável: executar `ai-spec sync-spec-hash` para registrar o hash do PRD.
var ErrPRDUntracked = errors.New("PRD não rastreável: tasks.md não contém spec-hash-prd — execute 'ai-spec sync-spec-hash'")

// SpecDriftHook valida spec-hash/drift no ponto PointRuntimePreOpen.
// Lê TasksDir via RuntimePreOpenEvent; SkipDriftGuard desabilita apenas este hook.
type SpecDriftHook struct {
	// skipDriftGuard espelha Job.SkipDriftGuard: quando true, hook retorna nil imediatamente.
	skipDriftGuard bool
}

// NewSpecDriftHook cria um SpecDriftHook com a flag de bypass fornecida.
// skipDriftGuard deve ser propagado de Job.SkipDriftGuard pelo prepareHooksDispatcher.
func NewSpecDriftHook(skipDriftGuard bool) *SpecDriftHook {
	return &SpecDriftHook{skipDriftGuard: skipDriftGuard}
}

// Name retorna o identificador do hook.
func (h *SpecDriftHook) Name() string { return "spec_drift" }

// Run verifica spec-hash e PRD-first quando TasksDir não está vazio.
// Sequência:
//  1. Tipo errado → ignorar silenciosamente.
//  2. TasksDir == "" → no-op (uso ad-hoc/F1 preservado; zero-value seguro).
//  3. SkipDriftGuard → no-op (bypass explícito).
//  4. tasks.md ausente (os.ErrNotExist) → no-op; demais erros de leitura → erro de infraestrutura.
//  5. NoHashFound em qualquer hash → ErrPRDUntracked.
//  6. report.Pass == false → ErrSpecDrift.
func (h *SpecDriftHook) Run(_ context.Context, evt Event) error {
	preOpen, ok := evt.(RuntimePreOpenEvent)
	if !ok {
		// Ponto errado: ignorar silenciosamente.
		return nil
	}

	// No-op quando TasksDir não fornecido (uso ad-hoc/F1 preservado).
	if preOpen.TasksDir == "" {
		return nil
	}

	// No-op quando bypass explícito via SkipDriftGuard.
	if h.skipDriftGuard {
		return nil
	}

	report, err := specdrift.NewCatalog().CheckDrift(preOpen.TasksDir)
	if err != nil {
		// tasks.md ausente → não é uma sessão PRD-tracked: no-op.
		// TasksDir também é usado pela memória 2-tier (F3) sem exigir tasks.md; no fluxo real
		// de task-loop o tasks.md sempre existe (é parseado antes), então RG-02 segue garantido.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		// Demais erros de infraestrutura (permissão negada, tasks.md ilegível, etc.) → abortar.
		return fmt.Errorf("spec_drift: verificar drift em %s: %w", preOpen.TasksDir, err)
	}

	// RG-02: PRD-first — tasks.md deve conter hash rastreável.
	for _, h := range report.Hashes {
		if h.NoHashFound {
			return fmt.Errorf("%w: arquivo=%s dir=%s", ErrPRDUntracked, h.File, preOpen.TasksDir)
		}
	}

	// RG-01: spec-hash divergente — abortar antes de c.Open.
	if !report.Pass {
		return fmt.Errorf("%w: dir=%s", ErrSpecDrift, preOpen.TasksDir)
	}

	return nil
}
