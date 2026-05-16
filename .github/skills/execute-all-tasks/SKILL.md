---
name: execute-all-tasks
version: 1.5.1
depends_on: [execute-task, agent-governance]
description: Orquestra execução completa de PRD spawnando subagent fresh por tarefa para isolar contexto. Respeita DAG, paraleliza onde tool suporta nativamente, halt-first, retomada idempotente. Use para PRD inteiro; não use para uma tarefa única — use execute-task.
---

# Executar Todas as Tarefas de um PRD

## Visão Geral

Delega cada tarefa a **subagent fresh** (primitiva nativa do tool). Subagent carrega só o necessário, executa via `execute-task`, retorna YAML compacto, morre. Orquestrador retém ≤100 tokens/tarefa.

Por tarefa: lê → carrega só governance + linguagem do diff + skills declaradas → executa → YAML → contexto descartado → próxima.

Paralelismo só quando: `Paralelizável` em tasks.md E tool suporta spawn nativo.

## Procedimentos

**Etapa 1: Validar PRD**
0. **Invocar hook programático** (enforcement real das fragilidades F17, F18, F27, F29):
   `bash .claude/hooks/pre-execute-all-tasks.sh <slug>` (resolver caminho na ordem `.claude/hooks/` → `.agents/hooks/` → `.gemini/hooks/` → `.codex/hooks/` → `.github/hooks/`). Exit ≠ 0 → `failed` repassando stderr do hook. Hook valida regex de tasks.md, gaps numéricos, cross-PRD spec-hash e ciclos. Se ausente em todos os caminhos, prosseguir com validação textual (modo legado).
1. **`unset AI_PREFLIGHT_DONE` (F17 — também executado pelo hook acima como redundância)** antes de qualquer comando — força orquestrador a rodar próprios gates; re-exporta apenas no prompt do subagent.
2. Input: slug curto, `prd-<slug>`, ou path. Normalizar para `${AI_TASKS_ROOT:-tasks}/${AI_PRD_PREFIX:-prd-}<slug>/`.
3. **Resolver lib de profundidade (B1, fallback agnóstico)**: procurar `check-invocation-depth.sh` na ordem `.agents/lib/` → `scripts/lib/` e fazer `source`. Ausente nas duas → `failed: check-invocation-depth.sh ausente em .agents/lib/ e scripts/lib/ — vendor a lib ou rode 'ai-spec-harness install'`. Comando canônico:
   ```bash
   _depth_lib=""
   for d in .agents/lib scripts/lib; do
     [[ -r "$d/check-invocation-depth.sh" ]] && { _depth_lib="$d/check-invocation-depth.sh"; break; }
   done
   [[ -n "$_depth_lib" ]] || { echo "failed: depth lib missing"; exit 1; }
   source "$_depth_lib" || exit 1
   ```
4. **Gate de binário `ai-spec` (B2, sem degradação silenciosa)**: validar presença antes dos comandos do pré-voo. Ausente → `needs_input` com instrução de instalação:
   ```bash
   if ! command -v ai-spec >/dev/null 2>&1; then
     echo "needs_input: binário 'ai-spec' não encontrado no PATH. Instale via 'brew install ai-spec-harness' (ou 'go install github.com/ai-spec-harness/ai-spec-harness/cmd/ai_spec_harness@latest'). O orquestrador não pode degradar silenciosamente para 'modo legado' — princípio: governança acima de automação mágica."
     exit 1
   fi
   ```
5. `ai-spec skills --verify` — `blocked` se drift.
6. Confirmar `prd.md`, `techspec.md`, `tasks.md` → `needs_input` se faltar.
7. `ai-spec check-spec-drift tasks/prd-<slug>/tasks.md` → `blocked` se RF não coberto.
8. **Validação git opt-in (F35, ativar via `AI_VALIDATE_GIT_HISTORY=1`)**: para cada tarefa `done`, extrair `DiffSHA` do report e validar `git cat-file -e <sha>`. SHA ausente (revert/rewrite) → `needs_input: tarefa <id> done mas DiffSHA <sha> não está no git log — possível revert. (a) re-execute, (b) edite status, (c) cancele`.

