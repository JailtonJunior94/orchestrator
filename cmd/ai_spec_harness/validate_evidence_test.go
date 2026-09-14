package aispecharness

import (
	"os"
	"path/filepath"
	"testing"
)

func runValidateEvidence(t *testing.T, kind, path string) int {
	t.Helper()
	cmd := newValidateEvidenceCmd()
	err := cmd.RunE(cmd, []string{kind, path})
	if err == nil {
		return 0
	}
	return NewExitResolver().CodeFor(err)
}

func writeEvidenceFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "report.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestValidateEvidenceCmd_AcceptsReviewKind(t *testing.T) {
	path := writeEvidenceFixture(t, `# Review
Veredito: APPROVED

## Achados
Sem achados

## Arquivos Revisados
- internal/foo.go

## Riscos Residuais
- nenhum

## Validações Executadas
- go build ./... -> exit 0

## Mapa de Critérios de Aceite
- [atendido] criterio comprovado -> go test ./... -> exit 0
`)
	if code := runValidateEvidence(t, "review", path); code != 0 {
		t.Fatalf("review valido deve aprovar, got exit %d", code)
	}
}

func TestValidateEvidenceCmd_ReviewRejectsTrivialEvidence(t *testing.T) {
	path := writeEvidenceFixture(t, `# Review
Veredito: APPROVED

## Achados
Sem achados

## Arquivos Revisados
- internal/foo.go

## Riscos Residuais
- nenhum

## Validações Executadas
- go build ./... -> exit 0

## Mapa de Critérios de Aceite
- [atendido] criterio trivial -> make it work -> done
`)
	if code := runValidateEvidence(t, "review", path); code != 1 {
		t.Fatalf("registro trivial deve reprovar, got exit %d", code)
	}
}

func TestValidateEvidenceCmd_RejectsUnknownKind(t *testing.T) {
	path := writeEvidenceFixture(t, "# Report\n")
	if code := runValidateEvidence(t, "inexistente", path); code != 2 {
		t.Fatalf("tipo invalido deve sair 2, got %d", code)
	}
}

func TestValidateEvidenceCmd_TaskWithoutEvidenceFails(t *testing.T) {
	path := writeEvidenceFixture(t, "Estado: done\n")
	if code := runValidateEvidence(t, "task", path); code != 1 {
		t.Fatalf("task sem evidencia deve reprovar, got exit %d", code)
	}
}
