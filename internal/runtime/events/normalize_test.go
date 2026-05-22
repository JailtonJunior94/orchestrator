package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// T-NORM-01: Claude bash → NormalizedName="bash"; RawName="bash" (alias para si mesmo)
func TestNormalizeClaudeBash(t *testing.T) {
	input := json.RawMessage(`{"command":"echo hello"}`)
	result, err := BuildNormalizedToolCall("claude", "bash", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}
	if result.RawName != "bash" {
		t.Errorf("RawName: queria %q, obtive %q", "bash", result.RawName)
	}
	if result.NormalizedName != "bash" {
		t.Errorf("NormalizedName: queria %q, obtive %q", "bash", result.NormalizedName)
	}
}

// T-NORM-02: Codex shell → NormalizedName="bash"; RawName="shell"
func TestNormalizeCodexShellToBash(t *testing.T) {
	input := json.RawMessage(`{"cmd":"echo hello"}`)
	result, err := BuildNormalizedToolCall("codex", "shell", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}
	if result.RawName != "shell" {
		t.Errorf("RawName: queria %q, obtive %q", "shell", result.RawName)
	}
	if result.NormalizedName != "bash" {
		t.Errorf("NormalizedName: queria %q, obtive %q", "bash", result.NormalizedName)
	}
}

// T-NORM-03: Codex search_query → NormalizedName="web_search"
func TestNormalizeCodexSearchQueryToWebSearch(t *testing.T) {
	input := json.RawMessage(`{"q":"golang testing"}`)
	result, err := BuildNormalizedToolCall("codex", "search_query", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}
	if result.NormalizedName != "web_search" {
		t.Errorf("NormalizedName: queria %q, obtive %q", "web_search", result.NormalizedName)
	}
}

// T-NORM-04: RawInput deve ser byte-identical após normalização (nunca mutado)
func TestNormalizationPreservesRawInput(t *testing.T) {
	rawJSON := `{"command":"ls -la","cwd":"/tmp"}`
	input := json.RawMessage(rawJSON)
	// capturar bytes originais antes
	original := make(json.RawMessage, len(input))
	copy(original, input)

	result, err := BuildNormalizedToolCall("claude", "bash", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}

	// RawInput deve ser byte-identical ao input original
	if string(result.RawInput) != rawJSON {
		t.Errorf("RawInput mutado: queria %q, obtive %q", rawJSON, string(result.RawInput))
	}
	// O slice original não deve ter sido alterado
	if string(original) != string(input) {
		t.Error("slice de input original foi mutado durante normalização")
	}
}

// T-NORM-05: cursor read_file com startLineNumberBaseOne → NormalizedInput tem start_line
func TestNormalizationCanonicalizesFields(t *testing.T) {
	input := json.RawMessage(`{"startLineNumberBaseOne":10,"endLineNumberBaseOne":20,"filePath":"main.go"}`)
	result, err := BuildNormalizedToolCall("cursor", "read_file", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}

	var normalized map[string]json.RawMessage
	if err := json.Unmarshal(result.NormalizedInput, &normalized); err != nil {
		t.Fatalf("NormalizedInput não é JSON válido: %v", err)
	}

	if _, ok := normalized["start_line"]; !ok {
		t.Error("NormalizedInput deveria ter chave 'start_line'")
	}
	if _, ok := normalized["end_line"]; !ok {
		t.Error("NormalizedInput deveria ter chave 'end_line'")
	}
	// chave original não deve aparecer
	if _, ok := normalized["startLineNumberBaseOne"]; ok {
		t.Error("NormalizedInput não deveria ter chave original 'startLineNumberBaseOne'")
	}
	// filePath não mapeado deve ser preservado
	if _, ok := normalized["filePath"]; !ok {
		t.Error("NormalizedInput deveria preservar chave não mapeada 'filePath'")
	}
}

// T-NORM-06: driver desconhecido → passthrough (NormalizedName=rawName); sem erro
func TestUnknownDriverFallsBack(t *testing.T) {
	input := json.RawMessage(`{"arg":"value"}`)
	result, err := BuildNormalizedToolCall("foo", "bar", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}
	if result.NormalizedName != "bar" {
		t.Errorf("passthrough: queria NormalizedName=%q, obtive %q", "bar", result.NormalizedName)
	}
	if result.RawName != "bar" {
		t.Errorf("RawName: queria %q, obtive %q", "bar", result.RawName)
	}
}

// T-NORM-07: workspace override vence sobre embedded
func TestProjectOverrideWinsOverEmbedded(t *testing.T) {
	// Criar workspace temporário com .agents/normalization-rules.yaml custom
	workDir := t.TempDir()
	agentsDir := filepath.Join(workDir, ".agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("criar .agents dir: %v", err)
	}

	// YAML customizado: driver "custom_driver" com alias "custom_tool" → "canonical_tool"
	customYAML := `version: 1
aliases:
  custom_driver:
    custom_tool: canonical_tool
input_mappings: {}
`
	customPath := filepath.Join(agentsDir, "normalization-rules.yaml")
	if err := os.WriteFile(customPath, []byte(customYAML), 0o644); err != nil {
		t.Fatalf("escrever custom YAML: %v", err)
	}

	input := json.RawMessage(`{}`)
	result, err := BuildNormalizedToolCall("custom_driver", "custom_tool", input, workDir)
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}

	// As regras locais devem vencer; alias customizado deve ser aplicado
	if result.NormalizedName != "canonical_tool" {
		t.Errorf("override não aplicado: queria NormalizedName=%q, obtive %q",
			"canonical_tool", result.NormalizedName)
	}
}

// Testes adicionais de cobertura

// TestBuildNormalizedToolCall_EmptyInput — input vazio retorna sem erro
func TestBuildNormalizedToolCall_EmptyInput(t *testing.T) {
	result, err := BuildNormalizedToolCall("claude", "bash", json.RawMessage{}, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}
	if result.RawName != "bash" {
		t.Errorf("RawName: queria %q, obtive %q", "bash", result.RawName)
	}
}

// TestBuildNormalizedToolCall_NonObjectInput — input não-objeto JSON retorna cópia sem erro
func TestBuildNormalizedToolCall_NonObjectInput(t *testing.T) {
	input := json.RawMessage(`"just a string"`)
	result, err := BuildNormalizedToolCall("codex", "shell", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}
	// NormalizedInput deve ser cópia do RawInput quando não é objeto
	if string(result.NormalizedInput) != string(input) {
		t.Errorf("NormalizedInput: queria %q, obtive %q", string(input), string(result.NormalizedInput))
	}
}

// TestLoadRules_InvalidVersion — versão desconhecida retorna erro
func TestLoadRules_InvalidVersion(t *testing.T) {
	workDir := t.TempDir()
	agentsDir := filepath.Join(workDir, ".agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("criar .agents dir: %v", err)
	}

	badYAML := `version: 99
aliases: {}
input_mappings: {}
`
	customPath := filepath.Join(agentsDir, "normalization-rules.yaml")
	if err := os.WriteFile(customPath, []byte(badYAML), 0o644); err != nil {
		t.Fatalf("escrever custom YAML: %v", err)
	}

	_, err := BuildNormalizedToolCall("claude", "bash", json.RawMessage(`{}`), workDir)
	if err == nil {
		t.Fatal("esperava erro para versão desconhecida, obtive nil")
	}
}

// TestNormalizationPreservesValuesOnRename — valor preservado quando chave renomeada
func TestNormalizationPreservesValuesOnRename(t *testing.T) {
	// Codex shell: cmd → command; valor deve ser preservado
	input := json.RawMessage(`{"cmd":"ls /tmp"}`)
	result, err := BuildNormalizedToolCall("codex", "shell", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}

	var normalized map[string]json.RawMessage
	if err := json.Unmarshal(result.NormalizedInput, &normalized); err != nil {
		t.Fatalf("NormalizedInput não é JSON válido: %v", err)
	}

	if v, ok := normalized["command"]; !ok {
		t.Error("NormalizedInput deveria ter chave 'command' (renomeada de 'cmd')")
	} else if string(v) != `"ls /tmp"` {
		t.Errorf("valor preservado: queria %q, obtive %q", `"ls /tmp"`, string(v))
	}
}

