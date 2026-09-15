#!/usr/bin/env bash
set -uo pipefail

readonly SESSION_END_BLOCK_EXIT=2

decision_output="${AISPEC_HOOK_DECISION_OUTPUT:-exit}"

hook_input=""
if [[ ! -t 0 ]]; then
  hook_input="$(cat)"
fi

stop_hook_active=0
if [[ -n "$hook_input" ]] && printf '%s' "$hook_input" \
  | grep -Eq '"stop_hook_active"[[:space:]]*:[[:space:]]*true'; then
  stop_hook_active=1
fi

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

tasks_root="${AI_TASKS_ROOT:-.specs}"
prd_prefix="${AI_PRD_PREFIX:-prd-}"

finding_anchor='^[[:space:]]*([-*+>][[:space:]]*)*(\*\*|__)?'
blocking_severity_re="${finding_anchor}(\[(critical|cr(i|í)tico|high|hard|alta|alto|blocker|security)\]|severidade[[:space:]]*:[[:space:]]*(critical|high|cr(i|í)tico|alta|alto)|severity[[:space:]]*:[[:space:]]*(critical|high))"
any_severity_re="${finding_anchor}(\[(critical|cr(i|í)tico|high|hard|alta|alto|blocker|security|medium|m(e|é)dia|important|importante|low|baixa|suggestion|sugest(a|ã)o)\]|severidade[[:space:]]*:[[:space:]]*(critical|high|medium|low|cr(i|í)tico|alta|alto|m(e|é)dia|baixa)|severity[[:space:]]*:[[:space:]]*(critical|high|medium|low))"

declares_remarks() {
  local report_file="$1"
  grep -Eq '^verdict=[[:space:]]*APPROVED_WITH_REMARKS[[:space:]]*$' "$report_file" && return 0
  grep -Eiq '^(veredito|verdict)[[:space:]]*:[[:space:]]*APPROVED_WITH_REMARKS[[:space:]]*$' "$report_file" && return 0
  grep -Eiq 'veredito do revisor[[:space:]]*:[[:space:]]*APPROVED_WITH_REMARKS([[:space:]]|$)' "$report_file" && return 0
  return 1
}

findings_body() {
  awk '
    /^[[:space:]]*```/ { fenced = !fenced; next }
    !fenced { print }
  ' "$1"
}

closes_with_remarks() {
  local report_file="$1"
  declares_remarks "$report_file" || return 1
  local body
  body="$(findings_body "$report_file")"
  printf '%s\n' "$body" | grep -Eiq "$blocking_severity_re" && return 1
  printf '%s\n' "$body" | grep -Eiq "$any_severity_re" || return 1
  return 0
}

structured_review_verdict() {
  local result_file="$1"
  [[ -f "$result_file" ]] || return 1
  local value
  value="$(grep -Eo '"review_verdict"[[:space:]]*:[[:space:]]*"[A-Za-z_]+"' "$result_file" \
    | head -n 1 \
    | sed -E 's/.*:[[:space:]]*"([A-Za-z_]+)".*/\1/')"
  [[ -n "$value" ]] || return 1
  printf '%s' "$value"
  return 0
}

has_approved_verdict() {
  local report_file="$1"
  grep -Eq '^verdict=[[:space:]]*APPROVED[[:space:]]*$' "$report_file" && return 0
  grep -Eiq '^(veredito|verdict)[[:space:]]*:[[:space:]]*APPROVED[[:space:]]*$' "$report_file" && return 0
  grep -Eiq 'veredito do revisor[[:space:]]*:[[:space:]]*APPROVED([[:space:]]|$)' "$report_file" && return 0
  closes_with_remarks "$report_file" && return 0
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
    report="$prd_dir/${task_id}_execution_report.md"
    case "$task_status" in
      done)
        label="tarefa fechada como done"
        ;;
      blocked)
        [[ -f "$report" ]] || continue
        label="tarefa blocked com relatorio de execucao escrito"
        ;;
      *)
        label="tarefa ativa"
        ;;
    esac
    if [[ ! -f "$report" ]]; then
      echo "[session-end] $label sem relatorio de execucao: $task_id ($tasks_file)" >&2
      reasons+=("$label sem relatorio de execucao: $task_id ($tasks_file)")
      blocked=1
      continue
    fi
    result_json="$prd_dir/${task_id}_execution_result.json"
    review_verdict=""
    review_verdict="$(structured_review_verdict "$result_json" || true)"
    case "$review_verdict" in
      approved)
        continue
        ;;
      changes_requested|needs_input)
        echo "[session-end] $label com decisao estruturada review_verdict=$review_verdict, que nao encerra o ciclo (dado estruturado prevalece sobre o texto do relatorio): $task_id ($result_json)" >&2
        reasons+=("$label com decisao estruturada review_verdict=$review_verdict, que nao encerra o ciclo (dado estruturado prevalece sobre o texto do relatorio): $task_id ($result_json)")
        blocked=1
        continue
        ;;
    esac
    if ! has_approved_verdict "$report"; then
      echo "[session-end] $label sem veredito que encerre o ciclo (APPROVED, ou APPROVED_WITH_REMARKS sem achado high/critical): $task_id ($report)" >&2
      reasons+=("$label sem veredito que encerre o ciclo (APPROVED, ou APPROVED_WITH_REMARKS sem achado high/critical): $task_id ($report)")
      blocked=1
    fi
  done < <(awk -F'|' '
    {
      id = $2
      status = $4
      gsub(/^[ \t]+|[ \t]+$/, "", id)
      gsub(/^[ \t]+|[ \t]+$/, "", status)
    }
    id ~ /^[0-9]+\.[0-9]+$/ && (status == "in_progress" || status == "done" || status == "blocked") { print id "\t" status }
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
  echo "[session-end] GATE DE ENCERRAMENTO BLOQUEADO — existe tarefa sem veredito que encerre o ciclo (APPROVED, ou APPROVED_WITH_REMARKS sem achado high/critical)." >&2
  if [[ "$stop_hook_active" -eq 1 ]]; then
    echo "[session-end] stop_hook_active=true — o bloqueio ja foi aplicado nesta retomada; liberando o encerramento para o agente reagir em vez de prender a sessao." >&2
    if [[ "$decision_output" == "json" ]]; then
      printf '{}\n'
    fi
    exit 0
  fi
  if [[ "$decision_output" == "json" ]]; then
    detail="GATE DE ENCERRAMENTO BLOQUEADO — existe tarefa sem veredito que encerre o ciclo (APPROVED, ou APPROVED_WITH_REMARKS sem achado high/critical)."
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
