package durable_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type FacadeSuite struct {
	suite.Suite
}

func TestFacadeSuite(t *testing.T) {
	suite.Run(t, new(FacadeSuite))
}

func (s *FacadeSuite) newFacade(fsys *fs.FakeFileSystem, projectDir, tasksDir string) *durable.Facade {
	layer := durable.NewLayerWithLocker(fsys, newInMemoryLayerLocker())
	return durable.NewFacadeWithLayerAndLocker(fsys, layer, newInMemoryLayerLocker(), durable.FacadeConfig{
		ProjectDir: projectDir,
		TasksDir:   tasksDir,
	})
}

func (s *FacadeSuite) TestRecordSessionRefusesSecretBeforeWritingAnything() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	pemBlock := "-----BEGIN " + "RSA PRIVATE KEY-----\nMIIBOgIBAAJBAK\n-----END " + "RSA PRIVATE KEY-----"

	_, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		ExitStatus:      "none",
		DeclaredSection: pemBlock,
	})

	s.True(errors.Is(err, durable.ErrSecretNotRedactable))

	exists := fsys.Exists("/project/.specs/prd-x/memory/task-1.0.md")
	s.False(exists, "sanitizacao deve preceder a persistencia: nada pode ser escrito quando o trecho sensivel nao e isolavel")
}

func (s *FacadeSuite) TestRecordSessionSanitizesBeforeConsolidating() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	token := "ghp_abcdefghijklmnopqrstuvwxyz012345"

	report, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		ExitStatus:      "none",
		DeclaredSection: "leaked token: " + token,
	})
	s.Require().NoError(err)
	s.Equal(1, report.Redactions)
	s.Positive(report.Writes)

	content, readErr := fsys.ReadFile("/project/.specs/prd-x/memory/MEMORY.md")
	s.Require().NoError(readErr)
	s.NotContains(string(content), token, "conteudo persistido nunca pode conter o segredo original — sanitizacao precede a escrita")
	s.Contains(string(content), "[REDACTED:")
}

func (s *FacadeSuite) TestRecordSessionWritesEphemeralAndPRDFacts() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	report, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		ExitStatus:      "none",
		EventsCount:     3,
		ToolCalls:       2,
		DeclaredSection: "decision: use exponential backoff for retries",
	})
	s.Require().NoError(err)
	s.Equal(2, report.Writes)
	s.True(fsys.Exists("/project/.specs/prd-x/memory/task-1.0.md"))
	s.True(fsys.Exists("/project/.specs/prd-x/memory/MEMORY.md"))
}

func (s *FacadeSuite) TestRecordSessionWithEmptySignalWritesNothing() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	report, err := facade.RecordSession(context.Background(), durable.SessionFacts{})
	s.Require().NoError(err)
	s.Equal(durable.MemoryReport{}, report)
}

func (s *FacadeSuite) TestRecordSessionClaimsBatonAndRenewsWithinSameProcess() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	first, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		DeclaredSection: "decision one",
	})
	s.Require().NoError(err)
	s.True(first.BatonClaimed)

	second, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		DeclaredSection: "decision two",
	})
	s.Require().NoError(err, "a mesma fachada de processo deve renovar o bastao, nunca se auto-recusar")
	s.True(second.BatonClaimed)
}

func (s *FacadeSuite) TestRecordSessionWritesFactsEvenWhenBatonClaimRefused() {
	fsys := fs.NewFakeFileSystem()
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/project/.specs/prd-x"}
	store := durable.NewHandoffLeaseStore(fsys, newInMemoryLayerLocker(), "/project/.specs/prd-x")
	_, seedErr := store.Claim(scope, durable.DefaultLeasePolicy, durable.LeaseOwner("pid:other-session"), durable.CurrentProcessRef(), durable.DefaultLeaseTTL, time.Now())
	s.Require().NoError(seedErr)

	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	report, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		DeclaredSection: "decision despite refused baton",
	})
	s.Require().NoError(err, "baton claim refusal must never abort fact consolidation (BUG-01, MD-001 precedence: no-loss before single-baton-owner)")
	s.False(report.BatonClaimed)
	s.NotEmpty(report.BatonRefusalReason)
	s.Positive(report.Writes)
	s.True(fsys.Exists("/project/.specs/prd-x/memory/task-1.0.md"))
	s.True(fsys.Exists("/project/.specs/prd-x/memory/MEMORY.md"))
}

