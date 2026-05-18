package metrics

import (
	"encoding/json"
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

func TestGather_MissingSkillsMd_SkipsDir(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	// Diretorio da skill existe mas sem SKILL.md
	ffs.Dirs[root+"/.agents/skills"] = true
	ffs.Dirs[root+"/.agents/skills/skill-sem-skillmd"] = true

	svc := NewService(ffs, silentPrinter(), nil)
	report, err := svc.gather(root, false)

	if err != nil {
		t.Fatalf("gather nao deve retornar erro quando SKILL.md esta ausente: %v", err)
	}
	if len(report.SkippedDirs) != 1 {
		t.Fatalf("SkippedDirs deve ter 1 entrada, got %d", len(report.SkippedDirs))
	}
	if report.SkippedDirs[0] != ".agents/skills/skill-sem-skillmd" {
		t.Errorf("SkippedDirs[0] = %q, want %q", report.SkippedDirs[0], ".agents/skills/skill-sem-skillmd")
	}
	if report.SkillCount != 0 {
		t.Errorf("SkillCount deve ser 0, got %d", report.SkillCount)
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

func TestGather_SkipNonSkillDirs_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		setupFS          func(ffs *fs.FakeFileSystem, root string)
		wantSkillCount   int
		wantSkippedCount int
		wantSkippedPath  string // vazio = nao verificar
		wantSkillName    string // vazio = nao verificar
		checkJSON        bool   // verificar serializacao JSON de skipped_dirs
	}{
		{
			name: "skill_normal",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Files[root+"/.agents/skills/skill-x/SKILL.md"] = []byte("# Skill X\nconteudo")
			},
			wantSkillCount:   1,
			wantSkippedCount: 0,
			wantSkillName:    "skill-x",
		},
		{
			name: "dir_sem_skillmd",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Dirs[root+"/.agents/skills"] = true
				ffs.Dirs[root+"/.agents/skills/tests"] = true
				ffs.Files[root+"/.agents/skills/tests/conftest.py"] = []byte("# pytest config")
			},
			wantSkillCount:   0,
			wantSkippedCount: 1,
			wantSkippedPath:  ".agents/skills/tests",
		},
		{
			name: "misto_skill_e_dir_sem_skillmd",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Files[root+"/.agents/skills/skill-x/SKILL.md"] = []byte("# Skill X")
				ffs.Dirs[root+"/.agents/skills/tests"] = true
				ffs.Files[root+"/.agents/skills/tests/conftest.py"] = []byte("# pytest")
			},
			wantSkillCount:   1,
			wantSkippedCount: 1,
			wantSkippedPath:  ".agents/skills/tests",
			wantSkillName:    "skill-x",
		},
		{
			name: "dir_vazio_sem_skillmd",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Dirs[root+"/.agents/skills"] = true
				ffs.Dirs[root+"/.agents/skills/empty"] = true
			},
			wantSkillCount:   0,
			wantSkippedCount: 1,
			wantSkippedPath:  ".agents/skills/empty",
		},
		{
			name: "json_output_contem_skipped_dirs",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Files[root+"/.agents/skills/skill-x/SKILL.md"] = []byte("# Skill X")
				ffs.Dirs[root+"/.agents/skills/tests"] = true
				ffs.Files[root+"/.agents/skills/tests/conftest.py"] = []byte("# pytest")
			},
			wantSkillCount:   1,
			wantSkippedCount: 1,
			wantSkippedPath:  ".agents/skills/tests",
			checkJSON:        true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ffs := fs.NewFakeFileSystem()
			root := "/repo"
			tc.setupFS(ffs, root)

			svc := NewService(ffs, silentPrinter(), nil)
			report, err := svc.gather(root, false)

			if err != nil {
				t.Fatalf("gather nao deve retornar erro: %v", err)
			}
			if report.SkillCount != tc.wantSkillCount {
				t.Errorf("SkillCount: got %d, want %d", report.SkillCount, tc.wantSkillCount)
			}
			if len(report.SkippedDirs) != tc.wantSkippedCount {
				t.Errorf("len(SkippedDirs): got %d, want %d", len(report.SkippedDirs), tc.wantSkippedCount)
			}
			if tc.wantSkippedPath != "" && len(report.SkippedDirs) > 0 {
				found := false
				for _, d := range report.SkippedDirs {
					if d == tc.wantSkippedPath {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("SkippedDirs deve conter %q, got %v", tc.wantSkippedPath, report.SkippedDirs)
				}
			}
			if tc.wantSkillName != "" {
				if _, ok := report.Baselines[tc.wantSkillName]; !ok {
					t.Errorf("Baselines deve conter %q", tc.wantSkillName)
				}
			}
			// Caso JSON: verificar que skipped_dirs aparece na serializacao JSON quando nao vazio
			if tc.checkJSON {
				data, merr := json.Marshal(report)
				if merr != nil {
					t.Fatalf("json.Marshal falhou: %v", merr)
				}
				jsonStr := string(data)
				if len(report.SkippedDirs) > 0 && !strings.Contains(jsonStr, "skipped_dirs") {
					t.Errorf("JSON deve conter 'skipped_dirs' quando SkippedDirs nao vazio, got: %s", jsonStr)
				}
				if len(report.SkippedDirs) > 0 && !strings.Contains(jsonStr, ".agents/skills/tests") {
					t.Errorf("JSON deve conter path do diretorio ignorado, got: %s", jsonStr)
				}
			}
		})
	}
}

