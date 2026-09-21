package capability

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/parity"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type HookCell struct {
	Provider   string `json:"provider"`
	Event      string `json:"event"`
	Family     string `json:"family"`
	State      string `json:"state"`
	Reason     string `json:"reason,omitempty"`
	Limitation string `json:"limitation,omitempty"`
	Test       string `json:"test,omitempty"`
}

type HookMatrix struct {
	Cells []HookCell `json:"cells"`
}

const (
	testPreToolSkillGate      = "TestPreToolHookDispatchBlocksWhenSkillPrerequisiteMissing"
	testOpenCodePreToolGate   = "TestOpenCodeGovernancePluginBlocksWhenSkillPrerequisiteMissing"
	testSessionEndGate        = "TestSessionEndHookDispatchBlocksActiveTaskWithoutApprovedVerdict"
	testOpenCodeSessionEnd    = "TestOpenCodeGovernancePluginSessionIdleBlocksActiveTaskWithoutApprovedVerdict"
	reasonGitPolicyOnlyEvent  = "git-policy is dispatched only at before_tool, via validate-preload.sh delegating to git-operation-gate.sh (.claude/settings.json, .codex/config.toml, .github/hooks/governance.json, opencode tool.execute.before); no wiring exists for this family at this event"
	reasonEvidenceOnlyEvent   = "evidence-gate is dispatched only at before_complete, via validate-session-end.sh (Stop/agentStop/session.idle); no wiring exists for this family at this event"
	reasonCheckpointOnlyEvent = "checkpoint (F25) runs only inside post-execute-task.sh, reachable at before_complete for the providers that wire subagent-stop-wrapper.sh; no wiring exists for this family at this event"
	reasonQualityGateUnwired  = "quality-gate (internal/qualitygate + internal/runtime/hooks.QualityGateHook) is registered only on the internal ACP orchestrator dispatcher (PointSessionPostEnd); it is not exposed through any provider-native hook configuration (.claude/settings.json, .codex/config.toml, .github/hooks/governance.json, .opencode plugin) yet, so no cell of this family can be declared verified or adapter without repeating the V-25 tautology; wiring it into native hook configs is future work, not part of this task's scope"
	reasonTelemetryUnwired    = "telemetry has no dedicated dispatch-proven wiring yet; ai-spec hookaudit append is invoked as a best-effort side effect inside git-operation-gate.sh (guarded by command -v ai-spec, silently no-op otherwise), which is not a genuine per-cell proof; building real telemetry dispatch and its proof is the scope of task 12.0"
	reasonCheckpointNoWiring  = "no wiring found for post-execute-task.sh (checkpoint F25) at before_complete for this provider"
	reasonNoFamilyAtEvent     = "no canonical family hook is wired at this event for this provider"
)

type hookCellSpec struct {
	Family     Family
	Event      hookcontract.EventKind
	Provider   string
	State      hookcontract.SupportState
	Reason     string
	Limitation string
	Test       string
}

func declaredHookCellSpecs() []hookCellSpec {
	claude := string(skills.ToolClaude)
	codex := string(skills.ToolCodex)
	copilot := string(skills.ToolCopilot)
	opencode := string(skills.ToolOpenCode)

	return []hookCellSpec{
		{
			Family: FamilyGitPolicy, Event: hookcontract.EventBeforeTool, Provider: claude,
			State: hookcontract.SupportVerified, Test: testPreToolSkillGate + "/" + claude,
		},
		{
			Family: FamilyGitPolicy, Event: hookcontract.EventBeforeTool, Provider: codex,
			State: hookcontract.SupportVerified, Test: testPreToolSkillGate + "/" + codex,
		},
		{
			Family: FamilyGitPolicy, Event: hookcontract.EventBeforeTool, Provider: copilot,
			State: hookcontract.SupportVerified, Test: testPreToolSkillGate + "/" + copilot,
		},
		{
			Family: FamilyGitPolicy, Event: hookcontract.EventBeforeTool, Provider: opencode,
			State: hookcontract.SupportVerified, Test: testOpenCodePreToolGate,
		},
		{
			Family: FamilyEvidenceGate, Event: hookcontract.EventBeforeComplete, Provider: claude,
			State: hookcontract.SupportVerified, Test: testSessionEndGate + "/" + claude,
		},
		{
			Family: FamilyEvidenceGate, Event: hookcontract.EventBeforeComplete, Provider: codex,
			State: hookcontract.SupportVerified, Test: testSessionEndGate + "/" + codex,
		},
		{
			Family: FamilyEvidenceGate, Event: hookcontract.EventBeforeComplete, Provider: copilot,
			State: hookcontract.SupportVerified, Test: testSessionEndGate + "/" + copilot,
		},
		{
			Family: FamilyEvidenceGate, Event: hookcontract.EventBeforeComplete, Provider: opencode,
			State: hookcontract.SupportAdapter, Test: testOpenCodeSessionEnd,
			Limitation: "OpenCode's session.idle fires at turn end but does not block; evidence-gate is dispatched but purely observational there, unlike Stop/agentStop in the other three CLIs",
		},
	}
}

