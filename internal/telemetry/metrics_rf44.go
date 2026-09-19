package telemetry

import "strconv"

type MetricSummary struct {
	Metric string         `json:"metric"`
	Count  int            `json:"count"`
	Sum    float64        `json:"sum,omitempty"`
	Avg    float64        `json:"avg,omitempty"`
	Values map[string]int `json:"values,omitempty"`
}

var numericRF44Metrics = map[string]bool{
	"duration":   true,
	"tool_calls": true,
	"retries":    true,
}

var categoricalRF44Metrics = map[string]bool{
	"provider":     true,
	"model":        true,
	"final_status": true,
}

type rf44Accumulator struct {
	count  int
	sum    float64
	values map[string]int
}

func (c *Catalog) summarizeRF44Metrics(entries []logEntry) []MetricSummary {
	names := RF44MetricsWithProductionWriter()
	accs := make(map[string]*rf44Accumulator, len(names))

	for _, e := range entries {
		for _, name := range names {
			v, ok := e.Metric(name)
			if !ok {
				continue
			}
			acc, exists := accs[name]
			if !exists {
				acc = &rf44Accumulator{values: make(map[string]int)}
				accs[name] = acc
			}
			acc.count++
			if numericRF44Metrics[name] {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					acc.sum += f
				}
			}
			if categoricalRF44Metrics[name] {
				acc.values[v]++
			}
		}
	}

	out := make([]MetricSummary, 0, len(accs))
	for _, name := range names {
		acc, ok := accs[name]
		if !ok {
			continue
		}
		ms := MetricSummary{Metric: name, Count: acc.count}
		if numericRF44Metrics[name] {
			ms.Sum = acc.sum
			ms.Avg = acc.sum / float64(acc.count)
		}
		if categoricalRF44Metrics[name] {
			ms.Values = acc.values
		}
		out = append(out, ms)
	}
	return out
}

func entryHasAnyRF44Metric(e logEntry) bool {
	for _, name := range RF44MetricsWithProductionWriter() {
		if _, ok := e.Metric(name); ok {
			return true
		}
	}
	return false
}
