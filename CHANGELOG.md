# Changelog

## 0.21.0 (2026-05-16)

### Features
- **governance:** fecha lacunas multi-tool e distribui vendor .agents/lib (504ec70)

## 0.20.0 (2026-05-15)

### Features
- **execute-all-tasks:** orquestrador PRD com hooks de enforcement (1e35ecb)

## 0.19.1 (2026-05-08)

### Bug Fixes
- **integration:** atualiza skill budgets apos hardening de invocacao (9635f4d)

## 0.19.0 (2026-05-08)

### Features
- **config:** adiciona runtime config carregada de .claude/config.yaml (a5701da)
- **triggers:** sistema de triggers de revisao por linguagem (fbbbc38)

### Documentation
- adiciona matriz de degradacao, equivalencia runloop e tests scripts (9e037ee)

### Chores
- **skills:** melhora robustez de invocacao e validacao de schema (66d1eec)

## 0.18.1 (2026-05-08)

### Bug Fixes
- **taskloop:** serializa stdout+stderr concorrentes em liveOut do runCmd (246427c)
- **upgrade:** elimina falso-positivo SCHEMA DIVERGENTE em upgrade --check (9ab214b)
- **version:** elimina data race em Version mutado por testes paralelos (2a385cf)
- **fs:** CopyFile sobrescreve destino read-only existente (aadd61c)
- **fs:** WriteFile sobrescreve arquivos read-only existentes (ba2efba)

## 0.18.0 (2026-05-06)

### Features
- **taskloop:** enriquece revisao consolidada com contexto PRD/spec e isola diff do bugfix (38555c1)

## 0.17.0 (2026-05-05)

### Features
- **taskloop:** adiciona Service.RunLoop com selector, acceptance, evidence, final reviewer, bugfix loop e reservation planner (5467f6b)

## [Unreleased]

