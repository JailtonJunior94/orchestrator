#!/usr/bin/env bash
# Valida o pacote de evidencias de um relatorio de bugfix.
# Uso: $0 [--rf <RF-ID>] [--no-rf] <bugfix_report.md>
#
# Opcoes:
#   --rf <RF-ID>  Verifica se o RF/requisito informado e mencionado no relatorio (rastreabilidade).
#                 Pode ser repetido para multiplos IDs: --rf RF-01 --rf RF-02
#   --no-rf       Desabilita a checagem default-on de rastreabilidade de origem (escape hatch).
#
# Por padrao (default-on), o validador exige que o relatorio comprove a origem do bug
# (campo "Origem:" referenciando RF, task, finding de review ou issue). Use --no-rf para opt-out.
#
# Exit 0 = aprovado, Exit 1 = reprovado, Exit 2 = uso incorreto.

set -euo pipefail

export LC_ALL=C

rf_ids=()
check_traceability=1

while [[ $# -gt 0 ]]; do
  case "$1" in
    --rf)
      if [[ $# -lt 2 ]]; then
        echo "ERRO: --rf requer um argumento (ex: --rf RF-01)"
        exit 2
      fi
      rf_ids+=("$2")
      shift 2
      ;;
    --no-rf)
      check_traceability=0
      shift
      ;;
    -*)
      echo "Opcao desconhecida: $1"
      echo "Uso: $0 [--rf <RF-ID>]... [--no-rf] <bugfix_report.md>"
      exit 2
      ;;
    *)
      break
      ;;
  esac
done

if [[ $# -ne 1 ]]; then
  echo "Uso: $0 [--rf <RF-ID>]... <bugfix_report.md>"
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
require_heading "bugs"                  "seção Bugs"
require_heading "comandos executados"   "seção Comandos Executados"
require_heading "riscos residuais"      "seção Riscos Residuais"

# Cada entrada de bug deve ter estado canonico
require_pattern "Estado[[:space:]]*:[[:space:]]*(fixed|blocked|skipped|failed)" \
  "estado canonico de bug (fixed|blocked|skipped|failed)"

# Causa raiz documentada
require_pattern "Causa[[:space:]]+raiz[[:space:]]*:" "campo Causa raiz"

# Teste de regressao documentado
require_pattern "Teste[[:space:]]+de[[:space:]]+regress" "referencia a teste de regressao"

# Evidencia de validacao
require_pattern "Validac" "campo Validacao"

# Totalizadores
require_pattern "Corrigidos[[:space:]]*:" "contagem de bugs corrigidos"

# Estado terminal canonico
if ! grep -Eiq "^[-*]?[[:space:]]*(Estado|estado|Estado final)[[:space:]]*:[[:space:]]*(done|blocked|failed|needs_input)" "$report_file"; then
  echo "FALTANDO: estado terminal canonico (done|blocked|failed|needs_input)"
  missing=1
fi

# Rastreabilidade de origem default-on: cada bug deve declarar de onde veio
# (RF, task, finding de review ou issue). Use --no-rf para opt-out.
if [[ "$check_traceability" -eq 1 ]]; then
  if ! grep -Eiq "^[-*]?[[:space:]]*Origem[[:space:]]*:[[:space:]]*\S" "$report_file" \
     && ! grep -Eiq "(RF-[0-9]|task[- ][0-9]|finding|issue[ -]#?[0-9])" "$report_file"; then
    echo "FALTANDO: rastreabilidade de origem (campo 'Origem:' com RF/task/finding/issue) — use --no-rf para opt-out"
    missing=1
  fi
fi

# Rastreabilidade RF: cada ID informado via --rf deve aparecer no relatorio
for rf_id in "${rf_ids[@]+"${rf_ids[@]}"}"; do
  if ! grep -Fiq "$rf_id" "$report_file"; then
    echo "FALTANDO: rastreabilidade RF '$rf_id' nao encontrada no relatorio"
    missing=1
  fi
done

# Validação estrutural por bug e reconciliação dos totalizadores. Uma ocorrência
# global não pode servir como prova para múltiplos bugs.
if ! python3 - "$report_file" <<'PY'
import re
import sys

text = open(sys.argv[1], encoding="utf-8").read()
bugs_section = re.search(r"(?ims)^##\s+Bugs\s*$\n(.*?)(?=^##\s+Comandos Executados\s*$)", text)
if not bugs_section:
    print("FALTANDO: blocos individuais de bugs")
    raise SystemExit(1)
blocks = re.split(r"(?im)(?=^-\s*ID\s*:)", bugs_section.group(1))
blocks = [block for block in blocks if re.match(r"(?im)^-\s*ID\s*:\s*\S+", block)]
if not blocks:
    print("FALTANDO: nenhum bloco '- ID:' encontrado")
    raise SystemExit(1)

failed = False
states = []
tests = 0
required = ("Severidade", "Origem", "Estado", "Causa raiz", "Arquivos alterados", "Teste de regressao", "Validacao")
for block in blocks:
    bug_id = re.search(r"(?im)^-\s*ID\s*:\s*(\S+)", block).group(1)
    for field in required:
        match = re.search(rf"(?im)^-\s*{field}\s*:\s*(.+)$", block)
        if not match or not match.group(1).strip():
            print(f"FALTANDO: {field} no bloco {bug_id}")
            failed = True
    state = re.search(r"(?im)^-\s*Estado\s*:\s*(fixed|blocked|skipped|failed)\s*$", block)
    states.append(state.group(1).lower() if state else "invalid")
    test = re.search(r"(?im)^-\s*Teste de regressao\s*:\s*(.+)$", block)
    if test and test.group(1).strip():
        tests += 1

def total(label):
    match = re.search(rf"(?im)^-\s*{label}\s*:\s*(\d+)\s*$", text)
    return int(match.group(1)) if match else None

declared_total = total("Total de bugs no escopo")
declared_fixed = total("Corrigidos")
declared_tests = total("Testes de regressao adicionados")
actual_fixed = states.count("fixed")
for label, declared, actual in (
    ("Total de bugs no escopo", declared_total, len(blocks)),
    ("Corrigidos", declared_fixed, actual_fixed),
    ("Testes de regressao adicionados", declared_tests, tests),
):
    if declared != actual:
        print(f"FALTANDO: totalizador {label}={declared!r} diverge dos blocos ({actual})")
        failed = True

pending = len(blocks) - actual_fixed
pending_line = re.search(r"(?im)^-\s*Pendentes\s*:\s*(.+)$", text)
if not pending_line or (pending == 0 and pending_line.group(1).strip().lower() != "nenhum") or (pending > 0 and pending_line.group(1).strip().lower() == "nenhum"):
    print(f"FALTANDO: Pendentes diverge dos blocos ({pending})")
    failed = True
raise SystemExit(1 if failed else 0)
PY
then
  missing=1
fi

if [[ $missing -ne 0 ]]; then
  echo ""
  echo "Validacao do pacote de evidencias de bugfix falhou: $report_file"
  exit 1
fi

echo "Validacao do pacote de evidencias de bugfix aprovada: $report_file"
