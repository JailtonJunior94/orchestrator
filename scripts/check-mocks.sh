#!/usr/bin/env bash
# check-mocks.sh
# Detecta drift entre as interfaces declaradas em mockery.yml e os mocks gerados
# em internal/**/mocks/. Regenera os mocks numa arvore temporaria e compara.
#
# Uso: ./scripts/check-mocks.sh
# Exit 0 quando os mocks estao sincronizados; exit 1 quando ha drift.
#
# Nao modifica arquivos versionados. Para corrigir drift, rode: make mocks

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

mockery_version="v2.53.4"

# Snapshot dos mocks atuais.
before="$(mktemp)"
find internal -path "*/mocks/*.go" -type f -print0 \
  | sort -z \
  | xargs -0 shasum > "$before"

# Regenera os mocks in-place.
go run "github.com/vektra/mockery/v2@${mockery_version}" --config mockery.yml >/dev/null 2>&1
bash "$repo_root/scripts/normalize-mocks.sh" >/dev/null 2>&1

after="$(mktemp)"
find internal -path "*/mocks/*.go" -type f -print0 \
  | sort -z \
  | xargs -0 shasum > "$after"

if ! diff -q "$before" "$after" >/dev/null; then
  echo "[HARD] mocks desatualizados — rode 'make mocks' e commite as alteracoes." >&2
  diff "$before" "$after" >&2 || true
  rm -f "$before" "$after"
  exit 1
fi

rm -f "$before" "$after"
echo "Mocks: OK (sincronizados com mockery.yml)"
