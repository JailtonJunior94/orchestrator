package skills_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/contextgen"
	"github.com/JailtonJunior94/ai-spec-harness/internal/skills"
)

func resolveRepoRootForWiring(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join(".", "..", ".."))
	if err != nil {
		t.Fatalf("resolver raiz do repositorio: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr != nil {
		t.Fatalf("go.mod nao encontrado em %s: %v", dir, statErr)
	}
	return dir
}

func readSourceForWiring(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	return string(data)
}

func containsLangWiring(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

var langsWithoutImplementationSkill = map[skills.Lang]bool{
	skills.LangJava: true,
}

func TestAllLangsAreExhaustivelyWired(t *testing.T) {
	root := resolveRepoRootForWiring(t)
	installSource := readSourceForWiring(t, filepath.Join(root, "internal", "install", "install.go"))
	toolchainSource := readSourceForWiring(t, filepath.Join(root, "internal", "detect", "toolchain.go"))

	for _, lang := range skills.AllLangs {
		lang := lang
		t.Run(string(lang), func(t *testing.T) {
			langSkills := skills.NewCatalog().LangSkills([]skills.Lang{lang})
			if len(langSkills) == 0 && !langsWithoutImplementationSkill[lang] {
				t.Fatalf("wiring 1/5 (LangSkills): %s sem skill de implementacao mapeada em internal/skills/skills.go:150-166", lang)
			}

			for _, skillName := range langSkills {
				pattern := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*true`, regexp.QuoteMeta(skillName)))
				if !pattern.MatchString(installSource) {
					t.Errorf("wiring 2/5 (langImplementationSkills): %s -> skill %q ausente em internal/install/install.go:1337-1343", lang, skillName)
				}
			}

			if !containsLangWiring(contextgen.ValidationLangOrder, string(lang)) {
				t.Errorf("wiring 3/5 (contextgen loop): %s ausente em contextgen.ValidationLangOrder (internal/contextgen/contextgen.go:339-340)", lang)
			}
			if _, ok := contextgen.ValidationLangLabels[string(lang)]; !ok {
				t.Errorf("wiring 3/5 (contextgen labels): %s sem label em contextgen.ValidationLangLabels (internal/contextgen/contextgen.go:339-340)", lang)
			}

			triggerPath := filepath.Join(root, ".agents", "skills", "agent-governance", "triggers", string(lang)+".yaml")
			if _, statErr := os.Stat(triggerPath); statErr != nil {
				t.Errorf("wiring 4/5 (trigger yaml): %s sem arquivo de trigger em %s", lang, triggerPath)
			}

			if !strings.Contains(toolchainSource, `case "`+string(lang)+`":`) {
				t.Errorf("wiring 5/5 (toolchain switch): %s sem case no switch de internal/detect/toolchain.go:148-163", lang)
			}
		})
	}
}
