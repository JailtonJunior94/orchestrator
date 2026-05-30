package agents_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/JailtonJunior94/ai-spec-harness/internal/agents"
)

type SchemaSuite struct {
	suite.Suite
}

func TestSchemaSuite(t *testing.T) {
	suite.Run(t, new(SchemaSuite))
}

func (s *SchemaSuite) TestValidateAgentFrontmatter() {
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

	scenarios := []struct {
		name    string
		content []byte
		dirName string
		expect  func(got agents.ResolvedAgent, err error)
	}{
		{
			name:    "deve produzir agente completo com frontmatter valido",
			content: validFrontmatter,
			dirName: "claude-revisor-rigoroso",
			expect: func(got agents.ResolvedAgent, err error) {
				s.NoError(err)
				s.Equal("claude-revisor-rigoroso", got.Name)
				s.False(got.Metadata.Description == "")
				s.Equal("1.0.0", got.Metadata.Version)
				s.Equal("claude", got.Runtime.IDE)
				s.Equal("high", got.Runtime.ReasoningEffort)
				s.Equal("bypass-permissions", got.Runtime.AccessMode)
				s.False(got.Prompt == "")
			},
		},
		{
			name: "deve retornar erro quando description esta ausente",
			content: []byte(`---
name: meu-agente
version: 1.0.0
---
`),
			dirName: "meu-agente",
			expect: func(got agents.ResolvedAgent, err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrFrontmatterInvalid))
			},
		},
		{
			name: "deve retornar erro quando version nao e semver",
			content: []byte(`---
name: meu-agente
description: Descricao do agente
version: nao-semver
---
`),
			dirName: "meu-agente",
			expect: func(got agents.ResolvedAgent, err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrFrontmatterInvalid))
			},
		},
		{
			name: "deve retornar erro quando runtime ide esta fora do enum",
			content: []byte(`---
name: meu-agente
description: Descricao do agente
version: 1.0.0
runtime:
  ide: vscode
---
`),
			dirName: "meu-agente",
			expect: func(got agents.ResolvedAgent, err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrFrontmatterInvalid))
			},
		},
		{
			name: "deve retornar erro quando reasoning effort esta fora do enum",
			content: []byte(`---
name: meu-agente
description: Descricao do agente
version: 1.0.0
runtime:
  ide: claude
  reasoning_effort: muito-alto
---
`),
			dirName: "meu-agente",
			expect: func(got agents.ResolvedAgent, err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrFrontmatterInvalid))
			},
		},
		{
			name: "deve retornar erro quando name diverge do diretorio",
			content: []byte(`---
name: nome-no-frontmatter
description: Descricao do agente
version: 1.0.0
---
`),
			dirName: "nome-diferente",
			expect: func(got agents.ResolvedAgent, err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrNameDirMismatch))
			},
		},
		{
			name: "deve ignorar verificacao de name quando dirName esta vazio",
			content: []byte(`---
name: qualquer-nome
description: Descricao
version: 1.0.0
---
`),
			dirName: "",
			expect: func(got agents.ResolvedAgent, err error) {
				s.NoError(err)
				s.Equal("qualquer-nome", got.Name)
			},
		},
		{
			name: "deve aceitar frontmatter sem bloco runtime",
			content: []byte(`---
name: agente-simples
description: Agente sem runtime
version: 2.0.1
---
`),
			dirName: "agente-simples",
			expect: func(got agents.ResolvedAgent, err error) {
				s.NoError(err)
				s.Empty(got.Runtime.IDE)
			},
		},
		{
			name: "deve aceitar version com prefixo v e prerelease",
			content: []byte(`---
name: agente-prerelease
description: Agente com versao pre-release
version: v1.0.0-beta.1
---
`),
			dirName: "agente-prerelease",
			expect: func(got agents.ResolvedAgent, err error) {
				s.NoError(err)
				s.Equal("v1.0.0-beta.1", got.Metadata.Version)
			},
		},
		{
			name: "deve retornar erro quando access mode e invalido",
			content: []byte(`---
name: meu-agente
description: Descricao
version: 1.0.0
runtime:
  access_mode: admin
---
`),
			dirName: "meu-agente",
			expect: func(got agents.ResolvedAgent, err error) {
				s.Error(err)
				s.True(errors.Is(err, agents.ErrFrontmatterInvalid))
			},
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			got, err := agents.NewCatalog().ValidateAgentFrontmatter(scenario.content, scenario.dirName)

			scenario.expect(got, err)
		})
	}
}

func (s *SchemaSuite) TestValidateAgentFrontmatterAceitaTodosRuntimeIDE() {
	scenarios := []struct {
		name string
		ide  string
	}{
		{name: "deve aceitar claude", ide: "claude"},
		{name: "deve aceitar codex", ide: "codex"},
		{name: "deve aceitar gemini", ide: "gemini"},
		{name: "deve aceitar copilot", ide: "copilot"},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			content := []byte("---\nname: agente-" + scenario.ide + "\ndescription: Agente\nversion: 1.0.0\nruntime:\n  ide: " + scenario.ide + "\n---\n")

			_, err := agents.NewCatalog().ValidateAgentFrontmatter(content, "agente-"+scenario.ide)

			s.NoError(err)
		})
	}
}

func (s *SchemaSuite) TestExtractPromptBody() {
	scenarios := []struct {
		name    string
		content []byte
		want    string
	}{
		{
			name: "deve extrair corpo apos frontmatter",
			content: []byte(`---
name: agente
description: Desc
version: 1.0.0
---

Voce e um agente.
- Priorize invariantes.
`),
			want: "Voce e um agente.\n- Priorize invariantes.",
		},
		{
			name: "deve retornar prompt vazio quando frontmatter nao tem corpo",
			content: []byte(`---
name: agente
description: Desc
version: 1.0.0
---
`),
			want: "",
		},
	}

	for _, scenario := range scenarios {
		s.Run(scenario.name, func() {
			got, err := agents.NewCatalog().ValidateAgentFrontmatter(scenario.content, "agente")

			s.NoError(err)
			s.Equal(scenario.want, got.Prompt)
		})
	}
}
