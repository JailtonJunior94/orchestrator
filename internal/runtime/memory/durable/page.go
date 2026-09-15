package durable

import (
	"fmt"
	"regexp"
	"strings"
	"time"

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

var frontMatterPattern = regexp.MustCompile(`(?s)\A---\n(.*?)\n---\n`)

type MarkdownPage struct{}

func NewMarkdownPage() *MarkdownPage {
	return &MarkdownPage{}
}

func (p *MarkdownPage) Parse(content []byte) ([]Fact, HumanBlock, error) {
	text := string(content)
	if strings.HasPrefix(text, "---\n") && frontMatterPattern.FindStringSubmatchIndex(text) == nil {
		return nil, HumanBlock{}, ErrPageUnreadable
	}

	var header PageHeader
	if match := frontMatterPattern.FindStringSubmatchIndex(text); match != nil {
		yamlBlock := text[match[2]:match[3]]
		if err := yaml.Unmarshal([]byte(yamlBlock), &header); err != nil {
			return nil, HumanBlock{}, fmt.Errorf("durable: invalid frontmatter: %w: %w", err, ErrPageUnreadable)
		}
		if err := validatePageHeader(header); err != nil {
			return nil, HumanBlock{}, err
		}
		text = text[match[1]:]
	}

	facts := make([]Fact, 0)
	segments := make([]string, 0, 1)
	cursor := 0
	for {
		start, key, yamlBlock, end, found := p.nextFactBlock(text, cursor)
		if !found {
			break
		}
		segments = append(segments, text[cursor:start])

		var metadata factMetadataYAML
		if err := yaml.Unmarshal([]byte(yamlBlock), &metadata); err != nil {
			return nil, HumanBlock{}, fmt.Errorf("durable: invalid fact metadata: %w: %w", err, ErrPageUnreadable)
		}

		fact := Fact{
			Identity: Identity{
				Key:  SemanticKey(key),
				Hash: metadata.Hash,
			},
			Content:    metadata.Content,
			Durability: metadata.Durability,
			Origin:     metadata.Origin,
			State:      metadata.State,
			Links:      metadata.Links,
		}
		if err := NewCatalog().ValidateFact(fact); err != nil {
			return nil, HumanBlock{}, fmt.Errorf("durable: invalid fact metadata: %w: %w", err, ErrPageUnreadable)
		}
		facts = append(facts, fact)

		cursor = end
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

func (p *MarkdownPage) nextFactBlock(text string, cursor int) (int, string, string, int, bool) {
	const factPrefix = "### Fact: "
	const metadataPrefix = "\n```fact-metadata\n"
	const metadataSuffix = "\n```"

	for offset := cursor; ; {
		relativeStart := strings.Index(text[offset:], factPrefix)
		if relativeStart < 0 {
			return 0, "", "", 0, false
		}
		start := offset + relativeStart
		if start > 0 && text[start-1] != '\n' {
			offset = start + len(factPrefix)
			continue
		}
		keyStart := start + len(factPrefix)
		keyEndRelative := strings.IndexByte(text[keyStart:], '\n')
		if keyEndRelative < 0 {
			return 0, "", "", 0, false
		}
		keyEnd := keyStart + keyEndRelative
		metadataStart := keyEnd
		if !strings.HasPrefix(text[metadataStart:], metadataPrefix) {
			offset = keyEnd + 1
			continue
		}
		metadataStart += len(metadataPrefix)
		metadataEndRelative := strings.Index(text[metadataStart:], metadataSuffix)
		if metadataEndRelative < 0 {
			return 0, "", "", 0, false
		}
		metadataEnd := metadataStart + metadataEndRelative
		end := metadataEnd + len(metadataSuffix)
		for end < len(text) && (text[end] == ' ' || text[end] == '\t') {
			end++
		}
		if end < len(text) && text[end] == '\n' {
			end++
		}
		return start, strings.TrimSpace(text[keyStart:keyEnd]), text[metadataStart:metadataEnd], end, true
	}
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
	header = completePageHeader(header, facts)
	if err := validatePageHeader(header); err != nil {
		return nil, err
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

func completePageHeader(header PageHeader, facts []Fact) PageHeader {
	if header.Identity == "" {
		header.Identity = "memory"
	}
	if header.Layer == TargetLayerUndefined {
		header.Layer = TargetLayerPRD
	}
	if header.OriginSession == "" {
		header.OriginSession = "manual"
		for _, fact := range facts {
			if fact.Origin.Session != "" {
				header.OriginSession = fact.Origin.Session
				break
			}
		}
	}
	if header.Date == "" {
		header.Date = time.Now().UTC().Format(time.RFC3339)
		for _, fact := range facts {
			if fact.Origin.Date != "" {
				header.Date = fact.Origin.Date
				break
			}
		}
	}
	if header.FormatVersion == 0 {
		header.FormatVersion = FormatVersionCurrent
	}
	return header
}

func validatePageHeader(header PageHeader) error {
	if header.Identity == "" || header.Layer == TargetLayerUndefined || header.OriginSession == "" || header.Date == "" || header.FormatVersion != FormatVersionCurrent {
		return ErrPageUnreadable
	}
	if _, err := time.Parse(time.RFC3339, header.Date); err != nil {
		return fmt.Errorf("durable: invalid frontmatter date: %w: %w", err, ErrPageUnreadable)
	}
	return nil
}
