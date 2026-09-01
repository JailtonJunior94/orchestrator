#!/usr/bin/env bash
# subagent-stop-wrapper.sh
# Wrapper invocado pelo Claude Code SubagentStop hook quando um subagent
# task-executor termina. O output final do subagent continua sendo o envelope
# YAML minimo exigido pelos adaptadores; o contrato operacional e' o resultado
# SDD v2 persistido no checkpoint JSON da tentativa. O wrapper extrai a
# identidade do envelope e encaminha somente o JSON versionado ao hook estrito.
#
# Convencao Claude Code:
#   - stdin: JSON com {"hook_event_name": "...", "subagent_output": "..."}
#   - exit 0: nao bloqueia
#   - exit 2 (com stderr): bloqueia subsequente operacao
#
# Filtragem: roda APENAS quando o subagent type = task-executor
# (matching feito em settings.local.json; este wrapper assume task-executor).
#
# Comportamento defensivo: erros internos do wrapper bloqueiam por padrao.
# Para modo legado nao-bloqueante, exporte STRICT_HOOK_FAILURES=0.

set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
HOOKS_DIR=""
for d in "$REPO_ROOT/.claude/hooks" "$REPO_ROOT/.agents/hooks" "$REPO_ROOT/.gemini/hooks" "$REPO_ROOT/.codex/hooks" "$REPO_ROOT/.github/hooks"; do
  if [[ -d "$d" ]]; then
    HOOKS_DIR="$d"
    break
  fi
done
[[ -n "$HOOKS_DIR" ]] || exit 0

POST_EXECUTE_HOOK="$HOOKS_DIR/post-execute-task.sh"
if [[ ! -x "$POST_EXECUTE_HOOK" && ! -f "$POST_EXECUTE_HOOK" ]]; then
  # Hook nao instalado — silenciosamente ignorar
  exit 0
fi

# Ler input do Claude Code (JSON via stdin)
input=$(cat)
[[ -z "$input" ]] && exit 0

# Extrair subagent_output (assumindo Claude Code JSON shape)
# Defensivo: tenta jq, fallback para grep+sed
yaml_output=""
if command -v jq >/dev/null 2>&1; then
  yaml_output=$(echo "$input" | jq -r '.subagent_output // .output // empty' 2>/dev/null)
fi
if [[ -z "$yaml_output" ]]; then
  # Fallback: assumir que o input INTEIRO eh o YAML (modo standalone)
  yaml_output="$input"
fi

# Extrair report_path do YAML
report_path=$(echo "$yaml_output" | grep -E "^report_path:[[:space:]]" | head -1 | sed 's/^report_path:[[:space:]]*//' | tr -d '"' | tr -d "'" | xargs)
[[ -z "$report_path" ]] && exit 0  # Sem report_path no YAML, nao eh task-executor

# Extrair prd-slug e task-id do report_path: <tasks-root>/<prd-prefix><slug>/<id>_execution_report.md
report_dir=$(dirname "$report_path")
report_dir_base=$(basename "$report_dir")
prd_prefix="${AI_PRD_PREFIX:-prd-}"
prd_slug=""
case "$report_dir_base" in
  "$prd_prefix"*) prd_slug="${report_dir_base#"$prd_prefix"}" ;;
esac
task_id=$(echo "$report_path" | sed -nE 's|.*/([0-9]+\.[0-9]+)_execution_report\.md$|\1|p')

if [[ -z "$prd_slug" || -z "$task_id" ]]; then
  # Path nao bate com convencao do execute-task — pode ser outro tipo de subagent
  exit 0
fi

# O YAML e' uma interface humana/adapter e nao pode ser interpretado como
# execution-result em modo estrito. A mesma identidade aponta para o checkpoint
# JSON versionado, escrito antes de tasks.md ser mutado pelo execute-task. O
# caminho vem da configuracao do workspace, nunca do report_path controlado pelo
# subagent; assim um envelope malicioso nao consegue direcionar o hook para JSON
# fora do repositorio.
tasks_root="${AI_TASKS_ROOT:-.specs}"
if [[ "$tasks_root" == /* || "$tasks_root" == *".."* ]]; then
  echo "[subagent-stop] AI_TASKS_ROOT invalido para resultado SDD v2: $tasks_root" >&2
  exit 2
fi
result_json="$REPO_ROOT/$tasks_root/${prd_prefix}${prd_slug}/.checkpoints/${task_id}.json"
if [[ ! -s "$result_json" ]]; then
  echo "[subagent-stop] resultado SDD v2 ausente: $result_json" >&2
  if [[ "${STRICT_HOOK_FAILURES:-1}" != "0" ]]; then
    exit 2
  fi
  exit 0
fi

stderr_tmp=$(mktemp /tmp/subagent-stop-err.XXXXXX)
trap "rm -f $stderr_tmp" EXIT

bash "$POST_EXECUTE_HOOK" "$prd_slug" "$task_id" "$result_json" 2>"$stderr_tmp"
hook_exit=$?

if [[ "$hook_exit" -ne 0 ]]; then
  if [[ "${STRICT_HOOK_FAILURES:-1}" != "0" ]]; then
    # Modo estrito: propaga falha como exit 2 (bloqueia operacao no Claude Code)
    cat "$stderr_tmp" >&2
    echo "[subagent-stop] HOOK FAILURE — bloqueando operacao (STRICT_HOOK_FAILURES!=0)" >&2
    exit 2
  fi
  # Default: emite stderr mas nao bloqueia
  cat "$stderr_tmp" >&2
  echo "[subagent-stop] Aviso: post-execute-task FAIL (exit=$hook_exit). Modo legado nao-bloqueante por STRICT_HOOK_FAILURES=0." >&2
fi

exit 0
