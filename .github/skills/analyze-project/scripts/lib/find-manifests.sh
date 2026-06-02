#!/usr/bin/env bash
# lib_find_manifests: enumera manifestos por padrao com profundidade limitada,
# excluindo diretorios de build/dependencias. Saida ordenada (LC_ALL=C sort).
# Uso: lib_find_manifests <dir> <pattern> [maxdepth=4]

lib_find_manifests() {
  local dir="${1:-.}"
  local pattern="$2"
  local maxdepth="${3:-4}"

  [[ -d "$dir" ]] || return 0
  [[ -n "$pattern" ]] || return 0

  find "$dir" -maxdepth "$maxdepth" -type f -name "$pattern" \
    -not -path "*/node_modules/*" \
    -not -path "*/vendor/*" \
    -not -path "*/dist/*" \
    -not -path "*/build/*" \
    -not -path "*/bin/*" \
    -not -path "*/obj/*" \
    -not -path "*/__pycache__/*" \
    -not -path "*/.git/*" \
    -not -path "*/.venv/*" \
    -not -path "*/target/*" \
    2>/dev/null | LC_ALL=C sort
}
