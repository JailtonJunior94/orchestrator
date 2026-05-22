package persistence

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	runtime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// metricsSectionHeader é o cabeçalho canônico da seção de métricas unificada (ADR-021).
// Mantém compatibilidade com o cabeçalho Claude-2026 quando só métricas Claude estão presentes.
const metricsSectionHeader = "## Métricas Claude-2026"

// sectionHeaderRe detecta o marcador exato da seção (início de linha).
var sectionHeaderRe = regexp.MustCompile(`(?m)^## Runtime ACP$`)

// nextSectionRe detecta o próximo cabeçalho de segundo nível após o marcador.
var nextSectionRe = regexp.MustCompile(`(?m)^## `)

// reportTemplate é o template da seção ## Runtime ACP (RF-10).
var reportTemplate = template.Must(template.New("runtime-acp").Parse(
	`## Runtime ACP

- runtime: acp
- launcher: {{.Launcher}}
- events_count: {{.EventsCount}}
- unknown_events_count: {{.UnknownEventsCount}}
- cancel_reason: {{.CancelReason}}
`))

// EnrichReport adiciona ou substitui a seção "## Runtime ACP" no execution_report.md (RF-10).
// Quando a Summary contém métricas (ADR-021), também faz append da seção de métricas unificada.
// A operação é idempotente: chamadas sucessivas produzem o mesmo arquivo.
func EnrichReport(reportPath string, summary runtime.Summary, fsys fs.FileSystem) error {
	clean := filepath.Clean(reportPath)

	existing, err := fsys.ReadFile(clean)
	if err != nil {
		// Arquivo não existe ainda; criar com apenas a seção.
		existing = []byte{}
	}

	section, err := renderSection(summary)
	if err != nil {
		return fmt.Errorf("persistence: renderizar seção Runtime ACP: %w", err)
	}

	updated := injectSection(string(existing), section)

	// ★ ADR-021: seção de métricas unificada — append opcional.
	// Omitida quando Metrics.IsZero() (evita poluição em sessões sem métricas).
	if metricsSection := RenderMetricsSection(summary.Metrics); metricsSection != "" {
		updated = injectMetricsSection(updated, metricsSection)
	}

	if err := fsys.WriteFile(clean, []byte(updated)); err != nil {
		return fmt.Errorf("persistence: escrever %s: %w", clean, err)
	}
	return nil
}

// RenderMetricsSection renderiza a seção de métricas unificada (ADR-021, RP-02).
// Retorna "" quando MetricSet.IsZero() == true (seção omitida, RP-02: nunca campos divergentes).
// Cabeçalho mantém compatibilidade com "## Métricas Claude-2026" quando só Claude está presente.
// Exportada para uso em testes e por internal/evidence.
func RenderMetricsSection(m events.MetricSet) string {
	fields := m.Fields()
	if len(fields) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(metricsSectionHeader)
	sb.WriteString("\n| Métrica | Valor |\n|---|---|\n")
	for _, f := range fields {
		fmt.Fprintf(&sb, "| %s | %d |\n", f.Name, f.Value)
	}
	return sb.String()
}

// RenderClaudeMetricsSection é mantida por compatibilidade retroativa com testes e leitores.
// Delega para RenderMetricsSection usando summary.Metrics (ADR-021).
// Deprecated: usar RenderMetricsSection(summary.Metrics) diretamente.
func RenderClaudeMetricsSection(summary runtime.Summary) string {
	return RenderMetricsSection(summary.Metrics)
}

// injectMetricsSection substitui ou faz append da seção de métricas.
// Idempotente: se a seção já existir, substitui; caso contrário faz append.
func injectMetricsSection(content, section string) string {
	headerRe := regexp.MustCompile(`(?m)^## Métricas Claude-2026`)
	loc := headerRe.FindStringIndex(content)
	if loc == nil {
		// Não existe: append.
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + "\n" + section
	}
	// Existe: substituir até fim do arquivo (seção deve ser a última).
	start := loc[0]
	return content[:start] + section
}

// renderSection renderiza o conteúdo da seção ## Runtime ACP.
func renderSection(summary runtime.Summary) (string, error) {
	var sb strings.Builder
	if err := reportTemplate.Execute(&sb, summary); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// injectSection substitui ou faz append da seção no conteúdo do relatório.
func injectSection(content, section string) string {
	loc := sectionHeaderRe.FindStringIndex(content)
	if loc == nil {
		// Seção não encontrada: fazer append.
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + "\n" + section
	}

	// Seção encontrada: substituir até o próximo ## ou fim do arquivo.
	start := loc[0]
	rest := content[loc[1]:]

	// Procurar próximo cabeçalho de segundo nível após o marcador.
	nextLoc := nextSectionRe.FindStringIndex(rest)
	if nextLoc == nil {
		// Não há seção seguinte; substituir até o fim.
		return content[:start] + section
	}

	// Há seção seguinte; substituir apenas o bloco da seção ACP.
	nextStart := loc[1] + nextLoc[0]
	return content[:start] + section + "\n" + content[nextStart:]
}
