#!/usr/bin/env bash
# post-wave.sh
# Escreve checkpoint incremental do orquestrador apos cada wave.
# Fecha F31 (orchestrator crash mid-flight resilience).
#
# Uso:
#   bash .claude/hooks/post-wave.sh <prd-slug> <wave-id> [results-yaml-file]
#
# Comportamento:
#   - Cria/atualiza .specs/prd-<slug>/_orchestration_report.partial.md
#   - Idempotente: reexecutar com o mesmo <wave-id> nao duplica a secao
#   - Atomico: escrita completa via temporario + rename, sob flock
#   - Sanitiza segredos do YAML de resultados antes de embutir no relatorio
#   - Quando o orquestrador concluir todas as waves, fica responsavel por
#     renomear .partial.md -> _orchestration_report.md (rename atomico)
#
# Exit:
#   0 — checkpoint escrito (ou wave ja registrada, idempotente)
#   1 — falha de IO
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
TASKS_ROOT="${AI_TASKS_ROOT:-.specs}"
PRD_PREFIX="${AI_PRD_PREFIX:-prd-}"
PRD_DIR="$REPO_ROOT/$TASKS_ROOT/$PRD_PREFIX$PRD_SLUG"
PARTIAL_MD="$PRD_DIR/_orchestration_report.partial.md"
LOCK_FILE="$PARTIAL_MD.lock"

if [[ ! -d "$PRD_DIR" ]]; then
  echo "FAIL: PRD dir nao existe: $PRD_DIR" >&2
  exit 1
fi

acquire_lock_mkdir() {
  local lock_dir="$1" timeout="$2" waited=0
  while true; do
    if mkdir "$lock_dir" 2>/dev/null; then
      echo "$$" > "$lock_dir/pid" 2>/dev/null || true
      return 0
    fi
    if [[ -f "$lock_dir/pid" ]]; then
      local holder_pid
      holder_pid="$(cat "$lock_dir/pid" 2>/dev/null || echo "")"
      if [[ -n "$holder_pid" ]] && ! kill -0 "$holder_pid" 2>/dev/null; then
        rm -rf "$lock_dir" 2>/dev/null || true
        continue
      fi
    fi
    if [[ $waited -ge $timeout ]]; then
      echo "FAIL: lock $lock_dir ocupado apos ${timeout}s" >&2
      return 1
    fi
    sleep 1
    waited=$((waited + 1))
  done
}

if [[ "${POST_WAVE_LOCK_HELD:-0}" != "1" ]]; then
  if command -v flock >/dev/null 2>&1; then
    exec env POST_WAVE_LOCK_HELD=1 flock -x -w 30 "$LOCK_FILE" "$0" "$@"
  fi
  LOCK_DIR="$LOCK_FILE.d"
  if ! acquire_lock_mkdir "$LOCK_DIR" 30; then
    exit 1
  fi
  trap 'rm -rf "$LOCK_DIR"' EXIT
fi

sanitize_stream() {
  sed -E \
    -e 's/-----BEGIN [A-Z ]*PRIVATE KEY-----.*-----END [A-Z ]*PRIVATE KEY-----/[REDACTED:pem_private_key]/g' \
    -e 's/(gh[po]_|sk-|AKIA)[A-Za-z0-9_-]{10,}/[REDACTED:provider_token]/g' \
    -e 's/eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/[REDACTED:jwt]/g' \
    -e 's/([Aa]uthorization:[[:space:]]*).+/\1[REDACTED:authorization_header]/g' \
    -e 's/^([A-Z][A-Z0-9_]*(SECRET|TOKEN|KEY|PASSWORD|PASS|CREDENTIAL|SESSION_ID)[A-Z0-9_]*=).+/\1[REDACTED:dotenv_secret]/g' \
    -e 's/^(api_key=).+/\1[REDACTED:dotenv_secret]/g' \
    -e 's#[a-zA-Z][a-zA-Z0-9+.-]*://[^[:space:]:/@]+:[^[:space:]:/@]+@[^[:space:]]+#[REDACTED:connection_string_credential]#g'
}

atomic_write() {
  local target="$1"
  local tmp
  tmp="$(mktemp "$PRD_DIR/.tmp-post-wave.XXXXXX")"
  cat > "$tmp"
  mv "$tmp" "$target"
}

ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

if [[ ! -f "$PARTIAL_MD" ]]; then
  atomic_write "$PARTIAL_MD" <<EOF
# Relatorio de Orquestracao (parcial)

PRD: $PRD_SLUG
Iniciado: $ts

## Waves Executadas

EOF
fi

WAVE_MARKER="### Wave $WAVE_ID —"
if grep -qF "$WAVE_MARKER" "$PARTIAL_MD" 2>/dev/null; then
  echo "post-wave: wave $WAVE_ID ja registrada em $PARTIAL_MD, ignorando (idempotente)" >&2
  exit 0
fi

{
  cat "$PARTIAL_MD"
  echo
  echo "### Wave $WAVE_ID — $ts"
  echo
  if [[ -n "$RESULTS_FILE" && -s "$RESULTS_FILE" ]]; then
    echo '```yaml'
    sanitize_stream < "$RESULTS_FILE"
    echo
    echo '```'
  else
    echo "(sem resultados anexados)"
  fi
} | atomic_write "$PARTIAL_MD"

echo "post-wave: checkpoint atualizado em $PARTIAL_MD" >&2
exit 0
