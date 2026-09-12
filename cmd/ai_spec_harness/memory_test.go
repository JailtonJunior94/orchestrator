package aispecharness

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type memoryTestLocker struct {
	mu     sync.Mutex
	locked map[string]bool
}

func newMemoryTestLocker() *memoryTestLocker {
	return &memoryTestLocker{locked: make(map[string]bool)}
}

func (l *memoryTestLocker) Lock(path string) (func() error, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.locked[path] {
		return nil, durable.ErrLayerLocked
	}
	l.locked[path] = true
	return func() error {
		l.mu.Lock()
		defer l.mu.Unlock()
		delete(l.locked, path)
		return nil
	}, nil
}

func newTestPrinter() (*output.Printer, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return &output.Printer{Out: out, Err: errOut}, out, errOut
}

type MemoryCommandSuite struct {
	suite.Suite
}

func TestMemoryCommandSuite(t *testing.T) {
	suite.Run(t, new(MemoryCommandSuite))
}

func (s *MemoryCommandSuite) newLayer(fsys *fs.FakeFileSystem) durable.Layer {
	return durable.NewLayerWithLocker(fsys, newMemoryTestLocker())
}

func (s *MemoryCommandSuite) seedFact(fsys *fs.FakeFileSystem, layer durable.Layer, scope durable.Scope, key, content string, durability durable.Durability) durable.Fact {
	catalog := durable.NewCatalog()
	fact := durable.Fact{
		Identity:   durable.Identity{Key: durable.SemanticKey(key), Hash: catalog.HashContent(content)},
		Content:    content,
		Durability: durability,
		Origin:     durable.FactOrigin{Session: "session-1", CLI: "claude", Task: "task-1.0.md", Date: "2026-09-11T00:00:00Z"},
	}
	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{fact})
	s.Require().NoError(err)
	_ = fsys
	return fact
}

func (s *MemoryCommandSuite) TestRunShowPrintsFactsWithOriginAndFlagsContradiction() {
	tests := []struct {
		name           string
		seedContradict bool
		wantFlag       string
	}{
		{name: "fato simples sem contradicao", seedContradict: false, wantFlag: ""},
		{name: "fato contraditorio sinalizado", seedContradict: true, wantFlag: "[CONTRADITORIO]"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			fsys := fs.NewFakeFileSystem()
			layer := s.newLayer(fsys)
			handler := &memoryCommand{}
			scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/project/.specs/prd-x"}

			s.seedFact(fsys, layer, scope, "decision.retry", "use exponential backoff", durable.DurabilityPRD)
			if tc.seedContradict {
				s.seedFact(fsys, layer, scope, "decision.retry", "use fixed backoff", durable.DurabilityPRD)
			}

			printer, out, _ := newTestPrinter()
			err := handler.runShow(context.Background(), printer, layer, "/project", "/project/.specs/prd-x", "")
			s.Require().NoError(err)

			s.Contains(out.String(), "origem: sessao=session-1 cli=claude task=task-1.0.md")
			if tc.wantFlag != "" {
				s.Contains(out.String(), tc.wantFlag)
			} else {
				s.NotContains(out.String(), "[CONTRADITORIO]")
			}
		})
	}
}

func (s *MemoryCommandSuite) TestRunShowWithNoMemoryPrintsEmptyMessage() {
	fsys := fs.NewFakeFileSystem()
	layer := s.newLayer(fsys)
	handler := &memoryCommand{}

	printer, out, _ := newTestPrinter()
	err := handler.runShow(context.Background(), printer, layer, "/project", "", "")
	s.Require().NoError(err)
	s.Contains(out.String(), "Nenhum fato de memoria encontrado")
}

