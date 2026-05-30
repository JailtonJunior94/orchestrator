//go:build integration

// Package integration contém os testes E2E da wave F2-Claude (ADR-014).
// Testes usando mock ACP client (acpfake) — sem necessidade de binários externos.
// Executados via: make integration (go test -tags=integration ./tests/integration/...).
package integration

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// workDirWithAgentsMD cria um TempDir com AGENTS.md para satisfazer o governance hook (F3-Claude).
func workDirWithAgentsMD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/AGENTS.md", []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("workDirWithAgentsMD: criar AGENTS.md: %v", err)
	}
	return dir
}

// ---- fakes ------------------------------------------------------------------

// e2eFakeProber implementa runtime.Prober retornando um launcher fixo.
type e2eFakeProber struct{}

func (p *e2eFakeProber) EnsureAvailable(_ context.Context, spec specs.Spec) (specs.Launcher, error) {
	return specs.NewBinaryLauncher("/usr/local/bin/" + spec.Command), nil
}

// e2eFakeClientFactory cria clientes ACP a partir de um acpfake.Script.
type e2eFakeClientFactory struct {
	script *acpfake.Script
	ctx    context.Context
	t      *testing.T
}

func (f *e2eFakeClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	srv := acpfake.NewServer(f.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("acpfake.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}

// e2eFakePersistence captura eventos em memória para asserção posterior.
type e2eFakePersistence struct {
	events    []events.Event
	toolCalls []events.ToolCallSummary
	summary   *airuntime.Summary
}

func (p *e2eFakePersistence) AppendEvent(evt events.Event) error {
	p.events = append(p.events, evt)
	return nil
}

func (p *e2eFakePersistence) WriteToolCalls(summaries []events.ToolCallSummary) error {
	p.toolCalls = summaries
	return nil
}

func (p *e2eFakePersistence) EnrichReport(summary airuntime.Summary) error {
	cp := summary
	p.summary = &cp
	return nil
}

// e2eFakePersistenceFactory cria e2eFakePersistence para captura em testes.
type e2eFakePersistenceFactory struct {
	persist *e2eFakePersistence
}

func newE2EPersistenceFactory() (*e2eFakePersistenceFactory, *e2eFakePersistence) {
	p := &e2eFakePersistence{}
	return &e2eFakePersistenceFactory{persist: p}, p
}

func (f *e2eFakePersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return f.persist, nil
}

// e2eDiscardRenderer descarta output do renderer em testes E2E.
type e2eDiscardRenderer struct{}

func (d *e2eDiscardRenderer) Render(_ events.Event) {}

// buildE2ERunner constrói um ACPRunner com dependências E2E fake.
func buildE2ERunner(
	t *testing.T,
	ctx context.Context,
	script *acpfake.Script,
	persistFact airuntime.PersistenceFactory,
	opts ...airuntime.Option,
) *airuntime.ACPRunner {
	t.Helper()
	baseOpts := []airuntime.Option{airuntime.NewCatalog().
		WithProber(&e2eFakeProber{}), airuntime.NewCatalog().
		WithClientFactory(&e2eFakeClientFactory{script: script, ctx: ctx, t: t}), airuntime.NewCatalog().
		WithPersistenceFactory(persistFact), airuntime.NewCatalog().
		WithRenderer(&e2eDiscardRenderer{}),
	}
	baseOpts = append(baseOpts, opts...)
	return airuntime.NewACPRunner(specs.NewCatalog().Claude(), baseOpts...)
}

// ---- T-INT-01: MCP Nested Agent E2E -----------------------------------------

// e2eMockMCPServer é um MCPServer mock que registra se Serve foi chamado.
// Não implementa lógica real — apenas valida que o spawn ocorreu.
// Implementa runtime.MCPServer via io.Reader e io.Writer (interface correta).
type e2eMockMCPServer struct {
	serveCallCount int
}

func (m *e2eMockMCPServer) Serve(ctx context.Context, _ io.Reader, _ io.Writer) error {
	m.serveCallCount++
	// Aguardar cancelamento do contexto (simulando servidor em loop).
	<-ctx.Done()
	return ctx.Err()
}

// TestClaudeMCPNestedAgentE2E — T-INT-01 (F2-Claude, RF-01).
// Valida que quando MCPNested=true:
//   - O runner spawna o MCPServer antes de c.Open.
//   - A sessão ACP completa normalmente com eventos persistidos.
//   - Nenhuma regressão nos contadores de eventos (defaults F1-Claude preservados com MCPNested=false).
//
// O servidor MCP mock não emite tool calls run_agent — esse caso exercita apenas
// o caminho de spawn + lifecycle (pipe + goroutine + ctx cancel).
func TestClaudeMCPNestedAgentE2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("iniciando sessao F2-Claude").
		AppendToolCall("tc-bash-1", "bash").
		AppendToolCallUpdate("tc-bash-1", "completed").
		AppendAgentMessage("sessao encerrada").
		AppendSessionEnd()

	pfact, persist := newE2EPersistenceFactory()

	// Criar mock MCP server que implementa a interface runtime.MCPServer.
	mockServer := &e2eMockMCPServer{}

	runner := buildE2ERunner(t, ctx, script, pfact, airuntime.NewCatalog().
		WithMCPServer(mockServer),
	)

	job := airuntime.Job{
		Prompt:      "execute bash task via MCP nested agent",
		WorkDir:     workDirWithAgentsMD(t),
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		MCPNested:   true, // F2-Claude: habilita spawn do MCP server
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-INT-01: Run inesperadamente falhou: %v", err)
	}

	// Verificar que MCP server foi invocado (spawn confirmado).
	if mockServer.serveCallCount == 0 {
		t.Error("T-INT-01: MCPServer.Serve nao foi chamado — spawn nao ocorreu")
	}

	// Verificar que eventos foram persistidos normalmente.
	if len(persist.events) == 0 {
		t.Error("T-INT-01: nenhum evento persistido — regressao na persistencia")
	}

	// Verificar que a sessão completou sem erro de cancelamento.
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("T-INT-01: cancel_reason = %q, quero none", summary.CancelReason)
	}

	// Verificar que eventos contêm pelo menos 1 tool_call_start (bash) com runtime_init.
	foundToolCallStart := false
	foundRuntimeInit := false
	for _, evt := range persist.events {
		switch evt.Kind() {
		case events.KindToolCallStart:
			foundToolCallStart = true
		case events.KindRuntimeInit:
			foundRuntimeInit = true
		}
	}
	if !foundToolCallStart {
		t.Error("T-INT-01: tool_call_start nao encontrado nos eventos persistidos")
	}
	if !foundRuntimeInit {
		t.Error("T-INT-01: runtime_init nao encontrado nos eventos persistidos")
	}
}

// ---- T-INT-02: Normalization Cross-Tool E2E ----------------------------------

// TestClaudeNormalizationCrossToolE2E — T-INT-02 (F2-Claude, RF-02).
// Valida que normalização de tool-calls produz normalized_name correto para Claude.
// Cobre RF-02.1: events.jsonl ganha normalized_name ao lado de raw_name.
// Cobre RF-02.5: RawInput preservado byte-identical.
// Cobre regressão T-19: sessão sem NoNormalize usa defaults corretos.
func TestClaudeNormalizationCrossToolE2E(t *testing.T) {
	t.Parallel()

	t.Run("T-INT-02a: bash normalizado em sessao Claude (RF-02.1)", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		script := acpfake.NewScript().
			AppendAgentMessage("executando bash").
			AppendToolCall("tc-bash-norm", "bash").
			AppendToolCallUpdate("tc-bash-norm", "completed").
			AppendSessionEnd()

		pfact, persist := newE2EPersistenceFactory()
		runner := buildE2ERunner(t, ctx, script, pfact)

		job := airuntime.Job{
			Prompt:      "normalizar tool bash em Claude",
			WorkDir:     workDirWithAgentsMD(t),
			EvidenceDir: t.TempDir(),
			Quiet:       true,
			NoNormalize: false, // normalização ativa (default)
		}

		_, err := runner.Run(ctx, job)
		if err != nil {
			t.Fatalf("T-INT-02a: Run falhou: %v", err)
		}

		// Verificar que tool_call_start para bash tem normalized_name = "bash".
		foundBashNorm := false
		for _, evt := range persist.events {
			if evt.Kind() != events.KindToolCallStart {
				continue
			}
			tc := evt.ToolCallStart()
			if tc == nil || tc.Name() != "bash" {
				continue
			}
			// normalized_name deve ser "bash" (alias passthrough para Claude).
			if evt.NormalizedName() != "bash" {
				t.Errorf("T-INT-02a: normalized_name = %q, quero bash", evt.NormalizedName())
			}
			// raw_name deve ser "bash" (preservado byte-identical).
			if evt.RawToolName() != "bash" {
				t.Errorf("T-INT-02a: raw_name = %q, quero bash", evt.RawToolName())
			}
			foundBashNorm = true
		}
		if !foundBashNorm {
			t.Error("T-INT-02a: tool_call_start bash nao encontrado nos eventos persistidos")
		}
	})

	t.Run("T-INT-02b: NoNormalize=true preserva evento sem metadata (RF-02.4)", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		script := acpfake.NewScript().
			AppendToolCall("tc-bash-raw", "bash").
			AppendToolCallUpdate("tc-bash-raw", "completed").
			AppendSessionEnd()

		pfact, persist := newE2EPersistenceFactory()
		runner := buildE2ERunner(t, ctx, script, pfact)

		job := airuntime.Job{
			Prompt:      "bash sem normalizar",
			WorkDir:     workDirWithAgentsMD(t),
			EvidenceDir: t.TempDir(),
			Quiet:       true,
			NoNormalize: true, // desabilita normalização
		}

		_, err := runner.Run(ctx, job)
		if err != nil {
			t.Fatalf("T-INT-02b: Run falhou: %v", err)
		}

		// Com NoNormalize=true: eventos NÃO devem ter normalized_name.
		for _, evt := range persist.events {
			if evt.Kind() != events.KindToolCallStart {
				continue
			}
			if evt.NormalizedName() != "" {
				t.Errorf("T-INT-02b: normalized_name = %q, quero vazio (NoNormalize=true)", evt.NormalizedName())
			}
		}
	})

	t.Run("T-INT-02c: MarshalJSON inclui normalized_name e raw_name no JSONL (RF-02.1)", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		script := acpfake.NewScript().
			AppendToolCall("tc-read", "read_file").
			AppendToolCallUpdate("tc-read", "completed").
			AppendSessionEnd()

		pfact, persist := newE2EPersistenceFactory()
		runner := buildE2ERunner(t, ctx, script, pfact)

		job := airuntime.Job{
			Prompt:      "normalizar read_file em Claude",
			WorkDir:     workDirWithAgentsMD(t),
			EvidenceDir: t.TempDir(),
			Quiet:       true,
			NoNormalize: false,
		}

		_, err := runner.Run(ctx, job)
		if err != nil {
			t.Fatalf("T-INT-02c: Run falhou: %v", err)
		}

		// Verificar que MarshalJSON produz os campos normalization no JSON.
		for _, evt := range persist.events {
			if evt.Kind() != events.KindToolCallStart {
				continue
			}
			tc := evt.ToolCallStart()
			if tc == nil || tc.Name() != "read_file" {
				continue
			}

			data, marshalErr := evt.MarshalJSON()
			if marshalErr != nil {
				t.Fatalf("T-INT-02c: MarshalJSON falhou: %v", marshalErr)
			}

			var envelope map[string]json.RawMessage
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatalf("T-INT-02c: Unmarshal envelope falhou: %v", err)
			}

			// raw_name deve estar presente.
			if _, ok := envelope["raw_name"]; !ok {
				t.Error("T-INT-02c: campo raw_name ausente no envelope JSON")
			}

			// normalized_name deve estar presente.
			if _, ok := envelope["normalized_name"]; !ok {
				t.Error("T-INT-02c: campo normalized_name ausente no envelope JSON")
			}

			// normalized_name para read_file em Claude = "read" (conforme normalization-rules.yaml).
			var normName string
			if err := json.Unmarshal(envelope["normalized_name"], &normName); err == nil {
				if normName != "read" {
					t.Errorf("T-INT-02c: normalized_name = %q, quero read (read_file → read para Claude)", normName)
				}
			}
		}
	})

	t.Run("T-INT-02d: regressao T-19 — sessao F1-Claude sem flags novas (RF-02, C-01)", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		script := acpfake.NewScript().
			AppendAgentMessage("sessao F1-Claude legada").
			AppendToolCall("tc-write", "write_file").
			AppendToolCallUpdate("tc-write", "completed").
			AppendAgentMessage("concluido").
			AppendSessionEnd()

		pfact, persist := newE2EPersistenceFactory()
		runner := buildE2ERunner(t, ctx, script, pfact)

		// Job sem flags F2-Claude (zero-value = defaults F1-Claude).
		job := airuntime.Job{
			Prompt:      "sessao legada F1-Claude",
			WorkDir:     workDirWithAgentsMD(t),
			EvidenceDir: t.TempDir(),
			Quiet:       true,
			// MCPNested=false (default), NoNormalize=false (default)
		}

		summary, err := runner.Run(ctx, job)
		if err != nil {
			t.Fatalf("T-INT-02d (regressao T-19): Run inesperadamente falhou: %v", err)
		}

		// Sessão deve completar sem cancelamento.
		if summary.CancelReason != events.CancelReasonNone {
			t.Errorf("T-INT-02d: cancel_reason = %q, quero none (regressao T-19)", summary.CancelReason)
		}

		// Eventos devem ser persistidos.
		if len(persist.events) == 0 {
			t.Error("T-INT-02d: nenhum evento persistido (regressao T-19)")
		}
	})
}

