# Tarefa 8.0: Guia de Instalação Universal + atualização de governança

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Documentar o comportamento final da Fundação Portátil: criar o **Guia de Instalação Universal**
(bootstrap portátil <30s, detecção automática, escopo global/projeto, `verify`) e atualizar os
artefatos de governança (AGENTS.md/CLAUDE.md e `docs/`) com a **hierarquia de configuração** e a
**precedência** determinística. Entregável do PRD ("Guia de Instalação Universal" + docs da camada de
config). Conforme [ADR-016](adr-016-config-hierarquico-universal.md) e
[ADR-019](adr-019-instalador-portatil-detect-verify.md).

<requirements>
- Guia de Instalação Universal: procedimento de bootstrap em qualquer codebase (sem `--tools`), escopo `--global`, `verify`, modos symlink/copy.
- Documentar a hierarquia `~/.aispec/config.yaml` (global) + projeto + flags e a precedência `flags > workspace > global > built-in`.
- Atualizar `AGENTS.md` (seção de config/comandos) e `CLAUDE.md` para refletir o comportamento implementado.
- Documentação consistente com o código entregue (sem features não implementadas).
- Atualizar a tabela de ADRs (AGENTS.md/CLAUDE.md) com ADR-016..019.
</requirements>

## Subtarefas

- [ ] 8.1 Escrever o Guia de Instalação Universal em `docs/` (bootstrap portátil, detecção, escopo, verify).
- [ ] 8.2 Documentar a hierarquia/precedência de configuração em `docs/` e referenciar nos governance files.
- [ ] 8.3 Atualizar `AGENTS.md` (config, comandos `install`/`verify`, tabela de ADRs 016–019).
- [ ] 8.4 Atualizar `CLAUDE.md` (seção de config e capacidades) de forma consistente.
- [ ] 8.5 Revisar consistência doc↔código (nenhuma capacidade documentada além do implementado).

## Detalhes de Implementação

Ver `techspec.md` §"Monitoramento e Observabilidade" e §"Conformidade com Padrões";
[ADR-016](adr-016-config-hierarquico-universal.md) §"Impacto em Documentação" e
[ADR-019](adr-019-instalador-portatil-detect-verify.md) §"Impacto em Documentação". Depende das
Tarefas 6.0 (instalador/verify) e 7.0 (paridade) para documentar comportamento final.

## Critérios de Sucesso

- Guia permite a um adotante externo instalar em codebase novo seguindo apenas o documento.
- Hierarquia e precedência de config documentadas e coerentes com o `config.Resolver` (Tarefa 1.0).
- `AGENTS.md`/`CLAUDE.md` atualizados; tabela de ADRs inclui 016–019.
- `make lint` (e validações de skills/docs aplicáveis) verdes; sem referência a feature inexistente.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

- `analyze-project` — gera/atualiza arquivos de governança (CLAUDE.md/AGENTS.md) e documentação orientada a IA do projeto.

## Testes da Tarefa

- [ ] Testes unitários: não aplicável (tarefa de documentação); validar via lint/validators de skills e revisão.
- [ ] Testes de integração: opcional — seguir o próprio Guia num `t.TempDir()` para confirmar que os passos funcionam.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `docs/` (novo Guia de Instalação Universal + doc da hierarquia de config)
- `AGENTS.md` (config, comandos, tabela de ADRs)
- `CLAUDE.md` (config e capacidades)
- `.specs/prd-fundacao-portatil/adr-016-*.md`, `adr-019-*.md` (referências)
