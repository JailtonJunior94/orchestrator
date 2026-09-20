package hookpolicy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/harness"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

const GitScopeArtifactRelativePath = ".agents/generated/git-scope.json"

type GitOperationArtifact struct {
	Subcommand       string `json:"subcommand"`
	Destructive      bool   `json:"destructive"`
	RequiresApproval bool   `json:"requires_approval"`
}

type GitScopeArtifact struct {
	Version     int                    `json:"version"`
	Fingerprint string                 `json:"fingerprint"`
	Operations  []GitOperationArtifact `json:"operations"`
}

func BuildGitScopeArtifact(scope GitScope) GitScopeArtifact {
	operations := scope.Operations()
	items := make([]GitOperationArtifact, 0, len(operations))
	for _, operation := range operations {
		items = append(items, GitOperationArtifact{
			Subcommand:       operation.Subcommand(),
			Destructive:      operation.Destructive(),
			RequiresApproval: operation.RequiresApproval(),
		})
	}
	return GitScopeArtifact{
		Version:     harness.SupportedVersion,
		Fingerprint: scope.Fingerprint(),
		Operations:  items,
	}
}

func encodeGitScopeArtifact(artifact GitScopeArtifact) ([]byte, error) {
	encoded, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("hookpolicy: encode git scope artifact: %w", err)
	}
	return append(encoded, '\n'), nil
}

type ArtifactService struct {
	fs      fs.FileSystem
	loader  harness.Loader
	printer *output.Printer
}

func NewArtifactService(filesystem fs.FileSystem, loader harness.Loader, printer *output.Printer) *ArtifactService {
	return &ArtifactService{fs: filesystem, loader: loader, printer: printer}
}

func (s *ArtifactService) buildArtifact(root string) (GitScopeArtifact, error) {
	contract, _, err := s.loader.Load(root)
	if err != nil {
		return GitScopeArtifact{}, fmt.Errorf("hookpolicy: load harness contract: %w", err)
	}
	scope, err := NewGitScope(contract)
	if err != nil {
		return GitScopeArtifact{}, err
	}
	return BuildGitScopeArtifact(scope), nil
}

func (s *ArtifactService) Generate(root string) (GitScopeArtifact, error) {
	artifact, err := s.buildArtifact(root)
	if err != nil {
		return GitScopeArtifact{}, err
	}

	encoded, err := encodeGitScopeArtifact(artifact)
	if err != nil {
		return GitScopeArtifact{}, err
	}

	path := filepath.Join(root, filepath.FromSlash(GitScopeArtifactRelativePath))
	if err := s.fs.WriteFileAtomic(path, encoded); err != nil {
		return GitScopeArtifact{}, fmt.Errorf("hookpolicy: write git scope artifact %s: %w", path, err)
	}

	if s.printer != nil {
		s.printer.Info("Escopo Git gerado: %d operacoes, fingerprint %s", len(artifact.Operations), artifact.Fingerprint)
	}
	return artifact, nil
}

func (s *ArtifactService) Check(root string) error {
	artifact, err := s.buildArtifact(root)
	if err != nil {
		return err
	}

	wantEncoded, err := encodeGitScopeArtifact(artifact)
	if err != nil {
		return err
	}

	path := filepath.Join(root, filepath.FromSlash(GitScopeArtifactRelativePath))
	got, err := s.fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("hookpolicy: read git scope artifact %s: %w (rode 'ai-spec hooks git-scope %s' para gerar)", path, err, root)
	}

	if !bytes.Equal(got, wantEncoded) {
		return fmt.Errorf("hookpolicy: git scope artifact %s diverge do contrato do harness; rode 'ai-spec hooks git-scope %s' para regenerar", path, root)
	}
	return nil
}
