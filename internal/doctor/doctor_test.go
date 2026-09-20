package doctor

import (
	"fmt"
	"io"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/manifest"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

type fakeGitRepo struct {
	isRepo bool
}

func (f *fakeGitRepo) IsRepo(path string) bool               { return f.isRepo }
func (f *fakeGitRepo) Root(path string) (string, error)      { return path, nil }
func (f *fakeGitRepo) RemoteURL(path string) (string, error) { return "", nil }

func silentPrinter() *output.Printer {
	return &output.Printer{Out: io.Discard, Err: io.Discard}
}

func setupService(fake *fs.FakeFileSystem, isRepo bool) *Service {
	mfst := manifest.NewStore(fake)
	gitRepo := &fakeGitRepo{isRepo: isRepo}
	svc := NewService(fake, silentPrinter(), mfst, gitRepo)
	svc.lookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	return svc
}

func lookPathOnly(available ...string) func(string) (string, error) {
	allowed := make(map[string]bool, len(available))
	for _, name := range available {
		allowed[name] = true
	}
	return func(name string) (string, error) {
		if allowed[name] {
			return "/usr/bin/" + name, nil
		}
		return "", fmt.Errorf("%s: executable file not found in $PATH", name)
	}
}

func TestCheckGit_ValidRepo(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), true)
	check := svc.checkGit("/project")
	if check.Status != "ok" {
		t.Errorf("checkGit status = %q, want ok", check.Status)
	}
}

func TestCheckGit_NotARepo(t *testing.T) {
	svc := setupService(fs.NewFakeFileSystem(), false)
	check := svc.checkGit("/project")
	if check.Status != "warn" {
		t.Errorf("checkGit status = %q, want warn", check.Status)
	}
}

func TestCheckSkillsDir_Present(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project/.agents/skills"] = true
	fake.Dirs["/project/.agents/skills/skill-a"] = true
	fake.Dirs["/project/.agents/skills/skill-b"] = true

	svc := setupService(fake, true)
	check := svc.checkSkillsDir("/project")
	if check.Status != "ok" {
		t.Errorf("checkSkillsDir status = %q, want ok", check.Status)
	}
}

func TestCheckSkillsDir_Absent(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)
	check := svc.checkSkillsDir("/project")
	if check.Status != "fail" {
		t.Errorf("checkSkillsDir status = %q, want fail", check.Status)
	}
}

func TestCheckManifest_Present(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Files["/project/.ai_spec_harness.json"] = []byte(`{
		"version": "1.0.0",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
		"source_dir": "/src",
		"link_mode": "copy",
		"tools": ["claude"],
		"langs": ["go"],
		"skills": ["skill-a"],
		"checksums": {}
	}`)

	svc := setupService(fake, true)
	check := svc.checkManifest("/project")
	if check.Status != "ok" {
		t.Errorf("checkManifest status = %q, want ok", check.Status)
	}
}

func TestCheckManifest_Absent(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)
	check := svc.checkManifest("/project")
	if check.Status != "warn" {
		t.Errorf("checkManifest status = %q, want warn", check.Status)
	}
}

func TestCheckManifest_Corrupted(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Files["/project/.ai_spec_harness.json"] = []byte(`{invalid}`)

	svc := setupService(fake, true)
	check := svc.checkManifest("/project")
	if check.Status != "warn" {
		t.Errorf("checkManifest status = %q, want warn", check.Status)
	}
}

func TestCheckPermissions_Writable(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project"] = true

	svc := setupService(fake, true)
	check := svc.checkPermissions("/project")
	if check.Status != "ok" {
		t.Errorf("checkPermissions status = %q, want ok", check.Status)
	}
}

func TestCheckPermissions_NotWritable(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project"] = true
	fake.NoWrite["/project"] = true

	svc := setupService(fake, true)
	check := svc.checkPermissions("/project")
	if check.Status != "fail" {
		t.Errorf("checkPermissions status = %q, want fail", check.Status)
	}
}

func TestCheckSymlinks_AllValid(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project/.agents/skills"] = true
	fake.Links["/project/.agents/skills/skill-a"] = "/src/skills/skill-a"
	fake.Dirs["/project/.agents/skills/skill-a"] = true

	svc := setupService(fake, true)
	checks := svc.checkSymlinks("/project")
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	if checks[0].Status != "ok" {
		t.Errorf("checkSymlinks status = %q, want ok", checks[0].Status)
	}
}

func TestCheckSymlinks_NoSkillsDir(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)
	checks := svc.checkSymlinks("/project")
	if len(checks) != 0 {
		t.Errorf("expected no checks when skills dir absent, got %d", len(checks))
	}
}

func TestRunChecks_FullPass(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	fake.Dirs["/project"] = true
	fake.Dirs["/project/.agents/skills"] = true
	fake.Dirs["/project/.agents/skills/skill-a"] = true
	fake.Files["/project/.ai_spec_harness.json"] = []byte(`{
		"version": "1.0.0",
		"created_at": "2026-01-01T00:00:00Z",
		"updated_at": "2026-01-01T00:00:00Z",
		"source_dir": "/src",
		"link_mode": "copy",
		"tools": ["claude"],
		"langs": ["go"],
		"skills": ["skill-a"],
		"checksums": {}
	}`)

	svc := setupService(fake, true)
	checks := svc.runCoreChecks("/project", nil, nil)

	for _, c := range checks {
		if c.Status == "fail" {
			t.Errorf("check %q failed: %s", c.Name, c.Detail)
		}
	}
}

func TestCheckHookInterpreter(t *testing.T) {
	tests := []struct {
		name       string
		available  []string
		wantStatus string
	}{
		{name: "python3 available", available: []string{"python3"}, wantStatus: "ok"},
		{name: "jq available without python3", available: []string{"jq"}, wantStatus: "ok"},
		{name: "both available", available: []string{"python3", "jq"}, wantStatus: "ok"},
		{name: "neither available", available: []string{}, wantStatus: "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := setupService(fs.NewFakeFileSystem(), true)
			svc.lookPath = lookPathOnly(tt.available...)
			check := svc.checkHookInterpreter()
			if check.Status != tt.wantStatus {
				t.Errorf("checkHookInterpreter status = %q, want %q (detail=%s)", check.Status, tt.wantStatus, check.Detail)
			}
		})
	}
}

func TestExecute_DirectoryNotFound(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	svc := setupService(fake, true)
	err := svc.Execute("/nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}
