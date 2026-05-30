package skills

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed skill-frontmatter.schema.json
var _frontmatterSchemaJSON []byte

const _frontmatterSchemaURI = "skill-frontmatter.schema.json"

// _loadFrontmatterSchema compila o JSON Schema embutido sob demanda, uma unica
// vez, de forma thread-safe (Regra 7.10 — sync.OnceValues no lugar de init()).
var _loadFrontmatterSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(_frontmatterSchemaJSON))
	if err != nil {
		return nil, fmt.Errorf("schema de frontmatter invalido: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(_frontmatterSchemaURI, schemaDoc); err != nil {
		return nil, fmt.Errorf("carregar schema de frontmatter: %w", err)
	}
	schema, err := compiler.Compile(_frontmatterSchemaURI)
	if err != nil {
		return nil, fmt.Errorf("compilar schema de frontmatter: %w", err)
	}
	return schema, nil
})

// frontmatterDoc e a representacao serializavel do Frontmatter para validacao via JSON Schema.
type frontmatterDoc struct {
	Name        string   `json:"name,omitempty"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Triggers    []string `json:"triggers,omitempty"`
	Lang        string   `json:"lang,omitempty"`
	LinkMode    string   `json:"link_mode,omitempty"`
	DependsOn   []string `json:"depends_on,omitempty"`
	MaxDepth    *int     `json:"max_depth,omitempty"`
}

// ValidateFrontmatterSchema valida o frontmatter de um SKILL.md contra o JSON Schema formal.
// skillName e usado para gerar mensagens de erro mais claras; pode ser vazio.
// Retorna erro descrevendo o campo invalido e a skill afetada.
func (catalog *Catalog) ValidateFrontmatterSchema(content []byte, skillName string) error {
	fm := NewCatalog().ParseFrontmatter(content)
	doc := NewCatalog().toFrontmatterDoc(fm)

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("serializar frontmatter: %w", err)
	}

	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("parsear frontmatter como JSON: %w", err)
	}

	schema, err := _loadFrontmatterSchema()
	if err != nil {
		return fmt.Errorf("carregar schema de frontmatter: %w", err)
	}

	if err := schema.Validate(payload); err != nil {
		if skillName != "" {
			return fmt.Errorf("skill %q: %s", skillName, NewCatalog().formatFrontmatterError(err))
		}
		return fmt.Errorf("%s", NewCatalog().formatFrontmatterError(err))
	}
	return nil
}

func (catalog *Catalog) toFrontmatterDoc(fm Frontmatter) frontmatterDoc {
	doc := frontmatterDoc{
		Name:        fm.Name,
		Version:     fm.Version,
		Description: fm.Description,
		Lang:        fm.Lang,
		LinkMode:    fm.LinkMode,
	}
	if len(fm.Triggers) > 0 {
		doc.Triggers = fm.Triggers
	}
	if len(fm.DependsOn) > 0 {
		doc.DependsOn = fm.DependsOn
	}
	if fm.MaxDepth > 0 {
		doc.MaxDepth = &fm.MaxDepth
	}
	return doc
}

func (catalog *Catalog) formatFrontmatterError(err error) string {
	var ve *jsonschema.ValidationError
	if ok := NewCatalog().asSchemaValidationError(err, &ve); ok {
		return ve.Error()
	}
	msgs := []string{}
	for _, line := range strings.Split(err.Error(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			msgs = append(msgs, line)
		}
	}
	return strings.Join(msgs, "; ")
}

func (catalog *Catalog) asSchemaValidationError(err error, target **jsonschema.ValidationError) bool {
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		*target = ve
		return true
	}
	return false
}
