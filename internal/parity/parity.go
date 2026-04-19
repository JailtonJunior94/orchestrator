// Package parity verifica equivalencia semantica minima entre os artefatos
// de governanca gerados para as ferramentas suportadas.
//
// # Invariantes vs. paridade textual
//
// O harness valida propriedades semanticas — presenca de referencias canonicas,
// documentacao de limites de enforcement, consistencia de caminhos — sem exigir
// igualdade textual entre artefatos de ferramentas diferentes.
//
// # Limites de enforcement por ferramenta
//
//   - Claude Code: enforcement programatico via hooks PreToolUse/PostToolUse.
//     Invariantes marcados ToolSpecific para Claude sao obrigatorios.
//
//   - Gemini CLI: sem hooks ou agents nativos. Compliance depende do modelo
//     seguir instrucoes procedurais. Invariantes BestEffort documentam a lacuna
//     mas nao bloqueiam.
//
//   - Codex: le AGENTS.md como instrucao de sessao. Sem hooks nativos.
//     config.toml lista metadados de skills para upgrade.sh — nao enforcement real.
//
//   - Copilot: carrega copilot-instructions.md automaticamente, sem hooks.
//     Compliance depende do modelo seguir instrucoes.
package parity

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// EnforcementLevel classifica o tipo de enforcement de um invariante semantico.
type EnforcementLevel string

const (
	// Common: requisito compartilhado por todas as ferramentas selecionadas.
	// Violacao indica drift critico entre destinos.
	Common EnforcementLevel = "common"

	// ToolSpecific: obrigatorio apenas para a ferramenta indicada.
	// Violacao indica ausencia de capacidade programatica esperada.
	ToolSpecific EnforcementLevel = "tool-specific"

	// BestEffort: compliance procedural sem enforcement automatico.
	// Violacao e registrada mas nao bloqueia.
	BestEffort EnforcementLevel = "best-effort"
)

// Invariant define um requisito semantico minimo verificavel.
type Invariant struct {
	ID          string
	Description string
	Level       EnforcementLevel
	// AppliesTo lista as ferramentas relevantes.
	// Nil significa que o invariante se aplica a todas as ferramentas selecionadas.
	AppliesTo []skills.Tool
	// Check executa a verificacao sobre o snapshot de artefatos gerados.
	Check func(s Snapshot) Result
}

// Snapshot contem os artefatos gerados para um conjunto de ferramentas.
type Snapshot struct {
	Tools      []skills.Tool
	ProjectDir string
	Files      map[string][]byte
	Dirs       map[string]bool
	Links      map[string]string
}

// File retorna o conteudo de um caminho relativo ao ProjectDir, ou "" se ausente.
func (s Snapshot) File(rel string) string {
	data, ok := s.Files[filepath.Join(s.ProjectDir, rel)]
	if !ok {
		return ""
	}
	return string(data)
}

// hasTool verifica se a ferramenta esta nas ferramentas selecionadas.
func (s Snapshot) hasTool(t skills.Tool) bool {
	for _, tool := range s.Tools {
		if tool == t {
			return true
		}
	}
	return false
}

// Result representa o resultado de uma verificacao de invariante.
type Result struct {
	OK     bool
	Reason string
}

func pass() Result                       { return Result{OK: true} }
func fail(reason string) Result          { return Result{OK: false, Reason: reason} }
func failf(f string, a ...any) Result    { return Result{OK: false, Reason: fmt.Sprintf(f, a...)} }

// CheckResult agrupa invariante, resultado e flag de skip.
type CheckResult struct {
	Invariant *Invariant
	Result    Result
	// Skipped indica que o invariante nao se aplica as ferramentas selecionadas.
	Skipped bool
}

