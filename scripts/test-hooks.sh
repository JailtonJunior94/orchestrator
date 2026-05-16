#!/usr/bin/env bash
# test-hooks.sh
# Suite empirica dos hooks do orquestrador. Constroi fixtures temporarios em
# tmp/ e valida que cada fragilidade dispara a deteccao esperada.
#
# Cobre: F2, F13, F17, F18, F24, F25, F27, F29, F35 e contrato YAML.
#
# Uso: bash scripts/test-hooks.sh
# Exit 0 = todos os testes passaram; exit 1 = algum teste falhou.

set -uo pipefail
# NAO usar -e: queremos rodar todos os testes mesmo se um falhar.

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.claude/hooks"
TMP_BASE=$(mktemp -d /tmp/test-hooks.XXXXXX)
TASKS_BASE="$TMP_BASE/tasks"
mkdir -p "$TASKS_BASE"

# Garante que os hooks testem o CLI da working tree atual, nao uma versao antiga
# instalada no sistema.
mkdir -p "$TMP_BASE/bin"
cat > "$TMP_BASE/bin/ai-spec" <<EOF
#!/usr/bin/env bash
cd "$REPO_ROOT" && go run . "\$@"
EOF
chmod +x "$TMP_BASE/bin/ai-spec"
export PATH="$TMP_BASE/bin:$PATH"

# Override env vars para fixtures temporarios.
export AI_TASKS_ROOT="$(realpath --relative-to="$REPO_ROOT" "$TASKS_BASE" 2>/dev/null || python3 -c "import os; print(os.path.relpath('$TASKS_BASE', '$REPO_ROOT'))")"
export AI_PRD_PREFIX="prd-"

passed=0
failed=0

assert_exit() {
  local desc="$1"
  local expected_exit="$2"
  local actual_exit="$3"
  if [[ "$actual_exit" -eq "$expected_exit" ]]; then
    echo "  ✓ $desc (exit=$actual_exit)"
    passed=$((passed+1))
  else
    echo "  ✗ $desc (esperado exit=$expected_exit, obtido=$actual_exit)"
    failed=$((failed+1))
  fi
}

assert_stderr_contains() {
  local desc="$1"
  local pattern="$2"
  local stderr_file="$3"
  if grep -qE "$pattern" "$stderr_file" 2>/dev/null; then
    echo "  ✓ stderr contem '$pattern'"
    passed=$((passed+1))
  else
    echo "  ✗ stderr NAO contem '$pattern'"
    echo "    stderr: $(cat "$stderr_file" 2>/dev/null | head -5)"
    failed=$((failed+1))
  fi
}

cleanup() {
  rm -rf "$TMP_BASE"
}
trap cleanup EXIT

# Helper: cria PRD fixture minimo
make_prd() {
  local slug="$1"
  shift  # restante = task lines
  local dir="$TASKS_BASE/prd-$slug"
  mkdir -p "$dir"
  echo "# PRD $slug" > "$dir/prd.md"
  echo "# Techspec $slug" > "$dir/techspec.md"

  local prd_hash
  prd_hash=$(ai-spec hash "$dir/prd.md")

  cat > "$dir/tasks.md" <<EOF
<!-- spec-hash-prd: $prd_hash -->
<!-- spec-hash-techspec: 0000 -->

# Tasks $slug

| # | Título | Status | Dependências | Paralelizável | Skills |
|---|--------|--------|--------------|---------------|--------|
EOF
  for line in "$@"; do
    echo "$line" >> "$dir/tasks.md"
  done
}

# Helper: simula edicao do prd.md sem regerar tasks.md (cria drift de hash)
desync_prd_hash() {
  local slug="$1"
  echo "# PRD $slug — EDITADO" > "$TASKS_BASE/prd-$slug/prd.md"
}

echo "==============================================="
echo "TEST HARNESS — hooks do orquestrador"
echo "==============================================="
echo

# ============================================================================
echo "--- F18: cross-PRD spec-hash drift ---"
# ============================================================================
make_prd "extb_v1" "| 1.0 | Foo | done | — | — | — |"
make_prd "extb_v1_dep" "| 1.0 | Bar | pending | extb_v1/1.0 | — | — |"
# Sem drift ainda — deve passar
stderr=$(mktemp)
bash "$HOOKS_DIR/pre-execute-all-tasks.sh" "extb_v1_dep" 2>"$stderr"; rc=$?
assert_exit "F18 sem drift = exit 0" 0 "$rc"

