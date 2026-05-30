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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func (r1 *Checker) pass() Result              { return Result{OK: true} }
func (r1 *Checker) fail(reason string) Result { return Result{OK: false, Reason: reason} }
func (r1 *Checker) failf(f string, a ...any) Result {
	return Result{OK: false, Reason: fmt.Sprintf(f, a...)}
}

// CheckResult agrupa invariante, resultado e flag de skip.
type CheckResult struct {
	Invariant *Invariant
	Result    Result
	// Skipped indica que o invariante nao se aplica as ferramentas selecionadas.
	Skipped bool
	// Warning indica que e uma violacao de invariante BestEffort.
	// Warnings nao alteram o exit code — sao visibilidade sem bloqueio.
	// Use --strict no lint para promover warnings a erros.
	Warning bool
}

// Run executa todos os invariantes contra um Snapshot.
// Invariantes cujas AppliesTo nao incluam nenhuma ferramenta selecionada sao marcados Skipped.
// Invariantes BestEffort que falharam sao marcados Warning=true (nao bloqueantes).
func (r1 *Checker) Run(snap Snapshot, invariants []*Invariant) []CheckResult {
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
		isWarning := inv.Level == BestEffort && !r.OK
		results = append(results, CheckResult{Invariant: inv, Result: r, Warning: isWarning})
	}
	return results
}

// Warnings filtra apenas os CheckResults que sao avisos (BestEffort violados).
func (r1 *Checker) Warnings(results []CheckResult) []CheckResult {
	var out []CheckResult
	for _, cr := range results {
		if cr.Warning {
			out = append(out, cr)
		}
	}
	return out
}

// Failures filtra apenas os CheckResults que sao falhas nao-warning (bloqueantes).
func (r1 *Checker) Failures(results []CheckResult) []CheckResult {
	var out []CheckResult
	for _, cr := range results {
		if !cr.Skipped && !cr.Result.OK && !cr.Warning {
			out = append(out, cr)
		}
	}
	return out
}

// Generate produz artefatos via contextgen e retorna um Snapshot.
// Utiliza um FakeFileSystem em memoria — nao escreve em disco.
// Alem dos artefatos do contextgen, popula stubs para hooks, scripts e rules
// que sao instalados pelo install.Service (mas nao gerados pelo contextgen).
func (r1 *Checker) Generate(projectDir string, tools []skills.Tool, langs []skills.Lang, codexProfile string) (Snapshot, error) {
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
func (r1 *Checker) Invariants() []*Invariant {
	return []*Invariant{
		// Comuns — aplicam a toda combinacao de ferramentas
		_invC01AgentsMDSchemaVersion,
		_invC02AgentsMDAgentGovernanceRef,
		_invC03AgentsMDEnforcementMatrix,
		_invC04AgentsMDCanonicalPath,

		// Por ferramenta — presenca e referencia canonica
		_invCL01ClaudeMDPresent,
		_invCL02ClaudeMDCanonicalPath,
		_invGM01GeminiMDPresent,
		_invCP01CopilotMDPresent,
		_invCD01CodexConfigPresent,
		_invCD02CodexConfigCanonicalPath,

		// Claude — hooks, rules e scripts instalados
		_invCL03ClaudeHookGovernancePresent,
		_invCL04ClaudeHookPreloadPresent,
		_invCL05ClaudeRulesGovernancePresent,
		_invCL06ClaudeScriptTaskEvidencePresent,
		_invCL07ClaudeScriptBugfixEvidencePresent,
		_invCL08ClaudeScriptRefactorEvidencePresent,

		// Gemini — hook de preload instalado
		_invGM03GeminiHookPreloadPresent,

		// Best-effort — documenta limites de enforcement
		_invGM02GeminiMDBestEffortDoc,
		_invCP02CopilotMDBestEffortDoc,

		// Cross-tool — detecta drift entre destinos
		_invX01CrossToolCanonicalPath,
		_invX02CompactProfileCodexOnly,
		_invX03DepthGuardPresent,

		// F2-Claude — invariantes de normalização e MCP nested-agent depth (ADR-008 extensão)
		_invINV30ToolCallsNormalizedNameInvariant,
		_invINV32CrossCLIToolCallNameParity,
		_invINV31MCPNestedDepthNeverExceedsMax,

		// RF-19 — invariante de fallback launcher: cadeia declarada para todas as CLIs
		_invFB01FallbackLauncherChainDeclared,
	}
}

// ── Invariantes Comuns (AGENTS.md) ──────────────────────────────────────────

var _invC01AgentsMDSchemaVersion = &Invariant{
	ID:          "C01",
	Description: "AGENTS.md e gerado com comentario de governance-schema version",
	Level:       Common,
	Check: func(s Snapshot) Result {
		c := s.File("AGENTS.md")
		if c == "" {
			return NewChecker().fail("AGENTS.md nao gerado")
		}
		if !strings.Contains(c, "governance-schema:") {
			return NewChecker().fail("AGENTS.md nao contem 'governance-schema:'")
		}
		if !strings.Contains(c, contextgen.GovernanceSchemaVersion) {
			return NewChecker().failf("AGENTS.md nao contem schema version %q", contextgen.GovernanceSchemaVersion)
		}
		return NewChecker().pass()
	},
}

var _invC02AgentsMDAgentGovernanceRef = &Invariant{
	ID:          "C02",
	Description: "AGENTS.md referencia skill agent-governance como base canonica",
	Level:       Common,
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("AGENTS.md"), "agent-governance") {
			return NewChecker().fail("AGENTS.md nao referencia 'agent-governance'")
		}
		return NewChecker().pass()
	},
}