**Etapa 2: Construir grafo**
1. Ler `tasks.md`. Parsear cada linha com regex canônicos (gerados por `create-tasks` v1.4+):
   - `status`: `^(pending|in_progress|needs_input|blocked|failed|done)$` → fora = `failed: malformed status on <id>`.
   - `dependências`: `^(—|(\w[\w-]*\/)?\d+\.\d+(,\s*(\w[\w-]*\/)?\d+\.\d+)*)$`. Cross-PRD via prefixo `<slug>/`. Resolução em 5 passos:
     1. Ler `tasks/prd-<slug>/tasks.md`. Ausente → `failed: cross-PRD target not found`.
     2. Tarefa inexistente naquele tasks.md → `failed: cross-PRD task not found: <slug>/<id>`.
     3. **Spec-hash do PRD referenciado (F18)**: extrair `spec-hash-prd` do header e comparar com `ai-spec hash tasks/prd-<slug>/prd.md`. Divergente → `blocked: cross-PRD <slug> tem spec drift; rode 'ai-spec check-spec-drift' e re-execute aquele PRD primeiro`.
     4. Validar status `done`.
     5. **Ciclo cross-PRD (F27)**: travessia recursiva limitada a 3 níveis verificando se algum elo aponta de volta para PRD ativo. Ciclo → `failed: cross-PRD circular dependency detected: <chain>`. Profundidade > 3 → `blocked: cross-PRD chain too deep (>3); refatorar`.
   - `paralelizável`: normalizar equivalentes seguros (`não`/`nao`/`NÃO` → `Não`, `com 2.0,3.0` → `Com 2.0, 3.0`, `-`/`none`/vazio → `—`) e então validar `^(—|Não|Com\s+\d+\.\d+(,\s*\d+\.\d+)*)$`. Valores ambíguos continuam `failed: malformed Paralelizável on <id>`.
2. Resolver `file_path` por convenção `task-<id>-*.md` ou `<id>_*.md`. Ambíguo → `needs_input`.
3. **Gaps de numeração (F29)**: extrair IDs e ordenar. Gap (ex.: 1.0, 3.0 sem 2.0) → warning + `needs_input` se não confirmado intencional.
4. Reportar snapshot inicial: total, contagem por estado, pendentes, done puladas.

**Etapa 3: Loop topológico**

Repetir até zerar `pending` ou disparar halt:
1. Re-ler `tasks.md`.
2. `ready = { t : t.status=="pending" E todas dep(t).status=="done" }`.
3. `ready == ∅` com `pending > 0` → `failed` reportando ciclo/dep órfã.
4. Compor wave: alguma `paralelizável=false` → só ela. Senão, todas `paralelizável=true` juntas.
5. Verificar suporte do tool a paralelismo nativo. Sem suporte → degradar sequencial.
6. Disparar Etapa 4. Coletar resultados. Qualquer `≠ done` após validação → Etapa 5 (halt).

**Etapa 4: Spawnar subagents**

**Soft timeout (result-discard, F14):** `AI_TASK_TIMEOUT_SECONDS` (default 1800s) configurável em `.claude/config.yaml`. Override por tarefa: `<!-- task-timeout-seconds: N -->` (regex `^task-timeout-seconds:\s*(\d+)\s*$`, sem unidades). **Limitação honesta**: tools (Claude/Gemini/Codex) não oferecem kill nativo; orquestrador apenas registra timeout, marca `failed: timeout after <budget>s`, descarta YAML tardio. **Subagent continua consumindo tokens em background** até completar naturalmente. Para hard kill, coordenação no nível do tool (matar sessão).

