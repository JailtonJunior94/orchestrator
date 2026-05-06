package taskloop

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// stubBugfixInvoker simula a invocacao da skill bugfix.
type stubBugfixInvoker struct {
	outputs []string
	calls   int
	err     error
	diffs   []string
}

func (s *stubBugfixInvoker) InvokeBugfix(ctx context.Context, findings []Finding, diff string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	idx := s.calls
	s.calls++
	s.diffs = append(s.diffs, diff)
	if idx >= len(s.outputs) {
		return "", nil
	}
	return s.outputs[idx], nil
}

// stubFinalReviewer retorna um veredito por chamada.
type stubFinalReviewer struct {
	results []FinalReviewResult
	calls   int
	err     error
	diffs   []string
}

func (s *stubFinalReviewer) ReviewConsolidated(ctx context.Context, diff string) (FinalReviewResult, error) {
	if s.err != nil {
		return FinalReviewResult{}, s.err
	}
	idx := s.calls
	s.calls++
	s.diffs = append(s.diffs, diff)
	if idx >= len(s.results) {
		return FinalReviewResult{Verdict: VerdictApproved}, nil
	}
	return s.results[idx], nil
}

// stubDiffCapturer retorna diffs sequenciais.
type stubDiffCapturer struct {
	diffs []string
	calls int
}

func (s *stubDiffCapturer) CaptureDiff(ctx context.Context) (string, error) {
	idx := s.calls
	s.calls++
	if idx >= len(s.diffs) {
		return "", nil
	}
	return s.diffs[idx], nil
}

func criticalFinding(msg string) Finding {
	return Finding{Severity: SeverityCritical, Message: msg}
}

func TestBugfixLoop_Run(t *testing.T) {
	criticals := []Finding{criticalFinding("[Critical] x")}
	approved := FinalReviewResult{Verdict: VerdictApproved}
	stillCritical := FinalReviewResult{Verdict: VerdictRejected, Findings: criticals}
	withRemarks := FinalReviewResult{Verdict: VerdictApprovedWithRemarks, Findings: []Finding{
		{Severity: SeverityImportant, Message: "[Important] y"},
	}}

	tests := []struct {
		name           string
		initial        []Finding
		invokerOutputs []string
		reviewResults  []FinalReviewResult
		maxIters       int
		wantErr        error
		wantIterations int
		wantEscalated  bool
		wantVerdict    ReviewVerdict
	}{
		{
			name:           "sem findings criticos retorna aprovado sem invocar bugfix",
			initial:        []Finding{{Severity: SeverityImportant}},
			wantIterations: 0,
			wantVerdict:    VerdictApproved,
		},
		{
			name:           "aprovado na 1a iteracao",
			initial:        criticals,
			invokerOutputs: []string{"causa raiz: nil pointer\nfix aplicado"},
			reviewResults:  []FinalReviewResult{approved},
			wantIterations: 1,
			wantVerdict:    VerdictApproved,
		},
		{
			name:           "aprovado com ressalvas na 1a iteracao",
			initial:        criticals,
			invokerOutputs: []string{"root cause: race condition"},
			reviewResults:  []FinalReviewResult{withRemarks},
			wantIterations: 1,
			wantVerdict:    VerdictApprovedWithRemarks,
		},
		{
			name:           "aprovado na 2a iteracao",
			initial:        criticals,
			invokerOutputs: []string{"fix 1", "fix 2"},
			reviewResults:  []FinalReviewResult{stillCritical, approved},
			wantIterations: 2,
			wantVerdict:    VerdictApproved,
		},
		{
			name:           "aprovado na 3a iteracao",
			initial:        criticals,
			invokerOutputs: []string{"fix 1", "fix 2", "fix 3"},
			reviewResults:  []FinalReviewResult{stillCritical, stillCritical, approved},
			wantIterations: 3,
			wantVerdict:    VerdictApproved,
		},
		{
			name:           "exaurido apos 3 iteracoes",
			initial:        criticals,
			invokerOutputs: []string{"fix 1", "fix 2", "fix 3"},
			reviewResults:  []FinalReviewResult{stillCritical, stillCritical, stillCritical},
			wantErr:        ErrBugfixExhausted,
			wantIterations: 3,
			wantEscalated:  true,
			wantVerdict:    VerdictRejected,
		},
		{
			name:           "respeita maxIters customizado",
			initial:        criticals,
			invokerOutputs: []string{"fix 1"},
			reviewResults:  []FinalReviewResult{stillCritical},
			maxIters:       1,
			wantErr:        ErrBugfixExhausted,
			wantIterations: 1,
			wantEscalated:  true,
			wantVerdict:    VerdictRejected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invoker := &stubBugfixInvoker{outputs: tt.invokerOutputs}
			reviewer := &stubFinalReviewer{results: tt.reviewResults}
			capturer := &stubDiffCapturer{diffs: []string{"d1", "d2", "d3"}}

			loop := NewBugfixLoop(invoker, reviewer, capturer, tt.maxIters)
			report, err := loop.Run(context.Background(), tt.initial, "diff-inicial")

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("erro esperado %v, obtido %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}

			if len(report.Iterations) != tt.wantIterations {
				t.Errorf("iteracoes: esperado %d, obtido %d", tt.wantIterations, len(report.Iterations))
			}
			if report.Escalated != tt.wantEscalated {
				t.Errorf("Escalated: esperado %v, obtido %v", tt.wantEscalated, report.Escalated)
			}
			if report.FinalVerdict != tt.wantVerdict {
				t.Errorf("FinalVerdict: esperado %s, obtido %s", tt.wantVerdict, report.FinalVerdict)
			}
			if report.FinalReview == nil {
				t.Fatalf("FinalReview ausente")
			}
			if report.FinalReview.Verdict != tt.wantVerdict {
				t.Errorf("FinalReview.Verdict: esperado %s, obtido %s", tt.wantVerdict, report.FinalReview.Verdict)
			}
		})
	}
}

