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
for task in a b c c2; do
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
<!-- evidence-contract: v2 -->
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

echo "Caso c2: prova de teste mvn/gradle no fim de linha"
task_c2="$TMP_BASE/task-c2.md"

mvn_eol_report() {
  local f="$1"
  {
    report_header "$task_c2"
    cat <<'EOF'
## Comandos Executados
- mvn test
EOF
    base_sections
    cat <<'EOF'
## Resultados de Validação
- Testes: pass
## Critérios de Aceite
- Critério um -> comprovado: mvn test -> BUILD SUCCESS
- Critério dois -> comprovado: mvn test -> BUILD SUCCESS
EOF
  } > "$f"
}

report_c2_mvn="$TMP_BASE/report-c2-mvn.md"
mvn_eol_report "$report_c2_mvn"
out_c2_mvn=$(bash "$VALIDATOR" "$report_c2_mvn" 2>&1); code_c2_mvn=$?
rm -f "$report_c2_mvn"
assert_exit "mvn test no fim da linha passa" 0 $code_c2_mvn
if [[ "$code_c2_mvn" -ne 0 ]]; then printf '    diagnostico: %s\n' "$out_c2_mvn"; fi

gradle_eol_report() {
  local f="$1" cmd="$2"
  {
    report_header "$task_c2"
    printf '## Comandos Executados\n- %s\n' "$cmd"
    base_sections
    cat <<EOF
## Resultados de Validação
- Testes: pass
## Critérios de Aceite
- Critério um -> comprovado: $cmd -> BUILD SUCCESSFUL
- Critério dois -> comprovado: $cmd -> BUILD SUCCESSFUL
EOF
  } > "$f"
}

report_c2_gradle="$TMP_BASE/report-c2-gradle.md"
gradle_eol_report "$report_c2_gradle" "gradle test"
bash "$VALIDATOR" "$report_c2_gradle" >/dev/null 2>&1; code_c2_gradle=$?
rm -f "$report_c2_gradle"
assert_exit "gradle test no fim da linha passa" 0 $code_c2_gradle

report_c2_gradlew="$TMP_BASE/report-c2-gradlew.md"
gradle_eol_report "$report_c2_gradlew" "./gradlew test"
bash "$VALIDATOR" "$report_c2_gradlew" >/dev/null 2>&1; code_c2_gradlew=$?
rm -f "$report_c2_gradlew"
assert_exit "./gradlew test no fim da linha passa" 0 $code_c2_gradlew

# Controle: "mvn test -q" já passava antes da correção (espaço depois de "test"
# satisfaz o catch-all antigo); precisa continuar passando depois dela também.
report_c2_mvn_q="$TMP_BASE/report-c2-mvn-q.md"
{
  report_header "$task_c2"
  cat <<'EOF'
## Comandos Executados
- mvn test -q
EOF
  base_sections
  cat <<'EOF'
## Resultados de Validação
- Testes: pass
## Critérios de Aceite
- Critério um -> comprovado: mvn test -q -> BUILD SUCCESS
- Critério dois -> comprovado: mvn test -q -> BUILD SUCCESS
EOF
} > "$report_c2_mvn_q"
bash "$VALIDATOR" "$report_c2_mvn_q" >/dev/null 2>&1; code_c2_mvn_q=$?
rm -f "$report_c2_mvn_q"
assert_exit "mvn test -q continua passando" 0 $code_c2_mvn_q

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

# --- Caso d2: opt-out explícito NAO reabre o legado (RF-53) ---
echo "Caso d2: AI_SDD_STRICT_EVIDENCE=0 nao reabre o gate de aceite"
out_d2=$(AI_SDD_STRICT_EVIDENCE=0 bash "$VALIDATOR" "$report_d" 2>&1); code_d2=$?
assert_exit "opt-out explícito continua reprovando" 1 $code_d2
if echo "$out_d2" | grep -q "fail-closed desde 0.31.0"; then
  echo "  ✓ opt-out recebe o diagnóstico de fail-closed"
  passed=$((passed+1))
