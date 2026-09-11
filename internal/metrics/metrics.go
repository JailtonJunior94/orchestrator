package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// FileMetric descreve metricas de um unico arquivo.
type FileMetric struct {
	Path      string `json:"path"`
	Words     int    `json:"words"`
	Chars     int    `json:"chars"`
	TokensEst int    `json:"tokens_est"`
}

// Report contem metricas completas de contexto.
type Report struct {
	Baselines   map[string]BaselineEntry `json:"baselines"`
	Flows       map[string]FlowEntry     `json:"flows"`
	SkillCount  int                      `json:"skill_count"`
	RefCount    int                      `json:"reference_count"`
	SkippedDirs []string                 `json:"skipped_dirs,omitempty"`
}

// BaselineEntry descreve o baseline de uma skill.
type BaselineEntry struct {
	Files     []string      `json:"files"`
	Words     int           `json:"words"`
	Chars     int           `json:"chars"`
	TokensEst int           `json:"tokens_est"`
	Cost      CostBreakdown `json:"cost"`
}

// FlowEntry descreve o custo de um fluxo operacional.
type FlowEntry struct {
	Files     []string `json:"files"`
	TokensEst int      `json:"tokens_est"`
}

// Service calcula metricas de contexto para governanca.
type Service struct {
	fs        fs.FileSystem
	printer   *output.Printer
	tokenizer Tokenizer
}

func NewService(fsys fs.FileSystem, printer *output.Printer, tok Tokenizer) *Service {
	if tok == nil {
		tok = NewCharEstimator()
	}
	return &Service{fs: fsys, printer: printer, tokenizer: tok}
}

// Execute calcula e imprime metricas. Retorna erro se o inventario obrigatorio estiver ausente.
func (s *Service) Execute(rootDir, format string, brief bool) error {
	report, err := s.gather(rootDir, brief)
	if err != nil {
		return err
	}

	if format == "json" {
		data, _ := json.MarshalIndent(report, "", "  ")
		s.printer.Info("%s", string(data))
		return nil
	}

	// Tabela
	s.printer.Info("Baselines (brief=%v):", brief)
	for stack, entry := range report.Baselines {
		s.printer.Info("- %s: words=%d chars=%d est_tokens=%d", stack, entry.Words, entry.Chars, entry.TokensEst)
	}
	s.printer.Info("")
	s.printer.Info("Flows:")
	for flow, entry := range report.Flows {
		s.printer.Info("- %s: est_tokens=%d", flow, entry.TokensEst)
	}
	s.printer.Info("")
	s.printer.Info("Skills: %d", report.SkillCount)
	s.printer.Info("References: %d", report.RefCount)

	if len(report.SkippedDirs) > 0 {
		s.printer.Info("")
		s.printer.Info("Diretorios ignorados (sem SKILL.md): %d", len(report.SkippedDirs))
		for _, d := range report.SkippedDirs {
			s.printer.Info("- %s", d)
		}
	}

	return nil
}

// gather descobre o inventario real do checkout e retorna erro se baseline obrigatoria estiver ausente.
// Baseline obrigatoria: cada diretorio em .agents/skills/ deve conter um SKILL.md.
func (s *Service) gather(rootDir string, brief bool) (Report, error) {
	report := Report{
		Baselines: make(map[string]BaselineEntry),
		Flows:     make(map[string]FlowEntry),
	}

	skillsDir := filepath.Join(rootDir, ".agents", "skills")
	if !s.fs.Exists(skillsDir) {
		return report, fmt.Errorf("diretorio de skills nao encontrado: %s", skillsDir)
	}

	entries, err := s.fs.ReadDir(skillsDir)
	if err != nil {
		return report, fmt.Errorf("erro ao ler diretorio de skills %s: %w", skillsDir, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		skillFile := filepath.Join(skillsDir, name, "SKILL.md")
		if !s.fs.Exists(skillFile) {
			relPath := ".agents/skills/" + name
			report.SkippedDirs = append(report.SkippedDirs, relPath)
			s.printer.Warn("diretorio sem SKILL.md ignorado: %s", relPath)
			continue
		}

		entry := BaselineEntry{}

		m := s.fileMetric(skillFile)
		entry.Files = append(entry.Files, skillFile)
		entry.Words += m.Words
		entry.Chars += m.Chars
		entry.TokensEst += m.TokensEst
		skillOnlyTokens := m.TokensEst

		refTokensTotal := 0
		refFileCount := 0
		refsDir := filepath.Join(skillsDir, name, "references")
		if refEntries, rerr := s.fs.ReadDir(refsDir); rerr == nil {
			for _, ref := range refEntries {
				if ref.IsDir() {
					continue
				}
				refPath := filepath.Join(refsDir, ref.Name())
				rm := s.fileMetric(refPath)
				entry.Files = append(entry.Files, refPath)

				if brief {
					// Em modo brief, simulamos 150 tokens por TL;DR em vez do arquivo completo
					entry.Words += 30      // estimativa
					entry.Chars += 150 * 3 // estimativa
					entry.TokensEst += 150
					refTokensTotal += 150
				} else {
					entry.Words += rm.Words
					entry.Chars += rm.Chars
					entry.TokensEst += rm.TokensEst
					refTokensTotal += rm.TokensEst
				}
				refFileCount++
				report.RefCount++
			}
		}

		incrementalRef := 0
		if refFileCount > 0 {
			incrementalRef = refTokensTotal / refFileCount
		}
		entry.Cost = CostBreakdown{
			OnDisk:         entry.TokensEst,
			Loaded:         skillOnlyTokens,
			IncrementalRef: incrementalRef,
			RefCount:       refFileCount,
		}

		report.Baselines[name] = entry
		report.SkillCount++
	}

	return report, nil
}

func (s *Service) fileMetric(path string) FileMetric {
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return FileMetric{Path: path}
	}
	text := string(data)
	return FileMetric{
		Path:      path,
		Words:     len(strings.Fields(text)),
		Chars:     len(text),
		TokensEst: s.tokenizer.EstimateTokens(text),
	}
}

