//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type workflowFile struct {
	On   map[string]interface{} `yaml:"on"`
	Jobs map[string]struct {
		Steps []struct {
			Run string `yaml:"run"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func loadWorkflow(t *testing.T, path string) workflowFile {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	var wf workflowFile
	if err := yaml.Unmarshal(data, &wf); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if wf.On == nil {
		t.Fatalf("%s: chave 'on' nao encontrada ou vazia apos parse", path)
	}
	return wf
}

func allRunCommands(wf workflowFile) []string {
	var out []string
	for _, job := range wf.Jobs {
		for _, step := range job.Steps {
			if step.Run != "" {
				out = append(out, step.Run)
			}
		}
	}
	return out
}

func TestReleasePolicy_DeterministicScenariosGateMergeLiveScenariosDoNot(t *testing.T) {
	repoRoot := findRepoRoot(t)
	testWF := loadWorkflow(t, filepath.Join(repoRoot, ".github", "workflows", "test.yml"))
	liveWorkflows := map[string]workflowFile{
		"hooks-live.yml": loadWorkflow(t, filepath.Join(repoRoot, ".github", "workflows", "hooks-live.yml")),
		"acp-live.yml":   loadWorkflow(t, filepath.Join(repoRoot, ".github", "workflows", "acp-live.yml")),
	}

	if _, ok := testWF.On["pull_request"]; !ok {
		t.Fatalf("test.yml (RF-36, baseline determinístico) precisa bloquear merge via pull_request; on=%v", testWF.On)
	}

	for name, wf := range liveWorkflows {
		if _, ok := wf.On["pull_request"]; ok {
			t.Errorf("%s (RF-37, cenario live) nao pode bloquear merge; nao deveria disparar em pull_request", name)
		}
		_, hasSchedule := wf.On["schedule"]
		_, hasDispatch := wf.On["workflow_dispatch"]
		if !hasSchedule && !hasDispatch {
			t.Errorf("%s deveria ser nightly (schedule) ou manual (workflow_dispatch); on=%v", name, wf.On)
		}
	}

	mergeGateCommands := allRunCommands(testWF)
	requiredDeterministicCommands := []string{
		"go test -coverprofile=coverage.out ./...",
		"-tags=integration",
		"go vet ./...",
		"make lint",
		"make budget",
		"make check-skills-sync",
		"make check-hooks-sync",
		"make check-scripts-sync",
	}
	for _, want := range requiredDeterministicCommands {
		found := false
		for _, cmd := range mergeGateCommands {
			if strings.Contains(cmd, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("test.yml (RF-36/RF-48, baseline determinístico) deveria executar %q como gate de merge", want)
		}
	}

	for name, wf := range liveWorkflows {
		for _, cmd := range allRunCommands(wf) {
			if strings.Contains(cmd, "-coverprofile=coverage.out") || strings.Contains(cmd, "-tags=integration") {
				t.Errorf("%s nao deveria duplicar a suite determinística; RF-36 vs RF-37", name)
			}
		}
	}
}
