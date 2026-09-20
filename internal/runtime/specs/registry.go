package specs

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	scriptPreTool     = ".agents/hooks/validate-preload.sh"
	scriptPostTool    = ".agents/hooks/validate-governance.sh"
	scriptSessionEnd  = ".agents/scripts/validate-session-end.sh"
	largeBudgetAbsent = 0
)

var cliHookKeyVocabulary = map[string]map[CanonicalPoint][]string{
	"claude": {
		PointSessionStart:   {"SessionStart"},
		PointPreTool:        {"PreToolUse"},
		PointPostTool:       {"PostToolUse"},
		PointBeforeComplete: {"Stop", "SubagentStop"},
		PointSessionEnd:     {"SessionEnd"},
	},
	"codex": {
		PointSessionStart:   {"SessionStart"},
		PointPreTool:        {"PreToolUse"},
		PointPostTool:       {"PostToolUse"},
		PointBeforeComplete: {"Stop", "SubagentStop"},
		PointSessionEnd:     {"SessionEnd"},
	},
	"copilot": {
		PointSessionStart:   {"sessionStart"},
		PointPreTool:        {"preToolUse"},
		PointPostTool:       {"postToolUse"},
		PointBeforeComplete: {"agentStop"},
		PointSessionEnd:     {"sessionEnd"},
	},
	"opencode": {
		PointSessionStart:   {"event.session.created"},
		PointPreTool:        {"tool.execute.before"},
		PointPostTool:       {"tool.execute.after"},
		PointBeforeComplete: {"session.idle"},
	},
}

var agentHookArtifacts = map[string]map[CanonicalPoint]string{
	"claude": {
		PointPreTool:        ".claude/hooks/validate-preload.sh",
		PointPostTool:       ".claude/hooks/validate-governance.sh",
		PointBeforeComplete: ".claude/hooks/validate-session-end.sh",
	},
	"codex": {
		PointPreTool:        ".codex/hooks/validate-preload.sh",
		PointPostTool:       ".codex/hooks/validate-governance.sh",
		PointBeforeComplete: ".codex/hooks/validate-session-end.sh",
	},
	"copilot": {
		PointPreTool:        ".github/hooks/validate-preload.sh",
		PointPostTool:       ".github/hooks/validate-governance.sh",
		PointBeforeComplete: ".github/hooks/validate-session-end.sh",
	},
	"opencode": {
		PointPreTool:        ".agents/hooks/validate-preload.sh",
		PointPostTool:       ".agents/hooks/validate-governance.sh",
		PointBeforeComplete: ".agents/scripts/validate-session-end.sh",
	},
}

type nativeConfigFormat int

const (
	nativeConfigJSON nativeConfigFormat = iota
	nativeConfigTOML
	nativeConfigJS
)

type nativeConfigSource struct {
	path     string
	format   nativeConfigFormat
	required bool
}

var agentNativeConfigs = map[string][]nativeConfigSource{
	"claude": {
		{path: ".claude/settings.json", format: nativeConfigJSON, required: true},
		{path: ".claude/settings.local.json", format: nativeConfigJSON},
	},
	"codex": {
		{path: ".codex/config.toml", format: nativeConfigTOML, required: true},
		{path: ".codex/hooks.json", format: nativeConfigJSON},
		{path: "~/.codex/hooks.json", format: nativeConfigJSON},
		{path: "~/.codex/config.toml", format: nativeConfigTOML},
	},
	"copilot": {
		{path: ".github/hooks/governance.json", format: nativeConfigJSON, required: true},
		{path: ".github/copilot/settings.json", format: nativeConfigJSON},
	},
	"opencode": {
		{path: ".opencode/plugin/governance.js", format: nativeConfigJS, required: true},
	},
}

func NativeConfigSourcesFor(agentID string) ([]nativeConfigSource, bool) {
	sources, ok := agentNativeConfigs[agentID]
	if !ok {
		return nil, false
	}
	return slices.Clone(sources), true
}

func NativeConfigPaths(agentID string) []string {
	sources, ok := agentNativeConfigs[agentID]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(sources))
	for _, source := range sources {
		out = append(out, source.path)
	}
	return out
}

func RecognizedNativeKeys(agentID string, point CanonicalPoint) ([]string, bool) {
	byPoint, ok := cliHookKeyVocabulary[agentID]
	if !ok {
		return nil, false
	}
	keys, ok := byPoint[point]
	if !ok {
		return nil, false
	}
	return slices.Clone(keys), true
}

