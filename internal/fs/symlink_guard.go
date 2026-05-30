package fs

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RefuseExternalSymlink valida que o destino nao atravessa symlink para fora do projeto.
func RefuseExternalSymlink(filesystem FileSystem, projectDir, destination string, followExternal bool) error {
	if followExternal {
		return nil
	}

	absProject, err := filepath.Abs(projectDir)
	if err != nil {
		return fmt.Errorf("resolver caminho do projeto: %w", err)
	}
	absProject = filepath.Clean(absProject)

	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolver caminho de destino: %w", err)
	}
	absDestination = filepath.Clean(absDestination)

	for _, path := range symlinkAncestors(absProject, absDestination) {
		if !filesystem.IsSymlink(path) {
			continue
		}
		resolved, err := filesystem.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolver symlink %s: %w", path, err)
		}
		resolvedAbs, err := filepath.Abs(resolved)
		if err != nil {
			return fmt.Errorf("resolver destino do symlink %s: %w", path, err)
		}
		resolvedAbs = filepath.Clean(resolvedAbs)
		if pathInsideRoot(absProject, resolvedAbs) {
			continue
		}
		rel, err := filepath.Rel(absProject, path)
		if err != nil {
			rel = path
		}
		return fmt.Errorf("'%s' e symlink para fora do projeto: %s -> %s; escrever aqui mutaria repositorio externo (use --follow-external-symlinks)", rel, rel, resolvedAbs)
	}

	return nil
}

func symlinkAncestors(absProject, absDestination string) []string {
	paths := []string{absProject}
	current := absProject
	rel, err := filepath.Rel(absProject, absDestination)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return paths
	}
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "." || part == "" {
			continue
		}
		current = filepath.Join(current, part)
		paths = append(paths, current)
	}
	return paths
}

func pathInsideRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}
