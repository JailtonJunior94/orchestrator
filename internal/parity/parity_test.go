package parity

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/probe"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// cloneFiles retorna uma copia rasa do mapa de arquivos para uso em testes de ausencia.
func cloneFiles(orig map[string][]byte) map[string][]byte {
	out := make(map[string][]byte, len(orig))
	for k, v := range orig {
		out[k] = v
	}
	return out
}

const testProjectDir = "/project"

// runAllInvariants executa o harness completo e reporta falhas.
// Invariantes Common e ToolSpecific causam t.Error; BestEffort causam t.Log.
func runAllInvariants(t *testing.T, tools []skills.Tool, codexProfile string) {
	t.Helper()
	snap, err := Generate(testProjectDir, tools, nil, codexProfile)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	assertInvariants(t, snap, Invariants())
}

// assertInvariants verifica os invariantes sobre um snapshot ja gerado.
func assertInvariants(t *testing.T, snap Snapshot, invariants []*Invariant) {
	t.Helper()
	results := Run(snap, invariants)
	for _, cr := range results {
		if cr.Skipped {
			continue
		}
		if cr.Result.OK {
			continue
		}
		switch cr.Invariant.Level {
		case Common, ToolSpecific:
			t.Errorf("[%s] FALHOU (%s): %s — %s",
				cr.Invariant.ID, cr.Invariant.Level,
				cr.Invariant.Description, cr.Result.Reason)
		case BestEffort:
			t.Logf("[%s] AVISO (best-effort): %s — %s",
				cr.Invariant.ID,
				cr.Invariant.Description, cr.Result.Reason)
		}
	}
}

// ── Cenarios por combinacao de ferramentas ───────────────────────────────────

func TestParity_AllTools(t *testing.T) {
	runAllInvariants(t, skills.AllTools, "full")
}

func TestParity_ClaudeOnly(t *testing.T) {
	runAllInvariants(t, []skills.Tool{skills.ToolClaude}, "full")
}

func TestParity_GeminiOnly(t *testing.T) {
	runAllInvariants(t, []skills.Tool{skills.ToolGemini}, "full")
}

func TestParity_CopilotOnly(t *testing.T) {
	runAllInvariants(t, []skills.Tool{skills.ToolCopilot}, "full")
}

func TestParity_CodexOnly(t *testing.T) {
	runAllInvariants(t, []skills.Tool{skills.ToolCodex}, "full")
}

func TestParity_ClaudeAndGemini(t *testing.T) {
	runAllInvariants(t, []skills.Tool{skills.ToolClaude, skills.ToolGemini}, "full")
}

func TestParity_ClaudeAndCodex(t *testing.T) {
	runAllInvariants(t, []skills.Tool{skills.ToolClaude, skills.ToolCodex}, "full")
}

// ── Profile Codex lean ───────────────────────────────────────────────────────

func TestParity_CodexLean_ExcludesPlanningSkills(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolCodex}, nil, "lean")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	content := snap.File(".codex/config.toml")
	if content == "" {
		t.Fatal(".codex/config.toml nao gerado")
	}

	planningSkills := []string{
		"analyze-project",
		"create-prd",
		"create-technical-specification",
		"create-tasks",
	}
	for _, skill := range planningSkills {
		if strings.Contains(content, skill) {
			t.Errorf("Codex lean profile nao deve conter skill de planejamento: %s", skill)
		}
	}

	// Skills base devem estar presentes mesmo no perfil lean
	for _, skill := range []string{"agent-governance", "bugfix", "review", "refactor", "execute-task"} {
		if !strings.Contains(content, skill) {
			t.Errorf("Codex lean profile deve conter skill base: %s", skill)
		}
	}
}

func TestParity_CodexFull_IncludesPlanningSkills(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolCodex}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	content := snap.File(".codex/config.toml")
	for _, skill := range []string{"analyze-project", "create-prd", "create-technical-specification", "create-tasks"} {
		if !strings.Contains(content, skill) {
			t.Errorf("Codex full profile deve conter skill de planejamento: %s", skill)
		}
	}
}

// ── Consistencia de AGENTS.md em todos os cenarios ──────────────────────────

