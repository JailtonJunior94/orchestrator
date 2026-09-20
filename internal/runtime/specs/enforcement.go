package specs

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
)

type CanonicalPoint int

const (
	PointSessionStart CanonicalPoint = iota + 1
	PointPreTool
	PointPostTool
	PointBeforeComplete
	PointSessionEnd
)

var canonicalPointEvents = []struct {
	point CanonicalPoint
	event hookcontract.EventKind
	name  string
}{
	{PointSessionStart, hookcontract.EventSessionStart, "session-start"},
	{PointPreTool, hookcontract.EventBeforeTool, "pre-tool"},
	{PointPostTool, hookcontract.EventAfterTool, "post-tool"},
	{PointBeforeComplete, hookcontract.EventBeforeComplete, "before-complete"},
	{PointSessionEnd, hookcontract.EventSessionEnd, "session-end"},
}

var canonicalPoints = derivedCanonicalPoints()

func derivedCanonicalPoints() []CanonicalPoint {
	byEvent := make(map[hookcontract.EventKind]CanonicalPoint, len(canonicalPointEvents))
	for _, projection := range canonicalPointEvents {
		byEvent[projection.event] = projection.point
	}
	points := make([]CanonicalPoint, 0, len(canonicalPointEvents))
	for _, event := range hookcontract.EventKinds() {
		if point, ok := byEvent[event]; ok {
			points = append(points, point)
		}
	}
	return points
}

func canonicalPointEvent(p CanonicalPoint) (hookcontract.EventKind, bool) {
	for _, projection := range canonicalPointEvents {
		if projection.point == p {
			return projection.event, true
		}
	}
	return 0, false
}

func canonicalPointName(p CanonicalPoint) (string, bool) {
	for _, projection := range canonicalPointEvents {
		if projection.point == p {
			return projection.name, true
		}
	}
	return "", false
}

var ErrUnknownCanonicalPoint = errors.New("unknown canonical point")

var ErrIncompleteCoverage = errors.New("enforcement does not cover every canonical point")

var ErrInvalidPrecondition = errors.New("invalid enforcement precondition")

var ErrUnrecognizedNativeKey = errors.New("native key not recognized by the CLI")

type PreconditionState int

const (
	PreconditionCurrent PreconditionState = iota + 1
	PreconditionInert
	PreconditionUnknown
)

type PreconditionKind int

const (
	PreconditionTrustedFolder PreconditionKind = iota + 1
	PreconditionNoKillSwitch
	PreconditionHandshake
	PreconditionTrustedHash
)

type PointCoverage struct {
	agentID      string
	point        CanonicalPoint
	nativeKey    string
	scriptPath   string
	artifactPath string
	state        hookcontract.SupportState
	reason       string
}

type EnforcementPrecondition struct {
	kind         PreconditionKind
	remedy       string
	requiresExec bool
	valid        bool
}

type Enforcement struct {
	coverage      []PointCoverage
	preconditions []EnforcementPrecondition
	valid         bool
}

func (c *Catalog) CanonicalPoints() []CanonicalPoint {
	return slices.Clone(canonicalPoints)
}

func (c *Catalog) CanonicalPointEvent(p CanonicalPoint) (hookcontract.EventKind, bool) {
	return canonicalPointEvent(p)
}

func (c *Catalog) ParseCanonicalPoint(s string) (CanonicalPoint, error) {
	trimmed := strings.TrimSpace(s)
	for _, projection := range canonicalPointEvents {
		if projection.name == trimmed {
			return projection.point, nil
		}
	}
	return 0, fmt.Errorf("%w: %q", ErrUnknownCanonicalPoint, s)
}

func (c *Catalog) NewPointCoverage(agentID string, point CanonicalPoint, nativeKey, scriptPath, artifactPath string) (PointCoverage, error) {
	if !point.Valid() {
		return PointCoverage{}, fmt.Errorf("%w: %d", ErrUnknownCanonicalPoint, int(point))
	}
	if strings.TrimSpace(agentID) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing agent id", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(nativeKey) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing native key", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(scriptPath) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing canonical script", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(artifactPath) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing installed artifact", ErrIncompleteCoverage, point)
	}
	recognized, known := RecognizedNativeKeys(agentID, point)
	if !known {
		return PointCoverage{}, fmt.Errorf("%w: agent %q has no declared hook vocabulary for point %s", ErrUnrecognizedNativeKey, agentID, point)
	}
	if !slices.Contains(recognized, nativeKey) {
		return PointCoverage{}, fmt.Errorf("%w: agent %q point %s declares %q; %s recognizes only %v", ErrUnrecognizedNativeKey, agentID, point, nativeKey, agentID, recognized)
	}
	return PointCoverage{agentID: agentID, point: point, nativeKey: nativeKey, scriptPath: scriptPath, artifactPath: artifactPath, state: hookcontract.SupportVerified}, nil
}

