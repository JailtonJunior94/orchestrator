package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resolverWithFS cria um DefaultResolver com readFile e isDir virtualizados
// via um mapa de arquivos em memoria, sem acessar o FS real.
func resolverWithFS(files map[string]string, dirs map[string]bool) *DefaultResolver {
	return &DefaultResolver{
		HomeDir: "",
		readFile: func(path string) ([]byte, error) {
			if data, ok := files[path]; ok {
				return []byte(data), nil
			}
			return nil, os.ErrNotExist
		},
		isDir: func(path string) bool {
			return dirs[path]
		},
	}
}

// TestResolverDefaultsOnlyWhenNoFiles valida que sem config global nem projeto
// o resultado e identico ao DefaultRuntime (regressao F1 / RF-16).
func TestResolverDefaultsOnlyWhenNoFiles(t *testing.T) {
	r := resolverWithFS(nil, nil)
	got, err := r.Resolve("/some/cwd", Runtime{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := DefaultRuntime()
	if got != want {
		t.Fatalf("esperado DefaultRuntime, got %+v", got)
	}
}

// TestResolverPrecedenceFullStack valida a cadeia completa:
// flags > projeto > global > built-in.
func TestResolverPrecedenceFullStack(t *testing.T) {
	cwd := "/project/sub/dir"
	projectConfigPath := "/project/.claude/config.yaml"
	globalConfigPath := "/home/user/.aispec/config.yaml"

	files := map[string]string{
		globalConfigPath: "tasks_root: global-tasks\nprd_prefix: g-\ncoverage_threshold: 60\n",
		projectConfigPath: "tasks_root: project-tasks\nevidence_dir: proj-ev\n",
	}
	dirs := map[string]bool{
		"/project/.git": true,
	}

	r := resolverWithFS(files, dirs)
	r.HomeDir = "/home/user"

	overrides := Runtime{TasksRoot: "flag-tasks"}
	got, err := r.Resolve(cwd, overrides)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	// flags > projeto > global > built-in
	if got.TasksRoot != "flag-tasks" {
		t.Errorf("TasksRoot: flag deveria vencer, got %q", got.TasksRoot)
	}
	// projeto sobrescreve global para evidence_dir
	if got.EvidenceDir != "proj-ev" {
		t.Errorf("EvidenceDir: projeto deveria vencer, got %q", got.EvidenceDir)
	}
	// global define prd_prefix (projeto nao declara)
	if got.PRDPrefix != "g-" {
		t.Errorf("PRDPrefix: global deveria prevalecer, got %q", got.PRDPrefix)
	}
	// global define coverage_threshold
	if got.CoverageThreshold != 60 {
		t.Errorf("CoverageThreshold: global deveria prevalecer, got %v", got.CoverageThreshold)
	}
}

// TestResolverUpwardWalkFindsProjectConfigFromDeepDir valida que o upward-walk
// encontra a config de projeto a partir de um subdiretorio profundo.
func TestResolverUpwardWalkFindsProjectConfigFromDeepDir(t *testing.T) {
	cwd := "/repo/a/b/c/d"
	projectConfigPath := "/repo/.claude/config.yaml"

	files := map[string]string{
		projectConfigPath: "tasks_root: walked-tasks\n",
	}
	dirs := map[string]bool{
		"/repo/.git": true,
	}

	r := resolverWithFS(files, dirs)
	got, err := r.Resolve(cwd, Runtime{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TasksRoot != "walked-tasks" {
		t.Errorf("upward-walk: esperado 'walked-tasks', got %q", got.TasksRoot)
	}
}

// TestResolverGlobalAbsentIsNonFatal valida que a ausencia de config global
// nao produz erro (degrada para projeto+defaults).
func TestResolverGlobalAbsentIsNonFatal(t *testing.T) {
	cwd := "/project"
	projectConfigPath := "/project/.claude/config.yaml"

	files := map[string]string{
		projectConfigPath: "tasks_root: proj-only\n",
	}

	r := resolverWithFS(files, nil)
	r.HomeDir = "/home/user" // diretorio home definido, mas sem arquivo global

	got, err := r.Resolve(cwd, Runtime{})
	if err != nil {
		t.Fatalf("ausencia de global nao deveria ser erro: %v", err)
	}
	if got.TasksRoot != "proj-only" {
		t.Errorf("esperado 'proj-only', got %q", got.TasksRoot)
	}
	// defaults para campos nao declarados
	if got.PRDPrefix != "prd-" {
		t.Errorf("PRDPrefix deveria ser default, got %q", got.PRDPrefix)
	}
}

// TestResolverMalformedGlobalPropagatesError valida que config global malformada
// propaga erro descritivo.
func TestResolverMalformedGlobalPropagatesError(t *testing.T) {
	r := resolverWithFS(map[string]string{
		"/home/user/.aispec/config.yaml": "tasks_root: [unclosed\n",
	}, nil)
	r.HomeDir = "/home/user"

	_, err := r.Resolve("", Runtime{})
	if err == nil {
		t.Fatal("config global malformada deveria retornar erro")
	}
}

// TestResolverMalformedProjectPropagatesError valida que config de projeto malformada
// propaga erro descritivo.
func TestResolverMalformedProjectPropagatesError(t *testing.T) {
	cwd := "/project"
	files := map[string]string{
		"/project/.claude/config.yaml": "tasks_root: [unclosed\n",
	}

	r := resolverWithFS(files, nil)
	_, err := r.Resolve(cwd, Runtime{})
	if err == nil {
		t.Fatal("config de projeto malformada deveria retornar erro")
	}
}

// TestResolverUpwardWalkStopsAtGitMarker valida que o walk para ao encontrar .git.
func TestResolverUpwardWalkStopsAtGitMarker(t *testing.T) {
	// Estrutura: /root/.git existe; /root/a/b e o cwd.
	// Nenhum config de projeto em nenhum nivel.
	// Resultado deve ser DefaultRuntime.
	dirs := map[string]bool{
		"/root/.git": true,
	}

	r := resolverWithFS(nil, dirs)
	got, err := r.Resolve("/root/a/b", Runtime{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != DefaultRuntime() {
		t.Fatalf("esperado DefaultRuntime sem config, got %+v", got)
	}
}

// TestResolverProjectCandidateOrderAispecBeforeClaude valida que .aispec/config.yaml
// tem precedencia sobre .claude/config.yaml quando ambos existem no mesmo diretorio.
func TestResolverProjectCandidateOrderAispecBeforeClaude(t *testing.T) {
	cwd := "/project"
	files := map[string]string{
		"/project/.aispec/config.yaml": "tasks_root: aispec-tasks\n",
		"/project/.claude/config.yaml": "tasks_root: claude-tasks\n",
	}

	r := resolverWithFS(files, nil)
	got, err := r.Resolve(cwd, Runtime{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.TasksRoot != "aispec-tasks" {
		t.Errorf(".aispec deveria ter precedencia, got %q", got.TasksRoot)
	}
}

// TestResolverNewOperationalFields valida que as novas chaves operacionais sao
// lidas e mescladas corretamente.
func TestResolverNewOperationalFields(t *testing.T) {
	cwd := "/project"
	files := map[string]string{
		"/project/.claude/config.yaml": strings.Join([]string{
			"timeout: 30s",
			"max_retries: 3",
			"retry_backoff_multiplier: 1.5",
			"concurrent: 4",
			"batch_size: 10",
			"default_tool: claude",
		}, "\n") + "\n",
	}

	r := resolverWithFS(files, nil)
	got, err := r.Resolve(cwd, Runtime{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if got.Timeout != "30s" {
		t.Errorf("Timeout: got %q, want %q", got.Timeout, "30s")
	}
	if got.MaxRetries != 3 {
		t.Errorf("MaxRetries: got %d, want 3", got.MaxRetries)
	}
	if got.RetryBackoffMultiplier != 1.5 {
		t.Errorf("RetryBackoffMultiplier: got %v, want 1.5", got.RetryBackoffMultiplier)
	}
	if got.Concurrent != 4 {
		t.Errorf("Concurrent: got %d, want 4", got.Concurrent)
	}
	if got.BatchSize != 10 {
		t.Errorf("BatchSize: got %d, want 10", got.BatchSize)
	}
	if got.DefaultTool != "claude" {
		t.Errorf("DefaultTool: got %q, want %q", got.DefaultTool, "claude")
	}
}

// TestResolverOperationalFieldsZeroValueIsF1 valida que campos operacionais com
// zero-value preservam o comportamento F1 (sem regressao).
func TestResolverOperationalFieldsZeroValueIsF1(t *testing.T) {
	r := resolverWithFS(nil, nil)
	got, err := r.Resolve("", Runtime{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Timeout != "" {
		t.Errorf("Timeout zero-value: got %q, want %q", got.Timeout, "")
	}
	if got.MaxRetries != 0 {
		t.Errorf("MaxRetries zero-value: got %d, want 0", got.MaxRetries)
	}
	if got.RetryBackoffMultiplier != 0 {
		t.Errorf("RetryBackoffMultiplier zero-value: got %v, want 0", got.RetryBackoffMultiplier)
	}
	if got.Concurrent != 0 {
		t.Errorf("Concurrent zero-value: got %d, want 0", got.Concurrent)
	}
	if got.BatchSize != 0 {
		t.Errorf("BatchSize zero-value: got %d, want 0", got.BatchSize)
	}
	if got.DefaultTool != "" {
		t.Errorf("DefaultTool zero-value: got %q, want %q", got.DefaultTool, "")
	}
}

// TestResolverHomeDirEmptySkipsGlobal valida que HomeDir vazio ignora config global
// sem tentar acessar nenhum path.
func TestResolverHomeDirEmptySkipsGlobal(t *testing.T) {
	// Nenhum arquivo disponivel; HomeDir vazio => sem global, sem erro.
	r := resolverWithFS(nil, nil)
	r.HomeDir = ""
	got, err := r.Resolve("", Runtime{})
	if err != nil {
		t.Fatalf("HomeDir vazio nao deveria causar erro: %v", err)
	}
	if got != DefaultRuntime() {
		t.Fatalf("esperado DefaultRuntime, got %+v", got)
	}
}

// TestResolverMergeFieldByField valida que o merge nao substitui campos ja definidos
// por fontes de maior precedencia quando a fonte de menor precedencia os declara.
func TestResolverMergeFieldByField(t *testing.T) {
	cwd := "/project"
	globalCfg := "/home/user/.aispec/config.yaml"
	projCfg := "/project/.claude/config.yaml"

	files := map[string]string{
		globalCfg: "tasks_root: global\nprd_prefix: g-\ncoverage_threshold: 60\n",
		projCfg:   "tasks_root: project\n",
	}

	r := resolverWithFS(files, nil)
	r.HomeDir = "/home/user"

	got, err := r.Resolve(cwd, Runtime{})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// projeto sobrescreve global para tasks_root
	if got.TasksRoot != "project" {
		t.Errorf("TasksRoot: projeto deveria vencer global, got %q", got.TasksRoot)
	}
	// global define prd_prefix (projeto nao declara => preservado)
	if got.PRDPrefix != "g-" {
		t.Errorf("PRDPrefix: global deveria ser preservado, got %q", got.PRDPrefix)
	}
	// global define coverage_threshold
	if got.CoverageThreshold != 60 {
		t.Errorf("CoverageThreshold: global deveria ser preservado, got %v", got.CoverageThreshold)
	}
}

// TestLoadRuntimeRegressionLegacy valida que LoadRuntime (wrapper sobre Resolver)
// produz saida byte-identica ao comportamento anterior quando nao ha config.
func TestLoadRuntimeRegressionLegacy(t *testing.T) {
	dir := t.TempDir()
	got, err := LoadRuntime(dir)
	if err != nil {
		t.Fatalf("LoadRuntime: %v", err)
	}
	want := DefaultRuntime()
	if got != want {
		t.Fatalf("regressao: LoadRuntime sem config deveria retornar DefaultRuntime, got %+v", got)
	}
}

// TestLoadRuntimeRegressionWithProjectConfig valida que LoadRuntime com config de projeto
// produz saida consistente com o comportamento legado.
func TestLoadRuntimeRegressionWithProjectConfig(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".claude", "config.yaml"), "tasks_root: my-tasks\nprd_prefix: my-\n")

	got, err := LoadRuntime(dir)
	if err != nil {
		t.Fatalf("LoadRuntime: %v", err)
	}
	if got.TasksRoot != "my-tasks" {
		t.Errorf("TasksRoot: got %q, want %q", got.TasksRoot, "my-tasks")
	}
	if got.PRDPrefix != "my-" {
		t.Errorf("PRDPrefix: got %q, want %q", got.PRDPrefix, "my-")
	}
	// defaults preenchidos
	if got.CoverageThreshold != 70.0 {
		t.Errorf("CoverageThreshold deveria ser default, got %v", got.CoverageThreshold)
	}
}
