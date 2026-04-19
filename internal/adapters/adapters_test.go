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

func TestGenerateGemini_withSkill(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	// Seed a non-agent-governance skill directory
	seedSkill(fsys, src, "custom-skill", "Does something useful.")

	g.GenerateGemini(src, proj)

	tomlFile := filepath.Join(proj, ".gemini", "commands", "custom-skill.toml")
	if !fsys.Exists(tomlFile) {
		t.Errorf("GenerateGemini should create %s", tomlFile)
	}
	data, _ := fsys.ReadFile(tomlFile)
	if !strings.Contains(string(data), "Does something useful") {
		t.Errorf("Gemini toml should contain description, got: %s", data)
	}
}

func TestGenerateGemini_skipsAgentGovernance(t *testing.T) {
	g, fsys := newTestGenerator()
	src := "/source"
	proj := "/project"

	seedSkill(fsys, src, "agent-governance", "Governance skill.")

	g.GenerateGemini(src, proj)

	tomlFile := filepath.Join(proj, ".gemini", "commands", "agent-governance.toml")
	if fsys.Exists(tomlFile) {
		t.Error("GenerateGemini should skip agent-governance skill")
	}
}

func TestGenerateGemini_noSkillsDir(t *testing.T) {
	g, _ := newTestGenerator()
	// No skills directory — should not panic
	g.GenerateGemini("/source", "/project")
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