func (s *MemoryCommandSuite) TestRunSearchFindsByTextAndByEntity() {
	fsys := fs.NewFakeFileSystem()
	layer := s.newLayer(fsys)
	handler := &memoryCommand{}
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/project/.specs/prd-x"}
	s.seedFact(fsys, layer, scope, "decision.retry", "use exponential backoff for retries", durable.DurabilityPRD)

	tests := []struct {
		name     string
		query    string
		byEntity bool
		want     string
	}{
		{name: "busca por texto encontra conteudo", query: "backoff", byEntity: false, want: "use exponential backoff"},
		{name: "busca por entidade encontra chave", query: "decision.retry", byEntity: true, want: "use exponential backoff"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			printer, out, _ := newTestPrinter()
			err := handler.runSearch(context.Background(), printer, layer, "/project", "/project/.specs/prd-x", "", tc.query, tc.byEntity)
			s.Require().NoError(err)
			s.Contains(out.String(), tc.want)
		})
	}
}

func (s *MemoryCommandSuite) TestRunSearchWithoutMatchReportsEmpty() {
	fsys := fs.NewFakeFileSystem()
	layer := s.newLayer(fsys)
	handler := &memoryCommand{}

	printer, out, _ := newTestPrinter()
	err := handler.runSearch(context.Background(), printer, layer, "/project", "", "", "nao-existe", false)
	s.Require().NoError(err)
	s.Contains(out.String(), "Nenhum resultado")
}

func (s *MemoryCommandSuite) TestRunExportWritesSelfContainedArtifact() {
	fsys := fs.NewFakeFileSystem()
	layer := s.newLayer(fsys)
	handler := &memoryCommand{}
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/project/.specs/prd-x"}
	s.seedFact(fsys, layer, scope, "decision.retry", "use exponential backoff", durable.DurabilityPRD)

	printer, out, _ := newTestPrinter()
	err := handler.runExport(context.Background(), printer, fsys, layer, "/project", "/project/.specs/prd-x", "", "/out/export.md")
	s.Require().NoError(err)
	s.Contains(out.String(), "Exportacao gravada")

	content, readErr := fsys.ReadFile("/out/export.md")
	s.Require().NoError(readErr)
	s.Contains(string(content), "decision.retry")
	s.Contains(string(content), "use exponential backoff")
}

func (s *MemoryCommandSuite) TestRunCompactArchivesExcessFactsAndPreservesHumanBlock() {
	fsys := fs.NewFakeFileSystem()
	layer := s.newLayer(fsys)
	handler := &memoryCommand{}
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/project/.specs/prd-x"}

	for i := 0; i < 5; i++ {
		key := durable.SemanticKey("decision.item-" + string(rune('a'+i)))
		content := "a fairly long durable decision content that pushes the page over a tiny compaction limit " + string(rune('a'+i))
		catalog := durable.NewCatalog()
		fact := durable.Fact{
			Identity:   durable.Identity{Key: key, Hash: catalog.HashContent(content)},
			Content:    content,
			Durability: durable.DurabilityPRD,
			Origin:     durable.FactOrigin{Task: "task-1.0.md"},
		}
		_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{fact})
		s.Require().NoError(err)
	}

	printer, out, _ := newTestPrinter()
	cfg := durable.CompactionConfig{LineLimit: 10, ByteLimit: 400}
	err := handler.runCompact(context.Background(), printer, layer, "/project", "/project/.specs/prd-x", "task-1.0.md", cfg)
	s.Require().NoError(err)
	s.Contains(out.String(), "fato(s) arquivado(s)")

	active, _, readErr := layer.Read(context.Background(), scope)
	s.Require().NoError(readErr)
	s.Less(len(active), 5, "compactacao deve reduzir o conjunto ativo quando o limite e excedido")
}

