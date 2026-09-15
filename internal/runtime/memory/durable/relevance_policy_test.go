package durable_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type RelevancePolicySuite struct {
	suite.Suite
}

func TestRelevancePolicySuite(t *testing.T) {
	suite.Run(t, new(RelevancePolicySuite))
}

func (s *RelevancePolicySuite) TestRankIsDeterministicAcrossRuns() {
	policy := durable.RelevancePolicy{}
	facts := []durable.Fact{
		{
			Identity:   durable.Identity{Key: "project.decision.retry"},
			Durability: durable.DurabilityDurable,
			Origin:     durable.FactOrigin{Task: "9.0"},
		},
		{
			Identity:   durable.Identity{Key: "task.note.timeout"},
			Durability: durable.DurabilityEphemeral,
			Origin:     durable.FactOrigin{Task: "3.0"},
		},
		{
			Identity:   durable.Identity{Key: "prd.risk.regression"},
			Durability: durable.DurabilityPRD,
			Origin:     durable.FactOrigin{Task: "3.0"},
		},
		{
			Identity:   durable.Identity{Key: "task.note.contradicted"},
			Durability: durable.DurabilityEphemeral,
			State:      durable.FactStateContradicted,
			Origin:     durable.FactOrigin{Task: "3.0"},
		},
	}

	first := policy.Rank(facts, "3.0")
	second := policy.Rank(facts, "3.0")

	s.Equal(first, second)
}

func (s *RelevancePolicySuite) TestRankPrioritizesActiveTask() {
	policy := durable.RelevancePolicy{}
	facts := []durable.Fact{
		{
			Identity:   durable.Identity{Key: "project.decision.retry"},
			Durability: durable.DurabilityDurable,
			Origin:     durable.FactOrigin{Task: "9.0"},
		},
		{
			Identity:   durable.Identity{Key: "task.note.timeout"},
			Durability: durable.DurabilityEphemeral,
			Origin:     durable.FactOrigin{Task: "3.0"},
		},
	}

	ranked := policy.Rank(facts, "3.0")

	s.Equal(durable.SemanticKey("task.note.timeout"), ranked[0].Identity.Key)
}

func (s *RelevancePolicySuite) TestRankDemotesContradictedFacts() {
	policy := durable.RelevancePolicy{}
	facts := []durable.Fact{
		{
			Identity:   durable.Identity{Key: "task.note.a"},
			Durability: durable.DurabilityEphemeral,
			State:      durable.FactStateContradicted,
		},
		{
			Identity:   durable.Identity{Key: "task.note.b"},
			Durability: durable.DurabilityEphemeral,
			State:      durable.FactStateActive,
		},
	}

	ranked := policy.Rank(facts, "")

	s.Equal(durable.SemanticKey("task.note.b"), ranked[0].Identity.Key)
	s.Equal(durable.SemanticKey("task.note.a"), ranked[1].Identity.Key)
}

func (s *RelevancePolicySuite) TestRankDoesNotMutateInput() {
	policy := durable.RelevancePolicy{}
	facts := []durable.Fact{
		{Identity: durable.Identity{Key: "b.key"}, Durability: durable.DurabilityDurable},
		{Identity: durable.Identity{Key: "a.key"}, Durability: durable.DurabilityDurable},
	}

	_ = policy.Rank(facts, "")

	s.Equal(durable.SemanticKey("b.key"), facts[0].Identity.Key)
	s.Equal(durable.SemanticKey("a.key"), facts[1].Identity.Key)
}
