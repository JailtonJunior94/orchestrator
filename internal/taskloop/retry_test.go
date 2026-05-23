// Testes de retry/backoff para acpInvoker (ADR-018, RF-04, tarefa 4.0).
// Package taskloop (interno) para acessar withACPInvokerSleepFn e tipos não exportados.
package taskloop

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// ---- Erros de teste ---------------------------------------------------------

// errRetryTransient é um erro transitório de teste.
var errRetryTransient = errors.New("transient io error")

// ---- Fakes de retry ---------------------------------------------------------

// retryCountingClassifier implementa RetryClassifier contando chamadas.
type retryCountingClassifier struct {
	calls atomic.Int32
	fatal bool // quando true, IsTransient retorna false para qualquer erro não-nil
}

func (c *retryCountingClassifier) IsTransient(err error) bool {
	if err == nil {
		return false
	}
	c.calls.Add(1)
	if c.fatal {
		return false
	}
	return !errors.Is(err, airuntime.ErrPermissionDenied)
}

// retryFailingProber é um Prober que falha nas primeiras N chamadas.
type retryFailingProber struct {
	failCount int
	failErr   error
	calls     atomic.Int32
	launcher  specs.Launcher
}

func (p *retryFailingProber) EnsureAvailable(_ context.Context, _ specs.Spec) (specs.Launcher, error) {
	n := int(p.calls.Add(1))
	if n <= p.failCount {
		return specs.Launcher{}, p.failErr
	}
	return p.launcher, nil
}

// retryTestClientFactory cria clientes ACP fake para testes de retry.
type retryTestClientFactory struct {
	script *acpfake.Script
	ctx    context.Context
	t      *testing.T
}

func (f *retryTestClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	srv := acpfake.NewServer(f.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("retryTestClientFactory.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}

// retryTestPersistence implementa Persistence em memória.
type retryTestPersistence struct{}

func (p *retryTestPersistence) AppendEvent(_ events.Event) error                { return nil }
func (p *retryTestPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *retryTestPersistence) EnrichReport(_ airuntime.Summary) error          { return nil }

type retryTestPersistenceFactory struct{}

func (f *retryTestPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return &retryTestPersistence{}, nil
}

type retryDiscardRenderer struct{}

func (r *retryDiscardRenderer) Render(_ events.Event) {}

// workDirWithAgentsMDRetry cria TempDir com AGENTS.md para satisfazer o governance hook.
func workDirWithAgentsMDRetry(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("workDirWithAgentsMDRetry: %v", err)
	}
	return dir
}

// ---- Testes de retry --------------------------------------------------------

// TestACPInvoker_Retry_TransientError verifica que erro transitório é reexecutado até MaxRetries.
func TestACPInvoker_Retry_TransientError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Prober falha 2 vezes com erro transitório, depois sucede.
	prober := &retryFailingProber{
		failCount: 2,
		failErr:   errRetryTransient,
		launcher:  specs.NewBinaryLauncher("/fake/claude-agent-acp"),
	}

	script := acpfake.NewScript().AppendAgentMessage("ok").AppendSessionEnd()
	factory := &retryTestClientFactory{script: script, ctx: ctx, t: t}

	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(factory),
		airuntime.WithPersistenceFactory(&retryTestPersistenceFactory{}),
		airuntime.WithRenderer(&retryDiscardRenderer{}),
	)

	classifier := &retryCountingClassifier{fatal: false}
	var sleepCalls atomic.Int32
	sleepFn := func(_ context.Context, _ time.Duration) error {
		sleepCalls.Add(1)
		return nil // sem espera real
	}

	invoker := NewACPInvoker(runner, true, 0,
		WithACPInvokerMaxRetries(3),
		WithACPInvokerRetryBackoffMultiplier(0), // sem espera
		WithACPInvokerRetryClassifier(classifier),
		withACPInvokerSleepFn(sleepFn),
	)

	workDir := workDirWithAgentsMDRetry(t)
	_, _, exitCode, err := invoker.Invoke(ctx, "prompt", workDir, "")
	if err != nil {
		t.Fatalf("Invoke com retry: erro inesperado: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	// Classifier deve ter sido chamado pelo menos 2 vezes (2 falhas transitórias).
	if classifier.calls.Load() < 2 {
		t.Errorf("classifier.calls = %d, want >= 2", classifier.calls.Load())
	}
	// Sleep deve ter sido chamado 2 vezes (uma por reexecução).
	if sleepCalls.Load() < 2 {
		t.Errorf("sleepCalls = %d, want >= 2 (uma por retry)", sleepCalls.Load())
	}
}

// TestACPInvoker_Retry_FatalError verifica que ErrPermissionDenied não reexecuta.
func TestACPInvoker_Retry_FatalError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prober falha com erro fatal nas primeiras 3 chamadas.
	prober := &retryFailingProber{
		failCount: 3,
		failErr:   airuntime.ErrPermissionDenied,
		launcher:  specs.NewBinaryLauncher("/fake/claude-agent-acp"),
	}

	script := acpfake.NewScript().AppendSessionEnd()
	factory := &retryTestClientFactory{script: script, ctx: ctx, t: t}

	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(factory),
		airuntime.WithPersistenceFactory(&retryTestPersistenceFactory{}),
		airuntime.WithRenderer(&retryDiscardRenderer{}),
	)

	classifier := airuntime.NewRetryClassifier()
	var sleepCalls atomic.Int32
	sleepFn := func(_ context.Context, _ time.Duration) error {
		sleepCalls.Add(1)
		return nil
	}

	invoker := NewACPInvoker(runner, true, 0,
		WithACPInvokerMaxRetries(3),
		WithACPInvokerRetryBackoffMultiplier(0),
		WithACPInvokerRetryClassifier(classifier),
		withACPInvokerSleepFn(sleepFn),
	)

	workDir := workDirWithAgentsMDRetry(t)
	_, _, _, err := invoker.Invoke(ctx, "prompt", workDir, "")
	// Deve retornar erro (fatal) e não ter dormido (sem retry).
	if err == nil {
		t.Fatal("Invoke deveria retornar erro para falha fatal")
	}
	if sleepCalls.Load() != 0 {
		t.Errorf("sleepCalls = %d, want 0 (erro fatal não deve reexecutar)", sleepCalls.Load())
	}
	// Prober deve ter sido chamado apenas 1 vez (primeira tentativa, sem retry).
	if prober.calls.Load() != 1 {
		t.Errorf("prober.calls = %d, want 1 (sem retry para erro fatal)", prober.calls.Load())
	}
}

