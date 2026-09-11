package adapters_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func newTestGenerator() (*adapters.Generator, *fs.FakeFileSystem) {
	fsys := fs.NewFakeFileSystem()
	buf := &bytes.Buffer{}
	p := &output.Printer{Out: buf, Err: buf, Verbose: true}
	return adapters.NewGenerator(fsys, p), fsys
}

// seedSkill plants a minimal SKILL.md for a given skill name.
func seedSkill(fsys *fs.FakeFileSystem, sourceDir, skill, description string) {
	path := filepath.Join(sourceDir, ".agents", "skills", skill, "SKILL.md")
	content := "---\ndescription: " + description + "\n---\n\nskill body"
	_ = fsys.WriteFile(path, []byte(content))
}

func TestBuildCodexConfig_empty(t *testing.T) {
	g, _ := newTestGenerator()
	result := g.BuildCodexConfig(nil)
	if result != "" {
		t.Errorf("BuildCodexConfig(nil) = %q, want empty", result)
	}
}

func TestBuildCodexConfig_skills(t *testing.T) {
	g, _ := newTestGenerator()
	result := g.BuildCodexConfig([]string{"bugfix", "review"})
	if !strings.Contains(result, "bugfix") {
		t.Error("BuildCodexConfig should contain 'bugfix'")
	}
	if !strings.Contains(result, "review") {
		t.Error("BuildCodexConfig should contain 'review'")
	}
	if !strings.Contains(result, "[[skills.config]]") {
		t.Error("BuildCodexConfig should contain '[[skills.config]]' header")
	}
	if !strings.Contains(result, "enabled = true") {
		t.Error("BuildCodexConfig should contain 'enabled = true'")
	}
}

func TestBuildCodexConfig_pathFormat(t *testing.T) {
	g, _ := newTestGenerator()
	result := g.BuildCodexConfig([]string{"my-skill"})
	if !strings.Contains(result, ".agents/skills/my-skill") {
		t.Errorf("BuildCodexConfig path = %q, want '.agents/skills/my-skill'", result)
	}
}

func TestGenerateClaude_withSkill(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	// Seed one processual skill
	seedSkill(fsys, src, "bugfix", "Corrects bugs automatically.")

	g.GenerateClaude(src, proj)

	agentFile := filepath.Join(proj, ".claude", "agents", "bugfixer.md")
	if !fsys.Exists(agentFile) {
		t.Errorf("GenerateClaude should create %s", agentFile)
	}
	data, _ := fsys.ReadFile(agentFile)
	if !strings.Contains(string(data), "bugfixer") {
		t.Errorf("bugfixer.md should contain 'bugfixer', got: %s", data)
	}
}

func TestGenerateClaude_noSkillFiles(t *testing.T) {
	g, fsys := newTestGenerator()
	// No skills seeded — generator should not crash, just produce nothing
	g.GenerateClaude("/source", "/project")

	agentsDir := filepath.Join("/project", ".claude", "agents")
	entries, _ := fsys.ReadDir(agentsDir)
	if len(entries) != 0 {
		t.Errorf("GenerateClaude without skill files should produce no agents, got %d", len(entries))
	}
}

func TestGenerateGitHub_withSkill(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	seedSkill(fsys, src, "review", "Reviews pull requests.")

	g.GenerateGitHub(src, proj)

	agentFile := filepath.Join(proj, ".github", "agents", "reviewer.agent.md")
	if !fsys.Exists(agentFile) {
		t.Errorf("GenerateGitHub should create %s", agentFile)
	}
}

func TestGenerateGitHub_allEightAgents(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	for _, skill := range adapters.ProcessualSkills {
		seedSkill(fsys, src, skill, "Description for "+skill+".")
	}

	g.GenerateGitHub(src, proj)

	agentsDir := filepath.Join(proj, ".github", "agents")
	entries, err := fsys.ReadDir(agentsDir)
	if err != nil {
		t.Fatalf("failed to read .github/agents: %v", err)
	}
	if len(entries) != 8 {
		t.Errorf("GenerateGitHub should create 8 agent files, got %d", len(entries))
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".agent.md") {
			t.Errorf("agent file %q should have .agent.md suffix", e.Name())
		}
	}
}

