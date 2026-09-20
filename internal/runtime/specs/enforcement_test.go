package specs_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/hookcontract"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func coverageFixture(t *testing.T, p specs.CanonicalPoint) specs.PointCoverage {
	t.Helper()
	keys, ok := specs.RecognizedNativeKeys("claude", p)
	if !ok || len(keys) == 0 {
		t.Fatalf("no recognized native key for claude point %s", p)
	}
	artifact, ok := specs.InstalledArtifactPath("claude", p)
	if ok {
		cov, err := specs.NewCatalog().NewPointCoverage("claude", p, keys[0], ".agents/hooks/validate-preload.sh", artifact)
		if err != nil {
			t.Fatalf("NewPointCoverage(%s): %v", p, err)
		}
		return cov
	}
	cov, err := specs.NewCatalog().NewObservedPointCoverage("claude", p, keys[0])
	if err != nil {
		t.Fatalf("NewObservedPointCoverage(%s): %v", p, err)
	}
	return cov
}

func allCanonicalPoints() []specs.CanonicalPoint {
	return []specs.CanonicalPoint{
		specs.PointSessionStart,
		specs.PointPreTool,
		specs.PointPostTool,
		specs.PointBeforeComplete,
		specs.PointSessionEnd,
	}
}

func TestNewPointCoverageRejectsNativeKeyTheCliDoesNotRecognize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		agentID   string
		point     specs.CanonicalPoint
		nativeKey string
	}{
		{"copilot", specs.PointBeforeComplete, "agentStop_TYPO"},
		{"copilot", specs.PointPreTool, "PreToolUse"},
		{"claude", specs.PointPreTool, "preToolUse"},
		{"opencode", specs.PointBeforeComplete, "Stop"},
		{"codex", specs.PointBeforeComplete, "SessionEnd"},
	}

	for _, tc := range cases {
		artifact, ok := specs.InstalledArtifactPath(tc.agentID, tc.point)
		if !ok {
			t.Fatalf("no installed artifact for %s point %s", tc.agentID, tc.point)
		}
		_, err := specs.NewCatalog().NewPointCoverage(tc.agentID, tc.point, tc.nativeKey, ".agents/hooks/validate-preload.sh", artifact)
		if !errors.Is(err, specs.ErrUnrecognizedNativeKey) {
			t.Errorf("agent %s point %s key %q: err = %v; want ErrUnrecognizedNativeKey — a key no CLI recognizes is exactly how a matrix cell goes inert",
				tc.agentID, tc.point, tc.nativeKey, err)
		}
	}
}

func TestNewPointCoverageRejectsUnknownAgentVocabulary(t *testing.T) {
	t.Parallel()

	_, err := specs.NewCatalog().NewPointCoverage("gemini", specs.PointPreTool, "PreToolUse", ".agents/hooks/validate-preload.sh", ".gemini/hooks/validate-preload.sh")
	if !errors.Is(err, specs.ErrUnrecognizedNativeKey) {
		t.Fatalf("err = %v; want ErrUnrecognizedNativeKey for an agent with no declared hook vocabulary", err)
	}
}

func TestNewEnforcementRejectsIncompleteCoverage(t *testing.T) {
	t.Parallel()

	all := allCanonicalPoints()

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

	var complete []specs.PointCoverage
	for _, p := range all {
		complete = append(complete, coverageFixture(t, p))
	}
	enf, err := specs.NewCatalog().NewEnforcement(complete)
	if err != nil {
		t.Fatalf("complete coverage rejected: %v", err)
	}
	if !enf.Valid() {
		t.Fatal("complete enforcement reported invalid")
	}
}

