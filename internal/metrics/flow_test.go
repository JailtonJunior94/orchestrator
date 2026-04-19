package metrics

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// repeatedBytes retorna um slice de n bytes para simular artefatos de tamanho controlado.
func repeatedBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = 'a'
	}
	return b
}

// TestMeasureFlow_CompactVsStandard verifica que o limite de tokens para profile compact
// e menor que o limite para profile standard na mesma ferramenta e fluxo.
// Este e um contrato de estimativa operacional, nao tokens reais do provedor.
func TestMeasureFlow_CompactVsStandard(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	// AGENTS.md compact simula ~5.000 chars; standard simula ~10.000 chars
	ffs.Files["/repo/AGENTS.compact.md"] = repeatedBytes(5000)
	ffs.Files["/repo/AGENTS.standard.md"] = repeatedBytes(10000)
	ffs.Files["/repo/.agents/skills/agent-governance/SKILL.md"] = repeatedBytes(3000)
	ffs.Files["/repo/.agents/skills/execute-task/SKILL.md"] = repeatedBytes(3000)

	svc := NewService(ffs, silentPrinter(), nil)

	compactArtifacts := []string{
		"/repo/AGENTS.compact.md",
		"/repo/.agents/skills/agent-governance/SKILL.md",
		"/repo/.agents/skills/execute-task/SKILL.md",
	}
	standardArtifacts := []string{
		"/repo/AGENTS.standard.md",
		"/repo/.agents/skills/agent-governance/SKILL.md",
		"/repo/.agents/skills/execute-task/SKILL.md",
	}

	compact := svc.MeasureFlow("claude", GovernanceProfileCompact, SkillProfileLean, FlowExecution, compactArtifacts)
	standard := svc.MeasureFlow("claude", GovernanceProfileStandard, SkillProfileFull, FlowExecution, standardArtifacts)

	// compact deve ter limite mais baixo que standard
	if compact.TokensLimit >= standard.TokensLimit {
		t.Errorf("compact.TokensLimit (%d) deve ser menor que standard.TokensLimit (%d)", compact.TokensLimit, standard.TokensLimit)
	}
	// compact deve ter menos tokens estimados (AGENTS.md menor)
	if compact.TokensEst >= standard.TokensEst {
		t.Errorf("compact.TokensEst (%d) deve ser menor que standard.TokensEst (%d)", compact.TokensEst, standard.TokensEst)
	}
}

// TestMeasureFlow_LeanVsFull verifica que planning flow nao tem budget definido para lean profile
// (planejamento nao faz parte do lean por design) e que full profile tem budget para planning.
func TestMeasureFlow_LeanVsFull(t *testing.T) {
	_, leanHasBudget := FlowBudget("codex", GovernanceProfileCompact, SkillProfileLean, FlowPlanning)
	_, fullHasBudget := FlowBudget("codex", GovernanceProfileStandard, SkillProfileFull, FlowPlanning)

	// lean nao deve ter budget de planejamento — o perfil lean exclui skills de planejamento
	if leanHasBudget {
		t.Error("lean profile nao deve ter budget para FlowPlanning — planejamento requer full profile")
	}
	// full deve ter budget de planejamento
	if !fullHasBudget {
		t.Error("full profile deve ter budget para FlowPlanning")
	}
}

// TestMeasureFlow_TwoTools verifica que claude e codex tem limites diferentes para o mesmo fluxo,
// refletindo suas capacidades de contexto distintas.
// Esta diferenca e contrato: codex tem janela menor, portanto limite mais restritivo.
func TestMeasureFlow_TwoTools(t *testing.T) {
	claudeLimit, claudeOk := FlowBudget("claude", GovernanceProfileStandard, SkillProfileFull, FlowExecution)
	codexLimit, codexOk := FlowBudget("codex", GovernanceProfileStandard, SkillProfileFull, FlowExecution)

	if !claudeOk {
		t.Fatal("claude deve ter budget definido para execution/standard/full")
	}
	if !codexOk {
		t.Fatal("codex deve ter budget definido para execution/standard/full")
	}
	if claudeLimit <= codexLimit {
		t.Errorf("claude (%d) deve ter limite maior que codex (%d) — janela de contexto maior", claudeLimit, codexLimit)
	}
}

// TestMeasureFlow_WithinBudget verifica que artefatos leves ficam dentro do budget.
// TokensEst e estimativa operacional, nao tokens reais do provedor.
func TestMeasureFlow_WithinBudget(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	// Artefatos pequenos: 3000 + 3000 = 6000 chars => ~1714 tokens est. (bem abaixo de 20000)
	ffs.Files["/repo/AGENTS.md"] = repeatedBytes(3000)
	ffs.Files["/repo/.agents/skills/agent-governance/SKILL.md"] = repeatedBytes(3000)

	svc := NewService(ffs, silentPrinter(), nil)
	m := svc.MeasureFlow("claude", GovernanceProfileStandard, SkillProfileFull, FlowExecution, []string{
		"/repo/AGENTS.md",
		"/repo/.agents/skills/agent-governance/SKILL.md",
	})

	if !m.WithinBudget {
		t.Errorf("artefatos leves devem estar dentro do budget: est=%d limit=%d", m.TokensEst, m.TokensLimit)
	}
}

// TestMeasureFlow_ExceedsBudget verifica que artefatos excessivos excedem o budget do perfil mais restritivo.
// codex compact lean execution tem limite de 5000 tokens est.
// 20000 chars / 3.5 ~= 5714 tokens est., que excede 5000.
func TestMeasureFlow_ExceedsBudget(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	// 20000 chars => ~5714 tokens est. > limite codex compact lean (5000)
	ffs.Files["/repo/AGENTS.md"] = repeatedBytes(20000)

	svc := NewService(ffs, silentPrinter(), nil)
	m := svc.MeasureFlow("codex", GovernanceProfileCompact, SkillProfileLean, FlowExecution, []string{
		"/repo/AGENTS.md",
	})

	if m.WithinBudget {
		t.Errorf("artefatos grandes devem exceder o budget do codex compact lean: est=%d limit=%d", m.TokensEst, m.TokensLimit)
	}
}

// TestMeasureFlow_UnknownBudget verifica que combinacoes sem budget definido nao falham.
// WithinBudget deve ser true quando o budget nao esta definido.
func TestMeasureFlow_UnknownBudget(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/repo/AGENTS.md"] = repeatedBytes(100000) // qualquer tamanho

	svc := NewService(ffs, silentPrinter(), nil)
	m := svc.MeasureFlow("ferramenta-desconhecida", GovernanceProfileStandard, SkillProfileFull, FlowExecution, []string{
		"/repo/AGENTS.md",
	})

	if !m.WithinBudget {
		t.Error("combinacao sem budget definido deve retornar WithinBudget=true")
	}
	if m.TokensLimit != 0 {
		t.Errorf("combinacao sem budget deve ter TokensLimit=0, got %d", m.TokensLimit)
	}
}
