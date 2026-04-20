package aispecharness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// schemaDefault representa a secao "default" do cli-schema.json.
type schemaDefault struct {
	Commands []schemaCommand `json:"commands"`
}

type schemaCommand struct {
	Name        string          `json:"name"`
	Subcommands []schemaCommand `json:"subcommands,omitempty"`
}

// flattenSchemaCommands retorna todos os caminhos de comando (ex: "telemetry log") do schema.
func flattenSchemaCommands(cmds []schemaCommand, prefix string) []string {
	var result []string
	for _, cmd := range cmds {
		path := cmd.Name
		if prefix != "" {
			path = prefix + " " + cmd.Name
		}
		result = append(result, path)
		if len(cmd.Subcommands) > 0 {
			result = append(result, flattenSchemaCommands(cmd.Subcommands, path)...)
		}
	}
	return result
}

// TestCLI_ContractMatchesSchema valida que todos os comandos no cli-schema.json
// tem implementacao correspondente no Cobra, e vice-versa.
func TestCLI_ContractMatchesSchema(t *testing.T) {
	schemaPath := filepath.Join("..", "..", "docs", "cli-schema.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("ler cli-schema.json: %v", err)
	}

	var raw struct {
		Default schemaDefault `json:"default"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parsear cli-schema.json: %v", err)
	}

	schemaCmds := flattenSchemaCommands(raw.Default.Commands, "")
	sort.Strings(schemaCmds)

	// Extrair comandos do Cobra (excluindo help e completion — auto-gerados)
	excluded := map[string]bool{"help": true, "completion": true}
	var cobraCmds []string
	for _, cmd := range rootCmd.Commands() {
		if cmd.Hidden || excluded[cmd.Name()] {
			continue
		}
		cobraCmds = append(cobraCmds, cmd.Name())
		for _, sub := range cmd.Commands() {
			if sub.Hidden || excluded[sub.Name()] {
				continue
			}
			cobraCmds = append(cobraCmds, cmd.Name()+" "+sub.Name())
		}
	}
	sort.Strings(cobraCmds)

	schemaSet := make(map[string]bool, len(schemaCmds))
	for _, c := range schemaCmds {
		schemaSet[c] = true
	}
	cobraSet := make(map[string]bool, len(cobraCmds))
	for _, c := range cobraCmds {
		cobraSet[c] = true
	}

	for _, c := range schemaCmds {
		if !cobraSet[c] {
			t.Errorf("comando no schema mas nao implementado no Cobra: %q", c)
		}
	}
	for _, c := range cobraCmds {
		if !schemaSet[c] {
			t.Errorf("comando implementado no Cobra mas ausente no schema: %q", c)
		}
	}
}
