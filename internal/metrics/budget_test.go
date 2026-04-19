package metrics

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
)

func TestBudget_ToolBudgetsDefined(t *testing.T) {
	for _, tool := range []string{"claude", "gemini", "codex", "copilot"} {
		if _, ok := ToolBudgets[tool]; !ok {
			t.Errorf("ToolBudgets: budget nao definido para %q", tool)
		}
	}
}

func TestBudget_CheckBudget_WithinLimit(t *testing.T) {
	content := strings.Repeat("a", 100) // ~28 tokens
	tokens, limit, ok := CheckBudget(content, "codex")
	if !ok {
		t.Errorf("conteudo pequeno deve estar dentro do budget codex (%d tokens, limite %d)", tokens, limit)
	}
}

func TestBudget_CheckBudget_ExceedsLimit(t *testing.T) {
	// Copilot tem limite de 2000 tokens (~7000 chars); criamos conteudo inflado
	inflated := strings.Repeat("palavra ", 8000) // ~16000 tokens
	tokens, limit, ok := CheckBudget(inflated, "copilot")
	if ok {
		t.Errorf("conteudo inflado deve exceder budget copilot (%d tokens, limite %d)", tokens, limit)
	}
	if tokens <= limit {
		t.Errorf("tokens (%d) devem ser maiores que o limite (%d)", tokens, limit)
	}
}

func TestBudget_CheckBudget_UnknownTool(t *testing.T) {
	content := strings.Repeat("x", 1000000)
	_, _, ok := CheckBudget(content, "ferramenta-desconhecida")
	if !ok {
		t.Error("ferramenta desconhecida deve sempre retornar ok=true")
	}
}

func TestBudget_GeneratedGovernance_WithinBudget(t *testing.T) {
	type toolCase struct {
		tool    skills.Tool
		toolKey string
	}
	cases := []toolCase{
		{skills.ToolClaude, "claude"},
		{skills.ToolGemini, "gemini"},
		{skills.ToolCodex, "codex"},
		{skills.ToolCopilot, "copilot"},
	}

	for _, tc := range cases {
		t.Run(tc.toolKey, func(t *testing.T) {
			ffs := fs.NewFakeFileSystem()
			ffs.Dirs["/project"] = true
			ffs.Dirs["/source"] = true
			g := contextgen.NewGenerator(ffs, output.New(false))

			err := g.Generate("/source", "/project", []skills.Tool{tc.tool}, nil, "full", false)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			// AGENTS.md e compartilhado; usar budget do claude (maior)
			agentsData := string(ffs.Files["/project/AGENTS.md"])
			tokens, limit, ok := CheckBudget(agentsData, "claude")
			if !ok {
				t.Errorf("AGENTS.md (%d tokens) excede budget claude (%d)", tokens, limit)
			}
		})
	}
}
