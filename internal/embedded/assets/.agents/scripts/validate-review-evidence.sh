#!/usr/bin/env bash

# Valida o pacote de evidencias de um relatorio de review (modo --auto-review, RF-20).
# Espelha a estrutura de validate-task-evidence.sh para simetria de garantia.
# Uso: $0 <review.md>
#
# Exit 0 = aprovado, Exit 1 = reprovado, Exit 2 = uso incorreto.

set -euo pipefail

# Nota: sem LC_ALL=C — padrões de seção contêm chars acentuados (veredito, críticos).

if [[ $# -ne 1 ]]; then
  echo "Uso: $0 <review.md>"
  exit 2
fi

report_file="$1"

if [[ ! -f "$report_file" ]]; then
  echo "ERRO: arquivo de review nao encontrado: $report_file"
  exit 2
fi

missing=0

require_pattern() {
  local pattern="$1"
  local label="$2"
  if ! grep -Eiq "$pattern" "$report_file"; then
    echo "FALTANDO: $label"
    missing=1
  fi
}

require_heading() {
  local pattern="$1"
  local label="$2"
  if ! grep -Eiq "^#+[[:space:]]+$pattern" "$report_file"; then
    echo "FALTANDO: $label"
    missing=1
  fi
}

# Veredito canonico obrigatorio
if ! grep -Eiq "veredito[[:space:]]*:[[:space:]]*(APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED)" "$report_file" \
   && ! grep -Eiq "verdict[[:space:]]*:[[:space:]]*(APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED)" "$report_file"; then
  echo "FALTANDO: veredito canonico (APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED)"
  missing=1
fi

# Secoes obrigatorias (espelham o output minimo da Etapa 6 da skill review)
require_heading "achados"                       "seção Achados"
require_heading "arquivos revisados"            "seção Arquivos Revisados"
require_heading "riscos residuais"              "seção Riscos Residuais"
require_heading "valida"                        "seção Validações Executadas"

# Coerencia veredito ↔ achados:
# - Se o veredito for REJECTED, deve existir ao menos um achado critical ou high.
# - Se houver achados, cada um precisa de severidade canonica; "Sem achados" e valido.
verdict_value="$(grep -Eio '(veredito|verdict)[[:space:]]*:[[:space:]]*(APPROVED_WITH_REMARKS|APPROVED|REJECTED|BLOCKED)' "$report_file" | head -1 | grep -Eio '(APPROVED_WITH_REMARKS|APPROVED|REJECTED|BLOCKED)' | head -1 | tr '[:lower:]' '[:upper:]')"

has_no_findings=0
if grep -Eiq "sem achados" "$report_file"; then
  has_no_findings=1
fi

if [[ "$has_no_findings" -eq 0 ]]; then
  # Exigir ao menos uma severidade canonica declarada quando ha achados
  if ! grep -Eiq "severidade[[:space:]]*:[[:space:]]*(critical|high|medium|low|cr[ií]tico|alta|m[eé]dia|baixa)" "$report_file" \
     && ! grep -Eiq "severity[[:space:]]*:[[:space:]]*(critical|high|medium|low)" "$report_file"; then
    echo "FALTANDO: severidade canonica em ao menos um achado (critical|high|medium|low) ou declaração 'Sem achados'"
    missing=1
  fi
fi

if [[ "$verdict_value" == "REJECTED" ]]; then
  if ! grep -Eiq "severidade[[:space:]]*:[[:space:]]*(critical|high|cr[ií]tico|alta)" "$report_file" \
     && ! grep -Eiq "severity[[:space:]]*:[[:space:]]*(critical|high)" "$report_file"; then
    echo "FALTANDO: veredito REJECTED exige ao menos um achado de severidade critical ou high comprovado"
    missing=1
  fi
fi

# Diff/alvo revisado: exigir evidencia de que algo foi efetivamente lido
require_pattern "(diff|branch|commit|arquivos? revisad)" "referência ao alvo revisado (diff/branch/commit/arquivos)"

if [[ $missing -ne 0 ]]; then
  echo ""
  echo "Validacao do pacote de evidencias de review falhou: $report_file"
  exit 1
fi

echo "Validacao do pacote de evidencias de review aprovada: $report_file"
