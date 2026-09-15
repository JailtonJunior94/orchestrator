#!/usr/bin/env bash
# sync-hooks.sh
# Sincroniza hooks do orquestrador (post-execute-task, pre-execute-all-tasks, post-wave)
# do diretorio canonico (.claude/hooks/) para os mirrors:
#   - .agents/hooks/, .codex/hooks/, .github/hooks/
#   - internal/embedded/assets/{.claude,.agents,.codex,.github}/hooks/
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
  "validate-governance.sh"
)

SESSION_END_HOOK="validate-session-end.sh"

declare -a mirror_dirs=(
  "$repo_root/.agents/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.agents/hooks"
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

session_source="$repo_root/.agents/scripts/$SESSION_END_HOOK"
if [[ ! -f "$session_source" ]]; then
  echo "WARN: gate canônico ausente: $session_source" >&2
  count_missing_canonical=$((count_missing_canonical + 1))
else
	cp "$session_source" "$repo_root/.claude/hooks/$SESSION_END_HOOK"
	chmod +x "$repo_root/.claude/hooks/$SESSION_END_HOOK"
  for mirror in "${mirror_dirs[@]}"; do
    mkdir -p "$mirror"
    cp "$session_source" "$mirror/$SESSION_END_HOOK"
    chmod +x "$mirror/$SESSION_END_HOOK"
  done
  echo "synced: $SESSION_END_HOOK -> ${#mirror_dirs[@]} mirrors"
fi

declare -a TOOL_VALIDATION_HOOKS=(
  "validate-preload.sh"
  "validate-governance.sh"
  "validate-session-end.sh"
)
declare -a tool_hook_roots=(
  ".claude"
  ".codex"
  ".github"
)
count_tool_validation=0
for root in "${tool_hook_roots[@]}"; do
  for hook in "${TOOL_VALIDATION_HOOKS[@]}"; do
    src="$repo_root/$root/hooks/$hook"
    if [[ ! -f "$src" ]]; then
      echo "WARN: validador canonico ausente: $src" >&2
      count_missing_canonical=$((count_missing_canonical + 1))
      continue
    fi
    chmod +x "$src"
    dst_dir="$repo_root/internal/embedded/assets/$root/hooks"
    mkdir -p "$dst_dir"
    cp "$src" "$dst_dir/$hook"
    chmod +x "$dst_dir/$hook"
    count_tool_validation=$((count_tool_validation + 1))
  done
done

declare -a AGENTS_VALIDATION_HOOKS=(
  "validate-preload.sh"
  "validate-governance.sh"
  "validate-session-end.sh"
)
for hook in "${AGENTS_VALIDATION_HOOKS[@]}"; do
  src="$repo_root/.agents/hooks/$hook"
  if [[ ! -f "$src" ]]; then
    echo "WARN: validador tool-neutro ausente: $src" >&2
    count_missing_canonical=$((count_missing_canonical + 1))
    continue
  fi
  chmod +x "$src"
  dst_dir="$repo_root/internal/embedded/assets/.agents/hooks"
  mkdir -p "$dst_dir"
  cp "$src" "$dst_dir/$hook"
  chmod +x "$dst_dir/$hook"
  count_tool_validation=$((count_tool_validation + 1))
done
echo "synced: $count_tool_validation validador(es) por ferramenta -> assets embarcados"

opencode_plugin_embedded="$repo_root/internal/embedded/assets/.opencode/plugin/governance.js"
if [[ -f "$opencode_plugin_embedded" ]]; then
  mkdir -p "$repo_root/.opencode/plugin"
  cp "$opencode_plugin_embedded" "$repo_root/.opencode/plugin/governance.js"
  echo "synced: plugin do OpenCode -> .opencode/plugin/governance.js"
else
  echo "WARN: plugin do OpenCode ausente: $opencode_plugin_embedded" >&2
  count_missing_canonical=$((count_missing_canonical + 1))
fi

echo
echo "sync-hooks: $count_synced hook(s) sincronizado(s) para ${#mirror_dirs[@]} mirror(s)"
if [[ "$count_missing_canonical" -gt 0 ]]; then
  echo "AVISO: $count_missing_canonical hook(s) canonico(s) ausente(s)" >&2
  exit 1
fi

exit 0
