package skillbump

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
	"github.com/JailtonJunior94/ai-spec-harness/internal/semver"
)

func TestServiceExecute(t *testing.T) {
	const (
		repoPath  = "/repo"
		skillsDir = ".agents/skills"
	)

	tests := []struct {
		name          string
		current       string
		tag           string
		diffOutput    string
		logOutput     string
		bodies        map[string]string
		dryRun        bool
		wantVersion   string
		wantResult    *BumpResult
		wantErrIs     error
		wantWarning   string
		wantUnchanged bool
	}{
		{
			name:        "skill md alterado com feat gera minor",
			current:     skillFile("execute-task", "1.0.0", "Conteudo"),
			tag:         skillFile("execute-task", "1.0.0", "Conteudo"),
			diffOutput:  ".agents/skills/execute-task/SKILL.md\n",
			logOutput:   "abc123 feat(execute-task): add retry\n",
			bodies:      map[string]string{"abc123": ""},
			wantVersion: "1.1.0",
			wantResult: &BumpResult{
				SkillName:       "execute-task",
				PreviousVersion: "1.0.0",
				NewVersion:      "1.1.0",
				BumpKind:        semver.BumpMinor,
				Reason:          "feat(execute-task): add retry",
			},
		},
		{
			name:        "references alterado com fix gera patch",
			current:     skillFile("review", "1.1.0", "Conteudo"),
			tag:         skillFile("review", "1.1.0", "Conteudo"),
			diffOutput:  ".agents/skills/review/references/guide.md\n",
			logOutput:   "def456 fix(review): typo\n",
			bodies:      map[string]string{"def456": ""},
			wantVersion: "1.1.1",
			wantResult: &BumpResult{
				SkillName:       "review",
				PreviousVersion: "1.1.0",
				NewVersion:      "1.1.1",
				BumpKind:        semver.BumpPatch,
				Reason:          "fix(review): typo",
			},
		},
		{
			name:        "assets alterado com merge commit faz fallback patch",
			current:     skillFile("review", "1.1.0", "Conteudo"),
			tag:         skillFile("review", "1.1.0", "Conteudo"),
			diffOutput:  ".agents/skills/review/assets/template.md\n",
			logOutput:   "fedcba Merge branch 'feature/review-assets'\n",
			bodies:      map[string]string{"fedcba": ""},
			wantVersion: "1.1.1",
			wantResult: &BumpResult{
				SkillName:       "review",
				PreviousVersion: "1.1.0",
				NewVersion:      "1.1.1",
				BumpKind:        semver.BumpPatch,
				Reason:          "Merge branch 'feature/review-assets'",
			},
		},
		{
			name:          "sem mudanca retorna ErrNoChanges",
			current:       skillFile("review", "1.1.0", "Conteudo"),
			tag:           skillFile("review", "1.1.0", "Conteudo"),
			diffOutput:    "",
			wantErrIs:     ErrNoChanges,
			wantUnchanged: true,
		},
		{
			name:          "frontmatter malformado emite warning e pula",
			current:       "# sem frontmatter\n",
			tag:           skillFile("review", "1.1.0", "Conteudo"),
			diffOutput:    ".agents/skills/review/SKILL.md\n",
			logOutput:     "def456 fix(review): typo\n",
			bodies:        map[string]string{"def456": ""},
			wantErrIs:     ErrNoChanges,
			wantWarning:   "aviso: skill review com frontmatter invalido, pulando",
			wantUnchanged: true,
		},
		{
			name:          "yaml invalido com version tambem emite warning e pula",
			current:       malformedYAMLSkill("review", "1.1.0"),
			tag:           skillFile("review", "1.1.0", "Conteudo"),
			diffOutput:    ".agents/skills/review/SKILL.md\n",
			logOutput:     "def456 fix(review): typo\n",
			bodies:        map[string]string{"def456": ""},
			wantErrIs:     ErrNoChanges,
			wantWarning:   "aviso: skill review com frontmatter invalido, pulando",
			wantUnchanged: true,
		},
		{
			name:          "sem campo version emite warning e pula",
			current:       skillWithoutVersion("review"),
			tag:           skillFile("review", "1.1.0", "Conteudo"),
			diffOutput:    ".agents/skills/review/SKILL.md\n",
			logOutput:     "def456 fix(review): typo\n",
			bodies:        map[string]string{"def456": ""},
			wantErrIs:     ErrNoChanges,
			wantWarning:   "aviso: skill review sem campo version no frontmatter, pulando",
			wantUnchanged: true,
		},
		{
			name:        "feat e fix na mesma skill usam minor",
			current:     skillFile("execute-task", "1.0.0", "Conteudo"),
			tag:         skillFile("execute-task", "1.0.0", "Conteudo"),
			diffOutput:  ".agents/skills/execute-task/references/guide.md\n",
			logOutput:   "abc123 fix(execute-task): typo\nfed456 feat(execute-task): add retry\n",
			bodies:      map[string]string{"abc123": "", "fed456": ""},
			wantVersion: "1.1.0",
			wantResult: &BumpResult{
				SkillName:       "execute-task",
				PreviousVersion: "1.0.0",
				NewVersion:      "1.1.0",
				BumpKind:        semver.BumpMinor,
				Reason:          "feat(execute-task): add retry",
			},
		},
		{
			name:        "dry-run retorna resultado sem alterar arquivo",
			current:     skillFile("execute-task", "1.0.0", "Conteudo"),
			tag:         skillFile("execute-task", "1.0.0", "Conteudo"),
			diffOutput:  ".agents/skills/execute-task/SKILL.md\n",
			logOutput:   "abc123 feat(execute-task): add retry\n",
			bodies:      map[string]string{"abc123": ""},
			dryRun:      true,
			wantVersion: "1.0.0",
			wantResult: &BumpResult{
				SkillName:       "execute-task",
				PreviousVersion: "1.0.0",
				NewVersion:      "1.1.0",
				BumpKind:        semver.BumpMinor,
				Reason:          "feat(execute-task): add retry",
			},
		},
		{
			name:          "segunda execucao sem nova mudanca nao rebumpa",
			current:       skillFile("execute-task", "1.1.0", "Conteudo"),
			tag:           skillFile("execute-task", "1.0.0", "Conteudo"),
			diffOutput:    ".agents/skills/execute-task/SKILL.md\n",
			logOutput:     "abc123 feat(execute-task): add retry\n",
			bodies:        map[string]string{"abc123": ""},
			wantErrIs:     ErrNoChanges,
			wantUnchanged: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fsys := fs.NewFakeFileSystem()
			currentPath := filepath.Join(repoPath, skillsDir, skillNameFromContent(tc.current), "SKILL.md")
			fsys.Files[currentPath] = []byte(tc.current)
			fsys.Dirs[filepath.Join(repoPath, skillsDir, skillNameFromContent(tc.current))] = true

			var out bytes.Buffer
			var errBuf bytes.Buffer
			printer := &output.Printer{Out: &out, Err: &errBuf}
			svc := NewService(fsys, printer)

			scriptDir := writeFakeGit(t, fakeGitSpec{
				describeOutput: "v1.0.0\n",
				diffOutput:     tc.diffOutput,
				logOutputs: map[string]string{
					filepath.Clean(filepath.Join(skillsDir, skillNameFromContent(tc.current))): tc.logOutput,
				},
				showBodies: tc.bodies,
				showFiles: map[string]string{
					"v1.0.0:" + filepath.ToSlash(filepath.Clean(filepath.Join(skillsDir, skillNameFromContent(tc.current), "SKILL.md"))): tc.tag,
				},
			})
			t.Setenv("PATH", scriptDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			results, err := svc.Execute(repoPath, skillsDir, tc.dryRun)
			if !errors.Is(err, tc.wantErrIs) && err != nil {
				t.Fatalf("Execute() error = %v, wantErrIs %v", err, tc.wantErrIs)
			}
			if tc.wantErrIs == nil && err != nil {
				t.Fatalf("Execute() unexpected error: %v", err)
			}

			if tc.wantResult == nil {
				if len(results) != 0 {
					t.Fatalf("Execute() results = %v, want empty", results)
				}
			} else {
				if len(results) != 1 {
					t.Fatalf("Execute() returned %d results, want 1", len(results))
				}
				if results[0] != *tc.wantResult {
					t.Fatalf("Execute() result = %#v, want %#v", results[0], *tc.wantResult)
				}
			}

			gotContent, readErr := fsys.ReadFile(currentPath)
			if readErr != nil {
				t.Fatalf("ReadFile(): %v", readErr)
			}
			if !strings.Contains(string(gotContent), "version: "+tc.wantVersion) && tc.wantVersion != "" {
				t.Fatalf("SKILL.md version = %q, want %q", string(gotContent), tc.wantVersion)
			}
			if tc.wantUnchanged && string(gotContent) != tc.current {
				t.Fatalf("SKILL.md changed unexpectedly:\n%s", string(gotContent))
			}
			if tc.wantWarning != "" && !strings.Contains(errBuf.String(), tc.wantWarning) {
				t.Fatalf("warning = %q, want substring %q", errBuf.String(), tc.wantWarning)
			}
		})
	}
}