func TestParity_AgentsMD_Standard_ContainsVerboseSections(t *testing.T) {
	// Com mais de uma ferramenta, AGENTS.md deve usar profile standard (com secoes verbose)
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude, skills.ToolCodex}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content := snap.File("AGENTS.md")
	for _, section := range []string{"## Diretrizes de Estrutura", "### Composicao Multi-Linguagem"} {
		if !strings.Contains(content, section) {
			t.Errorf("profile standard deve conter %q em instalacao multi-ferramenta", section)
		}
	}
}

func TestParity_AgentsMD_Compact_StripsVerboseSections(t *testing.T) {
	// Com apenas Codex, AGENTS.md deve usar profile compact (sem secoes verbose)
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolCodex}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	content := snap.File("AGENTS.md")
	if strings.Contains(content, "## Diretrizes de Estrutura") {
		t.Error("profile compact nao deve conter '## Diretrizes de Estrutura'")
	}
	if strings.Contains(content, "### Composicao Multi-Linguagem") {
		t.Error("profile compact nao deve conter '### Composicao Multi-Linguagem'")
	}
	// Secoes essenciais devem permanecer
	for _, section := range []string{"## Arquitetura", "## Validacao", "## Restricoes", "## Notas por Ferramenta"} {
		if !strings.Contains(content, section) {
			t.Errorf("profile compact deve preservar secao essencial: %q", section)
		}
	}
}

// ── Deteccao de drift entre ferramentas ─────────────────────────────────────

// TestParity_DriftDetection_MissingCanonicalPath verifica que o invariante X01
// detecta quando um artefato nao referencia o caminho canonico .agents/skills/.
// Esse teste confirma que o harness identifica drift, nao apenas ausencia de arquivo.
func TestParity_DriftDetection_MissingCanonicalPath(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude, skills.ToolGemini}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Introduzir drift: substituir CLAUDE.md por versao sem referencia canonica
	claudePath := filepath.Join(testProjectDir, "CLAUDE.md")
	snap.Files[claudePath] = []byte("# Claude\nConteudo sem referencia ao caminho canonico.")

	results := Run(snap, []*Invariant{invX01CrossToolCanonicalPath})
	if len(results) == 0 {
		t.Fatal("Run retornou zero resultados")
	}

	x01 := results[0]
	if x01.Skipped {
		t.Fatal("X01 nao deveria ser skipped para Claude+Gemini")
	}
	if x01.Result.OK {
		t.Error("X01 deveria detectar drift quando CLAUDE.md nao referencia '.agents/skills/'")
	}
	if !strings.Contains(x01.Result.Reason, "claude") {
		t.Errorf("mensagem de erro deveria identificar a ferramenta com drift, got: %q", x01.Result.Reason)
	}
}

// TestParity_DriftDetection_MissingArtifact verifica que o invariante X01
// reporta falha quando um artefato esperado esta ausente.
func TestParity_DriftDetection_MissingArtifact(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolGemini}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Remover GEMINI.md para simular artefato ausente
	geminiPath := filepath.Join(testProjectDir, "GEMINI.md")
	delete(snap.Files, geminiPath)

	results := Run(snap, []*Invariant{invX01CrossToolCanonicalPath})
	if len(results) == 0 {
		t.Fatal("Run retornou zero resultados")
	}
	if results[0].Result.OK {
		t.Error("X01 deveria detectar artefato ausente para Gemini")
	}
}

// ── Verificacao de que BestEffort nao bloqueia ───────────────────────────────

