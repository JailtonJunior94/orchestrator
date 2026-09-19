package install_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

func delegationScriptResolver(dir string) specs.ScriptResolver {
	return func(relPath string) ([]byte, error) {
		return os.ReadFile(filepath.Join(dir, filepath.FromSlash(relPath)))
	}
}

const delegationCanonical = ".agents/scripts/hook-prereq-gate.sh"

func delegationFixture(t *testing.T, wrapperBody string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".agents", "scripts"), 0o755); err != nil {
		t.Fatalf("mkdir canonical: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".claude", "hooks"), 0o755); err != nil {
		t.Fatalf("mkdir wrapper: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(delegationCanonical)), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write canonical: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "hooks", "wrapper.sh"), []byte(wrapperBody), 0o755); err != nil {
		t.Fatalf("write wrapper: %v", err)
	}
	return dir
}

func TestTextualMentionIsNotProofOfDelegation(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"comentario":                 "#!/usr/bin/env bash\nset -euo pipefail\n# delega para .agents/scripts/hook-prereq-gate.sh\nexit 0\n",
		"mensagem":                   "#!/usr/bin/env bash\nset -euo pipefail\necho \"ERRO: rode .agents/scripts/hook-prereq-gate.sh\" >&2\nexit 0\n",
		"variavel-atribuida-sem-uso": "#!/usr/bin/env bash\nset -euo pipefail\ngate=\"$PWD/.agents/scripts/hook-prereq-gate.sh\"\necho \"nada a fazer\"\nexit 0\n",
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := delegationFixture(t, body)
			if specs.ScriptDelegatesTo(delegationScriptResolver(dir), ".claude/hooks/wrapper.sh", delegationCanonical) {
				t.Fatalf("a mera mencao textual (%s) nao pode contar como delegacao ao validador canonico", name)
			}
		})
	}
}

func TestExecutionPositionIsProofOfDelegation(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"exec-direto":        "#!/usr/bin/env bash\nset -euo pipefail\nexec bash \"$PWD/.agents/scripts/hook-prereq-gate.sh\" \"$@\"\n",
		"variavel-executada": "#!/usr/bin/env bash\nset -euo pipefail\ngate=\"$PWD/.agents/scripts/hook-prereq-gate.sh\"\nexec env AGENTS_ROOT=\"$PWD\" bash \"$gate\" \"$@\"\n",
		"pipeline":           "#!/usr/bin/env bash\nset -euo pipefail\ngate=\"$PWD/.agents/scripts/hook-prereq-gate.sh\"\nprintf '%s' \"$payload\" | bash \"$gate\" \"$1\"\n",
		"source":             "#!/usr/bin/env bash\nset -euo pipefail\nsource \"$PWD/.agents/scripts/hook-prereq-gate.sh\"\n",
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := delegationFixture(t, body)
			if !specs.ScriptDelegatesTo(delegationScriptResolver(dir), ".claude/hooks/wrapper.sh", delegationCanonical) {
				t.Fatalf("invocacao real (%s) deve contar como delegacao ao validador canonico", name)
			}
		})
	}
}
