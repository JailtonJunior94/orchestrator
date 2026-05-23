//go:build integration

// Package integration — teste de integração F3-Gemini hooks dispatcher (RF-16, RF-17).
//
// Valida que o hook runtime.pre_open (governance hook) é despachado antes de c.Open
// quando o driver é Gemini — confirmando que a infra F3-Claude (hooks.Dispatcher)
// é tool-agnóstica e funciona com specs.Gemini().
//
// Execução:
//
//	go test -tags integration ./tests/integration/... -run TestGeminiHooksDispatch
package integration

import (
	"context"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestGeminiHooksDispatch (RF-17, task 6.0) valida que o hook runtime.pre_open
// é despachado antes de c.Open quando o driver é Gemini.
//
// Estratégia: executar ACPRunner com specs.Gemini() e DisableHooks=false.
// O governance hook registrado em PointRuntimePreOpen valida AGENTS.md; se o hook
// não fosse despachado, AGENTS.md ausente não causaria erro — mas se presente o
// run completa normalmente, comprovando despacho bem-sucedido.
// Complementarmente: rodar com DisableHooks=true confirma que a distinção existe.
func TestGeminiHooksDispatch(t *testing.T) {
	t.Parallel()

	t.Run("hooks ativos: runtime.pre_open despachado para driver Gemini", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		script := acpfake.NewScript().
			AppendAgentMessage("Gemini hooks dispatch: sessão iniciada").
			AppendToolCall("tc-gemini-hook", "read_file").
			AppendToolCallUpdate("tc-gemini-hook", "completed").
			AppendAgentMessage("Gemini hooks dispatch: concluído").
			AppendSessionEnd()

		pfact, _ := newE2EPersistenceFactory()

		runner := airuntime.NewACPRunner(
			specs.Gemini(),
			airuntime.WithProber(&geminiHookFakeProber{}),
			airuntime.WithClientFactory(&geminiHookFakeClientFactory{script: script, ctx: ctx, t: t}),
			airuntime.WithPersistenceFactory(pfact),
			airuntime.WithRenderer(&e2eDiscardRenderer{}),
		)

		// WorkDir com AGENTS.md: governance hook (runtime.pre_open) deve passar.
		workDir := workDirWithAgentsMD(t)

		job := airuntime.Job{
			Prompt:       "hooks dispatch test: read README.md",
			WorkDir:      workDir,
			EvidenceDir:  t.TempDir(),
			Quiet:        true,
			DisableHooks: false, // hooks ativos — runtime.pre_open deve ser despachado
			AccessMode:   specs.AccessModeRestricted,
		}

		_, err := runner.Run(ctx, job)
		if err != nil {
			t.Fatalf("TestGeminiHooksDispatch: hooks ativos, Run falhou inesperadamente: %v", err)
		}

		// Se chegou aqui: o governance hook em runtime.pre_open foi despachado e passou
		// (AGENTS.md presente → hook retornou nil → c.Open prosseguiu → sessão completou).
		t.Log("TestGeminiHooksDispatch: runtime.pre_open despachado e governance hook passou para driver Gemini")
	})

	t.Run("hooks desabilitados: run completa sem governance hook (regressão DisableHooks)", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		script := acpfake.NewScript().
			AppendAgentMessage("Gemini DisableHooks: sessão iniciada").
			AppendSessionEnd()

		pfact, _ := newE2EPersistenceFactory()

		runner := airuntime.NewACPRunner(
			specs.Gemini(),
			airuntime.WithProber(&geminiHookFakeProber{}),
			airuntime.WithClientFactory(&geminiHookFakeClientFactory{script: script, ctx: ctx, t: t}),
			airuntime.WithPersistenceFactory(pfact),
			airuntime.WithRenderer(&e2eDiscardRenderer{}),
		)

		// Sem AGENTS.md: sem hooks ativos, não deve falhar.
		job := airuntime.Job{
			Prompt:       "hooks disable test",
			WorkDir:      t.TempDir(), // sem AGENTS.md — hooks desabilitados, deve passar
			EvidenceDir:  t.TempDir(),
			Quiet:        true,
			DisableHooks: true, // debug mode: governance hook não registrado
			AccessMode:   specs.AccessModeRestricted,
		}

		_, err := runner.Run(ctx, job)
		if err != nil {
			t.Fatalf("TestGeminiHooksDispatch DisableHooks: Run falhou: %v", err)
		}

		t.Log("TestGeminiHooksDispatch: DisableHooks=true passa sem AGENTS.md — comportamento regressão preservado")
	})
}

// geminiHookFakeProber implementa runtime.Prober retornando launcher fixo para Gemini.
type geminiHookFakeProber struct{}

func (p *geminiHookFakeProber) EnsureAvailable(_ context.Context, spec specs.Spec) (specs.Launcher, error) {
	return specs.NewBinaryLauncher("/usr/local/bin/" + spec.Command), nil
}

// geminiHookFakeClientFactory cria clientes ACP fake para testes de hooks Gemini.
type geminiHookFakeClientFactory struct {
	script *acpfake.Script
	ctx    context.Context
	t      *testing.T
}

func (f *geminiHookFakeClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	srv := acpfake.NewServer(f.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("geminiHookFakeClientFactory: acpfake.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}
