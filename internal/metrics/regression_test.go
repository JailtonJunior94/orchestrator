package metrics

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type RegressionSuite struct {
	suite.Suite
}

func TestRegressionSuite(t *testing.T) {
	suite.Run(t, new(RegressionSuite))
}

// costBaseline e o formato de testdata/baselines/cost-baseline.json.
// Versionado por release para detectar crescimento silencioso de custo contextual.
type costBaseline struct {
	Version      string                   `json:"version"`
	GeneratedAt  string                   `json:"generated_at"`
	Note         string                   `json:"note"`
	TolerancePct int                      `json:"tolerance_pct"`
	Skills       map[string]CostBreakdown `json:"skills"`
}

func (s *RegressionSuite) TestCostRegressionCanonicalSkill() {
	baselineData, err := os.ReadFile("../../testdata/baselines/cost-baseline.json")
	s.NoError(err, "baseline nao encontrado em testdata/baselines/cost-baseline.json")

	var bl costBaseline
	err = json.Unmarshal(baselineData, &bl)
	s.NoError(err, "falha ao parsear cost-baseline.json")
	s.True(bl.TolerancePct > 0, "tolerance_pct deve ser positivo no baseline")

	svc := NewService(fs.NewOSFileSystem(), silentPrinter(), nil)
	report, err := svc.gather("../../testdata/baselines", false)
	s.NoError(err, "gather falhou sobre testdata/baselines")

	for skillName, expected := range bl.Skills {
		actual, ok := report.Baselines[skillName]
		if !s.True(ok, "skill %q nao encontrada no inventario atual (testdata/baselines/.agents/skills/)", skillName) {
			continue
		}

		s.T().Logf("skill=%q on_disk=%d loaded=%d incremental_ref=%d ref_count=%d",
			skillName,
			actual.Cost.OnDisk, actual.Cost.Loaded, actual.Cost.IncrementalRef, actual.Cost.RefCount)

		s.checkCostAxis(skillName, "on-disk", expected.OnDisk, actual.Cost.OnDisk, bl.TolerancePct)
		s.checkCostAxis(skillName, "loaded", expected.Loaded, actual.Cost.Loaded, bl.TolerancePct)
		if expected.IncrementalRef > 0 {
			s.checkCostAxis(skillName, "incremental-ref", expected.IncrementalRef, actual.Cost.IncrementalRef, bl.TolerancePct)
		}
	}
}

func (s *RegressionSuite) checkCostAxis(skill, axis string, baseline, actual, tolerancePct int) {
	s.T().Helper()
	if baseline == 0 {
		return
	}
	limit := int(math.Round(float64(baseline) * (1 + float64(tolerancePct)/100)))
	s.True(actual <= limit,
		"gate de regressao falhou: skill=%q eixo=%s | atual=%d > limite=%d (baseline=%d + %d%%) — "+
			"verifique se o crescimento e intencional e atualize cost-baseline.json",
		skill, axis, actual, limit, baseline, tolerancePct)
}
