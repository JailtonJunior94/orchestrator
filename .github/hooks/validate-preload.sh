#!/usr/bin/env bash
# Hook de pre-execucao para GitHub Copilot CLI: bloqueia execucao sem preload de
# governanca E sem skill de linguagem instalada (gate de descoberta cirurgical).
#
# Uso: registrar como preToolUse hook em .github/hooks/governance.json.
#
# Lifecycle:
#   1. Le file_path do JSON stdin (best-effort, formato comum PreToolUse).
#   2. Invoca .agents/scripts/hook-prereq-gate.sh — bloqueia se skill ausente,
#      emite em stderr a lista cirurgica de references a carregar.
#   3. Mantem check legado de GOVERNANCE_PRELOAD_CONFIRMED.
#
# Unlock: exportar GOVERNANCE_PRELOAD_CONFIRMED=1 na sessao atual.
# Consulte AGENTS.md para instrucoes completas de preload.

set -euo pipefail

HOOK_DIR="$(cd "$(dirname "$0")" && pwd)"
# Respeita AGENTS_ROOT externo (E2E ou install em outro CWD); default = ancestor do hook.
PROJECT_ROOT="${AGENTS_ROOT:-$(cd "$HOOK_DIR/../.." && pwd)}"

_stdin=""
file_path=""
if [[ ! -t 0 ]]; then
  _stdin="$(cat)"
  file_path="$(printf '%s' "$_stdin" | awk 'match($0,/"file_path"[[:space:]]*:[[:space:]]*"[^"]*"/){
    s=substr($0,RSTART,RLENGTH); sub(/.*"file_path"[[:space:]]*:[[:space:]]*"/, "", s); sub(/".*/, "", s); print s; exit
  }')"
fi

if [[ -n "$file_path" ]]; then
  PREREQ_GATE="$PROJECT_ROOT/.agents/scripts/hook-prereq-gate.sh"
  if [[ ! -f "$PREREQ_GATE" ]]; then
    PREREQ_GATE="$(cd "$HOOK_DIR/../.." && pwd)/.agents/scripts/hook-prereq-gate.sh"
  fi
  if [[ -f "$PREREQ_GATE" ]]; then
    if ! printf '%s' "$_stdin" | AGENTS_ROOT="$PROJECT_ROOT" bash "$PREREQ_GATE" "$file_path"; then
      exit 1
    fi
  fi
fi

if [[ "${GOVERNANCE_PRELOAD_CONFIRMED:-}" != "1" ]]; then
  case "$file_path" in
    *.go|*.py|*.ts|*.js|*.tsx|*.jsx|*.cs|"")
      if [[ -z "$file_path" ]]; then
        echo "ERRO: governanca nao carregada." >&2
        echo "Execute: export GOVERNANCE_PRELOAD_CONFIRMED=1" >&2
        echo "Consulte AGENTS.md para instrucoes completas." >&2
        exit 1
      fi
      echo "ERRO: governanca nao carregada para edicao de codigo ($file_path)." >&2
      echo "Execute: export GOVERNANCE_PRELOAD_CONFIRMED=1" >&2
      exit 1
      ;;
  esac
fi

exit 0
