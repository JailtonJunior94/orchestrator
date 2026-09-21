package telemetry

type MetricAvailability struct {
	Name                 string
	HasProductionWriter  bool
	UnknownJustification string
}

type unknownMetricEntry struct {
	name          string
	justification string
}

var rf44MetricUnknownEntries = []unknownMetricEntry{
	{"task_type", "no adapter emits a structured task type today; declared unknown until a production source exists"},
	{"risk", "risk classification is not reported by any adapter today; declared unknown until a production source exists"},
	{"tokens_in", "RF-46: this metric is only emitted when the provider reports it or a calculation method is identified; no adapter does this today"},
	{"tokens_out", "same justification as the equivalent input metric"},
	{"context_loaded", "loaded context volume is not instrumented by any adapter today; declared unknown until a production source exists"},
	{"skills_loaded", "loaded skill list is not emitted in structured telemetry today; declared unknown until a production source exists"},
	{"review_rounds", "review round count is not emitted in structured telemetry today; declared unknown until a production source exists"},
	{"tests_passed", "test outcome is not emitted in structured telemetry today; declared unknown until a production source exists"},
	{"human_intervention", "human intervention is not emitted in structured telemetry today; declared unknown until a production source exists"},
}

func rf44MetricUnknownJustifications() map[string]string {
	out := make(map[string]string, len(rf44MetricUnknownEntries))
	for _, entry := range rf44MetricUnknownEntries {
		out[entry.name] = entry.justification
	}
	return out
}

func RF44MetricAvailability() []MetricAvailability {
	withWriter := make(map[string]bool, len(RF44MetricsWithProductionWriter()))
	for _, name := range RF44MetricsWithProductionWriter() {
		withWriter[name] = true
	}

	justifications := rf44MetricUnknownJustifications()
	names := RF44MetricNames()
	out := make([]MetricAvailability, 0, len(names))
	for _, name := range names {
		out = append(out, MetricAvailability{
			Name:                 name,
			HasProductionWriter:  withWriter[name],
			UnknownJustification: justifications[name],
		})
	}
	return out
}
