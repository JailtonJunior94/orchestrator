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

contract_version=1
contract_marker="$(grep -Eio '<!--[[:space:]]*evidence-contract[[:space:]]*:[[:space:]]*v[0-9]+[[:space:]]*-->' "$report_file" | head -1 || true)"
if [[ -n "$contract_marker" ]]; then
  contract_version="$(printf '%s' "$contract_marker" | grep -Eo 'v[0-9]+' | head -1 | tr -d 'v')"
  if [[ "$contract_version" != "2" ]]; then
    echo "FALTANDO: versão de contrato de evidência desconhecida: v$contract_version (suportado: v2, ou ausência do marcador para o histórico v1)"
    missing=1
    contract_version=2
  fi
fi

if [[ "$contract_version" -eq 1 ]]; then
  cut_commit="0d84ccd3291c5eb8b762ec7ce6ac766e311dad17"
  report_dir="$(dirname "$report_file")"
  historical=0
  if git -C "$report_dir" rev-parse --verify --quiet "$cut_commit^{commit}" >/dev/null 2>&1; then
    tracked_path="$(git -C "$report_dir" ls-files --full-name -- "$(basename "$report_file")" 2>/dev/null | head -1 || true)"
    if [[ -n "$tracked_path" ]] && git -C "$report_dir" cat-file blob "$cut_commit:$tracked_path" 2>/dev/null | cmp -s - "$report_file"; then
      historical=1
    fi
  fi
  if [[ "$historical" -eq 0 ]]; then
    echo "FALTANDO: relatório sem marcador de contrato não é evidência histórica — seu conteúdo não corresponde," \
         "byte a byte, ao blob versionado no commit de corte $cut_commit. Trabalho novo (ou relatório histórico" \
         "editado depois do corte) deve declarar '<!-- evidence-contract: v2 -->' e cumprir as regras estritas" \
         "(mapa 1:1 de critérios)."
    missing=1
    contract_version=2
  fi
fi

report_state="$(grep -Eio 'estado[[:space:]]*:[[:space:]]*(blocked|failed|done)' "$report_file" | head -1 | sed -E 's/.*:[[:space:]]*//' | tr '[:upper:]' '[:lower:]' || true)"
report_task_id="$(grep -Eio '^-[[:space:]]*ID[[:space:]]*:[[:space:]]*[^[:space:]]+' "$report_file" | head -1 | sed -E 's/^-[[:space:]]*ID[[:space:]]*:[[:space:]]*//' || true)"
result_path_ref="$(grep -Eio '^result_path[[:space:]]*=[[:space:]]*[^[:space:]]+' "$report_file" | head -1 | sed -E 's/^result_path[[:space:]]*=[[:space:]]*//' || true)"

report_dir_abs="$(cd "$(dirname "$report_file")" && pwd)"
repo_root_abs="$report_dir_abs"
if git -C "$report_dir_abs" rev-parse --show-toplevel >/dev/null 2>&1; then
  repo_root_abs="$(git -C "$report_dir_abs" rev-parse --show-toplevel)"
fi

result_json=""
if [[ -n "$result_path_ref" ]]; then
  result_candidate="${result_path_ref%%#*}"
  if [[ -f "$repo_root_abs/$result_candidate" ]]; then
    result_json="$repo_root_abs/$result_candidate"
  elif [[ -f "$report_dir_abs/$result_candidate" ]]; then
    result_json="$report_dir_abs/$result_candidate"
  elif [[ -f "$result_candidate" ]]; then
    result_json="$result_candidate"
  fi
fi

result_status=""
if [[ -n "$result_json" ]]; then
  result_status="$(python3 -c '
import json
import sys

try:
    payload = json.load(open(sys.argv[1], encoding="utf-8"))
except Exception:
    sys.exit(1)
status = payload.get("status") if isinstance(payload, dict) else None
if not isinstance(status, str):
    sys.exit(1)
print(status)
' "$result_json" 2>/dev/null || true)"
fi

