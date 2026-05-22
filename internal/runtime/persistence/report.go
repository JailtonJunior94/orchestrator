package persistence

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	runtime "github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
)

// claudeMetricsSectionHeader é o cabeçalho canônico da seção de métricas Claude-2026.
// Detectado como regex para suportar presença OU ausência no validador de evidence (F4-Claude).
const claudeMetricsSectionHeader = "## Métricas Claude-2026"

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
// Quando a Summary contém métricas Claude-2026 (F4-Claude), também faz append da seção
// "## Métricas Claude-2026" ao final do report.
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

	// ★ F4-Claude: seção "Métricas Claude-2026" — append opcional.
	// Omitida quando todos os contadores são zero (evita poluição em relatórios legados).
	if metricsSection := RenderClaudeMetricsSection(summary); metricsSection != "" {
		updated = injectClaudeMetricsSection(updated, metricsSection)
	}

	if err := fsys.WriteFile(clean, []byte(updated)); err != nil {
		return fmt.Errorf("persistence: escrever %s: %w", clean, err)
	}
	return nil
}

// RenderClaudeMetricsSection renderiza a seção "## Métricas Claude-2026" para o report.
// Retorna "" quando todos os contadores são zero (seção omitida, compatibilidade retroativa).
// Exportada para uso em testes e por internal/evidence.
func RenderClaudeMetricsSection(summary runtime.Summary) string {
	if summary.CacheReadTokens == 0 &&
		summary.CacheCreationTokens == 0 &&
		summary.ThinkingTokens == 0 &&
		summary.ToolCallsNormalizedCount == 0 {
		return ""
	}
	return fmt.Sprintf(`%s
| Métrica | Valor |
|---|---|
| cache_read_tokens | %d |
| cache_creation_tokens | %d |
| thinking_tokens | %d |
| tool_calls_normalized | %d |
`, claudeMetricsSectionHeader,
		summary.CacheReadTokens,
		summary.CacheCreationTokens,
		summary.ThinkingTokens,
		summary.ToolCallsNormalizedCount,
	)
}

// injectClaudeMetricsSection substitui ou faz append da seção Métricas Claude-2026.
// Idempotente: se a seção já existir, substitui; caso contrário faz append.
func injectClaudeMetricsSection(content, section string) string {
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
