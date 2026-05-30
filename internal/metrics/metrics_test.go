package metrics

import (
	"encoding/json"
	"io"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

type MetricsSuite struct {
	suite.Suite
}

func TestMetricsSuite(t *testing.T) {
	suite.Run(t, new(MetricsSuite))
}

// silentPrinter retorna um Printer que descarta toda saida (nao polui stdout nos testes).
func silentPrinter() *output.Printer {
	return &output.Printer{Out: io.Discard, Err: io.Discard}
}

func (s *MetricsSuite) TestGather() {
	scenarios := []struct {
		name    string
		setupFS func(ffs *fs.FakeFileSystem, root string)
		expect  func(report Report, err error)
	}{
		{
			name: "deve coletar skills e referencias",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Files[root+"/.agents/skills/skill-a/SKILL.md"] = []byte("# Skill A\nconteudo da skill A com algumas palavras")
				ffs.Files[root+"/.agents/skills/skill-a/references/ref1.md"] = []byte("# Referencia\nconteudo da referencia")
				ffs.Files[root+"/.agents/skills/skill-b/SKILL.md"] = []byte("# Skill B\nconteudo da skill B")
			},
			expect: func(report Report, err error) {
				s.NoError(err)
				s.Equal(2, report.SkillCount)
				s.Equal(1, report.RefCount)
				s.Contains(report.Baselines, "skill-a")
				s.Contains(report.Baselines, "skill-b")
				s.True(report.Baselines["skill-a"].TokensEst > 0, "skill-a deve ter TokensEst > 0")
			},
		},
		{
			name: "deve ignorar diretorio sem SKILL.md",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Dirs[root+"/.agents/skills"] = true
				ffs.Dirs[root+"/.agents/skills/skill-sem-skillmd"] = true
			},
			expect: func(report Report, err error) {
				s.NoError(err)
				s.Len(report.SkippedDirs, 1)
				s.Equal(".agents/skills/skill-sem-skillmd", report.SkippedDirs[0])
				s.Zero(report.SkillCount)
			},
		},
		{
			name:    "deve retornar erro quando diretorio de skills nao existe",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {},
			expect: func(report Report, err error) {
				s.Error(err)
				s.Contains(err.Error(), "diretorio de skills nao encontrado")
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			root := "/repo"
			scenario.setupFS(ffs, root)
			svc := NewService(ffs, silentPrinter(), nil)

			report, err := svc.gather(root, false)

			scenario.expect(report, err)
		})
	}
}

func (s *MetricsSuite) TestExecutePropagatesGatherError() {
	ffs := fs.NewFakeFileSystem()
	svc := NewService(ffs, silentPrinter(), nil)

	err := svc.Execute("/repo", "table", false)

	s.Error(err)
}

func (s *MetricsSuite) TestTokenEstimateSanityCheck() {
	sample := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100)
	est := int(math.Round(float64(len(sample)) / 3.5))

	s.True(est >= 900 && est <= 1600, "estimativa fora de faixa aceitavel: %d tokens para %d chars", est, len(sample))
}

func (s *MetricsSuite) TestCharEstimatorDeterministic() {
	text := "The quick brown fox jumps over the lazy dog."
	tok := NewCharEstimator()
	a := tok.EstimateTokens(text)
	b := tok.EstimateTokens(text)

	s.Equal(a, b, "CharEstimator deve ser deterministico")
	s.True(a > 0, "CharEstimator deve retornar tokens > 0 para texto nao-vazio")
	s.Equal("chars/3.5", tok.Name())
}

func (s *MetricsSuite) TestTiktokenEstimatorWhenAvailable() {
	tok, err := NewTiktokenEstimator()
	if err != nil {
		s.T().Skipf("tiktoken nao disponivel (sem acesso ao modelo BPE): %v", err)
	}
	text := "The quick brown fox jumps over the lazy dog."
	tokens := tok.EstimateTokens(text)

	s.True(tokens > 0, "TiktokenEstimator deve retornar tokens > 0 para texto nao-vazio")
	s.True(tokens >= 5 && tokens <= 20, "TiktokenEstimator: contagem inesperada %d para frase conhecida", tokens)
	s.Equal("tiktoken/cl100k_base", tok.Name())
}

func (s *MetricsSuite) TestTiktokenEstimatorMoreAccurateThanChar() {
	tok, err := NewTiktokenEstimator()
	if err != nil {
		s.T().Skipf("tiktoken nao disponivel: %v", err)
	}
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 50)
	charEst := NewCharEstimator().EstimateTokens(text)
	tiktokenEst := tok.EstimateTokens(text)

	s.True(charEst > 0 && tiktokenEst > 0, "ambos estimadores devem retornar > 0")
	diff := charEst - tiktokenEst
	if diff < 0 {
		diff = -diff
	}
	pct := float64(diff) / float64(charEst) * 100
	s.True(pct <= 25, "divergencia entre chars/3.5 e tiktoken maior que esperada: %.1f%% (char=%d tiktoken=%d)", pct, charEst, tiktokenEst)
}