### Features
- **adapters/task-executor:** os 4 generators (Claude/GitHub/Gemini/Codex) agora emitem o bloco YAML literal do contrato de retorno (`status`/`report_path`/`summary`) no agent file de `task-executor`. Fecha falso-positivo `failed: contract violation` em `execute-all-tasks` quando o subagent retornava prosa em vez de YAML. Auditoria A02. Regressão: `TestGenerate_executeTaskYAMLContract_allTools` cobre os 4 tools + guard `TestGenerate_nonExecuteTaskHasNoYAMLContract`.
- **install/hooks:** nova função `copyToolValidationHooks` distribui `validate-preload.sh` e `validate-governance.sh` para Codex e Copilot (paridade total com Claude/Gemini). Auditoria A01. Hooks novos: `.codex/hooks/validate-governance.sh`, `.github/hooks/validate-preload.sh`, `.github/hooks/validate-governance.sh` (padrão env-based, espelhando convenção Gemini). Regressão: `TestInstall_Copilot_CopiesValidationHooks`, `TestInstall_Codex_CopiesGovernanceHook`.
- **install/.agents/lib:** novo passo 2.6 distribui vendor canônico shell (`check-invocation-depth.sh`, `parse-hook-input.sh`) para `<projeto>/.agents/lib/`. Skills e hooks deixam de depender exclusivamente do mirror legado `scripts/lib/` — cascata `.agents/lib/` → `scripts/lib/` resolve no primeiro. Regressão: `TestInstall_DistributesAgentsLib`, `TestInstall_AgentsLib_AbsentSource_NoError`.
- **execute-all-tasks:** nova skill orquestradora de PRD que spawna um subagent fresh por tarefa para isolar contexto (≤100 tokens/task no orquestrador), respeita o DAG declarado em `tasks.md`, paraleliza waves marcadas como `Paralelizável` quando o tool ativo suporta nativamente, halt-first em status não-`done`, retomada idempotente. v1.5.1.
- **adapters:** `GenerateGeminiAgents` e `GenerateCodexAgents` geram `task-executor` agent files automaticamente para Gemini (`.gemini/agents/`) e Codex (`.codex/agents/`), espelhando o padrão de Claude/Copilot — paridade multi-tool dos 8 agents processuais.
- **hooks/orchestrator:** 4 hooks bash de enforcement programático (`post-execute-task.sh`, `pre-execute-all-tasks.sh`, `post-wave.sh`, `subagent-stop-wrapper.sh`) cobrindo F2 (evidence path), F13 (path absoluto), F17 (PREFLIGHT_DONE), F18 (cross-PRD spec-hash), F24 (escalation de remark crítico), F25 (checkpoint), F27 (ciclo cross-PRD), F29 (gaps numéricos), F31 (orchestrator checkpoint), F35 (git revert).
- **install/SubagentStop:** `defaultClaudeSettings()` registra `subagent-stop-wrapper.sh` como SubagentStop hook com matcher `task-executor` — auto-invocação no Claude Code sem dependência de o LLM lembrar.
- **create-tasks v1.5.0:** templates `tasks-template.md` e `task-template.md` ganham coluna/seção `Skills` mandatória; Etapa 4.1 faz descoberta agnóstica de skills processuais via match semântico contra `description` em `.agents/skills/`; Etapa 5.5 valida sincronia entre coluna e seção.
- **create-technical-specification v1.2.0:** adiciona seção `## Resolução de paths` espelhando o padrão das outras 4 skills core (`AI_TASKS_ROOT`/`AI_PRD_PREFIX` documentados) — fecha lacuna de paridade documental que poderia divergir paths em repos com `tasks_root` customizado. Auditoria A03.
- **create-technical-specification v1.1.0:** template injeta `<!-- spec-hash-prd: ... -->` no header (rastreabilidade do PRD consumido + drift detection downstream).
- **create-prd v1.2.1:** Etapa 1.4 detecta artefatos downstream (techspec, tasks, reports) e exige confirmação humana (best-effort enforcement) antes de editar PRD existente.
- **execute-task v1.4.0:** carrega skills processuais declaradas em `## Skills Necessárias` (descoberta agnóstica); Stage 4 escala `APPROVED_WITH_REMARKS` com tags `[critical|security|blocker|high]` para `BLOCKED`; Stage 5 escreve checkpoint YAML antes de mutar `tasks.md` (proteção contra crash mid-flight).
- **scripts/test-hooks:** test harness com 14 asserts empíricos validando F18/F27/F29/F35/F25 com fixtures sintéticos descartáveis.
- **scripts/sync-hooks + check-hooks-sync:** automação de sincronia canônico→mirrors (.agents, .gemini, .codex, .github, embedded) com pre-commit hook + step CI.
- **docs/execute-all-tasks-guide.md:** guia operacional de 570 linhas cobrindo prompts copy-paste por tool, anti-padrões, fluxos end-to-end e tabela comparativa de capacidades por tool.
- **taskloop:** orquestrador `Service.RunLoop` com `TaskSelector`, `AcceptanceGate`, `EvidenceRecorder`, `FinalReviewer`, `BugfixLoop` e `ReservationPlanner` (RF-01 a RF-08).
- **taskloop:** RF-08(a) — decisão `ActionImplement` em `APPROVED_WITH_REMARKS` reentra o `BugfixLoop` com limite rígido de 3 iterações e escalonamento humano.
- **taskloop:** telemetria `implement_promoted` por finding promovido a `Critical` via decisão `Implement`.
- **taskloop:** `buildFinalReviewInput` injeta contexto estruturado (PRD, TechSpec, Tasks, task executada, lote concluído) no payload do reviewer consolidado — reviewer passa a ter acesso aos artefatos de especificação sem precisar inferir caminhos.
- **taskloop/bugfix:** `splitReviewContext` / `attachReviewContext` preservam o cabeçalho de contexto entre iterações do `BugfixLoop`: o contexto segue para o reviewer mas não polui o prompt do `BugfixInvoker`.

