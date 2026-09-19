# Estilo de Codigo — Idioma e Comentarios

- Rule ID: R-STYLE-001
- Severidade: hard
- Escopo: todo o codigo-fonte do repositorio (`.go`, scripts `.sh`, `.py`, `.js` e demais linguagens de implementacao).

## Objetivo

Padronizar idioma e ausencia de comentarios em todo codigo produzido ou editado por agentes de IA.
Regra mandatoria e inegociavel, definida pelo dono do repositorio.

## R-STYLE-001.1 — Codigo em ingles

Todo codigo-fonte e escrito em ingles:

- Identificadores: tipos, structs, interfaces, funcoes, metodos, variaveis, campos, constantes, enums.
- Nomes de pacote e nomes de arquivo de codigo.
- Mensagens de erro (`fmt.Errorf`, `errors.New`, panics), textos de log e strings internas.
- Nomes de teste (`TestXxx`, subtests `t.Run("...")`) e nomes de fixture.

Excecao unica: string que e contrato externo verificavel (saida CLI ja publicada, chave de
protocolo, formato consumido por terceiro) permanece no idioma do contrato. A excecao exige
justificativa explicita no relatorio de execucao.

Termo de dominio em PT-BR vindo de PRD, techspec, ADR ou modelo de dominio nunca migra
literalmente para identificador de codigo. Traduzir para ingles ao implementar, mesmo quando o
artefato de contrato usa o termo em portugues (ex.: `Fato` → `Fact`, `Pagina` → `Page`,
`Durabilidade` → `Durability`, `ChaveSemantica` → `SemanticKey`, `BlocoHumano` → `HumanBlock`).
Nome de arquivo de codigo segue a mesma regra (`fato.go` → `fact.go`). A rastreabilidade
PRD-para-codigo fica no relatorio de execucao e no mapeamento de RF, nao no nome do identificador.

## R-STYLE-001.2 — Zero comentarios no codigo

Nenhum comentario em codigo criado ou editado:

- Sem comentarios de linha (`//`), de bloco (`/* */`), de shell/python (`#`) fora de shebang.
- Sem doc-comments (`// FuncName ...`), sem TODO/FIXME/NOTE, sem codigo comentado.
- Shebang (`#!/usr/bin/env bash`) e diretivas obrigatorias da linguagem (`//go:embed`,
  `//go:build`, `# -*- coding: -*-`) nao sao comentarios e sao permitidos.
- Ao editar arquivo existente que contenha comentarios nas linhas tocadas, remover esses
  comentarios. Nao e obrigatorio varrer o arquivo inteiro, apenas o diff.

O codigo deve ser autoexplicativo: nomes claros, funcoes pequenas, early return. Quando um
comentario pareceria necessario, refatorar para eliminar a necessidade.

## R-STYLE-001.3 — Sem prefixo underline em identificadores Go

Go nao usa `_` como prefixo de identificador para indicar visibilidade. Proibido em qualquer
identificador de codigo Go (var, const, func, tipo, campo, parametro, metodo):

- Simbolo privado ao pacote: primeira letra minuscula (`verdictPrefixes`, `newCycle`).
- Simbolo exportado: primeira letra maiuscula (`VerdictPrefixes`, `NewCycle`).
- `_` isolado como blank identifier (`_, err :=`, `for _, x := range`) e permitido.
- `_` no meio de nome so em nome de teste/subtest quando a convencao ja exige (`Test_Foo`,
  arquivos `_test.go`, sufixos `_linux.go`).

Vale tambem para nomes que hoje existem no repo com esse prefixo: renomear ao tocar.

## Aplicacao

- Vale para toda skill de implementacao (`go-implementation`, `python-implementation`,
  `node-implementation`, `dotnet-csharp-implementation`) e para `execute-task` /
  `execute-all-tasks`.
- Sobrepoe qualquer convencao anterior de PT-BR em comentarios ou de doc-comments.
- Artefatos `.md`, relatorios de execucao, ADRs, PRDs, techspecs e changelog continuam em PT-BR.

## Precedencia

Integra a governanca transversal de `.claude/rules/` (nivel 1 de `governance.md`). Conflito com
guia de estilo de linguagem: prevalece esta regra por ser `hard` e convencao explicita local.
