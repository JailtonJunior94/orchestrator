package contextgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestCopilotGuidance(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	g := NewGenerator(ffs, output.New(false))
	err := g.Generate("/source", "/project", []skills.Tool{skills.ToolCopilot}, nil, "full", false)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	data, ok := ffs.Files["/project/.github/copilot-instructions.md"]
	if !ok {
		t.Fatal("copilot-instructions.md nao foi gerado")
	}
	content := string(data)
	if !strings.Contains(content, "Orientacoes Especificas para Copilot") {
		t.Errorf("copilot-instructions.md deve conter 'Orientacoes Especificas para Copilot', got:\n%s", content)
	}
	// verifica os 5 itens
	for i := 1; i <= 5; i++ {
		item := strings.Repeat("", 0) + string(rune('0'+i)) + "."
		if !strings.Contains(content, item) {
			t.Errorf("copilot-instructions.md deve conter item %d., got:\n%s", i, content)
		}
	}
}

func TestCompactProfile(t *testing.T) {
	t.Run("standard_contains_sections", func(t *testing.T) {
		ffs := fs.NewFakeFileSystem()
		ffs.Dirs["/project"] = true
		ffs.Dirs["/source"] = true
		g := NewGenerator(ffs, output.New(false))
		// Claude + Codex → standard
		err := g.Generate("/source", "/project", []skills.Tool{skills.ToolClaude, skills.ToolCodex}, nil, "full", false)
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}
		data := string(ffs.Files["/project/AGENTS.md"])
		if !strings.Contains(data, "## Diretrizes de Estrutura") {
			t.Errorf("profile standard deve conter '## Diretrizes de Estrutura'")
		}
		if !strings.Contains(data, "### Composicao Multi-Linguagem") {
			t.Errorf("profile standard deve conter '### Composicao Multi-Linguagem'")
		}
	})

	t.Run("compact_strips_sections", func(t *testing.T) {
		ffs := fs.NewFakeFileSystem()
		ffs.Dirs["/project"] = true
		ffs.Dirs["/source"] = true
		g := NewGenerator(ffs, output.New(false))
		// apenas Codex → compact
		err := g.Generate("/source", "/project", []skills.Tool{skills.ToolCodex}, nil, "full", false)
		if err != nil {
			t.Fatalf("Generate returned error: %v", err)
		}
		data := string(ffs.Files["/project/AGENTS.md"])
		if strings.Contains(data, "## Diretrizes de Estrutura") {
			t.Errorf("profile compact nao deve conter '## Diretrizes de Estrutura'")
		}
		if strings.Contains(data, "### Composicao Multi-Linguagem") {
			t.Errorf("profile compact nao deve conter '### Composicao Multi-Linguagem'")
		}
	})
}

func TestDryRun(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	var buf bytes.Buffer
	printer := &output.Printer{Out: &buf, Verbose: false}
	g := NewGenerator(ffs, printer)

	tools := []skills.Tool{skills.ToolClaude, skills.ToolGemini, skills.ToolCopilot, skills.ToolCodex}
	err := g.Generate("/source", "/project", tools, nil, "full", true)
	if err != nil {
		t.Fatalf("Generate dry-run returned error: %v", err)
	}

	// Nenhum arquivo deve ter sido escrito
	if len(ffs.Files) > 0 {
		t.Errorf("dry-run nao deve escrever arquivos, mas escreveu: %v", ffs.Files)
	}

	outStr := buf.String()
	for _, expected := range []string{"AGENTS.md", "CLAUDE.md", "GEMINI.md", "copilot-instructions.md", "config.toml"} {
		if !strings.Contains(outStr, expected) {
			t.Errorf("dry-run output deve mencionar %q, got:\n%s", expected, outStr)
		}
	}
	if !strings.Contains(outStr, GovernanceSchemaVersion) {
		t.Errorf("dry-run output deve mencionar schema version %q, got:\n%s", GovernanceSchemaVersion, outStr)
	}
}

