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
	Version       string            `json:"version"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	SourceDir     string            `json:"source_dir"`
	LinkMode      skills.LinkMode   `json:"link_mode"`
	Tools         []skills.Tool     `json:"tools"`
	Langs         []skills.Lang     `json:"langs"`
	Skills        []string          `json:"skills"`
	Checksums     map[string]string `json:"checksums"`
	CodexProfile  string            `json:"codex_profile,omitempty"`
	SkillVersions map[string]string `json:"skill_versions,omitempty"`

	// InstalledFiles rastreia, individualmente, os caminhos (relativos a
	// SourceDir/ProjectDir do projeto) que esta instalacao criou. Campo
	// aditivo (RF-05, RF-60): manifestos antigos no disco nao o possuem, e
	// sua ausencia (nil, distinto de slice vazio) sinaliza a desinstalacao a
	// cair no caminho conservador anunciado em vez de assumir uma lista fixa.
	InstalledFiles []string `json:"installed_files,omitempty"`
}

// HasFileTracking reporta se o manifesto rastreia arquivos individualmente
// (campo aditivo presente). Manifestos gravados antes desta tarefa nao o tem.
func (m *Manifest) HasFileTracking() bool {
	return m.InstalledFiles != nil
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