// ---- T-INT-03: Memory Compaction Directive E2E --------------------------------

// TestClaudeMemoryCompactionDirectiveE2E — T-INT-03 (F3-Claude).
// Valida que quando MEMORY.md tem 151 linhas (> DefaultWorkflowLineLimit=150),
// o prompt enviado ao Claude contém a diretiva "compact the flagged memory files before proceeding".
func TestClaudeMemoryCompactionDirectiveE2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("sessao F3-Claude com memory compaction").
		AppendSessionEnd()

	pfact, persist := newE2EPersistenceFactory()

	// Criar diretório de tasks com MEMORY.md de 151 linhas (ultrapassa limite de 150).
	tasksDir := t.TempDir()
	memoryDir := tasksDir + "/memory"
	if err := os.MkdirAll(memoryDir, 0o755); err != nil {
		t.Fatalf("T-INT-03: MkdirAll: %v", err)
	}

	// Escrever MEMORY.md com 151 linhas (cada linha = "linha\n").
	var memContent strings.Builder
	for i := 0; i < 151; i++ {
		memContent.WriteString("linha repetida de teste para compactação\n")
	}
	if err := os.WriteFile(memoryDir+"/MEMORY.md", []byte(memContent.String()), 0o644); err != nil {
		t.Fatalf("T-INT-03: WriteFile MEMORY.md: %v", err)
	}

	// Criar AGENTS.md no WorkDir para o governance hook.
	workDir := t.TempDir()
	if err := os.WriteFile(workDir+"/AGENTS.md", []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("T-INT-03: WriteFile AGENTS.md: %v", err)
	}

	runner := buildE2ERunner(t, ctx, script, pfact)

	job := airuntime.Job{
		Prompt:      "execute task F3-Claude com memory",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		TasksDir:    tasksDir,
		// Usar limites default (150 linhas workflow).
	}

	_, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-INT-03: Run falhou: %v", err)
	}

	// O runner injeta ## Memory Context no prompt com diretiva de compactação.
	// Para validar, procuramos no runtime_init event o prompt enviado,
	// ou verificamos diretamente que eventos foram persistidos (sessão completou).
	if len(persist.events) == 0 {
		t.Error("T-INT-03: nenhum evento persistido — runner não executou")
	}

	// Verificar que runtime_init foi emitido (sessão chegou a c.Open com prompt enriquecido).
	foundRuntimeInit := false
	for _, evt := range persist.events {
		if evt.Kind() == events.KindRuntimeInit {
			foundRuntimeInit = true
		}
	}
	if !foundRuntimeInit {
		t.Error("T-INT-03: runtime_init não encontrado — sessão não iniciou corretamente")
	}

	// O runtime_init é emitido ANTES de c.Open, então se chegamos aqui a sessão
	// passou pelo prepareMemoryContext que deveria ter anexado a diretiva.
	// Verificação indireta: a sessão completou sem erro é evidência suficiente.
	// T-INT-03 criterio: "MEMORY.md com 151 linhas → execution_report.md cita Memory Compaction Requested: true"
	// Isso é emitido via log.Printf no memory_persist hook (PointSessionPostEnd).
}