func TestExecuteNoTagFound(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	currentPath := "/repo/.agents/skills/review/SKILL.md"
	fsys.Files[currentPath] = []byte(skillFile("review", "1.0.0", "Conteudo"))
	fsys.Dirs["/repo/.agents/skills/review"] = true
	svc := NewService(fsys, &output.Printer{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}})

	scriptDir := writeFakeGit(t, fakeGitSpec{
		describeFails:  true,
		describeStderr: "fatal: No names found, cannot describe anything.\n",
	})
	t.Setenv("PATH", scriptDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := svc.Execute("/repo", ".agents/skills", false)
	if !errors.Is(err, ErrNoTagFound) {
		t.Fatalf("Execute() error = %v, want ErrNoTagFound", err)
	}
}

func TestExecutePropagaFalhaAoBuscarTag(t *testing.T) {
	fsys := fs.NewFakeFileSystem()
	currentPath := "/repo/.agents/skills/review/SKILL.md"
	fsys.Files[currentPath] = []byte(skillFile("review", "1.0.0", "Conteudo"))
	fsys.Dirs["/repo/.agents/skills/review"] = true
	svc := NewService(fsys, &output.Printer{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}})

	scriptDir := writeFakeGit(t, fakeGitSpec{
		describeFails:  true,
		describeStderr: "fatal: not a git repository (or any of the parent directories): .git\n",
	})
	t.Setenv("PATH", scriptDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	_, err := svc.Execute("/repo", ".agents/skills", false)
	if err == nil {
		t.Fatal("Execute() error = nil, want failure")
	}
	if errors.Is(err, ErrNoTagFound) {
		t.Fatalf("Execute() error = %v, nao deveria mascarar como ErrNoTagFound", err)
	}
	if !strings.Contains(err.Error(), "falha ao localizar ultima tag") {
		t.Fatalf("Execute() error = %v, want contexto sobre ultima tag", err)
	}
}

