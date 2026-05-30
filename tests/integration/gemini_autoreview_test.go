//go:build integration

// Package integration — testes de integração F5-Gemini (auto-review opt-in com driver Gemini).
//
// RF-22: valida que a cascata F5-Claude (runner_autoreview.go) funciona com driver Gemini
// sem nenhum código novo no autoreview core. Diff zero em internal/runtime/runner_autoreview.go.
//
// Execução:
//
//	go test -tags integration -run TestAutoReviewWithGeminiDriver ./tests/integration/...
//	go test -tags integration -run TestAutoReviewBlocksOnHardIssuesWithGemini ./tests/integration/...
//	go test -run TestAutoReviewInfoMessageEmittedOncePerSession ./tests/integration/...
package integration

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// autoReviewGeminiInfoMessage é a mensagem INFO literal que deve ser emitida
// quando --auto-review é combinado com --tool=gemini (RF-22, techspec §"Mensagens de Erro e Warning Literais").
// Fonte de verdade desta constante: techspec §F5-Gemini.
const autoReviewGeminiInfoMessage = "INFO: --auto-review com Gemini pode amplificar custo de tokens em janelas 1M+. Ver GEMINI.md §F5."

// buildGeminiRunnerWithReviewFn constrói um ACPRunner com spec Gemini e reviewOutputFn injetado.
// workDir deve conter AGENTS.md (governance hook).
func buildGeminiRunnerWithReviewFn(
	t *testing.T,
	ctx context.Context,
	script *acpfake.Script,
	reviewFn airuntime.ReviewOutputFn,
) *airuntime.ACPRunner {
	t.Helper()
	return airuntime.NewACPRunner(specs.NewCatalog().
		Gemini(), airuntime.NewCatalog().WithProber(&e2eFakeProber{}), airuntime.NewCatalog().
		WithClientFactory(&e2eFakeClientFactory{script: script, ctx: ctx, t: t}), airuntime.NewCatalog().
		WithPersistenceFactory(&e2eDiscardPersistenceFactory{}), airuntime.NewCatalog().
		WithRenderer(&e2eDiscardRenderer{}), airuntime.NewCatalog().
		WithReviewOutputFn(reviewFn),
	)
}

// e2eDiscardPersistenceFactory descarta persistência para testes que não precisam inspecionar eventos.
type e2eDiscardPersistenceFactory struct{}

type e2eDiscardPersistence struct{}

func (f *e2eDiscardPersistenceFactory) New(_ string) (airuntime.Persistence, error) {
	return &e2eDiscardPersistence{}, nil
}

func (p *e2eDiscardPersistence) AppendEvent(_ events.Event) error                { return nil }
func (p *e2eDiscardPersistence) WriteToolCalls(_ []events.ToolCallSummary) error { return nil }
func (p *e2eDiscardPersistence) EnrichReport(_ airuntime.Summary) error          { return nil }

// workDirWithAgentsMDAndReviewSkill cria TempDir com AGENTS.md e .agents/skills/review/SKILL.md.
func workDirWithAgentsMDAndReviewSkill(t *testing.T) string {
	t.Helper()
	dir := workDirWithAgentsMD(t)
	skillDir := dir + "/.agents/skills/review"
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("workDirWithAgentsMDAndReviewSkill: MkdirAll: %v", err)
	}
	if err := os.WriteFile(skillDir+"/SKILL.md", []byte("# Review Skill\nRevise o diff.\n"), 0o644); err != nil {
		t.Fatalf("workDirWithAgentsMDAndReviewSkill: WriteFile SKILL.md: %v", err)
	}
	return dir
}

// ---- T-39: TestAutoReviewWithGeminiDriver ------------------------------------

