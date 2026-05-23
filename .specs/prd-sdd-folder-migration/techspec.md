<!-- spec-hash-prd: 800855c57694e50b00e3f35d5953d7f5de3b5678e7cd429c857c62b1ec8b83cb -->

# TechSpec: Migracao Mandatoria do Root SDD para `.specs`

## Visao Tecnica

A migracao troca o root canonico dos artefatos SDD do root historico anterior para `.specs/` em todas as superficies versionadas. A mudanca e majoritariamente de paths e defaults, com impacto transversal em runtime Go, scripts shell, hooks, skills, assets embutidos, documentacao e fixtures de teste.

## Decisoes

- D-01: O diretorio fisico historico sera renomeado para `.specs/`.
- D-02: `config.DefaultRuntime().TasksRoot` passara a retornar `.specs`.
- D-03: `tasks_root` e `AI_TASKS_ROOT` permanecem como nomes publicos de configuracao/ambiente, mas com default `.specs`.
- D-04: Nao havera fallback automatico para o root historico anterior; caminhos antigos falham como qualquer path inexistente.
- D-05: `tasks.md` continua sendo o arquivo canonico de decomposicao de tarefas.

## Impactos por Subsistema

- Runtime/config: atualizar defaults, comentarios e testes de `internal/config`.
- Runtime/probe: atualizar paths de ADRs para `.specs/adr/...`.
- Task-loop/hooks: atualizar exemplos, mensagens e gates que montam paths de bundles.
- Skills/assets: atualizar instrucoes canonicas e mirrors sincronizados.
- Docs/testdata: trocar exemplos e links SDD para `.specs/`.
- Repo state: mover todos os bundles e ADRs versionados para `.specs/`.

## Estrategia de Implementacao

1. Criar o pacote PRD-first desta migracao em `.specs/prd-sdd-folder-migration/`.
2. Renomear o root historico para `.specs/`, preservando conteudo existente e mantendo o novo pacote.
3. Aplicar substituicao contextual do caminho legado para `.specs/` em referencias SDD.
4. Ajustar defaults e testes que esperam o root antigo.
5. Rodar sincronizadores de skills/hooks para propagar mirrors.
6. Validar com testes direcionados e gates globais proporcionais.

## Testes

- Unitarios de `internal/config`, `internal/runtime/probe`, hooks runtime e task-loop.
- Scripts de sync/check de skills e hooks.
- `make test`, `make vet`, `make lint` e, se viavel, `make integration`.
- Busca final por referencias legadas ao root historico.

## Riscos

- Links Markdown relativos podem quebrar apos a mudanca fisica.
- Mirrors read-only podem exigir sincronizacao em vez de edicao manual.
- Substituicoes textuais indiscriminadas podem trocar usos conceituais da palavra "tasks"; por isso a migracao deve focar em referencias de caminho SDD.
