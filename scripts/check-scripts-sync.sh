#!/usr/bin/env bash
# check-scripts-sync.sh
# Detecta drift entre o diretorio canonico (.agents/scripts/) dos validadores de evidencia
# e os mirrors (.claude/scripts/, internal/embedded/assets/.claude/scripts/).
#
# Fonte de verdade: .agents/scripts/ (tool-neutro, portatil). Resolucao em cascata pelas skills:
# .agents/scripts/ -> .claude/scripts/ -> scripts/.
#
# Uso: ./scripts/check-scripts-sync.sh
# Exit 0 = sincronizado; exit 1 = drift detectado.
# Para corrigir: ./scripts/sync-skills.sh

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.agents/scripts"

EVIDENCE_VALIDATORS=(
  "validate-task-evidence.sh"
  "validate-bugfix-evidence.sh"
  "validate-refactor-evidence.sh"
  "validate-review-evidence.sh"
)

declare -a mirror_dirs=(
  "$repo_root/.claude/scripts"
  "$repo_root/internal/embedded/assets/.claude/scripts"
  "$repo_root/internal/embedded/assets/.agents/scripts"
)

drift_count=0
ok_count=0

for validator in "${EVIDENCE_VALIDATORS[@]}"; do
  canon_path="$canonical/$validator"
  if [[ ! -f "$canon_path" ]]; then
    # validate-review-evidence.sh pode nao existir ate a Tarefa 7.0 — pular quando ausente no canonico.
    continue
  fi

  for mirror in "${mirror_dirs[@]}"; do
    mirror_path="$mirror/$validator"
    if [[ ! -f "$mirror_path" ]]; then
      echo "MISSING: $mirror_path"
      drift_count=$((drift_count + 1))
      continue
    fi
    if ! diff -q "$canon_path" "$mirror_path" >/dev/null 2>&1; then
      echo "DRIFT: $validator diverge entre $canonical e $mirror"
      drift_count=$((drift_count + 1))
    else
      ok_count=$((ok_count + 1))
    fi
  done
done

echo
echo "Validadores em sync: $ok_count"
echo "Drift / missing: $drift_count"

if [[ "$drift_count" -gt 0 ]]; then
  echo
  echo "Para corrigir: ./scripts/sync-skills.sh"
  exit 1
fi

echo "Todos os validadores de evidencia sincronizados."
exit 0