else
  echo "  ✗ opt-out sem diagnóstico de fail-closed (RF-53)"
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

REVIEW_TASK_FILE="$TMP_ROOT/task-review.md"
cat > "$REVIEW_TASK_FILE" <<'TASKEOF'
# Task de fixture para review

## Critérios de Aceite
- Critério um
- Critério dois
TASKEOF

VALID_MAP='## Mapa de Critérios de Aceite
- [atendido] Critério um -> go test ./... -> PASS
- [atendido] Critério dois -> internal/foo.go:42'

# Caso e: review.md válido (APPROVED, sem achados, mapa 1:1 completo) -> exit 0
echo "Caso e: review.md válido sem achados"
review_e="$TMP_BASE/review-e.md"
cat > "$review_e" <<EOF
# Relatório de Review
- Veredito: APPROVED
- Alvo revisado: diff da branch feature/x
- Task file: $REVIEW_TASK_FILE
$VALID_MAP
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
LC_ALL=C bash "$REVIEW_VALIDATOR" "$review_e" >/dev/null 2>&1; assert_exit "review válido passa sob LC_ALL=C" 0 $?

# Caso f: review.md REJECTED sem achado high/critical -> exit 1
echo "Caso f: review REJECTED sem achado bloqueante"
review_f="$TMP_BASE/review-f.md"
cat > "$review_f" <<EOF
# Relatório de Review
- Veredito: REJECTED
- Alvo revisado: diff
$VALID_MAP
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

echo "Caso f2: TEST_NAME_RE aceita nome de teste Java quando stack detectada e Java"
review_f2="$TMP_BASE/review-f2.md"
cat > "$review_f2" <<EOF
# Relatório de Review
- Veredito: APPROVED
- Alvo revisado: diff
- Task file: $REVIEW_TASK_FILE
## Mapa de Critérios de Aceite
- [atendido] Critério um -> ServiceTest.shouldReturnUser -> pass
- [atendido] Critério dois -> Service.java:10
## Achados
Sem achados.
## Arquivos Revisados
- Service.java
## Riscos Residuais
- nenhum
## Validações Executadas
- mvn test -> ok
EOF
bash "$REVIEW_VALIDATOR" "$review_f2" >/dev/null 2>&1; assert_exit "review Java com nome de teste JUnit passa" 0 $?

echo "Caso f3: stack Go rejeita nome de teste não-Go (não-afrouxamento)"
review_f3="$TMP_BASE/review-f3.md"
cat > "$review_f3" <<EOF
# Relatório de Review
- Veredito: APPROVED
- Alvo revisado: diff
- Task file: $REVIEW_TASK_FILE
## Mapa de Critérios de Aceite
- [atendido] Critério um -> shouldReturnUser -> pass
- [atendido] Critério dois -> foo.go:1
## Achados
Sem achados.
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
## Validações Executadas
- go test ./... -> ok
EOF
bash "$REVIEW_VALIDATOR" "$review_f3" >/dev/null 2>&1; assert_exit "review Go com nome de teste não-Go falha (não-afrouxamento)" 1 $?

# Caso g: review.md sem seção de validações -> exit 1
echo "Caso g: review sem seção de validações"
review_g="$TMP_BASE/review-g.md"
cat > "$review_g" <<EOF
# Relatório de Review
- Veredito: APPROVED
- Alvo revisado: diff
$VALID_MAP
## Achados
Sem achados.
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
EOF
bash "$REVIEW_VALIDATOR" "$review_g" >/dev/null 2>&1; assert_exit "review sem validações falha" 1 $?

# --- Casos do mapa 1:1 critério -> evidência (RF-47, RF-48, RF-49, RF-51, RF-53, RF-54) ---
review_body() {
  cat <<EOF
## Achados
Sem achados.
## Arquivos Revisados
- foo.go
## Riscos Residuais
- nenhum
## Validações Executadas
- go test ./... -> ok
EOF
}

