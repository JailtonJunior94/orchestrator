#!/usr/bin/env bash

set -euo pipefail

SCRIPT="${1:-.agents/scripts/validate-task-evidence.sh}"
TMP_ROOT=$(mktemp -d)
TMPDIR_BASE="$TMP_ROOT/repository"
trap 'rm -rf "$TMP_ROOT"' EXIT

mkdir -p "$TMPDIR_BASE"
git -C "$TMPDIR_BASE" init -q
git -C "$TMPDIR_BASE" config user.email "evidence-test@example.invalid"
git -C "$TMPDIR_BASE" config user.name "Evidence Test"
git -C "$TMPDIR_BASE" config commit.gpgsign false
printf 'base\n' >"$TMPDIR_BASE/tracked.txt"
# Task file real, rastreado no baseline: desde 0.31.0 o gate de aceite e
# fail-closed, entao a fixture valida precisa declarar criterios e o relatorio
# precisa comprova-los. Precisa existir antes do commit base para nao entrar no
# patch como arquivo novo.
mkdir -p "$TMPDIR_BASE/.specs/prd-portability-parity"
printf '# Tarefa 5.0\n\n## Critérios de Sucesso\n\n- Evidência é validada.\n' \
  >"$TMPDIR_BASE/.specs/prd-portability-parity/task-5.0.md"
git -C "$TMPDIR_BASE" add tracked.txt .specs
git -C "$TMPDIR_BASE" commit -qm "test: baseline"
printf 'estado final\n' >"$TMPDIR_BASE/tracked.txt"

go build -o "$TMP_ROOT/ai-spec" .
export AI_SPEC_BIN="$TMP_ROOT/ai-spec"

PASS=0
FAIL=0

mkdir -p "$TMPDIR_BASE/evidence"
git -C "$TMPDIR_BASE" diff --binary HEAD -- . >"$TMPDIR_BASE/evidence/patch.diff"
PATCH_SHA="$(shasum -a 256 "$TMPDIR_BASE/evidence/patch.diff" | awk '{print $1}')"
BASE_SHA="$(git -C "$TMPDIR_BASE" rev-parse HEAD)"
FINAL_STATE_SHA="$( { printf '%s\n' "$BASE_SHA"; cat "$TMPDIR_BASE/evidence/patch.diff"; } | shasum -a 256 | awk '{print $1}')"
printf 'PASS\n' >"$TMPDIR_BASE/evidence/test.log"
TEST_SHA="$(shasum -a 256 "$TMPDIR_BASE/evidence/test.log" | awk '{print $1}')"
cat >"$TMPDIR_BASE/result.json" <<EOF
{"schema_version":2,"run_id":"test","task_id":"5.0","attempt":1,"status":"done","base_sha":"$BASE_SHA","patch_sha256":"$PATCH_SHA","patch_ref":"evidence/patch.diff","final_state_sha256":"$FINAL_STATE_SHA","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"$TEST_SHA"}],"criteria":[{"id":"AC-1","evidence_ref":"evidence/test.log#pass"}],"evidence":["evidence/test.log"],"review_verdict":"approved"}
EOF
VALID_RESULT="$(cat "$TMPDIR_BASE/result.json")"

# run_case_raw escreve a fixture exatamente como recebida. Usar para exercitar o
# contrato de evidencia (marcador ausente/invalido e a regra de corte).
run_case_raw() {
  local label="$1"
  local content="$2"
  local want_exit="$3"
  local want_text="$4"

  local f="$TMPDIR_BASE/report_$PASS$FAIL.md"
  printf '%s' "$content" > "$f"

  local actual_exit=0
  local actual_out
  actual_out=$(bash "$SCRIPT" "$f" 2>&1) || actual_exit=$?
  rm -f "$f"

  if [[ "$actual_exit" -ne "$want_exit" ]]; then
    echo "FAIL [$label]: exit=$actual_exit, want=$want_exit"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi

  if [[ -n "$want_text" ]] && ! echo "$actual_out" | grep -qi "$want_text"; then
    echo "FAIL [$label]: output não contém '$want_text'"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi

  echo "PASS [$label]"
  PASS=$((PASS+1))
}

# run_case trata a fixture como relatorio novo: declara o contrato v2 e cobra as
# regras estritas. Relatorio novo sem marcador e trabalho novo tentando passar
# como historico, e a regra de corte reprova — coberto por TC18.
run_case() {
  local label="$1"
  local content="$2"
  local want_exit="$3"      # 0 = aprovado, 1 = falhou
  local want_text="$4"      # substring esperada no output

  local f="$TMPDIR_BASE/report_$PASS$FAIL.md"
  printf '<!-- evidence-contract: v2 -->\n%s' "$content" > "$f"

  local actual_exit=0
  local actual_out
  actual_out=$(bash "$SCRIPT" "$f" 2>&1) || actual_exit=$?
  rm -f "$f"

  if [[ "$actual_exit" -ne "$want_exit" ]]; then
    echo "FAIL [$label]: exit=$actual_exit, want=$want_exit"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi

  if [[ -n "$want_text" ]] && ! echo "$actual_out" | grep -qi "$want_text"; then
    echo "FAIL [$label]: output não contém '$want_text'"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi

  echo "PASS [$label]"
  PASS=$((PASS+1))
}

# ── Relatório mínimo válido ──────────────────────────────────────────────────
VALID_REPORT='# Relatório de Execução de Tarefa

## Tarefa
- ID: 5.0
- Arquivo: .specs/prd-portability-parity/task-5.0.md
- Estado: done

## Contexto Carregado
- PRD: (n/a)
- TechSpec: (n/a)
- Governança: go-implementation

## Comandos Executados
- make test -> pass

## Arquivos Alterados
- internal/taskloop/evidence.go

## Resultados de Validação
- Testes: pass
- Lint: pass
- Veredito do Revisor: APPROVED

## Diff Reviewed

sha=__PATCH_SHA__
verdict=APPROVED
tool=claude

## Execution Result

result_path=result.json

## Coverage

package=internal/taskloop
delta=+0.5%

## Critérios de Aceite
- Evidência é validada -> comprovado: saída de make test mostra pass

## Suposições
- Nenhuma.

## Riscos Residuais
- Nenhum.
'
VALID_REPORT="${VALID_REPORT//__PATCH_SHA__/$PATCH_SHA}"

run_case "TC1-valido" "$VALID_REPORT" 0 "aprovada"

# ── Sem SHA ──────────────────────────────────────────────────────────────────
NO_SHA_REPORT="${VALID_REPORT/sha=$PATCH_SHA/sha=INVALIDO}"
run_case "TC2-sem-sha" "$NO_SHA_REPORT" 1 "missing diff sha"

# ── Sem verdict ──────────────────────────────────────────────────────────────
NO_VERDICT="${VALID_REPORT/verdict=APPROVED
tool=claude/tool=claude}"
run_case "TC3-sem-verdict" "$NO_VERDICT" 1 "veredito do reviewer"

