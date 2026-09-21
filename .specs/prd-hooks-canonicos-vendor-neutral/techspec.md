<!-- spec-hash-prd: 2672e949229c9a0e9058e91882691b8e2b7362464b162d6d7aa209bdec082acb -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Hooks Canônicos, Determinísticos e Vendor-Neutral

## Resumo Executivo

Esta especificação implementa os 75 requisitos do PRD sobre uma base que a investigação revelou ser
**mais rica e mais frágil** do que o PRD supunha. Mais rica: `internal/capability/` já gera uma matriz
de 54 células sobre quatro provedores, `internal/fs/fs.go:173-204` já tem escrita atômica correta,
`internal/runtime/memory/durable/` já tem lock por camada com lease e detecção de processo morto, e o
ponto que o repositório chama de `session-end` cobre de fato a semântica `BeforeComplete` — o evento
mais difícil dos cinco. Mais frágil: existe um **bypass determinístico e verificado** do gate Git, os
erros de hook nos pontos de tool-call são **descartados com `_ =`**, a prova de dispatch da matriz de
capabilities ainda resolve **um booleano global** aplicado a todas as células afirmativas, e
`events.jsonl` — o artefato que sustenta toda a evidência — faz
rewrite completo sobre `os.WriteFile` com `O_TRUNC`, de modo que um crash não perde o último evento e
sim **a sessão inteira**.

A estratégia central é **uma fonte única de verdade nova e aditiva**, não uma reescrita. Um pacote
`internal/hookcontract` passa a possuir os cinco eventos canônicos, o payload versionado e o resultado
tipado. Os dois modelos existentes — os três pontos de `internal/runtime/specs` e os catorze pontos de
`internal/runtime/hooks` — tornam-se **projeções derivadas** dessa fonte, cada uma preservando sua
superfície pública atual, com um gate de CI que falha se as projeções divergirem do contrato. Isso é
possível a baixo risco porque a investigação provou que os dois modelos são **disjuntos em código**:
zero imports cruzados, e `NewPointCoverage`/`NewEnforcement` têm exatamente três call-sites de
produção, todos em `registry.go`.

A entrega é sequenciada por **pré-requisitos duros de segurança e de instrumentação**, não por
conveniência. Quatro correções precedem qualquer funcionalidade nova, porque cada uma delas, se
deixada para depois, transforma trabalho correto em regressão silenciosa: fechar o truncamento de
payload, isentar `.tmp-*` nos gates de patch, indexar a prova de dispatch por célula, e instalar
um teste de exaustividade de linguagens que converte os cinco `switch` sem `default` em falha de
compilação.

---

## Arquitetura do Sistema

### Visão Geral dos Componentes

#### Componentes novos

| Componente | Responsabilidade |
|---|---|
| `internal/hookcontract/` | **Fonte única** dos cinco eventos canônicos, do envelope de payload versionado e do resultado tipado. Não importa nenhum pacote de provedor nem de runtime. É a raiz da inversão de dependência. |
| `internal/hookcontract/event.go` | `EventKind` (VO fechado, 5 valores), `Capability` por provedor e evento, `SupportState`. |
| `internal/hookcontract/result.go` | `Decision` (`ALLOW`/`BLOCK`/`WARN`/`NOT_APPLICABLE`/`ERROR`), `Result` com `reason`, `policyID`, `gateID`, `evidence`, `metadata`. |
| `internal/hookcontract/payload.go` | `Envelope` versionado, `Decoder` estrito, `ErrUnknownEvent`, `ErrSchemaVersion`. |
| `internal/hookcontract/exitcode.go` | Tradução única e bidirecional exit code ↔ `Decision`, fail-closed. |
| `internal/hookpolicy/` | Ponte entre `internal/harness.Contract` e os gates. Hoje `GitPolicy` é decorativo: zero call-sites. Este pacote é o elo ausente. |
| `internal/hookpolicy/git.go` | Deriva a lista de operações Git interceptadas a partir da policy declarada, e a exporta em formato consumível pelo shell. |
| `internal/hookaudit/` | Leitor e validador do log de escapes — hoje o log tem escritores e **nenhum leitor**. |
| `.agents/lib/hook-payload.sh` | Substitui o parsing truncado de `parse-hook-input.sh`, sem limite de 64 KiB e sem fallback por `grep`. |
| `internal/detect/toolchain_java.go`, `toolchain_dotnet.go` | Resolução de `fmt`/`test`/`lint` para as duas stacks hoje ausentes do resolver. |
| `internal/skills/lang_exhaustive_wiring_test.go` | Gate que itera `AllLangs` e exige, por linguagem, skill declarada, entrada em `langImplementationSkills`, label em `contextgen` e arquivo de triggers. |

#### Componentes modificados

