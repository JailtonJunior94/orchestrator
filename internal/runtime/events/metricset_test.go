package events

import (
	"reflect"
	"testing"
)

func TestMetricSet_ZeroValue(t *testing.T) {
	t.Parallel()

	var m MetricSet
	if !m.IsZero() {
		t.Error("zero-value MetricSet.IsZero() deve ser true")
	}
	if fields := m.Fields(); fields != nil {
		t.Errorf("zero-value MetricSet.Fields() = %v; want nil", fields)
	}
}

func TestMetricSet_IsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ms    MetricSet
		wantZ bool
	}{
		{"zero_value", MetricSet{}, true},
		{"only_total", NewMetricSet(10, 0, 0, nil), false},
		{"only_cache_read", NewMetricSet(0, 5, 0, nil), false},
		{"only_thinking", NewMetricSet(0, 0, 3, nil), false},
		{"extra_nonzero", NewMetricSet(0, 0, 0, map[string]int{"foo": 1}), false},
		{"all_zero_explicit", NewMetricSet(0, 0, 0, nil), true},
		{"extra_all_zero", NewMetricSet(0, 0, 0, map[string]int{"foo": 0}), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.ms.IsZero()
			if got != tc.wantZ {
				t.Errorf("IsZero() = %v; want %v", got, tc.wantZ)
			}
		})
	}
}

func TestMetricSet_Merge_SumsFields(t *testing.T) {
	t.Parallel()

	a := NewMetricSet(100, 20, 10, map[string]int{"eff": 50})
	b := NewMetricSet(200, 30, 5, map[string]int{"eff": 25, "billed": 10})

	got := a.Merge(b)

	if got.TotalTokens() != 300 {
		t.Errorf("TotalTokens = %d; want 300", got.TotalTokens())
	}
	if got.CacheReadTokens() != 50 {
		t.Errorf("CacheReadTokens = %d; want 50", got.CacheReadTokens())
	}
	if got.ThinkingTokens() != 15 {
		t.Errorf("ThinkingTokens = %d; want 15", got.ThinkingTokens())
	}
	extra := got.Extra()
	if extra["eff"] != 75 {
		t.Errorf("extra[eff] = %d; want 75", extra["eff"])
	}
	if extra["billed"] != 10 {
		t.Errorf("extra[billed] = %d; want 10", extra["billed"])
	}
}

func TestMetricSet_Merge_Associative(t *testing.T) {
	t.Parallel()

	a := NewMetricSet(1, 2, 3, map[string]int{"x": 10})
	b := NewMetricSet(4, 5, 6, map[string]int{"x": 20})
	c := NewMetricSet(7, 8, 9, map[string]int{"x": 30})

	// (a.Merge(b)).Merge(c) deve ser igual a a.Merge(b.Merge(c))
	left := a.Merge(b).Merge(c)
	right := a.Merge(b.Merge(c))

	if left.TotalTokens() != right.TotalTokens() {
		t.Errorf("Merge não associativo para TotalTokens: left=%d right=%d", left.TotalTokens(), right.TotalTokens())
	}
	if left.CacheReadTokens() != right.CacheReadTokens() {
		t.Errorf("Merge não associativo para CacheReadTokens: left=%d right=%d", left.CacheReadTokens(), right.CacheReadTokens())
	}
	if left.ThinkingTokens() != right.ThinkingTokens() {
		t.Errorf("Merge não associativo para ThinkingTokens: left=%d right=%d", left.ThinkingTokens(), right.ThinkingTokens())
	}
	if left.Extra()["x"] != right.Extra()["x"] {
		t.Errorf("Merge não associativo para extra[x]: left=%d right=%d", left.Extra()["x"], right.Extra()["x"])
	}
}

func TestMetricSet_Merge_WithZero(t *testing.T) {
	t.Parallel()

	a := NewMetricSet(50, 10, 5, nil)
	zero := MetricSet{}

	// a.Merge(zero) deve ser igual a a.
	got := a.Merge(zero)
	if got.TotalTokens() != 50 || got.CacheReadTokens() != 10 || got.ThinkingTokens() != 5 {
		t.Errorf("Merge com zero-value alterou o resultado: %+v", got)
	}

	// zero.Merge(a) deve ser igual a a.
	got2 := zero.Merge(a)
	if got2.TotalTokens() != 50 || got2.CacheReadTokens() != 10 || got2.ThinkingTokens() != 5 {
		t.Errorf("Merge zero.Merge(a) alterou o resultado: %+v", got2)
	}
}

func TestMetricSet_Merge_Immutability(t *testing.T) {
	t.Parallel()

	a := NewMetricSet(10, 5, 3, map[string]int{"k": 7})
	b := NewMetricSet(20, 10, 6, map[string]int{"k": 3})

	_ = a.Merge(b)

	// a não deve ser mutado.
	if a.TotalTokens() != 10 {
		t.Errorf("a.TotalTokens() após Merge = %d; want 10 (imutabilidade violada)", a.TotalTokens())
	}
	if a.Extra()["k"] != 7 {
		t.Errorf("a.Extra()[k] após Merge = %d; want 7 (imutabilidade violada)", a.Extra()["k"])
	}
}

func TestMetricSet_Fields_OrderedNonZero(t *testing.T) {
	t.Parallel()

	m := NewMetricSet(100, 20, 0, map[string]int{"eff": 50, "billed": 0, "prompt": 10})

	fields := m.Fields()

	// total_tokens=100, cache_read_tokens=20, thinking_tokens=0 (omitido), eff=50, prompt=10, billed=0 (omitido)
	// Ordem esperada: canonical (cache_read_tokens, thinking_tokens[omitido], total_tokens) + extra alfabético (eff, prompt)
	expected := []MetricField{
		{Name: "cache_read_tokens", Value: 20},
		{Name: "total_tokens", Value: 100},
		{Name: "eff", Value: 50},
		{Name: "prompt", Value: 10},
	}

	if !reflect.DeepEqual(fields, expected) {
		t.Errorf("Fields() = %v; want %v", fields, expected)
	}
}

func TestMetricSet_Fields_NilWhenZero(t *testing.T) {
	t.Parallel()

	var m MetricSet
	if fields := m.Fields(); fields != nil {
		t.Errorf("Fields() de zero-value = %v; want nil", fields)
	}
}

func TestNewMetricSet_NegativeValuesClampedToZero(t *testing.T) {
	t.Parallel()

	m := NewMetricSet(-10, -5, -3, map[string]int{"x": -1})
	if !m.IsZero() {
		t.Errorf("NewMetricSet com valores negativos deve resultar em IsZero()==true; got %+v", m)
	}
}
