package telemetry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTestLog(t *testing.T, dir string, lines []string) {
	t.Helper()
	logPath := filepath.Join(dir, ".agents", "telemetry.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
}

func TestTrend_ReturnsFourWeeks(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().UTC()
	// Adicionar entradas desta semana
	for i := 0; i < 3; i++ {
		entry := now.Format(time.RFC3339) + " skill=go-implementation"
		writeTestLog(t, dir, []string{entry})
	}

	trend, err := Trend(dir)
	if err != nil {
		t.Fatalf("Trend: %v", err)
	}

	if len(trend.Weeks) != weeksForTrend {
		t.Errorf("esperado %d semanas, got %d", weeksForTrend, len(trend.Weeks))
	}
}

func TestTrend_EmptyLog_ReturnsFourEmptyWeeks(t *testing.T) {
	dir := t.TempDir()

	trend, err := Trend(dir)
	if err != nil {
		t.Fatalf("Trend: %v", err)
	}

	if len(trend.Weeks) != weeksForTrend {
		t.Errorf("esperado %d semanas, got %d", weeksForTrend, len(trend.Weeks))
	}
	for _, w := range trend.Weeks {
		if w.Invocations != 0 {
			t.Errorf("semana %s: esperado 0 invocacoes, got %d", w.Week, w.Invocations)
		}
	}
}

func TestFormatTrend_ContainsWeekLabels(t *testing.T) {
	data := TrendData{
		Weeks: []WeeklyBucket{
			{Week: "2026-W15", Invocations: 5},
			{Week: "2026-W16", Invocations: 10},
			{Week: "2026-W17", Invocations: 2},
			{Week: "2026-W18", Invocations: 0},
		},
	}
	result := FormatTrend(data)
	for _, w := range data.Weeks {
		if !containsStr(result, w.Week) {
			t.Errorf("FormatTrend deveria conter %q", w.Week)
		}
	}
}

func TestBudgetCheck_NoAlerts_WhenUnderBudget(t *testing.T) {
	dir := t.TempDir()
	// Nenhum log = sem alertas
	bc, err := BudgetCheck(dir, 0)
	if err != nil {
		t.Fatalf("BudgetCheck: %v", err)
	}
	if !bc.OK {
		t.Error("esperado OK=true quando nenhum dado de telemetria")
	}
	if len(bc.Alerts) != 0 {
		t.Errorf("esperado 0 alertas, got %d", len(bc.Alerts))
	}
}

func TestBudgetCheck_Alert_WhenOverBudget(t *testing.T) {
	dir := t.TempDir()

	now := time.Now().UTC()
	var lines []string
	// Gerar 600 invocacoes de agent-governance (budget = 500)
	for i := 0; i < 600; i++ {
		lines = append(lines, now.Format(time.RFC3339)+" skill=agent-governance")
	}
	writeTestLog(t, dir, lines)

	bc, err := BudgetCheck(dir, 0)
	if err != nil {
		t.Fatalf("BudgetCheck: %v", err)
	}
	if bc.OK {
		t.Error("esperado OK=false quando skill excede budget")
	}
	if len(bc.Alerts) == 0 {
		t.Error("esperado pelo menos 1 alerta")
	}
	found := false
	for _, a := range bc.Alerts {
		if a.Skill == "agent-governance" {
			found = true
			if a.Invocations < 600 {
				t.Errorf("invocacoes reportadas (%d) < esperado (600)", a.Invocations)
			}
		}
	}
	if !found {
		t.Error("esperado alerta para 'agent-governance'")
	}
}

func TestFormatTopSkills_ContainsSkillNames(t *testing.T) {
	skills := []SkillMetric{
		{Name: "go-implementation", Count: 50, Percentage: 50.0},
		{Name: "agent-governance", Count: 30, Percentage: 30.0},
		{Name: "bugfix", Count: 20, Percentage: 20.0},
	}
	result := FormatTopSkills(skills)
	for _, s := range skills {
		if !containsStr(result, s.Name) {
			t.Errorf("FormatTopSkills deveria conter %q", s.Name)
		}
	}
}

func TestFormatTopSkills_Empty_ReturnsNoData(t *testing.T) {
	result := FormatTopSkills(nil)
	if result == "" {
		t.Error("FormatTopSkills com nil nao deve retornar string vazia")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