### Bug Fixes
- **execute-task:** Stage 2 não dispara mais `needs_input` em tarefas non-code (docs, configs YAML/JSON, SQL, shell, MD); detecção de linguagem agora condicional ao diff (F1).
- **execute-all-tasks:** validação 4-pass do YAML retornado por subagent (formato canônico, status canônico, evidência física via `realpath` + `[ -s ]`, consistência com `tasks.md`) — fecha alucinação de path e crash silencioso (F2/F13/F25).
- **execute-all-tasks:** wait-all-then-halt em waves paralelas previne race em `tasks.md` (F3); orientação explícita para subagents usarem `flock -x` ou rename atômico em writes concorrentes.
- **execute-all-tasks:** regex canônicos estritos para `Status`, `Dependências` (com suporte cross-PRD `<slug>/<id>`), `Paralelizável` — sem parsing tolerante (F7/F12/F20).
- **execute-all-tasks:** soft timeout (result-discard, não kill) configurável via `AI_TASK_TIMEOUT_SECONDS` ou comentário `<!-- task-timeout-seconds: N -->` no task file (F10/F21).
- **execute-task:** pre-flight gates condicionais via `AI_PREFLIGHT_DONE=1` evitam re-execução redundante de `ai-spec skills --verify` em cada subagent quando orquestrador já validou (F8/F17).
- **scripts/lib/check-invocation-depth.sh:** valida `AI_TOOL` ∈ `{claude, codex, gemini, copilot}`; valor inválido → unset (modo agnóstico) com warn (F30).
- **scripts/check-skills-sync.sh:** loga `SKIP:` explicitamente quando skill é ignorada por falta de SKILL.md no canônico (F34).
- **internal/install:** `copyOrchestratorHooks` preserva permissão `+x` via `os.Chmod 0o755` após `CopyFile` (F40).
- **taskloop/reviewer:** `parseVerdict` ancorado em linha dedicada (`Verdict:`/`Veredito:`) para evitar falso positivo de `BLOCKED`/`REJECTED` por palavras-chave em texto livre.
- **taskloop/reviewer:** `partitionDiff` subdivide seção única oversize em hunks `@@` repetindo o cabeçalho do arquivo; truncamento explícito quando hunk único excede `maxDiffPartitionSize`.
- **taskloop/runloop:** ramo `bfErr` não-exhausted agora emite `final_review_verdict` preservando paridade com `VerdictRejected`.
- **taskloop/reviewer:** prompt do reviewer consolidado inclui `BLOCKED` como veredito válido — impedia que o agente retornasse veredito de bloqueio em contextos sem condições de aprovar.
- **taskloop/bugfix:** `BugfixLoop.Run` passava o `reviewInput` completo (com cabeçalho de contexto) ao `BugfixInvoker`; agora extrai e entrega apenas o diff puro, evitando contaminação do prompt de correção.

### Documentation
- **execute-all-tasks v1.5.1:** corrige linha 131 da tabela "Mapeamento por Tool" — Copilot agora descreve discovery honestamente (`.github/skills/` criado por `ai-spec install` espelhando `.agents/skills/`; demais mirrors opcionais) em vez de prometer auto-descoberta de mirrors que não existem em fresh install. Auditoria A04.
- **docs/prompts/audit-workflow-skills.md:** novo prompt de auditoria rigorosa dos 5 skills core (PRD/TechSpec/Tasks/ExecuteTask/ExecuteAllTasks) cobrindo rastreabilidade, carregamento sob demanda, paridade multi-tool e robustez.
- **docs/execute-all-tasks-guide.md:** novo guia operacional de 570 linhas — pré-requisitos obrigatórios, anatomia dos inputs (tasks.md, task files), regras invioláveis, prompts copy-paste por tool (Claude/Codex/Gemini/Copilot), regras DTPS para paralelismo, fluxos end-to-end e tabela comparativa de capacidades por tool.

### Chores
- **scripts/sync-skills + check-skills-sync:** sync agora cobre 4 hooks orquestrador para 9 mirrors (`.claude/.codex/.gemini/.github/.agents` + 5 embedded), vendor `.agents/lib/` para `internal/embedded/assets/.agents/lib/`; check valida zero drift de hooks orquestrador (4×9), presença de hooks de validação por tool (2×8) e paridade dupla das libs (legacy + embedded).
- **install:** `BaseSkills` inclui `execute-all-tasks`; `installCodex`/`installGemini`/`installClaude`/`installCopilot` distribuem `task-executor` agent files + 4 hooks orquestrador para 5 dirs (.claude, .agents, .gemini, .codex, .github).
- **install:** `defaultClaudeSettings()` registra `SubagentStop` matcher `task-executor` para auto-invocação do wrapper de validação programática.
- **contextgen + upgrade:** allowlists internas incluem `execute-all-tasks` para distribuição correta no Codex `config.toml`.
- **scripts:** novos `check-skills-sync.sh`, `check-hooks-sync.sh`, `sync-hooks.sh`, `test-hooks.sh`, `git-hooks/pre-commit`; integrados a Makefile (`check-skills-sync`, `check-hooks-sync`, `test-hooks`) e workflow `.github/workflows/test.yml`.
- **tasks/prd-portability-parity + prd-taskloop-execution-validation:** migração para v1.4+ template (coluna `Skills` em tasks.md, seção `## Skills Necessárias` em todos os 17 task files).

