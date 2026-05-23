# Documento de Requisitos do Produto (PRD) — Agent Registry Declarativo

<!-- spec-version: 1 -->

> **Insumo de pesquisa**: [docs/research/compozy-adaptation-analysis.md](../../docs/research/compozy-adaptation-analysis.md)
> **Fase do roadmap**: 1 de 5 (adaptação ao padrão Compozy)
> **Data**: 2026-05-21

## Visão Geral

O `ai-spec-harness` orquestra agentes de IA (Claude via ACP; Codex/Gemini/Copilot via CLI invokers), mas hoje exige código Go para registrar cada agente — `internal/runtime/specs/claude.go` é hardcoded e não há entidade `Agent` separada do `Spec` técnico. Isso bloqueia ergonomia, multi-agente declarativo, e qualquer composição downstream (MCP, memory hierárquico, hooks) que dependa de um ponto de acoplamento por agente.

Esta funcionalidade introduz **Agent Registry Declarativo**: agentes passam a ser definidos por arquivos `AGENT.md` com frontmatter YAML, descobertos em runtime em `~/.ai-harness/agents/` (global do usuário) e `.ai-harness/agents/` (workspace do projeto). Inspira-se diretamente no padrão `internal/core/agents/` do Compozy, preservando as vantagens forenses do harness (events.jsonl, tool_calls.md, execution_report.md, ActivityWatchdog) e o catálogo `Spec` existente como camada de runtime.

Beneficia desenvolvedores e operadores do harness que precisam declarar novos perfis de agente (ex.: "claude-revisor-sênior", "codex-refator-incremental") sem recompilar Go, e abre caminho para as Fases 2–5 do roadmap (MCP, Memory, Hooks, Multi-IDE via ACP) que dependem deste ponto de acoplamento.

## Objetivos

- **OB-01**: Permitir declarar um novo agente sem modificar código Go, apenas criando `<scope>/agents/<name>/AGENT.md` em escopo global ou workspace.
- **OB-02**: Garantir descoberta determinística com precedência **workspace > global** em caso de colisão de nomes.
- **OB-03**: Compor system prompt em runtime injetando metadata do agente selecionado e catálogo de agentes disponíveis (paridade com `internal/core/agents/execution.go:SystemPrompt` do Compozy).
- **OB-04**: Preservar 100% da compatibilidade com a flag `--tool <claude|codex|gemini|copilot>` existente e com a tabela `specs.Claude()` em `internal/runtime/specs/`.
- **OB-05**: Preservar invariantes de persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e watchdog de inatividade (`ActivityWatchdog` com `CancelCause`) sem regressão.

**Métricas mensuráveis**:
- 100% dos testes de regressão do harness atual continuam verdes após introdução do registry.
- Tempo de descoberta + resolução de um agente abaixo de 50ms p95 em workspace com até 20 agentes.
- Zero alterações em `internal/runtime/persistence/` e `internal/runtime/watchdog.go` no escopo desta fase.

## Histórias de Usuário

- **HU-01**: Como **desenvolvedor do harness**, quero declarar um agente "claude-revisor-rigoroso" em `~/.ai-harness/agents/claude-revisor-rigoroso/AGENT.md` para reutilizá-lo em qualquer projeto sem fork de código.
- **HU-02**: Como **mantenedor de um repositório**, quero declarar um agente específico do projeto em `.ai-harness/agents/<name>/AGENT.md` que sobrescreva o agente global homônimo, garantindo que o workspace tenha a palavra final.
- **HU-03**: Como **operador da CLI**, quero invocar `ai-spec-harness task-loop --agent claude-revisor-rigoroso ...` e ver o harness resolver o agente, montar prompt com metadata, e executar via ACP sem mudanças no fluxo de persistência atual.
- **HU-04**: Como **operador da CLI**, quero que `--model` e `--reasoning-effort` na linha de comando ainda sobreponham os defaults declarados em `AGENT.md` (precedência **flags > AGENT.md > harness defaults**).
- **HU-05**: Como **usuário que não declarou nenhum agente**, quero que o harness continue funcionando exatamente como hoje quando uso `--tool claude` sem `--agent` (retrocompatibilidade total).
- **HU-06**: Como **operador**, quero erro claro quando referencio `--agent <name>` inexistente, listando os agentes disponíveis encontrados em workspace e global.

