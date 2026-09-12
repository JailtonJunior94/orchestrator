package durable_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

type BudgetPolicySuite struct {
	suite.Suite
}

func TestBudgetPolicySuite(t *testing.T) {
	suite.Run(t, new(BudgetPolicySuite))
}

func (s *BudgetPolicySuite) TestResolveAppliesQuotaShares() {
	policy := durable.BudgetPolicy{}

	budget := policy.Resolve(specs.WindowStandard, durable.BudgetConfig{TotalTokens: 1000})

	s.Equal(1000, budget.TotalTokens)
	s.Require().Len(budget.Quotas, 3)

	byLayer := make(map[durable.TargetLayer]int, 3)
	for _, quota := range budget.Quotas {
		byLayer[quota.Layer] = quota.Tokens
	}

	s.Equal(500, byLayer[durable.TargetLayerProject])
	s.Equal(300, byLayer[durable.TargetLayerPRD])
	s.Equal(200, byLayer[durable.TargetLayerTask])
}

func (s *BudgetPolicySuite) TestResolveWindowLargeMultipliesTotal() {
	policy := durable.BudgetPolicy{}

	standard := policy.Resolve(specs.WindowStandard, durable.BudgetConfig{TotalTokens: 1000})
	large := policy.Resolve(specs.WindowLarge, durable.BudgetConfig{TotalTokens: 1000})

	s.Greater(large.TotalTokens, standard.TotalTokens)
}

func (s *BudgetPolicySuite) TestResolveZeroValueUsesDefault() {
	policy := durable.BudgetPolicy{}

	budget := policy.Resolve(specs.WindowStandard, durable.BudgetConfig{})

	s.Equal(durable.DefaultTotalBudgetTokens, budget.TotalTokens)
}

func (s *BudgetPolicySuite) TestAllocateRespectsQuotas() {
	policy := durable.BudgetPolicy{}
	budget := policy.Resolve(specs.WindowStandard, durable.BudgetConfig{TotalTokens: 100})

	facts := []durable.Fact{
		{
			Identity:   durable.Identity{Key: "project.a"},
			Durability: durable.DurabilityDurable,
			Content:    strings.Repeat("x", 700),
		},
	}

	allocation := policy.Allocate(budget, facts)

	s.Empty(allocation.Selected)
	s.Len(allocation.Omitted, 1)
}

func (s *BudgetPolicySuite) TestAllocateCedesSurplusBetweenLayers() {
	policy := durable.BudgetPolicy{}
	budget := policy.Resolve(specs.WindowStandard, durable.BudgetConfig{TotalTokens: 1000})

	facts := []durable.Fact{
		{
			Identity:   durable.Identity{Key: "task.a"},
			Durability: durable.DurabilityEphemeral,
			Content:    strings.Repeat("x", 3500),
		},
	}

	allocation := policy.Allocate(budget, facts)

	s.Require().Len(allocation.Selected, 1)
	s.Empty(allocation.Omitted)
}

func (s *BudgetPolicySuite) TestAllocateDeclaresOmissionsWhenExceedingTotal() {
	policy := durable.BudgetPolicy{}
	budget := policy.Resolve(specs.WindowStandard, durable.BudgetConfig{TotalTokens: 100})

	facts := []durable.Fact{
		{
			Identity:   durable.Identity{Key: "project.a"},
			Durability: durable.DurabilityDurable,
			Content:    strings.Repeat("x", 700),
		},
		{
			Identity:   durable.Identity{Key: "task.b"},
			Durability: durable.DurabilityEphemeral,
			Content:    strings.Repeat("y", 700),
		},
	}

	allocation := policy.Allocate(budget, facts)

	s.NotEmpty(allocation.Omitted)
	s.LessOrEqual(allocation.UsedTokens, budget.TotalTokens)
}

func (s *BudgetPolicySuite) TestAllocateIsDeterministicAcrossRuns() {
	policy := durable.BudgetPolicy{}
	budget := policy.Resolve(specs.WindowStandard, durable.BudgetConfig{TotalTokens: 500})

	facts := []durable.Fact{
		{Identity: durable.Identity{Key: "project.a"}, Durability: durable.DurabilityDurable, Content: strings.Repeat("x", 300)},
		{Identity: durable.Identity{Key: "task.b"}, Durability: durable.DurabilityEphemeral, Content: strings.Repeat("y", 300)},
	}

	first := policy.Allocate(budget, facts)
	second := policy.Allocate(budget, facts)

	s.Equal(first, second)
}