tasks_file="$report_dir_abs/tasks.md"
tasks_status=""
if [[ -n "$report_task_id" && -f "$tasks_file" ]]; then
  tasks_status="$(awk -F'|' -v want="$report_task_id" '
    {
      id = $2
      status = $4
      gsub(/^[ \t]+|[ \t]+$/, "", id)
      gsub(/^[ \t]+|[ \t]+$/, "", status)
    }
    id == want && status != "" { print status; exit }
  ' "$tasks_file" || true)"
fi

if [[ -n "$report_state" && -n "$result_status" && "$report_state" != "$result_status" ]]; then
  echo "FALTANDO: divergencia de estado — relatorio declara 'Estado: $report_state' e o execution-result" \
       "($result_json) declara status=\"$result_status\"; o estado terminal precisa ser o mesmo nos dois artefatos."
  missing=1
fi

if [[ -n "$report_state" && -n "$tasks_status" && "$report_state" != "$tasks_status" ]]; then
  echo "FALTANDO: divergencia de estado — relatorio declara 'Estado: $report_state' e tasks.md ($tasks_file)" \
       "registra '$tasks_status' para a tarefa $report_task_id."
  missing=1
fi

if [[ -n "$result_status" && -n "$tasks_status" && "$result_status" != "$tasks_status" ]]; then
  echo "FALTANDO: divergencia de estado — execution-result ($result_json) declara status=\"$result_status\" e" \
       "tasks.md ($tasks_file) registra '$tasks_status' para a tarefa $report_task_id."
  missing=1
fi

effective_state="$report_state"
if [[ "$result_status" == "done" ]]; then
  effective_state="done"
fi

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
testes_value="$(grep -Eio 'testes[[:space:]]*:[[:space:]]*(pass|fail|blocked)' "$report_file" | head -1 | grep -Eio '(pass|fail|blocked)' | head -1 | tr '[:upper:]' '[:lower:]' || true)"
if [[ "$testes_value" == "pass" ]]; then
  cmds_block="$(awk '
    /^#+[[:space:]]+Comandos Executados/ { capture=1; next }
    /^#+[[:space:]]/ { if (capture) capture=0 }
    capture { print }
  ' "$report_file")"
  if ! printf '%s\n' "$cmds_block" | grep -Eiq '(go test|gotestsum|pytest|unittest|npm (run )?test|yarn test|pnpm test|jest|vitest|mocha|make test|make integration|cargo test|dotnet test|ctest|rspec|phpunit|mvn test|mvn verify|gradle test|gradlew test|[^a-z]test([^a-z]|$))'; then
    echo "FALTANDO: 'Testes: pass' declarado sem comando de teste correspondente em '## Comandos Executados' (prova fraca)"
    missing=1
  fi
fi

# Gate de critérios de aceite (RF-01..RF-02): cada critério da task file deve ter comprovação
# no relatório. Resolução do task file via campo "Arquivo:". Task legada sem critérios → aviso não-fatal.
task_file_ref="$(grep -Eio '^-[[:space:]]*Arquivo[[:space:]]*:[[:space:]]*(.+)$' "$report_file" | head -1 | sed -E 's/^-[[:space:]]*Arquivo[[:space:]]*:[[:space:]]*//' | sed -E 's/[[:space:]]+$//' || true)"
task_path=""
if [[ -n "$task_file_ref" && "$task_file_ref" != *"<slug>"* && "$task_file_ref" != n/a* ]]; then
  if [[ -f "$task_file_ref" ]]; then
    task_path="$task_file_ref"
  elif [[ -f "$(dirname "$report_file")/$task_file_ref" ]]; then
    task_path="$(dirname "$report_file")/$task_file_ref"
  fi
fi

report_has_criteria=0
if grep -Eiq "^#+[[:space:]]+crit(e|é)rios de aceite" "$report_file"; then
  report_has_criteria=1
fi

if [[ "$contract_version" -eq 1 ]]; then
  echo "AVISO: contrato de evidência v1 (histórico) — a isenção cobre somente a forma da evidência." \
       "RF-53: o mapa 1:1 de critérios de aceite e o desfecho (veredito que encerra, RF-33) continuam cobrados."
