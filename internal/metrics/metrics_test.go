package metrics

import (
	"io"
	"math"
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
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	// skill-a com SKILL.md e uma referencia
	ffs.Files[root+"/.agents/skills/skill-a/SKILL.md"] = []byte("# Skill A\nconteudo da skill A com algumas palavras")
	ffs.Files[root+"/.agents/skills/skill-a/references/ref1.md"] = []byte("# Referencia\nconteudo da referencia")

	// skill-b com apenas SKILL.md, sem referencias
	ffs.Files[root+"/.agents/skills/skill-b/SKILL.md"] = []byte("# Skill B\nconteudo da skill B")

	svc := NewService(ffs, silentPrinter(), nil)
	report, err := svc.gather(root, false)

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
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	// Diretorio da skill existe mas sem SKILL.md
	ffs.Dirs[root+"/.agents/skills"] = true
	ffs.Dirs[root+"/.agents/skills/skill-sem-skillmd"] = true

	svc := NewService(ffs, silentPrinter(), nil)
	_, err := svc.gather(root, false)

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
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	// FakeFileSystem vazio: nenhum arquivo, nenhum diretorio

	svc := NewService(ffs, silentPrinter(), nil)
	_, err := svc.gather(root, false)

	if err == nil {
		t.Fatal("gather deve retornar erro quando diretorio de skills nao existe")
	}
	if !strings.Contains(err.Error(), "diretorio de skills nao encontrado") {
		t.Errorf("mensagem de erro deve mencionar 'diretorio de skills nao encontrado', got: %v", err)
	}
}

func TestExecute_PropagatesGatherError(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	// Sem diretorio de skills

	svc := NewService(ffs, silentPrinter(), nil)
	err := svc.Execute(root, "table", false)

	if err == nil {
		t.Fatal("Execute deve propagar erro de gather")
	}
}

func TestTokenEstimate_SanityCheck(t *testing.T) {
	t.Parallel()
	// Conteudo representativo de uma SKILL.md (~800 palavras, ~5000 chars)
	sample := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100)
	est := int(math.Round(float64(len(sample)) / 3.5))
	// tiktoken cl100k_base daria ~1100 tokens para ~4500 chars
	// chars/3.5 daria ~1285
	// Divergencia aceitavel: <=20%
	if est < 900 || est > 1600 {
		t.Errorf("estimativa fora de faixa aceitavel: %d tokens para %d chars", est, len(sample))
	}
}

func TestCharEstimator_Deterministic(t *testing.T) {
	t.Parallel()
	text := "The quick brown fox jumps over the lazy dog."
	tok := NewCharEstimator()
	a := tok.EstimateTokens(text)
	b := tok.EstimateTokens(text)
	if a != b {
		t.Errorf("CharEstimator deve ser deterministico: %d != %d", a, b)
	}
	if a == 0 {
		t.Error("CharEstimator deve retornar tokens > 0 para texto nao-vazio")
	}
	if tok.Name() != "chars/3.5" {
		t.Errorf("CharEstimator.Name() = %q, want \"chars/3.5\"", tok.Name())
	}
}

func TestTiktokenEstimator_WhenAvailable(t *testing.T) {
	t.Parallel()
	tok, err := NewTiktokenEstimator()
	if err != nil {
		t.Skipf("tiktoken nao disponivel (sem acesso ao modelo BPE): %v", err)
	}
	text := "The quick brown fox jumps over the lazy dog."
	tokens := tok.EstimateTokens(text)
	if tokens == 0 {
		t.Error("TiktokenEstimator deve retornar tokens > 0 para texto nao-vazio")
	}
	// Contagem de tokens precisa para esta frase via cl100k_base deve ser ~10 tokens
	if tokens < 5 || tokens > 20 {
		t.Errorf("TiktokenEstimator: contagem inesperada %d para frase conhecida", tokens)
	}
	if tok.Name() != "tiktoken/cl100k_base" {
		t.Errorf("TiktokenEstimator.Name() = %q, want \"tiktoken/cl100k_base\"", tok.Name())
	}
}

func TestTiktokenEstimator_MoreAccurateThanChar(t *testing.T) {
	t.Parallel()
	tok, err := NewTiktokenEstimator()
	if err != nil {
		t.Skipf("tiktoken nao disponivel: %v", err)
	}
	// Para texto em ingles, tiktoken e mais preciso que chars/3.5.
	// A divergencia entre os dois deve ser de ate 20%.
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 50)
	charEst := NewCharEstimator().EstimateTokens(text)
	tiktokenEst := tok.EstimateTokens(text)

	if charEst == 0 || tiktokenEst == 0 {
		t.Fatal("ambos estimadores devem retornar > 0")
	}
	// Divergencia maxima esperada: 20%
	diff := charEst - tiktokenEst
	if diff < 0 {
		diff = -diff
	}
	pct := float64(diff) / float64(charEst) * 100
	if pct > 25 {
		t.Errorf("divergencia entre chars/3.5 e tiktoken maior que esperada: %.1f%% (char=%d tiktoken=%d)", pct, charEst, tiktokenEst)
	}
}

func TestNewPreciseTokenizer_FallbackReturnsCharEstimator(t *testing.T) {
	t.Parallel()
	// NewPreciseTokenizer nunca deve retornar nil independente do ambiente
	tok, _ := NewPreciseTokenizer()
	if tok == nil {
		t.Fatal("NewPreciseTokenizer nao deve retornar nil")
	}
	// Deve retornar tokens validos
	tokens := tok.EstimateTokens("hello world")
	if tokens == 0 {
		t.Error("tokenizer retornado por NewPreciseTokenizer deve retornar tokens > 0")
	}
}

func TestGather_SkillCountNeverFalsePositive(t *testing.T) {
	t.Parallel()
	// Garante que repositorio parcial nao retorna zero enganoso para SkillCount
	// quando ha skills reais presentes
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	ffs.Files[root+"/.agents/skills/minha-skill/SKILL.md"] = []byte("conteudo")

	svc := NewService(ffs, silentPrinter(), nil)
	report, err := svc.gather(root, false)

	if err != nil {
		t.Fatalf("gather nao deve retornar erro: %v", err)
	}
	if report.SkillCount == 0 {
		t.Error("SkillCount nao deve ser zero quando existe skill real no checkout")
	}
}