// TestParity_BestEffort_DoesNotBlockOnMissingDoc confirma o comportamento do harness:
// invariantes BestEffort sao verificados e reportados, mas nao classificados como
// falhas criticas. O teste garante que o harness nao confunde best-effort com common.
func TestParity_BestEffort_DoesNotBlockOnMissingDoc(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolGemini}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Remover secao de best-effort do GEMINI.md (simula geracao incompleta)
	geminiPath := filepath.Join(testProjectDir, "GEMINI.md")
	original := string(snap.Files[geminiPath])
	// Truncar no inicio da secao de orientacoes especificas
	if idx := strings.Index(original, "## Orientacoes Especificas"); idx > 0 {
		snap.Files[geminiPath] = []byte(original[:idx])
	}

	results := Run(snap, []*Invariant{invGM02GeminiMDBestEffortDoc})
	if len(results) == 0 {
		t.Fatal("Run retornou zero resultados")
	}

	cr := results[0]
	if cr.Skipped {
		t.Fatal("GM02 nao deveria ser skipped para Gemini")
	}
	// Confirmar que o nivel e BestEffort (nao Common)
	if cr.Invariant.Level != BestEffort {
		t.Errorf("GM02 deveria ter nivel BestEffort, got: %s", cr.Invariant.Level)
	}
	// O resultado pode ser falha (a secao foi removida), mas isso nao deve causar t.Error no harness
	// O teste de integracao (runAllInvariants) usa t.Log para BestEffort, nunca t.Error
}

// ── Novos artefatos T12: presenca ───────────────────────────────────────────

func TestParity_NewArtifacts_Claude_Present(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	invariants := []*Invariant{
		invCL03ClaudeHookGovernancePresent,
		invCL04ClaudeHookPreloadPresent,
		invCL05ClaudeRulesGovernancePresent,
		invCL06ClaudeScriptTaskEvidencePresent,
		invCL07ClaudeScriptBugfixEvidencePresent,
		invCL08ClaudeScriptRefactorEvidencePresent,
	}
	for _, inv := range invariants {
		r := inv.Check(snap)
		if !r.OK {
			t.Errorf("[%s] deveria passar com artefato presente: %s", inv.ID, r.Reason)
		}
	}
}

func TestParity_NewArtifacts_Claude_Absent(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	absenceTests := []struct {
		inv  *Invariant
		path string
	}{
		{invCL03ClaudeHookGovernancePresent, ".claude/hooks/validate-governance.sh"},
		{invCL04ClaudeHookPreloadPresent, ".claude/hooks/validate-preload.sh"},
		{invCL05ClaudeRulesGovernancePresent, ".claude/rules/governance.md"},
		{invCL06ClaudeScriptTaskEvidencePresent, ".claude/scripts/validate-task-evidence.sh"},
		{invCL07ClaudeScriptBugfixEvidencePresent, ".claude/scripts/validate-bugfix-evidence.sh"},
		{invCL08ClaudeScriptRefactorEvidencePresent, ".claude/scripts/validate-refactor-evidence.sh"},
	}

	for _, tc := range absenceTests {
		t.Run(tc.inv.ID, func(t *testing.T) {
			absent := Snapshot{
				Tools:      snap.Tools,
				ProjectDir: snap.ProjectDir,
				Files:      cloneFiles(snap.Files),
				Dirs:       snap.Dirs,
				Links:      snap.Links,
			}
			delete(absent.Files, filepath.Join(testProjectDir, tc.path))

			r := tc.inv.Check(absent)
			if r.OK {
				t.Errorf("[%s] deveria falhar quando artefato esta ausente", tc.inv.ID)
			}
			if tc.inv.Level != ToolSpecific {
				t.Errorf("[%s] deveria ter nivel ToolSpecific, got: %s", tc.inv.ID, tc.inv.Level)
			}
		})
	}
}

func TestParity_NewArtifacts_Gemini_HookPreload_Present(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolGemini}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	r := invGM03GeminiHookPreloadPresent.Check(snap)
	if !r.OK {
		t.Errorf("[GM03] deveria passar com hook presente: %s", r.Reason)
	}
	if invGM03GeminiHookPreloadPresent.Level != BestEffort {
		t.Errorf("[GM03] deveria ter nivel BestEffort, got: %s", invGM03GeminiHookPreloadPresent.Level)
	}
}

func TestParity_NewArtifacts_Gemini_HookPreload_Absent(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolGemini}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	absent := Snapshot{
		Tools:      snap.Tools,
		ProjectDir: snap.ProjectDir,
		Files:      cloneFiles(snap.Files),
		Dirs:       snap.Dirs,
		Links:      snap.Links,
	}
	delete(absent.Files, filepath.Join(testProjectDir, ".gemini/hooks/validate-preload.sh"))

	r := invGM03GeminiHookPreloadPresent.Check(absent)
	if r.OK {
		t.Error("[GM03] deveria falhar quando hook esta ausente")
	}
}