func TestUpdateFrontmatterVersion(t *testing.T) {
	t.Run("preserva conteudo apos frontmatter", func(t *testing.T) {
		content := []byte("---\nname: review\nversion: 1.1.0\ndescription: texto\n---\n\n# Corpo\n")
		got, err := UpdateFrontmatterVersion(content, "1.1.1")
		if err != nil {
			t.Fatalf("UpdateFrontmatterVersion() error = %v", err)
		}
		want := "---\nname: review\nversion: 1.1.1\ndescription: texto\n---\n\n# Corpo\n"
		if string(got) != want {
			t.Fatalf("UpdateFrontmatterVersion() = %q, want %q", string(got), want)
		}
	})

	t.Run("insere version quando ausente", func(t *testing.T) {
		content := []byte("---\nname: review\ndescription: texto\n---\n")
		got, err := UpdateFrontmatterVersion(content, "1.1.1")
		if err != nil {
			t.Fatalf("UpdateFrontmatterVersion() error = %v", err)
		}
		want := "---\nname: review\ndescription: texto\nversion: 1.1.1\n---\n"
		if string(got) != want {
			t.Fatalf("UpdateFrontmatterVersion() = %q, want %q", string(got), want)
		}
	})
}

type fakeGitSpec struct {
	describeOutput string
	describeFails  bool
	describeStderr string
	diffOutput     string
	logOutputs     map[string]string
	showBodies     map[string]string
	showFiles      map[string]string
}