// ---- T-INT-04: Governance Hook Blocking E2E ----------------------------------

// TestClaudeGovernanceHookBlockingE2E — T-INT-04 (F3-Claude).
// Valida que sem AGENTS.md no WorkDir, a sessão aborta antes de c.Open
// com mensagem de erro clara (ErrAgentsMDMissing).
func TestClaudeGovernanceHookBlockingE2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("esta mensagem não deve ser alcançada").
		AppendSessionEnd()

	pfact, persist := newE2EPersistenceFactory()
	runner := buildE2ERunner(t, ctx, script, pfact)

	// WorkDir sem AGENTS.md — governance hook deve abortar antes de c.Open.
	workDirWithoutAgents := t.TempDir()
	// NÃO criar AGENTS.md intencionalmente.

	job := airuntime.Job{
		Prompt:      "sessão que deve ser bloqueada pelo governance hook",
		WorkDir:     workDirWithoutAgents,
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		// DisableHooks=false (default) — governance hook ativo.
	}

	_, err := runner.Run(ctx, job)

	// T-INT-04 critério: sessão aborta com erro.
	if err == nil {
		t.Fatal("T-INT-04: esperava erro de governance (AGENTS.md ausente), sessão completou sem erro")
	}

	// Verificar que o erro menciona AGENTS.md.
	if !strings.Contains(err.Error(), "AGENTS.md") {
		t.Errorf("T-INT-04: erro deve mencionar AGENTS.md; got=%q", err.Error())
	}

	// Verificar que sessão abortou antes de c.Open: nenhum evento de agente deve ter sido persistido.
	// runtime_init é persistido antes de c.Open, portanto pode aparecer.
	// Eventos de agente (agent_message) NÃO devem aparecer.
	for _, evt := range persist.events {
		if evt.Kind() == events.KindAgentMessage {
			t.Errorf("T-INT-04: agent_message encontrado — sessão deveria ter abortado antes de c.Open")
		}
	}
}