## 0.16.0 (2026-04-25)

### Features
- **specdrift:** add sync-spec-hash command and fix taskloop pipeline blockers (b0f005f)

## 0.15.1 (2026-04-24)

### Bug Fixes
- **taskloop/parser:** detectar colunas Status e Dependências dinamicamente no header da tabela (3e1ec70)

## 0.15.0 (2026-04-24)

### Features
- **taskloop:** add bugfix phase, unlimited iterations and context-enriched prompts (75601c6)

## 0.14.3 (2026-04-24)

### Bug Fixes
- **taskloop/parser:** deduplica IDs em ParseTasksFile para evitar bloqueio de tasks elegiveis (6f92a95)

## 0.14.2 (2026-04-24)

### Bug Fixes
- **skills:** add missing version and effects.md to bubbletea skill (524df10)

## 0.14.1 (2026-04-24)

### Bug Fixes
- **taskloop:** corrige overwrite de postStatus e extrai classifyIterationOutcome (db28c51)

## 0.14.0 (2026-04-23)

### Features
- **taskloop:** enforce isolation and resume in-progress tasks (ccb8639)

## 0.13.0 (2026-04-22)

### Features
- **cli:** add skill-bump command, version --skills flag and skill_versions manifest field (ac90988)

## [Unreleased]

### Bug Fixes
- **taskloop:** corrige sobrescrita incondicional de `postStatus` pelo `tasks.md` — o fallback para `tasks.md` agora so ocorre quando o task file nao atualizou o status (`postStatus == preStatus`), evitando que tarefas concluidas corretamente sejam marcadas como "status inalterado" e puladas
- **taskloop/parser:** `ParseTasksFile` agora deduplica entradas por ID, mantendo apenas a primeira ocorrencia — tabelas auxiliares (ex: "Cobertura de Requisitos") com os mesmos IDs numericos corrompiam o `statusMap` e bloqueavam tasks elegiveis (`pending` com deps satisfeitas) impedindo o `task-loop` de encontrar trabalho executavel
- **taskloop/parser:** `ParseTasksFile` detecta dinamicamente os indices das colunas Status e Dependencias a partir do header da tabela markdown — tabelas com coluna Arquivo entre Titulo e Status deslocavam os indices fixos, fazendo o parser ler o nome do arquivo como status e classificar todas as tasks como inelegiveis
- **taskloop/isolation:** `extractTaskRows` ignora silenciosamente linhas com menos de 5 colunas em vez de retornar erro — tabelas auxiliares como "Cobertura de Requisitos" que compartilham IDs numericos bloqueavam o snapshot de isolamento antes de cada iteracao do task-loop
- **taskloop/agent:** `detectReferences` nao identifica mais projetos Go como Python — o padrao `"pip"` foi substituido por `"pip install"`, `"pip3"` e outros sufixos especificos para evitar falso positivo com a palavra `"pipeline"`