| Componente | Mudança | Evidência do estado atual |
|---|---|---|
| `internal/runtime/specs/enforcement.go` | `CanonicalPoint` passa a ser projeção de `hookcontract.EventKind`; `NewEnforcement` deixa de exigir exatamente 3 pontos e passa a exigir **cobertura declarada** por ponto — onde `unsupported` é declaração válida, não ausência. | `enforcement.go:139-143` exige os 3; `registry.go:296,307` faria panic ao ampliar |
| `internal/runtime/specs/registry.go` | `cliHookKeyVocabulary` ganha `SessionStart` e o `SessionEnd` real; `agentNativeConfigs` corrige caminhos do Copilot e completa os do Codex | `registry.go:17-38`, `:77-89` |
| `internal/runtime/hooks/dispatcher.go` | `Kind()` passa a refletir o ponto real de despacho; pontos mapeiam para `hookcontract.EventKind` | `dispatcher.go:77,85` retornam o ponto errado em `post_build`/`post_complete` |
| `internal/runtime/runner.go` | Erros de hook em tool-call deixam de ser descartados | `runner.go:428,466` — `_ = ` |
| `internal/capability/evidence.go` | Indexar a prova por `(provider, capability)` e trocar o fallback mudo por erro tipado | `evidence.go:146` — `resolved` é um booleano global e o closure em `:147-149` ignora os dois argumentos; `evidence.go:169-173` devolve `false` sem distinguir erro de ambiente |
| `internal/runtime/persistence/jsonl.go` | Append real + detecção de corrupção | `jsonl.go:23,57-58` rewrite completo |
| `internal/detect/toolchain.go` | `manifestTypes` ganha Java e .NET **no fim**; `switch` ganha `default` que falha. .NET não participa da varredura recursiva de profundidade 4 que vive aqui e em `framework.go` — em `detect.go` as quatro linguagens usam verificação rasa, então a assimetria é entre camadas, não dentro de `detect.go` | `toolchain.go:99-108,148-163,423-464` |
| `internal/contextgen/contextgen.go` | Allow-list de 3 strings passa a derivar de `skills.AllLangs` | `contextgen.go:339-340` |
| `internal/triggers/triggers.go` | `.cs`, `.java`, `.kt` deixam de cair no fallback Go | `triggers.go:59-66,78-98` |
| `.agents/scripts/git-operation-gate.sh` | Escopo derivado da policy; parsing por palavra-de-comando; fail-closed em comando vazio | `git-operation-gate.sh:39-41,70` |
| `internal/doctor/doctor.go` | Seção de hooks e adapters acrescentada aos 15 checks já estratificados por camada e por provedor | `doctor.go:153-530` — os checks `Contrato do harness`, `Integridade de skills (lock)`, `Sincronia canonica/derivados`, `Skills disponiveis`, `Politicas aplicadas`, `Validadores de evidencia instalados` e `Pre-condicoes de enforcement` já existem |

### Relacionamentos e Fluxo de Dados

```text
                    internal/harness.Contract
                   (policy declarada, .agents/harness.yaml)
                                │
                                ▼
                       internal/hookpolicy
              (deriva escopo de gate a partir da policy)
                                │
                                ▼
        ┌───────────────  internal/hookcontract  ───────────────┐
        │        EventKind · Envelope · Result · Decision       │
        │              (não importa nenhum provedor)            │
        └───────┬───────────────────────────────────┬───────────┘
                │ projeção                          │ projeção
                ▼                                   ▼
   internal/runtime/specs                  internal/runtime/hooks
   (CanonicalPoint, confronto              (Dispatcher, modo ACP
    de config nativa, paridade)             orquestrado)
                │                                   │
                ▼                                   ▼
   .claude/settings.json                   GovernanceHook, SpecDriftHook,
   .codex/config.toml                      TokenBudgetHook, MemoryPersistHook,
   .github/copilot/settings.json           MemoryEvidenceHook
   .opencode/plugin/governance.js
                │
                ▼
   .agents/hooks/validate-preload.sh  ──►  .agents/scripts/git-operation-gate.sh
                                     ──►  .agents/scripts/hook-prereq-gate.sh
```

A direção da dependência é invariante: `hookcontract` não conhece provedor, runtime nem shell. Um
gate de CI (`TestHookContractHasNoProviderIdentifier`) falha se um identificador de provedor aparecer
no pacote — satisfazendo RF-07 por construção verificável, não por convenção.

---

## Design de Implementação

### Interfaces Chave

```go
package hookcontract

type EventKind int

const (
	EventSessionStart EventKind = iota + 1
	EventBeforeTool
	EventAfterTool
	EventBeforeComplete
	EventSessionEnd
)

type Decision int

const (
	DecisionAllow Decision = iota + 1
	DecisionBlock
	DecisionWarn
	DecisionNotApplicable
	DecisionError
)

type Result struct {
	decision Decision
	reason   string
	policyID string
	gateID   string
	evidence []EvidenceRef
	metadata map[string]string
	valid    bool
}

func NewResult(decision Decision, reason, policyID, gateID string) (Result, error)
func NewAllow() Result
func NewNotApplicable(reason string) Result
```

```go
package hookcontract

type SupportState int

const (
	SupportVerified SupportState = iota + 1
	SupportAdapter
	SupportUnsupported
)

type Capability struct {
	provider  string
	event     EventKind
	state     SupportState
	nativeKey string
	blocking  bool
	limitation string
	valid     bool
}

type CapabilityRegistry interface {
	Lookup(provider string, event EventKind) (Capability, error)
	Providers() []string
	Events() []EventKind
}
```

```go
package hookcontract

type Envelope struct {
	SchemaVersion int
	Event         EventKind
	Provider      string
	SessionID     string
	Payload       json.RawMessage
}

type Decoder interface {
	Decode(raw []byte) (Envelope, error)
}

var (
	ErrUnknownEvent    = errors.New("unknown canonical event")
	ErrSchemaVersion   = errors.New("unsupported payload schema version")
	ErrUnknownField    = errors.New("unknown field in payload envelope")
	ErrProviderUnknown = errors.New("provider not declared in capability registry")
)
```

```go
package hookcontract

type ExitCodeTranslator interface {
	ToDecision(code int, stderr string, critical bool) Result
	ToExitCode(result Result) int
}
```

`ToDecision` é o ponto único onde RF-14 é garantido: com `critical == true`, qualquer código não
mapeado produz `DecisionError`, e `DecisionError` em hook crítico é convertido em bloqueio pelo
chamador. Sem esse ponto único, cada script continuaria decidindo por conta própria — que é
exatamente o estado atual, com `exit 1` em `validate-governance.sh:45` e `exit 2` em
`git-operation-gate.sh:5`.

```go
package hookpolicy

type GitOperation struct {
	subcommand   string
	destructive  bool
	requiresApproval bool
}

type GitScope interface {
	Operations() []GitOperation
	Fingerprint() string
}

func NewGitScope(contract harness.Contract) (GitScope, error)
```

