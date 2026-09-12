package durable

import (
	"math"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

const (
	DefaultTotalBudgetTokens = 6000
	largeBudgetMultiplier    = 3
)

type BudgetConfig struct {
	TotalTokens int
}

type LayerQuota struct {
	Layer  TargetLayer
	Tokens int
}

type Budget struct {
	TotalTokens int
	Quotas      []LayerQuota
}

type BudgetAllocation struct {
	Selected    []Fact
	Omitted     []Identity
	UsedTokens  int
	UsedByLayer map[string]int
}

type layerShare struct {
	Layer TargetLayer
	Share float64
}

var budgetLayerShares = []layerShare{
	{Layer: TargetLayerProject, Share: 0.50},
	{Layer: TargetLayerPRD, Share: 0.30},
	{Layer: TargetLayerTask, Share: 0.20},
}

type BudgetPolicy struct{}

var DefaultBudgetPolicy = BudgetPolicy{}

func (p BudgetPolicy) Resolve(class specs.WindowClass, cfg BudgetConfig) Budget {
	total := cfg.TotalTokens
	if total == 0 {
		total = DefaultTotalBudgetTokens
	}
	if class == specs.WindowLarge {
		total *= largeBudgetMultiplier
	}

	quotas := make([]LayerQuota, 0, len(budgetLayerShares))
	for _, share := range budgetLayerShares {
		quotas = append(quotas, LayerQuota{
			Layer:  share.Layer,
			Tokens: int(math.Round(float64(total) * share.Share)),
		})
	}

	return Budget{TotalTokens: total, Quotas: quotas}
}

func (p BudgetPolicy) Allocate(budget Budget, facts []Fact) BudgetAllocation {
	remaining := make(map[TargetLayer]int, len(budget.Quotas))
	for _, quota := range budget.Quotas {
		remaining[quota.Layer] = quota.Tokens
	}

	catalog := NewCatalog()
	selected := make([]Fact, 0, len(facts))
	pending := make([]Fact, 0, len(facts))
	usedTotal := 0
	usedByLayer := make(map[TargetLayer]int, len(budget.Quotas))

	for _, f := range facts {
		layer, err := catalog.ResolveLayer(f.Durability)
		if err != nil {
			pending = append(pending, f)
			continue
		}

		cost := p.estimateTokens(f)
		if cost <= remaining[layer] && usedTotal+cost <= budget.TotalTokens {
			selected = append(selected, f)
			remaining[layer] -= cost
			usedTotal += cost
			usedByLayer[layer] += cost
			continue
		}
		pending = append(pending, f)
	}

	surplus := 0
	for _, quota := range budget.Quotas {
		surplus += remaining[quota.Layer]
	}

	stillPending := make([]Fact, 0, len(pending))
	for _, f := range pending {
		cost := p.estimateTokens(f)
		if cost <= surplus && usedTotal+cost <= budget.TotalTokens {
			selected = append(selected, f)
			surplus -= cost
			usedTotal += cost
			if layer, err := catalog.ResolveLayer(f.Durability); err == nil {
				usedByLayer[layer] += cost
			}
			continue
		}
		stillPending = append(stillPending, f)
	}

	omitted := make([]Identity, 0, len(stillPending))
	for _, f := range stillPending {
		omitted = append(omitted, f.Identity)
	}

	usedByLayerNamed := make(map[string]int, len(usedByLayer))
	for layer, used := range usedByLayer {
		usedByLayerNamed[layer.String()] = used
	}

	return BudgetAllocation{Selected: selected, Omitted: omitted, UsedTokens: usedTotal, UsedByLayer: usedByLayerNamed}
}

func (p BudgetPolicy) estimateTokens(f Fact) int {
	return int(math.Round(float64(len(f.Content)) / 3.5))
}
