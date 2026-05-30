#!/usr/bin/env bash

set -euo pipefail

# Nota: LC_ALL=C removido — padrões de seção contêm chars acentuados (ç, ã)
# que não são correspondidos com LC_ALL=C em UTF-8. Usar locale do sistema.

if [[ $# -ne 1 ]]; then
  echo "Uso: $0 <relatorio-execucao-tarefa.md>"
  exit 2
fi

report_file="$1"

if [[ ! -f "$report_file" ]]; then
  echo "ERRO: arquivo de relatório não encontrado: $report_file"
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

# Contexto carregado (PRD e TechSpec) — exigir como heading Markdown
require_heading "contexto carregado" "seção Contexto Carregado"
require_pattern "PRD[[:space:]]*:" "referência ao PRD consultado"
require_pattern "TechSpec[[:space:]]*:" "referência à TechSpec consultada"

# Seções obrigatórias — exigir como heading Markdown
require_heading "comandos executados" "seção Comandos Executados"
require_heading "arquivos alterados" "seção Arquivos Alterados"
require_heading "resultados de valida" "seção Resultados de Validação"
require_heading "suposi" "seção Suposições"
require_heading "riscos residuais" "seção Riscos Residuais"

# Exigir um estado terminal canônico
if ! grep -Eiq "estado[[:space:]]*:[[:space:]]*(blocked|failed|done)" "$report_file"; then
  echo "FALTANDO: estado terminal de execução (blocked|failed|done)"
  missing=1
fi

# Evidência de testes e lint
require_pattern "testes[[:space:]]*:[[:space:]]*(pass|fail|blocked)" "evidência de testes com resultado"
require_pattern "lint[[:space:]]*:[[:space:]]*(pass|fail|blocked)" "evidência de lint com resultado"

# Prova forte de testes (RF-03): "Testes: pass" exige um comando de teste correspondente
# na seção "## Comandos Executados". Sem comando → prova fraca → falha.
testes_value="$(grep -Eio 'testes[[:space:]]*:[[:space:]]*(pass|fail|blocked)' "$report_file" | head -1 | grep -Eio '(pass|fail|blocked)' | head -1 | tr '[:upper:]' '[:lower:]')"
if [[ "$testes_value" == "pass" ]]; then
  cmds_block="$(awk '
    /^#+[[:space:]]+Comandos Executados/ { capture=1; next }
    /^#+[[:space:]]/ { if (capture) capture=0 }
    capture { print }
  ' "$report_file")"
  if ! printf '%s\n' "$cmds_block" | grep -Eiq '(go test|gotestsum|pytest|unittest|npm (run )?test|yarn test|pnpm test|jest|vitest|mocha|make test|make integration|cargo test|dotnet test|ctest|rspec|phpunit|[^a-z]test[^a-z])'; then
    echo "FALTANDO: 'Testes: pass' declarado sem comando de teste correspondente em '## Comandos Executados' (prova fraca)"
    missing=1
  fi
fi

# Gate de critérios de aceite (RF-01..RF-02): cada critério da task file deve ter comprovação
# no relatório. Resolução do task file via campo "Arquivo:". Task legada sem critérios → aviso não-fatal.
task_file_ref="$(grep -Eio '^-[[:space:]]*Arquivo[[:space:]]*:[[:space:]]*(.+)$' "$report_file" | head -1 | sed -E 's/^-[[:space:]]*Arquivo[[:space:]]*:[[:space:]]*//' | sed -E 's/[[:space:]]+$//')"
task_path=""
if [[ -n "$task_file_ref" && "$task_file_ref" != *"<slug>"* && "$task_file_ref" != n/a* ]]; then
  if [[ -f "$task_file_ref" ]]; then
    task_path="$task_file_ref"
  elif [[ -f "$(dirname "$report_file")/$task_file_ref" ]]; then
    task_path="$(dirname "$report_file")/$task_file_ref"
  fi
fi

if [[ -n "$task_path" ]]; then
  criteria_count="$(awk '
    /^#+[[:space:]]+Crit[eé]rios de (Sucesso|Aceite)/ { capture=1; next }
    /^#+[[:space:]]/ { if (capture) capture=0 }
    capture && /^[[:space:]]*-[[:space:]]+/ { c++ }
    END { print c+0 }
  ' "$task_path")"

  if [[ "$criteria_count" -gt 0 ]]; then
    if ! grep -Eiq "^#+[[:space:]]+crit[eé]rios de aceite" "$report_file"; then
      echo "FALTANDO: seção '## Critérios de Aceite' no relatório (task define $criteria_count critério(s))"
      missing=1
    else
      proven_count="$(awk '
        /^#+[[:space:]]+Crit[eé]rios de Aceite/ { capture=1; next }
        /^#+[[:space:]]/ { if (capture) capture=0 }
        capture && /->[[:space:]]*comprovado[[:space:]]*:/ {
          if ($0 !~ /comprovado[[:space:]]*:[[:space:]]*(\[ev|\[evid|\[\][[:space:]]*$|$)/) p++
        }
        END { print p+0 }
      ' "$report_file")"
      if [[ "$proven_count" -lt "$criteria_count" ]]; then
        echo "FALTANDO: critérios de aceite comprovados ($proven_count) < definidos na task ($criteria_count)"
        missing=1
      fi
    fi
  else
    echo "AVISO: task file ($task_path) sem seção de critérios — gate de aceite ignorado (legado)."
  fi
else
  echo "AVISO: relatório sem referência resolvível a task file (campo 'Arquivo:') — gate de aceite ignorado (legado)."
fi

# Rastreabilidade PRD → teste: se o relatório referencia um PRD com arquivo real (não n/a),
# verificar que pelo menos um ID de requisito (ex: RF-01, RF01, REQ-1, REQ1) aparece no relatório.
prd_line="$(grep -Eio 'PRD[[:space:]]*:[[:space:]]*(.+)' "$report_file" | head -1 | sed 's/^PRD[[:space:]]*:[[:space:]]*//' | tr -d '[:space:]')"
if [[ -n "$prd_line" && "$prd_line" != n/a* && "$prd_line" != "(n/a)"* ]]; then
  if ! grep -Eiq "(RF-?[0-9]+|REQ-?[0-9]+)" "$report_file"; then
    echo "FALTANDO: nenhum ID de requisito (RF-nn ou REQ-nn) referenciado no relatório"
    missing=1
  fi
fi

# Rastreabilidade cruzada: verificar que cada RF-nn/REQ-nn citado no relatório existe no PRD referenciado.
prd_path="$prd_line"
if [[ -n "$prd_path" && "$prd_path" != n/a* && "$prd_path" != "(n/a)"* && -f "$prd_path" ]]; then
  # Extrair IDs do relatório e verificar cada um no PRD
  report_ids="$(grep -Eio '(RF-?[0-9]+|REQ-?[0-9]+)' "$report_file" | sort -u)"
  for req_id in $report_ids; do
    if ! grep -Fiq "$req_id" "$prd_path" 2>/dev/null; then
      echo "FALTANDO: requisito $req_id citado no relatório não encontrado no PRD ($prd_path)"
      missing=1
    fi
  done
elif [[ -n "$prd_path" && "$prd_path" != n/a* && "$prd_path" != "(n/a)"* ]]; then
  # PRD referenciado mas arquivo não encontrado — tentar caminho relativo ao relatório
  report_dir="$(dirname "$report_file")"
  if [[ -f "$report_dir/$prd_path" ]]; then
    report_ids="$(grep -Eio '(RF-?[0-9]+|REQ-?[0-9]+)' "$report_file" | sort -u)"
    for req_id in $report_ids; do
      if ! grep -Fiq "$req_id" "$report_dir/$prd_path" 2>/dev/null; then
        echo "FALTANDO: requisito $req_id citado no relatório não encontrado no PRD ($report_dir/$prd_path)"
        missing=1
      fi
    done
  fi
fi

# Veredito do revisor
if ! grep -Eiq "veredito do revisor[[:space:]]*:[[:space:]]*(APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED)" "$report_file"; then
  echo "FALTANDO: veredito do revisor com valor canônico"
  missing=1
fi

if [[ $missing -ne 0 ]]; then
  echo ""
  echo "Validação do pacote de evidências falhou: $report_file"
  exit 1
fi

echo "Validação do pacote de evidências aprovada: $report_file"
