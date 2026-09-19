package telemetry

import "testing"

func metricSummary(name string, count int, avg float64) MetricSummary {
	return MetricSummary{Metric: name, Count: count, Avg: avg, Sum: avg * float64(count)}
}

func TestCompareAblation_InsufficientSample(t *testing.T) {
	baseline := AblationBaseline{SampleCount: 10, Metrics: []MetricSummary{metricSummary("duration", 10, 100)}}
	withComponent := AblationBaseline{SampleCount: 0}

	result := NewCatalog().CompareAblation("review", ComponentSkill, baseline, withComponent)
	if result.Decision != AblationOnDemand {
		t.Fatalf("Decision = %v, want ON-DEMAND quando amostra with_component=0", result.Decision)
	}
	if len(result.Evidence) == 0 {
		t.Error("Evidence nao pode ser vazia — decisao precisa ser sustentada por fato numerico")
	}
}

func TestCompareAblation_NoComparableMetric(t *testing.T) {
	baseline := AblationBaseline{SampleCount: 5, Metrics: []MetricSummary{
		{Metric: "provider", Count: 5, Values: map[string]int{"claude": 5}},
	}}
	withComponent := AblationBaseline{SampleCount: 5, Metrics: []MetricSummary{
		{Metric: "provider", Count: 5, Values: map[string]int{"claude": 5}},
	}}

	result := NewCatalog().CompareAblation("bugfix", ComponentSkill, baseline, withComponent)
	if result.Decision != AblationOnDemand {
		t.Fatalf("Decision = %v, want ON-DEMAND quando nenhuma metrica de custo e comparavel", result.Decision)
	}
}

func TestCompareAblation_CostImproves(t *testing.T) {
	baseline := AblationBaseline{SampleCount: 20, Metrics: []MetricSummary{metricSummary("duration", 20, 200)}}
	withComponent := AblationBaseline{SampleCount: 20, Metrics: []MetricSummary{metricSummary("duration", 20, 100)}}

	result := NewCatalog().CompareAblation("agent-governance", ComponentSkill, baseline, withComponent)
	if result.Decision != AblationKeep {
		t.Fatalf("Decision = %v, want KEEP quando custo cai muito alem da tolerancia", result.Decision)
	}
}

func TestCompareAblation_CostWorsens(t *testing.T) {
	baseline := AblationBaseline{SampleCount: 20, Metrics: []MetricSummary{metricSummary("duration", 20, 100)}}
	withComponent := AblationBaseline{SampleCount: 20, Metrics: []MetricSummary{metricSummary("duration", 20, 200)}}

	result := NewCatalog().CompareAblation("validate-preload", ComponentHook, baseline, withComponent)
	if result.Decision != AblationRemove {
		t.Fatalf("Decision = %v, want REMOVE quando custo sobe muito alem da tolerancia", result.Decision)
	}
}

func TestCompareAblation_WithinTolerance(t *testing.T) {
	baseline := AblationBaseline{SampleCount: 20, Metrics: []MetricSummary{metricSummary("duration", 20, 100)}}
	withComponent := AblationBaseline{SampleCount: 20, Metrics: []MetricSummary{metricSummary("duration", 20, 103)}}

	result := NewCatalog().CompareAblation("approval-policy", ComponentPolicy, baseline, withComponent)
	if result.Decision != AblationOnDemand {
		t.Fatalf("Decision = %v, want ON-DEMAND quando delta esta dentro da tolerancia", result.Decision)
	}
}

func TestCompareAblation_BaselineZeroCostAppears(t *testing.T) {
	baseline := AblationBaseline{SampleCount: 10, Metrics: []MetricSummary{metricSummary("retries", 10, 0)}}
	withComponent := AblationBaseline{SampleCount: 10, Metrics: []MetricSummary{metricSummary("retries", 10, 2)}}

	result := NewCatalog().CompareAblation("codex-hook", ComponentHook, baseline, withComponent)
	if result.Decision != AblationRemove {
		t.Fatalf("Decision = %v, want REMOVE quando custo surge do zero", result.Decision)
	}
}

func TestBuildAblationBaseline_NeverIncludesMetricWithoutWriter(t *testing.T) {
	entries := []logEntry{
		{
			Skill: "task-loop",
			Ref:   "acp-session",
			Fields: map[string]string{
				"provider":       "claude",
				"duration_ms":    "500",
				"tokens_in":      "999",
				"tokens_out":     "111",
				"context_loaded": "42",
				"tests_passed":   "true",
			},
		},
	}

	baseline := NewCatalog().BuildAblationBaseline(entries)

	for _, forbidden := range []string{"tokens_in", "tokens_out", "context_loaded", "tests_passed", "task_type", "risk", "skills_loaded", "review_rounds", "human_intervention"} {
		if _, ok := baseline.metric(forbidden); ok {
			t.Errorf("baseline nao deveria conter metrica %q sem escritor de producao identificado", forbidden)
		}
	}
	if _, ok := baseline.metric("provider"); !ok {
		t.Error("baseline deveria conter 'provider', que tem escritor de producao identificado")
	}
}