func defaultUnsupportedReason(family Family, event hookcontract.EventKind, provider string) string {
	switch family {
	case FamilyQualityGate:
		return reasonQualityGateUnwired
	case FamilyTelemetry:
		return reasonTelemetryUnwired
	case FamilyCheckpoint:
		if event == hookcontract.EventBeforeComplete {
			return reasonCheckpointNoWiring
		}
		return reasonCheckpointOnlyEvent
	case FamilyGitPolicy:
		return reasonGitPolicyOnlyEvent
	case FamilyEvidenceGate:
		return reasonEvidenceOnlyEvent
	default:
		return reasonNoFamilyAtEvent
	}
}

func GenerateHooks() (HookMatrix, error) {
	declared := make(map[string]hookCellSpec)
	for _, spec := range declaredHookCellSpecs() {
		declared[hookCellKey(spec.Provider, spec.Event, spec.Family)] = spec
	}

	var cells []HookCell
	for _, event := range hookcontract.EventKinds() {
		for _, family := range Families() {
			for _, provider := range canonicalHookProviders() {
				key := hookCellKey(provider, event, family)
				if spec, ok := declared[key]; ok {
					cells = append(cells, HookCell{
						Provider:   spec.Provider,
						Event:      event.String(),
						Family:     string(spec.Family),
						State:      spec.State.String(),
						Reason:     spec.Reason,
						Limitation: spec.Limitation,
						Test:       spec.Test,
					})
					continue
				}
				cells = append(cells, HookCell{
					Provider: provider,
					Event:    event.String(),
					Family:   string(family),
					State:    hookcontract.SupportUnsupported.String(),
					Reason:   defaultUnsupportedReason(family, event, provider),
				})
			}
		}
	}

	if len(cells) == 0 {
		return HookMatrix{}, fmt.Errorf("generated hook capability matrix has zero cells")
	}

	sortHookCells(cells)
	return HookMatrix{Cells: cells}, nil
}

func canonicalHookProviders() []string {
	tools := parity.CanonicalProviders()
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, string(t))
	}
	return out
}

func hookCellKey(provider string, event hookcontract.EventKind, family Family) string {
	return provider + "|" + event.String() + "|" + string(family)
}

func sortHookCells(cells []HookCell) {
	eventOrder := make(map[string]int, len(hookcontract.EventKinds()))
	for i, e := range hookcontract.EventKinds() {
		eventOrder[e.String()] = i
	}
	familyOrder := make(map[string]int, len(Families()))
	for i, f := range Families() {
		familyOrder[string(f)] = i
	}
	providerOrder := make(map[string]int, len(canonicalHookProviders()))
	for i, p := range canonicalHookProviders() {
		providerOrder[p] = i
	}

	sort.Slice(cells, func(i, j int) bool {
		if eventOrder[cells[i].Event] != eventOrder[cells[j].Event] {
			return eventOrder[cells[i].Event] < eventOrder[cells[j].Event]
		}
		if familyOrder[cells[i].Family] != familyOrder[cells[j].Family] {
			return familyOrder[cells[i].Family] < familyOrder[cells[j].Family]
		}
		return providerOrder[cells[i].Provider] < providerOrder[cells[j].Provider]
	})
}

func RenderHooksBoth(m HookMatrix) (jsonBytes []byte, markdownBytes []byte, err error) {
	jsonBytes, err = RenderHooksJSON(m)
	if err != nil {
		return nil, nil, err
	}
	return jsonBytes, RenderHooksMarkdown(m), nil
}

func RenderHooksJSON(m HookMatrix) ([]byte, error) {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal hook capability matrix: %w", err)
	}
	return append(data, '\n'), nil
}

func RenderHooksMarkdown(m HookMatrix) []byte {
	var b strings.Builder
	b.WriteString("# Hook Capability Matrix\n\n")
	b.WriteString("Gerado a partir da declaracao de capability de internal/hookcontract e da fiacao real\n")
	b.WriteString("dos scripts canonicos (RF-59). Eixo provedor x evento canonico x familia de hook,\n")
	b.WriteString("coexistente com docs/capability-matrix.md (eixo provedor x invariante de paridade),\n")
	b.WriteString("nunca unificado com ele. Nao editar a mao: rode\n")
	b.WriteString("`UPDATE_SNAPSHOTS=1 go test ./internal/capability/...` para regenerar.\n\n")
	b.WriteString("| Provider | Event | Family | State | Reason | Limitation | Test |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, c := range m.Cells {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			escapeMarkdownCell(c.Provider),
			escapeMarkdownCell(c.Event),
			escapeMarkdownCell(c.Family),
			escapeMarkdownCell(c.State),
			escapeMarkdownCell(c.Reason),
			escapeMarkdownCell(c.Limitation),
			escapeMarkdownCell(c.Test),
		)
	}
	return []byte(b.String())
}