func TestCodexOnlyCompact(t *testing.T) {
	// Usa a fixture codex-only para validar profile compact com go.mod presente
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	fixtureDir := filepath.Join("..", "..", "testdata", "fixtures", "codex-only")
	goModData, err := os.ReadFile(filepath.Join(fixtureDir, "go.mod"))
	if err != nil {
		t.Fatalf("fixture go.mod nao encontrado: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), goModData, 0o644); err != nil {
		t.Fatalf("escrever go.mod: %v", err)
	}
	codexCfgData, err := os.ReadFile(filepath.Join(fixtureDir, ".codex", "config.toml"))
	if err != nil {
		t.Fatalf("fixture .codex/config.toml nao encontrado: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(projectDir, ".codex"), 0o755); err != nil {
		t.Fatalf("mkdir .codex: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".codex", "config.toml"), codexCfgData, 0o644); err != nil {
		t.Fatalf("escrever .codex/config.toml: %v", err)
	}

	fsys := fs.NewOSFileSystem()
	g := NewGenerator(fsys, output.New(false))

	// tools=[Codex] apenas → profile compact auto-detectado
	err = g.Generate(sourceDir, projectDir, []skills.Tool{skills.ToolCodex}, nil, "full", false)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md nao gerado: %v", err)
	}
	content := string(data)

	// Secoes verbose devem estar ausentes no profile compact
	if strings.Contains(content, "## Diretrizes de Estrutura") {
		t.Error("profile compact nao deve conter '## Diretrizes de Estrutura'")
	}
	if strings.Contains(content, "### Composicao Multi-Linguagem") {
		t.Error("profile compact nao deve conter '### Composicao Multi-Linguagem'")
	}

	// Secoes essenciais devem estar presentes
	for _, section := range []string{"## Arquitetura", "## Validacao", "## Restricoes"} {
		if !strings.Contains(content, section) {
			t.Errorf("profile compact deve conter %q", section)
		}
	}
}

func TestAgentsMdInvocationDepth(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	g := NewGenerator(ffs, output.New(false))
	err := g.Generate("/source", "/project", []skills.Tool{skills.ToolClaude}, nil, "full", false)
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	data := string(ffs.Files["/project/AGENTS.md"])
	if !strings.Contains(data, "check-invocation-depth") {
		t.Errorf("AGENTS.md deve conter referencia a 'check-invocation-depth', got:\n%s", data)
	}
	if !strings.Contains(data, "profundidade de invocacao") {
		t.Errorf("AGENTS.md deve conter 'profundidade de invocacao', got:\n%s", data)
	}
}

func assertMatchesSnapshot(t *testing.T, snapshotPath string, actual string) {
	t.Helper()

	if os.Getenv("UPDATE_SNAPSHOTS") == "1" {
		if err := os.MkdirAll(filepath.Dir(snapshotPath), 0o755); err != nil {
			t.Fatalf("criar diretorio de snapshots: %v", err)
		}
		if err := os.WriteFile(snapshotPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("escrever snapshot: %v", err)
		}
		return
	}

	expected, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("snapshot nao encontrado: %s (execute com UPDATE_SNAPSHOTS=1 para criar)", snapshotPath)
	}

	if string(expected) != actual {
		t.Errorf("snapshot divergente: %s\n\nDiff:\n%s", snapshotPath, snapshotDiff(string(expected), actual))
	}
}

func snapshotDiff(expected, actual string) string {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")
	var buf strings.Builder
	maxLen := len(expLines)
	if len(actLines) > maxLen {
		maxLen = len(actLines)
	}
	diffs := 0
	for i := 0; i < maxLen && diffs < 20; i++ {
		var exp, act string
		if i < len(expLines) {
			exp = expLines[i]
		}
		if i < len(actLines) {
			act = actLines[i]
		}
		if exp != act {
			fmt.Fprintf(&buf, "linha %d:\n  esperado: %q\n  obtido:   %q\n", i+1, exp, act)
			diffs++
		}
	}
	if diffs == 20 {
		fmt.Fprintf(&buf, "... (truncado apos 20 diferencas)\n")
	}
	return buf.String()
}

func TestContextgen_Snapshots(t *testing.T) {
	snapshotsDir := filepath.Join("..", "..", "testdata", "snapshots")

	fixtures := []struct {
		name  string
		dir   string
		tools []skills.Tool
	}{
		{"go-microservice", filepath.Join("..", "..", "testdata", "go-microservice"), []skills.Tool{skills.ToolClaude}},
		{"go-modular", filepath.Join("..", "..", "testdata", "go-modular"), []skills.Tool{skills.ToolClaude}},
		{"go-monolith", filepath.Join("..", "..", "testdata", "go-monolith"), []skills.Tool{skills.ToolClaude}},
		{"node-monorepo", filepath.Join("..", "..", "testdata", "node-monorepo"), []skills.Tool{skills.ToolGemini}},
		{"node-api", filepath.Join("..", "..", "testdata", "node-api"), []skills.Tool{skills.ToolClaude}},
		{"python-api", filepath.Join("..", "..", "testdata", "python-api"), []skills.Tool{skills.ToolClaude}},
		{"python-monorepo", filepath.Join("..", "..", "testdata", "python-monorepo"), []skills.Tool{skills.ToolClaude}},
		{"polyglot-monorepo", filepath.Join("..", "..", "testdata", "polyglot-monorepo"), []skills.Tool{skills.ToolClaude, skills.ToolGemini}},
	}

	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			projectDir := t.TempDir()
			fsys := fs.NewOSFileSystem()
			g := NewGenerator(fsys, output.New(false))

			if err := g.Generate(tc.dir, projectDir, tc.tools, nil, "full", false); err != nil {
				t.Fatalf("Generate(%s): %v", tc.name, err)
			}

			data, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
			if err != nil {
				t.Fatalf("AGENTS.md nao gerado para %s: %v", tc.name, err)
			}

			snapshotPath := filepath.Join(snapshotsDir, tc.name+".agents.md")
			assertMatchesSnapshot(t, snapshotPath, string(data))
		})
	}
}

