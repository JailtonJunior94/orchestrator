package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type Runtime struct {
	TasksRoot               string  `yaml:"tasks_root"`
	PRDPrefix               string  `yaml:"prd_prefix"`
	EvidenceDir             string  `yaml:"evidence_dir"`
	CoverageThreshold       float64 `yaml:"coverage_threshold"`
	LanguageDefault         string  `yaml:"language_default"`
	Timeout                 string  `yaml:"timeout"`
	MaxRetries              int     `yaml:"max_retries"`
	RetryBackoffMultiplier  float64 `yaml:"retry_backoff_multiplier"`
	Concurrent              int     `yaml:"concurrent"`
	BatchSize               int     `yaml:"batch_size"`
	DefaultTool             string  `yaml:"default_tool"`
	MaxBugfixIterations     int     `yaml:"max_bugfix_iterations"`
	HandoffLeaseTTL         string  `yaml:"handoff_lease_ttl"`
	DurableMemoryEnabled    bool    `yaml:"durable_memory_enabled"`
	DurableMemoryEnabledSet bool    `yaml:"-"`
}

func (r *Runtime) UnmarshalYAML(value *yaml.Node) error {
	type alias Runtime
	var decoded alias
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*r = Runtime(decoded)

	var probe struct {
		DurableMemoryEnabled *bool `yaml:"durable_memory_enabled"`
	}
	if err := value.Decode(&probe); err != nil {
		return err
	}
	if probe.DurableMemoryEnabled != nil {
		r.DurableMemoryEnabled = *probe.DurableMemoryEnabled
		r.DurableMemoryEnabledSet = true
	}
	return nil
}

type RuntimeProvider struct{}

func NewRuntimeProvider() *RuntimeProvider {
	return &RuntimeProvider{}
}

func (p *RuntimeProvider) DefaultRuntime() Runtime {
	return Runtime{
		TasksRoot:         ".specs",
		PRDPrefix:         "prd-",
		EvidenceDir:       "",
		CoverageThreshold: 70.0,
		LanguageDefault:   "",
	}
}

func (p *RuntimeProvider) LoadRuntime(repoRoot string) (Runtime, error) {
	r := NewDefaultResolver()
	r.HomeDir = ""
	return r.Resolve(repoRoot, Runtime{})
}

func (r Runtime) EnvVars() map[string]string {
	return map[string]string{
		"AI_TASKS_ROOT":         r.TasksRoot,
		"AI_PRD_PREFIX":         r.PRDPrefix,
		"AI_EVIDENCE_DIR":       r.EvidenceDir,
		"AI_COVERAGE_THRESHOLD": fmt.Sprintf("%g", r.CoverageThreshold),
		"AI_LANGUAGE_DEFAULT":   r.LanguageDefault,
	}
}
