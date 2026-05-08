package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Runtime agrupa configuracao de runtime consumida por skills, scripts e
// pelo orquestrador internal/taskloop. Carregada de .claude/config.yaml
// (fonte canonica) ou .agents/config.yaml (alias) na raiz do projeto.
//
// Defaults preservam o comportamento anterior a este arquivo (tasks/prd-<slug>),
// garantindo retrocompatibilidade com projetos que nao fornecam config.yaml.
type Runtime struct {
	TasksRoot         string  `yaml:"tasks_root"`
	PRDPrefix         string  `yaml:"prd_prefix"`
	EvidenceDir       string  `yaml:"evidence_dir"`
	CoverageThreshold float64 `yaml:"coverage_threshold"`
	LanguageDefault   string  `yaml:"language_default"`
}

// DefaultRuntime retorna a configuracao com defaults compativeis com o layout atual.
func DefaultRuntime() Runtime {
	return Runtime{
		TasksRoot:         "tasks",
		PRDPrefix:         "prd-",
		EvidenceDir:       "",
		CoverageThreshold: 70.0,
		LanguageDefault:   "",
	}
}

// runtimeCandidates lista os caminhos consultados, em ordem.
// O primeiro existente vence. Caminhos sao relativos a repoRoot.
func runtimeCandidates() []string {
	return []string{
		filepath.Join(".claude", "config.yaml"),
		filepath.Join(".agents", "config.yaml"),
	}
}

// LoadRuntime resolve config.yaml dentro de repoRoot e retorna a configuracao
// resultante. Quando nenhum arquivo existir, retorna DefaultRuntime sem erro.
// Quando o arquivo existir mas estiver malformado, propaga erro descritivo.
func LoadRuntime(repoRoot string) (Runtime, error) {
	cfg := DefaultRuntime()
	if repoRoot == "" {
		return cfg, nil
	}

	for _, rel := range runtimeCandidates() {
		abs := filepath.Join(repoRoot, rel)
		data, err := os.ReadFile(abs)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return cfg, fmt.Errorf("ler %s: %w", rel, err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", rel, err)
		}
		applyRuntimeDefaults(&cfg)
		return cfg, nil
	}

	return cfg, nil
}

// applyRuntimeDefaults preenche campos vazios apos parse parcial, preservando
// retrocompatibilidade com arquivos que so declaram subconjunto das chaves.
func applyRuntimeDefaults(cfg *Runtime) {
	d := DefaultRuntime()
	if cfg.TasksRoot == "" {
		cfg.TasksRoot = d.TasksRoot
	}
	if cfg.PRDPrefix == "" {
		cfg.PRDPrefix = d.PRDPrefix
	}
	if cfg.CoverageThreshold == 0 {
		cfg.CoverageThreshold = d.CoverageThreshold
	}
}

// EnvVars projeta a configuracao em variaveis de ambiente exportadas pelo
// script scripts/lib/check-invocation-depth.sh para consumo de skills e validators.
// O caller decide se aplica via os.Setenv ou se gera linhas `export FOO=bar`.
func (r Runtime) EnvVars() map[string]string {
	return map[string]string{
		"AI_TASKS_ROOT":         r.TasksRoot,
		"AI_PRD_PREFIX":         r.PRDPrefix,
		"AI_EVIDENCE_DIR":       r.EvidenceDir,
		"AI_COVERAGE_THRESHOLD": fmt.Sprintf("%g", r.CoverageThreshold),
		"AI_LANGUAGE_DEFAULT":   r.LanguageDefault,
	}
}
