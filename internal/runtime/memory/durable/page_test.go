package durable_test

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type PageSuite struct {
	suite.Suite
}

func TestPageSuite(t *testing.T) {
	suite.Run(t, new(PageSuite))
}

func (s *PageSuite) TestSerializeThenParseRoundTrip() {
	scenarios := []struct {
		name  string
		facts []durable.Fact
		human durable.HumanBlock
	}{
		{
			name:  "should preserve human content with no facts",
			facts: nil,
			human: durable.HumanBlock{Content: "Free-form notes written by a human.\n\nSecond paragraph.\n"},
		},
		{
			name: "should preserve human content alongside a single fact",
			facts: []durable.Fact{
				{
					Identity:   durable.Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
					Content:    "Retries three times before failing.",
					Durability: durable.DurabilityDurable,
					State:      durable.FactStateActive,
					Origin:     durable.FactOrigin{Session: "sess-1", CLI: "claude", Task: "2.0", Date: "2026-09-11T10:00:00Z"},
				},
			},
			human: durable.HumanBlock{Content: "Notes before the fact section.\n"},
		},
		{
			name: "should preserve human content ending with trailing newline",
			facts: []durable.Fact{
				{
					Identity:   durable.Identity{Key: "payment.timeout", Hash: "sha256:bbbb"},
					Content:    "Timeout raised to 30s.",
					Durability: durable.DurabilityPRD,
					State:      durable.FactStateProposed,
				},
			},
			human: durable.HumanBlock{Content: "Notes with trailing newline.\n"},
		},
		{
			name: "should preserve empty human content with multiple facts and links",
			facts: []durable.Fact{
				{
					Identity:   durable.Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
					Content:    "Original fact.",
					Durability: durable.DurabilityEphemeral,
					State:      durable.FactStateContradicted,
				},
				{
					Identity:   durable.Identity{Key: "payment.retry", Hash: "sha256:cccc"},
					Content:    "Updated fact replacing the previous one.",
					Durability: durable.DurabilityEphemeral,
					State:      durable.FactStateActive,
					Links: []durable.Link{
						{
							Type:   durable.LinkTypeReplaces,
							Target: durable.Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
						},
					},
				},
			},
			human: durable.HumanBlock{},
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			page := durable.NewMarkdownPage()

			output, err := page.Serialize(sc.facts, sc.human)
			s.Require().NoError(err)

			gotFacts, gotHuman, err := page.Parse(output)
			s.Require().NoError(err)

			s.Equal(sc.human.Content, gotHuman.Content)
			s.Len(gotFacts, len(sc.facts))

			for i, f := range sc.facts {
				s.Equal(f.Identity, gotFacts[i].Identity)
				s.Equal(f.Content, gotFacts[i].Content)
				s.Equal(f.Durability, gotFacts[i].Durability)
				s.Equal(f.State, gotFacts[i].State)
				s.Equal(f.Origin, gotFacts[i].Origin)
				s.Equal(f.Links, gotFacts[i].Links)
			}
		})
	}
}

func (s *PageSuite) TestSerializeRejectsFactWithoutSemanticKey() {
	page := durable.NewMarkdownPage()

	_, err := page.Serialize([]durable.Fact{
		{
			Identity:   durable.Identity{Key: ""},
			Durability: durable.DurabilityDurable,
		},
	}, durable.HumanBlock{})

	s.True(errors.Is(err, durable.ErrSemanticKeyMissing))
}

func (s *PageSuite) TestSerializeRejectsFactWithZeroValueDurability() {
	page := durable.NewMarkdownPage()

	_, err := page.Serialize([]durable.Fact{
		{
			Identity:   durable.Identity{Key: "payment.retry"},
			Durability: durable.DurabilityInvalid,
		},
	}, durable.HumanBlock{})

	s.True(errors.Is(err, durable.ErrDurabilityMissing))
}

func (s *PageSuite) TestParseRejectsMalformedFrontMatter() {
	page := durable.NewMarkdownPage()

	_, _, err := page.Parse([]byte("---\n[invalid yaml\n---\nbody\n"))

	s.True(errors.Is(err, durable.ErrPageUnreadable))
}

func (s *PageSuite) TestParseRejectsMalformedFactMetadata() {
	page := durable.NewMarkdownPage()

	content := "### Fact: payment.retry\n```fact-metadata\n[invalid yaml\n```\n"

	_, _, err := page.Parse([]byte(content))

	s.True(errors.Is(err, durable.ErrPageUnreadable))
}

func (s *PageSuite) TestParseIsMarkdownReadableAndGrepFriendly() {
	page := durable.NewMarkdownPage()

	output, err := page.Serialize([]durable.Fact{
		{
			Identity:   durable.Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
			Content:    "Retries three times before failing.",
			Durability: durable.DurabilityDurable,
			State:      durable.FactStateActive,
		},
	}, durable.HumanBlock{Content: "Human authored notes.\n"})
	s.Require().NoError(err)

	text := string(output)
	s.Contains(text, "### Fact: payment.retry")
	s.Contains(text, "Human authored notes.")
	s.Contains(text, "durability: durable")
}

