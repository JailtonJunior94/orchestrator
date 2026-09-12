package durable

import "sort"

type RelevancePolicy struct{}

var DefaultRelevancePolicy = RelevancePolicy{}

func (p RelevancePolicy) Rank(facts []Fact, activeTask string) []Fact {
	ranked := make([]Fact, len(facts))
	copy(ranked, facts)

	sort.SliceStable(ranked, func(i, j int) bool {
		return p.less(ranked[i], ranked[j], activeTask)
	})

	return ranked
}

func (p RelevancePolicy) less(a, b Fact, activeTask string) bool {
	aScore := p.score(a, activeTask)
	bScore := p.score(b, activeTask)
	if aScore != bScore {
		return aScore < bScore
	}
	return a.Identity.Key < b.Identity.Key
}

func (p RelevancePolicy) score(f Fact, activeTask string) int {
	score := 0

	if activeTask != "" && f.Origin.Task == activeTask {
		score -= 100
	}

	catalog := NewCatalog()
	if layer, err := catalog.ResolveLayer(f.Durability); err == nil {
		switch layer {
		case TargetLayerTask:
			score -= 10
		case TargetLayerPRD:
			score -= 5
		case TargetLayerProject:
			score -= 1
		}
	}

	if f.State == FactStateContradicted {
		score += 50
	}

	return score
}