// Run executa todos os invariantes contra um Snapshot.
// Invariantes cujas AppliesTo nao incluam nenhuma ferramenta selecionada sao marcados Skipped.
func Run(snap Snapshot, invariants []*Invariant) []CheckResult {
	active := make(map[skills.Tool]bool, len(snap.Tools))
	for _, t := range snap.Tools {
		active[t] = true
	}

	results := make([]CheckResult, 0, len(invariants))
	for _, inv := range invariants {
		if len(inv.AppliesTo) > 0 {
			relevant := false
			for _, t := range inv.AppliesTo {
				if active[t] {
					relevant = true
					break
				}
			}
			if !relevant {
				results = append(results, CheckResult{Invariant: inv, Skipped: true})
				continue
			}
		}
		r := inv.Check(snap)
		results = append(results, CheckResult{Invariant: inv, Result: r})
	}
	return results
}

// Generate produz artefatos via contextgen e retorna um Snapshot.
// Utiliza um FakeFileSystem em memoria — nao escreve em disco.
// Alem dos artefatos do contextgen, popula stubs para hooks, scripts e rules
// que sao instalados pelo install.Service (mas nao gerados pelo contextgen).
func Generate(projectDir string, tools []skills.Tool, langs []skills.Lang, codexProfile string) (Snapshot, error) {
	ffs := fs.NewFakeFileSystem()
	sourceDir := projectDir + "-src"
	ffs.Dirs[projectDir] = true
	ffs.Dirs[sourceDir] = true

	g := contextgen.NewGenerator(ffs, output.New(false))
	if err := g.Generate(sourceDir, projectDir, tools, langs, codexProfile, false); err != nil {
		return Snapshot{}, fmt.Errorf("gerar artefatos: %w", err)
	}

	toolSet := make(map[skills.Tool]bool, len(tools))
	for _, t := range tools {
		toolSet[t] = true
	}

	// Stubs para artefatos Claude instalados pelo install.Service
	if toolSet[skills.ToolClaude] {
		claudeStubs := []string{
			".claude/hooks/validate-governance.sh",
			".claude/hooks/validate-preload.sh",
			".claude/rules/governance.md",
			".claude/scripts/validate-task-evidence.sh",
			".claude/scripts/validate-bugfix-evidence.sh",
			".claude/scripts/validate-refactor-evidence.sh",
		}
		for _, p := range claudeStubs {
			_ = ffs.WriteFile(filepath.Join(projectDir, p), []byte("#!/bin/sh\n# stub"))
		}
	}

	// Stub para hook Gemini instalado pelo install.Service
	if toolSet[skills.ToolGemini] {
		_ = ffs.WriteFile(filepath.Join(projectDir, ".gemini/hooks/validate-preload.sh"), []byte("#!/bin/sh\n# stub"))
	}

	// Stub para guard de profundidade (cross-tool, sempre instalado com Claude)
	_ = ffs.WriteFile(filepath.Join(projectDir, "scripts/lib/check-invocation-depth.sh"), []byte("#!/bin/sh\n# stub"))

	return Snapshot{
		Tools:      tools,
		ProjectDir: projectDir,
		Files:      ffs.Files,
		Dirs:       ffs.Dirs,
		Links:      ffs.Links,
	}, nil
}

// Invariants retorna o conjunto canonico de invariantes semanticos minimos.
func Invariants() []*Invariant {
	return []*Invariant{
		// Comuns — aplicam a toda combinacao de ferramentas
		invC01AgentsMDSchemaVersion,
		invC02AgentsMDAgentGovernanceRef,
		invC03AgentsMDEnforcementMatrix,
		invC04AgentsMDCanonicalPath,

		// Por ferramenta — presenca e referencia canonica
		invCL01ClaudeMDPresent,
		invCL02ClaudeMDCanonicalPath,
		invGM01GeminiMDPresent,
		invCP01CopilotMDPresent,
		invCD01CodexConfigPresent,
		invCD02CodexConfigCanonicalPath,

		// Claude — hooks, rules e scripts instalados
		invCL03ClaudeHookGovernancePresent,
		invCL04ClaudeHookPreloadPresent,
		invCL05ClaudeRulesGovernancePresent,
		invCL06ClaudeScriptTaskEvidencePresent,
		invCL07ClaudeScriptBugfixEvidencePresent,
		invCL08ClaudeScriptRefactorEvidencePresent,

		// Gemini — hook de preload instalado
		invGM03GeminiHookPreloadPresent,

		// Best-effort — documenta limites de enforcement
		invGM02GeminiMDBestEffortDoc,
		invCP02CopilotMDBestEffortDoc,

		// Cross-tool — detecta drift entre destinos
		invX01CrossToolCanonicalPath,
		invX02CompactProfileCodexOnly,
		invX03DepthGuardPresent,
	}
}