MAP_CASE_FILE=""
map_case() {
  local name="$1" expected="$2" map_block="$3"
  local f="$TMP_BASE/review-$name.md"
  {
    printf '# Relatório de Review\n- Veredito: APPROVED\n- Alvo revisado: diff\n- Task file: %s\n' "$REVIEW_TASK_FILE"
    printf '%s\n' "$map_block"
    review_body
  } > "$f"
  local c1 c2
  bash "$REVIEW_VALIDATOR" "$f" >/dev/null 2>&1; c1=$?
  LC_ALL=C bash "$REVIEW_VALIDATOR" "$f" >/dev/null 2>&1; c2=$?
  assert_exit "mapa $name espera exit $expected" "$expected" "$c1"
  assert_exit "mapa $name espera exit $expected sob LC_ALL=C" "$expected" "$c2"
  MAP_CASE_FILE="$f"
}

echo "Caso h1: mapa 1:1 ausente falha"
map_case "h1-ausente" 1 ""

echo "Caso h2: mapa 1:1 incompleto (critério sem evidência) falha"
map_case "h2-incompleto" 1 '## Mapa de Critérios de Aceite
- [atendido] Critério um -> internal/foo.go:1
- [atendido] Critério dois'
map_missing_ev="$MAP_CASE_FILE"

echo "Caso h3: critério não verificável falha"
map_case "h3-nverif" 1 '## Mapa de Critérios de Aceite
- [não verificável] Critério um -> internal/foo.go:1'

echo "Caso h4: linha de evidência fora das três formas de RF-48 falha"
map_case "h4-badev" 1 '## Mapa de Critérios de Aceite
- [atendido] Critério um -> porque confio no autor da mudança'

echo "Caso h5: mapa 1:1 completo e válido passa"
map_case "h5-ok" 0 "$VALID_MAP"

echo "Caso h6: AI_SDD_STRICT_EVIDENCE=0 NÃO reabre o mapa 1:1"
out_h6=$(AI_SDD_STRICT_EVIDENCE=0 bash "$REVIEW_VALIDATOR" "$map_missing_ev" 2>&1); code_h6=$?
assert_exit "opt-out legado não faz o mapa incompleto passar" 1 $code_h6
if echo "$out_h6" | grep -q "gate de aceite ignorado"; then
  echo "  ✗ mapa 1:1 desligado por AI_SDD_STRICT_EVIDENCE=0 (regressão de RF-53)"
  failed=$((failed+1))
else
  echo "  ✓ mapa 1:1 permanece fail-closed sob AI_SDD_STRICT_EVIDENCE=0"
  passed=$((passed+1))
fi

# --- Caso i: RF-53 no validador de tarefa — relatório com critérios sem task file resolvível ---
echo "Caso i: relatório declara critérios sem task file resolvível é fail-closed mesmo com opt-out"
report_i="$TMP_BASE/report-i.md"
{
  report_header "$TMP_BASE/task-inexistente.md"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
EOF
  base_sections
  cat <<'EOF'
## Resultados de Validação
- Testes: pass
## Critérios de Aceite
- Critério um -> comprovado: ok
EOF
} > "$report_i"
out_i=$(AI_SDD_STRICT_EVIDENCE=0 bash "$VALIDATOR" "$report_i" 2>&1); code_i=$?
rm -f "$report_i"
assert_exit "critérios sem task file falha mesmo com opt-out" 1 $code_i
if echo "$out_i" | grep -q "gate de aceite ignorado"; then
  echo "  ✗ opt-out reabriu o confronto 1:1 (regressão de RF-53)"
  failed=$((failed+1))
else
  echo "  ✓ opt-out não alcança o confronto 1:1 (RF-53)"
  passed=$((passed+1))
fi

