<!-- spec-hash-prd: a358518c8d5309b2420e56f2bb4d2e4261fe2accf956f33fafbcf0f32882187d -->
<!-- MANDATÓRIO: preenchido por `create-technical-specification` Etapa 7.1 com sha256 do PRD consumido.
     Rastreabilidade: `create-tasks` e `execute-task` comparam este hash com o atual do prd.md
     para detectar drift entre techspec e PRD. NÃO remover este comentário ao editar a techspec. -->

# Especificação Técnica — Agent Registry Declarativo

> **PRD consumido**: [prd.md](./prd.md) (spec-version 1)
> **ADR materiais**: [011-agent-registry-declarativo](../adr/011-agent-registry-declarativo.md)
> **Fase**: 1 de 5 (adaptação ao padrão Compozy)

## Resumo Executivo

Introduz um novo package `internal/agents/` que descobre, valida e resolve agentes declarativos (`AGENT.md` com frontmatter YAML) em runtime. O design espelha intencionalmente `internal/core/agents/` do Compozy, mas se acopla ao catálogo `internal/runtime/specs/` existente em vez de substituí-lo: um `ResolvedAgent` **produz** uma `specs.Spec` via mapping `runtime.ide → specs.Claude()/...`, preservando a fonte de verdade do Launcher e a tabela de compatibilidade.

