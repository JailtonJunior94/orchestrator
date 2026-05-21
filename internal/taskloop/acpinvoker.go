package taskloop

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"regexp"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
)

var taskPathRe = regexp.MustCompile(`([A-Za-z0-9_./-]*task-\d+\.\d+[A-Za-z0-9_.-]*\.md)`)
var taskIDRe = regexp.MustCompile(`^task-(\d+\.\d+)`)

// acpInvoker adapta ACPRunner à interface AgentInvoker.
// É o adapter entre a camada de taskloop e o application service do runtime ACP.
type acpInvoker struct {
	runner          *airuntime.ACPRunner
	humanBuffer     *bytes.Buffer
	quiet           bool
	activityTimeout time.Duration
	// Codex-specific fields propagados ao Job (RF-15, RF-26 — ADR-013 D-02).
	// Claude/Copilot recebem os valores no Job mas Spec.BootstrapArgs no-op ignora.
	reasoningEffort string
	accessMode      specs.AccessMode
	addDirs         []string
}

// ACPInvokerOption é uma functional option para acpInvoker (ADR-013 D-02).
type ACPInvokerOption func(*acpInvoker)

// WithACPInvokerReasoningEffort configura o nível de esforço de raciocínio do Codex.
// Ignorado por Claude/Copilot via Spec.BootstrapArgs no-op.
func WithACPInvokerReasoningEffort(effort string) ACPInvokerOption {
	return func(a *acpInvoker) { a.reasoningEffort = effort }
}

// WithACPInvokerAccessMode configura o modo de acesso ao sistema de arquivos do Codex.
// Ignorado por Claude/Copilot via Spec.BootstrapArgs no-op.
func WithACPInvokerAccessMode(mode specs.AccessMode) ACPInvokerOption {
	return func(a *acpInvoker) { a.accessMode = mode }
}

// WithACPInvokerAddDirs configura diretórios adicionais que o Codex pode acessar.
// Ignorado por Claude/Copilot via Spec.BootstrapArgs no-op.
func WithACPInvokerAddDirs(dirs []string) ACPInvokerOption {
	return func(a *acpInvoker) { a.addDirs = dirs }
}

// NewACPInvoker cria um acpInvoker que delega execução ao runner fornecido.
// quiet suprime o output do renderer; activityTimeout configura o watchdog.
// opts são functional options opcionais (ex: WithACPInvokerReasoningEffort).
func NewACPInvoker(runner *airuntime.ACPRunner, quiet bool, activityTimeout time.Duration, opts ...ACPInvokerOption) AgentInvoker {
	buf := &bytes.Buffer{}
	runner.SetRenderer(io.MultiWriter(buf))
	inv := &acpInvoker{
		runner:          runner,
		humanBuffer:     buf,
		quiet:           quiet,
		activityTimeout: activityTimeout,
	}
	for _, o := range opts {
		o(inv)
	}
	return inv
}

// BinaryName retorna o nome lógico do invoker ACP.
func (c *acpInvoker) BinaryName() string {
	return "claude-agent-acp"
}

// Invoke executa o runner ACP e mapeia Summary para o contrato AgentInvoker.
// Retorna (stdout consolidado, "", exitCode, err) onde exitCode segue RF-10.
func (c *acpInvoker) Invoke(ctx context.Context, prompt, workDir, _ string) (string, string, int, error) {
	timeout, err := events.NewActivityTimeout(c.activityTimeout)
	if err != nil {
		return "", "", 1, err
	}

	evidenceDir := deriveEvidenceDir(workDir, prompt)
	job := airuntime.Job{
		Prompt:          prompt,
		WorkDir:         workDir,
		EvidenceDir:     evidenceDir,
		ActivityTimeout: timeout,
		Quiet:           c.quiet,
		// Codex-specific fields (RF-15, RF-26 — ADR-013 D-02).
		// Claude/Copilot recebem os valores mas Spec.BootstrapArgs no-op ignora.
		ReasoningEffort: c.reasoningEffort,
		AccessMode:      c.accessMode,
		AddDirs:         c.addDirs,
	}

	c.humanBuffer.Reset()

	summary, runErr := c.runner.Run(ctx, job)
	exitCode := MapExitCode(summary.CancelReason)
	_ = telemetry.LogACPSession(workDir, telemetry.ACPSessionEvent{
		Runtime:            "acp",
		Launcher:           summary.Launcher,
		EventsCount:        summary.EventsCount,
		UnknownEventsCount: summary.UnknownEventsCount,
		CancelReason:       string(summary.CancelReason),
	})

	stdout := c.humanBuffer.String()
	return stdout, "", exitCode, runErr
}

// SetLiveOutput implementa LiveOutputSetter, repassando o writer ao renderer.
func (c *acpInvoker) SetLiveOutput(w io.Writer) {
	combined := io.MultiWriter(c.humanBuffer, w)
	c.runner.SetRenderer(combined)
}

// MapExitCode mapeia CancelReason para exit code conforme RF-10.
//
//	none              → 0
//	activity_timeout  → 1
//	permission_denied → 3
//	demais            → 1
func MapExitCode(reason events.CancelReason) int {
	switch reason {
	case events.CancelReasonNone:
		return 0
	case events.CancelReasonPermissionDenied:
		return 3
	default:
		return 1
	}
}

func deriveEvidenceDir(workDir, prompt string) string {
	match := taskPathRe.FindStringSubmatch(prompt)
	if len(match) < 2 {
		return filepath.Join(workDir, "evidence", "acp")
	}
	base := filepath.Base(match[1])
	idMatch := taskIDRe.FindStringSubmatch(base)
	if len(idMatch) < 2 {
		return filepath.Join(workDir, "evidence", "acp")
	}
	return filepath.Join(workDir, "evidence", "task-"+idMatch[1])
}

// Garantir que acpInvoker implementa AgentInvoker e LiveOutputSetter.
var _ AgentInvoker = (*acpInvoker)(nil)
var _ LiveOutputSetter = (*acpInvoker)(nil)
