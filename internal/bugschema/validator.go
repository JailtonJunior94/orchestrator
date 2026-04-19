package bugschema

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Validate valida o array de bugs em bugsPath contra o schema JSON em schemaPath.
// schemaPath deve apontar para um arquivo JSON Schema valido (draft 2020-12).
// Retorna erro explicito se o schema nao for encontrado, for invalido,
// ou se o payload nao for conforme.
func Validate(bugsPath, schemaPath string) error {
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("ler schema %q: %w", schemaPath, err)
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return fmt.Errorf("parsear schema %q: %w", schemaPath, err)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaPath, schemaDoc); err != nil {
		return fmt.Errorf("carregar schema %q: %w", schemaPath, err)
	}

	schema, err := compiler.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("compilar schema %q: %w", schemaPath, err)
	}

	bugsData, err := os.ReadFile(bugsPath)
	if err != nil {
		return fmt.Errorf("ler arquivo de bugs %q: %w", bugsPath, err)
	}

	payload, err := jsonschema.UnmarshalJSON(bytes.NewReader(bugsData))
	if err != nil {
		return fmt.Errorf("parsear JSON de bugs: %w", err)
	}

	if err := schema.Validate(payload); err != nil {
		return fmt.Errorf("validacao falhou: %s", formatValidationError(err))
	}

	return nil
}

func formatValidationError(err error) string {
	var ve *jsonschema.ValidationError
	if ok := asValidationError(err, &ve); ok {
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

func asValidationError(err error, target **jsonschema.ValidationError) bool {
	if ve, ok := err.(*jsonschema.ValidationError); ok {
		*target = ve
		return true
	}
	return false
}
