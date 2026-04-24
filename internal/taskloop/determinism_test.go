package taskloop

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// TestBuildPromptDeterminismo verifica que BuildPrompt e deterministica:
// dado o mesmo taskFilePath e prdFolder, produz o mesmo output em chamadas distintas.
// Tambem verifica que inputs diferentes produzem outputs diferentes (RF-13, RF-14).
func TestBuildPromptDeterminismo(t *testing.T) {
	tests := []struct {
		name         string
		taskFilePath string
		prdFolder    string
	}{
		{
			name:         "task e prd folder canonicos",
			taskFilePath: "tasks/prd-feat/task-1.0.md",
			prdFolder:    "tasks/prd-feat",
		},
		{
			name:         "task de outro prd",
			taskFilePath: "tasks/prd-outro/task-2.0.md",
			prdFolder:    "tasks/prd-outro",
		},
		{
			name:         "task em subpasta profunda",
			taskFilePath: "tasks/prd-multiagente/sub/task-3.0.md",
			prdFolder:    "tasks/prd-multiagente",
		},
		{
			name:         "paths com espacos",
			taskFilePath: "tasks/prd com espacos/task-4.0.md",
			prdFolder:    "tasks/prd com espacos",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primeiro := BuildPrompt(tt.taskFilePath, tt.prdFolder)
			segundo := BuildPrompt(tt.taskFilePath, tt.prdFolder)

			if primeiro != segundo {
				t.Errorf("BuildPrompt nao e deterministica: chamadas com mesmos inputs produziram outputs diferentes\nprimeiro:\n%s\nsegundo:\n%s",
					primeiro, segundo)
			}
		})
	}
}

// TestBuildPromptInputsDiferentesProduzemOutputsDiferentes verifica que inputs distintos
// produzem outputs distintos (propriedade complementar ao determinismo — RF-13).
func TestBuildPromptInputsDiferentesProduzemOutputsDiferentes(t *testing.T) {
	base := BuildPrompt("tasks/prd-a/task-1.0.md", "tasks/prd-a")

	variantes := []struct {
		name         string
		taskFilePath string
		prdFolder    string
	}{
		{
			name:         "task file diferente mesmo prd",
			taskFilePath: "tasks/prd-a/task-2.0.md",
			prdFolder:    "tasks/prd-a",
		},
		{
			name:         "prd folder diferente",
			taskFilePath: "tasks/prd-b/task-1.0.md",
			prdFolder:    "tasks/prd-b",
		},
		{
			name:         "ambos diferentes",
			taskFilePath: "tasks/prd-c/task-5.0.md",
			prdFolder:    "tasks/prd-c",
		},
	}

	for _, v := range variantes {
		t.Run(v.name, func(t *testing.T) {
			variante := BuildPrompt(v.taskFilePath, v.prdFolder)
			if variante == base {
				t.Errorf("BuildPrompt com inputs diferentes produziu o mesmo output que o caso base\ninputs: taskFilePath=%q, prdFolder=%q",
					v.taskFilePath, v.prdFolder)
			}
		})
	}
}

