package taskloop

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// ResolutionAction representa a decisao tomada pelo operador para um Finding
// sob veredito APPROVED_WITH_REMARKS (RF-08, ADR-003).
type ResolutionAction string

const (
	ActionImplement ResolutionAction = "Implement"
	ActionDocument  ResolutionAction = "Document"
	ActionDiscard   ResolutionAction = "Discard"
)

// ErrInvalidResolution indica que a decisao do operador e invalida
// (acao desconhecida ou Discard sem justificativa).
var ErrInvalidResolution = errors.New("taskloop: decisao de ressalva invalida")

// ReservationDecision agrupa o achado e a decisao tomada para ele.
type ReservationDecision struct {
	Finding   Finding
	Action    ResolutionAction
	Rationale string
}

// ActionPlan agrega as decisoes de todas as ressalvas de uma revisao.
type ActionPlan struct {
	Decisions []ReservationDecision
}

// Prompter interage com o operador para coletar a decisao de cada Finding.
// Implementacoes devem retornar a acao escolhida e uma justificativa
// (obrigatoria para Discard, opcional para Implement/Document).
type Prompter interface {
	Ask(ctx context.Context, finding Finding) (ResolutionAction, string, error)
}

// ReservationPlanner resolve as ressalvas de uma revisao APPROVED_WITH_REMARKS
// produzindo um plano de acao auditavel (RF-08, ADR-003).
type ReservationPlanner interface {
	Resolve(ctx context.Context, findings []Finding) (ActionPlan, error)
}

type defaultReservationPlanner struct {
	prompter       Prompter
	nonInteractive bool
}

// NewReservationPlanner cria um ReservationPlanner.
// Quando nonInteractive=true, todas as decisoes assumem ActionDocument como
// default seguro e o prompter nao e invocado.
func NewReservationPlanner(prompter Prompter, nonInteractive bool) ReservationPlanner {
	return &defaultReservationPlanner{prompter: prompter, nonInteractive: nonInteractive}
}

// Resolve coleta uma decisao por Finding. Em modo nao-interativo, todas as
// decisoes sao ActionDocument. Em modo interativo, valida que Discard tem
// justificativa nao vazia.
func (p *defaultReservationPlanner) Resolve(ctx context.Context, findings []Finding) (ActionPlan, error) {
	plan := ActionPlan{}
	if len(findings) == 0 {
		return plan, nil
	}

	for _, f := range findings {
		if err := ctx.Err(); err != nil {
			return plan, fmt.Errorf("taskloop: contexto cancelado durante coleta de ressalvas: %w", err)
		}

		if p.nonInteractive {
			plan.Decisions = append(plan.Decisions, ReservationDecision{
				Finding:   f,
				Action:    ActionDocument,
				Rationale: "modo nao-interativo: documentado como follow-up por default",
			})
			continue
		}

		if p.prompter == nil {
			return plan, fmt.Errorf("%w: prompter ausente em modo interativo", ErrInvalidResolution)
		}

		action, rationale, err := p.prompter.Ask(ctx, f)
		if err != nil {
			return plan, fmt.Errorf("taskloop: erro ao coletar decisao do operador: %w", err)
		}

		decision := ReservationDecision{Finding: f, Action: action, Rationale: strings.TrimSpace(rationale)}
		if err := validateDecision(decision); err != nil {
			return plan, err
		}
		plan.Decisions = append(plan.Decisions, decision)
	}

	return plan, nil
}

func validateDecision(d ReservationDecision) error {
	switch d.Action {
	case ActionImplement, ActionDocument:
		return nil
	case ActionDiscard:
		if d.Rationale == "" {
			return fmt.Errorf("%w: Discard exige justificativa nao vazia", ErrInvalidResolution)
		}
		return nil
	default:
		return fmt.Errorf("%w: acao desconhecida %q", ErrInvalidResolution, d.Action)
	}
}

const (
	actionPlanStart = "<!-- action-plan:start -->"
	actionPlanEnd   = "<!-- action-plan:end -->"
)

// WriteActionPlanToTaskFile persiste o ActionPlan na secao "## Plano de Ação"
// do arquivo da ultima task de forma idempotente (substitui bloco delimitado).
func WriteActionPlanToTaskFile(fsys fs.FileSystem, taskFile string, plan ActionPlan) error {
	data, err := fsys.ReadFile(taskFile)
	if err != nil {
		return fmt.Errorf("taskloop: erro ao ler arquivo da task %s: %w", taskFile, err)
	}

	block := buildActionPlanBlock(plan)
	updated := replaceActionPlanBlock(data, block)

	if err := fsys.WriteFile(taskFile, updated); err != nil {
		return fmt.Errorf("taskloop: erro ao escrever plano de acao em %s: %w", taskFile, err)
	}
	return nil
}

func buildActionPlanBlock(plan ActionPlan) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "%s\n", actionPlanStart)
	fmt.Fprintf(&b, "## Plano de Ação\n\n")
	if len(plan.Decisions) == 0 {
		fmt.Fprintf(&b, "(nenhuma ressalva registrada)\n\n")
		fmt.Fprintf(&b, "%s\n", actionPlanEnd)
		return b.Bytes()
	}
	for i, d := range plan.Decisions {
		loc := d.Finding.File
		if d.Finding.Line > 0 {
			loc = fmt.Sprintf("%s:%d", d.Finding.File, d.Finding.Line)
		}
		if loc == "" {
			loc = "(sem localizacao)"
		}
		fmt.Fprintf(&b, "%d. **[%s]** %s — `%s`\n", i+1, d.Action, loc, d.Finding.Message)
		if d.Rationale != "" {
			fmt.Fprintf(&b, "   - Justificativa: %s\n", d.Rationale)
		}
	}
	fmt.Fprintf(&b, "\n%s\n", actionPlanEnd)
	return b.Bytes()
}

