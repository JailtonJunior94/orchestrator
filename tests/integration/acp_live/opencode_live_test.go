//go:build acp_live

package acp_live

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func detectOpenCode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("t.Skip: binário opencode ausente do PATH — instale-o para rodar o smoke live do OpenCode")
	}
}

func setupOpenCodeScratchProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644); err != nil {
		t.Fatalf("setup AGENTS.md: %v", err)
	}

	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := svc.Execute(config.InstallOptions{
		ProjectDir: dir,
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("install opencode governance into scratch project: %v", err)
	}
	return dir
}

func runOpenCodeKillSwitchScenario(t *testing.T, killSwitchEnvVar string) {
	t.Helper()
	detectOpenCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	workDir := setupOpenCodeScratchProject(t)

	t.Setenv(killSwitchEnvVar, "1")

	runner := airuntime.NewACPRunner(specs.NewCatalog().OpenCode(),
		airuntime.NewCatalog().WithPersistenceFactory(&nopPersistenceFactory{}))

	timeout, err := events.NewActivityTimeout(30 * time.Second)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	job := airuntime.Job{
		Prompt:      "echo OK",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		RuntimeConfig: airuntime.RuntimeConfig{
			Timeout: timeout,
		},
		Quiet: true,
	}

	summary, runErr := runner.Run(ctx, job)
	if runErr != nil {
		t.Fatalf("kill switch %s must have no observable effect on the orchestrated session (RF-21): %v", killSwitchEnvVar, runErr)
	}
	t.Logf("kill switch %s neutralized: events=%d cancel=%s", killSwitchEnvVar, summary.EventsCount, summary.CancelReason)
}

func TestACPLive_OpenCode_PureKillSwitchHasNoEffectOnOrchestratedSession(t *testing.T) {
	runOpenCodeKillSwitchScenario(t, "OPENCODE_PURE")
}

func TestACPLive_OpenCode_DisableProjectConfigKillSwitchHasNoEffectOnOrchestratedSession(t *testing.T) {
	runOpenCodeKillSwitchScenario(t, "OPENCODE_DISABLE_PROJECT_CONFIG")
}

func TestACPLive_OpenCode_HandshakeSucceedsWithRealPlugin(t *testing.T) {
	detectOpenCode(t)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	workDir := setupOpenCodeScratchProject(t)

	runner := airuntime.NewACPRunner(specs.NewCatalog().OpenCode(),
		airuntime.NewCatalog().WithPersistenceFactory(&nopPersistenceFactory{}))

	timeout, err := events.NewActivityTimeout(30 * time.Second)
	if err != nil {
		t.Fatalf("NewActivityTimeout: %v", err)
	}

	job := airuntime.Job{
		Prompt:      "echo OK",
		WorkDir:     workDir,
		EvidenceDir: t.TempDir(),
		RuntimeConfig: airuntime.RuntimeConfig{
			Timeout: timeout,
		},
		Quiet: true,
	}

	summary, runErr := runner.Run(ctx, job)
	if runErr != nil {
		t.Fatalf("handshake with real opencode + real plugin failed: %v", runErr)
	}
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("CancelReason = %q, want none", summary.CancelReason)
	}
}
