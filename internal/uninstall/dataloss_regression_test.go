package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestUninstall_TrackedManifest_DoesNotSweepUserFilesInGeneratedDirs(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	userFiles := map[string]string{
		"/project/.agents/skills/MINHA_SKILL.md": "USER SKILL\n",
		"/project/.claude/skills/minha-skill.md": "USER CLAUDE SKILL\n",
		"/project/.claude/agents/meu-agente.md":  "USER CLAUDE AGENT\n",
		"/project/.github/skills/minha-skill.md": "USER COPILOT SKILL\n",
		"/project/.github/agents/meu.agent.md":   "USER COPILOT AGENT\n",
		"/project/.codex/agents/meu.toml":        "user = true\n",
	}
	for path, content := range userFiles {
		ffs.Files[path] = []byte(content)
	}
	ffs.Files["/project/.agents/skills/agent-governance/SKILL.md"] = []byte("---\nversion: 1.0.0\n---\n")
	ffs.Files["/project/.claude/agents/review.md"] = []byte("wrapper")
	ffs.Files["/project/.ai_spec_harness.json"] = []byte(`{
		"version": "1.0.0-test",
		"installed_files": [
			".agents/skills/agent-governance/SKILL.md",
			".claude/agents/review.md"
		]
	}`)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ffs.Exists("/project/.agents/skills/agent-governance/SKILL.md") {
		t.Error("arquivo rastreado pelo manifesto deveria ser removido")
	}
	if ffs.Exists("/project/.claude/agents/review.md") {
		t.Error("wrapper rastreado pelo manifesto deveria ser removido")
	}
	for path, content := range userFiles {
		data, err := ffs.ReadFile(path)
		if err != nil {
			t.Errorf("arquivo do usuario %s destruido pelo sweep: %v", path, err)
			continue
		}
		if string(data) != content {
			t.Errorf("arquivo do usuario %s alterado: got %q, want %q", path, string(data), content)
		}
	}
}

func TestUninstall_WithoutManifest_PreservesAuthoredContextFilesAndWarns(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	authored := map[string]string{
		"/project/AGENTS.md":                       "# Meu AGENTS autoral\n\nregras minhas\n",
		"/project/CLAUDE.md":                       "# Meu CLAUDE autoral\n",
		"/project/.codex/config.toml":              "model = \"gpt-5\"\n",
		"/project/.github/copilot-instructions.md": "# instrucoes autorais\n",
	}
	for path, content := range authored {
		ffs.Files[path] = []byte(content)
	}

	var buf strings.Builder
	printer := &output.Printer{Out: &buf, Err: &buf, Verbose: true}
	if err := NewService(ffs, printer).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for path, content := range authored {
		data, err := ffs.ReadFile(path)
		if err != nil {
			t.Errorf("arquivo autoral %s destruido sem manifesto: %v", path, err)
			continue
		}
		if string(data) != content {
			t.Errorf("arquivo autoral %s alterado: got %q, want %q", path, string(data), content)
		}
	}
	out := buf.String()
	if !strings.Contains(out, "manifesto sem rastreamento por arquivo") {
		t.Errorf("ausencia de manifesto deve sempre anunciar o caminho conservador; got: %q", out)
	}
	if !strings.Contains(out, "nao corresponde ao conteudo gerado pelo harness") {
		t.Errorf("preservacao por conteudo nao reconhecido deve ser anunciada; got: %q", out)
	}
}

func TestUninstall_WithoutManifest_RemovesGeneratedContextFiles(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Files["/project/AGENTS.md"] = []byte(generatedAgentsMarkdown)
	ffs.Files["/project/CLAUDE.md"] = []byte(generatedToolMarkdown("Claude Code"))
	ffs.Files["/project/.github/copilot-instructions.md"] = []byte(generatedToolMarkdown("GitHub Copilot CLI"))
	ffs.Files["/project/.codex/config.toml"] = []byte(generatedCodexConfig)

	if err := NewService(ffs, output.New(false)).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, path := range []string{
		"/project/AGENTS.md",
		"/project/CLAUDE.md",
		"/project/.github/copilot-instructions.md",
		"/project/.codex/config.toml",
	} {
		if ffs.Exists(path) {
			t.Errorf("%s com conteudo gerado deveria ser removido", path)
		}
	}
}

