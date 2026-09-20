#!/usr/bin/env bash

set -euo pipefail

readonly PRELOAD_BLOCK_EXIT=2

preload_mode="${GOVERNANCE_PRELOAD_MODE:-fail}"
preload_confirmed="${GOVERNANCE_PRELOAD_CONFIRMED:-0}"

hook_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="${AGENTS_ROOT:-$(cd "$hook_dir/../.." && pwd)}"

parse_lib=""
for candidate in \
  "$project_root/.agents/lib/hook-payload.sh" \
  "$project_root/scripts/lib/hook-payload.sh" \
  "$hook_dir/../lib/hook-payload.sh" \
  "$hook_dir/../../scripts/lib/hook-payload.sh"; do
  if [[ -f "$candidate" ]]; then
    parse_lib="$candidate"
    break
  fi
done

if [[ -z "$parse_lib" ]]; then
  echo "ERRO: hook-payload.sh ausente em .agents/lib/ e scripts/lib/ — rode 'ai-spec-harness install .'" >&2
  exit "$PRELOAD_BLOCK_EXIT"
fi

source "$parse_lib"

payload=""
if [[ ! -t 0 ]]; then
  payload="$(cat)"
fi

if ! file_path="$(printf '%s' "$payload" | parse_file_path)"; then
  echo "ERRO: payload de hook invalido; governanca negada." >&2
  exit "$PRELOAD_BLOCK_EXIT"
fi
if ! command_text="$(printf '%s' "$payload" | parse_command_text)"; then
  echo "ERRO: payload de hook invalido; governanca negada." >&2
  exit "$PRELOAD_BLOCK_EXIT"
fi

gate_targets=()
if [[ -n "$file_path" ]]; then
  gate_targets+=("$file_path")
elif [[ -n "$command_text" ]]; then
  while IFS= read -r target; do
    [[ -n "$target" ]] && gate_targets+=("$target")
  done < <(extract_command_source_targets "$command_text")
fi

if [[ ${#gate_targets[@]} -gt 0 ]]; then
  prereq_gate="$project_root/.agents/scripts/hook-prereq-gate.sh"
  if [[ ! -f "$prereq_gate" ]]; then
    prereq_gate="$hook_dir/../scripts/hook-prereq-gate.sh"
  fi
  if [[ -f "$prereq_gate" ]]; then
    if ! printf '%s' "$payload" | AGENTS_ROOT="$project_root" bash "$prereq_gate" "${gate_targets[@]}"; then
      exit "$PRELOAD_BLOCK_EXIT"
    fi
  fi
fi

if [[ -n "$command_text" ]]; then
  git_operation_gate="$project_root/.agents/scripts/git-operation-gate.sh"
  if [[ ! -f "$git_operation_gate" ]]; then
    git_operation_gate="$hook_dir/../scripts/git-operation-gate.sh"
  fi
  if [[ -f "$git_operation_gate" ]]; then
    if ! printf '%s' "$payload" | AGENTS_ROOT="$project_root" bash "$git_operation_gate"; then
      exit "$PRELOAD_BLOCK_EXIT"
    fi
  fi
fi

case "$file_path" in
  *.go | *.py | *.ts | *.js | *.tsx | *.jsx | *.cs | *.mjs | *.cjs | *.mts | *.cts \
  | *.sh | *.bash | *.zsh | *.rb | *.java | *.kt | *.kts | *.sql | *.rs | *.swift \
  | *.php | *.c | *.h | *.cc | *.cpp | *.hpp | *.scala | *.ex | *.exs | *.pl | *.lua | "") ;;
  *) exit 0 ;;
esac

if [[ -n "$file_path" ]]; then
  echo "LEMBRETE: antes de editar codigo, confirme que AGENTS.md e agent-governance/SKILL.md foram lidos nesta sessao." >&2
fi

audit_escape() {
  local reason="$1"
  local target="${file_path:-<sem-alvo>}"
  local log="${GOVERNANCE_ESCAPE_LOG:-$project_root/.aispec/governance-escapes.log}"
  local stamp
  stamp="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)"
  echo "AUDITORIA: escape de governanca registrado ($reason) para $target" >&2
  mkdir -p "$(dirname "$log")" 2>/dev/null || return 0
  printf '%s\t%s\t%s\t%s\t%s\n' \
    "$stamp" "$reason" "$target" "${AI_TOOL:-desconhecido}" "${USER:-desconhecido}" \
    >> "$log" 2>/dev/null || true
}

if [[ "$preload_confirmed" == "1" ]]; then
  audit_escape "GOVERNANCE_PRELOAD_CONFIRMED=1"
  exit 0
fi

if [[ "$preload_mode" == "warn" ]]; then
  audit_escape "GOVERNANCE_PRELOAD_MODE=warn"
  echo "GOVERNANCE_PRELOAD_MODE=warn: prosseguindo sem bloqueio (opt-out explicito)." >&2
  exit 0
fi

if [[ -n "$file_path" ]]; then
  echo "ERRO: governanca nao carregada para edicao de codigo ($file_path)." >&2
else
  echo "ERRO: governanca nao carregada e nenhum alvo extraivel do payload — ausencia de alvo e negacao, nunca aprovacao." >&2
fi
echo "Para prosseguir: export GOVERNANCE_PRELOAD_CONFIRMED=1" >&2
exit "$PRELOAD_BLOCK_EXIT"
