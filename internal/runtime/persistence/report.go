package persistence

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	runtime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

const _metricsSectionHeader = "## Métricas Claude-2026"

var _sectionHeaderRe = regexp.MustCompile(`(?m)^## Runtime ACP$`)

var _nextSectionRe = regexp.MustCompile(`(?m)^## `)

const _memoryEvidenceSectionHeader = "## Evidência de Memória Durável"

var _memoryEvidenceHeaderRe = regexp.MustCompile(`(?m)^## Evidência de Memória Durável$`)

var _metricsSectionHeaderRe = regexp.MustCompile(`(?m)^## Métricas Claude-2026`)

var _reportTemplate = template.Must(template.New("runtime-acp").Parse(
	`## Runtime ACP

- runtime: acp
- launcher: {{.Launcher}}
- events_count: {{.EventsCount}}
- unknown_events_count: {{.UnknownEventsCount}}
- cancel_reason: {{.CancelReason}}
`))

func (c *Catalog) EnrichReport(reportPath string, summary runtime.Summary, fsys fs.FileSystem) error {
	clean := filepath.Clean(reportPath)

	existing, err := fsys.ReadFile(clean)
	if err != nil {
		existing = []byte{}
	}

	section, err := NewCatalog().renderSection(summary)
	if err != nil {
		return fmt.Errorf("persistence: renderizar seção Runtime ACP: %w", err)
	}

	updated := NewCatalog().injectSection(string(existing), section)

	if memorySection := NewCatalog().RenderMemoryEvidenceSection(summary); memorySection != "" {
		updated = NewCatalog().injectBoundedSectionBefore(updated, _memoryEvidenceHeaderRe, _metricsSectionHeaderRe, memorySection)
	}

	if metricsSection := NewCatalog().RenderMetricsSection(summary.Metrics); metricsSection != "" {
		updated = NewCatalog().injectMetricsSection(updated, metricsSection)
	}

	if err := fsys.WriteFile(clean, []byte(updated)); err != nil {
		return fmt.Errorf("persistence: escrever %s: %w", clean, err)
	}
	return nil
}

func (c *Catalog) RenderMetricsSection(m events.MetricSet) string {
	fields := m.Fields()
	if len(fields) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(_metricsSectionHeader)
	sb.WriteString("\n| Métrica | Valor |\n|---|---|\n")
	for _, f := range fields {
		fmt.Fprintf(&sb, "| %s | %d |\n", f.Name, f.Value)
	}
	return sb.String()
}

func (c *Catalog) RenderClaudeMetricsSection(summary runtime.Summary) string {
	return NewCatalog().RenderMetricsSection(summary.Metrics)
}

func (c *Catalog) RenderMemoryEvidenceSection(summary runtime.Summary) string {
	e := summary.MemoryEvidence
	if e == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(_memoryEvidenceSectionHeader)
	sb.WriteString("\n\n")
	fmt.Fprintf(&sb, "- session: %s\n", e.SessionID)
	fmt.Fprintf(&sb, "- cli: %s\n", e.CLI)
	fmt.Fprintf(&sb, "- task: %s\n", e.TaskFileName)
	fmt.Fprintf(&sb, "- read_facts_by_layer: %s\n", c.formatLayerCounts(e.FactsByLayer))
	fmt.Fprintf(&sb, "- read_facts_omitted: %d\n", e.FactsOmitted)
	fmt.Fprintf(&sb, "- read_facts_contradicted: %d\n", e.FactsContradicted)
	fmt.Fprintf(&sb, "- read_pages_unreadable: %d\n", e.PagesUnreadable)
	fmt.Fprintf(&sb, "- write_facts_by_layer: %s\n", c.formatLayerCounts(e.WritesByLayer))
	fmt.Fprintf(&sb, "- budget_consumed_by_layer: %s\n", c.formatLayerCounts(e.BudgetByLayer))
	fmt.Fprintf(&sb, "- compactions_executed: %d\n", e.Compactions)
	fmt.Fprintf(&sb, "- facts_archived_by_layer: %s\n", c.formatLayerCounts(e.ArchivedByLayer))
	fmt.Fprintf(&sb, "- redactions_applied: %d\n", e.Redactions)
	fmt.Fprintf(&sb, "- contradictions_detected: %d\n", e.Contradictions)
	fmt.Fprintf(&sb, "- baton_claimed: %v\n", e.BatonClaimed)
	return sb.String()
}

func (c *Catalog) formatLayerCounts(byLayer map[string]int) string {
	if len(byLayer) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(byLayer))
	for k := range byLayer {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, byLayer[k]))
	}
	return strings.Join(parts, ", ")
}

func (c *Catalog) injectMetricsSection(content, section string) string {
	return c.injectBoundedSection(content, _metricsSectionHeaderRe, section)
}

func (c *Catalog) renderSection(summary runtime.Summary) (string, error) {
	var sb strings.Builder
	if err := _reportTemplate.Execute(&sb, summary); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (c *Catalog) injectSection(content, section string) string {
	return c.injectBoundedSection(content, _sectionHeaderRe, section)
}

func (c *Catalog) injectBoundedSection(content string, headerRe *regexp.Regexp, section string) string {
	return c.injectBoundedSectionBefore(content, headerRe, nil, section)
}

func (c *Catalog) injectBoundedSectionBefore(content string, headerRe, beforeRe *regexp.Regexp, section string) string {
	loc := headerRe.FindStringIndex(content)
	if loc == nil {
		return c.appendSection(content, beforeRe, section)
	}

	start := loc[0]
	rest := content[loc[1]:]

	nextLoc := _nextSectionRe.FindStringIndex(rest)
	if nextLoc == nil {
		return content[:start] + section
	}

	nextStart := loc[1] + nextLoc[0]
	return content[:start] + section + "\n" + content[nextStart:]
}

func (c *Catalog) appendSection(content string, beforeRe *regexp.Regexp, section string) string {
	if beforeRe != nil {
		if loc := beforeRe.FindStringIndex(content); loc != nil {
			prefix := content[:loc[0]]
			if prefix != "" && !strings.HasSuffix(prefix, "\n") {
				prefix += "\n"
			}
			return prefix + section + "\n" + content[loc[0]:]
		}
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + section
}
