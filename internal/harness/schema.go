package harness

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed harness-contract.schema.json
var contractSchemaJSON []byte

const contractSchemaURI = "harness-contract.schema.json"

var loadContractSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(contractSchemaJSON))
	if err != nil {
		return nil, fmt.Errorf("invalid harness contract schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(contractSchemaURI, schemaDoc); err != nil {
		return nil, fmt.Errorf("load harness contract schema: %w", err)
	}
	schema, err := compiler.Compile(contractSchemaURI)
	if err != nil {
		return nil, fmt.Errorf("compile harness contract schema: %w", err)
	}
	return schema, nil
})

type Validator interface {
	Validate(raw map[string]any) error
}

type SchemaValidator struct {
	classifier *ErrorClassifier
}

var _ Validator = (*SchemaValidator)(nil)

func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{classifier: NewErrorClassifier()}
}

func (v *SchemaValidator) Validate(raw map[string]any) error {
	schema, err := loadContractSchema()
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("encode harness contract as JSON: %w", err)
	}

	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("decode harness contract JSON: %w", err)
	}

	if err := schema.Validate(payload); err != nil {
		return v.classifier.Classify(err, raw)
	}
	return nil
}
