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

COMMAND_RE='^(go[[:space:]]+(test|build|vet|run)[[:space:]]+[^[:space:]]+|gotestsum[[:space:]]+[^[:space:]]+|golangci-lint[[:space:]]+run([^[:alnum:]]|$)|gofmt[[:space:]]+[^[:space:]]+|bash[[:space:]]+[^[:space:]]+|sh[[:space:]]+[^[:space:]]+|make[[:space:]]+[A-Za-z0-9][A-Za-z0-9_.-]*|grep[[:space:]]+[^[:space:]]+|rg[[:space:]]+[^[:space:]]+|python3?[[:space:]]+[^[:space:]]+|pytest[[:space:]]+[^[:space:]]+|npm[[:space:]]+(run[[:space:]]+)?[^[:space:]]+|pnpm[[:space:]]+[^[:space:]]+|yarn[[:space:]]+[^[:space:]]+|cargo[[:space:]]+[^[:space:]]+|dotnet[[:space:]]+[^[:space:]]+|git[[:space:]]+(diff|log|show|status|rev-parse|grep|blame)([^[:alnum:]]|$)|shasum[[:space:]]+[^[:space:]]+|sha256sum[[:space:]]+[^[:space:]]+|\./[^[:space:]]+)'
TEST_NAME_RE='^(Test|Benchmark|Example)[A-Za-z0-9_/]*$'
TEST_RESULT_RE='^(pass|fail)$'
CANONICAL_RECORD_RE='^(pass(ed)?|fail(ed)?|exit[[:space:]]+[0-9]+)$'
TRIVIAL_RECORD_RE='^(ok|okay|done|feito|pronto|sim|yes|no|nao|certo|tudo certo|tudo ok|aprovado|abc|talvez|maybe|n/?a|[-._]+)$'
SIGNAL_RECORD_RE='([0-9]|[A-Za-z0-9_-]+/[A-Za-z0-9_./-]+|[A-Za-z0-9_-]+\.(go|py|ts|tsx|js|jsx|cs|rs|java|rb|sh|sql|md|ya?ml|json|toml)([^A-Za-z0-9]|$)|(Test|Benchmark|Example)[A-Za-z0-9_]+)'

substantive_record() {
  local record="$1"
  record="$(printf '%s' "$record" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')"
  [[ -n "$record" ]] || return 1
  grep -Eiq "$CANONICAL_RECORD_RE" <<<"$record" && return 0
  grep -Eiq "$TRIVIAL_RECORD_RE" <<<"$record" && return 1
  [[ "$(wc -w <<<"$record")" -ge 2 ]] || return 1
  grep -Eq "$SIGNAL_RECORD_RE" <<<"$record"
}

path_is_reviewed() {
  local reference="$1"
  local declared
  [[ -n "$reference" ]] || return 1
  [[ -n "$reviewed_files" ]] || return 1
  while IFS= read -r declared; do
    if [[ -z "$declared" ]]; then
      continue
    fi
    if [[ "$declared" == "$reference" || "$declared" == */"$reference" || "$reference" == */"$declared" ]]; then
      return 0
    fi
  done <<<"$reviewed_files"
  return 1
}

VERDICT_KEY_RE='(final[[:space:]]+)?(veredito|veredicto|verdict)([[:space:]]+final)?'
VERDICT_DECOR_RE='[[:space:]*_`]*'
VERDICT_TOKEN_RE='(APPROVED_WITH_REMARKS|APROVADO_COM_RESSALVAS|APROVADO[[:space:]]+COM[[:space:]]+RESSALVAS|APPROVED|APROVADO|REJECTED|REPROVADO|BLOCKED|BLOQUEADO)'
VERDICT_LINE_RE="^[[:space:]*_>#\`+-]*${VERDICT_KEY_RE}${VERDICT_DECOR_RE}:${VERDICT_DECOR_RE}${VERDICT_TOKEN_RE}[[:space:]*_\`.,;:)]*$"

if ! grep -Eiq "$VERDICT_LINE_RE" "$report_file"; then
  echo "FALTANDO: veredito canonico (APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED)"
  missing=1
fi

# Secoes obrigatorias (espelham o output minimo da Etapa 6 da skill review)
require_heading "achados"                       "seção Achados"
require_heading "arquivos revisados"            "seção Arquivos Revisados"
require_heading "riscos residuais"              "seção Riscos Residuais"
require_heading "valida"                        "seção Validações Executadas"


verdict_raw="$(grep -Eio "$VERDICT_LINE_RE" "$report_file" | head -1 | grep -Eio "$VERDICT_TOKEN_RE" | head -1 | tr -d '\n' | tr '[:lower:]' '[:upper:]' | tr -s ' \t' '_' || true)"

case "$verdict_raw" in
  "APPROVED_WITH_REMARKS"|"APROVADO_COM_RESSALVAS") verdict_value="APPROVED_WITH_REMARKS" ;;
  "APPROVED"|"APROVADO") verdict_value="APPROVED" ;;
  "REJECTED"|"REPROVADO") verdict_value="REJECTED" ;;
  "BLOCKED"|"BLOQUEADO") verdict_value="BLOCKED" ;;
  *) verdict_value="" ;;
