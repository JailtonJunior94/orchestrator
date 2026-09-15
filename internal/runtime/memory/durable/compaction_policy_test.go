package durable_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type CompactionPolicySuite struct {
	suite.Suite
}

func TestCompactionPolicySuite(t *testing.T) {
	suite.Run(t, new(CompactionPolicySuite))
}

func (s *CompactionPolicySuite) newFacts(n int) []durable.Fact {
	facts := make([]durable.Fact, 0, n)
	for i := 0; i < n; i++ {
		facts = append(facts, durable.Fact{
			Identity:   durable.Identity{Key: durable.SemanticKey(fmt.Sprintf("topic.fact-%02d", i)), Hash: "sha256:aaaa"},
			Content:    strings.Repeat("payload content line ", 20),
			Durability: durable.DurabilityDurable,
		})
	}
	return facts
}

func (s *CompactionPolicySuite) TestCompactReturnsRemainingWithinLimits() {
	policy := durable.CompactionPolicy{}
	human := durable.HumanBlock{Content: "# notes\nhuman authored content\n"}
	facts := s.newFacts(10)

	result, err := policy.Compact(facts, human, durable.CompactionConfig{LineLimit: 1000, ByteLimit: 32 * 1024}, "")

	s.Require().NoError(err)
	s.True(result.Achieved)
	s.Empty(result.ToArchive)
	s.Len(result.Remaining, 10)
}

func (s *CompactionPolicySuite) TestCompactArchivesUntilWithinLimits() {
	policy := durable.CompactionPolicy{}
	human := durable.HumanBlock{Content: "# notes\nhuman authored content\n"}
	facts := s.newFacts(20)

	result, err := policy.Compact(facts, human, durable.CompactionConfig{LineLimit: 25, ByteLimit: 2 * 1024}, "")

	s.Require().NoError(err)
	s.True(result.Achieved)
	s.NotEmpty(result.ToArchive)
	s.Less(len(result.Remaining), 20)
}

func (s *CompactionPolicySuite) TestCompactPreservesHumanBlockContentVerbatim() {
	policy := durable.CompactionPolicy{}
	humanContent := "# notes\nimportant human authored decision\n"
	human := durable.HumanBlock{Content: humanContent}
	facts := s.newFacts(20)

	result, err := policy.Compact(facts, human, durable.CompactionConfig{LineLimit: 25, ByteLimit: 2 * 1024}, "")

	s.Require().NoError(err)

	page := durable.NewMarkdownPage()
	rendered, err := page.Serialize(result.Remaining, human)
	s.Require().NoError(err)

	_, reparsedHuman, err := page.Parse(rendered)
	s.Require().NoError(err)
	s.Equal(humanContent, reparsedHuman.Content)
}

func (s *CompactionPolicySuite) TestCompactReportsLimitUnreachable() {
	policy := durable.CompactionPolicy{}
	human := durable.HumanBlock{Content: strings.Repeat("human content that alone exceeds the limit\n", 500)}
	facts := s.newFacts(1)

	result, err := policy.Compact(facts, human, durable.CompactionConfig{LineLimit: 5, ByteLimit: 100}, "")

	s.True(errors.Is(err, durable.ErrLimitUnreachable))
	s.False(result.Achieved)
	s.Empty(result.Remaining)
}

func (s *CompactionPolicySuite) TestCompactIsDeterministicAcrossRuns() {
	policy := durable.CompactionPolicy{}
	human := durable.HumanBlock{Content: "# notes\n"}
	facts := s.newFacts(15)

	first, err := policy.Compact(facts, human, durable.CompactionConfig{LineLimit: 20, ByteLimit: 1500}, "")
	s.Require().NoError(err)
	second, err := policy.Compact(facts, human, durable.CompactionConfig{LineLimit: 20, ByteLimit: 1500}, "")
	s.Require().NoError(err)

	s.Equal(first, second)
}
