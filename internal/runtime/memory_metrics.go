package runtime

import "fmt"

const (
	metricMemoryFactsWritten           = "memory_facts_written"
	metricMemoryFactsArchived          = "memory_facts_archived"
	metricMemoryFactsOmitted           = "memory_facts_omitted"
	metricMemoryFactsContradicted      = "memory_facts_contradicted"
	metricMemoryPagesUnreadable        = "memory_pages_unreadable"
	metricMemoryRedactionsApplied      = "memory_redactions_applied"
	metricMemoryCompactionsExecuted    = "memory_compactions_executed"
	metricMemoryContradictionsDetected = "memory_contradictions_detected"
	metricMemoryBatonClaims            = "memory_baton_claims"
	metricMemoryContextBuildMs         = "memory_context_build_ms"
	metricMemoryRecordSessionMs        = "memory_record_session_ms"
	metricMemoryBudgetLayerPrefix      = "memory_budget_tokens_"
	metricMemoryWritesLayerPrefix      = "memory_writes_"
	metricMemoryArchivedLayerPrefix    = "memory_archived_"
)

func (c *Catalog) buildDurableMemoryMetrics(e MemoryEvidence) map[string]int {
	extra := map[string]int{
		metricMemoryFactsOmitted:           e.FactsOmitted,
		metricMemoryFactsContradicted:      e.FactsContradicted,
		metricMemoryPagesUnreadable:        e.PagesUnreadable,
		metricMemoryRedactionsApplied:      e.Redactions,
		metricMemoryCompactionsExecuted:    e.Compactions,
		metricMemoryContradictionsDetected: e.Contradictions,
		metricMemoryContextBuildMs:         int(e.ContextBuildLatency),
		metricMemoryRecordSessionMs:        int(e.RecordLatency),
	}
	if e.BatonClaimed {
		extra[metricMemoryBatonClaims] = 1
	}

	totalWrites := 0
	for layer, count := range e.WritesByLayer {
		extra[fmt.Sprintf("%s%s", metricMemoryWritesLayerPrefix, layer)] = count
		totalWrites += count
	}
	extra[metricMemoryFactsWritten] = totalWrites

	totalArchived := 0
	for layer, count := range e.ArchivedByLayer {
		extra[fmt.Sprintf("%s%s", metricMemoryArchivedLayerPrefix, layer)] = count
		totalArchived += count
	}
	extra[metricMemoryFactsArchived] = totalArchived

	for layer, count := range e.BudgetByLayer {
		extra[fmt.Sprintf("%s%s", metricMemoryBudgetLayerPrefix, layer)] = count
	}

	return extra
}
