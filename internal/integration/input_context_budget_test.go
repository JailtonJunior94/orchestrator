//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/adapters"
	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/install"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/metrics"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
	"github.com/JailtonJunior94/ai-spec-harness/internal/taskloop"
)

type providerEntryFiles struct {
	tool  skills.Tool
	files []string
}

var declaredEntryContext = []providerEntryFiles{
	{
		tool: skills.ToolClaude,
		files: []string{
			"CLAUDE.md",
			"AGENTS.md",
			filepath.Join(".claude", "rules", "governance.md"),
			filepath.Join(".claude", "rules", "code-style.md"),
		},
	},
	{
		tool: skills.ToolCodex,
		files: []string{
			"AGENTS.md",
			filepath.Join(".codex", "config.toml"),
		},
	},
	{
		tool: skills.ToolCopilot,
		files: []string{
			filepath.Join(".github", "copilot-instructions.md"),
		},
	},
	{
		tool: skills.ToolOpenCode,
		files: []string{
			"AGENTS.md",
		},
	},
}

type inputContextBudget struct {
	measured int
	ceiling  int
}

var inputContextBudgets = map[skills.Tool]inputContextBudget{
	skills.ToolClaude:   {measured: 3719, ceiling: 4091},
	skills.ToolCodex:    {measured: 2570, ceiling: 2827},
	skills.ToolCopilot:  {measured: 507, ceiling: 558},
	skills.ToolOpenCode: {measured: 1793, ceiling: 1972},
}

func installEntryContextFixture(t *testing.T) string {
	t.Helper()
	root := govRepoRoot(t)
	projectDir := t.TempDir()
	fsys := fs.NewOSFileSystem()
	printer := output.New(false)
	mfst := manifest.NewStore(fsys)
	adpt := adapters.NewGenerator(fsys, printer)
	ctxg := contextgen.NewGenerator(fsys, printer)
	svc := install.NewService(fsys, printer, mfst, adpt, ctxg)

	if err := svc.Execute(config.InstallOptions{
		ProjectDir:  projectDir,
		SourceDir:   root,
		Tools:       []skills.Tool{skills.ToolClaude, skills.ToolCodex, skills.ToolCopilot, skills.ToolOpenCode},
		Langs:       []skills.Lang{skills.LangGo},
		LinkMode:    skills.LinkCopy,
		GenerateCtx: true,
	}); err != nil {
		t.Fatalf("install entry context fixture: %v", err)
	}
	return projectDir
}

func measureDeclaredEntryContext(t *testing.T, projectDir string, files []string) int {
	t.Helper()
	tokenizer := metrics.NewCharEstimator()
	total := 0
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(projectDir, rel))
		if err != nil {
			t.Fatalf("read declared entry file %s: %v", rel, err)
		}
		total += tokenizer.EstimateTokens(string(data))
	}
	return total
}

func TestTokenBudget_InputContextPerProvider(t *testing.T) {
	projectDir := installEntryContextFixture(t)

	for _, decl := range declaredEntryContext {
		decl := decl
		t.Run(string(decl.tool), func(t *testing.T) {
			t.Parallel()

			total := measureDeclaredEntryContext(t, projectDir, decl.files)
			budget, ok := inputContextBudgets[decl.tool]
			if !ok {
				t.Fatalf("no declared input context budget for provider %q; add inputContextBudgets[%q]", decl.tool, decl.tool)
			}

			t.Logf("provider=%s files=%v total=%d ceiling=%d", decl.tool, decl.files, total, budget.ceiling)

			if total > budget.ceiling {
				pct := float64(total-budget.ceiling) / float64(budget.ceiling) * 100
				t.Errorf(
					"provider %q input context exceeds ceiling: %d tokens > %d tokens (+%.1f%%)\n"+
						"To accept this growth deliberately, set inputContextBudgets[%q] = {measured: %d, ceiling: %d}",
					decl.tool, total, budget.ceiling, pct,
					decl.tool, total, int(float64(total)*1.10),
				)
			}
		})
	}
}

func TestTokenBudget_InputContextNumeratorDiffersAcrossProviders(t *testing.T) {
	projectDir := installEntryContextFixture(t)

	counts := make(map[skills.Tool]int, len(declaredEntryContext))
	for _, decl := range declaredEntryContext {
		counts[decl.tool] = measureDeclaredEntryContext(t, projectDir, decl.files)
	}

	reference := -1
	identical := true
	for _, total := range counts {
		if reference == -1 {
			reference = total
			continue
		}
		if total != reference {
			identical = false
		}
	}

	if identical {
		t.Fatalf(
			"input context numerator is identical across all four providers (%d tokens each): "+
				"this is the signal that the measurement derived from the harness prompt instead of "+
				"the provider-declared native entry files (RF-41.1, V-30); counts=%v", reference, counts,
		)
	}
}

func TestTokenBudget_InputContextNumeratorIncludesNativelyLoadedFilesForOpenCodeAndCodex(t *testing.T) {
	projectDir := installEntryContextFixture(t)

	nativelyLoadedFile := "AGENTS.md"
	catalog := taskloop.NewCatalog()
	prompt := catalog.BuildPrompt(
		filepath.Join(".specs", "prd-fixture", "task-1.0-fixture.md"),
		filepath.Join(".specs", "prd-fixture"),
		taskloop.PromptContext{},
	)
	promptCarriesAgentsMDBody := strings.Contains(prompt, "## Regras por Linguagem")

	for _, tool := range []skills.Tool{skills.ToolCodex, skills.ToolOpenCode} {
		var files []string
		for _, decl := range declaredEntryContext {
			if decl.tool == tool {
				files = decl.files
			}
		}

		found := false
		for _, f := range files {
			if f == nativelyLoadedFile {
				found = true
			}
		}
		if !found {
			t.Fatalf("provider %q: declared entry files %v must include %q, which this provider loads natively outside Job.Prompt (V-30)", tool, files, nativelyLoadedFile)
		}

		agentsMDPath := filepath.Join(projectDir, nativelyLoadedFile)
		if _, err := os.Stat(agentsMDPath); err != nil {
			t.Fatalf("provider %q: declared native file %s must exist on disk after install: %v", tool, nativelyLoadedFile, err)
		}
	}

	if promptCarriesAgentsMDBody {
		t.Fatalf("the prompt constructed by the harness (taskloop.BuildPrompt) must not embed AGENTS.md body content directly; it is loaded natively by Codex and OpenCode, outside Job.Prompt (V-30) — measuring only the prompt would undercount those two providers")
	}
}
