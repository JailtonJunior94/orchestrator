package specs_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/runtime/specs"
)

const acpSDKModulePath = "github.com/coder/acp-go-sdk"

func readGoModACPVersion(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		for i, field := range fields {
			if field != acpSDKModulePath || i+1 >= len(fields) {
				continue
			}
			if version := fields[i+1]; strings.HasPrefix(version, "v") {
				return version
			}
		}
	}
	t.Fatalf("module %q not declared in go.mod", acpSDKModulePath)
	return ""
}

func TestEveryRegisteredSpecSDKVersionMatchesGoMod(t *testing.T) {
	t.Parallel()

	want := readGoModACPVersion(t)
	catalog := specs.NewCatalog()

	agents := catalog.Registry()
	if len(agents) == 0 {
		t.Fatal("agent registry is empty; the assertion would pass vacuously")
	}

	for _, agent := range agents {
		spec, err := catalog.SpecOf(agent)
		if err != nil {
			t.Fatalf("SpecOf(%s): %v", agent.ID(), err)
		}
		if got := spec.SDKVersion(); got != want {
			t.Errorf("agent %q SDKVersion = %q; go.mod declares %q for %s",
				agent.ID(), got, want, acpSDKModulePath)
		}
	}
}

func TestEverySDKVersionConstantMatchesGoMod(t *testing.T) {
	t.Parallel()

	want := readGoModACPVersion(t)

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read specs package dir: %v", err)
	}

	constPattern := regexp.MustCompile(`(\w*SDKVersion)\s*=\s*"([^"]*)"`)
	found := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for _, match := range constPattern.FindAllStringSubmatch(string(data), -1) {
			found++
			if match[2] != want {
				t.Errorf("%s: constant %s = %q; go.mod declares %q", name, match[1], match[2], want)
			}
		}
	}

	if wantAtLeast := len(specs.NewCatalog().Registry()); found < wantAtLeast {
		t.Errorf("found %d *SDKVersion constants; expected at least one per registered agent (%d)", found, wantAtLeast)
	}
}