func TestUninstall_WithoutGovernanceBlock_IsByteIdenticalNoop(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	userJSON := map[string]string{
		"/project/opencode.json":         "{\n    \"theme\": \"tokyonight\",\n    \"$schema\": \"https://opencode.ai/config.json\"\n}\n",
		"/project/.github/settings.json": "{\n    \"zeta\": 1,\n    \"alpha\": 2\n}\n",
	}
	for path, content := range userJSON {
		ffs.Files[path] = []byte(content)
	}

	var buf strings.Builder
	printer := &output.Printer{Out: &buf, Err: &buf, Verbose: true}
	if err := NewService(ffs, printer).Execute("/project", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for path, content := range userJSON {
		data, err := ffs.ReadFile(path)
		if err != nil {
			t.Errorf("%s removido: %v", path, err)
			continue
		}
		if string(data) != content {
			t.Errorf("%s reformatado sem bloco de governanca a remover:\ngot:  %q\nwant: %q", path, string(data), content)
		}
	}
	if !strings.Contains(buf.String(), "Nada a remover") {
		t.Errorf("contagem deve reportar nada removido quando nada foi removido; got: %q", buf.String())
	}
}

const (
	authoredAgentsMD  = "# AGENTS autoral do time\n\nregra exclusiva ALPHA-7\n"
	authoredClaudeMD  = "# CLAUDE autoral do time\n\nnota exclusiva BETA-7\n"
	authoredCopilotMD = "# Copilot autoral do time\n\nnota exclusiva GAMMA-7\n"
	authoredCodexTOML = "model = \"gpt-5\"\n\n[minha_secao]\nchave = \"DELTA-7\"\n"
	authoredGeminiMD  = "# GEMINI autoral do time\n\nnota exclusiva EPSILON-7\n"
)

func criticalAuthoredFiles() map[string]string {
	return map[string]string{
		"AGENTS.md": authoredAgentsMD,
		"CLAUDE.md": authoredClaudeMD,
		filepath.Join(".github", "copilot-instructions.md"): authoredCopilotMD,
		filepath.Join(".codex", "config.toml"):              authoredCodexTOML,
	}
}

func seedFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
}

func assertByteIdentical(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, want := range files {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Errorf("arquivo do usuario %s destruido pelo roundtrip: %v", rel, err)
			continue
		}
		if string(data) != want {
			t.Errorf("arquivo do usuario %s nao e byte-identico:\ngot:  %q\nwant: %q", rel, string(data), want)
		}
	}
}

func TestRoundtrip_AuthoredContextFiles_SurviveInstallAndTwoUninstalls(t *testing.T) {
	harness := roundtripHarness{}
	root := t.TempDir()

	seeded := criticalAuthoredFiles()
	seeded["GEMINI.md"] = authoredGeminiMD
	seeded["opencode.json"] = "{\n    \"theme\": \"tokyonight\"\n}\n"
	seeded[filepath.Join(".agents", "skills", "minha-skill", "SKILL.md")] = "# skill do usuario\n"
	seeded[filepath.Join(".claude", "agents", "meu-agente.md")] = "# agente do usuario\n"
	seeded[filepath.Join(".codex", "agents", "meu.toml")] = "user = true\n"
	seedFiles(t, root, seeded)

	harness.install(t, root, []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode})

	for rel := range criticalAuthoredFiles() {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("%s destruido pelo install: %v", rel, err)
		}
		marker := map[string]string{
			"AGENTS.md": "ALPHA-7",
			"CLAUDE.md": "BETA-7",
			filepath.Join(".github", "copilot-instructions.md"): "GAMMA-7",
			filepath.Join(".codex", "config.toml"):              "DELTA-7",
		}[rel]
		if !strings.Contains(string(data), marker) {
			t.Errorf("install destruiu o conteudo autoral de %s (marcador %q ausente):\n%s", rel, marker, string(data))
		}
	}

	harness.uninstall(t, root)
	assertByteIdentical(t, root, seeded)

	harness.uninstall(t, root)
	assertByteIdentical(t, root, seeded)

	for rel, want := range map[string]string{
		"AGENTS.md": "governance-schema",
		"CLAUDE.md": "Ler `AGENTS.md` no inicio da sessao",
		filepath.Join(".github", "copilot-instructions.md"): "Ler `AGENTS.md` no inicio da sessao",
		filepath.Join(".codex", "config.toml"):              "[[skills.config]]",
	} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		if strings.Contains(string(data), want) {
			t.Errorf("residuo de governanca em %s apos dois uninstalls: %q ainda presente", rel, want)
		}
		if strings.Contains(string(data), "ai-spec-harness:") {
			t.Errorf("marcador do harness sobreviveu em %s:\n%s", rel, string(data))
		}
	}
}

