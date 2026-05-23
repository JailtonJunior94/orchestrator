# ADR-011 — Agent Registry Declarativo

## Metadados

- **Título**: Agent Registry Declarativo via `AGENT.md` com descoberta dupla e coexistência com `Spec` hardcoded
- **Data**: 2026-05-21
- **Status**: Proposta
- **Decisores**: time `ai-spec-harness` (mantenedor: jailton.junior94@outlook.com)
- **Relacionados**:
  - PRD: [`.specs/prd-agent-registry-declarativo/prd.md`](../prd-agent-registry-declarativo/prd.md)
  - TechSpec: [`.specs/prd-agent-registry-declarativo/techspec.md`](../prd-agent-registry-declarativo/techspec.md)
  - Pesquisa: [`docs/research/compozy-adaptation-analysis.md`](../../docs/research/compozy-adaptation-analysis.md)
  - ADR anterior: [`.specs/adr/009-acp-protocol-adoption.md`](009-acp-protocol-adoption.md)
  - ADR anterior: [`.specs/prd-acp-runtime-claude/adr-010-event-tagged-union.md`](../prd-acp-runtime-claude/adr-010-event-tagged-union.md)
  - ADR de base: [`docs/adr/002-fake-filesystem-testes.md`](../../docs/adr/002-fake-filesystem-testes.md)
  - ADR de base: [`docs/adr/006-telemetria-feedback-cycle.md`](../../docs/adr/006-telemetria-feedback-cycle.md)

## Contexto

Após finalizar a migração ACP para Claude (ADR-009/010), a comparação detalhada com `compozy/compozy` (relatório em `docs/research/compozy-adaptation-analysis.md`) evidenciou que **a diferença mais determinante entre os dois projetos é declarativa vs. imperativa**: Compozy modela cada agente como artefato (`AGENT.md` + `mcp.json`) descoberto em runtime; o `ai-spec-harness` modela como código compilado (`specs.Claude()` hardcoded em `internal/runtime/specs/claude.go`).

Sem uma entidade `Agent` declarativa, todas as próximas frentes de evolução (MCP, memory hierárquico, hooks, multi-IDE via ACP) precisariam inventar seu próprio mecanismo de configuração. Isso bloqueia ergonomia (registrar um novo perfil exige fork de código), reuso cross-projeto e composição multi-agente.

A decisão também precisa preservar dois pontos fortes do harness ausentes no Compozy: persistência forense por sessão (`events.jsonl` + `tool_calls.md` + `execution_report.md`) e watchdog de inatividade com `CancelCause`. Esses são diferenciais a manter intactos.

Restrições adicionais:
- Stack Go + ACP via `coder/acp-go-sdk` é não negociável.
- Catálogo `specs.Claude()` é fonte de verdade do Launcher e da resolução binário/npx (RF-03 de ADR-009).
- Tabela de compatibilidade tool↔model (`internal/taskloop/compatibility.go`) é fonte de verdade para validação de modelos.
- Parser YAML de skills (`internal/skills/frontmatter.go`) é compartilhado com toda a infraestrutura de skills — qualquer mudança exige zero regressão.

## Decisão

Introduzir **Agent Registry Declarativo** com sete sub-decisões materiais:

### D-01 — Declarativo sobre código (mas coexistindo)
`AGENT.md` é a fonte primária de **identidade** de um agente (nome, descrição, versão, defaults de runtime, prompt). `specs.Claude()` permanece como fonte de verdade **técnica** do Launcher e do binário/npx. Um `ResolvedAgent` **produz** um `Spec` via mapping `runtime.ide → specs.<Tool>()` — não o substitui.

**Escopo**: Fase 1 do roadmap de adaptação ao Compozy. Migração do `specs.Claude()` para `AGENT.md` embarcado é trabalho futuro, fora desta ADR.

### D-02 — Descoberta dupla (global + workspace) com workspace prevalecente
- Global: `${HOME}/.ai-harness/agents/<name>/AGENT.md`
- Workspace: `<workdir>/.ai-harness/agents/<name>/AGENT.md`
- Colisão: workspace ganha; global homônimo é registrado em campo `shadowed` separado e logado em nível `info`.

Espelha o padrão Compozy (`~/.compozy/agents` + `.compozy/agents`), adaptando o prefixo ao projeto.

### D-03 — Coexistência com `Spec` hardcoded
`specs.Claude()` continua existindo e `--tool claude` continua funcionando. `--agent <name>` é opt-in, mutuamente exclusivo com `--tool` e com flags do modo avançado. Quando ausente, todo o fluxo legado permanece inalterado.

### D-04 — Cache por instância, não global
`NewDefaultRegistry(fsys, workdir, home)` cria seu próprio cache. Testes instanciam nova registry em vez de chamar `ResetCache()` global. Diverge intencionalmente de `internal/runtime/probe/probe.go` (que usa `sync.Map` global).

**Justificativa**: registry é stateful e acessada de múltiplos componentes (`taskloop`, possivelmente comando CLI futuro `agents list`); cache global gera acoplamento entre testes paralelos e exige reset manual. Cache por instância tem ergonomia superior e está alinhado ao padrão Compozy.