func TestGenerateGitHub_templateContent(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	seedSkill(fsys, src, "bugfix", "Corrects bugs automatically.")
	g.GenerateGitHub(src, proj)

	agentFile := filepath.Join(proj, ".github", "agents", "bugfix.agent.md")
	data, err := fsys.ReadFile(agentFile)
	if err != nil {
		t.Fatalf("bugfix.agent.md not created: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "Corretor de Bugs") {
		t.Errorf("bugfix.agent.md should contain GitHub name 'Corretor de Bugs', got: %s", content)
	}
	if !strings.Contains(content, "bugfix") {
		t.Errorf("bugfix.agent.md should reference skill name 'bugfix', got: %s", content)
	}
	if !strings.Contains(content, "corrija os bugs") {
		t.Errorf("bugfix.agent.md should contain canonical instruction, got: %s", content)
	}
}

func TestGenerateGitHub_noSkillFiles(t *testing.T) {
	g, fsys := newTestGenerator()
	g.GenerateGitHub("/source", "/project")

	agentsDir := filepath.Join("/project", ".github", "agents")
	entries, _ := fsys.ReadDir(agentsDir)
	if len(entries) != 0 {
		t.Errorf("GenerateGitHub without skill files should produce no agents, got %d", len(entries))
	}
}

func TestGenerateCodexAgents_withSkill(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	seedSkill(fsys, src, "execute-task", "Executa tarefa aprovada.")

	g.GenerateCodexAgents(src, proj)

	agentFile := filepath.Join(proj, ".codex", "agents", "task-executor.toml")
	if !fsys.Exists(agentFile) {
		t.Fatalf("GenerateCodexAgents should create %s", agentFile)
	}
	data, _ := fsys.ReadFile(agentFile)
	body := string(data)
	if !strings.Contains(body, `name = "task-executor"`) {
		t.Errorf("agent should declare name = \"task-executor\", got: %s", body)
	}
	if !strings.Contains(body, `path = ".agents/skills/execute-task"`) {
		t.Errorf("agent should reference canonical skill path, got: %s", body)
	}
	if !strings.Contains(body, `developer_instructions = """`) {
		t.Errorf("agent should declare developer_instructions, got: %s", body)
	}
}

func TestGenerateCodexAgents_noSkillFiles(t *testing.T) {
	g, fsys := newTestGenerator()
	g.GenerateCodexAgents("/source", "/project")

	entries, _ := fsys.ReadDir(filepath.Join("/project", ".codex", "agents"))
	if len(entries) != 0 {
		t.Errorf("GenerateCodexAgents without skill files should produce no agents, got %d", len(entries))
	}
}

// TestGenerate_executeTaskYAMLContract_allTools verifica que TODOS os 3 adapters
// (Claude/GitHub/Codex) emitem o bloco YAML literal do contrato de retorno
// para o subagent `task-executor`. Regressao guard contra A02 (Copilot anteriormente
// descrevia o retorno em prosa, divergindo dos demais tools e quebrando a cadeia
// de validacao em 4 passos de execute-all-tasks).
func TestGenerate_executeTaskYAMLContract_allTools(t *testing.T) {
	const (
		statusLine     = "status: done | blocked | failed | needs_input"
		reportPathLine = "report_path: .specs/prd-<slug>/<id>_execution_report.md"
		summaryLine    = "summary: <1 linha>"
	)

	cases := []struct {
		name string
		gen  func(g *adapters.Generator, src, proj string)
		path string
	}{
		{"claude", func(g *adapters.Generator, src, proj string) { g.GenerateClaude(src, proj) },
			filepath.Join(".claude", "agents", "task-executor.md")},
		{"github", func(g *adapters.Generator, src, proj string) { g.GenerateGitHub(src, proj) },
			filepath.Join(".github", "agents", "task-executor.agent.md")},
		{"codex", func(g *adapters.Generator, src, proj string) { g.GenerateCodexAgents(src, proj) },
			filepath.Join(".codex", "agents", "task-executor.toml")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, fsys := newTestGenerator()
			src := "/source"
			proj := "/project"
			seedSkill(fsys, src, "execute-task", "Executa tarefa aprovada.")

			tc.gen(g, src, proj)

			agentFile := filepath.Join(proj, tc.path)
			if !fsys.Exists(agentFile) {
				t.Fatalf("%s adapter should create %s", tc.name, agentFile)
			}
			data, _ := fsys.ReadFile(agentFile)
			body := string(data)
			for _, line := range []string{statusLine, reportPathLine, summaryLine} {
				if !strings.Contains(body, line) {
					t.Errorf("%s adapter: agent file missing YAML contract line %q\nfull body:\n%s", tc.name, line, body)
				}
			}
		})
	}
}

// TestGenerate_nonExecuteTaskHasNoYAMLContract garante que o bloco YAML so eh
// injetado para `execute-task` — outros agents processuais (bugfix, review, etc.)
// nao devem carregar o contrato porque nao retornam YAML estruturado.
func TestGenerate_nonExecuteTaskHasNoYAMLContract(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	seedSkill(fsys, src, "bugfix", "Corrige bugs no escopo.")
	g.GenerateClaude(src, proj)

	agentFile := filepath.Join(proj, ".claude", "agents", "bugfixer.md")
	data, _ := fsys.ReadFile(agentFile)
	body := string(data)
	if strings.Contains(body, "status: done | blocked") {
		t.Errorf("bugfix agent should NOT contain execute-task YAML contract, got:\n%s", body)
	}
}

func TestProcessualSkills_notEmpty(t *testing.T) {
	if len(adapters.ProcessualSkills) == 0 {
		t.Error("ProcessualSkills should not be empty")
	}
}

func TestProcessualSkills_containsBugfix(t *testing.T) {
	found := false
	for _, s := range adapters.ProcessualSkills {
		if s == "bugfix" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ProcessualSkills should contain 'bugfix'")
	}
}