### Refactor
- **taskloop:** extrai `classifyIterationOutcome` — funcao pura sem parametro de ferramenta que centraliza a logica de decisao de resultado de iteracao (skip, abort, note, runReviewer), tornando-a isoladamente testavel
- **taskloop/agent:** `BuildPrompt` migrado para template `go:embed` (`executor_template.tmpl`) com criterios de execucao nao-negociaveis embutidos; `BuildPromptContext` extrai o sumario de arquitetura da `techspec.md` e detecta automaticamente as referencias relevantes a carregar (`go-implementation`, `ddd`, `security`, `tests`)
- **taskloop/reviewer:** `review_template.tmpl` ampliado com campos `CompletedTasks` e `RiskAreas`, focos de revisao estruturados (corretude, regressao, seguranca, testes, divida tecnica) e formato de saida esperada com veredicto final; `detectRiskAreas` detecta automaticamente areas de risco (performance, seguranca, contratos, concorrencia, persistencia) a partir da techspec e do diff
- **taskloop/reviewer:** `BuildBugfixPrompt` construido a partir do template `go:embed` (`bugfix_template.tmpl`) com placeholders para achados da revisao, diff original e contexto da task
- **version:** `ResolveFromExecutable()` resolve o arquivo `VERSION` adjacente ao binario seguindo symlinks, substituindo a leitura via `ReadVersionFile(sourceDir)` no `install` e `upgrade`
- **skills:** `isValidSemver` renomeado para `IsValidSemver` (exportado) para reutilizacao no pacote `upgrade`

### Tests
- **taskloop:** adiciona matriz de paridade semantica (10 cenarios x 4 ferramentas = 40 sub-testes), testes de isolamento de sessao entre tasks, testes de determinismo de prompt, testes de integracao com mock binaries e testes de reproducao do bug de status

### Features
- **specdrift:** novo comando `sync-spec-hash <tasks.md>` que recalcula os SHA-256 de `prd.md` e `techspec.md` e atualiza ou insere os comentarios `<!-- spec-hash-{label}: ... -->` em `tasks.md`; idempotente — nao reescreve o arquivo quando os hashes ja estao corretos
- **templates:** `prd-template.md` passa a incluir secao `## Requisitos Funcionais` com formato `RF-nn` obrigatorio; `tasks-template.md` passa a incluir comentarios `spec-hash` e secao `## Cobertura de Requisitos` — ambos atualizados nos quatro locais de copia (`.agents/`, `.claude/`, `.github/`, `internal/embedded/`)
- **taskloop:** retoma tasks com status efetivo `in_progress` antes de abrir novas pendentes, reconciliando `tasks.md` com o status real do arquivo individual da task
- **taskloop:** adiciona guardrails de isolamento para executor e reviewer; o loop aborta e restaura snapshot quando o agente altera outras rows de `tasks.md`, outros arquivos de task ou arquivos protegidos do PRD
- **taskloop:** suporta `MaxIterations == 0` como modo de iteracoes ilimitadas; o log de inicio exibe "ilimitado" no lugar do numero e o loop continua ate nao haver tasks pendentes
- **taskloop:** fase de bugfix automatica pos-revisao — quando o reviewer retorna exit code != 0, o executor e reinvocado via `invokeBugfix` com prompt de bugfix estruturado; a fase tem guardrails de isolamento proprios e o resultado e registrado em `BugfixResult` na iteracao
- **taskloop/report:** `BugfixResult` adicionado a `IterationResult`; o report avancado exibe sub-linha de duracao/exit-code do bugfix e secao detalhada com output e nota de isolamento
- **skills:** adiciona a skill externa `bubbletea` ao `skills-lock.json`
- **skill-bump:** novo comando `skill-bump <path>` que detecta skills alteradas desde a ultima tag via `git diff` e atualiza automaticamente o campo `version` no frontmatter `SKILL.md`; suporta `--dry-run` para inspecionar sem alterar arquivos
- **version:** flag `--skills` exibe versoes das skills; aceita `embedded`, `installed` ou sem valor (ambos) — ex.: `ai-spec-harness version --skills`
- **manifest:** campo `skill_versions` registra o mapa `skill → version` no manifesto durante `install` e `upgrade`
- **upgrade:** modo `--check` exibe divergencia entre a versao do CLI e a versao registrada no manifesto quando diferem

### Documentation
- remove arquivos de prompt obsoletos de `docs/prompt/` e `docs/prompts/`; adiciona `docs/prompts/taskloop-paridade-multiagente.md`

## 0.12.0 (2026-04-22)

### Features
- **taskloop:** add advanced mode with independent executor and reviewer profiles (fdbfee0)

## 0.11.2 (2026-04-21)

### Bug Fixes
- **taskloop:** support task-X.Y- filename convention in matchesTaskPrefix (39d1f2d)

## [0.11.2] - 2026-04-21

