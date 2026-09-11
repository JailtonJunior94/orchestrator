package install

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func TestInstall_OpenCode_FailsNoisilyWhenPluginRuntimeMissing(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	agentDet := &fakeAgentDetector{tools: []skills.Tool{skills.ToolOpenCode}}
	langDet := detect.NewFileDetector(ffs)
	lp := newFakeLookPather()
	svc := setupTestServiceFull(ffs, agentDet, langDet, lp)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	})
	if err == nil {
		t.Fatal("install must fail noisily when bash is unavailable (RF-23)")
	}
	if !errors.Is(err, ErrOpenCodePluginRuntimeMissing) {
		t.Fatalf("error = %v; want ErrOpenCodePluginRuntimeMissing", err)
	}
	if !strings.Contains(err.Error(), "bash") {
		t.Fatalf("error message must name the exact missing dependency; got %v", err)
	}
	if ffs.Exists("/project/opencode.json") {
		t.Error("opencode.json must not be written when the plugin runtime is missing")
	}
}

func TestInstall_OpenCode_SucceedsWhenPluginRuntimePresent(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	agentDet := &fakeAgentDetector{tools: []skills.Tool{skills.ToolOpenCode}}
	langDet := detect.NewFileDetector(ffs)
	lp := newFakeLookPather("bash")
	svc := setupTestServiceFull(ffs, agentDet, langDet, lp)

	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("unexpected error with bash available: %v", err)
	}
	if !ffs.Exists("/project/opencode.json") {
		t.Error("opencode.json must be written once the plugin runtime is present")
	}
}

func TestVerify_OpenCode_NeverReportsCurrentWhenPluginRuntimeMissing(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true

	lp := newFakeLookPather()
	svc := setupTestServiceFull(ffs, &fakeAgentDetector{}, detect.NewFileDetector(ffs), lp)

	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: "/project",
		Tools:      []skills.Tool{skills.ToolOpenCode},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	found := false
	for _, item := range items {
		if item.Kind != VerifyKindRuntime {
			continue
		}
		found = true
		if item.State == VerifyStateCurrent {
			t.Fatalf("runtime item reported current with bash missing: %+v", item)
		}
	}
	if !found {
		t.Fatal("Verify did not emit a runtime item for opencode's plugin runtime (RF-23)")
	}
}

func TestInstall_OpenCode_ModelFlagMapsToConfigKey(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	svc := setupTestService(ffs)
	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
		Model:      "claude-opus-4",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/opencode.json")
	if err != nil {
		t.Fatalf("opencode.json not written: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode opencode.json: %v", err)
	}
	if doc["model"] != "claude-opus-4" {
		t.Errorf("model = %v; want %q", doc["model"], "claude-opus-4")
	}
}

func TestInstall_OpenCode_WritesOnlyPermissionBlock(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/project/opencode.json"] = []byte(`{"$schema":"https://opencode.ai/config.json","theme":"dark"}`)

	svc := setupTestService(ffs)
	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := ffs.ReadFile("/project/opencode.json")
	if err != nil {
		t.Fatalf("opencode.json not written: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode opencode.json: %v", err)
	}
	if doc["$schema"] != "https://opencode.ai/config.json" {
		t.Errorf("$schema = %v; want preserved", doc["$schema"])
	}
	if doc["theme"] != "dark" {
		t.Errorf("theme = %v; want preserved", doc["theme"])
	}
	if _, ok := doc["permission"]; !ok {
		t.Error("permission block missing")
	}
	if _, ok := doc["skills"]; ok {
		t.Error("installer must never write skills key")
	}
	if _, ok := doc["instructions"]; ok {
		t.Error("installer must never write instructions key")
	}
}

func TestInstall_OpenCode_FailsWhenDefaultPermissionWildcardDeniesRequiredTool(t *testing.T) {
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	original := openCodePermissionFactory
	openCodePermissionFactory = func() map[string]any {
		return map[string]any{"edit": "deny"}
	}
	defer func() { openCodePermissionFactory = original }()

	svc := setupTestService(ffs)
	err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	})
	if !errors.Is(err, specs.ErrOpenCodePermissionWildcardDeny) {
		t.Fatalf("Execute() error = %v; want ErrOpenCodePermissionWildcardDeny", err)
	}
	if ffs.Exists("/project/opencode.json") {
		t.Error("opencode.json must not be written when the default permission denies a required tool")
	}
}

func TestInstall_OpenCode_IdempotentReinstall(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	svc := setupTestService(ffs)
	opts := config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	}
	if err := svc.Execute(opts); err != nil {
		t.Fatalf("first install: %v", err)
	}
	first, err := ffs.ReadFile("/project/opencode.json")
	if err != nil {
		t.Fatalf("read after first install: %v", err)
	}
	if err := svc.Execute(opts); err != nil {
		t.Fatalf("second install: %v", err)
	}
	second, err := ffs.ReadFile("/project/opencode.json")
	if err != nil {
		t.Fatalf("read after second install: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("reinstall is not idempotent:\nfirst=%s\nsecond=%s", first, second)
	}
}

func TestInstall_OpenCode_DoesNotCopySkillsOrCreateSymlinks(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Dirs["/source/.agents/skills/review"] = true
	ffs.Files["/source/.agents/skills/review/SKILL.md"] = []byte("# review")

	svc := setupTestService(ffs)
	if err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkSymlink,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ffs.Exists("/project/.opencode/skills") {
		t.Error(".opencode/skills must never be created")
	}
}

func TestInstall_OpenCode_DepositsPluginDir(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true

	svc := setupTestService(ffs)
	if err := svc.Execute(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !ffs.IsDir("/project/.opencode/plugin") {
		t.Error(".opencode/plugin directory not deposited")
	}
}

func TestInstall_OpenCode_ShipsCanonicalValidators(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project-opencode"] = true
	ffs.Dirs["/project-claude"] = true
	ffs.Dirs["/source"] = true
	ffs.Files["/source/.agents/scripts/validate-task-evidence.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.agents/scripts/validate-bugfix-evidence.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.agents/scripts/validate-refactor-evidence.sh"] = []byte("#!/usr/bin/env bash")
	ffs.Files["/source/.agents/scripts/validate-review-evidence.sh"] = []byte("#!/usr/bin/env bash")

	svcOpenCode := setupTestService(ffs)
	if err := svcOpenCode.Execute(config.InstallOptions{
		ProjectDir: "/project-opencode",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolOpenCode},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("opencode install: %v", err)
	}

	svcClaude := setupTestService(ffs)
	if err := svcClaude.Execute(config.InstallOptions{
		ProjectDir: "/project-claude",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
		LinkMode:   skills.LinkCopy,
	}); err != nil {
		t.Fatalf("claude install: %v", err)
	}

	for _, v := range []string{
		"validate-task-evidence.sh",
		"validate-bugfix-evidence.sh",
		"validate-refactor-evidence.sh",
		"validate-review-evidence.sh",
	} {
		openCodeHas := ffs.Exists("/project-opencode/.agents/scripts/" + v)
		claudeHas := ffs.Exists("/project-claude/.agents/scripts/" + v)
		if openCodeHas != claudeHas {
			t.Errorf("validator %q parity mismatch: opencode=%v claude=%v", v, openCodeHas, claudeHas)
		}
		if !openCodeHas {
			t.Errorf("validator %q missing from OpenCode-only project", v)
		}
	}
}
