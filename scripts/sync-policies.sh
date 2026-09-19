#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.agents/policies"
mirror="$repo_root/.claude/rules"

declare -a required_policies=(
  "governance.md"
  "code-style.md"
)

if [[ ! -d "$canonical" ]]; then
  echo "ERRO: diretorio canonico nao encontrado: $canonical" >&2
  exit 1
fi

mkdir -p "$mirror"

for policy in "${required_policies[@]}"; do
  src="$canonical/$policy"
  if [[ ! -f "$src" ]]; then
    echo "ERRO: policy declarada ausente no canonico: $src" >&2
    exit 1
  fi
  chmod u+w "$src" 2>/dev/null || true
  dest="$mirror/$policy"
  [[ -f "$dest" ]] && chmod u+w "$dest" 2>/dev/null || true
  cp -p "$src" "$dest"
  echo "synced: $policy -> $mirror"
done

echo "sync-policies: concluido"