func (c *Catalog) NewObservedPointCoverage(agentID string, point CanonicalPoint, nativeKey string) (PointCoverage, error) {
	if !point.Valid() {
		return PointCoverage{}, fmt.Errorf("%w: %d", ErrUnknownCanonicalPoint, int(point))
	}
	if strings.TrimSpace(agentID) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing agent id", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(nativeKey) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing native key", ErrIncompleteCoverage, point)
	}
	recognized, known := RecognizedNativeKeys(agentID, point)
	if !known {
		return PointCoverage{}, fmt.Errorf("%w: agent %q has no declared hook vocabulary for point %s", ErrUnrecognizedNativeKey, agentID, point)
	}
	if !slices.Contains(recognized, nativeKey) {
		return PointCoverage{}, fmt.Errorf("%w: agent %q point %s declares %q; %s recognizes only %v", ErrUnrecognizedNativeKey, agentID, point, nativeKey, agentID, recognized)
	}
	return PointCoverage{agentID: agentID, point: point, nativeKey: nativeKey, state: hookcontract.SupportVerified}, nil
}

func (c *Catalog) NewAdapterPointCoverage(agentID string, point CanonicalPoint, nativeKey, scriptPath, artifactPath, limitation string) (PointCoverage, error) {
	if !point.Valid() {
		return PointCoverage{}, fmt.Errorf("%w: %d", ErrUnknownCanonicalPoint, int(point))
	}
	if strings.TrimSpace(agentID) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing agent id", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(nativeKey) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing native key", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(limitation) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s declared adapter support without limitation", ErrIncompleteCoverage, point)
	}
	if (strings.TrimSpace(scriptPath) == "") != (strings.TrimSpace(artifactPath) == "") {
		return PointCoverage{}, fmt.Errorf("%w: point %s must declare both canonical script and installed artifact, or neither", ErrIncompleteCoverage, point)
	}
	recognized, known := RecognizedNativeKeys(agentID, point)
	if !known {
		return PointCoverage{}, fmt.Errorf("%w: agent %q has no declared hook vocabulary for point %s", ErrUnrecognizedNativeKey, agentID, point)
	}
	if !slices.Contains(recognized, nativeKey) {
		return PointCoverage{}, fmt.Errorf("%w: agent %q point %s declares %q; %s recognizes only %v", ErrUnrecognizedNativeKey, agentID, point, nativeKey, agentID, recognized)
	}
	return PointCoverage{agentID: agentID, point: point, nativeKey: nativeKey, scriptPath: scriptPath, artifactPath: artifactPath, state: hookcontract.SupportAdapter, reason: limitation}, nil
}

func (c *Catalog) NewUnsupportedPointCoverage(agentID string, point CanonicalPoint, reason string) (PointCoverage, error) {
	if !point.Valid() {
		return PointCoverage{}, fmt.Errorf("%w: %d", ErrUnknownCanonicalPoint, int(point))
	}
	if strings.TrimSpace(agentID) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing agent id", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(reason) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s declared unsupported without reason", ErrIncompleteCoverage, point)
	}
	return PointCoverage{agentID: agentID, point: point, state: hookcontract.SupportUnsupported, reason: reason}, nil
}

func (c *Catalog) NewEnforcementPrecondition(kind PreconditionKind, remedy string, requiresExec bool) (EnforcementPrecondition, error) {
	if kind < PreconditionTrustedFolder || kind > PreconditionTrustedHash {
		return EnforcementPrecondition{}, fmt.Errorf("%w: kind %d", ErrInvalidPrecondition, int(kind))
	}
	if strings.TrimSpace(remedy) == "" {
		return EnforcementPrecondition{}, fmt.Errorf("%w: empty actionable remedy", ErrInvalidPrecondition)
	}
	return EnforcementPrecondition{kind: kind, remedy: remedy, requiresExec: requiresExec, valid: true}, nil
}

