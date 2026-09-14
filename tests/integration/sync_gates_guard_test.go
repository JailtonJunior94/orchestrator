//go:build integration

package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var syncGateMirrorDirs = []string{
	".agents",
	".claude",
	".codex",
	".github",
	".opencode",
	"scripts",
	filepath.Join("internal", "embedded", "assets"),
}

// G7: ordem do mais barato para o mais caro. Os testes de mutacao retornam no
// primeiro gate que falha, entao rodar check-skills-sync.sh (o unico com custo
// de segundos) por ultimo corta a carga total da suite sem enfraquecer a prova.
var syncGateScripts = []string{
	filepath.Join("scripts", "check-hooks-sync.sh"),
	filepath.Join("scripts", "check-scripts-sync.sh"),
	filepath.Join("scripts", "check-skills-sync.sh"),
}

// G7: o snapshot completo do repositorio e feito UMA vez por processo. Cada
// subteste recebe um clone por hardlink (cp -Rl) a partir desse snapshot, que e
// ordens de grandeza mais barato que um cp -R por subteste. O snapshot e uma
// copia real e privada, entao uma escrita in-place acidental no clone nunca
// alcanca a working tree do repositorio.
var (
	syncGateBaseOnce sync.Once
	syncGateBaseDir  string
	syncGateBaseErr  error
)

func copyTree(src, dst string, hardlink bool) error {
	args := []string{"-R"}
	if hardlink {
		args = append(args, "-l")
	}
	args = append(args, src, dst)
	cmd := exec.Command("cp", args...)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cp %v: %w (%s)", args, err, out.String())
	}
	return nil
}

func syncGateBaseSnapshot(t *testing.T) string {
	t.Helper()
	syncGateBaseOnce.Do(func() {
		source := repoRootForDispatch(t)
		base, err := os.MkdirTemp("", "sync-gate-base-")
		if err != nil {
			syncGateBaseErr = err
			return
		}
		for _, rel := range syncGateMirrorDirs {
			src := filepath.Join(source, rel)
			if _, statErr := os.Stat(src); statErr != nil {
				syncGateBaseErr = fmt.Errorf("source dir %s missing: %w", rel, statErr)
				return
			}
			dst := filepath.Join(base, rel)
			if mkErr := os.MkdirAll(filepath.Dir(dst), 0o755); mkErr != nil {
				syncGateBaseErr = mkErr
				return
			}
			if cpErr := copyTree(src, dst, false); cpErr != nil {
				syncGateBaseErr = cpErr
				return
			}
		}
		syncGateBaseDir = base
	})
	if syncGateBaseErr != nil {
		t.Fatalf("build sync gate base snapshot: %v", syncGateBaseErr)
	}
	return syncGateBaseDir
}

func cloneRepoForSyncGate(t *testing.T) string {
	t.Helper()
	base := syncGateBaseSnapshot(t)
	clone := t.TempDir()
	for _, rel := range syncGateMirrorDirs {
		dst := filepath.Join(clone, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dst, err)
		}
		if err := copyTree(filepath.Join(base, rel), dst, true); err != nil {
			t.Fatalf("clone %s: %v", rel, err)
		}
	}
	return clone
}

// replaceInClone escreve conteudo novo sem tocar o inode compartilhado com o
// snapshot: remove o hardlink antes de criar o arquivo.
func replaceInClone(t *testing.T, clone, rel string, data []byte) {
	t.Helper()
	path := filepath.Join(clone, rel)
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("remove %s: %v", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func runSyncGate(t *testing.T, clone, script string) (string, int) {
	t.Helper()
	cmd := exec.Command("bash", filepath.Join(clone, script))
	cmd.Dir = clone
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %s: %v", script, err)
		}
		exitCode = ee.ExitCode()
	}
	return out.String(), exitCode
}

func TestSyncGatesAreGreenOnUntouchedClone(t *testing.T) {
	t.Parallel()
	clone := cloneRepoForSyncGate(t)

	for _, script := range syncGateScripts {
		out, exitCode := runSyncGate(t, clone, script)
		if exitCode != 0 {
			t.Fatalf("%s must be green on an untouched clone; exit=%d output=%s", script, exitCode, out)
		}
	}
}

