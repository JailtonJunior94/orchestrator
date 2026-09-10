package taskloop

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	taskfs "github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func newGitRepoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init")
	run("config", "user.email", "cycle@example.com")
	run("config", "user.name", "Cycle")
	run("commit", "--allow-empty", "-m", "base")
	return dir
}

func setupCycleFS(t *testing.T, criteria string) (*taskfs.FakeFileSystem, string, string) {
	t.Helper()
	repo := newGitRepoDir(t)
	prd := filepath.Join(repo, ".specs", "prd-cycle")

	fsys := taskfs.NewFakeFileSystem()
	fsys.Files[filepath.Join(repo, "AGENTS.md")] = []byte("# Agents\n")
	fsys.Files[filepath.Join(prd, "prd.md")] = []byte("# PRD\n")
	fsys.Files[filepath.Join(prd, "techspec.md")] = []byte("# TechSpec\n")
	fsys.Files[filepath.Join(prd, "task-1.0-test.md")] = fmt.Appendf(nil,
		"**Status:** pending\n\n## Critérios de Sucesso\n\n%s\n", criteria)
	fsys.Files[filepath.Join(prd, "tasks.md")] = tasksContent("1.0", "Cycle Task", "pending")

	return fsys, prd, repo
}

func cycleOptions(prd string) Options {
	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")
	return Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       10 * time.Second,
		ReportPath:    filepath.Join(prd, "report.md"),
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}
}

func TestExecuteConductsApprovalCycleFromTaskCriteria(t *testing.T) {
	fsys, prd, _ := setupCycleFS(t, "- [ ] build verde\n- [ ] testes passam")

	var reviewerCalls int
	var reviewerPrompt string
	var executorCalls int

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{binary: "claude", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				executorCalls++
				fsys.Files[filepath.Join(prd, "tasks.md")] = tasksContent("1.0", "Cycle Task", "done")
				return "executor output", "", 0, nil
			}}, nil
		case "codex":
			return &callbackInvoker{binary: "codex", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				reviewerCalls++
				reviewerPrompt = prompt
				return "no findings\n\nVerdict: APPROVED\n", "", 0, nil
			}}, nil
		default:
			return nil, fmt.Errorf("tool not configured in test: %s", tool)
		}
	}

	if err := svc.Execute(cycleOptions(prd)); err != nil {
		t.Fatalf("Execute unexpected error: %v", err)
	}

	if reviewerCalls != 1 {
		t.Fatalf("reviewer called %d times, want 1", reviewerCalls)
	}
	if executorCalls != 1 {
		t.Fatalf("executor called %d times, want 1 (no bugfix on approved path)", executorCalls)
	}
	if !strings.Contains(reviewerPrompt, "Diff consolidado") {
		t.Errorf("reviewer did not receive the cycle consolidated prompt:\n%s", reviewerPrompt)
	}

	reportStr := readFileString(t, fsys, filepath.Join(prd, "report.md"))
	if !strings.Contains(reportStr, "reviewer") {
		t.Errorf("report missing reviewer row:\n%s", reportStr)
	}
	if strings.Contains(reportStr, "| bugfix |") {
		t.Errorf("report should not contain a bugfix row on the approved path:\n%s", reportStr)
	}
}

func TestExecuteApprovalCycleRunsReviewAndFixInFreshSessions(t *testing.T) {
	fsys, prd, _ := setupCycleFS(t, "- [ ] regra de negocio coberta")

	var reviewerCalls int
	var executorPrompts []string

	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{binary: "claude", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				executorPrompts = append(executorPrompts, prompt)
				if len(executorPrompts) == 1 {
					fsys.Files[filepath.Join(prd, "tasks.md")] = tasksContent("1.0", "Cycle Task", "done")
					return "executor output", "", 0, nil
				}
				return "root cause: nil deref\nFail-before: go test ./... FAIL\nPass-after: go test ./... ok\n", "", 0, nil
			}}, nil
		case "codex":
			return &callbackInvoker{binary: "codex", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				reviewerCalls++
				if reviewerCalls == 1 {
					return "[Critical] [main.go:10] variavel nao inicializada\n\nVerdict: REJECTED\n", "", 1, nil
				}
				return "clean\n\nVerdict: APPROVED\n", "", 0, nil
			}}, nil
		default:
			return nil, fmt.Errorf("tool not configured in test: %s", tool)
		}
	}

	if err := svc.Execute(cycleOptions(prd)); err != nil {
		t.Fatalf("Execute unexpected error: %v", err)
	}

	if reviewerCalls != 2 {
		t.Fatalf("reviewer called %d times, want 2 (round + re-review in a fresh session)", reviewerCalls)
	}
	if len(executorPrompts) != 2 {
		t.Fatalf("executor called %d times, want 2 (execution + fix in a fresh session)", len(executorPrompts))
	}
	if !strings.Contains(executorPrompts[1], "skill bugfix") {
		t.Errorf("fix prompt does not reference the bugfix skill:\n%s", executorPrompts[1])
	}
	if !strings.Contains(executorPrompts[1], "variavel nao inicializada") {
		t.Errorf("fix prompt did not receive the review findings:\n%s", executorPrompts[1])
	}

	reportStr := readFileString(t, fsys, filepath.Join(prd, "report.md"))
	if !strings.Contains(reportStr, "| bugfix |") {
		t.Errorf("report missing bugfix row after fix:\n%s", reportStr)
	}
}

func TestExecuteFallsBackToLegacyReviewWhenTaskFileHasNoCriteria(t *testing.T) {
	fsys, prd := setupBaseFS("pending")

	var reviewerPrompt string
	svc := NewService(fsys, newTestPrinter())
	svc.binaryChecker = noBinaryCheck
	svc.invokerFactory = func(tool string) (AgentInvoker, error) {
		switch tool {
		case "claude":
			return &callbackInvoker{binary: "claude", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				fsys.Files[prd+"/tasks.md"] = tasksContent("1.0", "Test Task", "done")
				return "executor output", "", 0, nil
			}}, nil
		case "codex":
			return &callbackInvoker{binary: "codex", fn: func(ctx context.Context, prompt, workDir, model string) (string, string, int, error) {
				reviewerPrompt = prompt
				return "approved", "", 0, nil
			}}, nil
		default:
			return nil, fmt.Errorf("tool not configured in test: %s", tool)
		}
	}

	execProfile, _ := NewExecutionProfile("executor", "claude", "")
	revProfile, _ := NewExecutionProfile("reviewer", "codex", "")
	opts := Options{
		PRDFolder:     prd,
		MaxIterations: 5,
		Timeout:       5 * time.Second,
		ReportPath:    prd + "/report.md",
		Profiles: &ProfileConfig{
			Mode:     "avancado",
			Executor: execProfile,
			Reviewer: &revProfile,
		},
		AllowUnknownModel: true,
	}

	if err := svc.Execute(opts); err != nil {
		t.Fatalf("Execute unexpected error: %v", err)
	}
	if strings.Contains(reviewerPrompt, "Diff consolidado") {
		t.Errorf("expected legacy review prompt, got the cycle consolidated one:\n%s", reviewerPrompt)
	}
}

func readFileString(t *testing.T, fsys *taskfs.FakeFileSystem, path string) string {
	t.Helper()
	data, err := fsys.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
