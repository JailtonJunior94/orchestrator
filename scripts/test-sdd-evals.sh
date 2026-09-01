#!/usr/bin/env bash
# Executa o corpus SDD sem runtimes remotos. Cada fixture adversaria precisa ser
# rejeitada pelo contrato versionado; o único caso positivo é o controle.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
corpus_root="$repo_root/evals/sdd"
manifest="$corpus_root/manifest.json"
binary="${TMPDIR:-/tmp}/ai-spec-sdd-evals-${RANDOM}${RANDOM}"

cleanup() { rm -f "$binary"; }
trap cleanup EXIT

[[ -f "$manifest" ]] || { echo "manifesto do corpus ausente: $manifest" >&2; exit 1; }
grep -Fq '"version": 1' "$manifest" || { echo 'versao do corpus invalida' >&2; exit 1; }

fixture_count=$(find "$corpus_root/fixtures" -type f -name '*.json' | wc -l | tr -d '[:space:]')
[[ "$fixture_count" -ge 20 ]] || { echo "corpus insuficiente: $fixture_count fixtures" >&2; exit 1; }

for category in schema integrity proof paths review; do
  count=$(find "$corpus_root/fixtures/$category" -type f -name '*.json' | wc -l | tr -d '[:space:]')
  [[ "$count" -ge 2 ]] || { echo "categoria $category insuficiente: $count" >&2; exit 1; }
done

(cd "$repo_root" && go build -trimpath -o "$binary" .)

passed=0
rejected=0
while IFS= read -r fixture; do
  name="$(basename "$fixture")"
  case "$name" in
    *.execution.reject.json)
      if "$binary" validate-result execution "$fixture" >/dev/null 2>&1; then
        echo "fixture deveria ser rejeitada: ${fixture#$repo_root/}" >&2
        exit 1
      fi
      rejected=$((rejected + 1))
      ;;
    *.review.reject.json)
      if "$binary" validate-result review "$fixture" >/dev/null 2>&1; then
        echo "fixture deveria ser rejeitada: ${fixture#$repo_root/}" >&2
        exit 1
      fi
      rejected=$((rejected + 1))
      ;;
    *.execution.accept.json)
      "$binary" validate-result execution "$fixture" >/dev/null
      passed=$((passed + 1))
      ;;
    *)
      echo "nome de fixture sem contrato de expectativa: ${fixture#$repo_root/}" >&2
      exit 1
      ;;
  esac
done < <(find "$corpus_root/fixtures" -type f -name '*.json' | sort)

[[ "$passed" -ge 1 ]] || { echo 'controle positivo ausente' >&2; exit 1; }
[[ "$rejected" -ge 20 ]] || { echo "casos adversariais insuficientes: $rejected" >&2; exit 1; }
echo "sdd evals: fixtures=$fixture_count accepted=$passed rejected=$rejected prompt_tokens=0 external_cost_usd=0"
