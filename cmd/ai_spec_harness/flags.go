package aispecharness

import (
	"fmt"
	"os"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/spf13/cobra"
)

// flagHelper agrupa validacoes e parsers de flags usados pelos comandos.
type flagHelper struct{}

func newFlagHelper() *flagHelper {
	return &flagHelper{}
}

// requireFlag valida que uma flag obrigatoria esta presente e nao vazia.
// Diferencia flag ausente de flag presente mas vazia, retornando mensagem
// amigavel em PT-BR com exemplo de uso real do comando.
func (h *flagHelper) requireFlag(cmd *cobra.Command, name, example string) error {
	f := cmd.Flags().Lookup(name)
	if f == nil || f.Value.String() == "" {
		if cmd.Flags().Changed(name) {
			return fmt.Errorf("flag --%s nao pode ficar vazia.\nExemplo:\n  %s", name, example)
		}
		return fmt.Errorf("flag --%s e obrigatoria.\nExemplo:\n  %s", name, example)
	}
	return nil
}

func (h *flagHelper) parseToolsFlag(raw string) ([]skills.Tool, error) {
	// Vazio => auto-detect (ADR-019): retorna nil sem erro para que Execute acione
	// AgentDetector. Presenca da flag e override explicito.
	if raw == "" {
		return nil, nil
	}
	if raw == "all" {
		return skills.AllTools, nil
	}
	var tools []skills.Tool
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		t, ok := skills.NewCatalog().ParseTool(s)
		if !ok {
			return nil, fmt.Errorf("ferramenta invalida: %s (opcoes: claude, gemini, codex, copilot, all)", s)
		}
		tools = append(tools, t)
	}
	return tools, nil
}

// parseFocusPaths converte a flag --focus-paths (comma-separated) ou a env var
// FOCUS_PATHS (newline ou comma-separated) em uma slice de caminhos.
func (h *flagHelper) parseFocusPaths(raw string) []string {
	if raw == "" {
		raw = os.Getenv("FOCUS_PATHS")
	}
	if raw == "" {
		return nil
	}
	var paths []string
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

func (h *flagHelper) parseLangsFlag(raw string) ([]skills.Lang, error) {
	if raw == "" || raw == "none" {
		return nil, nil
	}
	if raw == "all" {
		return skills.AllLangs, nil
	}
	var langs []skills.Lang
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		l, ok := skills.NewCatalog().ParseLang(s)
		if !ok {
			return nil, fmt.Errorf("linguagem invalida: %s (opcoes: go, node, python, all)", s)
		}
		langs = append(langs, l)
	}
	return langs, nil
}
