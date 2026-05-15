#!/usr/bin/env bash
# sync-hooks.sh
# Sincroniza hooks do orquestrador (post-execute-task, pre-execute-all-tasks, post-wave)
# do diretorio canonico (.claude/hooks/) para os mirrors:
#   - .agents/hooks/, .gemini/hooks/, .codex/hooks/, .github/hooks/
#   - internal/embedded/assets/{.claude,.agents,.gemini,.codex,.github}/hooks/
#
# Estrategia: rsync com --delete dos hooks orquestrador apenas (preserva outros hooks
# como validate-governance, validate-preload).
#
# Uso: ./scripts/sync-hooks.sh

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.claude/hooks"

if [[ ! -d "$canonical" ]]; then
  echo "ERRO: diretório canônico não encontrado: $canonical" >&2
  exit 1
fi

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

count_synced=0
count_missing_canonical=0

for hook in "${ORCHESTRATOR_HOOKS[@]}"; do
  src="$canonical/$hook"
  if [[ ! -f "$src" ]]; then
    echo "WARN: hook canonico ausente: $src" >&2
    count_missing_canonical=$((count_missing_canonical + 1))
    continue
  fi

  # Garantir +x no canonico
  chmod +x "$src"

  for mirror in "${mirror_dirs[@]}"; do
    mkdir -p "$mirror"
    cp "$src" "$mirror/$hook"
    chmod +x "$mirror/$hook"
  done
  count_synced=$((count_synced + 1))
  echo "synced: $hook -> ${#mirror_dirs[@]} mirrors"
done

echo
echo "sync-hooks: $count_synced hook(s) sincronizado(s) para ${#mirror_dirs[@]} mirror(s)"
if [[ "$count_missing_canonical" -gt 0 ]]; then
  echo "AVISO: $count_missing_canonical hook(s) canonico(s) ausente(s)" >&2
  exit 1
fi

exit 0