func TestSyncGatesFailWhenCentralArtifactIsDeleted(t *testing.T) {
	t.Parallel()

	artifacts := []string{
		filepath.Join(".agents", "scripts", "validate-session-end.sh"),
		filepath.Join(".opencode", "plugin", "governance.js"),
		filepath.Join("internal", "embedded", "assets", ".opencode", "plugin", "governance.js"),
		filepath.Join(".agents", "hooks", "validate-preload.sh"),
		filepath.Join(".agents", "scripts", "hook-prereq-gate.sh"),
		filepath.Join(".agents", "scripts", "validate-task-evidence.sh"),
		filepath.Join(".agents", "scripts", "resolve-references.sh"),
		filepath.Join(".agents", "scripts", "validate-skill-prerequisites.sh"),
		filepath.Join(".agents", "lib", "check-invocation-depth.sh"),
		filepath.Join(".agents", "lib", "parse-hook-input.sh"),
		filepath.Join(".agents", "hooks", "post-execute-task.sh"),
		filepath.Join(".agents", "hooks", "subagent-stop-wrapper.sh"),
		filepath.Join(".agents", "hooks", "validate-session-end.sh"),
		filepath.Join(".claude", "scripts", "validate-session-end.sh"),
	}

	for _, artifact := range artifacts {
		t.Run(artifact, func(t *testing.T) {
			t.Parallel()
			clone := cloneRepoForSyncGate(t)
			if err := os.Remove(filepath.Join(clone, artifact)); err != nil {
				t.Fatalf("remove %s: %v", artifact, err)
			}

			failed := false
			for _, script := range syncGateScripts {
				out, exitCode := runSyncGate(t, clone, script)
				if exitCode != 0 {
					failed = true
					continue
				}
				t.Logf("%s stayed green after deleting %s; output=%s", script, artifact, out)
			}
			if !failed {
				t.Fatalf("deleting %s must make at least one sync gate fail; both gates stayed green", artifact)
			}
		})
	}
}

func TestSkillsSyncGateFailsWhenCanonicalSkillIsMissingFromMirror(t *testing.T) {
	t.Parallel()

	mirrors := []string{
		filepath.Join(".claude", "skills"),
		filepath.Join(".github", "skills"),
		filepath.Join("internal", "embedded", "assets", ".agents", "skills"),
	}

	for _, mirror := range mirrors {
		t.Run(mirror, func(t *testing.T) {
			t.Parallel()
			clone := cloneRepoForSyncGate(t)
			skill := filepath.Join(clone, mirror, "domain-modeling-production")
			if _, err := os.Stat(skill); err != nil {
				t.Fatalf("mirror %s must ship domain-modeling-production: %v", mirror, err)
			}
			if err := os.RemoveAll(skill); err != nil {
				t.Fatalf("remove %s: %v", skill, err)
			}

			out, exitCode := runSyncGate(t, clone, filepath.Join("scripts", "check-skills-sync.sh"))
			if exitCode == 0 {
				t.Fatalf("check-skills-sync.sh stayed green after removing %s from %s; output=%s", "domain-modeling-production", mirror, out)
			}
		})
	}
}

