package changelog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/semver"
)

func makeCommits() []semver.Commit {
	return []semver.Commit{
		{Hash: "abc1234567", Type: "feat", Subject: "add --ref flag for git ref resolution", Raw: "feat(install): add --ref flag for git ref resolution"},
		{Hash: "def5678901", Type: "fix", Subject: "fix monorepo detection for pnpm workspaces", Raw: "fix(detect): fix monorepo detection for pnpm workspaces"},
		{Hash: "ghi9012345", Type: "feat", Subject: "remove deprecated --legacy flag", Breaking: true, Raw: "feat(config)!: remove deprecated --legacy flag"},
		{Hash: "jkl3456789", Type: "chore", Subject: "update dependencies", Raw: "chore: update dependencies"},
	}
}

func TestGroupByType(t *testing.T) {
	commits := makeCommits()
	groups := GroupByType(commits)

	if len(groups["feat"]) != 2 {
		t.Fatalf("expected 2 feat entries, got %d", len(groups["feat"]))
	}
	if len(groups["fix"]) != 1 {
		t.Fatalf("expected 1 fix entry, got %d", len(groups["fix"]))
	}
	if len(groups["chore"]) != 1 {
		t.Fatalf("expected 1 chore entry, got %d", len(groups["chore"]))
	}

	// Breaking flag propagated
	found := false
	for _, e := range groups["feat"] {
		if e.Breaking {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one breaking feat entry")
	}
}

func TestGroupByType_SkipsNoType(t *testing.T) {
	commits := []semver.Commit{
		{Hash: "abc1234567", Type: "", Subject: "some random message", Raw: "some random message"},
	}
	groups := GroupByType(commits)
	if len(groups) != 0 {
		t.Errorf("expected empty groups, got %v", groups)
	}
}

func TestGroupByType_ScopeExtracted(t *testing.T) {
	commits := []semver.Commit{
		{Hash: "abc1234567", Type: "feat", Subject: "add feature", Raw: "feat(myScope): add feature"},
	}
	groups := GroupByType(commits)
	if groups["feat"][0].Scope != "myScope" {
		t.Errorf("expected scope 'myScope', got '%s'", groups["feat"][0].Scope)
	}
}

func TestRenderSection(t *testing.T) {
	groups := GroupByType(makeCommits())
	section := RenderSection("1.3.0", "2026-04-20", groups)

	checks := []string{
		"## 1.3.0 (2026-04-20)",
		"### Features",
		"### Bug Fixes",
		"### Breaking Changes",
		"**install:**",
		"**detect:**",
		"abc1234",
		"def5678",
	}
	for _, want := range checks {
		if !strings.Contains(section, want) {
			t.Errorf("expected section to contain %q\nGot:\n%s", want, section)
		}
	}
}

func TestRenderSection_NoBreaking(t *testing.T) {
	commits := []semver.Commit{
		{Hash: "abc1234567", Type: "feat", Subject: "simple feature", Raw: "feat: simple feature"},
	}
	groups := GroupByType(commits)
	section := RenderSection("1.1.0", "2026-01-01", groups)

	if strings.Contains(section, "Breaking Changes") {
		t.Error("section should not contain Breaking Changes when there are none")
	}
}

func TestUpdateChangelog_NewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")

	section := "## 1.0.0 (2026-01-01)\n\n### Features\n- init (abc1234)\n\n"
	if err := UpdateChangelog(path, section); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.HasPrefix(content, "# Changelog\n") {
		t.Error("new file should start with '# Changelog'")
	}
	if !strings.Contains(content, section) {
		t.Error("new file should contain the new section")
	}
}

func TestUpdateChangelog_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CHANGELOG.md")

	existing := "# Changelog\n\n## 1.0.0 (2026-01-01)\n\n### Features\n- old feature (old1234)\n\n"
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	newSection := "## 1.1.0 (2026-04-20)\n\n### Features\n- new feature (new5678)\n\n"
	if err := UpdateChangelog(path, newSection); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	// New section must appear before old section.
	newIdx := strings.Index(content, "## 1.1.0")
	oldIdx := strings.Index(content, "## 1.0.0")
	if newIdx < 0 || oldIdx < 0 {
		t.Fatalf("both sections should be present\n%s", content)
	}
	if newIdx > oldIdx {
		t.Error("new section should appear before old section")
	}

	// Header must appear exactly once.
	if strings.Count(content, "# Changelog") != 1 {
		t.Error("header should appear exactly once")
	}
}
