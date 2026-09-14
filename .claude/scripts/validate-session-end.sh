#!/usr/bin/env bash
set -uo pipefail

readonly SESSION_END_BLOCK_EXIT=2

decision_output="${AISPEC_HOOK_DECISION_OUTPUT:-exit}"

if [[ ! -t 0 ]]; then
  cat >/dev/null
fi

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

tasks_root="${AI_TASKS_ROOT:-.specs}"
prd_prefix="${AI_PRD_PREFIX:-prd-}"

has_approved_verdict() {
  local report_file="$1"
  grep -Eq '^verdict=[[:space:]]*APPROVED[[:space:]]*$' "$report_file" && return 0
  grep -Eiq '^(veredito|verdict)[[:space:]]*:[[:space:]]*APPROVED[[:space:]]*$' "$report_file" && return 0
  grep -Eiq 'veredito do revisor[[:space:]]*:[[:space:]]*APPROVED([[:space:]]|$)' "$report_file" && return 0
  return 1
}

blocked=0
reasons=()

shopt -s nullglob
for tasks_file in "$REPO_ROOT/$tasks_root/${prd_prefix}"*/tasks.md; do
  [[ -f "$tasks_file" ]] || continue
  prd_dir="$(dirname "$tasks_file")"

  while IFS=$'\t' read -r task_id task_status; do
    [[ -z "$task_id" ]] && continue
    if [[ "$task_status" == "done" ]]; then
      label="tarefa fechada como done"
    else
      label="tarefa ativa"
    fi
    report="$prd_dir/${task_id}_execution_report.md"
    if [[ ! -f "$report" ]]; then
      echo "[session-end] $label sem relatorio de execucao: $task_id ($tasks_file)" >&2
      reasons+=("$label sem relatorio de execucao: $task_id ($tasks_file)")
      blocked=1
      continue
    fi
    if ! has_approved_verdict "$report"; then
      echo "[session-end] $label sem veredito APPROVED registrado: $task_id ($report)" >&2
      reasons+=("$label sem veredito APPROVED registrado: $task_id ($report)")
      blocked=1
    fi
  done < <(awk -F'|' '
    {
      id = $2
      status = $4
      gsub(/^[ \t]+|[ \t]+$/, "", id)
      gsub(/^[ \t]+|[ \t]+$/, "", status)
    }
    id ~ /^[0-9]+\.[0-9]+$/ && (status == "in_progress" || status == "done") { print id "\t" status }
  ' "$tasks_file")
done
shopt -u nullglob

json_escape() {
  local raw="$1"
  raw="${raw//\\/\\\\}"
  raw="${raw//\"/\\\"}"
  printf '%s' "$raw"
}

if [[ "$blocked" -ne 0 ]]; then
  echo "[session-end] GATE DE ENCERRAMENTO BLOQUEADO — existe tarefa sem veredito APPROVED registrado." >&2
  if [[ "$decision_output" == "json" ]]; then
    detail="GATE DE ENCERRAMENTO BLOQUEADO — existe tarefa sem veredito APPROVED registrado."
    for reason in "${reasons[@]}"; do
      detail+=" | $reason"
    done
    printf '{"decision":"block","reason":"%s"}\n' "$(json_escape "$detail")"
  fi
  exit "$SESSION_END_BLOCK_EXIT"
fi

if [[ "$decision_output" == "json" ]]; then
  printf '{}\n'
fi

exit 0
