package uninstall

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

func TestUninstall_RemovesCurrentClaudeSettingsTemplate(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.claude/settings.local.json"] = []byte(installClaudeSettingsTemplate)

	var buf strings.Builder
	printer := &output.Printer{Out: &buf, Err: &buf, Verbose: true}
	if err := NewService(ffs, printer).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ffs.Exists("/project/.claude/settings.local.json") {
		t.Error("settings.local.json gerado pelo install atual deveria ser removido")
	}
	if strings.Contains(buf.String(), "contem configuracoes alem dos hooks de governanca") {
		t.Error("aviso factualmente falso nao deveria ser emitido para o template gerado")
	}
}

func TestUninstall_RemovesLegacyClaudeSettingsTemplate(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.claude/settings.local.json"] = []byte(legacyClaudeSettingsTemplate)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ffs.Exists("/project/.claude/settings.local.json") {
		t.Error("settings.local.json de versao anterior tambem deveria ser removido")
	}
}

func TestUninstall_KeepsCustomizedClaudeSettings(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	customized := `{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          {"type": "command", "command": "bash .claude/hooks/validate-preload.sh"},
          {"type": "command", "command": "bash scripts/meu-hook-pessoal.sh"}
        ]
      }
    ]
  }
}
`
	ffs.Files["/project/.claude/settings.local.json"] = []byte(customized)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ffs.Exists("/project/.claude/settings.local.json") {
		t.Error("settings.local.json com hook do usuario NAO deveria ser removido")
	}
}

func TestUninstall_ReverseMergesCopilotSettings(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.github/settings.json"] = []byte(`{
  "hooks": {
    "preToolUse": [
      {"type": "command", "bash": "bash .github/hooks/validate-preload.sh"}
    ],
    "postToolUse": [
      {"type": "command", "bash": "bash .github/hooks/validate-governance.sh"}
    ]
  }
}
`)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ffs.Exists("/project/.github/settings.json") {
		t.Fatal(".github/settings.json apenas com hooks de governanca deveria ser removido")
	}
}

func TestUninstall_PreservesUserKeysInCopilotSettings(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.github/settings.json"] = []byte(`{
  "custom": true,
  "hooks": {
    "preToolUse": [
      {"type": "command", "bash": "bash .github/hooks/validate-preload.sh"},
      {"type": "command", "bash": "bash scripts/meu-hook.sh"}
    ]
  }
}
`)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := ffs.ReadFile("/project/.github/settings.json")
	if err != nil {
		t.Fatalf(".github/settings.json com chaves do usuario nao pode ser removido: %v", err)
	}
	content := string(data)
	if strings.Contains(content, ".github/hooks/") {
		t.Errorf("hooks de governanca deveriam sair no merge reverso: %s", content)
	}
	if !strings.Contains(content, "meu-hook.sh") || !strings.Contains(content, "custom") {
		t.Errorf("conteudo do usuario deveria ser preservado: %s", content)
	}
}

func TestUninstall_ReverseMergesOpenCodeConfig(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/opencode.json"] = []byte(`{
  "$schema": "https://opencode.ai/config.json",
  "theme": "tokyonight",
  "permission": {
    "bash": {
      "rm -rf*": "deny",
      "git push --force*": "deny",
      "git reset --hard*": "deny",
      "meu-comando*": "allow"
    }
  }
}
`)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := ffs.ReadFile("/project/opencode.json")
	if err != nil {
		t.Fatalf("opencode.json do usuario nao pode ser removido: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "rm -rf*") {
		t.Errorf("bloco de permissao do harness deveria sair: %s", content)
	}
	if !strings.Contains(content, "meu-comando*") || !strings.Contains(content, "tokyonight") {
		t.Errorf("regras e campos do usuario deveriam ser preservados: %s", content)
	}
}

func TestUninstall_RemovesOpenCodeConfigWhenOnlyHarnessBlock(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/opencode.json"] = []byte(`{
  "permission": {
    "bash": {
      "rm -rf*": "deny",
      "git push --force*": "deny",
      "git reset --hard*": "deny"
    }
  }
}
`)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ffs.Exists("/project/opencode.json") {
		t.Error("opencode.json contendo apenas o bloco do harness deveria ser removido")
	}
}

func TestUninstall_ManifestTrackedFilesPreserveUserPlugin(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/.agents/skills/review/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.opencode/plugin/governance.js"] = []byte("export const governance = {}")
	ffs.Files["/project/.opencode/plugin/my-user-plugin.js"] = []byte("export const user = {}")
	ffs.Files["/project/.ai_spec_harness.json"] = []byte(`{
		"version": "1.0.0-test",
		"installed_files": [".opencode/plugin/governance.js"],
		"merged_files": ["opencode.json"]
	}`)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ffs.Exists("/project/.opencode/plugin/governance.js") {
		t.Error("governance.js rastreado deveria ser removido")
	}
	if !ffs.Exists("/project/.opencode/plugin/my-user-plugin.js") {
		t.Error("plugin do usuario no mesmo diretorio NAO deveria ser removido")
	}
}
