#!/usr/bin/env bash
set -uo pipefail

if [[ ! -t 0 ]]; then
  cat >/dev/null
fi

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

tasks_root="${AI_TASKS_ROOT:-.specs}"
prd_prefix="${AI_PRD_PREFIX:-prd-}"

blocked=0

shopt -s nullglob
for tasks_file in "$REPO_ROOT/$tasks_root/${prd_prefix}"*/tasks.md; do
  [[ -f "$tasks_file" ]] || continue
  prd_dir="$(dirname "$tasks_file")"

  while IFS= read -r task_id; do
    [[ -z "$task_id" ]] && continue
    report="$prd_dir/${task_id}_execution_report.md"
    if [[ ! -f "$report" ]]; then
      echo "[session-end] tarefa ativa sem relatorio de execucao: $task_id ($tasks_file)" >&2
      blocked=1
      continue
    fi
    if ! grep -Eiq "(veredito|verdict)[[:space:]]*:[[:space:]]*(APPROVED|APPROVED_WITH_REMARKS)" "$report"; then
      echo "[session-end] tarefa ativa sem veredito APPROVED registrado: $task_id ($report)" >&2
      blocked=1
    fi
  done < <(awk -F'|' '
    {
      id = $2
      status = $4
      gsub(/^[ \t]+|[ \t]+$/, "", id)
      gsub(/^[ \t]+|[ \t]+$/, "", status)
    }
    id ~ /^[0-9]+\.[0-9]+$/ && status == "in_progress" { print id }
  ' "$tasks_file")
done
shopt -u nullglob

if [[ "$blocked" -ne 0 ]]; then
  echo "[session-end] GATE DE ENCERRAMENTO BLOQUEADO — existe tarefa ativa sem veredito APPROVED registrado." >&2
  exit 1
fi

exit 0
