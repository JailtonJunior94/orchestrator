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

# ── Gate de versao do binario ai-spec ────────────────────────────────────────
# Um ai-spec anterior ao contrato v2 reprovava a prova fisica por motivo falso
# ("prova fisica invalida"). O validador agora exige versao minima e falha com
# mensagem explicita sobre a toolchain.
STALE_BIN="$TMP_ROOT/stale-ai-spec"
cat >"$STALE_BIN" <<'STALE'
#!/usr/bin/env bash
if [[ "${1:-}" == "version" ]]; then
  echo "ai-spec-harness 1.1.0 (commit: deadbeef, built: 2026-09-05T20:00:32Z)"
  exit 0
fi
echo "erro: subcomando desconhecido" >&2
exit 1
STALE
chmod +x "$STALE_BIN"

VERSION_FIXTURE="$TMPDIR_BASE/report_version_gate.md"
printf '<!-- evidence-contract: v2 -->\n%s' "$VALID_REPORT" >"$VERSION_FIXTURE"

stale_exit=0
stale_out=$(AI_SPEC_BIN="$STALE_BIN" bash "$SCRIPT" "$VERSION_FIXTURE" 2>&1) || stale_exit=$?
if [[ "$stale_exit" -eq 1 ]] \
  && grep -qi "toolchain ai-spec incompativel" <<<"$stale_out" \
  && grep -qi "1.1.0" <<<"$stale_out" \
  && ! grep -qi "prova fisica invalida" <<<"$stale_out"; then
  echo "PASS [TC25-binario-obsoleto-falha-com-mensagem-de-versao]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC25-binario-obsoleto-falha-com-mensagem-de-versao]: exit=$stale_exit"
  echo "  output: $stale_out"
  FAIL=$((FAIL+1))
fi

absent_exit=0
absent_out=$(AI_SPEC_BIN="$TMP_ROOT/inexistente-ai-spec" bash "$SCRIPT" "$VERSION_FIXTURE" 2>&1) || absent_exit=$?
if [[ "$absent_exit" -eq 1 ]] \
  && grep -qi "toolchain ai-spec incompativel" <<<"$absent_out" \
  && grep -qi "ausente ou nao executavel" <<<"$absent_out" \
  && ! grep -qi "prova fisica invalida" <<<"$absent_out"; then
  echo "PASS [TC26-binario-ausente-falha-com-mensagem-de-versao]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC26-binario-ausente-falha-com-mensagem-de-versao]: exit=$absent_exit"
  echo "  output: $absent_out"
  FAIL=$((FAIL+1))
fi

compatible_exit=0
compatible_out=$(bash "$SCRIPT" "$VERSION_FIXTURE" 2>&1) || compatible_exit=$?
if [[ "$compatible_exit" -eq 0 ]] && ! grep -qi "toolchain ai-spec incompativel" <<<"$compatible_out"; then
  echo "PASS [TC27-binario-compativel-passa]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC27-binario-compativel-passa]: exit=$compatible_exit"
  echo "  output: $compatible_out"
  FAIL=$((FAIL+1))
fi
rm -f "$VERSION_FIXTURE"


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
run_case "TC13-remarks-without-declared-severity-fails-closed" "$REMARKS_REPORT" 1 "RF-33, fail-closed"

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
if [[ "$remarks_exit" -eq 1 ]] && grep -qi "RF-33" <<<"$remarks_out"; then
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
if [[ "$strict_exit" -eq 1 ]] && grep -qi "RF-33" <<<"$strict_out"; then
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

printf 'PASS\n' >"$TMPDIR_BASE/evidence/test.log"
git -C "$TMPDIR_BASE" diff --binary HEAD -- . >"$TMPDIR_BASE/evidence/patch.diff"
FRESH_BASE_SHA="$(git -C "$TMPDIR_BASE" rev-parse HEAD)"
FRESH_PATCH_SHA="$(shasum -a 256 "$TMPDIR_BASE/evidence/patch.diff" | awk '{print $1}')"
FRESH_FINAL_SHA="$( { printf '%s\n' "$FRESH_BASE_SHA"; cat "$TMPDIR_BASE/evidence/patch.diff"; } | shasum -a 256 | awk '{print $1}')"
FRESH_RESULT="${VALID_RESULT//$BASE_SHA/$FRESH_BASE_SHA}"
FRESH_RESULT="${FRESH_RESULT//$PATCH_SHA/$FRESH_PATCH_SHA}"
FRESH_RESULT="${FRESH_RESULT//$FINAL_STATE_SHA/$FRESH_FINAL_SHA}"
FRESH_REPORT="${VALID_REPORT//$PATCH_SHA/$FRESH_PATCH_SHA}"