### Fixed
- **taskloop:** `matchesTaskPrefix` agora reconhece a convenção `task-X.Y-descricao.md` além das convenções `X-` e `X.Y-`, evitando que tasks com esse padrão sejam ignoradas e bloqueiem toda a cadeia de dependências no task-loop

## 0.11.1 (2026-04-21)

### Bug Fixes
- **taskloop:** isolate Unix syscalls behind build tags for Windows cross-compilation (1a4cfec)

## [0.11.1] - 2026-04-21

### Fixed
- **taskloop:** isolate Unix-only syscall fields (`Setpgid`, `syscall.Kill`) behind `//go:build !windows` to fix cross-compilation failure for `windows_amd64` and `windows_arm64` targets

## 0.11.0 (2026-04-21)

### Features
- **taskloop:** add report summary, update agent flags and inject AGENTS.md in prompt (43d50d0)

### Documentation
- **readme:** add task execution alternatives without task-loop (1ff3f75)
- **readme:** explain task-loop as automatic equivalent of per-session discipline (005c373)
- **readme:** add table of contents with anchor links for all sections (ff32669)
- **readme:** add execute-task per-session discipline with context reset and fidelity rules (adcacaa)
- **readme:** add mandatory skills usage guide with contracts and checklists (e104cab)

## [0.11.0] - 2026-04-21

### Added
- **taskloop:** seção "Resumo" no relatório com contagem de tasks por status (sucesso, puladas, falhadas)
- **taskloop:** instrução explícita de leitura de AGENTS.md no prompt do agente (RF-04, contrato de carga base)
- **taskloop:** flag `--bare` no Claude para pular carregamento automático de CLAUDE.md
- **taskloop:** atualizar flags do Codex para `exec --dangerously-bypass-approvals-and-sandbox`
- **taskloop:** atualizar flags do Copilot para `--autopilot --yolo`

### Fixed
- **taskloop:** guard clause em `ReadTaskFileStatus` para campos de status vazios ou com whitespace

### Changed
- **taskloop:** substituir `os.ReadFile` e `os.Stat` por `fs.FileSystem` em parser e taskloop para testabilidade com FakeFileSystem

### Documentation
- **readme:** adicionar sumário, guia mandatório de skills, disciplina execute-task por sessão e alternativas de execução sem task-loop
- **prompts:** adicionar prompt de execução sequencial automatizada de tasks

## 0.10.4 (2026-04-20)

### Bug Fixes
- **taskloop:** kill process group on cancel and update codex flags (39505f4)

### Documentation
- **readme:** add execute-task and task-loop usage examples (0bb6688)
- **readme:** reorganize sections and expand usage guides (9b4016b)

## 0.10.3 (2026-04-20)

### Bug Fixes
- **brew:** use Ruby chmod method to satisfy Homebrew FormulaAudit (d500dfe)

## 0.10.2 (2026-04-20)

### Bug Fixes
- **brew:** add chmod around xattr to fix post_install permission denied (a52e58a)

## 0.10.1 (2026-04-20)

### Bug Fixes
- **ci:** corrigir smoke test com comando inexistente detect (2c9aeb6)

## 0.10.0 (2026-04-20)

### Features
- **cli:** adicionar skills check, telemetria trend e lint strict — v0.10.0 (3ef9792)

### Documentation
- **readme:** documentar todas as opcoes de correcao do Gatekeeper do macOS (30a3785)
- **readme:** documentar workaround do Gatekeeper do macOS para binario nao assinado (b9d1823)

## 0.10.0 (2026-04-20)

### Added

