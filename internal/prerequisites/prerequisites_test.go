package prerequisites_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/prerequisites"
)

const projectDir = "/project"

func setupFS(files ...string) *fs.FakeFileSystem {
	fake := fs.NewFakeFileSystem()
	fake.Dirs[projectDir] = true
	for _, f := range files {
		fake.Files[f] = []byte{}
	}
	return fake
}

// --- go-implementation ---

func TestGoImplementation_WithGoMod(t *testing.T) {
	fsys := setupFS(projectDir + "/go.mod")
	passed, results, err := prerequisites.Verify("go-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when go.mod exists")
	}
	if len(results) != 1 || !results[0].Found {
		t.Error("expected result[0].Found=true")
	}
}

func TestGoImplementation_WithGoWork(t *testing.T) {
	fsys := setupFS(projectDir + "/go.work")
	passed, _, err := prerequisites.Verify("go-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when go.work exists")
	}
}

func TestGoImplementation_Missing(t *testing.T) {
	fsys := setupFS()
	passed, results, err := prerequisites.Verify("go-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when go.mod/go.work absent")
	}
	if len(results) != 1 || results[0].Found {
		t.Error("expected result[0].Found=false")
	}
}

// --- node-implementation ---

func TestNodeImplementation_Pass(t *testing.T) {
	fsys := setupFS(projectDir + "/package.json")
	passed, _, err := prerequisites.Verify("node-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when package.json exists")
	}
}

func TestNodeImplementation_Fail(t *testing.T) {
	fsys := setupFS()
	passed, _, err := prerequisites.Verify("node-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when package.json absent")
	}
}

// --- python-implementation ---

func TestPythonImplementation_PyprojectToml(t *testing.T) {
	fsys := setupFS(projectDir + "/pyproject.toml")
	passed, _, err := prerequisites.Verify("python-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when pyproject.toml exists")
	}
}

func TestPythonImplementation_SetupPy(t *testing.T) {
	fsys := setupFS(projectDir + "/setup.py")
	passed, _, err := prerequisites.Verify("python-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when setup.py exists")
	}
}

func TestPythonImplementation_RequirementsTxt(t *testing.T) {
	fsys := setupFS(projectDir + "/requirements.txt")
	passed, _, err := prerequisites.Verify("python-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when requirements.txt exists")
	}
}

func TestPythonImplementation_Fail(t *testing.T) {
	fsys := setupFS()
	passed, _, err := prerequisites.Verify("python-implementation", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when no python files exist")
	}
}

// --- create-tasks ---

func TestCreateTasks_BothPresent(t *testing.T) {
	fsys := setupFS(projectDir+"/prd.md", projectDir+"/techspec.md")
	passed, results, err := prerequisites.Verify("create-tasks", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when prd.md and techspec.md exist")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestCreateTasks_MissingPrd(t *testing.T) {
	fsys := setupFS(projectDir + "/techspec.md")
	passed, _, err := prerequisites.Verify("create-tasks", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when prd.md is missing")
	}
}

func TestCreateTasks_MissingTechspec(t *testing.T) {
	fsys := setupFS(projectDir + "/prd.md")
	passed, _, err := prerequisites.Verify("create-tasks", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when techspec.md is missing")
	}
}

func TestCreateTasks_BothMissing(t *testing.T) {
	fsys := setupFS()
	passed, _, err := prerequisites.Verify("create-tasks", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when both files are missing")
	}
}

// --- execute-task ---

func TestExecuteTask_Pass(t *testing.T) {
	fsys := setupFS(projectDir + "/tasks.md")
	passed, _, err := prerequisites.Verify("execute-task", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when tasks.md exists")
	}
}

func TestExecuteTask_Fail(t *testing.T) {
	fsys := setupFS()
	passed, _, err := prerequisites.Verify("execute-task", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when tasks.md is missing")
	}
}

// --- create-technical-specification ---

func TestCreateTechnicalSpecification_Pass(t *testing.T) {
	fsys := setupFS(projectDir + "/prd.md")
	passed, _, err := prerequisites.Verify("create-technical-specification", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true when prd.md exists")
	}
}

func TestCreateTechnicalSpecification_Fail(t *testing.T) {
	fsys := setupFS()
	passed, _, err := prerequisites.Verify("create-technical-specification", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if passed {
		t.Error("expected passed=false when prd.md is missing")
	}
}

// --- bugfix (optional) ---

func TestBugfix_WithBugsJson(t *testing.T) {
	fsys := setupFS(projectDir + "/bugs.json")
	passed, results, err := prerequisites.Verify("bugfix", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !passed {
		t.Error("expected passed=true (bugfix has no required files)")
	}
	if len(results) != 1 || !results[0].Found {
		t.Error("expected optional bugs.json to be found")
	}
}

func TestBugfix_WithoutBugsJson(t *testing.T) {
	fsys := setupFS()
	passed, results, err := prerequisites.Verify("bugfix", projectDir, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// bugfix has no required files — should still pass
	if !passed {
		t.Error("expected passed=true even when bugs.json is absent")
	}
	if len(results) != 1 || results[0].Found {
		t.Error("expected optional result with Found=false")
	}
	if !results[0].Optional {
		t.Error("expected result to be optional")
	}
}

// --- error cases ---

func TestUnknownSkill(t *testing.T) {
	fsys := setupFS()
	_, _, err := prerequisites.Verify("nonexistent-skill", projectDir, fsys)
	if err == nil {
		t.Error("expected error for unknown skill")
	}
}

func TestDirectoryNotFound(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	_, _, err := prerequisites.Verify("go-implementation", "/nonexistent", fsys)
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

// --- KnownSkills ---

func TestKnownSkills(t *testing.T) {
	skills := prerequisites.KnownSkills()
	expected := []string{
		"go-implementation",
		"node-implementation",
		"python-implementation",
		"create-tasks",
		"execute-task",
		"create-technical-specification",
		"bugfix",
	}
	if len(skills) != len(expected) {
		t.Errorf("expected %d known skills, got %d", len(expected), len(skills))
	}
}
