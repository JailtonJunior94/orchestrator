package durable

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const FormatVersionCurrent = 1

const factMetadataFence = "```fact-metadata"

type Page interface {
	Parse(content []byte) ([]Fact, HumanBlock, error)
	Serialize(facts []Fact, human HumanBlock) ([]byte, error)
}

type PageHeader struct {
	Identity      string      `yaml:"identity"`
	Layer         TargetLayer `yaml:"layer"`
	OriginSession string      `yaml:"origin_session"`
	Date          string      `yaml:"date"`
	FormatVersion int         `yaml:"format_version"`
}

type HumanBlock struct {
	Header      PageHeader
	Content     string
	interleaved bool
}

type factMetadataYAML struct {
	Hash       ContentHash `yaml:"hash"`
	Durability Durability  `yaml:"durability"`
	State      FactState   `yaml:"state"`
	Origin     FactOrigin  `yaml:"origin"`
	Content    string      `yaml:"content"`
	Links      []Link      `yaml:"links,omitempty"`
}

var factBlockPattern = regexp.MustCompile("(?ms)^### Fact: (.+?)\n```fact-metadata\n(.*?)\n```[ \t]*\n?")

var frontMatterPattern = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n`)

type MarkdownPage struct{}

func NewMarkdownPage() *MarkdownPage {
	return &MarkdownPage{}
}

func (p *MarkdownPage) Parse(content []byte) ([]Fact, HumanBlock, error) {
	text := string(content)

	var header PageHeader
	if match := frontMatterPattern.FindStringSubmatchIndex(text); match != nil {
		yamlBlock := text[match[2]:match[3]]
		if err := yaml.Unmarshal([]byte(yamlBlock), &header); err != nil {
			return nil, HumanBlock{}, fmt.Errorf("durable: invalid frontmatter: %w: %w", err, ErrPageUnreadable)
		}
		text = text[match[1]:]
	}

	matches := factBlockPattern.FindAllStringSubmatchIndex(text, -1)

	facts := make([]Fact, 0, len(matches))
	segments := make([]string, 0, len(matches)+1)

	cursor := 0
	for _, m := range matches {
		segments = append(segments, text[cursor:m[0]])

		key := strings.TrimSpace(text[m[2]:m[3]])
		yamlBlock := text[m[4]:m[5]]

		var metadata factMetadataYAML
		if err := yaml.Unmarshal([]byte(yamlBlock), &metadata); err != nil {
			return nil, HumanBlock{}, fmt.Errorf("durable: invalid fact metadata: %w: %w", err, ErrPageUnreadable)
		}

		facts = append(facts, Fact{
			Identity: Identity{
				Key:  SemanticKey(key),
				Hash: metadata.Hash,
			},
			Content:    metadata.Content,
			Durability: metadata.Durability,
			Origin:     metadata.Origin,
			State:      metadata.State,
			Links:      metadata.Links,
		})

		cursor = m[1]
	}
	segments = append(segments, text[cursor:])

	interleaved := false
	for i := 1; i < len(segments); i++ {
		if segments[i] != "" {
			interleaved = true
			break
		}
	}

	var human strings.Builder
	for _, segment := range segments {
		human.WriteString(segment)
	}

	return facts, HumanBlock{Header: header, Content: human.String(), interleaved: interleaved}, nil
}

func (p *MarkdownPage) Serialize(facts []Fact, human HumanBlock) ([]byte, error) {
	catalog := NewCatalog()
	for _, f := range facts {
		if err := catalog.ValidateFact(f); err != nil {
			return nil, err
		}
	}

	if human.interleaved && len(facts) > 0 {
		return nil, ErrHumanBlockInterleaved
	}

	if len(facts) > 0 && human.Content != "" && !strings.HasSuffix(human.Content, "\n") {
		return nil, fmt.Errorf("durable: human content must already end with a newline before facts are appended: %w", ErrHumanContentNotNormalized)
	}

	var b strings.Builder

	header := human.Header
	if header.FormatVersion == 0 {
		header.FormatVersion = FormatVersionCurrent
	}
	headerYAML, err := yaml.Marshal(header)
	if err != nil {
		return nil, fmt.Errorf("durable: failed to serialize frontmatter: %w", err)
	}
	b.WriteString("---\n")
	b.Write(headerYAML)
	b.WriteString("---\n")

	b.WriteString(human.Content)

	for _, f := range facts {
		metadata := factMetadataYAML{
			Hash:       f.Identity.Hash,
			Durability: f.Durability,
			State:      f.State,
			Origin:     f.Origin,
			Content:    f.Content,
			Links:      f.Links,
		}
		metadataYAML, err := yaml.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("durable: failed to serialize fact: %w", err)
		}
		b.WriteString("### Fact: ")
		b.WriteString(string(f.Identity.Key))
		b.WriteString("\n")
		b.WriteString(factMetadataFence)
		b.WriteString("\n")
		b.Write(metadataYAML)
		b.WriteString("```\n")
	}

	output := []byte(b.String())

	_, humanReparsed, err := p.Parse(output)
	if err != nil {
		return nil, fmt.Errorf("durable: failed to validate round-trip: %w", err)
	}

	if humanReparsed.Content != human.Content {
		return nil, ErrRoundTripNotPreserved
	}

	return output, nil
}
