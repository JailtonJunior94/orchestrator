package events

import (
	"maps"
	"sort"
)

// MetricField representa um par (nome, valor) de métrica não-zero.
// Usado por MetricSet.Fields() para renderização ordenada.
type MetricField struct {
	Name  string
	Value int
}

// MetricSet é um Value Object que acumula contadores canônicos de métricas de sessão.
// Campos canônicos: totalTokens, cacheReadTokens, thinkingTokens.
// Campos driver-específicos ficam em extra (ex: effective_context_tokens para Gemini).
//
// Imutável por design: Merge retorna novo valor sem mutar o receptor (R-DDD-001).
// Zero-value é válido: IsZero() == true; Fields() == nil (comportamento F1).
type MetricSet struct {
	totalTokens     int
	cacheReadTokens int
	thinkingTokens  int
	extra           map[string]int // campos driver-específicos
}

// NewMetricSet cria um MetricSet com os contadores canônicos explicitamente fornecidos.
// extra pode ser nil para métricas sem campos driver-específicos.
// Valores negativos são tratados como zero (defensivo).
func NewMetricSet(totalTokens, cacheReadTokens, thinkingTokens int, extra map[string]int) MetricSet {
	m := MetricSet{
		totalTokens:     max0(totalTokens),
		cacheReadTokens: max0(cacheReadTokens),
		thinkingTokens:  max0(thinkingTokens),
	}
	if len(extra) > 0 {
		m.extra = make(map[string]int, len(extra))
		for k, v := range extra {
			m.extra[k] = max0(v)
		}
	}
	return m
}

// max0 retorna 0 quando v é negativo, garantindo contadores >= 0.
func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// TotalTokens retorna o total de tokens consumidos.
func (m MetricSet) TotalTokens() int { return m.totalTokens }

// CacheReadTokens retorna o total de tokens lidos do cache.
func (m MetricSet) CacheReadTokens() int { return m.cacheReadTokens }

// ThinkingTokens retorna o total de tokens de raciocínio (extended thinking).
func (m MetricSet) ThinkingTokens() int { return m.thinkingTokens }

// Extra retorna uma cópia do mapa de campos driver-específicos.
// Retorna mapa vazio (nunca nil) quando não há campos extras.
func (m MetricSet) Extra() map[string]int {
	out := make(map[string]int, len(m.extra))
	maps.Copy(out, m.extra)
	return out
}

// Merge soma campo a campo os contadores canônicos e os campos extra de m e o.
// Retorna um novo MetricSet sem mutar nenhum dos receptores (imutabilidade de VO).
// Soma associativa: (a.Merge(b)).Merge(c) == a.Merge(b.Merge(c)).
func (m MetricSet) Merge(o MetricSet) MetricSet {
	result := MetricSet{
		totalTokens:     m.totalTokens + o.totalTokens,
		cacheReadTokens: m.cacheReadTokens + o.cacheReadTokens,
		thinkingTokens:  m.thinkingTokens + o.thinkingTokens,
	}
	// Mesclar campos extra dos dois lados.
	totalExtra := len(m.extra) + len(o.extra)
	if totalExtra > 0 {
		result.extra = make(map[string]int, totalExtra)
		for k, v := range m.extra {
			result.extra[k] += v
		}
		for k, v := range o.extra {
			result.extra[k] += v
		}
	}
	return result
}

// IsZero retorna true quando todos os contadores canônicos são zero e extra está vazio.
// Zero-value de MetricSet satisfaz IsZero() == true (comportamento F1).
func (m MetricSet) IsZero() bool {
	if m.totalTokens != 0 || m.cacheReadTokens != 0 || m.thinkingTokens != 0 {
		return false
	}
	for _, v := range m.extra {
		if v != 0 {
			return false
		}
	}
	return true
}

// Fields retorna apenas os campos não-zero do MetricSet, ordenados por nome.
// Ordem dos campos canônicos: cache_read_tokens, thinking_tokens, total_tokens.
// Campos extra são ordenados alphabeticamente após os canônicos.
// Retorna nil quando IsZero() == true.
func (m MetricSet) Fields() []MetricField {
	if m.IsZero() {
		return nil
	}

	// Campos canônicos em ordem determinística.
	canonical := []struct {
		name  string
		value int
	}{
		{"cache_read_tokens", m.cacheReadTokens},
		{"thinking_tokens", m.thinkingTokens},
		{"total_tokens", m.totalTokens},
	}

	var out []MetricField
	for _, c := range canonical {
		if c.value != 0 {
			out = append(out, MetricField{Name: c.name, Value: c.value})
		}
	}

	// Campos extra em ordem alfabética.
	extraKeys := make([]string, 0, len(m.extra))
	for k := range m.extra {
		extraKeys = append(extraKeys, k)
	}
	sort.Strings(extraKeys)
	for _, k := range extraKeys {
		if v := m.extra[k]; v != 0 {
			out = append(out, MetricField{Name: k, Value: v})
		}
	}

	return out
}