func (s *MetricsSuite) TestGatherSkipNonSkillDirs() {
	scenarios := []struct {
		name             string
		setupFS          func(ffs *fs.FakeFileSystem, root string)
		wantSkillCount   int
		wantSkippedCount int
		wantSkippedPath  string
		wantSkillName    string
		checkJSON        bool
	}{
		{
			name: "deve coletar skill normal",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Files[root+"/.agents/skills/skill-x/SKILL.md"] = []byte("# Skill X\nconteudo")
			},
			wantSkillCount:   1,
			wantSkippedCount: 0,
			wantSkillName:    "skill-x",
		},
		{
			name: "deve ignorar diretorio de testes sem SKILL.md",
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
			name: "deve coletar skill e ignorar diretorio sem SKILL.md",
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
			name: "deve ignorar diretorio vazio sem SKILL.md",
			setupFS: func(ffs *fs.FakeFileSystem, root string) {
				ffs.Dirs[root+"/.agents/skills"] = true
				ffs.Dirs[root+"/.agents/skills/empty"] = true
			},
			wantSkillCount:   0,
			wantSkippedCount: 1,
			wantSkippedPath:  ".agents/skills/empty",
		},
		{
			name: "deve serializar skipped_dirs em JSON",
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

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			ffs := fs.NewFakeFileSystem()
			root := "/repo"
			scenario.setupFS(ffs, root)
			svc := NewService(ffs, silentPrinter(), nil)

			report, err := svc.gather(root, false)

			s.NoError(err)
			s.Equal(scenario.wantSkillCount, report.SkillCount)
			s.Len(report.SkippedDirs, scenario.wantSkippedCount)
			if scenario.wantSkippedPath != "" {
				s.Contains(report.SkippedDirs, scenario.wantSkippedPath)
			}
			if scenario.wantSkillName != "" {
				s.Contains(report.Baselines, scenario.wantSkillName)
			}
			if scenario.checkJSON {
				data, err := json.Marshal(report)
				s.NoError(err)
				jsonStr := string(data)
				s.Contains(jsonStr, "skipped_dirs")
				s.Contains(jsonStr, ".agents/skills/tests")
			}
		})
	}
}

func (s *MetricsSuite) TestGatherSkippedDirsAlwaysForwardSlash() {
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	ffs.Dirs[root+"/.agents/skills"] = true
	ffs.Dirs[root+"/.agents/skills/tests"] = true
	ffs.Files[root+"/.agents/skills/tests/conftest.py"] = []byte("# pytest")

	svc := NewService(ffs, silentPrinter(), nil)
	report, err := svc.gather(root, false)

	s.NoError(err)
	s.Len(report.SkippedDirs, 1)
	got := report.SkippedDirs[0]
	s.False(strings.Contains(got, "\\"), "SkippedDirs[0] contem backslash (separador Windows): %q — deve sempre usar forward slash", got)
	s.Equal(".agents/skills/tests", got)
}

func (s *MetricsSuite) TestNewPreciseTokenizerFallbackReturnsCharEstimator() {
	tok, _ := NewPreciseTokenizer()

	s.NotNil(tok, "NewPreciseTokenizer nao deve retornar nil")
	tokens := tok.EstimateTokens("hello world")
	s.True(tokens > 0, "tokenizer retornado por NewPreciseTokenizer deve retornar tokens > 0")
}

func (s *MetricsSuite) TestGatherSkillCountNeverFalsePositive() {
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	ffs.Files[root+"/.agents/skills/minha-skill/SKILL.md"] = []byte("conteudo")

	svc := NewService(ffs, silentPrinter(), nil)
	report, err := svc.gather(root, false)

	s.NoError(err)
	s.NotZero(report.SkillCount, "SkillCount nao deve ser zero quando existe skill real no checkout")
}

func (s *MetricsSuite) TestExecuteTextFormatWithSkippedDirs() {
	ffs := fs.NewFakeFileSystem()
	root := "/repo"
	ffs.Files[root+"/.agents/skills/skill-x/SKILL.md"] = []byte("# Skill X\nconteudo")
	ffs.Dirs[root+"/.agents/skills/tests"] = true
	ffs.Files[root+"/.agents/skills/tests/conftest.py"] = []byte("# pytest")

	var outBuf strings.Builder
	printer := &output.Printer{Out: &outBuf, Err: io.Discard}
	svc := NewService(ffs, printer, nil)

	err := svc.Execute(root, "table", false)

	s.NoError(err)
	out := outBuf.String()
	s.Contains(out, "Diretorios ignorados (sem SKILL.md): 1")
	s.Contains(out, ".agents/skills/tests")
}

func (s *MetricsSuite) TestFormatReportContainsSkillsAndRefs() {
	r := Report{
		Baselines: map[string]BaselineEntry{
			"skill-a": {Words: 10, Chars: 50, TokensEst: 14},
		},
		Flows:      map[string]FlowEntry{},
		SkillCount: 1,
		RefCount:   0,
	}

	out := NewCatalog().FormatReport(r)

	s.Contains(out, "skill-a")
	s.Contains(out, "Skills: 1")
}
