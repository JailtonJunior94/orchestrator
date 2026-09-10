package taskloop

import (
	"context"
	"fmt"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

var (
	_ approval.Reviewer   = (*reviewerPort)(nil)
	_ approval.Fixer      = (*fixerPort)(nil)
	_ approval.Repository = (*repositoryPort)(nil)
)

const (
	parityEvidenceCommand = "parity-stage"
	parityEvidenceRecord  = "criteria gate deferred to task 5.0"
	defaultFindingFile    = "unspecified"
	defaultFindingRule    = "review-finding"
)

type reviewerPort struct {
	reviewer FinalReviewer
}

func newReviewerPort(reviewer FinalReviewer) *reviewerPort {
	return &reviewerPort{reviewer: reviewer}
}

func (p *reviewerPort) Review(ctx context.Context, request approval.ReviewRequest) (approval.ReviewerOutput, error) {
	result, err := p.reviewer.ReviewConsolidated(ctx, request.Target().String())
	if err != nil {
		return approval.ReviewerOutput{}, err
	}

	findings, err := translateReviewFindings(result.Findings)
	if err != nil {
		return approval.ReviewerOutput{}, err
	}

	criteriaMap, err := parityCriteriaMap(request)
	if err != nil {
		return approval.ReviewerOutput{}, err
	}

	return approval.NewReviewerOutput(result.RawOutput, findings, criteriaMap), nil
}

func parityCriteriaMap(request approval.ReviewRequest) (approval.CriteriaMap, error) {
	var criteria []approval.AcceptanceCriterion
	for criterion := range request.Criteria() {
		criteria = append(criteria, criterion)
	}

	criteriaMap, err := approval.NewCriteriaMap(criteria)
	if err != nil {
		return approval.CriteriaMap{}, err
	}

	evidence, err := approval.NewCommandEvidence(parityEvidenceCommand, parityEvidenceRecord)
	if err != nil {
		return approval.CriteriaMap{}, err
	}

	for _, criterion := range criteria {
		criteriaMap, err = criteriaMap.WithEvidence(criterion, evidence)
		if err != nil {
			return approval.CriteriaMap{}, err
		}
	}

	return criteriaMap, nil
}

func translateReviewFindings(in []Finding) ([]approval.Finding, error) {
	out := make([]approval.Finding, 0, len(in))
	for _, finding := range in {
		file := strings.TrimSpace(finding.File)
		if file == "" {
			file = defaultFindingFile
		}

		translated, err := approval.NewFinding(translateSeverity(finding.Severity), file, defaultFindingRule, finding.Message)
		if err != nil {
			return nil, err
		}
		out = append(out, translated)
	}
	return out, nil
}

func translateSeverity(severity Severity) approval.Severity {
	switch severity {
	case SeverityCritical:
		return approval.SeverityCritical
	case SeverityImportant:
		return approval.SeverityMedium
	default:
		return approval.SeverityLow
	}
}

type bugfixEvidenceRecorder struct {
	entries []bugfixEvidence
}

func newBugfixEvidenceRecorder() *bugfixEvidenceRecorder {
	return &bugfixEvidenceRecorder{}
}

func (r *bugfixEvidenceRecorder) record(entry bugfixEvidence) {
	r.entries = append(r.entries, entry)
}

func (r *bugfixEvidenceRecorder) Entries() []bugfixEvidence {
	return append([]bugfixEvidence(nil), r.entries...)
}

type fixerPort struct {
	invoker  BugfixInvoker
	recorder *bugfixEvidenceRecorder
}

func newFixerPort(invoker BugfixInvoker, recorder *bugfixEvidenceRecorder) *fixerPort {
	return &fixerPort{invoker: invoker, recorder: recorder}
}

func (p *fixerPort) Fix(ctx context.Context, request approval.FixRequest) error {
	output, err := p.invoker.InvokeBugfix(ctx, reverseFindings(request), request.Target().String())
	if err != nil {
		return err
	}

	evidence, err := NewCatalog().extractBugfixEvidence(output)
	if err != nil {
		return err
	}

	p.recorder.record(evidence)
	return nil
}

func reverseFindings(request approval.FixRequest) []Finding {
	var out []Finding
	for finding := range request.Findings() {
		out = append(out, Finding{
			Severity: reverseSeverity(finding.Severity()),
			File:     finding.File(),
			Message:  finding.Description(),
		})
	}
	return out
}

func reverseSeverity(severity approval.Severity) Severity {
	switch severity {
	case approval.SeverityCritical, approval.SeverityHigh:
		return SeverityCritical
	case approval.SeverityMedium:
		return SeverityImportant
	default:
		return SeveritySuggestion
	}
}

type repositoryPort struct {
	capturer DiffCapturer
	workDir  string
}

func newRepositoryPort(capturer DiffCapturer, workDir string) *repositoryPort {
	return &repositoryPort{capturer: capturer, workDir: workDir}
}

func (p *repositoryPort) Checkpoint(ctx context.Context) (approval.Checkpoint, error) {
	out, err := NewCatalog().commandOutput(ctx, p.workDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return approval.Checkpoint{}, fmt.Errorf("taskloop: git rev-parse HEAD: %w", err)
	}
	return approval.NewCheckpoint(strings.TrimSpace(string(out)))
}

func (p *repositoryPort) FullTarget(ctx context.Context) (approval.ReviewTarget, error) {
	return p.captureTarget(ctx)
}

func (p *repositoryPort) Delta(ctx context.Context, _ approval.Checkpoint) (approval.ReviewTarget, error) {
	return p.captureTarget(ctx)
}

func (p *repositoryPort) captureTarget(ctx context.Context) (approval.ReviewTarget, error) {
	diff, err := p.capturer.CaptureDiff(ctx)
	if err != nil {
		return approval.ReviewTarget{}, err
	}
	return approval.NewReviewTarget(diff), nil
}