func TestSkillsSyncGateAllowlistsNonSkillDirectories(t *testing.T) {
	t.Parallel()
	clone := cloneRepoForSyncGate(t)

	if _, err := os.Stat(filepath.Join(clone, ".agents", "skills", "tests", "conftest.py")); err != nil {
		t.Fatalf(".agents/skills/tests must stay a non-skill pytest directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(clone, ".agents", "skills", "tests", "SKILL.md")); err == nil {
		t.Fatalf(".agents/skills/tests gained a SKILL.md; it must leave the non-skill allowlist")
	}

	out, exitCode := runSyncGate(t, clone, filepath.Join("scripts", "check-skills-sync.sh"))
	if exitCode != 0 {
		t.Fatalf("check-skills-sync.sh must stay green with the allowlisted non-skill dir; exit=%d output=%s", exitCode, out)
	}
	if !strings.Contains(out, "SKIP: tests (allowlist") {
		t.Fatalf("allowlisted non-skill dir must be reported explicitly, never silently; output=%s", out)
	}
}

func TestCodexLegacyHooksJSONIsNotShipped(t *testing.T) {
	t.Parallel()

	repo := repoRootForDispatch(t)
	if _, err := os.Stat(filepath.Join(repo, ".codex", "hooks.json")); err == nil {
		t.Fatalf(".codex/hooks.json must not coexist with .codex/config.toml; the Codex CLI loads hooks from a single representation per layer")
	}

	clone := cloneRepoForSyncGate(t)
	replaceInClone(t, clone, filepath.Join(".codex", "hooks.json"), []byte("{}\n"))

	out, exitCode := runSyncGate(t, clone, filepath.Join("scripts", "check-hooks-sync.sh"))
	if exitCode == 0 {
		t.Fatalf("check-hooks-sync.sh stayed green with a legacy .codex/hooks.json; output=%s", out)
	}
}

func TestClaudeLocalOnlyHookStaysOutOfEmbeddedMirror(t *testing.T) {
	t.Parallel()

	const hook = "validate-token-budget.sh"
	clone := cloneRepoForSyncGate(t)
	local := filepath.Join(clone, ".claude", "hooks", hook)
	mirror := filepath.Join(clone, "internal", "embedded", "assets", ".claude", "hooks", hook)

	if _, err := os.Stat(local); err != nil {
		t.Fatalf(".claude/hooks/%s must exist as a local-only hook: %v", hook, err)
	}
	if _, err := os.Stat(mirror); err == nil {
		t.Fatalf("%s is declared local-only but is embedded; update the allowlist or drop the mirror", hook)
	}

	out, exitCode := runSyncGate(t, clone, filepath.Join("scripts", "check-hooks-sync.sh"))
	if exitCode != 0 {
		t.Fatalf("check-hooks-sync.sh must stay green with the declared local-only hook; exit=%d output=%s", exitCode, out)
	}
	if !strings.Contains(out, "LOCAL-ONLY: .claude/hooks/"+hook) {
		t.Fatalf("local-only exclusion must be reported explicitly, never silently; output=%s", out)
	}

	data, err := os.ReadFile(local)
	if err != nil {
		t.Fatalf("read %s: %v", local, err)
	}
	replaceInClone(t, clone, filepath.Join("internal", "embedded", "assets", ".claude", "hooks", hook), data)
	if out, exitCode := runSyncGate(t, clone, filepath.Join("scripts", "check-hooks-sync.sh")); exitCode == 0 {
		t.Fatalf("check-hooks-sync.sh stayed green after embedding a local-only hook; output=%s", out)
	}
}

// G1: a ausencia do diretorio canonico inteiro era aprovada por vacuidade —
// blocos de paridade guardados por `if [[ -d ... ]]` sem ramo de falha.
func TestSyncGatesFailWhenCanonicalDirectoryIsDeleted(t *testing.T) {
	t.Parallel()

	dirs := []string{
		filepath.Join(".agents", "lib"),
		filepath.Join(".agents", "hooks"),
		filepath.Join(".agents", "scripts"),
		filepath.Join(".agents", "skills"),
	}

	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			t.Parallel()
			clone := cloneRepoForSyncGate(t)
			if err := os.RemoveAll(filepath.Join(clone, dir)); err != nil {
				t.Fatalf("remove %s: %v", dir, err)
			}

			for _, script := range syncGateScripts {
				if _, exitCode := runSyncGate(t, clone, script); exitCode != 0 {
					return
				}
			}
			t.Fatalf("deleting the whole canonical dir %s must make at least one sync gate fail; every gate stayed green", dir)
		})
	}
}

