#!/usr/bin/env bash
# check-skills-sync.sh
# Detecta drift entre o diretório canônico (.agents/skills/) e os mirrors:
# .claude/skills/, .github/skills/, internal/embedded/assets/.agents/skills/
#
# Uso: ./scripts/check-skills-sync.sh
# Exit 0 quando todos os mirrors estão sincronizados; exit 1 quando há drift.
#
# Não modifica arquivos. Para corrigir drift, rode: ./scripts/sync-skills.sh

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.agents/skills"

if [[ ! -d "$canonical" ]]; then
  echo "ERRO: diretório canônico não encontrado: $canonical" >&2
  exit 1
fi

declare -a mirrors=(
  "$repo_root/.claude/skills"
  "$repo_root/.github/skills"
  "$repo_root/internal/embedded/assets/.agents/skills"
)

drift_count=0
ok_count=0

for mirror in "${mirrors[@]}"; do
  if [[ ! -d "$mirror" ]]; then
    echo "WARN: mirror não existe: $mirror"
    continue
  fi

  # Verifica apenas skills que existem em ambos canonical e mirror.
  # Mirrors são subset por design.
  # Para o embedded mirror, replicamos a lógica de sync-skills.sh:
  # skills sem SKILL.md no canônico são preservadas (não sincronizadas).
  is_embedded=false
  case "$mirror" in *internal/embedded/*) is_embedded=true ;; esac

  for skill_dir in "$mirror"/*/; do
    skill_name="$(basename "$skill_dir")"
    canon_skill="$canonical/$skill_name"

    if [[ ! -d "$canon_skill" ]]; then
      echo "DRIFT: $mirror/$skill_name existe mas canonical $canon_skill não"
      drift_count=$((drift_count + 1))
      continue
    fi

    # Replicar comportamento de sync-skills.sh: para embedded mirror,
    # se canônico não tem SKILL.md, sync é pulado — mirror permanece como está.
    # F34: log explícito para auditoria — skill "fantasma" no embedded merece atenção.
    if [[ "$is_embedded" == "true" && ! -f "$canon_skill/SKILL.md" ]]; then
      echo "SKIP: $skill_name (canonical $canon_skill sem SKILL.md; embedded preservado)"
      ok_count=$((ok_count + 1))
      continue
    fi

    # diff -r retorna 0 se idêntico, 1 se diferente.
    # Filtramos saída pra mostrar só nomes de arquivos divergentes.
    if ! diff -r "$canon_skill" "$skill_dir" > /dev/null 2>&1; then
      echo "DRIFT: $skill_name diverge entre $canonical e $mirror"
      diff -rq "$canon_skill" "$skill_dir" 2>&1 | sed 's/^/  /' || true
      drift_count=$((drift_count + 1))
    else
      ok_count=$((ok_count + 1))
    fi
  done
done

echo
echo "Skills em sync: $ok_count"
echo "Skills com drift: $drift_count"

if [[ "$drift_count" -gt 0 ]]; then
  echo
  echo "Para corrigir: ./scripts/sync-skills.sh"
  exit 1
fi

echo "Todos os mirrors sincronizados com .agents/skills/"
exit 0