func TestParity_NewArtifacts_DepthGuard_Present(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	r := invX03DepthGuardPresent.Check(snap)
	if !r.OK {
		t.Errorf("[X03] deveria passar com guard presente: %s", r.Reason)
	}
	if invX03DepthGuardPresent.Level != Common {
		t.Errorf("[X03] deveria ter nivel Common, got: %s", invX03DepthGuardPresent.Level)
	}
	if invX03DepthGuardPresent.AppliesTo != nil {
		t.Errorf("[X03] AppliesTo deveria ser nil (aplica a todos), got: %v", invX03DepthGuardPresent.AppliesTo)
	}
}

func TestParity_NewArtifacts_DepthGuard_Absent(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	absent := Snapshot{
		Tools:      snap.Tools,
		ProjectDir: snap.ProjectDir,
		Files:      cloneFiles(snap.Files),
		Dirs:       snap.Dirs,
		Links:      snap.Links,
	}
	delete(absent.Files, filepath.Join(testProjectDir, "scripts/lib/check-invocation-depth.sh"))

	r := invX03DepthGuardPresent.Check(absent)
	if r.OK {
		t.Error("[X03] deveria falhar quando guard esta ausente")
	}
}

func TestParity_NewArtifacts_Gemini_Skipped_WhenClaudeOnly(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	results := Run(snap, []*Invariant{invGM03GeminiHookPreloadPresent})
	if len(results) == 0 {
		t.Fatal("Run retornou zero resultados")
	}
	if !results[0].Skipped {
		t.Error("[GM03] deveria ser skipped em instalacao Claude-only")
	}
}

// ── INV-30 e INV-31 — F2-Claude normalization + MCP nested-agent ─────────────

// TestParity_INV30_PassesWithMatchingNormalizedNames valida que INV-30 passa
// quando Claude e Codex têm o mesmo normalized_name para a mesma operação.
func TestParity_INV30_PassesWithMatchingNormalizedNames(t *testing.T) {
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolCodex},
		ProjectDir: testProjectDir,
		Files: map[string][]byte{
			testProjectDir + "/tests/fixtures/parity/claude_bash.jsonl": []byte(
				`{"kind":"tool_call_start","raw_name":"bash","normalized_name":"bash"}` + "\n",
			),
			testProjectDir + "/tests/fixtures/parity/codex_shell.jsonl": []byte(
				`{"kind":"tool_call_start","raw_name":"shell","normalized_name":"bash"}` + "\n",
			),
		},
		Dirs:  map[string]bool{},
		Links: map[string]string{},
	}
	r := invINV30ToolCallsNormalizedNameInvariant.Check(snap)
	if !r.OK {
		t.Errorf("INV-30 deveria passar com normalized_name em comum: %s", r.Reason)
	}
}

// TestParity_INV30_FailsWhenNormalizedNamesDiverge valida que INV-30 falha
// quando Claude e Codex têm normalized_names diferentes.
func TestParity_INV30_FailsWhenNormalizedNamesDiverge(t *testing.T) {
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolCodex},
		ProjectDir: testProjectDir,
		Files: map[string][]byte{
			testProjectDir + "/tests/fixtures/parity/claude_bash.jsonl": []byte(
				`{"kind":"tool_call_start","raw_name":"bash","normalized_name":"bash"}` + "\n",
			),
			testProjectDir + "/tests/fixtures/parity/codex_shell.jsonl": []byte(
				`{"kind":"tool_call_start","raw_name":"shell","normalized_name":"execute"}` + "\n",
			),
		},
		Dirs:  map[string]bool{},
		Links: map[string]string{},
	}
	r := invINV30ToolCallsNormalizedNameInvariant.Check(snap)
	if r.OK {
		t.Error("INV-30 deveria falhar quando normalized_names divergem entre Claude e Codex")
	}
}