func InstalledArtifactPath(agentID string, point CanonicalPoint) (string, bool) {
	byPoint, ok := agentHookArtifacts[agentID]
	if !ok {
		return "", false
	}
	path, ok := byPoint[point]
	return path, ok
}

var ErrUnknownAgent = errors.New("agent not in registry")

var ErrToolNotInCatalog = errors.New("tool not in ACP catalog")

var ErrCatalogOutOfSync = errors.New("ACP catalog out of sync with agent registry")

var agentRegistry = NewCatalog().buildRegistry()

type AgentIdentity struct {
	id          string
	displayName string
	valid       bool
}

type DetectionSignals struct {
	command  string
	homeDirs []string
}

type Agent struct {
	identity       AgentIdentity
	specID         string
	enforcement    Enforcement
	signals        DetectionSignals
	envPolicy      EnvPolicy
	adrPath        string
	standardBudget int
	largeBudget    int
	valid          bool
}

func (c *Catalog) NewAgentIdentity(id, displayName string) (AgentIdentity, error) {
	if strings.TrimSpace(id) == "" {
		return AgentIdentity{}, errors.New("agent identity requires non-empty id")
	}
	if strings.TrimSpace(displayName) == "" {
		return AgentIdentity{}, fmt.Errorf("agent identity %q requires non-empty displayName", id)
	}
	return AgentIdentity{id: id, displayName: displayName, valid: true}, nil
}

func (c *Catalog) Registry() []Agent {
	return slices.Clone(agentRegistry)
}

func (c *Catalog) CanonicalOrder() []string {
	out := make([]string, 0, len(agentRegistry))
	for _, a := range agentRegistry {
		out = append(out, a.identity.id)
	}
	return out
}

func (c *Catalog) AgentByID(id string) (Agent, error) {
	for _, a := range agentRegistry {
		if a.identity.id == id {
			return a, nil
		}
	}
	return Agent{}, fmt.Errorf("%w: %q", ErrUnknownAgent, id)
}

func (c *Catalog) ACPSpecCatalog() map[string]func() Spec {
	out := make(map[string]func() Spec, len(agentRegistry))
	for _, a := range agentRegistry {
		spec, err := c.SpecOf(a)
		if err != nil {
			panic(fmt.Sprintf("ACP catalog: %v", err))
		}
		resolved := spec
		out[a.identity.id] = func() Spec { return resolved }
	}
	return out
}

func (c *Catalog) VerifyCatalogSync(externalIDs []string) error {
	order := c.CanonicalOrder()
	if len(externalIDs) != len(order) {
		return fmt.Errorf("%w: catalog has %d tools, registry has %d", ErrCatalogOutOfSync, len(externalIDs), len(order))
	}
	expected := make(map[string]bool, len(order))
	for _, id := range order {
		expected[id] = true
	}
	for _, id := range externalIDs {
		if !expected[id] {
			return fmt.Errorf("%w: tool %q not in registry", ErrCatalogOutOfSync, id)
		}
	}
	return nil
}

func (c *Catalog) ResolveACPSpec(tool string) (Spec, error) {
	agent, err := c.AgentByID(tool)
	if err != nil {
		return Spec{}, fmt.Errorf("%w: %q", ErrToolNotInCatalog, tool)
	}
	return c.SpecOf(agent)
}

func (c *Catalog) SpecOf(a Agent) (Spec, error) {
	switch a.specID {
	case "claude":
		return NewCatalog().Claude(), nil
	case "codex":
		return NewCatalog().Codex(), nil
	case "copilot":
		return NewCatalog().Copilot(), nil
	case "opencode":
		return NewCatalog().OpenCode(), nil
	default:
		return Spec{}, fmt.Errorf("%w: specID %q", ErrUnknownAgent, a.specID)
	}
}

