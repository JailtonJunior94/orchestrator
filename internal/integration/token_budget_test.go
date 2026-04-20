//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/metrics"
)

// budgetSkillMD e o limite maximo de tokens estimados para o SKILL.md de uma skill.
// ADR-004 define lazy-loading: apenas SKILL.md e carregado inicialmente (~3000 tokens).
// Referencias sao carregadas sob demanda, nao contam para o budget base.
const budgetSkillMD = 4000

// budgetSingleRef e o limite maximo de tokens para uma unica referencia.
// Cada referencia carregada incrementalmente nao deve exceder este limite.
const budgetSingleRef = 3000

func TestTokenBudget_EmbeddedSkills(t *testing.T) {
	root := govRepoRoot(t)
	embeddedDir := govEmbeddedSkillsDir(root)
	tokenizer := metrics.NewCharEstimator()

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
			t.Parallel()
			skillMDPath := filepath.Join(embeddedDir, skillName, "SKILL.md")
			skillData, err := os.ReadFile(skillMDPath)
			if err != nil {
				t.Fatalf("SKILL.md nao encontrado para %q: %v", skillName, err)
			}

			skillTokens := tokenizer.EstimateTokens(string(skillData))

			// Validar budget do SKILL.md (carregado inicialmente)
			t.Logf("skill=%s skill_tokens=%d budget=%d", skillName, skillTokens, budgetSkillMD)
			if skillTokens > budgetSkillMD {
				t.Errorf("skill %q: SKILL.md tem %d tokens, excede budget de %d tokens",
					skillName, skillTokens, budgetSkillMD)
			}

			// Validar budget individual de cada referencia (carregada sob demanda)
			matches := referencesRegexp.FindAllSubmatch(skillData, -1)
			for _, m := range matches {
				refFile := string(m[1])
				refPath := filepath.Join(embeddedDir, skillName, "references", refFile)
				refData, err := os.ReadFile(refPath)
				if err != nil {
					continue
				}
				refTokens := tokenizer.EstimateTokens(string(refData))
				if refTokens > budgetSingleRef {
					t.Errorf("skill %q: referencia %q tem %d tokens, excede budget de %d tokens",
						skillName, refFile, refTokens, budgetSingleRef)
				}
			}
		})
	}
}
