#!/usr/bin/env bash
# Valida o pacote de evidencias de um relatorio de refatoracao.
# Uso: $0 <refactor_report.md>
#
# Exit 0 = aprovado, Exit 1 = reprovado, Exit 2 = uso incorreto.

set -euo pipefail

export LC_ALL=C

if [[ $# -ne 1 ]]; then
  echo "Uso: $0 <refactor_report.md>"
  exit 2
fi

report_file="$1"

if [[ ! -f "$report_file" ]]; then
  echo "ERRO: arquivo de relatorio nao encontrado: $report_file"
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

# Secoes obrigatorias
require_heading "escopo"                      "seção Escopo"
require_heading "invariantes"                 "seção Invariantes Preservadas"
require_heading "mudanc"                      "seção Mudanças"
require_heading "comandos executados"         "seção Comandos Executados"
require_heading "resultados de validac"       "seção Resultados de Validação"
require_heading "riscos residuais"            "seção Riscos Residuais"

# Modo documentado
require_pattern "Modo[[:space:]]*:[[:space:]]*(advisory|execution)" \
  "campo Modo (advisory|execution)"

# Estado terminal canonico
if ! grep -Eiq "Estado[[:space:]]*:[[:space:]]*(needs_input|blocked|failed|done)" "$report_file"; then
  echo "FALTANDO: estado terminal canonico (needs_input|blocked|failed|done)"
  missing=1
fi

# Evidencia de testes e lint
require_pattern "Testes[[:space:]]*:[[:space:]]*(pass|fail|blocked|n/a)" \
  "evidencia de testes com resultado"
require_pattern "Lint[[:space:]]*:[[:space:]]*(pass|fail|blocked|n/a)" \
  "evidencia de lint com resultado"

# Veredito do revisor (obrigatorio em modo execution)
if grep -Eiq "Modo[[:space:]]*:[[:space:]]*execution" "$report_file"; then
  refactor_blocking_re='(\[(critical|cr(i|í)tico|high|hard|alta|alto|blocker|security)\]|severidade[[:space:]]*:[[:space:]]*(critical|high|cr(i|í)tico|alta|alto)|severity[[:space:]]*:[[:space:]]*(critical|high))'
  refactor_any_re='(\[(critical|cr(i|í)tico|high|hard|alta|alto|blocker|security|medium|m(e|é)dia|important|importante|low|baixa|suggestion|sugest(a|ã)o)\]|severidade[[:space:]]*:[[:space:]]*(critical|high|medium|low|cr(i|í)tico|alta|alto|m(e|é)dia|baixa)|severity[[:space:]]*:[[:space:]]*(critical|high|medium|low))'
  if grep -Eiq "Veredito do Revisor[[:space:]]*:[[:space:]]*APPROVED_WITH_REMARKS([^A-Za-z_]|$)" "$report_file"; then
    if grep -Eiq "$refactor_blocking_re" "$report_file"; then
      echo "FALTANDO: APPROVED_WITH_REMARKS nao encerra com achado high/critical declarado (RF-33)"
      missing=1
    elif ! grep -Eiq "$refactor_any_re" "$report_file"; then
      echo "FALTANDO: APPROVED_WITH_REMARKS sem achado declarado com severidade canonica: ausencia de high/critical nao verificavel (RF-33, fail-closed)"
      missing=1
    fi
  elif ! grep -Eiq "Veredito do Revisor[[:space:]]*:[[:space:]]*(APPROVED|REJECTED|BLOCKED|n/a)([^A-Za-z_]|$)" "$report_file"; then
    echo "FALTANDO: veredito do revisor aceito (APPROVED|APPROVED_WITH_REMARKS sem high/critical|REJECTED|BLOCKED|n/a) (RF-33)"
    missing=1
  fi
fi

if [[ $missing -ne 0 ]]; then
  echo ""
  echo "Validacao do pacote de evidencias de refatoracao falhou: $report_file"
  exit 1
fi

echo "Validacao do pacote de evidencias de refatoracao aprovada: $report_file"