- **skills check:** novo comando para verificar versões de skills externas contra `skills-lock.json`; detecta upgrades compatíveis (minor/patch) e potenciais quebras de interface (major bump) (`cmd/ai_spec_harness/skills.go`, `internal/skillscheck/`)
- **inspect --brief / --complexity:** exibe referências carregadas por skill por nível de complexidade via `contextgen.Loader` (`cmd/ai_spec_harness/inspect.go`)
- **telemetry report --trend:** evolução semanal de invocações nas últimas 4 semanas em formato texto ou JSON (`internal/telemetry/trend.go`)
- **telemetry report --budget-check:** verifica budget de invocações por skill com saída em texto ou JSON
- **telemetry report --top-skills:** ranking de skills por volume de uso
- **lint --strict:** trata invariantes `BestEffort` de paridade como erros; sem a flag, exibe apenas avisos (`cmd/ai_spec_harness/lint.go`)
- **contextgen.Loader:** carrega referências por skill com suporte a `brief` e `complexity` (`internal/contextgen/loader.go`)
- **requireFlag:** validação de flags obrigatórias com mensagem amigável em PT-BR e exemplo de uso real (`cmd/ai_spec_harness/flags.go`)
- **CodeQL:** workflow de análise de segurança estática adicionado ao CI (`.github/workflows/codeql.yml`)
- **hook validate-token-budget:** verificação de budget de tokens integrada ao Claude Code (`.claude/hooks/validate-token-budget.sh`)
- **docs/troubleshooting.md:** guia de 12 problemas comuns com sintoma, causa, solução e verificação
- **docs/skill-schema.json:** schema JSON para validação de `SKILL.md`

### Changed

- **lint:** detecta tools e langs automaticamente e verifica invariantes `BestEffort` de paridade em toda execução
- **inspect:** integra `contextgen.Loader` para exibir referências por skill nos modos `--brief` e `--complexity`
- **telemetry report:** expandido com três novos modos de saída (`--trend`, `--budget-check`, `--top-skills`)
- **test.yml:** pipeline de CI expandido com novos targets, cobertura por pacote e testes de integração
- **Makefile:** novos targets para fuzz, cobertura e validação de schema
- **skills references:** atualizações em `agent-governance`, `go-implementation` e `object-calisthenics-go`

### Tests

- Fuzz tests adicionados em `internal/config`, `internal/detect` e `internal/manifest`
- Testes de integração para budget de tokens e skills externas (`internal/integration/token_budget_skill_test.go`)
- Contrato CLI expandido com novos subcomandos (`cmd/ai_spec_harness/cli_contract_test.go`)
- Novos testes: `flags_test.go`, `validation_test.go`, `skillscheck_test.go`, `trend_test.go`, `loader_test.go`

---

## 0.9.2 (2026-04-20)

### Bug Fixes
- **brew:** remover quarentena do Gatekeeper via post_install na Formula (3e21aab)

## 0.9.1 (2026-04-20)

### Bug Fixes
- **ci:** corrigir dirty state do GoReleaser por semver_output.txt no workspace (7540c7f)

### Documentation
- **readme:** atualizar instalacao para Formula (brew install) (32206f2)

## [Unreleased]

### Added

- **taskloop — modo avancado (executor + reviewer independentes):** flags `--executor-tool`, `--executor-model`, `--reviewer-tool` e `--reviewer-model` permitem configurar agentes distintos para execucao e revisao; modo simples via `--tool` permanece inalterado e retro-compativel
- **taskloop — `ExecutionProfile` (Value Object):** representa a configuracao de um papel (executor ou reviewer) com validacao fail-fast no construtor; campos `role`, `tool`, `provider` e `model`
- **taskloop — `CompatibilityTable`:** cataloga combinacoes ferramenta+modelo reconhecidas (Claude, Codex, Gemini, Copilot); flag `--allow-unknown-model` para aceitar combinacoes fora do catalogo sem erro
- **taskloop — `BuildReviewPrompt` com `go:embed`:** gera prompt de revisao a partir de template embutido (`review_template.tmpl`) ou de template customizado via `--reviewer-prompt-template`
- **taskloop — deteccao de auth error e guidance por ferramenta:** `isAuthError` detecta padroes de falha de autenticacao no output do agente; `authGuidance` retorna instrucao especifica por ferramenta; `warnClaudeAuth` alerta antes do loop quando ANTHROPIC_API_KEY esta ausente
- **taskloop — `LiveOutputSetter` interface:** permite injetar um `io.Writer` para streaming de output do agente em tempo real; usado internamente por `runCmd` para tee do stdout
- **taskloop — fallback model nativo por papel:** flags `--executor-fallback-model` e `--reviewer-fallback-model` passam `--fallback-model` ao claudeInvoker; outros invokers ignoram silenciosamente
- **taskloop — fallback tool pre-loop:** flag `--fallback-tool` para validacao de disponibilidade antes do inicio do loop principal
- **taskloop — relatorio modo avancado:** `ReviewResult` armazena resultado da revisao; `Report` ganha campos `Mode`, `ExecutorProfile` e `ReviewerProfile`; renderizacao separada para modo simples e avancado com coluna Papel
- **taskloop — dry-run avancado:** exibe modo, perfis resolvidos com status de compatibilidade, template de revisao e preview do prompt para a primeira task elegivel
- **metrics — modo `brief` no gather:** estimativa de tokens por referencia de skill em modo resumido (150 tokens fixos por entrada) em vez de contagem real do arquivo completo
- **docs — `docs/task-loop-reference.md`:** guia consolidado de flags, heuristicas e alternativas do task-loop (referenciado no README)
- **docs — `docs/skills-usage-guide.md`:** guia de uso das skills com contratos de entrada, prompts mandatorios e criterios de aceite
- **docs — prompts de execucao sequencial:** prompts em `docs/prompts/` para execucao automatizada de tasks com modelos por papel
- Nova skill `finalize-changelog-readme-push` para consolidar atualizacao de `CHANGELOG.md`, revisao de `README.md`, `git add .`, commit semantico e `git push` com guardrails de confirmacao

