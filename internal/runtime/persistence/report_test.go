package persistence_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/events"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/persistence"
)

func makeSummary() runtime.Summary {
	return runtime.Summary{
		Launcher:           "binary",
		EventsCount:        42,
		UnknownEventsCount: 1,
		CancelReason:       events.CancelReasonNone,
	}
}

func TestEnrichReport_FreshAppend(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n\n## Resultado\n\n- status: done\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := persistence.EnrichReport("/report/execution_report.md", makeSummary(), fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if !strings.Contains(content, "## Runtime ACP") {
		t.Error("seção '## Runtime ACP' não encontrada no relatório")
	}
	if !strings.Contains(content, "launcher: binary") {
		t.Error("campo 'launcher' não encontrado")
	}
	if !strings.Contains(content, "events_count: 42") {
		t.Error("campo 'events_count' não encontrado")
	}
	if !strings.Contains(content, "unknown_events_count: 1") {
		t.Error("campo 'unknown_events_count' não encontrado")
	}
	if !strings.Contains(content, "cancel_reason: none") {
		t.Error("campo 'cancel_reason' não encontrado")
	}
	// Conteúdo original preservado.
	if !strings.Contains(content, "## Resultado") {
		t.Error("seção original '## Resultado' foi removida")
	}
}

func TestEnrichReport_EmptyFile(t *testing.T) {
	fsys := fs.NewFakeFileSystem()

	if err := persistence.EnrichReport("/report/execution_report.md", makeSummary(), fsys); err != nil {
		t.Fatalf("EnrichReport em arquivo inexistente: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	if !strings.Contains(string(data), "## Runtime ACP") {
		t.Error("seção '## Runtime ACP' não gerada em arquivo vazio")
	}
}

func TestEnrichReport_ReplaceExisting(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	initial := "# Relatório\n\n## Runtime ACP\n\n- runtime: acp\n- launcher: npx\n- events_count: 5\n- unknown_events_count: 0\n- cancel_reason: none\n\n## Outra Seção\n\n- conteúdo\n"
	if err := fsys.WriteFile("/report/execution_report.md", []byte(initial)); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := runtime.Summary{
		Launcher:           "binary",
		EventsCount:        99,
		UnknownEventsCount: 3,
		CancelReason:       events.CancelReasonActivityTimeout,
	}

	if err := persistence.EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	// Verifica que o novo conteúdo foi aplicado.
	if !strings.Contains(content, "launcher: binary") {
		t.Error("campo 'launcher' não atualizado para binary")
	}
	if !strings.Contains(content, "events_count: 99") {
		t.Error("campo 'events_count' não atualizado")
	}
	if !strings.Contains(content, "cancel_reason: activity_timeout") {
		t.Error("campo 'cancel_reason' não atualizado")
	}
	// Verifica que lançador antigo foi removido.
	if strings.Contains(content, "launcher: npx") {
		t.Error("launcher antigo 'npx' ainda presente após substituição")
	}
	// Seção seguinte preservada.
	if !strings.Contains(content, "## Outra Seção") {
		t.Error("seção '## Outra Seção' foi removida indevidamente")
	}
}

func TestEnrichReport_Idempotency(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	if err := fsys.WriteFile("/report/execution_report.md", []byte("# Relatório\n")); err != nil {
		t.Fatalf("setup: %v", err)
	}
	summary := makeSummary()

	if err := persistence.EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("primeira chamada: %v", err)
	}
	data1, _ := fsys.ReadFile("/report/execution_report.md")

	if err := persistence.EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("segunda chamada: %v", err)
	}
	data2, _ := fsys.ReadFile("/report/execution_report.md")

	if string(data1) != string(data2) {
		t.Errorf("idempotência falhou:\napós 1ª chamada:\n%s\napós 2ª chamada:\n%s",
			string(data1), string(data2))
	}
}

func TestEnrichReport_WriteError(t *testing.T) {
	efs := &errFS{
		FakeFileSystem: fs.NewFakeFileSystem(),
		writeErr:       errors.New("disco cheio"),
	}
	err := persistence.EnrichReport("/report/execution_report.md", makeSummary(), efs)
	if err == nil {
		t.Fatal("esperava erro de WriteFile")
	}
}

func TestEnrichReport_SectionAtEndOfFile(t *testing.T) {
	// Seção ACP é a última do arquivo (sem seção seguinte).
	fsys := fs.NewFakeFileSystem()
	initial := "# Relatório\n\n## Runtime ACP\n\n- runtime: acp\n- launcher: npx\n- events_count: 5\n- unknown_events_count: 0\n- cancel_reason: none\n"
	if err := fsys.WriteFile("/report/execution_report.md", []byte(initial)); err != nil {
		t.Fatalf("setup: %v", err)
	}

	summary := runtime.Summary{
		Launcher:           "binary",
		EventsCount:        10,
		UnknownEventsCount: 0,
		CancelReason:       events.CancelReasonNone,
	}

	if err := persistence.EnrichReport("/report/execution_report.md", summary, fsys); err != nil {
		t.Fatalf("EnrichReport: %v", err)
	}

	data, _ := fsys.ReadFile("/report/execution_report.md")
	content := string(data)

	if !strings.Contains(content, "launcher: binary") {
		t.Error("launcher não atualizado")
	}
	if strings.Contains(content, "launcher: npx") {
		t.Error("launcher antigo npx ainda presente")
	}
	// Seção não deve aparecer duas vezes.
	count := strings.Count(content, "## Runtime ACP")
	if count != 1 {
		t.Errorf("## Runtime ACP aparece %d vezes, esperava 1", count)
	}
}
