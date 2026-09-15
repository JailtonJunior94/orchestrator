package install

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func seedGovernanceArtifacts(ffs *fs.FakeFileSystem) {
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	ffs.Dirs["/source/.opencode/plugin"] = true
	ffs.Dirs["/project/.opencode/plugin"] = true
	ffs.Files["/source/.opencode/plugin/governance.js"] = []byte("export const governance = {}\n")
	ffs.Files["/project/.opencode/plugin/governance.js"] = []byte("export const governance = {}\n")
	for _, hook := range NewHelper().governanceHookFiles() {
		ffs.Dirs["/source/.claude/hooks"] = true
		ffs.Dirs["/project/.claude/hooks"] = true
		ffs.Files["/source/.claude/hooks/"+hook] = []byte("#!/usr/bin/env bash\necho " + hook + "\n")
		ffs.Files["/project/.claude/hooks/"+hook] = []byte("#!/usr/bin/env bash\necho " + hook + "\n")
	}
}

func artifactItems(items []VerifyItem) map[string]VerifyState {
	out := make(map[string]VerifyState)
	for _, item := range items {
		if item.Kind == VerifyKindArtifact {
			out[item.Skill] = item.State
		}
	}
	return out
}

func verifyArtifacts(t *testing.T, ffs *fs.FakeFileSystem, tool skills.Tool) map[string]VerifyState {
	t.Helper()
	items, err := setupTestService(ffs).Verify(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{tool},
	})
	if err != nil {
		t.Fatalf("Verify falhou: %v", err)
	}
	return artifactItems(items)
}

func TestVerify_ReportsMissingOpenCodeGovernancePlugin(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedGovernanceArtifacts(ffs)

	before := verifyArtifacts(t, ffs, skills.ToolOpenCode)
	if before[".opencode/plugin/governance.js"] != VerifyStateCurrent {
		t.Fatalf("plugin instalado deveria ser current; got %q", before[".opencode/plugin/governance.js"])
	}

	delete(ffs.Files, "/project/.opencode/plugin/governance.js")

	after := verifyArtifacts(t, ffs, skills.ToolOpenCode)
	if after[".opencode/plugin/governance.js"] != VerifyStateMissing {
		t.Fatalf("apagar o plugin de governanca deve ser detectado pelo verify; got %q", after[".opencode/plugin/governance.js"])
	}
}

func TestVerify_ReportsDriftedOpenCodeGovernancePlugin(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedGovernanceArtifacts(ffs)

	ffs.Files["/project/.opencode/plugin/governance.js"] = []byte("export const governance = { tampered: true }\n")

	items := verifyArtifacts(t, ffs, skills.ToolOpenCode)
	if items[".opencode/plugin/governance.js"] != VerifyStateDrifted {
		t.Fatalf("plugin adulterado deve ser drifted; got %q", items[".opencode/plugin/governance.js"])
	}
}

func TestVerify_ReportsMissingToolHookScripts(t *testing.T) {
	t.Parallel()
	ffs := fs.NewFakeFileSystem()
	seedGovernanceArtifacts(ffs)

	before := verifyArtifacts(t, ffs, skills.ToolClaude)
	if before[".claude/hooks/validate-governance.sh"] != VerifyStateCurrent {
		t.Fatalf("hook instalado deveria ser current; got %q", before[".claude/hooks/validate-governance.sh"])
	}

	delete(ffs.Files, "/project/.claude/hooks/validate-governance.sh")

	after := verifyArtifacts(t, ffs, skills.ToolClaude)
	if after[".claude/hooks/validate-governance.sh"] != VerifyStateMissing {
		t.Fatalf("apagar o hook de governanca deve ser detectado pelo verify; got %q", after[".claude/hooks/validate-governance.sh"])
	}
}
