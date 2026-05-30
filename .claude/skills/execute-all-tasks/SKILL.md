---
name: execute-all-tasks
version: 1.8.0
category: governance
depends_on: [execute-task, agent-governance]
description: Orquestra execução completa de PRD spawnando subagent fresh por tarefa para isolar contexto. Respeita DAG, paraleliza onde tool suporta nativamente, halt-first, retomada idempotente. Use para PRD inteiro; não use para uma tarefa única — use execute-task.
---

# Executar Todas as Tarefas de um PRD

## Visão Geral

Delega cada tarefa a subagent. Subagent carrega só o necessário, executa via `execute-task` e retorna YAML compacto. Orquestrador retém ≤100 tokens/tarefa.

Por tarefa: lê → carrega só governance + linguagem do diff + skills declaradas → executa → YAML → contexto descartado → próxima.

Paralelismo só quando: `Paralelizável` em tasks.md E tool suporta spawn nativo.

## Procedimentos

**Etapa 1: Validar PRD**
0. **Invocar hook programático** (enforcement real das fragilidades F17, F18, F27, F29):
   `bash .claude/hooks/pre-execute-all-tasks.sh <slug>` (resolver caminho na cascata portátil `.agents/hooks/` → `.claude/hooks/` → `.gemini/hooks/` → `.codex/hooks/` → `.github/hooks/`). Exit ≠ 0 → `failed` repassando stderr do hook. Hook valida regex de tasks.md, gaps numéricos, cross-PRD spec-hash e ciclos. **Ausente em todos os caminhos (sem degradação silenciosa)**: como `.agents/hooks/` é instalado por `ai-spec install` em qualquer tool capaz, a ausência total indica integridade quebrada → `failed: hook de governança 'pre-execute-all-tasks.sh' ausente em todos os caminhos — reinstale via 'ai-spec install'`. Não prosseguir em "modo legado".
1. **`unset AI_PREFLIGHT_DONE` (F17 — também executado pelo hook acima como redundância)** antes de qualquer comando — força orquestrador a rodar próprios gates; re-exporta apenas no prompt do subagent.
2. Input: slug curto, `prd-<slug>`, ou path. Normalizar para `${AI_TASKS_ROOT:-.specs}/${AI_PRD_PREFIX:-prd-}<slug>/`.
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
7. `ai-spec check-spec-drift .specs/prd-<slug>/tasks.md` → `blocked` se RF não coberto.
8. **Validação git opt-in (F35, ativar via `AI_VALIDATE_GIT_HISTORY=1`)**: para cada tarefa `done`, extrair `DiffSHA` do report e validar `git cat-file -e <sha>`. SHA ausente (revert/rewrite) → `needs_input: tarefa <id> done mas DiffSHA <sha> não está no git log — possível revert. (a) re-execute, (b) edite status, (c) cancele`.

**Etapa 2: Construir grafo**
1. Ler `tasks.md`. Parsear cada linha com regex canônicos (gerados por `create-tasks` v1.4+):
   - `status`: `^(pending|in_progress|needs_input|blocked|failed|done)$` → fora = `failed: malformed status on <id>`.
   - `dependências`: `^(—|(\w[\w-]*\/)?\d+\.\d+(,\s*(\w[\w-]*\/)?\d+\.\d+)*)$`. Cross-PRD via prefixo `<slug>/`. Resolução em 5 passos:
     1. Ler `.specs/prd-<slug>/tasks.md`. Ausente → `failed: cross-PRD target not found`.
     2. Tarefa inexistente naquele tasks.md → `failed: cross-PRD task not found: <slug>/<id>`.
     3. **Spec-hash do PRD referenciado (F18)**: extrair `spec-hash-prd` do header e comparar com `ai-spec hash .specs/prd-<slug>/prd.md`. Divergente → `blocked: cross-PRD <slug> tem spec drift; rode 'ai-spec check-spec-drift' e re-execute aquele PRD primeiro`.
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

**Timeout + orçamento de tokens (F14, RF-21):** `AI_TASK_TIMEOUT_SECONDS` (default 1800s) e
`AI_TASK_TOKEN_BUDGET` (default 0 = sem limite; zero-value preserva comportamento F1) configuráveis
em `.claude/config.yaml`/`.agents/config.yaml`. Override de timeout por tarefa:
`<!-- task-timeout-seconds: N -->` (regex `^task-timeout-seconds:\s*(\d+)\s*$`, sem unidades).
Quando o subagent reportar uso de tokens acumulado acima de `AI_TASK_TOKEN_BUDGET`, marcar
`failed: token budget <budget> exceeded` e não relançar.

**Kill no timeout — depende da primitiva de spawn do tool (verificado 2026):**

