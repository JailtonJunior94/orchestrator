package telemetry

import "testing"

func hookEntry(fields map[string]string) logEntry {
	return logEntry{Skill: "hook", Ref: "test", Fields: fields}
}

func TestAblation_BaselineWithoutHook(t *testing.T) {
	withoutHook := []logEntry{
		hookEntry(map[string]string{"duration_ms": "100"}),
		hookEntry(map[string]string{"duration_ms": "120"}),
	}
	withHook := []logEntry{
		hookEntry(map[string]string{"duration_ms": "110"}),
		hookEntry(map[string]string{"duration_ms": "115"}),
	}

	catalog := NewCatalog()
	baseline := catalog.BuildAblationBaseline(withoutHook)
	if baseline.SampleCount != len(withoutHook) {
		t.Fatalf("SampleCount = %d, want %d", baseline.SampleCount, len(withoutHook))
	}

	result := catalog.DecideHookAblation("some-on-demand-hook", withoutHook, withHook)
	if result.Decision != AblationOnDemand {
		t.Fatalf("Decision = %v, want ON-DEMAND for a small cost delta within tolerance", result.Decision)
	}
}

func TestAblation_NonCriticalHookRemovedWhenCostRises(t *testing.T) {
	withoutHook := make([]logEntry, 0, 20)
	withHook := make([]logEntry, 0, 20)
	for i := 0; i < 20; i++ {
		withoutHook = append(withoutHook, hookEntry(map[string]string{"duration_ms": "100"}))
		withHook = append(withHook, hookEntry(map[string]string{"duration_ms": "300"}))
	}

	catalog := NewCatalog()
	result := catalog.DecideHookAblation("cosmetic-hook", withoutHook, withHook)
	if result.Decision != AblationRemove {
		t.Fatalf("Decision = %v, want REMOVE when a non-critical hook adds cost far beyond tolerance", result.Decision)
	}
}

func TestAblation_CriticalSecurityHookSurvivesRemoveDecision(t *testing.T) {
	withoutHook := make([]logEntry, 0, 20)
	withHook := make([]logEntry, 0, 20)
	for i := 0; i < 20; i++ {
		withoutHook = append(withoutHook, hookEntry(map[string]string{"duration_ms": "100"}))
		withHook = append(withHook, hookEntry(map[string]string{"duration_ms": "300"}))
	}

	catalog := NewCatalog()
	result := catalog.DecideHookAblation("validate-governance", withoutHook, withHook)
	if result.Decision != AblationKeep {
		t.Fatalf("Decision = %v, want KEEP — a critical security hook must survive a REMOVE-triggering ablation (RF-65)", result.Decision)
	}
	if !IsCriticalSecurityHook("validate-governance") {
		t.Fatal("validate-governance must be classified as a critical security hook")
	}
}