# --- Caso j: relatório com a seção '## Evidência de Memória Durável' (task 8.0, MD-005) ---
# Prova que a seção nova, inserida entre 'Comandos Executados' e 'Resultados de
# Validação' (como o EnrichReport faz antes da seção de métricas), não desliga
# a captura de nenhum gate existente — nem em locale de bytes (LC_ALL=C).
echo "Caso j: relatório com seção de Evidência de Memória Durável não quebra gates"
rm -f "$TMP_BASE"/review-*.md
report_j="$TMP_BASE/report-j.md"
{
  report_header "$task_b"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
## Evidência de Memória Durável

- session: 20260910T101010.000000000-1234
- cli: claude
- task: task-8.0.md
- read_facts_by_layer: task=2, prd=1
- read_facts_omitted: 0
- read_facts_contradicted: 0
- read_pages_unreadable: 0
- write_facts_by_layer: task=2
- budget_consumed_by_layer: prd=30, task=40
- compactions_executed: 1
- facts_archived_by_layer: task=1
- redactions_applied: 1
- contradictions_detected: 1
- baton_claimed: true
EOF
  base_sections
  cat <<'EOF'
## Resultados de Validação
- Testes: pass
## Critérios de Aceite
- Critério um -> comprovado: saída de go test mostra PASS
- Critério dois -> comprovado: arquivo foo.go contém a função
## Métricas Claude-2026
| Métrica | Valor |
|---|---|
| total_tokens | 100 |
EOF
} > "$report_j"
out_j=$(bash "$VALIDATOR" "$report_j" 2>&1); code_j=$?
assert_exit "relatório com evidência de memória durável passa" 0 $code_j
if [[ "$code_j" -ne 0 ]]; then printf '    diagnostico: %s\n' "$out_j"; fi

out_j_c=$(LC_ALL=C bash "$VALIDATOR" "$report_j" 2>&1); code_j_c=$?
rm -f "$report_j"
assert_exit "relatório com evidência de memória durável passa sob LC_ALL=C" 0 $code_j_c
if [[ "$code_j_c" -ne 0 ]]; then printf '    diagnostico: %s\n' "$out_j_c"; fi

# --- Caso k: o corte do contrato de evidência é um commit fixo, não HEAD ---
# Antes da correção, o corte era a referência móvel "HEAD": bastava commitar um
# relatório para ele virar "evidência histórica" e ganhar a isenção v1. Agora o
# corte é um commit específico e a isenção exige o conteúdo idêntico ao selado
# naquele commit. Em qualquer repositório que não seja este, nenhum relatório
# novo é histórico — nem depois de commitado.
echo "Caso k: relatório sem marcador não vira histórico ao ser commitado"
K_REPO="$TMP_ROOT/k-repo"
mkdir -p "$K_REPO"
git -C "$K_REPO" init -q
git -C "$K_REPO" config user.email "validators-test@example.invalid"
git -C "$K_REPO" config user.name "Validators Test"
git -C "$K_REPO" config commit.gpgsign false
printf 'base\n' >"$K_REPO/seed.txt"
git -C "$K_REPO" add . >/dev/null 2>&1
git -C "$K_REPO" commit -qm "test: baseline k" >/dev/null 2>&1
report_k="$K_REPO/report-k.md"
{
  printf '# Relatório de Execução de Tarefa\n'
  report_header "$task_b" | tail -n +2
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
EOF
  base_sections
  cat <<'EOF'
## Resultados de Validação
- Testes: pass
## Critérios de Aceite
- Critério um -> comprovado: saída de go test mostra PASS
- Critério dois -> comprovado: arquivo foo.go contém a função
EOF
} > "$report_k"

bash "$VALIDATOR" "$report_k" >/dev/null 2>&1; code_k_pre=$?
assert_exit "relatório sem marcador e não commitado falha" 1 $code_k_pre

git -C "$K_REPO" add "$(basename "$report_k")" >/dev/null 2>&1
git -C "$K_REPO" commit -qm "test: commita relatorio sem marcador" >/dev/null 2>&1

out_k=$(bash "$VALIDATOR" "$report_k" 2>&1); code_k=$?
assert_exit "commitar NÃO concede a isenção histórica (corte fixo)" 1 $code_k
if printf '%s' "$out_k" | grep -q "commit de corte"; then
  echo "  ✓ a ruptura aponta o commit de corte fixo, não HEAD"
  passed=$((passed+1))
else
  echo "  ✗ a ruptura deveria citar o commit de corte fixo"
  printf '    diagnostico: %s\n' "$out_k"
  failed=$((failed+1))
fi
if printf '%s' "$out_k" | grep -q "AVISO: contrato de evid"; then
  echo "  ✗ isenção v1 concedida a trabalho novo commitado (defeito do corte móvel)"
  failed=$((failed+1))
else
  echo "  ✓ nenhuma isenção v1 concedida a trabalho novo commitado"
  passed=$((passed+1))
fi
rm -f "$report_k"

# --- Caso l: cruzamento de estado entre relatório, execution-result e tasks.md ---
# Este é o gate que expõe a manobra de editar tasks.md para "blocked" a fim de
# escapar do encerramento, deixando relatório e JSON em "done". A reconciliação
# legítima vai de tasks.md EM DIREÇÃO à evidência; o gate precisa reprovar
# enquanto os três não concordarem, nos três pares possíveis.
echo "Caso l: divergência de estado entre tasks.md, relatório e execution-result"
report_l="$TMP_BASE/report-l.md"
{
  report_header "$task_b"
  cat <<'EOF'
## Comandos Executados
- go test ./... -> ok
EOF
  base_sections
  cat <<'EOF'
## Resultados de Validação
- Testes: pass
## Critérios de Aceite
- Critério um -> comprovado: saída de go test mostra PASS
- Critério dois -> comprovado: arquivo foo.go contém a função
EOF
} > "$report_l"

printf '| # | Título | Status |\n|---|--------|--------|\n| 1.0 | Tarefa | blocked |\n' >"$TMP_BASE/tasks.md"
out_l1=$(bash "$VALIDATOR" "$report_l" 2>&1); code_l1=$?
assert_exit "tasks.md=blocked com relatório/JSON=done reprova" 1 $code_l1
if printf '%s' "$out_l1" | grep -q "divergencia de estado"; then
  echo "  ✓ a ruptura nomeia a divergência de estado"
  passed=$((passed+1))
else
  echo "  ✗ a divergência tasks.md x relatório passou despercebida"
  printf '    diagnostico: %s\n' "$out_l1"
  failed=$((failed+1))
fi

# Nota: escrever tasks.md no fixture altera o snapshot físico (o patch inclui
# arquivos não rastreados), então o exit 0 ponta a ponta não é alcançável aqui.
# A afirmação testada é exatamente a da reconciliação: alinhar tasks.md à
# evidência faz a ruptura de divergência desaparecer.
printf '| # | Título | Status |\n|---|--------|--------|\n| 1.0 | Tarefa | done |\n' >"$TMP_BASE/tasks.md"
out_l2=$(bash "$VALIDATOR" "$report_l" 2>&1)
if printf '%s' "$out_l2" | grep -q "divergencia de estado"; then
  echo "  ✗ reconciliar tasks.md deveria eliminar a ruptura de divergência"
  printf '    diagnostico: %s\n' "$out_l2"
  failed=$((failed+1))
else
  echo "  ✓ tasks.md reconciliado com a evidência elimina a divergência"
  passed=$((passed+1))
fi

printf '| # | Título | Status |\n|---|--------|--------|\n| 1.0 | Tarefa | failed |\n' >"$TMP_BASE/tasks.md"
bash "$VALIDATOR" "$report_l" >/dev/null 2>&1; code_l3=$?
assert_exit "qualquer terceiro estado em tasks.md também reprova" 1 $code_l3

rm -f "$TMP_BASE/tasks.md" "$report_l"