Prompt do subagent:
- Paths absolutos do task file, prd.md, techspec.md, tasks.md.
- "Invoque `execute-task`. Carrega APENAS necessário. Não saia do escopo."
- "`export AI_INVOCATION_DEPTH=0` + resolver `check-invocation-depth.sh` em cascata (`.agents/lib/` → `scripts/lib/`) e fazer `source`."
- "`export AI_PREFLIGHT_DONE=1` — orquestrador já validou; pule esses gates."
- Contrato de retorno (idêntico em todos os tools):
  ```yaml
  status: done | blocked | failed | needs_input
  report_path: tasks/prd-<slug>/<id>_execution_report.md
  summary: <1 linha>
  ```
  - **`report_path` DEVE ser relativo à raiz do repositório** (F13). Absoluto rejeitado; relativo ao subdir do subagent rejeitado. Validação resolve via `realpath --no-symlinks <repo_root>/<path>`.
  - Sem diffs, código ou logs.

**Cadeia de validação ao YAML retornado:**

**Fallback de YAML ausente (F25, crash entre execute-task Stage 5/6):**
- Sem retorno ou corrompido: verificar `tasks/prd-<slug>/.checkpoints/<id>.yaml` (escrito por `execute-task` Stage 5.3).
- Existe e parseável: usar como YAML válido (nota no relatório: "recuperado de checkpoint timestamp=<ts>"). Após consumir, `rm` para evitar reuso.
- Ausente: `failed: no return and no checkpoint`.

Cadeia (do retorno OU checkpoint) — pode ser executada por **hook programático** (enforcement real) ou inline:

**Hook recomendado**: `echo "$YAML" | bash .claude/hooks/post-execute-task.sh <slug> <task-id>` (busca em `.claude/hooks/` → `.agents/hooks/` → outros mirrors). Exit ≠ 0 = falha em F2/F13/F24/F25/F35; reclassificar tarefa para `failed` repassando stderr do hook. Se hook ausente, executar inline conforme abaixo.

1. **Formato canônico**: bloco com exatamente `status`, `report_path`, `summary`, sem campos extras, campos duplicados, comentários ou texto livre/diff → `failed: contract violation`.
2. **Status canônico**: ∈ `{done, blocked, failed, needs_input}`. Fora → `failed: invalid status`.
3. **Evidência física (F2+F13)** para `done`: normalizar `realpath --no-symlinks <repo_root>/<report_path>`, validar `[ -s "<resolved>" ]`. Ausente/vazio → `failed: missing evidence (resolved=<path>)`. Path absoluto rejeitado.
4. **Consistência tasks.md** para `done`: re-ler tasks.md, confirmar status atualizado para `done`. Divergente → `failed: status drift`.

**Etapa 5: Halt-first + relatório**
1. **Wait-all-then-halt (F3, contra race)**:
   - Spawnar todos da wave. Aguardar TODOS concluírem antes de decidir.
   - Aplicar cadeia de validação a cada retorno.
   - Só então decidir halt — subagents paralelos podem mutar tasks.md concorrentemente; halt prematuro deixa writes pendentes.
2. **File lock** em writes de tasks.md: subagents usam `flock -x`/rename atômico/partials (orientação do prompt → `execute-task` Stage 5.5).
3. **Checkpoint do orquestrador (F31) — invocar hook**:
   - Após cada wave concluída e validada: `bash .claude/hooks/post-wave.sh <slug> <wave-id> <results-yaml-file>` (busca nos mirrors padrão). Hook escreve `tasks/prd-<slug>/_orchestration_report.partial.md` append-only.
   - Próxima invocação detecta `.partial.md` na Etapa 1: lê, consolida com tasks.md atual, usa como ponto de partida.
   - Ao concluir todas as waves: rename atômico `.partial.md` → `_orchestration_report.md`.
   - Se ambos existem na Etapa 1: prefere `.partial.md` + warning para usuário decidir.
   - Hook ausente → escrever `.partial.md` inline com mesmo conteúdo (modo legado).
4. Renderizar `_orchestration_report.md` (template em `assets/`) com snapshot inicial vs final, tabela executadas, puladas, waves, próximos passos.
5. NÃO mutar tasks.md no orquestrador — só subagents via `execute-task`.

**Etapa 6: Encerrar**
Retornar status: `done` (todas done), `partial` (alguma não-done), `failed` (pré-voo abortou), `needs_input`.

## Mapeamento por Tool

Contrato de retorno idêntico; primitiva varia.

