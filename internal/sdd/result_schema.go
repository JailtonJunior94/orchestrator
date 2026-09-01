package sdd

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed execution-result.schema.json
var _executionResultSchemaJSON []byte

//go:embed review-result.schema.json
var _reviewResultSchemaJSON []byte

const (
	_executionResultSchemaURI = "execution-result.schema.json"
	_reviewResultSchemaURI    = "review-result.schema.json"
)

// ReviewResult é o contrato independente produzido pelo revisor read-only.
type ReviewResult struct {
	SchemaVersion    int              `json:"schema_version"`
	RunID            string           `json:"run_id"`
	TaskID           string           `json:"task_id"`
	Attempt          int              `json:"attempt"`
	BaseSHA          string           `json:"base_sha"`
	PatchSHA256      string           `json:"patch_sha256"`
	FinalStateSHA256 string           `json:"final_state_sha256"`
	Tests            []TestProof      `json:"tests"`
	Criteria         []CriterionProof `json:"criteria"`
	Evidence         []string         `json:"evidence"`
	Verdict          string           `json:"verdict"`
}

var _loadExecutionResultSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	return NewResultValidator().compileSchema(_executionResultSchemaURI, _executionResultSchemaJSON)
})

var _loadReviewResultSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	return NewResultValidator().compileSchema(_reviewResultSchemaURI, _reviewResultSchemaJSON)
})

// ResultValidator valida os contratos JSON que atravessam executor, revisor e checkpoint.
type ResultValidator struct{}

// NewResultValidator cria um validador sem estado e seguro para reuso concorrente.
func NewResultValidator() *ResultValidator {
	return &ResultValidator{}
}

// ValidateExecutionJSON rejeita JSON incompleto, campos desconhecidos e provas inconsistentes.
func (v *ResultValidator) ValidateExecutionJSON(content []byte) (ExecutionResult, error) {
	if err := v.validateJSON(content, _loadExecutionResultSchema); err != nil {
		return ExecutionResult{}, fmt.Errorf("validar resultado de execucao: %w", err)
	}
	var result ExecutionResult
	if err := json.Unmarshal(content, &result); err != nil {
		return ExecutionResult{}, fmt.Errorf("decodificar resultado de execucao: %w", err)
	}
	if err := v.validateEvidencePaths(result.Evidence, result.Criteria); err != nil {
		return ExecutionResult{}, err
	}
	return result, nil
}

// ValidateCheckpointJSON usa o mesmo contrato da execucao. Assim checkpoint,
// resultado do executor e estado posterior compartilham identidade, status e
// provas, sem um vocabulário paralelo suscetível a drift.
func (v *ResultValidator) ValidateCheckpointJSON(content []byte) (ExecutionResult, error) {
	result, err := v.ValidateExecutionJSON(content)
	if err != nil {
		return ExecutionResult{}, fmt.Errorf("validar checkpoint: %w", err)
	}
	return result, nil
}

// ValidateReviewJSON rejeita JSON incompleto, campos desconhecidos e vereditos fora do vocabulário.
func (v *ResultValidator) ValidateReviewJSON(content []byte) (ReviewResult, error) {
	if err := v.validateJSON(content, _loadReviewResultSchema); err != nil {
		return ReviewResult{}, fmt.Errorf("validar resultado de revisao: %w", err)
	}
	var result ReviewResult
	if err := json.Unmarshal(content, &result); err != nil {
		return ReviewResult{}, fmt.Errorf("decodificar resultado de revisao: %w", err)
	}
	if err := v.validateEvidencePaths(result.Evidence, result.Criteria); err != nil {
		return ReviewResult{}, err
	}
	return result, nil
}

func (v *ResultValidator) validateEvidencePaths(evidence []string, criteria []CriterionProof) error {
	for _, path := range evidence {
		if err := v.validateEvidencePath(path); err != nil {
			return fmt.Errorf("validar evidencia: %w", err)
		}
	}
	for _, criterion := range criteria {
		if err := v.validateEvidencePath(criterion.EvidenceRef); err != nil {
			return fmt.Errorf("validar evidencia do criterio %s: %w", criterion.ID, err)
		}
	}
	return nil
}

func (v *ResultValidator) validateEvidencePath(reference string) error {
	path, _, _ := strings.Cut(reference, "#")
	normalized := strings.ReplaceAll(path, "\\", "/")
	if path == "" || filepath.IsAbs(path) || filepath.VolumeName(path) != "" ||
		(len(normalized) >= 3 && normalized[1] == ':' && normalized[2] == '/') {
		return fmt.Errorf("referencia de evidencia deve ser caminho relativo: %q", reference)
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return fmt.Errorf("referencia de evidencia contem escape de diretorio: %q", reference)
		}
	}
	return nil
}

func (v *ResultValidator) validateJSON(content []byte, loader func() (*jsonschema.Schema, error)) error {
	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("parsear JSON: %w", err)
	}
	schema, err := loader()
	if err != nil {
		return err
	}
	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("contrato invalido: %s", v.formatSchemaError(err))
	}
	return nil
}

func (v *ResultValidator) compileSchema(uri string, content []byte) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("parsear schema %s: %w", uri, err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(uri, doc); err != nil {
		return nil, fmt.Errorf("carregar schema %s: %w", uri, err)
	}
	schema, err := compiler.Compile(uri)
	if err != nil {
		return nil, fmt.Errorf("compilar schema %s: %w", uri, err)
	}
	return schema, nil
}

func (v *ResultValidator) formatSchemaError(err error) string {
	if validationError, ok := err.(*jsonschema.ValidationError); ok {
		return validationError.Error()
	}
	return strings.Join(strings.Fields(err.Error()), " ")
}
