package telemetry

import "fmt"

type ComponentKind string

const (
	ComponentSkill  ComponentKind = "skill"
	ComponentHook   ComponentKind = "hook"
	ComponentPolicy ComponentKind = "policy"
)

type AblationDecision string

const (
	AblationKeep     AblationDecision = "KEEP"
	AblationOnDemand AblationDecision = "ON-DEMAND"
	AblationRemove   AblationDecision = "REMOVE"
)

type AblationBaseline struct {
	SampleCount int
	Metrics     []MetricSummary
}

func (c *Catalog) BuildAblationBaseline(entries []logEntry) AblationBaseline {
	return AblationBaseline{
		SampleCount: len(entries),
		Metrics:     c.summarizeRF44Metrics(entries),
	}
}

func (b AblationBaseline) metric(name string) (MetricSummary, bool) {
	for _, m := range b.Metrics {
		if m.Metric == name {
			return m, true
		}
	}
	return MetricSummary{}, false
}

type AblationResult struct {
	Component     string
	Kind          ComponentKind
	Baseline      AblationBaseline
	WithComponent AblationBaseline
	Decision      AblationDecision
	Evidence      []string
}

const ablationCostToleranceRatio = 0.10

var ablationCostMetrics = []string{"duration", "tool_calls", "retries"}

func (c *Catalog) CompareAblation(component string, kind ComponentKind, baseline, withComponent AblationBaseline) AblationResult {
	result := AblationResult{
		Component:     component,
		Kind:          kind,
		Baseline:      baseline,
		WithComponent: withComponent,
	}

	if withComponent.SampleCount == 0 {
		result.Decision = AblationOnDemand
		result.Evidence = append(result.Evidence,
			"with_component_sample_count=0: evidencia insuficiente para KEEP ou REMOVE")
		return result
	}

	var baselineCost, withCost float64
	var compared int
	for _, name := range ablationCostMetrics {
		bm, bok := baseline.metric(name)
		wm, wok := withComponent.metric(name)
		if !bok || !wok {
			continue
		}
		compared++
		baselineCost += bm.Avg
		withCost += wm.Avg
		result.Evidence = append(result.Evidence, fmt.Sprintf(
			"%s: baseline_avg=%.2f(n=%d) with_avg=%.2f(n=%d)", name, bm.Avg, bm.Count, wm.Avg, wm.Count))
	}

	if compared == 0 {
		result.Decision = AblationOnDemand
		result.Evidence = append(result.Evidence,
			"nenhuma metrica de custo (duration/tool_calls/retries) presente em ambas as populacoes")
		return result
	}

	if baselineCost == 0 && withCost == 0 {
		result.Decision = AblationOnDemand
		result.Evidence = append(result.Evidence, "custo agregado zero nas duas populacoes: sem sinal de custo")
		return result
	}
	if baselineCost == 0 {
		result.Decision = AblationRemove
		result.Evidence = append(result.Evidence, "custo agregado surgiu apenas com o componente (baseline_cost=0)")
		return result
	}

	delta := (withCost - baselineCost) / baselineCost
	switch {
	case delta < -ablationCostToleranceRatio:
		result.Decision = AblationKeep
		result.Evidence = append(result.Evidence, fmt.Sprintf(
			"custo agregado caiu %.1f%%, alem da tolerancia de %.0f%% — manter", -delta*100, ablationCostToleranceRatio*100))
	case delta > ablationCostToleranceRatio:
		result.Decision = AblationRemove
		result.Evidence = append(result.Evidence, fmt.Sprintf(
			"custo agregado subiu %.1f%%, alem da tolerancia de %.0f%% — remover", delta*100, ablationCostToleranceRatio*100))
	default:
		result.Decision = AblationOnDemand
		result.Evidence = append(result.Evidence, fmt.Sprintf(
			"delta de custo agregado %.1f%% dentro da tolerancia de %.0f%%: evidencia inconclusiva", delta*100, ablationCostToleranceRatio*100))
	}
	return result
}
