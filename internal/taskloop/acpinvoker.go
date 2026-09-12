package taskloop

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	airuntime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/telemetry"
)

var taskPathRe = regexp.MustCompile(`([A-Za-z0-9_./-]*task-\d+\.\d+[A-Za-z0-9_.-]*\.md)`)
var taskIDRe = regexp.MustCompile(`^task-(\d+\.\d+)`)

type acpInvoker struct {
	runner          *airuntime.ACPRunner
	humanBuffer     *bytes.Buffer
	quiet           bool
	activityTimeout time.Duration

	reasoningEffort string
	accessMode      specs.AccessMode
	addDirs         []string

	mcpNested   bool
	noNormalize bool

	memoryLimits         memory.Limits
	memoryLimitsExplicit bool
	disableHooks         bool
	tasksDir             string

	skipDriftGuard bool

	autoReview bool

	maxRetries             int
	retryBackoffMultiplier float64
	retryBaseDelay         time.Duration
	retryClassifier        airuntime.RetryClassifier
	maxBugfixIterations    int

	durableMemoryEnabled bool
	handoffLeaseTTL      time.Duration

	sleepFn func(context.Context, time.Duration) error
}

type ACPInvokerOption func(*acpInvoker)

func (c *Catalog) WithACPInvokerReasoningEffort(effort string) ACPInvokerOption {
	return func(a *acpInvoker) { a.reasoningEffort = effort }
}

func (c *Catalog) WithACPInvokerAccessMode(mode specs.AccessMode) ACPInvokerOption {
	return func(a *acpInvoker) { a.accessMode = mode }
}

func (c *Catalog) WithACPInvokerAddDirs(dirs []string) ACPInvokerOption {
	return func(a *acpInvoker) { a.addDirs = dirs }
}

func (c *Catalog) WithACPInvokerMCPNested(enabled bool) ACPInvokerOption {
	return func(a *acpInvoker) { a.mcpNested = enabled }
}

func (c *Catalog) WithACPInvokerNoNormalize(disabled bool) ACPInvokerOption {
	return func(a *acpInvoker) { a.noNormalize = disabled }
}

func (c *Catalog) WithACPInvokerMemoryLimitLines(workflowLines, taskLines int) ACPInvokerOption {
	return func(a *acpInvoker) {
		a.memoryLimits.WorkflowLines = workflowLines
		a.memoryLimits.TaskLines = taskLines
	}
}

func (c *Catalog) WithACPInvokerMemoryLimitBytes(workflowBytes, taskBytes int) ACPInvokerOption {
	return func(a *acpInvoker) {
		a.memoryLimits.WorkflowBytes = workflowBytes
		a.memoryLimits.TaskBytes = taskBytes
	}
}

func (c *Catalog) WithACPInvokerMemoryLimitsExplicit(explicit bool) ACPInvokerOption {
	return func(a *acpInvoker) { a.memoryLimitsExplicit = explicit }
}

func (c *Catalog) WithACPInvokerDisableHooks(disabled bool) ACPInvokerOption {
	return func(a *acpInvoker) { a.disableHooks = disabled }
}

func (c *Catalog) WithACPInvokerTasksDir(dir string) ACPInvokerOption {
	return func(a *acpInvoker) { a.tasksDir = dir }
}

func (c *Catalog) WithACPInvokerAutoReview(enabled bool) ACPInvokerOption {
	return func(a *acpInvoker) { a.autoReview = enabled }
}

func (c *Catalog) WithACPInvokerMaxBugfixIterations(n int) ACPInvokerOption {
	return func(a *acpInvoker) { a.maxBugfixIterations = n }
}

func (c *Catalog) WithACPInvokerDurableMemory(enabled bool) ACPInvokerOption {
	return func(a *acpInvoker) { a.durableMemoryEnabled = enabled }
}

func (c *Catalog) WithACPInvokerHandoffLeaseTTL(ttl time.Duration) ACPInvokerOption {
	return func(a *acpInvoker) { a.handoffLeaseTTL = ttl }
}

func (c *Catalog) WithACPInvokerSkipDriftGuard(enabled bool) ACPInvokerOption {
	return func(a *acpInvoker) { a.skipDriftGuard = enabled }
}

func (c *Catalog) WithACPInvokerMaxRetries(n int) ACPInvokerOption {
	return func(a *acpInvoker) { a.maxRetries = n }
}

func (c *Catalog) WithACPInvokerRetryBackoffMultiplier(m float64) ACPInvokerOption {
	return func(a *acpInvoker) { a.retryBackoffMultiplier = m }
}

func (c *Catalog) WithACPInvokerRetryBaseDelay(d time.Duration) ACPInvokerOption {
	return func(a *acpInvoker) { a.retryBaseDelay = d }
}

