package approval

import "testing"

func closesTestFinding(t *testing.T, severity Severity) Finding {
	t.Helper()
	finding, err := NewFinding(severity, "a.go:1", "rule", "description")
	if err != nil {
		t.Fatalf("NewFinding(%d): %v", int(severity), err)
	}
	return finding
}

func TestVerdictClosesUnderRF33(t *testing.T) {
	cases := []struct {
		name       string
		verdict    Verdict
		severities []Severity
		want       bool
	}{
		{"approved without findings closes", VerdictApproved, nil, true},
		{"approved with low finding closes", VerdictApproved, []Severity{SeverityLow}, true},
		{"remarks with medium only closes", VerdictApprovedWithRemarks, []Severity{SeverityMedium}, true},
		{"remarks with low only closes", VerdictApprovedWithRemarks, []Severity{SeverityLow}, true},
		{"remarks with medium and low closes", VerdictApprovedWithRemarks, []Severity{SeverityMedium, SeverityLow}, true},
		{"remarks with one high does not close", VerdictApprovedWithRemarks, []Severity{SeverityLow, SeverityHigh}, false},
		{"remarks with one critical does not close", VerdictApprovedWithRemarks, []Severity{SeverityMedium, SeverityCritical}, false},
		{"remarks without declared findings does not close", VerdictApprovedWithRemarks, nil, false},
		{"rejected never closes", VerdictRejected, []Severity{SeverityLow}, false},
		{"blocked never closes", VerdictBlocked, []Severity{SeverityLow}, false},
		{"rejected without findings never closes", VerdictRejected, nil, false},
		{"blocked without findings never closes", VerdictBlocked, nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := make([]Finding, 0, len(tc.severities))
			for _, severity := range tc.severities {
				findings = append(findings, closesTestFinding(t, severity))
			}
			if got := tc.verdict.Closes(findings); got != tc.want {
				t.Fatalf("%s.Closes(%d findings) = %v, want %v", tc.verdict, len(findings), got, tc.want)
			}
		})
	}
}

func TestNewApprovalProofHonorsRF33(t *testing.T) {
	medium := []Finding{closesTestFinding(t, SeverityMedium)}
	high := []Finding{closesTestFinding(t, SeverityHigh)}

	if _, err := NewApprovalProof(VerdictApprovedWithRemarks, completeCriteriaMap(t), medium); err != nil {
		t.Fatalf("remarks without blocking findings must produce a proof: %v", err)
	}
	if _, err := NewApprovalProof(VerdictApprovedWithRemarks, completeCriteriaMap(t), high); err == nil {
		t.Fatal("remarks with a high finding must not produce a proof")
	}
	if _, err := NewApprovalProof(VerdictApprovedWithRemarks, completeCriteriaMap(t), nil); err == nil {
		t.Fatal("remarks without declared findings must fail closed")
	}
}

func TestApprovalProofKeepsRemarksAsDeclaredDebt(t *testing.T) {
	medium := []Finding{closesTestFinding(t, SeverityMedium)}
	proof, err := NewApprovalProof(VerdictApprovedWithRemarks, completeCriteriaMap(t), medium)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := proof.Remarks(); len(got) != 1 || got[0].Severity() != SeverityMedium {
		t.Fatalf("Remarks() = %+v, want the medium finding preserved", got)
	}
	if proof.Verdict() != VerdictApprovedWithRemarks {
		t.Fatalf("Verdict() = %s, want APPROVED_WITH_REMARKS preserved", proof.Verdict())
	}
}