`Fingerprint()` é um SHA-256 da lista derivada, gravado em `.agents/generated/git-scope.json` e lido
pelo shell. É o mecanismo que satisfaz RF-19: um gate compara o fingerprint do artefato gerado com o
recalculado a partir do contrato, e falha na divergência. Sem isso, "derivado da policy" seria uma
afirmação não verificável — e a investigação mostrou que a afirmação equivalente já existe hoje e é
falsa (`GitPolicy` tem zero call-sites de produção).

### Modelos de Dados

**Envelope de payload (schema v1), lido pelo shell e pelo Go:**

```json
{
  "schema_version": 1,
  "event": "before_tool",
  "provider": "claude",
  "session_id": "…",
  "payload": { "tool_name": "Bash", "tool_input": { "command": "…" } }
}
```

**Resultado canônico persistido (`.aispec/hook-decisions.jsonl`), append real:**

```json
{
  "ts": "2026-09-18T12:00:00Z",
  "event": "before_tool",
  "provider": "claude",
  "hook": "git-policy",
  "decision": "BLOCK",
  "reason": "git push requires explicit approval",
  "policy_id": "P-GIT-001",
  "gate_id": "G-GIT-PUSH",
  "duration_ms": 12,
  "evidence": ["…"]
}
```

Este arquivo substitui `.aispec/governance-escapes.log`, que hoje é TSV sem leitor, sem rotação e
corrompível por `\t` no comando (RF-16, RF-23). O formato TSV atual é preservado em paralelo por um
release para não quebrar consumidores externos não identificados, e removido na tarefa de limpeza.

**Matriz de capabilities estendida** — `testdata/hook-capability-matrix.json`, eixo
`provider × event × family`, distinto da matriz existente (`testdata/capability-matrix.json`, eixo
`provider × invariante de paridade`). As duas coexistem; a nova é gerada a partir de
`hookcontract.CapabilityRegistry` e validada contra teste por célula.

### Endpoints de API

Não aplicável — o produto é uma CLI. As superfícies novas de linha de comando são:

- `ai-spec doctor <path> --hooks` — seção de hooks e adapters (RF-60), exit code ≠ 0 em falha.
- `ai-spec hooks inventory` — emite o inventário em JSON e Markdown (RF-01, RF-05).
- `ai-spec hooks capabilities` — imprime a matriz e o estado de cada célula (RF-59).

---

## Pontos de Integração