# --- Casos de fronteira do gate de operação Git (RF-57, RF-68) ---
# Prova de bypass verificado end-to-end: head -c 65536 truncava JSON acima de
# 64 KiB, json.load lançava, o except engolia a exceção, command_text voltava
# vazio e git-operation-gate.sh:39-41 respondia com exit 0. A matriz cobre as
# quatro fronteiras de tamanho (1 KiB, 64 KiB, 65 KiB, 1 MiB) em duas variantes
# cada (benigna -> exit 0; git push -> exit 2), mais os cenários de falha
# fechada (JSON inválido, comando ausente, campo command homônimo aninhado).
GIT_GATE="$REPO_ROOT/.agents/scripts/git-operation-gate.sh"

gate_payload() {
  local size_bytes="$1" mode="$2"
  python3 -c "
import json
import sys

size = int(sys.argv[1])
mode = sys.argv[2]
padding = 'A' * size
if mode == 'malicious':
    command = padding + '; git push origin main'
else:
    command = 'echo ' + padding
print(json.dumps({'tool_input': {'command': command}}))
" "$size_bytes" "$mode"
}

assert_gate_exit() {
  local desc="$1" expected="$2" payload_file="$3"
  local actual
  bash "$GIT_GATE" <"$payload_file" >/dev/null 2>&1
  actual=$?
  assert_exit "$desc" "$expected" "$actual"
}

echo "Matriz de fronteira do gate de operação Git (RF-57)"
declare -a boundary_sizes=(1024 65536 66560 1048576)
declare -a boundary_labels=("1 KiB" "64 KiB" "65 KiB" "1 MiB")
for idx in "${!boundary_sizes[@]}"; do
  size="${boundary_sizes[$idx]}"
  label="${boundary_labels[$idx]}"

  payload_benign="$TMP_BASE/git-gate-${label// /}-benign.json"
  gate_payload "$size" benign >"$payload_benign"
  assert_gate_exit "fronteira $label benigna passa" 0 "$payload_benign"
  rm -f "$payload_benign"

  payload_malicious="$TMP_BASE/git-gate-${label// /}-malicious.json"
  gate_payload "$size" malicious >"$payload_malicious"
  assert_gate_exit "fronteira $label com git push bloqueia" 2 "$payload_malicious"
  rm -f "$payload_malicious"
done

# --- Reprodução exata do bypass documentado no ADR-003: ~70 KB de padding
# seguido de "; echo hi ; git push origin main". Antes da correção: exit 0.
echo "Reprodução da prova de exploração do ADR-003 (payload de ~70 KB)"
exploit_payload="$TMP_BASE/git-gate-exploit.json"
python3 -c "
import json
padding = 'A' * 70000
command = padding + '; echo hi ; git push origin main'
print(json.dumps({'tool_input': {'command': command}}))
" >"$exploit_payload"
assert_gate_exit "payload de exploração (~70KB) bloqueia" 2 "$exploit_payload"

control_payload="$TMP_BASE/git-gate-control.json"
printf '%s' '{"tool_input":{"command":"git push origin main"}}' >"$control_payload"
assert_gate_exit "controle sem padding bloqueia" 2 "$control_payload"
rm -f "$exploit_payload" "$control_payload"

# --- Falha fechada: JSON inválido nunca produz exit 0 (RF-57) ---
echo "Falha fechada: JSON inválido bloqueia"
invalid_payload="$TMP_BASE/git-gate-invalid.json"
printf 'isto nao e json' >"$invalid_payload"
assert_gate_exit "JSON inválido bloqueia" 2 "$invalid_payload"
rm -f "$invalid_payload"

# --- Negação por ausência de alvo (RF-68): comando vazio/ausente bloqueia,
# nunca aprova. Alinhado com validate-preload.sh:109.
echo "Negação por ausência de alvo (RF-68)"
empty_payload="$TMP_BASE/git-gate-empty.json"
printf '%s' '{}' >"$empty_payload"
assert_gate_exit "payload sem comando extraível bloqueia" 2 "$empty_payload"
rm -f "$empty_payload"

empty_stdin="$TMP_BASE/git-gate-empty-stdin.json"
printf '' >"$empty_stdin"
assert_gate_exit "stdin vazio bloqueia" 2 "$empty_stdin"
rm -f "$empty_stdin"

