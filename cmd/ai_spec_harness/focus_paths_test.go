package aispecharness

import (
	"os"
	"reflect"
	"testing"
)

func TestParseFocusPaths_CommaSeparated(t *testing.T) {
	got := parseFocusPaths("services/go-api/handler.go,services/go-api/main.go")
	want := []string{"services/go-api/handler.go", "services/go-api/main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFocusPaths comma: got %v, want %v", got, want)
	}
}

func TestParseFocusPaths_NewlineSeparated(t *testing.T) {
	got := parseFocusPaths("services/go-api/handler.go\nservices/web/app.ts")
	want := []string{"services/go-api/handler.go", "services/web/app.ts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFocusPaths newline: got %v, want %v", got, want)
	}
}

func TestParseFocusPaths_Single(t *testing.T) {
	got := parseFocusPaths("services/go-api/handler.go")
	want := []string{"services/go-api/handler.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFocusPaths single: got %v, want %v", got, want)
	}
}

func TestParseFocusPaths_Empty(t *testing.T) {
	got := parseFocusPaths("")
	if got != nil {
		t.Errorf("parseFocusPaths empty: got %v, want nil", got)
	}
}

func TestParseFocusPaths_EnvVar_NewlineSeparated(t *testing.T) {
	t.Setenv("FOCUS_PATHS", "services/go-api/handler.go\nservices/web/app.ts")

	got := parseFocusPaths("")
	want := []string{"services/go-api/handler.go", "services/web/app.ts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFocusPaths env newline: got %v, want %v", got, want)
	}
}

func TestParseFocusPaths_EnvVar_CommaSeparated(t *testing.T) {
	t.Setenv("FOCUS_PATHS", "services/go-api/handler.go,services/web/app.ts")

	got := parseFocusPaths("")
	want := []string{"services/go-api/handler.go", "services/web/app.ts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFocusPaths env comma: got %v, want %v", got, want)
	}
}

func TestParseFocusPaths_FlagTakesPrecedenceOverEnv(t *testing.T) {
	t.Setenv("FOCUS_PATHS", "from/env/file.go")

	got := parseFocusPaths("from/flag/file.go")
	want := []string{"from/flag/file.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFocusPaths flag precedence: got %v, want %v", got, want)
	}
}

func TestParseFocusPaths_TrimsWhitespace(t *testing.T) {
	got := parseFocusPaths("  services/go-api/handler.go , services/web/app.ts  ")
	want := []string{"services/go-api/handler.go", "services/web/app.ts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFocusPaths trim: got %v, want %v", got, want)
	}
}

// TestInstallFocusPaths_E2E verifies that --focus-paths causes the install command
// to use the correct toolchain in contextgen (Go subproject wins over Node at root).
func TestInstallFocusPaths_E2E(t *testing.T) {
	projectDir := t.TempDir()
	sourceDir := t.TempDir()

	// Minimal source tree (no AGENTS.md so contextgen writes freely)
	if err := os.MkdirAll(sourceDir+"/.claude/hooks", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceDir+"/.claude/hooks/validate-governance.sh", []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sourceDir+"/.claude/hooks/validate-preload.sh", []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a monorepo: Node at root, Go in services/api/
	if err := os.WriteFile(projectDir+"/package.json", []byte(`{"name":"root","scripts":{"test":"jest"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projectDir+"/services/api", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectDir+"/services/api/go.mod", []byte("module example.com/api\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run install with --focus-paths pointing at the Go subproject
	installTools = "claude"
	installLangs = ""
	installMode = "copy"
	installDryRun = false
	installSource = sourceDir
	installRef = ""
	installNoCtx = false
	installCodexProfile = "full"
	installFocusPaths = "services/api/handler.go"
	t.Cleanup(func() {
		installTools = ""
		installLangs = ""
		installMode = "symlink"
		installDryRun = false
		installSource = ""
		installRef = ""
		installNoCtx = false
		installCodexProfile = "full"
		installFocusPaths = ""
	})

	err := runInstall(installCmd, []string{projectDir})
	if err != nil {
		t.Fatalf("install with --focus-paths failed: %v", err)
	}

	// Verify AGENTS.md was generated and contains Go toolchain commands
	data, err := os.ReadFile(projectDir + "/AGENTS.md")
	if err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}
	content := string(data)

	// With focus on services/api/handler.go, Go toolchain should be detected
	if !containsAny(content, "go test ./...", "gofmt") {
		t.Errorf("AGENTS.md should contain Go toolchain commands when focus is on Go subproject, got snippet: %q",
			content[:min(200, len(content))])
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
