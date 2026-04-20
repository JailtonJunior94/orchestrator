package telemetry

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const weeksForTrend = 4

// WeeklyBucket agrupa invocacoes de uma semana.
type WeeklyBucket struct {
	Week        string `json:"week"`        // "2026-W15"
	Invocations int    `json:"invocations"` // total de invocacoes na semana
}

// TrendData contem a evolucao semanal de invocacoes.
type TrendData struct {
	Weeks []WeeklyBucket `json:"weeks"`
}

// Trend calcula a evolucao de invocacoes por semana nas ultimas weeksForTrend semanas.
func Trend(rootDir string) (TrendData, error) {
	logPath := filepath.Join(rootDir, ".agents", "telemetry.log")
	entries, err := parseLogEntries(logPath, time.Duration(weeksForTrend)*7*24*time.Hour)
	if err != nil {
		return TrendData{}, err
	}

	// Agrupar por semana ISO (yyyy-Www)
	buckets := make(map[string]int)
	for _, e := range entries {
		year, week := e.Timestamp.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		buckets[key]++
	}

	// Gerar as ultimas weeksForTrend semanas (mesmo sem dados)
	now := time.Now().UTC()
	weeks := make([]WeeklyBucket, weeksForTrend)
	for i := weeksForTrend - 1; i >= 0; i-- {
		t := now.AddDate(0, 0, -7*i)
		year, week := t.ISOWeek()
		key := fmt.Sprintf("%d-W%02d", year, week)
		weeks[weeksForTrend-1-i] = WeeklyBucket{
			Week:        key,
			Invocations: buckets[key],
		}
	}

	return TrendData{Weeks: weeks}, nil
}

// FormatTrend formata TrendData como tabela ASCII.
func FormatTrend(data TrendData) string {
	if len(data.Weeks) == 0 {
		return "Sem dados de tendencia.\n"
	}

	maxVal := 0
	for _, w := range data.Weeks {
		if w.Invocations > maxVal {
			maxVal = w.Invocations
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Tendencia de Invocacoes (ultimas %d semanas):\n", weeksForTrend)
	fmt.Fprintf(&sb, "%-12s  %-5s  %s\n", "Semana", "Count", "Barra")
	fmt.Fprintf(&sb, "%s\n", strings.Repeat("-", 50))

	barWidth := 30
	for _, w := range data.Weeks {
		bar := 0
		if maxVal > 0 {
			bar = w.Invocations * barWidth / maxVal
		}
		fmt.Fprintf(&sb, "%-12s  %-5d  %s\n", w.Week, w.Invocations, strings.Repeat("█", bar))
	}
	return sb.String()
}

// BudgetAlert representa um alerta de budget excedido para uma skill.
type BudgetAlert struct {
	Skill       string `json:"skill"`
	Invocations int    `json:"invocations"`
	Message     string `json:"message"`
}

// BudgetCheckData contem o resultado da verificacao de budget.
type BudgetCheckData struct {
	Alerts []BudgetAlert `json:"alerts"`
	OK     bool          `json:"ok"`
}

// defaultSkillBudgetInvocations define o limite de invocacoes esperadas por skill.
// Superar este limite pode indicar carregamento excessivo ou loop de governanca.
var defaultSkillBudgetInvocations = map[string]int{
	"agent-governance":        500,
	"go-implementation":       300,
	"object-calisthenics-go":  100,
	"bugfix":                  200,
	"review":                  200,
	"refactor":                100,
	"execute-task":            200,
	"create-prd":              50,
	"create-tasks":            50,
	"create-technical-specification": 50,
}

// BudgetCheck verifica se alguma skill excedeu o budget de invocacoes esperado.
func BudgetCheck(rootDir string, since time.Duration) (BudgetCheckData, error) {
	logPath := filepath.Join(rootDir, ".agents", "telemetry.log")
	entries, err := parseLogEntries(logPath, since)
	if err != nil {
		return BudgetCheckData{}, err
	}

	counts := make(map[string]int)
	for _, e := range entries {
		if e.Skill != "" {
			counts[e.Skill]++
		}
	}

	var alerts []BudgetAlert
	names := make([]string, 0, len(counts))
	for k := range counts {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, skill := range names {
		budget, ok := defaultSkillBudgetInvocations[skill]
		if !ok {
			continue
		}
		count := counts[skill]
		if count > budget {
			pct := float64(count-budget) / float64(budget) * 100
			alerts = append(alerts, BudgetAlert{
				Skill:       skill,
				Invocations: count,
				Message: fmt.Sprintf("skill '%s': %d invocacoes > budget de %d (+%.0f%%)",
					skill, count, budget, pct),
			})
		}
	}

	return BudgetCheckData{Alerts: alerts, OK: len(alerts) == 0}, nil
}

// FormatBudgetCheck formata BudgetCheckData como texto legivel.
func FormatBudgetCheck(data BudgetCheckData) string {
	if data.OK || len(data.Alerts) == 0 {
		return "Budget check: todas as skills dentro do limite esperado.\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Budget check: %d skill(s) acima do limite\n", len(data.Alerts))
	for _, a := range data.Alerts {
		fmt.Fprintf(&sb, "  AVISO: %s\n", a.Message)
	}
	return sb.String()
}

// FormatTopSkills formata as top skills como tabela ASCII ordenada por frequencia.
func FormatTopSkills(skills []SkillMetric) string {
	if len(skills) == 0 {
		return "Sem dados de uso de skills.\n"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Top Skills por Uso:\n")
	fmt.Fprintf(&sb, "%-4s  %-30s  %-8s  %s\n", "Pos", "Skill", "Count", "% do Total")
	fmt.Fprintf(&sb, "%s\n", strings.Repeat("-", 55))
	for i, s := range skills {
		fmt.Fprintf(&sb, "%-4d  %-30s  %-8d  %.1f%%\n", i+1, s.Name, s.Count, s.Percentage)
	}
	return sb.String()
}