func (c *Catalog) WithACPInvokerRetryClassifier(rc airuntime.RetryClassifier) ACPInvokerOption {
	return func(a *acpInvoker) { a.retryClassifier = rc }
}

func (c *Catalog) acpInvokerSleepFnOption(fn func(context.Context, time.Duration) error) ACPInvokerOption {
	return func(a *acpInvoker) { a.sleepFn = fn }
}

func NewACPInvoker(runner *airuntime.ACPRunner, quiet bool, activityTimeout time.Duration, opts ...ACPInvokerOption) AgentInvoker {
	buf := &bytes.Buffer{}
	runner.SetRenderer(io.MultiWriter(buf))
	inv := &acpInvoker{
		runner:          runner,
		humanBuffer:     buf,
		quiet:           quiet,
		activityTimeout: activityTimeout,
		retryClassifier: airuntime.NewRetryClassifier(),
		sleepFn:         NewCatalog().defaultSleepFn,
	}
	for _, o := range opts {
		o(inv)
	}
	return inv
}

func (c *Catalog) defaultSleepFn(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *acpInvoker) BinaryName() string {
	return "claude-agent-acp"
}

func (c *acpInvoker) Invoke(ctx context.Context, prompt, workDir, _ string) (string, string, int, error) {
	timeout, err := events.NewActivityTimeout(c.activityTimeout)
	if err != nil {
		return "", "", 1, err
	}

	evidenceDir := NewCatalog().deriveEvidenceDir(workDir, prompt)
	job := airuntime.Job{
		Prompt:      prompt,
		WorkDir:     workDir,
		EvidenceDir: evidenceDir,
		RuntimeConfig: airuntime.RuntimeConfig{
			Timeout:              timeout,
			MaxBugfixIterations:  c.maxBugfixIterations,
			HandoffLeaseTTL:      c.handoffLeaseTTL,
			DurableMemoryEnabled: c.durableMemoryEnabled,
		},
		Quiet: c.quiet,

		ReasoningEffort: c.reasoningEffort,
		AccessMode:      c.accessMode,
		AddDirs:         c.addDirs,

		MCPNested:   c.mcpNested,
		NoNormalize: c.noNormalize,

		MemoryLimits:         c.memoryLimits,
		MemoryLimitsExplicit: c.memoryLimitsExplicit,
		DisableHooks:         c.disableHooks,
		TasksDir:             c.tasksDir,
		SkipDriftGuard:       c.skipDriftGuard,

		AutoReview: c.autoReview,
	}

	var (
		summary       airuntime.Summary
		runErr        error
		retryAttempts int
	)

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {

			wait := airuntime.NewCatalog().BackoffDuration(c.retryBaseDelay, c.retryBackoffMultiplier, attempt)
			if sleepErr := c.sleepFn(ctx, wait); sleepErr != nil {

				runErr = fmt.Errorf("retry %d/%d: contexto cancelado: %w", attempt, c.maxRetries, sleepErr)
				break
			}
			fmt.Fprintf(os.Stderr, "[acpinvoker] retry %d/%d após falha transitória\n", attempt, c.maxRetries)
			retryAttempts = attempt
		}

		c.humanBuffer.Reset()
		summary, runErr = c.runner.Run(ctx, job)

		if runErr == nil || !c.retryClassifier.IsTransient(runErr) {
			break
		}
	}

	summary.RetryAttempts = retryAttempts

	exitCode := NewCatalog().MapExitCode(summary.CancelReason)
	_ = telemetry.NewCatalog().LogACPSession(workDir, telemetry.ACPSessionEvent{
		Runtime:            "acp",
		Launcher:           summary.Launcher,
		EventsCount:        summary.EventsCount,
		UnknownEventsCount: summary.UnknownEventsCount,
		CancelReason:       string(summary.CancelReason),
		SlowPublishes:      summary.SlowPublishes,
		DroppedUpdates:     summary.DroppedUpdates,
		RetryAttempts:      summary.RetryAttempts,
	})

	stdout := c.humanBuffer.String()
	return stdout, "", exitCode, runErr
}

func (c *acpInvoker) SetLiveOutput(w io.Writer) {
	combined := io.MultiWriter(c.humanBuffer, w)
	c.runner.SetRenderer(combined)
}

func (c *Catalog) MapExitCode(reason events.CancelReason) int {
	switch reason {
	case events.CancelReasonNone:
		return 0
	case events.CancelReasonPermissionDenied:
		return 3
	default:
		return 1
	}
}

func (c *Catalog) deriveEvidenceDir(workDir, prompt string) string {
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

var _ AgentInvoker = (*acpInvoker)(nil)
var _ LiveOutputSetter = (*acpInvoker)(nil)
