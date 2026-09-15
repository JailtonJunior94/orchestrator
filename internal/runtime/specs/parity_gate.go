package specs

import (
	"bytes"
	"fmt"
)

type AgentEnforcement struct {
	Agent       string
	Enforcement Enforcement
}

type ParityViolation struct {
	Agent  string
	Point  CanonicalPoint
	Reason string
}

func (v ParityViolation) String() string {
	if v.Point.Valid() {
		return fmt.Sprintf("agent=%s point=%s reason=%s", v.Agent, v.Point, v.Reason)
	}
	return fmt.Sprintf("agent=%s reason=%s", v.Agent, v.Reason)
}

type DispatchProofFunc func(agentID string, point CanonicalPoint) bool

type ScriptResolver func(relPath string) ([]byte, error)

func ValidateParityMatrix(cells []AgentEnforcement, requiredAgents []string, dispatchProven DispatchProofFunc, resolve ScriptResolver) []ParityViolation {
	points := NewCatalog().CanonicalPoints()

	byAgent := make(map[string]Enforcement, len(cells))
	for _, cell := range cells {
		byAgent[cell.Agent] = cell.Enforcement
	}

	var violations []ParityViolation
	scriptByPoint := make(map[CanonicalPoint]string, len(points))

	for _, agentID := range requiredAgents {
		enf, ok := byAgent[agentID]
		if !ok {
			violations = append(violations, ParityViolation{Agent: agentID, Reason: "agent missing from registry"})
			continue
		}
		if !enf.Valid() {
			violations = append(violations, ParityViolation{Agent: agentID, Reason: "invalid enforcement"})
			continue
		}
		for _, point := range points {
			cov, covered := enf.CoverageFor(point)
			if !covered {
				violations = append(violations, ParityViolation{Agent: agentID, Point: point, Reason: "missing canonical point coverage"})
				continue
			}
			if prev, seen := scriptByPoint[point]; seen {
				if prev != cov.ScriptPath() {
					violations = append(violations, ParityViolation{Agent: agentID, Point: point, Reason: fmt.Sprintf("validator diverges: %q vs %q", cov.ScriptPath(), prev)})
				}
			} else {
				scriptByPoint[point] = cov.ScriptPath()
			}
			if dispatchProven == nil || !dispatchProven(agentID, point) {
				violations = append(violations, ParityViolation{Agent: agentID, Point: point, Reason: "no dispatch proof test associated"})
			}
			violations = append(violations, confrontInstalledArtifact(agentID, point, cov, resolve)...)
			violations = append(violations, confrontNativeKeyDeclaration(agentID, point, cov, resolve)...)
		}
	}
	return violations
}

func confrontInstalledArtifact(agentID string, point CanonicalPoint, cov PointCoverage, resolve ScriptResolver) []ParityViolation {
	if resolve == nil {
		return []ParityViolation{{Agent: agentID, Point: point, Reason: "declared validator never confronted with the installed artifact: no script resolver supplied"}}
	}
	canonical, err := resolve(cov.ScriptPath())
	if err != nil {
		return []ParityViolation{{Agent: agentID, Point: point, Reason: fmt.Sprintf("declared canonical validator %q does not exist on disk: %v", cov.ScriptPath(), err)}}
	}
	artifact, err := resolve(cov.ArtifactPath())
	if err != nil {
		return []ParityViolation{{Agent: agentID, Point: point, Reason: fmt.Sprintf("installed artifact %q does not exist on disk: %v", cov.ArtifactPath(), err)}}
	}
	if bytes.Equal(artifact, canonical) {
		return nil
	}
	if scriptExecutesTarget(artifact, cov.ScriptPath()) {
		return nil
	}
	return []ParityViolation{{Agent: agentID, Point: point, Reason: fmt.Sprintf("installed artifact %q neither mirrors nor executes the canonical validator %q", cov.ArtifactPath(), cov.ScriptPath())}}
}

func confrontNativeKeyDeclaration(agentID string, point CanonicalPoint, cov PointCoverage, resolve ScriptResolver) []ParityViolation {
	if resolve == nil {
		return []ParityViolation{{Agent: agentID, Point: point, Reason: fmt.Sprintf("declared native key %q never confronted with the CLI config: no resolver supplied", cov.NativeKey())}}
	}
	return confrontNativeConfig(agentID, point, cov, resolve)
}