printf '%s\n' "$FRESH_RESULT" >"$TMPDIR_BASE/result.json"
run_case "TC28-done-with-honest-evidence-passes" "$FRESH_REPORT" 0 "aprovada"

printf 'TAMPERED\n' >"$TMPDIR_BASE/evidence/test.log"
run_case "TC29-done-with-tampered-hash-fails-physical-proof" "$FRESH_REPORT" 1 "nao corresponde a evidencia fisica"
printf 'PASS\n' >"$TMPDIR_BASE/evidence/test.log"

NOT_DONE_RESULT="${FRESH_RESULT/\"status\":\"done\"/\"status\":\"blocked\"}"
NOT_DONE_RESULT="${NOT_DONE_RESULT/\"review_verdict\":\"approved\"/\"review_verdict\":\"changes_requested\"}"
printf '%s\n' "$NOT_DONE_RESULT" >"$TMPDIR_BASE/result.json"

escape_exit=0
escape_out=$(printf '<!-- evidence-contract: v2 -->\n%s' "$FRESH_REPORT" >"$TMPDIR_BASE/escape.md" && bash "$SCRIPT" "$TMPDIR_BASE/escape.md" 2>&1) || escape_exit=$?
rm -f "$TMPDIR_BASE/escape.md"
if [[ "$escape_exit" -eq 1 ]] \
  && grep -qi "veredito aprovador (verdict=APPROVED) sobre execution-result nao-done" <<<"$escape_out" \
  && ! grep -qi "done incompleto" <<<"$escape_out"; then
  echo "PASS [TC30-not-done-with-approving-verdict-fails-as-escape]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC30-not-done-with-approving-verdict-fails-as-escape]: exit=$escape_exit"
  echo "  output: $escape_out"
  FAIL=$((FAIL+1))
fi

REMARKS_REPORT="${FRESH_REPORT/verdict=APPROVED/verdict=REJECTED}"
remarks_exit=0
remarks_out=$(printf '<!-- evidence-contract: v2 -->\n%s' "$REMARKS_REPORT" >"$TMPDIR_BASE/remarks.md" && bash "$SCRIPT" "$TMPDIR_BASE/remarks.md" 2>&1) || remarks_exit=$?
rm -f "$TMPDIR_BASE/remarks.md"
if [[ "$remarks_exit" -eq 1 ]] \
  && grep -qi "NAO APLICAVEL: prova fisica dispensada" <<<"$remarks_out" \
  && grep -qi "veredito do reviewer não encerra o ciclo de aprovação" <<<"$remarks_out" \
  && grep -qi "status='blocked'" <<<"$remarks_out" \
  && ! grep -qi "prova fisica invalida" <<<"$remarks_out"; then
  echo "PASS [TC31-not-done-with-non-approving-verdict-skips-physical-proof]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC31-not-done-with-non-approving-verdict-skips-physical-proof]: exit=$remarks_exit"
  echo "  output: $remarks_out"
  FAIL=$((FAIL+1))
fi

printf '%s\n' "${FRESH_RESULT/\"schema_version\":2/\"schema_version\":1}" >"$TMPDIR_BASE/result.json"
run_case "TC32-wrong-schema-version-reports-malformed" "$FRESH_REPORT" 1 "schema_version 1 diferente de 2"

printf '%s\n' "${FRESH_RESULT/,\"base_sha\":\"$FRESH_BASE_SHA\"/}" >"$TMPDIR_BASE/result.json"
run_case "TC33-missing-required-field-reports-malformed" "$FRESH_REPORT" 1 "campos obrigatorios ausentes: base_sha"

printf '%s\n' "$FRESH_RESULT" >"$TMPDIR_BASE/result.json"