# --- Regressão do fallback por grep removido: campo "command" homônimo
# aninhado em outra estrutura não deve ser extraído (nem aprovado nem
# bloqueado por conteúdo alheio); a ausência de extração é negação (RF-68).
echo "Campo command homônimo aninhado não é extraído (regressão do fallback grep)"
nested_payload="$TMP_BASE/git-gate-nested.json"
printf '%s' '{"tool_input":{"nested":{"tool_input":{"command":"git push origin main"}}}}' >"$nested_payload"
assert_gate_exit "comando homônimo aninhado não vaza para aprovação" 2 "$nested_payload"
rm -f "$nested_payload"

benign_nested_payload="$TMP_BASE/git-gate-nested-benign.json"
printf '%s' '{"tool_input":{"command":"echo hi","metadata":{"command":"git push origin main"}}}' >"$benign_nested_payload"
assert_gate_exit "comando direto benigno com metadata homônima passa" 0 "$benign_nested_payload"
rm -f "$benign_nested_payload"


git_gate_command_payload() {
  python3 -c "
import json
import sys
print(json.dumps({'tool_input': {'command': sys.argv[1]}}))
" "$1"
}

assert_git_gate_command() {
  local desc="$1" expected="$2" command="$3"
  local payload_file="$TMP_BASE/git-gate-cmd.json"
  git_gate_command_payload "$command" >"$payload_file"
  local actual
  bash "$GIT_GATE" <"$payload_file" >/dev/null 2>&1
  actual=$?
  assert_exit "$desc" "$expected" "$actual"
  rm -f "$payload_file"
}

echo "Matriz adversarial do gate de operação Git (RF-20, RF-21, RF-24)"
declare -a adversarial_commands=(
  "git -C /repo push"
  "git -c user.name=x commit -m y"
  '$(echo git) push'
  'eval "git push"'
  'sh -c "git push"'
  "/usr/bin/git push"
  "git.exe push"
  "git reset --hard"
  "git clean -fd"
  "git checkout -- ."
  "git restore ."
)
for command in "${adversarial_commands[@]}"; do
  assert_git_gate_command "escapa hoje via regex, deve bloquear/exigir aprovação: $command" 2 "$command"
done

echo "Guarda de regressão: separadores compostos && e || antes de operação git"
declare -a chained_separator_commands=(
  "echo ok && git push"
  "git status && git push --force"
  "true || git clean -fdx"
  "echo ok ; git reset --hard"
  "echo ok &(git push)"
  "echo ok ;(git push)"
  "echo ok )&git push"
  "echo ok )|git push"
)
for command in "${chained_separator_commands[@]}"; do
  assert_git_gate_command "encadeamento não pode escapar do gate: $command" 2 "$command"
done

echo "Matriz de falso positivo do gate de operação Git (RF-22)"
declare -a false_positive_commands=(
  "echo git commit"
  "man git commit"
  'echo "a|git commit -m x"'
  'echo "a|git push origin main"'
)
for command in "${false_positive_commands[@]}"; do
  assert_git_gate_command "bloqueia hoje por menção textual, deve permitir: $command" 0 "$command"
done

echo "Guarda de não-regressão do gate de operação Git"
declare -a non_regression_block_commands=(
  "env git push"
  "command git push"
  "git push --force"
  "git push --force-with-lease"
  "git push -f origin main"
  "git push origin +main"
  "git push"
)
for command in "${non_regression_block_commands[@]}"; do
  assert_git_gate_command "já bloqueia hoje, deve continuar bloqueando: $command" 2 "$command"
done

assert_git_gate_command 'menção textual em grep não bloqueia (não-regressão)' 0 'grep -r "git push" docs/'
assert_git_gate_command 'nome de usuário "git" como valor de flag de wrapper não bloqueia' 0 'sudo -u git whoami'

echo "Falha fechada do classificador em comando com aspa não fechada"
assert_git_gate_command 'aspa não fechada com git bloqueia (falha fechada)' 2 'git push "unterminated'
assert_git_gate_command 'aspa não fechada sem git bloqueia (falha fechada)' 2 'echo "unterminated'

