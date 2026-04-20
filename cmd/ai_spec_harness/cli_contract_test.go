package aispecharness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// schemaDefault representa a secao "default" do cli-schema.json.
type schemaDefault struct {
	Commands []schemaCommand `json:"commands"`
}

type schemaCommand struct {
	Name        string                     `json:"name"`
	Flags       map[string]schemaFlag      `json:"flags,omitempty"`
	Subcommands []schemaCommand            `json:"subcommands,omitempty"`
}

type schemaFlag struct {
	Type     string `json:"type"`
	Required bool   `json:"required"`
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

// flattenSchemaCommandMap retorna mapa de caminho -> schemaCommand para todos os comandos.
func flattenSchemaCommandMap(cmds []schemaCommand, prefix string) map[string]schemaCommand {
	result := make(map[string]schemaCommand)
	for _, cmd := range cmds {
		path := cmd.Name
		if prefix != "" {
			path = prefix + " " + cmd.Name
		}
		result[path] = cmd
		for k, v := range flattenSchemaCommandMap(cmd.Subcommands, path) {
			result[k] = v
		}
	}
	return result
}

// cobraCommandMap retorna mapa de caminho -> *cobra.Command para todos os comandos.
func cobraCommandMap(cmds []*cobra.Command, prefix string, excluded map[string]bool) map[string]*cobra.Command {
	result := make(map[string]*cobra.Command)
	for _, cmd := range cmds {
		if cmd.Hidden || excluded[cmd.Name()] {
			continue
		}
		path := cmd.Name()
		if prefix != "" {
			path = prefix + " " + cmd.Name()
		}
		result[path] = cmd
		for k, v := range cobraCommandMap(cmd.Commands(), path, excluded) {
			result[k] = v
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

// TestCLI_ContractFlagsMatchSchema valida que flags definidas no cli-schema.json
// existem na implementacao Cobra, e vice-versa (sem drift de flags).
func TestCLI_ContractFlagsMatchSchema(t *testing.T) {
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

	// Flags globais declaradas no schema como globalFlags (nao repetir por comando)
	globalSchemaFlags := map[string]bool{"verbose": true}
	// Flags internas do cobra excluidas da verificacao
	cobraInternalFlags := map[string]bool{"help": true}

	excluded := map[string]bool{"help": true, "completion": true}
	schemaMap := flattenSchemaCommandMap(raw.Default.Commands, "")
	cobraMap := cobraCommandMap(rootCmd.Commands(), "", excluded)

	for cmdPath, sCmd := range schemaMap {
		cmdPath := cmdPath
		sCmd := sCmd
		cobraCmd, ok := cobraMap[cmdPath]
		if !ok {
			continue // ja reportado em TestCLI_ContractMatchesSchema
		}

		t.Run(cmdPath, func(t *testing.T) {
			// Flags no schema mas ausentes no Cobra
			for flagName := range sCmd.Flags {
				found := cobraCmd.Flags().Lookup(flagName) != nil ||
					cobraCmd.PersistentFlags().Lookup(flagName) != nil
				if !found {
					t.Errorf("flag --%s definida no schema para %q mas ausente no Cobra", flagName, cmdPath)
				}
			}

			// Flags no Cobra mas ausentes no schema
			cobraCmd.Flags().VisitAll(func(f *pflag.Flag) {
				name := f.Name
				if cobraInternalFlags[name] || globalSchemaFlags[name] {
					return
				}
				if _, inSchema := sCmd.Flags[name]; !inSchema {
					t.Errorf("flag --%s implementada no Cobra para %q mas ausente no schema", name, cmdPath)
				}
			})
		})
	}
}