# ── Tool inválida ────────────────────────────────────────────────────────────
BAD_TOOL="${VALID_REPORT/tool=claude/tool=vscode}"
run_case "TC4-tool-invalida" "$BAD_TOOL" 1 "tool não canônica"

# ── Delta -3.0% (coverage regression) ───────────────────────────────────────
REGRESS_DELTA="${VALID_REPORT/delta=+0.5%/delta=-3.0%}"
run_case "TC5-delta-regressao" "$REGRESS_DELTA" 1 "coverage regression"

# ── Delta +0.5% (deve passar) ───────────────────────────────────────────────
run_case "TC6-delta-ok" "$VALID_REPORT" 0 "aprovada"

MISSING_RESULT="${VALID_REPORT/result_path=result.json/result_path=missing.json}"
run_case "TC7-result-ausente" "$MISSING_RESULT" 1 "prova fisica invalida"

BAD_PATCH_RESULT="${VALID_REPORT/sha=$PATCH_SHA/sha=cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc}"
run_case "TC8-patch-divergente" "$BAD_PATCH_RESULT" 1 "patch_sha256 diverge"

printf 'patch adulterado\n' >"$TMPDIR_BASE/evidence/patch.diff"
run_case "TC9-patch-fisico-divergente" "$VALID_REPORT" 1 "snapshot fisico canonico"

FAKE_PATCH_SHA="$(shasum -a 256 "$TMPDIR_BASE/evidence/patch.diff" | awk '{print $1}')"
printf '%s\n' "${VALID_RESULT//$PATCH_SHA/$FAKE_PATCH_SHA}" >"$TMPDIR_BASE/result.json"
SELF_CONSISTENT_REPORT="${VALID_REPORT//$PATCH_SHA/$FAKE_PATCH_SHA}"
run_case "TC10-patch-arbitrario-autoconsistente" "$SELF_CONSISTENT_REPORT" 1 "estado final recomputado"

git -C "$TMPDIR_BASE" diff --binary HEAD -- . >"$TMPDIR_BASE/evidence/patch.diff"
printf '%s\n' "${VALID_RESULT//$FINAL_STATE_SHA/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb}" >"$TMPDIR_BASE/result.json"
run_case "TC11-estado-final-inventado" "$VALID_REPORT" 1 "estado final recomputado"

BLOCKED_REPORT='## Relatório de Execução de Tarefa

## Tarefa
- ID: 5.0
- Arquivo: .specs/prd-portability-parity/task-5.0.md
- Estado: blocked

