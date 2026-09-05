#!/usr/bin/env bash
# test-validators.sh
# Suite de fixtures para o gate anti-falso-positivo de validate-task-evidence.sh (RF-01..RF-03).
# Cobre os casos a/b/c/d da techspec "Abordagem de Testes":
#   a) critério de aceite não comprovado -> exit 1
#   b) todos os critérios comprovados -> exit 0
#   c) "Testes: pass" sem comando de teste -> exit 1
#   d) task legada sem seção de critérios -> exit 0 (aviso não-fatal)
#
# Uso: bash scripts/test-validators.sh
# Exit 0 = todos passaram; 1 = algum falhou.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VALIDATOR="$REPO_ROOT/.agents/scripts/validate-task-evidence.sh"
scratch_root="${TMPDIR:-$REPO_ROOT/.scratch}"
mkdir -p "$scratch_root"
TMP_ROOT=$(mktemp -d "$scratch_root/test-validators.XXXXXX")
TMP_BASE="$TMP_ROOT/repository"
mkdir -p "$TMP_BASE"
git -C "$TMP_BASE" init -q
git -C "$TMP_BASE" config user.email "validators-test@example.invalid"
git -C "$TMP_BASE" config user.name "Validators Test"
git -C "$TMP_BASE" config commit.gpgsign false
for task in a b c; do
  printf '# Tarefa X\n## Critérios de Sucesso\n- Critério um funciona.\n- Critério dois funciona.\n' >"$TMP_BASE/task-$task.md"
done
printf '# Tarefa Legada\n## Visão Geral\nSem critérios formais.\n' >"$TMP_BASE/task-d.md"
printf 'base\n' >"$TMP_BASE/tracked.txt"
git -C "$TMP_BASE" add .
git -C "$TMP_BASE" commit -qm "test: baseline"
printf 'estado final\n' >"$TMP_BASE/tracked.txt"
mkdir -p "$TMP_BASE/evidence"

