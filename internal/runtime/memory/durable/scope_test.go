package durable_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/memory/durable"
)

type ScopeSuite struct {
	suite.Suite
}

func TestScopeSuite(t *testing.T) {
	suite.Run(t, new(ScopeSuite))
}

func (s *ScopeSuite) TestActivePathResolvesForPRDLayer() {
	scope := durable.Scope{Layer: durable.TargetLayerPRD, TasksDir: "/project/.specs/prd-x"}

	path, err := scope.ActivePath()

	s.Require().NoError(err)
	s.Equal(filepath.Join(scope.TasksDir, "memory", "MEMORY.md"), path)
}

func (s *ScopeSuite) TestActivePathRejectsMissingTasksDirForPRDLayer() {
	scope := durable.Scope{Layer: durable.TargetLayerPRD}

	_, err := scope.ActivePath()

	s.True(errors.Is(err, durable.ErrTasksDirMissing))
}
