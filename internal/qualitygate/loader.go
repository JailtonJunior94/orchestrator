package qualitygate

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

const PolicyArtifactRelativePath = ".agents/quality-gate-policy.yaml"

//go:embed default-policy.yaml
var defaultPolicyYAML []byte

type PolicyLoader interface {
	Load(projectDir string) (PolicySet, error)
}

type DefaultPolicyLoader struct {
	fs fs.FileSystem
}

var _ PolicyLoader = (*DefaultPolicyLoader)(nil)

func NewDefaultPolicyLoader(filesystem fs.FileSystem) *DefaultPolicyLoader {
	return &DefaultPolicyLoader{fs: filesystem}
}

func (l *DefaultPolicyLoader) Load(projectDir string) (PolicySet, error) {
	path := filepath.Join(projectDir, filepath.FromSlash(PolicyArtifactRelativePath))

	data, err := l.fs.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DecodePolicySet(defaultPolicyYAML)
		}
		return PolicySet{}, fmt.Errorf("qualitygate: read policy artifact %s: %w", path, err)
	}

	return DecodePolicySet(data)
}

func DefaultPolicySet() (PolicySet, error) {
	return DecodePolicySet(defaultPolicyYAML)
}