GOTOOLCHAIN="${GOTOOLCHAIN:-auto}" go build -o "$TMP_ROOT/ai-spec" "$REPO_ROOT"
export AI_SPEC_BIN="$TMP_ROOT/ai-spec"
git -C "$TMP_BASE" diff --binary HEAD -- . >"$TMP_BASE/evidence/patch.diff"
PATCH_SHA="$(shasum -a 256 "$TMP_BASE/evidence/patch.diff" | awk '{print $1}')"
BASE_SHA="$(git -C "$TMP_BASE" rev-parse HEAD)"
FINAL_STATE_SHA="$( { printf '%s\n' "$BASE_SHA"; cat "$TMP_BASE/evidence/patch.diff"; } | shasum -a 256 | awk '{print $1}')"
printf 'PASS\n' >"$TMP_BASE/evidence/test.log"
TEST_SHA="$(shasum -a 256 "$TMP_BASE/evidence/test.log" | awk '{print $1}')"
cat >"$TMP_BASE/result.json" <<EOF
{"schema_version":2,"run_id":"test","task_id":"1.0","attempt":1,"status":"done","base_sha":"$BASE_SHA","patch_sha256":"$PATCH_SHA","patch_ref":"evidence/patch.diff","final_state_sha256":"$FINAL_STATE_SHA","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"$TEST_SHA"}],"criteria":[{"id":"AC-1","evidence_ref":"evidence/test.log#pass"}],"evidence":["evidence/test.log"],"review_verdict":"approved"}
EOF

passed=0
failed=0

cleanup() { rm -rf "$TMP_ROOT"; }
trap cleanup EXIT

assert_exit() {
  local desc="$1" expected="$2" actual="$3"
  if [[ "$actual" -eq "$expected" ]]; then
    echo "  ✓ $desc (exit=$actual)"
    passed=$((passed+1))
  else
    echo "  ✗ $desc (esperado=$expected, obtido=$actual)"
    failed=$((failed+1))
  fi
}

# Task file fixture com 2 critérios de sucesso.
task_with_criteria() {
  cat > "$1" <<'EOF'
# Tarefa X
## Critérios de Sucesso
- Critério um funciona.
- Critério dois funciona.
EOF
}

# Task file legado sem seção de critérios.
task_without_criteria() {
  cat > "$1" <<'EOF'
# Tarefa Legada
## Visão Geral
Sem critérios formais.
EOF
}

# Cabeçalho comum do relatório, parametrizado pela ref de task file.
report_header() {
  local task_ref="$1"
  cat <<EOF
# Relatório de Execução de Tarefa
## Tarefa
- ID: 1.0
- Arquivo: $task_ref
- Estado: done
## Contexto Carregado
- PRD: n/a
- TechSpec: n/a
## Diff Reviewed
sha=$PATCH_SHA
verdict=APPROVED
tool=claude
## Execution Result
result_path=result.json
## Coverage
package=fixture
delta=+0.0%
EOF
}

base_sections() {
  cat <<'EOF'
## Arquivos Alterados
- foo.go
## Resultados de Validação
- Lint: pass
- Veredito do Revisor: APPROVED
## Suposições
- nenhuma
## Riscos Residuais
- nenhum
EOF
}

# --- Caso a: critério não comprovado -> exit 1 ---
echo "Caso a: critério de aceite não comprovado"
task_a="$TMP_BASE/task-a.md"; task_with_criteria "$task_a"
report_a="$TMP_BASE/report-a.md"
{
  report_header "$task_a"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
## Resultados de Validação
- Testes: pass
EOF
  base_sections
  cat <<'EOF'
## Critérios de Aceite
- Critério um -> comprovado: [evidência]
EOF
} > "$report_a"
bash "$VALIDATOR" "$report_a" >/dev/null 2>&1; code_a=$?
rm -f "$report_a"
assert_exit "critério não comprovado falha" 1 $code_a

# --- Caso a2 (regressão): mesmo gate sob locale de bytes (mawk/LC_ALL=C) ---
# Guarda contra classes de regex multibyte em bracket ([eé]) que fazem awk
# byte-oriented deixar de casar "Critérios" e desligar o gate silenciosamente.
echo "Caso a2: critério não comprovado sob LC_ALL=C"
report_a2="$TMP_BASE/report-a2.md"
{
  report_header "$task_a"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
## Resultados de Validação
- Testes: pass
EOF
  base_sections
  cat <<'EOF'
## Critérios de Aceite
- Critério um -> comprovado: [evidência]
EOF
} > "$report_a2"
out_a2=$(LC_ALL=C bash "$VALIDATOR" "$report_a2" 2>&1); code_a2=$?
rm -f "$report_a2"
assert_exit "critério não comprovado falha sob LC_ALL=C" 1 $code_a2
if echo "$out_a2" | grep -q "gate de aceite ignorado"; then
  echo "  ✗ gate de aceite desligado sob LC_ALL=C (fail-open)"
  failed=$((failed+1))
else
  echo "  ✓ gate de aceite permanece ativo sob LC_ALL=C"
  passed=$((passed+1))
fi

# --- Caso b: todos comprovados -> exit 0 ---
echo "Caso b: todos os critérios comprovados"
task_b="$TMP_BASE/task-b.md"; task_with_criteria "$task_b"
report_b="$TMP_BASE/report-b.md"
{
  report_header "$task_b"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
EOF
  base_sections
  cat <<'EOF'
## Critérios de Aceite
- Critério um -> comprovado: saída de go test mostra PASS
- Critério dois -> comprovado: arquivo foo.go contém a função
EOF
  echo "## Resultados de Validação"
  echo "- Testes: pass"
} > "$report_b"
# Ajuste: garantir Testes: pass presente uma vez e comando de teste presente.
out_b=$(bash "$VALIDATOR" "$report_b" 2>&1); code_b=$?
rm -f "$report_b"
assert_exit "todos comprovados passa" 0 $code_b
if [[ "$code_b" -ne 0 ]]; then printf '    diagnostico: %s\n' "$out_b"; fi

# --- Caso c: Testes: pass sem comando -> exit 1 ---
echo "Caso c: Testes: pass sem comando de teste"
task_c="$TMP_BASE/task-c.md"; task_with_criteria "$task_c"
report_c="$TMP_BASE/report-c.md"
{
  report_header "$task_c"
  cat <<'EOF'
## Comandos Executados
- echo hello -> ok
EOF
  base_sections
  cat <<'EOF'
## Critérios de Aceite
- Critério um -> comprovado: ok
- Critério dois -> comprovado: ok
EOF
  echo "## Resultados de Validação"
  echo "- Testes: pass"
} > "$report_c"
bash "$VALIDATOR" "$report_c" >/dev/null 2>&1; code_c=$?
rm -f "$report_c"
assert_exit "Testes pass sem comando falha" 1 $code_c

# --- Caso d: task legada sem critérios -> exit 1 (fail-closed desde 0.31.0) ---
echo "Caso d: task legada sem seção de critérios"
task_d="$TMP_BASE/task-d.md"; task_without_criteria "$task_d"
report_d="$TMP_BASE/report-d.md"
{
  report_header "$task_d"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
EOF
  base_sections
  echo "## Resultados de Validação"
  echo "- Testes: pass"
} > "$report_d"
out_d=$(bash "$VALIDATOR" "$report_d" 2>&1); code_d=$?
assert_exit "task legada falha por padrão" 1 $code_d
if echo "$out_d" | grep -q "fail-closed desde 0.31.0"; then
  echo "  ✓ diagnóstico de fail-closed presente"
  passed=$((passed+1))
else
  echo "  ✗ diagnóstico de fail-closed ausente"
  failed=$((failed+1))
fi

# --- Caso d2: opt-out explícito reabre o legado, com aviso ruidoso ---
echo "Caso d2: AI_SDD_STRICT_EVIDENCE=0 reabre o legado"
out_d2=$(AI_SDD_STRICT_EVIDENCE=0 bash "$VALIDATOR" "$report_d" 2>&1); code_d2=$?
assert_exit "opt-out explícito passa" 0 $code_d2
if echo "$out_d2" | grep -q "NAO comprova os criterios"; then
  echo "  ✓ opt-out avisa que a evidência não comprova critérios"
  passed=$((passed+1))
else
  echo "  ✗ opt-out silencioso (regressão do BUG-127)"
  failed=$((failed+1))
fi
rm -f "$report_d"

# --- Caso d3: referência de task file não resolvível também é fail-closed ---
echo "Caso d3: referência de task file inexistente"
report_d3="$TMP_BASE/report-d3.md"
{
  report_header "$TMP_BASE/task-inexistente.md"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
EOF
  base_sections
  echo "## Resultados de Validação"
  echo "- Testes: pass"
} > "$report_d3"
bash "$VALIDATOR" "$report_d3" >/dev/null 2>&1; code_d3=$?
rm -f "$report_d3"
assert_exit "referência não resolvível falha por padrão" 1 $code_d3

# --- Casos de review-evidence (RF-20) ---
REVIEW_VALIDATOR="$REPO_ROOT/.agents/scripts/validate-review-evidence.sh"

# Caso e: review.md válido (APPROVED, sem achados) -> exit 0
echo "Caso e: review.md válido sem achados"
review_e="$TMP_BASE/review-e.md"
cat > "$review_e" <<'EOF'
# Relatório de Review
- Veredito: APPROVED
- Alvo revisado: diff da branch feature/x
## Achados
Sem achados.
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
## Validações Executadas
- go test ./... -> ok
EOF
bash "$REVIEW_VALIDATOR" "$review_e" >/dev/null 2>&1; assert_exit "review válido passa" 0 $?

# Caso f: review.md REJECTED sem achado high/critical -> exit 1
echo "Caso f: review REJECTED sem achado bloqueante"
review_f="$TMP_BASE/review-f.md"
cat > "$review_f" <<'EOF'
# Relatório de Review
- Veredito: REJECTED
- Alvo revisado: diff
## Achados
- Severidade: low
- Arquivo: foo.go
- Impacto: cosmético
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
## Validações Executadas
- go test ./... -> ok
EOF
bash "$REVIEW_VALIDATOR" "$review_f" >/dev/null 2>&1; assert_exit "REJECTED sem high/critical falha" 1 $?

# Caso g: review.md sem seção de validações -> exit 1
echo "Caso g: review sem seção de validações"
review_g="$TMP_BASE/review-g.md"
cat > "$review_g" <<'EOF'
# Relatório de Review
- Veredito: APPROVED
- Alvo revisado: diff
## Achados
Sem achados.
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
EOF
bash "$REVIEW_VALIDATOR" "$review_g" >/dev/null 2>&1; assert_exit "review sem validações falha" 1 $?

echo
echo "Passaram: $passed | Falharam: $failed"
[[ "$failed" -eq 0 ]] || exit 1
echo "Todos os testes de validador passaram."
exit 0
