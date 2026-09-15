package durable

import (
	"sort"
	"strings"
)

type SearchPolicy struct{}

var DefaultSearchPolicy = SearchPolicy{}

func (p SearchPolicy) Text(facts []Fact, query string) []Fact {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil
	}
	matches := make([]Fact, 0, len(facts))
	for _, f := range facts {
		if strings.Contains(strings.ToLower(f.Content), needle) ||
			strings.Contains(strings.ToLower(string(f.Identity.Key)), needle) {
			matches = append(matches, f)
		}
	}
	return p.sorted(matches)
}

func (p SearchPolicy) Entity(facts []Fact, entity string) []Fact {
	needle := strings.ToLower(strings.TrimSpace(entity))
	if needle == "" {
		return nil
	}
	matches := make([]Fact, 0, len(facts))
	for _, f := range facts {
		if strings.Contains(strings.ToLower(string(f.Identity.Key)), needle) ||
			strings.Contains(strings.ToLower(f.Origin.Task), needle) ||
			strings.Contains(strings.ToLower(f.Origin.Session), needle) ||
			strings.Contains(strings.ToLower(f.Origin.CLI), needle) {
			matches = append(matches, f)
		}
	}
	return p.sorted(matches)
}

func (p SearchPolicy) sorted(facts []Fact) []Fact {
	sort.SliceStable(facts, func(i, j int) bool {
		return facts[i].Identity.Key < facts[j].Identity.Key
	})
	return facts
}
