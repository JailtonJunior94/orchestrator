package hookinventory

import (
	"io"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
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
	return filesystem
}
