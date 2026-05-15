# Guia Completo: `execute-all-tasks`

> Orquestrador agnóstico de PRD que executa todas as tarefas em subagents fresh (contexto isolado por tarefa), respeita o DAG declarado em `tasks.md`, paraleliza quando o tool ativo suporta, halt-first em falha, retomada idempotente.

Este guia é a referência operacional definitiva. Cobre quando usar, pré-requisitos obrigatórios, prompts completos copy-paste para cada um dos 4 tools (Claude Code, Codex CLI, Gemini CLI, Copilot CLI), padrões de eficiência, anti-padrões com falsos positivos conhecidos, e fluxos end-to-end reais.

---

## Sumário

1. [Quando usar / Quando NÃO usar](#1-quando-usar--quando-não-usar)
2. [Pré-requisitos obrigatórios](#2-pré-requisitos-obrigatórios)
3. [Anatomia dos inputs](#3-anatomia-dos-inputs)
4. [Regras invioláveis](#4-regras-invioláveis)
5. [Prompts completos por tool](#5-prompts-completos-por-tool)
6. [Melhores práticas de paralelismo](#6-melhores-práticas-de-paralelismo)
7. [Halt-first: como recuperar](#7-halt-first-como-recuperar)
8. [Retomada idempotente](#8-retomada-idempotente)
9. [Monitoramento e evidência](#9-monitoramento-e-evidência)
10. [Anti-padrões e falsos positivos](#10-anti-padrões-e-falsos-positivos)
11. [Fluxos end-to-end](#11-fluxos-end-to-end)
12. [Tabela de comparação por tool](#12-tabela-de-comparação-por-tool)

---

## 1. Quando usar / Quando NÃO usar

### Use quando

- Um PRD inteiro precisa ser executado e tem **≥ 3 tarefas** com dependências entre si.
- As tarefas tocam contextos pesados (múltiplos pacotes, leituras grandes) e você quer evitar acúmulo na janela.
- Você quer **paralelizar** tarefas independentes para reduzir wall-clock.
- O PRD foi criado com `create-tasks` e `tasks.md` está canônico (tabela com `Paralelizável`, `Dependências`, `Status`).
- Você precisa de **retomada** depois de uma falha sem re-executar o que já passou.

### NÃO use quando

- Você tem **uma única tarefa** para executar. Use `execute-task` direto.
- O PRD não tem `tasks.md` canônico. Crie primeiro com `create-tasks`.
- Você está **explorando código** ou **planejando**. Use `analyze-project` / `create-prd`.
- O PRD tem tarefas que dependem de input humano em cada uma. O halt-first vai disparar imediatamente — sem ganho.
- Você está dentro de **outro subagent**. No Claude isso é bloqueante (depth limit = 1). Sempre invoque no top-level.

---

## 2. Pré-requisitos obrigatórios

São hard requirements. Se algum falhar, o orquestrador retorna `failed` ou `needs_input` sem executar nada.

### 2.1 Estrutura de arquivos

```
tasks/prd-<slug>/
├── prd.md          ← obrigatório
├── techspec.md     ← obrigatório
├── tasks.md        ← obrigatório (tabela canônica)
├── task-1.0-*.md   ← obrigatório por tarefa
├── task-2.0-*.md
└── ...
```

### 2.2 Lockfile íntegro

```bash
ai-spec skills --verify
# deve retornar exit 0
```

Se houver drift, **conserte antes** com `ai-spec skills --update` ou equivalente. O orquestrador para com `blocked` se detectar drift.

### 2.3 Spec drift / RF coverage

```bash
ai-spec check-spec-drift tasks/prd-<slug>/tasks.md
# deve retornar exit 0
```

Se houver RF não coberto, para com `blocked`. Reabra a fase de planejamento.

### 2.4 Sessão top-level

- **Claude Code**: invoque em sessão nova (`claude` no terminal limpo). Nunca de dentro de outro Agent.
- **Codex CLI**: invoque em sessão raiz (`codex`). Não use dentro de um subagent ativo (verifique com `/agent`).
- **Gemini CLI**: invoque em sessão raiz (`gemini`). Não use dentro de um `@agent-name` ativo.
- **Copilot CLI**: invoque em sessão raiz (`gh copilot`). Não use dentro de um `/fleet` aninhado.

### 2.5 Working directory

Rode a partir da **raiz do repositório** (onde mora `AGENTS.md`, `scripts/`, `.agents/`). O script `scripts/lib/check-invocation-depth.sh` resolve por path relativo a essa raiz.

---

## 3. Anatomia dos inputs

### 3.1 `tasks.md` canônico

Exemplo mínimo válido:

```markdown
<!-- spec-hash-prd: <sha256> -->
<!-- spec-hash-techspec: <sha256> -->

# Resumo das Tarefas de Implementação para <Feature>

## Metadados
- **PRD:** `tasks/prd-<slug>/prd.md`
- **Especificação Técnica:** `tasks/prd-<slug>/techspec.md`
- **Total de tarefas:** 5
- **Tarefas paralelizáveis:** 1.0, 2.0 (independentes)

## Tarefas

| # | Título | Status | Dependências | Paralelizável | Fase |
|---|--------|--------|--------------|---------------|------|
| 1.0 | Setup módulo X | pending | — | Com 2.0 | 1 |
| 2.0 | Setup módulo Y | pending | — | Com 1.0 | 1 |
| 3.0 | Integração X+Y | pending | 1.0, 2.0 | Não | 2 |
| 4.0 | Testes E2E | pending | 3.0 | Não | 3 |
| 5.0 | Documentação | pending | 4.0 | Não | 3 |
```

**Campos exigidos:**
- `#`: ID no formato `X.Y` (decimal).
- `Status`: um dos canônicos — `pending`, `in_progress`, `needs_input`, `blocked`, `failed`, `done`.
- `Dependências`: lista separada por vírgula (`1.0, 2.0`) ou `—` para nenhuma.
- `Paralelizável`: começa com `Com ` ou contém `yes`/`sim` → habilita paralelo. `Não`/`no` → sequencial.

### 3.2 Arquivo individual da tarefa

Cada `task-X.Y-<slug>.md` deve ter:

```markdown
# Tarefa X.Y: <Título>

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral
<1-2 frases descrevendo o que deve ser feito>

<requirements>
- <requisito 1>
- <requisito 2>
</requirements>

## Subtarefas
1. [ ] <passo 1>
2. [ ] <passo 2>

## Critérios de Sucesso
- <critério observável 1>

## Testes da Tarefa
- <test name>: <expectativa>

## Arquivos Relevantes
- <path>
```

### 3.3 Flag `Paralelizável` honesto

**Marque `Paralelizável=Com X.0, Y.0` SOMENTE se:**
- As tarefas tocam **pacotes/diretórios distintos**.
- Não há dependência implícita (ex.: ambas modificam o mesmo arquivo de config).
- Os testes podem rodar concorrentemente sem flaky.

**Marque `Paralelizável=Não` quando:**
- Houver risco de merge conflict.
- Alguma das tarefas mexe em estado global (DB schema, env vars, lockfiles).
- A ordem é semanticamente relevante mesmo sem dependência declarada.

---

## 4. Regras invioláveis

Estas regras vêm da SKILL.md e são **enforced** pelo orquestrador. Violá-las gera `failed`.

1. **Toda tarefa roda em subagent fresh.** O orquestrador nunca executa `execute-task` inline.
2. **Contrato de retorno do subagent é estrito.** YAML com `status`, `report_path`, `summary` (≤ 1 linha). Violação = `failed: contract violation`.
3. **Paralelismo só dispara quando:** (a) `Paralelizável=true` em tasks.md E (b) tool ativo suporta spawn paralelo. Caso contrário, degrada para sequencial.
4. **Orquestrador NUNCA muta `tasks.md`.** Apenas os subagents (via `execute-task` Stage 5) atualizam status.
5. **Halt-first.** Qualquer status diferente de `done` interrompe novas waves.
6. **Não há retry automático.** Tarefa em `failed`/`blocked`/`needs_input` é devolvida ao humano.

---

## 5. Prompts completos por tool

Estes prompts são copy-paste. Substitua `<slug>` pelo slug do seu PRD (sem prefixo `prd-`).

### 5.1 Claude Code

#### A. Invocação por slash command (preferido)

```
/execute-all-tasks <slug>
```

Exemplo concreto:

```
/execute-all-tasks portability-parity
```

#### B. Invocação descritiva (auto-trigger pela description)

```
Execute todas as tarefas do PRD `<slug>` respeitando o DAG declarado em tasks.md
e o flag Paralelizável. Use a skill execute-all-tasks. Pare imediatamente em qualquer
tarefa que retornar status diferente de done.
```

#### C. Invocação com restrições explícitas

```
Use a skill execute-all-tasks no PRD `<slug>`. Restrições obrigatórias:
1. Cada tarefa DEVE rodar em um subagent task-executor fresh (não execute inline).
2. Respeite o flag Paralelizável de tasks.md — não force paralelismo onde está marcado Não.
3. Halt-first: pare a próxima wave assim que qualquer subagent retornar status diferente de done.
4. Não mute tasks.md diretamente — somente os subagents via execute-task podem atualizar status.
5. Ao final (sucesso ou halt), escreva tasks/prd-<slug>/_orchestration_report.md.
```

#### D. Como NÃO invocar (anti-padrão)

```
❌ Vai executando as tasks uma por uma, lê o código, faz, testa, vai pra próxima
   — sem mencionar a skill nem subagent
```

Sem invocação explícita da skill, Claude tenta executar tudo inline na sessão principal — perde isolamento de contexto, vai estourar janela em PRDs com >5 tarefas.

#### Sessão recomendada

```bash
cd ~/projetos/meu-app
# Sessão fresh, sem outras tarefas no histórico:
claude
> /execute-all-tasks portability-parity
```

---

### 5.2 OpenAI Codex CLI

#### A. Invocação descritiva (recomendada)

Codex 2026 prefere instrução textual a slash commands. Use o prompt completo:

```
Use the execute-all-tasks skill on PRD `<slug>`.

Mandatory behavior:
1. Read tasks/prd-<slug>/tasks.md, prd.md and techspec.md before scheduling anything.
2. Build the DAG from the Dependencies column. Compute ready set as: status=pending AND all deps done.
3. For each ready task, spawn a fresh task-executor subagent. Pass the absolute path of the task file plus PRD/techspec/tasks paths. Each subagent must invoke the execute-task skill internally.
4. If a wave has multiple parallelizable tasks (Paralelizável=Com ...), spawn all those subagents in parallel and wait for all results before scheduling the next wave.
5. Each subagent must return ONLY a YAML block with status, report_path, summary. If a subagent returns anything else, treat as failed: contract violation.
6. Halt-first: stop scheduling new waves the moment any subagent returns status != done.
7. Never mutate tasks.md from the orchestrator — only the subagents (via execute-task) update status.
8. After all tasks complete or a halt occurs, write tasks/prd-<slug>/_orchestration_report.md using the template from .agents/skills/execute-all-tasks/assets/orchestration-report-template.md.
```

#### B. Invocação curta (quando Codex já carregou a skill)

```
Run execute-all-tasks on PRD <slug>. Strict contract: subagent per task, halt-first, no inline execution.
```

#### Sessão recomendada

```bash
cd ~/projetos/meu-app
codex --codex-profile full
> Use the execute-all-tasks skill on PRD portability-parity. [...prompt A completo...]
```

#### Atenção específica do Codex

- O default `--codex-profile full` carrega todas as skills (incluindo `execute-all-tasks`). Com `lean`, planning skills são filtradas mas a base permanece.
- Codex herda config (model, sandbox_mode) do agente pai para subagents. Garanta que o pai tem acesso a write/exec necessário para tests/lint, senão subagents falham por permissão (vai retornar `blocked`).

---

### 5.3 Gemini CLI

#### A. Slash command (preferido — slim e determinístico)

```
/execute-all-tasks <slug>
```

#### B. Invocação via `@agent` (controle fino)

```
@task-executor não — preciso do orquestrador.

Use a skill execute-all-tasks no PRD `<slug>`. Para cada tarefa elegível, delegue ao
subagent @task-executor com prompt auto-contido (path da task + prd + techspec).
Cada @task-executor deve retornar APENAS YAML com status, report_path, summary.
Halt-first em qualquer status != done. Não mute tasks.md.
```

#### C. Invocação descritiva (auto-orquestração)

```
Orquestre a execução completa do PRD `<slug>` usando a skill execute-all-tasks.
Respeite tasks.md, o flag Paralelizável e o DAG. Pare na primeira tarefa não-done
e gere o relatório agregado em tasks/prd-<slug>/_orchestration_report.md.
```

#### Sessão recomendada

```bash
cd ~/projetos/meu-app
gemini
> /execute-all-tasks portability-parity
```

#### Atenção específica do Gemini

- **Quota**: dispatch paralelo dispara N chamadas LLM simultâneas. Se sua chave tem rate limit baixo (ex.: free tier 60 RPM), force sequencial editando tasks.md temporariamente: `Paralelizável: Não` em todas as linhas.
- O `.gemini/commands/execute-all-tasks.toml` é wrapper. A definição do subagent vive em `.gemini/agents/task-executor.md` (distribuído pelo installer).
- Confirme com `/agents` que `task-executor` está listado antes de invocar.

---

### 5.4 GitHub Copilot CLI

#### A. Slash command (preferido)

```
/execute-all-tasks <slug>
```

#### B. Invocação via Custom Agent + `/fleet`

```
/fleet split task-executor

Use a skill execute-all-tasks no PRD `<slug>`. Para cada tarefa marcada Paralelizável,
distribua entre instâncias paralelas do task-executor via /fleet. Para sequenciais,
execute uma por vez. Halt-first. Relatório agregado em _orchestration_report.md.
```

#### C. Invocação descritiva (auto-discovery)

```
Execute todas as tarefas pendentes do PRD `<slug>` usando o orquestrador
execute-all-tasks. Subagent task-executor por tarefa, halt-first, não mute tasks.md.
```

#### Sessão recomendada

```bash
cd ~/projetos/meu-app
gh copilot
> /execute-all-tasks portability-parity
```

#### Atenção específica do Copilot

- Copilot 2026 é **stateful**: sessions persistem em `~/.copilot/session-state/`. Pra rodar dois PRDs em paralelo sem interferência, abra **sessions em terminais separados** (cada uma terá worktree próprio automaticamente).
- Para feedback contínuo, deixe a aba do Copilot aberta — ele streama progresso por subagent.
- `/fleet` consolida resultados automaticamente — o orquestrador detecta e mapeia a wave paralela pra esse comando nativo.

---

## 6. Melhores práticas de paralelismo

### 6.1 Quando vale a pena

Wave de 3-5 tarefas paralelas reduz wall-clock em ~3-4x (cada subagent é independente). Para waves de 2 tarefas, o overhead de spawn pode anular o ganho.

### 6.2 Como tornar `Paralelizável` honesto

Use a regra **DTPS** ao revisar `tasks.md`:

- **D**iretório: tarefas tocam pastas distintas?
- **T**estes: testes podem rodar concorrentes sem race?
- **P**ackage: alteram pacotes/módulos diferentes?
- **S**tate: nenhuma toca estado global compartilhado (DB schema, env, lockfile)?

Se todos os 4 sim → `Paralelizável=Com X.0, Y.0`. Senão → `Não`.

### 6.3 Forçando sequencial temporariamente

Cenários onde você quer desabilitar paralelismo apesar do flag:
- Debug — quer logs sequenciais e reprodutíveis.
- Quota apertada na API do LLM.
- Suspeita de conflito não-detectado em testes.

Edite `tasks.md` substituindo todos `Com ...` por `Não` (commit num branch de debug, não no main).

---

## 7. Halt-first: como recuperar

Quando o orquestrador para por status não-done, faça:

### Passo 1: identifique a tarefa que parou

Abra `tasks/prd-<slug>/_orchestration_report.md`. A última entrada com status `≠ done` é a culpada.

### Passo 2: leia o relatório individual

```
tasks/prd-<slug>/<id>_execution_report.md
```

O campo `Resultados de Validação` e `Veredito do Revisor` apontam a causa raiz.

### Passo 3: aja conforme o status

| Status | Significado | Ação |
|---|---|---|
| `failed` | erro técnico ou contract violation | Corrija o problema (bug, teste flaky, code). Use `bugfix` skill se for canônico. |
| `blocked` | dependência externa ou veredito BLOCKED do reviewer | Resolva a dependência (lockfile drift, env faltando, API externa). Não conserte com `bugfix`. |
| `needs_input` | decisão humana pendente | Tome a decisão, atualize o task file (ou prd/techspec se for divergência), re-invoque. |

### Passo 4: re-invoque o orquestrador

```
/execute-all-tasks <slug>
```

A retomada é automática — tarefas `done` são puladas. Apenas a tarefa corrigida + dependentes pendentes rodam.

### Passo 5: NÃO faça

- ❌ Editar `tasks.md` para mudar `failed` → `pending` à mão. O `execute-task` faz isso quando retomar.
- ❌ Apagar o `<id>_execution_report.md`. É auditoria.
- ❌ Pular a tarefa que falhou e tentar rodar as posteriores manualmente. Quebra a integridade do DAG.

---

## 8. Retomada idempotente

O orquestrador é projetado para ser **invocado N vezes** sem efeito colateral. Cada invocação:

1. Lê `tasks.md` do zero.
2. Recalcula `ready = pending AND deps done`.
3. Só agenda tarefas em `pending`.

Cenários onde retomar é seguro:
- Após halt em qualquer tarefa.
- Após o usuário fechar a sessão antes de terminar.
- Após `Ctrl+C` no meio de uma wave (subagents em voo terminam, próxima invocação retoma do snapshot atual).
- Após o lockfile ter sido atualizado externamente.

Cenários onde retomar **NÃO é seguro**:
- Se você mutou manualmente o status de uma tarefa de `failed` para `pending` sem corrigir a causa. O orquestrador vai re-executar e provavelmente falhar de novo.
- Se você mexeu em `prd.md` ou `techspec.md` depois das tarefas estarem `done`. O drift gate vai disparar `needs_input`.

---

## 9. Monitoramento e evidência

### 9.1 Durante a execução

**Claude Code**: olhe o terminal — cada Agent call aparece como bloco separado com o título da tarefa. Subagents paralelos aparecem como blocos lado a lado.

**Codex CLI**: use `--verbose` para ver o spawn de cada subagent. `/agent list` mostra threads ativos.

**Gemini CLI**: o painel de status mostra subagents em execução. `/agents status` lista os ativos.

**Copilot CLI**: a session view mostra progresso por subagent em real-time.

### 9.2 Após a execução

Artefatos gerados:

```
tasks/prd-<slug>/
├── <id>_execution_report.md         ← um por tarefa executada
├── _orchestration_report.md          ← rollup do orquestrador
└── tasks.md                          ← status atualizado pelos subagents
```

### 9.3 Telemetria (opcional)

```bash
export GOVERNANCE_TELEMETRY=1
```

Eventos emitidos:
- `orchestration.start` — slug, total_tasks, pending_count.
- `orchestration.wave` — wave_id, mode (sequential/parallel), task_count.
- `orchestration.task_complete` — task_id, status, duration_ms.
- `orchestration.halt` — first_non_done_task, reason.
- `orchestration.end` — final_status (done/partial/failed), total_duration_ms.

Relatório: `ai-spec telemetry report`.

---

## 10. Anti-padrões e falsos positivos

Lista crítica de erros que parecem funcionar mas vão te queimar em projeto real.

### 10.1 ❌ Invocar a skill dentro de outro subagent

```
# Você está num task-executor → tenta invocar execute-all-tasks
```

**Por quê é problema**: Claude tem depth limit 1 — falha silenciosa, cai pra inline. Outros tools não documentam o limite, comportamento imprevisível.

**Correto**: Sempre invoque no top-level. Saia do subagent atual antes.

### 10.2 ❌ "Vai executando" sem invocar a skill

```
> "Execute as tasks do PRD X, uma de cada vez"
```

**Por quê é problema**: o agente vai fazer execute-task inline na sessão principal. Janela estoura, contexto acumula, sem isolamento.

**Correto**: invoque pela slash ou descrição que cita explicitamente `execute-all-tasks`.

### 10.3 ❌ Marcar tudo como `Paralelizável=Com 1.0, 2.0, ...`

```
| 1.0 | Add field X to schema | pending | — | Com 2.0 | 1 |
| 2.0 | Add field Y to schema | pending | — | Com 1.0 | 1 |
```

**Por quê é problema**: ambas mexem no mesmo schema → conflict merge entre subagents paralelos. Vai gerar `failed` aleatório por race.

**Correto**: `Paralelizável=Não` para tarefas que tocam mesmo arquivo/recurso. Use a regra DTPS (seção 6.2).

### 10.4 ❌ Re-invocar imediatamente após halt sem investigar

```
> /execute-all-tasks portability-parity   # halt: task 3.0 failed
> /execute-all-tasks portability-parity   # esperando que "dê certo dessa vez"
```

**Por quê é problema**: o orquestrador respeita `failed` como verdade — vai pular essa task (não está mais em `pending`). Wait, na verdade vai re-executar porque o status canônico final só é `done`. Mas como a causa raiz não foi corrigida, vai falhar de novo. Você perde tempo.

**Correto**: leia o `<id>_execution_report.md`, conserte a causa, depois re-invoque.

### 10.5 ❌ Editar `tasks.md` manualmente para "forçar" progresso

```
# Mudar status: failed → done manualmente em tasks.md
```

**Por quê é problema**: contorna a auditoria. Não há `<id>_execution_report.md` correspondente. Spec drift gate pode disparar. Quebra a integridade da cadeia.

**Correto**: corrija a causa raiz, deixe o `execute-task` (dentro do subagent) atualizar o status.

### 10.6 ❌ Rodar de subdiretório

```bash
cd tasks/prd-portability-parity
claude
> /execute-all-tasks portability-parity   # falha: scripts/lib/... não encontrado
```

**Por quê é problema**: paths relativos quebram. `check-invocation-depth.sh` não resolve.

**Correto**: sempre da raiz do repo.

### 10.7 ❌ Confiar que `Paralelizável=Não` em todas as tarefas é "mais seguro"

```
| 1.0 | ... | pending | — | Não | 1 |
| 2.0 | ... | pending | — | Não | 1 |
| 3.0 | ... | pending | — | Não | 1 |
```

**Por quê é problema**: subprometer paralelismo é um falso positivo no sentido inverso — você não quebra nada, mas perde 3-5x throughput em PRDs grandes. Em time real, isso significa horas extras de espera.

**Correto**: aplique DTPS honestamente. Se 1.0 e 2.0 são genuinamente independentes, marque paralelizável.

### 10.8 ❌ Misturar invocação manual de `execute-task` com `execute-all-tasks`

```
# Sessão A: orquestrador rodando
# Sessão B: usuário invoca /execute-task 3.0 manualmente em paralelo
```

**Por quê é problema**: duas sessões mutando `tasks.md` ao mesmo tempo — race. Resultados imprevisíveis.

**Correto**: deixe o orquestrador conduzir. Se precisar intervir, pare-o primeiro (`Ctrl+C` ou esperar halt).

### 10.9 ❌ Esperar dry-run / preview

```
> /execute-all-tasks portability-parity --dry-run
```

**Por quê é problema**: a skill **não suporta dry-run**. A flag será ignorada, o orquestrador executa de verdade. Você acha que é simulação, mas é real.

**Correto**: para preview, leia `tasks.md` manualmente, simule o DAG mentalmente, ou rode em um worktree separado se quiser sandbox.

### 10.10 ❌ Confiar no contrato em tools onde subagent é parcial

```
# Codex: assumir que subagent é janela 100% fresca
```

**Por quê é problema**: Codex herda config (model, sandbox_mode, mcp_servers) do agente pai. Não é fresh total. Vazamentos de config podem causar comportamento diferente de Claude.

**Correto**: documente que Codex tem isolamento parcial. Para PRDs sensíveis, prefira Claude ou Copilot. Veja a tabela de comparação (seção 12).

---

## 11. Fluxos end-to-end

### 11.1 Fluxo 1 — PRD novo, do zero ao deploy

```bash
# Sessão única, Claude Code
cd ~/projetos/meu-app
claude

# 1. Discovery / requisitos
> /us-to-prd "Como admin, quero exportar usuários em CSV..."
# → tasks/prd-export-csv/prd.md

# 2. Refinar PRD
> /create-prd
# → revisão, aprovação

# 3. Spec técnica
> /create-technical-specification
# → tasks/prd-export-csv/techspec.md

# 4. Decompor em tasks
> /create-tasks
# → tasks/prd-export-csv/tasks.md + task-1.0-*.md ... task-5.0-*.md

# 5. EXECUTAR TUDO (a nova skill)
> /execute-all-tasks export-csv
# → orquestra, gera <id>_execution_report.md por tarefa,
#   _orchestration_report.md no final

# 6. Validar
> Leia tasks/prd-export-csv/_orchestration_report.md e confirme que tudo está done.

# 7. Release
> /github-diff-changelog-publisher
> /github-release-publication-flow
```

### 11.2 Fluxo 2 — Retomada após halt

```bash
# Sessão original
> /execute-all-tasks portability-parity
# Output: halt na tarefa 3.0 (failed: lockfile drift)

# Investigação
> Leia tasks/prd-portability-parity/3.0_execution_report.md
# Output: "Veredito: REJECTED. Causa: skills-lock.json drift detectado"

# Correção
> ai-spec skills --update
> # commit do lockfile atualizado

# Retomada (mesma sessão ou nova)
> /execute-all-tasks portability-parity
# Output: pula 1.0/2.0 (done), retoma de 3.0
```

### 11.3 Fluxo 3 — PRD grande dividido em sessões

PRD com 20 tarefas. Dividir em fases:

```bash
# Sessão 1 — fase 1 (tarefas 1.0-5.0)
claude
> /execute-all-tasks meu-prd
# Output: 5.0 done, halt em 6.0 (needs_input — decisão de arquitetura)

# Resolva o needs_input no task-6.0
# Edite o task file, atualize prd/techspec se necessário, commit

# Sessão 2 — retoma da 6.0 em diante (sessão FRESH)
claude   # sessão nova, janela limpa
> /execute-all-tasks meu-prd
# Output: pula 1.0-5.0 (done), executa 6.0-20.0
```

Por que sessão nova: a sessão 1 acumulou ~5 sumários de tarefas + contexto inicial. Janela está saudável, mas sessão fresh = janela 100% livre, executa as 15 restantes com mais folga.

### 11.4 Fluxo 4 — Cross-tool validation (paridade)

Use o mesmo PRD em ambos Claude e Copilot pra comparar comportamento:

```bash
# Terminal 1
cd ~/projetos/exemplo-claude
ai-spec install . --tools claude --langs go
cp -R ~/projetos/source/tasks/prd-foo ./tasks/
claude
> /execute-all-tasks foo

# Terminal 2 (em paralelo)
cd ~/projetos/exemplo-copilot
ai-spec install . --tools copilot --langs go
cp -R ~/projetos/source/tasks/prd-foo ./tasks/
gh copilot
> /execute-all-tasks foo

# Compare os dois _orchestration_report.md ao final
diff ~/projetos/exemplo-claude/tasks/prd-foo/_orchestration_report.md \
     ~/projetos/exemplo-copilot/tasks/prd-foo/_orchestration_report.md
```

### 11.5 Fluxo 5 — Sequencial forçado (quota apertada / debug)

```bash
# Backup do tasks.md
cp tasks/prd-foo/tasks.md tasks/prd-foo/tasks.md.bak

# Patch: troca todos Paralelizável para Não
sed -i.tmp 's/| Com [^|]*|/| Não |/g' tasks/prd-foo/tasks.md
rm tasks/prd-foo/tasks.md.tmp

# Rodar sequencial
gemini
> /execute-all-tasks foo

# Restaurar
mv tasks/prd-foo/tasks.md.bak tasks/prd-foo/tasks.md
```

---

## 12. Tabela de comparação por tool

| Capacidade | Claude Code | Codex CLI | Gemini CLI | Copilot CLI |
|---|---|---|---|---|
| Slash command `/execute-all-tasks` | ✅ nativo | ⚠️ prefira descrição | ✅ nativo via `.gemini/commands/*.toml` | ✅ nativo |
| Subagent `task-executor` formal | ✅ `.claude/agents/task-executor.md` | ✅ `.codex/agents/task-executor.toml` | ✅ `.gemini/agents/task-executor.md` | ✅ `.github/agents/task-executor.agent.md` |
| Isolamento de contexto por subagent | ✅ janela 100% fresca (oficial) | ⚠️ thread isolada + herança de config | ✅ isolated context loop (oficial) | ✅ janela própria (oficial) |
| Paralelismo nativo | ✅ múltiplas Agent calls/mensagem | ✅ subagents concorrentes | ✅ dispatch paralelo | ✅ `/fleet` |
| Depth limit explícito | ⚠️ **1 nível** (oficial) — não invocar dentro de subagent | ❓ não documentado | ❓ não documentado | ❓ não documentado |
| Statefulness da sessão | ⚠️ session-bounded | ✅ persiste em `~/.codex/sessions/` | ⚠️ session-bounded | ✅ persiste em `~/.copilot/session-state/` |
| Worktree isolation entre sessions paralelas | ❌ manual | ❌ manual | ❌ manual | ✅ automático |
| Confiança alta pra produção | ✅ máxima (referência) | ⚠️ média (isolamento parcial) | ✅ alta | ✅ alta |
| Custo por orquestração (relativo) | $$ | $$ | $$ | $$ |

### Recomendação por cenário

| Cenário | Tool recomendado | Motivo |
|---|---|---|
| PRD crítico em produção, primeira vez rodando a skill | **Claude Code** | Isolamento mais forte, doc oficial mais clara, validação empírica neste guia |
| Pipeline CI/CD com múltiplos PRDs simultâneos | **Copilot CLI** | Worktree automático isola sessions sem coordenação manual |
| Time grande com cota OpenAI | **Codex CLI** | Mas marque tudo `Paralelizável=Não` ou monitore drift de config |
| Validação contínua / iteração rápida | **Gemini CLI** | Slash command direto + paralelismo nativo bom para dev local |

---

## Apêndice A — Checklist pré-invocação

Use antes de cada `/execute-all-tasks`:

- [ ] `tasks/prd-<slug>/prd.md` existe e está aprovado
- [ ] `tasks/prd-<slug>/techspec.md` existe e está aprovado
- [ ] `tasks/prd-<slug>/tasks.md` tem tabela canônica com `Status`, `Dependências`, `Paralelizável`
- [ ] Cada `task-X.Y-*.md` existe e tem `Critérios de Sucesso`
- [ ] `ai-spec skills --verify` retorna 0
- [ ] `ai-spec check-spec-drift tasks/prd-<slug>/tasks.md` retorna 0
- [ ] Estou na raiz do repositório
- [ ] Sessão do tool é fresh (sem outros subagents ativos)
- [ ] Flag `Paralelizável` está honesto (regra DTPS aplicada)
- [ ] `GOVERNANCE_TELEMETRY=1` (opcional, recomendado)

## Apêndice B — Checklist pós-execução

Use após cada orquestração:

- [ ] `_orchestration_report.md` foi gerado
- [ ] Status final é `done` (todas tarefas concluídas) ou `partial` (com explicação clara)
- [ ] Cada tarefa executada tem `<id>_execution_report.md`
- [ ] `tasks.md` foi atualizado pelos subagents (não pelo usuário)
- [ ] Diffs SHA registrados nos relatórios fazem sentido com o commit atual
- [ ] Suíte de testes do projeto passa localmente
- [ ] Se houve halt: causa raiz identificada e documentada antes de retomar

---

## Apêndice C — Referências

- SKILL canônica: `.agents/skills/execute-all-tasks/SKILL.md`
- Template de relatório agregado: `.agents/skills/execute-all-tasks/assets/orchestration-report-template.md`
- Skill delegada (execução por tarefa): `.agents/skills/execute-task/SKILL.md`
- Governança transversal: `.claude/rules/governance.md`
- AGENTS.md — base contract de todas as skills
- Doc oficial Claude Code Sub-agents: https://code.claude.com/docs/en/sub-agents
- Doc oficial Codex Subagents: https://developers.openai.com/codex/subagents
- Doc oficial Gemini CLI Subagents: https://github.com/google-gemini/gemini-cli/blob/main/docs/core/subagents.md
- Doc oficial Copilot CLI Custom Agents: https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli
