#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.agents/policies"
mirror="$repo_root/.claude/rules"

declare -a required_policies=(
  "governance.md"
  "code-style.md"
)

drift_count=0
ok_count=0

if [[ ! -d "$canonical" ]]; then
  echo "DRIFT: diretorio canonico ausente: $canonical"
  drift_count=$((drift_count + 1))
fi

for policy in "${required_policies[@]}"; do
  canon_path="$canonical/$policy"
  mirror_path="$mirror/$policy"

  if [[ ! -f "$canon_path" ]]; then
    echo "DRIFT: policy canonica declarada mas ausente: $canon_path"
    drift_count=$((drift_count + 1))
    continue
  fi

  if [[ ! -f "$mirror_path" ]]; then
    echo "DRIFT: espelho ausente: $mirror_path"
    drift_count=$((drift_count + 1))
    continue
  fi

  if ! diff -q "$canon_path" "$mirror_path" > /dev/null 2>&1; then
    echo "DRIFT: $policy diverge entre $canonical e $mirror"
    diff -u "$canon_path" "$mirror_path" 2>&1 | sed 's/^/  /' || true
    drift_count=$((drift_count + 1))
  else
    ok_count=$((ok_count + 1))
  fi
done

if [[ -d "$mirror" ]]; then
  for existing in "$mirror"/*.md; do
    [[ -f "$existing" ]] || continue
    base="$(basename "$existing")"
    declared=0
    for entry in "${required_policies[@]}"; do
      [[ "$entry" == "$base" ]] && { declared=1; break; }
    done
    if [[ "$declared" -eq 0 ]]; then
      echo "DRIFT: $mirror/$base existe mas nao esta na lista declarada required_policies de $0"
      drift_count=$((drift_count + 1))
    fi
  done
fi

if [[ -d "$canonical" ]]; then
  for existing in "$canonical"/*.md; do
    [[ -f "$existing" ]] || continue
    base="$(basename "$existing")"
    declared=0
    for entry in "${required_policies[@]}"; do
      [[ "$entry" == "$base" ]] && { declared=1; break; }
    done
    if [[ "$declared" -eq 0 ]]; then
      echo "DRIFT: $canonical/$base existe mas nao esta na lista declarada required_policies de $0"
      drift_count=$((drift_count + 1))
    fi
  done
fi

echo
echo "Policies em sync: $ok_count"
echo "Drift / missing: $drift_count"

if [[ "$drift_count" -gt 0 ]]; then
  echo
  echo "Para corrigir: ./scripts/sync-policies.sh"
  exit 1
fi

echo "Todas as policies sincronizadas entre .agents/policies/ e .claude/rules/"
exit 0