## Contexto Carregado
- PRD: (n/a)
- TechSpec: (n/a)

## Comandos Executados
- go test ./... -> pass

## Arquivos Alterados
- internal/taskloop/evidence.go

## Resultados de Validação
- Testes: blocked
- Lint: pass
- Veredito do Revisor: APPROVED

## Diff Reviewed

sha=0123456789012345678901234567890123456789012345678901234567890123
verdict=APPROVED
tool=claude

## Coverage

delta=+0.5%

## Critérios de Aceite
- Bloqueado aguardando evidência externa.

## Suposições
- Nenhuma.

## Riscos Residuais
- A evidência live não foi observada.
'
run_case "TC12-blocked-sem-prova-fisica" "$BLOCKED_REPORT" 0 "aprovada"

# ── RF-33: APPROVED_WITH_REMARKS nao encerra o ciclo (BUG-D2) ───────────────
REMARKS_REPORT="${BLOCKED_REPORT//verdict=APPROVED/verdict=APPROVED_WITH_REMARKS}"
REMARKS_REPORT="${REMARKS_REPORT//Veredito do Revisor: APPROVED/Veredito do Revisor: APPROVED_WITH_REMARKS}"
run_case "TC13-approved-with-remarks-nao-encerra" "$REMARKS_REPORT" 1 "não encerra o ciclo de aprovação"

# ── RF-51/RF-53: done sem task file resolvivel falha incondicionalmente (BUG-D11) ──
NO_TASK_FILE_REPORT='# Relatório de Execução de Tarefa

## Tarefa
- ID: 5.0
- Arquivo: (n/a)
- Estado: done

## Contexto Carregado
- PRD: (n/a)
- TechSpec: (n/a)

## Comandos Executados
- make test -> pass

## Arquivos Alterados
- internal/taskloop/evidence.go

## Resultados de Validação
- Testes: pass
- Lint: pass
- Veredito do Revisor: APPROVED

## Diff Reviewed

sha=0123456789012345678901234567890123456789012345678901234567890123
verdict=APPROVED
tool=claude

## Coverage

package=internal/taskloop
delta=+0.5%

## Suposições
- Nenhuma.

## Riscos Residuais
- Nenhum.
'

run_case "TC14-done-sem-task-file-strict" "$NO_TASK_FILE_REPORT" 1 "não há task file resolvível"

AI_SDD_STRICT_EVIDENCE=0 \
  run_case "TC15-done-sem-task-file-optout" "$NO_TASK_FILE_REPORT" 1 "não há task file resolvível"

printf '# Tarefa 6.0\n\n## Escopo\n\n- Sem criterios declarados.\n' \
  >"$TMPDIR_BASE/.specs/prd-portability-parity/task-6.0.md"

NO_CRITERIA_REPORT='# Relatório de Execução de Tarefa

## Tarefa
- ID: 6.0
- Arquivo: .specs/prd-portability-parity/task-6.0.md
- Estado: done

## Contexto Carregado
- PRD: (n/a)
- TechSpec: (n/a)

## Comandos Executados
- make test -> pass

## Arquivos Alterados
- internal/taskloop/evidence.go

## Resultados de Validação
- Testes: pass
- Lint: pass
- Veredito do Revisor: APPROVED

## Diff Reviewed

sha=0123456789012345678901234567890123456789012345678901234567890123
verdict=APPROVED
tool=claude

## Coverage

package=internal/taskloop
delta=+0.5%

## Suposições
- Nenhuma.

## Riscos Residuais
- Nenhum.
'

run_case "TC16-task-sem-criterios-strict" "$NO_CRITERIA_REPORT" 1 "não declara nenhum critério de aceite"

AI_SDD_STRICT_EVIDENCE=0 \
  run_case "TC17-task-sem-criterios-optout" "$NO_CRITERIA_REPORT" 1 "não declara nenhum critério de aceite"

# ── Contrato de evidencia (BUG-X2 / RF-04 / RF-56) ───────────────────────────

# TC18: relatorio novo (nao existe no ref de corte) sem marcador nao pode se passar
# por historico. A isencao v1 nao e rota de fuga para trabalho novo.
run_case_raw "TC18-v1-novo-reprova" "$VALID_REPORT" 1 "não é evidência histórica"

# TC19: marcador de versao desconhecida reprova fechado.
run_case_raw "TC19-contrato-desconhecido" "<!-- evidence-contract: v9 -->
$VALID_REPORT" 1 "versão de contrato de evidência desconhecida"

