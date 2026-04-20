#!/usr/bin/env bash
# Hook de pos-edicao para Gemini CLI: avisa quando arquivos de governanca sao modificados.
#
# Uso: chamar manualmente apos editar um arquivo, ou via --hook do Gemini CLI:
#   gemini --hook "bash .gemini/hooks/validate-governance.sh {file}"
#
# Recebe o caminho do arquivo como $1 (Gemini CLI passa o arquivo editado).
#
# Modos (via variavel de ambiente GEMINI_GOVERNANCE_MODE):
#   fail  — emite aviso em stderr, exit 1 (bloqueia a edicao) [DEFAULT]
#   warn  — emite aviso em stderr, exit 0 (nao bloqueia, opt-out explicito)
#
# Para desabilitar o bloqueio (nao recomendado fora de testes):
#   export GEMINI_GOVERNANCE_MODE=warn

set -euo pipefail

GEMINI_GOVERNANCE_MODE="${GEMINI_GOVERNANCE_MODE:-fail}"

file_path="${1:-}"

[[ -n "$file_path" ]] || exit 0

case "$file_path" in
  */.agents/skills/*/SKILL.md|*/.agents/skills/*/references/*.md|*/AGENTS.md)
    echo "AVISO: arquivo de governanca modificado: $file_path" >&2
    echo "Verifique se esta edicao e intencional e se nao quebra o contrato de upgrade." >&2
    if [[ "$GEMINI_GOVERNANCE_MODE" == "fail" ]]; then
      exit 1
    fi
    ;;
esac

exit 0