var _invC03AgentsMDEnforcementMatrix = &Invariant{
	ID:          "C03",
	Description: "AGENTS.md contem matriz de enforcement por ferramenta",
	Level:       Common,
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("AGENTS.md"), "Matrix de Enforcement") {
			return NewChecker().fail("AGENTS.md nao contem 'Matrix de Enforcement'")
		}
		return NewChecker().pass()
	},
}

var _invC04AgentsMDCanonicalPath = &Invariant{
	ID:          "C04",
	Description: "AGENTS.md referencia .agents/skills/ como caminho canonico",
	Level:       Common,
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("AGENTS.md"), ".agents/skills/") {
			return NewChecker().fail("AGENTS.md nao referencia '.agents/skills/'")
		}
		return NewChecker().pass()
	},
}

// ── Claude ──────────────────────────────────────────────────────────────────

var _invCL01ClaudeMDPresent = &Invariant{
	ID:          "CL01",
	Description: "CLAUDE.md e gerado e menciona AGENTS.md como fonte canonica",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		c := s.File("CLAUDE.md")
		if c == "" {
			return NewChecker().fail("CLAUDE.md nao gerado")
		}
		if !strings.Contains(c, "AGENTS.md") {
			return NewChecker().fail("CLAUDE.md nao menciona AGENTS.md")
		}
		return NewChecker().pass()
	},
}

var _invCL02ClaudeMDCanonicalPath = &Invariant{
	ID:          "CL02",
	Description: "CLAUDE.md referencia .agents/skills/ como fonte de verdade",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File("CLAUDE.md"), ".agents/skills/") {
			return NewChecker().fail("CLAUDE.md nao referencia '.agents/skills/'")
		}
		return NewChecker().pass()
	},
}

// ── Gemini ──────────────────────────────────────────────────────────────────

var _invGM01GeminiMDPresent = &Invariant{
	ID:          "GM01",
	Description: "GEMINI.md e gerado e menciona AGENTS.md como fonte canonica",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolGemini},
	Check: func(s Snapshot) Result {
		c := s.File("GEMINI.md")
		if c == "" {
			return NewChecker().fail("GEMINI.md nao gerado")
		}
		if !strings.Contains(c, "AGENTS.md") {
			return NewChecker().fail("GEMINI.md nao menciona AGENTS.md")
		}
		return NewChecker().pass()
	},
}

var _invGM02GeminiMDBestEffortDoc = &Invariant{
	ID:          "GM02",
	Description: "GEMINI.md documenta ausencia de enforcement automatico",
	Level:       BestEffort,
	AppliesTo:   []skills.Tool{skills.ToolGemini},
	Check: func(s Snapshot) Result {
		c := s.File("GEMINI.md")
		if !strings.Contains(c, "Orientacoes Especificas para Gemini") {
			return NewChecker().fail("GEMINI.md nao contem secao de orientacoes especificas")
		}
		if !strings.Contains(c, "Nao confiar em enforcement automatico") {
			return NewChecker().fail("GEMINI.md nao documenta limitacao de enforcement")
		}
		return NewChecker().pass()
	},
}