| Tool | Primitiva de spawn | Kill nativo no timeout? | Ação ao estourar |
|---|---|---|---|
| Codex CLI | `codex exec` (subprocesso OS) | **Sim** — `kill <pid>` do processo `codex exec` | hard kill do PID; `failed: timeout after <budget>s (killed)` |
| Gemini CLI | `gemini --acp` (subprocesso OS) | **Sim** — encerrar a sessão ACP/subprocesso | hard kill do subprocesso ACP; `failed: timeout after <budget>s (killed)` |
| Copilot CLI | custom agent / `/fleet` (sessão/subprocesso) | **Parcial** — matar a sessão quando spawnada como subprocesso | kill da sessão quando possível; senão soft-discard |
| Claude Code | `Agent` (in-process, mesma sessão) | **Não** — sem kill nativo | soft timeout: registra, marca `failed: timeout after <budget>s`, descarta YAML tardio |

**Limitação honesta (Claude)**: a primitiva `Agent` roda in-process; o orquestrador apenas
descarta o YAML tardio — o subagent continua consumindo tokens até completar naturalmente. Para os
tools que spawnam subprocesso OS (Codex `codex exec`, Gemini `gemini --acp`, Copilot fleet), o
orquestrador **mata o processo** (`kill <pid>`) ao estourar o timeout/orçamento, interrompendo o
consumo. Registrar no `_orchestration_report.md` se o kill foi efetivo (`killed`) ou apenas
soft-discard (`discarded`).

Prompt do subagent:
- Paths absolutos do task file, prd.md, techspec.md, tasks.md.
- "Invoque `execute-task`. Carrega APENAS necessário. Não saia do escopo."
- "`export AI_INVOCATION_DEPTH=0` + resolver `check-invocation-depth.sh` em cascata (`.agents/lib/` → `scripts/lib/`) e fazer `source`."
- "`export AI_PREFLIGHT_DONE=1` — orquestrador já validou; pule esses gates."
- Contrato de retorno (idêntico em todos os tools):
  ```yaml
  status: done | blocked | failed | needs_input
  report_path: .specs/prd-<slug>/<id>_execution_report.md
  summary: <1 linha>
  ```
  - **`report_path` DEVE ser relativo à raiz do repositório** (F13). Absoluto rejeitado; relativo ao subdir do subagent rejeitado. Validação resolve via `realpath --no-symlinks <repo_root>/<path>`.
  - Sem diffs, código ou logs.

**Cadeia de validação ao YAML retornado:**

**Fallback de YAML ausente (F25, crash entre execute-task Stage 5/6):**
- Sem retorno ou corrompido: verificar `.specs/prd-<slug>/.checkpoints/<id>.yaml` (escrito por `execute-task` Stage 5.3).
- Existe e parseável: usar como YAML válido (nota no relatório: "recuperado de checkpoint timestamp=<ts>"). Após consumir, `rm` para evitar reuso.
- Ausente: `failed: no return and no checkpoint`.

Cadeia (do retorno OU checkpoint) — pode ser executada por **hook programático** (enforcement real) ou inline:

**Hook recomendado**: `echo "$YAML" | bash .claude/hooks/post-execute-task.sh <slug> <task-id>` (cascata portátil `.agents/hooks/` → `.claude/hooks/` → outros mirrors). Exit ≠ 0 = falha em F2/F13/F24/F25/F35; reclassificar tarefa para `failed` repassando stderr do hook. **Sem degradação silenciosa**: ausente em todos os caminhos → `failed: hook 'post-execute-task.sh' ausente — reinstale via 'ai-spec install'`; a cadeia de validação inline abaixo é o contrato que o hook impõe, nunca um substituto silencioso.

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
   - Após cada wave concluída e validada: `bash .claude/hooks/post-wave.sh <slug> <wave-id> <results-yaml-file>` (busca nos mirrors padrão). Hook escreve `.specs/prd-<slug>/_orchestration_report.partial.md` append-only.
   - Próxima invocação detecta `.partial.md` na Etapa 1: lê, consolida com tasks.md atual, usa como ponto de partida.
   - Ao concluir todas as waves: rename atômico `.partial.md` → `_orchestration_report.md`.
   - Se ambos existem na Etapa 1: prefere `.partial.md` + warning para usuário decidir.
   - Hook ausente em todos os caminhos → `failed: hook 'post-wave.sh' ausente — reinstale via 'ai-spec install'` (sem modo legado silencioso; `.agents/hooks/` é instalado por padrão).
4. Renderizar `_orchestration_report.md` (template em `assets/`) com snapshot inicial vs final, tabela executadas, puladas, waves, próximos passos.
5. NÃO mutar tasks.md no orquestrador — só subagents via `execute-task`.