func (s *FacadeSuite) TestRecordSessionWritesOtherLayersEvenWhenOneLayerRoundTripViolationOccurs() {
	fsys := fs.NewFakeFileSystem()
	s.Require().NoError(fsys.MkdirAll("/project/.specs/prd-x/memory"))
	s.Require().NoError(fsys.WriteFile("/project/.specs/prd-x/memory/MEMORY.md", []byte("human note without a trailing newline")))

	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	report, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		ExitStatus:      "none",
		DeclaredSection: "decision recorded despite a prd-layer round-trip violation",
	})
	s.Require().NoError(err, "a round-trip violation on one layer must never abort consolidation of the other layers (RF-36 recusa explicita, nao aborto de sessao)")
	s.Equal(1, report.WritesByLayer["task"], "task layer, unaffected by the prd page violation, must still be written")
	s.NotContains(report.WritesByLayer, "prd", "prd layer write must be skipped, not silently corrupted or partially applied")
	s.True(fsys.Exists("/project/.specs/prd-x/memory/task-1.0.md"))

	unchanged, readErr := fsys.ReadFile("/project/.specs/prd-x/memory/MEMORY.md")
	s.Require().NoError(readErr)
	s.Equal("human note without a trailing newline", string(unchanged), "the offending page must remain byte-identical, never partially rewritten")
}

func (s *FacadeSuite) TestBuildContextReturnsEmptyWhenNoMemoryExists() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "")

	memCtx, err := facade.BuildContext(context.Background(), durable.MemoryScope{})
	s.Require().NoError(err)
	s.Empty(memCtx.Block)
}

func (s *FacadeSuite) TestBuildContextRendersConsolidatedFacts() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	_, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		DeclaredSection: "durable decision fact",
	})
	s.Require().NoError(err)

	memCtx, err := facade.BuildContext(context.Background(), durable.MemoryScope{TaskFileName: "task-1.0.md"})
	s.Require().NoError(err)
	s.Contains(memCtx.Block, "durable decision fact")
}

func (s *FacadeSuite) TestBuildContextIsolatesUnreadablePageInsteadOfFailing() {
	fsys := fs.NewFakeFileSystem()
	malformed := "### Fact: broken.metadata\n```fact-metadata\nhash: [unterminated\n```\n"
	s.Require().NoError(fsys.MkdirAll("/project/.specs/prd-x/memory"))
	s.Require().NoError(fsys.WriteFile("/project/.specs/prd-x/memory/MEMORY.md", []byte(malformed)))

	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	memCtx, err := facade.BuildContext(context.Background(), durable.MemoryScope{})
	s.Require().NoError(err, "pagina ilegivel deve ser isolada, nao abortar a sessao (RF-22)")
	s.Positive(memCtx.Unreadable)
}

func (s *FacadeSuite) TestRecordSessionTracesFactToSessionCLIAndDate() {
	fsys := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(fsys, newInMemoryLayerLocker())
	facade := durable.NewFacadeWithLayerAndLocker(fsys, layer, newInMemoryLayerLocker(), durable.FacadeConfig{
		ProjectDir: "/project",
		TasksDir:   "/project/.specs/prd-x",
	})

	_, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		DeclaredSection: "durable decision fact",
		SessionID:       "sess-42",
		CLI:             "claude",
	})
	s.Require().NoError(err)

	facts, _, readErr := layer.Read(context.Background(), durable.Scope{
		Layer:      durable.TargetLayerPRD,
		ProjectDir: "/project",
		TasksDir:   "/project/.specs/prd-x",
	})
	s.Require().NoError(readErr)
	s.Require().NotEmpty(facts)
	s.Equal("sess-42", facts[0].Origin.Session)
	s.Equal("claude", facts[0].Origin.CLI)
	s.Equal("task-1.0.md", facts[0].Origin.Task)
	s.NotEmpty(facts[0].Origin.Date)
}

func (s *FacadeSuite) TestRecordSessionReportsWritesByLayer() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	report, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		EventsCount:     3,
		ToolCalls:       2,
		DeclaredSection: "decision: use exponential backoff for retries",
	})
	s.Require().NoError(err)
	s.Equal(1, report.WritesByLayer["task"])
	s.Equal(1, report.WritesByLayer["prd"])
}

func (s *FacadeSuite) TestBuildContextReportsBudgetByLayer() {
	fsys := fs.NewFakeFileSystem()
	facade := s.newFacade(fsys, "/project", "/project/.specs/prd-x")

	_, err := facade.RecordSession(context.Background(), durable.SessionFacts{
		TaskFileName:    "task-1.0.md",
		DeclaredSection: "durable decision fact",
	})
	s.Require().NoError(err)

	memCtx, err := facade.BuildContext(context.Background(), durable.MemoryScope{TaskFileName: "task-1.0.md"})
	s.Require().NoError(err)
	s.Positive(memCtx.BudgetByLayer["prd"])
	s.GreaterOrEqual(memCtx.BuildLatencyMs, int64(0))
}
