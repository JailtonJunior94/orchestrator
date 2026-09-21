package hookinventory

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skillscheck"
	"github.com/stretchr/testify/require"
)

func TestInventoryGenerateAndCheck(t *testing.T) {
	filesystems := []struct {
		name   string
		mutate func(*fs.FakeFileSystem)
		valid  bool
	}{
		{name: "generated projections match the scanned sources", valid: true},
		{
			name: "new hook without regenerated projection fails", valid: false,
			mutate: func(filesystem *fs.FakeFileSystem) {
				filesystem.Files["repo/.agents/hooks/new-hook.sh"] = []byte("#!/usr/bin/env bash")
			},
		},
		{
			name: "orphaned projection entry fails", valid: false,
			mutate: func(filesystem *fs.FakeFileSystem) {
				delete(filesystem.Files, "repo/.agents/hooks/validate-preload.sh")
			},
		},
	}
	for _, scenario := range filesystems {
		t.Run(scenario.name, func(t *testing.T) {
			filesystem := newInventoryFileSystem()
			printer := &output.Printer{Out: io.Discard, Err: io.Discard}
			service := NewService(filesystem, printer)
			_, err := service.Generate("repo")
			require.NoError(t, err)
			if scenario.mutate != nil {
				scenario.mutate(filesystem)
			}
			err = service.Check("repo")
			if scenario.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

func newInventoryFileSystem() *fs.FakeFileSystem {
	filesystem := fs.NewFakeFileSystem()
	paths := []string{
		".agents/hooks/validate-preload.sh", ".claude/hooks/post-wave.sh", ".codex/hooks/validate-preload.sh",
		".github/hooks/governance.json", ".agents/scripts/validate-governance-references.sh",
		".claude/scripts/validate-task-evidence.sh", "internal/runtime/hooks/governance.go",
		".opencode/plugin/governance.js", ".agents/lib/parse-hook-input.sh", "scripts/git-hooks/pre-commit",
	}
	for _, path := range paths {
		filesystem.Files["repo/"+path] = []byte("content")
	}
	filesystem.Files["repo/skills-lock.json"] = []byte(`{"version":1,"skills":{},"hooks":{}}`)
	return filesystem
}

func TestClassificationRemoveDoesNotAuthorizeDeletion(t *testing.T) {
	filesystem := newInventoryFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	service := NewService(filesystem, printer)

	entries, err := service.Generate("repo")
	require.NoError(t, err)

	removeEntries := make([]Entry, 0)
	for _, entry := range entries {
		if entry.Classification == "REMOVE" {
			removeEntries = append(removeEntries, entry)
		}
	}
	require.NotEmpty(t, removeEntries, "esperado ao menos uma entrada classificada REMOVE para validar RF-04")

	for _, entry := range removeEntries {
		_, readErr := filesystem.ReadFile(filepath.Join("repo", entry.Location))
		require.NoErrorf(t, readErr, "RF-04: hook classificado REMOVE nao pode ser removido sem cobertura previa do invariante no core (%s)", entry.Location)
	}

	require.NoError(t, service.Check("repo"))
}

func TestGenerateSyncsSkillsLockAndCheckDetectsHookDrift(t *testing.T) {
	filesystem := newInventoryFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	service := NewService(filesystem, printer)

	entries, err := service.Generate("repo")
	require.NoError(t, err)

	criticalEntries := 0
	for _, entry := range entries {
		if entry.IntegrityHash != "" {
			criticalEntries++
		}
	}
	require.Positive(t, criticalEntries, "esperado ao menos um hook critico com IntegrityHash")

	lockData, err := filesystem.ReadFile("repo/skills-lock.json")
	require.NoError(t, err)
	var lock skillscheck.LockFile
	require.NoError(t, json.Unmarshal(lockData, &lock))
	require.Contains(t, lock.Hooks, ".agents/hooks/validate-preload.sh")
	require.Equal(t, ".agents/hooks/validate-preload.sh", lock.Hooks[".agents/hooks/validate-preload.sh"].Path)

	require.NoError(t, service.Check("repo"))

	skillcheckService := skillscheck.NewService(filesystem, printer)
	failures, err := skillcheckService.Verify("repo")
	require.NoError(t, err)
	require.Empty(t, failures)

	filesystem.Files["repo/.agents/hooks/validate-preload.sh"] = []byte("content-alterado")

	failures, err = skillcheckService.Verify("repo")
	require.NoError(t, err)
	require.Len(t, failures, 1, "RF-70: alteracao em hook critico deve ser detectavel pelo check de integridade (skills-lock.json)")
	require.Equal(t, "hash diverge do registrado em skills-lock.json", failures[0].Reason)

	err = service.Check("repo")
	require.Error(t, err, "Check deve falhar quando o inventario diverge do hook alterado antes de reexecutar Generate")

	_, err = service.Generate("repo")
	require.NoError(t, err)
	require.NoError(t, service.Check("repo"))

	failures, err = skillcheckService.Verify("repo")
	require.NoError(t, err)
	require.Empty(t, failures, "skills-lock.json deve refletir o novo hash apos regenerar o inventario")
}

func TestFailOpenCitationsMatchRealHookContent(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	cases := []struct {
		name    string
		path    string
		pattern *regexp.Regexp
	}{
		{
			name:    "validate-governance.sh missing hook-payload.sh dependency",
			path:    ".agents/hooks/validate-governance.sh",
			pattern: governancePayloadMissingPattern,
		},
		{
			name:    "validate-governance.sh missing file_path target",
			path:    ".agents/hooks/validate-governance.sh",
			pattern: governanceMissingTargetPattern,
		},
		{
			name:    "validate-preload.sh unrecognized extension",
			path:    ".agents/hooks/validate-preload.sh",
			pattern: preloadUnrecognizedExtPattern,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(repoRoot, testCase.path))
			require.NoError(t, err)
			_, ok := lineOfPattern(string(data), testCase.pattern)
			require.Truef(t, ok, "padrao de fail-open nao encontrado em %s; a citacao em failOpen() nao acompanha mais o conteudo real do arquivo", testCase.path)
		})
	}

	t.Run("git-operation-gate.sh payload capture block", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(repoRoot, ".agents/scripts/git-operation-gate.sh"))
		require.NoError(t, err)
		_, _, ok := lineRangeOfPattern(string(data), gitOperationPayloadStartPattern, gitOperationPayloadEndPattern, 5)
		require.True(t, ok, "bloco de captura de payload (fail-open) nao encontrado em git-operation-gate.sh; a citacao em failOpen() nao acompanha mais o conteudo real do arquivo")
	})

	filesystem := fs.NewFakeFileSystem()
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	service := NewService(filesystem, printer)

	governanceData, err := os.ReadFile(filepath.Join(repoRoot, ".agents/hooks/validate-governance.sh"))
	require.NoError(t, err)
	missingLibLine, ok := lineOfPattern(string(governanceData), governancePayloadMissingPattern)
	require.True(t, ok)
	missingTargetLine, ok := lineOfPattern(string(governanceData), governanceMissingTargetPattern)
	require.True(t, ok)
	mode, ok := service.failOpen(".agents/hooks/validate-governance.sh", governanceData)
	require.True(t, ok)
	require.Contains(t, mode, strconv.Itoa(missingLibLine)+","+strconv.Itoa(missingTargetLine))

	preloadData, err := os.ReadFile(filepath.Join(repoRoot, ".agents/hooks/validate-preload.sh"))
	require.NoError(t, err)
	unrecognizedExtLine, ok := lineOfPattern(string(preloadData), preloadUnrecognizedExtPattern)
	require.True(t, ok)
	mode, ok = service.failOpen(".agents/hooks/validate-preload.sh", preloadData)
	require.True(t, ok)
	require.Contains(t, mode, ":"+strconv.Itoa(unrecognizedExtLine))

	gitOperationGateData, err := os.ReadFile(filepath.Join(repoRoot, ".agents/scripts/git-operation-gate.sh"))
	require.NoError(t, err)
	blockStart, blockEnd, ok := lineRangeOfPattern(string(gitOperationGateData), gitOperationPayloadStartPattern, gitOperationPayloadEndPattern, 5)
	require.True(t, ok)
	mode, ok = service.failOpen(".agents/scripts/git-operation-gate.sh", gitOperationGateData)
	require.True(t, ok)
	require.Contains(t, mode, strconv.Itoa(blockStart)+"-"+strconv.Itoa(blockEnd))
}
