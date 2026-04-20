// Package skillscheck verifica o estado de versao de skills externas no lock file.
// Detecta divergencias de versao entre skills-lock.json e o SKILL.md instalado,
// classificando upgrades como compativeis ou potencialmente quebra de interface.
package skillscheck

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

// LockEntry representa uma entrada no skills-lock.json.
type LockEntry struct {
	Source       string `json:"source"`
	SourceType   string `json:"sourceType"`
	Version      string `json:"version,omitempty"`
	ComputedHash string `json:"computedHash"`
}

// LockFile representa a estrutura do skills-lock.json.
type LockFile struct {
	Version int                  `json:"version"`
	Skills  map[string]LockEntry `json:"skills"`
}

// VersionDrift classifica o tipo de mudanca de versao.
type VersionDrift string

const (
	DriftNone     VersionDrift = "ok"          // versao identica
	DriftMinor    VersionDrift = "minor"        // patch ou minor: compativel
	DriftBreaking VersionDrift = "breaking"     // major bump: potencialmente quebra
	DriftNoLock   VersionDrift = "no-lock"      // skill instalada sem lock entry
	DriftNoSkill  VersionDrift = "no-skill"     // lock entry sem skill instalada
	DriftUnknown  VersionDrift = "unknown"      // versao ausente em lock ou SKILL.md
)

// SkillVersionCheck armazena o resultado da verificacao de versao de uma skill.
type SkillVersionCheck struct {
	Name        string
	LockedVer   string
	InstalledVer string
	Drift       VersionDrift
	Breaking    bool
}

// Service executa verificacoes de versao de skills externas.
type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: fsys, printer: printer}
}

// Check verifica o estado de versao de todas as skills externas no projectDir.
func (s *Service) Check(projectDir string) ([]SkillVersionCheck, error) {
	lockPath := filepath.Join(projectDir, "skills-lock.json")
	lockData, err := s.fs.ReadFile(lockPath)
	if err != nil {
		return nil, fmt.Errorf("ler skills-lock.json: %w", err)
	}

	var lock LockFile
	if err := json.Unmarshal(lockData, &lock); err != nil {
		return nil, fmt.Errorf("parsear skills-lock.json: %w", err)
	}

	skillsDir := filepath.Join(projectDir, ".agents", "skills")
	var results []SkillVersionCheck

	for skillName, entry := range lock.Skills {
		skillMDPath := filepath.Join(skillsDir, skillName, "SKILL.md")
		skillData, err := s.fs.ReadFile(skillMDPath)
		if err != nil {
			results = append(results, SkillVersionCheck{
				Name:      skillName,
				LockedVer: entry.Version,
				Drift:     DriftNoSkill,
			})
			continue
		}

		fm := skills.ParseFrontmatter(skillData)
		installedVer := fm.Version

		drift := classifyDrift(entry.Version, installedVer)
		results = append(results, SkillVersionCheck{
			Name:         skillName,
			LockedVer:    entry.Version,
			InstalledVer: installedVer,
			Drift:        drift,
			Breaking:     drift == DriftBreaking,
		})
	}

	return results, nil
}

// classifyDrift classifica a mudanca entre versao do lock e versao instalada.
func classifyDrift(locked, installed string) VersionDrift {
	if locked == "" || installed == "" {
		return DriftUnknown
	}
	if locked == installed {
		return DriftNone
	}
	lockedMajor := parseMajor(locked)
	installedMajor := parseMajor(installed)
	if lockedMajor < 0 || installedMajor < 0 {
		return DriftUnknown
	}
	if installedMajor > lockedMajor {
		return DriftBreaking
	}
	return DriftMinor
}

// parseMajor extrai o numero de versao major de uma string semver (ex: "1.2.3" -> 1).
// Retorna -1 se o formato for invalido.
func parseMajor(v string) int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 {
		return -1
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}
	return n
}
