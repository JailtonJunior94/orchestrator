<!-- spec-hash-prd: 0d7928814324ab79c927b439febb697240266a932d7b59b34357f5ebf98ca57e -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Harness Portátil e Vendor-Neutral

## Resumo Executivo

Esta entrega converte um harness que **funciona** em um harness que é **verificável**. O baseline
`v2.0.1` já instala governança para quatro CLIs, já aplica enforcement por hooks, já modela o Ciclo de
Aprovação e já valida evidência de forma fail-closed. O que falta é a propriedade que a User Story pede:
um contrato de engenharia com fonte canônica única, declarado em vez de emergente, distribuível de forma
reversível, e cuja equivalência entre provedores seja **provada por artefato versionado** em vez de
afirmada em prosa.

A abordagem é deliberadamente conservadora em arquitetura e agressiva em verificação. Nenhum pacote
existente é reescrito; nenhum contrato público de CLI muda. Cinco componentes novos são introduzidos —
validador de contrato, par de espelhamento de policies, gerador de capability matrix, camada
transacional de escrita e gate canônico de operação Git — e cada um se conecta ao mecanismo equivalente
que o repositório já opera, em vez de criar um paralelo. A regra que orienta todas as decisões de
desenho é: **onde já existe um gate, estender; onde não existe, criar seguindo o padrão do que existe.**

Três achados da fase de inventário redefiniram o escopo em relação à leitura inicial da User Story, e
estão refletidos aqui: o enforcement de auto-commit **não existe** (é prosa em SKILL.md, não código);
parte dos invariantes de paridade é **auto-satisfeita por construção** e não pode sustentar célula de
matriz; e os testes E2E de paridade **não rodam em nenhum job de CI**. Especificar em cima dessas
premissas sem corrigi-las produziria exatamente o falso positivo que a entrega existe para eliminar.

## Arquitetura do Sistema

### Visão Geral dos Componentes

Componentes **novos** (todos em `internal/`, um pacote por responsabilidade, conforme `AGENTS.md`):

| Componente | Responsabilidade | RFs |
|---|---|---|
| `internal/harness` (planejado) | Carregar, validar e resolver o Harness Contract v1. Parse estrito com rejeição de campo desconhecido; contrato default embarcado quando o arquivo não existe. | RF-02..RF-08 |
| `internal/capability` (planejado) | Derivar a capability matrix dos invariantes já existentes, emitir os dois artefatos (JSON canônico + Markdown renderizado) e expor o resultado ao gate de sincronia. | RF-18..RF-21 |
| `internal/install/txn` (planejado) | Transação de escrita em lote: staging, detecção de conflito por `path → checksum`, commit atômico e rollback do lote. | RF-24..RF-28 |

Componentes **modificados**:

| Componente | Mudança | RFs |
|---|---|---|
| `internal/manifest` | Campo aditivo `omitempty` com `path → checksum` dos arquivos instalados. Pré-requisito duro de todo o bloco D. | RF-24 |
| `internal/install` | Relatório estruturado com cinco categorias; consumo da transação; distribuição de `code-style.md`. | RF-23.1, RF-12 |
| `internal/upgrade` | Aplicação transacional; fim do `RemoveAll` antes de `CopyDir`; correção do fail-open de hash. | RF-27, RF-28 |
| `internal/doctor` | Agregação em blocos Core + um por provedor, consumindo `install.Verify` como biblioteca em vez de reimplementar. | RF-30..RF-34 |
| `internal/parity` | Saneamento dos invariantes auto-satisfeitos; escopo derivado de `AppliesTo`. | RF-18.1 |
| `internal/telemetry` | Chaves novas na escrita **e** extensão do leitor, sem a qual o campo é gravado e invisível. | RF-43.1, RF-44 |
| `internal/metrics` | Medição do contexto de entrada por provedor, com numerador declarado por provedor. | RF-41, RF-41.1 |

Artefatos **canônicos** novos fora de `internal/`:

| Artefato | Papel |
|---|---|
| `.agents/harness.yaml` (planejado) | Contrato v1 no projeto consumidor. Ausência aplica o default embarcado. |
| `.agents/policies/` (planejado) | Origem canônica única das regras transversais. `.claude/rules/` passa a ser derivado. |
| `scripts/sync-policies.sh` (planejado) e `scripts/check-policies-sync.sh` (planejado) | Quarto par de espelhamento, dedicado. |
| `.agents/scripts/git-operation-gate.sh` (planejado) | Gate canônico de operação Git, herdado pelos quatro provedores. |

### Relacionamentos e Fluxo de Dados

A cadeia de confiança da entrega, da origem ao provedor:

```text
.agents/policies/  .agents/skills/  .agents/scripts/   ← origem canônica única
        │                │                │
        │   scripts/{sync,check}-*-sync.sh (4 pares)   ← gate: divergência falha o build
        ▼                ▼                ▼
.claude/  .github/  .codex/  .opencode/  internal/embedded/assets/
        │
        │   go:embed → ExtractToTempDir → install/upgrade (transacional)
        ▼
             projeto consumidor
        │
        ├── .agents/harness.yaml  → internal/harness (strict)  → doctor / gates
        ├── .agents/scripts/*.sh  → os 4 provedores por registry.go:11-14
        └── .ai_spec_harness.json → path → checksum → detecção de conflito
```

Dois pontos merecem destaque porque são onde a arquitetura atual falha hoje:

1. **A origem canônica é parcial.** `internal/embedded/assets/.claude/rules/` contém apenas
   `governance.md`; `code-style.md` nunca chega ao consumidor. O elo quebrado não é o asset — é o
   `CopyFile` de nome hardcoded em `internal/install/install.go:834-839`, replicado em
   `internal/upgrade/upgrade.go:399-402` e na lista fixa de `internal/uninstall/uninstall.go:594`.
2. **A escrita não tem transação.** A atomicidade existe por arquivo, em `WriteFileAtomic`
   (`internal/fs/fs.go:173-203`), mas `WriteFile`, `CopyFile` e `CopyDir` escrevem direto
   (`internal/install/write_tracker.go:104-134 (baseline pre-Tarefa 8.0, consolidado em internal/tracking/tracker.go)`). O pior caso observável é
   `internal/upgrade/upgrade.go:193-194`: `RemoveAll` seguido de `CopyDir`, onde a falha do copy deixa a
   skill **destruída**, emite apenas `Warn` e o laço continua, podendo terminar com exit 0.

## Design de Implementação

### Interfaces Chave

Contrato do harness — carregamento e validação, com erro tipado e sem dependência de filesystem real:

```go
package harness

type Contract struct {
    Version  int
    Git      GitPolicy
    Approval ApprovalPolicy
    Quality  QualityPolicy
    Evidence EvidencePolicy
    Skills   SkillsPolicy
}

type Loader interface {
    Load(projectDir string) (Contract, Source, error)
}

type Validator interface {
    Validate(raw map[string]any) error
}
```

`Source` distingue `SourceFile` de `SourceDefault`, para o diagnóstico reportar honestamente que o
projeto não declara contrato em vez de fingir que declara.

Capability matrix — derivação e renderização a partir de fonte única:

```go
package capability

type Cell struct {
    Capability string
    Provider   skills.Tool
    Support    SupportLevel
    Evidence   EvidenceRef
}

type Generator interface {
    Generate(invariants []parity.Invariant, agents []specs.Agent) (Matrix, error)
}

type Renderer interface {
    RenderJSON(m Matrix) ([]byte, error)
    RenderMarkdown(m Matrix) ([]byte, error)
}
```

`EvidenceRef` nomeia o teste que sustenta a célula. Célula com `Support` de suportado e `EvidenceRef`
vazia é violação — é exatamente a condição que RF-20 manda falhar.

Transação de escrita em lote, decorando o `FileSystem` já existente:

```go
package txn

type Transaction interface {
    Stage(path string, content []byte) error
    Conflicts() []Conflict
    Commit() error
    Rollback() error
}

type Conflict struct {
    Path     string
    Expected string
    Actual   string
}
```

### Modelos de Dados

Extensão aditiva do manifesto, seguindo o precedente já testado de `installed_files`/`merged_files`:

```go
type Manifest struct {
    InstalledFiles  []string          `json:"installed_files,omitempty"`
    MergedFiles     []string          `json:"merged_files,omitempty"`
    FileChecksums   map[string]string `json:"file_checksums,omitempty"`
}
```

`FileChecksums` é a **dependência dura** de RF-24, RF-26 e RF-27. O campo `Checksums` existente é
indexado por nome de skill, guarda o hash de um único `SKILL.md`, e é write-only: escrito em
`internal/install/install.go:1582-1593` e `internal/upgrade/upgrade.go:638`, e lido por nenhum código de
produção. Sem `path → checksum`, "conflito" não é decidível e o requisito seria falso positivo por
construção.

Relatório de instalação, hoje inexistente como tipo — só há saída textual por `internal/output`:

```go
type FileOutcome string

const (
    OutcomeCreated   FileOutcome = "created"
    OutcomeUpdated   FileOutcome = "updated"
    OutcomePreserved FileOutcome = "preserved"
    OutcomeMerged    FileOutcome = "merged"
    OutcomeConflict  FileOutcome = "conflict"
)
```

A precedência entre categorias é explícita, não implícita: `conflict` vence todas; `merged` vence
`created`. Isso corrige a precedência implícita atual de `writeTracker.record`
(`internal/install/write_tracker.go:49-58 (baseline pre-Tarefa 8.0, consolidado em internal/tracking/tracker.go)`), onde `MarkMerged` remove o path de `created` sem que a
regra esteja declarada.

### Endpoints de API

Não aplicável: a superfície é CLI, não serviço. As mudanças de superfície são:

| Comando | Mudança | Compatibilidade |
|---|---|---|
| `ai-spec init` | Alias de `install` | Aditivo |
| `ai-spec sync` | Alias de `upgrade` | Aditivo |
| `ai-spec doctor` | Blocos Core + 4 provedores | Saída muda; exit code mantém 0/1 |
| `ai-spec verify` | Nenhuma | Contrato preservado integralmente |

## Pontos de Integração

Não há integração com serviço externo nesta entrega. Há integração com **quatro CLIs de terceiros**, e
o contrato com cada uma já está modelado em `internal/runtime/specs/registry.go`. Duas restrições
herdadas permanecem invioláveis: a detecção **nunca executa binário**
(`internal/detect/agent.go`, R-SEC-001), e a verificação que exige comunicação com um CLI permanece
opt-in explícito, fora do caminho estático.

As credenciais dos provedores não são exigidas para instalar, sincronizar ou diagnosticar. Os cenários
que exigem CLI real vivem no fluxo nightly, com gate explícito de credencial antes de qualquer execução.

## Monitoramento e Observabilidade

O repositório não expõe métricas Prometheus nem dashboards Grafana para a CLI; a observabilidade é por
telemetria opt-in append-only (ADR-006) e por saída estruturada de comando. Esta entrega preserva esse
modelo e o estende em dois pontos:

1. **Telemetria comparável.** As chaves de RF-44 são acrescentadas ao formato `chave=valor` já vigente.
   A qualificação decisiva é que **escrever a chave não a torna observável**: o parser atual reconhece
   apenas `skill=` e `ref=` e descarta todo o resto sem `default`
   (`internal/telemetry/parser.go:50-57`). Todo campo que precise ser lido exige estender o parser e o
   registro de entrada — isso é escopo, não consequência automática.
2. **Atribuição de falha.** Toda falha de diagnóstico e de conformidade nomeia a camada responsável —
   `core`, `instalação`, `adapter`, `provider` ou `validação` — como campo da saída, não como inferência
   do usuário. É o que RNF04 pede e o que hoje não existe.

Métrica que o provedor não expõe é **ausente**, nunca zero e nunca inferida. Isso vale nominalmente para
`context_loaded`, que só existe onde o provedor o expõe e que **nunca** é usado como gate: um gate com
força variável por provedor é a capability universal silenciosa que RF-21 proíbe.

## Abordagem de Testes

Três restrições verificadas no repositório governam todo o desenho de teste desta entrega, e nenhuma
delas é presumida:

1. **`FakeFileSystem` não é simulador de filesystem.** `WriteFileAtomic` (`internal/fs/fake.go:145-148`)
   é alias byte-idêntico de `WriteFile` (`:140-143`) — sem temp file, sem `rename`, sem ponto de falha
   intermediário, contra as quatro etapas da implementação real (`internal/fs/fs.go:173-204`). E
   **nenhuma escrita do fake pode falhar**: as seis operações de escrita retornam `nil`
   incondicionalmente e sequer consultam o mapa `NoWrite`; o único gancho é `Writable`
   (`internal/fs/fake.go:219-221`), que a produção precisa chamar **antes** de escrever.
2. **Um passo de CI pode existir e não executar nada.** `.github/workflows/test.yml:53-57` filtra
   `-run TestSnapshot` contra um teste chamado `TestContextgen_Snapshots`: exit 0, zero testes (V-33).
   Nesta entrega **não** se afirma que snapshots são validados em CI, e todo gate novo nasce com teste
   de mutação.
3. **Gate sem prova de falha é decoração.** O padrão canônico é
   `cmd/ai_spec_harness/catalog_sync_test.go:27-34`: injeta divergência artificial e exige o erro
   sentinela. Todo gate novo replica essa forma.

### Estratégia por bloco

| Bloco | Nível dominante | Por quê |
|---|---|---|
| A — Contrato | Unitário + fuzz | Validação é função pura de bytes → decisão tipada; nenhum IO real |
| B — Canonicalização | Integração | O objeto é um par de scripts e o estado do repositório; só falseável mutando árvore real |
| C — Capability matrix | Unitário (geração) + integração (gate do gate) | Geração é pura; sincronia gerado↔commitado exige árvore |
| D — Distribuição | Unitário (decisão) + integração com filesystem real (transação) | A decisão de conflito é pura; a transação é propriedade do filesystem |
| E — Doctor | Unitário com fake + integração para exit code | Agregação é lógica pura; exit code é contrato de processo |
| F — Conformidade | Integração (determinístico) + live (nightly) | A classificação normativa do PRD mapeia 1:1 em build tag |
| G — Telemetria | Unitário | Escrita e parse são funções puras sobre linhas `chave=valor` |
| H — Não-regressão | Gates existentes + contrato de CLI + guards de registro | Não se cria suíte nova: reforça-se que a existente roda |

### Testes Unitários

**Convenções herdadas, obrigatórias.** `testify/suite` + tabela de cenários com campo `name`
(`internal/config/resolver_test.go`, `internal/approval/translator_test.go`); `FakeFileSystem` em vez
do FS real; `output.New(false)` ou `io.Discard` para o `Printer`; mocks gerados só para interfaces
declaradas em `mockery.yml`, com drift barrado por `make check-mocks` (`scripts/check-mocks.sh`).
As interfaces novas desta entrega entram em `mockery.yml` — e isso é passo **manual sem rede de
segurança**, porque o gate detecta mock desatualizado mas é cego a interface nova não declarada
(V-39). Para fronteira de filesystem não se usa mock: usa-se `FakeFileSystem` (ADR-002).

#### Bloco A — contrato (RF-02..RF-08)

Pacote novo `internal/harness` (planejado), testado com fixtures próprias. Cenários table-driven:
campo desconhecido, tipo inválido, `version` ausente, `version: 2`, contrato ausente (aplica default
embarcado e reporta `default`, RF-07), contrato válido mínimo.

**O ponto crítico — provar fail-closed sem falso positivo.** A armadilha já está em produção:
`internal/skills/schema.go` e `internal/agents/schema.go` projetam o documento num struct tipado
**antes** de validar. Campo desconhecido é descartado no `json.Marshal` do struct, e o
`additionalProperties: false` de `internal/agents/agent-frontmatter.schema.json` valida o vácuo — o
schema declara estrito e nunca exercita a cláusula. O contraexemplo correto também já existe:
`internal/sdd/result_schema.go` leva os bytes brutos direto a `jsonschema.UnmarshalJSON`, e é por isso
que `evals/sdd/fixtures/schema/02-unknown-field.execution.reject.json` é de fato rejeitada.

A ponte do contrato é: `yaml.Unmarshal` → `map[string]any` → `json.Marshal` →
`jsonschema.UnmarshalJSON` → `Validate` → **só então** decodificação tipada. O guardião que falha se
essa ponte for removida tem três pernas: (1) o payload preserva a chave desconhecida — falha se
alguém trocar a ponte por struct tipado; (2) o schema rejeita esse payload — prova que a rejeição vem
do schema; (3) a projeção tipada **seria aceita** — prova que a perna 2 não é tautológica. Se a perna
3 falhar um dia, o argumento anti-vácuo mudou e a perna 2 precisa ser rederivada conscientemente.

Complementos: fuzz do parser registrado em `Makefile` e no step `Fuzz` de `.github/workflows/test.yml`
(o comentário nesse step registra que BUG-130 escapou por semanas porque o passo não falhava o job);
teste de colisão de chave contrato × config (RF-05); cenário em `internal/config/resolver_test.go`
fixando que chave desconhecida **continua** resolvendo (RF-06 — o endurecimento não pode vazar do
contrato para o `config.yaml`); e, para RF-08, a própria assinatura do construtor, que não recebe
`http.Client`, `exec.Cmd` nem `LookPather`.

#### Bloco B — canonicalização (RF-09..RF-14)

Unitário cobre pouco: o objeto é arquivo e script. O que é unitário é a **distribuição** (RF-12):
`internal/install/install_test.go` ganha cenário que instala com `FakeFileSystem` e exige
`R-STYLE-001` presente no destino — o defeito V-01 é exatamente uma ausência que nenhum teste
observava.

#### Bloco C — capability matrix (RF-15..RF-21)

