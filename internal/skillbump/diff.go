package skillbump

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/semver"
)

// ChangedSkills retorna as skills alteradas entre duas refs.
func ChangedSkills(repoPath, fromRef, toRef, skillsDir string) ([]string, error) {
	cmd := exec.Command("git", "-C", filepath.Clean(repoPath), "diff", "--name-only", fromRef, toRef, "--", filepath.Clean(skillsDir))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("falha ao comparar skills: %w", err)
	}

	seen := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}

		name, ok := extractSkillName(line, skillsDir)
		if !ok {
			continue
		}
		seen[name] = struct{}{}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)

	return names, nil
}

// CommitsForSkill retorna commits que tocaram uma skill.
func CommitsForSkill(repoPath, fromRef, toRef, skillsDir, skillName string) ([]semver.Commit, error) {
	skillPath := filepath.Clean(filepath.Join(skillsDir, skillName))
	cmd := exec.Command("git", "-C", filepath.Clean(repoPath), "log", "--format=%H %s", fromRef+".."+toRef, "--", skillPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("falha ao listar commits da skill %s: %w", skillName, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	commits := make([]semver.Commit, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}

		commit, err := parseCommit(repoPath, parts[0], parts[1])
		if err != nil {
			return nil, fmt.Errorf("falha ao carregar commit %s da skill %s: %w", parts[0], skillName, err)
		}
		commits = append(commits, commit)
	}

	return commits, nil
}

func readSkillFileAtRef(repoPath, ref, skillsDir, skillName string) ([]byte, error) {
	skillFile := filepath.ToSlash(filepath.Clean(filepath.Join(skillsDir, skillName, "SKILL.md")))
	cmd := exec.Command("git", "-C", filepath.Clean(repoPath), "show", ref+":"+skillFile)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("falha ao ler skill %s na ref %s: %w", skillName, ref, err)
	}

	return out, nil
}

func parseCommit(repoPath, hash, subject string) (semver.Commit, error) {
	commit := semver.Commit{
		Hash: hash,
		Raw:  subject,
	}

	if strings.Contains(subject, "BREAKING CHANGE") {
		commit.Breaking = true
	}

	idx := strings.Index(subject, ":")
	if idx < 0 {
		return commit, nil
	}

	prefix := subject[:idx]
	commit.Subject = strings.TrimSpace(subject[idx+1:])
	if strings.HasSuffix(prefix, "!") {
		commit.Breaking = true
		prefix = strings.TrimSuffix(prefix, "!")
	}

	if parenIdx := strings.Index(prefix, "("); parenIdx >= 0 {
		prefix = prefix[:parenIdx]
	}
	commit.Type = strings.TrimSpace(prefix)

	body, err := readCommitBody(repoPath, hash)
	if err != nil {
		return semver.Commit{}, err
	}
	if strings.Contains(body, "BREAKING CHANGE") {
		commit.Breaking = true
	}

	return commit, nil
}

func readCommitBody(repoPath, hash string) (string, error) {
	cmd := exec.Command("git", "-C", filepath.Clean(repoPath), "show", "--format=%b", "-s", hash)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("falha ao ler corpo do commit %s: %w", hash, err)
	}
	return string(out), nil
}

func extractSkillName(path, skillsDir string) (string, bool) {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	cleanSkillsDir := strings.TrimSuffix(filepath.ToSlash(filepath.Clean(skillsDir)), "/")
	prefix := cleanSkillsDir + "/"
	if !strings.HasPrefix(cleanPath, prefix) {
		return "", false
	}

	relative := strings.TrimPrefix(cleanPath, prefix)
	parts := strings.Split(relative, "/")
	if len(parts) < 2 {
		return "", false
	}

	name := parts[0]
	switch {
	case len(parts) == 2 && parts[1] == "SKILL.md":
		return name, true
	case len(parts) >= 3 && (parts[1] == "references" || parts[1] == "assets"):
		return name, true
	default:
		return "", false
	}
}
