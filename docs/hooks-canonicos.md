# Hooks canonicos, deterministicos e vendor-neutral

Documento gerado pela tarefa 13.0 do PRD `hooks-canonicos-vendor-neutral` (RF-75). Descreve os cinco
eventos canonicos, as cinco familias de hooks, as policies, os gates, a matriz de capabilities por
provedor, as limitacoes por provedor e por stack, e o procedimento de troubleshooting. Racional
detalhado de cada gate de evidencia: [`docs/evidence-gates.md`](evidence-gates.md). Estados
`unsupported` declarados: [`docs/degradation-matrix.md`](degradation-matrix.md). Modos de falha
operacionais: [`docs/troubleshooting.md`](troubleshooting.md).

## Os cinco eventos canonicos

Fonte unica: `internal/hookcontract.EventKind` (`internal/hookcontract/event.go`). Nenhum outro
pacote redefine o conjunto de eventos; `ParseEventKind` recusa nome desconhecido com
`ErrUnknownEvent`, nunca um zero-value silencioso.

| Evento | Constante | Quando dispara |
|---|---|---|
| `session_start` | `EventSessionStart` | inicio de sessao do agente |
| `before_tool` | `EventBeforeTool` | antes de uma chamada de ferramenta (Bash, Edit, Write, apply_patch) |
| `after_tool` | `EventAfterTool` | depois que uma chamada de ferramenta ja executou |
| `before_complete` | `EventBeforeComplete` | antes do agente encerrar a sessao/tarefa |
| `session_end` | `EventSessionEnd` | encerramento de sessao |

## As cinco familias de hooks

Fonte unica: `internal/capability.Family` (`internal/capability/family.go`).

| Familia | Proposito | Script/pacote principal |
|---|---|---|
| `git-policy` | nega `git commit`/`git push` nao solicitados e remocao destrutiva (`rm -rf` e variantes) | `.agents/scripts/git-operation-gate.sh`, `internal/hookpolicy/git.go` |
| `quality-gate` | roda testes/lint/fmt por tarefa, com fingerprint para dedupe | `internal/qualitygate/` |
| `evidence-gate` | exige `execution_report.md` completo, com criterio de aceite comprovado | `.agents/scripts/validate-task-evidence.sh`, `.agents/hooks/validate-session-end.sh` |
| `checkpoint` | exige `.checkpoints/<id>.json` valido antes de `status=done` (F25) | `.agents/hooks/post-execute-task.sh`, `internal/sdd.ResultValidator` |
| `telemetry` | metricas de hook opt-in, sempre com `unknown` explicito quando indisponivel | `internal/telemetry/` |

## Policies e gates

- **`git-operation-gate.sh`** — classificacao estrutural (tokenizer Python embutido, nao regex) do
  comando; nunca compara texto de saida entre CLIs. Bloqueia com exit `2`. Escapes dedicados e
  auditados: `GOVERNANCE_GIT_OPERATION_CONFIRMED=1` / `GOVERNANCE_GIT_OPERATION_MODE=warn` para
  operacao Git; `GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED=1` para comando destrutivo (sem modo
  `warn`); `GOVERNANCE_INTERPRETER_OPERATION_CONFIRMED=1` para invocacao de interpretador com
  conteudo dinamico (`$(...)`, crase, `eval`, `sh -c`) que a analise estatica nao consegue resolver.
  Comando Git de leitura (`status`, `diff`, `log`) nunca e bloqueado.
- **`validate-preload.sh`** — hook de `before_tool`; delega a `git-operation-gate.sh` antes de
  qualquer escape de preload, para que a decisao de policy Git seja avaliada independentemente de
  `GOVERNANCE_PRELOAD_CONFIRMED`. Tambem verifica pre-requisito de skill por tipo de arquivo tocado.
- **`validate-governance.sh`** — hook de `after_tool`. **Informa** violacao (edicao de `AGENTS.md`/
  `SKILL.md`), com `exit 1` **depois** que a edicao ja ocorreu — nenhuma das quatro CLIs bloqueia em
  `AfterTool`. `GOVERNANCE_HOOK_MODE=warn` suprime o aviso pos-edicao para tarefas que precisam
  editar esses arquivos deliberadamente.
- **`validate-session-end.sh`** — hook de `before_complete`; recusa encerrar sessao com tarefa
  `in_progress` sem veredito `APPROVED` no relatorio.
- **`post-execute-task.sh`** — gates F2 (evidencia), F24 (escalonamento de achado critico), F25
  (checkpoint), F35 (validacao de `DiffSHA` contra o git log).
- **`internal/hookcontract.HookTimeoutRegistry`/`MeasureHook`** — todo hook tem timeout declarado;
  vencer o prazo produz `Decision: BLOCK`, `gate_id: hook-timeout-guard` (timeout e negacao, nunca
  aprovacao — RF-66).
- **`internal/hookcontract.CheckRecursionGuard`** — bloqueia por construcao cadeia
  hook → ferramenta → hook alem de `MaxInvocationDepth` (RF-67).

## Matriz de capabilities por provedor

Duas matrizes distintas, geradas de codigo (nunca escritas a mao):

