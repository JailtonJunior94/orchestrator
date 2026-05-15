#!/usr/bin/env bash
# post-wave.sh
# Escreve checkpoint incremental do orquestrador apos cada wave.
# Fecha F31 (orchestrator crash mid-flight resilience).
#
# Uso:
#   bash .claude/hooks/post-wave.sh <prd-slug> <wave-id> [results-yaml-file]
#
# Comportamento:
#   - Cria/atualiza tasks/prd-<slug>/_orchestration_report.partial.md
#   - Append-only: cada chamada adiciona uma seção da wave
#   - Quando o orquestrador concluir todas as waves, fica responsavel por
#     renomear .partial.md -> _orchestration_report.md (rename atomico)
#
# Exit:
#   0 — checkpoint escrito
#   2 — argumentos invalidos

set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "Uso: $0 <prd-slug> <wave-id> [results-yaml-file]" >&2
  exit 2
fi

PRD_SLUG="$1"
WAVE_ID="$2"
RESULTS_FILE="${3:-}"

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
TASKS_ROOT="${AI_TASKS_ROOT:-tasks}"
PRD_PREFIX="${AI_PRD_PREFIX:-prd-}"
PRD_DIR="$REPO_ROOT/$TASKS_ROOT/$PRD_PREFIX$PRD_SLUG"
PARTIAL_MD="$PRD_DIR/_orchestration_report.partial.md"

if [[ ! -d "$PRD_DIR" ]]; then
  echo "FAIL: PRD dir nao existe: $PRD_DIR" >&2
  exit 1
fi

ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Header inicial se ainda nao existe
if [[ ! -f "$PARTIAL_MD" ]]; then
  cat > "$PARTIAL_MD.tmp" <<EOF
# Relatorio de Orquestracao (parcial)

PRD: $PRD_SLUG
Iniciado: $ts

## Waves Executadas

EOF
  mv "$PARTIAL_MD.tmp" "$PARTIAL_MD"
fi

# Append entry da wave
{
  echo
  echo "### Wave $WAVE_ID — $ts"
  echo
  if [[ -n "$RESULTS_FILE" && -s "$RESULTS_FILE" ]]; then
    echo '```yaml'
    cat "$RESULTS_FILE"
    echo
    echo '```'
  else
    echo "(sem resultados anexados)"
  fi
} >> "$PARTIAL_MD"

echo "post-wave: checkpoint atualizado em $PARTIAL_MD" >&2
exit 0
