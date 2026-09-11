package specs

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type CanonicalPoint int

const (
	PointPreTool CanonicalPoint = iota + 1
	PointPostTool
	PointSessionEnd
)

var canonicalPoints = []CanonicalPoint{
	PointPreTool,
	PointPostTool,
	PointSessionEnd,
}

var ErrUnknownCanonicalPoint = errors.New("unknown canonical point")

var ErrIncompleteCoverage = errors.New("enforcement does not cover the three canonical points")

var ErrInvalidPrecondition = errors.New("invalid enforcement precondition")

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
	point      CanonicalPoint
	nativeKey  string
	scriptPath string
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

func (c *Catalog) ParseCanonicalPoint(s string) (CanonicalPoint, error) {
	switch strings.TrimSpace(s) {
	case "pre-tool":
		return PointPreTool, nil
	case "post-tool":
		return PointPostTool, nil
	case "session-end":
		return PointSessionEnd, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrUnknownCanonicalPoint, s)
	}
}

func (c *Catalog) NewPointCoverage(point CanonicalPoint, nativeKey, scriptPath string) (PointCoverage, error) {
	if !point.Valid() {
		return PointCoverage{}, fmt.Errorf("%w: %d", ErrUnknownCanonicalPoint, int(point))
	}
	if strings.TrimSpace(nativeKey) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing native key", ErrIncompleteCoverage, point)
	}
	if strings.TrimSpace(scriptPath) == "" {
		return PointCoverage{}, fmt.Errorf("%w: point %s missing canonical script", ErrIncompleteCoverage, point)
	}
	return PointCoverage{point: point, nativeKey: nativeKey, scriptPath: scriptPath}, nil
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
		if cov.nativeKey == "" || cov.scriptPath == "" {
			return Enforcement{}, fmt.Errorf("%w: point %s missing native key or canonical script", ErrIncompleteCoverage, cov.point)
		}
		if seen[cov.point] {
			return Enforcement{}, fmt.Errorf("%w: duplicate point %s", ErrIncompleteCoverage, cov.point)
		}
		seen[cov.point] = true
	}
	for _, p := range canonicalPoints {
		if !seen[p] {
			return Enforcement{}, fmt.Errorf("%w: missing point %s", ErrIncompleteCoverage, p)
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
	return p >= PointPreTool && p <= PointSessionEnd
}

func (p CanonicalPoint) String() string {
	switch p {
	case PointPreTool:
		return "pre-tool"
	case PointPostTool:
		return "post-tool"
	case PointSessionEnd:
		return "session-end"
	default:
		return "unknown"
	}
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
