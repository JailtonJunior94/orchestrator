package uninstall

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

type roundtripHarness struct{}

func (h roundtripHarness) install(t *testing.T, projectDir string, tools []skills.Tool) {
	t.Helper()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	svc := install.NewService(
		fsys,
		printer,
		manifest.NewStore(fsys),
		adapters.NewGenerator(fsys, printer),
		contextgen.NewGenerator(fsys, printer),
	)
	err := svc.Execute(config.InstallOptions{
		ProjectDir:   projectDir,
		Tools:        tools,
		Langs:        []skills.Lang{skills.LangGo},
		LinkMode:     skills.LinkCopy,
		GenerateCtx:  true,
		CodexProfile: "full",
	})
	if err != nil {
		t.Fatalf("install(%v): %v", tools, err)
	}
}

func (h roundtripHarness) uninstall(t *testing.T, projectDir string) {
	t.Helper()
	fsys := fs.NewOSFileSystem()
	if err := NewService(fsys, output.New(false)).Execute(projectDir, false); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
}

func (h roundtripHarness) tree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		if rel == "." {
			return nil
		}
		out = append(out, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(out)
	return out
}

func (h roundtripHarness) seedUserFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	seeded := map[string]string{
		filepath.Join(".opencode", "plugin", "my-user-plugin.js"): "export const userPlugin = {}\n",
		filepath.Join("docs", "notes.md"):                         "# minhas notas\n",
		"AGENTS.local.md":                                         "# extensao local\n",
	}
	for rel, content := range seeded {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
	return seeded
}

func TestRoundtrip_InstallUninstall_LeavesOnlyUserFiles(t *testing.T) {
	harness := roundtripHarness{}
	cases := map[string][]skills.Tool{
		"claude":   {skills.ToolClaude},
		"codex":    {skills.ToolCodex},
		"copilot":  {skills.ToolCopilot},
		"opencode": {skills.ToolOpenCode},
		"all":      {skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode},
	}

	for name, tools := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			seeded := harness.seedUserFiles(t, root)
			before := harness.tree(t, root)

			harness.install(t, root, tools)
			harness.uninstall(t, root)
			harness.uninstall(t, root)

			after := harness.tree(t, root)
			if strings.Join(before, "\n") != strings.Join(after, "\n") {
				t.Fatalf("roundtrip deixou residuo/perda.\nantes:\n%s\ndepois:\n%s", strings.Join(before, "\n"), strings.Join(after, "\n"))
			}
			for rel, content := range seeded {
				data, err := os.ReadFile(filepath.Join(root, rel))
				if err != nil {
					t.Fatalf("arquivo do usuario %s perdido: %v", rel, err)
				}
				if string(data) != content {
					t.Errorf("arquivo do usuario %s alterado: got %q, want %q", rel, string(data), content)
				}
			}
		})
	}
}

func TestRoundtrip_RepeatedInstallUninstall_LeavesOnlyUserFiles(t *testing.T) {
	harness := roundtripHarness{}
	cases := map[string][]skills.Tool{
		"copilot": {skills.ToolCopilot},
		"codex":   {skills.ToolCodex},
		"all":     {skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode},
	}

	for name, tools := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			seeded := harness.seedUserFiles(t, root)
			before := harness.tree(t, root)

			harness.install(t, root, tools)
			harness.install(t, root, tools)
			harness.uninstall(t, root)
			harness.uninstall(t, root)

			after := harness.tree(t, root)
			if strings.Join(before, "\n") != strings.Join(after, "\n") {
				t.Fatalf("converged reinstall left residue or lost files.\nbefore:\n%s\nafter:\n%s", strings.Join(before, "\n"), strings.Join(after, "\n"))
			}
			for rel, content := range seeded {
				data, err := os.ReadFile(filepath.Join(root, rel))
				if err != nil {
					t.Fatalf("user file %s was deleted: %v", rel, err)
				}
				if string(data) != content {
					t.Errorf("user file %s was modified: got %q, want %q", rel, string(data), content)
				}
			}
		})
	}
}

func TestRoundtrip_PreservesUserOpenCodeConfigAndClaudeMarkdown(t *testing.T) {
	harness := roundtripHarness{}
	root := t.TempDir()

	userConfig := "{\n  \"$schema\": \"https://opencode.ai/config.json\",\n  \"theme\": \"tokyonight\"\n}\n"
	if err := os.WriteFile(filepath.Join(root, "opencode.json"), []byte(userConfig), 0o644); err != nil {
		t.Fatalf("seed opencode.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte("# claude do usuario\n"), 0o644); err != nil {
		t.Fatalf("seed CLAUDE.md: %v", err)
	}

	harness.install(t, root, []skills.Tool{skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode})
	harness.uninstall(t, root)

	data, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatalf("opencode.json do usuario removido: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "permission") {
		t.Errorf("bloco permission do harness deveria sair no merge reverso: %s", content)
	}
	if !strings.Contains(content, "tokyonight") || !strings.Contains(content, "$schema") {
		t.Errorf("campos do usuario deveriam ser preservados: %s", content)
	}

	claude, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md do usuario removido (install sem claude nunca o criou): %v", err)
	}
	if string(claude) != "# claude do usuario\n" {
		t.Errorf("CLAUDE.md do usuario alterado: %q", string(claude))
	}
}

func TestRoundtrip_ClaudeSettingsTemplateMatchesInstaller(t *testing.T) {
	harness := roundtripHarness{}
	root := t.TempDir()
	harness.install(t, root, []skills.Tool{skills.ToolClaude})

	data, err := os.ReadFile(filepath.Join(root, ".claude", "settings.local.json"))
	if err != nil {
		t.Fatalf("settings.local.json nao foi criado pelo install: %v", err)
	}
	if string(data) != installClaudeSettingsTemplate {
		t.Fatalf("fixture installClaudeSettingsTemplate divergiu do template real do install:\n%s", string(data))
	}
}
