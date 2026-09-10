package approval

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ApprovalProofSuite struct {
	suite.Suite
}

func TestApprovalProofSuite(t *testing.T) {
	suite.Run(t, new(ApprovalProofSuite))
}

func (s *ApprovalProofSuite) TestRejectsApprovedWithRemarks() {
	_, err := NewApprovalProof(VerdictApprovedWithRemarks, completeCriteriaMap(s.T()))
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestRejectsRejected() {
	_, err := NewApprovalProof(VerdictRejected, completeCriteriaMap(s.T()))
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestRejectsBlocked() {
	_, err := NewApprovalProof(VerdictBlocked, completeCriteriaMap(s.T()))
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestRejectsZeroValueVerdict() {
	var zero Verdict
	_, err := NewApprovalProof(zero, completeCriteriaMap(s.T()))
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestRejectsIncompleteMap() {
	_, err := NewApprovalProof(VerdictApproved, incompleteCriteriaMap(s.T()))
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestRejectsUnverifiableCriterion() {
	_, err := NewApprovalProof(VerdictApproved, unverifiableCriteriaMap(s.T()))
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestRejectsEmptyMap() {
	_, err := NewApprovalProof(VerdictApproved, CriteriaMap{})
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestApproveRejectsZeroValueProof() {
	cycle := newStubCycle(s.T())
	_, err := cycle.approve(ApprovalProof{})
	s.Require().ErrorIs(err, ErrInsufficientProof)
	s.Equal(StateInReview, cycle.State())
}

func (s *ApprovalProofSuite) TestApprovedResultRejectsZeroValueProof() {
	_, err := NewApprovedResult(ApprovalProof{}, nil)
	s.Require().ErrorIs(err, ErrInsufficientProof)
}

func (s *ApprovalProofSuite) TestValidProofOpensApproval() {
	proof, err := NewApprovalProof(VerdictApproved, completeCriteriaMap(s.T()))
	s.Require().NoError(err)
	s.True(proof.Valid())
	s.Equal(VerdictApproved, proof.Verdict())
}
