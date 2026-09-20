package hookcontract

import (
	"errors"
	"testing"
)

func TestResult_BlockRequiresReasonAndPolicy(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		policyID string
		gateID   string
		wantErr  bool
	}{
		{"missing reason and policy", "", "", "", true},
		{"missing reason only", "", "P-1", "", true},
		{"missing policy and gate", "denied", "", "", true},
		{"reason and policyID", "denied", "P-1", "", false},
		{"reason and gateID", "denied", "", "G-1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResult(DecisionBlock, tt.reason, tt.policyID, tt.gateID)
			if tt.wantErr && !errors.Is(err, ErrInvalidResult) {
				t.Fatalf("error = %v, want ErrInvalidResult", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestResult_NonBlockDecisionsDoNotRequireReason(t *testing.T) {
	for _, decision := range []Decision{DecisionAllow, DecisionWarn, DecisionNotApplicable, DecisionError} {
		if _, err := NewResult(decision, "", "", ""); err != nil {
			t.Fatalf("decision %v unexpected error: %v", decision, err)
		}
	}
}

func TestResult_RejectsInvalidDecision(t *testing.T) {
	if _, err := NewResult(Decision(99), "reason", "P-1", ""); !errors.Is(err, ErrInvalidResult) {
		t.Fatalf("error = %v, want ErrInvalidResult", err)
	}
}

func TestNewAllow_IsAlwaysValid(t *testing.T) {
	result := NewAllow()
	if !result.Valid() {
		t.Fatal("NewAllow() must be Valid()")
	}
	if result.Decision() != DecisionAllow {
		t.Fatalf("Decision() = %v, want DecisionAllow", result.Decision())
	}
}

func TestNewNotApplicable_CarriesReason(t *testing.T) {
	result := NewNotApplicable("no policy configured for this event")
	if result.Decision() != DecisionNotApplicable {
		t.Fatalf("Decision() = %v, want DecisionNotApplicable", result.Decision())
	}
	if result.Reason() != "no policy configured for this event" {
		t.Fatalf("Reason() = %q", result.Reason())
	}
}

func TestResult_WithEvidenceAndMetadataAreImmutableCopies(t *testing.T) {
	base := NewAllow()
	withEvidence := base.WithEvidence("evidence-1", "evidence-2")
	if len(base.Evidence()) != 0 {
		t.Fatal("base Result must remain unaffected by WithEvidence")
	}
	if len(withEvidence.Evidence()) != 2 {
		t.Fatalf("Evidence() len = %d, want 2", len(withEvidence.Evidence()))
	}

	withMetadata := base.WithMetadata(map[string]string{"key": "value"})
	if len(base.Metadata()) != 0 {
		t.Fatal("base Result must remain unaffected by WithMetadata")
	}
	if withMetadata.Metadata()["key"] != "value" {
		t.Fatal("WithMetadata did not persist key")
	}
}

func TestDecision_StringValues(t *testing.T) {
	tests := map[Decision]string{
		DecisionAllow:         "ALLOW",
		DecisionBlock:         "BLOCK",
		DecisionWarn:          "WARN",
		DecisionNotApplicable: "NOT_APPLICABLE",
		DecisionError:         "ERROR",
	}
	for decision, want := range tests {
		if got := decision.String(); got != want {
			t.Fatalf("Decision(%d).String() = %q, want %q", int(decision), got, want)
		}
	}
}
