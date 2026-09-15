package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/approval"
	"github.com/JailtonJunior94/ai-spec-harness/internal/invocation"
)

var (
	_ approval.Reviewer   = (*ReviewerAdapter)(nil)
	_ approval.Fixer      = (*FixerAdapter)(nil)
	_ approval.Repository = (*RepositoryAdapter)(nil)
)

type RepositoryAdapter struct {
	catalog *Catalog
	workDir string
	issued  []approval.Checkpoint
}

func NewRepositoryAdapter(workDir string) *RepositoryAdapter {
	return &RepositoryAdapter{catalog: NewCatalog(), workDir: workDir}
}

func (a *RepositoryAdapter) Checkpoint(_ context.Context) (approval.Checkpoint, error) {
	head, err := a.catalog.revParseHead(a.workDir)
	if err != nil {
		return approval.Checkpoint{}, err
	}
	checkpoint, err := approval.NewCheckpoint(head)
	if err != nil {
		return approval.Checkpoint{}, err
	}
	a.issued = append(a.issued, checkpoint)
	return checkpoint, nil
}

func (a *RepositoryAdapter) CheckpointAt(round int) (approval.Checkpoint, bool) {
	if round < 1 || round > len(a.issued) {
		return approval.Checkpoint{}, false
	}
	return a.issued[round-1], true
}

func (a *RepositoryAdapter) FullTarget(ctx context.Context) (approval.ReviewTarget, error) {
	diff := a.catalog.collectGitDiffContext(ctx, a.workDir)
	if diff == reviewDiffUnavailable {
		return approval.NewReviewTarget(""), nil
	}
	return approval.NewReviewTarget(diff), nil
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

type cutPointHistory interface {
	CheckpointAt(round int) (approval.Checkpoint, bool)
}

const taskStatusBlocked = "blocked"

var (
	taskStatusFieldRe = regexp.MustCompile(`(?i)\*\*Status:\*\*\s*(.+)`)
	taskTableRowRe    = regexp.MustCompile(`^\|\s*(\d+\.\d+)\s*\|`)
	taskIDReference   = regexp.MustCompile(`\d+\.\d+`)
)

type TaskStatusWriter struct {
	tasksDir     string
	taskFileName string
}

func NewTaskStatusWriter(tasksDir, taskFileName string) *TaskStatusWriter {
	return &TaskStatusWriter{tasksDir: strings.TrimSpace(tasksDir), taskFileName: strings.TrimSpace(taskFileName)}
}

func (w *TaskStatusWriter) Force(status string) error {
	if w.tasksDir == "" || w.taskFileName == "" {
		return nil
	}
	if err := w.writeTaskFileStatus(status); err != nil {
		return err
	}
	return w.writeTasksTableStatus(status)
}

func (w *TaskStatusWriter) writeTaskFileStatus(status string) error {
	path := filepath.Join(w.tasksDir, w.taskFileName)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("force task status %q at %q: %w", status, path, err)
	}
	if !taskStatusFieldRe.Match(content) {
		return fmt.Errorf("force task status %q at %q: no status field found", status, path)
	}
	updated := taskStatusFieldRe.ReplaceAll(content, []byte("**Status:** "+status))
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return fmt.Errorf("force task status %q at %q: %w", status, path, err)
	}
	return nil
}

func DetectTaskTableColumns(cols []string, statusIdx, depsIdx *int) bool {
	foundStatus := false
	detectedStatus, detectedDeps := *statusIdx, *depsIdx
	for i, col := range cols {
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "status":
			detectedStatus = i
			foundStatus = true
		case "dependências", "dependencias", "dependência", "dependencia", "deps":
			detectedDeps = i
		}
	}
	if !foundStatus {
		return false
	}
	*statusIdx, *depsIdx = detectedStatus, detectedDeps
	return true
}

func detectTaskStatusColumn(lines []string) int {
	statusIdx, depsIdx := 3, 4
	for _, line := range lines {
		cols := strings.Split(strings.TrimSpace(line), "|")
		if DetectTaskTableColumns(cols, &statusIdx, &depsIdx) {
			break
		}
	}
	return statusIdx
}

func (w *TaskStatusWriter) writeTasksTableStatus(status string) error {
	taskID := taskIDReference.FindString(w.taskFileName)
	if taskID == "" {
		return nil
	}
	path := filepath.Join(w.tasksDir, "tasks.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("force task status %q for task %s at %q: %w", status, taskID, path, err)
	}

	lines := strings.Split(string(content), "\n")
	statusIdx := detectTaskStatusColumn(lines)

	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !taskTableRowRe.MatchString(trimmed) {
			continue
		}
		columns := strings.Split(trimmed, "|")
		if len(columns) <= statusIdx || strings.TrimSpace(columns[1]) != taskID {
			continue
		}
		columns[statusIdx] = " " + status + " "
		lines[i] = strings.Join(columns, "|")
		changed = true
		break
	}
	if !changed {
		return fmt.Errorf("force task status %q for task %s at %q: task row not found", status, taskID, path)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("force task status %q for task %s at %q: %w", status, taskID, path, err)
	}
	return nil
}