func TestRoundtrip_GeneratedContextFiles_AreRemovedWhenNotAuthored(t *testing.T) {
	harness := roundtripHarness{}
	root := t.TempDir()

	harness.install(t, root, []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode})
	harness.uninstall(t, root)
	harness.uninstall(t, root)

	for rel := range criticalAuthoredFiles() {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			t.Errorf("%s gerado do zero pelo install deveria ter sido removido", rel)
		}
	}
}

func TestRoundtrip_TwoUninstalls_PreserveUserFilesByteIdentical(t *testing.T) {
	harness := roundtripHarness{}
	root := t.TempDir()

	seeded := map[string]string{
		"opencode.json": "{\n    \"theme\": \"tokyonight\",\n    \"$schema\": \"https://opencode.ai/config.json\"\n}\n",
		filepath.Join(".github", "settings.json"):                     "{\n    \"zeta\": 1,\n    \"alpha\": 2\n}\n",
		filepath.Join(".agents", "skills", "minha-skill", "SKILL.md"): "# skill do usuario\n",
		filepath.Join(".claude", "skills", "minha-skill.md"):          "# skill claude do usuario\n",
		filepath.Join(".claude", "agents", "meu-agente.md"):           "# agente do usuario\n",
		filepath.Join(".github", "agents", "meu.agent.md"):            "# agente copilot do usuario\n",
		filepath.Join(".github", "skills", "minha-skill.md"):          "# skill copilot do usuario\n",
		filepath.Join(".codex", "agents", "meu.toml"):                 "user = true\n",
		filepath.Join(".opencode", "plugin", "meu-plugin.js"):         "export const user = {}\n",
	}
	seedFiles(t, root, seeded)

	harness.install(t, root, []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode})
	harness.uninstall(t, root)
	harness.uninstall(t, root)

	reformattedByInstall := map[string]bool{filepath.Join(".github", "settings.json"): true}
	for rel, content := range seeded {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Errorf("arquivo do usuario %s destruido pelo roundtrip: %v", rel, err)
			continue
		}
		if reformattedByInstall[rel] {
			continue
		}
		if string(data) != content {
			t.Errorf("arquivo do usuario %s nao e byte-identico:\ngot:  %q\nwant: %q", rel, string(data), content)
		}
	}

	settings, err := os.ReadFile(filepath.Join(root, ".github", "settings.json"))
	if err != nil {
		t.Fatalf(".github/settings.json do usuario destruido: %v", err)
	}
	if strings.Contains(string(settings), ".github/hooks/") {
		t.Errorf("hooks de governanca deveriam sair no merge reverso: %s", settings)
	}
	if !strings.Contains(string(settings), "zeta") || !strings.Contains(string(settings), "alpha") {
		t.Errorf("chaves do usuario deveriam sobreviver ao roundtrip: %s", settings)
	}
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md", filepath.Join(".github", "copilot-instructions.md"), filepath.Join(".codex", "config.toml")} {
		if _, err := os.Stat(filepath.Join(root, rel)); err == nil {
			t.Errorf("%s gerado pelo install deveria ter sido removido", rel)
		}
	}
}

func TestUninstall_NeverInstalledDir_LeavesEveryFileByteIdentical(t *testing.T) {
	harness := roundtripHarness{}
	root := t.TempDir()

	seeded := map[string]string{
		"AGENTS.md":     "# Meu AGENTS autoral\n\nregras minhas\n",
		"CLAUDE.md":     "# Meu CLAUDE autoral\n",
		"opencode.json": "{\n    \"theme\": \"tokyonight\"\n}\n",
		filepath.Join(".github", "settings.json"):                     "{\n    \"zeta\": 1,\n    \"alpha\": 2\n}\n",
		filepath.Join(".github", "copilot-instructions.md"):           "# instrucoes autorais\n",
		filepath.Join(".codex", "config.toml"):                        "model = \"gpt-5\"\n",
		filepath.Join(".agents", "skills", "minha-skill", "SKILL.md"): "# skill do usuario\n",
		filepath.Join(".claude", "agents", "meu-agente.md"):           "# agente do usuario\n",
		filepath.Join(".codex", "agents", "meu.toml"):                 "user = true\n",
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

	harness.uninstall(t, root)
	harness.uninstall(t, root)

	for rel, content := range seeded {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Errorf("arquivo do usuario %s destruido em diretorio nunca instalado: %v", rel, err)
			continue
		}
		if string(data) != content {
			t.Errorf("arquivo do usuario %s alterado:\ngot:  %q\nwant: %q", rel, string(data), content)
		}
	}
}
