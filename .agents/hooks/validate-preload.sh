#!/usr/bin/env bash

set -euo pipefail

readonly PRELOAD_BLOCK_EXIT=2

preload_mode="${GOVERNANCE_PRELOAD_MODE:-fail}"
preload_confirmed="${GOVERNANCE_PRELOAD_CONFIRMED:-0}"

hook_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="${AGENTS_ROOT:-$(cd "$hook_dir/../.." && pwd)}"

parse_lib=""
for candidate in \
  "$project_root/.agents/lib/parse-hook-input.sh" \
  "$project_root/scripts/lib/parse-hook-input.sh" \
  "$hook_dir/../lib/parse-hook-input.sh" \
  "$hook_dir/../../scripts/lib/parse-hook-input.sh"; do
  if [[ -f "$candidate" ]]; then
    parse_lib="$candidate"
    break
  fi
done

if [[ -z "$parse_lib" ]]; then
  echo "ERRO: parse-hook-input.sh ausente em .agents/lib/ e scripts/lib/ — rode 'ai-spec-harness install .'" >&2
  exit "$PRELOAD_BLOCK_EXIT"
fi

source "$parse_lib"

payload=""
if [[ ! -t 0 ]]; then
  payload="$(cat)"
fi

file_path="$(printf '%s' "$payload" | parse_file_path)"
command_text="$(printf '%s' "$payload" | parse_command_text)"

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

case "$file_path" in
  *.go | *.py | *.ts | *.js | *.tsx | *.jsx | *.cs | "") ;;
  *) exit 0 ;;
esac

if [[ -n "$file_path" ]]; then
  echo "LEMBRETE: antes de editar codigo, confirme que AGENTS.md e agent-governance/SKILL.md foram lidos nesta sessao." >&2
fi

if [[ "$preload_confirmed" == "1" ]]; then
  exit 0
fi

if [[ "$preload_mode" == "warn" ]]; then
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