func (c *Catalog) NewEnforcement(coverage []PointCoverage, preconditions ...EnforcementPrecondition) (Enforcement, error) {
	seen := make(map[CanonicalPoint]bool, len(canonicalPoints))
	for _, cov := range coverage {
		if !cov.point.Valid() {
			return Enforcement{}, fmt.Errorf("%w: invalid point", ErrIncompleteCoverage)
		}
		switch cov.state {
		case hookcontract.SupportUnsupported:
			if strings.TrimSpace(cov.reason) == "" {
				return Enforcement{}, fmt.Errorf("%w: point %s declared unsupported without reason", ErrIncompleteCoverage, cov.point)
			}
		case hookcontract.SupportAdapter:
			if cov.nativeKey == "" {
				return Enforcement{}, fmt.Errorf("%w: point %s missing native key", ErrIncompleteCoverage, cov.point)
			}
			if strings.TrimSpace(cov.reason) == "" {
				return Enforcement{}, fmt.Errorf("%w: point %s declared adapter support without limitation", ErrIncompleteCoverage, cov.point)
			}
			if (cov.scriptPath == "") != (cov.artifactPath == "") {
				return Enforcement{}, fmt.Errorf("%w: point %s must declare both canonical script and installed artifact, or neither", ErrIncompleteCoverage, cov.point)
			}
		default:
			if cov.nativeKey == "" {
				return Enforcement{}, fmt.Errorf("%w: point %s missing native key", ErrIncompleteCoverage, cov.point)
			}
			if (cov.scriptPath == "") != (cov.artifactPath == "") {
				return Enforcement{}, fmt.Errorf("%w: point %s must declare both canonical script and installed artifact, or neither", ErrIncompleteCoverage, cov.point)
			}
		}
		if seen[cov.point] {
			return Enforcement{}, fmt.Errorf("%w: duplicate point %s", ErrIncompleteCoverage, cov.point)
		}
		seen[cov.point] = true
	}
	for _, p := range canonicalPoints {
		if !seen[p] {
			return Enforcement{}, fmt.Errorf("%w: missing declaration for point %s", ErrIncompleteCoverage, p)
		}
	}
	for _, pre := range preconditions {
		if !pre.valid {
			return Enforcement{}, fmt.Errorf("%w: not built via NewEnforcementPrecondition", ErrInvalidPrecondition)
		}
	}
	return Enforcement{
		coverage:      slices.Clone(coverage),
		preconditions: slices.Clone(preconditions),
		valid:         true,
	}, nil
}

func (p CanonicalPoint) Valid() bool {
	return p >= PointSessionStart && p <= PointSessionEnd
}

func (p CanonicalPoint) String() string {
	if name, ok := canonicalPointName(p); ok {
		return name
	}
	return "unknown"
}

func (k PreconditionKind) String() string {
	switch k {
	case PreconditionTrustedFolder:
		return "trusted-folder"
	case PreconditionNoKillSwitch:
		return "no-kill-switch"
	case PreconditionHandshake:
		return "handshake"
	case PreconditionTrustedHash:
		return "trusted-hash"
	default:
		return "unknown"
	}
}

func (s PreconditionState) IsFailure() bool {
	return s == PreconditionInert
}

func (s PreconditionState) String() string {
	switch s {
	case PreconditionCurrent:
		return "current"
	case PreconditionInert:
		return "inert"
	case PreconditionUnknown:
		return "unknown"
	default:
		return "invalid"
	}
}

func (c PointCoverage) Point() CanonicalPoint { return c.point }

func (c PointCoverage) NativeKey() string { return c.nativeKey }

func (c PointCoverage) ScriptPath() string { return c.scriptPath }

func (c PointCoverage) ArtifactPath() string { return c.artifactPath }

func (c PointCoverage) AgentID() string { return c.agentID }

func (c PointCoverage) State() hookcontract.SupportState { return c.state }

func (c PointCoverage) Reason() string { return c.reason }

func (c PointCoverage) Limitation() string { return c.reason }

func (p EnforcementPrecondition) Kind() PreconditionKind { return p.kind }

func (p EnforcementPrecondition) Remedy() string { return p.remedy }

func (p EnforcementPrecondition) RequiresExec() bool { return p.requiresExec }

func (p EnforcementPrecondition) Valid() bool { return p.valid }

func (e Enforcement) Valid() bool { return e.valid }

func (e Enforcement) Coverage() []PointCoverage { return slices.Clone(e.coverage) }

func (e Enforcement) Preconditions() []EnforcementPrecondition { return slices.Clone(e.preconditions) }

func (e Enforcement) CoverageFor(point CanonicalPoint) (PointCoverage, bool) {
	for _, cov := range e.coverage {
		if cov.point == point {
			return cov, true
		}
	}
	return PointCoverage{}, false
}
