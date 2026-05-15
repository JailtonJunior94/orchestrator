#!/usr/bin/env bash
# subagent-stop-wrapper.sh
# Wrapper invocado pelo Claude Code SubagentStop hook quando um subagent
# task-executor termina. Parseia o output do subagent (YAML), extrai
# prd-slug + task-id, e invoca post-execute-task.sh para validacao.
#
# Convencao Claude Code:
#   - stdin: JSON com {"hook_event_name": "...", "subagent_output": "..."}
#   - exit 0: nao bloqueia
#   - exit 2 (com stderr): bloqueia subsequente operacao
#
# Filtragem: roda APENAS quando o subagent type = task-executor
# (matching feito em settings.local.json; este wrapper assume task-executor).
#
# Comportamento defensivo: erros internos do wrapper NAO bloqueiam o subagent
# por padrao (exit 0). Override via STRICT_HOOK_FAILURES=1 propaga FAIL.

set -uo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
HOOKS_DIR="$REPO_ROOT/.claude/hooks"
[[ -d "$HOOKS_DIR" ]] || HOOKS_DIR="$REPO_ROOT/.agents/hooks"

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

# Extrair prd-slug e task-id do report_path: tasks/prd-<slug>/<id>_execution_report.md
prd_slug=$(echo "$report_path" | sed -nE 's|^[^/]*/prd-([^/]+)/.+$|\1|p')
task_id=$(echo "$report_path" | sed -nE 's|.*/([0-9]+\.[0-9]+)_execution_report\.md$|\1|p')

if [[ -z "$prd_slug" || -z "$task_id" ]]; then
  # Path nao bate com convencao do execute-task — pode ser outro tipo de subagent
  exit 0
fi

# Invocar hook de validacao
yaml_tmp=$(mktemp /tmp/subagent-stop.yaml.XXXXXX)
echo "$yaml_output" > "$yaml_tmp"
trap "rm -f $yaml_tmp" EXIT

stderr_tmp=$(mktemp /tmp/subagent-stop-err.XXXXXX)
trap "rm -f $stderr_tmp $yaml_tmp" EXIT

bash "$POST_EXECUTE_HOOK" "$prd_slug" "$task_id" "$yaml_tmp" 2>"$stderr_tmp"
hook_exit=$?

if [[ "$hook_exit" -ne 0 ]]; then
  if [[ "${STRICT_HOOK_FAILURES:-0}" == "1" ]]; then
    # Modo estrito: propaga falha como exit 2 (bloqueia operacao no Claude Code)
    cat "$stderr_tmp" >&2
    echo "[subagent-stop] HOOK FAILURE — bloqueando operacao (STRICT_HOOK_FAILURES=1)" >&2
    exit 2
  fi
  # Default: emite stderr mas nao bloqueia
  cat "$stderr_tmp" >&2
  echo "[subagent-stop] Aviso: post-execute-task FAIL (exit=$hook_exit). Para bloquear, exporte STRICT_HOOK_FAILURES=1." >&2
fi

exit 0
