package metrics

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

// costBaseline e o formato de testdata/baselines/cost-baseline.json.
// Versionado por release para detectar crescimento silencioso de custo contextual.
type costBaseline struct {
	Version      string                   `json:"version"`
	GeneratedAt  string                   `json:"generated_at"`
	Note         string                   `json:"note"`
	TolerancePct int                      `json:"tolerance_pct"`
	Skills       map[string]CostBreakdown `json:"skills"`
}

// TestCostRegression_CanonicalSkill e o gate de regressao de custo por release.
// Falha quando o custo dos artefatos canonicos excede o baseline versionado pela
// tolerancia definida em testdata/baselines/cost-baseline.json.
//
// Para atualizar o baseline apos uma mudanca intencional nos artefatos:
//  1. Verificar que o crescimento e esperado e revisado.
//  2. Executar 'go test ./internal/metrics/... -v -run TestCostRegression' para ver os valores atuais.
//  3. Atualizar os valores em testdata/baselines/cost-baseline.json.
//  4. Atualizar o campo 'version' e 'generated_at' para o novo release.
func TestCostRegression_CanonicalSkill(t *testing.T) {
	baselineData, err := os.ReadFile("../../testdata/baselines/cost-baseline.json")
	if err != nil {
		t.Fatalf("baseline nao encontrado em testdata/baselines/cost-baseline.json: %v", err)
	}

	var bl costBaseline
	if err := json.Unmarshal(baselineData, &bl); err != nil {
		t.Fatalf("falha ao parsear cost-baseline.json: %v", err)
	}

	if bl.TolerancePct <= 0 {
		t.Fatal("tolerance_pct deve ser positivo no baseline")
	}

	svc := NewService(fs.NewOSFileSystem(), silentPrinter(), nil)
	report, err := svc.gather("../../testdata/baselines", false)
	if err != nil {
		t.Fatalf("gather falhou sobre testdata/baselines: %v", err)
	}

	for skillName, expected := range bl.Skills {
		actual, ok := report.Baselines[skillName]
		if !ok {
			t.Errorf("skill %q nao encontrada no inventario atual (testdata/baselines/.agents/skills/)", skillName)
			continue
		}

		t.Logf("skill=%q on_disk=%d loaded=%d incremental_ref=%d ref_count=%d",
			skillName,
			actual.Cost.OnDisk, actual.Cost.Loaded, actual.Cost.IncrementalRef, actual.Cost.RefCount)

		checkCostAxis(t, skillName, "on-disk", expected.OnDisk, actual.Cost.OnDisk, bl.TolerancePct)
		checkCostAxis(t, skillName, "loaded", expected.Loaded, actual.Cost.Loaded, bl.TolerancePct)
		if expected.IncrementalRef > 0 {
			checkCostAxis(t, skillName, "incremental-ref", expected.IncrementalRef, actual.Cost.IncrementalRef, bl.TolerancePct)
		}
	}
}

// checkCostAxis verifica que o custo atual nao excede baseline × (1 + tolerance/100).
// Baseline zero e ignorado (eixo nao definido).
func checkCostAxis(t *testing.T, skill, axis string, baseline, actual, tolerancePct int) {
	t.Helper()
	if baseline == 0 {
		return
	}
	limit := int(math.Round(float64(baseline) * (1 + float64(tolerancePct)/100)))
	if actual > limit {
		t.Errorf(
			"gate de regressao falhou: skill=%q eixo=%s | atual=%d > limite=%d (baseline=%d + %d%%) — "+
				"verifique se o crescimento e intencional e atualize cost-baseline.json",
			skill, axis, actual, limit, baseline, tolerancePct)
	}
}
