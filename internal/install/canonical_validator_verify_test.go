package install

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/config"
	"github.com/JailtonJunior94/ai-spec-harness/internal/detect"
	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

var canonicalValidatorFixtures = map[string]string{
	"/.agents/hooks/validate-preload.sh":       "#!/usr/bin/env bash\nreal preload gate\n",
	"/.agents/hooks/validate-governance.sh":    "#!/usr/bin/env bash\nreal governance gate\n",
	"/.agents/scripts/hook-prereq-gate.sh":     "#!/usr/bin/env bash\nreal prereq gate\n",
	"/.agents/scripts/validate-session-end.sh": "#!/usr/bin/env bash\nreal session end gate\n",
	"/.agents/lib/parse-hook-input.sh":         "#!/usr/bin/env bash\nreal payload parser\n",
}

func canonicalValidatorFS(t *testing.T) *fs.FakeFileSystem {
	t.Helper()
	ffs := fs.NewFakeFileSystem()
	ffs.Dirs["/project"] = true
	ffs.Dirs["/source"] = true
	for rel, body := range canonicalValidatorFixtures {
		ffs.Files["/source"+rel] = []byte(body)
		ffs.Files["/project"+rel] = []byte(body)
	}
	return ffs
}

func canonicalValidatorStates(t *testing.T, ffs *fs.FakeFileSystem) map[string]VerifyState {
	t.Helper()
	svc := setupTestServiceFull(ffs, &fakeAgentDetector{}, detect.NewFileDetector(ffs), newFakeLookPather("bash"))
	items, err := svc.Verify(config.InstallOptions{
		ProjectDir: "/project",
		SourceDir:  "/source",
		Tools:      []skills.Tool{skills.ToolClaude},
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	states := make(map[string]VerifyState)
	for _, item := range items {
		if item.Kind == VerifyKindArtifact {
			states[item.Skill] = item.State
		}
	}
	return states
}

func TestVerifyChecksTheCanonicalValidatorsUnderAgents(t *testing.T) {
	t.Parallel()

	states := canonicalValidatorStates(t, canonicalValidatorFS(t))
	for _, want := range []string{
		".agents/hooks/validate-preload.sh",
		".agents/hooks/validate-governance.sh",
		".agents/scripts/hook-prereq-gate.sh",
		".agents/scripts/validate-session-end.sh",
		".agents/lib/parse-hook-input.sh",
	} {
		state, ok := states[want]
		if !ok {
			t.Fatalf("verify emitted no item for %q — tampering with the gate logic would stay invisible (O-07)", want)
		}
		if state != VerifyStateCurrent {
			t.Fatalf("item %q state = %v; want current for an intact install", want, state)
		}
	}
}

func TestVerifyReportsTamperedCanonicalValidatorAsDrifted(t *testing.T) {
	t.Parallel()

	ffs := canonicalValidatorFS(t)
	for _, rel := range []string{
		"/.agents/hooks/validate-preload.sh",
		"/.agents/hooks/validate-governance.sh",
		"/.agents/scripts/hook-prereq-gate.sh",
		"/.agents/scripts/validate-session-end.sh",
	} {
		ffs.Files["/project"+rel] = []byte("#!/usr/bin/env bash\nexit 0\n")
	}
	delete(ffs.Files, "/project/.agents/lib/parse-hook-input.sh")

	states := canonicalValidatorStates(t, ffs)
	for _, want := range []string{
		".agents/hooks/validate-preload.sh",
		".agents/hooks/validate-governance.sh",
		".agents/scripts/hook-prereq-gate.sh",
		".agents/scripts/validate-session-end.sh",
	} {
		if states[want] != VerifyStateDrifted {
			t.Fatalf("validator %q replaced by 'exit 0' reported as %v; verify must report drifted", want, states[want])
		}
	}
	if states[".agents/lib/parse-hook-input.sh"] != VerifyStateMissing {
		t.Fatalf("deleted .agents/lib/parse-hook-input.sh reported as %v; want missing", states[".agents/lib/parse-hook-input.sh"])
	}
}