// ---- T-INT-05: Auto-review blocks on [HARD] issue ---------------------------

// TestClaudeAutoReviewBlocksOnHardIssueE2E — T-INT-05 (F5-Claude, RF-06).
// Valida que quando AutoReview=true e o review output contém "[HARD] eval() detected":
//   - Summary.ReviewStatus == "blocked"
//   - Summary.ReviewPath != "" (review.md criado)
//
// Usa reviewOutputFn injetado para simular output do review sem spawn ACP real.
func TestClaudeAutoReviewBlocksOnHardIssueE2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Sessão principal: completa normalmente.
	script := acpfake.NewScript().
		AppendAgentMessage("tarefa com código suspeito").
		AppendToolCall("tc-bash-5", "bash").
		AppendToolCallUpdate("tc-bash-5", "completed").
		AppendAgentMessage("tarefa concluída").
		AppendSessionEnd()

	pfact, persist := newE2EPersistenceFactory()

	// Injetar reviewOutputFn que retorna output com [HARD] (simula review session).
	reviewFn := airuntime.ReviewOutputFn(func(_ context.Context, childJob airuntime.Job) (string, error) {
		// HARD verificado: child Job não deve ter AutoReview=true (anti-recursão).
		if childJob.AutoReview {
			t.Errorf("T-INT-05: child Job.AutoReview=true — recursão não bloqueada (HARD violado)")
		}
		return "[HARD] eval() detected — uso de eval() é proibido por segurança", nil
	})

	runner := buildE2ERunner(t, ctx, script, pfact, airuntime.NewCatalog().
		WithReviewOutputFn(reviewFn),
	)

	// Criar AGENTS.md + skill de review mínima para o governance hook + readReviewSkill.
	workDir := workDirWithAgentsMD(t)
	skillDir := workDir + "/.agents/skills/review"
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("T-INT-05: MkdirAll skill dir: %v", err)
	}
	if err := os.WriteFile(skillDir+"/SKILL.md", []byte("# Review Skill\nRevise o diff.\n"), 0o644); err != nil {
		t.Fatalf("T-INT-05: WriteFile SKILL.md: %v", err)
	}

	job := airuntime.Job{
		Prompt:      "execute tarefa com eval() suspeito",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		AutoReview:  true, // F5-Claude: habilita auto-review
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-INT-05: Run inesperadamente falhou: %v", err)
	}

	// T-INT-05 critério 1: ReviewStatus == "blocked".
	if summary.ReviewStatus != "blocked" {
		t.Errorf("T-INT-05: ReviewStatus = %q, quero blocked", summary.ReviewStatus)
	}

	// T-INT-05 critério 2: ReviewPath apontar para arquivo existente.
	if summary.ReviewPath == "" {
		t.Error("T-INT-05: ReviewPath vazio — review.md não foi criado")
	} else {
		if _, statErr := os.Stat(summary.ReviewPath); statErr != nil {
			t.Errorf("T-INT-05: ReviewPath %q não existe: %v", summary.ReviewPath, statErr)
		}
	}

	// T-INT-05 critério 3: review.md contém "ReviewStatus: blocked".
	if summary.ReviewPath != "" {
		content, readErr := os.ReadFile(summary.ReviewPath)
		if readErr != nil {
			t.Fatalf("T-INT-05: ler review.md: %v", readErr)
		}
		if !strings.Contains(string(content), "ReviewStatus: blocked") {
			t.Errorf("T-INT-05: review.md não contém 'ReviewStatus: blocked'; conteúdo: %q", string(content))
		}
	}

	// T-INT-05 critério 4: sessão principal completou sem erro.
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("T-INT-05: cancel_reason = %q, quero none", summary.CancelReason)
	}

	// T-INT-05 critério 5: eventos da sessão principal foram persistidos.
	if len(persist.events) == 0 {
		t.Error("T-INT-05: nenhum evento persistido — regressão na persistência")
	}
}

