package durable_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type FactSuite struct {
	suite.Suite
}

func TestFactSuite(t *testing.T) {
	suite.Run(t, new(FactSuite))
}

func (s *FactSuite) TestValidateFact() {
	scenarios := []struct {
		name    string
		fact    durable.Fact
		wantErr error
	}{
		{
			name: "should reject fact without semantic key",
			fact: durable.Fact{
				Identity:   durable.Identity{Key: ""},
				Durability: durable.DurabilityDurable,
			},
			wantErr: durable.ErrSemanticKeyMissing,
		},
		{
			name: "should reject fact with zero-value durability",
			fact: durable.Fact{
				Identity:   durable.Identity{Key: "topic.subject"},
				Durability: durable.DurabilityInvalid,
			},
			wantErr: durable.ErrDurabilityMissing,
		},
		{
			name: "should accept fact with key and durability declared",
			fact: durable.Fact{
				Identity:   durable.Identity{Key: "topic.subject"},
				Durability: durable.DurabilityEphemeral,
			},
			wantErr: nil,
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			catalog := durable.NewCatalog()

			err := catalog.ValidateFact(sc.fact)

			if sc.wantErr == nil {
				s.NoError(err)
				return
			}
			s.True(errors.Is(err, sc.wantErr))
		})
	}
}

func (s *FactSuite) TestResolveLayer() {
	scenarios := []struct {
		name       string
		durability durable.Durability
		wantLayer  durable.TargetLayer
		wantErr    error
	}{
		{
			name:       "should resolve ephemeral to task layer",
			durability: durable.DurabilityEphemeral,
			wantLayer:  durable.TargetLayerTask,
		},
		{
			name:       "should resolve prd durability to prd layer",
			durability: durable.DurabilityPRD,
			wantLayer:  durable.TargetLayerPRD,
		},
		{
			name:       "should resolve durable to project layer",
			durability: durable.DurabilityDurable,
			wantLayer:  durable.TargetLayerProject,
		},
		{
			name:       "should reject zero-value durability",
			durability: durable.DurabilityInvalid,
			wantErr:    durable.ErrDurabilityMissing,
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			catalog := durable.NewCatalog()

			layer, err := catalog.ResolveLayer(sc.durability)

			if sc.wantErr != nil {
				s.True(errors.Is(err, sc.wantErr))
				return
			}
			s.NoError(err)
			s.Equal(sc.wantLayer, layer)
		})
	}
}

func (s *FactSuite) TestHashContentIsDeterministic() {
	catalog := durable.NewCatalog()

	first := catalog.HashContent("payment retried after timeout")
	second := catalog.HashContent("payment retried after timeout")

	s.Equal(first, second)
	s.NotEmpty(first)
}

func (s *FactSuite) TestHashContentDiffersForDifferentContent() {
	catalog := durable.NewCatalog()

	first := catalog.HashContent("payment retried after timeout")
	second := catalog.HashContent("payment retried after five attempts")

	s.NotEqual(first, second)
}

func (s *FactSuite) TestDetectCollision() {
	base := durable.Fact{
		Identity: durable.Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
	}

	scenarios := []struct {
		name      string
		candidate durable.Fact
		want      durable.CollisionResult
	}{
		{
			name: "should report no collision for different key",
			candidate: durable.Fact{
				Identity: durable.Identity{Key: "payment.timeout", Hash: "sha256:aaaa"},
			},
			want: durable.CollisionNone,
		},
		{
			name: "should report idempotent collision for same key and hash",
			candidate: durable.Fact{
				Identity: durable.Identity{Key: "payment.retry", Hash: "sha256:aaaa"},
			},
			want: durable.CollisionIdempotent,
		},
		{
			name: "should report contradictory collision for same key and divergent hash",
			candidate: durable.Fact{
				Identity: durable.Identity{Key: "payment.retry", Hash: "sha256:bbbb"},
			},
			want: durable.CollisionContradictory,
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			catalog := durable.NewCatalog()

			got := catalog.DetectCollision(base, sc.candidate)

			s.Equal(sc.want, got)
		})
	}
}

func (s *FactSuite) TestDeriveSemanticKeyIsDeterministicAcrossSessions() {
	catalog := durable.NewCatalog()
	signal := durable.StructuredSignal{
		Kind:    "Decision",
		Subject: "Retry Policy",
		Scope:   "Payments",
	}

	firstSession := durable.NewCatalog()
	secondSession := durable.NewCatalog()

	first, err := firstSession.DeriveSemanticKey(signal)
	s.Require().NoError(err)

	second, err := secondSession.DeriveSemanticKey(signal)
	s.Require().NoError(err)

	s.Equal(first, second)
	s.Equal(catalog.HashContent("x"), catalog.HashContent("x"))
}

func (s *FactSuite) TestDeriveSemanticKeyRejectsMissingFields() {
	scenarios := []struct {
		name   string
		signal durable.StructuredSignal
	}{
		{
			name:   "should reject missing kind",
			signal: durable.StructuredSignal{Subject: "retry policy"},
		},
		{
			name:   "should reject missing subject",
			signal: durable.StructuredSignal{Kind: "decision"},
		},
	}

	for _, sc := range scenarios {
		s.Run(sc.name, func() {
			catalog := durable.NewCatalog()

			_, err := catalog.DeriveSemanticKey(sc.signal)

			s.True(errors.Is(err, durable.ErrSemanticKeyMissing))
		})
	}
}
