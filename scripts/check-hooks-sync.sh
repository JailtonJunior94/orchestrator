#!/usr/bin/env bash
# check-hooks-sync.sh
# Detecta drift entre o diretorio canonico (.claude/hooks/) dos hooks do orquestrador
# e os mirrors (.agents/hooks/, .codex/hooks/, .github/hooks/,
# internal/embedded/assets/{.claude,.agents,.codex,.github}/hooks/).
#
# Uso: ./scripts/check-hooks-sync.sh
# Exit 0 = sincronizado; exit 1 = drift detectado.
# Para corrigir: ./scripts/sync-hooks.sh

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.claude/hooks"

ORCHESTRATOR_HOOKS=(
  "post-execute-task.sh"
  "pre-execute-all-tasks.sh"
  "post-wave.sh"
  "subagent-stop-wrapper.sh"
)

declare -a mirror_dirs=(
  "$repo_root/.agents/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.agents/hooks"
  "$repo_root/internal/embedded/assets/.codex/hooks"
  "$repo_root/internal/embedded/assets/.github/hooks"
)

drift_count=0
ok_count=0

for hook in "${ORCHESTRATOR_HOOKS[@]}"; do
  canon_path="$canonical/$hook"
  if [[ ! -f "$canon_path" ]]; then
    echo "MISSING: $canon_path (canonico ausente)"
    drift_count=$((drift_count + 1))
    continue
  fi

  for mirror in "${mirror_dirs[@]}"; do
    mirror_path="$mirror/$hook"
    if [[ ! -f "$mirror_path" ]]; then
      echo "MISSING: $mirror_path"
      drift_count=$((drift_count + 1))
      continue
    fi
    if ! diff -q "$canon_path" "$mirror_path" >/dev/null 2>&1; then
      echo "DRIFT: $hook diverge entre $canonical e $mirror"
      drift_count=$((drift_count + 1))
    else
      ok_count=$((ok_count + 1))
    fi
  done
done

echo
echo "Hooks em sync: $ok_count"
echo "Drift / missing: $drift_count"

session_end_canonical="$repo_root/.agents/scripts/validate-session-end.sh"
declare -a session_end_gate_dirs=(
  "$repo_root/.claude/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.codex/hooks"
  "$repo_root/internal/embedded/assets/.github/hooks"
  "$repo_root/internal/embedded/assets/.agents/scripts"
)
session_end_drift=0
if [[ -f "$session_end_canonical" ]]; then
  for mirror in "${session_end_gate_dirs[@]}"; do
    mirror_path="$mirror/validate-session-end.sh"
    if [[ ! -f "$mirror_path" ]]; then
      echo "MISSING: $mirror_path (gate de encerramento)"
      session_end_drift=$((session_end_drift + 1))
      continue
    fi
    if ! diff -q "$session_end_canonical" "$mirror_path" >/dev/null 2>&1; then
      echo "DRIFT: validate-session-end.sh diverge entre .agents/scripts e $mirror"
      session_end_drift=$((session_end_drift + 1))
    fi
  done
  echo "Gate de encerramento em sync: $((${#session_end_gate_dirs[@]} - session_end_drift))/${#session_end_gate_dirs[@]} mirrors"
fi

opencode_plugin_drift=0
opencode_plugin_embedded="$repo_root/internal/embedded/assets/.opencode/plugin/governance.js"
if [[ ! -f "$opencode_plugin_embedded" ]]; then
  echo "MISSING: $opencode_plugin_embedded (plugin do OpenCode)"
  opencode_plugin_drift=1
elif [[ -f "$repo_root/.opencode/plugin/governance.js" ]] && ! diff -q "$repo_root/.opencode/plugin/governance.js" "$opencode_plugin_embedded" >/dev/null 2>&1; then
  echo "DRIFT: governance.js diverge entre .opencode/plugin e internal/embedded/assets/.opencode/plugin"
  opencode_plugin_drift=1
fi

if [[ "$drift_count" -gt 0 || "$session_end_drift" -gt 0 || "$opencode_plugin_drift" -gt 0 ]]; then
  echo
  echo "Para corrigir: ./scripts/sync-hooks.sh"
  exit 1
fi

echo "Todos os hooks orquestrador sincronizados."
exit 0