esac

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
  reviewed_files="$(awk '
    tolower($0) ~ /^#+[[:space:]]+arquivos revisados/ { capture=1; next }
    /^#+/ { capture=0 }
    capture { print }
  ' "$report_file" | grep -Eo '[A-Za-z0-9_][A-Za-z0-9_./-]*\.[A-Za-z0-9]+' | sed -E 's#^\./##' | sort -u || true)"

  if [[ -z "$reviewed_files" ]]; then
    echo "FALTANDO: seção 'Arquivos Revisados' sem nenhum arquivo listado (RF-48)"
    missing=1
  fi

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
      "atendido") : ;;
      "nao atendido"|"não atendido")
        if [[ "$verdict_value" == "APPROVED" ]]; then
          echo "FALTANDO: criterio marcado 'nao atendido' proibe APPROVED (RF-50): $line"
          missing=1
        fi
        ;;
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
      ref_file="$(grep -Eo '[[:alnum:]_./-]+:[0-9]+' <<<"$evidence" | head -1 | sed -E 's/:[0-9]+$//' | sed -E 's#^\./##' || true)"
      if ! path_is_reviewed "$ref_file"; then
        echo "FALTANDO: evidencia arquivo:linha referencia arquivo ausente da seção 'Arquivos Revisados' (RF-48): $line"
        missing=1
      fi
    elif [[ "$evidence" == *"->"* ]]; then
      ev_name="$(printf '%s' "${evidence%%->*}" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')"
      ev_record="$(printf '%s' "${evidence#*->}" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')"
      if grep -Eq "$TEST_NAME_RE" <<<"$ev_name"; then
        if ! grep -Eiq "$TEST_RESULT_RE" <<<"$ev_record"; then
          echo "FALTANDO: evidencia de teste sem resultado canonico pass/fail (RF-48): $line"
          missing=1
        fi
      elif ! grep -Eiq "$COMMAND_RE" <<<"$ev_name"; then
        echo "FALTANDO: evidencia nao e comando executavel, referencia arquivo:linha nem resultado de teste (RF-48): $line"
        missing=1
      elif ! substantive_record "$ev_record"; then
        echo "FALTANDO: evidencia nao registra saida significativa do comando — registro trivial nao e prova (RF-48): $line"
        missing=1
      fi
    else
      echo "FALTANDO: linha de evidencia fora das tres formas de RF-48 (comando+saida, arquivo:linha, teste+resultado): $line"
      missing=1
    fi
  done < "$report_file"

  if [[ "$criteria_lines" -eq 0 ]]; then
    echo "FALTANDO: seção 'Mapa de Criterios de Aceite' sem nenhuma linha de criterio ('- ' seguido de marcador entre colchetes)"
    missing=1
  fi

  task_ref="$(grep -Eio '^-[[:space:]]*(task[[:space:]]*file|arquivo da task)[[:space:]]*:[[:space:]]*(.+)$' "$report_file" \
    | head -1 | sed -E 's/^-[[:space:]]*[^:]+:[[:space:]]*//' | sed -E 's/[[:space:]]+$//' || true)"
  task_path=""
  if [[ -n "$task_ref" && "$task_ref" != *"<"* && "$task_ref" != n/a* ]]; then
    if [[ -f "$task_ref" ]]; then
      task_path="$task_ref"
    elif [[ -f "$(dirname "$report_file")/$task_ref" ]]; then
      task_path="$(dirname "$report_file")/$task_ref"
    fi
  fi

  if [[ -z "$task_path" ]]; then
    echo "FALTANDO: task file nao resolvivel para confronto 1:1 do mapa de criterios (RF-51):" \
         "declare '- Task file: <caminho>' apontando para a task revisada."
    missing=1
  else
    task_criteria="$(awk '
      tolower($0) ~ /^#+[[:space:]]+(crit(e|é)rios de (sucesso|aceite)|definition of done|acceptance criteria)/ { capture=1; next }
      /^#+/ { capture=0 }
      capture && /^[[:space:]]*-[[:space:]]+/ {
        item=$0
        sub(/^[[:space:]]*-[[:space:]]+/, "", item)
        sub(/^\[[^]]*\][[:space:]]*/, "", item)
        sub(/^[[:space:]]+/, "", item)
        sub(/[[:space:]]+$/, "", item)
        if (item != "") c++
      }
      END { print c+0 }
    ' "$task_path")"
    if [[ "$task_criteria" -eq 0 ]]; then
      echo "FALTANDO: task file ($task_path) nao declara nenhum criterio de aceite —" \
           "mapa 1:1 nao confrontavel (RF-51)."
      missing=1
    elif [[ "$criteria_lines" -lt "$task_criteria" ]]; then
      echo "FALTANDO: mapa 1:1 incompleto — $criteria_lines linha(s) de criterio no review para" \
           "$task_criteria criterio(s) definido(s) em $task_path (RF-47/RF-51)."
      missing=1
    fi
  fi
fi

if [[ $missing -ne 0 ]]; then
  echo ""
  echo "Validacao do pacote de evidencias de review falhou: $report_file"
  exit 1
fi

echo "Validacao do pacote de evidencias de review aprovada: $report_file"
