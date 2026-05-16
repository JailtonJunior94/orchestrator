#!/usr/bin/env bash
# Hook de pos-edicao para Codex CLI: avisa quando arquivos de governanca sao modificados.
#
# Uso: registrar em .codex/config.toml como hook de pos-execucao, ou chamar manualmente
# apos editar um arquivo:
#   bash .codex/hooks/validate-governance.sh <file_path>
#
# Modos (via variavel de ambiente CODEX_GOVERNANCE_MODE):
#   fail  — emite aviso em stderr, exit 1 (bloqueia a edicao) [DEFAULT]
#   warn  — emite aviso em stderr, exit 0 (nao bloqueia, opt-out explicito)
#
# Para desabilitar o bloqueio (nao recomendado fora de testes):
#   export CODEX_GOVERNANCE_MODE=warn

set -euo pipefail

CODEX_GOVERNANCE_MODE="${CODEX_GOVERNANCE_MODE:-fail}"

file_path="${1:-}"

[[ -n "$file_path" ]] || exit 0

case "$file_path" in
  */.agents/skills/*/SKILL.md|*/.agents/skills/*/references/*.md|*/AGENTS.md)
    echo "AVISO: arquivo de governanca modificado: $file_path" >&2
    echo "Verifique se esta edicao e intencional e se nao quebra o contrato de upgrade." >&2
    if [[ "$CODEX_GOVERNANCE_MODE" == "fail" ]]; then
      exit 1
    fi
    ;;
esac

exit 0