// TestCopilotRunToBash — copilot run → bash
func TestCopilotRunToBash(t *testing.T) {
	input := json.RawMessage(`{"script":"./run.sh"}`)
	result, err := BuildNormalizedToolCall("copilot", "run", input, "")
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}
	if result.NormalizedName != "bash" {
		t.Errorf("NormalizedName: queria %q, obtive %q", "bash", result.NormalizedName)
	}
}

// T-32: TestNormalizeToolCallGeminiInheritsCommon — Gemini emite read_file → normalizado "read"; raw_name preservado.
// Valida resolveInherit: gemini está em inherit_common; sem entrada explícita em aliases.
func TestNormalizeToolCallGeminiInheritsCommon(t *testing.T) {
	input := json.RawMessage(`{"path":"/tmp/README.md"}`)
	result, err := BuildNormalizedToolCall("gemini", "read_file", input, "")
	if err != nil {
		t.Fatalf("T-32: BuildNormalizedToolCall inesperado: %v", err)
	}
	if result.NormalizedName != "read" {
		t.Errorf("T-32: NormalizedName: queria %q (via inherit:common), obtive %q", "read", result.NormalizedName)
	}
	if result.RawName != "read_file" {
		t.Errorf("T-32: RawName: queria %q, obtive %q", "read_file", result.RawName)
	}
}

// T-32b: TestNormalizeGeminiPreservesRawName — raw_name preservado lado a lado com normalized_name.
// Cobre bash (alias para si mesmo) e str_replace_editor → edit.
func TestNormalizeGeminiPreservesRawName(t *testing.T) {
	tests := []struct {
		rawName        string
		expectedNorm   string
	}{
		{"bash", "bash"},
		{"write_file", "write"},
		{"str_replace_editor", "edit"},
		{"unknown_tool", "unknown_tool"}, // passthrough: sem alias para ferramenta desconhecida
	}

	for _, tc := range tests {
		t.Run(tc.rawName, func(t *testing.T) {
			input := json.RawMessage(`{}`)
			result, err := BuildNormalizedToolCall("gemini", tc.rawName, input, "")
			if err != nil {
				t.Fatalf("T-32b: BuildNormalizedToolCall inesperado para %q: %v", tc.rawName, err)
			}
			// RawName nunca mutado.
			if result.RawName != tc.rawName {
				t.Errorf("T-32b: RawName mutado: queria %q, obtive %q", tc.rawName, result.RawName)
			}
			// NormalizedName correto via inherit:common.
			if result.NormalizedName != tc.expectedNorm {
				t.Errorf("T-32b: NormalizedName(%q): queria %q, obtive %q",
					tc.rawName, tc.expectedNorm, result.NormalizedName)
			}
		})
	}
}

// TestResolveInheritDoesNotOverrideExplicit — driver com entrada explícita em aliases
// não é sobrescrito por resolveInherit mesmo que apareça em inherit_common.
func TestResolveInheritDoesNotOverrideExplicit(t *testing.T) {
	workDir := t.TempDir()
	agentsDir := filepath.Join(workDir, ".agents")
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatalf("criar .agents dir: %v", err)
	}

	// YAML com gemini em inherit_common E em aliases explícito — aliases explícito deve prevalecer.
	customYAML := `version: 1
common_aliases:
  read_file: read
inherit_common:
  - gemini
aliases:
  gemini:
    read_file: raw_passthrough
input_mappings: {}
`
	customPath := filepath.Join(agentsDir, "normalization-rules.yaml")
	if err := os.WriteFile(customPath, []byte(customYAML), 0o644); err != nil {
		t.Fatalf("escrever custom YAML: %v", err)
	}

	input := json.RawMessage(`{}`)
	result, err := BuildNormalizedToolCall("gemini", "read_file", input, workDir)
	if err != nil {
		t.Fatalf("BuildNormalizedToolCall inesperado: %v", err)
	}

	// Entrada explícita em aliases deve prevalecer sobre inherit_common.
	if result.NormalizedName != "raw_passthrough" {
		t.Errorf("explícito não prevaleceu: queria %q, obtive %q", "raw_passthrough", result.NormalizedName)
	}
}
