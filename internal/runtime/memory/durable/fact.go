package durable

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type SemanticKey string

type ContentHash string

type Identity struct {
	Key  SemanticKey
	Hash ContentHash
}

type Durability uint8

const (
	DurabilityInvalid Durability = iota
	DurabilityEphemeral
	DurabilityPRD
	DurabilityDurable
)

func (d Durability) String() string {
	switch d {
	case DurabilityEphemeral:
		return "ephemeral"
	case DurabilityPRD:
		return "prd"
	case DurabilityDurable:
		return "durable"
	default:
		return ""
	}
}

func durabilityFromString(s string) (Durability, error) {
	switch s {
	case "ephemeral":
		return DurabilityEphemeral, nil
	case "prd":
		return DurabilityPRD, nil
	case "durable":
		return DurabilityDurable, nil
	case "":
		return DurabilityInvalid, nil
	default:
		return DurabilityInvalid, fmt.Errorf("durable: unknown durability %q: %w", s, ErrPageUnreadable)
	}
}

func (d Durability) MarshalYAML() (any, error) {
	return d.String(), nil
}

func (d *Durability) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := durabilityFromString(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

type FactState uint8

const (
	FactStateUndefined FactState = iota
	FactStateProposed
	FactStateActive
	FactStateContradicted
	FactStatePromoted
	FactStateArchived
)

func (e FactState) String() string {
	switch e {
	case FactStateProposed:
		return "proposed"
	case FactStateActive:
		return "active"
	case FactStateContradicted:
		return "contradicted"
	case FactStatePromoted:
		return "promoted"
	case FactStateArchived:
		return "archived"
	default:
		return ""
	}
}

func factStateFromString(s string) (FactState, error) {
	switch s {
	case "proposed":
		return FactStateProposed, nil
	case "active":
		return FactStateActive, nil
	case "contradicted":
		return FactStateContradicted, nil
	case "promoted":
		return FactStatePromoted, nil
	case "archived":
		return FactStateArchived, nil
	case "":
		return FactStateUndefined, nil
	default:
		return FactStateUndefined, fmt.Errorf("durable: unknown fact state %q: %w", s, ErrPageUnreadable)
	}
}

func (e FactState) MarshalYAML() (any, error) {
	return e.String(), nil
}

func (e *FactState) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := factStateFromString(s)
	if err != nil {
		return err
	}
	*e = parsed
	return nil
}

type LinkType uint8

const (
	LinkTypeUndefined LinkType = iota
	LinkTypeReplaces
	LinkTypeCauses
	LinkTypeFixes
	LinkTypeContradicts
)

func (t LinkType) String() string {
	switch t {
	case LinkTypeReplaces:
		return "replaces"
	case LinkTypeCauses:
		return "causes"
	case LinkTypeFixes:
		return "fixes"
	case LinkTypeContradicts:
		return "contradicts"
	default:
		return ""
	}
}

func linkTypeFromString(s string) (LinkType, error) {
	switch s {
	case "replaces":
		return LinkTypeReplaces, nil
	case "causes":
		return LinkTypeCauses, nil
	case "fixes":
		return LinkTypeFixes, nil
	case "contradicts":
		return LinkTypeContradicts, nil
	case "":
		return LinkTypeUndefined, nil
	default:
		return LinkTypeUndefined, fmt.Errorf("durable: unknown link type %q: %w", s, ErrPageUnreadable)
	}
}

func (t LinkType) MarshalYAML() (any, error) {
	return t.String(), nil
}

func (t *LinkType) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := linkTypeFromString(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

type Link struct {
	Type   LinkType `yaml:"type"`
	Target Identity `yaml:"target"`
}

type FactOrigin struct {
	Session string `yaml:"session"`
	CLI     string `yaml:"cli"`
	Task    string `yaml:"task"`
	Date    string `yaml:"date"`
}

type Fact struct {
	Identity   Identity
	Content    string
	Durability Durability
	Origin     FactOrigin
	State      FactState
	Links      []Link
}

type TargetLayer uint8

const (
	TargetLayerUndefined TargetLayer = iota
	TargetLayerTask
	TargetLayerPRD
	TargetLayerProject
)

func (c TargetLayer) String() string {
	switch c {
	case TargetLayerTask:
		return "task"
	case TargetLayerPRD:
		return "prd"
	case TargetLayerProject:
		return "project"
	default:
		return ""
	}
}

func targetLayerFromString(s string) (TargetLayer, error) {
	switch s {
	case "task":
		return TargetLayerTask, nil
	case "prd":
		return TargetLayerPRD, nil
	case "project":
		return TargetLayerProject, nil
	case "":
		return TargetLayerUndefined, nil
	default:
		return TargetLayerUndefined, fmt.Errorf("durable: unknown layer %q: %w", s, ErrPageUnreadable)
	}
}

func (c TargetLayer) MarshalYAML() (any, error) {
	return c.String(), nil
}

func (c *TargetLayer) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := targetLayerFromString(s)
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

func (c *Catalog) ValidateFact(f Fact) error {
	if f.Identity.Key == "" {
		return ErrSemanticKeyMissing
	}
	if f.Durability == DurabilityInvalid {
		return ErrDurabilityMissing
	}
	return nil
}

func (c *Catalog) ResolveLayer(d Durability) (TargetLayer, error) {
	switch d {
	case DurabilityEphemeral:
		return TargetLayerTask, nil
	case DurabilityPRD:
		return TargetLayerPRD, nil
	case DurabilityDurable:
		return TargetLayerProject, nil
	default:
		return TargetLayerUndefined, ErrDurabilityMissing
	}
}

func (c *Catalog) HashContent(content string) ContentHash {
	normalized := strings.TrimSpace(content)
	sum := sha256.Sum256([]byte(normalized))
	return ContentHash("sha256:" + hex.EncodeToString(sum[:]))
}

type CollisionResult uint8

const (
	CollisionNone CollisionResult = iota
	CollisionIdempotent
	CollisionContradictory
)

func (c *Catalog) DetectCollision(existing, candidate Fact) CollisionResult {
	if existing.Identity.Key != candidate.Identity.Key {
		return CollisionNone
	}
	if existing.Identity.Hash == candidate.Identity.Hash {
		return CollisionIdempotent
	}
	return CollisionContradictory
}

type StructuredSignal struct {
	Kind    string
	Subject string
	Scope   string
}

func (c *Catalog) DeriveSemanticKey(signal StructuredSignal) (SemanticKey, error) {
	kind := normalizeToken(signal.Kind)
	subject := normalizeToken(signal.Subject)
	scope := normalizeToken(signal.Scope)
	if kind == "" || subject == "" {
		return "", ErrSemanticKeyMissing
	}
	parts := []string{kind, subject}
	if scope != "" {
		parts = append(parts, scope)
	}
	return SemanticKey(strings.Join(parts, ".")), nil
}

func normalizeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}
