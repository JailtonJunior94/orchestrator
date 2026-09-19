# Inventário de Regras — Universal vs. Específico de Fornecedor

- **PRD:** `.specs/prd-harness-portatil-vendor-neutral/prd.md`
- **Requisitos cobertos:** RF-01, RF-14, RF-54
- **Tarefa:** 2.0 — Inventário de regras universais e específicas de fornecedor
- **Escopo varrido:** `AGENTS.md`, `CLAUDE.md`, `CODEX.md`, `COPILOT.md`, `.claude/`, `.codex/`,
  `.opencode/`, `.github/`
- **Método:** leitura integral dos arquivos-raiz de governança; `diff`/`diff -rq` entre pares
  candidatos a mirror; leitura dos scripts de sincronização (`scripts/sync-skills.sh`,
  `scripts/check-scripts-sync.sh`) para confirmar canonicidade; comparação contra
  `internal/embedded/assets/` para confirmar gaps de distribuição (V-01).

## Legenda de Classificação

| Rótulo | Significado |
|--------|-------------|
| `universal` | Regra que governa comportamento independente de ferramenta; deveria ter origem canônica única |
| `claude-specific` | Regra ou mecanismo exclusivo do Claude Code |
| `codex-specific` | Regra ou mecanismo exclusivo do Codex CLI / `codex-acp` |
| `copilot-specific` | Regra ou mecanismo exclusivo do GitHub Copilot CLI/Chat |
| `opencode-specific` | Regra ou mecanismo exclusivo do OpenCode |

Cada linha classifica uma **regra** (unidade semântica de governança), não um arquivo isolado —
uma mesma regra pode ter cópias ou referências em múltiplos arquivos; todas estão listadas em
"Onde aparece".

## Inventário

