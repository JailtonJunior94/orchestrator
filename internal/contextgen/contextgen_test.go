package contextgen

import (
	"bytes"
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
