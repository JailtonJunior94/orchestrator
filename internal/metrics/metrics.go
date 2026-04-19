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

// BaselineEntry descreve o baseline de uma stack.
type BaselineEntry struct {
	Files     []string `json:"files"`
	Words     int      `json:"words"`
	Chars     int      `json:"chars"`
	TokensEst int      `json:"tokens_est"`
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

// Execute calcula e imprime metricas.
func (s *Service) Execute(rootDir, format string) error {
	report := s.gather(rootDir)

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

func (s *Service) gather(rootDir string) Report {
	report := Report{
		Baselines: make(map[string]BaselineEntry),
		Flows:     make(map[string]FlowEntry),
	}

	// Shared files for baselines
	shared := []string{
		"AGENTS.md",
		".agents/skills/agent-governance/SKILL.md",
	}

	// Per-stack baselines
	stacks := map[string][]string{
		"go": append(shared,
			".agents/skills/go-implementation/SKILL.md",
			".agents/skills/go-implementation/references/architecture.md",
		),
		"node": append(shared,
			".agents/skills/node-implementation/SKILL.md",
			".agents/skills/node-implementation/references/architecture.md",
		),
		"python": append(shared,
			".agents/skills/python-implementation/SKILL.md",
			".agents/skills/python-implementation/references/architecture.md",
		),
	}

	for name, files := range stacks {
		entry := BaselineEntry{Files: files}
		for _, f := range files {
			m := s.fileMetric(filepath.Join(rootDir, f))
			entry.Words += m.Words
			entry.Chars += m.Chars
			entry.TokensEst += m.TokensEst
		}
		report.Baselines[name] = entry
	}

	// Flows
	flows := map[string][]string{
		"execute-task (Go)": {
			"AGENTS.md",
			".agents/skills/agent-governance/SKILL.md",
			".agents/skills/go-implementation/SKILL.md",
			".agents/skills/go-implementation/references/architecture.md",
			".agents/skills/execute-task/SKILL.md",
			".agents/skills/review/SKILL.md",
		},
		"review": {
			"AGENTS.md",
			".agents/skills/agent-governance/SKILL.md",
			".agents/skills/review/SKILL.md",
		},
		"bugfix (Go)": {
			"AGENTS.md",
			".agents/skills/agent-governance/SKILL.md",
			".agents/skills/go-implementation/SKILL.md",
			".agents/skills/go-implementation/references/architecture.md",
			".agents/skills/bugfix/SKILL.md",
		},
	}

	for name, files := range flows {
		entry := FlowEntry{Files: files}
		for _, f := range files {
			m := s.fileMetric(filepath.Join(rootDir, f))
			entry.TokensEst += m.TokensEst
		}
		report.Flows[name] = entry
	}

	// Counts
	skillsDir := filepath.Join(rootDir, ".agents", "skills")
	if entries, err := s.fs.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				report.SkillCount++
				refsDir := filepath.Join(skillsDir, e.Name(), "references")
				if refEntries, err := s.fs.ReadDir(refsDir); err == nil {
					report.RefCount += len(refEntries)
				}
			}
		}
	}

	return report
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
