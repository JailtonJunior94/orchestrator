# Relatório de Orquestração — PRD hooks-canonicos-vendor-neutral

Status: **concluído — 13/13 tarefas `done`**.

Este relatório substitui `_orchestration_report.partial.md` (removido), que registrava apenas o
estado bloqueado inicial de 2026-09-19 (pré-condição externa da tarefa 1.0, resolvida antes de
qualquer trabalho de implementação começar). O PRD foi executado do início ao fim depois disso, em
tarefas sequenciais/paralelas conforme `tasks.md`, e fechado pela tarefa 13.0 nesta sessão.

## Tabela final

| # | Título | Status | Commit de fechamento |
|---|---|---|---|
| 1.0 | Inventário e classificação de hooks com gate anti-órfão | done | (comitado junto com 2.0/3.0 em `55f9f13`) |
| 2.0 | Parsing de payload sem truncamento e negação por ausência de alvo | done | `55f9f13` |
| 3.0 | Instrumentação de pré-requisito: tmp em gates, exaustividade de linguagens e prova por célula | done | (comitado junto com 1.0/2.0 em `55f9f13`) |
| 4.0 | Contrato canônico de hooks em `internal/hookcontract` | done | `25ce1c0` |
| 5.0 | Projeções dos dois modelos de ponto, correção de `Kind` e erros não descartados | done | `ed8ebd8` |
| 6.0 | Harness contract aplicado e gate de operação git derivado da policy | done | `6abf111` |
| 7.0 | Igualdade entre stacks: dotnet completo e java como cidadão de primeira classe | done | `ba321a4` |
| 8.0 | Adapters dos quatro provedores com SessionStart e SessionEnd reais | done | `aaee239` |
| 9.0 | Matriz de capabilities por evento e família com doctor de hooks | done | `0b67199` |
| 10.0 | Quality gate disparado por evento com seleção por risco e deduplicação | done | `33e7e4a` |
| 11.0 | Evidence gate, checkpoint atômico, detecção de corrupção e sanitização | done | `2360bd4` |
| 12.0 | Telemetria comum, ablation, timeout e guarda de recursão | done | `7e2591e` |
| 13.0 | Conformidade cross-provider, não-regressão e documentação | done | pendente de commit (ver "Selo de evidência" abaixo) |

## O que a tarefa 13.0 entregou

- Suíte de conformidade RF-61 completa: de 4 para 14 cenários contra os quatro provedores, com os
  dez novos em `tests/integration/conformance_suite_rf61_test.go` — seis com dispatch nativo real e
  distinto por provedor (evidência ausente/inválida via `validate-session-end.sh` + plugin OpenCode,
  adapter corrompido, bypass de comando), cinco validando núcleo Go provider-invariante por
  construção (quality-gate, checkpoint, telemetria, evento desconhecido), e capability não suportada
  discriminando por célula real da matriz gerada.
- `internal/conformance.RF61Scenarios()` como catálogo explícito e distinto de `Manifest()`, com
  teste de regressão que impede colisão silenciosa entre os dois.
- RF-62 (nunca comparação textual entre CLIs) e gate-of-the-gate (mutação/par discriminante que
  deixa cada cenário vermelho) verificados cenário a cenário.
- RF-63: a cadeia de release já bloqueia em regressão de policy crítica via `make integration`/
  `make check-capability-matrix-sync`, ambos gates de CI existentes — os dez cenários novos entram
  nessa mesma cadeia.
- RF-74: gate final completo (`make test lint vet coverage` + todos os `check-*` + `test-hooks` +
  `test-validators` + `integration`) verde, executado três vezes nesta sessão (antes do bugfix,
  depois do bugfix, e como fechamento final) — sempre coverage total 82.0%, lint "0 issues.", zero
  falha real. Diff de comportamento observável dos hooks `KEEP` da tarefa 1.0: vazio (nenhum arquivo
  de hook shell tocado).
- RF-75: `docs/hooks-canonicos.md` criado; `docs/evidence-gates.md`, `docs/degradation-matrix.md`,
  `docs/troubleshooting.md` e `docs/runtime-claude-capabilities.md` atualizados; afirmação falsa em
  `CLAUDE.md:20` corrigida (o hook `validate-governance.sh` informa após a edição, não bloqueia
  antes dela) com `AGENTS.md` mantido consistente.

