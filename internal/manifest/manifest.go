package manifest

import (
	"encoding/json"
	"time"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

const ManifestFile = ".ai_spec_harness.json"

// Manifest persiste metadados da instalacao para upgrades futuros.
type Manifest struct {
	Version   string            `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	SourceDir string            `json:"source_dir"`
	LinkMode  skills.LinkMode   `json:"link_mode"`
	Tools     []skills.Tool     `json:"tools"`
	Langs     []skills.Lang     `json:"langs"`
	Skills    []string          `json:"skills"`
	Checksums map[string]string `json:"checksums"`
}

// Store gerencia leitura e escrita do manifesto.
type Store struct {
	fs fs.FileSystem
}

func NewStore(fsys fs.FileSystem) *Store {
	return &Store{fs: fsys}
}

func (s *Store) Load(projectDir string) (*Manifest, error) {
	path := projectDir + "/" + ManifestFile
	data, err := s.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Store) Save(projectDir string, m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return s.fs.WriteFile(projectDir+"/"+ManifestFile, data)
}

func (s *Store) Exists(projectDir string) bool {
	return s.fs.Exists(projectDir + "/" + ManifestFile)
}
