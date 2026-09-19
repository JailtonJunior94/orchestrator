package harness

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

type LoaderSuite struct {
	suite.Suite
}

func TestLoaderSuite(t *testing.T) {
	suite.Run(t, new(LoaderSuite))
}

func (s *LoaderSuite) newFakeFS(projectDir, fixtureName string) fs.FileSystem {
	fake := fs.NewFakeFileSystem()
	if fixtureName != "" {
		data := readFixtureFile(s.T(), fixtureName)
		fake.Files[filepath.Join(projectDir, ".agents", "harness.yaml")] = data
	}
	return fake
}

func (s *LoaderSuite) TestLoadAppliesEmbeddedDefaultWhenProjectHasNoContractFile() {
	loader := NewDefaultLoader(s.newFakeFS("/project", ""))

	contract, source, err := loader.Load("/project")

	s.Require().NoError(err)
	s.Equal(SourceDefault, source)

	defaultContract, defaultErr := NewEmbeddedDefault().Contract()
	s.Require().NoError(defaultErr)
	s.Equal(defaultContract, contract)
}

func (s *LoaderSuite) TestLoadReadsProjectDeclaredContract() {
	loader := NewDefaultLoader(s.newFakeFS("/project", "valid.yaml"))

	contract, source, err := loader.Load("/project")

	s.Require().NoError(err)
	s.Equal(SourceFile, source)
	s.Equal(SupportedVersion, contract.Version)
}

func (s *LoaderSuite) TestLoadRejectsProjectDeclaredContractWithUnknownField() {
	loader := NewDefaultLoader(s.newFakeFS("/project", "unknown-field.yaml"))

	_, _, err := loader.Load("/project")

	s.Require().Error(err)
	var typed *UnknownFieldError
	s.Require().True(errors.As(err, &typed))
}

func (s *LoaderSuite) TestLoadRejectsIncompatibleVersion() {
	loader := NewDefaultLoader(s.newFakeFS("/project", "version-2.yaml"))

	_, _, err := loader.Load("/project")

	s.Require().Error(err)
	var typed *UnsupportedVersionError
	s.Require().True(errors.As(err, &typed))
}
