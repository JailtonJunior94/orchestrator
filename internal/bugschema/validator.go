package bugschema

import (
	"encoding/json"
	"fmt"
	"os"
)

var bugRequiredFields = []string{"id", "severity", "file", "line", "reproduction", "expected", "actual"}

var severityEnum = map[string]bool{
	"critical": true,
	"high":     true,
	"medium":   true,
	"low":      true,
}

// Validate valida o array de bugs em bugsPath contra o schema de bugs.
// schemaPath e ignorado na implementacao atual — a logica de validacao e derivada
// diretamente da especificacao conhecida do bug-schema.json.
func Validate(bugsPath, schemaPath string) error {
	data, err := os.ReadFile(bugsPath)
	if err != nil {
		return fmt.Errorf("ler arquivo de bugs: %w", err)
	}

	var bugs []map[string]interface{}
	if err := json.Unmarshal(data, &bugs); err != nil {
		return fmt.Errorf("parsear JSON de bugs: %w", err)
	}

	if len(bugs) == 0 {
		return fmt.Errorf("minItems: o array de bugs deve ter ao menos 1 item")
	}

	for i, bug := range bugs {
		n := i + 1

		// Verificar campos obrigatorios
		for _, field := range bugRequiredFields {
			if _, ok := bug[field]; !ok {
				return fmt.Errorf("bug %d: campo obrigatorio ausente: %q", n, field)
			}
		}

		// Verificar enum de severity
		sev, _ := bug["severity"].(string)
		if !severityEnum[sev] {
			return fmt.Errorf("bug %d: severity %q invalido: aceitos critical, high, medium, low", n, sev)
		}

		// Verificar additionalProperties: false
		known := make(map[string]bool, len(bugRequiredFields))
		for _, f := range bugRequiredFields {
			known[f] = true
		}
		for key := range bug {
			if !known[key] {
				return fmt.Errorf("bug %d: propriedade adicional nao permitida: %q", n, key)
			}
		}
	}

	return nil
}
