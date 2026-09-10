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
  if ! grep -Eiq "severidade[[:space:]]*:[[:space:]]*(critical|high|medium|low|cr(i|í)tico|alta|m(e|é)dia|baixa)" "$report_file" \
     && ! grep -Eiq "severity[[:space:]]*:[[:space:]]*(critical|high|medium|low)" "$report_file"; then
    echo "FALTANDO: severidade canonica em ao menos um achado (critical|high|medium|low) ou declaração 'Sem achados'"
    missing=1
  fi
fi

if [[ "$verdict_value" == "REJECTED" ]]; then
  if ! grep -Eiq "severidade[[:space:]]*:[[:space:]]*(critical|high|cr(i|í)tico|alta)" "$report_file" \
     && ! grep -Eiq "severity[[:space:]]*:[[:space:]]*(critical|high)" "$report_file"; then
    echo "FALTANDO: veredito REJECTED exige ao menos um achado de severidade critical ou high comprovado"
    missing=1
  fi
fi

# Diff/alvo revisado: exigir evidencia de que algo foi efetivamente lido
require_pattern "(diff|branch|commit|arquivos? revisad)" "referência ao alvo revisado (diff/branch/commit/arquivos)"

map_heading_re='^#+[[:space:]]+mapa de crit(e|é)rios de aceite'
if ! grep -Eiq "$map_heading_re" "$report_file"; then
  echo "FALTANDO: seção 'Mapa de Criterios de Aceite' (mapa 1:1 criterio -> evidencia, RF-47/RF-51)"
  missing=1
else
  in_map=0
  criteria_lines=0
  while IFS= read -r line; do
    if [[ "$line" =~ ^#+[[:space:]] ]]; then
      if grep -Eiq "$map_heading_re" <<<"#$line" || grep -Eiq 'mapa de crit(e|é)rios de aceite' <<<"$line"; then
        in_map=1
      else
        in_map=0
      fi
      continue
    fi
    [[ "$in_map" -eq 1 ]] || continue
    [[ "$line" =~ ^-[[:space:]]*'[' ]] || continue
    criteria_lines=$((criteria_lines + 1))

    marker="$(printf '%s' "$line" | sed -E 's/^-[[:space:]]*\[([^]]*)\].*/\1/' | tr 'A-Z' 'a-z' | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')"
    if [[ "$line" == *"->"* ]]; then
      evidence="${line#*->}"
      evidence="${evidence# }"
    else
      evidence=""
    fi

    case "$marker" in
      "atendido"|"nao atendido"|"não atendido") : ;;
      "nao verificavel"|"nao verificável"|"não verificavel"|"não verificável")
        echo "FALTANDO: criterio marcado 'nao verificavel' proibe APPROVED (RF-49): $line"
        missing=1
        ;;
      *)
        echo "FALTANDO: marcador de criterio invalido no mapa 1:1 (use: atendido, nao atendido, nao verificavel): $line"
        missing=1
        ;;
    esac

    if [[ -z "${evidence// /}" ]]; then
      echo "FALTANDO: criterio sem linha de evidencia no mapa 1:1 (esperado '-> <evidencia>'): $line"
      missing=1
      continue
    fi

    if grep -Eq '(^|[^[:alnum:]_])[[:alnum:]_./-]+:[0-9]+' <<<"$evidence"; then
      :
    elif grep -Eiq '(test|teste|spec).*(pass|fail|passed|failed|(^|[^[:alpha:]])ok([^[:alpha:]]|$)|erro|error)' <<<"$evidence"; then
      :
    elif grep -Eiq '(go (test|build|vet)|gotestsum|bash |sh |npm|pnpm|yarn|pytest|make |grep |python|cargo |dotnet |shasum|awk |sed |cat |\./)' <<<"$evidence" \
         && grep -Eiq '(->|=>|exit|sa(i|í)da|output|pass|fail|(^|[^[:alpha:]])ok([^[:alpha:]]|$))' <<<"$evidence"; then
      :
    else
      echo "FALTANDO: linha de evidencia fora das tres formas de RF-48 (comando+saida, arquivo:linha, teste+resultado): $line"
      missing=1
    fi
  done < "$report_file"

  if [[ "$criteria_lines" -eq 0 ]]; then
    echo "FALTANDO: seção 'Mapa de Criterios de Aceite' sem nenhuma linha de criterio ('- ' seguido de marcador entre colchetes)"
    missing=1
  fi
fi

if [[ $missing -ne 0 ]]; then
  echo ""
  echo "Validacao do pacote de evidencias de review falhou: $report_file"
  exit 1
fi

echo "Validacao do pacote de evidencias de review aprovada: $report_file"