func (c *Catalog) buildRegistry() []Agent {
	return []Agent{
		c.newAgent(
			"claude", "Claude (ACP)", "claude", "claude-agent-acp",
			[]string{".claude"},
			".specs/adr/009-acp-protocol-adoption.md",
			70000, largeBudgetAbsent,
			c.fullCliEnforcement("claude", "SessionStart", "PreToolUse", "PostToolUse", "Stop", "SessionEnd"),
		),
		c.newAgent(
			"codex", "Codex (ACP)", "codex", "codex-acp",
			[]string{".codex"},
			".specs/adr/013-codex-cli-acp-native.md",
			13000, largeBudgetAbsent,
			c.fullCliEnforcement("codex", "SessionStart", "PreToolUse", "PostToolUse", "Stop", "SessionEnd"),
			c.mustPrecondition(PreconditionTrustedHash, "register the hook hash via the Codex interactive interface before orchestrating", true),
		),
		c.newAgent(
			"copilot", "GitHub Copilot CLI (ACP)", "copilot", "copilot",
			[]string{".copilot", ".github/copilot"},
			".specs/adr/012-copilot-cli-acp-native.md",
			2000, largeBudgetAbsent,
			c.fullCliEnforcement("copilot", "sessionStart", "preToolUse", "postToolUse", "agentStop", "sessionEnd"),
			c.mustPrecondition(PreconditionTrustedFolder, "add the project folder to the Copilot CLI trusted folders list", false),
		),
		c.newAgentWithEnvPolicy(
			"opencode", "OpenCode (ACP)", "opencode", "opencode",
			[]string{".config/opencode"},
			".specs/adr/020-opencode-acp-subcomando.md",
			4000, 500_000,
			c.NewEnvPolicy(OpenCodeKillSwitchVars...),
			c.openCodeEnforcement(),
			c.mustPrecondition(PreconditionNoKillSwitch, "unset OPENCODE_PURE, OPENCODE_DISABLE_PROJECT_CONFIG, OPENCODE_DISABLE_EXTERNAL_SKILLS, OPENCODE_DISABLE_DEFAULT_PLUGINS and --pure before orchestrating", false),
			c.mustPrecondition(PreconditionHandshake, "wait for the governance plugin load handshake before the first prompt", true),
		),
	}
}

func (c *Catalog) fullCliEnforcement(agentID, sessionStartKey, preKey, postKey, beforeCompleteKey, sessionEndKey string) Enforcement {
	coverage := []PointCoverage{
		c.mustObservedCoverage(agentID, PointSessionStart, sessionStartKey),
		c.mustCoverage(agentID, PointPreTool, preKey, scriptPreTool),
		c.mustCoverage(agentID, PointPostTool, postKey, scriptPostTool),
		c.mustCoverage(agentID, PointBeforeComplete, beforeCompleteKey, scriptSessionEnd),
		c.mustObservedCoverage(agentID, PointSessionEnd, sessionEndKey),
	}
	return c.mustEnforcement(coverage)
}

const openCodeSessionStartLimitation = "OpenCode has no native session-start hook; approximated via the event bus with event.type == session.created, which fires without blocking capability"

const openCodeBeforeCompleteLimitation = "OpenCode's session.idle fires at turn end but does not block; it is observational only, unlike Stop/SubagentStop/agentStop in the other three CLIs"

const openCodeSessionEndUnsupportedReason = "OpenCode has no native session-end hook and no documented approximation for it"

func (c *Catalog) openCodeEnforcement() Enforcement {
	coverage := []PointCoverage{
		c.mustAdapterCoverage("opencode", PointSessionStart, "event.session.created", "", openCodeSessionStartLimitation),
		c.mustCoverage("opencode", PointPreTool, "tool.execute.before", scriptPreTool),
		c.mustCoverage("opencode", PointPostTool, "tool.execute.after", scriptPostTool),
		c.mustAdapterCoverage("opencode", PointBeforeComplete, "session.idle", scriptSessionEnd, openCodeBeforeCompleteLimitation),
		c.mustUnsupportedCoverage("opencode", PointSessionEnd, openCodeSessionEndUnsupportedReason),
	}
	return c.mustEnforcement(coverage)
}

func (c *Catalog) mustEnforcement(coverage []PointCoverage) Enforcement {
	enf, err := c.NewEnforcement(coverage)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid enforcement: %v", err))
	}
	return enf
}

func (c *Catalog) mustCoverage(agentID string, point CanonicalPoint, nativeKey, scriptPath string) PointCoverage {
	artifactPath, ok := InstalledArtifactPath(agentID, point)
	if !ok {
		panic(fmt.Sprintf("agent registry: agent %q declares no installed artifact for point %s", agentID, point))
	}
	cov, err := c.NewPointCoverage(agentID, point, nativeKey, scriptPath, artifactPath)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid coverage: %v", err))
	}
	return cov
}

func (c *Catalog) mustObservedCoverage(agentID string, point CanonicalPoint, nativeKey string) PointCoverage {
	cov, err := c.NewObservedPointCoverage(agentID, point, nativeKey)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid observed coverage: %v", err))
	}
	return cov
}

