// Package events — normalize.go implementa a normalização de tool-calls driver-aware.
// Replica o padrão de compozy/internal/core/agent/tool_call_input.go::buildNormalizedToolUseBlock.
// Regras de alias carregadas de .agents/normalization-rules.yaml via go:embed;
// override por projeto: workspace .agents/normalization-rules.yaml vence sobre embedded.
//
// RF-02 — task-2.0 (PRD: Claude-CLI 2026, paridade Compozy acima da camada ACP).
package events

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

//go:embed normalization-rules.yaml
var defaultRulesYAML []byte

// ErrUnknownRulesVersion é retornado quando a versão do schema de regras é desconhecida.
var ErrUnknownRulesVersion = errors.New("versão desconhecida de normalization-rules.yaml")

// normalizationRules contém o schema de normalização de tool-calls por driver.
type normalizationRules struct {
	Version       int                                     `yaml:"version"`
	// CommonAliases contém aliases compartilhados por drivers que declaram inherit_common.
	// Espelha commonToolTitleAliases de Compozy (tool_call_name.go:84).
	CommonAliases map[string]string                       `yaml:"common_aliases"`
	// InheritCommon lista drivers que herdam CommonAliases sem overrides.
	InheritCommon []string                                `yaml:"inherit_common"`
	Aliases       map[string]map[string]string            `yaml:"aliases"`
	InputMappings map[string]map[string]map[string]string `yaml:"input_mappings"`
}

// NormalizedToolCall contém o resultado da normalização de uma tool-call.
// RawName e RawInput NUNCA são mutados — preservados lado a lado com os campos normalizados.
type NormalizedToolCall struct {
	// RawName é o nome original da ferramenta, preservado byte-identical.
	RawName string
	// NormalizedName é o nome canônico da ferramenta após lookup de alias.
	// Quando não há mapping, NormalizedName == RawName (passthrough).
	NormalizedName string
	// RawInput é o input JSON original, preservado byte-identical. Nunca mutado.
	RawInput json.RawMessage
	// NormalizedInput é uma cópia de RawInput com chaves renomeadas conforme input_mappings.
	// Quando não há mapping, NormalizedInput é uma cópia de RawInput.
	NormalizedInput json.RawMessage
}


// loadRules carrega as regras de normalização.
// Workspace override: procura .agents/normalization-rules.yaml no diretório atual
// e em workDir (quando fornecido). Vence sobre o embedded default.
// Schema fechado: version != 1 retorna ErrUnknownRulesVersion.
func loadRules(workDir string) (*normalizationRules, error) {
	data := defaultRulesYAML

	// Verificar override por projeto (workspace .agents/normalization-rules.yaml)
	candidateDirs := []string{workDir, "."}
	for _, dir := range candidateDirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, ".agents", "normalization-rules.yaml")
		content, err := os.ReadFile(candidate)
		if err == nil {
			data = content
			break
		}
	}

	var rules normalizationRules
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(false) // tolerante a campos extras
	if err := dec.Decode(&rules); err != nil {
		return nil, fmt.Errorf("decodificar normalization-rules.yaml: %w", err)
	}

	if rules.Version != 1 {
		return nil, fmt.Errorf("%w: recebido %d, esperado 1", ErrUnknownRulesVersion, rules.Version)
	}

	resolveInherit(&rules)

	return &rules, nil
}

// resolveInherit expande drivers declarados em inherit_common para o mapa aliases.
// Para cada driverID em InheritCommon, se não houver entrada em Aliases, injeta
// uma cópia de CommonAliases (sem mutação). Drivers com entrada explícita não são sobrescritos.
func resolveInherit(rules *normalizationRules) {
	if len(rules.InheritCommon) == 0 || len(rules.CommonAliases) == 0 {
		return
	}
	if rules.Aliases == nil {
		rules.Aliases = make(map[string]map[string]string)
	}
	for _, driverID := range rules.InheritCommon {
		if _, exists := rules.Aliases[driverID]; exists {
			// Entrada explícita prevalece — não sobrescrever.
			continue
		}
		// Copiar CommonAliases para o driver.
		inherited := make(map[string]string, len(rules.CommonAliases))
		maps.Copy(inherited, rules.CommonAliases)
		rules.Aliases[driverID] = inherited
	}
}