func (c *Catalog) estimateTokens(text string) int {
	return int(math.Round(float64(len(text)) / 3.5))
}

var ToolBudgets = NewCatalog().toolBudgets()

var ToolBudgetsLarge = NewCatalog().toolBudgetsLarge()

func (c *Catalog) toolBudgets() map[string]int {
	registry := specs.NewCatalog().Registry()
	out := make(map[string]int, len(registry))
	for _, agent := range registry {
		out[agent.ID()] = agent.StandardBudget()
	}
	return out
}

func (c *Catalog) toolBudgetsLarge() map[string]int {
	registry := specs.NewCatalog().Registry()
	out := make(map[string]int, len(registry))
	for _, agent := range registry {
		if agent.LargeBudget() > 0 {
			out[agent.ID()] = agent.LargeBudget()
		}
	}
	return out
}

var ErrToolBudgetsLargeCoverage = errors.New("tool budgets large: coverage gate failed")

func checkToolBudgetsLargeCoverage(registry []specs.Agent, budgetsLarge map[string]int) error {
	known := make(map[string]bool, len(registry))
	for _, agent := range registry {
		known[agent.ID()] = true
		if agent.LargeBudget() > 0 {
			if _, ok := budgetsLarge[agent.ID()]; !ok {
				return fmt.Errorf("%w: agent %q has a positive LargeBudget but no ToolBudgetsLarge entry", ErrToolBudgetsLargeCoverage, agent.ID())
			}
		}
	}
	for id := range budgetsLarge {
		if !known[id] {
			return fmt.Errorf("%w: orphan key %q in ToolBudgetsLarge", ErrToolBudgetsLargeCoverage, id)
		}
	}
	return nil
}

func (c *Catalog) CheckToolBudgetsLargeCoverage() error {
	return checkToolBudgetsLargeCoverage(specs.NewCatalog().Registry(), ToolBudgetsLarge)
}

// CheckBudgetForClass verifica o budget levando em conta a WindowClass (ADR-023).
// WindowLarge: tenta ToolBudgetsLarge[tool]; se ausente, cai em ToolBudgets[tool].
// WindowStandard (ou zero-value): usa ToolBudgets[tool] (comportamento F1 preservado).
func (c *Catalog) CheckBudgetForClass(content string, tool string, large bool) (tokens int, limit int, ok bool) {
	tokens = NewCatalog().estimateTokens(content)
	if large {
		if l, exists := ToolBudgetsLarge[tool]; exists {
			return tokens, l, tokens <= l
		}
	}
	l, exists := ToolBudgets[tool]
	if !exists {
		return tokens, 0, true
	}
	return tokens, l, tokens <= l
}

// CheckBudget estima tokens do conteudo e verifica se esta dentro do budget da ferramenta.
// Retorna tokens estimados, o limite da ferramenta e se esta dentro do budget.
// Se a ferramenta nao tiver budget definido, ok sera sempre true.
func (c *Catalog) CheckBudget(content string, tool string) (tokens int, limit int, ok bool) {
	tokens = NewCatalog().estimateTokens(content)
	limit, exists := ToolBudgets[tool]
	if !exists {
		return tokens, 0, true
	}
	return tokens, limit, tokens <= limit
}

// FormatReport formata o report em string legivel.
func (c *Catalog) FormatReport(r Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Baselines:\n")
	for stack, entry := range r.Baselines {
		fmt.Fprintf(&b, "- %s: words=%d chars=%d est_tokens=%d\n", stack, entry.Words, entry.Chars, entry.TokensEst)
	}
	fmt.Fprintf(&b, "\nFlows:\n")
	for flow, entry := range r.Flows {
		fmt.Fprintf(&b, "- %s: est_tokens=%d\n", flow, entry.TokensEst)
	}
	fmt.Fprintf(&b, "\nSkills: %d\nReferences: %d\n", r.SkillCount, r.RefCount)
	return b.String()
}
