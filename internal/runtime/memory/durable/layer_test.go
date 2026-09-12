package durable_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type inMemoryLayerLocker struct {
	mu       sync.Mutex
	locked   map[string]bool
	order    []string
	refuseOn map[string]bool
}

func newInMemoryLayerLocker() *inMemoryLayerLocker {
	return &inMemoryLayerLocker{
		locked:   make(map[string]bool),
		refuseOn: make(map[string]bool),
	}
}

func (l *inMemoryLayerLocker) Lock(path string) (func() error, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.refuseOn[path] {
		return nil, durable.ErrLayerLocked
	}
	if l.locked[path] {
		return nil, durable.ErrLayerLocked
	}
	l.locked[path] = true
	l.order = append(l.order, path)

	return func() error {
		l.mu.Lock()
		defer l.mu.Unlock()
		delete(l.locked, path)
		return nil
	}, nil
}

type directWriteRefusingFileSystem struct {
	*fs.FakeFileSystem
	directWriteCalls int
}

func newDirectWriteRefusingFileSystem() *directWriteRefusingFileSystem {
	return &directWriteRefusingFileSystem{FakeFileSystem: fs.NewFakeFileSystem()}
}

func (f *directWriteRefusingFileSystem) WriteFile(path string, data []byte) error {
	f.directWriteCalls++
	return errors.New("direct write must not be called: use WriteFileAtomic")
}

type LayerSuite struct {
	suite.Suite
}

func TestLayerSuite(t *testing.T) {
	suite.Run(t, new(LayerSuite))
}

func (s *LayerSuite) durableFact(key, hash, content string) durable.Fact {
	return durable.Fact{
		Identity:   durable.Identity{Key: durable.SemanticKey(key), Hash: durable.ContentHash(hash)},
		Content:    content,
		Durability: durable.DurabilityDurable,
		Origin:     durable.FactOrigin{Session: "sess-1", CLI: "claude", Task: "5.0"},
	}
}

func (s *LayerSuite) TestPathResolution() {
	scenarios := []struct {
		name    string
		scope   durable.Scope
		wantErr error
	}{
		{
			name:  "should resolve project layer under .aispec/memory",
			scope: durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"},
		},
		{
			name:  "should resolve prd layer preserving the current MEMORY.md path",
			scope: durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"},
		},
		{
			name:  "should resolve task layer using the task file base name",
			scope: durable.Scope{Layer: durable.TargetLayerTask, TasksDir: "/repo/.specs/prd-x", TaskFileName: "task-5.0-foo.md"},
		},
		{
			name:    "should fail when prd layer has no tasks directory",
			scope:   durable.Scope{Layer: durable.TargetLayerPRD},
			wantErr: durable.ErrTasksDirMissing,
		},
		{
			name:    "should fail when task layer has no task file name",
			scope:   durable.Scope{Layer: durable.TargetLayerTask, TasksDir: "/repo/.specs/prd-x"},
			wantErr: durable.ErrTaskFileNameMissing,
		},
		{
			name:    "should fail when project layer has no project directory",
			scope:   durable.Scope{Layer: durable.TargetLayerProject},
			wantErr: durable.ErrProjectDirMissing,
		},
		{
			name:    "should fail when layer is undefined",
			scope:   durable.Scope{},
			wantErr: durable.ErrLayerUndefined,
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			filesystem := fs.NewFakeFileSystem()
			layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())

			_, _, err := layer.Read(context.Background(), sc.scope)

			if sc.wantErr != nil {
				s.True(errors.Is(err, sc.wantErr))
				return
			}
			s.NoError(err)
		})
	}
}

func (s *LayerSuite) TestProjectLayerPathMatchesDotAispecMemory() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"}

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("arch.decision", "sha256:aaaa", "content")})
	s.Require().NoError(err)

	s.True(filesystem.Exists("/repo/.aispec/memory/PROJECT.md"))
}

func (s *LayerSuite) TestPRDLayerPathMatchesCurrentStorePath() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("prd.decision", "sha256:aaaa", "content")})
	s.Require().NoError(err)

	s.True(filesystem.Exists("/repo/.specs/prd-x/memory/MEMORY.md"))
}

func (s *LayerSuite) TestReadOnEmptyRepositoryDoesNotFail() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"}

	facts, human, err := layer.Read(context.Background(), scope)

	s.NoError(err)
	s.Empty(facts)
	s.Equal(durable.HumanBlock{}, human)
}

func (s *LayerSuite) TestConsolidateThenReadReturnsAddedFact() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	result, err := layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("payment.retry", "sha256:aaaa", "retries three times")})
	s.Require().NoError(err)
	s.Equal([]durable.Identity{{Key: "payment.retry", Hash: "sha256:aaaa"}}, result.Added)

	facts, _, err := layer.Read(context.Background(), scope)
	s.Require().NoError(err)
	s.Require().Len(facts, 1)
	s.Equal(durable.FactStateActive, facts[0].State)
}