1. **`internal/capability.Generate()`** (`testdata/capability-matrix.json`) — invariantes de
   paridade `internal/parity` por provedor (RF-18/ADR-008): `supported`, `unsupported`,
   `provider capability`, `unknown`.
2. **`internal/capability.GenerateHooks()`** (`testdata/hook-capability-matrix.json`) — evento x
   familia x provedor, `hookcontract.SupportState` (`verified`, `adapter`, `unsupported`). Ver
   `docs/degradation-matrix.md` para a leitura completa dos estados `unsupported` desta matriz.

`ai-spec doctor` consulta ambas e reporta adapter ausente, hook nao executavel, configuracao
invalida, versao incompativel e divergencia contrato/adapter, sem chamar LLM (RF-60).

## Limitacoes por provedor

| Provedor | Limitacao | Onde esta declarada |
|---|---|---|
| Codex CLI | Hook alterado fica *enrolled mas untrusted* e e **pulado silenciosamente** apos mudanca de hash (bug publico openai/codex#46210, R-07). Nao corrigivel pelo harness; `doctor --codex-trust` detecta via RPC read-only | `internal/runtime/precondition`, `internal/doctor/doctor.go:checkCodexTrustedHash` |
| GitHub Copilot CLI | `preToolUse` e fail-closed para exit 2/non-zero, mas **fail-open no timeout** (R-08). Declarado como `SupportAdapter` com `limitation` preenchido, nunca `verified` | `testdata/hook-capability-matrix.json` |
| OpenCode | Unico provedor sem `SessionEnd` nativo; usa `session.idle` (`BeforeComplete`) **nao-bloqueante** como aproximacao. Nega por `throw` de plugin JS, nao por exit code — nunca declarado como paridade simulada (P07) | `internal/capability/hookmatrix.go`, plugin `.opencode/plugin/governance.js` |
| Claude Code | Sem hook shell nativo dedicado a `session_start`/`session_end` hoje; ver `docs/runtime-claude-capabilities.md` | `docs/runtime-claude-capabilities.md` |
| Todos | `quality-gate` e `telemetry` ainda sem wiring nativo (registrados so no orquestrador ACP interno); nenhum provedor tem essas familias como `verified` em nenhum evento hoje | `testdata/hook-capability-matrix.json` |

## Limitacoes por stack

O quality gate resolve comandos por stack detectada (`internal/detect`), cinco alvo: Go, Node,
Python, .NET, Java (Maven e Gradle). `internal/skills.AllLangs` e o teste de exaustividade
`TestAllLangsAreExhaustivelyWired` garantem que nenhuma stack tem `switch` sem `default` silencioso.
Projeto poliglota: o desempate de deteccao e "primeiro com score maximo vence" — marcador de stack
nova sempre entra ao **fim** de `manifestTypes`, nunca no meio (`internal/detect/toolchain.go`),
para nao alterar o vencedor em monorepos existentes.

## Troubleshooting

Ver [`docs/troubleshooting.md`](troubleshooting.md) para os procedimentos completos, com sintoma,
causa raiz, solucao e verificacao, cobrindo: timeout de hook, guarda de recursao, checkpoint
corrompido e evidencia invalida — alem dos problemas de instalacao, configuracao e memoria duravel
ja documentados la.

## Conformidade cross-provider (RF-61 a RF-63)

`tests/integration/conformance_suite_test.go` e `tests/integration/conformance_suite_rf61_test.go`
cobrem os catorze cenarios de RF-61 (`internal/conformance.RF61Scenarios()`), com fixture
compartilhada e resultado derivado do contrato (RF-62, nunca igualdade textual entre CLIs). Cada
cenario tem uma mutacao (par valido/invalido, ou remocao do script canonico) que o deixa vermelho —
propriedade *gate-of-the-gate* estendida de
`TestConformanceSuite_SuiteFailsWhenCanonicalGitOperationGateIsRemoved`. Nem todo cenario dispatcha
de fato pelos quatro wrappers nativos: `git-policy` (1-4) e `evidence-gate` (6-7) e `adapter
retornando erro`/`bypass de comando` (13-14) disparam o script/plugin real instalado por provedor;
`quality-gate` (5), `checkpoint` (8-9), `telemetry` (10) e o `evento desconhecido` (11) validam
nucleo Go sem ramificacao por CLI — essas quatro familias ainda nao tem wiring nativo provado por
provedor em nenhum evento (`docs/degradation-matrix.md`), entao o cenario roda o mesmo validador uma
vez por rotulo de provedor, o que prova a invariancia por construcao, nao quatro dispatches
distintos; `capability nao suportada` (12) discrimina genuinamente por `cell.Provider` na matriz
gerada. `internal/conformance.RF61Scenarios()`
e `internal/conformance.Manifest()` sao catalogos **distintos e intencionalmente nao-convergentes**:
o primeiro enumera cenarios de conformidade em nivel de comando/gate (RF-61); o segundo classifica
determinismo dos catorze cenarios de produto da User Story, em nivel de sessao. A mistura dos dois
nomes seria uma colisao silenciosa — `TestRF61Scenarios_AreADistinctCatalogFromTheManifest`
(`internal/conformance/rf61_test.go`) falha caso isso volte a acontecer.
