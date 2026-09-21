package telemetry

import "testing"

func TestRF44MetricAvailability_EveryMetricIsExplicit(t *testing.T) {
	for _, availability := range RF44MetricAvailability() {
		if !availability.HasProductionWriter && availability.UnknownJustification == "" {
			t.Fatalf("metric %q has neither a production writer nor an unknown justification: implicit gap", availability.Name)
		}
		if availability.HasProductionWriter && availability.UnknownJustification != "" {
			t.Fatalf("metric %q has both a production writer and an unknown justification: ambiguous declaration", availability.Name)
		}
	}
}

func TestRF44MetricAvailability_CoversAllFifteenMetrics(t *testing.T) {
	got := RF44MetricAvailability()
	if len(got) != len(RF44MetricNames()) {
		t.Fatalf("len(RF44MetricAvailability()) = %d, want %d", len(got), len(RF44MetricNames()))
	}
}

func TestRF44MetricAvailability_TokensAreDeclaredUnknown(t *testing.T) {
	for _, availability := range RF44MetricAvailability() {
		if availability.Name != "tokens_in" && availability.Name != "tokens_out" {
			continue
		}
		if availability.HasProductionWriter {
			t.Fatalf("%s must not have a production writer until a provider reports it or a calculation method is identified (RF-46)", availability.Name)
		}
		if availability.UnknownJustification == "" {
			t.Fatalf("%s must carry an explicit unknown justification", availability.Name)
		}
	}
}