// TestBuildReviewPromptDeterminismo verifica que BuildReviewPrompt e deterministica:
// dado o mesmo templatePath, ReviewTemplateData e FileSystem, produz o mesmo output
// em chamadas distintas (RF-13, RF-14).
func TestBuildReviewPromptDeterminismo(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	customTemplate := `Review task: {{.TaskFile}}
PRD: {{.PRDFolder}}
Spec: {{.TechSpec}}
Tasks: {{.TasksFile}}
Diff: {{.Diff}}`
	_ = fsys.WriteFile("/custom/review.tmpl", []byte(customTemplate))

	tests := []struct {
		name         string
		templatePath string
		data         ReviewTemplateData
	}{
		{
			name:         "template default com dados completos",
			templatePath: "",
			data: ReviewTemplateData{
				TaskFile:  "tasks/prd-feat/task-1.0.md",
				PRDFolder: "tasks/prd-feat",
				TechSpec:  "tasks/prd-feat/techspec.md",
				TasksFile: "tasks/prd-feat/tasks.md",
				Diff:      "diff --git a/main.go b/main.go\n+// change",
			},
		},
		{
			name:         "template default com diff indisponivel",
			templatePath: "",
			data: ReviewTemplateData{
				TaskFile:  "tasks/prd-outro/task-2.0.md",
				PRDFolder: "tasks/prd-outro",
				TechSpec:  "tasks/prd-outro/techspec.md",
				TasksFile: "tasks/prd-outro/tasks.md",
				Diff:      "(diff indisponivel)",
			},
		},
		{
			name:         "template custom com dados completos",
			templatePath: "/custom/review.tmpl",
			data: ReviewTemplateData{
				TaskFile:  "tasks/prd-feat/task-1.0.md",
				PRDFolder: "tasks/prd-feat",
				TechSpec:  "tasks/prd-feat/techspec.md",
				TasksFile: "tasks/prd-feat/tasks.md",
				Diff:      "diff content here",
			},
		},
		{
			name:         "template default dados vazios",
			templatePath: "",
			data:         ReviewTemplateData{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			primeiro, err := BuildReviewPrompt(tt.templatePath, tt.data, fsys)
			if err != nil {
				t.Fatalf("BuildReviewPrompt (1a chamada) retornou erro inesperado: %v", err)
			}

			segundo, err := BuildReviewPrompt(tt.templatePath, tt.data, fsys)
			if err != nil {
				t.Fatalf("BuildReviewPrompt (2a chamada) retornou erro inesperado: %v", err)
			}

			if primeiro != segundo {
				t.Errorf("BuildReviewPrompt nao e deterministica: chamadas com mesmos inputs produziram outputs diferentes\nprimeiro:\n%s\nsegundo:\n%s",
					primeiro, segundo)
			}
		})
	}
}

// TestBuildReviewPromptInputsDiferentesProduzemOutputsDiferentes verifica que inputs distintos
// produzem outputs distintos para BuildReviewPrompt (propriedade complementar ao determinismo — RF-13).
func TestBuildReviewPromptInputsDiferentesProduzemOutputsDiferentes(t *testing.T) {
	fsys := fs.NewFakeFileSystem()

	baseData := ReviewTemplateData{
		TaskFile:  "tasks/prd-a/task-1.0.md",
		PRDFolder: "tasks/prd-a",
		TechSpec:  "tasks/prd-a/techspec.md",
		TasksFile: "tasks/prd-a/tasks.md",
		Diff:      "diff base",
	}

	base, err := BuildReviewPrompt("", baseData, fsys)
	if err != nil {
		t.Fatalf("BuildReviewPrompt base retornou erro: %v", err)
	}

	variantes := []struct {
		name string
		data ReviewTemplateData
	}{
		{
			name: "task file diferente",
			data: ReviewTemplateData{
				TaskFile:  "tasks/prd-a/task-2.0.md",
				PRDFolder: "tasks/prd-a",
				TechSpec:  "tasks/prd-a/techspec.md",
				TasksFile: "tasks/prd-a/tasks.md",
				Diff:      "diff base",
			},
		},
		{
			name: "prd folder diferente",
			data: ReviewTemplateData{
				TaskFile:  "tasks/prd-b/task-1.0.md",
				PRDFolder: "tasks/prd-b",
				TechSpec:  "tasks/prd-b/techspec.md",
				TasksFile: "tasks/prd-b/tasks.md",
				Diff:      "diff base",
			},
		},
		{
			name: "diff diferente",
			data: ReviewTemplateData{
				TaskFile:  "tasks/prd-a/task-1.0.md",
				PRDFolder: "tasks/prd-a",
				TechSpec:  "tasks/prd-a/techspec.md",
				TasksFile: "tasks/prd-a/tasks.md",
				Diff:      "diff alterado",
			},
		},
	}

	for _, v := range variantes {
		t.Run(v.name, func(t *testing.T) {
			variante, err := BuildReviewPrompt("", v.data, fsys)
			if err != nil {
				t.Fatalf("BuildReviewPrompt variante retornou erro: %v", err)
			}
			if variante == base {
				t.Errorf("BuildReviewPrompt com dados diferentes produziu o mesmo output que o caso base\ndados: %+v", v.data)
			}
		})
	}
}
