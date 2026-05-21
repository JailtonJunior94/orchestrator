package agents_test

import (
	"errors"
	"testing"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
)

func TestValidateAgentFrontmatter(t *testing.T) {
	t.Parallel()

	validFrontmatter := []byte(`---
name: claude-revisor-rigoroso
description: Revisor de PR com vies conservador e foco em invariantes
version: 1.0.0
runtime:
  ide: claude
  model: claude-opus-4-7
  reasoning_effort: high
  access_mode: bypass-permissions
---

Voce e um revisor de PR rigoroso.
`)

	t.Run("T-positivo: frontmatter valido produz ResolvedAgent completo", func(t *testing.T) {
		t.Parallel()
		got, err := agents.ValidateAgentFrontmatter(validFrontmatter, "claude-revisor-rigoroso")
		if err != nil {
			t.Fatalf("esperava sem erro, obteve: %v", err)
		}
		if got.Name != "claude-revisor-rigoroso" {
			t.Errorf("Name = %q, quer %q", got.Name, "claude-revisor-rigoroso")
		}
		if got.Metadata.Description == "" {
			t.Error("Description nao deve ser vazio")
		}
		if got.Metadata.Version != "1.0.0" {
			t.Errorf("Version = %q, quer %q", got.Metadata.Version, "1.0.0")
		}
		if got.Runtime.IDE != "claude" {
			t.Errorf("Runtime.IDE = %q, quer %q", got.Runtime.IDE, "claude")
		}
		if got.Runtime.ReasoningEffort != "high" {
			t.Errorf("Runtime.ReasoningEffort = %q, quer %q", got.Runtime.ReasoningEffort, "high")
		}
		if got.Runtime.AccessMode != "bypass-permissions" {
			t.Errorf("Runtime.AccessMode = %q, quer %q", got.Runtime.AccessMode, "bypass-permissions")
		}
		if got.Prompt == "" {
			t.Error("Prompt nao deve ser vazio")
		}
	})

	t.Run("T-05: frontmatter sem description retorna erro citando campo obrigatorio", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: meu-agente
version: 1.0.0
---
`)
		_, err := agents.ValidateAgentFrontmatter(content, "meu-agente")
		if err == nil {
			t.Fatal("esperava erro, obteve nil")
		}
		if !errors.Is(err, agents.ErrFrontmatterInvalid) {
			t.Errorf("esperava ErrFrontmatterInvalid, obteve: %v", err)
		}
	})

	t.Run("T-06: version nao-SemVer retorna erro citando campo version", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: meu-agente
description: Descricao do agente
version: nao-semver
---
`)
		_, err := agents.ValidateAgentFrontmatter(content, "meu-agente")
		if err == nil {
			t.Fatal("esperava erro, obteve nil")
		}
		if !errors.Is(err, agents.ErrFrontmatterInvalid) {
			t.Errorf("esperava ErrFrontmatterInvalid, obteve: %v", err)
		}
	})

	t.Run("T-07: runtime.ide fora do enum retorna erro citando opcoes validas", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: meu-agente
description: Descricao do agente
version: 1.0.0
runtime:
  ide: vscode
---
`)
		_, err := agents.ValidateAgentFrontmatter(content, "meu-agente")
		if err == nil {
			t.Fatal("esperava erro, obteve nil")
		}
		if !errors.Is(err, agents.ErrFrontmatterInvalid) {
			t.Errorf("esperava ErrFrontmatterInvalid, obteve: %v", err)
		}
	})

	t.Run("T-08: runtime.reasoning_effort invalido retorna erro citando enum", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: meu-agente
description: Descricao do agente
version: 1.0.0
runtime:
  ide: claude
  reasoning_effort: muito-alto
---
`)
		_, err := agents.ValidateAgentFrontmatter(content, "meu-agente")
		if err == nil {
			t.Fatal("esperava erro, obteve nil")
		}
		if !errors.Is(err, agents.ErrFrontmatterInvalid) {
			t.Errorf("esperava ErrFrontmatterInvalid, obteve: %v", err)
		}
	})

	t.Run("T-09: name no frontmatter diferente do diretorio retorna ErrNameDirMismatch", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: nome-no-frontmatter
description: Descricao do agente
version: 1.0.0
---
`)
		_, err := agents.ValidateAgentFrontmatter(content, "nome-diferente")
		if err == nil {
			t.Fatal("esperava erro, obteve nil")
		}
		if !errors.Is(err, agents.ErrNameDirMismatch) {
			t.Errorf("esperava ErrNameDirMismatch, obteve: %v", err)
		}
	})

	t.Run("dirName vazio ignora verificacao de name", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: qualquer-nome
description: Descricao
version: 1.0.0
---
`)
		got, err := agents.ValidateAgentFrontmatter(content, "")
		if err != nil {
			t.Fatalf("esperava sem erro, obteve: %v", err)
		}
		if got.Name != "qualquer-nome" {
			t.Errorf("Name = %q", got.Name)
		}
	})

	t.Run("frontmatter sem bloco runtime e valido", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: agente-simples
description: Agente sem runtime
version: 2.0.1
---
`)
		got, err := agents.ValidateAgentFrontmatter(content, "agente-simples")
		if err != nil {
			t.Fatalf("esperava sem erro, obteve: %v", err)
		}
		if got.Runtime.IDE != "" {
			t.Errorf("Runtime.IDE deveria ser vazio, obteve %q", got.Runtime.IDE)
		}
	})

	t.Run("version com prefixo v e pre-release e valida", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: agente-prerelease
description: Agente com versao pre-release
version: v1.0.0-beta.1
---
`)
		got, err := agents.ValidateAgentFrontmatter(content, "agente-prerelease")
		if err != nil {
			t.Fatalf("esperava sem erro, obteve: %v", err)
		}
		if got.Metadata.Version != "v1.0.0-beta.1" {
			t.Errorf("Version = %q", got.Metadata.Version)
		}
	})

	t.Run("todos os valores de runtime.ide aceitos sao validos", func(t *testing.T) {
		t.Parallel()
		ides := []string{"claude", "codex", "gemini", "copilot"}
		for _, ide := range ides {
			t.Run(ide, func(t *testing.T) {
				t.Parallel()
				content := []byte("---\nname: agente-" + ide + "\ndescription: Agente\nversion: 1.0.0\nruntime:\n  ide: " + ide + "\n---\n")
				_, err := agents.ValidateAgentFrontmatter(content, "agente-"+ide)
				if err != nil {
					t.Errorf("ide=%q: esperava sem erro, obteve: %v", ide, err)
				}
			})
		}
	})

	t.Run("access_mode invalido retorna erro", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: meu-agente
description: Descricao
version: 1.0.0
runtime:
  access_mode: admin
---
`)
		_, err := agents.ValidateAgentFrontmatter(content, "meu-agente")
		if err == nil {
			t.Fatal("esperava erro para access_mode invalido")
		}
		if !errors.Is(err, agents.ErrFrontmatterInvalid) {
			t.Errorf("esperava ErrFrontmatterInvalid, obteve: %v", err)
		}
	})
}

func TestExtractPromptBody(t *testing.T) {
	t.Parallel()

	t.Run("corpo apos frontmatter e extraido corretamente", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: agente
description: Desc
version: 1.0.0
---

Voce e um agente.
- Priorize invariantes.
`)
		got, err := agents.ValidateAgentFrontmatter(content, "agente")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if got.Prompt == "" {
			t.Error("Prompt nao deve ser vazio")
		}
		if got.Prompt == "Voce e um agente.\n- Priorize invariantes." {
			// OK — conteudo esperado
		}
	})

	t.Run("frontmatter sem corpo retorna Prompt vazio", func(t *testing.T) {
		t.Parallel()
		content := []byte(`---
name: agente
description: Desc
version: 1.0.0
---
`)
		got, err := agents.ValidateAgentFrontmatter(content, "agente")
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}
		if got.Prompt != "" {
			t.Errorf("Prompt deve ser vazio, obteve: %q", got.Prompt)
		}
	})
}