func (c *Catalog) mustAdapterCoverage(agentID string, point CanonicalPoint, nativeKey, scriptPath, limitation string) PointCoverage {
	var artifactPath string
	if scriptPath != "" {
		var ok bool
		artifactPath, ok = InstalledArtifactPath(agentID, point)
		if !ok {
			panic(fmt.Sprintf("agent registry: agent %q declares no installed artifact for point %s", agentID, point))
		}
	}
	cov, err := c.NewAdapterPointCoverage(agentID, point, nativeKey, scriptPath, artifactPath, limitation)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid adapter coverage: %v", err))
	}
	return cov
}

func (c *Catalog) mustUnsupportedCoverage(agentID string, point CanonicalPoint, reason string) PointCoverage {
	cov, err := c.NewUnsupportedPointCoverage(agentID, point, reason)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid unsupported coverage: %v", err))
	}
	return cov
}

func (c *Catalog) mustPrecondition(kind PreconditionKind, remedy string, requiresExec bool) EnforcementPrecondition {
	pre, err := c.NewEnforcementPrecondition(kind, remedy, requiresExec)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid precondition: %v", err))
	}
	return pre
}

func (c *Catalog) newAgent(
	id, displayName, specID, command string,
	homeDirs []string,
	adrPath string,
	standardBudget, largeBudget int,
	enf Enforcement,
	preconditions ...EnforcementPrecondition,
) Agent {
	return c.newAgentWithEnvPolicy(id, displayName, specID, command, homeDirs, adrPath, standardBudget, largeBudget, EnvPolicy{}, enf, preconditions...)
}

func (c *Catalog) newAgentWithEnvPolicy(
	id, displayName, specID, command string,
	homeDirs []string,
	adrPath string,
	standardBudget, largeBudget int,
	envPolicy EnvPolicy,
	enf Enforcement,
	preconditions ...EnforcementPrecondition,
) Agent {
	identity, err := c.NewAgentIdentity(id, displayName)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid identity: %v", err))
	}
	if len(preconditions) > 0 {
		enf, err = c.NewEnforcement(enf.Coverage(), preconditions...)
		if err != nil {
			panic(fmt.Sprintf("agent registry: invalid enforcement with preconditions: %v", err))
		}
	}
	return Agent{
		identity:       identity,
		specID:         specID,
		enforcement:    enf,
		signals:        DetectionSignals{command: command, homeDirs: slices.Clone(homeDirs)},
		envPolicy:      envPolicy,
		adrPath:        adrPath,
		standardBudget: standardBudget,
		largeBudget:    largeBudget,
		valid:          true,
	}
}

func (i AgentIdentity) ID() string { return i.id }

func (i AgentIdentity) DisplayName() string { return i.displayName }

func (i AgentIdentity) Valid() bool { return i.valid }

func (i AgentIdentity) String() string { return i.id }

func (s DetectionSignals) Command() string { return s.command }

func (s DetectionSignals) HomeDirs() []string { return slices.Clone(s.homeDirs) }

func (a Agent) Identity() AgentIdentity { return a.identity }

func (a Agent) ID() string { return a.identity.id }

func (a Agent) SpecID() string { return a.specID }

func (a Agent) Enforcement() Enforcement { return a.enforcement }

func (a Agent) Signals() DetectionSignals { return a.signals }

func (a Agent) InheritsEnv() bool { return a.envPolicy.IsZero() }

func (a Agent) EnvPolicy() EnvPolicy { return a.envPolicy }

func (a Agent) RequiresHandshake() bool {
	for _, p := range a.enforcement.preconditions {
		if p.kind == PreconditionHandshake {
			return true
		}
	}
	return false
}

func (a Agent) ADRPath() string { return a.adrPath }

func (a Agent) StandardBudget() int { return a.standardBudget }

func (a Agent) LargeBudget() int { return a.largeBudget }

func (a Agent) Valid() bool { return a.valid }

func (a Agent) Equal(other Agent) bool {
	if a.valid != other.valid ||
		a.identity != other.identity ||
		a.specID != other.specID ||
		a.envPolicy.IsZero() != other.envPolicy.IsZero() ||
		!slices.Equal(a.envPolicy.StripVars(), other.envPolicy.StripVars()) ||
		a.adrPath != other.adrPath ||
		a.standardBudget != other.standardBudget ||
		a.largeBudget != other.largeBudget ||
		a.signals.command != other.signals.command {
		return false
	}
	if !slices.Equal(a.signals.homeDirs, other.signals.homeDirs) {
		return false
	}
	return slices.EqualFunc(a.enforcement.coverage, other.enforcement.coverage, func(x, y PointCoverage) bool {
		return x == y
	})
}