// TestACPInvoker_Retry_DefaultsSequential verifica que MaxRetries=0 executa exatamente uma vez.
func TestACPInvoker_Retry_DefaultsSequential(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().AppendAgentMessage("ok").AppendSessionEnd()
	prober := &retryFailingProber{
		failCount: 0,
		launcher:  specs.NewBinaryLauncher("/fake/claude-agent-acp"),
	}
	factory := &retryTestClientFactory{script: script, ctx: ctx, t: t}
	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(factory),
		airuntime.WithPersistenceFactory(&retryTestPersistenceFactory{}),
		airuntime.WithRenderer(&retryDiscardRenderer{}),
	)

	var sleepCalls atomic.Int32
	sleepFn := func(_ context.Context, _ time.Duration) error {
		sleepCalls.Add(1)
		return nil
	}

	// MaxRetries default (não configurado = 0): uma tentativa, sem retry.
	invoker := NewACPInvoker(runner, true, 0,
		withACPInvokerSleepFn(sleepFn),
	)

	workDir := workDirWithAgentsMDRetry(t)
	_, _, exitCode, err := invoker.Invoke(ctx, "prompt", workDir, "")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	// Nenhum sleep deve ter ocorrido (MaxRetries=0 sem retry).
	if sleepCalls.Load() != 0 {
		t.Errorf("sleepCalls = %d, want 0 (MaxRetries=0 não deve dormir)", sleepCalls.Load())
	}
	// Prober deve ter sido chamado exatamente 1 vez.
	if prober.calls.Load() != 1 {
		t.Errorf("prober.calls = %d, want 1 (sem retry)", prober.calls.Load())
	}
}

