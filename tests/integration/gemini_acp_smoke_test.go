//go:build integration

// Package integration — smoke test do Gemini CLI via ACP nativo (RF-12, T-4.0).
//
// Skip conditions:
//   - `testing.Short()` (-short flag): skip silencioso documentado.
//   - `gemini` não disponível no PATH: usando fake ACP server (acpfake).
//
// Execução:
//
//	go test -tags integration ./tests/integration/... -run TestGeminiACPSmoke
//	go test -tags integration -short ./tests/integration/... -run TestGeminiACPSmoke  # skip
package integration

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/acpfake"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/client"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// TestGeminiACPSmoke (RF-12) valida que o harness pode invocar Gemini via ACP
// e produzir events.jsonl para uma task simples.
//
// Usa fake ACP server (acpfake) — não requer binário real do gemini.
// Skip quando -short ativo e gemini não está no PATH.
func TestGeminiACPSmoke(t *testing.T) {
	// Skip condition: -short ativo sem gemini no PATH.
	if testing.Short() {
		_, pathErr := exec.LookPath("gemini")
		if pathErr != nil {
			t.Skip("TestGeminiACPSmoke: skip — -short ativo e gemini não encontrado no PATH")
		}
	}

	// Log se gemini não disponível mas não está em -short: usa fake.
	_, pathErr := exec.LookPath("gemini")
	geminiAvailable := pathErr == nil
	if !geminiAvailable {
		t.Log("TestGeminiACPSmoke: gemini não disponível no PATH — usando fake ACP server (acpfake)")
	}

	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Script fake: simula resposta Gemini completa com um tool call.
	script := acpfake.NewScript().
		AppendAgentMessage("Gemini ACP smoke: iniciando task simples").
		AppendToolCall("smoke_tc", "read_file").
		AppendToolCallUpdate("smoke_tc", "completed").
		AppendAgentMessage("Gemini ACP smoke: task concluída").
		AppendSessionEnd()

	fakeFS := fs.NewFakeFileSystem()
	pfact := persistence.NewSessionPersistenceFactory(fakeFS)

	// Prober fake: resolve "gemini" sem subprocess real.
	smokeProber := &smokeGeminiProber{geminiAvailable: geminiAvailable}

	runner := airuntime.NewACPRunner(
		specs.Gemini(),
		airuntime.WithProber(smokeProber),
		airuntime.WithClientFactory(&smokeGeminiFakeClientFactory{script: script, ctx: ctx, t: t}),
		airuntime.WithPersistenceFactory(pfact),
	)

	evidenceDir := "/evidence/gemini-smoke"
	job := airuntime.Job{
		Prompt:       "smoke test: leia o arquivo README.md e resuma em uma linha",
		WorkDir:      workDirWithAgentsMD(t),
		EvidenceDir:  evidenceDir,
		Quiet:        true,
		AccessMode:   specs.AccessModeRestricted,
		DisableHooks: true,
	}

	summary, err := runner.Run(ctx, job)
	if err != nil {
		t.Fatalf("Gemini smoke: Run: %v", err)
	}

	// RF-12: events.jsonl deve ser produzido para task simples.
	eventsPath := evidenceDir + "/events.jsonl"
	data, readErr := fakeFS.ReadFile(eventsPath)
	if readErr != nil {
		t.Fatalf("Gemini smoke: events.jsonl não encontrado: %v", readErr)
	}
	if len(data) == 0 {
		t.Fatal("Gemini smoke: events.jsonl está vazio")
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("Gemini smoke: events.jsonl lines = %d, want >= 2", len(lines))
	}

	// Primeira linha deve ser runtime_init.
	if !strings.Contains(lines[0], `"runtime_init"`) {
		t.Errorf("Gemini smoke: primeira linha de events.jsonl deve ser runtime_init; got=%q", lines[0])
	}

	// Última linha deve ser session_end.
	lastLine := lines[len(lines)-1]
	if !strings.Contains(lastLine, `"session_end"`) {
		t.Errorf("Gemini smoke: última linha de events.jsonl deve ser session_end; got=%q", lastLine)
	}

	// tool_calls.md deve existir e conter o tool name.
	tcPath := evidenceDir + "/tool_calls.md"
	tcData, tcErr := fakeFS.ReadFile(tcPath)
	if tcErr != nil {
		t.Fatalf("Gemini smoke: tool_calls.md não encontrado: %v", tcErr)
	}
	if !strings.Contains(string(tcData), "read_file") {
		t.Errorf("Gemini smoke: tool_calls.md deve conter 'read_file'; conteúdo=%q", string(tcData))
	}

	// Summary deve ter CancelReason=none e EventsCount>=4.
	if summary.CancelReason != events.CancelReasonNone {
		t.Errorf("Gemini smoke: CancelReason = %q, want none", summary.CancelReason)
	}
	if summary.EventsCount < 4 {
		t.Errorf("Gemini smoke: EventsCount = %d, want >= 4", summary.EventsCount)
	}

	t.Logf("Gemini smoke: OK — %d events, launcher=%s, tool_calls=%d",
		summary.EventsCount, summary.Launcher, len(summary.ToolCalls))
}

// smokeGeminiProber implementa runtime.Prober retornando launcher fixo para Gemini.
// Não executa subprocesso real.
type smokeGeminiProber struct {
	geminiAvailable bool
}

func (p *smokeGeminiProber) EnsureAvailable(_ context.Context, spec specs.Spec) (specs.Launcher, error) {
	if p.geminiAvailable {
		return specs.NewBinaryLauncher("/usr/local/bin/gemini"), nil
	}
	// Fallback: reportar como npx (fake — não executa realmente).
	return specs.NewNpxLauncher(specs.GeminiNpmPackage, specs.GeminiNpmVersion), nil
}

// smokeGeminiFakeClientFactory cria clientes conectados ao acpfake (sem subprocess real).
type smokeGeminiFakeClientFactory struct {
	script *acpfake.Script
	ctx    context.Context
	t      *testing.T
}

func (f *smokeGeminiFakeClientFactory) New(workDir string) client.Client {
	f.t.Helper()
	srv := acpfake.NewServer(f.script)
	pc, err := srv.Start(f.ctx)
	if err != nil {
		f.t.Fatalf("smokeGeminiFakeClientFactory: acpfake.Start: %v", err)
	}
	return client.NewTestClient(workDir, pc.ClientWriter, pc.ClientReader)
}