HISTORICAL="$TMPDIR_BASE/.specs/prd-portability-parity/9.9_execution_report.md"
HISTORICAL_BODY="$BLOCKED_REPORT"
HISTORICAL_BODY="${HISTORICAL_BODY/## Critérios de Aceite
- Bloqueado aguardando evidência externa.
/}"
REMARKS_HISTORICAL="$TMPDIR_BASE/.specs/prd-portability-parity/9.8_execution_report.md"
printf '%s' "$HISTORICAL_BODY" >"$HISTORICAL"
printf '%s' "${HISTORICAL_BODY//verdict=APPROVED/verdict=APPROVED_WITH_REMARKS}" >"$REMARKS_HISTORICAL"
git -C "$TMPDIR_BASE" add .specs >/dev/null 2>&1
git -C "$TMPDIR_BASE" commit -qm "test: historical evidence" >/dev/null 2>&1
historical_exit=0
historical_out=$(bash "$SCRIPT" "$HISTORICAL" 2>&1) || historical_exit=$?
if [[ "$historical_exit" -eq 0 ]] && grep -qi "isenção cobre somente a forma da evidência" <<<"$historical_out"; then
  echo "PASS [TC20-v1-isenta-forma]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC20-v1-isenta-forma]: exit=$historical_exit"
  echo "  output: $historical_out"
  FAIL=$((FAIL+1))
fi

remarks_exit=0
remarks_out=$(bash "$SCRIPT" "$REMARKS_HISTORICAL" 2>&1) || remarks_exit=$?
if [[ "$remarks_exit" -eq 1 ]] && grep -qi "não encerra o ciclo de aprovação" <<<"$remarks_out"; then
  echo "PASS [TC20b-v1-nao-isenta-desfecho]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC20b-v1-nao-isenta-desfecho]: exit=$remarks_exit"
  echo "  output: $remarks_out"
  FAIL=$((FAIL+1))
fi
rm -f "$REMARKS_HISTORICAL"

# TC21: o mesmo relatorio historico, agora declarando v2, passa a ser cobrado pelas
# regras estritas e reprova no veredito.
printf '<!-- evidence-contract: v2 -->\n%s' "${HISTORICAL_BODY//verdict=APPROVED/verdict=APPROVED_WITH_REMARKS}" >"$HISTORICAL"
strict_exit=0
strict_out=$(bash "$SCRIPT" "$HISTORICAL" 2>&1) || strict_exit=$?
if [[ "$strict_exit" -eq 1 ]] && grep -qi "não encerra o ciclo de aprovação" <<<"$strict_out"; then
  echo "PASS [TC21-v2-estrito-cobra]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC21-v2-estrito-cobra]: exit=$strict_exit"
  echo "  output: $strict_out"
  FAIL=$((FAIL+1))
fi
rm -f "$HISTORICAL"

NON_GIT_DIR="$TMP_ROOT/sem-git"
mkdir -p "$NON_GIT_DIR"
printf '%s' "$VALID_REPORT" >"$NON_GIT_DIR/report.md"
nongit_exit=0
nongit_out=$(bash "$SCRIPT" "$NON_GIT_DIR/report.md" 2>&1) || nongit_exit=$?
if [[ "$nongit_exit" -eq 1 ]] && grep -qi "não é evidência histórica" <<<"$nongit_out"; then
  echo "PASS [TC23-sem-git-nao-isenta]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC23-sem-git-nao-isenta]: exit=$nongit_exit"
  echo "  output: $nongit_out"
  FAIL=$((FAIL+1))
fi

printf '%s' "$HISTORICAL_BODY" >"$HISTORICAL"
git -C "$TMPDIR_BASE" add .specs >/dev/null 2>&1
git -C "$TMPDIR_BASE" commit -qm "test: cut ref fixture" >/dev/null 2>&1
OLD_REF="$(git -C "$TMPDIR_BASE" rev-parse HEAD~1)"
env_exit=0
env_out=$(AI_EVIDENCE_CONTRACT_CUT_REF="$OLD_REF" bash "$SCRIPT" "$HISTORICAL" 2>&1) || env_exit=$?
if [[ "$env_exit" -eq 0 ]] && ! grep -qi "não é evidência histórica" <<<"$env_out"; then
  echo "PASS [TC24-cut-ref-ignora-env]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC24-cut-ref-ignora-env]: exit=$env_exit"
  echo "  output: $env_out"
  FAIL=$((FAIL+1))
fi
rm -f "$HISTORICAL"

# ── BUG-X7: ramo fail-closed alcancavel sem a linha "- Arquivo:" ─────────────
# Sob `set -euo pipefail` a extracao do task file matava o script, pulando este e
# todos os gates a jusante em silencio.
run_case "TC22-sem-linha-arquivo-alcanca-fail-closed" "${VALID_REPORT/- Arquivo: .specs\/prd-portability-parity\/task-5.0.md/}" 1 "não há task file resolvível"

echo ""
echo "Resultado: $PASS passaram, $FAIL falharam"
if [[ $FAIL -ne 0 ]]; then
  exit 1
fi