func TestNewEnforcementAcceptsDeclaredUnsupportedPoint(t *testing.T) {
	t.Parallel()

	coverage := []specs.PointCoverage{
		coverageFixture(t, specs.PointSessionStart),
		coverageFixture(t, specs.PointPreTool),
		coverageFixture(t, specs.PointPostTool),
		coverageFixture(t, specs.PointBeforeComplete),
	}
	unsupported, err := specs.NewCatalog().NewUnsupportedPointCoverage("opencode", specs.PointSessionEnd, "provider has no native session-end hook")
	if err != nil {
		t.Fatalf("NewUnsupportedPointCoverage unexpected error: %v", err)
	}
	if unsupported.State() != hookcontract.SupportUnsupported {
		t.Fatalf("State() = %v, want SupportUnsupported", unsupported.State())
	}
	coverage = append(coverage, unsupported)

	enf, err := specs.NewCatalog().NewEnforcement(coverage)
	if err != nil {
		t.Fatalf("NewEnforcement rejected coverage with declared SupportUnsupported point: %v", err)
	}
	if !enf.Valid() {
		t.Fatal("enforcement with declared unsupported point reported invalid")
	}
	cov, ok := enf.CoverageFor(specs.PointSessionEnd)
	if !ok {
		t.Fatal("CoverageFor(PointSessionEnd) not found")
	}
	if cov.State() != hookcontract.SupportUnsupported {
		t.Fatalf("CoverageFor(PointSessionEnd).State() = %v, want SupportUnsupported", cov.State())
	}
	if cov.Reason() == "" {
		t.Fatal("unsupported coverage lost its declared reason")
	}
}

func TestNewUnsupportedPointCoverageRequiresReason(t *testing.T) {
	t.Parallel()

	if _, err := specs.NewCatalog().NewUnsupportedPointCoverage("opencode", specs.PointSessionEnd, "   "); !errors.Is(err, specs.ErrIncompleteCoverage) {
		t.Fatalf("empty reason err = %v; want ErrIncompleteCoverage", err)
	}
	if _, err := specs.NewCatalog().NewUnsupportedPointCoverage("", specs.PointSessionEnd, "x"); !errors.Is(err, specs.ErrIncompleteCoverage) {
		t.Fatalf("empty agent id err = %v; want ErrIncompleteCoverage", err)
	}
	if _, err := specs.NewCatalog().NewUnsupportedPointCoverage("opencode", specs.CanonicalPoint(99), "x"); !errors.Is(err, specs.ErrUnknownCanonicalPoint) {
		t.Fatalf("invalid point err = %v; want ErrUnknownCanonicalPoint", err)
	}
}

func TestNewAdapterPointCoverageRequiresLimitation(t *testing.T) {
	t.Parallel()

	if _, err := specs.NewCatalog().NewAdapterPointCoverage("opencode", specs.PointSessionStart, "event.session.created", "", "", "   "); !errors.Is(err, specs.ErrIncompleteCoverage) {
		t.Fatalf("empty limitation err = %v; want ErrIncompleteCoverage", err)
	}
	cov, err := specs.NewCatalog().NewAdapterPointCoverage("opencode", specs.PointSessionStart, "event.session.created", "", "", "no native session-start hook")
	if err != nil {
		t.Fatalf("valid adapter coverage rejected: %v", err)
	}
	if cov.State() != hookcontract.SupportAdapter {
		t.Fatalf("State() = %v, want SupportAdapter", cov.State())
	}
	if cov.Limitation() == "" {
		t.Fatal("adapter coverage lost its declared limitation")
	}
	if _, err := specs.NewCatalog().NewAdapterPointCoverage("opencode", specs.PointSessionStart, "event.session.created", ".agents/hooks/validate-preload.sh", "", "x"); !errors.Is(err, specs.ErrIncompleteCoverage) {
		t.Fatalf("script without artifact err = %v; want ErrIncompleteCoverage", err)
	}
}

func TestCanonicalPointsDeriveFromHookContract(t *testing.T) {
	t.Parallel()

	catalog := specs.NewCatalog()
	points := catalog.CanonicalPoints()
	if len(points) != 5 {
		t.Fatalf("len(points) = %d, want 5", len(points))
	}

	want := map[specs.CanonicalPoint]hookcontract.EventKind{
		specs.PointSessionStart:   hookcontract.EventSessionStart,
		specs.PointPreTool:        hookcontract.EventBeforeTool,
		specs.PointPostTool:       hookcontract.EventAfterTool,
		specs.PointBeforeComplete: hookcontract.EventBeforeComplete,
		specs.PointSessionEnd:     hookcontract.EventSessionEnd,
	}
	for _, point := range points {
		event, ok := catalog.CanonicalPointEvent(point)
		if !ok {
			t.Fatalf("CanonicalPointEvent(%s) has no projection", point)
		}
		if event != want[point] {
			t.Errorf("CanonicalPointEvent(%s) = %v, want %v", point, event, want[point])
		}
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
	for _, name := range []string{"session-start", "pre-tool", "post-tool", "before-complete", "session-end"} {
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
