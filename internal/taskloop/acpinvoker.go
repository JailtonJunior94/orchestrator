package taskloop

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// acpInvoker adapta ACPRunner à interface AgentInvoker.
// É o adapter entre a camada de taskloop e o application service do runtime ACP.
type acpInvoker struct {
	runner          *airuntime.ACPRunner
	humanBuffer     *bytes.Buffer
	quiet           bool
	activityTimeout time.Duration
}

// NewACPInvoker cria um acpInvoker que delega execução ao runner fornecido.
// quiet suprime o output do renderer; activityTimeout configura o watchdog.
func NewACPInvoker(runner *airuntime.ACPRunner, quiet bool, activityTimeout time.Duration) AgentInvoker {
	buf := &bytes.Buffer{}
	runner.SetRenderer(io.MultiWriter(buf))
	return &acpInvoker{
		runner:          runner,
		humanBuffer:     buf,
		quiet:           quiet,
		activityTimeout: activityTimeout,
	}
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

	evidenceDir := filepath.Join(workDir, "evidence", "acp")
	job := airuntime.Job{
		Prompt:          prompt,
		WorkDir:         workDir,
		EvidenceDir:     evidenceDir,
		ActivityTimeout: timeout,
		Quiet:           c.quiet,
	}

	c.humanBuffer.Reset()

	summary, runErr := c.runner.Run(ctx, job)
	exitCode := MapExitCode(summary.CancelReason)

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

// Garantir que acpInvoker implementa AgentInvoker e LiveOutputSetter.
var _ AgentInvoker = (*acpInvoker)(nil)
var _ LiveOutputSetter = (*acpInvoker)(nil)

