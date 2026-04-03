package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jailtonjunior/orchestrator/internal/bootstrap"
	installapp "github.com/jailtonjunior/orchestrator/internal/install/application"
	install "github.com/jailtonjunior/orchestrator/internal/install/domain"
	"github.com/jailtonjunior/orchestrator/internal/runtime"
	"github.com/jailtonjunior/orchestrator/internal/runtime/domain"
	"github.com/jailtonjunior/orchestrator/internal/state"
	"github.com/jailtonjunior/orchestrator/internal/workflows"
)

func TestListCommand(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	cmd := NewListCommand(&bootstrap.App{
		Runtime: fakeRuntimeService{workflows: []string{"dev-workflow"}},
	})
	cmd.SetOut(out)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "dev-workflow\n" {
		t.Fatalf("out = %q", out.String())
	}
}

func TestRunCommandFlags(t *testing.T) {
	t.Parallel()

	cmd := NewRunCommand(&bootstrap.App{Runtime: fakeRuntimeService{}})
	cmd.SetArgs([]string{"dev-workflow"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing input error")
	}

	cmd = NewRunCommand(&bootstrap.App{Runtime: fakeRuntimeService{}})
	cmd.SetArgs([]string{"dev-workflow", "--input", "a", "--file", "b"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected mutual exclusion error")
	}
}

func TestRunCommandTranslatesWorkflowNotFound(t *testing.T) {
	t.Parallel()

	cmd := NewRunCommand(&bootstrap.App{Runtime: fakeRuntimeService{runErr: workflows.ErrWorkflowNotFound}})
	cmd.SetArgs([]string{"missing", "--input", "test"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "orq list") {
		t.Fatalf("error = %v", err)
	}
}

func TestContinueCommandTranslatesNoPendingRuns(t *testing.T) {
	t.Parallel()

	cmd := NewContinueCommand(&bootstrap.App{Runtime: fakeRuntimeService{continueErr: state.ErrNoPendingRuns}})
	cmd.SetArgs(nil)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), ".orq/runs") {
		t.Fatalf("error = %v", err)
	}
}

func TestInstallCommandRendersPreviewAndExecutesAfterConfirmation(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	service := &fakeInstallService{
		previewResult: &installapp.OperationPreview{
			Operation:     install.OperationInstall,
			Scope:         install.ScopeProject,
			InventoryPath: ".orq/install/inventory.json",
			Plan:          mustPlan(t, install.OperationInstall, install.ScopeProject, nil, mustChange(t, install.ProviderClaude, install.ScopeProject, "claude:skill:reviewer", install.ActionInstall, ".claude/skills/reviewer/SKILL.md", ".codex/skills/reviewer/SKILL.md")),
		},
		installResult: &installapp.OperationResult{
			Operation: install.OperationInstall,
			Scope:     install.ScopeProject,
			Providers: []installapp.ProviderResult{{
				Provider:           install.ProviderClaude,
				PlannedChangeCount: 1,
				AppliedChangeCount: 1,
				Verification:       install.VerificationStatusPartial,
				InventorySaved:     true,
			}},
		},
	}

	cmd := NewInstallCommand(&bootstrap.App{Install: service})
	cmd.SetOut(out)
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetArgs([]string{"--provider", "claude", "--asset", "reviewer"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	rendered := out.String()
	if !strings.Contains(rendered, "Operação: install") {
		t.Fatalf("preview missing from output: %q", rendered)
	}
	if !strings.Contains(rendered, "Confirmar operação?") {
		t.Fatalf("confirmation prompt missing from output: %q", rendered)
	}
	if !strings.Contains(rendered, "Resultado: install (project)") {
		t.Fatalf("result missing from output: %q", rendered)
	}
	if service.installReq.Scope != install.ScopeProject {
		t.Fatalf("install scope = %q", service.installReq.Scope)
	}
	if !service.installReq.Interactive {
		t.Fatal("install request must be interactive when --yes is absent")
	}
}

func TestInstallCommandStopsOnConflictsWhenPolicyIsAbort(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	service := &fakeInstallService{
		previewResult: &installapp.OperationPreview{
			Operation:     install.OperationInstall,
			Scope:         install.ScopeProject,
			InventoryPath: ".orq/install/inventory.json",
			Plan: mustPlan(t, install.OperationInstall, install.ScopeProject, []install.Conflict{{
				Provider:   install.ProviderClaude,
				AssetID:    "claude:command:review",
				TargetPath: filepath.Join("repo", ".claude", "commands", "review.md"),
				Reason:     "target already exists",
			}}, mustChange(t, install.ProviderClaude, install.ScopeProject, "claude:command:review", install.ActionSkip, ".claude/commands/review.md", filepath.Join("repo", ".claude", "commands", "review.md"))),
		},
	}

	cmd := NewInstallCommand(&bootstrap.App{Install: service})
	cmd.SetOut(out)
	cmd.SetArgs([]string{"--provider", "claude"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if service.installCalled {
		t.Fatal("install must not run when abort policy finds conflicts")
	}
	if !strings.Contains(err.Error(), "--conflict skip") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(out.String(), "Conflitos:") {
		t.Fatalf("conflicts missing from preview output: %q", out.String())
	}
}

func TestInstallUpdateCommandMapsFlagsToDTOs(t *testing.T) {
	t.Parallel()

	service := &fakeInstallService{
		previewResult: &installapp.OperationPreview{
			Operation:     install.OperationUpdate,
			Scope:         install.ScopeGlobal,
			InventoryPath: "/home/user/.local/state/orq/install/inventory.json",
			Plan:          mustPlan(t, install.OperationUpdate, install.ScopeGlobal, nil, mustChange(t, install.ProviderCodex, install.ScopeGlobal, "claude:skill:reviewer", install.ActionUpdate, ".claude/skills/reviewer/SKILL.md", "/home/user/.codex/skills/reviewer/SKILL.md")),
		},
		updateResult: &installapp.OperationResult{
			Operation: install.OperationUpdate,
			Scope:     install.ScopeGlobal,
		},
	}

	cmd := NewInstallCommand(&bootstrap.App{Install: service})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"update", "--global", "--provider", "codex", "--asset", "reviewer", "--kind", "skill", "--conflict", "overwrite", "--yes"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if service.previewReq.Operation != install.OperationUpdate {
		t.Fatalf("preview operation = %q", service.previewReq.Operation)
	}
	if service.updateReq.Scope != install.ScopeGlobal {
		t.Fatalf("update scope = %q", service.updateReq.Scope)
	}
	if len(service.updateReq.Providers) != 1 || service.updateReq.Providers[0] != install.ProviderCodex {
		t.Fatalf("update providers = %#v", service.updateReq.Providers)
	}
	if len(service.updateReq.AssetKinds) != 1 || service.updateReq.AssetKinds[0] != install.AssetKindSkill {
		t.Fatalf("update kinds = %#v", service.updateReq.AssetKinds)
	}
	if service.updateReq.ConflictPolicy != install.ConflictPolicyOverwrite {
		t.Fatalf("update conflict policy = %q", service.updateReq.ConflictPolicy)
	}
	if service.updateReq.Interactive {
		t.Fatal("update request must be non-interactive when --yes is set")
	}
}

func TestInstallListCommandRendersInventoryView(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	cmd := NewInstallCommand(&bootstrap.App{Install: &fakeInstallService{
		listResult: &installapp.InventoryView{
			Scope:         install.ScopeProject,
			InventoryPath: ".orq/install/inventory.json",
			Items: []installapp.InventoryItem{{
				Provider:     install.ProviderClaude,
				Name:         "reviewer",
				Kind:         install.AssetKindSkill,
				Managed:      true,
				Verification: install.VerificationStatusComplete,
				TargetPath:   ".claude/skills/reviewer/SKILL.md",
			}},
		},
	}})
	cmd.SetOut(out)
	cmd.SetArgs([]string{"list", "--provider", "claude"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "reviewer (skill)") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestInstallVerifyCommandRendersVerificationReport(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	cmd := NewInstallCommand(&bootstrap.App{Install: &fakeInstallService{
		verifyResult: &installapp.VerificationReport{
			Scope:         install.ScopeProject,
			InventoryPath: ".orq/install/inventory.json",
			Providers: []installapp.VerificationProviderReport{{
				Provider:       install.ProviderGemini,
				Status:         install.VerificationStatusPartial,
				InventorySaved: true,
				Details:        []string{"functional verification is not implemented for this provider"},
			}},
		},
	}})
	cmd.SetOut(out)
	cmd.SetArgs([]string{"verify", "--provider", "gemini"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "status=partial") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestRenderOperationPreviewIncludesConflicts(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	preview := &installapp.OperationPreview{
		Operation:     install.OperationRemove,
		Scope:         install.ScopeProject,
		InventoryPath: ".orq/install/inventory.json",
		Plan: mustPlan(t, install.OperationRemove, install.ScopeProject, []install.Conflict{{
			Provider:   install.ProviderCopilot,
			AssetID:    "copilot:instruction:agents",
			TargetPath: "AGENTS.md",
			Reason:     "managed asset differs from current target",
		}}, mustChange(t, install.ProviderCopilot, install.ScopeProject, "copilot:instruction:agents", install.ActionRemove, "AGENTS.md", "AGENTS.md")),
	}

	if err := renderOperationPreview(out, preview); err != nil {
		t.Fatal(err)
	}
	rendered := out.String()
	if !strings.Contains(rendered, "Resumo:") || !strings.Contains(rendered, "Conflitos:") {
		t.Fatalf("rendered preview = %q", rendered)
	}
}

func TestTranslateErrorProviderPath(t *testing.T) {
	t.Parallel()

	err := translateError(errors.New(`provider "claude" binary "claude" not found in PATH`))
	if !strings.Contains(err.Error(), "Instale o CLI") {
		t.Fatalf("error = %v", err)
	}
}

type fakeRuntimeService struct {
	runErr      error
	continueErr error
	workflows   []string
}

func (f fakeRuntimeService) Run(context.Context, string, string) (*runtime.RunResult, error) {
	if f.runErr != nil {
		return nil, f.runErr
	}
	wf, _ := domain.NewWorkflowName("dev-workflow")
	stepName, _ := domain.NewStepName("prd")
	providerName, _ := domain.NewProviderName("claude")
	run, _ := domain.NewRun("id", wf, "input", []domain.StepDefinition{{
		Name:     stepName,
		Provider: providerName,
		Input:    "input",
	}}, mustTime())
	return &runtime.RunResult{Run: run}, nil
}

func (f fakeRuntimeService) Continue(context.Context, string) (*runtime.RunResult, error) {
	return nil, f.continueErr
}

func (f fakeRuntimeService) ListWorkflows(context.Context) ([]string, error) {
	if len(f.workflows) == 0 {
		return []string{"dev-workflow"}, nil
	}
	return f.workflows, nil
}

type fakeInstallService struct {
	previewReq    installapp.PreviewRequest
	installReq    installapp.OperationRequest
	updateReq     installapp.OperationRequest
	removeReq     installapp.OperationRequest
	listReq       installapp.ListRequest
	verifyReq     installapp.VerifyRequest
	previewCalled bool
	installCalled bool
	updateCalled  bool
	removeCalled  bool
	listCalled    bool
	verifyCalled  bool
	previewResult *installapp.OperationPreview
	installResult *installapp.OperationResult
	updateResult  *installapp.OperationResult
	removeResult  *installapp.OperationResult
	listResult    *installapp.InventoryView
	verifyResult  *installapp.VerificationReport
	previewErr    error
	installErr    error
	updateErr     error
	removeErr     error
	listErr       error
	verifyErr     error
}

func (f *fakeInstallService) Preview(_ context.Context, req installapp.PreviewRequest) (*installapp.OperationPreview, error) {
	f.previewCalled = true
	f.previewReq = req
	return f.previewResult, f.previewErr
}

func (f *fakeInstallService) Install(_ context.Context, req installapp.InstallRequest) (*installapp.OperationResult, error) {
	f.installCalled = true
	f.installReq = req.OperationRequest
	return f.installResult, f.installErr
}

func (f *fakeInstallService) Update(_ context.Context, req installapp.UpdateRequest) (*installapp.OperationResult, error) {
	f.updateCalled = true
	f.updateReq = req.OperationRequest
	return f.updateResult, f.updateErr
}

func (f *fakeInstallService) Remove(_ context.Context, req installapp.RemoveRequest) (*installapp.OperationResult, error) {
	f.removeCalled = true
	f.removeReq = req.OperationRequest
	return f.removeResult, f.removeErr
}

func (f *fakeInstallService) List(_ context.Context, req installapp.ListRequest) (*installapp.InventoryView, error) {
	f.listCalled = true
	f.listReq = req
	return f.listResult, f.listErr
}

func (f *fakeInstallService) Verify(_ context.Context, req installapp.VerifyRequest) (*installapp.VerificationReport, error) {
	f.verifyCalled = true
	f.verifyReq = req
	return f.verifyResult, f.verifyErr
}

func mustPlan(t *testing.T, operation install.Operation, scope install.Scope, conflicts []install.Conflict, changes ...install.PlannedChange) *installapp.Plan {
	t.Helper()

	plan := &installapp.Plan{
		Operation: operation,
		Scope:     scope,
		Changes:   changes,
	}
	for _, conflict := range conflicts {
		for idx, change := range plan.Changes {
			if change.AssetID() != conflict.AssetID || change.Provider() != conflict.Provider {
				continue
			}
			updated, err := install.NewPlannedChange(
				change.Provider(),
				change.Scope(),
				change.AssetID(),
				change.Action(),
				change.SourcePath(),
				change.TargetPath(),
				change.ManagedPath(),
				&conflict,
				change.Verification(),
			)
			if err != nil {
				t.Fatalf("NewPlannedChange() error = %v", err)
			}
			plan.Changes[idx] = updated
		}
	}
	plan.Summary = installapp.PlanSummary{
		InstallCount:  countActions(changes, install.ActionInstall),
		UpdateCount:   countActions(changes, install.ActionUpdate),
		RemoveCount:   countActions(changes, install.ActionRemove),
		SkippedCount:  countActions(changes, install.ActionSkip),
		ConflictCount: len(conflicts),
	}
	return plan
}

func mustChange(t *testing.T, provider install.Provider, scope install.Scope, assetID string, action install.Action, sourcePath string, targetPath string) install.PlannedChange {
	t.Helper()

	change, err := install.NewPlannedChange(
		provider,
		scope,
		assetID,
		action,
		sourcePath,
		targetPath,
		targetPath,
		nil,
		install.VerificationStatusPartial,
	)
	if err != nil {
		t.Fatalf("NewPlannedChange() error = %v", err)
	}
	return change
}

func countActions(changes []install.PlannedChange, action install.Action) int {
	total := 0
	for _, change := range changes {
		if change.Action() == action {
			total++
		}
	}
	return total
}

func mustTime() time.Time {
	return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
}