type RoundEvidenceSink interface {
	MkdirAll(path string) error
	Exists(path string) bool
	WriteFile(path string, data []byte) error
}

type RoundEvidenceWriter struct {
	baseDir string
	sink    RoundEvidenceSink
}

func NewRoundEvidenceWriter(baseDir string) *RoundEvidenceWriter {
	return &RoundEvidenceWriter{baseDir: strings.TrimSpace(baseDir)}
}

func NewRoundEvidenceWriterWithSink(baseDir string, sink RoundEvidenceSink) *RoundEvidenceWriter {
	return &RoundEvidenceWriter{baseDir: strings.TrimSpace(baseDir), sink: sink}
}

func (w *RoundEvidenceWriter) Dir(round int) string {
	if w == nil || w.baseDir == "" {
		return ""
	}
	return NewCatalog().roundReviewEvidenceDir(w.baseDir, round)
}

func (w *RoundEvidenceWriter) Write(round int, content string) (string, error) {
	dir := w.Dir(round)
	if dir == "" {
		return "", nil
	}
	if w.sink == nil {
		return NewCatalog().writeRoundReviewEvidence(dir, content)
	}
	path := filepath.Join(dir, roundReviewEvidenceFile)
	if w.sink.Exists(path) {
		return "", fmt.Errorf("write round review evidence at %q: %w", path, fs.ErrExist)
	}
	if err := w.sink.MkdirAll(dir); err != nil {
		return "", fmt.Errorf("write round review evidence at %q: %w", path, err)
	}
	if err := w.sink.WriteFile(path, []byte(content)); err != nil {
		return "", fmt.Errorf("write round review evidence at %q: %w", path, err)
	}
	return path, nil
}

type ReviewerAdapter struct {
	runner    *ACPRunner
	baseJob   Job
	cutPoints cutPointHistory
}

func NewReviewerAdapter(runner *ACPRunner, baseJob Job) *ReviewerAdapter {
	return NewReviewerAdapterWithRepository(runner, baseJob, NewRepositoryAdapter(baseJob.WorkDir))
}

func NewReviewerAdapterWithRepository(runner *ACPRunner, baseJob Job, repository *RepositoryAdapter) *ReviewerAdapter {
	return &ReviewerAdapter{runner: runner, baseJob: baseJob, cutPoints: repository}
}

func (a *ReviewerAdapter) Review(ctx context.Context, request approval.ReviewRequest) (approval.ReviewerOutput, error) {
	job := a.baseJob
	job.AutoReview = false

	skillBody, err := a.runner.readReviewSkill(job.WorkDir)
	if err != nil {
		return approval.ReviewerOutput{}, err
	}
	job.Prompt = NewCatalog().buildReviewPrompt(skillBody, request.Target().String())

	evidenceWriter := NewRoundEvidenceWriter(a.baseJob.EvidenceDir)
	if roundDir := evidenceWriter.Dir(request.Round()); roundDir != "" {
		job.EvidenceDir = roundDir
	}

	restoreEnv := NewCatalog().applyRoundReviewEnv(request.Round(), a.priorCutPoint(request.Round()))
	rawText, err := a.runner.spawnReviewSession(ctx, job)
	restoreEnv()
	if err != nil {
		return approval.ReviewerOutput{}, err
	}
	if _, err := evidenceWriter.Write(request.Round(), rawText); err != nil {
		return approval.ReviewerOutput{}, err
	}
	criteriaMap, err := approval.ParseCriteriaMap(rawText, request)
	if err != nil {
		return approval.ReviewerOutput{}, err
	}
	return approval.NewReviewerOutput(rawText, parseCycleFindings(rawText), criteriaMap), nil
}

func parseCycleFindings(rawText string) []approval.Finding {
	return approval.ParseReviewFindings(rawText)
}

func (a *ReviewerAdapter) priorCutPoint(round int) string {
	if round < 2 || a.cutPoints == nil {
		return ""
	}
	checkpoint, ok := a.cutPoints.CheckpointAt(round - 1)
	if !ok {
		return ""
	}
	return checkpoint.String()
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

	restoreDepth := invocation.NewGuard().ResetDepth()
	defer restoreDepth()

	_, err := a.runner.Run(ctx, job)
	return err
}

func (a *FixerAdapter) buildFixPrompt(request approval.FixRequest) string {
	var sb strings.Builder
	sb.WriteString("## Findings to fix\n\n")
	for finding := range request.Findings() {
		fmt.Fprintf(&sb, "- [%s] %s (%s): %s\n", finding.Severity().String(), finding.Location(), finding.Rule(), finding.Description())
	}
	sb.WriteString("\n## Target\n\n```diff\n")
	sb.WriteString(request.Target().String())
	sb.WriteString("\n```\n")
	return sb.String()
}
