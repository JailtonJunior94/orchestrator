#!/usr/bin/env bash
# normalize-mocks.sh
# Pos-processa os mocks gerados pelo mockery para usar 'any' no lugar de
# 'interface{}', mantendo conformidade com a Regra 7.1 (any em vez de interface{}).
#
# Uso: ./scripts/normalize-mocks.sh
# Idempotente: rodar varias vezes produz o mesmo resultado.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

count=0
while IFS= read -r f; do
  # interface{} -> any (cobre 'interface{}' isolado e em mapas/genericos).
  perl -0pi -e 's/interface\{\}/any/g' "$f"
  count=$((count + 1))
done < <(find internal -path "*/mocks/*.go" -type f)

echo "Mocks normalizados: ${count} arquivo(s)"
