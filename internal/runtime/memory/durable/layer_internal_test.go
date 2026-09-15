package durable

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type noopLayerLocker struct{}

func (noopLayerLocker) Lock(string) (func() error, error) {
	return func() error { return nil }, nil
}

type LayerInternalSuite struct {
	suite.Suite
}

func TestLayerInternalSuite(t *testing.T) {
	suite.Run(t, new(LayerInternalSuite))
}

func (s *LayerInternalSuite) TestReadAllIncludesArchivedAndPromotedFacts() {
	filesystem := fs.NewFakeFileSystem()
	concrete := NewLayerWithLocker(filesystem, noopLayerLocker{}).(*layer)
	scope := Scope{Layer: TargetLayerPRD, TasksDir: "/project/.specs/prd-x"}
	ctx := context.Background()

	fact := Fact{
		Identity:   Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
		Durability: DurabilityPRD,
	}
	_, err := concrete.Consolidate(ctx, scope, []Fact{fact})
	s.Require().NoError(err)

	err = concrete.Archive(ctx, scope, []Identity{fact.Identity})
	s.Require().NoError(err)

	active, _, err := concrete.Read(ctx, scope)
	s.Require().NoError(err)
	s.Empty(active)

	all, _, err := concrete.ReadAll(ctx, scope)
	s.Require().NoError(err)
	s.Len(all, 1)
	s.Equal(FactStateArchived, all[0].State)
}

func (s *LayerInternalSuite) TestMergeFactsDelegatesToCatalogDetectCollision() {
	filesystem := fs.NewFakeFileSystem()
	concrete := NewLayerWithLocker(filesystem, noopLayerLocker{}).(*layer)

	scenarios := []struct {
		name               string
		existing           Fact
		candidate          Fact
		wantCollision      CollisionResult
		wantMergedLen      int
		wantCandidateState FactState
	}{
		{
			name:               "should keep single fact unchanged on idempotent collision",
			existing:           Fact{Identity: Identity{Key: "payment.retry", Hash: "sha256:aaaa"}, Durability: DurabilityPRD, State: FactStateActive},
			candidate:          Fact{Identity: Identity{Key: "payment.retry", Hash: "sha256:aaaa"}, Durability: DurabilityPRD},
			wantCollision:      CollisionIdempotent,
			wantMergedLen:      1,
			wantCandidateState: FactStateActive,
		},
		{
			name:               "should contradict existing fact on hash divergence for same key",
			existing:           Fact{Identity: Identity{Key: "payment.retry", Hash: "sha256:aaaa"}, Durability: DurabilityPRD, State: FactStateActive},
			candidate:          Fact{Identity: Identity{Key: "payment.retry", Hash: "sha256:bbbb"}, Durability: DurabilityPRD},
			wantCollision:      CollisionContradictory,
			wantMergedLen:      2,
			wantCandidateState: FactStateActive,
		},
		{
			name:               "should add unrelated fact without collision for different key",
			existing:           Fact{Identity: Identity{Key: "payment.retry", Hash: "sha256:aaaa"}, Durability: DurabilityPRD, State: FactStateActive},
			candidate:          Fact{Identity: Identity{Key: "payment.timeout", Hash: "sha256:cccc"}, Durability: DurabilityPRD},
			wantCollision:      CollisionNone,
			wantMergedLen:      2,
			wantCandidateState: FactStateActive,
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			catalog := NewCatalog()
			s.Equal(sc.wantCollision, catalog.DetectCollision(sc.existing, sc.candidate))

			merged, _ := concrete.mergeFacts([]Fact{sc.existing}, []Fact{sc.candidate})

			s.Len(merged, sc.wantMergedLen)
			last := merged[len(merged)-1]
			s.Equal(sc.candidate.Identity, last.Identity)
			s.Equal(sc.wantCandidateState, last.State)
		})
	}
}
