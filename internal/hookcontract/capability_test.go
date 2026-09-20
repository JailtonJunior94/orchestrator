package hookcontract

import (
	"errors"
	"testing"
)

func TestCapability_UnsupportedIsNotSuccess(t *testing.T) {
	capability, err := NewCapability("provider-x", EventSessionEnd, SupportUnsupported, "", false, "provider has no native session-end hook")
	if err != nil {
		t.Fatalf("NewCapability unexpected error: %v", err)
	}
	if capability.State() == SupportVerified {
		t.Fatal("SupportUnsupported must not equal SupportVerified")
	}
	if capability.State() != SupportUnsupported {
		t.Fatalf("State() = %v, want SupportUnsupported", capability.State())
	}
}

func TestCapability_AdapterRequiresLimitation(t *testing.T) {
	tests := []struct {
		name       string
		limitation string
		wantErr    bool
	}{
		{"missing limitation", "", true},
		{"declared limitation", "fires on session.idle, not blocking", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCapability("provider-x", EventBeforeComplete, SupportAdapter, "native-key", true, tt.limitation)
			if tt.wantErr && !errors.Is(err, ErrInvalidCapability) {
				t.Fatalf("error = %v, want ErrInvalidCapability", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCapability_RejectsEmptyProviderAndInvalidEvent(t *testing.T) {
	if _, err := NewCapability("", EventBeforeTool, SupportVerified, "k", true, ""); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("empty provider error = %v, want ErrInvalidCapability", err)
	}
	if _, err := NewCapability("provider-x", EventKind(99), SupportVerified, "k", true, ""); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("invalid event error = %v, want ErrInvalidCapability", err)
	}
	if _, err := NewCapability("provider-x", EventBeforeTool, SupportState(99), "k", true, ""); !errors.Is(err, ErrInvalidCapability) {
		t.Fatalf("invalid state error = %v, want ErrInvalidCapability", err)
	}
}

func TestSupportState_ThreeDistinctValues(t *testing.T) {
	if SupportVerified == SupportUnsupported || SupportAdapter == SupportUnsupported || SupportVerified == SupportAdapter {
		t.Fatal("SupportVerified, SupportAdapter and SupportUnsupported must be pairwise distinct")
	}
}