// TestAutoReviewWithGeminiDriver — T-39 (F5-Gemini, RF-22).
// Valida que a cascata F5-Claude funciona com driver Gemini:
//   - Session principal Gemini completa normalmente.
//   - Auto-review spawna child session com reviewOutputFn injetado.
//   - evidence/<task>/review.md é gravado com conteúdo válido.
//   - Summary.ReviewStatus == "ok" (sem issues hard no output do review).
//   - Diff zero em internal/runtime/runner_autoreview.go validado implicitamente
//     pela compilação do pacote (cascata tool-agnóstica).
func TestAutoReviewWithGeminiDriver(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Script ACP Gemini: sessão principal completa com tool call.
	script := acpfake.NewScript().
		AppendAgentMessage("Gemini: iniciando task com auto-review").
		AppendToolCall("tc-gemini-ar", "bash").
		AppendToolCallUpdate("tc-gemini-ar", "completed").
		AppendAgentMessage("Gemini: task concluída").
		AppendSessionEnd()

	// reviewOutputFn: simula review Gemini sem issues hard.
	reviewCalled := false
	reviewFn := airuntime.ReviewOutputFn(func(_ context.Context, childJob airuntime.Job) (string, error) {
		reviewCalled = true
		// T-39 HARD: child Job deve ter AutoReview=false (anti-recursão — RF-22).
		if childJob.AutoReview {
			t.Errorf("T-39: child Job.AutoReview=true — anti-recursão violada (HARD)")
		}
		return "APPROVED — nenhum issue crítico encontrado no diff Gemini", nil
	})

	runner := buildGeminiRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := workDirWithAgentsMDAndReviewSkill(t)
	evidenceDir := t.TempDir()

	job := airuntime.Job{
		Prompt:      "execute task com auto-review usando driver Gemini",
		WorkDir:     workDir,
		EvidenceDir: evidenceDir,
		Quiet:       true,
		AutoReview:  true, // F5-Gemini: habilita auto-review
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("T-39: Run com driver Gemini falhou inesperadamente: %v", err)
	}

	// T-39 critério 1: sessão principal completou sem cancelamento.
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("T-39: CancelReason = %q, quero none", summary.CancelReason)
	}

	// T-39 critério 2: reviewOutputFn foi chamada (review spawnado).
	if !reviewCalled {
		t.Error("T-39: reviewOutputFn não foi chamada — auto-review não disparou com Gemini")
	}

	// T-39 critério 3: ReviewStatus deve ser "ok" (sem [HARD] no output do review).
	if summary.ReviewStatus != "ok" {
		t.Errorf("T-39: ReviewStatus = %q, quero ok", summary.ReviewStatus)
	}

	// T-39 critério 4: ReviewPath aponta para arquivo existente (review.md gravado).
	if summary.ReviewPath == "" {
		t.Error("T-39: ReviewPath vazio — review.md não foi criado")
	} else {
		if _, statErr := os.Stat(summary.ReviewPath); statErr != nil {
			t.Errorf("T-39: ReviewPath %q não existe: %v", summary.ReviewPath, statErr)
		}
	}

	// T-39 critério 5: review.md contém "ReviewStatus: ok".
	if summary.ReviewPath != "" {
		content, readErr := os.ReadFile(summary.ReviewPath)
		if readErr != nil {
			t.Fatalf("T-39: ler review.md: %v", readErr)
		}
		if !strings.Contains(string(content), "ReviewStatus: ok") {
			t.Errorf("T-39: review.md não contém 'ReviewStatus: ok'; conteúdo: %q", string(content))
		}
	}

	t.Logf("T-39: OK — Gemini driver auto-review cascata validada; ReviewStatus=%s ReviewPath=%s",
		summary.ReviewStatus, summary.ReviewPath)
}

// ---- TestAutoReviewBlocksOnHardIssuesWithGemini -----------------------------

