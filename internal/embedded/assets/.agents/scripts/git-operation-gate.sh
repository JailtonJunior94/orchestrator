#!/usr/bin/env bash

set -euo pipefail

readonly GIT_OPERATION_BLOCK_EXIT=2
readonly GIT_OPERATION_GATED_SUBCOMMANDS="git commit e git push"
readonly GIT_OPERATION_BLOCKED_MESSAGE="GOVERNANCE BLOQUEIO: operacao git nao solicitada ($GIT_OPERATION_GATED_SUBCOMMANDS) negada pelo gate canonico de operacao git."
readonly DESTRUCTIVE_OPERATION_BLOCKED_MESSAGE="GOVERNANCE BLOQUEIO: comando destrutivo detectado (remocao recursiva e forcada) negado independentemente do estado de preload."

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="${AGENTS_ROOT:-$(cd "$script_dir/../.." && pwd)}"

parse_lib=""
for candidate in \
  "$project_root/.agents/lib/hook-payload.sh" \
  "$project_root/scripts/lib/hook-payload.sh" \
  "$script_dir/../lib/hook-payload.sh" \
  "$script_dir/../../scripts/lib/hook-payload.sh"; do
  if [[ -f "$candidate" ]]; then
    parse_lib="$candidate"
    break
  fi
done

if [[ -z "$parse_lib" ]]; then
  echo "ERRO: hook-payload.sh ausente em .agents/lib/ e scripts/lib/ — rode 'ai-spec-harness install .'" >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

source "$parse_lib"

payload=""
if [[ ! -t 0 ]]; then
  payload="$(cat)"
fi

if ! command_text="$(printf '%s' "$payload" | parse_command_text)"; then
  echo "GOVERNANCE BLOQUEIO: payload de hook invalido; operacao git negada." >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

if [[ -z "$command_text" ]]; then
  echo "GOVERNANCE BLOQUEIO: comando ausente ou nao extraivel; operacao git negada." >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

audit_git_operation_escape() {
  local reason="$1"
  local log="${GOVERNANCE_ESCAPE_LOG:-$project_root/.aispec/governance-escapes.log}"
  local stamp
  stamp="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)"
  echo "AUDITORIA: escape de operacao git registrado ($reason) para comando: $command_text" >&2
  mkdir -p "$(dirname "$log")" 2>/dev/null || return 0
  printf '%s\t%s\t%s\t%s\t%s\n' \
    "$stamp" "$reason" "$command_text" "${AI_TOOL:-desconhecido}" "${USER:-desconhecido}" \
    >> "$log" 2>/dev/null || true
}

audit_destructive_operation_escape() {
  local reason="$1"
  local log="${GOVERNANCE_ESCAPE_LOG:-$project_root/.aispec/governance-escapes.log}"
  local stamp
  stamp="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)"
  echo "AUDITORIA: escape de operacao destrutiva registrado ($reason) para comando: $command_text" >&2
  mkdir -p "$(dirname "$log")" 2>/dev/null || return 0
  printf '%s\t%s\t%s\t%s\t%s\n' \
    "$stamp" "$reason" "$command_text" "${AI_TOOL:-desconhecido}" "${USER:-desconhecido}" \
    >> "$log" 2>/dev/null || true
}

is_unsolicited_git_write_segment() {
  local segment="$1"
  printf '%s' "$segment" \
    | grep -Eq '(^|[[:space:]])git([[:space:]]+-[^[:space:]]+)*[[:space:]]+(commit|push)([[:space:]]|$)'
}

is_destructive_removal_segment() {
  local segment="$1"
  if ! printf '%s' "$segment" | grep -Eq '(^|[[:space:]/])rm([[:space:]]|$)'; then
    return 1
  fi
  if printf '%s' "$segment" \
    | grep -Eq -- '(^|[[:space:]])-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*([[:space:]]|$)|(^|[[:space:]])-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*([[:space:]]|$)'; then
    return 0
  fi
  if printf '%s' "$segment" | grep -Eq '(^|[[:space:]])(-r|-R|--recursive)([[:space:]]|$)' \
    && printf '%s' "$segment" | grep -Eq '(^|[[:space:]])(-f|--force)([[:space:]]|$)'; then
    return 0
  fi
  return 1
}

git_write_detected=0
destructive_detected=0

while IFS= read -r segment || [[ -n "$segment" ]]; do
  [[ -n "$segment" ]] || continue
  if is_unsolicited_git_write_segment "$segment"; then
    git_write_detected=1
  fi
  if is_destructive_removal_segment "$segment"; then
    destructive_detected=1
  fi
done < <(printf '%s' "$command_text" | tr ';|&' '\n')

destructive_confirmed="${GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED:-0}"
if [[ "$destructive_detected" -eq 1 ]]; then
  if [[ "$destructive_confirmed" == "1" ]]; then
    audit_destructive_operation_escape "GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED=1"
  else
    echo "$DESTRUCTIVE_OPERATION_BLOCKED_MESSAGE" >&2
    echo "Comando: $command_text" >&2
    echo "Para prosseguir com pedido explicito do usuario: export GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED=1" >&2
    exit "$GIT_OPERATION_BLOCK_EXIT"
  fi
fi

git_operation_confirmed="${GOVERNANCE_GIT_OPERATION_CONFIRMED:-0}"
git_operation_mode="${GOVERNANCE_GIT_OPERATION_MODE:-fail}"

if [[ "$git_write_detected" -eq 1 ]]; then
  if [[ "$git_operation_confirmed" == "1" ]]; then
    audit_git_operation_escape "GOVERNANCE_GIT_OPERATION_CONFIRMED=1"
    exit 0
  fi
  if [[ "$git_operation_mode" == "warn" ]]; then
    audit_git_operation_escape "GOVERNANCE_GIT_OPERATION_MODE=warn"
    echo "GOVERNANCE_GIT_OPERATION_MODE=warn: prosseguindo sem bloqueio (opt-out explicito)." >&2
    exit 0
  fi
  echo "$GIT_OPERATION_BLOCKED_MESSAGE" >&2
  echo "Comando: $command_text" >&2
  echo "Para prosseguir com pedido explicito do usuario: export GOVERNANCE_GIT_OPERATION_CONFIRMED=1" >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

exit 0
