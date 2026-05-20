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
	if err := fsys.WriteFile(clean, []byte(updated)); err != nil {
		return fmt.Errorf("persistence: escrever %s: %w", clean, err)
	}
	return nil
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