Geração determinística (gerar duas vezes produz bytes idênticos), escopo derivado de `AppliesTo` e
não de `Level` (V-26), e recusa de invariante auto-satisfeito (V-25) como sustentação de célula
suportada. Este último é o teste que impede o falso positivo institucionalizado: alimenta o gerador
com um invariante cuja verificação é satisfeita pelo próprio `Checker.Generate` e exige que a célula
**não** seja marcada suportada. O gate de célula-sem-teste **estende**
`internal/runtime/specs/parity_gate.go:30-72`, que já retorna `"no dispatch proof test associated"`;
a extensão é de dados — novas entradas no `dispatchProofRegistry` — não de mecanismo.

#### Bloco D — conflito e transação (RF-22..RF-29)

**Achado que muda o desenho (V-32):** o `FakeFileSystem` não falha escrita. `NoWrite`
(`internal/fs/fake.go`) só é consultado por `Writable`; `WriteFile` e `WriteFileAtomic`
(`internal/fs/fake.go:140-148`) gravam incondicionalmente e retornam `nil`. Fazer `WriteFile` honrar
`NoWrite` mudaria o comportamento de toda a base de testes existente — risco desproporcional. A
solução é um decorador **local ao teste**, no mesmo padrão de composição que `writeTracker` já usa ao
embutir `fs.FileSystem`: uma struct que embute o fake, conta escritas e falha a partir da n-ésima.
Nada em produção muda, e o duble compartilhado permanece intocado.

Cenários table-driven sobre o ponto de falha: falha na primeira escrita do lote; no meio; na última;
ao gravar o manifesto; e **durante o rollback** — este último exigindo erro que nomeia o que não foi
revertido, nunca engolido. A asserção de "estado idêntico" compara snapshot do conjunto de arquivos
ordenado e hasheado antes e depois, não arquivo a arquivo, senão reversão parcial passa.

Conflito: arquivo não gerenciado presente no destino não é sobrescrito nem removido (RF-26); arquivo
gerenciado com checksum divergente vira categoria `conflito` e aborta o lote (RF-24/RF-27); com a
flag de sobrescrita, aplica e nomeia cada arquivo; regra de governança editada à mão cai na mesma
máquina, sem caso especial (D-19). O campo `path → checksum` é aditivo e `omitempty`, com fixture de
manifesto da `v2.0.1` provando leitura retrocompatível.

#### Bloco E — doctor (RF-30..RF-34)

`internal/doctor/doctor_test.go` é estendido, não substituído — ele já injeta `FakeFileSystem` com
`NoWrite`. Cenários novos: um bloco por provedor; atribuição de falha a
`core`/`instalação`/`adapter`/`provider`/`validação` como campo **tipado**, não string; exit code ≠ 0
em invariante obrigatório; ausência total de rede. `verify` ganha teste de não-regressão explícito
que fixa flags e saída atuais.

#### Bloco G — telemetria e ablation (RF-43..RF-47)

O parser hoje entende só `skill=` e `ref=` (V-29, `internal/telemetry/parser.go:50-57`). Cada chave
nova que precise ser **lida** ganha cenário de round-trip escrita→leitura; chave desconhecida
continua ignorada sem erro (ADR-006). **Métrica ausente ≠ zero** (RF-45): cenário que emite evento
sem `context_loaded` e exige que `report`/`summary`/`trend` **omitam** o campo — é a regressão mais
provável do bloco. Mais tabela de campos sensíveis que falha se qualquer um for emitido (RF-46), e
comparação `baseline` × `baseline + componente` determinística, sem LLM (RF-47).

#### Prova de transação: as duas saídas consideradas, e por que uma foi rejeitada

**(a) Estender o `FakeFileSystem` com injeção de falha determinística por path.** Viável — exigiria que
as operações de escrita consultassem um mapa de falhas. Custo real: o fake passaria a ser um **segundo
simulador de atomicidade**, cuja fidelidade em relação à implementação real não é verificada por nada.
Se adotada, a extensão seria entregável próprio com teste próprio, e **ainda assim** não provaria que o
`rename` publica atomicamente — provaria que o fake devolve o erro que o fake foi mandado devolver.

**(b) Prova de RF-28 em teste de integração com filesystem real em diretório temporário.** É o padrão já
estabelecido no repositório: `internal/parity/e2e_parity_test.go:42-50` monta o serviço de instalação
real com as mesmas dependências de produção.

**Decisão: (b) para RF-28; (a) rejeitada.** A propriedade sob teste — "o disco nunca fica em estado
parcial" — é propriedade do filesystem, não da lógica de negócio. Provada contra um simulador, ela vale
o que o simulador vale, e o simulador atual vale zero (V-32). Contra filesystem real em diretório
temporário a prova é direta e barata: nenhum container, nenhuma dependência nova, padrão já em CI.

A **decisão de conflito** continua em teste unitário com o fake, e isso é legítimo porque é decisão
pura: `FileHash` do fake computa SHA-256 real sobre o conteúdo em memória, então a comparação
`manifesto[path].checksum` × `FileHash(path)` é genuinamente exercitada. O que muda de nível é só a
**escrita**. Esta distinção é declarada explicitamente porque confundir as duas é como se produz um
teste verde que não prova nada.

A falha é induzida por permissão real no disco — diretório ou arquivo somente-leitura num path do meio
da ordem determinística de escrita —, variando qual arquivo falha entre primeiro, meio e último.
**Falhar apenas no último é o caso que mais frequentemente passa por acaso.** As asserções: erro
retornado, todos os hashes idênticos ao estado inicial, **nenhum arquivo temporário remanescente** (a
implementação real cria `.tmp-*` no diretório de destino) e relatório nomeando o que foi revertido.

⚠️ **Divergência de registro a corrigir (V-41).** `Makefile:26-27` roda cinco pacotes de integração;
`.github/workflows/test.yml:159` roda apenas três. Pacote novo com teste `integration` colocado fora
desses três **não roda no gate de PR**. Todo alvo novo desta entrega vai para um dos três, ou o workflow
é corrigido na mesma tarefa — e RF-20.1 já exige essa correção para o pacote de paridade.

### Testes de Integração

> **Decisão necessária: este projeto precisa de integration tests?** — **Sim, e já os tem; esta
> entrega os amplia, e explicitamente NÃO adota testcontainers.**

Aplicando os três critérios do template a este caso:

- **Fronteiras de IO críticas onde mocks não garantem correção — sim.** Não há banco, fila ou cache,
  mas há três fronteiras equivalentes em criticidade: o **filesystem real** (permissão, atomicidade,
  ordem de rename — invisíveis ao `FakeFileSystem`, V-32), o **processo externo bash** (os hooks
  canônicos são scripts executados) e o **runtime Node** (o plugin OpenCode é ESM carregado por um
  harness `.mjs`). Nenhuma das três é decidível por mock: o objeto sob teste é o próprio artefato
  instalado.
- **Incidente em que unit passou e a integração real falhou — sim, documentado três vezes.** V-24: os
  E2E de paridade são verdes à mão e nenhum job os executa. V-33: o passo de snapshot da CI filtra um
  nome de teste inexistente e passa com zero testes. E `parity_dispatch_proof_test.go:138-184` registra
  um terceiro: prova de disparo por regex sobre texto era forjável por `t.Logf`.
- **Custo proporcional ao risco — sim, e marginal.** A infraestrutura existe (`//go:build integration`,
  `t.TempDir()`, clones baratos por `cp -Rl`), roda em ubuntu-24.04 e macos-15, e **não** provisiona
  serviço algum. Testcontainers seria dependência nova sem demanda concreta — proibido pelo modo de
  trabalho #4 e pela Restrição Técnica 2.

Três critérios em "sim". Integration tests são adotados, com `//go:build integration`, **sem**
testcontainers e sem serviço externo. Os testes que exigem CLI real ficam fora deste nível, sob
`acp_live` e `hooks_live`.

#### Gate do gate — um por gate novo

Cada gate novo recebe seu par adversarial, no molde de `tests/integration/sync_gates_guard_test.go`,
que já prova verde-no-clone-intocado e vermelho-sob-mutação.