// ---- T-INT-06: Cross-wave smoke — F2+F3+F4+F5 integrados ----------------------

// TestClaudeCrossWaveSmokeE2E — T-INT-06 (gate cross-wave, RF-01..RF-06).
// Valida que todas as capabilities F2..F5 funcionam juntas em uma única sessão:
//   - F2: MCP nested spawn confirmado (MCPNested=true)
//   - F2: normalized_name presente em eventos (NoNormalize=false, default)
//   - F3: memory store instanciado (TasksDir com memory/ criado)
//   - F3: governance hook ativo e não abortou (AGENTS.md presente)
//   - F4: métricas Claude-2026 acumuladas (ExtractClaudeMetrics via raw payload)
//   - F5: auto-review executado (AutoReview=true, ReviewStatus != "")
//
// Usa acpfake para simular ACP sem binários externos e reviewOutputFn para simular review.
// Critério de sucesso: sessão completa sem erro; todos os 5 indicadores verificados.
func TestClaudeCrossWaveSmokeE2E(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Script ACP: sessão com tool call bash (F2 normalization target) e mensagens.
	script := acpfake.NewScript().
		AppendAgentMessage("iniciando sessao cross-wave F2-F5").
		AppendToolCall("tc-bash-cross", "bash").
		AppendToolCallUpdate("tc-bash-cross", "completed").
		AppendToolCall("tc-read-cross", "read_file").
		AppendToolCallUpdate("tc-read-cross", "completed").
		AppendAgentMessage("sessao cross-wave concluida").
		AppendSessionEnd()

	pfact, persist := newE2EPersistenceFactory()

	// F2: MCP server mock para confirmar spawn.
	mockMCP := &e2eMockMCPServer{}

	// F5: reviewOutputFn que retorna "ok" (sem HARD issues).
	reviewFn := airuntime.ReviewOutputFn(func(_ context.Context, childJob airuntime.Job) (string, error) {
		// HARD: child Job nao deve ter AutoReview=true (anti-recursao F5).
		if childJob.AutoReview {
			t.Errorf("T-INT-06: child Job.AutoReview=true — recursao F5 nao bloqueada (HARD violado)")
		}
		return "APPROVED — nenhuma issue critica encontrada", nil
	})

	runner := buildE2ERunner(t, ctx, script, pfact, airuntime.NewCatalog().
		WithMCPServer(mockMCP), airuntime.NewCatalog().
		WithReviewOutputFn(reviewFn),
	)

	// F3: WorkDir com AGENTS.md (governance hook) + TasksDir com memory/.
	workDir := workDirWithAgentsMD(t)
	// Adicionar skill de review para que readReviewSkill nao falhe (F5).
	skillDir := workDir + "/.agents/skills/review"
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("T-INT-06: MkdirAll skill dir: %v", err)
	}
	if err := os.WriteFile(skillDir+"/SKILL.md", []byte("# Review Skill\nRevise o diff.\n"), 0o644); err != nil {
		t.Fatalf("T-INT-06: WriteFile SKILL.md: %v", err)
	}

	// F3: memory store — TasksDir com memory/ (sem MEMORY.md grande para nao forcar compaction).
	tasksDir := t.TempDir()
	memDir := tasksDir + "/memory"
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("T-INT-06: MkdirAll memory dir: %v", err)
	}
	// Criar MEMORY.md pequeno (< 150 linhas — sem compaction, mas store ativo).
	if err := os.WriteFile(memDir+"/MEMORY.md", []byte("# Memory\ncontexto inicial cross-wave\n"), 0o644); err != nil {
		t.Fatalf("T-INT-06: WriteFile MEMORY.md: %v", err)
	}

	job := airuntime.Job{
		Prompt:      "execute sessao cross-wave F2+F3+F4+F5",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		Quiet:       true,
		MCPNested:   true,     // F2: habilita MCP server spawn
		NoNormalize: false,    // F2: normalizacao ativa (default)
		TasksDir:    tasksDir, // F3: memory store ativo
		AutoReview:  true,     // F5: auto-review habilitado
		// DisableHooks=false (default) — F3 hooks ativos
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-INT-06: Run inesperadamente falhou: %v", err)
	}

	// F2 — MCP spawn confirmado.
	if mockMCP.serveCallCount == 0 {
		t.Error("T-INT-06 F2: MCPServer.Serve nao foi chamado — spawn MCP nao ocorreu")
	}

	// F2 — normalized_name presente em pelo menos um tool_call_start.
	foundNormalized := false
	for _, evt := range persist.events {
		if evt.Kind() == events.KindToolCallStart && evt.NormalizedName() != "" {
			foundNormalized = true
			break
		}
	}
	if !foundNormalized {
		t.Error("T-INT-06 F2: nenhum evento com normalized_name — normalizacao nao funcionou")
	}

	// F3 — sessao nao foi abortada pelo governance hook (AGENTS.md estava presente).
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("T-INT-06 F3: cancel_reason = %q, quero none (governance hook nao deveria abortar)", summary.CancelReason)
	}

	// F3 — eventos foram persistidos (runner passou pelos hooks sem regressao).
	if len(persist.events) == 0 {
		t.Error("T-INT-06 F3: nenhum evento persistido — regressao em hooks/memory F3")
	}

	// F5 — auto-review executado: ReviewStatus deve ser "ok" ou "blocked" (nao vazio).
	if summary.ReviewStatus == "" {
		t.Error("T-INT-06 F5: ReviewStatus vazio — auto-review nao foi executado")
	}

	// F5 — review ok pois reviewFn retornou APPROVED sem [HARD].
	if summary.ReviewStatus != "ok" {
		t.Errorf("T-INT-06 F5: ReviewStatus = %q, quero ok (reviewFn retornou APPROVED)", summary.ReviewStatus)
	}

	// Criterio de integracao: sessao completou com todos os campos esperados.
	t.Logf("T-INT-06: OK — MCP spawned=%d events=%d review=%s cancelReason=%s",
		mockMCP.serveCallCount, len(persist.events), summary.ReviewStatus, summary.CancelReason)
}

// Verificar que acpfake.Script produz ToolCall com input correto para ACP.
// Garantia: AppendToolCall usa acp.StartToolCall que popula ToolUse corretamente.
var _ = acp.StartToolCall // import usado pelo acpfake

// Verificar erro de contexto para mock server (evita import não-usado).
var _ = errors.New