### D-05 — Precedência CLI > AGENT.md > harness defaults
Resolução final de `{ide, model, reasoning_effort, access_mode}` segue: **CLI flags > AGENT.md runtime defaults > harness defaults**. Implementada via `RuntimeOverride` com flags explícitas (`ExplicitIDE`, `ExplicitModel`, etc.) para distinguir "não setado" de "setado vazio".

Espelha `applyRuntimePrecedence` do Compozy (`internal/core/agents/execution.go`).

### D-06 — `--agent` mutuamente exclusiva com `--tool` e modo avançado
Resolve a questão em aberto Q1 do PRD: erro de validação de flags (paridade semântica com `ErrFlagsConflitantes` existente em `internal/taskloop/profile.go`).

### D-07 — Shadowed agents reportados via log info
Resolve Q3 do PRD: sem flag dedicada `--show-shadowed` nesta fase. Log nível `info` no `Discover` quando ocorre colisão. Decisão pode ser revisitada se telemetria detectar uso frequente.

## Alternativas Consideradas

### A — Substituir `specs.Claude()` por `AGENT.md` embarcado em `embed.FS`
**Vantagens**: paridade total com Compozy desde o dia 1; zero código de catálogo em Go.
**Desvantagens**: alto blast radius (toda a cadeia probe/runtime precisa adaptar), risco de regressão em RF-03 (resolução binary/npx), bloqueia esta fase em refatoração que não agrega valor incremental ao usuário.
**Motivo de rejeição**: prefere-se incremento aditivo. Migração pode ser uma ADR futura quando o registry estiver maduro.

### B — Descoberta apenas em workspace (sem global)
**Vantagens**: implementação mais simples (uma origem só).
**Desvantagens**: perde o principal ganho ergonômico (reuso de agentes entre projetos), diverge do padrão Compozy.
**Motivo de rejeição**: simplicidade não compensa perda funcional principal.

### C — Cache global via `sync.Map` (padrão de `probe.cache`)
**Vantagens**: consistência com padrão existente no harness; menos parâmetros ao construtor.
**Desvantagens**: dificulta testes paralelos; exige `ResetCache` manual entre testes; gera estado global oculto.
**Motivo de rejeição**: D-04 prefere ergonomia de teste e isolamento.

### D — Parser YAML completo via `gopkg.in/yaml.v3`
**Vantagens**: parser robusto, suporta nested arbitrário, validação semântica forte.
**Desvantagens**: introduz dependência nova, diverge da estratégia atual do harness (parser textual minimalista em `internal/skills/frontmatter.go`), aumenta superfície de bugs.
**Motivo de rejeição**: campos previstos no schema cabem perfeitamente na estratégia atual generalizada; introduzir yaml.v3 é over-engineering nesta fase.

### E — Permitir `--agent` em paralelo com `--tool` (override CLI total)
**Vantagens**: máxima flexibilidade.
**Desvantagens**: matriz de combinações explode, mensagens de erro perdem clareza, conflito implícito entre "agent diz tool=claude" e "CLI diz tool=codex" exige resolução opaca.
**Motivo de rejeição**: D-06 prefere falha explícita e composição via runtime override de campos específicos (`--model`, `--reasoning-effort`).

## Consequências

### Benefícios Esperados

- **Ergonomia**: declarar um novo agente exige apenas criar um arquivo Markdown — sem recompilação Go.
- **Reuso cross-projeto**: agentes globais em `~/.ai-harness/agents/` são reaproveitáveis entre qualquer workspace.
- **Override por projeto**: workspace prevalece, garantindo que cada repositório tenha controle final sobre seus agentes.
- **Composição dinâmica de prompt**: catálogo de agentes injetado no prompt permite hand-off entre agentes em fases futuras (paridade com Compozy).
- **Destrava roadmap**: MCP (F2), Memory (F3), Hooks (F4) e Multi-IDE via ACP (F5) ganham ponto de acoplamento natural via `AGENT.md`/`mcp.json` adjacente.
- **Preserva pontos fortes**: persistência forense e watchdog permanecem intactos (RF-19), reforçando o diferencial do harness.

### Trade-offs e Custos

- **Duas fontes de verdade temporárias**: `AGENT.md` (identidade) + `specs.Claude()` (técnica). Aumenta cognitive load até migração futura.
- **Parser de frontmatter compartilhado**: extração de `ParseFrontmatterFields` exige cuidado de regressão (mitigação: suíte de skills antes da extração).
- **Espaço de configuração maior**: usuários têm 3 maneiras de configurar runtime (CLI, AGENT.md, defaults). Documentação no `--help` e exemplo canônico no README mitigam.
- **Drift potencial com Compozy**: schemas podem divergir ao longo do tempo. Mitigado por mapeamento explícito documentado nesta ADR e revisitação periódica.

### Riscos e Mitigações

