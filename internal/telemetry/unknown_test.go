package telemetry

import (
	"errors"
	"testing"
)

func TestTelemetry_UnavailableIsUnknown(t *testing.T) {
	got, err := ResolveTokenMetric("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != UnknownMetricValue {
		t.Fatalf("ResolveTokenMetric(unavailable) = %q, want %q", got, UnknownMetricValue)
	}

	if got := ResolveMetric(false, ""); got != UnknownMetricValue {
		t.Fatalf("ResolveMetric(unavailable) = %q, want %q", got, UnknownMetricValue)
	}
	if got := ResolveMetric(true, ""); got != UnknownMetricValue {
		t.Fatalf("ResolveMetric(available but empty) = %q, want %q", got, UnknownMetricValue)
	}
}

func TestTelemetry_EstimationWithoutIdentifiedMethodFails(t *testing.T) {
	if _, err := ResolveTokenMetric("123", ""); !errors.Is(err, ErrUnidentifiedEstimationMethod) {
		t.Fatalf("error = %v, want ErrUnidentifiedEstimationMethod when a value is present without a method", err)
	}
}

func TestTelemetry_ValueWithIdentifiedMethodSucceeds(t *testing.T) {
	got, err := ResolveTokenMetric("123", "provider_reported")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "123" {
		t.Fatalf("ResolveTokenMetric() = %q, want %q", got, "123")
	}
}

func TestTelemetry_AdversarialNoSilentEstimation(t *testing.T) {
	inputs := []struct {
		value  string
		method string
	}{
		{"1", ""},
		{"999999", ""},
		{"0", ""},
	}
	for _, in := range inputs {
		if _, err := ResolveTokenMetric(in.value, in.method); !errors.Is(err, ErrUnidentifiedEstimationMethod) {
			t.Fatalf("value=%q method=%q: error = %v, want ErrUnidentifiedEstimationMethod — no path may fabricate a value silently",
				in.value, in.method, err)
		}
	}
}
