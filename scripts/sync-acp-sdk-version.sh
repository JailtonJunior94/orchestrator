#!/usr/bin/env bash

set -euo pipefail

GOMOD="${GOMOD_PATH:-go.mod}"
SPECS_DIR="${SPECS_DIR_PATH:-internal/runtime/specs}"
PACKAGE="github.com/coder/acp-go-sdk"

if [[ ! -f "$GOMOD" ]]; then
  echo "ERRO: $GOMOD não encontrado" >&2
  exit 1
fi

if [[ ! -d "$SPECS_DIR" ]]; then
  echo "ERRO: $SPECS_DIR não encontrado" >&2
  exit 1
fi

sdk_version="$(grep -F "$PACKAGE" "$GOMOD" | awk '{for(i=1;i<=NF;i++) if($i ~ /^v[0-9]/) {print $i; exit}}' | head -1)"

if [[ -z "$sdk_version" ]]; then
  echo "ERRO: pacote '${PACKAGE}' não encontrado em $GOMOD" >&2
  exit 1
fi

if ! [[ "$sdk_version" =~ ^v[0-9]+\.[0-9]+\.[0-9] ]]; then
  echo "ERRO: versão '$sdk_version' não é semântica (esperado vX.Y.Z)" >&2
  exit 1
fi

found_total=0
updated_total=0

while IFS= read -r spec_file; do
  [[ -n "$spec_file" ]] || continue

  while IFS= read -r const_name; do
    [[ -n "$const_name" ]] || continue
    found_total=$((found_total + 1))

    current_version="$(grep -E "${const_name}[[:space:]]*=[[:space:]]*\"v[^\"]*\"" "$spec_file" \
      | sed -E 's/.*"(v[^"]+)".*/\1/' | head -1)"

    if [[ "$current_version" == "$sdk_version" ]]; then
      continue
    fi

    sed -i.bak -E "s|(${const_name}[[:space:]]*=[[:space:]]*\")v[^\"]*(\")|\1${sdk_version}\2|" "$spec_file"
    rm -f "${spec_file}.bak"
    updated_total=$((updated_total + 1))
    echo "sync-acp-sdk-version: ${const_name} atualizada de $current_version para $sdk_version em $spec_file" >&2
  done < <(grep -oE '[A-Za-z0-9_]*SDKVersion[[:space:]]*=[[:space:]]*"v[^"]*"' "$spec_file" \
    | sed -E 's/[[:space:]]*=.*//' | sort -u)

done < <(find "$SPECS_DIR" -maxdepth 1 -name '*.go' -not -name '*_test.go' | sort)

if [[ "$found_total" -eq 0 ]]; then
  echo "ERRO: nenhuma constante *SDKVersion encontrada em $SPECS_DIR" >&2
  exit 1
fi

if [[ "$updated_total" -eq 0 ]]; then
  exit 0
fi

echo "sync-acp-sdk-version: $updated_total de $found_total constante(s) sincronizada(s) com $GOMOD" >&2
exit 0