fi

if [[ "$effective_state" == "done" && -n "$task_path" ]]; then
  criteria_count="$(awk '
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
  elif [[ "$report_has_criteria" -eq 1 ]]; then
    echo "FALTANDO: relatório declara '## Critérios de Aceite' mas a task file ($task_path) não tem" \
         "seção de critérios — mapa 1:1 não confrontável (RF-53); AI_SDD_STRICT_EVIDENCE não reabre este gate."
    missing=1
  else
    echo "FALTANDO: task file ($task_path) não declara nenhum critério de aceite —" \
         "mapa 1:1 não confrontável (RF-53). O gate de aceite e fail-closed desde 0.31.0;" \
         "declare os criterios na task file e comprove-os no relatorio."
    missing=1
  fi
elif [[ "$effective_state" == "done" && "$report_has_criteria" -eq 1 ]]; then
  echo "FALTANDO: relatório declara '## Critérios de Aceite' mas não há task file resolvível para" \
       "confronto 1:1 (RF-53); AI_SDD_STRICT_EVIDENCE não reabre este gate."
  missing=1
elif [[ "$effective_state" == "done" ]]; then
  echo "FALTANDO: relatório declara 'done' mas não há task file resolvível (campo 'Arquivo:') para" \
       "confronto 1:1 dos critérios (RF-51/RF-53); AI_SDD_STRICT_EVIDENCE não reabre este gate."
  missing=1
fi

# Rastreabilidade PRD → teste: se o relatório referencia um PRD com arquivo real (não n/a),
# verificar que pelo menos um ID de requisito (ex: RF-01, RF01, REQ-1, REQ1) aparece no relatório.
prd_line="$(grep -Eio 'PRD[[:space:]]*:[[:space:]]*(.+)' "$report_file" | head -1 | sed 's/^PRD[[:space:]]*:[[:space:]]*//' | tr -d '[:space:]' || true)"
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
  report_ids="$(grep -Eio '(RF-?[0-9]+|REQ-?[0-9]+)' "$report_file" | sort -u || true)"
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
    report_ids="$(grep -Eio '(RF-?[0-9]+|REQ-?[0-9]+)' "$report_file" | sort -u || true)"
    for req_id in $report_ids; do
      if ! grep -Fiq "$req_id" "$report_dir/$prd_path" 2>/dev/null; then
        echo "FALTANDO: requisito $req_id citado no relatório não encontrado no PRD ($report_dir/$prd_path)"
        missing=1
      fi
    done
  fi
fi

AISPEC_FINDING_ANCHOR='(^|[|])[[:space:]]*(([-*+>]|[0-9]+[.)]|#+|\[[ xX]\])[[:space:]]*)*'
AISPEC_FINDING_EMPHASIS='(\*\*|__|`)?'
AISPEC_BLOCKING_SEVERITY_RE="${AISPEC_FINDING_ANCHOR}${AISPEC_FINDING_EMPHASIS}(\[(critical|cr(i|í)tico|high|hard|alta|alto|blocker|security)\]|severidade${AISPEC_FINDING_EMPHASIS}[[:space:]]*:[[:space:]]*${AISPEC_FINDING_EMPHASIS}(critical|high|cr(i|í)tico|alta|alto)|severity${AISPEC_FINDING_EMPHASIS}[[:space:]]*:[[:space:]]*${AISPEC_FINDING_EMPHASIS}(critical|high))"
AISPEC_ANY_SEVERITY_RE="${AISPEC_FINDING_ANCHOR}${AISPEC_FINDING_EMPHASIS}(\[(critical|cr(i|í)tico|high|hard|alta|alto|blocker|security|medium|m(e|é)dia|important|importante|low|baixa|suggestion|sugest(a|ã)o)\]|severidade${AISPEC_FINDING_EMPHASIS}[[:space:]]*:[[:space:]]*${AISPEC_FINDING_EMPHASIS}(critical|high|medium|low|cr(i|í)tico|alta|alto|m(e|é)dia|baixa)|severity${AISPEC_FINDING_EMPHASIS}[[:space:]]*:[[:space:]]*${AISPEC_FINDING_EMPHASIS}(critical|high|medium|low))"