### Fixed

- **ci:** corrigir dirty state do GoReleaser causado por `semver_output.txt` não rastreado no workspace git

### Tests

- Expandir cobertura de testes unitários em `detect`, `install`, `metrics`, `scaffold`, `uninstall`, `upgrade` e `wrapper`
- Adicionar testes de integração para skills externas e orçamento de tokens
- Adicionar benchmarks para `metrics`, `parity` e `skills/schema`
- Adicionar contrato CLI em `cmd/ai_spec_harness/cli_contract_test.go`

### CI

- Atualizar `test.yml` com melhorias no pipeline de testes
- Adicionar script `scripts/check-package-coverage.sh` para verificação de cobertura por pacote

### Docs

- Adicionar ADR-006: telemetria opt-in com append-only log
- Adicionar ADR-007: workaround stateless para Copilot CLI
- Adicionar ADR-008: paridade multi-tool com 29 invariantes semânticas em 3 níveis
- Adicionar `.aiignore` e `.claudeignore` para controle de contexto dos agentes
- Atualizar governança operacional em `AGENTS.md`, `CLAUDE.md`, `CODEX.md`, `COPILOT.md` e `GEMINI.md`
- Expandir `docs/cli-schema.json` com novos comandos
- Atualizar `Makefile` com novos targets

## 0.9.0 (2026-04-20)

### Features
- **release:** migrar Homebrew de Cask para Formula (c77f61a)

### CI
- **release:** adicionar step para corrigir ordem de stanzas no Homebrew Cask após GoReleaser (9143e49)

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.8.0] - 2026-04-20

### Breaking Changes

- Resetar repositório para catálogo de skills (`65f25ae`)

### Added

- Implementar CLI Go de governança de IA (`7083197`)
- Adicionar distribuição via Homebrew e expandir CLI de governança (`937c15b`)
- Adicionar assets embutidos e validações de spec (`aa62ac9`)
- Separar modelo de custo em três eixos e adicionar gate de regressão — `metrics` (`c004882`)
- Adicionar workflow de dry-run e comandos semver-next e changelog — `release` (`1cd2b1a`)
- Adicionar pacote para resolver git refs em diretório temporário — `gitref` (`e8b29dc`)
- Adicionar flag `--ref` para install e upgrade a partir de git ref (`45e0ce3`)
- Adicionar scoring por focus-paths e suporte a monorepo Python — `detect` (`786d145`)
- Adicionar wrapper e verificação de pré-requisitos de skills (`61b18ae`)
- Expand skills baseline and document task loop flow — `governance` (`6237f5f`)
- Adicionar feedback loop de telemetria, spec-driven e governança multi-agente (`4d7a780`)
- Adicionar Codex, Copilot e parser de telemetria (`66dd041`)

### Fixed

- Usar /tmp para semver_output e ajustar validação de working tree — `release-dry-run` (`e5c3529`)
- Alinhar flags de autonomia total para todas as ferramentas — `taskloop` (`9517323`)
- Corrigir bad substitution ao interpolar mensagem de commit no bash — `ci` (`3c3b0d9`)