| Integração | Natureza | Tratamento de erro |
|---|---|---|
| Claude Code | `settings.json` (projeto, versionado) e `settings.local.json` (local). A investigação confirmou a precedência oficial: user < plugin < project < local < managed | RF-52: o instalador passa a escrever a fiação canônica em `.claude/settings.json` (versionado) e o registro marca essa fonte como `required: true`. `settings.local.json` continua aceito como sobreposição local, mas deixa de ser a única fonte |
| Codex CLI | `.codex/config.toml` **e** `.codex/hooks.json` (planejado), ambos por camada; trust por hash | Hook alterado fica *enrolled mas untrusted* e é **pulado silenciosamente** (bug público openai/codex#46210). O adapter declara `PreconditionTrustedHash` como bloqueante e o `doctor` consulta o trust por RPC read-only — mecanismo já existente em `internal/runtime/precondition` |
| GitHub Copilot CLI | `.github/hooks/*.json` e `.github/copilot/settings.json` (planejado) — caminho de projeto-alvo escrito pelo instalador (RF-52/8.0); o repositório declarava antes `.github/settings.json` (planejado), caminho que a documentação oficial não usa e que deixou de ser gerado após a correção da 8.0 | `preToolUse` é fail-closed para exit 2 e non-zero, mas **fail-open no timeout**. Modelado explicitamente como limitação declarada, não corrigida |
| OpenCode | Plugin JS in-process; nega por `throw`, não por exit code | Único provedor sem `SessionEnd` e com `BeforeComplete` (`session.idle`) **não bloqueante**. Declarado `SupportAdapter` com `limitation` preenchido e `SupportUnsupported` para `SessionEnd` — nunca paridade simulada (P07) |
| Binário `ai-spec` | Invocado por `post-execute-task.sh:87` e `validate-task-evidence.sh:416-433` com checagem de versão mínima | Já é fail-closed; preservado |
| `git` | `rev-parse`, `cat-file`, `merge-base`, `ls-files` em vários gates | Preservado |

---

## Abordagem de Testes

### Estratégia por bloco

O repositório já tem três camadas (`make test`, `make integration` com build tag, `sdd-evals`) e um
mecanismo de **prova de dispatch por célula** com coleta por subprocesso real
(`parity_dispatch_proof_test.go:288`), blindado contra forja por `Logf` e por `Skip`
(`parity_dispatch_proof_injection_test.go:26,52`). Esse mecanismo é o ativo mais valioso da suíte e
é **estendido**, não substituído.

### Testes Unitários

**`internal/hookcontract`** — tabela por evento e por decisão:

- `TestEventKind_ParseAndString` — round-trip dos 5 eventos; entrada desconhecida produz
  `ErrUnknownEvent`, nunca zero-value silencioso (RF-10).
- `TestEnvelope_RejectsUnknownSchemaVersion` e `TestEnvelope_RejectsUnknownField` — parse estrito
  (RF-09). Fuzz sobre o decoder, seguindo `internal/harness/fuzz_test.go`.
- `TestResult_BlockRequiresReasonAndPolicy` — `NewResult(DecisionBlock, "", …)` falha (RF-13).
- `TestExitCodeTranslator_CriticalUnknownCodeBecomesError` — código não mapeado com `critical=true`
  produz `DecisionError` (RF-14).
- `TestExitCodeTranslator_PreservesCurrentSemantics` — tabela com os códigos reais hoje em uso:
  `git-operation-gate.sh` exit 2, `validate-governance.sh` exit 1, `validate-session-end.sh` exit 2,
  `validate-preload.sh` exit 2. Este teste é a **rede de não-regressão** de RF-15.
- `TestHookContractHasNoProviderIdentifier` — varre o AST do pacote e falha se `claude`, `codex`,
  `copilot` ou `opencode` aparecer em identificador ou literal (RF-07).

**`internal/hookpolicy`**:

- `TestGitScope_DerivesFromContract` — `auto_commit: true` remove `commit` do escopo; `false` mantém.
  Hoje esse elo não existe: o teste nasce vermelho e é a prova de RF-19.
- `TestGitScope_FingerprintChangesWithPolicy`.

**`internal/capability`**:

- `TestDispatchProof_DiscriminatesByCell` — prova que duas células com provas distintas recebem
  veredito distinto. Hoje `evidence.go:146-149` devolve o mesmo booleano para todas; nasce
  vermelho (RF-59).
- `TestDispatchProof_ReportsEnvironmentFailure` — a falha de `repoRootFromWorkingDir` produz erro
  reportado, não um veredito de "nenhuma prova" indistinguível (`evidence.go:169-173`).

**`internal/skills`**:

- `TestAllLangsAreExhaustivelyWired` — para cada `Lang` em `AllLangs`, exige: ≥1 skill em
  `LangSkills`, entrada em `install.langImplementationSkills`, label em `contextgen`, arquivo de
  triggers, e `case` no resolver de toolchain. Este teste é **pré-requisito** de qualquer trabalho
  multi-stack: ele converte os cinco `switch` sem `default` em falha determinística, e nasce vermelho
  para `LangDotNet` antes mesmo de Java entrar.

**`internal/detect`**:

- `stack_coverage_test.go` ganha cenários `java-maven`, `java-gradle` e `dotnet-api`.
  **Restrição de ordem:** os marcadores novos vão ao **fim** de `manifestTypes` — o desempate é
  "primeiro com score máximo vence" (`toolchain.go:139-145`), e inserir no meio altera o vencedor em
  monorepos poliglotas.

**Shell** — `scripts/test-validators.sh` ganha casos hoje inexistentes (o gate Git tem **zero**
referências nos scripts de teste):

- Matriz adversarial: `git -C /repo push`, `git -c user.name=x commit`, `$(echo git) push`,
  `eval "git push"`, `sh -c "git push"`, `/usr/bin/git push`, `git reset --hard`, `git clean -fd`,
  `git checkout -- .`, `git restore .`, `git push --force`, `git push -f`, `git push origin +main`.
- Matriz de falso positivo, com resultado esperado **PERMITE**: `echo git commit`, `man git commit`,
  `git log --grep="commit"`, `echo "a|git push"`, `grep -r "git commit" docs/`.
- **Payload acima de 64 KiB** com `git push` no fim — hoje EXIT=0 verificado; deve ser EXIT=2.
- Comando vazio / não extraível — hoje EXIT=0; deve ser bloqueio, alinhando com
  `validate-preload.sh:109`.
- JSON malformado chegando ao parser.

### Testes de Integração

O repositório **já tem** fronteiras de IO críticas, já teve incidente de gate que não executava
(V-24, V-33 do PRD dependente) e já mantém a infraestrutura. Integração é mandatória. Não se usa
testcontainers: as dependências são filesystem, subprocesso e binários de CLI, cobertos por
`t.TempDir()` e pelos fakes existentes — introduzir Docker aqui seria dependência sem demanda
concreta.

- `tests/integration/hook_contract_projection_test.go` — prova que `specs.CanonicalPoint` e os pontos
  de `hooks.Dispatcher` são projeções fiéis de `hookcontract.EventKind`, e falha se divergirem
  (RF-08).
- `tests/integration/conformance_suite_test.go` — estendido dos 5 comandos atuais para os **14
  cenários** de RF-61, executados contra os 4 provedores, com fixtures compartilhadas.
- `tests/integration/hook_payload_boundary_test.go` — payloads de 1 KiB, 64 KiB, 65 KiB e 1 MiB;
  todos devem produzir a mesma decisão.
- `tests/integration/atomic_write_isolation_test.go` — prova que `.tmp-*` não aparece no patch
  semântico nem dispara violação de isolamento. **Precede** qualquer migração de escrita.
- `tests/integration/portability_test.go` — fixtures `java-maven`, `java-gradle`, `dotnet-api`
  acrescentadas ao **fim** de `cases` (a asserção `wantStackSub` é substring contígua).
- `internal/runtime/persistence/jsonl_crash_recovery_test.go` — trunca `events.jsonl` no meio de uma
  linha e exige detecção explícita, não consumo silencioso.

### Testes E2E

`tests/integration/hooks_live/` e `acp_live/` já executam contra CLIs reais em workflow nightly e não
são gate de merge. A matriz de células (`hooks_live/matrix.go:20-25`) passa de 4 × 3 para 4 × 5,
com as células `unsupported` declaradas produzindo *skip registrado*, nunca *pass*.

Cenário de aceite ponta a ponta do PRD, executado por provedor: `ai-spec install .` → tentativa de
`git push` não autorizada → bloqueio com `policy_id` → quality gate → evidence gate → checkpoint →
telemetria.

### Rastreabilidade — RF × decisão × arquivo × teste

| RF | Decisão | Arquivo principal | Teste |
|---|---|---|---|
| RF-01, RF-02 | Inventário gerado a partir do código, não redigido à mão | `internal/hookinventory/` | `TestInventoryCoversEveryWiredHook` |
| RF-03, RF-04 | Classificação declarada no inventário; `REMOVE` não executa | `docs/hook-inventory.md` | `TestEveryInventoryItemHasClassification` |
| RF-05 | Gate anti-órfão | `scripts/check-hooks-inventory.sh` | `sync_gates_guard_test.go` |
| RF-06 | 5 eventos em fonte única | `hookcontract/event.go` | `TestEventKind_ParseAndString` |
| RF-07 | Gate de AST sem identificador de provedor | `hookcontract/` | `TestHookContractHasNoProviderIdentifier` |
| RF-08 | Projeção verificada dos dois modelos | `specs/enforcement.go`, `hooks/dispatcher.go` | `hook_contract_projection_test.go` |
| RF-09 | Envelope versionado com parse estrito | `hookcontract/payload.go` | `TestEnvelope_Rejects*` + fuzz |
| RF-10 | `ErrUnknownEvent` | `hookcontract/payload.go` | idem |
| RF-11 | `SupportUnsupported` como estado próprio | `hookcontract/event.go` | `TestCapability_UnsupportedIsNotSuccess` |
| RF-12, RF-13 | `Result` com construtor validador | `hookcontract/result.go` | `TestResult_BlockRequiresReasonAndPolicy` |
| RF-14 | `critical` no tradutor | `hookcontract/exitcode.go` | `TestExitCodeTranslator_Critical*` |
| RF-15 | Tradutor único, tabela dos códigos atuais | `hookcontract/exitcode.go` | `TestExitCodeTranslator_PreservesCurrentSemantics` |
| RF-16 | `hook-decisions.jsonl` com leitor | `internal/hookaudit/` | `TestAuditLogRoundTrip` |
| RF-17 | `SchemaVersion` + tabela de migração | `hookcontract/payload.go` | `TestSchemaVersionMigration` |
| RF-18 a RF-24 | Escopo derivado + parsing por palavra-de-comando | `hookpolicy/git.go`, `git-operation-gate.sh` | matriz adversarial + matriz de FP |
| RF-25 a RF-30 | Quality gate por evento, com fingerprint | `internal/qualitygate/` | `TestQualityGate_SkipsWhenFingerprintUnchanged` |
| RF-31 a RF-34 | Toolchain das 5 stacks + prova de teste | `detect/toolchain_*.go`, `validate-task-evidence.sh` | `stack_coverage_test.go`, `test-validators.sh` |
| RF-35 a RF-38 | Evidence gate preservado, independente de provedor | `validate-task-evidence.sh` | `session_end_gate_test.go` |
| RF-39 a RF-44 | Checkpoint atômico com detecção de corrupção | `persistence/`, `post-wave.sh` | `internal/runtime/persistence/jsonl_crash_recovery_test.go` |
| RF-45 a RF-50 | Schema comum com `unknown` explícito | `internal/telemetry/` | `TestTelemetry_UnavailableIsUnknown` |
| RF-51, RF-52 | `.claude/settings.json` versionado e `required` | `install.go`, `registry.go` | `native_config_gate_test.go` |
| RF-53 a RF-55 | Adapters Codex, OpenCode, Copilot | `specs/registry.go` | suíte por adapter |
| RF-56 | Gate de não-duplicação de policy em adapter | `specs/` | `TestAdapterContainsNoPolicyDecision` |
| RF-57 | Payload não confiável | `hookcontract/payload.go`, `hook-payload.sh` | `hook_payload_boundary_test.go` |
| RF-58 | Suíte por adapter | `tests/integration/` | dispatch proof por célula |
| RF-59 | Matriz por evento × família, prova discriminada | `internal/capability/` | `TestDispatchProof_DiscriminatesByCell` |
| RF-60 | `doctor --hooks` sobre os 15 checks existentes | `internal/doctor/` | `doctor_hooks_test.go`, ao lado de `multiprovider_test.go` |
| RF-61 a RF-63 | 14 cenários × 4 provedores | `conformance_suite_test.go` | idem |
| RF-64, RF-65 | Ablation com baseline | `internal/telemetry/ablation.go` | `TestAblation_BaselineWithoutHook` |
| RF-66, RF-67 | Timeout por hook + guarda de recursão | `hookcontract/`, scripts | `TestHookTimeout*`, `TestRecursionGuard` |
| RF-68 a RF-70 | Parsing estrutural, paths normalizados, integridade | `hook-payload.sh`, `fs/` | `hook_payload_boundary_test.go` |
| RF-71, RF-72 | Instalação por provedor configurado, sem daemon | `install.go` | `install_test.go` |
| RF-73 | Portabilidade nas 5 stacks | `portability_test.go` | idem |
| RF-74 | Não-regressão | todos | suíte completa + `make coverage` |
| RF-75 | Documentação | `docs/hooks-canonicos.md` | revisão |

---

## Sequenciamento de Desenvolvimento

### Ordem de Build

**Fase 0 — Pré-requisitos duros. Nada começa antes.**

1. **Acrescentar `.tmp-*` à lista de exclusões de `buildExclusions` (`orchestrator.go:456-512`) e à
   tolerância de `isolation.go:322`.** Hoje o padrão **não consta** em nenhuma das duas — a etapa é
   adicioná-lo, não removê-lo.
   `WriteFileAtomic` cria `.tmp-*` no diretório de destino. Qualquer migração de escrita sob
   `.specs/` feita antes disto faz o arquivo temporário aparecer como untracked no patch semântico,
   altera o `PatchSHA256` e dispara o fail-closed de `orchestrator.go:352-354` — um falso positivo
   que pararia a entrega inteira. **Este é o item de maior alavancagem do plano.**
2. **Fechar o truncamento de payload.** `.agents/lib/hook-payload.sh` sem limite de 64 KiB e sem
   fallback por `grep`; comando não extraível passa a ser negação, alinhando
   `git-operation-gate.sh:39-41` com `validate-preload.sh:109`. É a correção do único bypass
   verificado end-to-end.
3. **`TestAllLangsAreExhaustivelyWired`.** Nasce vermelho para `LangDotNet`. Sem ele, adicionar
   `LangJava` compila, passa toda a suíte atual e produz comportamento errado em silêncio.
4. **Prova de dispatch indexada por célula.** A prova ficou mais forte desde a primeira redação
   deste documento — `dispatchProvenFromTests` agora exige execução real (`go test -json -count=1`,
   `evidence.go:128-137`) além da declaração por AST, e `skip`/`fail` revogam o `pass`
   (`evidence.go:114-117`), travado por `evidence_regression_test.go`. Mas continua **não
   discriminando por célula**: `resolved` é um booleano único (`evidence.go:146`) e o closure ignora
   `provider` e `capabilityID` (`evidence.go:147-149`). As 44 células afirmativas seguem provadas em
   bloco. Sem indexar antes, toda célula do eixo novo herdaria a mesma prova global.
   Complemento: o fallback de `repoRootFromWorkingDir` (`evidence.go:169-173`) devolve `false` mudo —
   fail-closed e seguro, porém indistinguível de "nenhuma prova existe".

**Fase 1 — Contrato canônico (RF-06 a RF-17).** `internal/hookcontract` é aditivo: nada o consome
ainda. Baixo risco, alto desbloqueio.

**Fase 2 — Projeções (RF-08).** `specs.CanonicalPoint` e `hooks.Dispatcher` passam a derivar do
contrato. Requer relaxar `NewEnforcement` de "exatamente 3 pontos" para "cobertura declarada por
ponto". Toca ~10 testes que asseram `len(points) != 3` — todos identificados: `registry_test.go:131`,
`hooks_parity_matrix_test.go:95,101`, `parity_dispatch_proof_test.go:280`, `registry_test.go:141`,
`precondition_report_test.go:35,66`, `precondition_test.go:333`, `hooks_live/matrix.go:20`,
`live_test.go:147,393`.

**Fase 3 — Git policy (RF-18 a RF-24).** `hookpolicy` + reescrita do gate com parsing por
palavra-de-comando. A matriz adversarial e a de falso positivo são escritas **antes** da correção.

**Fase 4 — Multi-stack (RF-31 a RF-34).** .NET primeiro (completa o que já está em `AllLangs` e
corrige três bugs ativos: ausência no resolver de toolchain, ausência em `DetectPrimaryStack` e
fallback de triggers para Go — sendo que a ausência nas duas varreduras recursivas de profundidade 4,
em `toolchain.go:423-464` e `framework.go:31,57,83,102`, é a causa de `src/Api/Api.csproj` não ser
resolvido em monorepo), Java depois. Marcadores sempre **no fim** de `manifestTypes`; `"C#/.NET"` e
`"Java/Kotlin"` sempre **no fim** de `DetectPrimaryStack`.

**Fase 5 — Adapters e matriz (RF-51 a RF-59).** Inclui `SessionStart` e o `SessionEnd` real, que a
pesquisa oficial confirmou existir em Claude, Codex e Copilot e não existir em OpenCode.

**Fase 6 — Quality gate e telemetria (RF-25 a RF-30, RF-45 a RF-50).**

**Fase 7 — Atomicidade e checkpoint (RF-39 a RF-44).** Na ordem recomendada pela investigação:
artefatos terminais e write-once primeiro (`seal_evidence.go:72`, `report.go:69`,
`toolcalls.go:23`); `tasks.md` depois, unificando com o `flock` que a skill já tenta por fora;
`events.jsonl` por último e como **redesenho** (append real), não troca de chamada — trocar a linha
esconde o problema sem resolver o rewrite O(n) nem a ausência de merge inter-processo.
`memory/store.go` em `WriteModeAppend` **não é migrado**: o `O_APPEND` do kernel já serializa
corretamente e converter para read-modify-write pioraria a concorrência.

**Fase 8 — Doctor, conformidade, ablation, documentação (RF-60 a RF-75).**

### Dependências Técnicas

**Pré-requisito bloqueante — o PRD dependente está inteiro fora do git.** As 13 tarefas estão
marcadas `done`, mas `HEAD` é `a8b7721` (release v2.0.1, anterior ao PRD) e `git status --short`
retorna 119 entradas com zero commits. Estão untracked, entre outros: `internal/capability/`,
`internal/txn/`, `internal/harness/`, `internal/conformance/`, `internal/batchreport/`,
`internal/telemetry/ablation.go`, `internal/doctor/multiprovider_test.go`,
`tests/integration/conformance_suite_test.go`, `.agents/policies/`, `scripts/sync-policies.sh`,
`testdata/capability-matrix.json` — e o próprio `.specs/prd-harness-portatil-vendor-neutral/`.
`Makefile` e `.github/workflows/test.yml`, que carregam os gates novos, estão modificados e não
staged. O código compila e passa `go vet`. **Nenhuma tarefa deste PRD pode começar antes que essa
base entre no histórico**: qualquer `git clean`, `git stash` ou troca de branch a destrói.

Estado real das quatro tarefas que este documento assumia pendentes:

| Tarefa | Estado verificado |
|---|---|
| 6.0 capability matrix | **Entregue.** `internal/capability/` com 5 arquivos, 54 células em `testdata/capability-matrix.json` (claude 18, codex 13, opencode 12, copilot 11), alvo `Makefile:90-91` e step em `.github/workflows/test.yml:119-120` |
| 8.0 aliases e conflito transacional | **Entregue.** `init` e `sync` são aliases reais (`cmd/ai_spec_harness/install.go:25`, `upgrade.go:24`); `internal/txn` com journal, rollback e conflito por checksum; integrado em `install.go:305-308` e `upgrade.go:241-244`; manifesto com `Checksums map[string]string` (`manifest/manifest.go:23`) |
| 10.0 conformidade cross-provider | **Parcial.** `tests/integration/conformance_suite_test.go` é real — 4 provedores, runner único, gate-of-the-gate — mas cobre **5 dos 14 cenários** de RF-61. Faltam os cenários 5 a 14 (teste falhando, evidência ausente e inválida, checkpoint válido e corrompido, telemetria sem token, evento desconhecido, capability não suportada, erro de adapter, bypass por variação). Budget de contexto entregue em `internal/integration/input_context_budget_test.go` |
| 11.0 telemetria, ablation e doctor | **Entregue em grande parte.** `internal/doctor/doctor.go` reescrito com 15 checks estratificados por camada e blocos por provedor; `internal/doctor/multiprovider_test.go` com 17 testes; `internal/telemetry/ablation.go` com `BuildAblationBaseline` e `CompareAblation`. Falta o schema formal comum de telemetria por provedor — existe só o contrato implícito em `telemetry/parser.go:12,22` |

Duas tarefas marcadas `done` estão incompletas em relação ao que este PRD precisa: a **10.0** pelos
nove cenários faltantes, cobertos aqui por RF-61; e a **3.0**, cujo artefato principal
`.agents/harness.yaml` **não existe em lugar nenhum do repositório** e cujos campos
`GitPolicy.AutoCommit`/`AutoPush` continuam com **zero call-sites de produção**
(`internal/harness/contract.go:13-14` são apenas as declarações). O contrato carrega e é ignorado —
o elo que RF-19 exige construir permanece ausente, como a [ADR-003](adr-003-parsing-estrutural-sem-truncamento.md)
já registrava.

- Nenhuma dependência de infraestrutura externa, serviço ou rede.
- Nenhuma dependência Go nova. `jsonschema/v6` já cobre a validação do envelope.

---

## Monitoramento e Observabilidade

A telemetria do repositório é opt-in por `GOVERNANCE_TELEMETRY=1`, append-only, formato
`<ts> chave=valor`, por decisão registrada na ADR-006. Este desenho **preserva** o formato e o
opt-in; não introduz Prometheus nem Grafana, que seriam dependência sem demanda para uma CLI local.

Entradas novas, todas com `unknown` explícito quando indisponível (RF-46):

```text
hook.duration_ms hook=<nome> event=<evento> provider=<provider> value=<n>
hook.decision hook=<nome> event=<evento> provider=<provider> decision=<ALLOW|BLOCK|…>
hook.timeout hook=<nome> event=<evento> provider=<provider>
hook.ablation baseline=<com|sem> hook=<nome> duration_ms=<n>
```

Logs: os bloqueios passam a emitir `policy_id` e `gate_id` em stderr, satisfazendo RNF05. Segredos
nunca são registrados — e aqui há uma lacuna real a fechar: a sanitização existe em um único ponto
(`durable/facade.go:231`) e **não cobre** `events.jsonl`, `execution_report.md`, `tool_calls.md`, o
MEMORY.md legado nem o `.partial.md` do `post-wave.sh`, que faz `cat` literal do YAML de resultados.
RF-49 exige estender o `SanitizationPolicy` existente a esses cinco caminhos.

---

## Considerações Técnicas

### Decisões Chave

Cada decisão material tem ADR própria neste diretório:

1. [ADR-001 — Canonical Hook Contract v1 como fonte única com projeções derivadas](adr-001-hook-contract-fonte-unica-projecoes.md)
2. [ADR-002 — Resultado tipado e tradução única de exit code, fail-closed para hook crítico](adr-002-resultado-tipado-traducao-exit-code.md)
3. [ADR-003 — Parsing estrutural sem truncamento e negação por ausência de alvo](adr-003-parsing-estrutural-sem-truncamento.md)
4. [ADR-004 — Prova de dispatch discriminada por célula](adr-004-prova-dispatch-por-celula.md)
5. [ADR-005 — Enumeração de linguagens com teste de exaustividade](adr-005-exaustividade-de-linguagens.md)
6. [ADR-006 — Atomicidade e detecção de corrupção dos artefatos operacionais](adr-006-atomicidade-artefatos-operacionais.md)

### Riscos Conhecidos

| # | Risco | Impacto | Mitigação |
|---|---|---|---|
| R-01 | Relaxar `NewEnforcement` de 3 pontos fixos causa panic no init do catálogo (`registry.go:296,307`) e derruba a suíte inteira | Alto | Os 10 testes afetados estão enumerados na Fase 2. A mudança é feita em um único lote com eles |
| R-02 | Ampliar a regex de prova de teste **afrouxa** o gate — risco inverso do usual: evidências antes rejeitadas passam a ser aceitas | Médio | A regex não tem nenhum teste automatizado hoje. `scripts/test-validators.sh` ganha casos **antes** da ampliação |
| R-03 | `TEST_NAME_RE` (`validate-review-evidence.sh:46`) é `^(Test\|Benchmark\|Example)…$`. Ampliar para JUnit/xUnit/pytest/jest afrouxa o gate para **todas** as linguagens, Go inclusive | Médio | Tornar a regex condicionada à stack detectada, não global |
| R-04 | `WriteFileAtomic` cria inode novo: perde dono, grupo, ACL, xattr e **quebra hard link**. `install.go:966` espelha hooks entre cinco diretórios | Alto | Fase 7 não toca caminhos de instalação. Verificar antes se algum espelhamento usa hard link |
| R-05 | `WriteFileAtomic` **destrói symlink de destino** por rename, onde hoje escreve através dele | Médio | Auditar cada caminho migrado contra `RefuseExternalSymlink` |
| R-06 | Rename gera `RENAME`, não `MODIFY`; observadores em tempo real perdem o handle | Baixo | `agent_livewriter.go` precisa de verificação antes da Fase 7 |
| R-07 | Codex pula hooks untrusted **em silêncio** após alteração de hash (openai/codex#46210) | Alto | Não é corrigível pelo harness. Declarado como limitação; `doctor --codex-trust` já detecta e passa a ser parte do gate de release |
| R-08 | Copilot é fail-open no timeout de `preToolUse` | Médio | Declarado na matriz como `SupportAdapter` com `limitation`; nunca marcado `verified` |
| R-09 | A matriz de capabilities nova (evento × família) pode ser confundida com a existente (invariante de paridade) | Baixo | Arquivos e comandos distintos; a documentação declara os dois eixos |
| R-10 | **Todo o PRD dependente está fora do git**: 119 entradas em `git status`, zero commits, `HEAD` em `a8b7721`. `git clean`/`stash`/checkout destrói `internal/capability/`, `internal/txn/`, `internal/harness/`, `internal/conformance/` e as 13 pastas de evidência | Crítico | Commitar a base **antes** de qualquer tarefa deste PRD. É pré-requisito bloqueante, não recomendação |
| R-11 | Mudar `Kind()` de `PromptBuildEvent` e `ToolCallEvent` corrige um bug preexistente e pode alterar comportamento de consumidor não identificado | Médio | Só `dispatcher_test.go:114` usa a constante; a correção entra com teste explícito |
| R-12 | Deixar de descartar o erro em `runner.go:428,466` **passa a bloquear** onde hoje não bloqueia | Alto | Nenhum hook de produção está registrado nesses dois pontos hoje. A mudança é inerte até a Fase 6 registrar o primeiro — por isso vem antes dela, não depois |

### Conformidade com Padrões

- **R-STYLE-001** (hard): todo código novo em inglês, zero comentários, sem prefixo `_` em
  identificador Go. Os identificadores de domínio em português deste documento — Evento, Resultado,
  Política — traduzem para `Event`, `Result`, `Policy`. A rastreabilidade fica no relatório de
  execução, não no nome.
- **R-GOV-001**: precedência de regras respeitada; nenhuma referência de linguagem sobrepõe a
  governança transversal.
- **R-DDD-001**: `Result`, `Capability`, `GitOperation` e `Envelope` são VOs com construtor
  validador e campos não exportados, seguindo o padrão já estabelecido em
  `specs/enforcement.go:87-121` e `harness/contract.go`. Zero struct literal fora de teste e factory.
- **R-ERR-001**: erros sentinela por pacote (`ErrUnknownEvent`, `ErrSchemaVersion`); wrapping com
  `%w`; mensagens internas curtas, em inglês, estáveis.
- **R-SEC-001**: input de provedor tratado como não confiável (RF-57); paths normalizados; sem `eval`
  sobre input externo; segredos fora de log e evidência (RF-49). O bypass de 64 KiB é violação ativa
  desta regra e por isso está na Fase 0.
- **R-TEST-001**: table-driven; `FakeFileSystem` no unitário (ADR-002 do repositório);
  `t.TempDir()` na integração; `io.Discard` no `Printer`; sem rede real; sem `sleep`.
- **ADR-001 do repositório** (go:embed): todo asset novo precisa de mirror em
  `internal/embedded/assets/`, sob pena de o consumidor receber governança parcial.
- **ADR-004** (lazy loading): nenhuma reference nova é carregada em bloco.

**Desvio intencional registrado:** `internal/hookcontract` introduz um pacote novo em vez de estender
`internal/runtime/specs`. Alternativa aderente rejeitada: colocar os cinco eventos dentro de `specs`.
Rejeitada porque `specs` já importa `skills` e conhece identificadores de provedor, o que tornaria
RF-07 impossível de verificar por gate de AST — a regra exige que o contrato não importe nomes de
fornecedor, e um pacote que já os contém não pode provar isso.

### Arquivos Relevantes e Dependentes

**Núcleo a criar:** `internal/hookcontract/{event,result,payload,exitcode,capability}.go`,
`internal/hookpolicy/git.go`, `internal/hookaudit/`, `internal/hookinventory/`,
`internal/qualitygate/`, `.agents/lib/hook-payload.sh`, `internal/detect/toolchain_java.go`,
`internal/detect/toolchain_dotnet.go`.

**A modificar, com a linha exata:** `internal/runtime/specs/enforcement.go:13-22,139-143`;
`registry.go:17-38,77-89,288-299`; `internal/runtime/hooks/dispatcher.go:19-27,77,85`;
`internal/runtime/runner.go:428,466`; `internal/capability/evidence.go:146-149,169-173`;
`internal/runtime/persistence/jsonl.go:23,57-58`; `internal/detect/toolchain.go:99-108,148-163`;
`internal/detect/detect.go:88,108-118`; `internal/detect/framework.go:125-166`;
`internal/contextgen/contextgen.go:275-310,339-340,364-386,397-417`;
`internal/triggers/triggers.go:59-66,78-98`; `internal/skills/skills.go:88,91,95,154-162`;
`internal/install/install.go:1337-1342,1357-1404,1596-1609`;
`internal/taskloop/orchestrator.go:456-512`; `internal/taskloop/isolation.go:322`;
`internal/runtime/approval_adapters.go:124,192`; `internal/embedded/embedded.go:61`;
`internal/doctor/doctor.go:78-100`; `cmd/ai_spec_harness/flags.go:85`;
`cmd/ai_spec_harness/install.go:43`.

**Shell a modificar:** `.agents/scripts/git-operation-gate.sh`, `.agents/lib/parse-hook-input.sh`,
`.agents/hooks/validate-preload.sh`, `.agents/scripts/validate-task-evidence.sh:180`,
`.agents/scripts/validate-review-evidence.sh:45-46`, `.agents/hooks/post-wave.sh:45,58-70`,
`.opencode/plugin/governance.js`.

**Espelhos obrigatórios** (`make check-skills-sync check-hooks-sync check-scripts-sync`):
`.claude/`, `.codex/`, `.github/`, `internal/embedded/assets/`. A investigação encontrou dois itens
**fora de qualquer lista de sync** — `scripts/git-hooks/pre-commit` e `.github/hooks/governance.json`
— e um espelho ausente: `check-invocation-depth.sh` não existe em
`internal/embedded/assets/scripts/lib/`. RF-05 cobre a correção dessas três lacunas.

**Testes a criar ou estender:** os enumerados na seção de rastreabilidade, mais
`scripts/test-validators.sh` (que hoje tem **zero** referências ao gate Git) e
`scripts/test-hooks.sh`.
