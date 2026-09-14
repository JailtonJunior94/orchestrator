package contextgen

import (
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

const authoredClaudeMD = `# Regras do time de pagamentos

Nunca alterar o esquema de ledger sem aprovacao do time de risco.

## Runbook de plantao

1. Conferir a fila de liquidacao antes de qualquer deploy.
2. Abrir incidente se o lag passar de 5 minutos.
`

type recordingFakeFS struct {
	*fs.FakeFileSystem
	merged []string
}

func (r *recordingFakeFS) MarkMerged(path string) {
	r.merged = append(r.merged, path)
}

func newRecordingFS() *recordingFakeFS {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	return &recordingFakeFS{FakeFileSystem: ffs}
}

func generateClaude(t *testing.T, rfs *recordingFakeFS) string {
	t.Helper()
	g := NewGenerator(rfs, output.New(false))
	if err := g.Generate("/source", "/project", []skills.Tool{skills.ToolClaude}, nil, "full", false); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	data, ok := rfs.Files["/project/CLAUDE.md"]
	if !ok {
		t.Fatal("CLAUDE.md nao foi gerado")
	}
	return string(data)
}

func TestGeneratePreservesAuthoredClaudeMD(t *testing.T) {
	rfs := newRecordingFS()
	if err := rfs.WriteFile("/project/CLAUDE.md", []byte(authoredClaudeMD)); err != nil {
		t.Fatalf("semear CLAUDE.md autoral: %v", err)
	}

	content := generateClaude(t, rfs)

	for _, line := range []string{
		"# Regras do time de pagamentos",
		"Nunca alterar o esquema de ledger sem aprovacao do time de risco.",
		"## Runbook de plantao",
		"Abrir incidente se o lag passar de 5 minutos.",
	} {
		if !strings.Contains(content, line) {
			t.Errorf("conteudo autoral perdido: %q ausente do CLAUDE.md resultante", line)
		}
	}

	if !strings.Contains(content, "AGENTS.md") {
		t.Error("CLAUDE.md deve continuar carregando a governanca gerada pelo harness")
	}
	if !strings.Contains(content, userContentBegin) || !strings.Contains(content, userContentEnd) {
		t.Error("o bloco autoral preservado deve ficar delimitado pelos marcadores do harness")
	}
}

func TestGenerateMarksPreexistingClaudeMDAsMerged(t *testing.T) {
	rfs := newRecordingFS()
	if err := rfs.WriteFile("/project/CLAUDE.md", []byte(authoredClaudeMD)); err != nil {
		t.Fatalf("semear CLAUDE.md autoral: %v", err)
	}

	generateClaude(t, rfs)

	if len(rfs.merged) != 1 || rfs.merged[0] != "/project/CLAUDE.md" {
		t.Errorf("CLAUDE.md preexistente deveria ser classificado como merged; marcados: %v", rfs.merged)
	}
}

func TestGenerateDoesNotMarkFreshClaudeMDAsMerged(t *testing.T) {
	rfs := newRecordingFS()

	generateClaude(t, rfs)

	if len(rfs.merged) != 0 {
		t.Errorf("CLAUDE.md criado do zero nao e merge; marcados: %v", rfs.merged)
	}
}

func TestGenerateClaudeMDMergeIsIdempotent(t *testing.T) {
	rfs := newRecordingFS()
	if err := rfs.WriteFile("/project/CLAUDE.md", []byte(authoredClaudeMD)); err != nil {
		t.Fatalf("semear CLAUDE.md autoral: %v", err)
	}

	first := generateClaude(t, rfs)
	second := generateClaude(t, rfs)

	if first != second {
		t.Errorf("merge de CLAUDE.md nao e idempotente\nprimeira:\n%s\nsegunda:\n%s", first, second)
	}
	if got := strings.Count(second, userContentBegin); got != 1 {
		t.Errorf("bloco preservado duplicado: %d marcadores de abertura", got)
	}
}

func TestGenerateClaudeMDWithoutAuthoredContentStaysClean(t *testing.T) {
	rfs := newRecordingFS()

	first := generateClaude(t, rfs)
	second := generateClaude(t, rfs)

	if first != second {
		t.Error("regeracao de CLAUDE.md puro deveria ser byte-identica")
	}
	if strings.Contains(second, userContentBegin) {
		t.Error("CLAUDE.md sem conteudo autoral nao deve ganhar bloco preservado")
	}
}
