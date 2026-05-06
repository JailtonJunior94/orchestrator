package taskloop

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrBugfixExhausted indica que o ciclo bugfix -> review atingiu o limite de
// iteracoes sem aprovacao. Quando este erro e retornado, BugfixLoopReport.Escalated
// e true e cabe ao chamador disparar o relatorio de escalonamento humano.
var ErrBugfixExhausted = errors.New("taskloop: limite de 3 iteracoes de bugfix atingido")

// DefaultMaxBugfixIterations e o limite rigido de iteracoes de bugfix (RF-06, ADR-003).
const DefaultMaxBugfixIterations = 3

// BugfixInvoker invoca a skill bugfix sobre achados criticos.
// Implementacoes devem repassar findings e diff ao agente e devolver a saida bruta.
type BugfixInvoker interface {
	InvokeBugfix(ctx context.Context, findings []Finding, diff string) (string, error)
}

// DiffCapturer captura o diff acumulado apos cada iteracao de bugfix
// para alimentar a proxima revisao consolidada.
type DiffCapturer interface {
	CaptureDiff(ctx context.Context) (string, error)
}

// BugfixIteration registra o resultado de uma unica iteracao do ciclo bugfix -> review.
type BugfixIteration struct {
	Sequence         int
	RootCause        string
	BugfixOutput     string
	ReviewVerdict    ReviewVerdict
	CriticalFindings []Finding
}

// BugfixLoopReport agrega o resultado do ciclo de bugfix.
type BugfixLoopReport struct {
	Iterations   []BugfixIteration
	FinalVerdict ReviewVerdict
	FinalReview  *FinalReviewResult
	Escalated    bool
}

// BugfixLoop orquestra ciclos de bugfix -> review ate aprovacao ou exaustao do
// limite de iteracoes (RF-06, RF-07, ADR-003).
type BugfixLoop struct {
	invoker  BugfixInvoker
	reviewer FinalReviewer
	capturer DiffCapturer
	maxIters int
}

// NewBugfixLoop cria um BugfixLoop. maxIters <= 0 usa DefaultMaxBugfixIterations.
func NewBugfixLoop(invoker BugfixInvoker, reviewer FinalReviewer, capturer DiffCapturer, maxIters int) *BugfixLoop {
	if maxIters <= 0 {
		maxIters = DefaultMaxBugfixIterations
	}
	return &BugfixLoop{
		invoker:  invoker,
		reviewer: reviewer,
		capturer: capturer,
		maxIters: maxIters,
	}
}

// Run executa o ciclo bugfix -> review enquanto houver achados Critical.
// initialFindings sao os achados da revisao final consolidada que motivou o loop.
// initialDiff e o diff submetido a essa primeira revisao.
//
// Retorna BugfixLoopReport com o detalhe por iteracao. Quando exausto sem aprovacao,
// retorna ErrBugfixExhausted e marca Escalated=true; chamador deve gerar relatorio humano.
func (b *BugfixLoop) Run(ctx context.Context, initialFindings []Finding, initialDiff string) (BugfixLoopReport, error) {
	report := BugfixLoopReport{}

	critical := filterCritical(initialFindings)
	if len(critical) == 0 {
		report.FinalVerdict = VerdictApproved
		report.FinalReview = &FinalReviewResult{Verdict: VerdictApproved}
		return report, nil
	}

	reviewContext, pureDiff := splitReviewContext(initialDiff)
	for i := 1; i <= b.maxIters; i++ {
		if err := ctx.Err(); err != nil {
			return report, fmt.Errorf("taskloop: bugfix iteracao %d cancelada: %w", i, err)
		}

		out, err := b.invoker.InvokeBugfix(ctx, critical, pureDiff)
		if err != nil {
			return report, fmt.Errorf("taskloop: bugfix iteracao %d: %w", i, err)
		}

		newDiff, err := b.capturer.CaptureDiff(ctx)
		if err != nil {
			return report, fmt.Errorf("taskloop: capturar diff apos bugfix %d: %w", i, err)
		}
		reviewerInput := attachReviewContext(reviewContext, newDiff)

		rev, err := b.reviewer.ReviewConsolidated(ctx, reviewerInput)
		if err != nil {
			return report, fmt.Errorf("taskloop: review apos bugfix %d: %w", i, err)
		}

		nextCritical := filterCritical(rev.Findings)
		report.Iterations = append(report.Iterations, BugfixIteration{
			Sequence:         i,
			RootCause:        extractRootCause(out),
			BugfixOutput:     out,
			ReviewVerdict:    rev.Verdict,
			CriticalFindings: nextCritical,
		})

		if len(nextCritical) == 0 {
			report.FinalVerdict = rev.Verdict
			report.FinalReview = &rev
			return report, nil
		}

		critical = nextCritical
		pureDiff = newDiff
	}

	last := report.Iterations[len(report.Iterations)-1]
	report.FinalVerdict = last.ReviewVerdict
	report.FinalReview = &FinalReviewResult{
		Verdict:  last.ReviewVerdict,
		Findings: append([]Finding(nil), last.CriticalFindings...),
	}
	report.Escalated = true
	return report, ErrBugfixExhausted
}

// splitReviewContext separa o cabecalho de contexto (entregue ao reviewer) do
// diff puro (entregue ao BugfixInvoker). Se o input nao tiver o separador,
// reviewContext vem vazio e o input inteiro e tratado como diff puro.
func splitReviewContext(input string) (reviewContext, diff string) {
	header, rest, ok := strings.Cut(input, reviewContextSeparator)
	if !ok {
		return "", input
	}
	return header, rest
}

func attachReviewContext(reviewContext, diff string) string {
	if reviewContext == "" {
		return diff
	}
	if strings.HasPrefix(diff, "Contexto da revisao consolidada:\n") {
		return diff
	}
	return reviewContext + reviewContextSeparator + diff
}

func filterCritical(findings []Finding) []Finding {
	out := make([]Finding, 0, len(findings))
	for _, f := range findings {
		if f.Severity == SeverityCritical {
			out = append(out, f)
		}
	}
	return out
}

// extractRootCause busca uma linha contendo "causa raiz" (PT-BR) ou "root cause" (EN)
// na saida do bugfix. Retorna string vazia se nao encontrar — campo opcional no relatorio.
func extractRootCause(bugfixOutput string) string {
	for _, line := range strings.Split(bugfixOutput, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "causa raiz") || strings.Contains(lower, "root cause") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
