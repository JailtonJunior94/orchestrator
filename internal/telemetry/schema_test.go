package telemetry

import (
	"errors"
	"testing"
)

func TestSchema_AcceptsEveryCommonMetric(t *testing.T) {
	schema := CommonSchema()
	for _, name := range RF44MetricNames() {
		if err := schema.ValidateKey(name); err != nil {
			t.Fatalf("ValidateKey(%q) unexpected error: %v", name, err)
		}
	}
}

func TestSchema_AcceptsNamespacedProviderKey(t *testing.T) {
	schema := CommonSchema()
	if err := schema.ValidateKey("provider.opencode.plugin_load_ms"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchema_RejectsProviderKeyLeakingIntoCommonSpace(t *testing.T) {
	schema := CommonSchema()
	if err := schema.ValidateKey("opencode_plugin_load_ms"); !errors.Is(err, ErrProviderMetricLeakage) {
		t.Fatalf("error = %v, want ErrProviderMetricLeakage for an un-namespaced provider-specific key", err)
	}
}

func TestSchema_RejectsMalformedProviderNamespace(t *testing.T) {
	schema := CommonSchema()
	if err := schema.ValidateKey("provider.opencode"); !errors.Is(err, ErrProviderMetricLeakage) {
		t.Fatalf("error = %v, want ErrProviderMetricLeakage for a namespace missing the metric key", err)
	}
}

func TestSchema_ValidateNoLeakageDetectsViolationAmongValidKeys(t *testing.T) {
	schema := CommonSchema()
	err := schema.ValidateNoLeakage([]string{"provider", "model", "leaked_key"})
	if !errors.Is(err, ErrProviderMetricLeakage) {
		t.Fatalf("error = %v, want ErrProviderMetricLeakage", err)
	}
}
