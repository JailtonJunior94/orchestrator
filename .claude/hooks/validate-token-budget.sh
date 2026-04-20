#!/usr/bin/env bash
# validate-token-budget.sh — Hook opcional (nao bloqueante) de verificacao de budget de tokens.
# Emite warning se o total de bytes de governance files exceder o budget estimado.
# Uso: chamado via Claude Code hook PreToolUse (opcional).

set -euo pipefail

BUDGET_BYTES="${TOKEN_BUDGET_BYTES:-75000}"  # ~18.750 tokens a 4 bytes/token
PROJECT_DIR="${PROJECT_DIR:-.}"

total=0
while IFS= read -r -d '' f; do
  size=$(wc -c < "$f" 2>/dev/null || echo 0)
  total=$((total + size))
done < <(find "$PROJECT_DIR/.agents/skills" "$PROJECT_DIR/AGENTS.md" \
  -maxdepth 4 -type f -name "*.md" -print0 2>/dev/null)

estimated_tokens=$((total / 4))

if [ "$total" -gt "$BUDGET_BYTES" ]; then
  echo "AVISO: contexto de governanca estimado em ~${estimated_tokens} tokens (${total} bytes) excede budget de $((BUDGET_BYTES / 4)) tokens." >&2
  echo "       Considere usar --complexity=trivial ou --brief para reduzir o contexto." >&2
fi

exit 0
