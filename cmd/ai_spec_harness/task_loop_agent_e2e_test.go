package aispecharness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

func writeAgentFixture(t *testing.T, root, name, ide string) {
	t.Helper()
	dir := filepath.Join(root, ".ai-harness", "agents", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := "---\nname: " + name + "\ndescription: agente de teste end-to-end\nversion: 1.0.0\nruntime:\n  ide: " + ide + "\n---\n\ncorpo\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENT.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile AGENT.md: %v", err)
	}
}

func writePRDFixture(t *testing.T, root string) string {
	t.Helper()
	prdDir := filepath.Join(root, "prd-e2e")
	if err := os.MkdirAll(prdDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	tasks := "| ID | Titulo | Status | Dependencias | Paralelizavel |\n" +
		"|----|--------|--------|--------------|---------------|\n" +
		"| 1.0 | Tarefa de teste | pending | - | nao |\n"
	files := map[string]string{
		"tasks.md":    tasks,
		"prd.md":      "# prd\n",
		"techspec.md": "# techspec\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(prdDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	return prdDir
}

func TestTaskLoopCLI_AgentModeReachesAgentResolution(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	writeAgentFixture(t, root, "e2e-agent", "claude")
	prdDir := writePRDFixture(t, root)

	err := runTaskLoopArgs(t, "--agent", "e2e-agent", "--dry-run", "--skip-drift-guard", prdDir)

	if errors.Is(err, taskloop.ErrToolInvalida) {
		t.Fatalf("--agent nao deve cair em ResolveProfiles com tool vazia: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "ferramenta nao suportada") {
		t.Fatalf("--agent nao deve reportar ferramenta nao suportada: %v", err)
	}
	if err != nil {
		t.Fatalf("--agent com runtime.ide valido deveria executar em dry-run, obteve: %v", err)
	}
}

func TestTaskLoopCLI_AgentModeRemovedIDEIsTyped(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("MkdirAll .git: %v", err)
	}
	writeAgentFixture(t, root, "legacy-agent", "gemini")
	prdDir := writePRDFixture(t, root)

	err := runTaskLoopArgs(t, "--agent", "legacy-agent", "--dry-run", "--skip-drift-guard", prdDir)

	if err == nil {
		t.Fatal("esperado erro para agente com runtime.ide gemini")
	}
	var removed *skills.RemovedAgentError
	if !errors.As(err, &removed) {
		t.Fatalf("erro deve ser *skills.RemovedAgentError, obteve %T: %v", err, err)
	}
	if removed.Agent != "gemini" {
		t.Errorf("RemovedAgentError.Agent = %q; want gemini", removed.Agent)
	}
}
