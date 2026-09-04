#!/usr/bin/env bash
# check-mocks.sh
# Detecta drift entre as interfaces declaradas em mockery.yml e os mocks gerados
# em internal/**/mocks/. A geração ocorre numa árvore isolada e descartável.
#
# Uso: ./scripts/check-mocks.sh
# Exit 0 quando os mocks estao sincronizados; exit 1 quando ha drift.
#
# Nao modifica arquivos versionados. Para corrigir drift, rode: make mocks

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

mockery_version="v3.7.4"
mockery_bin="${MOCKERY_BIN:-go}"
generation_root="$(mktemp -d)"
output="$(mktemp)"
cleanup() {
  rm -rf "$generation_root"
  rm -f "$output"
}
trap cleanup EXIT

cp go.mod go.sum mockery.yml "$generation_root/"
cp -R internal "$generation_root/internal"
mkdir -p "$generation_root/scripts"
cp scripts/normalize-mocks.sh "$generation_root/scripts/normalize-mocks.sh"

run_mockery() {
  if (cd "$generation_root" && "$mockery_bin" run "github.com/vektra/mockery/v3@${mockery_version}" --config mockery.yml) >"$output" 2>&1; then
    return 0
  fi

  echo "[HARD] mockery ${mockery_version} falhou ao gerar os mocks; diagnostico:" >&2
  cat "$output" >&2
  return 1
}

before="$(mktemp)"
after="$(mktemp)"
trap 'cleanup; rm -f "$before" "$after"' EXIT
(cd "$repo_root" && find internal -path "*/mocks/*.go" -type f -print0 | sort -z | xargs -0 shasum) >"$before"

run_mockery
bash "$generation_root/scripts/normalize-mocks.sh" "$generation_root" >/dev/null 2>&1

(cd "$generation_root" && find internal -path "*/mocks/*.go" -type f -print0 | sort -z | xargs -0 shasum) >"$after"

if ! diff -q "$before" "$after" >/dev/null; then
  echo "[HARD] mocks desatualizados — rode 'make mocks' e commite as alteracoes." >&2
  diff "$before" "$after" >&2 || true
  exit 1
fi

echo "Mocks: OK (sincronizados com mockery.yml)"
