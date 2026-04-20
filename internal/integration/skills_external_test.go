//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// externalRefsRe extrai mencoes a references/xxx.md em SKILL.md.
var externalRefsRe = regexp.MustCompile("`references/([a-zA-Z0-9._-]+\\.md)`")

func TestExternalSkills_FrontmatterAndReferences(t *testing.T) {
	root := govRepoRoot(t)
	lockPath := filepath.Join(root, "skills-lock.json")

	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("ler skills-lock.json: %v", err)
	}
	var lock lockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("parsear skills-lock.json: %v", err)
	}

	installedDir := filepath.Join(root, ".agents", "skills")

	for skillName := range lock.Skills {
		skillName := skillName
		t.Run(skillName, func(t *testing.T) {
			t.Parallel()
			skillDir := filepath.Join(installedDir, skillName)

			// Check directory exists
			info, err := os.Stat(skillDir)
			if err != nil {
				t.Skipf("skill %q nao instalada (diretorio ausente): %v", skillName, err)
				return
			}
			if !info.IsDir() {
				t.Fatalf("esperado diretorio para skill %q, encontrado arquivo", skillName)
			}

			// Check SKILL.md exists
			skillMDPath := filepath.Join(skillDir, "SKILL.md")
			skillData, err := os.ReadFile(skillMDPath)
			if err != nil {
				t.Fatalf("SKILL.md nao encontrado para skill externa %q: %v", skillName, err)
			}

			// Validate frontmatter via JSON Schema
			if err := skills.ValidateFrontmatterSchema(skillData, skillName); err != nil {
				t.Errorf("skill externa %q: frontmatter invalido: %v", skillName, err)
			}

			// Validate name field matches directory name
			fm := skills.ParseFrontmatter(skillData)
			if fm.Name != "" && fm.Name != skillName {
				t.Errorf("skill %q: campo name %q diverge do nome do diretorio", skillName, fm.Name)
			}

			// Validate all referenced files exist
			matches := externalRefsRe.FindAllSubmatch(skillData, -1)
			for _, m := range matches {
				refFile := string(m[1])
				refPath := filepath.Join(skillDir, "references", refFile)
				if _, err := os.Stat(refPath); err != nil {
					t.Errorf("skill %q: referencia %q declarada no SKILL.md mas nao existe: %s",
						skillName, "references/"+refFile, refPath)
				}
			}
		})
	}
}
