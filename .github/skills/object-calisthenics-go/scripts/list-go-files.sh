#!/usr/bin/env bash
set -euo pipefail

if ! command -v rg >/dev/null 2>&1; then
  echo "rg nao encontrado no PATH" >&2
  exit 1
fi

# Array preenchido via laco `read` em vez do builtin de bash 4+ para leitura de
# array: o macOS entrega bash 3.2 em /bin/bash, onde esse builtin nao existe e,
# sob `set -e`, abortava a skill inteira. O laco e portavel e preserva os nomes
# byte-identicos, inclusive com espaco.
files=()
while IFS= read -r linha; do
  files+=("$linha")
done < <(rg --files -g '*.go' -g '!vendor/**' -g '!**/testdata/**' -g '!**/node_modules/**')

if [ "${#files[@]}" -eq 0 ]; then
  echo "nenhum arquivo Go encontrado no workspace atual" >&2
  exit 1
fi

printf '%s\n' "${files[@]}"