## Funcionalidades Core

### F-01: Descoberta Recursiva de Agentes
Escanear `~/.ai-harness/agents/*/AGENT.md` (global) e `<workdir>/.ai-harness/agents/*/AGENT.md` (workspace). Em colisão de nomes, workspace prevalece. Resultado é um catálogo `[]ResolvedAgent` consumível em runtime.

### F-02: Schema e Parser de AGENT.md
Frontmatter YAML obrigatório com campos `name`, `description`, `version`, e bloco `runtime.{ide, model, reasoning_effort, access_mode}`. Validação via JSON Schema (mesma estratégia de `internal/skills/schema.go`). Corpo do `AGENT.md` (após frontmatter) é o **prompt do agente**, injetado na composição de system prompt.

### F-03: Resolução de Agente por Nome
Função `Registry.Resolve(name string) (ResolvedAgent, error)` retorna o agente, aplicando precedência workspace > global. Erros explícitos para nome não encontrado (listando candidatos) e frontmatter inválido (apontando linha/campo).

### F-04: Composição Dinâmica de System Prompt
Quando um `ResolvedAgent` está ativo, o harness compõe o prompt do executor injetando: (a) base template existente, (b) bloco de metadata do agente, (c) catálogo dos agentes disponíveis, (d) corpo do `AGENT.md`. Espelha `SystemPrompt` do Compozy.

### F-05: Precedência de Runtime Config
Resolução final de `{tool, model, reasoning_effort, access_mode}` segue: **CLI flags > AGENT.md runtime defaults > harness defaults**. Implementação alinhada a `applyRuntimePrecedence` do Compozy.

### F-06: Integração com Taskloop (Opt-In)
Adicionar campo `AgentName string` em `taskloop.Options`. Quando preenchido (via nova flag `--agent <name>`), `ProfileConfig` é derivado do `ResolvedAgent`; quando vazio, fluxo atual de `--tool/--model` permanece inalterado.

### F-07: Mensagens de Erro Acionáveis
Falhas de descoberta, parse ou resolução retornam erro com: nome buscado, escopos consultados, candidatos encontrados, e recomendação concreta (ex.: "verifique frontmatter em ~/.ai-harness/agents/foo/AGENT.md linha 7").

## Requisitos Funcionais

