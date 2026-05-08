package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultRuntimeBackwardsCompatible(t *testing.T) {
	cfg := DefaultRuntime()
	if cfg.TasksRoot != "tasks" {
		t.Fatalf("TasksRoot default invalido: %q", cfg.TasksRoot)
	}
	if cfg.PRDPrefix != "prd-" {
		t.Fatalf("PRDPrefix default invalido: %q", cfg.PRDPrefix)
	}
	if cfg.CoverageThreshold != 70.0 {
		t.Fatalf("CoverageThreshold default invalido: %v", cfg.CoverageThreshold)
	}
}

func TestLoadRuntimeReturnsDefaultsWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadRuntime(dir)
	if err != nil {
		t.Fatalf("LoadRuntime sem arquivo retornou erro: %v", err)
	}
	if cfg != DefaultRuntime() {
		t.Fatalf("LoadRuntime nao retornou defaults: %+v", cfg)
	}
}

func TestLoadRuntimePrefersClaudeOverAgents(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".claude", "config.yaml"), "tasks_root: claude-tasks\nprd_prefix: c-\n")
	mustWrite(t, filepath.Join(dir, ".agents", "config.yaml"), "tasks_root: agents-tasks\nprd_prefix: a-\n")

	cfg, err := LoadRuntime(dir)
	if err != nil {
		t.Fatalf("LoadRuntime: %v", err)
	}
	if cfg.TasksRoot != "claude-tasks" {
		t.Fatalf(".claude/config.yaml deveria prevalecer, got %q", cfg.TasksRoot)
	}
	if cfg.PRDPrefix != "c-" {
		t.Fatalf("PRDPrefix esperado 'c-', got %q", cfg.PRDPrefix)
	}
}

func TestLoadRuntimeFallsBackToAgents(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".agents", "config.yaml"), "tasks_root: agents-tasks\n")

	cfg, err := LoadRuntime(dir)
	if err != nil {
		t.Fatalf("LoadRuntime: %v", err)
	}
	if cfg.TasksRoot != "agents-tasks" {
		t.Fatalf("TasksRoot esperado 'agents-tasks', got %q", cfg.TasksRoot)
	}
}

func TestLoadRuntimeFillsPartialFile(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".claude", "config.yaml"), "evidence_dir: custom/evidence\n")

	cfg, err := LoadRuntime(dir)
	if err != nil {
		t.Fatalf("LoadRuntime: %v", err)
	}
	if cfg.EvidenceDir != "custom/evidence" {
		t.Fatalf("EvidenceDir esperado 'custom/evidence', got %q", cfg.EvidenceDir)
	}
	if cfg.TasksRoot != "tasks" {
		t.Fatalf("TasksRoot deveria fallback para default, got %q", cfg.TasksRoot)
	}
	if cfg.PRDPrefix != "prd-" {
		t.Fatalf("PRDPrefix deveria fallback para default, got %q", cfg.PRDPrefix)
	}
	if cfg.CoverageThreshold != 70.0 {
		t.Fatalf("CoverageThreshold deveria fallback, got %v", cfg.CoverageThreshold)
	}
}

func TestLoadRuntimeMalformedYAMLPropagatesError(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".claude", "config.yaml"), "tasks_root: [unclosed\n")

	if _, err := LoadRuntime(dir); err == nil {
		t.Fatal("LoadRuntime deveria retornar erro para YAML malformado")
	}
}

func TestEnvVarsProjectsAllKeys(t *testing.T) {
	r := Runtime{
		TasksRoot:         "tk",
		PRDPrefix:         "p-",
		EvidenceDir:       "ev",
		CoverageThreshold: 75.5,
		LanguageDefault:   "go",
	}
	env := r.EnvVars()
	want := map[string]string{
		"AI_TASKS_ROOT":         "tk",
		"AI_PRD_PREFIX":         "p-",
		"AI_EVIDENCE_DIR":       "ev",
		"AI_COVERAGE_THRESHOLD": "75.5",
		"AI_LANGUAGE_DEFAULT":   "go",
	}
	for k, v := range want {
		if env[k] != v {
			t.Errorf("EnvVars[%q] = %q, want %q", k, env[k], v)
		}
	}
}

func TestLoadRuntimeEmptyRepoRoot(t *testing.T) {
	cfg, err := LoadRuntime("")
	if err != nil {
		t.Fatalf("LoadRuntime(\"\"): %v", err)
	}
	if cfg != DefaultRuntime() {
		t.Fatalf("repoRoot vazio deveria retornar defaults, got %+v", cfg)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