func (s *LayerSuite) TestConsolidateDoesNotRemoveFactsNotMentioned() {
	filesystem := fs.NewFakeFileSystem()
	locker := newInMemoryLayerLocker()
	layer := durable.NewLayerWithLocker(filesystem, locker)
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("payment.retry", "sha256:aaaa", "first fact")})
	s.Require().NoError(err)

	_, err = layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("payment.timeout", "sha256:bbbb", "second fact")})
	s.Require().NoError(err)

	facts, _, err := layer.Read(context.Background(), scope)
	s.Require().NoError(err)
	s.Len(facts, 2)
}

func (s *LayerSuite) TestConsolidateIdenticalContentIsIdempotent() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	fact := s.durableFact("payment.retry", "sha256:aaaa", "retries three times")

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{fact})
	s.Require().NoError(err)

	result, err := layer.Consolidate(context.Background(), scope, []durable.Fact{fact})
	s.Require().NoError(err)
	s.Empty(result.Added)
	s.Equal([]durable.Identity{fact.Identity}, result.Unchanged)

	facts, _, err := layer.Read(context.Background(), scope)
	s.Require().NoError(err)
	s.Len(facts, 1)
}

func (s *LayerSuite) TestConsolidateContradictingFactMarksOldAndKeepsBoth() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	original := s.durableFact("payment.retry", "sha256:aaaa", "retries three times")
	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{original})
	s.Require().NoError(err)

	contradicting := s.durableFact("payment.retry", "sha256:bbbb", "retries five times")
	result, err := layer.Consolidate(context.Background(), scope, []durable.Fact{contradicting})
	s.Require().NoError(err)
	s.Equal([]durable.Identity{original.Identity}, result.Contradicted)
	s.Equal([]durable.Identity{contradicting.Identity}, result.Added)

	activeFacts, _, err := layer.Read(context.Background(), scope)
	s.Require().NoError(err)
	s.Require().Len(activeFacts, 2)

	byHash := make(map[durable.ContentHash]durable.Fact, 2)
	for _, f := range activeFacts {
		byHash[f.Identity.Hash] = f
	}
	s.Equal(durable.FactStateContradicted, byHash["sha256:aaaa"].State)
	s.Equal(durable.FactStateActive, byHash["sha256:bbbb"].State)
	s.Require().Len(byHash["sha256:bbbb"].Links, 1)
	s.Equal(durable.LinkTypeContradicts, byHash["sha256:bbbb"].Links[0].Type)
	s.Equal(original.Identity, byHash["sha256:bbbb"].Links[0].Target)
}

func (s *LayerSuite) TestConsolidateRejectsFactWithoutDurability() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	invalid := s.durableFact("payment.retry", "sha256:aaaa", "content")
	invalid.Durability = durable.DurabilityInvalid

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{invalid})

	s.True(errors.Is(err, durable.ErrDurabilityMissing))
	s.False(filesystem.Exists("/repo/.specs/prd-x/memory/MEMORY.md"))
}

func (s *LayerSuite) TestArchiveRemovesFactFromActiveSetButKeepsItOnDisk() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	fact := s.durableFact("payment.retry", "sha256:aaaa", "retries three times")
	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{fact})
	s.Require().NoError(err)

	err = layer.Archive(context.Background(), scope, []durable.Identity{fact.Identity})
	s.Require().NoError(err)

	activeFacts, _, err := layer.Read(context.Background(), scope)
	s.Require().NoError(err)
	s.Empty(activeFacts)

	raw, err := filesystem.ReadFile("/repo/.specs/prd-x/memory/MEMORY.md")
	s.Require().NoError(err)
	s.Contains(string(raw), "payment.retry")
	s.Contains(string(raw), "state: archived")
}

func (s *LayerSuite) TestArchiveThenReconsolidateReverses() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	fact := s.durableFact("payment.retry", "sha256:aaaa", "retries three times")
	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{fact})
	s.Require().NoError(err)

	err = layer.Archive(context.Background(), scope, []durable.Identity{fact.Identity})
	s.Require().NoError(err)

	result, err := layer.Consolidate(context.Background(), scope, []durable.Fact{fact})
	s.Require().NoError(err)
	s.Equal([]durable.Identity{fact.Identity}, result.Restored)

	activeFacts, _, err := layer.Read(context.Background(), scope)
	s.Require().NoError(err)
	s.Require().Len(activeFacts, 1)
	s.Equal(durable.FactStateActive, activeFacts[0].State)
}

func (s *LayerSuite) TestPromoteMovesFactAcrossLayers() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	from := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}
	to := durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"}

	fact := s.durableFact("arch.decision", "sha256:aaaa", "decided to use X")
	_, err := layer.Consolidate(context.Background(), from, []durable.Fact{fact})
	s.Require().NoError(err)

	err = layer.Promote(context.Background(), from, to, fact.Identity)
	s.Require().NoError(err)

	fromFacts, _, err := layer.Read(context.Background(), from)
	s.Require().NoError(err)
	s.Empty(fromFacts)

	toFacts, _, err := layer.Read(context.Background(), to)
	s.Require().NoError(err)
	s.Require().Len(toFacts, 1)
	s.Equal(fact.Identity, toFacts[0].Identity)
	s.Equal(durable.FactStateActive, toFacts[0].State)
}

