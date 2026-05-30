//go:build integration

package integration

// TestTokenBudget_PerSkill — TASK-018
//
// Regressao de budget de tokens por skill. Falha se SKILL.md + todas as referencias
// de uma skill ultrapassarem o budget maximo definido (pior caso = modo complex).
//
// Objetivo: prevenir inflacao silenciosa de tokens durante sprints.
// Regra: budget definido com margem de 10% sobre valor medido no momento da criacao.
// Atualizar os budgets abaixo quando uma skill crescer intencionalmente.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/metrics"
)

// skillBudgets define o budget maximo de tokens por skill (SKILL.md + todas as referencias).
// Valores com margem de 10% sobre o tamanho atual medido.
// Atualizar conscientemente ao expandir skills — essa e a intencao do teste.
var skillBudgets = map[string]int{
	"agent-governance":       12393, // SKILL.md + 15 refs (atualizado em 2026 apos multiple-choice-protocol.md + severity-mapping.md; margem 10% sobre 11267 medido)
	"go-implementation":      28049, // SKILL.md (R0-R7 inline, always-on) + 19 refs (atualizado em 2026-05-30 apos integracao das Regras Estritas v1.2.0, commit e09c09f; margem de 10% sobre 25499 medido)
	"object-calisthenics-go": 4000,  // SKILL.md + 3 refs
}

// TestTokenBudget_PerSkill verifica que nenhuma skill excede seu budget de tokens
// no pior caso (carregamento completo de todas as referencias).
func TestTokenBudget_PerSkill(t *testing.T) {
	root := govRepoRoot(t)
	embeddedDir := govEmbeddedSkillsDir(root)
	tokenizer := metrics.NewCharEstimator()

	for skillName, maxTokens := range skillBudgets {
		skillName := skillName
		maxTokens := maxTokens
		t.Run(skillName, func(t *testing.T) {
			t.Parallel()

			total, breakdown, err := estimateSkillTokens(embeddedDir, skillName, tokenizer)
			if err != nil {
				t.Fatalf("estimar tokens da skill %q: %v", skillName, err)
			}

			t.Logf("skill=%s skill_md=%d refs=%d total=%d budget=%d",
				skillName, breakdown.SkillMD, breakdown.Refs, total, maxTokens)

			if total > maxTokens {
				pct := float64(total-maxTokens) / float64(maxTokens) * 100
				t.Errorf(
					"skill %q excede budget: %d tokens > %d tokens (+%.1f%%)\n"+
						"  SKILL.md: %d tokens\n"+
						"  referencias: %d tokens\n"+
						"  Para aceitar esse crescimento, atualize skillBudgets[%q] = %d",
					skillName, total, maxTokens, pct,
					breakdown.SkillMD, breakdown.Refs,
					skillName, int(float64(total)*1.10),
				)
			}
		})
	}
}

// TestTokenBudget_AllSkillsHaveReasonableSize verifica que nenhuma skill embarcada
// excede um limite universal (protetor contra outliers nao listados em skillBudgets).
// Skills com budget explicito em skillBudgets sao governadas conscientemente por
// TestTokenBudget_PerSkill e ficam isentas deste limite universal: o objetivo aqui
// e capturar skills NAO listadas que cresceram silenciosamente, nao reaplicar um
// teto sobre limites ja aprovados de forma deliberada.
func TestTokenBudget_AllSkillsHaveReasonableSize(t *testing.T) {
	const universalMax = 25000 // limite absoluto para qualquer skill SEM budget explicito

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

			if _, hasExplicitBudget := skillBudgets[skillName]; hasExplicitBudget {
				t.Skipf("skill %q tem budget explicito em skillBudgets — governada por TestTokenBudget_PerSkill", skillName)
			}

			total, breakdown, err := estimateSkillTokens(embeddedDir, skillName, tokenizer)
			if err != nil {
				t.Skipf("skill %q: %v", skillName, err)
			}

			t.Logf("skill=%s skill_md=%d refs=%d total=%d",
				skillName, breakdown.SkillMD, breakdown.Refs, total)

			if total > universalMax {
				t.Errorf("skill %q tem %d tokens — acima do limite universal de %d",
					skillName, total, universalMax)
			}
		})
	}
}

// tokenBreakdown descreve o uso de tokens separado por componente.
type tokenBreakdown struct {
	SkillMD int
	Refs    int
}

// estimateSkillTokens calcula tokens totais de uma skill (SKILL.md + todas as referencias).
func estimateSkillTokens(embeddedDir, skillName string, tokenizer metrics.Tokenizer) (int, tokenBreakdown, error) {
	var bd tokenBreakdown

	skillMDPath := filepath.Join(embeddedDir, skillName, "SKILL.md")
	skillData, err := os.ReadFile(skillMDPath)
	if err != nil {
		return 0, bd, fmt.Errorf("ler SKILL.md: %w", err)
	}
	bd.SkillMD = tokenizer.EstimateTokens(string(skillData))

	refsDir := filepath.Join(embeddedDir, skillName, "references")
	refEntries, err := os.ReadDir(refsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return bd.SkillMD, bd, nil
		}
		return 0, bd, fmt.Errorf("ler diretorio references: %w", err)
	}

	for _, ref := range refEntries {
		if ref.IsDir() || filepath.Ext(ref.Name()) != ".md" {
			continue
		}
		refPath := filepath.Join(refsDir, ref.Name())
		refData, err := os.ReadFile(refPath)
		if err != nil {
			continue
		}
		bd.Refs += tokenizer.EstimateTokens(string(refData))
	}

	return bd.SkillMD + bd.Refs, bd, nil
}