MEDIUM_REPORT="${FRESH_REPORT/verdict=APPROVED/verdict=APPROVED_WITH_REMARKS}"
MEDIUM_REPORT="${MEDIUM_REPORT/Veredito do Revisor: APPROVED/Veredito do Revisor: APPROVED_WITH_REMARKS}"
MEDIUM_REPORT="$MEDIUM_REPORT

## Achados
- [MEDIUM] internal/taskloop/evidence.go:1 nomenclatura inconsistente
"
run_case "TC34-remarks-with-medium-only-closes" "$MEDIUM_REPORT" 0 "aprovada"

HIGH_REPORT="${MEDIUM_REPORT/\[MEDIUM\]/[HIGH]}"
run_case "TC35-remarks-with-high-does-not-close" "$HIGH_REPORT" 1 "achado high/critical declarado"

CRITICAL_REPORT="${MEDIUM_REPORT/\[MEDIUM\]/[CRITICAL]}"
run_case "TC36-remarks-with-critical-does-not-close" "$CRITICAL_REPORT" 1 "achado high/critical declarado"

BLOCKER_REPORT="${MEDIUM_REPORT/\[MEDIUM\]/[blocker]}"
run_case "TC37-remarks-with-blocker-does-not-close" "$BLOCKER_REPORT" 1 "achado high/critical declarado"

printf '%s\n' "$NOT_DONE_RESULT" >"$TMPDIR_BASE/result.json"
escape_remarks_exit=0
escape_remarks_out=$(printf '<!-- evidence-contract: v2 -->\n%s' "$MEDIUM_REPORT" >"$TMPDIR_BASE/escape2.md" && bash "$SCRIPT" "$TMPDIR_BASE/escape2.md" 2>&1) || escape_remarks_exit=$?
rm -f "$TMPDIR_BASE/escape2.md"
if [[ "$escape_remarks_exit" -eq 1 ]] \
  && grep -qi "veredito aprovador (verdict=APPROVED_WITH_REMARKS) sobre execution-result nao-done" <<<"$escape_remarks_out"; then
  echo "PASS [TC38-not-done-with-closing-remarks-fails-as-escape]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC38-not-done-with-closing-remarks-fails-as-escape]: exit=$escape_remarks_exit"
  echo "  output: $escape_remarks_out"
  FAIL=$((FAIL+1))
fi

printf '%s\n' "$FRESH_RESULT" >"$TMPDIR_BASE/result.json"

REMARKS_BASE="${FRESH_REPORT/verdict=APPROVED/verdict=APPROVED_WITH_REMARKS}"
REMARKS_BASE="${REMARKS_BASE/Veredito do Revisor: APPROVED/Veredito do Revisor: APPROVED_WITH_REMARKS}"

remarks_with() {
  printf '%s\n## Achados\n%s\n' "$REMARKS_BASE" "$1"
}

run_case "TC39-remarks-high-em-tabela-nao-encerra" \
  "$(remarks_with '| [HIGH] | internal/taskloop/evidence.go:1 | corrida de dados |
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 1 "achado high/critical declarado"

run_case "TC40-remarks-critical-em-lista-numerada-nao-encerra" \
  "$(remarks_with '1. [CRITICAL] internal/taskloop/evidence.go:1 vazamento de segredo
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 1 "achado high/critical declarado"

run_case "TC41-remarks-high-em-crase-nao-encerra" \
  "$(remarks_with '- `[HIGH]` internal/taskloop/evidence.go:1 corrida de dados
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 1 "achado high/critical declarado"

run_case "TC42-remarks-high-em-heading-nao-encerra" \
  "$(remarks_with '### [HIGH] internal/taskloop/evidence.go:1 corrida de dados
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 1 "achado high/critical declarado"

run_case "TC43-remarks-high-em-checkbox-nao-encerra" \
  "$(remarks_with '- [ ] [HIGH] internal/taskloop/evidence.go:1 corrida de dados
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 1 "achado high/critical declarado"

run_case "TC44-remarks-campo-severidade-em-negrito-nao-encerra" \
  "$(remarks_with '- **Severidade**: high
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 1 "achado high/critical declarado"

