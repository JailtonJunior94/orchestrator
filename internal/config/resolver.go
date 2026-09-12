package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var projectCandidateNames = []string{
	filepath.Join(".aispec", "config.yaml"),
	filepath.Join(".claude", "config.yaml"),
	filepath.Join(".agents", "config.yaml"),
}

var projectMarkers = []string{".git", ".aispec", ".claude", ".agents"}

type Resolver interface {
	Resolve(cwd string, overrides Runtime) (Runtime, error)
}

type DefaultResolver struct {
	HomeDir string

	readFile func(path string) ([]byte, error)

	isDir func(path string) bool
}

var _ Resolver = (*DefaultResolver)(nil)

func NewDefaultResolver() *DefaultResolver {
	homeDir, _ := os.UserHomeDir()
	return &DefaultResolver{
		HomeDir:  homeDir,
		readFile: os.ReadFile,
		isDir: func(path string) bool {
			info, err := os.Stat(path)
			return err == nil && info.IsDir()
		},
	}
}

func (r *DefaultResolver) Resolve(cwd string, overrides Runtime) (Runtime, error) {
	result := NewRuntimeProvider().DefaultRuntime()

	if r.HomeDir != "" {
		globalPath := filepath.Join(r.HomeDir, ".aispec", "config.yaml")
		globalCfg, err := r.loadFile(globalPath)
		if err != nil {
			return result, err
		}
		if globalCfg != nil {
			r.mergeInto(&result, *globalCfg)
		}
	}

	if cwd != "" {
		projPath, err := r.findProjectConfig(cwd)
		if err != nil {
			return result, err
		}
		if projPath != "" {
			projCfg, err := r.loadFile(projPath)
			if err != nil {
				return result, err
			}
			if projCfg != nil {
				r.mergeInto(&result, *projCfg)
			}
		}
	}

	r.mergeInto(&result, overrides)

	return result, nil
}

func (r *DefaultResolver) loadFile(path string) (*Runtime, error) {
	data, err := r.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		rel := path
		return nil, fmt.Errorf("ler %s: %w", rel, err)
	}
	var cfg Runtime
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

func (r *DefaultResolver) findProjectConfig(cwd string) (string, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("resolver cwd %s: %w", cwd, err)
	}

	current := abs
	for {

		for _, name := range projectCandidateNames {
			candidate := filepath.Join(current, name)
			data, err := r.readFile(candidate)
			if err == nil {
				_ = data
				return candidate, nil
			}
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("ler %s: %w", candidate, err)
			}
		}

		for _, marker := range projectMarkers {
			markerPath := filepath.Join(current, marker)
			if r.isDir(markerPath) {
				return "", nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {

			break
		}
		current = parent
	}

	return "", nil
}

func (r *DefaultResolver) mergeInto(dst *Runtime, src Runtime) {
	if src.TasksRoot != "" {
		dst.TasksRoot = src.TasksRoot
	}
	if src.PRDPrefix != "" {
		dst.PRDPrefix = src.PRDPrefix
	}
	if src.EvidenceDir != "" {
		dst.EvidenceDir = src.EvidenceDir
	}
	if src.CoverageThreshold != 0 {
		dst.CoverageThreshold = src.CoverageThreshold
	}
	if src.LanguageDefault != "" {
		dst.LanguageDefault = src.LanguageDefault
	}
	if src.Timeout != "" {
		dst.Timeout = src.Timeout
	}
	if src.MaxRetries != 0 {
		dst.MaxRetries = src.MaxRetries
	}
	if src.RetryBackoffMultiplier != 0 {
		dst.RetryBackoffMultiplier = src.RetryBackoffMultiplier
	}
	if src.Concurrent != 0 {
		dst.Concurrent = src.Concurrent
	}
	if src.BatchSize != 0 {
		dst.BatchSize = src.BatchSize
	}
	if src.DefaultTool != "" {
		dst.DefaultTool = src.DefaultTool
	}
	if src.MaxBugfixIterations != 0 {
		dst.MaxBugfixIterations = src.MaxBugfixIterations
	}
	if src.HandoffLeaseTTL != "" {
		dst.HandoffLeaseTTL = src.HandoffLeaseTTL
	}
	if src.DurableMemoryEnabledSet {
		dst.DurableMemoryEnabled = src.DurableMemoryEnabled
	}
}
