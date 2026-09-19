package capability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fakeParitySource = `package parity

type ParitySuite struct{}

func (s *ParitySuite) TestParity_AllTools() {}
`

func writeFixtureModule(t *testing.T, testBody string) string {
	t.Helper()
	dir := t.TempDir()

	goMod := "module capabilityfixture\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	testFile := "package fixture\n\nimport \"testing\"\n\nfunc TestParitySuite(t *testing.T) {\n\tt.Run(\"TestParity_AllTools\", func(t *testing.T) {\n\t\t" +
		testBody + "\n\t})\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "fixture_test.go"), []byte(testFile), 0o644); err != nil {
		t.Fatalf("write fixture test: %v", err)
	}
	return dir
}

func mustHaveSyntacticMethod(t *testing.T) {
	t.Helper()
	methodName := EvidenceTest
	if idx := strings.LastIndex(EvidenceTest, "/"); idx >= 0 {
		methodName = EvidenceTest[idx+1:]
	}
	if !ParityTestMethodExists([]byte(fakeParitySource), evidenceTestSuite, methodName) {
		t.Fatal("fixture setup invalid: the syntactic method must exist to reproduce the bug scenario (old behavior relied only on this check)")
	}
}

func TestDispatchProvenFromParityTests_SkippedTestNeverSatisfiesGate(t *testing.T) {
	mustHaveSyntacticMethod(t)
	dir := writeFixtureModule(t, `t.Skip("regression: a skipped body must never prove dispatch")`)

	proven := dispatchProvenFromTests([]byte(fakeParitySource), evidenceTestSuite, EvidenceTest, dir, "./...")
	if proven("claude", "C01") {
		t.Fatal("a skipped test body satisfies syntactic existence (old check) but must never satisfy DispatchProvenFromParityTests after the fix")
	}
}

func TestDispatchProvenFromParityTests_FailedTestNeverSatisfiesGate(t *testing.T) {
	mustHaveSyntacticMethod(t)
	dir := writeFixtureModule(t, `t.Fatal("regression: a failing body must never prove dispatch")`)

	proven := dispatchProvenFromTests([]byte(fakeParitySource), evidenceTestSuite, EvidenceTest, dir, "./...")
	if proven("claude", "C01") {
		t.Fatal("a failing test body satisfies syntactic existence (old check) but must never satisfy DispatchProvenFromParityTests after the fix")
	}
}

func TestDispatchProvenFromParityTests_PassingTestSatisfiesGate(t *testing.T) {
	mustHaveSyntacticMethod(t)
	dir := writeFixtureModule(t, "")

	proven := dispatchProvenFromTests([]byte(fakeParitySource), evidenceTestSuite, EvidenceTest, dir, "./...")
	if !proven("claude", "C01") {
		t.Fatal("a genuinely executed and passing test must satisfy DispatchProvenFromParityTests")
	}
}