// TestParity_INV30_PassesWhenFixturesAbsent valida que INV-30 passa (skipped)
// quando as fixtures estão ausentes — não bloquear ambientes sem fixtures.
func TestParity_INV30_PassesWhenFixturesAbsent(t *testing.T) {
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude, skills.ToolCodex},
		ProjectDir: testProjectDir,
		Files:      map[string][]byte{},
		Dirs:       map[string]bool{},
		Links:      map[string]string{},
	}
	r := invINV30ToolCallsNormalizedNameInvariant.Check(snap)
	if !r.OK {
		t.Errorf("INV-30 deveria passar (sem bloquear) quando fixtures ausentes: %s", r.Reason)
	}
}

// TestParity_INV30_SkippedWhenCopilotOnly valida que INV-30 é skipped
// quando apenas Copilot está selecionado (requer Claude ou Codex).
func TestParity_INV30_SkippedWhenCopilotOnly(t *testing.T) {
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolCopilot},
		ProjectDir: testProjectDir,
		Files:      map[string][]byte{},
		Dirs:       map[string]bool{},
		Links:      map[string]string{},
	}
	results := Run(snap, []*Invariant{invINV30ToolCallsNormalizedNameInvariant})
	if len(results) == 0 {
		t.Fatal("Run retornou zero resultados")
	}
	if !results[0].Skipped {
		t.Error("INV-30 deveria ser skipped quando apenas Copilot está selecionado (sem Claude nem Codex)")
	}
}

// TestParity_INV31_PassesWithNoNestedAgentEvents valida que INV-31 passa
// quando não há eventos nested_agent (safe-default).
func TestParity_INV31_PassesWithNoNestedAgentEvents(t *testing.T) {
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude},
		ProjectDir: testProjectDir,
		Files: map[string][]byte{
			testProjectDir + "/evidence/task1/events.jsonl": []byte(
				`{"kind":"tool_call_start","raw_name":"bash"}` + "\n",
			),
		},
		Dirs:  map[string]bool{},
		Links: map[string]string{},
	}
	r := invINV31MCPNestedDepthNeverExceedsMax.Check(snap)
	if !r.OK {
		t.Errorf("INV-31 deveria passar sem eventos nested_agent: %s", r.Reason)
	}
}

// TestParity_INV31_PassesWithDepthWithinLimit valida que INV-31 passa
// quando depth ≤ AISPEC_MAX_AGENT_DEPTH.
func TestParity_INV31_PassesWithDepthWithinLimit(t *testing.T) {
	depth := 2
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude},
		ProjectDir: testProjectDir,
		Files: map[string][]byte{
			testProjectDir + "/evidence/task1/events.jsonl": []byte(nestedAgentLine(depth) + "\n"),
		},
		Dirs:  map[string]bool{},
		Links: map[string]string{},
	}
	r := invINV31MCPNestedDepthNeverExceedsMax.Check(snap)
	if !r.OK {
		t.Errorf("INV-31 deveria passar com depth=%d ≤ max=3: %s", depth, r.Reason)
	}
}

// TestParity_INV31_FailsWhenDepthExceedsMax valida que INV-31 falha
// quando depth > AISPEC_MAX_AGENT_DEPTH.
func TestParity_INV31_FailsWhenDepthExceedsMax(t *testing.T) {
	depth := 5 // acima do default 3
	snap := Snapshot{
		Tools:      []skills.Tool{skills.ToolClaude},
		ProjectDir: testProjectDir,
		Files: map[string][]byte{
			testProjectDir + "/evidence/task1/events.jsonl": []byte(nestedAgentLine(depth) + "\n"),
		},
		Dirs:  map[string]bool{},
		Links: map[string]string{},
	}
	r := invINV31MCPNestedDepthNeverExceedsMax.Check(snap)
	if r.OK {
		t.Errorf("INV-31 deveria falhar com depth=%d > max=3", depth)
	}
}

// TestParity_INV31_ApliesToAllTools valida que INV-31 aplica a todas as ferramentas.
func TestParity_INV31_ApliesToAllTools(t *testing.T) {
	if invINV31MCPNestedDepthNeverExceedsMax.AppliesTo != nil {
		t.Errorf("INV-31 AppliesTo deveria ser nil (aplica a todos), got: %v",
			invINV31MCPNestedDepthNeverExceedsMax.AppliesTo)
	}
}

