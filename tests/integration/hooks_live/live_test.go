//go:build hooks_live

package hooks_live

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/embedded"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/precondition"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestHooksLiveMatrixDispatchesThroughRealCLIs(t *testing.T) {
	if os.Getenv("AISPEC_HOOKS_LIVE") != "1" {
		t.Fatal("AISPEC_HOOKS_LIVE=1 is required: this suite proves native CLI dispatch and must not fall back to a contract harness")
	}
	for _, cell := range LiveMatrix() {
		t.Run(cell.Agent+"/"+cell.Point.String(), func(t *testing.T) {
			projectDir := installNativeHookFixture(t)
			run := invokeNativeCLI(t, projectDir, cell)
			assertNativeOutcome(t, projectDir, cell, run)
		})
	}
}

func installNativeHookFixture(t *testing.T) string {
	t.Helper()
	sourceDir, cleanup, err := embedded.NewExtractor().ExtractToTempDir()
	if err != nil {
		t.Fatalf("extract embedded assets: %v", err)
	}
	t.Cleanup(cleanup)
	projectDir := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create native hook fixture: %v", err)
	}
	if err := exec.Command("git", "init", projectDir).Run(); err != nil {
		t.Fatalf("initialize native hook fixture repository: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(projectDir); err != nil {
			t.Errorf("clean native hook fixture: %v", err)
		}
	})
	fileSystem := fs.NewOSFileSystem()
	service := install.NewService(fileSystem, output.New(false), manifest.NewStore(fileSystem), adapters.NewGenerator(fileSystem, output.New(false)), contextgen.NewGenerator(fileSystem, output.New(false)))
	if err := service.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   sourceDir,
		Tools:       []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: true,
	}); err != nil {
		t.Fatalf("install native hook fixture: %v", err)
	}
	configureCopilotRepositorySettings(t, projectDir)
	return projectDir
}

type nativeRun struct {
	output             string
	exitCode           int
	sessionIdleBlocked bool
}

func invokeNativeCLI(t *testing.T, projectDir string, cell MatrixCell) nativeRun {
	t.Helper()
	prepareNativeFixture(t, projectDir, cell)
	if cell.Point == specs.PointSessionEnd {
		writeActiveTask(t, projectDir)
	}
	env, codexProfile, sessionIdleMarker := configureNativeFixtureTrust(t, projectDir, cell)
	command, args, sentinel := nativeInvocation(t, projectDir, cell, codexProfile)
	if sentinel != "" {
		env = append(env, "AISPEC_OPENCODE_GOVERNANCE_SENTINEL="+sentinel)
	}
	if sessionIdleMarker != "" {
		env = append(env, "AISPEC_OPENCODE_SESSION_IDLE_SENTINEL="+sessionIdleMarker)
	}
	if _, err := exec.LookPath(command); err != nil {
		t.Fatalf("native CLI %q is required for every live matrix cell: %v", command, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = projectDir
	cmd.Env = env
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("native CLI %q exceeded the 90s live-cell deadline: %v; output=%s", command, ctx.Err(), output.String())
	}
	if sentinel != "" {
		if _, statErr := os.Stat(sentinel); statErr != nil {
			t.Fatalf("opencode governance plugin did not load before the prompt: %v; output=%s", statErr, output.String())
		}
	}
	run := nativeRun{output: output.String()}
	if sessionIdleMarker != "" {
		status, markerErr := os.ReadFile(sessionIdleMarker)
		if markerErr != nil {
			t.Fatalf("native OpenCode lifecycle did not emit session.idle: %v; output=%s", markerErr, output.String())
		}
		run.sessionIdleBlocked = string(status) == "blocked"
	}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			run.exitCode = exitErr.ExitCode()
			return run
		}
		t.Fatalf("run native CLI %q: %v; output=%s", command, err, output.String())
	}
	return run
}