- **RF-01**: Descobrir `AGENT.md` em `~/.ai-harness/agents/<name>/AGENT.md` (global) ignorando subdiretórios sem `AGENT.md` válido.
- **RF-02**: Descobrir `AGENT.md` em `<workdir>/.ai-harness/agents/<name>/AGENT.md` (workspace) com mesma regra.
- **RF-03**: Em colisão de nome entre global e workspace, workspace prevalece; agente global homônimo é marcado como **shadowed** no catálogo (visível ao usuário via flag de diagnóstico).
- **RF-04**: Parsear frontmatter YAML usando o parser generalizado de `internal/skills/frontmatter.go` (extração para função reutilizável é exigida).
- **RF-05**: Validar frontmatter contra JSON Schema embarcado (padrão de `internal/skills/schema.go`). Campos obrigatórios: `name`, `description`, `version`. Campos opcionais: `runtime.ide`, `runtime.model`, `runtime.reasoning_effort`, `runtime.access_mode`.
- **RF-06**: `name` no frontmatter deve coincidir com o nome do diretório pai; divergência produz erro de validação.
- **RF-07**: `version` deve seguir SemVer; versões inválidas produzem erro de validação.
- **RF-08**: `runtime.ide` aceita valores `claude`, `codex`, `gemini`, `copilot` (mesma enumeração de `internal/skills/skills.go:Tool`). Outros valores produzem erro.
- **RF-09**: `runtime.model` é validado contra `internal/taskloop/compatibility.go:CompatibilityTable` para o `runtime.ide` declarado. Valores incompatíveis produzem erro, exceto quando `--allow-unknown-model` está ativo (paridade com flag existente).
- **RF-10**: Expor `Registry` com métodos `Discover(ctx) ([]ResolvedAgent, error)` e `Resolve(name string) (ResolvedAgent, error)`.
- **RF-11**: Adicionar flag CLI `--agent <name>` em `cmd/ai_spec_harness/task_loop.go`. Mutuamente exclusiva com `--tool` (configuração simples) — uso conjunto produz erro de validação de flags.
- **RF-12**: Adicionar campo `AgentName string` em `taskloop.Options` (`internal/taskloop/taskloop.go:20-42`). Quando setado, `ProfileConfig` é derivado do `ResolvedAgent`.
- **RF-13**: Quando `AgentName` está setado e flags de runtime (`--model`, `--reasoning-effort`) também são passadas, as flags da CLI prevalecem sobre os defaults do `AGENT.md`.
- **RF-14**: Quando `AgentName` não está setado, fluxo atual de `--tool/--model` permanece inalterado em comportamento e resultado (retrocompatibilidade verificável por suíte de regressão).
- **RF-15**: `BuildPromptContext` (`internal/taskloop/agent.go:82-96`) deve aceitar um `*ResolvedAgent` opcional. Quando presente, o prompt final inclui (na ordem): template base atual, bloco metadata do agente, bloco catálogo de agentes, corpo do `AGENT.md`.
- **RF-16**: Bloco de catálogo de agentes lista apenas o nome e a descrição (1 linha) de cada agente, incluindo o agente ativo marcado como `[active]`. Limite de 200 entradas para evitar prompt bloat.
- **RF-17**: Erros de descoberta/parse/resolução retornam mensagem contendo: caminho do arquivo, escopo (global/workspace), campo problemático e correção sugerida.
- **RF-18**: Resolução de agente é cacheada por processo (paridade com `probe/probe.go:cache`). Cache pode ser explicitamente invalidado em testes via construtor.
- **RF-19**: Persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e `ActivityWatchdog` permanecem inalterados em comportamento e código fonte. Nenhuma modificação em `internal/runtime/persistence/` ou `internal/runtime/watchdog.go` é permitida nesta fase.
- **RF-20**: O `Spec` técnico (`internal/runtime/specs/`) continua sendo a fonte de verdade do **runtime**. `ResolvedAgent` **produz** um `Spec` (mapping `runtime.ide → specs.<Tool>()`), não substitui.

## Experiência do Usuário

Esta funcionalidade é majoritariamente backend/CLI; UX se materializa em três pontos:

1. **CLI**:
   - Nova flag `--agent <name>`.
   - Erro de agente não encontrado lista candidatos descobertos:
     ```
     erro: agente "foo" não encontrado.
     candidatos disponíveis:
       - claude-revisor-rigoroso (workspace)
       - codex-refator-incremental (global)
     ```
   - Comando diagnóstico opcional (fora do escopo desta fase): `ai-spec-harness agents list`.

2. **Arquivo `AGENT.md`** (formato esperado):
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

3. **Composição de System Prompt** (ordem fixa): template base → metadata → catálogo → corpo do `AGENT.md`. Ordem é determinística para que o agente possa raciocinar sobre suas próprias capacidades e sobre as alternativas disponíveis.

## Restrições Técnicas de Alto Nível

- **Linguagem e protocolo**: implementação em Go, mantendo ACP via `coder/acp-go-sdk` como protocolo de comunicação com Claude.
- **Reuso obrigatório**: parser YAML existente em `internal/skills/frontmatter.go:23-67` deve ser generalizado para função reutilizável (não duplicar); padrão de JSON Schema validation em `internal/skills/schema.go:47-70` deve ser replicado para schema de `Agent` com schema embarcado via `go:embed`.
- **Filesystem abstraction**: descoberta deve operar sobre `fs.FileSystem` para preservar testabilidade com FakeFileSystem (ADR-002).
- **Invariantes preservadas**:
  - Persistência forense (`events.jsonl`, `tool_calls.md`, `execution_report.md`) e watchdog de inatividade não podem ser modificados nesta fase.
  - Catálogo `Spec` existente (`internal/runtime/specs/`) permanece autoritativo para resolução de Launcher; `ResolvedAgent` produz um `Spec`, não substitui.
  - Tabela de compatibilidade tool↔model (`internal/taskloop/compatibility.go`) é fonte de verdade para validação de `runtime.model`.
