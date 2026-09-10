package taskloop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskcriteria"
)

type recordingFinalReviewer struct {
	inner FinalReviewer
	last  FinalReviewResult
	calls int
}

func (r *recordingFinalReviewer) ReviewConsolidated(ctx context.Context, diff string) (FinalReviewResult, error) {
	result, err := r.inner.ReviewConsolidated(ctx, diff)
	if err != nil {
		return FinalReviewResult{}, err
	}
	r.last = result
	r.calls++
	return result, nil
}

type cycleBugfixInvoker struct {
	invoker    AgentInvoker
	workDir    string
	model      string
	data       BugfixTemplateData
	lastOutput string
	calls      int
}

func (b *cycleBugfixInvoker) InvokeBugfix(ctx context.Context, findings []Finding, diff string) (string, error) {
	data := b.data
	data.ReviewFindings = formatCycleFindings(findings)
	data.Diff = diff

	prompt, err := NewCatalog().BuildBugfixPrompt(data)
	if err != nil {
		return "", err
	}

	stdout, _, _, invokeErr := b.invoker.Invoke(ctx, prompt, b.workDir, b.model)
	b.calls++
	b.lastOutput = stdout
	if invokeErr != nil {
		return "", invokeErr
	}
	return stdout, nil
}

func formatCycleFindings(findings []Finding) string {
	lines := make([]string, 0, len(findings))
	for _, finding := range findings {
		location := strings.TrimSpace(finding.File)
		if finding.Line > 0 {
			location = fmt.Sprintf("%s:%d", location, finding.Line)
		}
		if location == "" {
			lines = append(lines, fmt.Sprintf("- [%s] %s", finding.Severity, finding.Message))
			continue
		}
		lines = append(lines, fmt.Sprintf("- [%s] [%s] %s", finding.Severity, location, finding.Message))
	}
	return strings.Join(lines, "\n")
}

type cycleDiffCapturer struct {
	workDir string
}

func (d *cycleDiffCapturer) CaptureDiff(ctx context.Context) (string, error) {
	return NewCatalog().captureGitDiff(ctx, d.workDir), nil
}

func acceptanceCriteriaFromTaskFile(content []byte) ([]approval.AcceptanceCriterion, error) {
	seen := make(map[string]bool)
	var criteria []approval.AcceptanceCriterion
	for _, description := range taskcriteria.Extract(content) {
		key := strings.TrimSpace(description)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		criterion, err := approval.NewAcceptanceCriterion(description)
		if err != nil {
			return nil, err
		}
		criteria = append(criteria, criterion)
	}
	return criteria, nil
}

func cycleAgentIdentity(opts Options) string {
	if strings.TrimSpace(opts.AgentName) != "" {
		return opts.AgentName
	}
	if opts.Profiles != nil {
		return opts.Profiles.Executor.Tool()
	}
	return opts.Tool
}

func bugfixResultFromCycle(invoker *cycleBugfixInvoker, elapsed time.Duration, approved, interrupted bool) *BugfixResult {
	if invoker.calls == 0 {
		return nil
	}
	result := &BugfixResult{Duration: elapsed, Output: invoker.lastOutput}
	switch {
	case interrupted:
		result.ExitCode = 1
		result.Note = "approval cycle interrupted during fix"
	case approved:
		result.ExitCode = 0
	default:
		result.ExitCode = 1
		result.Note = "applied fixes did not reach approval"
	}
	return result
}

func (s *Service) conductApprovalCycle(
	ctx context.Context,
	opts Options,
	task TaskEntry,
	criteria []approval.AcceptanceCriterion,
	relTaskFile, relPRD, workDir string,
) (*ReviewResult, *BugfixResult) {
	reviewerInvoker, err := s.createInvokerWithFallback(opts.Profiles.Reviewer.Tool(), opts.ReviewerFallbackModel)
	if err != nil {
		return &ReviewResult{Note: fmt.Sprintf("failed to create reviewer invoker: %v", err)}, nil
	}

	executorTool := opts.Tool
	executorModel := ""
	if opts.Profiles != nil {
		executorTool = opts.Profiles.Executor.Tool()
		executorModel = opts.Profiles.Executor.Model()
	}
	executorInvoker, err := s.createInvokerWithFallback(executorTool, opts.ExecutorFallbackModel)
	if err != nil {
		return &ReviewResult{Note: fmt.Sprintf("failed to create executor invoker: %v", err)}, nil
	}

	reviewer := &recordingFinalReviewer{
		inner: NewFinalReviewer(reviewerInvoker, workDir, opts.Profiles.Reviewer.Model()),
	}
	bugfixInvoker := &cycleBugfixInvoker{
		invoker: executorInvoker,
		workDir: workDir,
		model:   executorModel,
		data: BugfixTemplateData{
			TaskFile:  relTaskFile,
			PRDFolder: relPRD,
			TechSpec:  filepath.Join(relPRD, "techspec.md"),
			TasksFile: filepath.Join(relPRD, "tasks.md"),
		},
	}
	recorder := newBugfixEvidenceRecorder()

	taskIdentity, err := approval.NewTaskIdentity("task-" + task.ID)
	if err != nil {
		return &ReviewResult{Note: fmt.Sprintf("invalid task identity: %v", err)}, nil
	}
	agentIdentity, err := approval.NewAgentIdentity(cycleAgentIdentity(opts))
	if err != nil {
		return &ReviewResult{Note: fmt.Sprintf("invalid agent identity: %v", err)}, nil
	}
	policy, err := approval.NewApprovalPolicy()
	if err != nil {
		return &ReviewResult{Note: fmt.Sprintf("invalid approval policy: %v", err)}, nil
	}

	cycle, err := approval.NewCycle(
		taskIdentity,
		agentIdentity,
		policy,
		criteria,
		newReviewerPort(reviewer),
		newFixerPort(bugfixInvoker, recorder),
		newRepositoryPort(&cycleDiffCapturer{workDir: workDir}, workDir),
	)
	if err != nil {
		return &ReviewResult{Note: fmt.Sprintf("failed to build approval cycle: %v", err)}, nil
	}

	start := time.Now()
	result, runErr := cycle.Run(ctx)
	elapsed := time.Since(start)

	review := &ReviewResult{Duration: elapsed, Output: reviewer.last.RawOutput}
	if runErr != nil {
		review.ExitCode = 1
		review.Note = fmt.Sprintf("approval cycle error: %v", runErr)
		return review, bugfixResultFromCycle(bugfixInvoker, elapsed, false, true)
	}

	if result.Approved() {
		review.ExitCode = 0
	} else {
		review.ExitCode = 1
		review.Note = fmt.Sprintf("approval cycle closed without approval: %s", result.Reason())
	}

	return review, bugfixResultFromCycle(bugfixInvoker, elapsed, result.Approved(), false)
}