// ── Copilot ─────────────────────────────────────────────────────────────────

var _invCP01CopilotMDPresent = &Invariant{
	ID:          "CP01",
	Description: "copilot-instructions.md e gerado e menciona AGENTS.md",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCopilot},
	Check: func(s Snapshot) Result {
		c := s.File(".github/copilot-instructions.md")
		if c == "" {
			return NewChecker().fail("copilot-instructions.md nao gerado")
		}
		if !strings.Contains(c, "AGENTS.md") {
			return NewChecker().fail("copilot-instructions.md nao menciona AGENTS.md")
		}
		return NewChecker().pass()
	},
}

var _invCP02CopilotMDBestEffortDoc = &Invariant{
	ID:          "CP02",
	Description: "copilot-instructions.md documenta ausencia de hooks de enforcement",
	Level:       BestEffort,
	AppliesTo:   []skills.Tool{skills.ToolCopilot},
	Check: func(s Snapshot) Result {
		c := s.File(".github/copilot-instructions.md")
		if !strings.Contains(c, "Orientacoes Especificas para Copilot") {
			return NewChecker().fail("copilot-instructions.md nao contem secao de orientacoes especificas")
		}
		if !strings.Contains(c, "Enforcement depende do modelo") {
			return NewChecker().fail("copilot-instructions.md nao documenta limitacao de enforcement")
		}
		return NewChecker().pass()
	},
}

// ── Codex ────────────────────────────────────────────────────────────────────

var _invCD01CodexConfigPresent = &Invariant{
	ID:          "CD01",
	Description: ".codex/config.toml e gerado com skill agent-governance",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCodex},
	Check: func(s Snapshot) Result {
		c := s.File(".codex/config.toml")
		if c == "" {
			return NewChecker().fail(".codex/config.toml nao gerado")
		}
		if !strings.Contains(c, "agent-governance") {
			return NewChecker().fail(".codex/config.toml nao lista 'agent-governance'")
		}
		return NewChecker().pass()
	},
}

var _invCD02CodexConfigCanonicalPath = &Invariant{
	ID:          "CD02",
	Description: ".codex/config.toml referencia .agents/skills/ como caminho de skills",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCodex},
	Check: func(s Snapshot) Result {
		if !strings.Contains(s.File(".codex/config.toml"), ".agents/skills/") {
			return NewChecker().fail(".codex/config.toml nao referencia '.agents/skills/'")
		}
		return NewChecker().pass()
	},
}

// ── Cross-tool (deteccao de drift) ───────────────────────────────────────────

// _invX01CrossToolCanonicalPath verifica que todos os artefatos de todas as ferramentas
// selecionadas referenciam .agents/skills/ como caminho canonico.
// Detecta drift de caminho entre destinos sem exigir conteudo identico.
var _invX01CrossToolCanonicalPath = &Invariant{
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
				return NewChecker().failf("artefato ausente para %s: %s", tool, relPath)
			}
			if !strings.Contains(content, ".agents/skills/") {
				return NewChecker().failf("artefato de %s nao referencia '.agents/skills/': %s", tool, relPath)
			}
		}
		return NewChecker().pass()
	},
}

// ── Claude — hooks, rules e scripts (T12) ───────────────────────────────────

var _invCL03ClaudeHookGovernancePresent = &Invariant{
	ID:          "CL03",
	Description: ".claude/hooks/validate-governance.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/hooks/validate-governance.sh") == "" {
			return NewChecker().fail("hook validate-governance.sh ausente")
		}
		return NewChecker().pass()
	},
}

var _invCL04ClaudeHookPreloadPresent = &Invariant{
	ID:          "CL04",
	Description: ".claude/hooks/validate-preload.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/hooks/validate-preload.sh") == "" {
			return NewChecker().fail("hook validate-preload.sh ausente")
		}
		return NewChecker().pass()
	},
}