func (s *LayerSuite) TestPromoteRejectsEphemeralFact() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	from := durable.Scope{Layer: durable.TargetLayerTask, TasksDir: "/repo/.specs/prd-x", TaskFileName: "task-5.0-foo.md"}
	to := durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"}

	fact := s.durableFact("scratch.note", "sha256:aaaa", "ephemeral scratch note")
	fact.Durability = durable.DurabilityEphemeral
	_, err := layer.Consolidate(context.Background(), from, []durable.Fact{fact})
	s.Require().NoError(err)

	err = layer.Promote(context.Background(), from, to, fact.Identity)

	s.True(errors.Is(err, durable.ErrPromotionWithoutMark))
}

func (s *LayerSuite) TestPromoteMissingFactReportsNotFound() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	from := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}
	to := durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"}

	err := layer.Promote(context.Background(), from, to, durable.Identity{Key: "missing.fact", Hash: "sha256:aaaa"})

	s.True(errors.Is(err, durable.ErrFactNotFound))
}

func (s *LayerSuite) TestConsolidateWritesOnlyViaAtomicWrite() {
	filesystem := newDirectWriteRefusingFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("payment.retry", "sha256:aaaa", "content")})

	s.Require().NoError(err)
	s.Equal(0, filesystem.directWriteCalls)
}

func (s *LayerSuite) TestConsolidateSurfacesLockContentionWithoutCorruptingPage() {
	filesystem := fs.NewFakeFileSystem()
	locker := newInMemoryLayerLocker()
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}
	layer := durable.NewLayerWithLocker(filesystem, locker)

	lockPath := "/repo/.specs/prd-x/memory/MEMORY.lock"
	locker.refuseOn[lockPath] = true

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("payment.retry", "sha256:aaaa", "content")})

	s.True(errors.Is(err, durable.ErrLayerLocked))
	s.False(filesystem.Exists("/repo/.specs/prd-x/memory/MEMORY.md"))
}

func (s *LayerSuite) TestConsolidateReleasesLockAfterWrite() {
	filesystem := fs.NewFakeFileSystem()
	locker := newInMemoryLayerLocker()
	layer := durable.NewLayerWithLocker(filesystem, locker)
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}

	_, err := layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("payment.retry", "sha256:aaaa", "content")})
	s.Require().NoError(err)

	_, err = layer.Consolidate(context.Background(), scope, []durable.Fact{s.durableFact("payment.timeout", "sha256:bbbb", "content")})
	s.Require().NoError(err)
}

func (s *LayerSuite) TestPromoteLocksBothLayersInDeterministicOrder() {
	filesystem := fs.NewFakeFileSystem()
	locker := newInMemoryLayerLocker()
	layer := durable.NewLayerWithLocker(filesystem, locker)
	from := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}
	to := durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"}

	fact := s.durableFact("arch.decision", "sha256:aaaa", "decided to use X")
	_, err := layer.Consolidate(context.Background(), from, []durable.Fact{fact})
	s.Require().NoError(err)
	locker.order = nil

	err = layer.Promote(context.Background(), from, to, fact.Identity)
	s.Require().NoError(err)

	s.Require().Len(locker.order, 2)
	s.Less(locker.order[0], locker.order[1])
}

func (s *LayerSuite) TestReadOnInvalidPageInOneScopeDoesNotAffectAnotherScope() {
	filesystem := fs.NewFakeFileSystem()
	layer := durable.NewLayerWithLocker(filesystem, newInMemoryLayerLocker())
	corruptedScope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/repo/.specs/prd-x"}
	healthyScope := durable.Scope{Layer: durable.TargetLayerProject, ProjectDir: "/repo"}

	s.Require().NoError(filesystem.WriteFile("/repo/.specs/prd-x/memory/MEMORY.md", []byte("---\n[invalid yaml\n---\nbody\n")))

	healthyFact := s.durableFact("arch.decision", "sha256:aaaa", "content")
	_, err := layer.Consolidate(context.Background(), healthyScope, []durable.Fact{healthyFact})
	s.Require().NoError(err)

	_, _, err = layer.Read(context.Background(), corruptedScope)
	s.True(errors.Is(err, durable.ErrPageUnreadable))

	facts, _, err := layer.Read(context.Background(), healthyScope)
	s.Require().NoError(err)
	s.Len(facts, 1)
}

func (s *LayerSuite) TestSearchPolicyFindsByTextAndByEntity() {
	facts := []durable.Fact{
		s.durableFact("payment.retry", "sha256:aaaa", "retries three times before failing"),
		s.durableFact("payment.timeout", "sha256:bbbb", "timeout raised to 30s"),
	}

	policy := durable.SearchPolicy{}

	textMatches := policy.Text(facts, "timeout")
	s.Require().Len(textMatches, 1)
	s.Equal(durable.SemanticKey("payment.timeout"), textMatches[0].Identity.Key)

	entityMatches := policy.Entity(facts, "payment")
	s.Len(entityMatches, 2)

	s.Empty(policy.Text(facts, ""))
	s.Empty(policy.Entity(facts, ""))
}