func assertNativeOutcome(t *testing.T, projectDir string, cell MatrixCell, run nativeRun) {
	t.Helper()
	switch cell.Point {
	case specs.PointPreTool:
		assertPreToolOutcome(t, projectDir, cell, run)
	case specs.PointPostTool:
		assertPostToolOutcome(t, projectDir, cell, run)
	case specs.PointSessionEnd:
		assertSessionEndOutcome(t, cell, run)
	}
}

func assertPreToolOutcome(t *testing.T, projectDir string, cell MatrixCell, run nativeRun) {
	t.Helper()
	mutation := filepath.Join(projectDir, "hooks-live-pre-tool-denied.go")
	data, err := os.ReadFile(mutation)
	if err != nil {
		t.Fatalf("read native %s pre-tool mutation target: %v", cell.Agent, err)
	}
	mutated := string(data) != "package placeholder\n"
	expected := PreToolDenialDiagnostics(cell.Agent)
	dispatched := outputCarriesDiagnostic(run.output, expected)
	switch {
	case !dispatched && !mutated:
		t.Fatalf("native %s pre-tool hook never dispatched: the mutation target is intact but the output carries none of the canonical validator diagnostics %q. An unmutated file is not evidence of denial — it is equally consistent with an edit the agent never attempted, so this cell proves nothing; output=%s", cell.Agent, expected, run.output)
	case !dispatched && mutated:
		t.Fatalf("native %s pre-tool hook never dispatched and the mutation was applied: no gate stood between the agent and the file; output=%s", cell.Agent, run.output)
	case dispatched && mutated:
		t.Fatalf("native %s pre-tool hook dispatched and emitted its diagnostic but did not deny the mutation; output=%s", cell.Agent, run.output)
	}
}

func outputCarriesDiagnostic(out string, diagnostics []string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(out, diagnostic) {
			return true
		}
	}
	return false
}

func assertPostToolOutcome(t *testing.T, projectDir string, cell MatrixCell, run nativeRun) {
	t.Helper()
	mutation := filepath.Join(projectDir, "AGENTS.md")
	data, err := os.ReadFile(mutation)
	if err != nil {
		t.Fatalf("read native post-tool mutation: %v", err)
	}
	if !strings.Contains(string(data), "hooks-live-post-tool") {
		t.Fatalf("native %s post-tool hook did not allow the requested mutation; output=%s", cell.Agent, run.output)
	}
	expected := PostToolDiagnostics(cell.Agent)
	if !outputCarriesDiagnostic(run.output, expected) {
		t.Fatalf("native %s post-tool hook did not emit any canonical validator diagnostic %q; output=%s", cell.Agent, expected, run.output)
	}
}

func assertSessionEndOutcome(t *testing.T, cell MatrixCell, run nativeRun) {
	t.Helper()
	if cell.Agent == "opencode" {
		if !run.sessionIdleBlocked {
			t.Fatalf("native OpenCode lifecycle did not invoke session.idle and block the active task; output=%s", run.output)
		}
		return
	}
	expected := SessionEndDiagnostics(cell.Agent)
	if !outputCarriesDiagnostic(run.output, expected) {
		t.Fatalf("native %s session-end hook did not emit any canonical blocking diagnostic %q; output=%s", cell.Agent, expected, run.output)
	}
}

func nativeInvocation(t *testing.T, projectDir string, cell MatrixCell, codexProfile string) (string, []string, string) {
	t.Helper()
	prompt := nativePrompt(cell.Point)
	switch cell.Agent {
	case "claude":
		return "claude", []string{"--print", "--verbose", "--setting-sources", "project,local", "--permission-mode", "acceptEdits", "--include-hook-events", "--output-format", "stream-json", prompt}, ""
	case "codex":
		return "codex", []string{"exec", "--cd", projectDir, "--profile", codexProfile, "--approve-for-me", prompt}, ""
	case "copilot":
		return "copilot", []string{"-C", projectDir, "--add-dir", projectDir, "--prompt", prompt, "--allow-all-tools", "--stream", "off", "--output-format", "json"}, ""
	case "opencode":
		sentinel := filepath.Join(t.TempDir(), "governance-plugin-loaded")
		return "opencode", []string{"run", "--dir", projectDir, "--auto", prompt}, sentinel
	default:
		return "", nil, ""
	}
}

