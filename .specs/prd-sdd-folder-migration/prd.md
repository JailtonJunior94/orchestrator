<!-- spec-version: 1 -->

# PRD: Migracao Mandatoria do Root SDD para `.specs`

## Objetivo

Migrar a convencao de diretorio raiz dos artefatos SDD do projeto do root historico anterior para `.specs/`, em hard break, cobrindo runtime, scripts, hooks, skills, assets embutidos, documentacao, testes e artefatos versionados.

## Contexto

O projeto usava um root historico nao oculto para PRDs, techspecs, ADRs, tarefas, checkpoints e evidencias. A nova convencao deve tratar esses artefatos como infraestrutura de projeto em uma dot-folder (`.specs/`), sem manter caminho legado como default ou fallback silencioso.

## Requisitos Funcionais

- RF-01: O diretorio versionado historico deve ser substituido por `.specs/`, preservando conteudo, historico semantico dos artefatos e relacoes de hash.
- RF-02: O default de configuracao do runtime deve passar de `tasks` para `.specs`, mantendo as chaves publicas de override `tasks_root` e `AI_TASKS_ROOT`.
- RF-03: Hooks, scripts e comandos de validacao devem reconhecer `.specs/prd-*` como local canonico dos bundles SDD.
- RF-04: Skills canonicas, mirrors por ferramenta e assets embutidos devem instruir criacao, leitura e execucao de artefatos em `.specs/`.
- RF-05: Documentacao, exemplos, ADR links, research notes e testdata devem apontar para `.specs/` quando se referirem ao root SDD.
- RF-06: A migracao deve remover referencias legadas ao caminho historico no contexto SDD; a palavra "tasks" permanece valida para o conceito de tarefa, comando/skill e arquivo `tasks.md`.
- RF-07: `check-spec-drift`, `sync-spec-hash` e `task-loop` devem continuar funcionando quando chamados com `.specs/prd-*/tasks.md` ou `.specs/prd-*`.

## Fora de Escopo

- Renomear o arquivo `tasks.md`.
- Renomear comandos ou skills como `create-tasks`, `execute-task`, `execute-all-tasks` ou `task-loop`.
- Criar fallback automatico para o root historico anterior.
- Alterar o algoritmo de spec-hash.

## Criterios de Sucesso

- A busca por caminhos SDD legados nao encontra referencias sem justificativa.
- Os testes de configuracao esperam `.specs` como default.
- Os hooks de spec-drift usam `.specs/prd-*`.
- Os mirrors de skills/hooks e o bundle embedded estao sincronizados.
- `ai-spec check-spec-drift .specs/prd-sdd-folder-migration/tasks.md` passa.

## Suposicoes e Questoes em Aberto

- A interface publica de override continua usando os nomes historicos `tasks_root` e `AI_TASKS_ROOT` para evitar quebra desnecessaria de configuracao.
- O hard break se aplica ao default e aos caminhos documentados; usuarios ainda podem passar qualquer caminho explicito existente para comandos que aceitam path.
