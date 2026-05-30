package changelog

import (
	"fmt"
	"os"
	"strings"

	"github.com/JailtonJunior94/ai-spec-harness/internal/semver"
)

// Generator gera secoes de changelog.
type Generator struct{}

// NewGenerator cria um Generator stateless.
func NewGenerator() *Generator {
	return &Generator{}
}

// ChangelogEntry represents a single entry in the changelog.
type ChangelogEntry struct {
	Type        string
	Scope       string
	Description string
	Hash        string
	Breaking    bool
}

var _typeLabels = map[string]string{
	"feat":     "Features",
	"fix":      "Bug Fixes",
	"perf":     "Performance Improvements",
	"refactor": "Refactoring",
	"docs":     "Documentation",
	"chore":    "Chores",
	"test":     "Tests",
	"ci":       "CI",
	"build":    "Build",
	"style":    "Style",
}

// _typeOrder defines the display order for changelog sections.
var _typeOrder = []string{"feat", "fix", "perf", "refactor", "docs", "chore", "test", "ci", "build", "style"}

// GroupByType groups commits into ChangelogEntry slices keyed by commit type.
func (g *Generator) GroupByType(commits []semver.Commit) map[string][]ChangelogEntry {
	groups := make(map[string][]ChangelogEntry)
	for _, c := range commits {
		if c.Type == "" {
			continue
		}
		// Parse scope from raw subject: feat(scope): desc
		scope := ""
		raw := c.Raw
		idx := strings.Index(raw, ":")
		if idx >= 0 {
			prefix := raw[:idx]
			if paren := strings.Index(prefix, "("); paren >= 0 {
				end := strings.Index(prefix, ")")
				if end > paren {
					scope = prefix[paren+1 : end]
				}
			}
		}
		entry := ChangelogEntry{
			Type:        c.Type,
			Scope:       scope,
			Description: c.Subject,
			Hash:        g.shortHash(c.Hash),
			Breaking:    c.Breaking,
		}
		groups[c.Type] = append(groups[c.Type], entry)
	}
	return groups
}

func (g *Generator) shortHash(hash string) string {
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

// RenderSection generates the Markdown for a single version section.
func (g *Generator) RenderSection(version, date string, groups map[string][]ChangelogEntry) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## %s (%s)\n", version, date)

	// Collect breaking changes across all types.
	var breaking []ChangelogEntry
	for _, entries := range groups {
		for _, e := range entries {
			if e.Breaking {
				breaking = append(breaking, e)
			}
		}
	}

	// Write sections in order.
	for _, t := range _typeOrder {
		entries, ok := groups[t]
		if !ok || len(entries) == 0 {
			continue
		}
		label, ok := _typeLabels[t]
		if !ok {
			label = strings.ToUpper(t[:1]) + t[1:]
		}
		fmt.Fprintf(&sb, "\n### %s\n", label)
		for _, e := range entries {
			if e.Breaking {
				continue // printed separately
			}
			sb.WriteString(g.renderLine(e))
		}
	}

	if len(breaking) > 0 {
		sb.WriteString("\n### Breaking Changes\n")
		for _, e := range breaking {
			sb.WriteString(g.renderLine(e))
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

func (g *Generator) renderLine(e ChangelogEntry) string {
	if e.Scope != "" {
		return fmt.Sprintf("- **%s:** %s (%s)\n", e.Scope, e.Description, e.Hash)
	}
	return fmt.Sprintf("- %s (%s)\n", e.Description, e.Hash)
}

const _changelogHeader = "# Changelog\n"

// UpdateChangelog inserts newSection into filePath after the `# Changelog` header.
// If the file does not exist it is created from scratch.
func (g *Generator) UpdateChangelog(filePath, newSection string) error {
	existing := ""
	data, err := os.ReadFile(filePath)
	if err == nil {
		existing = string(data)
	}

	var result string
	if existing == "" {
		result = _changelogHeader + "\n" + newSection
	} else if strings.HasPrefix(existing, _changelogHeader) {
		rest := existing[len(_changelogHeader):]
		result = _changelogHeader + "\n" + newSection + strings.TrimLeft(rest, "\n")
	} else {
		// File exists but has no standard header — prepend everything.
		result = _changelogHeader + "\n" + newSection + existing
	}

	return os.WriteFile(filePath, []byte(result), 0644)
}

// GenerateChangelog orchestrates fetching commits, grouping, rendering and writing.
func (g *Generator) GenerateChangelog(repoPath, version, date, filePath string) (string, error) {
	semverService := semver.NewService()
	d, err := semverService.Evaluate(repoPath)
	if err != nil {
		return "", fmt.Errorf("evaluating semver: %w", err)
	}

	commits, err := semverService.ParseConventionalCommits(repoPath, d.CommitRange)
	if err != nil {
		return "", fmt.Errorf("parsing commits: %w", err)
	}

	groups := g.GroupByType(commits)
	section := g.RenderSection(version, date, groups)
	return section, nil
}