func configureNativeFixtureTrust(t *testing.T, projectDir string, cell MatrixCell) ([]string, string, string) {
	t.Helper()
	env := os.Environ()
	var codexProfile string
	var sessionIdleMarker string

	switch cell.Agent {
	case "codex":
		codexProfile = configureCodexHookTrust(t, projectDir)
	case "copilot":
		env = append(env, CopilotRepoHooksOptInEnvVar+"="+CopilotRepoHooksOptInEnvValue)
	case "opencode":
		if cell.Point == specs.PointSessionEnd {
			sessionIdleMarker = filepath.Join(t.TempDir(), "opencode-session-idle")
		}
	}
	return env, codexProfile, sessionIdleMarker
}

func configureCopilotRepositorySettings(t *testing.T, projectDir string) {
	t.Helper()
	hookConfigPath := filepath.Join(projectDir, ".github", "hooks", "governance.json")
	hookConfig, err := os.ReadFile(hookConfigPath)
	if err != nil {
		t.Fatalf("read installed Copilot hook configuration: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(hookConfig, &settings); err != nil {
		t.Fatalf("parse installed Copilot hook configuration: %v", err)
	}
	hooks, ok := settings["hooks"]
	if !ok {
		t.Fatal("installed Copilot hook configuration does not declare hooks")
	}
	repositorySettings, err := json.Marshal(map[string]any{"hooks": hooks})
	if err != nil {
		t.Fatalf("serialize Copilot repository settings: %v", err)
	}
	settingsPath := filepath.Join(projectDir, ".github", "settings.json")
	if err := os.WriteFile(settingsPath, repositorySettings, 0o600); err != nil {
		t.Fatalf("write Copilot repository settings: %v", err)
	}
}

func listCodexProjectHooks(t *testing.T, trustedProjectDir, projectTrust string) []precondition.CodexHookStatus {
	t.Helper()
	listingHome := t.TempDir()
	if err := os.WriteFile(filepath.Join(listingHome, "config.toml"), []byte(projectTrust), 0o600); err != nil {
		t.Fatalf("write Codex listing home used to read project hook hashes: %v", err)
	}
	previousHome, hadHome := os.LookupEnv("CODEX_HOME")
	if err := os.Setenv("CODEX_HOME", listingHome); err != nil {
		t.Fatalf("point Codex at the isolated listing home: %v", err)
	}
	defer func() {
		var restoreErr error
		if hadHome {
			restoreErr = os.Setenv("CODEX_HOME", previousHome)
		} else {
			restoreErr = os.Unsetenv("CODEX_HOME")
		}
		if restoreErr != nil {
			t.Errorf("restore CODEX_HOME after listing project hooks: %v", restoreErr)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client := precondition.CodexAppServerClient{Binary: "codex", WorkDir: trustedProjectDir}
	hooks, err := client.HooksList(ctx)
	if err != nil {
		t.Fatalf("list Codex project hooks for the trusted fixture: %v", err)
	}

	states := make([]precondition.CodexHookStatus, 0, len(hooks.Hooks))
	for _, hook := range hooks.Hooks {
		if hook.Source == "project" && hook.Key != "" && hook.CurrentHash != "" {
			states = append(states, hook)
		}
	}
	if len(states) == 0 {
		t.Fatalf("Codex app-server exposed no project hooks for the trusted fixture %s: the installed project must declare them in .codex/config.toml or .codex/hooks.json", trustedProjectDir)
	}
	return states
}

func configureCodexHookTrust(t *testing.T, projectDir string) string {
	t.Helper()
	trustedProjectDir, err := filepath.EvalSymlinks(projectDir)
	if err != nil {
		t.Fatalf("resolve Codex fixture project path: %v", err)
	}
	projectTrust := "[projects." + strconv.Quote(trustedProjectDir) + "]\ntrust_level = \"trusted\"\n"
	states := listCodexProjectHooks(t, trustedProjectDir, projectTrust)

	var config strings.Builder
	config.WriteString(projectTrust)
	config.WriteByte('\n')
	config.WriteString("[hooks.state]\n")
	for _, state := range states {
		config.WriteString("\n[hooks.state.")
		config.WriteString(strconv.Quote(state.Key))
		config.WriteString("]\ntrusted_hash = ")
		config.WriteString(strconv.Quote(state.CurrentHash))
		config.WriteByte('\n')
	}
	codexHome := os.Getenv("CODEX_HOME")
	if codexHome == "" {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			t.Fatalf("resolve Codex home: %v", homeErr)
		}
		codexHome = filepath.Join(homeDir, ".codex")
	}
	profile, err := os.CreateTemp(codexHome, "hooks-live-*.config.toml")
	if err != nil {
		t.Fatalf("create temporary Codex trust profile: %v", err)
	}
	profilePath := profile.Name()
	if err := profile.Close(); err != nil {
		t.Fatalf("close temporary Codex trust profile: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(profilePath); err != nil && !os.IsNotExist(err) {
			t.Errorf("remove temporary Codex trust profile: %v", err)
		}
	})
	if err := os.WriteFile(profilePath, []byte(config.String()), 0o600); err != nil {
		t.Fatalf("write temporary Codex trust profile: %v", err)
	}
	return strings.TrimSuffix(filepath.Base(profilePath), ".config.toml")
}

func nativePrompt(point specs.CanonicalPoint) string {
	switch point {
	case specs.PointPreTool:
		return "Use the edit tool now to replace the contents of hooks-live-pre-tool-denied.go with package hookslive. Do not explain or use another tool."
	case specs.PointPostTool:
		return "Use the edit tool now to append hooks-live-post-tool to AGENTS.md. Do not explain or use another tool."
	default:
		return "End the session now without taking any action."
	}
}

func prepareNativeFixture(t *testing.T, projectDir string, cell MatrixCell) {
	t.Helper()
	if cell.Point == specs.PointPreTool {
		path := filepath.Join(projectDir, ".agents", "skills", "go-implementation")
		if err := os.RemoveAll(path); err != nil {
			t.Fatalf("remove Go skill to exercise pre-tool denial: %v", err)
		}
		mutation := filepath.Join(projectDir, "hooks-live-pre-tool-denied.go")
		if err := os.WriteFile(mutation, []byte("package placeholder\n"), 0o644); err != nil {
			t.Fatalf("create native pre-tool mutation target: %v", err)
		}
	}
	if cell.Agent == "opencode" && cell.Point == specs.PointPostTool {
		path := filepath.Join(projectDir, ".agents", "hooks", "validate-governance.sh")
		validator := "#!/usr/bin/env bash\nprintf '%s\\n' 'hooks-live post-tool validator invoked' >&2\nexit 1\n"
		if err := os.WriteFile(path, []byte(validator), 0o755); err != nil {
			t.Fatalf("install observable opencode post-tool validator: %v", err)
		}
	}
}

func writeActiveTask(t *testing.T, projectDir string) {
	t.Helper()
	prdDir := filepath.Join(projectDir, ".specs", "prd-native-hook")
	if err := os.MkdirAll(prdDir, 0o755); err != nil {
		t.Fatalf("create active task fixture: %v", err)
	}
	tasks := "| # | Título | Status | Dependências | Paralelizável | Skills |\n|---|--------|--------|-------------|---------------|--------|\n| 1.0 | Native hook | in_progress | — | — | — |\n"
	if err := os.WriteFile(filepath.Join(prdDir, "tasks.md"), []byte(tasks), 0o644); err != nil {
		t.Fatalf("write active task fixture: %v", err)
	}
}
