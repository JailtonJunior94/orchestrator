package hookpolicy_test

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/harness"
	"github.com/JailtonJunior94/ai-spec-harness/internal/hookpolicy"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/stretchr/testify/require"
)

const fixtureContract = `version: 1
git:
  auto_commit: false
  auto_push: false
approval:
  require_for_destructive_operations: true
quality:
  require_tests: true
  require_lint: true
evidence:
  require_execution_report: true
skills:
  discovery_mode: declared
`

func newArtifactService(filesystem fs.FileSystem) *hookpolicy.ArtifactService {
	loader := harness.NewDefaultLoader(filesystem)
	printer := &output.Printer{Out: io.Discard, Err: io.Discard}
	return hookpolicy.NewArtifactService(filesystem, loader, printer)
}

func TestArtifactService_GenerateWritesDeterministicJSON(t *testing.T) {
	filesystem := fs.NewFakeFileSystem()
	filesystem.Files["repo/.agents/harness.yaml"] = []byte(fixtureContract)
	service := newArtifactService(filesystem)

	artifact, err := service.Generate("repo")
	require.NoError(t, err)
	require.NotEmpty(t, artifact.Fingerprint)
	require.NotEmpty(t, artifact.Operations)

	written, ok := filesystem.Files["repo/.agents/generated/git-scope.json"]
	require.True(t, ok, "expected git-scope.json to be written")

	var decoded hookpolicy.GitScopeArtifact
	require.NoError(t, json.Unmarshal(written, &decoded))
	require.Equal(t, artifact.Fingerprint, decoded.Fingerprint)
}

func TestArtifactService_GenerateFallsBackToEmbeddedDefault(t *testing.T) {
	filesystem := fs.NewFakeFileSystem()
	service := newArtifactService(filesystem)

	artifact, err := service.Generate("repo")
	require.NoError(t, err)
	require.NotEmpty(t, artifact.Operations)
}

func TestArtifactService_CheckPassesAfterGenerate(t *testing.T) {
	filesystem := fs.NewFakeFileSystem()
	filesystem.Files["repo/.agents/harness.yaml"] = []byte(fixtureContract)
	service := newArtifactService(filesystem)

	_, err := service.Generate("repo")
	require.NoError(t, err)

	require.NoError(t, service.Check("repo"))
}

func TestArtifactService_CheckFailsWhenArtifactMissing(t *testing.T) {
	filesystem := fs.NewFakeFileSystem()
	filesystem.Files["repo/.agents/harness.yaml"] = []byte(fixtureContract)
	service := newArtifactService(filesystem)

	err := service.Check("repo")
	require.Error(t, err)
}

func TestArtifactService_CheckFailsWhenArtifactHandEdited(t *testing.T) {
	filesystem := fs.NewFakeFileSystem()
	filesystem.Files["repo/.agents/harness.yaml"] = []byte(fixtureContract)
	service := newArtifactService(filesystem)

	_, err := service.Generate("repo")
	require.NoError(t, err)

	filesystem.Files["repo/.agents/generated/git-scope.json"] = []byte(`{"version":1,"fingerprint":"tampered","operations":[]}`)

	err = service.Check("repo")
	require.Error(t, err)
	require.Contains(t, err.Error(), "diverge do contrato")
}

func TestArtifactService_CheckDetectsPolicyDrift(t *testing.T) {
	filesystem := fs.NewFakeFileSystem()
	filesystem.Files["repo/.agents/harness.yaml"] = []byte(fixtureContract)
	service := newArtifactService(filesystem)

	_, err := service.Generate("repo")
	require.NoError(t, err)

	filesystem.Files["repo/.agents/harness.yaml"] = []byte(`version: 1
git:
  auto_commit: true
  auto_push: false
approval:
  require_for_destructive_operations: true
quality:
  require_tests: true
  require_lint: true
evidence:
  require_execution_report: true
skills:
  discovery_mode: declared
`)

	err = service.Check("repo")
	require.Error(t, err)
}