findings_body_file="$(mktemp)"
trap 'rm -f "$findings_body_file"' EXIT
awk '
  /^[[:space:]]*```/ { fenced = !fenced; next }
  !fenced { print }
' "$report_file" >"$findings_body_file"

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
elif [[ "$review_verdict" == "APPROVED_WITH_REMARKS" ]]; then
  if grep -Eiq "$AISPEC_BLOCKING_SEVERITY_RE" "$findings_body_file"; then
    echo "FALTANDO: veredito APPROVED_WITH_REMARKS não encerra com achado high/critical declarado (RF-33)."
    missing=1
  elif ! grep -Eiq "$AISPEC_ANY_SEVERITY_RE" "$findings_body_file"; then
    echo "FALTANDO: veredito APPROVED_WITH_REMARKS sem achado declarado com severidade canônica:" \
         "ausência de high/critical não verificável (RF-33, fail-closed)."
    missing=1
  fi
elif [[ "$review_verdict" != "APPROVED" ]]; then
  echo "FALTANDO: veredito do reviewer não encerra o ciclo de aprovação: $review_verdict (RF-33:" \
       "encerram APPROVED, ou APPROVED_WITH_REMARKS sem achado high/critical)."
  missing=1
fi

review_tool="$(grep -E '^tool=[[:space:]]*(claude|codex|copilot|opencode)[[:space:]]*$' "$report_file" | head -1 | sed -E 's/^tool=[[:space:]]*//; s/[[:space:]]*$//')" || true
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
if [[ "$effective_state" == "done" ]]; then
  if ! python3 - "$report_file" <<'PY'
import hashlib
import json
import os
import re
import subprocess
import sys

MIN_AI_SPEC_VERSION = (2, 0, 0)

FINDING_ANCHOR = (
    r"(?:^|\|)[ \t]*(?:(?:[-*+>]|[0-9]+[.)]|#+|\[[ xX]\])[ \t]*)*(?:\*\*|__|`)?"
)
EMPHASIS = r"(?:\*\*|__|`)?"
BLOCKING_SEVERITY = re.compile(
    FINDING_ANCHOR
    + r"(\[(critical|cr(?:i|\u00ed)tico|high|hard|alta|alto|blocker|security)\]"
    + r"|severidade" + EMPHASIS + r"\s*:\s*" + EMPHASIS + r"(critical|high|cr(?:i|\u00ed)tico|alta|alto)"
    + r"|severity" + EMPHASIS + r"\s*:\s*" + EMPHASIS + r"(critical|high))",
    re.IGNORECASE | re.MULTILINE,
)
ANY_SEVERITY = re.compile(
    FINDING_ANCHOR
    + r"(\[(critical|cr(?:i|\u00ed)tico|high|hard|alta|alto|blocker|security|medium|m(?:e|\u00e9)dia"
    + r"|important|importante|low|baixa|suggestion|sugest(?:a|\u00e3)o)\]"
    + r"|severidade" + EMPHASIS + r"\s*:\s*" + EMPHASIS
    + r"(critical|high|medium|low|cr(?:i|\u00ed)tico|alta|alto|m(?:e|\u00e9)dia|baixa)"
    + r"|severity" + EMPHASIS + r"\s*:\s*" + EMPHASIS + r"(critical|high|medium|low))",
    re.IGNORECASE | re.MULTILINE,
)


def findings_body(raw):
    body = []
    fenced = False
    for line in raw.split("\n"):
        if re.match(r"^[ \t]*```", line):
            fenced = not fenced
            continue
        if not fenced:
            body.append(line)
    return "\n".join(body)


class ToolchainError(Exception):
    pass


