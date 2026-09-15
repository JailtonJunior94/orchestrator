package runtime

import "testing"

func TestBuildDurableMemoryMetrics_PerLayerKeysUseNamedPrefixes(t *testing.T) {
	t.Parallel()

	evidence := MemoryEvidence{
		WritesByLayer:       map[string]int{"task": 2, "prd": 1},
		ArchivedByLayer:     map[string]int{"task": 1},
		BudgetByLayer:       map[string]int{"task": 40},
		FactsOmitted:        1,
		FactsContradicted:   1,
		PagesUnreadable:     2,
		Redactions:          3,
		Compactions:         1,
		Contradictions:      1,
		BatonClaimed:        true,
		ContextBuildLatency: 5,
		RecordLatency:       7,
	}

	extra := NewCatalog().buildDurableMemoryMetrics(evidence)

	cases := map[string]int{
		"memory_writes_task":             2,
		"memory_writes_prd":              1,
		"memory_archived_task":           1,
		"memory_budget_tokens_task":      40,
		"memory_facts_omitted":           1,
		"memory_facts_contradicted":      1,
		"memory_pages_unreadable":        2,
		"memory_redactions_applied":      3,
		"memory_compactions_executed":    1,
		"memory_contradictions_detected": 1,
		"memory_baton_claims":            1,
		"memory_context_build_ms":        5,
		"memory_record_session_ms":       7,
		"memory_facts_written":           3,
		"memory_facts_archived":          1,
	}

	for key, want := range cases {
		got, ok := extra[key]
		if !ok {
			t.Errorf("chave de métrica ausente: %s", key)
			continue
		}
		if got != want {
			t.Errorf("%s = %d; want %d", key, got, want)
		}
	}
}

func TestBuildDurableMemoryMetrics_NoBatonClaimOmitsKey(t *testing.T) {
	t.Parallel()

	extra := NewCatalog().buildDurableMemoryMetrics(MemoryEvidence{})

	if _, ok := extra[metricMemoryBatonClaims]; ok {
		t.Error("chave memory_baton_claims não deveria aparecer quando BatonClaimed==false")
	}
}
