#!/usr/bin/env bash
# Testes table-driven para .claude/scripts/validate-task-evidence.sh (RF-05).
# Cobertura: 6 casos — válido, sem sha, sem verdict, sem tool, delta -3.0%, delta +0.5%.

set -euo pipefail

SCRIPT="${1:-.claude/scripts/validate-task-evidence.sh}"
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

run_case() {
  local label="$1"
  local content="$2"
  local want_exit="$3"      # 0 = aprovado, 1 = falhou
  local want_text="$4"      # substring esperada no output

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

echo ""
echo "Resultado: $PASS passaram, $FAIL falharam"
if [[ $FAIL -ne 0 ]]; then
  exit 1
fi