A entidade `Agent` é opt-in via nova flag CLI `--agent <name>` (mutuamente exclusiva com `--tool` e com flags do modo avançado). Quando ausente, todo o fluxo atual `--tool/--model` permanece intacto, garantindo retrocompatibilidade total e preservação das invariantes forenses (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e do `ActivityWatchdog`, que não sofrem modificação nesta fase.

## Arquitetura do Sistema

### Visão Geral dos Componentes

**Novos**
- `internal/agents/agent.go` — Value object `ResolvedAgent` + tipos `Metadata`, `RuntimeDefaults`.
- `internal/agents/discovery.go` — Descoberta recursiva em `~/.ai-harness/agents/` e `<workdir>/.ai-harness/agents/`.
- `internal/agents/registry.go` — Interface `Registry` (Discover/Resolve) + implementação default com cache por construtor.
- `internal/agents/schema.go` — JSON Schema embarcado para frontmatter de Agent + validação.
- `internal/agents/agent-frontmatter.schema.json` — JSON Schema embedded via `go:embed`.
- `internal/agents/precedence.go` — `applyRuntimePrecedence(cfg *RuntimeOverride, defaults RuntimeDefaults)` espelhando Compozy.
- `internal/agents/prompt.go` — `BuildAgentBlocks(agent *ResolvedAgent, catalog []ResolvedAgent) (metadataBlock, catalogBlock string)`.
- `.specs/adr/011-agent-registry-declarativo.md` — ADR justificando declarativo-sobre-código, escopo global+workspace, precedência runtime e coexistência com `specs.Claude()`.

**Modificados**
- `internal/skills/frontmatter.go` — Extrair função genérica `ParseFrontmatterFields(content []byte) map[string]string` (linhas a refatorar: 23-67). Manter `ParseFrontmatter`/`ParseFrontmatterName` como wrappers que consomem a função genérica, garantindo zero regressão para skills.
- `internal/taskloop/taskloop.go` — Campo `AgentName string` em `Options` (~linha 20-42); injeção opcional de `*agents.ResolvedAgent` no fluxo.
- `internal/taskloop/profile.go` — Nova regra em `ResolveProfiles` e/ou função adjacente `ResolveProfileFromAgent(agent *agents.ResolvedAgent, cliOverrides RuntimeOverride) (*ProfileConfig, error)`.
- `internal/taskloop/agent.go` — `BuildPromptContext` aceita `*agents.ResolvedAgent` opcional e enriquece prompt na ordem: template base → metadata → catálogo → corpo do AGENT.md.
- `cmd/ai_spec_harness/task_loop.go` — Nova flag `--agent <name>`; validação de exclusividade com `--tool` e flags do modo avançado.

**Inalterados (invariante)**
- `internal/runtime/persistence/*` — Persistência forense intacta.
- `internal/runtime/watchdog.go` — ActivityWatchdog intacto.
- `internal/runtime/specs/*` — Catálogo `Spec` intacto; `ResolvedAgent` produz um `Spec` via mapping.

### Relacionamentos e Fluxo de Dados

```
CLI: --agent foo --model bar
         │
         ▼
cmd/ai_spec_harness/task_loop.go
   • valida exclusividade flags
   • monta taskloop.Options{AgentName: "foo", Model: "bar"}
         │
         ▼
taskloop.Service.Run(opts)
   • opts.AgentName != "" →
      agent, _ := agents.NewDefaultRegistry(fs, workdir).Resolve("foo")
      catalog, _ := registry.Discover()
   • applyRuntimePrecedence(&opts.Runtime, agent.Runtime)
   • specs.<map(agent.Runtime.IDE)>() → Spec
         │
         ▼
runtime.ACPRunner.Run(spec, prompt enriquecido com blocos do agent)
   • prompt = BuildPromptContext(prdFolder, workDir, fsys, agent, catalog)
         │
         ▼
Persistência forense (inalterada) + ActivityWatchdog (inalterado)
```

## Design de Implementação

### Interfaces Chave

```go
// internal/agents/agent.go
type ResolvedAgent struct {
    Name     string
    Scope    Scope   // ScopeWorkspace | ScopeGlobal
    Path     string  // caminho absoluto do AGENT.md
    Metadata Metadata
    Runtime  RuntimeDefaults
    Prompt   string  // corpo do AGENT.md (após frontmatter)
}

type Metadata struct {
    Description string
    Version     string  // SemVer validado
}

type RuntimeDefaults struct {
    IDE             string  // claude | codex | gemini | copilot
    Model           string  // opcional; validado contra CompatibilityTable
    ReasoningEffort string  // opcional
    AccessMode      string  // opcional; ex: "bypass-permissions"
}

type Scope int
const (
    ScopeGlobal Scope = iota
    ScopeWorkspace
)
```

```go
// internal/agents/registry.go
type Registry interface {
    Discover(ctx context.Context) ([]ResolvedAgent, error)
    Resolve(name string) (ResolvedAgent, error)
}

// Construtor default — cache válido pelo tempo de vida da instância.
// Reset entre testes: instanciar nova Registry, não há ResetCache global.
func NewDefaultRegistry(fsys fs.FileSystem, workdir, home string) Registry
```

```go
// internal/agents/precedence.go
type RuntimeOverride struct {
    IDE             string
    Model           string
    ReasoningEffort string
    AccessMode      string
    // Flags explícitas: tracking se cada campo veio da CLI.
    ExplicitIDE             bool
    ExplicitModel           bool
    ExplicitReasoningEffort bool
    ExplicitAccessMode      bool
}

// applyRuntimePrecedence preenche cfg vazio com defaults do agente,
// preservando valores explicitamente setados via CLI.
// Espelha applyRuntimePrecedence do Compozy (internal/core/agents/execution.go).
func applyRuntimePrecedence(cfg *RuntimeOverride, defaults RuntimeDefaults)
```

```go
// internal/agents/prompt.go
// BuildAgentBlocks produz os dois blocos textuais a serem injetados pelo
// taskloop.BuildPromptContext na ordem fixa: metadata → catálogo.
// Catálogo é ordenado lexicograficamente por name (RF-16); até 200 entradas.
func BuildAgentBlocks(agent *ResolvedAgent, catalog []ResolvedAgent) (metadata, catalogBlock string)
```

```go
// internal/skills/frontmatter.go — REFATORAÇÃO
// Nova função genérica reutilizável por skills/ e agents/.
// Wrapper: ParseFrontmatter() chama esta função e mapeia para Frontmatter.
func ParseFrontmatterFields(content []byte) map[string]string

// Mantém comportamento atual (zero regressão para skills).
func ParseFrontmatter(content []byte) Frontmatter // wrapper
```

### Modelos de Dados

**Formato `AGENT.md`** (exemplo canônico):

```markdown
---
name: claude-revisor-rigoroso
description: Revisor de PR com viés conservador e foco em invariantes
version: 1.0.0
runtime:
  ide: claude
  model: claude-opus-4-7
  reasoning_effort: high
  access_mode: bypass-permissions
---

Você é um revisor de PR rigoroso. Priorize:
- invariantes ACP/governança
- segurança e tratamento de erros
- paridade cross-CLI
```

**JSON Schema** (`internal/agents/agent-frontmatter.schema.json`, embarcado):

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["name", "description", "version"],
  "additionalProperties": false,
  "properties": {
    "name":        { "type": "string", "pattern": "^[a-z0-9][a-z0-9-]*$" },
    "description": { "type": "string", "minLength": 1 },
    "version":     { "type": "string", "pattern": "^v?\\d+\\.\\d+\\.\\d+(-[A-Za-z0-9.-]+)?$" },
    "runtime": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "ide":              { "type": "string", "enum": ["claude", "codex", "gemini", "copilot"] },
        "model":            { "type": "string" },
        "reasoning_effort": { "type": "string", "enum": ["low", "medium", "high"] },
        "access_mode":      { "type": "string", "enum": ["bypass-permissions", "review", "readonly"] }
      }
    }
  }
}
```

**Estrutura de descoberta** (regras determinísticas):
- Global: `${HOME}/.ai-harness/agents/<name>/AGENT.md`
- Workspace: `<workdir>/.ai-harness/agents/<name>/AGENT.md`
- Em colisão: workspace prevalece; global homônimo é registrado e logado em nível `info` (Q3).
- `name` no frontmatter deve coincidir com o diretório pai (RF-06).

### Endpoints de API

N/A — funcionalidade local de CLI/biblioteca.

## Pontos de Integração

**Internos**:
- `internal/skills/frontmatter.go` — função genérica extraída; consumida por `internal/skills/*` e `internal/agents/*`.
- `internal/runtime/specs/*` — `ResolvedAgent.Runtime.IDE` é mapeado em runtime para a função de catálogo apropriada (`Claude()`, futuramente `Codex()`/`Gemini()`/`Copilot()` em ACP nativo). No escopo desta fase, apenas `Claude()` está disponível em ACP; demais IDEs continuam roteados via CLI invokers existentes.
- `internal/taskloop/compatibility.go` — `CompatibilityTable` valida `runtime.model` quando `runtime.ide` está declarado.

**Externos**: nenhum.

## Abordagem de Testes

### Testes Unitários

| Caso de teste | Componente | Cenário | Resultado esperado |
|---|---|---|---|
| T-01 | discovery | Sem agentes em nenhum escopo | `Discover()` retorna `[]`, sem erro |
| T-02 | discovery | Apenas global | Lista apenas global, `Scope=ScopeGlobal` |
| T-03 | discovery | Apenas workspace | Lista apenas workspace, `Scope=ScopeWorkspace` |
| T-04 | discovery | Colisão global+workspace | Workspace prevalece; global homônimo retornado em `shadowed` (campo separado) |
| T-05 | schema | Frontmatter sem `description` | Erro citando campo obrigatório |
| T-06 | schema | `version` não-SemVer | Erro citando campo `version` |
| T-07 | schema | `runtime.ide` fora do enum | Erro citando opções válidas |
| T-08 | schema | `runtime.reasoning_effort` inválido | Erro citando enum |
| T-09 | discovery | `name` no frontmatter ≠ nome do diretório | Erro RF-06 |
| T-10 | precedence | CLI `--model` setado, `AGENT.md` tem model diferente | `RuntimeOverride.Model` preserva valor da CLI |
| T-11 | precedence | CLI sem `--model`, `AGENT.md` tem model | `RuntimeOverride.Model` = AGENT.md |
| T-12 | precedence | CLI sem flags, `AGENT.md` sem runtime | `RuntimeOverride` permanece vazio (caller usa harness defaults) |
| T-13 | compatibility | `runtime.model` incompatível com `runtime.ide` | Erro, exceto se `--allow-unknown-model` |
| T-14 | registry | Cache: 2 chamadas a `Resolve("foo")` | Disco lido 1 vez (verificar com FakeFileSystem com contador) |
| T-15 | prompt | `BuildAgentBlocks` com agent + catalog de 3 entradas | Bloco metadata + catálogo ordenado lex, `[active]` no agente certo |
| T-16 | prompt | Catálogo com 250 entradas | Truncado para 200 (RF-16) |
| T-17 | taskloop | `Options.AgentName == ""` | Fluxo legado intacto: nenhuma chamada a registry |
| T-18 | taskloop | `Options.AgentName == "foo"` válido | Resolve → produz Spec via mapping → executa ACPRunner |
| T-19 | taskloop | `Options.AgentName == "foo"` não encontrado | Erro acionável citando candidatos descobertos |
| T-20 | CLI | `--agent foo --tool claude` | Erro `ErrFlagsConflitantes`-like |
| T-21 | CLI | `--agent foo --executor-tool codex` | Erro de conflito com modo avançado (Q1) |
| T-22 | frontmatter refactor | Skills atuais ainda parseiam corretamente | Suíte de regressão completa de `internal/skills` permanece verde |

**Mocks**: apenas `fs.FileSystem` (via `internal/fs/fake.go`, ADR-002). Sem mock de filesystem real, sem mock de HOME via env.

**Estrutura de mocks de HOME e workdir**: Discovery aceita `home` e `workdir` como parâmetros explícitos no construtor — não chama `os.UserHomeDir()` internamente. Testes passam paths arbitrários do FakeFileSystem.

### Testes de Integração

> Avaliação contra critérios do template:
> - [ ] Fronteira IO crítica onde mocks não garantem correção? **Não** — descoberta é leitura de filesystem; FakeFileSystem cobre.
> - [ ] Incidente prévio com mocks divergindo? **Não nesta fase**.
> - [ ] Custo de containers proporcional ao risco? **Desproporcional**.
>
> **Decisão**: integration tests não são necessários nesta fase. Cobertura de FakeFileSystem é suficiente para validar todos os RFs. Reavaliar em F2 (MCP) quando houver subprocessos stdio.

### Testes E2E

Cenário "smoke" único validando integração CLI ↔ taskloop ↔ registry sem ACP real:
- Cria `AGENT.md` em diretório temporário do teste
- Invoca `task_loop` com `--agent <name>` e mock de ACPRunner (`runnerStub` já existente)
- Verifica que o prompt enriquecido contém os blocos metadata + catálogo
- Verifica que os artefatos forenses são produzidos exatamente como no fluxo legado (RF-19)

## Sequenciamento de Desenvolvimento

### Ordem de Build

1. **Extração do parser genérico** (`internal/skills/frontmatter.go` → `ParseFrontmatterFields`) com testes de regressão para skills antes de qualquer mudança em `internal/agents/`. Mitigação do risco R-01.
2. **Package `internal/agents/` — schema** (JSON Schema embarcado + validação + testes T-05/T-06/T-07/T-08).
3. **Package `internal/agents/` — domínio** (`ResolvedAgent`, `Metadata`, `RuntimeDefaults`, `Scope`).
4. **Discovery** com `fs.FileSystem` (testes T-01/T-02/T-03/T-04/T-09).
5. **Registry com cache por instância** (testes T-14/T-19).
6. **Precedence + Prompt builders** (testes T-10/T-11/T-12/T-15/T-16).
7. **Integração `taskloop.Options.AgentName`** + `BuildPromptContext` aceita `*ResolvedAgent` (testes T-17/T-18).
8. **CLI flag `--agent`** + validação de exclusividade (testes T-20/T-21).
9. **Validação de compatibilidade** `runtime.model` × `CompatibilityTable` (teste T-13).
10. **E2E smoke + auditoria de invariantes forenses** (RF-19, T-22 + revisão de diff em `internal/runtime/persistence/` e `internal/runtime/watchdog.go` deve ser zero).
11. **ADR-011** registrado em `.specs/adr/011-agent-registry-declarativo.md`; AGENTS.md atualizado com referência.

### Dependências Técnicas

- Nenhuma dependência externa nova; `github.com/santhosh-tekuri/jsonschema/v6` já é usado em `internal/skills/schema.go`.
- Sem mudança em `go.mod`.

## Monitoramento e Observabilidade

- **Telemetria opt-in** (`GOVERNANCE_TELEMETRY=1`, padrão de [ADR-006](../../docs/adr/006-telemetria-feedback-cycle.md)): emitir evento `runtime_init` enriquecido com `agent_name`, `agent_version`, `agent_scope` quando `--agent` for usado.
- **Log estruturado**:
  - Nível `info`: shadowing (agente global ocultado por workspace homônimo).
  - Nível `info`: cache hit/miss em testes (gated por debug flag, não em produção).
  - Nível `error`: falha de validação de frontmatter com path + campo + linha (quando o parser conseguir reportar linha).
- **Dashboards Grafana**: nenhum novo painel nesta fase. Eventos `runtime_init` já são consumidos pelos dashboards existentes; o enriquecimento é aditivo.

## Considerações Técnicas

### Decisões Chave

Cada decisão material abaixo está registrada em ADR-011 ([`.specs/adr/011-agent-registry-declarativo.md`](../adr/011-agent-registry-declarativo.md)):

- **D-01 — Declarativo sobre código**: `AGENT.md` é fonte primária de identidade do agente; `specs.Claude()` permanece como fonte secundária técnica do Launcher.
- **D-02 — Descoberta dupla com workspace prevalecente**: global + workspace, com workspace ganhando colisão. Espelha Compozy (`~/.compozy/agents` + `.compozy/agents`).
- **D-03 — Coexistência com Spec hardcoded**: `ResolvedAgent` **produz** um `Spec`, não substitui. Migrar `specs.Claude()` para `AGENT.md` é trabalho futuro, não desta fase.
- **D-04 — Cache por instância, não global**: `NewDefaultRegistry()` cria cache próprio. Testes instanciam nova registry em vez de chamar `ResetCache()`. Diverge de `probe/probe.go` que usa `sync.Map` global — escolha consciente para evitar acoplamento entre testes paralelos (resolve Q2).
- **D-05 — Precedência CLI > AGENT.md > defaults**: `RuntimeOverride` com flags explícitas (`Explicit*` bool) para distinguir "não setado" de "setado vazio". Espelha `applyRuntimePrecedence` do Compozy.
- **D-06 — `--agent` mutuamente exclusiva com `--tool` e modo avançado**: erro de validação no CLI, paridade com `ErrFlagsConflitantes`. Resolve Q1.
- **D-07 — Shadowed agents reportados via log info**: não há flag `--show-shadowed` nesta fase. Resolve Q3.

**Alternativas rejeitadas** (detalhe em ADR-011):
- A: Substituir `specs.Claude()` por `AGENT.md` embarcado em `embed.FS`. Rejeitado: alto risco, alta blast radius, exige migração de toda a cadeia de probe/runtime. Adiado para fase futura.
- B: Descoberta apenas em workspace (sem global). Rejeitado: perde valor de reuso cross-projeto, que é o principal ganho ergonômico.
- C: Cache global via `sync.Map` (padrão de `probe.cache`). Rejeitado: dificulta testes paralelos; cache por instância tem ergonomia melhor.
- D: Parser YAML completo (ex.: `gopkg.in/yaml.v3`) em vez de generalizar o parser de skills. Rejeitado: introduz dependência nova, divergência da estratégia atual do harness. Generalizar `ParseFrontmatterFields` é suficiente para os campos previstos.

### Riscos Conhecidos

| ID | Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|---|
| R-01 | Regressão no parser de frontmatter compartilhado com skills | Média | Alto | Extração do `ParseFrontmatterFields` é o primeiro passo (Ordem de Build #1), com toda a suíte `internal/skills` rodando antes de qualquer mudança em `internal/agents/`. |
| R-02 | Drift com Compozy: schemas divergirem ao longo do tempo | Média | Médio | ADR-011 documenta mapeamento explícito entre campos `AGENT.md` do harness e do Compozy. Releases futuras revisitam este ADR. |
| R-03 | Prompt bloat com catálogo grande de agentes | Baixa | Médio | RF-16 limita a 200 entradas. Monitorar via tamanho de prompt em telemetria. |
| R-04 | Confusão entre `--tool` (modo simples) e `--agent` | Alta | Baixo | Mensagem de erro explícita listando alternativas; documentação no `--help`. |
| R-05 | Quebra silenciosa de invariantes forenses ao tocar em `taskloop.Service.Run` | Baixa | **Crítico** | Revisão obrigatória do diff em `internal/runtime/persistence/` e `internal/runtime/watchdog.go` (deve ser **zero**). Teste T-22 valida que artefatos forenses são idênticos. |

### Conformidade com Padrões

Regras aplicáveis de `.claude/rules/` e `.agents/skills/agent-governance/references/`:

- **R-GOV-001** (`.claude/rules/governance.md`): este techspec respeita precedência — `agent-governance` é citado, suposições explicitadas. Conflito hard/guideline: `D-04` (cache por instância) é guideline divergente de `probe.cache` (também guideline); justificativa documentada.
- **R-DDD-001** (`agent-governance/references/ddd.md`): `ResolvedAgent`, `Metadata`, `RuntimeDefaults`, `Scope` são value objects imutáveis. Construtores validantes (`NewDefaultRegistry`, parsing+validação). Sem instanciação por literal externa ao package.
- **R-SEC-001** (`agent-governance/references/security.md`): parsing de `AGENT.md` é puramente textual — sem `exec`, sem expansão de variáveis de ambiente no conteúdo, sem template execution em runtime. Caminhos resolvidos via `filepath.Join` (sem traversal).
- **R-ERR-001** (`agent-governance/references/error-handling.md`): erros sentinela (`ErrAgentNotFound`, `ErrFrontmatterInvalid`, `ErrNameDirMismatch`) compostos com `fmt.Errorf("%w: ...")` para `errors.Is`.
- **R-TEST-001** (`agent-governance/references/testing.md`): tabela de testes (22 casos) cobre cenários positivos e negativos; FakeFileSystem cobre toda fronteira de IO.
- **ADR-002**: descoberta e leitura de `AGENT.md` operam sobre `fs.FileSystem` para preservar testabilidade.
- **ADR-006**: telemetria opt-in append-only; novos campos de `runtime_init` são aditivos.
- **ADR-010**: tagged union de eventos preservada — `runtime_init` continua sendo o único kind afetado e apenas no payload.

### Arquivos Relevantes e Dependentes

**Novos**:
- `internal/agents/agent.go`
- `internal/agents/discovery.go`
- `internal/agents/registry.go`
- `internal/agents/schema.go`
- `internal/agents/agent-frontmatter.schema.json`
- `internal/agents/precedence.go`
- `internal/agents/prompt.go`
- `internal/agents/agent_test.go`, `discovery_test.go`, `registry_test.go`, `schema_test.go`, `precedence_test.go`, `prompt_test.go`
- `.specs/adr/011-agent-registry-declarativo.md`

**Modificados**:
- `internal/skills/frontmatter.go` — extração de `ParseFrontmatterFields`
- `internal/skills/frontmatter_test.go` — testes de regressão preservados + novos para função genérica
- `internal/taskloop/taskloop.go` — campo `AgentName` em `Options`
- `internal/taskloop/profile.go` — função auxiliar para derivar `ProfileConfig` de `ResolvedAgent` + override de runtime
- `internal/taskloop/agent.go` — assinatura de `BuildPromptContext` aceita `*agents.ResolvedAgent`
- `internal/taskloop/taskloop_test.go`, `internal/taskloop/profile_test.go`, `internal/taskloop/agent_test.go` — casos T-17/T-18/T-19/T-20/T-21
- `cmd/ai_spec_harness/task_loop.go` — flag `--agent`
- `cmd/ai_spec_harness/task_loop_test.go` — validação de exclusividade de flags
- `AGENTS.md` — link para ADR-011 + nota sobre Agent Registry

**Inalterados (invariante de fase)**:
- `internal/runtime/persistence/*`
- `internal/runtime/watchdog.go`
- `internal/runtime/specs/claude.go` e adjacentes
- `internal/runtime/client/*`
- `internal/runtime/events/*`

---

## Resolução de Suposições e Questões em Aberto do PRD

| ID | Item | Resolução |
|---|---|---|
| A1 | Path global `~/.ai-harness/agents/` | **Confirmado**. Espelha `~/.compozy/agents/` com prefixo do projeto. |
| A2 | Precedência universal CLI > AGENT.md > defaults | **Confirmado**. Implementada via `RuntimeOverride` com tracking explícito (`Explicit*` bool). |
| A3 | Parser YAML de skills é generalizável | **Confirmado**. Função `ParseFrontmatterFields(content) map[string]string` extraída como passo 1; suíte de regressão de skills protege contra divergência. |
| A4 | Catálogo cabe em janela de contexto | **Mitigado** por RF-16 (limite 200 entradas, ordem lexicográfica). Reavaliar se telemetria detectar prompts >2k linhas. |
| A5 | `--allow-unknown-model` cobre modelos não listados | **Confirmado**. Mesma semântica aplicada em T-13. |
| Q1 | `--agent` + modo avançado | **Resolvido**: erro de validação (D-06), mesma classe `ErrFlagsConflitantes`. |
| Q2 | Política de invalidação de cache | **Resolvido**: cache por instância (D-04). Sem TTL. Testes instanciam nova registry. |
| Q3 | Exposição de shadowed agents | **Resolvido**: log nível `info` no Discover (D-07). Sem flag dedicada nesta fase. |
