<!-- spec-hash-prd: 36b6efdd09b6f3fae7881c3bd21ab3eb3af77a4869f5d25ad14ed707923bb18d -->
# Especificação Técnica — SDD robusto e verificável

## Resumo Executivo

O núcleo será um pacote Go de estado SDD, consumido por comandos Cobra e por adaptadores shell
finos. O estado é a fonte operacional; Markdown é uma projeção humana. Toda transição é validada,
escrita atomicamente e registrada em eventos append-only. O modo estrito recusa dados legados ou
evidências incompletas; a compatibilidade legada é somente leitura/warning durante duas versões.

## Componentes

| Componente | Responsabilidade |
|---|---|
| `internal/sdd` | schema, validação, transições, hashes, escrita atômica e eventos |
| `internal/sdd/tasks` | parser estrutural de Markdown, DAG, ownership e cobertura |
| `internal/sdd/orchestrator` | plano idempotente, lock de escritor, tentativas e isolamento |
| `internal/sdd/review` | snapshot cumulativo e contrato determinístico de revisão |
| `cmd/ai_spec_harness` | `validate`, `approve`, `invalidate`, `orchestrate` e migração |
| `.agents/hooks` | adaptadores que chamam CLI; nunca interpretam YAML/Markdown como verdade |
| `evals/sdd` | corpus adversarial e resultados agregados |

## Contratos

`sdd-state.json` (schema v2) contém `schema_version`, `run_id`, artefatos e hashes aprovados,
requisitos RF/NFR, DAG, aprovações, tarefas, referências de evidência e eventos. Estados canônicos:
`draft`, `approved`, `stale`, `executing`, `blocked`, `needs_input`, `failed`, `done`.

`execution-result.json` e `review-result.json` incluem: `schema_version`, `run_id`, `task_id`,
`attempt`, `status`, `base_sha`, `patch_sha256`, `final_state_sha256`, comandos de teste com
exit-code/output digest, critérios com prova individual, evidências e `verdict`. A revisão aceita
somente `approved`, `changes_requested` ou `needs_input`; crítica/alta implica bloqueio.

## Fluxo e Transições

1. `validate` lê Markdown, constrói o modelo estrutural e falha em esquema, hash, vínculo,
   ownership ou DAG inválido.
2. `approve <artifact>` calcula digest e promove somente artefato validado; PRD aprova techspec,
   techspec aprova tasks.
3. `invalidate --from` marca os descendentes aprovados como `stale`, sem reescrever hashes.
4. `orchestrate` obtém lock exclusivo, cria tentativa, persiste evento, executa apenas tarefas
   prontas e confirma `done` depois de validar resultado e revisão.
5. Crash entre transições deixa evento/tentativa recuperável; retomada reconcilia o último evento,
   nunca assume `done`.

## Segurança e Concorrência

Toda referência de arquivo passa por `filepath.EvalSymlinks`/`Abs` e `filepath.Rel`; o destino deve
ser relativo, não conter `..` e permanecer dentro da raiz resolvida. Resultados e relatórios são
paths declarados, não interpretados por shell. Escrita usa arquivo temporário no mesmo diretório,
`fsync` quando disponível e rename. Um lock por PRD é o escritor global.

O orquestrador calcula `parallel_group` somente para nós sem dependência mútua e ownership de
arquivos disjunto. Escrita paralela exige worktree isolado e capacidade detectada. Sem isolador,
ou com conflito/ownership desconhecido, a tarefa é sequencial ou `blocked` conforme a política.

## Rastreamento e Diff

O snapshot de patch é gerado a partir de `git diff --binary HEAD`, staged diff e arquivos novos
enumerados explicitamente. O contrato armazena SHA da base, SHA-256 do patch e SHA-256 do estado
final relevante. Commit SHA pode ser adicionado, mas não é condição para prova.

## Revisão e Bugfix

O revisor recebe somente artefatos aprovados, tarefa, snapshot cumulativo e testes; é read-only e
não recebe o raciocínio do executor. A linguagem é detectada pelos arquivos alterados. Bugfix
normaliza entrada, exige RF/contrato/issue de origem, teste de regressão que falha antes e passa
depois, e uma revisão fresca por `run_id`.

## Matriz de Rastreabilidade

| RF/NFR | Decisão | Tarefa | Teste |
|---|---|---|---|
| RF-01..05 | contratos e hooks finos | 1.0, 2.0 | shell adversarial + Go unit |
| RF-06..08 | estado e CLI | 3.0 | unit + integração de comandos |
| RF-09..10 | parser estrutural e DAG | 4.0 | table/fuzz |
| RF-11..14 | orquestrador seguro | 5.0, 6.0 | crash/race/integration |
| RF-15..16 | revisão e bugfix | 7.0 | snapshots multi-language |
| RF-17 | skills e documentação | 8.0 | sync/paridade |
| RF-18, NFR-04 | evals e CI | 9.0 | corpus e workflow |
| NFR-01..05 | compatibilidade e qualidade | 3.0..10.0 | e2e/CI |

## Testes

Unidades table-driven usam FakeFileSystem; testes de filesystem, worktree e crash usam `t.TempDir()`
e build tag `integration`. Fuzz cobre paths, schema e DAG. A CI executa validadores shell, `go test`,
integração, race e smoke adapters nas três plataformas. Nenhum teste chama runtime ou rede reais.

## Migração e Rollback

`migrate-sdd --dry-run` mostra arquivos e não altera estado; a migração real requer confirmação.
Leitores aceitam Markdown v1 por duas versões menores em warning-only. O rollback remove somente
estado v2 criado pela execução identificada, jamais reescreve Markdown aprovado.

## ADRs

- [ADR-020](adr-020-sdd-state-source-of-truth.md): estado versionado como fonte operacional.
- [ADR-021](adr-021-sdd-single-writer-isolation.md): escritor único e isolamento obrigatório.

## Arquivos Relevantes

- `internal/specdrift`, `internal/taskloop`, `internal/fs`, `internal/runtime`, `cmd/ai_spec_harness`
- `.agents/{hooks,scripts,skills}`, mirrors `.claude`, `.github`, `internal/embedded/assets`
- `scripts/test-validators.sh`, `scripts/test-hooks.sh`, `Makefile`, `.github/workflows`
