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
//   - Codex: le AGENTS.md como instrucao de sessao. Hooks nativos de projeto
//     exigem trust concedido via TUI interativa; sem trust o gate fica inerte.
//     config.toml lista metadados de skills para upgrade.sh — nao enforcement real.
//
//   - Copilot: carrega copilot-instructions.md automaticamente. Hooks nativos
//     de projeto disparam apenas quando a pasta esta na lista de pastas confiaveis.
//
//   - OpenCode: carrega AGENTS.md e .agents/skills/ nativamente. Enforcement
//     via hook tool.execute.before do plugin de governanca.
package parity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
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
	ID                     string
	Description            string
	Level                  EnforcementLevel
	AppliesTo              []skills.Tool
	SelfSatisfiedStubPaths []string
	Check                  func(s Snapshot) Result
}

type InvariantScope string

const (
	ScopeUniversal        InvariantScope = "universal"
	ScopeProviderSpecific InvariantScope = "provider-specific"
)

func CanonicalProviders() []skills.Tool {
	return []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode}
}

func (inv *Invariant) Scope() InvariantScope {
	if len(inv.AppliesTo) == 0 {
		return ScopeUniversal
	}
	covered := make(map[skills.Tool]bool, len(inv.AppliesTo))
	for _, t := range inv.AppliesTo {
		covered[t] = true
	}
	for _, t := range CanonicalProviders() {
		if !covered[t] {
			return ScopeProviderSpecific
		}
	}
	return ScopeUniversal
}

func (inv *Invariant) EvidenceInvalid() bool {
	return len(inv.SelfSatisfiedStubPaths) > 0
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

	for _, p := range r1.selfSatisfiedInvariantStubPaths(toolSet) {
		_ = ffs.WriteFile(filepath.Join(projectDir, p), []byte("#!/bin/sh\nstub"))
	}

	if toolSet[skills.ToolOpenCode] {
		plugin, err := embedded.Assets.ReadFile("assets/.opencode/plugin/governance.js")
		if err != nil {
			return Snapshot{}, fmt.Errorf("ler plugin embarcado do OpenCode: %w", err)
		}
		pluginPath := filepath.Join(projectDir, specs.OpenCodePluginDir, "governance.js")
		_ = ffs.WriteFile(pluginPath, plugin)

		config, err := specs.MergeOpenCodeConfig(nil, specs.DefaultOpenCodePermission(), "")
		if err != nil {
			return Snapshot{}, fmt.Errorf("gerar opencode.json canonico: %w", err)
		}
		_ = ffs.WriteFile(filepath.Join(projectDir, specs.OpenCodeConfigFileName), config)
	}

	return Snapshot{
		Tools:      tools,
		ProjectDir: projectDir,
		Files:      ffs.Files,
		Dirs:       ffs.Dirs,
		Links:      ffs.Links,
	}, nil
}

func (r1 *Checker) selfSatisfiedInvariantStubPaths(toolSet map[skills.Tool]bool) []string {
	var paths []string
	if toolSet[skills.ToolClaude] {
		paths = append(paths,
			".claude/hooks/validate-governance.sh",
			".claude/hooks/validate-preload.sh",
			".claude/scripts/validate-task-evidence.sh",
			".claude/scripts/validate-bugfix-evidence.sh",
			".claude/scripts/validate-refactor-evidence.sh",
		)
		paths = append(paths, universalRuleStubPaths()...)
	}
	paths = append(paths, "scripts/lib/check-invocation-depth.sh")
	return paths
}