// nestedAgentLine cria uma linha JSON para evento nested_agent com depth dado.
func nestedAgentLine(depth int) string {
	return fmt.Sprintf(`{"kind":"nested_agent","depth":%d}`, depth)
}

// ── Invariantes skipped para ferramentas nao selecionadas ───────────────────

func TestParity_SkippedInvariants_ClaudeOnly(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	results := Run(snap, Invariants())

	// Invariantes de Gemini, Copilot e Codex devem ser skipped
	expectedSkipped := map[string]bool{
		"GM01": true, "GM02": true, "GM03": true,
		"CP01": true, "CP02": true,
		"CD01": true, "CD02": true,
	}
	skipped := make(map[string]bool)
	for _, cr := range results {
		if cr.Skipped {
			skipped[cr.Invariant.ID] = true
		}
	}
	for id := range expectedSkipped {
		if !skipped[id] {
			t.Errorf("invariante %s deveria ser skipped para instalacao Claude-only", id)
		}
	}

	// Invariantes de Claude (CL*) e comuns (C0x) nao devem ser skipped
	for _, cr := range results {
		id := cr.Invariant.ID
		isClaudeOrCommon := strings.HasPrefix(id, "CL") ||
			(len(id) >= 2 && id[0] == 'C' && id[1] >= '0' && id[1] <= '9')
		if isClaudeOrCommon && cr.Skipped {
			t.Errorf("invariante %s nao deveria ser skipped para instalacao Claude-only", id)
		}
	}
}

// ── Matriz 4×4 table-driven (RF-18) ────────────────────────────────────────

// allToolSubsets retorna todas as combinacoes nao-vazias das 4 CLIs.
// 4 tools => 2^4 - 1 = 15 subconjuntos + fullset = 16 combinacoes.
func allToolSubsets() [][]skills.Tool {
	all := skills.AllTools // [claude, gemini, codex, copilot]
	n := len(all)
	sets := make([][]skills.Tool, 0, 1<<n)
	for mask := 1; mask < (1 << n); mask++ {
		var subset []skills.Tool
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				subset = append(subset, all[i])
			}
		}
		sets = append(sets, subset)
	}
	return sets
}

// toolSubsetName produz nome legivel para o subconjunto de ferramentas.
func toolSubsetName(tools []skills.Tool) string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = string(t)
	}
	return strings.Join(names, "+")
}

// TestParity_Matrix4x4_AllSubsets executa a suíte completa de invariantes para
// cada combinacao nao-vazia das 4 CLIs (15 subconjuntos) no profile "full".
// Cobre RF-18: paridade 4x4 via table-driven com Generate+Run.
func TestParity_Matrix4x4_AllSubsets(t *testing.T) {
	subsets := allToolSubsets()
	for _, tools := range subsets {
		tools := tools // captura para goroutine
		name := toolSubsetName(tools)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			runAllInvariants(t, tools, "full")
		})
	}
}

// TestParity_Matrix4x4_CodexSubsets_CompactProfile executa combinacoes Codex-only
// com profile compact para garantir paridade em profile alternativo.
func TestParity_Matrix4x4_CodexSubsets_CompactProfile(t *testing.T) {
	tests := []struct {
		name    string
		tools   []skills.Tool
		profile string
	}{
		{"codex_only_compact", []skills.Tool{skills.ToolCodex}, "full"},
		{"codex_only_lean", []skills.Tool{skills.ToolCodex}, "lean"},
		{"claude_codex_full", []skills.Tool{skills.ToolClaude, skills.ToolCodex}, "full"},
		{"all_tools_full", skills.AllTools, "full"},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runAllInvariants(t, tc.tools, tc.profile)
		})
	}
}

// TestParity_Matrix4x4_InvariantCoverage verifica que todos os 4 grupos de
// invariantes (C*, CL*, GM*, CP*, CD*, X*, FB*) estao representados na suite.
func TestParity_Matrix4x4_InvariantCoverage(t *testing.T) {
	invariants := Invariants()

	prefixes := map[string]bool{
		"C":   false, // Common (C01-C04)
		"CL":  false, // Claude
		"GM":  false, // Gemini
		"CP":  false, // Copilot
		"CD":  false, // Codex
		"X":   false, // Cross-tool
		"INV": false, // F2-Claude
		"FB":  false, // Fallback (RF-19)
	}

	for _, inv := range invariants {
		id := inv.ID
		for prefix := range prefixes {
			if strings.HasPrefix(id, prefix) {
				prefixes[prefix] = true
			}
		}
	}

	for prefix, found := range prefixes {
		if !found {
			t.Errorf("grupo de invariantes com prefixo %q nao encontrado na suite", prefix)
		}
	}
}