- **Segurança**: leitura de `~/.ai-harness/agents/` segue R-SEC-001 — sem execução de comandos shell, sem expansão de variáveis no conteúdo do `AGENT.md` durante parse.
- **Limites operacionais**: descoberta deve cobrir até 100 agentes globais + 100 workspace sem degradação observável (p95 < 50ms).
- **Telemetria**: emitir evento `runtime_init` enriquecido com `agent_name`, `agent_version`, `agent_scope (global|workspace)` quando `--agent` é usado (compatível com `GOVERNANCE_TELEMETRY=1`).
- **Determinismo de ordem do catálogo**: lista de agentes no prompt segue ordem lexicográfica de `name` para evitar prompt churn entre runs.

## Fora de Escopo

Os itens abaixo **não** fazem parte desta Fase 1. São PRDs futuros, documentados em `docs/research/compozy-adaptation-analysis.md`:

- **MCP Integration** (`mcp.json` por agente, subprocess stdio): Fase 2.
- **Memory Layer hierárquico** (`MEMORY.md` workflow + task-local com compactação atômica): Fase 3.
- **Tool Input Normalization cross-CLI**: Fase 3.
- **Hook System** (`prompt.pre_build`, `prompt.post_build`, `prompt.pre_system`): Fase 4.
- **Runtime Retry/Backoff configurável** no AGENT.md: Fase 4.
- **Streaming backpressure metrics** (slow publishes, dropped updates): Fase 4.
- **Multi-IDE real via ACP** (substituir CLI invokers de Codex/Gemini/Copilot por SDKs ACP nativos): Fase 5, condicional à maturidade dos SDKs upstream.
- **Comando CLI `agents list`** para diagnóstico: pode ser entregue como follow-up tático, fora deste PRD.
- **Migração do `specs.Claude()` hardcoded para `AGENT.md` embutido**: decisão técnica adiada para o TechSpec; este PRD exige apenas coexistência, não migração.
- **Mudanças em CI ou scripts `make`**: fora de escopo.

## Suposições e Questões em Aberto

**Suposições assumidas** (devem ser validadas no TechSpec):
- A1: `~/.ai-harness/agents/` é o caminho global apropriado (Compozy usa `~/.compozy/`; manter prefixo `ai-harness` por convenção do projeto).
- A2: A precedência **flags > AGENT.md > harness defaults** é universal para todos os campos de runtime; não há campo que justifique inversão.
- A3: O parser YAML de `internal/skills/frontmatter.go` é generalizável sem regressão em consumidores atuais (skills).
- A4: O catálogo de agentes injetado no prompt cabe na janela de contexto mesmo com 100+ agentes (mitigado por limite de 200 entradas em RF-16; pode ser refinado se houver problema).
- A5: A flag `--allow-unknown-model` existente (`internal/taskloop/profile.go`) cobre o caso de modelos novos não listados na tabela de compatibilidade — mesma semântica é aplicada na validação de `runtime.model` do AGENT.md.

**Questões em aberto** (não bloqueantes para PRD; serão resolvidas no TechSpec):
- Q1: Conflito entre `--agent <name>` e flags do modo avançado (`--executor-tool`, `--reviewer-tool`) — comportamento exato precisa ser definido (provavelmente erro de validação de flags, como ocorre hoje entre `--tool` e `--executor-tool`).
- Q2: Política exata de invalidação de cache do registry — TTL? Apenas via construtor? (proposta inicial: apenas via construtor, sem TTL).
- Q3: Como expor agentes "shadowed" ao usuário — flag `--show-shadowed`? Log de warning? (proposta inicial: log de info quando shadowing ocorre).