func writeFakeGit(t *testing.T, spec fakeGitSpec) string {
	t.Helper()

	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(): %v", err)
	}

	writeDataFile(t, dataDir, "describe", spec.describeOutput)
	writeDataFile(t, dataDir, "describe_stderr", spec.describeStderr)
	for key, value := range spec.logOutputs {
		writeDataFile(t, dataDir, "log_"+sanitizeKey(key), value)
	}
	for key, value := range spec.showBodies {
		writeDataFile(t, dataDir, "body_"+sanitizeKey(key), value)
	}
	for key, value := range spec.showFiles {
		writeDataFile(t, dataDir, "file_"+sanitizeKey(key), value)
	}
	writeDataFile(t, dataDir, "diff", spec.diffOutput)

	script := fmt.Sprintf(`#!/bin/sh
set -eu
data_dir=%q
if [ "$1" = "-C" ]; then
  shift
  shift
fi

sanitize() {
  printf '%%s' "$1" | tr '/:.' '___'
}

cmd="$1"
shift
case "$cmd" in
  describe)
    if [ %q = "true" ]; then
      if [ -f "$data_dir/describe_stderr" ]; then
        cat "$data_dir/describe_stderr" >&2
      fi
      exit 1
    fi
    cat "$data_dir/describe"
    ;;
  diff)
    cat "$data_dir/diff"
    ;;
  log)
    last=""
    for arg in "$@"; do
      last="$arg"
    done
    key="$(sanitize "$last")"
    cat "$data_dir/log_$key"
    ;;
  show)
    if [ "$1" = "--format=%%b" ]; then
      hash="$3"
      key="$(sanitize "$hash")"
      cat "$data_dir/body_$key"
      exit 0
    fi
    key="$(sanitize "$1")"
    cat "$data_dir/file_$key"
    ;;
  *)
    echo "comando git nao suportado: $cmd" >&2
    exit 1
    ;;
esac
`, dataDir, map[bool]string{true: "true", false: "false"}[spec.describeFails])

	scriptPath := filepath.Join(dir, "git")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(git): %v", err)
	}

	return dir
}

func writeDataFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", name, err)
	}
}

func sanitizeKey(value string) string {
	replacer := strings.NewReplacer("/", "_", ":", "_", ".", "_")
	return replacer.Replace(value)
}

func skillFile(name, version, body string) string {
	return fmt.Sprintf("---\nname: %s\nversion: %s\ndescription: skill %s\n---\n\n# %s\n", name, version, name, body)
}

func skillWithoutVersion(name string) string {
	return fmt.Sprintf("---\nname: %s\ndescription: skill %s\n---\n\n# Corpo\n", name, name)
}

func malformedYAMLSkill(name, version string) string {
	return fmt.Sprintf("---\nname: %s\nversion: %s\ndescription: [quebrado\n---\n\n# Corpo\n", name, version)
}

func skillNameFromContent(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "name: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "name: "))
		}
	}
	return "review"
}
