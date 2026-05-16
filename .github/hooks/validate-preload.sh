#!/usr/bin/env bash
# Hook de pre-execucao para GitHub Copilot CLI: bloqueia execucao sem preload de governanca.
#
# Uso: registrar em .github/copilot-instructions.md ou invocar manualmente antes
# de cada sessao para garantir que AGENTS.md e agent-governance/SKILL.md estao
# carregados no contexto:
#
#   bash .github/hooks/validate-preload.sh
#
# Unlock: exportar GOVERNANCE_PRELOAD_CONFIRMED=1 na sessao atual.
# Consulte AGENTS.md para instrucoes completas de preload.

set -euo pipefail

if [[ "${GOVERNANCE_PRELOAD_CONFIRMED:-}" != "1" ]]; then
  echo "ERRO: governanca nao carregada." >&2
  echo "Execute: export GOVERNANCE_PRELOAD_CONFIRMED=1" >&2
  echo "Consulte AGENTS.md para instrucoes completas." >&2
  exit 1
fi