def parse_semver(raw):
    found = re.search(r"(\d+)\.(\d+)\.(\d+)", raw)
    if not found:
        return None
    return (int(found.group(1)), int(found.group(2)), int(found.group(3)))


def resolve_validator():
    binary = os.environ.get("AI_SPEC_BIN", "ai-spec")
    minimum = ".".join(str(part) for part in MIN_AI_SPEC_VERSION)
    try:
        probe = subprocess.run(
            [binary, "version"],
            text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            check=False,
        )
    except OSError as error:
        raise ToolchainError(
            f"binario {binary!r} ausente ou nao executavel ({error}); "
            f"exigido ai-spec-harness >= {minimum}. "
            "Construa a partir do HEAD (go build -o ./ai-spec .) e exporte AI_SPEC_BIN."
        ) from error
    output = " ".join(probe.stdout.split())
    if probe.returncode != 0:
        raise ToolchainError(
            f"{binary!r} nao respondeu a 'version' (exit {probe.returncode}): {output}; "
            f"exigido ai-spec-harness >= {minimum}."
        )
    version = parse_semver(output)
    if version is None:
        raise ToolchainError(
            f"nao foi possivel identificar a versao de {binary!r}: {output}; "
            f"exigido ai-spec-harness >= {minimum}."
        )
    if version < MIN_AI_SPEC_VERSION:
        current = ".".join(str(part) for part in version)
        raise ToolchainError(
            f"{binary!r} esta na versao {current}, anterior a {minimum} exigida por este contrato "
            "de evidencia; um binario obsoleto reprova a prova fisica por motivo falso. "
            "Construa a partir do HEAD (go build -o ./ai-spec .) e exporte AI_SPEC_BIN."
        )
    return binary


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
    validator = resolve_validator()
except ToolchainError as error:
    print(f"FALTANDO: toolchain ai-spec incompativel: {error}")
    raise SystemExit(1)

try:
    result_path = contained(match.group(1))
    result = json.load(open(result_path, encoding="utf-8"))
    required = {"schema_version", "run_id", "task_id", "attempt", "status", "base_sha", "patch_sha256", "patch_ref", "final_state_sha256", "tests", "criteria", "evidence", "review_verdict"}
    if result.get("schema_version") != 2:
        raise ValueError(
            f"execution-result malformado: schema_version {result.get('schema_version')!r} diferente de 2"
        )
    absent = sorted(required - set(result))
    if absent:
        raise ValueError(
            f"execution-result v2 malformado: campos obrigatorios ausentes: {', '.join(absent)}"
        )
    status = result.get("status")
    if status != "done":
        declared = re.search(r"(?im)^verdict\s*=\s*(\S+)\s*$", text)
        verdict = declared.group(1).strip().upper() if declared else ""
        closes = verdict == "APPROVED"
        if verdict == "APPROVED_WITH_REMARKS":
            body = findings_body(text)
            closes = bool(ANY_SEVERITY.search(body)) and not BLOCKING_SEVERITY.search(body)
        if closes:
            print(
                f"FALTANDO: veredito aprovador (verdict={verdict}) sobre execution-result nao-done "
                f"(status={status!r}): resultado nao concluido nao pode acompanhar veredito aprovador; "
                "prova fisica so se aplica a resultado done"
            )
            raise SystemExit(1)
        print(
            f"NAO APLICAVEL: prova fisica dispensada — execution-result status={status!r} (nao-done) "
            f"com veredito {verdict or 'ausente'} nao aprovador; a reprovacao cabe ao gate de veredito"
        )
        raise SystemExit(0)
    task = re.search(r"(?im)^-\s*ID\s*:\s*(\S+)\s*$", text)
    patch = re.search(r"(?im)^sha\s*=\s*([0-9a-f]{64})\s*$", text)
    if not task or task.group(1) != result["task_id"]:
        raise ValueError("task_id diverge do relatorio")
    if not patch or patch.group(1).lower() != result["patch_sha256"].lower():
        raise ValueError("patch_sha256 diverge do Diff Reviewed")
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
