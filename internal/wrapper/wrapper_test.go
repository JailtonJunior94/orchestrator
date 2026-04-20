package wrapper_test

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/wrapper"
)

const projectDir = "/project"

func setupFS(files []string, dirs []string) *fs.FakeFileSystem {
	fake := fs.NewFakeFileSystem()
	fake.Dirs[projectDir] = true
	for _, f := range files {
		fake.Files[f] = []byte{}
	}
	for _, d := range dirs {
		fake.Dirs[d] = true
	}
	return fake
}

// fullFS monta um fake filesystem com todos os artefatos necessários para o cenário feliz.
// skill deve ser "go-implementation" (requer go.mod).
func fullFS() *fs.FakeFileSystem {
	return setupFS(
		[]string{
			projectDir + "/AGENTS.md",
			projectDir + "/go.mod",
		},
		[]string{
			projectDir + "/.agents/skills/agent-governance",
		},
	)
}

// --- Cenário feliz ---

func TestExecute_HappyPath_Codex(t *testing.T) {
	t.Parallel()
	fsys := fullFS()
	instruction, err := wrapper.Execute("codex", "go-implementation", projectDir, nil, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if instruction == "" {
		t.Error("expected non-empty instruction")
	}
	if !strings.Contains(instruction, "codex") {
		t.Errorf("expected instruction to mention 'codex', got: %s", instruction)
	}
}

func TestExecute_HappyPath_Gemini(t *testing.T) {
	t.Parallel()
	fsys := fullFS()
	instruction, err := wrapper.Execute("gemini", "go-implementation", projectDir, nil, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(instruction, "gemini") {
		t.Errorf("expected instruction to mention 'gemini', got: %s", instruction)
	}
}

func TestExecute_HappyPath_Copilot(t *testing.T) {
	t.Parallel()
	fsys := fullFS()
	instruction, err := wrapper.Execute("copilot", "go-implementation", projectDir, nil, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(instruction, "copilot") || !strings.Contains(strings.ToLower(instruction), "copilot") {
		t.Errorf("expected instruction to mention 'copilot', got: %s", instruction)
	}
}

func TestExecute_HappyPath_WithArgs(t *testing.T) {
	t.Parallel()
	fsys := fullFS()
	instruction, err := wrapper.Execute("codex", "go-implementation", projectDir, []string{"--verbose", "--timeout=30"}, fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(instruction, "--verbose") {
		t.Errorf("expected instruction to include extra args, got: %s", instruction)
	}
}

// --- Ferramenta inválida ---

func TestExecute_InvalidTool(t *testing.T) {
	t.Parallel()
	fsys := fullFS()
	_, err := wrapper.Execute("claude", "go-implementation", projectDir, nil, fsys)
	if err == nil {
		t.Error("expected error for invalid tool 'claude'")
	}
	if !strings.Contains(err.Error(), "ferramenta invalida") {
		t.Errorf("expected 'ferramenta invalida' in error, got: %v", err)
	}
}

func TestExecute_UnknownTool(t *testing.T) {
	t.Parallel()
	fsys := fullFS()
	_, err := wrapper.Execute("unknown-tool", "go-implementation", projectDir, nil, fsys)
	if err == nil {
		t.Error("expected error for unknown tool")
	}
}

// --- AGENTS.md ausente ---

func TestExecute_MissingAgentsMD(t *testing.T) {
	t.Parallel()
	fsys := setupFS(
		[]string{projectDir + "/go.mod"},
		[]string{projectDir + "/.agents/skills/agent-governance"},
	)
	_, err := wrapper.Execute("codex", "go-implementation", projectDir, nil, fsys)
	if err == nil {
		t.Error("expected error when AGENTS.md is missing")
	}
	if !strings.Contains(err.Error(), "AGENTS.md") {
		t.Errorf("expected error to mention 'AGENTS.md', got: %v", err)
	}
}

// --- agent-governance ausente ---

func TestExecute_MissingAgentGovernance(t *testing.T) {
	t.Parallel()
	fsys := setupFS(
		[]string{
			projectDir + "/AGENTS.md",
			projectDir + "/go.mod",
		},
		[]string{},
	)
	_, err := wrapper.Execute("codex", "go-implementation", projectDir, nil, fsys)
	if err == nil {
		t.Error("expected error when agent-governance dir is missing")
	}
	if !strings.Contains(err.Error(), "agent-governance") {
		t.Errorf("expected error to mention 'agent-governance', got: %v", err)
	}
}

// --- Prerequisites falham ---

func TestExecute_PrerequisitesFail(t *testing.T) {
	t.Parallel()
	// go-implementation requer go.mod — não fornecemos
	fsys := setupFS(
		[]string{projectDir + "/AGENTS.md"},
		[]string{projectDir + "/.agents/skills/agent-governance"},
	)
	_, err := wrapper.Execute("codex", "go-implementation", projectDir, nil, fsys)
	if err == nil {
		t.Error("expected error when prerequisites are not met")
	}
	if !strings.Contains(err.Error(), "pre-requisitos") {
		t.Errorf("expected error to mention 'pre-requisitos', got: %v", err)
	}
}

func TestExecute_UnknownSkill(t *testing.T) {
	t.Parallel()
	fsys := setupFS(
		[]string{projectDir + "/AGENTS.md"},
		[]string{projectDir + "/.agents/skills/agent-governance"},
	)
	_, err := wrapper.Execute("codex", "nonexistent-skill", projectDir, nil, fsys)
	if err == nil {
		t.Error("expected error for unknown skill")
	}
}

// --- Budget excedido ---

func TestExecute_BudgetExceeded_Copilot(t *testing.T) {
	t.Parallel()
	// copilot tem limite de 2000 tokens — populamos AGENTS.md com conteúdo grande
	bigContent := strings.Repeat("palavra ", 3000) // ~3000 palavras × ~5 chars = 15000 chars → ~4285 tokens > 2000
	fsys := setupFS(
		[]string{projectDir + "/go.mod"},
		[]string{projectDir + "/.agents/skills/agent-governance"},
	)
	fsys.Files[projectDir+"/AGENTS.md"] = []byte(bigContent)

	_, err := wrapper.Execute("copilot", "go-implementation", projectDir, nil, fsys)
	if err == nil {
		t.Error("expected error when budget is exceeded for copilot")
	}
	if !strings.Contains(err.Error(), "budget excedido") {
		t.Errorf("expected 'budget excedido' in error, got: %v", err)
	}
}

// --- ValidTools ---

func TestValidTools_ContainsExpected(t *testing.T) {
	t.Parallel()
	expected := []string{"codex", "gemini", "copilot"}
	for _, tool := range expected {
		if !wrapper.ValidTools[tool] {
			t.Errorf("expected tool %q to be in ValidTools", tool)
		}
	}
	if wrapper.ValidTools["claude"] {
		t.Error("claude should not be in ValidTools (uses hooks)")
	}
}