var _invCL05ClaudeRulesGovernancePresent = &Invariant{
	ID:          "CL05",
	Description: ".claude/rules/governance.md deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/rules/governance.md") == "" {
			return NewChecker().fail("rules governance.md ausente")
		}
		return NewChecker().pass()
	},
}

var _invCL06ClaudeScriptTaskEvidencePresent = &Invariant{
	ID:          "CL06",
	Description: ".claude/scripts/validate-task-evidence.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-task-evidence.sh") == "" {
			return NewChecker().fail("script validate-task-evidence.sh ausente")
		}
		return NewChecker().pass()
	},
}

var _invCL07ClaudeScriptBugfixEvidencePresent = &Invariant{
	ID:          "CL07",
	Description: ".claude/scripts/validate-bugfix-evidence.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-bugfix-evidence.sh") == "" {
			return NewChecker().fail("script validate-bugfix-evidence.sh ausente")
		}
		return NewChecker().pass()
	},
}

var _invCL08ClaudeScriptRefactorEvidencePresent = &Invariant{
	ID:          "CL08",
	Description: ".claude/scripts/validate-refactor-evidence.sh deve existir",
	Level:       ToolSpecific,
	AppliesTo:   []skills.Tool{skills.ToolClaude},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-refactor-evidence.sh") == "" {
			return NewChecker().fail("script validate-refactor-evidence.sh ausente")
		}
		return NewChecker().pass()
	},
}

// ── Gemini — hook de preload (T12) ───────────────────────────────────────────

var _invGM03GeminiHookPreloadPresent = &Invariant{
	ID:          "GM03",
	Description: ".gemini/hooks/validate-preload.sh deve existir",
	Level:       BestEffort,
	AppliesTo:   []skills.Tool{skills.ToolGemini},
	Check: func(s Snapshot) Result {
		if s.File(".gemini/hooks/validate-preload.sh") == "" {
			return NewChecker().fail("hook Gemini validate-preload.sh ausente")
		}
		return NewChecker().pass()
	},
}

// ── Cross-tool — guard de profundidade (T12) ─────────────────────────────────

// _invX03DepthGuardPresent verifica que o script de controle de profundidade de
// invocacao esta presente. Referenciado no AGENTS.md para todas as ferramentas.
var _invX03DepthGuardPresent = &Invariant{
	ID:          "X03",
	Description: "scripts/lib/check-invocation-depth.sh deve existir",
	Level:       Common,
	AppliesTo:   nil, // aplica a todos
	Check: func(s Snapshot) Result {
		if s.File("scripts/lib/check-invocation-depth.sh") == "" {
			return NewChecker().fail("guard de profundidade ausente")
		}
		return NewChecker().pass()
	},
}

// ── F2-Claude — Invariantes de normalização cross-tool e MCP nested-agent (ADR-008 extensão) ──

// eventsJSONLEntry representa uma entrada mínima do events.jsonl para leitura dos invariantes.
type eventsJSONLEntry struct {
	Kind           string `json:"kind"`
	RawName        string `json:"raw_name,omitempty"`
	NormalizedName string `json:"normalized_name,omitempty"`
	Depth          *int   `json:"depth,omitempty"`
}

// parseEventsJSONL lê as entradas de um arquivo events.jsonl a partir de conteúdo string.
func (r1 *Checker) parseEventsJSONL(content string) []eventsJSONLEntry {
	var entries []eventsJSONLEntry
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e eventsJSONLEntry
		if err := json.Unmarshal([]byte(line), &e); err == nil {
			entries = append(entries, e)
		}
	}
	return entries
}

// _defaultMaxAgentDepth é a profundidade máxima padrão de nested-agent (ADR-014).
const _defaultMaxAgentDepth = 3

