package metrics

import (
	"io"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// silentPrinter retorna um Printer que descarta toda saida (nao polui stdout nos testes).
func silentPrinter() *output.Printer {
	return &output.Printer{Out: io.Discard, Err: io.Discard}
}

func TestGather_HappyPath(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	// skill-a com SKILL.md e uma referencia
	ffs.Files[root+"/.agents/skills/skill-a/SKILL.md"] = []byte("# Skill A\nconteudo da skill A com algumas palavras")
	ffs.Files[root+"/.agents/skills/skill-a/references/ref1.md"] = []byte("# Referencia\nconteudo da referencia")

	// skill-b com apenas SKILL.md, sem referencias
	ffs.Files[root+"/.agents/skills/skill-b/SKILL.md"] = []byte("# Skill B\nconteudo da skill B")

	svc := NewService(ffs, silentPrinter())
	report, err := svc.gather(root)

	if err != nil {
		t.Fatalf("gather nao deve retornar erro: %v", err)
	}
	if report.SkillCount != 2 {
		t.Errorf("SkillCount: got %d, want 2", report.SkillCount)
	}
	if report.RefCount != 1 {
		t.Errorf("RefCount: got %d, want 1", report.RefCount)
	}
	if _, ok := report.Baselines["skill-a"]; !ok {
		t.Error("Baselines deve conter skill-a")
	}
	if _, ok := report.Baselines["skill-b"]; !ok {
		t.Error("Baselines deve conter skill-b")
	}
	if report.Baselines["skill-a"].TokensEst == 0 {
		t.Error("skill-a deve ter TokensEst > 0")
	}
}

func TestGather_MissingSkillsMd_ReturnsError(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	// Diretorio da skill existe mas sem SKILL.md
	ffs.Dirs[root+"/.agents/skills"] = true
	ffs.Dirs[root+"/.agents/skills/skill-sem-skillmd"] = true

	svc := NewService(ffs, silentPrinter())
	_, err := svc.gather(root)

	if err == nil {
		t.Fatal("gather deve retornar erro quando SKILL.md esta ausente")
	}
	if !strings.Contains(err.Error(), "SKILL.md ausente") {
		t.Errorf("mensagem de erro deve mencionar 'SKILL.md ausente', got: %v", err)
	}
	if !strings.Contains(err.Error(), "skill-sem-skillmd") {
		t.Errorf("mensagem de erro deve mencionar o nome da skill, got: %v", err)
	}
}

func TestGather_MissingSkillsDir_ReturnsError(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	// FakeFileSystem vazio: nenhum arquivo, nenhum diretorio

	svc := NewService(ffs, silentPrinter())
	_, err := svc.gather(root)

	if err == nil {
		t.Fatal("gather deve retornar erro quando diretorio de skills nao existe")
	}
	if !strings.Contains(err.Error(), "diretorio de skills nao encontrado") {
		t.Errorf("mensagem de erro deve mencionar 'diretorio de skills nao encontrado', got: %v", err)
	}
}

func TestExecute_PropagatesGatherError(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	// Sem diretorio de skills

	svc := NewService(ffs, silentPrinter())
	err := svc.Execute(root, "table")

	if err == nil {
		t.Fatal("Execute deve propagar erro de gather")
	}
}

func TestGather_SkillCountNeverFalsePositive(t *testing.T) {
	// Garante que repositorio parcial nao retorna zero enganoso para SkillCount
	// quando ha skills reais presentes
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	ffs.Files[root+"/.agents/skills/minha-skill/SKILL.md"] = []byte("conteudo")

	svc := NewService(ffs, silentPrinter())
	report, err := svc.gather(root)

	if err != nil {
		t.Fatalf("gather nao deve retornar erro: %v", err)
	}
	if report.SkillCount == 0 {
		t.Error("SkillCount nao deve ser zero quando existe skill real no checkout")
	}
}
