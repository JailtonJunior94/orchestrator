package skillbump

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/semver"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

var (
	ErrNoTagFound = errors.New("nenhuma tag v* encontrada")
	ErrNoChanges  = errors.New("nenhuma skill com mudanca detectada")
)

// Service orquestra o bump de versao das skills alteradas.
type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

// BumpResult descreve o bump aplicado ou planejado para uma skill.
type BumpResult struct {
	SkillName       string          `json:"skill_name"`
	PreviousVersion string          `json:"previous_version"`
	NewVersion      string          `json:"new_version"`
	BumpKind        semver.BumpKind `json:"bump_kind"`
	Reason          string          `json:"reason"`
}

func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: fsys, printer: printer}
}

// Execute compara as skills alteradas desde a ultima tag e aplica bump de versao.
func (s *Service) Execute(repoPath, skillsDir string, dryRun bool) ([]BumpResult, error) {
	repoRoot, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("resolver caminho do repositorio: %w", err)
	}

	lastTag, err := findLastTag(repoRoot)
	if err != nil {
		if errors.Is(err, ErrNoTagFound) {
			return nil, fmt.Errorf("%w: execute skill-bump apos a primeira release", ErrNoTagFound)
		}
		return nil, err
	}
	if lastTag == "" {
		return nil, fmt.Errorf("%w: execute skill-bump apos a primeira release", ErrNoTagFound)
	}

	changedSkills, err := ChangedSkills(repoRoot, lastTag, "HEAD", skillsDir)
	if err != nil {
		return nil, err
	}
	if len(changedSkills) == 0 {
		return nil, ErrNoChanges
	}

	results := make([]BumpResult, 0, len(changedSkills))
	var writeErrs []error

	for _, skillName := range changedSkills {
		result, changed, err := s.bumpSkill(repoRoot, lastTag, skillsDir, skillName, dryRun)
		if err != nil {
			writeErrs = append(writeErrs, err)
			continue
		}
		if changed {
			results = append(results, result)
		}
	}

	switch {
	case len(writeErrs) > 0:
		return results, errors.Join(writeErrs...)
	case len(results) == 0:
		return nil, ErrNoChanges
	default:
		return results, nil
	}
}

func (s *Service) bumpSkill(repoRoot, lastTag, skillsDir, skillName string, dryRun bool) (BumpResult, bool, error) {
	commits, err := CommitsForSkill(repoRoot, lastTag, "HEAD", skillsDir, skillName)
	if err != nil {
		return BumpResult{}, false, err
	}

	bumpKind := plannedBump(commits)
	skillPath := filepath.Join(repoRoot, skillsDir, skillName, "SKILL.md")
	content, err := s.fs.ReadFile(skillPath)
	if err != nil {
		return BumpResult{}, false, fmt.Errorf("ler frontmatter de %s: %w", skillName, err)
	}

	if err := validateFrontmatter(content); err != nil {
		s.printer.Warn("aviso: skill %s com frontmatter invalido, pulando", skillName)
		return BumpResult{}, false, nil
	}

	current := skills.ParseFrontmatter(content)
	switch {
	case current.Version == "":
		s.printer.Warn("aviso: skill %s sem campo version no frontmatter, pulando", skillName)
		return BumpResult{}, false, nil
	case !skills.IsValidSemver(current.Version):
		s.printer.Warn("aviso: skill %s com frontmatter invalido, pulando", skillName)
		return BumpResult{}, false, nil
	}

	targetVersion := semver.ComputeNext(current.Version, bumpKind)
	if previousContent, err := readSkillFileAtRef(repoRoot, lastTag, skillsDir, skillName); err == nil {
		previous := skills.ParseFrontmatter(previousContent)
		if previous.Version != "" && skills.IsValidSemver(previous.Version) {
			targetVersion = semver.ComputeNext(previous.Version, bumpKind)
		}
	}

	if current.Version == targetVersion || skills.SemverGreater(current.Version, targetVersion) {
		return BumpResult{}, false, nil
	}

	if !dryRun {
		updated, err := UpdateFrontmatterVersion(content, targetVersion)
		if err != nil {
			return BumpResult{}, false, fmt.Errorf("falha ao atualizar frontmatter de %s: %w", skillName, err)
		}
		if err := s.fs.WriteFile(skillPath, updated); err != nil {
			return BumpResult{}, false, fmt.Errorf("falha ao atualizar frontmatter de %s: %w", skillName, err)
		}
	}

	return BumpResult{
		SkillName:       skillName,
		PreviousVersion: current.Version,
		NewVersion:      targetVersion,
		BumpKind:        bumpKind,
		Reason:          describeReason(commits, bumpKind),
	}, true, nil
}

func plannedBump(commits []semver.Commit) semver.BumpKind {
	if bump := semver.DetermineBump(commits); bump != semver.BumpNone {
		return bump
	}
	return semver.BumpPatch
}

func describeReason(commits []semver.Commit, bump semver.BumpKind) string {
	for _, commit := range commits {
		if commitMatchesBump(commit, bump) {
			return commit.Raw
		}
	}
	if len(commits) > 0 {
		return commits[0].Raw
	}
	return "conteudo alterado"
}

func commitMatchesBump(commit semver.Commit, bump semver.BumpKind) bool {
	switch bump {
	case semver.BumpMajor:
		return commit.Breaking
	case semver.BumpMinor:
		return commit.Type == "feat"
	case semver.BumpPatch:
		return true
	default:
		return false
	}
}

func findLastTag(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", filepath.Clean(repoPath), "describe", "--tags", "--abbrev=0", "--match", "v*")
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if strings.Contains(message, "No names found") || strings.Contains(message, "cannot describe anything") {
			return "", ErrNoTagFound
		}
		return "", fmt.Errorf("falha ao localizar ultima tag: %w", err)
	}

	tag := strings.TrimSpace(string(out))
	if tag == "" {
		return "", ErrNoTagFound
	}

	return tag, nil
}