func replaceActionPlanBlock(data, block []byte) []byte {
	startMarker := []byte(actionPlanStart)
	endMarker := []byte(actionPlanEnd)

	startIdx := bytes.Index(data, startMarker)
	endIdx := bytes.Index(data, endMarker)

	if startIdx == -1 || endIdx == -1 || endIdx < startIdx {
		trimmed := bytes.TrimRight(data, "\n")
		var b bytes.Buffer
		b.Write(trimmed)
		b.WriteByte('\n')
		b.Write(block)
		return b.Bytes()
	}

	endOfBlock := endIdx + len(endMarker)
	var b bytes.Buffer
	b.Write(data[:startIdx])
	b.Write(block)
	after := data[endOfBlock:]
	if len(after) > 0 && after[0] == '\n' {
		after = after[1:]
	}
	if len(after) > 0 {
		b.WriteByte('\n')
		b.Write(after)
	}
	return b.Bytes()
}

// AppendFollowUpTasks anexa novas linhas de task em tasks.md para cada decisao
// ActionDocument do plano. Cada follow-up recebe um ID derivado do maior ID
// existente (ex: "9.0", "10.0"...) e status "pending" sem dependencias.
func AppendFollowUpTasks(fsys fs.FileSystem, tasksFile string, plan ActionPlan) error {
	docs := filterDocumentDecisions(plan)
	if len(docs) == 0 {
		return nil
	}

	data, err := fsys.ReadFile(tasksFile)
	if err != nil {
		return fmt.Errorf("taskloop: erro ao ler %s: %w", tasksFile, err)
	}

	existing, err := ParseTasksFile(data)
	if err != nil {
		return fmt.Errorf("taskloop: erro ao parsear %s: %w", tasksFile, err)
	}
	nextID := nextFollowUpID(existing)

	var rows bytes.Buffer
	for i, d := range docs {
		title := summarizeFinding(d.Finding)
		fmt.Fprintf(&rows, "| %d.0 | %s | pending | — | Não |\n", nextID+i, title)
	}

	updated := appendRowsAfterTable(data, rows.Bytes())
	if err := fsys.WriteFile(tasksFile, updated); err != nil {
		return fmt.Errorf("taskloop: erro ao escrever %s: %w", tasksFile, err)
	}
	return nil
}

func filterDocumentDecisions(plan ActionPlan) []ReservationDecision {
	out := make([]ReservationDecision, 0, len(plan.Decisions))
	for _, d := range plan.Decisions {
		if d.Action == ActionDocument {
			out = append(out, d)
		}
	}
	return out
}

func nextFollowUpID(tasks []TaskEntry) int {
	max := 0
	for _, t := range tasks {
		// ID no formato "N.0" ou "N.M"; pegar parte inteira inicial
		dot := strings.Index(t.ID, ".")
		raw := t.ID
		if dot > 0 {
			raw = t.ID[:dot]
		}
		var n int
		_, err := fmt.Sscanf(raw, "%d", &n)
		if err == nil && n > max {
			max = n
		}
	}
	return max + 1
}

func summarizeFinding(f Finding) string {
	msg := strings.TrimSpace(f.Message)
	if msg == "" {
		msg = "follow-up de ressalva"
	}
	if loc := f.File; loc != "" {
		if f.Line > 0 {
			return fmt.Sprintf("Follow-up: %s (%s:%d)", truncateText(msg, 80), loc, f.Line)
		}
		return fmt.Sprintf("Follow-up: %s (%s)", truncateText(msg, 80), loc)
	}
	return fmt.Sprintf("Follow-up: %s", truncateText(msg, 80))
}

func truncateText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// appendRowsAfterTable insere as novas linhas imediatamente apos a ultima linha
// nao vazia da tabela de tasks. A tabela e detectada pela presenca do header
// "| # | Título". Se a tabela nao for detectada, as linhas sao anexadas ao final.
func appendRowsAfterTable(data, rows []byte) []byte {
	lines := strings.Split(string(data), "\n")
	headerIdx := -1
	for i, l := range lines {
		trim := strings.TrimSpace(l)
		if strings.HasPrefix(trim, "| #") || strings.HasPrefix(trim, "|#") {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		var b bytes.Buffer
		b.Write(bytes.TrimRight(data, "\n"))
		b.WriteByte('\n')
		b.Write(rows)
		return b.Bytes()
	}

	lastRow := headerIdx + 1 // separator row
	for i := headerIdx + 2; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trim, "|") {
			lastRow = i
			continue
		}
		break
	}

	rowsStr := strings.TrimRight(string(rows), "\n")
	newLines := make([]string, 0, len(lines)+1)
	newLines = append(newLines, lines[:lastRow+1]...)
	newLines = append(newLines, rowsStr)
	newLines = append(newLines, lines[lastRow+1:]...)
	return []byte(strings.Join(newLines, "\n"))
}