func TestBugfixLoop_DefaultMaxIterations(t *testing.T) {
	loop := NewBugfixLoop(nil, nil, nil, 0)
	if loop.maxIters != DefaultMaxBugfixIterations {
		t.Fatalf("maxIters default esperado %d, obtido %d", DefaultMaxBugfixIterations, loop.maxIters)
	}
	loop = NewBugfixLoop(nil, nil, nil, -5)
	if loop.maxIters != DefaultMaxBugfixIterations {
		t.Fatalf("maxIters default (negativo) esperado %d, obtido %d", DefaultMaxBugfixIterations, loop.maxIters)
	}
}

func TestBugfixLoop_PreserveReviewContextAcrossIterations(t *testing.T) {
	criticals := []Finding{criticalFinding("[Critical] x")}
	reviewer := &stubFinalReviewer{results: []FinalReviewResult{{Verdict: VerdictApproved}}}
	invoker := &stubBugfixInvoker{outputs: []string{"causa raiz: ajuste no prompt"}}
	loop := NewBugfixLoop(
		invoker,
		reviewer,
		&stubDiffCapturer{diffs: []string{"diff-after-bugfix"}},
		1,
	)

	rawDiff := "diff --git a/a.go b/a.go\n"
	initial := "Contexto da revisao consolidada:\n- PRD: /tmp/prd.md\n- TechSpec: /tmp/techspec.md\n" + reviewContextSeparator + rawDiff
	_, err := loop.Run(context.Background(), criticals, initial)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	// Bug 1 (regressao): reviewer recebe payload com contexto + diff atualizado.
	if len(reviewer.diffs) != 1 {
		t.Fatalf("reviewer recebeu %d diffs, want 1", len(reviewer.diffs))
	}
	got := reviewer.diffs[0]
	for _, want := range []string{
		"Contexto da revisao consolidada:",
		"- PRD: /tmp/prd.md",
		"- TechSpec: /tmp/techspec.md",
		reviewContextSeparator + "diff-after-bugfix",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("payload do reviewer nao contem %q:\n%s", want, got)
		}
	}

	// Bug 2 (regressao): BugfixInvoker recebe diff puro, sem o cabecalho de contexto.
	if len(invoker.diffs) != 1 {
		t.Fatalf("invoker recebeu %d diffs, want 1", len(invoker.diffs))
	}
	if invoker.diffs[0] != rawDiff {
		t.Fatalf("invoker deveria receber diff puro %q, recebeu %q", rawDiff, invoker.diffs[0])
	}
	if strings.Contains(invoker.diffs[0], "Contexto da revisao consolidada:") {
		t.Fatalf("invoker recebeu cabecalho de contexto indevido:\n%s", invoker.diffs[0])
	}
}

