package scaffold

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/JailtonJunior94/ai-spec-harness/internal/fs"
	"github.com/JailtonJunior94/ai-spec-harness/internal/output"
)

// Service gera o scaffold de uma nova skill de linguagem.
type Service struct {
	fs      fs.FileSystem
	printer *output.Printer
}

func NewService(fsys fs.FileSystem, printer *output.Printer) *Service {
	return &Service{fs: fsys, printer: printer}
}

// Execute cria o scaffold de uma skill de linguagem.
func (s *Service) Execute(langName, rootDir string) error {
	skillName := langName + "-implementation"
	langUpper := strings.ToUpper(langName)
	skillDir := rootDir + "/.agents/skills/" + skillName

	s.printer.Info("Criando scaffold para skill: %s", skillName)

	// SKILL.md
	if err := s.fs.MkdirAll(skillDir + "/references"); err != nil {
		return err
	}

	skillContent := fmt.Sprintf(`---
name: %s
version: 1.0.0
description: Implementa alteracoes em codigo %s usando governanca base, convencoes de projeto e validacao proporcional. Use quando a tarefa exigir adicionar, corrigir, refatorar ou validar codigo %s. Nao use para tarefas sem codigo %s.
---

# Implementacao %s

## Procedimentos

**Etapa 1: Carregar base obrigatoria**
1. Confirmar que o contrato de carga base definido em `+"`"+`AGENTS.md`+"`"+` foi cumprido.
2. Identificar arquivo de configuracao do projeto (TODO: adaptar ao ecossistema).
3. Executar `+"`"+`bash .agents/skills/agent-governance/scripts/detect-toolchain.sh`+"`"+` para descobrir comandos de fmt, test e lint.

**Etapa 2: Selecionar apenas o contexto necessario**
1. Ler `+"`"+`references/conventions.md`+"`"+` quando a tarefa envolver estrutura de projeto, organizacao de modulos ou padroes de importacao.
2. Ler `+"`"+`references/architecture.md`+"`"+` quando a tarefa envolver layout de diretorios, injecao de dependencias ou fronteiras entre camadas.
3. Ler `+"`"+`references/testing.md`+"`"+` quando a tarefa envolver estrategia de testes, fixtures ou cobertura.

**Economia de contexto**
Se mais de 4 referencias forem necessarias para a mesma tarefa, priorizar as 3 mais criticas para o escopo da mudanca e registrar as demais como contexto nao carregado.

**Etapa 3: Modelar a alteracao**
1. Identificar o menor conjunto seguro de mudancas que satisfaz a solicitacao.
2. Mapear o comportamento afetado, as dependencias envolvidas e o risco de regressao.
3. Respeitar o estilo existente do projeto.

**Etapa 4: Implementar**
1. Editar o codigo seguindo as convencoes do contexto analisado.
2. Atualizar ou adicionar testes para toda mudanca de comportamento.
3. Adaptar exemplos ao contexto real em vez de replica-los literalmente.

**Etapa 5: Validar**
1. Seguir Etapa 4 de `+"`"+`.agents/skills/agent-governance/SKILL.md`+"`"+`.
2. Usar os comandos de fmt, test e lint detectados pelo toolchain.

## Tratamento de Erros
* Se nenhum arquivo de configuracao do projeto for encontrado, parar antes de assumir versao ou dependencias.
* Se o projeto usar monorepo, validar apenas os packages afetados pela mudanca.
* Se houver conflito entre esta skill e a governanca base, seguir a restricao mais segura e registrar a suposicao.
`, skillName, langUpper, langUpper, langUpper, langUpper)

	if err := s.fs.WriteFile(skillDir+"/SKILL.md", []byte(skillContent)); err != nil {
		return err
	}

	// Reference stubs
	refs := []string{
		"conventions", "architecture", "testing", "error-handling",
		"api", "patterns", "messaging", "observability", "concurrency",
		"resilience", "persistence", "security", "build",
		"examples-domain-flow",
	}

	for _, ref := range refs {
		title := strings.ReplaceAll(ref, "-", " ")
		title = titleCase(title)
		content := fmt.Sprintf(`> **Carregar quando:** TODO — **Escopo:** TODO

# %s

## Objetivo
TODO: descrever objetivo desta referencia para %s.

## Diretrizes
- TODO

## Proibido
- TODO
`, title, langUpper)
		if err := s.fs.WriteFile(skillDir+"/references/"+ref+".md", []byte(content)); err != nil {
			return err
		}
	}

	// Gemini command
	geminiCmd := rootDir + "/.gemini/commands/" + skillName + ".toml"
	geminiContent := fmt.Sprintf(`description = "Implementa alteracoes em codigo %s usando a habilidade canonica %s."
prompt = """
Use `+"`"+`.agents/skills/%s/SKILL.md`+"`"+` como fluxo canonico desta tarefa.
Leia os assets e references sob demanda conforme descrito no SKILL.md.
Nao invente um processo paralelo neste comando.

Aplicar a habilidade a esta solicitacao:
{{args}}
"""
`, langUpper, skillName, skillName)
	_ = s.fs.MkdirAll(rootDir + "/.gemini/commands")
	if err := s.fs.WriteFile(geminiCmd, []byte(geminiContent)); err != nil {
		return err
	}

	s.printer.Info("")
	s.printer.Info("Scaffold criado com sucesso:")
	s.printer.Info("  Skill:   %s/SKILL.md", skillDir)
	s.printer.Info("  Refs:    %s/references/ (%d stubs)", skillDir, len(refs))
	s.printer.Info("  Gemini:  %s", geminiCmd)
	s.printer.Info("")
	s.printer.Info("Passos manuais restantes:")
	s.printer.Info("  1. Preencher os stubs em references/ com conteudo real.")
	s.printer.Info("  2. Adicionar bloco de linguagem em install/upgrade.")
	s.printer.Info("  3. Atualizar AGENTS.md (Regras por Linguagem).")

	return nil
}

func titleCase(s string) string {
	prev := ' '
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(rune(prev)) || prev == '-' {
			prev = r
			return unicode.ToUpper(r)
		}
		prev = r
		return r
	}, s)
}