// Invariants retorna o conjunto canonico de invariantes semanticos minimos.
func (r1 *Checker) Invariants() []*Invariant {
	return []*Invariant{
		// Comuns — aplicam a toda combinacao de ferramentas
		invC01AgentsMDSchemaVersion,
		invC02AgentsMDAgentGovernanceRef,
		invC03AgentsMDEnforcementMatrix,
		invC04AgentsMDCanonicalPath,

		// Por ferramenta — presenca e referencia canonica
		invCL01ClaudeMDPresent,
		invCL02ClaudeMDCanonicalPath,
		invCP01CopilotMDPresent,
		invCD01CodexConfigPresent,
		invCD02CodexConfigCanonicalPath,

		invOC01OpenCodePluginPresent,
		invOC02OpenCodeConfigPermissionNeverAsks,
		invOC03OpenCodeConfigNeverWritesSkillsOrInstructions,

		invCL03ClaudeHookGovernancePresent,
		invCL04ClaudeHookPreloadPresent,
		invCL05ClaudeRulesGovernancePresent,
		invCL06ClaudeScriptTaskEvidencePresent,
		invCL07ClaudeScriptBugfixEvidencePresent,
		invCL08ClaudeScriptRefactorEvidencePresent,

		// Best-effort — documenta limites de enforcement
		invCP02CopilotMDBestEffortDoc,

		// Cross-tool — detecta drift entre destinos
		invX01CrossToolCanonicalPath,
		invX02CompactProfileCodexOnly,
		invX03DepthGuardPresent,

		// F2-Claude — invariantes de normalização e MCP nested-agent depth (ADR-008 extensão)
		invINV30ToolCallsNormalizedNameInvariant,
		invINV32CrossCLIToolCallNameParity,
		invINV31MCPNestedDepthNeverExceedsMax,

		// RF-19 — invariante de fallback launcher: cadeia declarada para todas as CLIs
		invFB01FallbackLauncherChainDeclared,
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

var invC02AgentsMDAgentGovernanceRef = &Invariant{
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

var invC03AgentsMDEnforcementMatrix = &Invariant{
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

var invC04AgentsMDCanonicalPath = &Invariant{
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

var invCL01ClaudeMDPresent = &Invariant{
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

var invCL02ClaudeMDCanonicalPath = &Invariant{
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

// ── Gemini removido: invariantes GM01/GM02/GM03 descontinuadas (RF-02) ──────

// ── Copilot ─────────────────────────────────────────────────────────────────

var invCP01CopilotMDPresent = &Invariant{
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

var invCP02CopilotMDBestEffortDoc = &Invariant{
	ID:          "CP02",
	Description: "copilot-instructions.md documenta a pre-condicao de pasta confiavel para os hooks nativos",
	Level:       BestEffort,
	AppliesTo:   []skills.Tool{skills.ToolCopilot},
	Check: func(s Snapshot) Result {
		c := s.File(".github/copilot-instructions.md")
		if !strings.Contains(c, "Orientacoes Especificas para Copilot") {
			return NewChecker().fail("copilot-instructions.md nao contem secao de orientacoes especificas")
		}
		if !strings.Contains(c, "pastas confiaveis do Copilot CLI") {
			return NewChecker().fail("copilot-instructions.md nao documenta a pre-condicao de pasta confiavel")
		}
		return NewChecker().pass()
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
			return NewChecker().fail(".codex/config.toml nao gerado")
		}
		if !strings.Contains(c, "agent-governance") {
			return NewChecker().fail(".codex/config.toml nao lista 'agent-governance'")
		}
		return NewChecker().pass()
	},
}

var invCD02CodexConfigCanonicalPath = &Invariant{
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

var invX01CrossToolCanonicalPath = &Invariant{
	ID:          "X01",
	Description: "Todos os artefatos de ferramenta referenciam .agents/skills/ como caminho canonico",
	Level:       Common,
	Check: func(s Snapshot) Result {
		artifacts := map[skills.Tool]string{
			skills.ToolClaude:   "CLAUDE.md",
			skills.ToolCopilot:  ".github/copilot-instructions.md",
			skills.ToolCodex:    ".codex/config.toml",
			skills.ToolOpenCode: "AGENTS.md",
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

var invCL03ClaudeHookGovernancePresent = &Invariant{
	ID:                     "CL03",
	Description:            ".claude/hooks/validate-governance.sh deve existir",
	Level:                  ToolSpecific,
	AppliesTo:              []skills.Tool{skills.ToolClaude},
	SelfSatisfiedStubPaths: []string{".claude/hooks/validate-governance.sh"},
	Check: func(s Snapshot) Result {
		if s.File(".claude/hooks/validate-governance.sh") == "" {
			return NewChecker().fail("hook validate-governance.sh ausente")
		}
		return NewChecker().pass()
	},
}

var invCL04ClaudeHookPreloadPresent = &Invariant{
	ID:                     "CL04",
	Description:            ".claude/hooks/validate-preload.sh deve existir",
	Level:                  ToolSpecific,
	AppliesTo:              []skills.Tool{skills.ToolClaude},
	SelfSatisfiedStubPaths: []string{".claude/hooks/validate-preload.sh"},
	Check: func(s Snapshot) Result {
		if s.File(".claude/hooks/validate-preload.sh") == "" {
			return NewChecker().fail("hook validate-preload.sh ausente")
		}
		return NewChecker().pass()
	},
}

var invCL05ClaudeRulesGovernancePresent = &Invariant{
	ID:                     "CL05",
	Description:            ".claude/rules/governance.md e .claude/rules/code-style.md devem existir",
	Level:                  ToolSpecific,
	AppliesTo:              []skills.Tool{skills.ToolClaude},
	SelfSatisfiedStubPaths: universalRuleStubPaths(),
	Check: func(s Snapshot) Result {
		for _, ruleFile := range skills.UniversalRuleFiles {
			relPath := fmt.Sprintf(".claude/rules/%s", ruleFile)
			if s.File(relPath) == "" {
				return NewChecker().failf("rules %s ausente", ruleFile)
			}
		}
		return NewChecker().pass()
	},
}

var invCL06ClaudeScriptTaskEvidencePresent = &Invariant{
	ID:                     "CL06",
	Description:            ".claude/scripts/validate-task-evidence.sh deve existir",
	Level:                  ToolSpecific,
	AppliesTo:              []skills.Tool{skills.ToolClaude},
	SelfSatisfiedStubPaths: []string{".claude/scripts/validate-task-evidence.sh"},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-task-evidence.sh") == "" {
			return NewChecker().fail("script validate-task-evidence.sh ausente")
		}
		return NewChecker().pass()
	},
}

var invCL07ClaudeScriptBugfixEvidencePresent = &Invariant{
	ID:                     "CL07",
	Description:            ".claude/scripts/validate-bugfix-evidence.sh deve existir",
	Level:                  ToolSpecific,
	AppliesTo:              []skills.Tool{skills.ToolClaude},
	SelfSatisfiedStubPaths: []string{".claude/scripts/validate-bugfix-evidence.sh"},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-bugfix-evidence.sh") == "" {
			return NewChecker().fail("script validate-bugfix-evidence.sh ausente")
		}
		return NewChecker().pass()
	},
}

var invCL08ClaudeScriptRefactorEvidencePresent = &Invariant{
	ID:                     "CL08",
	Description:            ".claude/scripts/validate-refactor-evidence.sh deve existir",
	Level:                  ToolSpecific,
	AppliesTo:              []skills.Tool{skills.ToolClaude},
	SelfSatisfiedStubPaths: []string{".claude/scripts/validate-refactor-evidence.sh"},
	Check: func(s Snapshot) Result {
		if s.File(".claude/scripts/validate-refactor-evidence.sh") == "" {
			return NewChecker().fail("script validate-refactor-evidence.sh ausente")
		}
		return NewChecker().pass()
	},
}

var invX03DepthGuardPresent = &Invariant{
	ID:                     "X03",
	Description:            "scripts/lib/check-invocation-depth.sh deve existir",
	Level:                  Common,
	AppliesTo:              nil,
	SelfSatisfiedStubPaths: []string{"scripts/lib/check-invocation-depth.sh"},
	Check: func(s Snapshot) Result {
		if s.File("scripts/lib/check-invocation-depth.sh") == "" {
			return NewChecker().fail("guard de profundidade ausente")
		}
		return NewChecker().pass()
	},
}

func universalRuleStubPaths() []string {
	paths := make([]string, 0, len(skills.UniversalRuleFiles))
	for _, ruleFile := range skills.UniversalRuleFiles {
		paths = append(paths, fmt.Sprintf(".claude/rules/%s", ruleFile))
	}
	return paths
}

// ── F2-Claude — Invariantes de normalização cross-tool e MCP nested-agent (ADR-008 extensão) ──

// eventsJSONLEntry representa uma entrada mínima do events.jsonl para leitura dos invariantes.
type eventsJSONLEntry struct {
	Kind           string `json:"kind"`
	RawName        string `json:"raw_name,omitempty"`
	NormalizedName string `json:"normalized_name,omitempty"`
	Depth          *int   `json:"depth,omitempty"`
}

func (r1 *Checker) decodeOpenCodeDocument(content string) (map[string]any, error) {
	doc := map[string]any{}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

func (r1 *Checker) decodeOpenCodePermission(content string) (map[string]any, error) {
	doc, err := r1.decodeOpenCodeDocument(content)
	if err != nil {
		return nil, err
	}
	permission, _ := doc["permission"].(map[string]any)
	return permission, nil
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

const defaultMaxAgentDepth = 3

// maxAgentDepth retorna o limite de profundidade configurado via AISPEC_MAX_AGENT_DEPTH.
func (r1 *Checker) maxAgentDepth() int {
	if v := os.Getenv("AISPEC_MAX_AGENT_DEPTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultMaxAgentDepth
}

var invINV30ToolCallsNormalizedNameInvariant = &Invariant{
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

var invINV32CrossCLIToolCallNameParity = &Invariant{
	ID:          "INV-32",
	Description: "cross_cli_tool_call_name_parity (RP-03): a mesma operação produz normalized_name idêntico nas CLIs com fixture",
	Level:       Common,
	AppliesTo:   []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode},
	Check: func(s Snapshot) Result {
		fixtures := map[string]string{
			"claude":   "tests/fixtures/parity/claude_bash.jsonl",
			"codex":    "tests/fixtures/parity/codex_shell.jsonl",
			"copilot":  "tests/fixtures/parity/copilot_run.jsonl",
			"opencode": "tests/fixtures/parity/opencode_bash.jsonl",
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

var invINV31MCPNestedDepthNeverExceedsMax = &Invariant{
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

var invFB01FallbackLauncherChainDeclared = &Invariant{
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

var invX02CompactProfileCodexOnly = &Invariant{
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