// G2: drift de conteudo em QUALQUER copia do gate de encerramento precisa ser
// visivel. A copia em .claude/scripts/ nao estava em nenhuma lista de gate.
func TestSyncGatesFailOnSessionEndDriftInEveryMirror(t *testing.T) {
	t.Parallel()

	mirrors := []string{
		filepath.Join(".agents", "hooks", "validate-session-end.sh"),
		filepath.Join(".claude", "hooks", "validate-session-end.sh"),
		filepath.Join(".claude", "scripts", "validate-session-end.sh"),
		filepath.Join(".codex", "hooks", "validate-session-end.sh"),
		filepath.Join(".github", "hooks", "validate-session-end.sh"),
		filepath.Join("internal", "embedded", "assets", ".agents", "hooks", "validate-session-end.sh"),
		filepath.Join("internal", "embedded", "assets", ".agents", "scripts", "validate-session-end.sh"),
		filepath.Join("internal", "embedded", "assets", ".claude", "hooks", "validate-session-end.sh"),
		filepath.Join("internal", "embedded", "assets", ".claude", "scripts", "validate-session-end.sh"),
		filepath.Join("internal", "embedded", "assets", ".codex", "hooks", "validate-session-end.sh"),
		filepath.Join("internal", "embedded", "assets", ".github", "hooks", "validate-session-end.sh"),
	}

	for _, mirror := range mirrors {
		t.Run(mirror, func(t *testing.T) {
			t.Parallel()
			clone := cloneRepoForSyncGate(t)
			original, err := os.ReadFile(filepath.Join(clone, mirror))
			if err != nil {
				t.Fatalf("read %s: %v", mirror, err)
			}
			replaceInClone(t, clone, mirror, append(original, []byte("\nexit 0\n")...))

			for _, script := range syncGateScripts {
				if _, exitCode := runSyncGate(t, clone, script); exitCode != 0 {
					return
				}
			}
			t.Fatalf("drift in %s must make at least one sync gate fail; every gate stayed green", mirror)
		})
	}
}

// G3: suite de validador existente, verde e nao referenciada e gate inativo.
func TestNoOrphanValidatorTestSuites(t *testing.T) {
	t.Parallel()

	repo := repoRootForDispatch(t)
	makefile, err := os.ReadFile(filepath.Join(repo, "Makefile"))
	if err != nil {
		t.Fatalf("read Makefile: %v", err)
	}
	// Mesma classe do G6: mencao textual nao e invocacao. So conta o que esta em
	// linha de receita (iniciada por TAB) e fora de comentario.
	var recipes strings.Builder
	for _, line := range strings.Split(string(makefile), "\n") {
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		command := strings.TrimSpace(strings.TrimPrefix(line, "\t"))
		command = strings.TrimPrefix(command, "@")
		command = strings.TrimPrefix(command, "-")
		if strings.HasPrefix(command, "#") {
			continue
		}
		recipes.WriteString(command)
		recipes.WriteByte('\n')
	}
	invoked := recipes.String()

	entries, err := os.ReadDir(filepath.Join(repo, "tests", "scripts"))
	if err != nil {
		t.Fatalf("read tests/scripts: %v", err)
	}

	// Suites deliberadamente fora do Makefile precisam entrar aqui com motivo.
	declaredOutOfBand := map[string]string{}

	found := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.sh") {
			continue
		}
		found++
		rel := "tests/scripts/" + entry.Name()
		if _, ok := declaredOutOfBand[entry.Name()]; ok {
			continue
		}
		if !strings.Contains(invoked, rel) {
			t.Errorf("suite orfa: %s existe mas nenhum alvo do Makefile a invoca; ligue-a (test-validators/check-spec-paths) ou declare-a em declaredOutOfBand com motivo", rel)
		}
	}
	if found == 0 {
		t.Fatalf("inventario vazio: tests/scripts/ deve conter ao menos uma suite *_test.sh")
	}
}

// G3: o alvo que roda as suites precisa continuar no gate de CI.
func TestValidatorSuitesRunInCI(t *testing.T) {
	t.Parallel()

	repo := repoRootForDispatch(t)
	workflow, err := os.ReadFile(filepath.Join(repo, ".github", "workflows", "test.yml"))
	if err != nil {
		t.Fatalf("read test.yml: %v", err)
	}
	for _, target := range []string{"make test-validators", "make check-spec-paths", "make test-hooks"} {
		if !strings.Contains(string(workflow), target) {
			t.Errorf("%q precisa rodar no gate de PR (.github/workflows/test.yml)", target)
		}
	}
}