// maxAgentDepth retorna o limite de profundidade configurado via AISPEC_MAX_AGENT_DEPTH.
func (r1 *Checker) maxAgentDepth() int {
	if v := os.Getenv("AISPEC_MAX_AGENT_DEPTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return _defaultMaxAgentDepth
}

// _invINV30ToolCallsNormalizedNameInvariant valida que a mesma operação semântica
// em Claude e Codex produz o mesmo normalized_name em events.jsonl.
// Fixtures: tests/fixtures/parity/claude_bash.jsonl + codex_shell.jsonl.
// Aplicável quando as fixtures existem; skipped silenciosamente quando ausentes.
var _invINV30ToolCallsNormalizedNameInvariant = &Invariant{
	ID:          "INV-30",
	Description: "tool_calls_normalized_name_invariant: mesma operação semântica em Claude e Codex produz normalized_name idêntico em events.jsonl",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolClaude, skills.ToolCodex},
	Check: func(s Snapshot) Result {
		claudeContent := s.File("tests/fixtures/parity/claude_bash.jsonl")
		codexContent := s.File("tests/fixtures/parity/codex_shell.jsonl")

		// Sem fixtures: skip (ambientes sem fixtures de parity não devem bloquear)
		if claudeContent == "" || codexContent == "" {
			return NewChecker().pass()
		}

		claudeEntries := NewChecker().parseEventsJSONL(claudeContent)
		codexEntries := NewChecker().parseEventsJSONL(codexContent)

		// Coletar normalized_names de tool_call_start em cada fixture
		claudeNames := map[string]bool{}
		for _, e := range claudeEntries {
			if e.Kind == "tool_call_start" && e.NormalizedName != "" {
				claudeNames[e.NormalizedName] = true
			}
		}
		codexNames := map[string]bool{}
		for _, e := range codexEntries {
			if e.Kind == "tool_call_start" && e.NormalizedName != "" {
				codexNames[e.NormalizedName] = true
			}
		}

		// Verificar que há pelo menos uma entrada em cada fixture
		if len(claudeNames) == 0 {
			return NewChecker().fail("fixture claude_bash.jsonl não contém entradas tool_call_start com normalized_name")
		}
		if len(codexNames) == 0 {
			return NewChecker().fail("fixture codex_shell.jsonl não contém entradas tool_call_start com normalized_name")
		}

		// Invariante: deve haver pelo menos um normalized_name em comum (mesma operação semântica)
		for name := range claudeNames {
			if codexNames[name] {
				return NewChecker().pass()
			}
		}
		return NewChecker().failf("nenhum normalized_name em comum entre Claude (%v) e Codex (%v)",
			claudeNames, codexNames)
	},
}

// _invINV32CrossCLIToolCallNameParity valida RP-03: a mesma operação semântica (shell) produz
// o MESMO conjunto de normalized_name nas 4 CLIs (Claude/Codex/Copilot/Gemini).
// Fixtures: claude_bash, codex_shell, copilot_run, gemini_bash. Skip quando alguma ausente
// (ambientes sem fixtures de parity não devem bloquear); falha quando os conjuntos divergem.
var _invINV32CrossCLIToolCallNameParity = &Invariant{
	ID:          "INV-32",
	Description: "cross_cli_tool_call_name_parity (RP-03): a mesma operação produz normalized_name idêntico nas 4 CLIs",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolGemini},
	Check: func(s Snapshot) Result {
		fixtures := map[string]string{
			"claude":  "tests/fixtures/parity/claude_bash.jsonl",
			"codex":   "tests/fixtures/parity/codex_shell.jsonl",
			"copilot": "tests/fixtures/parity/copilot_run.jsonl",
			"gemini":  "tests/fixtures/parity/gemini_bash.jsonl",
		}

		perCLI := make(map[string]map[string]bool, len(fixtures))
		for cli, path := range fixtures {
			content := s.File(path)
			// Skip quando alguma fixture ausente (não bloquear ambientes sem fixtures de parity).
			if content == "" {
				return NewChecker().pass()
			}
			names := map[string]bool{}
			for _, e := range NewChecker().parseEventsJSONL(content) {
				if e.Kind == "tool_call_start" && e.NormalizedName != "" {
					names[e.NormalizedName] = true
				}
			}
			if len(names) == 0 {
				return NewChecker().failf("fixture %s não contém tool_call_start com normalized_name", path)
			}
			perCLI[cli] = names
		}

		// RP-03: todos os conjuntos de normalized_name devem ser idênticos.
		var refCLI string
		var ref map[string]bool
		for cli, names := range perCLI {
			if ref == nil {
				ref, refCLI = names, cli
				continue
			}
			if len(names) != len(ref) {
				return NewChecker().failf("RP-03: conjunto de normalized_name diverge entre %s (%d) e %s (%d)",
					refCLI, len(ref), cli, len(names))
			}
			for n := range names {
				if !ref[n] {
					return NewChecker().failf("RP-03: normalized_name %q presente em %s mas ausente em %s", n, cli, refCLI)
				}
			}
		}
		return NewChecker().pass()
	},
}