# Agora forcar drift
desync_prd_hash "extb_v1"
bash "$HOOKS_DIR/pre-execute-all-tasks.sh" "extb_v1_dep" 2>"$stderr"; rc=$?
assert_exit "F18 com drift = exit 1" 1 "$rc"
assert_stderr_contains "FAIL F18 detectada" "FAIL F18: cross-PRD 'extb_v1' tem spec drift" "$stderr"
rm -f "$stderr"

# ============================================================================
echo
echo "--- F18: cross-PRD task ausente / nao done ---"
# ============================================================================
make_prd "ext_status" "| 1.0 | Foo | pending | — | — | — |"
make_prd "ext_status_dep" "| 1.0 | Bar | pending | ext_status/1.0 | — | — |"
stderr=$(mktemp)
bash "$HOOKS_DIR/pre-execute-all-tasks.sh" "ext_status_dep" 2>"$stderr"; rc=$?
assert_exit "F18 task externa nao done = exit 1" 1 "$rc"
assert_stderr_contains "FAIL F18 task not done detectada" "FAIL F18: cross-PRD task not done: ext_status/1.0" "$stderr"

make_prd "ext_missing_dep" "| 1.0 | Bar | pending | ext_status/2.0 | — | — |"
bash "$HOOKS_DIR/pre-execute-all-tasks.sh" "ext_missing_dep" 2>"$stderr"; rc=$?
assert_exit "F18 task externa ausente = exit 1" 1 "$rc"
assert_stderr_contains "FAIL F18 task not found detectada" "FAIL F18: cross-PRD task not found: ext_status/2.0" "$stderr"
rm -f "$stderr"

# ============================================================================
echo
echo "--- F27: cross-PRD circular dependency ---"
# ============================================================================
# Criar A → B → A (ciclo)
make_prd "circ_a" "| 1.0 | A1 | pending | circ_b/1.0 | — | — |"
make_prd "circ_b" "| 1.0 | B1 | pending | circ_a/1.0 | — | — |"
stderr=$(mktemp)
bash "$HOOKS_DIR/pre-execute-all-tasks.sh" "circ_a" 2>"$stderr"; rc=$?
assert_exit "F27 ciclo detectado = exit 1" 1 "$rc"
assert_stderr_contains "FAIL F27 detectada" "FAIL F27: ciclo cross-PRD detectado" "$stderr"
rm -f "$stderr"

# ============================================================================
echo
echo "--- F29: gaps numericos ---"
# ============================================================================
make_prd "gaps" \
  "| 1.0 | A | pending | — | — | — |" \
  "| 3.0 | C | pending | — | — | — |" \
  "| 5.0 | E | pending | — | — | — |"
stderr=$(mktemp)
bash "$HOOKS_DIR/pre-execute-all-tasks.sh" "gaps" 2>"$stderr"; rc=$?
assert_exit "F29 gaps sem confirmacao = exit 1" 1 "$rc"
assert_stderr_contains "FAIL F29 detectado" "FAIL F29: gaps na numeracao" "$stderr"

AI_ALLOW_TASK_ID_GAPS=1 bash "$HOOKS_DIR/pre-execute-all-tasks.sh" "gaps" 2>"$stderr"; rc=$?
assert_exit "F29 gaps com confirmacao explicita = exit 0" 0 "$rc"
assert_stderr_contains "WARN F29 detectado" "WARN F29: gaps aceitos por AI_ALLOW_TASK_ID_GAPS=1" "$stderr"
rm -f "$stderr"

# ============================================================================
echo
echo "--- F35: git revert (DiffSHA inexistente) ---"
# ============================================================================
make_prd "revert" "| 1.0 | A | done | — | — | — |"
# Criar report com DiffSHA fake
cat > "$TASKS_BASE/prd-revert/1.0_execution_report.md" <<EOF
# Report