| # | Regra | Classificação | Onde aparece | Estado de canonicalização | Evidência |
|---|-------|---------------|---------------|---------------------------|-----------|
| 1 | R-GOV-001 — Governança de regras (precedência, conflito, evidência) | `universal` | `.claude/rules/governance.md`; embarcada em `internal/embedded/assets/.claude/rules/governance.md`; referenciada por prosa em `CODEX.md`, `COPILOT.md` | Canônica hoje em caminho vendor-specific (`.claude/rules/`), violando RNF01 (V-02). Distribuída para consumidores via asset embarcado, mas a origem editável não é vendor-neutral. | `.claude/rules/governance.md`; `internal/embedded/assets/.claude/rules/governance.md`; `CODEX.md:119`; `COPILOT.md:97` |
| 2 | R-STYLE-001 — Estilo de código (idioma inglês + zero comentários + sem prefixo `_`) | `universal` | `.claude/rules/code-style.md`; referenciada por link em `CODEX.md`, `COPILOT.md`; **duplicada em prosa própria** (não referenciada) em `.github/copilot-instructions.md:5` | Canônica em caminho vendor-specific (V-02) e **ausente dos assets embarcados** — consumidor que instala o harness não recebe a regra (V-01). Cópia solta em `.github/copilot-instructions.md` diverge textualmente da fonte (prosa reduzida a uma linha). | `.claude/rules/code-style.md`; grep vazio em `internal/embedded/assets/.claude/rules/` para `code-style.md`; `.github/copilot-instructions.md:5` |
| 3 | Contrato de carga base (ler `AGENTS.md` + `agent-governance/SKILL.md` antes de editar) | `universal` | Canônico em `AGENTS.md` (seção "Contrato de carga base"); referenciado (não duplicado) por `CLAUDE.md` via `@AGENTS.md`, por `CODEX.md:117-123`, por `COPILOT.md:95-101`, por `.codex/agents/task-executor.toml`, por `.codex/docs/workaround-preload.md:31` | Corretamente canonicalizado: um único texto-fonte, demais arquivos apenas apontam para ele. Nenhuma ação necessária em RF-13. | `AGENTS.md` (seção "Contrato de carga base"); `CLAUDE.md:1` |
| 4 | Stack / Comandos / Convenções / Estrutura do repositório | `universal` | Canônico em `AGENTS.md` (seções "Stack", "Comandos", "Convenções", "Estrutura"); **duplicado independentemente** (prosa própria, não link) em `.github/copilot-instructions.md` inteiro | Duplicação manual confirmada e **já divergente**: `.github/copilot-instructions.md:7` diz "Homebrew Cask", `AGENTS.md` (seção Stack) diz "Homebrew Formula" — exemplo vivo do risco que RF-13 endereça. Fora do escopo de ação de P3/P4 desta técspec (que fecham apenas #1 e #2 nesta entrega); registrado aqui para eventual RF-13 futura. | `AGENTS.md` (seção "Stack"); `.github/copilot-instructions.md:1-28` |
| 5 | Skills processuais e de linguagem (`create-prd`, `execute-task`, `go-implementation`, etc.) | `universal` | Canônico em `.agents/skills/`; mirrors gerados em `.claude/skills/`, `.github/skills/`, `internal/embedded/assets/.agents/skills/` via `scripts/sync-skills.sh` + gate `check-skills-sync`; referenciado por path direto (sem cópia) em `.codex/config.toml` (`[[skills.config]] path = ".agents/skills/..."`) | Corretamente canonicalizado. Mirrors são gerados e comparados por `make check-skills-sync`; edição direta do mirror quebra o gate. Nenhuma ação necessária em RF-13. | `scripts/sync-skills.sh:2,10`; `.codex/config.toml:1-91`; `Makefile` alvo `check-skills-sync` |
| 6 | Hooks de ciclo de vida (`validate-preload.sh`, `validate-governance.sh`, `validate-session-end.sh`, `post-execute-task.sh`, `post-wave.sh`, `pre-execute-all-tasks.sh`, `subagent-stop-wrapper.sh`) | `universal` | Canônico em `.agents/hooks/`; mirrors byte-idênticos confirmados por `diff` em `.claude/hooks/`, `.codex/hooks/`, `.github/hooks/`; referenciado por path direto (sem cópia) em `.opencode/plugin/governance.js` (`CANONICAL_PRE_TOOL_SCRIPT`, `CANONICAL_POST_TOOL_SCRIPT`) | Corretamente canonicalizado, protegido por `check-hooks-sync`. OpenCode não mantém cópia alguma — referencia o canônico diretamente, forma mais forte de canonicalização entre os quatro. Nenhuma ação necessária em RF-13. | `diff .claude/hooks/validate-governance.sh .codex/hooks/validate-governance.sh` → idêntico; `diff .claude/hooks/validate-governance.sh .github/hooks/validate-governance.sh` → idêntico; `.opencode/plugin/governance.js:8-10` |
| 7 | Validadores de evidência (`validate-task-evidence.sh`, `validate-bugfix-evidence.sh`, `validate-refactor-evidence.sh`, `validate-review-evidence.sh`, `resolve-references.sh`, `hook-prereq-gate.sh`, `validate-skill-prerequisites.sh`, `validate-governance-references.sh`) | `universal` | Canônico em `.agents/scripts/`; mirror em `.claude/scripts/` e `internal/embedded/assets/.claude/scripts/`, protegido por `scripts/check-scripts-sync.sh`; resolução em cascata pelas skills (`.agents/scripts/` → `.claude/scripts/` → `scripts/`) | Corretamente canonicalizado. Codex/Copilot/OpenCode não mantêm cópia própria — resolvem o script via cascata de path a partir do canônico. Nenhuma ação necessária em RF-13. | `scripts/check-scripts-sync.sh:3-10,20-30` |
| 8 | Definições de subagente `task-executor`, `reviewer`, `bugfixer`, `refactorer`, `prd-writer`, `technical-specification-writer`, `task-planner`, `project-analyzer` | `universal` (conteúdo/papel) com **formato adaptado por fornecedor** | `.claude/agents/*.md` (frontmatter `skills:` + corpo); `.github/agents/*.agent.md` (formato "coding agent" do Copilot); `.codex/agents/task-executor.toml` (formato TOML do Codex) | Conteúdo semântico idêntico entre `.claude/agents/task-executor.md` e `.github/agents/task-executor.agent.md` (`diff` mostra apenas adaptação de terminologia — "subagente" vs "agente" — e metadado de formato), mas **sem gate de sincronia** (não coberto por `check-skills-sync`/`check-hooks-sync`/`check-scripts-sync`, V-18 lista só três pares). Mantido manualmente hoje; fora do escopo de ação de RF-11/RF-13 nesta entrega (a técspec fecha só #1 e #2 em P3/P4); registrado para eventual expansão futura de par de espelhamento. | `diff .claude/agents/task-executor.md .github/agents/task-executor.agent.md`; `.codex/agents/task-executor.toml:4-18` |
| 9 | Runtime ACP do Claude Code: hooks Go (`internal/runtime/hooks/`), flags `--mcp-nested`, `--auto-review`, `--no-normalize`, `--disable-hooks`, `--durable-memory`, `--memory-workflow-limit-lines` | `claude-specific` | `CLAUDE.md` (seção "Runtime orquestrado"), `docs/runtime-claude-capabilities.md` | Corretamente isolado — não há equivalente nos outros três provedores neste PRD. | `CLAUDE.md` (seção "Runtime orquestrado") |
| 10 | Hooks shell locais do Claude Code (`.claude/hooks/*.sh` registrados via `ai-spec install .` em `.claude/settings.local.json`) e subagentes carregados via `.claude/agents/` | `claude-specific` | `CLAUDE.md` (seção "Skills, subagentes e hooks") | Mecanismo de carregamento (não o conteúdo dos hooks, já classificado no item 6) é exclusivo do formato Claude Code. | `CLAUDE.md` (seção "Skills, subagentes e hooks") |
| 11 | Modo ACP do Codex (`codex-acp`), flags `--reasoning-effort`, `--access-mode`, nomenclatura `codex` vs `codex-acp`, modo legado `codex exec --yolo` | `codex-specific` | `CODEX.md` (seções "Modo Recomendado", "Flags Codex-especificas", "Modo Legado") | Sem equivalente direto nos outros provedores; aviso de nomenclatura é particular ao ecossistema Codex/Zed. | `CODEX.md:5-109` |
| 12 | `sandbox_mode = "workspace-write"`, `approval_policy = "on-request"`, registro de skills habilitadas e wiring de hooks PreToolUse/PostToolUse/Stop | `codex-specific` | `.codex/config.toml` | Mecanismo de configuração nativo do Codex CLI; não tem equivalente estrutural nos outros três. | `.codex/config.toml:94-114` |
| 13 | Workaround de gap de PreToolUse (route-around) do Codex e alternativas de mitigação | `codex-specific` | `.codex/docs/workaround-preload.md` | Documenta uma lacuna específica do hook nativo do Codex; não aplicável aos demais provedores. | `.codex/docs/workaround-preload.md` |
| 14 | Modo ACP do Copilot CLI (`copilot --acp`), flags e fallback `npx @github/copilot`, modo legado `copilot --autopilot --yolo` | `copilot-specific` | `COPILOT.md` (seções "Modo Recomendado", "Modo Legado") | Sem equivalente direto nos outros provedores. | `COPILOT.md:5-88` |
| 15 | Carregamento automático de `.github/copilot-instructions.md` pelo Copilot Chat (VS Code / GitHub.com) | `copilot-specific` (mecanismo de carregamento) | `COPILOT.md` (seção "Copilot Chat — Suporte Nativo"); `.github/copilot-instructions.md` | O **mecanismo** de auto-carregamento é exclusivo do Copilot Chat. O **conteúdo** do arquivo, porém, duplica regra universal (ver item 4) em vez de apenas apontar para `AGENTS.md` — e a mesma lacuna do item 2/4, não um novo mecanismo. | `COPILOT.md:123-130` |
| 16 | Definições "coding agent" do GitHub (`.github/agents/*.agent.md`) como formato de distribuição | `copilot-specific` (formato) | `.github/agents/*.agent.md` | Formato de arquivo é exclusivo do GitHub Copilot coding agent; conteúdo é o item 8. | `.github/agents/task-executor.agent.md` |
| 17 | Plugin de governança OpenCode (`tool.execute.before/after`, classificação `MUTATING_TOOLS`/`READ_ONLY_TOOLS`, sentinelas de ambiente, `event: session.idle`) | `opencode-specific` | `.opencode/plugin/governance.js` | Lógica de wiring (quais tools são mutating, como interceptar hooks) é exclusiva da API de plugin do OpenCode; ela **invoca** os hooks canônicos (item 6), mas a lógica de decisão em si não tem equivalente nos outros três. | `.opencode/plugin/governance.js:11-12,267-320` |
| 18 | Workflows de CI/CD (`.github/workflows/*.yml`, `.github/actions/setup-ai-spec/action.yml`) | Fora do esquema de classificação desta tarefa | `.github/workflows/`, `.github/actions/` | Não são regras de governança de agente de IA — são pipeline de build/release do próprio repositório, sem equivalente conceitual em `claude-specific`/`codex-specific`/`copilot-specific`/`opencode-specific`. Escaneados por completude de RF-01 ("varrer... `.github/`"), mas não recebem um dos cinco rótulos porque não há "regra distribuída entre fornecedores" a classificar — é infraestrutura de CI, não contrato de comportamento de agente. | `.github/workflows/test.yml`, `.github/workflows/codeql.yml`, `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml`, `.github/workflows/acp-live.yml`, `.github/workflows/hooks-live.yml`, `.github/workflows/test-setup-action.yml`, `.github/actions/setup-ai-spec/action.yml` |
| 19 | `.agents/config.yaml` (`max_tasks_per_prd: 13`) | `universal` (configuração, não regra de comportamento) | `.agents/config.yaml` | Cascata de configuração já documentada em `AGENTS.md`/`docs/config-hierarchy.md`; item citado pelo PRD como exemplo de chave lida apenas pela prosa da skill, não por código. Não é uma regra a duplicar entre fornecedores — é config de execução do harness. Registrado por completude do escaneamento de `.agents/`. | `.agents/config.yaml:1`; PRD, seção "Riscos de Integração" |

## Subconjunto `universal` (fechado — entrada da Tarefa 4.0)

A tarefa 4.0 (RF-09..RF-11, RF-16) canonicaliza **exatamente duas regras** para
`.agents/policies/`, conforme a Ordem de Build da técspec (P3: "as duas regras movidas byte a
byte"):

1. **R-GOV-001** (`.claude/rules/governance.md`) — item 1 do inventário.
2. **R-STYLE-001** (`.claude/rules/code-style.md`) — item 2 do inventário.

Os demais itens classificados `universal` (3, 5, 6, 7) **já estão canonicalizados
corretamente** por mecanismos existentes (`AGENTS.md` como fonte única; pares de espelhamento de
skills/hooks/scripts) e não entram no escopo de migração de P3/P4 — não há regressão a corrigir
neles nesta entrega.

Os itens 4 e 8, classificados `universal` com duplicação manual identificada, **ficam fora do
escopo de ação de RF-11/RF-13 nesta entrega** por decisão explícita da especificação técnica (P3/P4
fecham só #1 e #2). Permanecem registrados aqui como achado de auditoria para não se perderem —
uma regra `universal` duplicada continua sendo `universal` mesmo quando a ação de remoção de
duplicata não está no lote atual.

Nenhum item deste inventário permanece classificado como indefinido.

## RF-14 e RF-54 — Decisão sobre `.agents/workflows/`

Esta tarefa **não cria** `.agents/workflows/`. A decisão e seu gatilho de reabertura já estão
registrados no PRD e são apenas referenciados — não reescritos — aqui, conforme instrução da
tarefa (`## Detalhes de Implementação`: "Não duplicar conteúdo aqui"):

- **RF-14** (`prd.md`, Bloco B, RF-14): nenhum conteúdo de workflow universal foi identificado
  durante o levantamento deste inventário (itens 1-19 acima) que não caiba em skill (`.agents/skills/`)
  ou em policy (`.agents/policies/`, a ser criado em P3/4.0). Confirmado varrendo `.claude/`,
  `.codex/`, `.opencode/`, `.github/`: todo mecanismo de orquestração encontrado é ou skill
  (item 5), ou hook (item 6), ou script de validação (item 7), ou wiring vendor-specific (itens
  9-17) — nenhum é um "workflow universal" autônomo sem lar.
- **RF-54** (`prd.md`, Bloco H, RF-54): o item "workflows universais são compartilhados" do
  Definition of Done da User Story é respondido **por ausência de objeto**: não existe workflow
  universal identificado que não caiba em skill ou policy. Este inventário é a evidência empírica
  que sustenta essa resposta — a varredura completa das oito localizações do escopo (RF-01) não
  revelou nenhum candidato.
- **Gatilho de reabertura** (já registrado em `prd.md`, seção *Fora de Escopo*): "reabre quando
  houver conteúdo de workflow universal identificado que não caiba em skill ou policy". Este
  inventário confirma que, na data de execução desta tarefa, esse gatilho **não foi acionado**.

## Critérios de Sucesso — Verificação

- **Inventário commitado e revisável em diff, sem nenhum item classificado como indefinido:**
  atendido — 19 itens, cada um com um dos cinco rótulos ou com justificativa explícita de exclusão
  do esquema (item 18).
- **O subconjunto `universal` está fechado e é a lista de entrada da tarefa 4.0:** atendido — seção
  dedicada acima nomeia as duas regras que entram em P3/P4 (itens 1 e 2) e distingue-as das demais
  regras `universal` já canonicalizadas ou fora do lote atual.
- **Nenhum arquivo foi movido nesta tarefa:** atendido — esta tarefa é somente leitura sobre o
  repositório; nenhum `Write`/`Edit`/`git mv` foi executado em arquivo de governança. Único arquivo
  criado é este próprio inventário.