func (s *PageSuite) TestSerializeRejectsHumanContentWithoutTrailingNewlineBeforeFacts() {
	page := durable.NewMarkdownPage()

	_, err := page.Serialize([]durable.Fact{
		{
			Identity:   durable.Identity{Key: "payment.timeout", Hash: "sha256:bbbb"},
			Durability: durable.DurabilityPRD,
		},
	}, durable.HumanBlock{Content: "Notes without trailing newline"})

	s.True(errors.Is(err, durable.ErrHumanContentNotNormalized))
}

func (s *PageSuite) TestSerializeRejectsInterleavedHumanContentBetweenFacts() {
	page := durable.NewMarkdownPage()

	content := "---\n" +
		"identity: x\n" +
		"layer: task\n" +
		"origin_session: s\n" +
		"date: \"2026-01-01T00:00:00Z\"\n" +
		"format_version: 1\n" +
		"---\n" +
		"Text before first fact.\n" +
		"### Fact: a.one\n" +
		"```fact-metadata\n" +
		"hash: sha256:aaaa\n" +
		"durability: durable\n" +
		"state: active\n" +
		"```\n" +
		"Text between facts.\n" +
		"### Fact: a.two\n" +
		"```fact-metadata\n" +
		"hash: sha256:bbbb\n" +
		"durability: durable\n" +
		"state: active\n" +
		"```\n"

	facts, human, err := page.Parse([]byte(content))
	s.Require().NoError(err)
	s.Require().Len(facts, 2)

	_, err = page.Serialize(facts, human)

	s.True(errors.Is(err, durable.ErrHumanBlockInterleaved))
}

func (s *PageSuite) TestSerializeRejectsRoundTripWhenHumanContentMasqueradesAsFact() {
	page := durable.NewMarkdownPage()

	human := durable.HumanBlock{
		Content: "### Fact: fake\n```fact-metadata\nhash: sha256:aaaa\ndurability: durable\nstate: active\n```\n",
	}

	_, err := page.Serialize(nil, human)

	s.True(errors.Is(err, durable.ErrRoundTripNotPreserved))
}

func (s *PageSuite) TestSerializeThenParsePreservesFullPageHeader() {
	page := durable.NewMarkdownPage()

	header := durable.PageHeader{
		Identity:      "prd-memoria-duravel-agentes",
		Layer:         durable.TargetLayerProject,
		OriginSession: "sess-42",
		Date:          "2026-09-11T10:00:00Z",
		FormatVersion: durable.FormatVersionCurrent,
	}

	output, err := page.Serialize(nil, durable.HumanBlock{Header: header, Content: "Notes.\n"})
	s.Require().NoError(err)

	_, gotHuman, err := page.Parse(output)
	s.Require().NoError(err)

	s.Equal(header, gotHuman.Header)
}

func (s *PageSuite) TestSerializeThenParseRoundTripsEachLinkType() {
	scenarios := []struct {
		name     string
		linkType durable.LinkType
	}{
		{name: "should round trip replaces", linkType: durable.LinkTypeReplaces},
		{name: "should round trip causes", linkType: durable.LinkTypeCauses},
		{name: "should round trip fixes", linkType: durable.LinkTypeFixes},
		{name: "should round trip contradicts", linkType: durable.LinkTypeContradicts},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			page := durable.NewMarkdownPage()

			facts := []durable.Fact{
				{
					Identity:   durable.Identity{Key: "payment.retry", Hash: "sha256:cccc"},
					Durability: durable.DurabilityEphemeral,
					State:      durable.FactStateActive,
					Links: []durable.Link{
						{
							Type:   sc.linkType,
							Target: durable.Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
						},
					},
				},
			}

			output, err := page.Serialize(facts, durable.HumanBlock{})
			s.Require().NoError(err)

			gotFacts, _, err := page.Parse(output)
			s.Require().NoError(err)
			s.Require().Len(gotFacts, 1)
			s.Require().Len(gotFacts[0].Links, 1)
			s.Equal(sc.linkType, gotFacts[0].Links[0].Type)
		})
	}
}

func (s *PageSuite) TestParseRejectsUnknownLinkTypeValue() {
	page := durable.NewMarkdownPage()

	content := "### Fact: payment.retry\n```fact-metadata\n" +
		"hash: sha256:aaaa\n" +
		"durability: durable\n" +
		"state: active\n" +
		"links:\n" +
		"  - type: unknown-link-type\n" +
		"```\n"

	_, _, err := page.Parse([]byte(content))

	s.True(errors.Is(err, durable.ErrPageUnreadable))
}

func (s *PageSuite) TestRoundTripOnRealRepositoryCorpus() {
	repoRoot := "../../../.."
	corpus := []string{
		repoRoot + "/AGENTS.md",
		repoRoot + "/README.md",
		repoRoot + "/docs/config-hierarchy.md",
		repoRoot + "/docs/guia-instalacao-universal.md",
	}

	page := durable.NewMarkdownPage()

	for _, path := range corpus {
		s.Run(path, func() {
			original, err := os.ReadFile(path)
			s.Require().NoError(err)
			s.Require().NotEmpty(original)

			facts, human, err := page.Parse(original)
			s.Require().NoError(err)
			s.Empty(facts)
			s.Equal(string(original), human.Content)

			output, err := page.Serialize(facts, human)
			s.Require().NoError(err)

			_, reparsedHuman, err := page.Parse(output)
			s.Require().NoError(err)
			s.Equal(human.Content, reparsedHuman.Content)
			s.Equal(string(original), reparsedHuman.Content)
		})
	}
}
