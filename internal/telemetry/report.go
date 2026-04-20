package telemetry

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxTopN = 5

// ReportData contém as métricas acionáveis derivadas do log de telemetria.
type ReportData struct {
	Period            string        `json:"period"`
	TotalInvocations  int           `json:"total_invocations"`
	Skills            []SkillMetric `json:"skills"`
	Refs              []RefMetric   `json:"refs"`
	EstimatedTokens   int           `json:"estimated_tokens"`
	RefsPerInvocation float64       `json:"refs_per_invocation"`
	Alerts            []string      `json:"alerts"`
}

// SkillMetric representa uma skill com contagem e percentual de uso.
type SkillMetric struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// RefMetric representa uma referência com sua contagem de carregamentos.
type RefMetric struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// Report lê .agents/telemetry.log, aplica filtro de período e retorna métricas acionáveis.
// Linhas malformadas são ignoradas sem erro. Log ausente retorna ReportData zero-value com err=nil.
func Report(rootDir string, since time.Duration) (ReportData, error) {
	logPath := filepath.Join(rootDir, ".agents", "telemetry.log")
	entries, err := parseLogEntries(logPath, since)
	if err != nil {
		return ReportData{}, err
	}

	skillCounts := make(map[string]int)
	refCounts := make(map[string]int)
	// skillRefs rastreia se cada skill carregou ao menos uma ref
	skillRefs := make(map[string]bool)
	totalRefLoads := 0

	for _, e := range entries {
		if e.Skill == "" {
			continue
		}
		skillCounts[e.Skill]++
		if e.Ref != "" {
			refCounts[e.Ref]++
			totalRefLoads++
			skillRefs[e.Skill] = true
		} else if _, seen := skillRefs[e.Skill]; !seen {
			skillRefs[e.Skill] = false
		}
	}

	total := 0
	for _, c := range skillCounts {
		total += c
	}
	if total == 0 {
		return ReportData{}, nil
	}

	skills := topSkills(skillCounts, total)
	refs := topRefs(refCounts)
	alerts := buildAlerts(skillCounts, skillRefs)

	var refsPerInv float64
	if total > 0 {
		refsPerInv = float64(totalRefLoads) / float64(total)
	}

	period := "all"
	if since > 0 {
		period = fmt.Sprintf("últimas %s", since)
	}

	return ReportData{
		Period:            period,
		TotalInvocations:  total,
		Skills:            skills,
		Refs:              refs,
		EstimatedTokens:   totalRefLoads * tokensPerRefLoad,
		RefsPerInvocation: refsPerInv,
		Alerts:            alerts,
	}, nil
}

// FormatText formata ReportData como texto legível.
func FormatText(data ReportData) string {
	if data.TotalInvocations == 0 {
		return "Sem dados de telemetria no período especificado.\n"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Relatório de Telemetria\n")
	fmt.Fprintf(&sb, "Período: %s\n", data.Period)
	fmt.Fprintf(&sb, "Total de invocações: %d\n", data.TotalInvocations)

	fmt.Fprintf(&sb, "\nSkills Mais Usadas (top %d):\n", maxTopN)
	for i, s := range data.Skills {
		fmt.Fprintf(&sb, "  %d. %-30s %d  (%.1f%%)\n", i+1, s.Name, s.Count, s.Percentage)
	}

	fmt.Fprintf(&sb, "\nReferências Mais Carregadas (top %d):\n", maxTopN)
	if len(data.Refs) == 0 {
		fmt.Fprintf(&sb, "  (nenhuma referência carregada)\n")
	}
	for i, r := range data.Refs {
		fmt.Fprintf(&sb, "  %d. %-30s %d\n", i+1, r.Name, r.Count)
	}

	fmt.Fprintf(&sb, "\nMétricas:\n")
	fmt.Fprintf(&sb, "  Refs por invocação (média): %.1f\n", data.RefsPerInvocation)
	fmt.Fprintf(&sb, "  Tokens estimados:           %d (%d refs × %d tok/ref)\n",
		data.EstimatedTokens, data.EstimatedTokens/max1(tokensPerRefLoad), tokensPerRefLoad)

	if len(data.Alerts) > 0 {
		fmt.Fprintf(&sb, "\nAlertas:\n")
		for _, a := range data.Alerts {
			fmt.Fprintf(&sb, "  ⚠ %s\n", a)
		}
	}

	return sb.String()
}

// FormatJSON serializa ReportData como JSON.
func FormatJSON(data ReportData) ([]byte, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serializar relatório: %w", err)
	}
	return b, nil
}

func topSkills(counts map[string]int, total int) []SkillMetric {
	type kv struct {
		k string
		v int
	}
	pairs := make([]kv, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	if len(pairs) > maxTopN {
		pairs = pairs[:maxTopN]
	}
	out := make([]SkillMetric, len(pairs))
	for i, p := range pairs {
		pct := 0.0
		if total > 0 {
			pct = float64(p.v) / float64(total) * 100
		}
		out[i] = SkillMetric{Name: p.k, Count: p.v, Percentage: pct}
	}
	return out
}

func topRefs(counts map[string]int) []RefMetric {
	type kv struct {
		k string
		v int
	}
	pairs := make([]kv, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].v != pairs[j].v {
			return pairs[i].v > pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	if len(pairs) > maxTopN {
		pairs = pairs[:maxTopN]
	}
	out := make([]RefMetric, len(pairs))
	for i, p := range pairs {
		out[i] = RefMetric{Name: p.k, Count: p.v}
	}
	return out
}

func buildAlerts(skillCounts map[string]int, skillRefs map[string]bool) []string {
	var alerts []string
	names := make([]string, 0, len(skillCounts))
	for k := range skillCounts {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		if hasRef, ok := skillRefs[name]; ok && !hasRef {
			alerts = append(alerts, fmt.Sprintf(
				"skill '%s' invocada %d vez(es) sem carregar nenhuma referência — possível bypass de governança",
				name, skillCounts[name],
			))
		}
	}
	return alerts
}

func max1(v int) int {
	if v == 0 {
		return 1
	}
	return v
}
