package config

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestLocalSource_SourceDir(t *testing.T) {
	tests := []struct {
		name string
		dir  string
	}{
		{"absolute path", "/tmp/governance"},
		{"relative path", "./source"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &LocalSource{Dir: tt.dir}
			if got := src.SourceDir(); got != tt.dir {
				t.Errorf("SourceDir() = %q, want %q", got, tt.dir)
			}
		})
	}
}

func TestInstallOptions_Fields(t *testing.T) {
	opts := InstallOptions{
		ProjectDir:   "/project",
		SourceDir:    "/source",
		Tools:        []skills.Tool{skills.ToolClaude, skills.ToolGemini},
		Langs:        []skills.Lang{skills.LangGo},
		LinkMode:     skills.LinkSymlink,
		DryRun:       true,
		GenerateCtx:  true,
		CodexProfile: "full",
		FocusPaths:   []string{"src/"},
	}

	if opts.ProjectDir != "/project" {
		t.Errorf("ProjectDir = %q, want /project", opts.ProjectDir)
	}
	if len(opts.Tools) != 2 {
		t.Errorf("Tools len = %d, want 2", len(opts.Tools))
	}
	if len(opts.Langs) != 1 {
		t.Errorf("Langs len = %d, want 1", len(opts.Langs))
	}
	if !opts.DryRun {
		t.Error("DryRun should be true")
	}
	if opts.LinkMode != skills.LinkSymlink {
		t.Errorf("LinkMode = %q, want symlink", opts.LinkMode)
	}
}

func TestUpgradeOptions_Fields(t *testing.T) {
	opts := UpgradeOptions{
		ProjectDir:   "/project",
		SourceDir:    "/source",
		CheckOnly:    true,
		Langs:        []skills.Lang{skills.LangNode, skills.LangPython},
		CodexProfile: "lean",
	}

	if opts.ProjectDir != "/project" {
		t.Errorf("ProjectDir = %q, want /project", opts.ProjectDir)
	}
	if !opts.CheckOnly {
		t.Error("CheckOnly should be true")
	}
	if len(opts.Langs) != 2 {
		t.Errorf("Langs len = %d, want 2", len(opts.Langs))
	}
	if opts.CodexProfile != "lean" {
		t.Errorf("CodexProfile = %q, want lean", opts.CodexProfile)
	}
}

func TestLocalSource_ImplementsInterface(t *testing.T) {
	var _ SourceProvider = &LocalSource{}
}