| Risco | Impacto | Estratégia |
|---|---|---|
| Regressão no parser de frontmatter compartilhado | Alto (skills quebram) | Extrair `ParseFrontmatterFields` antes de criar `internal/agents/`; suíte de regressão completa de skills deve passar entre as duas mudanças. |
| Drift com schema Compozy ao longo do tempo | Médio | Mapeamento explícito nesta ADR; revisão a cada release minor; tabela comparativa em `docs/research/compozy-adaptation-analysis.md`. |
| Prompt bloat com catálogo grande | Médio | RF-16 limita a 200 entradas em ordem lex; telemetria de tamanho de prompt. |
| Quebra silenciosa de invariantes forenses | **Crítico** | Diff em `internal/runtime/persistence/` e `internal/runtime/watchdog.go` **deve ser zero** nesta fase; teste E2E T-22 valida artefatos forenses. |

**Plano de rollback**: a flag `--agent` é opt-in e o package `internal/agents/` é additivo. Reverter consiste em remover a flag e o package (commit único); fluxo legado permanece funcionando porque nada foi removido. Refator do parser de frontmatter exige cuidado adicional — se necessário reverter, restaurar a versão original de `internal/skills/frontmatter.go` do commit anterior.

## Plano de Implementação

1. Extrair `ParseFrontmatterFields(content []byte) map[string]string` em `internal/skills/frontmatter.go`, mantendo `ParseFrontmatter` como wrapper. Validar com suíte completa de `internal/skills` (zero regressão).
2. Criar `internal/agents/` com value objects (`ResolvedAgent`, `Metadata`, `RuntimeDefaults`, `Scope`) + JSON Schema embarcado.
3. Implementar `discovery.go` operando sobre `fs.FileSystem` (ADR-002).
4. Implementar `registry.go` com cache por instância (D-04).
5. Implementar `precedence.go` (D-05) e `prompt.go` (RF-16).
6. Adicionar campo `AgentName` em `taskloop.Options`; alterar `BuildPromptContext` para aceitar `*ResolvedAgent` opcional; alterar `taskloop.Service.Run` para resolver agente quando `AgentName != ""`.
7. Adicionar flag CLI `--agent` em `cmd/ai_spec_harness/task_loop.go` com validação de exclusividade (D-06).
8. Validar `runtime.model` × `CompatibilityTable` quando ambos presentes.
9. E2E smoke: workspace temporário com `AGENT.md` → invoca task-loop → valida prompt enriquecido + artefatos forenses idênticos.
10. Atualizar `AGENTS.md` com referência a esta ADR.

**Critérios de adoção concluída**:
- Todos os 22 casos de teste (T-01 a T-22) verdes.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.
- Documentação no `--help` da flag `--agent`.
- Exemplo canônico de `AGENT.md` em `docs/` (path a definir na implementação).

## Monitoramento e Validação

- **Métricas**: tamanho do prompt enriquecido (linhas, bytes); cache hit/miss da registry (via teste, não em produção).
- **Logs**: shadowing (info), falha de validação de frontmatter (error), agente não encontrado (error).
- **Telemetria opt-in** (`GOVERNANCE_TELEMETRY=1`): campos `agent_name`, `agent_version`, `agent_scope` no evento `runtime_init` quando `--agent` for usado.
- **Critérios de sucesso**:
  - Zero regressão nos testes existentes.
  - Tempo de descoberta + resolução < 50ms p95 em workspace com 20 agentes.
  - Adoção interna: ≥1 agente declarado no formato `AGENT.md` em uso real até o fim da Fase 1.
- **Critérios para revisão**:
  - Schema diverge significativamente do Compozy em mais de 2 campos.
  - Prompt bloat reportado por usuários (>2k linhas em sessões reais).
  - Surge necessidade de cache global (improvável; revisar D-04 se ocorrer).

## Impacto em Documentação e Operação

- `AGENTS.md`: adicionar seção "Agent Registry" com link a esta ADR e ao PRD.
- `docs/research/compozy-adaptation-analysis.md`: já existente; será atualizado na Fase 5 do roadmap com seção "Próximos PRDs".
- `cmd/ai_spec_harness/task_loop.go --help`: documentar flag `--agent` e exclusividade.
- Exemplo canônico de `AGENT.md` em `docs/agents/example/AGENT.md` (path proposto; final na implementação).
- Runbook não exige mudança — invariantes forenses preservados.

## Revisão Futura

Esta ADR deve ser revisitada nos seguintes marcos:

1. **Início da Fase 2 (MCP Integration)**: validar se decisão D-04 (cache por instância) continua adequada quando registry passa a gerenciar subprocessos MCP.
2. **Quando Compozy modificar significativamente seu schema `AGENT.md`**: atualizar mapeamento de campos ou aceitar drift consciente.
3. **Após 6 meses de adoção interna**: revisar D-03 (coexistência com `specs.Claude()`) — pode ser hora de migrar definitivamente para `AGENT.md` embarcado.

**Eventos que invalidam premissas**:
- Compozy deprecia o padrão `AGENT.md` em favor de outro formato.
- ACP SDK passa a oferecer mecanismo nativo de descoberta de agentes.
- Surge requisito de agentes serem definidos via banco de dados ou serviço remoto (fora de `~/.ai-harness/agents/`).
