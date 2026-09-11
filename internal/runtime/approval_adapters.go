package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
)

var cycleFileLineReference = regexp.MustCompile(`[\w./-]+\.[A-Za-z0-9]+:\d+`)

const (
	parityEvidenceCommand = "parity-stage"
	parityEvidenceRecord  = "criteria gate deferred to task 5.0"
	cycleFindingFile      = "unspecified"
	cycleFindingRule      = "review-finding"
)

var cycleSeverityMarkers = []struct {
	token    string
	severity approval.Severity
}{
	{"[critical]", approval.SeverityCritical},
	{"[crítico]", approval.SeverityCritical},
	{"[critico]", approval.SeverityCritical},
	{"[hard]", approval.SeverityHigh},
	{"[high]", approval.SeverityHigh},
	{"[alta]", approval.SeverityHigh},
	{"[alto]", approval.SeverityHigh},
	{"[medium]", approval.SeverityMedium},
	{"[important]", approval.SeverityMedium},
	{"[importante]", approval.SeverityMedium},
	{"[low]", approval.SeverityLow},
	{"[suggestion]", approval.SeverityLow},
	{"[sugestão]", approval.SeverityLow},
	{"[sugestao]", approval.SeverityLow},
}

var (
	_ approval.Reviewer   = (*ReviewerAdapter)(nil)
	_ approval.Fixer      = (*FixerAdapter)(nil)
	_ approval.Repository = (*RepositoryAdapter)(nil)
)

type RepositoryAdapter struct {
	catalog *Catalog
	workDir string
}

func NewRepositoryAdapter(workDir string) *RepositoryAdapter {
	return &RepositoryAdapter{catalog: NewCatalog(), workDir: workDir}
}

func (a *RepositoryAdapter) Checkpoint(_ context.Context) (approval.Checkpoint, error) {
	head, err := a.catalog.revParseHead(a.workDir)
	if err != nil {
		return approval.Checkpoint{}, err
	}
	return approval.NewCheckpoint(head)
}

func (a *RepositoryAdapter) FullTarget(_ context.Context) (approval.ReviewTarget, error) {
	return approval.NewReviewTarget(a.catalog.collectGitDiff(a.workDir)), nil
}

func (a *RepositoryAdapter) Delta(_ context.Context, since approval.Checkpoint) (approval.ReviewTarget, error) {
	diff, err := a.catalog.runGitDiff(a.workDir, since.String())
	if err != nil {
		return approval.ReviewTarget{}, err
	}
	return approval.NewReviewTarget(diff), nil
}

func (c *Catalog) revParseHead(workDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = workDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

type ReviewerAdapter struct {
	runner  *ACPRunner
	baseJob Job
	repo    approval.Repository
}

func NewReviewerAdapter(runner *ACPRunner, baseJob Job) *ReviewerAdapter {
	return &ReviewerAdapter{runner: runner, baseJob: baseJob, repo: NewRepositoryAdapter(baseJob.WorkDir)}
}

func (a *ReviewerAdapter) Review(ctx context.Context, request approval.ReviewRequest) (approval.ReviewerOutput, error) {
	job := a.baseJob
	job.AutoReview = false

	skillBody, err := a.runner.readReviewSkill(job.WorkDir)
	if err != nil {
		return approval.ReviewerOutput{}, err
	}
	job.Prompt = NewCatalog().buildReviewPrompt(skillBody, request.Target().String())

	priorSHA, err := a.priorCutPoint(ctx, request.Round())
	if err != nil {
		return approval.ReviewerOutput{}, err
	}

	restoreEnv := NewCatalog().applyRoundReviewEnv(request.Round(), priorSHA)
	rawText, err := a.runner.spawnReviewSession(ctx, job)
	restoreEnv()
	if err != nil {
		return approval.ReviewerOutput{}, err
	}
	criteriaMap, err := parityCriteriaMap(request)
	if err != nil {
		return approval.ReviewerOutput{}, err
	}
	return approval.NewReviewerOutput(rawText, parseCycleFindings(rawText), criteriaMap), nil
}

func parseCycleFindings(rawText string) []approval.Finding {
	var findings []approval.Finding
	for _, line := range strings.Split(rawText, "\n") {
		severity, ok := severityFromLine(strings.ToLower(line))
		if !ok {
			continue
		}
		file := cycleFindingFile
		if ref := cycleFileLineReference.FindString(line); ref != "" {
			file = ref
		}
		finding, err := approval.NewFinding(severity, file, cycleFindingRule, strings.TrimSpace(line))
		if err != nil {
			continue
		}
		findings = append(findings, finding)
	}
	return findings
}

func severityFromLine(lowerLine string) (approval.Severity, bool) {
	for _, marker := range cycleSeverityMarkers {
		if strings.Contains(lowerLine, marker.token) {
			return marker.severity, true
		}
	}
	return 0, false
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

func (a *ReviewerAdapter) priorCutPoint(ctx context.Context, round int) (string, error) {
	if round < 2 {
		return "", nil
	}
	checkpoint, err := a.repo.Checkpoint(ctx)
	if err != nil {
		return "", err
	}
	return checkpoint.String(), nil
}

type FixerAdapter struct {
	runner  *ACPRunner
	baseJob Job
}

func NewFixerAdapter(runner *ACPRunner, baseJob Job) *FixerAdapter {
	return &FixerAdapter{runner: runner, baseJob: baseJob}
}

func (a *FixerAdapter) Fix(ctx context.Context, request approval.FixRequest) error {
	job := a.baseJob
	job.AutoReview = false
	job.Prompt = a.buildFixPrompt(request)

	_, err := a.runner.Run(ctx, job)
	return err
}

func (a *FixerAdapter) buildFixPrompt(request approval.FixRequest) string {
	var sb strings.Builder
	sb.WriteString("## Findings to fix\n\n")
	for finding := range request.Findings() {
		fmt.Fprintf(&sb, "- [%s] %s (%s): %s\n", finding.Severity().String(), finding.File(), finding.Rule(), finding.Description())
	}
	sb.WriteString("\n## Target\n\n```diff\n")
	sb.WriteString(request.Target().String())
	sb.WriteString("\n```\n")
	return sb.String()
}