// TestACPInvoker_Retry_BackoffCalled verifica que sleepFn recebe durações crescentes.
func TestACPInvoker_Retry_BackoffCalled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prober falha 2 vezes com erro transitório.
	prober := &retryFailingProber{
		failCount: 2,
		failErr:   errRetryTransient,
		launcher:  specs.NewBinaryLauncher("/fake/claude-agent-acp"),
	}

	script := acpfake.NewScript().AppendAgentMessage("ok").AppendSessionEnd()
	factory := &retryTestClientFactory{script: script, ctx: ctx, t: t}

	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(factory),
		airuntime.WithPersistenceFactory(&retryTestPersistenceFactory{}),
		airuntime.WithRenderer(&retryDiscardRenderer{}),
	)

	sleepDurations := make([]time.Duration, 0, 3)
	sleepFn := func(_ context.Context, d time.Duration) error {
		sleepDurations = append(sleepDurations, d)
		return nil
	}

	invoker := NewACPInvoker(runner, true, 0,
		WithACPInvokerMaxRetries(3),
		WithACPInvokerRetryBaseDelay(10*time.Millisecond),
		WithACPInvokerRetryBackoffMultiplier(2.0),
		WithACPInvokerRetryClassifier(airuntime.NewRetryClassifier()),
		withACPInvokerSleepFn(sleepFn),
	)

	workDir := workDirWithAgentsMDRetry(t)
	_, _, _, err := invoker.Invoke(ctx, "prompt", workDir, "")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	// Deve ter dormido 2 vezes (2 falhas transitórias, depois sucesso).
	if len(sleepDurations) != 2 {
		t.Errorf("sleepDurations len = %d, want 2", len(sleepDurations))
		return
	}
	// Segunda espera deve ser >= primeira (backoff crescente).
	if sleepDurations[1] < sleepDurations[0] {
		t.Errorf("backoff não crescente: sleep[0]=%v sleep[1]=%v", sleepDurations[0], sleepDurations[1])
	}
}

// TestACPInvoker_Retry_LogsRetryAttempts é a regressão de M1: retry_attempts deve ser
// propagado ao Summary e surfacado na telemetria (ADR-018, RF-04). Antes da correção, a
// contagem era escrita em uma variável local descartada — invisível em qualquer artefato.
func TestACPInvoker_Retry_LogsRetryAttempts(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Prober falha 2 vezes (transitório), sucede na 3ª → 2 reexecuções.
	prober := &retryFailingProber{
		failCount: 2,
		failErr:   errRetryTransient,
		launcher:  specs.NewBinaryLauncher("/fake/claude-agent-acp"),
	}
	script := acpfake.NewScript().AppendAgentMessage("ok").AppendSessionEnd()
	factory := &retryTestClientFactory{script: script, ctx: ctx, t: t}
	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(factory),
		airuntime.WithPersistenceFactory(&retryTestPersistenceFactory{}),
		airuntime.WithRenderer(&retryDiscardRenderer{}),
	)
	invoker := NewACPInvoker(runner, true, 0,
		WithACPInvokerMaxRetries(3),
		WithACPInvokerRetryBackoffMultiplier(0),
		WithACPInvokerRetryClassifier(&retryCountingClassifier{}),
		withACPInvokerSleepFn(func(context.Context, time.Duration) error { return nil }),
	)

	workDir := workDirWithAgentsMDRetry(t)
	if _, _, _, err := invoker.Invoke(ctx, "prompt", workDir, ""); err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(workDir, ".agents", "telemetry.log"))
	if err != nil {
		t.Fatalf("ReadFile telemetry.log: %v", err)
	}
	if got := string(data); !strings.Contains(got, "retry_attempts=2") {
		t.Fatalf("telemetry.log deve conter retry_attempts=2; got: %q", got)
	}
}

// TestACPInvoker_NoRetry_OmitsRetryAttempts garante que retry_attempts=0 (sucesso na primeira
// tentativa) é omitido da telemetria — invariante F1 (sem poluição de logs).
func TestACPInvoker_NoRetry_OmitsRetryAttempts(t *testing.T) {
	t.Setenv("GOVERNANCE_TELEMETRY", "1")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prober := &retryFailingProber{failCount: 0, launcher: specs.NewBinaryLauncher("/fake/claude-agent-acp")}
	script := acpfake.NewScript().AppendAgentMessage("ok").AppendSessionEnd()
	factory := &retryTestClientFactory{script: script, ctx: ctx, t: t}
	runner := airuntime.NewACPRunner(
		specs.Claude(),
		airuntime.WithProber(prober),
		airuntime.WithClientFactory(factory),
		airuntime.WithPersistenceFactory(&retryTestPersistenceFactory{}),
		airuntime.WithRenderer(&retryDiscardRenderer{}),
	)
	invoker := NewACPInvoker(runner, true, 0,
		withACPInvokerSleepFn(func(context.Context, time.Duration) error { return nil }),
	)

	workDir := workDirWithAgentsMDRetry(t)
	if _, _, _, err := invoker.Invoke(ctx, "prompt", workDir, ""); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(workDir, ".agents", "telemetry.log"))
	if err != nil {
		t.Fatalf("ReadFile telemetry.log: %v", err)
	}
	if got := string(data); strings.Contains(got, "retry_attempts") {
		t.Fatalf("telemetry.log NÃO deve conter retry_attempts quando 0 (F1); got: %q", got)
	}
}
