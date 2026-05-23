package persistence

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
)

// WriteToolCalls gera o arquivo tool_calls.md (RF-09).
// Sem summaries: escreve "Nenhum tool call registrado.\n".
// Com summaries: lista numerada com nome, status e tool_call_id.
func WriteToolCalls(path string, summaries []events.ToolCallSummary, fsys fs.FileSystem) error {
	clean := filepath.Clean(path)
	dir := filepath.Dir(clean)
	if err := fsys.MkdirAll(dir); err != nil {
		return fmt.Errorf("persistence: criar diretório %s: %w", dir, err)
	}

	content := buildToolCallsContent(summaries)
	if err := fsys.WriteFile(clean, []byte(content)); err != nil {
		return fmt.Errorf("persistence: escrever %s: %w", clean, err)
	}
	return nil
}

// buildToolCallsContent constrói o conteúdo do tool_calls.md.
func buildToolCallsContent(summaries []events.ToolCallSummary) string {
	if len(summaries) == 0 {
		return "Nenhum tool call registrado.\n"
	}

	var sb strings.Builder
	sb.WriteString("# Tool Calls\n\n")
	for i, s := range summaries {
		status := "pending"
		if s.Final {
			status = "done"
		}
		// Formato: - {n}. **{name}** — status: {status} — id: `{tool_call_id}`
		fmt.Fprintf(&sb, "- %d. **%s** — status: %s — id: `%s`\n",
			i+1,
			s.Name,
			status,
			s.ID.String(),
		)
	}
	return sb.String()
}
