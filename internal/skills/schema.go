package skills

import (
	"bytes"
	"encoding/json"
	_ "embed"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed skill-frontmatter.schema.json
var frontmatterSchemaJSON []byte

const frontmatterSchemaURI = "skill-frontmatter.schema.json"

var compiledFrontmatterSchema *jsonschema.Schema

func init() {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(frontmatterSchemaJSON))
	if err != nil {
		panic(fmt.Sprintf("schema de frontmatter invalido: %v", err))
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(frontmatterSchemaURI, schemaDoc); err != nil {
		panic(fmt.Sprintf("carregar schema de frontmatter: %v", err))
	}
	compiledFrontmatterSchema, err = compiler.Compile(frontmatterSchemaURI)
	if err != nil {
		panic(fmt.Sprintf("compilar schema de frontmatter: %v", err))
	}
}

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
func ValidateFrontmatterSchema(content []byte, skillName string) error {
	fm := ParseFrontmatter(content)
	doc := toFrontmatterDoc(fm)

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("serializar frontmatter: %w", err)
	}

	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("parsear frontmatter como JSON: %w", err)
	}

	if err := compiledFrontmatterSchema.Validate(payload); err != nil {
		if skillName != "" {
			return fmt.Errorf("skill %q: %s", skillName, formatFrontmatterError(err))
		}
		return fmt.Errorf("%s", formatFrontmatterError(err))
	}
	return nil
}

func toFrontmatterDoc(fm Frontmatter) frontmatterDoc {
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

func formatFrontmatterError(err error) string {
	var ve *jsonschema.ValidationError
	if ok := asSchemaValidationError(err, &ve); ok {
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

func asSchemaValidationError(err error, target **jsonschema.ValidationError) bool {
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		*target = ve
		return true
	}
	return false
}
