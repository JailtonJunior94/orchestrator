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

// ErrBugfixEvidenceIncomplete indica que o executor nao forneceu a prova minima
// exigida para uma tentativa de correcao. Sem a reproducao que falhava e o teste
// que passou depois da alteracao, a revisao fresca nao e evidencia suficiente.
var ErrBugfixEvidenceIncomplete = errors.New("taskloop: evidencia de bugfix incompleta")

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
	Origin           string
	RootCause        string
	FailBefore       string
	PassAfter        string
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

	critical := NewCatalog().filterCritical(initialFindings)
	if len(critical) == 0 {
		report.FinalVerdict = VerdictApproved
		report.FinalReview = &FinalReviewResult{Verdict: VerdictApproved}
		return report, nil
	}

	reviewContext, pureDiff := NewCatalog().splitReviewContext(initialDiff)
	for i := 1; i <= b.maxIters; i++ {
		if err := ctx.Err(); err != nil {
			return report, fmt.Errorf("taskloop: bugfix iteracao %d cancelada: %w", i, err)
		}

		out, err := b.invoker.InvokeBugfix(ctx, critical, pureDiff)
		if err != nil {
			return report, fmt.Errorf("taskloop: bugfix iteracao %d: %w", i, err)
		}
		evidence, err := NewCatalog().extractBugfixEvidence(out)
		if err != nil {
			return report, fmt.Errorf("taskloop: bugfix iteracao %d: %w", i, err)
		}

		newDiff, err := b.capturer.CaptureDiff(ctx)
		if err != nil {
			return report, fmt.Errorf("taskloop: capturar diff apos bugfix %d: %w", i, err)
		}
		reviewerInput := NewCatalog().attachReviewContext(reviewContext, newDiff)

		rev, err := b.reviewer.ReviewConsolidated(ctx, reviewerInput)
		if err != nil {
			return report, fmt.Errorf("taskloop: review apos bugfix %d: %w", i, err)
		}

		nextCritical := NewCatalog().filterCritical(rev.Findings)
		report.Iterations = append(report.Iterations, BugfixIteration{
			Sequence:         i,
			Origin:           NewCatalog().formatBugfixOrigin(critical),
			RootCause:        NewCatalog().extractRootCause(out),
			FailBefore:       evidence.FailBefore,
			PassAfter:        evidence.PassAfter,
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

type bugfixEvidence struct {
	FailBefore string
	PassAfter  string
}

// extractBugfixEvidence exige marcadores explicitos no retorno do executor.
// O texto e preservado no relatorio, mas estes campos estruturados permitem ao
// orquestrador falhar fechado antes de aceitar uma nova revisao como prova.
func (c *Catalog) extractBugfixEvidence(output string) (bugfixEvidence, error) {
	evidence := bugfixEvidence{
		FailBefore: c.findEvidenceValue(output, "fail-before", "falha antes"),
		PassAfter:  c.findEvidenceValue(output, "pass-after", "passa depois"),
	}
	if evidence.FailBefore == "" || evidence.PassAfter == "" {
		return bugfixEvidence{}, fmt.Errorf("%w: informe Fail-before e Pass-after", ErrBugfixEvidenceIncomplete)
	}
	return evidence, nil
}

func (c *Catalog) findEvidenceValue(output string, labels ...string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		for _, label := range labels {
			prefix := strings.ToLower(label) + ":"
			if strings.HasPrefix(lower, prefix) {
				return strings.TrimSpace(trimmed[len(prefix):])
			}
		}
	}
	return ""
}

// formatBugfixOrigin preserva a origem dos achados sem depender de texto livre
// emitido pelo executor. A origem e sempre a revisao fresca que acionou a tentativa.
func (c *Catalog) formatBugfixOrigin(findings []Finding) string {
	origins := make([]string, 0, len(findings))
	for _, finding := range findings {
		location := strings.TrimSpace(finding.File)
		if finding.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, finding.Line)
		}
		if location == "" {
			location = "sem localizacao"
		}
		origins = append(origins, "finding de review: "+location)
	}
	return strings.Join(origins, ", ")
}

// splitReviewContext separa o cabecalho de contexto (entregue ao reviewer) do
// diff puro (entregue ao BugfixInvoker). Se o input nao tiver o separador,
// reviewContext vem vazio e o input inteiro e tratado como diff puro.
func (c *Catalog) splitReviewContext(input string) (reviewContext, diff string) {
	header, rest, ok := strings.Cut(input, _reviewContextSeparator)
	if !ok {
		return "", input
	}
	return header, rest
}

func (c *Catalog) attachReviewContext(reviewContext, diff string) string {
	if reviewContext == "" {
		return diff
	}
	if strings.HasPrefix(diff, "Contexto da revisao consolidada:\n") {
		return diff
	}
	return reviewContext + _reviewContextSeparator + diff
}

func (c *Catalog) filterCritical(findings []Finding) []Finding {
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
func (c *Catalog) extractRootCause(bugfixOutput string) string {
	for _, line := range strings.Split(bugfixOutput, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "causa raiz") || strings.Contains(lower, "root cause") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}