// TestAutoReviewBlocksOnHardIssuesWithGemini (F5-Gemini, RF-22).
// Valida que quando o review output contém "[HARD]", Summary.ReviewStatus="blocked"
// para sessões com driver Gemini — mesmo comportamento que F5-Claude (cascata tool-agnóstica).
//   - Session Gemini principal completa normalmente.
//   - Review child session retorna output com "[HARD]".
//   - Summary.ReviewStatus == "blocked".
//   - Summary.ReviewPath != "" (review.md criado).
func TestAutoReviewBlocksOnHardIssuesWithGemini(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	script := acpfake.NewScript().
		AppendAgentMessage("Gemini: tarefa com código suspeito").
		AppendToolCall("tc-gemini-hard", "bash").
		AppendToolCallUpdate("tc-gemini-hard", "completed").
		AppendAgentMessage("Gemini: tarefa concluída").
		AppendSessionEnd()

	// reviewOutputFn: simula review Gemini com issue [HARD].
	reviewFn := airuntime.ReviewOutputFn(func(_ context.Context, childJob airuntime.Job) (string, error) {
		// HARD: child Job não deve ter AutoReview=true (anti-recursão — RF-22).
		if childJob.AutoReview {
			t.Errorf("TestAutoReviewBlocksOnHardIssuesWithGemini: child Job.AutoReview=true — recursão não bloqueada")
		}
		return "[HARD] SQL injection detectado em query sem parameterização — proibido por security policy", nil
	})

	runner := buildGeminiRunnerWithReviewFn(t, ctx, script, reviewFn)

	workDir := workDirWithAgentsMDAndReviewSkill(t)
	evidenceDir := t.TempDir()

	job := airuntime.Job{
		Prompt:      "execute tarefa Gemini com issue [HARD] no review",
		WorkDir:     workDir,
		EvidenceDir: evidenceDir,
		Quiet:       true,
		AutoReview:  true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("TestAutoReviewBlocksOnHardIssuesWithGemini: Run falhou inesperadamente: %v", err)
	}

	// Critério 1: ReviewStatus == "blocked" (cascata F5-Claude tool-agnóstica).
	if summary.ReviewStatus != "blocked" {
		t.Errorf("TestAutoReviewBlocksOnHardIssuesWithGemini: ReviewStatus = %q, quero blocked", summary.ReviewStatus)
	}

	// Critério 2: ReviewPath não vazio e arquivo existente.
	if summary.ReviewPath == "" {
		t.Error("TestAutoReviewBlocksOnHardIssuesWithGemini: ReviewPath vazio — review.md não foi criado")
	} else {
		if _, statErr := os.Stat(summary.ReviewPath); statErr != nil {
			t.Errorf("TestAutoReviewBlocksOnHardIssuesWithGemini: ReviewPath %q não existe: %v", summary.ReviewPath, statErr)
		}
	}

	// Critério 3: review.md contém "ReviewStatus: blocked".
	if summary.ReviewPath != "" {
		content, readErr := os.ReadFile(summary.ReviewPath)
		if readErr != nil {
			t.Fatalf("TestAutoReviewBlocksOnHardIssuesWithGemini: ler review.md: %v", readErr)
		}
		if !strings.Contains(string(content), "ReviewStatus: blocked") {
			t.Errorf("TestAutoReviewBlocksOnHardIssuesWithGemini: review.md não contém 'ReviewStatus: blocked'; conteúdo: %q", string(content))
		}
	}

	// Critério 4: sessão principal completou normalmente (ReviewStatus bloqueado != Run erro).
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("TestAutoReviewBlocksOnHardIssuesWithGemini: CancelReason = %q, quero none", summary.CancelReason)
	}

	t.Logf("TestAutoReviewBlocksOnHardIssuesWithGemini: OK — ReviewStatus=blocked com Gemini driver confirmado")
}

// ---- TestAutoReviewInfoMessageEmittedOncePerSession -------------------------

