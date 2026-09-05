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

# Modo estrito (NFR-01): fail-closed nos escapes de legado do gate de aceite.
# Default preserva o comportamento warning-only da janela de compatibilidade.
strict_evidence="${AI_SDD_STRICT_EVIDENCE:-0}"

# _STRICT_DEFAULT_VERSION e a versao em que o modo estrito passa a ser o padrao.
# NFR-01 concede warning-only por duas versoes menores; o fluxo SDD entrou em
# 0.29.0, entao a janela cobre 0.29 e 0.30 e fecha em 0.31.0.
_STRICT_DEFAULT_VERSION="0.31.0"

# legacy_escape emite aviso na janela de compatibilidade e falha em modo estrito.
# O aviso anuncia a versao do flip: BUG-127 mostrou que um escape silencioso e
# indistinguivel de um gate aprovando, entao quem depende do legado precisa ver
# o prazo em toda execucao, nao so no CHANGELOG.
legacy_escape() {
  local reason="$1"
  if [[ "$strict_evidence" == "1" ]]; then
    echo "FALTANDO: $reason (modo estrito: AI_SDD_STRICT_EVIDENCE=1)"
    missing=1
  else
    echo "AVISO: $reason — gate de aceite ignorado (legado)."
    echo "AVISO: este escape sera removido em v$_STRICT_DEFAULT_VERSION, quando o modo estrito" \
         "passa a ser o padrao. Valide agora com AI_SDD_STRICT_EVIDENCE=1."
  fi
}

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
    /^#+[[:space:]]+Crit(e|é)rios de (Sucesso|Aceite)/ { capture=1; next }
    /^#+[[:space:]]/ { if (capture) capture=0 }
    capture && /^[[:space:]]*-[[:space:]]+/ { c++ }
    END { print c+0 }
  ' "$task_path")"

  if [[ "$criteria_count" -gt 0 ]]; then
    if ! grep -Eiq "^#+[[:space:]]+crit(e|é)rios de aceite" "$report_file"; then
      echo "FALTANDO: seção '## Critérios de Aceite' no relatório (task define $criteria_count critério(s))"
      missing=1
    else
      proven_count="$(awk '
        /^#+[[:space:]]+Crit(e|é)rios de Aceite/ { capture=1; next }
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
    legacy_escape "task file ($task_path) sem seção de critérios"
  fi
else
  legacy_escape "relatório sem referência resolvível a task file (campo 'Arquivo:')"
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

# Contrato mínimo do diff revisado (RF-03). Estes valores não podem ser apenas
# declarados no texto: precisam estar no bloco estruturado produzido pelo executor.
diff_sha="$(grep -E '^sha=[[:space:]]*[0-9a-fA-F]{40}([0-9a-fA-F]{24})?[[:space:]]*$' "$report_file" | head -1 | sed -E 's/^sha=[[:space:]]*//; s/[[:space:]]*$//')" || true
if [[ -z "$diff_sha" ]]; then
  echo "FALTANDO: missing diff sha imutável (sha= deve ter 40 ou 64 hexadecimais)"
  missing=1
fi

review_verdict="$(grep -E '^verdict=[[:space:]]*(APPROVED|APPROVED_WITH_REMARKS|REJECTED|BLOCKED)[[:space:]]*$' "$report_file" | head -1 | sed -E 's/^verdict=[[:space:]]*//; s/[[:space:]]*$//')" || true
if [[ -z "$review_verdict" ]]; then
  echo "FALTANDO: veredito do reviewer no bloco Diff Reviewed"
  missing=1
elif [[ "$review_verdict" != "APPROVED" && "$review_verdict" != "APPROVED_WITH_REMARKS" ]]; then
  echo "FALTANDO: veredito do reviewer não aprova execução: $review_verdict"
  missing=1
fi

review_tool="$(grep -E '^tool=[[:space:]]*(claude|codex|gemini|copilot)[[:space:]]*$' "$report_file" | head -1 | sed -E 's/^tool=[[:space:]]*//; s/[[:space:]]*$//')" || true
if [[ -z "$review_tool" ]]; then
  echo "FALTANDO: tool não canônica ou ausente no bloco Diff Reviewed"
  missing=1
fi

