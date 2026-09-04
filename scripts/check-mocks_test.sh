#!/usr/bin/env bash
# check-mocks_test.sh
# Exercita o caminho de falha do gate sem executar o gerador real.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

mkdir -p "$fixture/scripts" "$fixture/internal/example/mocks"
cp "$repo_root/scripts/check-mocks.sh" "$fixture/scripts/check-mocks.sh"
cp "$repo_root/scripts/normalize-mocks.sh" "$fixture/scripts/normalize-mocks.sh"
touch "$fixture/internal/example/mocks/mock.go"
printf 'module example.test/mocks\n\ngo 1.26.2\n' >"$fixture/go.mod"
: >"$fixture/go.sum"
printf 'with-expecter: true\n' >"$fixture/mockery.yml"

fake_mockery="$fixture/mockery-failure"
printf '%s\n' '#!/usr/bin/env bash' 'printf "parcial" > internal/example/mocks/mock.go' 'echo "erro de geracao simulado" >&2' 'exit 42' > "$fake_mockery"
chmod +x "$fake_mockery"

output="$fixture/output"
if (
  cd "$fixture"
  MOCKERY_BIN="$fake_mockery" bash scripts/check-mocks.sh
) >"$output" 2>&1; then
  echo "[HARD] check-mocks deveria falhar quando o gerador falha." >&2
  exit 1
fi

grep -F '[HARD] mockery v3.7.4 falhou ao gerar os mocks; diagnostico:' "$output" >/dev/null
grep -F 'erro de geracao simulado' "$output" >/dev/null
[[ ! -s "$fixture/internal/example/mocks/mock.go" ]] || { echo "falha contaminou worktree" >&2; exit 1; }

fake_drift="$fixture/mockery-drift"
printf '%s\n' '#!/usr/bin/env bash' 'printf "drift" > internal/example/mocks/mock.go' >"$fake_drift"
chmod +x "$fake_drift"
if (cd "$fixture" && MOCKERY_BIN="$fake_drift" bash scripts/check-mocks.sh) >"$output" 2>&1; then
  echo "check-mocks deveria detectar drift" >&2
  exit 1
fi
grep -F 'mocks desatualizados' "$output" >/dev/null
[[ ! -s "$fixture/internal/example/mocks/mock.go" ]] || { echo "drift contaminou worktree" >&2; exit 1; }

fake_success="$fixture/mockery-success"
printf '%s\n' '#!/usr/bin/env bash' 'exit 0' >"$fake_success"
chmod +x "$fake_success"
(cd "$fixture" && MOCKERY_BIN="$fake_success" bash scripts/check-mocks.sh) >/dev/null

echo "check-mocks: sucesso, drift e falha isolados OK"
