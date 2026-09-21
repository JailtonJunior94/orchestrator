package hooksync

import (
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/tracking"
)

var OrchestratorHooks = []string{
	"post-execute-task.sh",
	"pre-execute-all-tasks.sh",
	"post-wave.sh",
	"subagent-stop-wrapper.sh",
}

var ToolValidationHooks = []string{
	"validate-preload.sh",
	"validate-governance.sh",
	"validate-session-end.sh",
}

var AgentsScriptsFiles = []string{
	"validate-task-evidence.sh",
	"validate-bugfix-evidence.sh",
	"validate-refactor-evidence.sh",
	"validate-review-evidence.sh",
	"hook-prereq-gate.sh",
	"resolve-references.sh",
	"validate-skill-prerequisites.sh",
	"validate-governance-references.sh",
	"validate-session-end.sh",
	"git-operation-gate.sh",
}

var AgentsLibFiles = []string{
	"check-invocation-depth.sh",
	"parse-hook-input.sh",
	"hook-payload.sh",
}

func markExecutable(fsys fs.FileSystem, path string) {
	if tracker, ok := fsys.(*tracking.Tracker); ok {
		tracker.MarkExecutable(path)
	}
}

func copyFiles(fsys fs.FileSystem, srcDir, dstDir string, names []string) error {
	for _, name := range names {
		src := filepath.Join(srcDir, name)
		if !fsys.Exists(src) {
			continue
		}
		if err := fsys.MkdirAll(dstDir); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, name)
		if err := fsys.CopyFile(src, dst); err != nil {
			return err
		}
		markExecutable(fsys, dst)
	}
	return nil
}

func CopyOrchestratorHooks(fsys fs.FileSystem, sourceDir, projectDir, toolHookDir string) error {
	return copyFiles(
		fsys,
		filepath.Join(sourceDir, toolHookDir),
		filepath.Join(projectDir, toolHookDir),
		OrchestratorHooks,
	)
}

func CopyToolValidationHooks(fsys fs.FileSystem, sourceDir, projectDir, toolHookDir string) error {
	return copyFiles(
		fsys,
		filepath.Join(sourceDir, toolHookDir),
		filepath.Join(projectDir, toolHookDir),
		ToolValidationHooks,
	)
}

func CopyAgentsScripts(fsys fs.FileSystem, sourceDir, projectDir string) error {
	dstDir := filepath.Join(projectDir, ".agents", "scripts")
	primarySrc := filepath.Join(sourceDir, ".agents", "scripts")
	fallbackSrc := filepath.Join(sourceDir, ".claude", "scripts")

	for _, name := range AgentsScriptsFiles {
		src := filepath.Join(primarySrc, name)
		if !fsys.Exists(src) {
			src = filepath.Join(fallbackSrc, name)
		}
		if !fsys.Exists(src) {
			continue
		}
		if err := fsys.MkdirAll(dstDir); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, name)
		if err := fsys.CopyFile(src, dst); err != nil {
			return err
		}
		markExecutable(fsys, dst)
	}
	return nil
}

func CopyAgentsLib(fsys fs.FileSystem, sourceDir, projectDir string) error {
	return copyFiles(
		fsys,
		filepath.Join(sourceDir, ".agents", "lib"),
		filepath.Join(projectDir, ".agents", "lib"),
		AgentsLibFiles,
	)
}

func SyncAll(fsys fs.FileSystem, sourceDir, projectDir string, toolHookDirs []string) error {
	for _, toolHookDir := range toolHookDirs {
		if err := CopyOrchestratorHooks(fsys, sourceDir, projectDir, toolHookDir); err != nil {
			return err
		}
		if err := CopyToolValidationHooks(fsys, sourceDir, projectDir, toolHookDir); err != nil {
			return err
		}
	}
	if err := CopyAgentsScripts(fsys, sourceDir, projectDir); err != nil {
		return err
	}
	if err := CopyAgentsLib(fsys, sourceDir, projectDir); err != nil {
		return err
	}
	return nil
}
