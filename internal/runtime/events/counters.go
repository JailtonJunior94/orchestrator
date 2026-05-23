package events

import (
	"sort"
	"time"
)

// toolCallRecord registra dados de uma chamada de ferramenta individual.
type toolCallRecord struct {
	id        ToolCallID
	name      string
	startedAt time.Time
	updates   int
	final     bool
}

// ToolCallSummary é o snapshot exportado de uma chamada de ferramenta.
type ToolCallSummary struct {
	// ID é o identificador único da chamada.
	ID ToolCallID
	// Name é o nome da ferramenta.
	Name string
	// StartedAt é o timestamp de início.
	StartedAt time.Time
	// Updates é o número de eventos tool_call_update recebidos.
	Updates int
	// Final indica se um update final foi recebido.
	Final bool
}

// ToolCallCounters é uma coleção de primeira classe que agrega contadores de tool calls.
// Segue OC #4 (coleção de primeira classe).
type ToolCallCounters struct {
	byID         map[ToolCallID]toolCallRecord
	unknownKinds map[string]struct{}
}

// NewToolCallCounters cria uma instância vazia de ToolCallCounters.
func NewToolCallCounters() *ToolCallCounters {
	return &ToolCallCounters{
		byID:         make(map[ToolCallID]toolCallRecord),
		unknownKinds: make(map[string]struct{}),
	}
}

// Record incorpora um evento nos contadores.
// Ignora eventos sem ToolCallID para tool_call_start/update.
func (c *ToolCallCounters) Record(evt Event) {
	switch evt.Kind() {
	case KindToolCallStart:
		p := evt.ToolCallStart()
		if p == nil {
			return
		}
		id := evt.ToolCallID()
		if id.IsZero() {
			return
		}
		if _, exists := c.byID[id]; !exists {
			c.byID[id] = toolCallRecord{
				id:        id,
				name:      p.Name(),
				startedAt: evt.Timestamp(),
			}
		}
	case KindToolCallUpdate:
		p := evt.ToolCallUpdate()
		if p == nil {
			return
		}
		id := evt.ToolCallID()
		if id.IsZero() {
			return
		}
		rec, exists := c.byID[id]
		if !exists {
			rec = toolCallRecord{
				id:        id,
				startedAt: evt.Timestamp(),
			}
		}
		rec.updates++
		if p.Final() {
			rec.final = true
		}
		c.byID[id] = rec
	case KindUnknown:
		p := evt.Unknown()
		if p != nil && p.RawKind() != "" {
			c.unknownKinds[p.RawKind()] = struct{}{}
		}
	}
}

// ToolCalls retorna um snapshot ordenado por timestamp de início.
func (c *ToolCallCounters) ToolCalls() []ToolCallSummary {
	result := make([]ToolCallSummary, 0, len(c.byID))
	for _, rec := range c.byID {
		result = append(result, ToolCallSummary{
			ID:        rec.id,
			Name:      rec.name,
			StartedAt: rec.startedAt,
			Updates:   rec.updates,
			Final:     rec.final,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.Before(result[j].StartedAt)
	})
	return result
}

// MappedCount retorna o número de tool calls distintos registrados.
func (c *ToolCallCounters) MappedCount() int {
	return len(c.byID)
}

// UnknownKinds retorna os raw kinds desconhecidos encontrados, deduplicados e ordenados.
func (c *ToolCallCounters) UnknownKinds() []string {
	result := make([]string, 0, len(c.unknownKinds))
	for k := range c.unknownKinds {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
