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

checkpoint_root="$repo_root/.specs/prd-sdd-robusto/.checkpoints"
if find "$checkpoint_root" -maxdepth 1 -type f \( -name '*.yaml' -o -name '*.yml' \) | grep -q .; then
  echo "checkpoint legado fora do contrato JSON v2" >&2
  exit 1
fi
while IFS= read -r checkpoint; do
  "$binary" validate-result execution "$checkpoint" >/dev/null
done < <(find "$checkpoint_root" -maxdepth 1 -type f -name '*.json' | sort)
"$binary" validate-sdd "$repo_root/.specs/prd-sdd-robusto" >/dev/null

python3 - "$manifest" "$binary" "$corpus_root/fixtures" <<'PY'
import json
import pathlib
import subprocess
import sys
import time

manifest = json.load(open(sys.argv[1], encoding="utf-8"))
binary = sys.argv[2]
fixtures = sorted(pathlib.Path(sys.argv[3]).rglob("*.json"))
true_positive = true_negative = false_positive = false_negative = 0
latencies = []
for fixture in fixtures:
    expected_accept = fixture.name.endswith(".accept.json")
    if ".execution." in fixture.name:
        kind = "execution"
    elif ".review." in fixture.name:
        kind = "review"
    else:
        raise SystemExit(f"fixture sem contrato de expectativa: {fixture}")
    started = time.perf_counter_ns()
    result = subprocess.run([binary, "validate-result", kind, str(fixture)], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    latencies.append((time.perf_counter_ns() - started) / 1_000_000)
    actual_accept = result.returncode == 0
    if expected_accept and actual_accept:
        true_positive += 1
    elif expected_accept:
        false_positive += 1
    elif actual_accept:
        false_negative += 1
    else:
        true_negative += 1

total = len(fixtures)
adversarial = true_negative + false_negative
positive = true_positive + false_positive
metrics = {
    "fixtures": total,
    "accepted": true_positive,
    "rejected": true_negative,
    "quality_rate": (true_positive + true_negative) / total,
    "escape_rate": false_negative / adversarial if adversarial else 0.0,
    "false_positive_rate": false_positive / positive if positive else 0.0,
    "false_negative_rate": false_negative / adversarial if adversarial else 0.0,
    "latency_ms": {"total": round(sum(latencies), 3), "mean": round(sum(latencies) / total, 3), "max": round(max(latencies), 3)},
}
thresholds = manifest["quality_metrics"]
failures = []
if adversarial < thresholds["minimum_adversarial_fixtures"]:
    failures.append("corpus adversarial insuficiente")
if metrics["quality_rate"] < thresholds["minimum_quality_rate"]:
    failures.append("quality_rate abaixo do threshold")
for metric, key in (("escape_rate", "maximum_escape_rate"), ("false_positive_rate", "maximum_false_positive_rate"), ("false_negative_rate", "maximum_false_negative_rate")):
    if metrics[metric] > thresholds[key]:
        failures.append(f"{metric} acima do threshold")
if metrics["latency_ms"]["total"] > thresholds["maximum_total_latency_ms"]:
    failures.append("latencia total excedeu threshold")
print("sdd evals: " + json.dumps(metrics, sort_keys=True, separators=(",", ":")))
if failures:
    raise SystemExit("; ".join(failures))
PY