// ── Invariantes Comuns (AGENTS.md) ──────────────────────────────────────────

var invC01AgentsMDSchemaVersion = &Invariant{
	ID:          "C01",
	Description: "AGENTS.md e gerado com comentario de governance-schema version",
	Level:       Common,
	Check: func(s Snapshot) Result {
		c := s.File("AGENTS.md")
		if c == "" {
			return fail("AGENTS.md nao gerado")
		}
		if !strings.Contains(c, "governance-schema:") {
			return fail("AGENTS.md nao contem 'governance-schema:'")
		}
		if !strings.Contains(c, contextgen.GovernanceSchemaVersion) {
			return failf("AGENTS.md nao contem schema version %q", contextgen.GovernanceSchemaVersion)
		}
		return pass()
	},
}

var invC02AgentsMDAgentGovernanceRef = &Invariant{
	ID:          "C02",
	Description: "AGENTS.md referencia skill agent-governance como base canonica",
	Level:       Common,
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("AGENTS.md"), "agent-governance") {
			return fail("AGENTS.md nao referencia 'agent-governance'")
		}
		return pass()
	},
}

var invC03AgentsMDEnforcementMatrix = &Invariant{
	ID:          "C03",
	Description: "AGENTS.md contem matriz de enforcement por ferramenta",
	Level:       Common,
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("AGENTS.md"), "Matrix de Enforcement") {
			return fail("AGENTS.md nao contem 'Matrix de Enforcement'")
		}
		return pass()
	},
}

var invC04AgentsMDCanonicalPath = &Invariant{
	ID:          "C04",
	Description: "AGENTS.md referencia .agents/skills/ como caminho canonico",
	Level:       Common,
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("AGENTS.md"), ".agents/skills/") {
			return fail("AGENTS.md nao referencia '.agents/skills/'")
		}
		return pass()
	},
}

// ── Claude ──────────────────────────────────────────────────────────────────

var invCL01ClaudeMDPresent = &Invariant{
	ID:          "CL01",
	Description: "CLAUDE.md e gerado e menciona AGENTS.md como fonte canonica",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		c := s.File("CLAUDE.md")
		if c == "" {
			return fail("CLAUDE.md nao gerado")
		}
		if !strings.Contains(c, "AGENTS.md") {
			return fail("CLAUDE.md nao menciona AGENTS.md")
		}
		return pass()
	},
}

var invCL02ClaudeMDCanonicalPath = &Invariant{
	ID:          "CL02",
	Description: "CLAUDE.md referencia .agents/skills/ como fonte de verdade",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("CLAUDE.md"), ".agents/skills/") {
			return fail("CLAUDE.md nao referencia '.agents/skills/'")
		}
		return pass()
	},
}

// ── Gemini ──────────────────────────────────────────────────────────────────

var invGM01GeminiMDPresent = &Invariant{
	ID:          "GM01",
	Description: "GEMINI.md e gerado e menciona AGENTS.md como fonte canonica",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolGemini},
	Check: func(s Snapshot) Result {
		c := s.File("GEMINI.md")
		if c == "" {
			return fail("GEMINI.md nao gerado")
		}
		if !strings.Contains(c, "AGENTS.md") {
			return fail("GEMINI.md nao menciona AGENTS.md")
		}
		return pass()
	},
}

