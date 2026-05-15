#!/usr/bin/env bash
# check-hooks-sync.sh
# Detecta drift entre o diretorio canonico (.claude/hooks/) dos hooks do orquestrador
# e os mirrors (.agents/hooks/, .gemini/hooks/, .codex/hooks/, .github/hooks/,
# internal/embedded/assets/{.claude,.agents,.gemini,.codex,.github}/hooks/).
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
  "$repo_root/.gemini/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.agents/hooks"
  "$repo_root/internal/embedded/assets/.gemini/hooks"
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

if [[ "$drift_count" -gt 0 ]]; then
  echo
  echo "Para corrigir: ./scripts/sync-hooks.sh"
  exit 1
fi

echo "Todos os hooks orquestrador sincronizados."
exit 0