**1. `policies-sync` (RF-11).** Mutações exigidas no clone: apagar a policy canônica → falha; apagar só
o espelho → falha; **editar só o espelho** → falha; editar só a origem → falha; apagar o diretório
canônico inteiro → falha. A terceira linha é a que dá sentido a RF-11 ("editar o derivado deixa de ser
possível sem quebrar CI"); sem ela o gate detecta apenas remoção. Complemento do Gate de Fase 2: teste
que compara o conteúdo de `.claude/rules/*.md` pós-migração com o de `v2.0.1` e exige **identidade
byte-a-byte** — a mudança é de origem, não de destino (RF-10).

**2. Capability matrix (RF-19/RF-20).** Alterar uma célula no JSON → falha, com a mensagem nomeando o
comando de regeneração. Alterar **apenas** o Markdown → falha: provar os dois artefatos separadamente é
o que justifica D-17. Reajustar largura de coluna regenerando → **passa**, provando que a comparação não
é de prosa formatada. Marcar célula suportada cujo teste não está no registro de prova → falha. Renomear
um teste de prova sem atualizar o registro → falha.

**3. Gate de operação Git (RF-40.1/RF-40.2).** É o gate novo mais sensível, porque hoje não existe
enforcement algum (V-23). A asserção é **tripla**, no molde de
`tests/integration/hooks_matrix_dispatch_test.go:110-133` — exit code, ausência do marcador de cadeia
quebrada e presença do veredito do validador canônico —, porque exit ≠ 0 sozinho também é o que um
wrapper quebrado produz, e wrapper quebrado lido como gate funcionando é precisamente o falso positivo
que essa asserção existe para matar.

| Entrada | Exigência nos 4 provedores |
|---|---|
| `git commit -m "x"` não solicitado | Negado, com veredito do validador canônico |
| `git push` não solicitado | Negado |
| `git status`, `git diff`, `git log` | **Permitido** — o gate não pode virar bloqueio de leitura |
| `rm -rf /` **com** `GOVERNANCE_PRELOAD_CONFIRMED=1` | **Negado** (RF-40.2). Hoje é liberado |
| Neutralizar o script canônico no clone | O teste adversarial **falha** — gate do gate |

A linha `rm -rf /` **inverte a asserção de um teste hoje verde**
(`tests/integration/opencode_shell_gate_test.go:96-116`). Esse teste precisa ser **particionado**:
comando inócuo sem alvo continua liberado sob preload confirmado; comando destrutivo passa a ser negado
independentemente. Alterar a asserção sem particionar seria regressão de RF-48 disfarçada de requisito
novo.

#### Conformidade cross-provider, nível determinístico (RF-35/36/38/39)

Manifesto da suíte replicando a tabela normativa de classificação do PRD — revisável em diff, como
RF-35 exige —, com teste do próprio manifesto que falha se a classificação divergir do PRD. **Fixture
única medida por invariante** (RF-38): a mesma entrada percorre os quatro provedores e a asserção é
sobre decisão (exit code, veredito, arquivo produzido), jamais sobre igualdade textual de resposta.
RF-39 vira campo tipado: `core` / `adapter` / `provider` / `indeterminado` — e `indeterminado` é valor
legítimo e declarado, não lacuna.

Os cenários de configuração inválida e skill adulterada ganham também fixtures no formato
`*.reject.json` / `*.accept.json` que `scripts/test-sdd-evals.sh` já consome, herdando a matriz 2×2
(TP/TN/FP/FN), `escape_rate` e os thresholds de `evals/sdd/manifest.json` — hoje
`maximum_escape_rate: 0.0` e `minimum_quality_rate: 1.0`. Adicionar categoria exige tocar **dois**
lugares: `required_categories` no manifesto e o laço de categorias no script, que exige ≥ 2 fixtures por
categoria. Esquecer um dos dois produz gate parcialmente ativo.

#### Budget de contexto de entrada (RF-41/41.1/42)

Alvo `make budget`, já em CI. Herda integralmente a convenção vigente: mapa de tetos explícito, **margem
de 10% sobre o medido** (V-22) e mensagem de falha informando valor medido, teto e valor sugerido. O
budget **por skill** não é tocado.

O ponto que decide se este gate vale alguma coisa é RF-41.1: o numerador **não** pode sair do prompt
construído pelo harness, porque OpenCode e Codex carregam `AGENTS.md` e skills nativamente, fora de
`Job.Prompt` (V-30). O numerador é a lista declarada de arquivos que cada provedor carrega na entrada, e
essa lista tem teste próprio: para cada provedor, um caso que falha se um arquivo carregado nativamente
não estiver declarado. Sem esse teste, o gate fica verde enquanto o contexto real de dois dos quatro
provedores cresce — o falso positivo mais caro do Bloco F. **Sinal de alarme durante a implementação:
se o numerador dos quatro provedores for idêntico, a medição está derivando do prompt.**

#### Registro obrigatório — anti-gate-órfão

Um gate que existe e não roda é pior que gate ausente, porque produz confiança. Todo gate novo só é
considerado entregue quando aparece nos **quatro** registros abaixo — e os dois últimos são listas
hardcoded que nenhum mecanismo atualiza sozinho:

| # | Registro | Onde |
|---|---|---|
| 1 | Alvo `make` | `Makefile`, incluindo `.PHONY` |
| 2 | Step de CI | `.github/workflows/test.yml` — mais `./internal/parity/...` no job `integration` por RF-20.1 |
| 3 | Lista de scripts e diretórios do guardião | `tests/integration/sync_gates_guard_test.go:16-24,26-33` |
| 4 | Lista de alvos que rodam em CI | `tests/integration/sync_gates_guard_test.go:424-437` |

`TestNoOrphanValidatorTestSuites` (`:370-421`) cobre apenas `tests/scripts/*_test.sh` e só conta linha
de receita iniciada por TAB e fora de comentário — menção textual não é invocação. Registros 3 e 4 são
hardcoded e **não têm guarda de segunda ordem**: são checklist de execução da tarefa, e a evidência de
que foram feitos entra no relatório de execução (RF-57).

Registros adicionais conforme a área tocada: `mockery.yml` para interfaces novas — passo manual **sem
rede de segurança**, porque o gate é cego a interface não declarada (V-39); `scripts/check-package-coverage.sh`
para os pacotes novos; `Makefile` e o step `Fuzz` para o fuzz do parser de contrato; `docs/cli-schema.json`
para `init` e `sync`, cujo par com o Cobra já é verificado nos dois sentidos por
`cmd/ai_spec_harness/cli_contract_test.go` — RF-22 fica satisfeito por esse teste existente, sem arquivo
novo.

### Testes E2E

E2E aqui significa **CLI real, autenticada, fora do caminho crítico de merge** — separação já
estabelecida em V-15 e reafirmada por D-04 e pela Restrição Técnica 8. Não há frontend; não há
automação de browser.

Os testes ponta a ponta que usam `acpfake` já rodam no job `integration` e são gate; continuam como
estão. O nível **live** (RF-37) roda sob `hooks_live`, exigindo `AISPEC_HOOKS_LIVE=1` e as quatro CLIs
configuradas, no workflow nightly em ambiente protegido, e **reporta sem bloquear merge**. O papel dele
é estreito e precisa ficar estreito: confirmar que a CLI real ainda **roteia** a operação pelo ponto
instrumentado. O bloqueio em si já é provado deterministicamente por RF-36/RF-40 — se o live fosse a
única prova, regressão em política crítica apareceria no dia seguinte, sem bloquear merge (D-14). O caso
live só aprova em `dispatched && !mutated`, no padrão de `assertPreToolOutcome`
(`tests/integration/hooks_live/live_test.go:152-171`), que distingue três estados de falha e rejeita
inferência por ausência.

**Política de release (RF-56), que é o que dá sentido à separação:** regressão em cenário determinístico
bloqueia merge; regressão em qualquer gate do baseline bloqueia release; falha em cenário live não
bloqueia nenhum dos dois e é reportada no relatório de release. Essa política é **verificada por teste
de configuração**, não por convenção: um caso falha se um cenário classificado como determinístico não
estiver executando em `test.yml`, e falha se um cenário classificado como live estiver.

**Não-regressão do baseline (RF-48/49/50):** o conjunto hoje verde é o critério de saída da Fase 8 —
`make test`, `integration`, `lint`, `vet`, `coverage`, `coverage-packages`,
`check-skills-sync check-hooks-sync check-scripts-sync`, `check-mocks`, `test-hooks`, `test-validators`,
`check-spec-paths`, `test-portable-skills`, `budget`, `test-sdd-evals`. Nenhum é reescrito por esta
entrega. RF-50 ganha teste próprio: fixture de projeto instalado na `v2.0.1` — manifesto sem
`path → checksum`, sem contrato declarado — contra o binário novo, exigindo operação normal sem ação do
usuário.

### Rastreabilidade — RF × nível × arquivo

Arquivos sem sufixo já existem; `(planejado)` ainda não existe.

| RF | Nível | Arquivo |
|---|---|---|
| RF-01 | Artefato (verificado por RF-13) | inventário de regras `(planejado)` |
| RF-02, RF-04, RF-07, RF-08 | Unitário | pacote de contrato `(planejado)` |
| RF-03 | Unitário + fuzz | guardião de ponte de três pernas `(planejado)`; alvo de fuzz no `Makefile` |
| RF-05 | Unitário | teste de colisão de chave contrato × config `(planejado)` |
| RF-06 | Unitário | `internal/config/resolver_test.go` (leniência preservada) |
| RF-09, RF-16 | Integração (gate) | `scripts/check-policies-sync.sh` `(planejado)` |
| RF-10, RF-11, RF-13 | Integração (gate do gate) | guardião de policies `(planejado)`; `tests/integration/sync_gates_guard_test.go` |
| RF-12 | Unitário | `internal/install/install_test.go` |
| RF-14, RF-54 | Decisão registrada, sem objeto | PRD, *Fora de Escopo* |
| RF-15 | Integração | `tests/integration/pretool_decision_parity_test.go` |
| RF-17, RF-18, RF-18.1, RF-21 | Unitário + golden | gerador de matriz `(planejado)`; artefatos JSON e Markdown `(planejado)` |
| RF-19 | Integração (gate do gate) | guardião de matriz `(planejado)` |
| RF-20 | Unitário + integração | `internal/runtime/specs/parity_gate.go`, `parity_dispatch_proof_test.go` |
| RF-20.1 | CI | `.github/workflows/test.yml` (incluir `./internal/parity/...`); `internal/parity/e2e_parity_test.go` |
| RF-22 | Unitário | `cmd/ai_spec_harness/cli_contract_test.go`, `docs/cli-schema.json` |
| RF-23 | Unitário | `internal/install/install_test.go` |
| RF-23.1, RF-24, RF-26, RF-27 | Unitário | testes de conflito `(planejado)`; teste de manifesto `(planejado)` |
| RF-25 | Unitário | `internal/upgrade/upgrade_test.go` |
| RF-28, RF-29 | Unitário + integração | testes de transação `(planejado)` com decorador de falha `(planejado)` |
| RF-30, RF-31, RF-34 | Unitário | `internal/doctor/doctor_test.go` |
| RF-32, RF-33 | Unitário | testes de doctor por provedor `(planejado)` |
| RF-35, RF-36, RF-38, RF-39, RF-56 | Integração | suíte de conformidade determinística `(planejado)` + manifesto `(planejado)` |
| RF-37 | E2E (live, nightly) | caso live de operação Git `(planejado)`; `.github/workflows/hooks-live.yml` |
| RF-40, RF-40.1, RF-40.2 | Integração + E2E | suíte de gate Git `(planejado)`; particionamento de `tests/integration/opencode_shell_gate_test.go` |
| RF-41, RF-41.1, RF-42 | Integração (budget) | budget de contexto de entrada `(planejado)` |
| RF-43, RF-43.1, RF-44, RF-45 | Unitário | `internal/telemetry/parser_test.go`, `internal/telemetry/acp_test.go` |
| RF-46 | Unitário | teste de redação de campos sensíveis `(planejado)` |
| RF-47 | Unitário | mecanismo de ablation `(planejado)` |
| RF-48 | CI (baseline) | `.github/workflows/test.yml` e `Makefile` integrais |
| RF-49, RF-52, RF-53 | Integração | `tests/integration/portability_test.go` |
| RF-50 | Integração | fixture de projeto instalado na `v2.0.1` `(planejado)` |
| RF-51 | Gate de script | `scripts/check-spec-paths.sh` |
| RF-55 | Integração | teste de ausência de git implícito `(planejado)` |
| RF-57 | Evidência | relatório de execução por tarefa; `make test-validators` |
| RF-58 | Unitário | decorador de falha **local ao teste** `(planejado)`; `internal/fs/fake.go` **não** é alterado (V-32) |
| RF-59 | CI + gate do gate | corrigir o filtro `-run` em `.github/workflows/test.yml`; mutation test por gate novo, molde de `cmd/ai_spec_harness/catalog_sync_test.go` |
| RF-60 | Integração (gate) | teste de integridade de `skills-lock.json` `(planejado)` + alvo `make` e step de CI que executem a recomputação (V-34) |
| RF-61 | Unitário + integração | fonte única de lista entre `internal/install/install.go` e `internal/uninstall/uninstall.go`; round-trip em `internal/uninstall/roundtrip_test.go` |
| RF-62 | Unitário | promoção dos helpers de delegação para `internal/runtime/specs/delegation.go`; `internal/install/hooks_parity_matrix_test.go` vira consumidor (V-36) |
| RF-63 | Regra documental | verificado por revisão; nenhuma afirmação de capacidade herdada sem âncora |

## Sequenciamento de Desenvolvimento

### Ordem de Build

A ordem deriva das dependências técnicas **reais** entre requisitos, não da numeração dos RFs. Um passo
só abre quando o gate de saída do passo do qual depende fecha. O faseamento do PRD continua valendo como
agrupamento de governança; esta é a ordem de execução.

Três princípios governam a ordem: **instrumento antes de entrega** — gate que não executa não protege
nada, e há dois casos empíricos disso (V-24, V-33); **substrato probatório antes de prova** — não se
testa transação com duble que não pode falhar (V-32); **mecanismo antes de teste adversarial** — não se
testa o que não existe (V-23).

**P0 — Reparo da instrumentação de CI (RF-20.1, RF-59, RF-60, RF-63).** Incluir `./internal/parity/...` com
`-tags=integration` no job `integration`; corrigir o filtro `-run` do passo de snapshot, que hoje faz um
passo passar sem executar nada (V-33). Depende de nada — é o primeiro por decisão explícita. **Por quê:**
V-24 registra que os E2E de paridade existem, passam à mão e nenhum job os executa; todo passo
subsequente altera código coberto por eles. Inclui ainda RF-60 (a verificação de
integridade de skills passa a ser executada por `Makefile` e CI, fechando V-34) e RF-63 (registro
explícito de que snapshots, integridade de skills e métricas por fluxo **não** eram capacidades
herdadas). *Gate de saída:* os nomes dos testes aparecem **nominalmente no log** do job de CI —
ausência de nome no log é falha, não sucesso; cada gate reparado falha sob injeção deliberada de
divergência, provado por execução e não por inspeção visual; editar um `SKILL.md` sem atualizar o lock
derruba o CI.

**P0.1 — Base probatória de filesystem e fail-closed de hash (RF-58).** Ou o duble ganha injeção de
falha explícita, ou a prova de RF-28 passa a viver em teste de integração com filesystem real (V-32);
propagação dos erros de hash hoje descartados; desambiguação de `DirHash`, que devolve `("", nil)` para
não-diretório e torna "inexistente" e "vazio" indistinguíveis (V-38). *Gate:* teste que injeta falha no
meio de uma sequência de escrita e observa erro propagado; teste que distingue diretório ausente de
vazio; nenhum caminho produz sucesso a partir de duas leituras falhas. **Precede P8** — sem isto, o
teste de reversão passa sem exercer reversão nenhuma.

**P1 — Inventário de regras (RF-01).** Classifica cada regra como universal ou específica de fornecedor.
Entrada obrigatória do Bloco B: sem a classificação, "regra universal com mais de uma origem" não é
decidível. *Gate:* inventário commitado; o subconjunto `universal` fechado é a lista de entrada de P4.

A partir daqui abrem-se **cinco frentes paralelas**.

**P2 — Harness Contract v1 (RF-02..RF-08).** Schema embarcado, `sync.OnceValues`, validador stateless,
ponte YAML→JSON **preservadora**, erros tipados, default embarcado, gate de não-duplicação de chave.
Não depende de `internal/config/` — a ADR-001 decide explicitamente não depender do resolver leniente.
*Gate:* chave desconhecida produz erro e exit ≠ 0; **o teste falha quando a ponte preservadora é
removida** (se passa nos dois modos, está validando o vácuo); `config.yaml` com campo desconhecido
continua sendo aceito; este repositório declara o próprio contrato (self-dogfooding).

**P3 — Canonicalização de policies (RF-09..RF-11) — indivisível com o registro anti-gate-órfão.**
Origem canônica com as duas regras movidas **byte a byte**; espelho gerado; par dedicado de scripts com
lista **declarada, nunca glob**; registro no `Makefile` **e** no `test.yml` **e** na lista hardcoded do
guardião. *Indivisibilidade:* fechar P3 sem os três registros é entregar a aparência do gate sem o gate.
*Gate:* o gate falha ao editar o espelho **e** ao apagar o canônico; conteúdo carregado pelo Claude Code
byte-idêntico ao anterior; os três pares existentes permanecem verdes — evidência de que a decisão de
não generalizar foi respeitada.

**P4 — Distribuição de `R-STYLE-001` (RF-12, RF-13, RF-61).** Depende de P3 e de tocar **quatro** pontos
onde o caminho é declarado por nome, não por diretório: o `CopyFile` hardcoded do install, o
`syncFileIfPresent` do upgrade, a lista fixa do uninstall (V-35) e a invariante CL05 da paridade. Os
quatro **no mesmo lote**: asset embarcado sem os demais entrega uma regra e não a outra, sem erro; e
omitir o uninstall deixa o arquivo órfão impedindo o prune do diretório. RF-61 torna a lista **fonte
única** entre install e uninstall, com round-trip exigindo árvore limpa. *Ordem interna de RF-13:*
primeiro o teste que cobre o comportamento governado pela duplicata, **depois** a remoção.

**P5 — Saneamento dos invariantes (RF-18.1).** Escopo passa a derivar de `AppliesTo`, não de `Level`
(V-26); invariantes satisfeitos pelos stubs que o próprio gerador injeta (V-25) são marcados como
evidência inválida. **Precede RF-18** porque sanear depois significaria publicar e então corrigir a
matriz — e o artefato commitado é a superfície pública do compromisso vendor-neutral. *Gate:* teste que
amarra o conjunto de stubs injetados ao conjunto marcado como evidência inválida — stub novo não
classificado falha o build.

**P6 — Capability matrix e seus gates (RF-17..RF-21).** Um modelo em memória, **dois emissores na mesma
execução**; gate golden-file; extensão de `ValidateParityMatrix` preservando a literal
`"no dispatch proof test associated"`; gate do gate. Depende de P5 (fonte saneada) e de P2 (a célula de
contrato só existe depois que o contrato existe). *Estados honestos obrigatórios:* suportado com teste
resolvido, não suportado, capability de provedor e **desconhecido** — `EvaluateHandshake`, enquanto
stub, produz desconhecido, nunca suportado. *Gate:* contagem de células suportadas sem teste resolvido é
**zero**; contagem de desconhecidas é rastreada e justificada célula a célula.

**P7 — Fundação do manifesto: `path → checksum` e hash fail-closed.** Campo aditivo `omitempty`;
correção dos `sourceHash, _ :=` e do retorno mudo de `recordCopiedTree`; install e upgrade passam a
gravar o checksum por path, e o upgrade passa a usar o `writeTracker` hoje exclusivo do install.
**Pré-requisito duro** de RF-24/26/27: sem ele, conflito não é proposição decidível e o requisito viraria
falso positivo por construção (V-08). A detecção atual descarta erro de hash e produz `"" == ""` →
**falso `StatusOK`**: fail-open exatamente onde precisa ser fail-closed. *Gate:* manifesto legado e novo
convivem; nenhuma falha de leitura de hash produz `StatusOK`; existe leitura de produção do campo novo.

**P8 — Conflito, transação e aliases (RF-22..RF-29, RF-23.1, RF-58).** Categoria `conflict` com
precedência **explícita em código**; relatório estruturado com cinco categorias; transação por staging +
promoção + journal; abort-on-conflict por default; aliases `init`/`sync`. Depende de P7 estritamente. Os
aliases são independentes e entregáveis a qualquer momento após P0. RF-58 entra aqui: o decorador de
falha local ao teste precede qualquer prova de RF-28 — não se testa transação com duble que não falha.
*Gate:* injeção de falha no meio do lote resulta em árvore **idêntica** à inicial; o padrão `RemoveAll` →
`CopyDir` deixa de existir como operação irreversível; o atalho que pula a escrita do manifesto é
removido; nenhuma flag ou saída existente alterada.

**P9 — Gate canônico de operação Git (RF-40.1, RF-40.2).** Script canônico fail-closed com exit 2 e
diagnóstico próprio; delegação a partir do hook canônico; plugin OpenCode consultando o mesmo critério
(sem alterar o conjunto de ferramentas mutantes, pois `bash` já pertence a ele); escapes próprios
auditados; registro nos dois gates de espelhamento. Independente de P2–P8. **Precede P10:** não se testa
o que não existe (V-23). *Gate:* com preload confirmado, operação Git não solicitada permanece
**bloqueada**; o teste que hoje libera destrutivo sob preload é particionado.

**P10 — Suíte de conformidade cross-provider (RF-35..RF-40).** Runner **único** abstraindo `bash` e
harness Node, generalizando o único teste que hoje cobre os quatro provedores; asserção tripla; caso live
confirmatório. Depende de P9, P2, P6 e P11. *Gate:* a suíte **falha quando o gate é removido** —
validar por remoção deliberada antes do merge, nunca por inspeção visual; os quatro provedores retornam
o mesmo exit code.

**P11 — Budget de contexto de entrada (RF-41, RF-41.1, RF-42).** Depende de P3/P4: o conjunto de regras
auto-carregadas muda com a canonicalização, e medir antes produziria baseline obsoleto no mesmo dia.
*Restrição dura de numerador:* não pode derivar apenas do prompt do harness (V-30). *Gate:* teste que
prova que o numerador de OpenCode e Codex inclui os arquivos carregados nativamente.

**P12 — Telemetria e ablation (RF-43..RF-47).** Estender o **leitor**, não só a escrita (V-29): escrever
a chave não a torna observável. *Gate:* chaves novas emitidas **e visíveis** no relatório; nenhum leitor
existente quebra; zero métrica inventada ou preenchida com zero.

**P13 — Doctor multi-provider (RF-30..RF-34).** É o **agregador** — vem depois de tudo que agrega. A
alocação de RF-60 é explícita para não gerar retrabalho: a **execução automática** da verificação de
integridade (alvo `make` + step de CI) pertence a **P0**, porque R-12 exige que ela esteja no lugar
**antes** de o doctor depender dela; P13 apenas **consome** o resultado no bloco Core. Sem essa ordem, o
bloco Core reportaria integridade que nada garante (V-34). *Gate:* `doctor` reporta os quatro provedores; exit
≠ 0 em falha obrigatória; somente verificações estáticas; `verify` preserva integralmente contrato,
flags, saída e códigos de saída.

**P14 — Fechamento (RF-48..RF-57, RF-62, RF-63).** Todos os gates verdes; testes de alternância entre
CLIs, de ausência de daemon e de ausência de git implícito; política de release declarada; documentação
reconciliada; evidência persistida; changelog **minor**. RF-62 é isolado **num passo próprio** porque
endurecer o gate de delegação pode deixar a árvore vermelha antes de ficar verde — não pode estar no
caminho crítico de outra entrega.

### Dependências Técnicas

| # | Dependência | Razão factual |
|---|---|---|
| D-a | `path → checksum` **precede** RF-24/26/27 (P7 → P8) | V-08: sem hash por path, conflito não é decidível |
| D-b | RF-18.1 **precede** RF-18 (P5 → P6) | V-25 e V-26: derivar de invariante vacuoso ou de `Level` produz matriz errada por construção |
| D-c | RF-02 **precede** a célula de contrato (P2 → P6) | Sem contrato definido, a célula não tem objeto |
| D-d/e | RF-09/RF-10/RF-11 **precedem** RF-12 (P3 → P4) | Só há o que distribuir depois que origem e espelho existem |
| D-f | RF-12 exige tocar **quatro** pontos no mesmo lote | Install, upgrade, uninstall (V-35) e CL05 — parcial é silencioso |
| D-g | RF-40.1 **precede** RF-36/RF-40 (P9 → P10) | V-23: o gate não existe; não se testa o que não existe |
| D-h | RF-20.1 e RF-59 **precedem** tudo (P0) | V-24 e V-33: são os gates que protegem o resto |
| D-i | Registro anti-gate-órfão é **indivisível** de cada gate novo | Guards de lista hardcoded não detectam gate ausente do CI |
| D-j | RF-41.1 exige numerador por provedor | V-30: OpenCode e Codex carregam fora de `Job.Prompt` |
| D-k | RF-43.1 exige estender o parser | V-29: o leitor reconhece apenas duas chaves |
| D-l | RF-58 **precede** qualquer prova de RF-28 | V-32: duble que não falha não prova transação |
| D-m | RF-13: teste **antes** da remoção da duplicata | Exigência textual do próprio RF-13 |

**Paralelização.** Após P0 e P1, cinco frentes independentes: **Contrato** (P2), **Policies** (P3→P4),
**Paridade/Matriz** (P5→P6), **Distribuição** (P7→P8) e **Gate Git** (P9). Duas colisões a coordenar por
ordem de merge: P4 e P8 tocam os mesmos arquivos de install e upgrade; P4 e P5 tocam ambos o arquivo de
paridade. Entregáveis a qualquer momento após P0, de baixo acoplamento: os aliases `init`/`sync`, a
correção do filtro `-run`, e a decisão documental de não criar o diretório de workflows.
**Convergências obrigatórias:** P10 abre quando P9, P2, P6 e P11 fecham; P13 quando P2, P3, P7 e P8
fecham; P14 quando tudo fecha.

**Caminho crítico:** `P0 → P1 → P3 → P4 → P11 → P10 → P13 → P14`. Concentra três acoplamentos em série:
a canonicalização altera o conjunto de regras auto-carregadas, que redefine o numerador do budget, que
decide um cenário da suíte, cujo resultado o `doctor` agrega. Contrato e Gate Git têm folga; a frente de
Distribuição também, desde que P7 comece cedo, já que destrava a maior massa de trabalho.

**Passo de maior risco: P8.** É o único que reescreve o caminho de escrita em produção, substituindo por
staging + journal um padrão hoje destrutivo e irreversível. O passo que conserta o pior caso é também o
que mais pode piorá-lo, porque o erro se materializa **no disco do consumidor**, não em um gate.
Contenções obrigatórias: teste de injeção de falha resultando em árvore idêntica; journal registrando a
ordem de promoção **antes** de qualquer escrita destrutiva, permitindo retomada manual com paths
nominados; relatório que nomeia o que foi e o que não foi revertido; benchmark antes e depois. Segundo em
risco: **P4**, que atinge 26 call sites — mitigado estruturalmente por nenhum hook shell referenciar o
caminho, então um call site esquecido degrada orientação, não quebra execução.

## Considerações Técnicas

### Decisões Chave

| ADR | Decisão em uma frase |
|-----|----------------------|
| [ADR-001 — Harness Contract v1 com parse estrito](adr-001-harness-contract-v1-strict.md) | A política vive em arquivo próprio, separado do `config.yaml` operacional e lido por parser estrito com schema embarcado, porque política é fail-closed e *tuning* é best-effort — misturá-los exigiria um parser híbrido, origem clássica de falso positivo. |
| [ADR-002 — Canonicalização de policies com par dedicado](adr-002-canonicalizacao-policies-espelhamento.md) | A origem canônica única das regras transversais migra para `.agents/`, `.claude/rules/` vira espelho gerado, e o espelhamento ganha par **dedicado** em vez de generalizar os três pares que protegem a entrega inteira. |
| [ADR-003 — Capability matrix gerada, em JSON + Markdown](adr-003-capability-matrix-gerada.md) | A matriz é gerada dos invariantes — após saneamento obrigatório — e commitada em JSON (que o gate compara) e Markdown (que o mantenedor lê), **estendendo** o gate de evidência existente em vez de criar validador paralelo. |
| [ADR-004 — Sync transacional com conflito explícito](adr-004-sync-transacional-conflito.md) | O manifesto passa a registrar `path → checksum`, conflito vira categoria de primeira classe que aborta o lote por default, e a aplicação vira transacional, eliminando `RemoveAll` → `CopyDir` como operação irreversível. |
| [ADR-005 — Gate canônico de operação Git](adr-005-gate-operacao-git-destrutividade.md) | A proibição de commit e push não solicitados deixa de ser prosa e vira script canônico delegado a partir do hook que os quatro provedores já compartilham, com destrutividade avaliada **independentemente** do estado de preload. |

### Riscos Conhecidos

**R-01 — Falso positivo do schema estrito: a ponte YAML→JSON descarta chaves antes da validação.**
*Impacto: crítico.* O antipadrão já está em produção: `internal/skills/schema.go:52-64` projeta em struct
tipado antes de validar, e o `additionalProperties: false` valida o vácuo. Se o contrato seguir esse
caminho, RF-03 reporta sucesso exatamente no caso que deveria bloquear. *Mitigação:* ponte obrigatoriamente
preservadora (`map[string]any` + normalização recursiva de chaves para string, já que YAML admite chave
não-string); projeção tipada só **depois** da validação; reuso daquele caminho é proibido nesta entrega.
*Detecção precoce:* o teste deve **falhar** quando a ponte preservadora é removida — rodar esse
experimento de remoção deliberada antes do merge. Teste que passa nos dois modos está validando o vácuo.

**R-02 — Derivar capability matrix de invariante auto-satisfeito (V-25/V-26).** *Impacto: alto.* Uma
célula suportada derivada de invariante que o próprio gerador satisfaz afirma que o gerador escreveu o
que o gerador escreve — tautologia publicada como evidência, no artefato que é a superfície pública do
compromisso vendor-neutral. *Mitigação:* RF-18.1 como passo anterior à geração; escopo por `AppliesTo`;
invariante auto-satisfeito aparece na matriz, mas nunca com suporte declarado. *Detecção precoce:* teste
que amarra os stubs injetados ao conjunto marcado como evidência inválida; caso adversarial no gate do
gate.

**R-03 — Gate novo nasce órfão.** *Impacto: alto.* O guardião de suítes cobre apenas
`tests/scripts/*_test.sh` e a lista de alvos de CI é hardcoded: gate com alvo no `Makefile` e ausente do
`test.yml` não é detectado por guard nenhum. É a mesma classe de V-24 e V-33. Esta entrega cria **três**
gates novos, triplicando a exposição. *Mitigação:* registro indivisível da entrega de cada gate, nos
quatro pontos. Não usar `-run` com nome parcial: executar o pacote inteiro. *Detecção precoce:* remoção
deliberada antes do merge — o gate precisa ficar vermelho; inspeção visual do YAML não conta.

**R-04 — `RemoveAll` antes de `CopyDir` destruindo skill em falha.** *Impacto: alto e irreversível no
disco do consumidor.* Hoje a cópia falhada destrói a skill, emite apenas aviso, o laço continua e a
execução pode terminar com exit 0; e o atalho de retorno antecipado pode nem reescrever o manifesto.
*Mitigação:* staging + promoção + journal; ordem de promoção registrada **antes** de qualquer escrita
destrutiva; relatório nomeando o que foi e o que não foi revertido; `recordCopiedTree` deixa de retornar
em silêncio — em regime transacional, árvore não rastreável invalida o lote. *Detecção precoce:* teste de
injeção de falha no meio do lote escrito **junto** do passo, não depois; teste de falha durante a própria
reversão; benchmark antes e depois.

**R-05 — Telemetria: chave escrita mas invisível ao parser (V-29).** *Impacto: médio, mas silencioso.* O
sintoma é o pior possível: a entrega parece feita, o arquivo tem os dados, e o relatório mostra vazio —
indistinguível de "métrica ausente", que é estado legítimo por RF-45. *Mitigação:* RF-43.1 coloca a
extensão do leitor dentro do escopo. *Detecção precoce:* round-trip por campo lendo **via o parser de
produção**, não por inspeção do arquivo; asserção que distingue "ausente por indisponibilidade do
provedor" de "ausente por parser cego".

**R-06 — Medição de contexto subestimar OpenCode e Codex (V-30).** *Impacto: alto.* Numerador derivado
do prompt mede uma fração do contexto real desses dois provedores: o gate fica verde enquanto o contexto
cresce. Pior que não ter gate, porque cria confiança injustificada. *Mitigação:* RF-41.1 — numerador é o
conjunto de arquivos que **aquele** provedor carrega, declarado e testado; declarado é gate, observado é
telemetria e nunca gate. *Detecção precoce:* se o numerador dos quatro provedores for idêntico, a
medição está derivando do prompt.

**R-07 — Migração de `.claude/rules/` quebrar os 26 call sites.** *Impacto: médio a alto, contido.*
*Mitigante estrutural verificado:* **nenhum hook shell referencia o caminho** — a carga é declaração
documental por ferramenta, não wiring executável, então call site esquecido em documentação degrada
orientação, não quebra execução. *Mitigação:* primeiro os Go que quebram teste, depois a documentação; o
espelho mantém caminho e conteúdo byte-idênticos; rollback é reversão de commit, sem migração de dados.
*Detecção precoce:* rodar os testes imediatamente após a movimentação dos arquivos, antes de qualquer
outra alteração — eles falham na hora e nomeiam o call site.

**R-08 — Aprovação por vacuidade nos gates.** *Impacto: alto.* Gate sobre glob aprova diretório apagado;
gate de matriz aprova matriz vazia. O verde significa "não havia nada a comparar". *Mitigação:* lista
declarada nos dois scripts do par, nunca glob; ausência do canônico é drift explícito — restrição já
registrada no gate de skills e herdada. *Detecção precoce:* apagar o canônico e exigir vermelho; gerar
matriz vazia e exigir reprovação — casos permanentes, não experimentos manuais.

**R-09 — Falso positivo de conflito forçando uso rotineiro da flag de sobrescrita.** *Impacto: médio.*
Conflito por normalização de linha, permissão ou symlink vira fricção e gera pressão para desativar o
mecanismo — o modo de falha que mata gates fail-closed. *Mitigação:* hash sobre bytes com regra única
documentada; manifesto legado sem o campo significa "sem rastreio", **nunca** divergência, senão todo
projeto instalado antes desta entrega classificaria tudo como conflito. *Detecção precoce:* casos por
variação de EOL e permissão; teste de manifesto legado exigindo zero conflitos.

**R-10 — Cadeia de delegação quebrada lendo como gate funcionando.** *Impacto: alto.* O hook pode
retornar exit 2 por erro de invocação, não por decisão de política. *Mitigação:* asserção **tripla** em
toda a suíte; diagnóstico canônico próprio amarrado ao script emissor pelo guard sem build tag; runner
único exigindo o mesmo exit code dos quatro. *Detecção precoce:* quebrar deliberadamente a delegação e
exigir vermelho; acompanhar o log de escapes — volume crescente indica falso positivo ou fluxo legítimo
mal modelado.

**R-11 — Consolidar helpers de delegação afrouxando o gate (V-36).** *Impacto: alto.* A produção aceita
menção textual onde o teste exige posição de execução. Consolidar na direção errada — teste passando a
consumir a produção — **afrouxaria** o gate silenciosamente. *Mitigação:* RF-62 fixa a direção: promover
a lógica do teste para produção, nunca o contrário; passo isolado, fora do caminho crítico. *Detecção
precoce:* esperar que o gate fique **mais vermelho antes de ficar verde** — se consolidar e nada falhar,
a promoção provavelmente não aconteceu.

**R-12 — Integridade de skills nunca recomputada (V-34).** *Impacto: alto.* `SKILL.md` editado sem
atualizar o lock passa em `make test`, em `go test ./...` e na CI inteira; RF-31 reportaria integridade
que nada garante. *Mitigação:* RF-60 põe a verificação automática no `Makefile` e no CI **antes** de o
doctor depender dela. *Detecção precoce:* editar deliberadamente um `SKILL.md` sem regerar o lock e
exigir CI vermelha — validação por quebra, não por inspeção.

**R-13 — Ablation medir o que nunca foi medido (V-37).** *Impacto: médio.* `Report.Flows` é criado
vazio e nunca populado, e o subsistema que o preencheria não tem chamador de produção. RF-47 entregaria
um baseline construído sobre um mapa vazio, apresentando ausência de dado como linha de base.
*Mitigação:* RF-63 — onde a capacidade não existe, é criada por requisito explícito ou o escopo é
declarado reduzido. *Detecção precoce:* assertar no baseline que toda métrica usada tem escritor de
produção identificado; métrica sem escritor entra como **ausente**, nunca como zero.

**R-14 — `DirHash` confunde "inexistente" com "vazio" (V-38).** *Impacto: médio.* Um destino apagado
hasheia igual a um destino vazio e pode produzir sucesso onde deveria haver conflito ou erro — fail-open
no ponto que P8 torna crítico. *Mitigação:* desambiguar em P0.1, antes de qualquer consumidor novo.
*Detecção precoce:* teste com três resultados distintos para ausente, vazio e com conteúdo.

**R-15 — `make check-mocks` é cego a interface nova (V-39).** *Impacto: baixo-médio.* As interfaces
novas entram sem mock e sem sinal, empurrando testes para doubles ad-hoc. *Mitigação:* declarar em
`mockery.yml` toda interface nova **no mesmo commit** que a introduz. *Detecção precoce:* revisão de
diff — interface exportada nova sem entrada correspondente é achado bloqueante, porque nenhum gate o
pega.

**R-16 — Teste de integração novo fora do gate de PR (V-41).** *Impacto: médio.* O alvo do `Makefile`
roda cinco pacotes de integração; o CI roda três. Pacote novo fora desses três não executa no gate.
*Mitigação:* todo alvo novo vai para um dos três, ou o workflow é corrigido na mesma tarefa.
*Detecção precoce:* conferir o nome do teste no log de CI — a mesma disciplina que R-03 exige.

### Conformidade com Padrões

| Regra | Severidade | Como incide |
|---|---|---|
| **R-STYLE-001.1 — Código em inglês** | hard | Todo identificador, pacote, nome de arquivo, mensagem de erro e nome de teste em inglês. **Armadilha:** os termos de domínio deste PRD são PT-BR e **não migram literalmente** — "conflito" → `conflict`, "política" → `policy`, "espelho" → `mirror`, "lote" → `batch`. A rastreabilidade PRD→código fica no relatório de execução, nunca no identificador. Exceção única desta entrega: a literal `"no dispatch proof test associated"`, que é contrato já publicado e é preservada — exige justificativa no relatório. |
| **R-STYLE-001.2 — Zero comentários** | hard | Vale para Go **e** para os shells novos: shebang é permitido, nada além dele. Diretivas obrigatórias não são comentários e são necessárias aqui: `//go:embed` do schema e do asset default, `//go:build integration`. Ao editar arquivo existente, remover comentários **nas linhas tocadas** — incide nominalmente no comentário estrutural de `tests/integration/hooks_matrix_dispatch_test.go:114-117`, que P10 toca. |
| **R-STYLE-001.3 — Sem prefixo underline** | hard | `_` isolado como blank identifier é permitido — o que importa aqui porque P7 **elimina** justamente os `sourceHash, _ :=`: a correção não é renomear, é parar de descartar o erro. |
| **R-GOV-001 — Governança de regras** | hard | É objeto **e** norma da entrega: migra para a origem canônica em P3. Incide em três pontos: toda alteração justificável pelo PRD ou por necessidade técnica demonstrável — fecha a porta para refatoração oportunista em P4 e P8; relatórios com arquivos alterados, validações, riscos residuais e suposições; e **não executar git destrutivo sem pedido explícito** — a regra que P9 transforma de prosa em código executável. |
| **R-SEC-001 — Segurança** | hard | Quatro incidências: RF-46 (nenhum segredo persistido, com teste que falha se campo sensível for emitido); entrada externa não confiável — contrato de consumidor validado antes do uso, com validação estática pura (RF-08: sem rede, sem binário de provedor, sem LLM); sem ações destrutivas não solicitadas — literalmente o que P9 passa a executar; e o log de escape é artefato de auditoria e não pode conter segredo. |
| **R-TEST-001 — Testes** | hard para correção e determinismo | Validadores de input com teste unitário — o teste de chave desconhecida é critério de aceite, não complemento; caminho feliz **e** falha, com o "gate do gate" como forma canônica; determinismo sem estado global — ordenação da matriz fixada por teste, `sync` determinístico; cobertura nos gates de 75% total e 70% por pacote crítico; doubles de subprocesso sob runner único; e a regra de ordem de RF-13: teste **antes** da remoção. |
| **R-ERR-001 — Tratamento de erros** | hard | A regra mais acionada, porque três riscos acima são erros silenciados: sem silenciar (`sourceHash, _ :=` e o retorno mudo de `recordCopiedTree` violam isso hoje); wrapping com `fmt.Errorf("context: %w", err)` e texto em inglês; erros de domínio como **tipos distintos** — campo desconhecido, tipo inválido e versão incompatível não são o mesmo erro; e mensagens que dizem o que falhou, onde e qual ação é possível — o erro de versão nomeia encontrada, suportada e ação; a falha do gate documenta o comando de regeneração; o conflito nomeia o arquivo e aponta a origem canônica. |

Incidentes e **não** reabertos: ADR-001 (assets via `go:embed`), ADR-002 (`FakeFileSystem` em unitário),
ADR-003 (invariantes semânticas vs diff textual), ADR-005 (`skills-lock.json` SHA-256), ADR-006
(telemetria opt-in append-only), ADR-008 (invariantes de paridade) e ADR-020 (OpenCode ACP).
Documentação em PT-BR; Conventional Commits com tipo em inglês e corpo em português.

### Arquivos Relevantes e Dependentes

**Criados:** contrato do repositório e pacote de contrato (P2); as duas policies canônicas e o par de
scripts de espelhamento (P3); asset de `code-style.md` embarcado (P4); os dois artefatos de capability
matrix (P6); script canônico de operação Git (P9); suíte de conformidade (P10). Explicitamente **não**
criado: o diretório de workflows — RF-14 e RF-54, com gatilho de reabertura registrado.

**Modificados:** `.github/workflows/test.yml` e `Makefile` (P0, P3, P6, P10);
`tests/integration/sync_gates_guard_test.go` (P3, P9, P10); `internal/parity/parity.go` e
`parity_test.go` (P4, P5); `internal/runtime/specs/parity_gate.go` e `delegation.go` (P6, P14);
`internal/manifest/manifest.go` (P7); `internal/install/install.go` e `write_tracker.go` (P4, P7, P8);
`internal/upgrade/upgrade.go` (P4, P7, P8); `internal/uninstall/uninstall.go` (P4);
`cmd/ai_spec_harness/root.go` e `internal/output/output.go` (P8); `.agents/hooks/validate-preload.sh`,
`.opencode/plugin/governance.js`, `scripts/check-scripts-sync.sh` e `scripts/sync-skills.sh` (P9);
`.github/workflows/hooks-live.yml` (P10); `internal/contextgen/contextgen.go` e
`internal/taskloop/agent.go` (P11); `internal/telemetry/parser.go` e `acp.go` (P12);
`internal/doctor/doctor.go` (P13); os arquivos de governança por ferramenta e a documentação (P4, P14).

**Dependentes — não são alvo, mas precisam ser verificados:** os testes de install e upgrade que
referenciam o caminho das rules, que quebram na movimentação e **são o sinal de detecção precoce** de
R-07; `internal/manifest/task_2_0_test.go`, precedente de campo aditivo cujo contrato frágil porém
deliberado de `HasFileTracking()` precisa continuar valendo; `internal/install/hooks_parity_matrix_test.go`,
que permanece como está e serve de referência de mecanismo de prova — se mudar, ela e a matriz gerada
passam a contar histórias diferentes; `internal/runtime/specs/registry.go`, que declara o validador
canônico único e é o que faz o gate de P9 ser herdado sem linha por provedor — mudança ali desfaz a
premissa; `EvaluateHandshake`, cujo estado de stub mascara suporte real se sair do stub sem regeneração
da matriz; o padrão golden-file do contextgen, que o gate de sincronia replica; o budget por skill, que
permanece intocado e fornece a convenção de margem; os testes que hoje cobrem três e quatro provedores,
que P10 generaliza; `cmd/ai_spec_harness/lint.go`, único consumidor de paridade em produção, que lê
apenas `Warnings()` (V-27) e cujo comportamento é declarado **não impactado**; `cmd/ai_spec_harness/verify.go`,
cujo contrato é preservado integralmente; `internal/config/resolver.go` e `runtime.go`, que **não** são
alterados (RF-06) e são dependentes no sentido inverso — o gate de não-duplicação compara seus nomes de
chave contra o schema do contrato; e os cinco scripts dos três pares existentes, que **não** são
generalizados — que permaneçam verdes é o sinal de que a decisão foi respeitada.