// TestBugfixLoop_InvokerReceivesPureDiffAcrossIterations trava Bug 2 em iteracoes
// subsequentes: o BugfixInvoker deve continuar recebendo apenas o diff bruto
// retornado pelo DiffCapturer, sem o cabecalho de contexto consolidado.
func TestBugfixLoop_InvokerReceivesPureDiffAcrossIterations(t *testing.T) {
	criticals := []Finding{criticalFinding("[Critical] x")}
	reviewer := &stubFinalReviewer{results: []FinalReviewResult{
		{Verdict: VerdictRejected, Findings: []Finding{criticalFinding("[Critical] x")}},
		{Verdict: VerdictApproved},
	}}
	invoker := &stubBugfixInvoker{outputs: []string{"causa raiz: a", "causa raiz: b"}}
	capturer := &stubDiffCapturer{diffs: []string{"diff-iter-1", "diff-iter-2"}}
	loop := NewBugfixLoop(invoker, reviewer, capturer, 2)

	initial := "Contexto da revisao consolidada:\n- PRD: /tmp/prd.md\n" + reviewContextSeparator + "diff-inicial"
	_, err := loop.Run(context.Background(), criticals, initial)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	wantInvokerDiffs := []string{"diff-inicial", "diff-iter-1"}
	if len(invoker.diffs) != len(wantInvokerDiffs) {
		t.Fatalf("invoker.diffs len = %d, want %d", len(invoker.diffs), len(wantInvokerDiffs))
	}
	for i, want := range wantInvokerDiffs {
		if invoker.diffs[i] != want {
			t.Fatalf("iteracao %d: invoker recebeu %q, want %q", i+1, invoker.diffs[i], want)
		}
	}
	for i, got := range reviewer.diffs {
		if !strings.HasPrefix(got, "Contexto da revisao consolidada:") {
			t.Fatalf("reviewer iteracao %d perdeu o contexto:\n%s", i+1, got)
		}
	}
}

func TestBugfixLoop_PropagaErroDoBugfix(t *testing.T) {
	bugErr := errors.New("falha de invocacao")
	loop := NewBugfixLoop(
		&stubBugfixInvoker{err: bugErr},
		&stubFinalReviewer{},
		&stubDiffCapturer{},
		3,
	)
	_, err := loop.Run(context.Background(), []Finding{criticalFinding("x")}, "diff")
	if !errors.Is(err, bugErr) {
		t.Fatalf("esperado wrap de %v, obtido %v", bugErr, err)
	}
}

func TestBugfixLoop_PropagaErroDoReviewer(t *testing.T) {
	revErr := errors.New("review falhou")
	loop := NewBugfixLoop(
		&stubBugfixInvoker{outputs: []string{"out"}},
		&stubFinalReviewer{err: revErr},
		&stubDiffCapturer{},
		3,
	)
	_, err := loop.Run(context.Background(), []Finding{criticalFinding("x")}, "diff")
	if !errors.Is(err, revErr) {
		t.Fatalf("esperado wrap de %v, obtido %v", revErr, err)
	}
}

func TestBugfixLoop_RespeitaCancelamentoDeContexto(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	loop := NewBugfixLoop(
		&stubBugfixInvoker{outputs: []string{"o"}},
		&stubFinalReviewer{results: []FinalReviewResult{{Verdict: VerdictApproved}}},
		&stubDiffCapturer{},
		3,
	)
	_, err := loop.Run(ctx, []Finding{criticalFinding("x")}, "diff")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperado context.Canceled, obtido %v", err)
	}
}

func TestExtractRootCause(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"ausente", "fix aplicado", ""},
		{"causa raiz pt-br", "Linha 1\nCausa Raiz: nil pointer em foo()\nLinha 3", "Causa Raiz: nil pointer em foo()"},
		{"root cause en", "step 1\nRoot cause: race in handler\n", "Root cause: race in handler"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRootCause(tt.in)
			if got != tt.want {
				t.Errorf("esperado %q, obtido %q", tt.want, got)
			}
		})
	}
}

func TestFilterCritical(t *testing.T) {
	in := []Finding{
		{Severity: SeverityCritical, Message: "a"},
		{Severity: SeverityImportant, Message: "b"},
		{Severity: SeverityCritical, Message: "c"},
		{Severity: SeveritySuggestion, Message: "d"},
	}
	got := filterCritical(in)
	if len(got) != 2 {
		t.Fatalf("esperado 2 criticos, obtido %d", len(got))
	}
	if got[0].Message != "a" || got[1].Message != "c" {
		t.Errorf("ordem/conteudo inesperados: %+v", got)
	}
}

func TestOptions_MaxBugfixIterationsField(t *testing.T) {
	// Garante que o campo existe e e atribuivel — gate de regressao para a API.
	opts := Options{MaxBugfixIterations: 5}
	if opts.MaxBugfixIterations != 5 {
		t.Fatalf("campo MaxBugfixIterations nao persistiu: %d", opts.MaxBugfixIterations)
	}
}

func TestErrBugfixExhausted_Mensagem(t *testing.T) {
	if !strings.Contains(ErrBugfixExhausted.Error(), "3 iteracoes") {
		t.Errorf("mensagem deve mencionar limite de 3 iteracoes: %s", ErrBugfixExhausted.Error())
	}
}