func (s *MemoryCommandSuite) TestRunMigrateConvertsLegacyContentAndRefusesReapplication() {
	fsys := fs.NewFakeFileSystem()
	handler := &memoryCommand{}
	tasksDir := "/project/.specs/prd-x"
	legacyContent := "# Workflow Memory\n\n## Last Session Summary\n\n- Task: task-1.0.md\n- Exit Status: success\n"

	s.Require().NoError(fsys.MkdirAll(tasksDir + "/memory"))
	s.Require().NoError(fsys.WriteFile(tasksDir+"/memory/MEMORY.md", []byte(legacyContent)))

	printer, out, _ := newTestPrinter()
	err := handler.runMigrate(printer, fsys, tasksDir)
	s.Require().NoError(err)
	s.Contains(out.String(), "Migracao concluida")

	backup, backupErr := fsys.ReadFile(tasksDir + "/memory/MEMORY.md" + migrationBackupSuffix)
	s.Require().NoError(backupErr)
	s.Equal(legacyContent, string(backup), "o backup deve preservar o conteudo original byte a byte")

	migrated, migratedErr := fsys.ReadFile(tasksDir + "/memory/MEMORY.md")
	s.Require().NoError(migratedErr)
	s.Contains(string(migrated), legacyContent, "o conteudo legado deve ser preservado integralmente apos a migracao")
	s.Contains(string(migrated), "format_version: 1")

	reapplyErr := handler.runMigrate(output.New(false), fsys, tasksDir)
	s.Require().Error(reapplyErr)
	s.True(errors.Is(reapplyErr, durable.ErrMigrationAlreadyApplied), "reaplicar a migracao deve ser recusada com o erro tipado")
}

func (s *MemoryCommandSuite) TestRunMigrateWithoutLegacyFileIsANoop() {
	fsys := fs.NewFakeFileSystem()
	handler := &memoryCommand{}

	printer, out, _ := newTestPrinter()
	err := handler.runMigrate(printer, fsys, "/project/.specs/prd-empty")
	s.Require().NoError(err)
	s.Contains(out.String(), "Nada a migrar")
}

func (s *MemoryCommandSuite) TestRunMigrateRefusesExternalSymlink() {
	fsys := fs.NewFakeFileSystem()
	handler := &memoryCommand{}
	tasksDir := "/project/.specs/prd-x"
	legacyContent := "# Workflow Memory\n\n## Last Session Summary\n\n- Task: task-1.0.md\n- Exit Status: success\n"

	s.Require().NoError(fsys.MkdirAll("/outside"))
	s.Require().NoError(fsys.MkdirAll(tasksDir))
	s.Require().NoError(fsys.Symlink("/outside", tasksDir+"/memory"))
	s.Require().NoError(fsys.WriteFile(tasksDir+"/memory/MEMORY.md", []byte(legacyContent)))

	printer, _, _ := newTestPrinter()
	err := handler.runMigrate(printer, fsys, tasksDir)
	s.Require().Error(err, "escrever atraves de symlink para fora do PRD deve ser recusado (BUG-12)")
	s.Contains(err.Error(), "symlink")
}

func (s *MemoryCommandSuite) TestHandoffClaimRefusesExternalSymlink() {
	fsys := fs.NewFakeFileSystem()
	handler := &memoryCommand{locker: newMemoryTestLocker()}
	tasksDir := "/project/.specs/prd-x"

	s.Require().NoError(fsys.MkdirAll("/outside"))
	s.Require().NoError(fsys.MkdirAll(tasksDir))
	s.Require().NoError(fsys.Symlink("/outside", tasksDir+"/memory"))

	printer, _, _ := newTestPrinter()
	err := handler.runHandoffClaim(printer, fsys, tasksDir, "owner-a", time.Minute)
	s.Require().Error(err, "gravar o lease de bastao atraves de symlink para fora do PRD deve ser recusado (BUG-12)")
	s.Contains(err.Error(), "symlink")
}

