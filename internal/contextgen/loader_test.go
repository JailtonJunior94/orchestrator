package contextgen

import (
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
)

func writeOrFatal(t *testing.T, fake *fs.FakeFileSystem, path, content string) {
	t.Helper()
	if err := fake.WriteFile(path, []byte(content)); err != nil {
		t.Fatalf("WriteFile %q: %v", path, err)
	}
}

func TestLoader_LoadReference_Brief_ExtractsTLDR(t *testing.T) {
	content := `# Titulo da Referencia

<!-- TL;DR
Resumo executivo desta referencia sobre tratamento de erros em Go.
Keywords: errors, wrapping, sentinel, fmt.Errorf
Load complete when: tarefa envolve criacao de tipos de erro customizados ou errors.As.
-->

## Conteudo completo aqui
Conteudo extenso que nao deve aparecer em modo brief.
`
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("refs/error-handling.md", []byte(content)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loader := NewLoader(fake)
	result, err := loader.LoadReference("refs/error-handling.md", LoadOptions{Brief: true})
	if err != nil {
		t.Fatalf("LoadReference: %v", err)
	}

	if result == content {
		t.Error("modo brief deveria retornar apenas o TL;DR, nao o conteudo completo")
	}
	if len(result) >= len(content) {
		t.Errorf("modo brief deveria ser menor que o conteudo completo: brief=%d full=%d", len(result), len(content))
	}
	if result == "" {
		t.Error("resultado em modo brief nao deveria ser vazio")
	}
}

func TestLoader_LoadReference_Brief_FallbackWhenNoTLDR(t *testing.T) {
	content := `# Referencia sem TL;DR

## Conteudo
Esta referencia nao tem bloco TL;DR.
`
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("refs/sem-tldr.md", []byte(content)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loader := NewLoader(fake)
	result, err := loader.LoadReference("refs/sem-tldr.md", LoadOptions{Brief: true})
	if err != nil {
		t.Fatalf("LoadReference: %v", err)
	}

	if result != content {
		t.Error("sem TL;DR, modo brief deve retornar conteudo completo (fallback seguro)")
	}
}

func TestLoader_LoadReference_FullMode_ReturnsComplete(t *testing.T) {
	content := `# Referencia

<!-- TL;DR
Resumo.
Keywords: a, b
Load complete when: sempre.
-->

## Conteudo completo
Texto extenso.
`
	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("refs/completa.md", []byte(content)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loader := NewLoader(fake)
	result, err := loader.LoadReference("refs/completa.md", LoadOptions{Brief: false})
	if err != nil {
		t.Fatalf("LoadReference: %v", err)
	}

	if result != content {
		t.Error("modo completo deve retornar o conteudo inteiro")
	}
}

func TestLoader_LoadReference_Brief_SizeSmaller(t *testing.T) {
	briefBlock := `Resumo executivo desta referencia.
Keywords: a, b, c
Load complete when: condicao especifica.`
	content := "# Titulo\n\n<!-- TL;DR\n" + briefBlock + "\n-->\n\n" +
		"## Secao 1\n" + "Conteudo muito extenso. " + string(make([]byte, 2000)) + "\n"

	fake := fs.NewFakeFileSystem()
	if err := fake.WriteFile("refs/grande.md", []byte(content)); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	loader := NewLoader(fake)
	brief, err := loader.LoadReference("refs/grande.md", LoadOptions{Brief: true})
	if err != nil {
		t.Fatalf("LoadReference brief: %v", err)
	}

	if len(brief)*100/len(content) >= 40 {
		t.Errorf("output brief deveria ser <40%% do tamanho completo: brief=%d full=%d (%.0f%%)",
			len(brief), len(content), float64(len(brief))*100/float64(len(content)))
	}
}

func TestLoader_LoadSkillReferences_Trivial_ReturnsEmpty(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/architecture.md", "# Arch\nconteudo")
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/testing.md", "# Testing\nconteudo")

	loader := NewLoader(fake)
	refs, err := loader.LoadSkillReferences(".agents/skills/go-impl", LoadOptions{
		Complexity: ComplexityTrivial,
	})
	if err != nil {
		t.Fatalf("LoadSkillReferences: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("nivel trivial: esperado 0 referencias, got %d", len(refs))
	}
}

func TestLoader_LoadSkillReferences_Standard_LoadsOnlyStandardRefs(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/error-handling.md", "# Errors\nconteudo")
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/testing.md", "# Testing\nconteudo")
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/architecture.md", "# Arch\nconteudo")
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/concurrency.md", "# Concurrency\nconteudo")

	loader := NewLoader(fake)
	refs, err := loader.LoadSkillReferences(".agents/skills/go-impl", LoadOptions{
		Complexity: ComplexityStandard,
	})
	if err != nil {
		t.Fatalf("LoadSkillReferences: %v", err)
	}

	if _, ok := refs["error-handling.md"]; !ok {
		t.Error("nivel standard: error-handling.md deve ser carregado")
	}
	if _, ok := refs["testing.md"]; !ok {
		t.Error("nivel standard: testing.md deve ser carregado")
	}
	if _, ok := refs["architecture.md"]; ok {
		t.Error("nivel standard: architecture.md NAO deve ser carregado")
	}
	if _, ok := refs["concurrency.md"]; ok {
		t.Error("nivel standard: concurrency.md NAO deve ser carregado")
	}
}

func TestLoader_LoadSkillReferences_Complex_LoadsAll(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/error-handling.md", "# Errors\nconteudo")
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/testing.md", "# Testing\nconteudo")
	writeOrFatal(t, fake, ".agents/skills/go-impl/references/architecture.md", "# Arch\nconteudo")

	loader := NewLoader(fake)
	refs, err := loader.LoadSkillReferences(".agents/skills/go-impl", LoadOptions{
		Complexity: ComplexityComplex,
	})
	if err != nil {
		t.Fatalf("LoadSkillReferences: %v", err)
	}

	if len(refs) != 3 {
		t.Errorf("nivel complex: esperado 3 referencias, got %d", len(refs))
	}
}

func TestLoader_TokenEconomy_TrivialLessThanStandardLessThanComplex(t *testing.T) {
	fake := fs.NewFakeFileSystem()
	for _, name := range []string{"error-handling.md", "testing.md", "architecture.md", "concurrency.md"} {
		writeOrFatal(t, fake, ".agents/skills/go-impl/references/"+name, "# "+name+"\n"+string(make([]byte, 1000)))
	}

	loader := NewLoader(fake)
	countTokens := func(level ComplexityLevel) int {
		refs, _ := loader.LoadSkillReferences(".agents/skills/go-impl", LoadOptions{Complexity: level})
		total := 0
		for _, v := range refs {
			total += len(v)
		}
		return total
	}

	trivial := countTokens(ComplexityTrivial)
	standard := countTokens(ComplexityStandard)
	complex := countTokens(ComplexityComplex)

	if trivial != 0 {
		t.Errorf("trivial deve ter 0 tokens de referencias, got %d", trivial)
	}
	if standard >= complex {
		t.Errorf("standard (%d) deve ter menos tokens que complex (%d)", standard, complex)
	}
}