// BuildNormalizedToolCall normaliza uma tool-call dado o driverID, rawName e rawInput.
//
// Garantias:
//   - RawName e RawInput nunca são mutados.
//   - Driver desconhecido OU rawName não-mapeado → passthrough (NormalizedName = rawName); sem erro.
//   - NormalizedInput é uma cópia de RawInput com chaves renomeadas; valores preservados.
//
// workDir é usado para localizar o override de projeto (.agents/normalization-rules.yaml).
// Passe "" para usar apenas o diretório de trabalho atual.
func BuildNormalizedToolCall(driverID, rawName string, rawInput json.RawMessage, workDir string) (NormalizedToolCall, error) {
	rules, err := loadRules(workDir)
	if err != nil {
		return NormalizedToolCall{}, fmt.Errorf("carregar regras de normalização: %w", err)
	}

	return buildNormalized(rules, driverID, rawName, rawInput)
}

// BuildNormalizedToolCallByDriver normaliza uma tool-call usando specs.DriverID como fonte de verdade.
// Propaga o DriverID VO (ADR-020, Tarefa 3.0); resolve na fronteira — driver zero-value (inválido)
// propaga specs.ErrUnknownDriver imediatamente, antes de carregar as regras.
//
// Garantias idênticas a BuildNormalizedToolCall:
//   - RawName e RawInput nunca são mutados.
//   - rawName não-mapeado → passthrough (NormalizedName = rawName); sem erro.
//   - NormalizedInput é uma cópia de RawInput com chaves renomeadas; valores preservados.
//
// workDir é usado para localizar o override de projeto (.agents/normalization-rules.yaml).
func BuildNormalizedToolCallByDriver(driver specs.DriverID, rawName string, rawInput json.RawMessage, workDir string) (NormalizedToolCall, error) {
	if driver.String() == "" {
		return NormalizedToolCall{}, fmt.Errorf("%w: driver zero-value não permitido", specs.ErrUnknownDriver)
	}
	rules, err := loadRules(workDir)
	if err != nil {
		return NormalizedToolCall{}, fmt.Errorf("carregar regras de normalização: %w", err)
	}
	return buildNormalized(rules, driver.String(), rawName, rawInput)
}

// buildNormalized executa a normalização usando regras já carregadas.
func buildNormalized(rules *normalizationRules, driverID, rawName string, rawInput json.RawMessage) (NormalizedToolCall, error) {
	// 1. Lookup de alias: aliases[driverID][rawName] → normalizedName.
	//    Miss = passthrough.
	normalizedName := rawName
	if aliases, ok := rules.Aliases[driverID]; ok {
		if mapped, ok := aliases[rawName]; ok && mapped != "" {
			normalizedName = mapped
		}
	}

	// 2. Normalizar input: renomear chaves conforme input_mappings[driverID][rawName].
	//    Preservar valores; não decodificar quando não há mapping.
	normalizedInput, err := normalizeInput(rules, driverID, rawName, rawInput)
	if err != nil {
		return NormalizedToolCall{}, fmt.Errorf("normalizar input: %w", err)
	}

	return NormalizedToolCall{
		RawName:         rawName,
		NormalizedName:  normalizedName,
		RawInput:        rawInput,
		NormalizedInput: normalizedInput,
	}, nil
}

// normalizeInput aplica renomeação de chaves conforme input_mappings.
// Quando não há mapping aplicável, retorna uma cópia de rawInput.
// Nunca muta rawInput.
func normalizeInput(rules *normalizationRules, driverID, rawName string, rawInput json.RawMessage) (json.RawMessage, error) {
	if len(rawInput) == 0 {
		return json.RawMessage{}, nil
	}

	// Verificar se há mapping para este driver e rawName.
	toolMappings := map[string]string{}
	if driverMappings, ok := rules.InputMappings[driverID]; ok {
		if tm, ok := driverMappings[rawName]; ok {
			toolMappings = tm
		}
	}

	// Sem mapping: retornar cópia de rawInput (nunca o mesmo slice).
	if len(toolMappings) == 0 {
		copied := make(json.RawMessage, len(rawInput))
		copy(copied, rawInput)
		return copied, nil
	}

	// Decodificar, renomear chaves e re-encodar.
	var inputMap map[string]json.RawMessage
	if err := json.Unmarshal(rawInput, &inputMap); err != nil {
		// Input não é objeto JSON: retornar cópia sem modificação.
		copied := make(json.RawMessage, len(rawInput))
		copy(copied, rawInput)
		return copied, nil
	}

	normalized := make(map[string]json.RawMessage, len(inputMap))
	for k, v := range inputMap {
		if newKey, ok := toolMappings[k]; ok && newKey != "" {
			normalized[newKey] = v
		} else {
			normalized[k] = v
		}
	}

	encoded, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("re-encodar input normalizado: %w", err)
	}

	return encoded, nil
}