run_case "TC45-mencao-em-prosa-nao-bloqueia" \
  "$(remarks_with 'Veredito `APPROVED_WITH_REMARKS`, sem tag `[CRITICAL]`/`[HIGH]` bloqueante.
- Veredito do reviewer: APPROVED_WITH_REMARKS (sem tag `[critical]`/`[blocker]`)
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 0 "aprovada"

run_case "TC46-achado-citado-em-cerca-nao-bloqueia" \
  "$(remarks_with '```text
- [CRITICAL] exemplo citado dentro de bloco de codigo
| [HIGH] | exemplo | em tabela citada |
```
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 0 "aprovada"

run_case "TC47-escape-combinado-nao-encerra" \
  "$(remarks_with '| [HIGH] | internal/taskloop/evidence.go:1 | corrida de dados |
1. [CRITICAL] internal/taskloop/evidence.go:2 vazamento de segredo
- `[HIGH]` internal/taskloop/evidence.go:4 leitura sem lock
- [LOW] internal/taskloop/evidence.go:3 renomear variavel')" 1 "achado high/critical declarado"

BLOCKED_PROSE_REPORT="${FRESH_REPORT/- Estado: done/- Estado: blocked}"

run_case "TC48-estado-divergente-do-execution-result" "$BLOCKED_PROSE_REPORT" 1 "divergencia de estado"

printf 'TAMPERED\n' >"$TMPDIR_BASE/evidence/test.log"
tamper_exit=0
printf '<!-- evidence-contract: v2 -->\n%s' "$BLOCKED_PROSE_REPORT" >"$TMPDIR_BASE/tamper.md"
tamper_out=$(bash "$SCRIPT" "$TMPDIR_BASE/tamper.md" 2>&1) || tamper_exit=$?
rm -f "$TMPDIR_BASE/tamper.md"
if [[ "$tamper_exit" -eq 1 ]] \
  && grep -qi "divergencia de estado" <<<"$tamper_out" \
  && grep -qi "nao corresponde a evidencia fisica" <<<"$tamper_out"; then
  echo "PASS [TC49-prova-fisica-nao-e-dispensada-por-edicao-de-prosa]"
  PASS=$((PASS+1))
else
  echo "FAIL [TC49-prova-fisica-nao-e-dispensada-por-edicao-de-prosa]: exit=$tamper_exit"
  echo "  output: $tamper_out"
  FAIL=$((FAIL+1))
fi
printf 'PASS\n' >"$TMPDIR_BASE/evidence/test.log"

TASKS_DIR="$TMPDIR_BASE/.specs/prd-tasks-md"
mkdir -p "$TASKS_DIR"

run_tasks_md_case() {
  local label="$1"
  local tasks_status="$2"
  local want_exit="$3"
  local want_text="$4"

  printf '# Tasks\n\n| ID | Titulo | Status |\n|----|--------|--------|\n| 5.0 | Evidencia | %s |\n' \
    "$tasks_status" >"$TASKS_DIR/tasks.md"
  printf '<!-- evidence-contract: v2 -->\n%s' "$BLOCKED_REPORT" >"$TASKS_DIR/5.0_execution_report.md"

  local actual_exit=0
  local actual_out
  actual_out=$(bash "$SCRIPT" "$TASKS_DIR/5.0_execution_report.md" 2>&1) || actual_exit=$?
  rm -f "$TASKS_DIR/5.0_execution_report.md" "$TASKS_DIR/tasks.md"

  if [[ "$actual_exit" -eq "$want_exit" ]] \
    && { [[ -z "$want_text" ]] || grep -qi "$want_text" <<<"$actual_out"; }; then
    echo "PASS [$label]"
    PASS=$((PASS+1))
  else
    echo "FAIL [$label]: exit=$actual_exit, want=$want_exit"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
  fi
}

run_tasks_md_case "TC50-estado-divergente-de-tasks-md" done 1 "registra 'done' para a tarefa 5.0"
run_tasks_md_case "TC51-estado-consistente-com-tasks-md" blocked 0 "aprovada"
rm -rf "$TASKS_DIR"

echo ""
echo "Resultado: $PASS passaram, $FAIL falharam"
if [[ $FAIL -ne 0 ]]; then
  exit 1
fi