func TestBuildCodexConfig(t *testing.T) {
	t.Run("lean_omits_analyze-project", func(t *testing.T) {
		ffs := fs.NewFakeFileSystem()
		g := NewGenerator(ffs, output.New(false))
		content := g.buildCodexConfig("/project", "lean")
		if strings.Contains(content, "analyze-project") {
			t.Errorf("lean config should not contain analyze-project, got:\n%s", content)
		}
		if strings.Contains(content, "create-prd") {
			t.Errorf("lean config should not contain create-prd, got:\n%s", content)
		}
		if strings.Contains(content, "create-technical-specification") {
			t.Errorf("lean config should not contain create-technical-specification, got:\n%s", content)
		}
		if strings.Contains(content, "create-tasks") {
			t.Errorf("lean config should not contain create-tasks, got:\n%s", content)
		}
		if !strings.Contains(content, "agent-governance") {
			t.Errorf("lean config should still contain agent-governance, got:\n%s", content)
		}
	})

	t.Run("full_includes_analyze-project", func(t *testing.T) {
		ffs := fs.NewFakeFileSystem()
		g := NewGenerator(ffs, output.New(false))
		content := g.buildCodexConfig("/project", "full")
		if !strings.Contains(content, "analyze-project") {
			t.Errorf("full config should contain analyze-project, got:\n%s", content)
		}
		if !strings.Contains(content, "create-prd") {
			t.Errorf("full config should contain create-prd, got:\n%s", content)
		}
		if !strings.Contains(content, "create-technical-specification") {
			t.Errorf("full config should contain create-technical-specification, got:\n%s", content)
		}
		if !strings.Contains(content, "create-tasks") {
			t.Errorf("full config should contain create-tasks, got:\n%s", content)
		}
	})
}
