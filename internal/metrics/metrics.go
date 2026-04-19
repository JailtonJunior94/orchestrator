package metrics

import (
	"encoding/json"
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
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
	Baselines  map[string]BaselineEntry `json:"baselines"`
	Flows      map[string]FlowEntry     `json:"flows"`
	SkillCount int                      `json:"skill_count"`
	RefCount   int                      `json:"reference_count"`
}

// BaselineEntry descreve o baseline de uma skill.
type BaselineEntry struct {
	Files     []string     `json:"files"`
	Words     int          `json:"words"`
	Chars     int          `json:"chars"`
	TokensEst int          `json:"tokens_est"`
	Cost      CostBreakdown `json:"cost"`
}

// FlowEntry descreve o custo de um fluxo operacional.
type FlowEntry struct {
	Files     []string `json:"files"`
	TokensEst int      `json:"tokens_est"`
}

// Service calcula metricas de contexto para governanca.
type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: fsys, printer: printer}
}

// Execute calcula e imprime metricas. Retorna erro se o inventario obrigatorio estiver ausente.
func (s *Service) Execute(rootDir, format string) error {
	report, err := s.gather(rootDir)
	if err != nil {
		return err
	}

	if format == "json" {
		data, _ := json.MarshalIndent(report, "", "  ")
		s.printer.Info("%s", string(data))
		return nil
	}

	// Tabela
	s.printer.Info("Baselines:")
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

	return nil
}

// gather descobre o inventario real do checkout e retorna erro se baseline obrigatoria estiver ausente.
// Baseline obrigatoria: cada diretorio em .agents/skills/ deve conter um SKILL.md.
func (s *Service) gather(rootDir string) (Report, error) {
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
			return report, fmt.Errorf("skill %q: SKILL.md ausente em %s (baseline obrigatoria)", name, skillFile)
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
				entry.Words += rm.Words
				entry.Chars += rm.Chars
				entry.TokensEst += rm.TokensEst
				refTokensTotal += rm.TokensEst
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
		TokensEst: estimateTokens(text),
	}
}

func estimateTokens(text string) int {
	return int(math.Round(float64(len(text)) / 3.5))
}

// ToolBudgets define o limite maximo de tokens estimados por ferramenta.
var ToolBudgets = map[string]int{
	"claude":  70000,
	"gemini":  4000,
	"codex":   13000,
	"copilot": 2000,
}

// CheckBudget estima tokens do conteudo e verifica se esta dentro do budget da ferramenta.
// Retorna tokens estimados, o limite da ferramenta e se esta dentro do budget.
// Se a ferramenta nao tiver budget definido, ok sera sempre true.
func CheckBudget(content string, tool string) (tokens int, limit int, ok bool) {
	tokens = estimateTokens(content)
	limit, exists := ToolBudgets[tool]
	if !exists {
		return tokens, 0, true
	}
	return tokens, limit, tokens <= limit
}

// FormatReport formata o report em string legivel.
func FormatReport(r Report) string {
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