## Ciclo de revisão da tarefa 13.0

Rodada 1 do subagent `reviewer`: **REJECTED**, 2 achados `[HIGH]` — (a) seis dos dez cenários novos
não dispachavam de fato pelos quatro provedores (loop decorativo, documentação overclaimando
"contra os quatro provedores" para famílias sem wiring nativo), com foco especial nos cenários de
evidência ausente/inválida, que ignoravam infraestrutura de dispatch real já existente
(`validate-session-end.sh`); (b) ausência do `13.0_execution_report.md`. Ambos corrigidos na mesma
sessão: cenários 06/07 reescritos para dispatch real por provedor; os cinco cenários núcleo-apenas
renomeados e a documentação reescrita para declarar explicitamente essa distinção; relatório de
execução criado com evidência física completa (`evidence/task-13.0-hooks-canonicos-vendor-neutral/`,
`13.0_execution_result.json` validado por `ai-spec validate-result execution --verify-physical`).

## Débitos técnicos herdados, avaliados e conscientemente não corrigidos por esta tarefa

Nenhum é regressão introduzida por 13.0 nem requisito de RF-61/62/63/74/75; todos já estavam
registrados nos relatórios das tarefas de origem:

- **9.0**: `checkHookContractVersion` usa contagem de `Coverage()` como proxy de versão;
  `checkpoint` genuinamente wired para claude/copilot sem teste de dispatch fim-a-fim (mantido
  `unsupported` com razão explícita, decisão consciente preservada pelos cenários 8/9 da 13.0).
- **10.0**: wiring de produção de `qualitygate` sempre usa `TaskType`/`Risk` default, nunca
  metadado real da tarefa.
- **11.0**: risco residual de conteúdo em `sdd-state.json`/`validate-sdd` para projetos que fecham
  tarefas via a skill leve sem o envelope completo — confirmado nesta sessão que este próprio PRD
  não usa `validate-sdd` no seu gate (`sdd-state.json` ausente, comportamento pré-existente).
- **12.0**: `TestHookTelemetry_FailureDoesNotPropagateOrCountAsRetry` é tautológico; a garantia real
  vem da assinatura dos métodos de telemetria, não de um teste contra o dispatcher de produção.
- **[LOW, achado da rodada 1 de revisão da 13.0]**: cabeçalho de `.claude/hooks/validate-governance.sh`
  (e seus sete mirrors) mantém comentário desatualizado ("bloqueia a edição"). Não corrigido nesta
  tarefa por já violar R-STYLE-001.2 de forma pré-existente e generalizada no bloco inteiro; corrigir
  apenas a linha citada seria inconsistente com a regra hard. Registrado para uma tarefa futura que
  já esteja tocando esses oito arquivos.

## Selo de evidência (RF-14)

O harness não cria commits. `13.0_execution_result.json` está pronto e validado
(`ai-spec validate-result execution --verify-physical` = OK), mas o selo (`ai-spec seal-evidence`)
só é possível depois que o commit desta tarefa existir. **Selo pendente** — a ser aplicado pelo
coordenador após o commit manual, com:

```
ai-spec seal-evidence .specs/prd-hooks-canonicos-vendor-neutral/13.0_execution_result.json \
  --prd-dir .specs/prd-hooks-canonicos-vendor-neutral --commit <sha-do-commit>
```

## Gate de não-regressão do repositório inteiro (RF-74)

Executado três vezes nesta sessão, sempre verde:

```
make test lint vet coverage \
     check-skills-sync check-hooks-sync check-scripts-sync \
     check-policies-sync check-capability-matrix-sync \
     check-spec-paths check-mocks test-hooks test-validators integration
```

Coverage total: 82.0% (gate 75%). Lint: 0 issues. `bash scripts/check-hooks-inventory.sh`: sem
saída (aprovado). `go test -tags=hook_dispatch_proof,capability_instrumentation ./internal/capability/...`:
10/10 pass.
