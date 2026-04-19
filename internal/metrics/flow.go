package metrics

// GovernanceProfile representa o perfil de governanca (compacidade do AGENTS.md).
// compact: sections verbose removidas, menor footprint de tokens.
// standard: AGENTS.md completo com todas as sections.
type GovernanceProfile string

const (
	GovernanceProfileCompact  GovernanceProfile = "compact"
	GovernanceProfileStandard GovernanceProfile = "standard"
)

// SkillProfile representa o perfil de carregamento de skills.
// lean: exclui skills de planejamento (analyze-project, create-prd, etc.).
// full: inclui todas as skills, inclusive planejamento.
type SkillProfile string

const (
	SkillProfileLean SkillProfile = "lean"
	SkillProfileFull SkillProfile = "full"
)

// FlowKind representa o fluxo operacional canonico medido.
type FlowKind string

const (
	// FlowExecution: agent-governance + skill operacional da tarefa
	FlowExecution FlowKind = "execution"
	// FlowPlanning: agent-governance + analyze-project + create-prd + create-technical-specification + create-tasks
	FlowPlanning FlowKind = "planning"
	// FlowReview: agent-governance + review
	FlowReview FlowKind = "review"
	// FlowBugfix: agent-governance + bugfix
	FlowBugfix FlowKind = "bugfix"
)

// FlowMeasurement e uma estimativa operacional de custo de tokens para uma
// combinacao especifica de ferramenta/profile/fluxo.
type FlowMeasurement struct {
	Tool              string            `json:"tool"`
	GovernanceProfile GovernanceProfile `json:"governance_profile"`
	SkillProfile      SkillProfile      `json:"skill_profile"`
	Flow              FlowKind          `json:"flow"`
	ArtifactPaths     []string          `json:"artifact_paths"`
	// TokensEst e uma estimativa operacional (chars/3.5), nao tokens reais do provedor.
	TokensEst    int  `json:"tokens_est"`
	TokensLimit  int  `json:"tokens_limit"`
	WithinBudget bool `json:"within_budget"`
}

type flowBudgetKey struct {
	Tool              string
	GovernanceProfile GovernanceProfile
	SkillProfile      SkillProfile
	Flow              FlowKind
}

// flowBudgets define thresholds conservadores de tokens por fluxo/ferramenta/profile.
// Todos os valores sao estimativas operacionais (chars/3.5), nao tokens reais do provedor.
//
// Origem dos thresholds:
//   - AGENTS.md compact: ~5.000 chars (~1.430 tokens est.) — sections verbose removidas
//   - AGENTS.md standard: ~10.000 chars (~2.860 tokens est.) — AGENTS.md completo
//   - SKILL.md medio: ~3.000 chars (~860 tokens est.) por skill
//   - referencias: ~2.000 chars (~570 tokens est.) por referencia
//
// Limites definidos como ~4-8x a soma esperada dos artefatos para margem operacional segura.
// Ferramentas com janela de contexto maior (claude, gemini) recebem limites mais generosos.
// Planning nao e definido para lean — o perfil lean exclui skills de planejamento por design.
var flowBudgets = map[flowBudgetKey]int{
	// claude — janela de contexto grande; standard/full
	{Tool: "claude", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowExecution}: 20000,
	{Tool: "claude", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowPlanning}:  40000,
	{Tool: "claude", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowReview}:    20000,
	{Tool: "claude", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowBugfix}:    20000,
	// claude — compact/lean: sections verbose removidas, planejamento fora de escopo
	{Tool: "claude", GovernanceProfile: GovernanceProfileCompact, SkillProfile: SkillProfileLean, Flow: FlowExecution}: 12000,
	{Tool: "claude", GovernanceProfile: GovernanceProfileCompact, SkillProfile: SkillProfileLean, Flow: FlowReview}:    12000,
	{Tool: "claude", GovernanceProfile: GovernanceProfileCompact, SkillProfile: SkillProfileLean, Flow: FlowBugfix}:    12000,

	// codex — contexto mais restrito; standard/full
	{Tool: "codex", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowExecution}: 8000,
	{Tool: "codex", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowPlanning}:  15000,
	{Tool: "codex", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowReview}:    8000,
	{Tool: "codex", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowBugfix}:    8000,
	// codex — compact/lean: perfil mais restritivo, planejamento fora de escopo
	{Tool: "codex", GovernanceProfile: GovernanceProfileCompact, SkillProfile: SkillProfileLean, Flow: FlowExecution}: 5000,
	{Tool: "codex", GovernanceProfile: GovernanceProfileCompact, SkillProfile: SkillProfileLean, Flow: FlowReview}:    5000,
	{Tool: "codex", GovernanceProfile: GovernanceProfileCompact, SkillProfile: SkillProfileLean, Flow: FlowBugfix}:    5000,

	// gemini — janela grande; standard/full
	{Tool: "gemini", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowExecution}: 15000,
	{Tool: "gemini", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowPlanning}:  30000,
	{Tool: "gemini", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowReview}:    15000,
	{Tool: "gemini", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowBugfix}:    15000,

	// copilot — contexto intermediario; standard/full
	{Tool: "copilot", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowExecution}: 8000,
	{Tool: "copilot", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowPlanning}:  15000,
	{Tool: "copilot", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowReview}:    8000,
	{Tool: "copilot", GovernanceProfile: GovernanceProfileStandard, SkillProfile: SkillProfileFull, Flow: FlowBugfix}:    8000,
}

// FlowBudget retorna o limite de tokens para uma combinacao de fluxo/ferramenta/profile.
// Retorna 0 e false se a combinacao nao tiver budget definido (ex: lean + planning).
func FlowBudget(tool string, gp GovernanceProfile, sp SkillProfile, flow FlowKind) (int, bool) {
	limit, ok := flowBudgets[flowBudgetKey{Tool: tool, GovernanceProfile: gp, SkillProfile: sp, Flow: flow}]
	return limit, ok
}

// MeasureFlow computa a estimativa operacional de tokens para um fluxo real, lendo
// os artefatos do sistema de arquivos. TokensEst e uma estimativa (chars/3.5),
// nao uma contagem real de tokens do provedor.
// Se a combinacao nao tiver budget definido, WithinBudget e true (budget desconhecido nao falha).
func (s *Service) MeasureFlow(tool string, gp GovernanceProfile, sp SkillProfile, flow FlowKind, artifactPaths []string) FlowMeasurement {
	m := FlowMeasurement{
		Tool:              tool,
		GovernanceProfile: gp,
		SkillProfile:      sp,
		Flow:              flow,
		ArtifactPaths:     artifactPaths,
	}
	for _, p := range artifactPaths {
		fm := s.fileMetric(p)
		m.TokensEst += fm.TokensEst
	}
	m.TokensLimit, _ = FlowBudget(tool, gp, sp, flow)
	m.WithinBudget = m.TokensLimit == 0 || m.TokensEst <= m.TokensLimit
	return m
}