// ── Invariante de fallback launcher (RF-19) ─────────────────────────────────

// fakeLookPather implementa probe.LookPather para testes unitarios.
type fakeLookPather struct {
	available map[string]string
}

func newFakeLookPather(available map[string]string) *fakeLookPather {
	return &fakeLookPather{available: available}
}

func (f *fakeLookPather) LookPath(name string) (string, error) {
	if path, ok := f.available[name]; ok {
		return path, nil
	}
	return "", errors.New("not found: " + name)
}

// TestParity_FallbackArgvParity_AllSpecs valida RF-19: para cada uma das 4 specs,
// o launcher resolvido via fallback (binario direto ausente) tem o mesmo tipo de resultado
// que o launcher direto (ambos sao "binary"); e o argv do fallback corresponde
// aos FixedArgs declarados na spec.
// Usa LookPather fake — sem dependencia de binario real no PATH.
func TestParity_FallbackArgvParity_AllSpecs(t *testing.T) {
	t.Parallel()

	const npxPath = "/usr/local/bin/npx"

	tests := []struct {
		name      string
		spec      specs.Spec
		wantKind  string
		wantCmd   string
		wantArgs  []string
	}{
		{
			name:     "claude_fallback_parity",
			spec:     specs.Claude(),
			wantKind: "binary",
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.ClaudeNpmPackage + "@" + specs.ClaudeNpmVersion},
		},
		{
			name:     "codex_fallback_parity",
			spec:     specs.Codex(),
			wantKind: "binary",
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.CodexNpmPackage + "@" + specs.CodexNpmVersion},
		},
		{
			name:     "gemini_fallback_parity",
			spec:     specs.Gemini(),
			wantKind: "binary",
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.GeminiNpmPackage + "@" + specs.GeminiNpmVersion, "--acp"},
		},
		{
			name:     "copilot_fallback_parity",
			spec:     specs.Copilot(),
			wantKind: "binary",
			wantCmd:  npxPath,
			wantArgs: []string{"--yes", specs.CopilotNpmPackage + "@" + specs.CopilotNpmVersion, "--acp"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Usar ID unico para isolar cache do probe entre sub-testes.
			sp := tc.spec
			sp.ID = tc.name + "-rf19"

			// Somente npx disponivel — binario direto ausente (RF-19).
			look := newFakeLookPather(map[string]string{"npx": npxPath})

			launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
			if err != nil {
				t.Fatalf("probe.EnsureAvailable falhou com apenas npx disponivel: %v", err)
			}
			if launcher.Kind() != tc.wantKind {
				t.Errorf("kind = %q, want %q (fallback deve ser BinaryLauncher generico — ADR-017)", launcher.Kind(), tc.wantKind)
			}
			cmd, args := launcher.Command()
			if cmd != tc.wantCmd {
				t.Errorf("command = %q, want %q", cmd, tc.wantCmd)
			}
			if !slices.Equal(args, tc.wantArgs) {
				t.Errorf("args = %v, want %v (FixedArgs do fallback devem ser preservados literalmente)", args, tc.wantArgs)
			}
		})
	}
}

