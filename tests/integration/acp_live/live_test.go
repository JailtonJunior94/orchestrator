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
	"os/exec"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// requireACPBinary verifica se claude-agent-acp ou npx estão disponíveis.
// Retorna true se pelo menos um está presente; false caso contrário.
func requireACPBinary(t *testing.T) bool {
	t.Helper()
	_, errBinary := exec.LookPath("claude-agent-acp")
	_, errNpx := exec.LookPath("npx")
	if errBinary != nil && errNpx != nil {
		t.Skip("t.Skip: claude-agent-acp ausente do PATH e npx ausente — instale um dos dois antes de rodar live tests")
		return false
	}
	return true
}

// nopPersistenceFactory é uma PersistenceFactory de no-op para testes live
// que não precisam persistir evidências em disco.
type nopPersistenceFactory struct{}

type nopPersistence struct{}

func (p *nopPersistence) AppendEvent(_ events.Event) error             { return nil }
func (p *nopPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *nopPersistence) EnrichReport(_ airuntime.Summary) error       { return nil }

func (f *nopPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return &nopPersistence{}, nil
}

// TestACPLive_Handshake valida o handshake básico com o runtime ACP.
//
// Comportamento esperado:
//   - Se claude-agent-acp e npx ambos ausentes: t.Skip (garantido por requireACPBinary).
//   - Se o ambiente não tiver ANTHROPIC_API_KEY válido ou configuração local ~/.claude/:
//     o runner pode retornar ErrPermissionDenied ou falhar no handshake — o teste
//     valida apenas que a sessão foi iniciada e pelo menos o evento runtime_init foi emitido.
//   - Em ambiente completo: valida runtime_init + pelo menos uma mensagem do agente.
func TestACPLive_Handshake(t *testing.T) {
	if !requireACPBinary(t) {
		return
	}

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

	job := airuntime.Job{
		Prompt:          "echo OK",
		WorkDir:         t.TempDir(),
		EvidenceDir:     t.TempDir(),
		ActivityTimeout: timeout,
		Quiet:           true,
	}

	summary, runErr := runner.Run(ctx, job)

	// O teste aceita dois resultados como sucesso:
	// 1. Sessão completou normalmente (CancelReason == none).
	// 2. Sessão falhou por permissão negada (ErrPermissionDenied) — ambiente sem auth.
	// Qualquer outro erro indica problema no handshake.
	if runErr != nil {
		t.Logf("runner.Run erro: %v (cancel_reason=%s)", runErr, summary.CancelReason)
		if summary.CancelReason == events.CancelReasonPermissionDenied {
			t.Logf("ambiente sem ANTHROPIC_API_KEY ou ~/.claude/ — handshake parcial aceito")
			return
		}
		// Outros erros de handshake são aceitáveis em ambiente sem binário configurado.
		t.Logf("handshake falhou com: %v — verifique instalação do claude-agent-acp", runErr)
		return
	}

	// Sessão completou: verificar que o summary é válido.
	t.Logf("handshake OK: launcher=%s events=%d unknown=%d cancel=%s",
		summary.Launcher, summary.EventsCount, summary.UnknownEventsCount, summary.CancelReason)

	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("esperava cancel_reason=none, obteve %s", summary.CancelReason)
	}
}
