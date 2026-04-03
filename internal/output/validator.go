package output

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	// ErrInvalidJSON indicates syntactically invalid JSON.
	ErrInvalidJSON = errors.New("invalid json payload")
	// ErrSchemaValidation indicates schema validation failures.
	ErrSchemaValidation = errors.New("json payload does not satisfy schema")
)

// ValidateJSON parses and optionally validates a JSON document.
func ValidateJSON(_ context.Context, payload []byte, schema []byte) error {
	if !json.Valid(payload) {
		return ErrInvalidJSON
	}

	if len(schema) == 0 {
		return nil
	}

	var schemaDocument any
	if err := json.Unmarshal(schema, &schemaDocument); err != nil {
		return fmt.Errorf("%w: %v", ErrSchemaValidation, err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("schema.json", schemaDocument); err != nil {
		return fmt.Errorf("%w: %v", ErrSchemaValidation, err)
	}

	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSchemaValidation, err)
	}

	var document any
	if err := json.Unmarshal(payload, &document); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}

	if err := compiled.Validate(document); err != nil {
		return fmt.Errorf("%w: %v", ErrSchemaValidation, err)
	}

	return nil
}
