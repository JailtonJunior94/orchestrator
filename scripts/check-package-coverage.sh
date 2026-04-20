#!/usr/bin/env bash
set -euo pipefail

THRESHOLD=${1:-70}
FAILED=0

CRITICAL_PACKAGES=(
    "./internal/parity/..."
    "./internal/specdrift/..."
    "./internal/evidence/..."
    "./internal/taskloop/..."
    "./internal/bugschema/..."
    "./internal/skills/..."
    "./internal/metrics/..."
    "./internal/detect/..."
)

for pkg in "${CRITICAL_PACKAGES[@]}"; do
    COV=$(go test -coverprofile=/dev/null "$pkg" 2>/dev/null | grep -oE '[0-9]+\.[0-9]+%' | head -1 | tr -d '%')
    if [ -z "$COV" ]; then
        echo "SKIP: $pkg (sem testes ou erro)"
        continue
    fi
    if [ "$(echo "$COV < $THRESHOLD" | bc -l)" = "1" ]; then
        echo "FAIL: $pkg — ${COV}% < ${THRESHOLD}%"
        FAILED=1
    else
        echo "PASS: $pkg — ${COV}%"
    fi
done

exit $FAILED