| Tool | Primitiva | Subagent | Paralelismo | Depth |
|---|---|---|---|---|
| Claude Code | `Agent` ([ref](https://code.claude.com/docs/en/sub-agents)) | `.claude/agents/task-executor.md` | múltiplas Agent calls/mensagem | **1** — review/bugfix são skill calls |
| Codex CLI | `.codex/agents/*.toml` ([ref](https://developers.openai.com/codex/subagents)) | `.codex/agents/task-executor.toml` | concorrentes | assumir 1 |
| Gemini CLI | `.gemini/agents/*.md` ([ref](https://github.com/google-gemini/gemini-cli/blob/main/docs/core/subagents.md)) | `.gemini/agents/task-executor.md` | dispatch paralelo | n/d |
| Copilot CLI | Custom Agents ([ref](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli)). Auto-descobre `.github/skills/` (criado por `ai-spec install` espelhando `.agents/skills/`); demais mirrors opcionais | `.github/agents/task-executor.agent.md` | `/fleet` ou multi-session | n/d |

Degradação controlada: tool sem subagent nativo → sequencial sem isolamento; registrar no report.

**Validação empírica obrigatória dos agent files (F26)** antes do primeiro uso real:

| Tool | Comando | Esperado |
|---|---|---|
| Claude | `claude /agents` ou `ls .claude/agents/` | `task-executor` aparece |
| Codex | `codex agent list` | `task-executor` (formato `.codex/agents/task-executor.toml` é inferido) |
| Gemini | `gemini agents list` | `task-executor` (formato `.gemini/agents/task-executor.md` é inferido) |
| Copilot | `gh copilot /skills reload` + `ls .github/agents/` | `task-executor.agent.md` |

Para Codex/Gemini, formatos foram inferidos de docs 2026, não validados empiricamente. Se CLI não reconhecer, spawn falha silenciosamente, orquestrador degrada para inline. **Sintoma**: janela do orquestrador acumula contexto após primeira tarefa em vez de ≤100 tokens. Se notar, revalidar formato.

## Regras invioláveis

1. Toda tarefa em subagent fresh — orquestrador nunca executa `execute-task` inline.
2. Contrato YAML estrito; violação = `failed: contract violation`.
3. Paralelismo só com flag em tasks.md E suporte nativo do tool.
4. Não coordenar arquivos entre paralelos — confiar no `Paralelizável`.
5. Orquestrador inline apenas: parsing tasks.md, DAG, report final, pré-voo, checkpoint.

## Tratamento de Erros

* **DAG inválido**: `failed` com cadeia. Sem reparo automático.
* **Contrato violado**: `failed: contract violation`, halt, relatório, encerrar.
* **Subagent não-done**: respeitar. Não re-executar.
* **tasks.md mutado externamente**: `needs_input`.
* **Profundidade**: orquestrador top-level (depth 0); cada subagent reinicia `AI_INVOCATION_DEPTH=0`.

## Resolução de paths

`tasks/prd-<slug>/` resolve para `${AI_TASKS_ROOT:-tasks}/${AI_PRD_PREFIX:-prd-}<slug>/`. Configurar em `.claude/config.yaml`/`.agents/config.yaml` (`tasks_root`, `prd_prefix`, `task_timeout_seconds`). Vars exportadas por `check-invocation-depth.sh`, resolvido em cascata `.agents/lib/` → `scripts/lib/` (vendor canônico em `.agents/lib/`, mirror legado em `scripts/lib/`).

## Contrato resumido

| Campo | Valor |
|-------|-------|
| Input | slug ou path |
| Pré-condições | prd/techspec/tasks presentes; lockfile íntegro; RF coverage OK |
| Saída por tarefa | YAML `{status, report_path, summary}` validado em 4 passos + fallback checkpoint |
| Saída agregada | `tasks/prd-<slug>/_orchestration_report.md` (com `.partial.md` durante execução) |
| Status final | `done \| partial \| failed \| needs_input` |
| Mutação direta tasks.md | Não |
| Re-execução automática | Não |
| Paralelismo | Mapping por Tool + flag `Paralelizável` |
| Timeout default | 1800s soft (não kill) |
