package specs_test

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

// TestClaudeSnapshot valida cada campo de Claude() campo a campo (snapshot test).
// Qualquer alteração não intencional na spec falha aqui.
func TestClaudeSnapshot(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Claude()

	if got, want := spec.ID, "claude"; got != want {
		t.Errorf("ID = %q; want %q", got, want)
	}

	if got, want := spec.DisplayName, "Claude (ACP)"; got != want {
		t.Errorf("DisplayName = %q; want %q", got, want)
	}

	if got, want := spec.Command, "claude-agent-acp"; got != want {
		t.Errorf("Command = %q; want %q", got, want)
	}

	if len(spec.FixedArgs) != 0 {
		t.Errorf("FixedArgs = %v; want empty", spec.FixedArgs)
	}

	if got, want := spec.AccessModeFlag, "--bypass-permissions"; got != want {
		t.Errorf("AccessModeFlag = %q; want %q", got, want)
	}

	// Valida o fallback npx
	if got, want := len(spec.Fallbacks), 1; got != want {
		t.Fatalf("len(Fallbacks) = %d; want %d", got, want)
	}

	fb := spec.Fallbacks[0]

	if got, want := fb.Command, "npx"; got != want {
		t.Errorf("Fallbacks[0].Command = %q; want %q", got, want)
	}

	wantArgs := []string{"--yes", specs.ClaudeNpmPackage + "@" + specs.ClaudeNpmVersion}

	if got, want := len(fb.FixedArgs), len(wantArgs); got != want {
		t.Fatalf("len(Fallbacks[0].FixedArgs) = %d; want %d", got, want)
	}

	for i, want := range wantArgs {
		if got := fb.FixedArgs[i]; got != want {
			t.Errorf("Fallbacks[0].FixedArgs[%d] = %q; want %q", i, got, want)
		}
	}
}

// TestClaudeStability garante que Claude() retorna a mesma estrutura entre chamadas.
func TestClaudeStability(t *testing.T) {
	t.Parallel()

	a := specs.NewCatalog().Claude()
	b := specs.NewCatalog().Claude()

	if a.ID != b.ID {
		t.Errorf("ID inconsistente: %q != %q", a.ID, b.ID)
	}

	if a.Command != b.Command {
		t.Errorf("Command inconsistente: %q != %q", a.Command, b.Command)
	}

	if a.AccessModeFlag != b.AccessModeFlag {
		t.Errorf("AccessModeFlag inconsistente: %q != %q", a.AccessModeFlag, b.AccessModeFlag)
	}

	if len(a.Fallbacks) != len(b.Fallbacks) {
		t.Errorf("len(Fallbacks) inconsistente: %d != %d", len(a.Fallbacks), len(b.Fallbacks))
	}
}

// TestLauncherKind valida Launcher.Kind() para os dois modos.
func TestLauncherKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		launcher specs.Launcher
		wantKind string
	}{
		{
			name:     "binary launcher",
			launcher: specs.NewBinaryLauncher("claude-agent-acp"),
			wantKind: "binary",
		},
		{
			name:     "npx launcher",
			launcher: specs.NewNpxLauncher(specs.ClaudeNpmPackage, specs.ClaudeNpmVersion),
			wantKind: "npx",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.launcher.Kind(); got != tc.wantKind {
				t.Errorf("Kind() = %q; want %q", got, tc.wantKind)
			}
		})
	}
}

// TestLauncherCommand valida o comando e args retornados por Launcher.Command().
func TestLauncherCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		launcher specs.Launcher
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "binary launcher sem args",
			launcher: specs.NewBinaryLauncher("claude-agent-acp"),
			wantCmd:  "claude-agent-acp",
			wantArgs: []string{},
		},
		{
			name:     "binary launcher com args",
			launcher: specs.NewBinaryLauncher("claude-agent-acp", "--bypass-permissions"),
			wantCmd:  "claude-agent-acp",
			wantArgs: []string{"--bypass-permissions"},
		},
		{
			name:     "npx launcher",
			launcher: specs.NewNpxLauncher("@agentclientprotocol/claude-agent-acp", "0.1.0"),
			wantCmd:  "npx",
			wantArgs: []string{"--yes", "@agentclientprotocol/claude-agent-acp@0.1.0"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotCmd, gotArgs := tc.launcher.Command()

			if gotCmd != tc.wantCmd {
				t.Errorf("Command() cmd = %q; want %q", gotCmd, tc.wantCmd)
			}

			if len(gotArgs) != len(tc.wantArgs) {
				t.Fatalf("Command() len(args) = %d; want %d", len(gotArgs), len(tc.wantArgs))
			}

			for i, want := range tc.wantArgs {
				if gotArgs[i] != want {
					t.Errorf("Command() args[%d] = %q; want %q", i, gotArgs[i], want)
				}
			}
		})
	}
}

// TestClaudeNpmFallbackFormat valida o formato exato do fallback npx da spec Claude.
func TestClaudeNpmFallbackFormat(t *testing.T) {
	t.Parallel()

	spec := specs.NewCatalog().Claude()

	if len(spec.Fallbacks) == 0 {
		t.Fatal("Claude() deve ter ao menos um fallback")
	}

	fb := spec.Fallbacks[0]

	// O fallback deve ter exatamente 2 args: "--yes" e "pkg@ver"
	if len(fb.FixedArgs) != 2 {
		t.Fatalf("Fallback deve ter 2 FixedArgs; got %d: %v", len(fb.FixedArgs), fb.FixedArgs)
	}

	if fb.FixedArgs[0] != "--yes" {
		t.Errorf("FixedArgs[0] = %q; want %q", fb.FixedArgs[0], "--yes")
	}

	wantPkgArg := specs.ClaudeNpmPackage + "@" + specs.ClaudeNpmVersion
	if fb.FixedArgs[1] != wantPkgArg {
		t.Errorf("FixedArgs[1] = %q; want %q", fb.FixedArgs[1], wantPkgArg)
	}
}
