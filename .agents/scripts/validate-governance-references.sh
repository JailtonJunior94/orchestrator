#!/usr/bin/env bash
# validate-governance-references.sh
# Gate tool-neutro: valida que cada reference .md em agent-governance/references/
# possui o cabeçalho mínimo necessário para lazy-loading e auditoria.
#
# Critérios obrigatórios (causam exit != 0):
#   1. Bloco TL;DR delimitado por <!-- TL;DR ... -->
#   2. Linha "- Rule ID: R-XXX-NNN"
#   3. Linha "- Severidade: hard|guideline" (ou indicação inline equivalente)
#
# Uso:
#   bash .agents/scripts/validate-governance-references.sh [path/to/references]
# Sem argumento: descobre todas as references em .agents/skills/*/references/.

set -euo pipefail

ROOT_DIR="${ROOT_DIR:-$(pwd)}"
TARGET="${1:-}"

discover_references() {
  if [[ -n "$TARGET" ]]; then
    if [[ -d "$TARGET" ]]; then
      find "$TARGET" -maxdepth 1 -type f -name "*.md" 2>/dev/null | LC_ALL=C sort
    else
      printf '%s\n' "$TARGET"
    fi
  else
    find "$ROOT_DIR/.agents/skills" -path "*/references/*.md" -type f 2>/dev/null | LC_ALL=C sort
  fi
}

ok=0
fail=0
failed_files=()

validate_file() {
  local file="$1"
  local errs=()

  # 1. TL;DR block (multi-line tolerant)
  if ! awk '/<!-- TL;DR/{found=1} END{exit found?0:1}' "$file"; then
    errs+=("missing TL;DR block (<!-- TL;DR ...)")
  fi

  # 2. Rule ID line (formato R-XXX-NNN)
  if ! grep -Eq '^- Rule ID:[[:space:]]*R-[A-Z]+-[0-9]{3}' "$file"; then
    errs+=("missing Rule ID line (- Rule ID: R-XXX-NNN)")
  fi

  # 3. Severidade declarada (hard | guideline | informativo)
  if ! grep -Eqi '^- Severidade:[[:space:]]*(hard|guideline|informativo)' "$file"; then
    errs+=("missing Severidade line (- Severidade: hard|guideline|informativo)")
  fi

  if [[ ${#errs[@]} -eq 0 ]]; then
    ok=$((ok + 1))
  else
    fail=$((fail + 1))
    failed_files+=("$file")
    printf '[FAIL] %s\n' "$file" >&2
    for e in "${errs[@]}"; do
      printf '       - %s\n' "$e" >&2
    done
  fi
}

while IFS= read -r file; do
  [[ -z "$file" ]] && continue
  # Ignorar arquivos auxiliares conhecidos (json, etc.)
  case "$file" in
    *.json) continue ;;
  esac
  validate_file "$file"
done < <(discover_references)

total=$((ok + fail))
if [[ "$fail" -eq 0 ]]; then
  printf 'OK: %d/%d references validadas\n' "$ok" "$total"
  exit 0
else
  printf 'FAIL: %d/%d references com problemas\n' "$fail" "$total" >&2
  exit 1
fi