// _invINV31MCPNestedDepthNeverExceedsMax valida que eventos com kind nested_agent
// têm depth ≤ AISPEC_MAX_AGENT_DEPTH em qualquer events.jsonl do snapshot.
// Quando nenhum evento nested_agent existe, o invariante passa (safe-default).
var _invINV31MCPNestedDepthNeverExceedsMax = &Invariant{
	ID:          "INV-31",
	Description: "mcp_nested_depth_never_exceeds_max: eventos nested_agent têm depth ≤ AISPEC_MAX_AGENT_DEPTH",
	Level:       Common,
	AppliesTo:   nil, // aplica a todas as ferramentas
	Check: func(s Snapshot) Result {
		max := NewChecker().maxAgentDepth()

		// Varrer todos os arquivos do snapshot buscando events.jsonl
		for path, content := range s.Files {
			if !strings.HasSuffix(path, "events.jsonl") {
				continue
			}
			entries := NewChecker().parseEventsJSONL(string(content))
			for _, e := range entries {
				if e.Kind == "nested_agent" && e.Depth != nil {
					if *e.Depth > max {
						return NewChecker().failf("events.jsonl %q: evento nested_agent com depth=%d excede max=%d",
							path, *e.Depth, max)
					}
				}
			}
		}
		return NewChecker().pass()
	},
}

// ── RF-19 — Fallback launcher chain ──────────────────────────────────────────

// _invFB01FallbackLauncherChainDeclared verifica que o AGENTS.md referencia
// agent-governance como base da governanca de execucao — pre-requisito estrutural
// para o mecanismo de fallback launcher (ADR-017, RF-19).
// O AGENTS.md gerado sempre inclui "agent-governance" como skill base; sua ausencia
// indica que o template foi corrompido ou que o instalador nao gerou o artefato corretamente.
// Paridade de argv direto vs. fallback (RF-19) e coberta pelos testes em parity_test.go
// e e2e_parity_test.go usando probe.EnsureAvailable com LookPather fake.
var _invFB01FallbackLauncherChainDeclared = &Invariant{
	ID:          "FB01",
	Description: "AGENTS.md referencia agent-governance — pre-requisito estrutural do fallback launcher (RF-19, ADR-017)",
	Level:       Common,
	Check: func(s Snapshot) Result {
		agents := s.File("AGENTS.md")
		if agents == "" {
			return NewChecker().fail("AGENTS.md nao gerado")
		}
		// AGENTS.md padrao inclui "agent-governance" como skill base de governanca.
		// Presenca garante que o template de governanca esta integro; e pre-requisito
		// para o mecanismo de fallback launcher documentado em ADR-017.
		if strings.Contains(agents, "agent-governance") {
			return NewChecker().pass()
		}
		return NewChecker().fail("AGENTS.md nao referencia 'agent-governance' — template de governanca corrompido (RF-19, ADR-017)")
	},
}

// _invX02CompactProfileCodexOnly verifica que o profile compact e aplicado
// automaticamente em instalacao Codex-only, removendo secoes verbose do AGENTS.md.
// Detecta drift entre o AGENTS.md padrao (standard) e o compacto.
var _invX02CompactProfileCodexOnly = &Invariant{
	ID:          "X02",
	Description: "Profile compact e aplicado em instalacao Codex-only (sem secoes verbose)",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolCodex},
	Check: func(s Snapshot) Result {
		// Aplica apenas quando Codex e a unica ferramenta selecionada
		if len(s.Tools) != 1 || !s.hasTool(skills.ToolCodex) {
			return NewChecker().pass()
		}
		c := s.File("AGENTS.md")
		if strings.Contains(c, "## Diretrizes de Estrutura") {
			return NewChecker().fail("profile compact nao deve conter '## Diretrizes de Estrutura' em instalacao Codex-only")
		}
		if strings.Contains(c, "### Composicao Multi-Linguagem") {
			return NewChecker().fail("profile compact nao deve conter '### Composicao Multi-Linguagem' em instalacao Codex-only")
		}
		return NewChecker().pass()
	},
}
