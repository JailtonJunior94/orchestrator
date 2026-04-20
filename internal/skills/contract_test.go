//go:build integration

package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// skillsLock mirrors the structure of skills-lock.json.
type skillsLock struct {
	Version int                      `json:"version"`
	Skills  map[string]skillLockEntry `json:"skills"`
}

type skillLockEntry struct {
	Source       string `json:"source"`
	SourceType   string `json:"sourceType"`
	ComputedHash string `json:"computedHash"`
}

// repoRoot returns the absolute path to the repository root (two levels up from internal/skills/).
func repoRoot(t *testing.T) string {
	t.Helper()
	// This file lives in internal/skills/, so go up two directories.
	dir, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Fatalf("failed to resolve repo root: %v", err)
	}
	return dir
}

func readLockFile(t *testing.T, root string) skillsLock {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "skills-lock.json"))
	if err != nil {
		t.Fatalf("failed to read skills-lock.json: %v", err)
	}
	var lock skillsLock
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("failed to parse skills-lock.json: %v", err)
	}
	return lock
}

func TestComplementarySkills_HaveLockEntry(t *testing.T) {
	root := repoRoot(t)
	lock := readLockFile(t, root)

	tests := make([]struct {
		skill string
	}, len(ComplementarySkills))
	for i, s := range ComplementarySkills {
		tests[i].skill = s
	}

	for _, tt := range tests {
		t.Run(tt.skill, func(t *testing.T) {
			if _, ok := lock.Skills[tt.skill]; !ok {
				t.Errorf("complementary skill %q is missing from skills-lock.json", tt.skill)
			}
		})
	}
}

func TestLockedSkills_DirectoryExists(t *testing.T) {
	root := repoRoot(t)
	lock := readLockFile(t, root)
	installedDir := filepath.Join(root, ".agents", "skills")

	for _, skill := range ComplementarySkills {
		t.Run(skill, func(t *testing.T) {
			if _, ok := lock.Skills[skill]; !ok {
				t.Skipf("skill %q not in lock file", skill)
			}
			dir := filepath.Join(installedDir, skill)
			info, err := os.Stat(dir)
			if err != nil {
				t.Fatalf("installed skill directory does not exist: %s", dir)
			}
			if !info.IsDir() {
				t.Fatalf("expected directory but found file: %s", dir)
			}
		})
	}
}

func TestInstalledSkills_ValidFrontmatter(t *testing.T) {
	root := repoRoot(t)
	installedDir := filepath.Join(root, ".agents", "skills")

	for _, skill := range ComplementarySkills {
		t.Run(skill, func(t *testing.T) {
			skillMD := filepath.Join(installedDir, skill, "SKILL.md")
			data, err := os.ReadFile(skillMD)
			if err != nil {
				t.Fatalf("SKILL.md not found: %v", err)
			}

			fm := ParseFrontmatter(data)
			if fm.Name == "" {
				t.Errorf("SKILL.md frontmatter has empty Name for skill %q", skill)
			}
		})
	}
}

func TestEmbeddedSkills_ValidSchema(t *testing.T) {
	root := repoRoot(t)
	embeddedDir := filepath.Join(root, "internal", "embedded", "assets", ".agents", "skills")

	entries, err := os.ReadDir(embeddedDir)
	if err != nil {
		t.Fatalf("ler diretorio de skills embarcadas: %v", err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skillName := e.Name()
		t.Run(skillName, func(t *testing.T) {
			skillMD := filepath.Join(embeddedDir, skillName, "SKILL.md")
			data, err := os.ReadFile(skillMD)
			if err != nil {
				t.Fatalf("SKILL.md nao encontrado: %v", err)
			}
			if err := ValidateFrontmatterSchema(data, skillName); err != nil {
				t.Errorf("skill embarcada %q falhou no JSON Schema: %v", skillName, err)
			}
		})
	}
}