var invGM02GeminiMDBestEffortDoc = &Invariant{
	ID:          "GM02",
	Description: "GEMINI.md documenta ausencia de enforcement automatico",
	Level:       BestEffort,
	AppliesTo:   []skills.Tool{skills.ToolGemini},
	Check: func(s Snapshot) Result {
		c := s.File("GEMINI.md")
		if !strings.Contains(c, "Orientacoes Especificas para Gemini") {
			return fail("GEMINI.md nao contem secao de orientacoes especificas")
		}
		if !strings.Contains(c, "Nao confiar em enforcement automatico") {
			return fail("GEMINI.md nao documenta limitacao de enforcement")
		}
		return pass()
	},
}

// ── Copilot ─────────────────────────────────────────────────────────────────

var invCP01CopilotMDPresent = &Invariant{
	ID:          "CP01",
	Description: "copilot-instructions.md e gerado e menciona AGENTS.md",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCopilot},
	Check: func(s Snapshot) Result {
		c := s.File(".github/copilot-instructions.md")
		if c == "" {
			return fail("copilot-instructions.md nao gerado")
		}
		if !strings.Contains(c, "AGENTS.md") {
			return fail("copilot-instructions.md nao menciona AGENTS.md")
		}
		return pass()
	},
}

var invCP02CopilotMDBestEffortDoc = &Invariant{
	ID:          "CP02",
	Description: "copilot-instructions.md documenta ausencia de hooks de enforcement",
	Level:       BestEffort,
	AppliesTo:   []skills.Tool{skills.ToolCopilot},
	Check: func(s Snapshot) Result {
		c := s.File(".github/copilot-instructions.md")
		if !strings.Contains(c, "Orientacoes Especificas para Copilot") {
			return fail("copilot-instructions.md nao contem secao de orientacoes especificas")
		}
		if !strings.Contains(c, "Enforcement depende do modelo") {
			return fail("copilot-instructions.md nao documenta limitacao de enforcement")
		}
		return pass()
	},
}

// ── Codex ────────────────────────────────────────────────────────────────────

var invCD01CodexConfigPresent = &Invariant{
	ID:          "CD01",
	Description: ".codex/config.toml e gerado com skill agent-governance",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCodex},
	Check: func(s Snapshot) Result {
		c := s.File(".codex/config.toml")
		if c == "" {
			return fail(".codex/config.toml nao gerado")
		}
		if !strings.Contains(c, "agent-governance") {
			return fail(".codex/config.toml nao lista 'agent-governance'")
		}
		return pass()
	},
}

var invCD02CodexConfigCanonicalPath = &Invariant{
	ID:          "CD02",
	Description: ".codex/config.toml referencia .agents/skills/ como caminho de skills",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCodex},
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File(".codex/config.toml"), ".agents/skills/") {
			return fail(".codex/config.toml nao referencia '.agents/skills/'")
		}
		return pass()
	},
}

// ── Cross-tool (deteccao de drift) ───────────────────────────────────────────

// invX01CrossToolCanonicalPath verifica que todos os artefatos de todas as ferramentas
// selecionadas referenciam .agents/skills/ como caminho canonico.
// Detecta drift de caminho entre destinos sem exigir conteudo identico.
var invX01CrossToolCanonicalPath = &Invariant{
	ID:          "X01",
	Description: "Todos os artefatos de ferramenta referenciam .agents/skills/ como caminho canonico",
	Level:       Common,
	Check: func(s Snapshot) Result {
		artifacts := map[skills.Tool]string{
			skills.ToolClaude:  "CLAUDE.md",
			skills.ToolGemini:  "GEMINI.md",
			skills.ToolCopilot: ".github/copilot-instructions.md",
			skills.ToolCodex:   ".codex/config.toml",
		}
		for _, tool := range s.Tools {
			relPath, ok := artifacts[tool]
			if !ok {
				continue
			}
			content := s.File(relPath)
			if content == "" {
				return failf("artefato ausente para %s: %s", tool, relPath)
			}
			if !strings.Contains(content, ".agents/skills/") {
				return failf("artefato de %s nao referencia '.agents/skills/': %s", tool, relPath)
			}
		}
		return pass()
	},
}

// ── Claude — hooks, rules e scripts (T12) ───────────────────────────────────