// TestParity_FallbackArgvParity_DirectBinaryWins valida que quando o binario direto
// esta disponivel, ele e preferido ao fallback (ADR-017: canonico primeiro).
func TestParity_FallbackArgvParity_DirectBinaryWins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		spec        specs.Spec
		binaryName  string
		binaryPath  string
	}{
		{"claude_direct", specs.Claude(), "claude-agent-acp", "/usr/local/bin/claude-agent-acp"},
		{"codex_direct", specs.Codex(), "codex-acp", "/usr/local/bin/codex-acp"},
		{"gemini_direct", specs.Gemini(), "gemini", "/usr/local/bin/gemini"},
		{"copilot_direct", specs.Copilot(), "copilot", "/usr/local/bin/copilot"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sp := tc.spec
			sp.ID = tc.name + "-direct-rf19"

			// Ambos binario direto e npx disponíveis — direto deve vencer.
			look := newFakeLookPather(map[string]string{
				tc.binaryName: tc.binaryPath,
				"npx":         "/usr/local/bin/npx",
			})

			launcher, err := probe.EnsureAvailable(context.Background(), sp, look)
			if err != nil {
				t.Fatalf("probe.EnsureAvailable falhou: %v", err)
			}
			cmd, _ := launcher.Command()
			if cmd != tc.binaryPath {
				t.Errorf("binario direto deve vencer sobre fallback: command = %q, want %q", cmd, tc.binaryPath)
			}
		})
	}
}

// TestParity_FallbackArgvParity_NoBinaryNoFallback valida que quando nem o binario
// direto nem o fallback estao disponiveis, EnsureAvailable retorna ErrLauncherUnavailable.
func TestParity_FallbackArgvParity_NoBinaryNoFallback(t *testing.T) {
	t.Parallel()

	allSpecs := []specs.Spec{
		specs.Claude(),
		specs.Codex(),
		specs.Gemini(),
		specs.Copilot(),
	}

	for _, sp := range allSpecs {
		sp := sp
		t.Run(sp.ID+"_no_binary_no_fallback", func(t *testing.T) {
			t.Parallel()

			sp.ID = sp.ID + "-nobin-rf19"

			// Nenhum binario disponivel.
			look := newFakeLookPather(map[string]string{})

			_, err := probe.EnsureAvailable(context.Background(), sp, look)
			if err == nil {
				t.Fatal("esperava ErrLauncherUnavailable, mas nao houve erro")
			}
		})
	}
}

// TestParity_FB01_Invariant_PassesOnValidSnapshot valida que FB01 passa
// para snapshots gerados normalmente (agent-governance presente em AGENTS.md).
func TestParity_FB01_Invariant_PassesOnValidSnapshot(t *testing.T) {
	for _, tools := range [][]skills.Tool{
		{skills.ToolClaude},
		{skills.ToolCodex},
		skills.AllTools,
	} {
		tools := tools
		t.Run(toolSubsetName(tools), func(t *testing.T) {
			t.Parallel()
			snap, err := Generate(testProjectDir, tools, nil, "full")
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			r := invFB01FallbackLauncherChainDeclared.Check(snap)
			if !r.OK {
				t.Errorf("[FB01] deveria passar para snapshot valido: %s", r.Reason)
			}
		})
	}
}

// TestParity_FB01_Invariant_FailsOnCorruptedAgentsMD valida que FB01 falha
// quando AGENTS.md nao contem agent-governance (template corrompido).
func TestParity_FB01_Invariant_FailsOnCorruptedAgentsMD(t *testing.T) {
	snap, err := Generate(testProjectDir, []skills.Tool{skills.ToolClaude}, nil, "full")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Substituir AGENTS.md por conteudo corrompido (sem referencia a agent-governance).
	agentsPath := filepath.Join(testProjectDir, "AGENTS.md")
	snap.Files[agentsPath] = []byte("<!-- governance-schema: 1.0.0 -->\n# Regras\nConteudo corrompido sem skills referenciadas.")

	r := invFB01FallbackLauncherChainDeclared.Check(snap)
	if r.OK {
		t.Error("[FB01] deveria falhar quando AGENTS.md nao contem agent-governance")
	}
}

// TestParity_FB01_InvariantLevel verifica que FB01 tem nivel Common (nao BestEffort).
func TestParity_FB01_InvariantLevel(t *testing.T) {
	if invFB01FallbackLauncherChainDeclared.Level != Common {
		t.Errorf("[FB01] Level = %q, want Common", invFB01FallbackLauncherChainDeclared.Level)
	}
	if invFB01FallbackLauncherChainDeclared.AppliesTo != nil {
		t.Errorf("[FB01] AppliesTo deveria ser nil (aplica a todos), got %v", invFB01FallbackLauncherChainDeclared.AppliesTo)
	}
}
