package approval

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FingerprintSuite struct {
	suite.Suite
	calc FingerprintCalculator
}

func TestFingerprintSuite(t *testing.T) {
	suite.Run(t, new(FingerprintSuite))
}

func (s *FingerprintSuite) SetupTest() {
	s.calc = NewFingerprintCalculator()
}

func (s *FingerprintSuite) findingWith(sev Severity, file, rule string) Finding {
	return mustFinding(s.T(), sev, file, rule)
}

func (s *FingerprintSuite) TestStableUnderReordering() {
	a := s.findingWith(SeverityHigh, "a.go", "R1")
	b := s.findingWith(SeverityLow, "b.go", "R2")

	first := s.calc.Compute([]Finding{a, b})
	second := s.calc.Compute([]Finding{b, a})

	s.True(first.Equal(second))
}

func (s *FingerprintSuite) TestStableUnderDuplicates() {
	a := s.findingWith(SeverityHigh, "a.go", "R1")

	first := s.calc.Compute([]Finding{a})
	second := s.calc.Compute([]Finding{a, a, a})

	s.True(first.Equal(second))
}

func (s *FingerprintSuite) TestStableUnderLineNumberChange() {
	before := s.calc.Compute([]Finding{s.findingWith(SeverityHigh, "a.go", "R1")})
	afterFinding, err := NewFinding(SeverityHigh, "a.go", "R1", "now reported at another line")
	s.Require().NoError(err)
	after := s.calc.Compute([]Finding{afterFinding})

	s.True(before.Equal(after))
}

func (s *FingerprintSuite) TestStableUnderAgentIdentifierChange() {
	before := s.calc.Compute([]Finding{s.findingWith(SeverityHigh, "a.go", "R1")})
	afterFinding, err := NewFinding(SeverityHigh, "a.go", "R1", "agent id BUG-9999")
	s.Require().NoError(err)
	after := s.calc.Compute([]Finding{afterFinding})

	s.True(before.Equal(after))
}

func (s *FingerprintSuite) TestUnstableUnderSeverityChange() {
	first := s.calc.Compute([]Finding{s.findingWith(SeverityHigh, "a.go", "R1")})
	second := s.calc.Compute([]Finding{s.findingWith(SeverityCritical, "a.go", "R1")})

	s.False(first.Equal(second))
}

func (s *FingerprintSuite) TestUnstableUnderFileChange() {
	first := s.calc.Compute([]Finding{s.findingWith(SeverityHigh, "a.go", "R1")})
	second := s.calc.Compute([]Finding{s.findingWith(SeverityHigh, "b.go", "R1")})

	s.False(first.Equal(second))
}

func (s *FingerprintSuite) TestUnstableUnderRuleChange() {
	first := s.calc.Compute([]Finding{s.findingWith(SeverityHigh, "a.go", "R1")})
	second := s.calc.Compute([]Finding{s.findingWith(SeverityHigh, "a.go", "R2")})

	s.False(first.Equal(second))
}

func (s *FingerprintSuite) TestUncomputedNeverEqual() {
	var a, b Fingerprint
	s.False(a.Equal(b))

	computed := s.calc.Compute(nil)
	s.False(a.Equal(computed))
	s.False(computed.Equal(a))
}

func (s *FingerprintSuite) TestComputedEmptyIsStable() {
	s.True(s.calc.Compute(nil).Equal(s.calc.Compute([]Finding{})))
}

func FuzzFingerprint(f *testing.F) {
	f.Add("a.go", "R1", 3, "b.go", "R2", 1)
	f.Add("", "", 0, "", "", 0)
	f.Add("x", "y", 99, "x", "y", -5)

	calc := NewFingerprintCalculator()
	f.Fuzz(func(t *testing.T, f1, r1 string, s1 int, f2, r2 string, s2 int) {
		build := func(file, rule string, sev int) []Finding {
			normalized := Severity((sev%4+4)%4 + 1)
			if strings.TrimSpace(file) == "" {
				file = "placeholder.go"
			}
			if strings.TrimSpace(rule) == "" {
				rule = "R0"
			}
			finding, err := NewFinding(normalized, file, rule, "")
			if err != nil {
				t.Fatalf("NewFinding: %v", err)
			}
			return []Finding{finding}
		}

		set := append(build(f1, r1, s1), build(f2, r2, s2)...)
		reversed := append(build(f2, r2, s2), build(f1, r1, s1)...)

		if !calc.Compute(set).Equal(calc.Compute(reversed)) {
			t.Fatalf("fingerprint not order-independent for %q/%q and %q/%q", f1, r1, f2, r2)
		}
		if !calc.Compute(set).Equal(calc.Compute(set)) {
			t.Fatal("fingerprint not deterministic")
		}
	})
}
