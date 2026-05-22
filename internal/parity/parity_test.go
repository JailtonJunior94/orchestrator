package parity

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

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