echo "Ofuscação por expansão de variável não pode escapar do gate"
assert_git_gate_command 'expansão de IFS não escapa: git${IFS}push${IFS}--force' 2 'git${IFS}push${IFS}origin${IFS}main${IFS}--force'
assert_git_gate_command 'expansão de variável arbitrária não escapa: $G $P' 2 'G=git; P=push; $G $P --force'

echo "Não-regressão: expansão de variável comum sem relação com git não bloqueia (BUG CRITICAL, review da tarefa 6.0)"
declare -a plain_variable_expansion_commands=(
  "echo \$PATH"
  "cat \$HOME/.bashrc"
  "grep -rn foo \$REPO_ROOT"
  "for f in *.go; do echo \$f; done"
  "go test ./..."
  "make test"
)
for command in "${plain_variable_expansion_commands[@]}"; do
  assert_git_gate_command "expansão de variável comum sem git deve passar: $command" 0 "$command"
done

echo "Wrappers transparentes de execução não podem escapar do gate"
declare -a wrapper_commands=(
  "exec git push --force"
  "time git push --force"
  "nohup git push --force"
  "sudo git push --force"
  "nice git push --force"
  "timeout 5 git push --force"
  "sudo nohup git push --force"
  "timeout -k 5 30 git push --force"
  "sudo -u git -- git push --force"
)
for command in "${wrapper_commands[@]}"; do
  assert_git_gate_command "wrapper transparente não pode escapar: $command" 2 "$command"
done

echo "Continuação de linha não pode escapar do gate"
line_cont_payload="$TMP_BASE/git-gate-line-continuation.json"
python3 -c "
import json
cmd = 'git \\\\\npush --force'
print(json.dumps({'tool_input': {'command': cmd}}))
" >"$line_cont_payload"
assert_gate_exit "continuação de linha antes do subcomando não escapa" 2 "$line_cont_payload"
rm -f "$line_cont_payload"

echo "Subcomando git dinâmico via variável de shell não pode escapar do gate (BUG CRITICAL, re-revisão da tarefa 6.0 pós-cec5d9c)"
declare -a dynamic_subcommand_commands=(
  'git $SUBCMD --force'
  'SUBCMD=push; git $SUBCMD --force'
  'git reset $MODE'
  'git -C /repo $SUBCMD --force'
)
for command in "${dynamic_subcommand_commands[@]}"; do
  assert_git_gate_command "subcomando git dinâmico deve bloquear (fail-closed): $command" 2 "$command"
done

echo "Controle: subcomando git literal com apenas flag/argumento dinâmico já bloqueia por outro motivo"
assert_git_gate_command 'git commit $FLAG -m msg (commit sempre regulado)' 2 'git commit $FLAG -m msg'
assert_git_gate_command 'git push --$SOMEFLAG (push sempre regulado)' 2 'git push --$SOMEFLAG'

echo "Controle: expansão de variável comum sem relação com git continua passando (não reabrir falso positivo original)"
assert_git_gate_command 'echo $PATH deve continuar passando' 0 'echo $PATH'
assert_git_gate_command 'cat $HOME/.bashrc deve continuar passando' 0 'cat $HOME/.bashrc'

echo "Controle: casos já cobertos anteriormente (positivos e negativos) devem permanecer estáveis"
assert_git_gate_command 'controle positivo: git push origin main bloqueia' 2 'git push origin main'
assert_git_gate_command 'bypass IFS já fechado continua fechado' 2 'git${IFS}push${IFS}--force'
assert_git_gate_command 'bypass $G $P já fechado continua fechado' 2 'G=git; P=push; $G $P --force'
assert_git_gate_command 'sudo git push origin main bloqueia' 2 'sudo git push origin main'

echo
echo "Passaram: $passed | Falharam: $failed"
[[ "$failed" -eq 0 ]] || exit 1
echo "Todos os testes de validador passaram."
exit 0
