package specs

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	scriptPreTool     = ".agents/scripts/hook-prereq-gate.sh"
	scriptPostTool    = ".agents/hooks/post-execute-task.sh"
	scriptSessionEnd  = ".agents/scripts/validate-session-end.sh"
	largeBudgetAbsent = 0
)

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
			c.canonicalEnforcement("PreToolUse", "PostToolUse", "Stop"),
		),
		c.newAgent(
			"codex", "Codex (ACP)", "codex", "codex-acp",
			[]string{".codex"},
			".specs/adr/013-codex-cli-acp-native.md",
			13000, largeBudgetAbsent,
			c.canonicalEnforcement("PreToolUse", "PostToolUse", "Stop"),
			c.mustPrecondition(PreconditionTrustedHash, "register the hook hash via the Codex interactive interface before orchestrating", true),
		),
		c.newAgent(
			"copilot", "GitHub Copilot CLI (ACP)", "copilot", "copilot",
			[]string{".copilot", ".github/copilot"},
			".specs/adr/012-copilot-cli-acp-native.md",
			2000, largeBudgetAbsent,
			c.canonicalEnforcement("preToolUse", "postToolUse", "agentStop"),
			c.mustPrecondition(PreconditionTrustedFolder, "add the project folder to the Copilot CLI trusted folders list", false),
		),
		c.newAgentWithEnvPolicy(
			"opencode", "OpenCode (ACP)", "opencode", "opencode",
			[]string{".config/opencode"},
			".specs/prd-harness-quatro-clis-loop-aprovacao/adr-003-opencode-acp-subcomando.md",
			4000, 500_000,
			c.NewEnvPolicy(OpenCodeKillSwitchVars...),
			c.canonicalEnforcement("tool.execute.before", "tool.execute.after", "session.idle"),
			c.mustPrecondition(PreconditionNoKillSwitch, "unset OPENCODE_PURE, OPENCODE_DISABLE_PROJECT_CONFIG, OPENCODE_DISABLE_EXTERNAL_SKILLS and --pure before orchestrating", false),
			c.mustPrecondition(PreconditionHandshake, "wait for the governance plugin load handshake before the first prompt", true),
		),
	}
}

func (c *Catalog) canonicalEnforcement(preKey, postKey, endKey string) Enforcement {
	coverage := []PointCoverage{
		c.mustCoverage(PointPreTool, preKey, scriptPreTool),
		c.mustCoverage(PointPostTool, postKey, scriptPostTool),
		c.mustCoverage(PointSessionEnd, endKey, scriptSessionEnd),
	}
	enf, err := c.NewEnforcement(coverage)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid enforcement: %v", err))
	}
	return enf
}

func (c *Catalog) mustCoverage(point CanonicalPoint, nativeKey, scriptPath string) PointCoverage {
	cov, err := c.NewPointCoverage(point, nativeKey, scriptPath)
	if err != nil {
		panic(fmt.Sprintf("agent registry: invalid coverage: %v", err))
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