sha=deadbeefcafe1234567890abcdef1234567890ab
verdict=APPROVED
EOF
yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $AI_TASKS_ROOT/prd-revert/1.0_execution_report.md
summary: ok
EOF
# Criar checkpoint pra evitar F25 FAIL
mkdir -p "$TASKS_BASE/prd-revert/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-revert/.checkpoints/1.0.yaml"

stderr=$(mktemp)
AI_VALIDATE_GIT_HISTORY=1 bash "$HOOKS_DIR/post-execute-task.sh" "revert" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F35 com SHA fake = exit 1" 1 "$rc"
assert_stderr_contains "FAIL F35 detectada" "FAIL F35: DiffSHA deadbeef" "$stderr"

# Sem opt-in deve passar
bash "$HOOKS_DIR/post-execute-task.sh" "revert" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F35 sem opt-in = exit 0 (skip)" 0 "$rc"
rm -f "$stderr" "$yaml"

# ============================================================================
echo
echo "--- F25: checkpoint ausente bloqueia (default FAIL) ---"
# ============================================================================
make_prd "nochkpt" "| 1.0 | A | done | — | — | — |"
echo "report" > "$TASKS_BASE/prd-nochkpt/1.0_execution_report.md"
yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $AI_TASKS_ROOT/prd-nochkpt/1.0_execution_report.md
summary: ok
EOF
stderr=$(mktemp)
bash "$HOOKS_DIR/post-execute-task.sh" "nochkpt" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F25 sem checkpoint default = exit 1 (FAIL)" 1 "$rc"
assert_stderr_contains "FAIL F25 detectada" "FAIL F25: checkpoint ausente" "$stderr"

# Com env override = WARN
AI_ALLOW_MISSING_CHECKPOINT=1 bash "$HOOKS_DIR/post-execute-task.sh" "nochkpt" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F25 com AI_ALLOW_MISSING_CHECKPOINT=1 = exit 0" 0 "$rc"
assert_stderr_contains "WARN F25 detectado em modo back compat" "WARN F25: checkpoint ausente.*back compat" "$stderr"
rm -f "$stderr" "$yaml"

# ============================================================================
echo
echo "--- Contrato YAML e status drift ---"
# ============================================================================
make_prd "yaml_contract" "| 1.0 | A | done | — | — | — |"
echo "report" > "$TASKS_BASE/prd-yaml_contract/1.0_execution_report.md"
mkdir -p "$TASKS_BASE/prd-yaml_contract/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-yaml_contract/.checkpoints/1.0.yaml"

yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $AI_TASKS_ROOT/prd-yaml_contract/1.0_execution_report.md
summary: ok
extra: proibido
EOF
stderr=$(mktemp)
bash "$HOOKS_DIR/post-execute-task.sh" "yaml_contract" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "YAML com campo extra = exit 1" 1 "$rc"
assert_stderr_contains "contract violation campo extra" "contract violation" "$stderr"

cat > "$yaml" <<EOF
status: done
report_path: $AI_TASKS_ROOT/prd-yaml_contract/1.0_execution_report.md
EOF
bash "$HOOKS_DIR/post-execute-task.sh" "yaml_contract" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "YAML sem summary = exit 1" 1 "$rc"
assert_stderr_contains "contract violation summary ausente" "summary" "$stderr"
rm -f "$yaml" "$stderr"

make_prd "status_drift" "| 1.0 | A | pending | — | — | — |"
echo "report" > "$TASKS_BASE/prd-status_drift/1.0_execution_report.md"
mkdir -p "$TASKS_BASE/prd-status_drift/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-status_drift/.checkpoints/1.0.yaml"
yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $AI_TASKS_ROOT/prd-status_drift/1.0_execution_report.md
summary: ok
EOF
stderr=$(mktemp)
bash "$HOOKS_DIR/post-execute-task.sh" "status_drift" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "status drift done vs pending = exit 1" 1 "$rc"
assert_stderr_contains "status drift detectado" "status drift" "$stderr"
rm -f "$yaml" "$stderr"

# ============================================================================
echo
echo "==============================================="
echo "Resultado: $passed asserts OK, $failed asserts FAIL"
echo "==============================================="
if [[ "$failed" -gt 0 ]]; then
  exit 1
fi
exit 0
