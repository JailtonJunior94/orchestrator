package specs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

var npmInstallLine = regexp.MustCompile(`npm\s+install\s+(?:--global|-g)\b`)

var npmPinToken = regexp.MustCompile(`(@[A-Za-z0-9][\w.-]*/[\w.-]+|[A-Za-z0-9][\w.-]*)@([\w.-]+)`)

func workflowPinRegistry() map[string]string {
	return map[string]string{
		specs.ClaudeCodeNpmPackage: specs.ClaudeCodeNpmVersion,
		specs.CodexCLINpmPackage:   specs.CodexCLINpmVersion,
		specs.ClaudeNpmPackage:     specs.ClaudeNpmVersion,
		specs.CodexNpmPackage:      specs.CodexNpmVersion,
		specs.CopilotNpmPackage:    specs.CopilotNpmVersion,
		specs.OpenCodeNpmPackage:   specs.OpenCodeNpmVersion,
	}
}

type workflowPin struct {
	workflow string
	line     int
	raw      string
	pkg      string
	version  string
}

func collectWorkflowPins(t *testing.T, workflow string) []workflowPin {
	t.Helper()

	path := filepath.Join("..", "..", "..", ".github", "workflows", workflow)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	pins := make([]workflowPin, 0, 4)
	lines := strings.Split(string(data), "\n")
	continuation := false
	for index, line := range lines {
		code := line
		if hash := strings.Index(code, "#"); hash >= 0 {
			code = code[:hash]
		}
		trimmed := strings.TrimSpace(code)
		bounds := npmInstallLine.FindStringIndex(trimmed)
		if bounds == nil && !continuation {
			continue
		}
		payload := trimmed
		if bounds != nil {
			payload = trimmed[bounds[1]:]
		}
		continuation = strings.HasSuffix(trimmed, `\`)
		payload = strings.TrimSuffix(strings.TrimSpace(payload), `\`)
		for _, field := range strings.Fields(payload) {
			if strings.HasPrefix(field, "-") {
				continue
			}
			match := npmPinToken.FindStringSubmatch(field)
			if match == nil {
				pins = append(pins, workflowPin{workflow: workflow, line: index + 1, raw: field})
				continue
			}
			pins = append(pins, workflowPin{workflow: workflow, line: index + 1, raw: field, pkg: match[1], version: match[2]})
		}
	}
	return pins
}

func TestWorkflowNpmPinsMatchRegistryConstants(t *testing.T) {
	t.Parallel()

	registry := workflowPinRegistry()
	pins := make([]workflowPin, 0, 6)
	for _, workflow := range []string{"acp-live.yml", "hooks-live.yml"} {
		pins = append(pins, collectWorkflowPins(t, workflow)...)
	}

	if len(pins) == 0 {
		t.Fatal("no npm install pins found in .github/workflows — the sync gate would be vacuous")
	}
	if len(pins) != 6 {
		t.Errorf("expected 6 npm pins across acp-live.yml and hooks-live.yml, found %d: %+v", len(pins), pins)
	}

	for _, pin := range pins {
		if pin.pkg == "" {
			t.Errorf("%s:%d: npm install target %q has no pinned @version", pin.workflow, pin.line, pin.raw)
			continue
		}
		want, known := registry[pin.pkg]
		if !known {
			t.Errorf("%s:%d: package %q pinned at %q has no constant in internal/runtime/specs", pin.workflow, pin.line, pin.pkg, pin.version)
			continue
		}
		if pin.version != want {
			t.Errorf("%s:%d: package %q pinned at %q but registry constant is %q", pin.workflow, pin.line, pin.pkg, pin.version, want)
		}
	}
}
