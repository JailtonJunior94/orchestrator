package detect

import (
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func (d *ToolchainDetector) detectDotNet(projectDir string) (ToolchainEntry, bool) {
	if !d.detectDotNetManifest(projectDir) {
		return ToolchainEntry{}, false
	}
	return ToolchainEntry{
		Fmt:  "dotnet format --verify-no-changes",
		Test: "dotnet test --no-build",
		Lint: "dotnet build --no-restore",
	}, true
}

func (d *ToolchainDetector) detectDotNetManifest(projectDir string) bool {
	if d.fs.Exists(filepath.Join(projectDir, "global.json")) {
		return true
	}
	return len(d.findManifestsBySuffix(projectDir, ".csproj")) > 0 ||
		len(d.findManifestsBySuffix(projectDir, ".sln")) > 0
}

func (d *ToolchainDetector) findManifestsBySuffix(projectDir, suffix string) []string {
	return NewCatalog().findManifestsBySuffixRecursive(d.fs, projectDir, suffix, d.maxDepth)
}

func (r1 *Catalog) findManifestsBySuffixRecursive(fsys fs.FileSystem, baseDir, suffix string, maxDepth int) []string {
	var results []string
	NewCatalog().findManifestsBySuffixHelper(fsys, baseDir, suffix, 0, maxDepth, &results)
	return results
}

func (r1 *Catalog) findManifestsBySuffixHelper(fsys fs.FileSystem, dir, suffix string, depth, maxDepth int, results *[]string) {
	if depth > maxDepth {
		return
	}

	ignoreDirs := map[string]bool{
		"node_modules": true, "vendor": true, "dist": true,
		"build": true, "__pycache__": true, ".git": true, "bin": true, "obj": true,
	}

	entries, err := fsys.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() {
			if ignoreDirs[e.Name()] {
				continue
			}
			NewCatalog().findManifestsBySuffixHelper(fsys, filepath.Join(dir, e.Name()), suffix, depth+1, maxDepth, results)
			continue
		}
		if strings.HasSuffix(e.Name(), suffix) {
			*results = append(*results, filepath.Join(dir, e.Name()))
		}
	}
}