var invCL03ClaudeHookGovernancePresent = &Invariant{
	ID:          "CL03",
	Description: ".claude/hooks/validate-governance.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/hooks/validate-governance.sh") == "" {
			return fail("hook validate-governance.sh ausente")
		}
		return pass()
	},
}

var invCL04ClaudeHookPreloadPresent = &Invariant{
	ID:          "CL04",
	Description: ".claude/hooks/validate-preload.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/hooks/validate-preload.sh") == "" {
			return fail("hook validate-preload.sh ausente")
		}
		return pass()
	},
}

var invCL05ClaudeRulesGovernancePresent = &Invariant{
	ID:          "CL05",
	Description: ".claude/rules/governance.md deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/rules/governance.md") == "" {
			return fail("rules governance.md ausente")
		}
		return pass()
	},
}

var invCL06ClaudeScriptTaskEvidencePresent = &Invariant{
	ID:          "CL06",
	Description: ".claude/scripts/validate-task-evidence.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-task-evidence.sh") == "" {
			return fail("script validate-task-evidence.sh ausente")
		}
		return pass()
	},
}

var invCL07ClaudeScriptBugfixEvidencePresent = &Invariant{
	ID:          "CL07",
	Description: ".claude/scripts/validate-bugfix-evidence.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-bugfix-evidence.sh") == "" {
			return fail("script validate-bugfix-evidence.sh ausente")
		}
		return pass()
	},
}

var invCL08ClaudeScriptRefactorEvidencePresent = &Invariant{
	ID:          "CL08",
	Description: ".claude/scripts/validate-refactor-evidence.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-refactor-evidence.sh") == "" {
			return fail("script validate-refactor-evidence.sh ausente")
		}
		return pass()
	},
}

// ── Gemini — hook de preload (T12) ───────────────────────────────────────────

var invGM03GeminiHookPreloadPresent = &Invariant{
	ID:          "GM03",
	Description: ".gemini/hooks/validate-preload.sh deve existir",
	Level:       BestEffort,
	AppliesTo:   []skills.Tool{skills.ToolGemini},
	Check: func(s Snapshot) Result {
		if s.File(".gemini/hooks/validate-preload.sh") == "" {
			return fail("hook Gemini validate-preload.sh ausente")
		}
		return pass()
	},
}

// ── Cross-tool — guard de profundidade (T12) ─────────────────────────────────

// invX03DepthGuardPresent verifica que o script de controle de profundidade de
// invocacao esta presente. Referenciado no AGENTS.md para todas as ferramentas.
var invX03DepthGuardPresent = &Invariant{
	ID:          "X03",
	Description: "scripts/lib/check-invocation-depth.sh deve existir",
	Level:       Common,
	AppliesTo:   nil, // aplica a todos
	Check: func(s Snapshot) Result {
		if s.File("scripts/lib/check-invocation-depth.sh") == "" {
			return fail("guard de profundidade ausente")
		}
		return pass()
	},
}

// invX02CompactProfileCodexOnly verifica que o profile compact e aplicado
// automaticamente em instalacao Codex-only, removendo secoes verbose do AGENTS.md.
// Detecta drift entre o AGENTS.md padrao (standard) e o compacto.
var invX02CompactProfileCodexOnly = &Invariant{
	ID:          "X02",
	Description: "Profile compact e aplicado em instalacao Codex-only (sem secoes verbose)",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCodex},
	Check: func(s Snapshot) Result {
		// Aplica apenas quando Codex e a unica ferramenta selecionada
		if len(s.Tools) != 1 || !s.hasTool(skills.ToolCodex) {
			return pass()
		}
		c := s.File("AGENTS.md")
		if strings.Contains(c, "## Diretrizes de Estrutura") {
			return fail("profile compact nao deve conter '## Diretrizes de Estrutura' em instalacao Codex-only")
		}
		if strings.Contains(c, "### Composicao Multi-Linguagem") {
			return fail("profile compact nao deve conter '### Composicao Multi-Linguagem' em instalacao Codex-only")
		}
		return pass()
	},
}