# Cobertura não pode regredir. A ausência da métrica falha fechada.
coverage_delta="$(grep -E '^delta=[[:space:]]*[+-]?[0-9]+([.][0-9]+)?%[[:space:]]*$' "$report_file" | head -1 | sed -E 's/^delta=[[:space:]]*//; s/%[[:space:]]*$//')" || true
if [[ -z "$coverage_delta" ]]; then
  echo "FALTANDO: coverage delta ausente ou inválido"
  missing=1
elif awk -v delta="$coverage_delta" 'BEGIN { exit !(delta < 0) }'; then
  printf 'FALTANDO: coverage regression detectada (delta=%s%%)\n' "$coverage_delta"
  missing=1
fi

# Estado done exige o resultado JSON v2 e evidências físicas contidas. O digest
# de cada teste deve corresponder ao conteúdo de pelo menos um arquivo declarado.
if grep -Eiq "estado[[:space:]]*:[[:space:]]*done" "$report_file"; then
  if ! python3 - "$report_file" <<'PY'
import hashlib
import json
import os
import re
import subprocess
import sys

report = os.path.realpath(sys.argv[1])
text = open(report, encoding="utf-8").read()
match = re.search(r"(?im)^result_path\s*=\s*(\S+)\s*$", text)
if not match:
    print("FALTANDO: result_path do execution-result.json")
    raise SystemExit(1)

try:
    root = subprocess.check_output(
        ["git", "-C", os.path.dirname(report), "rev-parse", "--show-toplevel"],
        text=True, stderr=subprocess.DEVNULL,
    ).strip()
except (OSError, subprocess.CalledProcessError):
    root = os.path.dirname(report)
root = os.path.realpath(root)

def contained(reference):
    relative = reference.split("#", 1)[0]
    if not relative or os.path.isabs(relative):
        raise ValueError(f"referencia nao relativa: {reference}")
    path = os.path.realpath(os.path.join(root, relative))
    if os.path.commonpath([root, path]) != root or not os.path.isfile(path):
        raise ValueError(f"evidencia ausente ou fora do repositorio: {reference}")
    return path

try:
    result_path = contained(match.group(1))
    result = json.load(open(result_path, encoding="utf-8"))
    required = {"schema_version", "run_id", "task_id", "attempt", "status", "base_sha", "patch_sha256", "patch_ref", "final_state_sha256", "tests", "criteria", "evidence", "review_verdict"}
    if result.get("schema_version") != 2 or result.get("status") != "done" or not required.issubset(result):
        raise ValueError("execution-result v2 done incompleto")
    task = re.search(r"(?im)^-\s*ID\s*:\s*(\S+)\s*$", text)
    patch = re.search(r"(?im)^sha\s*=\s*([0-9a-f]{64})\s*$", text)
    if not task or task.group(1) != result["task_id"]:
        raise ValueError("task_id diverge do relatorio")
    if not patch or patch.group(1).lower() != result["patch_sha256"].lower():
        raise ValueError("patch_sha256 diverge do Diff Reviewed")
    validator = os.environ.get("AI_SPEC_BIN", "ai-spec")
    validation = subprocess.run(
        [validator, "validate-result", "execution", result_path,
         "--task-id", result["task_id"], "--verify-physical",
         "--prd-dir", os.path.dirname(report), "--exclude", report],
        text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
        check=False,
    )
    if validation.returncode != 0:
        detail = " ".join(validation.stdout.split())
        raise ValueError(f"snapshot fisico canonico invalido: {detail}")
    digests = {}
    for reference in result["evidence"]:
        path = contained(reference)
        digests[reference.split("#", 1)[0].replace("\\", "/")] = hashlib.sha256(open(path, "rb").read()).hexdigest()
    for criterion in result["criteria"]:
        reference = criterion["evidence_ref"].split("#", 1)[0].replace("\\", "/")
        if reference not in digests:
            raise ValueError(f"criterio {criterion.get('id')} sem evidencia declarada")
    for proof in result["tests"]:
        if proof.get("exit_code") != 0 or proof.get("output_sha256") not in digests.values():
            raise ValueError(f"teste sem log fisico correspondente: {proof.get('command')}")
except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as error:
    print(f"FALTANDO: prova fisica invalida: {error}")
    raise SystemExit(1)
PY
  then
    missing=1
  fi
fi

if [[ $missing -ne 0 ]]; then
  echo ""
  echo "Validação do pacote de evidências falhou: $report_file"
  exit 1
fi

echo "Validação do pacote de evidências aprovada: $report_file"