**Etapa 6: Encerrar**
Retornar status: `done` (todas done), `partial` (alguma não-done), `failed` (pré-voo abortou), `needs_input`.

## Mapeamento por Tool

Contrato de retorno idêntico; primitiva varia.

| Tool | Primitiva | Subagent | Paralelismo | Depth |
|---|---|---|---|---|
| Claude Code | `Agent` ([ref](https://code.claude.com/docs/en/sub-agents)) | `.claude/agents/task-executor.md` (nativo) | múltiplas Agent calls/mensagem | **1** — review/bugfix são skill calls |
| Codex CLI | `codex exec` (subprocesso não-interativo) + `--profile` + MCP nested ([ref](https://developers.openai.com/codex)) | sem diretório de agents nativo — isolamento via `codex exec` | concorrentes (subprocessos) | assumir 1 |
| Gemini CLI | `gemini skills` + extensões + ACP (`--acp`) nested ([ref](https://github.com/google-gemini/gemini-cli)) | sem diretório de agents nativo — isolamento via `gemini --acp`/skills | dispatch paralelo via subprocessos | n/d |
| Copilot CLI | Custom Agents ([ref](https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/create-custom-agents-for-cli)). Auto-descobre `.github/skills/` (criado por `ai-spec install` espelhando `.agents/skills/`); demais mirrors opcionais | `.github/agents/task-executor.agent.md` (nativo) | `/fleet` ou multi-session | n/d |

Degradação controlada: tool sem subagent nativo → sequencial sem isolamento; registrar no report.

**Inventário e validação empírica de subagentes (F26) — verificado em 2026:**

| Tool | Diretório de agents nativo? | Mecanismo de isolamento verificado | Comando de verificação |
|---|---|---|---|
| Claude Code | **Sim** (`.claude/agents/`) | subagentes dedicados | `ls .claude/agents/` (8 agents) |
| Copilot CLI | **Sim** (`.github/agents/`) | custom agents | `ls .github/agents/` (8 agents) |
| Codex CLI 0.135 | **Não** | `codex exec` (não-interativo) + `--profile` + `codex mcp` nested | `codex --help` (sem subcomando `agents`; há `exec`, `mcp`, `--profile`) |
| Gemini CLI 0.44 | **Não** | `gemini skills` + `gemini extensions` + ACP (`gemini --acp`) | `gemini skills list`; `gemini --help` (sem subcomando `agents`) |

**Classificação epistêmica (registrar no report)**: Claude/Copilot = **verificado** (diretório
nativo + agents enumeráveis). Codex/Gemini = **verificado que NÃO há diretório `agents` nativo** —
o isolamento usa `codex exec`/`gemini --acp` como subprocesso. O arquivo de conveniência
`.codex/agents/task-executor.toml` e `.gemini/agents/task-executor.md` é um **prompt seed para o
subprocesso**, não um registro enforçado pelo CLI. **Não fingir enforcement nativo** para Codex/Gemini:
se o orquestrador não conseguir spawnar subprocesso isolado, degradar para execução inline sequencial
e registrar explicitamente `subagente: inline (tool sem agents nativo)` no `_orchestration_report.md`.
**Sintoma de fallback**: a janela do orquestrador acumula contexto após a primeira tarefa em vez de
≤100 tokens — sinaliza que o isolamento por subprocesso não ativou.

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

`.specs/prd-<slug>/` resolve para `${AI_TASKS_ROOT:-.specs}/${AI_PRD_PREFIX:-prd-}<slug>/`. Configurar em `.claude/config.yaml`/`.agents/config.yaml` (`tasks_root`, `prd_prefix`, `task_timeout_seconds`). Vars exportadas por `check-invocation-depth.sh`, resolvido em cascata `.agents/lib/` → `scripts/lib/` (vendor canônico em `.agents/lib/`, mirror legado em `scripts/lib/`).

## Contrato resumido

| Campo | Valor |
|-------|-------|
| Input | slug ou path |
| Pré-condições | prd/techspec/tasks presentes; lockfile íntegro; RF coverage OK |
| Saída por tarefa | YAML `{status, report_path, summary}` validado em 4 passos + fallback checkpoint |
| Saída agregada | `.specs/prd-<slug>/_orchestration_report.md` (com `.partial.md` durante execução) |
| Status final | `done \| partial \| failed \| needs_input` |
| Mutação direta tasks.md | Não |
| Re-execução automática | Não |
| Paralelismo | Mapping por Tool + flag `Paralelizável` |
| Timeout default | 1800s; kill nativo p/ Codex/Gemini/Copilot (subprocesso), soft-discard p/ Claude (in-process) |
| Orçamento de tokens | `AI_TASK_TOKEN_BUDGET` (default 0 = ilimitado; zero-value preserva F1) |