func (s *MemoryCommandSuite) TestHandoffClaimStatusRelease() {
	fsys := fs.NewFakeFileSystem()
	handler := &memoryCommand{locker: newMemoryTestLocker()}
	tasksDir := "/project/.specs/prd-x"

	claimPrinter, claimOut, _ := newTestPrinter()
	err := handler.runHandoffClaim(claimPrinter, fsys, tasksDir, "owner-a", time.Minute)
	s.Require().NoError(err)
	s.Contains(claimOut.String(), "bastao reivindicado por owner-a")

	statusPrinter, statusOut, _ := newTestPrinter()
	err = handler.runHandoffStatus(statusPrinter, fsys, tasksDir)
	s.Require().NoError(err)
	s.Contains(statusOut.String(), "owner-a")

	concurrentPrinter, _, concurrentErrOut := newTestPrinter()
	err = handler.runHandoffClaim(concurrentPrinter, fsys, tasksDir, "owner-b", time.Minute)
	s.Require().Error(err, "uma segunda reivindicacao com dono ainda vivo e dentro do prazo deve ser recusada")
	s.Contains(concurrentErrOut.String(), "bastao recusado")

	releaseWrongOwnerPrinter, _, releaseWrongErrOut := newTestPrinter()
	err = handler.runHandoffRelease(releaseWrongOwnerPrinter, fsys, tasksDir, "owner-b")
	s.Require().Error(err, "apenas o dono atual pode liberar o bastao")
	s.Contains(releaseWrongErrOut.String(), "apenas o dono")

	releasePrinter, releaseOut, _ := newTestPrinter()
	err = handler.runHandoffRelease(releasePrinter, fsys, tasksDir, "owner-a")
	s.Require().NoError(err)
	s.Contains(releaseOut.String(), "bastao liberado por owner-a")

	statusAfterPrinter, statusAfterOut, _ := newTestPrinter()
	err = handler.runHandoffStatus(statusAfterPrinter, fsys, tasksDir)
	s.Require().NoError(err)
	s.Contains(statusAfterOut.String(), "Nenhum bastao ativo")
}

func (s *MemoryCommandSuite) TestHandoffClaimAllowsTakeoverAfterDeadlineExpires() {
	fsys := fs.NewFakeFileSystem()
	handler := &memoryCommand{locker: newMemoryTestLocker()}
	tasksDir := "/project/.specs/prd-x"

	printer, _, _ := newTestPrinter()
	err := handler.runHandoffClaim(printer, fsys, tasksDir, "owner-a", time.Millisecond)
	s.Require().NoError(err)

	time.Sleep(5 * time.Millisecond)

	takeoverPrinter, takeoverOut, _ := newTestPrinter()
	err = handler.runHandoffClaim(takeoverPrinter, fsys, tasksDir, "owner-b", time.Minute)
	s.Require().NoError(err, "apos o prazo vencer, uma nova reivindicacao deve suceder como tomada")
	s.Contains(takeoverOut.String(), "bastao tomado de owner-a")
}

func (s *MemoryCommandSuite) TestIsUnresolvedScopeClassifiesSentinelErrors() {
	handler := &memoryCommand{}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "diretorio de projeto ausente", err: durable.ErrProjectDirMissing, want: true},
		{name: "diretorio de tasks ausente", err: durable.ErrTasksDirMissing, want: true},
		{name: "nome de arquivo de task ausente", err: durable.ErrTaskFileNameMissing, want: true},
		{name: "camada indefinida", err: durable.ErrLayerUndefined, want: true},
		{name: "erro nao relacionado a resolucao de escopo", err: durable.ErrPageUnreadable, want: false},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			s.Equal(tc.want, handler.isUnresolvedScope(tc.err))
		})
	}
}

func (s *MemoryCommandSuite) TestBuildScopesResolvesLayersFromInputs() {
	handler := &memoryCommand{}

	onlyProject := handler.buildScopes("/project", "", "")
	s.Len(onlyProject, 1)
	s.Equal(durable.TargetLayerProject, onlyProject[0].Layer)

	projectAndPRD := handler.buildScopes("/project", "/project/.specs/prd-x", "")
	s.Len(projectAndPRD, 2)

	all := handler.buildScopes("/project", "/project/.specs/prd-x", "task-1.0.md")
	s.Len(all, 3)
	s.Equal(durable.TargetLayerTask, all[2].Layer)
	s.Equal("task-1.0.md", all[2].TaskFileName)
}
