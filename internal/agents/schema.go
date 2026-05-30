package agents

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

//go:embed agent-frontmatter.schema.json
var _agentFrontmatterSchemaJSON []byte

const _agentFrontmatterSchemaURI = "agent-frontmatter.schema.json"

// _loadAgentFrontmatterSchema compila o JSON Schema embutido sob demanda, uma
// unica vez, de forma thread-safe (Regra 7.10 — sync.OnceValues no lugar de init()).
var _loadAgentFrontmatterSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(_agentFrontmatterSchemaJSON))
	if err != nil {
		return nil, fmt.Errorf("schema de frontmatter de agente invalido: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(_agentFrontmatterSchemaURI, schemaDoc); err != nil {
		return nil, fmt.Errorf("carregar schema de frontmatter de agente: %w", err)
	}
	schema, err := compiler.Compile(_agentFrontmatterSchemaURI)
	if err != nil {
		return nil, fmt.Errorf("compilar schema de frontmatter de agente: %w", err)
	}
	return schema, nil
})

// agentFrontmatterDoc e a representacao serializavel do frontmatter de AGENT.md para validacao JSON Schema.
type agentFrontmatterDoc struct {
	Name        string           `json:"name,omitempty"`
	Description string           `json:"description,omitempty"`
	Version     string           `json:"version,omitempty"`
	Runtime     *agentRuntimeDoc `json:"runtime,omitempty"`
}

type agentRuntimeDoc struct {
	IDE             string `json:"ide,omitempty"`
	Model           string `json:"model,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	AccessMode      string `json:"access_mode,omitempty"`
}

// ValidateAgentFrontmatter parseia e valida o frontmatter de um AGENT.md, retornando o ResolvedAgent
// correspondente ou um erro sentinela descritivo. dirName e o nome do diretorio pai (para RF-06);
// pode ser vazio para ignorar a verificacao de coincidencia de nome.
func (c *Catalog) ValidateAgentFrontmatter(content []byte, dirName string) (ResolvedAgent, error) {
	fields := skills.NewCatalog().ParseFrontmatterFields(content)

	doc := agentFrontmatterDoc{
		Name:        fields["name"],
		Description: fields["description"],
		Version:     fields["version"],
	}

	// Coletar campos de runtime se presentes.
	if ide := fields["runtime.ide"]; ide != "" ||
		fields["runtime.model"] != "" ||
		fields["runtime.reasoning_effort"] != "" ||
		fields["runtime.access_mode"] != "" {
		doc.Runtime = &agentRuntimeDoc{
			IDE:             fields["runtime.ide"],
			Model:           fields["runtime.model"],
			ReasoningEffort: fields["runtime.reasoning_effort"],
			AccessMode:      fields["runtime.access_mode"],
		}
	}

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return ResolvedAgent{}, fmt.Errorf("%w: serializar frontmatter: %v", ErrFrontmatterInvalid, err)
	}

	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
	if err != nil {
		return ResolvedAgent{}, fmt.Errorf("%w: parsear frontmatter como JSON: %v", ErrFrontmatterInvalid, err)
	}

	schema, err := _loadAgentFrontmatterSchema()
	if err != nil {
		return ResolvedAgent{}, fmt.Errorf("%w: %v", ErrFrontmatterInvalid, err)
	}

	if err := schema.Validate(payload); err != nil {
		return ResolvedAgent{}, fmt.Errorf("%w: %s", ErrFrontmatterInvalid, NewCatalog().formatAgentSchemaError(err))
	}

	// RF-06: name no frontmatter deve coincidir com o nome do diretorio pai.
	if dirName != "" && doc.Name != "" && doc.Name != dirName {
		return ResolvedAgent{}, fmt.Errorf("%w: campo name %q diverge do diretorio %q",
			ErrNameDirMismatch, doc.Name, dirName)
	}

	agent := ResolvedAgent{
		Name: doc.Name,
		Metadata: Metadata{
			Description: doc.Description,
			Version:     doc.Version,
		},
		Prompt: NewCatalog().extractPromptBody(content),
	}

	if doc.Runtime != nil {
		agent.Runtime = RuntimeDefaults{
			IDE:             doc.Runtime.IDE,
			Model:           doc.Runtime.Model,
			ReasoningEffort: doc.Runtime.ReasoningEffort,
			AccessMode:      doc.Runtime.AccessMode,
		}
	}

	return agent, nil
}

// extractPromptBody extrai o corpo do AGENT.md apos o segundo delimitador ---.
func (c *Catalog) extractPromptBody(content []byte) string {
	text := string(content)
	count := 0
	lines := strings.Split(text, "\n")
	bodyStart := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "---" {
			count++
			if count == 2 {
				bodyStart = i + 1
				break
			}
		}
	}
	if bodyStart < 0 || bodyStart >= len(lines) {
		return ""
	}
	return strings.TrimSpace(strings.Join(lines[bodyStart:], "\n"))
}

func (c *Catalog) formatAgentSchemaError(err error) string {
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		return ve.Error()
	}
	msgs := []string{}
	for line := range strings.SplitSeq(err.Error(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			msgs = append(msgs, line)
		}
	}
	return strings.Join(msgs, "; ")
}
