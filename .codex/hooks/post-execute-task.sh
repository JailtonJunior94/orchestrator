#!/usr/bin/env bash
# post-execute-task.sh
# Validacao programatica pos-execute-task.
# Fecha as fragilidades F2 (evidence physical), F13 (path absoluto),
# F24 (escalation de remark critico), F25 (checkpoint), F35 (git revert).
#
# Uso (modo CLI direto, invocavel pelo orquestrador via Bash tool):
#   bash .claude/hooks/post-execute-task.sh <prd-slug> <task-id> <yaml-file>
#
# Uso (modo stdin para pipelines):
#   echo "$YAML" | bash .claude/hooks/post-execute-task.sh <prd-slug> <task-id>
#
# Exit codes:
#   0 — todas validacoes passaram (warnings nao bloqueiam)
#   1 — pelo menos uma validacao falhou; mensagens em stderr
#   2 — argumentos invalidos
#
# Ativacao opcional de validacoes mais caras:
#   AI_VALIDATE_GIT_HISTORY=1 — habilita F35 (git cat-file no DiffSHA)

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Uso: $0 <prd-slug> <task-id> [yaml-file]" >&2
  exit 2
fi

PRD_SLUG="$1"
TASK_ID="$2"

# Modo arquivo ou stdin
if [[ $# -ge 3 ]]; then
  YAML_FILE="$3"
else
  YAML_FILE=$(mktemp /tmp/post-execute-task.yaml.XXXXXX)
  trap "rm -f $YAML_FILE" EXIT
  cat > "$YAML_FILE"
fi

if [[ ! -s "$YAML_FILE" ]]; then
  echo "FAIL: YAML vazio ou inexistente: $YAML_FILE" >&2
  exit 1
fi

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
TASKS_ROOT="${AI_TASKS_ROOT:-tasks}"
PRD_PREFIX="${AI_PRD_PREFIX:-prd-}"
PRD_DIR="$REPO_ROOT/$TASKS_ROOT/$PRD_PREFIX$PRD_SLUG"

errors=0
warnings=0

# === Parse YAML estritamente ===
status=$(grep -E "^status:[[:space:]]" "$YAML_FILE" | head -1 | sed 's/^status:[[:space:]]*//' | tr -d '"' | tr -d "'" | xargs)
report_path=$(grep -E "^report_path:[[:space:]]" "$YAML_FILE" | head -1 | sed 's/^report_path:[[:space:]]*//' | tr -d '"' | tr -d "'" | xargs)

# Validar status canonico
if ! [[ "$status" =~ ^(done|blocked|failed|needs_input)$ ]]; then
  echo "FAIL: status invalido ou ausente: '$status'" >&2
  exit 1
fi

# === F2 + F13: evidence physical + path relativo ===
if [[ -z "$report_path" ]]; then
  echo "FAIL F2: report_path ausente no YAML" >&2
  errors=$((errors+1))
elif [[ "$report_path" =~ ^/ ]]; then
  echo "FAIL F13: report_path absoluto rejeitado: $report_path (deve ser relativo a raiz do repo)" >&2
  errors=$((errors+1))
else
  resolved="$REPO_ROOT/$report_path"
  if [[ ! -s "$resolved" ]]; then
    echo "FAIL F2: missing evidence — $resolved ausente ou vazio" >&2
    errors=$((errors+1))
  fi
fi

# === F24: escalation de remark critico se status=done ===
if [[ "$status" == "done" && -n "$report_path" && -s "$REPO_ROOT/$report_path" ]]; then
  if grep -iE "\[(critical|security|blocker|high)\]" "$REPO_ROOT/$report_path" >/dev/null 2>&1; then
    echo "FAIL F24: report contem remark critico em tarefa marcada done — escalar para BLOCKED:" >&2
    grep -inE "\[(critical|security|blocker|high)\]" "$REPO_ROOT/$report_path" | head -3 | sed 's/^/  /' >&2
    errors=$((errors+1))
  fi
fi

# === F25: checkpoint deve existir se status=done ===
# Default: FAIL bloqueante. Override via AI_ALLOW_MISSING_CHECKPOINT=1 (back compat com execute-task <v1.4)
if [[ "$status" == "done" ]]; then
  checkpoint="$PRD_DIR/.checkpoints/${TASK_ID}.yaml"
  if [[ ! -s "$checkpoint" ]]; then
    if [[ "${AI_ALLOW_MISSING_CHECKPOINT:-0}" == "1" ]]; then
      echo "WARN F25: checkpoint ausente em $checkpoint (back compat: AI_ALLOW_MISSING_CHECKPOINT=1)" >&2
      warnings=$((warnings+1))
    else
      echo "FAIL F25: checkpoint ausente em $checkpoint — execute-task Stage 5.3 nao escreveu YAML antes de mutar tasks.md. Re-execute a tarefa ou exporte AI_ALLOW_MISSING_CHECKPOINT=1 (nao recomendado)." >&2
      errors=$((errors+1))
    fi
  fi
fi

# === F35: validar DiffSHA contra git log (opt-in) ===
if [[ "${AI_VALIDATE_GIT_HISTORY:-0}" == "1" && "$status" == "done" && -n "$report_path" && -s "$REPO_ROOT/$report_path" ]]; then
  diff_sha=$(grep -E "^sha=" "$REPO_ROOT/$report_path" | head -1 | sed 's/^sha=//' | xargs)
  if [[ -n "$diff_sha" ]]; then
    if ! git -C "$REPO_ROOT" cat-file -e "$diff_sha" 2>/dev/null; then
      echo "FAIL F35: DiffSHA $diff_sha do report nao esta no git log atual (revert/branch deletado/history rewrite?)" >&2
      errors=$((errors+1))
    fi
  fi
fi

# === Resumo ===
if [[ "$errors" -gt 0 ]]; then
  echo "post-execute-task: $errors erro(s), $warnings warning(s) para $PRD_SLUG/$TASK_ID" >&2
  exit 1
fi

if [[ "$warnings" -gt 0 ]]; then
  echo "post-execute-task: OK com $warnings warning(s) para $PRD_SLUG/$TASK_ID" >&2
else
  echo "post-execute-task: OK ($PRD_SLUG/$TASK_ID, status=$status)" >&2
fi
exit 0
