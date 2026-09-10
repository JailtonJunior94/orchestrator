package specs_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func coverageFixture(t *testing.T, p specs.CanonicalPoint) specs.PointCoverage {
	t.Helper()
	cov, err := specs.NewCatalog().NewPointCoverage(p, "NativeKey", ".agents/scripts/hook-prereq-gate.sh")
	if err != nil {
		t.Fatalf("NewPointCoverage(%s): %v", p, err)
	}
	return cov
}

func TestNewEnforcementRejectsIncompleteCoverage(t *testing.T) {
	t.Parallel()

	all := []specs.CanonicalPoint{
		specs.PointPreTool,
		specs.PointPostTool,
		specs.PointSessionEnd,
	}

	for _, missing := range all {
		var coverage []specs.PointCoverage
		for _, p := range all {
			if p == missing {
				continue
			}
			coverage = append(coverage, coverageFixture(t, p))
		}
		_, err := specs.NewCatalog().NewEnforcement(coverage)
		if !errors.Is(err, specs.ErrIncompleteCoverage) {
			t.Errorf("missing %s: err = %v; want ErrIncompleteCoverage", missing, err)
		}
	}

	complete := []specs.PointCoverage{
		coverageFixture(t, specs.PointPreTool),
		coverageFixture(t, specs.PointPostTool),
		coverageFixture(t, specs.PointSessionEnd),
	}
	enf, err := specs.NewCatalog().NewEnforcement(complete)
	if err != nil {
		t.Fatalf("complete coverage rejected: %v", err)
	}
	if !enf.Valid() {
		t.Fatal("complete enforcement reported invalid")
	}
}

func TestCanonicalPointIsClosedSet(t *testing.T) {
	t.Parallel()

	if specs.CanonicalPoint(0).Valid() {
		t.Fatal("zero-value CanonicalPoint reported valid")
	}
	if specs.CanonicalPoint(99).Valid() {
		t.Fatal("out-of-set value reported valid")
	}
	if _, err := specs.NewCatalog().ParseCanonicalPoint("bogus"); !errors.Is(err, specs.ErrUnknownCanonicalPoint) {
		t.Fatalf("ParseCanonicalPoint(bogus) err = %v; want ErrUnknownCanonicalPoint", err)
	}
	for _, name := range []string{"pre-tool", "post-tool", "session-end"} {
		p, err := specs.NewCatalog().ParseCanonicalPoint(name)
		if err != nil || !p.Valid() {
			t.Errorf("ParseCanonicalPoint(%q) = %v, %v", name, p, err)
		}
	}
}

func TestEnforcementPreconditionRemedyAndStates(t *testing.T) {
	t.Parallel()

	if _, err := specs.NewCatalog().NewEnforcementPrecondition(specs.PreconditionTrustedFolder, "   ", false); !errors.Is(err, specs.ErrInvalidPrecondition) {
		t.Fatalf("empty remedy accepted: %v", err)
	}
	if _, err := specs.NewCatalog().NewEnforcementPrecondition(specs.PreconditionKind(0), "x", false); !errors.Is(err, specs.ErrInvalidPrecondition) {
		t.Fatalf("zero kind accepted: %v", err)
	}
	pre, err := specs.NewCatalog().NewEnforcementPrecondition(specs.PreconditionTrustedHash, "do X", true)
	if err != nil {
		t.Fatalf("valid construction failed: %v", err)
	}
	if pre.Remedy() != "do X" || !pre.RequiresExec() {
		t.Fatalf("precondition did not preserve fields: %+v", pre)
	}

	if !specs.PreconditionInert.IsFailure() {
		t.Error("inert must count as failure")
	}
	if specs.PreconditionCurrent.IsFailure() || specs.PreconditionUnknown.IsFailure() {
		t.Error("current/unknown must not count as failure")
	}
	if specs.PreconditionUnknown.String() != "unknown" || specs.PreconditionInert.String() != "inert" || specs.PreconditionCurrent.String() != "current" {
		t.Errorf("state names diverge: %s/%s/%s", specs.PreconditionCurrent, specs.PreconditionInert, specs.PreconditionUnknown)
	}
}