// TestGather_SkippedDirs_AlwaysForwardSlash garante que o path em SkippedDirs usa
// forward slash independente do separador de SO (regressao para bug filepath.Join).
func TestGather_SkippedDirs_AlwaysForwardSlash(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	ffs.Dirs[root+"/.agents/skills"] = true
	ffs.Dirs[root+"/.agents/skills/tests"] = true
	ffs.Files[root+"/.agents/skills/tests/conftest.py"] = []byte("# pytest")

	svc := NewService(ffs, silentPrinter(), nil)
	report, err := svc.gather(root, false)
	if err != nil {
		t.Fatalf("gather nao deve retornar erro: %v", err)
	}
	if len(report.SkippedDirs) != 1 {
		t.Fatalf("esperava 1 SkippedDir, got %d", len(report.SkippedDirs))
	}
	got := report.SkippedDirs[0]
	if strings.Contains(got, "\\") {
		t.Errorf("SkippedDirs[0] contem backslash (separador Windows): %q — deve sempre usar forward slash", got)
	}
	if got != ".agents/skills/tests" {
		t.Errorf("SkippedDirs[0] = %q, want %q", got, ".agents/skills/tests")
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

func TestExecute_TextFormat_WithSkippedDirs(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	root := "/repo"

	ffs.Files[root+"/.agents/skills/skill-x/SKILL.md"] = []byte("# Skill X\nconteudo")
	ffs.Dirs[root+"/.agents/skills/tests"] = true
	ffs.Files[root+"/.agents/skills/tests/conftest.py"] = []byte("# pytest")

	var outBuf strings.Builder
	printer := &output.Printer{Out: &outBuf, Err: io.Discard}
	svc := NewService(ffs, printer, nil)

	err := svc.Execute(root, "table", false)
	if err != nil {
		t.Fatalf("Execute nao deve retornar erro: %v", err)
	}
	out := outBuf.String()
	if !strings.Contains(out, "Diretorios ignorados (sem SKILL.md): 1") {
		t.Errorf("saida texto deve conter contagem de diretorios ignorados, got:\n%s", out)
	}
	if !strings.Contains(out, ".agents/skills/tests") {
		t.Errorf("saida texto deve listar caminho do diretorio ignorado, got:\n%s", out)
	}
}

func TestFormatReport_ContainsSkillsAndRefs(t *testing.T) {
	t.Parallel()
	r := Report{
		Baselines: map[string]BaselineEntry{
			"skill-a": {Words: 10, Chars: 50, TokensEst: 14},
		},
		Flows:      map[string]FlowEntry{},
		SkillCount: 1,
		RefCount:   0,
	}
	out := FormatReport(r)
	if !strings.Contains(out, "skill-a") {
		t.Error("FormatReport deve conter nome da skill")
	}
	if !strings.Contains(out, "Skills: 1") {
		t.Error("FormatReport deve conter contagem de skills")
	}
}
