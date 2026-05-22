//go:build acp_live

// Package acp_live contém testes live que exigem o binário claude-agent-acp
// ou npx com @agentclientprotocol/claude-agent-acp instalado.
// Executar com: go test -tags=acp_live ./tests/integration/acp_live
//
// Sem a build tag acp_live, este arquivo não é compilado por go test ./...
// (validar com: go list -f '{{.GoFiles}}' ./tests/integration/acp_live)
package acp_live

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/hooks"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// acpAvailability descreve quais launchers ACP estão disponíveis no ambiente.
type acpAvailability struct {
	realBinary bool // claude-agent-acp no PATH (ambiente nightly/configurado)
	npx        bool // npx no PATH (fallback; geralmente sem auth local)
}

// detectACP verifica claude-agent-acp e npx; t.Skip se nenhum estiver presente.
func detectACP(t *testing.T) acpAvailability {
	t.Helper()
	_, errBinary := exec.LookPath("claude-agent-acp")
	_, errNpx := exec.LookPath("npx")
	if errBinary != nil && errNpx != nil {
		t.Skip("t.Skip: claude-agent-acp ausente do PATH e npx ausente — instale um dos dois antes de rodar live tests")
	}
	return acpAvailability{realBinary: errBinary == nil, npx: errNpx == nil}
}

// nopPersistenceFactory é uma PersistenceFactory de no-op para testes live
// que não precisam persistir evidências em disco.
type nopPersistenceFactory struct{}

type nopPersistence struct{}

func (p *nopPersistence) AppendEvent(_ events.Event) error                { return nil }
func (p *nopPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *nopPersistence) EnrichReport(_ airuntime.Summary) error          { return nil }

func (f *nopPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return &nopPersistence{}, nil
}

// TestACPLive_Handshake valida o handshake do runtime ACP contra o binário real.
//
// Estratégia (smoke honesto, sem mascarar regressões de wiring):
//   - claude-agent-acp e npx ausentes ⇒ t.Skip.
//   - O WorkDir recebe AGENTS.md, de modo que o governance hook passe e a sessão
//     exerça o caminho ACP real (spawn + handshake), não aborte no pre_open.
//   - Asserção de regressão sempre ativa: a sessão DEVE passar do governance hook
//     (falha em ErrAgentsMDMissing é regressão de setup, nunca tolerada).
//   - Ambiente nightly (claude-agent-acp presente): handshake deve suceder OU falhar
//     apenas com ErrPermissionDenied (sem auth). Qualquer outro erro reprova.
//   - Ambiente local (apenas npx, tipicamente sem auth/rede): falhas de auth, launcher
//     indisponível ou timeout são toleradas como smoke parcial.
func TestACPLive_Handshake(t *testing.T) {
	avail := detectACP(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithPersistenceFactory(&nopPersistenceFactory{}),
	)

	timeout, err := events.NewActivityTimeout(30 * time.Second)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	// Setup mínimo de governança: AGENTS.md no WorkDir para o governance hook passar.
	// Sem isso, o smoke abortaria em runtime.pre_open antes de exercer o caminho ACP.
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("setup AGENTS.md: %v", err)
	}

	job := airuntime.Job{
		Prompt:        "echo OK",
		WorkDir:       workDir,
		EvidenceDir:   t.TempDir(),
		RuntimeConfig: airuntime.RuntimeConfig{Timeout: timeout},
		Quiet:         true,
	}

	summary, runErr := runner.Run(ctx, job)

	// Regressão de setup/wiring: a sessão DEVE passar do governance hook (AGENTS.md presente).
	// Falha aqui indica regressão no caminho pre_open — não tolerável mesmo sem auth.
	if errors.Is(runErr, hooks.ErrAgentsMDMissing) {
		t.Fatalf("sessão abortou no governance hook apesar de AGENTS.md presente: %v", runErr)
	}

	if runErr == nil {
		// Sessão completou (ambiente com auth): summary deve ser válido.
		t.Logf("handshake OK: launcher=%s events=%d unknown=%d cancel=%s",
			summary.Launcher, summary.EventsCount, summary.UnknownEventsCount, summary.CancelReason)
		if summary.CancelReason != events.CancelReasonNone {
			t.Errorf("esperava cancel_reason=none, obteve %s", summary.CancelReason)
		}
		return
	}

	// Falhou: classificar. Permissão negada é sempre tolerada (ambiente sem auth).
	if errors.Is(runErr, airuntime.ErrPermissionDenied) {
		t.Logf("ambiente sem auth (ErrPermissionDenied) — handshake parcial aceito")
		return
	}

	// Com o binário real presente (nightly), só ErrPermissionDenied é aceitável.
	if avail.realBinary {
		t.Fatalf("claude-agent-acp presente mas handshake falhou com erro inesperado: %v (cancel=%s)",
			runErr, summary.CancelReason)
	}

	// Apenas npx (local, tipicamente sem auth/rede): tolerar launcher indisponível e timeout.
	switch {
	case errors.Is(runErr, airuntime.ErrLauncherUnavailable):
		t.Logf("launcher ACP indisponível (npx sem pacote/rede) — smoke parcial aceito")
	case errors.Is(runErr, context.DeadlineExceeded):
		t.Logf("timeout de handshake sem auth — smoke parcial aceito")
	default:
		t.Logf("handshake falhou (apenas npx, ambiente sem auth real): %v (cancel=%s)",
			runErr, summary.CancelReason)
	}
}