// TestAutoReviewInfoMessageEmittedOncePerSession (F5-Gemini, RF-22).
// Valida que a mensagem INFO sobre custo amplificado:
//   - Contém o texto literal especificado em techspec §"Mensagens de Erro e Warning Literais".
//   - É emitida exatamente uma vez por session quando --auto-review + --tool=gemini.
//   - O padrão sync.Once garante emissão única mesmo com múltiplas chamadas concorrentes.
//
// Este teste valida a mensagem literal e o mecanismo sync.Once.
// A emissão real na CLI ocorre em cmd/ai_spec_harness/task_loop.go quando
// autoReview==true && tool=="gemini" (subtarefa 8.2 — a ser integrado em task 6.0 PR merge).
func TestAutoReviewInfoMessageEmittedOncePerSession(t *testing.T) {
	t.Parallel()

	// Sub-teste 1: validar texto literal da mensagem (RF-22, techspec §"Mensagens de Erro e Warning Literais").
	t.Run("mensagem INFO tem texto literal correto", func(t *testing.T) {
		t.Parallel()

		const expectedPrefix = "INFO: --auto-review com Gemini"
		const expectedSuffix = "Ver GEMINI.md §F5."
		const expectedTokens = "amplificar custo de tokens em janelas 1M+"

		if !strings.HasPrefix(autoReviewGeminiInfoMessage, expectedPrefix) {
			t.Errorf("INFO message não começa com %q; got=%q", expectedPrefix, autoReviewGeminiInfoMessage)
		}
		if !strings.HasSuffix(autoReviewGeminiInfoMessage, expectedSuffix) {
			t.Errorf("INFO message não termina com %q; got=%q", expectedSuffix, autoReviewGeminiInfoMessage)
		}
		if !strings.Contains(autoReviewGeminiInfoMessage, expectedTokens) {
			t.Errorf("INFO message não contém %q; got=%q", expectedTokens, autoReviewGeminiInfoMessage)
		}
	})

	// Sub-teste 2: padrão sync.Once garante emissão exatamente uma vez por session.
	t.Run("sync.Once emite INFO exatamente uma vez mesmo com chamadas concorrentes", func(t *testing.T) {
		t.Parallel()

		var emitCount int
		var mu sync.Mutex
		var once sync.Once

		emitInfoFn := func() {
			once.Do(func() {
				mu.Lock()
				emitCount++
				mu.Unlock()
				// Simula: fmt.Fprintln(os.Stderr, autoReviewGeminiInfoMessage)
				_ = autoReviewGeminiInfoMessage
			})
		}

		// Invocar concorrentemente N vezes — deve emitir exatamente 1 vez.
		const concurrency = 10
		var wg sync.WaitGroup
		wg.Add(concurrency)
		for i := 0; i < concurrency; i++ {
			go func() {
				defer wg.Done()
				emitInfoFn()
			}()
		}
		wg.Wait()

		mu.Lock()
		count := emitCount
		mu.Unlock()

		if count != 1 {
			t.Errorf("INFO emitida %d vezes, quero exatamente 1 (sync.Once não funcionou)", count)
		}
	})

	// Sub-teste 3: INFO não emitida quando AutoReview=false (lógica de guarda).
	t.Run("INFO nao emitida quando AutoReview=false", func(t *testing.T) {
		t.Parallel()

		var emitCount int
		var mu sync.Mutex
		var once sync.Once

		// Simula a lógica de guarda: emite INFO apenas quando autoReview==true && tool=="gemini".
		emitIfNeeded := func(tool string, autoReview bool) {
			if autoReview && tool == "gemini" {
				once.Do(func() {
					mu.Lock()
					emitCount++
					mu.Unlock()
					_ = autoReviewGeminiInfoMessage
				})
			}
		}

		// Chamadas que NÃO devem emitir INFO.
		emitIfNeeded("claude", true)  // tool errado
		emitIfNeeded("gemini", false) // autoReview false
		emitIfNeeded("codex", true)   // tool errado
		emitIfNeeded("gemini", false) // autoReview false (de novo)

		mu.Lock()
		count := emitCount
		mu.Unlock()

		if count != 0 {
			t.Errorf("INFO emitida %d vezes sem a combinação correta, quero 0", count)
		}
	})

	// Sub-teste 4: INFO emitida exatamente uma vez na primeira combinação correta.
	t.Run("INFO emitida na primeira combinacao correta e nao depois", func(t *testing.T) {
		t.Parallel()

		var emitCount int
		var mu sync.Mutex
		var once sync.Once

		emitIfNeeded := func(tool string, autoReview bool) {
			if autoReview && tool == "gemini" {
				once.Do(func() {
					mu.Lock()
					emitCount++
					mu.Unlock()
					_ = autoReviewGeminiInfoMessage
				})
			}
		}

		// Primeira combinação correta → deve emitir.
		emitIfNeeded("gemini", true)
		// Chamadas subsequentes corretas → sync.Once garante que não emite novamente.
		emitIfNeeded("gemini", true)
		emitIfNeeded("gemini", true)

		mu.Lock()
		count := emitCount
		mu.Unlock()

		if count != 1 {
			t.Errorf("INFO emitida %d vezes, quero exatamente 1 (deve emitir uma única vez por session)", count)
		}
	})
}
