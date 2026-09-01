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
TASKS_BASE="$TMP_BASE/.spec"
mkdir -p "$TASKS_BASE"

# Garante que os hooks testem o binario construido da working tree atual, nao
# uma versao Homebrew possivelmente desatualizada no PATH.
AI_SPEC_BIN="$REPO_ROOT/ai-spec"
export AI_SPEC_BIN
(cd "$REPO_ROOT" && go build -o ./ai-spec .)
mkdir -p "$TMP_BASE/bin"

# Os cenarios historicos abaixo exercitam somente o fallback YAML opt-in. O
# contrato estrito JSON v2 e validado explicitamente no primeiro cenario F02.
export AI_SDD_LEGACY_HOOK_CONTRACT=1

# Override env vars para fixtures temporarios.
AI_TASKS_ROOT="$(realpath --relative-to="$REPO_ROOT" "$TASKS_BASE" 2>/dev/null || python3 -c "import os; print(os.path.relpath('$TASKS_BASE', '$REPO_ROOT'))")"
export AI_TASKS_ROOT
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
  prd_hash="$($AI_SPEC_BIN hash "$dir/prd.md")"

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

# ============================================================================
echo "--- F02: adaptador JSON estrito ---"
# ============================================================================
strict_result=$(mktemp)
cat > "$strict_result" <<EOF
{"schema_version":2,"run_id":"run-hooks","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"approved"}
EOF
stderr=$(mktemp)
env -u AI_SDD_LEGACY_HOOK_CONTRACT bash "$HOOKS_DIR/post-execute-task.sh" "hooks" "2.0" "$strict_result" 2>"$stderr"; rc=$?
assert_exit "F02 resultado JSON completo = exit 0" 0 "$rc"
printf '{}' > "$strict_result"
env -u AI_SDD_LEGACY_HOOK_CONTRACT bash "$HOOKS_DIR/post-execute-task.sh" "hooks" "2.0" "$strict_result" 2>"$stderr"; rc=$?
assert_exit "F02 resultado JSON incompleto = exit 1" 1 "$rc"
rm -f "$strict_result" "$stderr"

# O wrapper deve converter o envelope YAML do subagent no checkpoint JSON SDD
# correspondente antes de chamar o hook estrito.
echo "--- F02b: wrapper encaminha checkpoint JSON v2 ---"
wrapper_root="$(mktemp -d "$REPO_ROOT/.tmp-test-hooks-wrapper.XXXXXX")"
wrapper_tasks_root="${wrapper_root#"$REPO_ROOT"/}"
wrapper_dir="$wrapper_root/prd-wrapper"
mkdir -p "$wrapper_dir/.checkpoints"
cat > "$wrapper_dir/.checkpoints/2.0.json" <<EOF
{"schema_version":2,"run_id":"run-wrapper","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"approved"}
EOF
wrapper_yaml="status: done
report_path: $wrapper_tasks_root/prd-wrapper/2.0_execution_report.md
summary: resultado versionado"
printf '%b\n' "$wrapper_yaml" | env -u AI_SDD_LEGACY_HOOK_CONTRACT AI_TASKS_ROOT="$wrapper_tasks_root" STRICT_HOOK_FAILURES=1 bash "$HOOKS_DIR/subagent-stop-wrapper.sh" 2>"$stderr"; rc=$?
assert_exit "F02b YAML e encaminhado ao checkpoint JSON v2" 0 "$rc"
rm -rf "$wrapper_root"
rm -f "$stderr"

# Os cenários do bloco 3 instalam shims de git/ai-spec neste diretório para
# controlar somente o pre-commit. O contrato SDD estrito acima já usou
# explicitamente AI_SPEC_BIN apontando ao binário local construído.
export PATH="$TMP_BASE/bin:$PATH"

echo

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
revert_report=".test-hooks-revert-report.md"
cat > "$REPO_ROOT/$revert_report" <<EOF
# Report

sha=deadbeefcafe1234567890abcdef1234567890ab
verdict=APPROVED
EOF
yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $revert_report
summary: ok
EOF
# Criar checkpoint pra evitar F25 FAIL
mkdir -p "$TASKS_BASE/prd-revert/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-revert/.checkpoints/1.0.yaml"

stderr=$(mktemp)
AI_VALIDATE_GIT_HISTORY=1 bash "$HOOKS_DIR/post-execute-task.sh" "revert" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F35 com SHA fake = exit 1" 1 "$rc"
assert_stderr_contains "FAIL F35 detectada" "FAIL F35: DiffSHA deadbeef" "$stderr"

# Default-on (RF-04): sem env explicito F35 ainda dispara no SHA fake
bash "$HOOKS_DIR/post-execute-task.sh" "revert" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F35 default-on (sem env) = exit 1" 1 "$rc"

# Opt-out explicito (AI_VALIDATE_GIT_HISTORY=0) deve pular F35 e passar
AI_VALIDATE_GIT_HISTORY=0 bash "$HOOKS_DIR/post-execute-task.sh" "revert" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F35 opt-out (=0) = exit 0 (skip)" 0 "$rc"
rm -f "$stderr" "$yaml" "$REPO_ROOT/$revert_report"

# ============================================================================
echo
echo "--- F13: containment de report_path ---"
# ============================================================================
make_prd "path_containment" "| 1.0 | A | done | — | — | — |"
mkdir -p "$TASKS_BASE/prd-path_containment/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-path_containment/.checkpoints/1.0.yaml"
echo "report seguro" > "$TASKS_BASE/prd-path_containment/1.0_execution_report.md"
yaml=$(mktemp)
stderr=$(mktemp)

cat > "$yaml" <<EOF
status: done
report_path: /tmp/report.md
summary: ok
EOF
bash "$HOOKS_DIR/post-execute-task.sh" "path_containment" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F13 path absoluto = exit 1" 1 "$rc"
assert_stderr_contains "F13 path absoluto detectado" "FAIL F13: report_path" "$stderr"

cat > "$yaml" <<EOF
status: done
report_path: ../fora.md
summary: ok
EOF
bash "$HOOKS_DIR/post-execute-task.sh" "path_containment" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F13 traversal = exit 1" 1 "$rc"
assert_stderr_contains "F13 traversal detectado" "path absoluto ou traversal" "$stderr"

outside_report="$TMP_BASE/outside-report.md"
echo "não deve ser aceito" > "$outside_report"
symlink_report="$REPO_ROOT/.test-hook-external-report-link.md"
ln -s "$outside_report" "$symlink_report"
cat > "$yaml" <<EOF
status: done
report_path: .test-hook-external-report-link.md
summary: ok
EOF
bash "$HOOKS_DIR/post-execute-task.sh" "path_containment" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F13 symlink externo = exit 1" 1 "$rc"
assert_stderr_contains "F13 symlink externo detectado" "resolve fora do repositório" "$stderr"
rm -f "$symlink_report" "$yaml" "$stderr"

# ============================================================================
echo
echo "--- F25: checkpoint ausente bloqueia (default FAIL) ---"
# ============================================================================
make_prd "nochkpt" "| 1.0 | A | done | — | — | — |"
nochkpt_report=".test-hooks-nochkpt-report.md"
echo "report" > "$REPO_ROOT/$nochkpt_report"
yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $nochkpt_report
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
rm -f "$stderr" "$yaml" "$REPO_ROOT/$nochkpt_report"

# ============================================================================
echo
echo "--- Contrato YAML e status drift ---"
# ============================================================================
make_prd "yaml_contract" "| 1.0 | A | done | — | — | — |"
yaml_contract_report=".test-hooks-yaml-contract-report.md"
echo "report" > "$REPO_ROOT/$yaml_contract_report"
mkdir -p "$TASKS_BASE/prd-yaml_contract/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-yaml_contract/.checkpoints/1.0.yaml"

yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $yaml_contract_report
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
rm -f "$yaml" "$stderr" "$REPO_ROOT/$yaml_contract_report"

make_prd "status_drift" "| 1.0 | A | pending | — | — | — |"
status_drift_report=".test-hooks-status-drift-report.md"
echo "report" > "$REPO_ROOT/$status_drift_report"
mkdir -p "$TASKS_BASE/prd-status_drift/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-status_drift/.checkpoints/1.0.yaml"
yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $status_drift_report
summary: ok
EOF
stderr=$(mktemp)
bash "$HOOKS_DIR/post-execute-task.sh" "status_drift" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "status drift done vs pending = exit 1" 1 "$rc"
assert_stderr_contains "status drift detectado" "status drift" "$stderr"
rm -f "$yaml" "$stderr" "$REPO_ROOT/$status_drift_report"

# ============================================================================
echo
echo "--- Bloco 3: spec-drift gate no pre-commit hook ---"
# ============================================================================
#
# Estratégia: criar um TMP_REPO_ROOT com estrutura mínima de PRD e sobrescrever
# git e ai-spec no PATH com shims controlados por variáveis de ambiente.
#
# STAGED_FILES: arquivo de texto com o conteúdo que o shim de `git` retorna em
#               `git diff --cached --name-only`.
# AISPEC_EXIT:  código de saída que o shim de `ai-spec check-spec-drift` retorna.
# AISPEC_VERSION_OUTPUT: saída que o shim retorna para `ai-spec version`.
#
# O shim de `git`:
#   rev-parse --show-toplevel  → TMP_REPO_ROOT
#   diff --cached --name-only  → conteúdo de STAGED_FILES
#   qualquer outro subcomando  → delegado ao git real (necessário para grep/cd)

TMP_REPO_ROOT=$(mktemp -d /tmp/test-hooks-repo.XXXXXX)
STAGED_FILES=$(mktemp /tmp/staged-files.XXXXXX)
AISPEC_EXIT_FILE=$(mktemp /tmp/aispec-exit.XXXXXX)
AISPEC_VERSION_FILE=$(mktemp /tmp/aispec-version.XXXXXX)
export STAGED_FILES AISPEC_EXIT_FILE AISPEC_VERSION_FILE TMP_REPO_ROOT

# Shim de git: controla rev-parse e diff --cached
cat > "$TMP_BASE/bin/git" <<'GITSHIM'
#!/usr/bin/env bash
if [[ "$1" == "rev-parse" && "$2" == "--show-toplevel" ]]; then
  echo "$TMP_REPO_ROOT"
elif [[ "$1" == "diff" && "$2" == "--cached" && "$3" == "--name-only" ]]; then
  cat "$STAGED_FILES" 2>/dev/null
else
  /usr/bin/git "$@"
fi
GITSHIM
chmod +x "$TMP_BASE/bin/git"

# Shim de ai-spec para o bloco 3: controla version e check-spec-drift
cat > "$TMP_BASE/bin/ai-spec" <<'SPECSHIM'
#!/usr/bin/env bash
if [[ "$1" == "version" ]]; then
  cat "$AISPEC_VERSION_FILE" 2>/dev/null || echo "v0.21.0"
elif [[ "$1" == "check-spec-drift" ]]; then
  exit_code=$(cat "$AISPEC_EXIT_FILE" 2>/dev/null || echo "0")
  exit "$exit_code"
elif [[ "$1" == "hash" ]]; then
  # Para o helper make_prd que já existe na suíte
  cd "$REPO_ROOT" && go run . hash "$2"
else
  cd "$REPO_ROOT" && go run . "$@"
fi
SPECSHIM
chmod +x "$TMP_BASE/bin/ai-spec"

# Shim de ai-spec ausente (para cenário "sem ai-spec no PATH")
# Contém apenas o shim de git, sem ai-spec — simula PATH sem o binário.
NOSPEC_BIN=$(mktemp -d /tmp/nospec-bin.XXXXXX)
cat > "$NOSPEC_BIN/git" <<'GITSHIM2'
#!/usr/bin/env bash
if [[ "$1" == "rev-parse" && "$2" == "--show-toplevel" ]]; then
  echo "$TMP_REPO_ROOT"
elif [[ "$1" == "diff" && "$2" == "--cached" && "$3" == "--name-only" ]]; then
  cat "$STAGED_FILES" 2>/dev/null
else
  /usr/bin/git "$@"
fi
GITSHIM2
chmod +x "$NOSPEC_BIN/git"

PRE_COMMIT_HOOK="$REPO_ROOT/scripts/git-hooks/pre-commit"

run_hook_with_staged() {
  local staged_content="$1"
  local aispec_version="${2:-v0.21.0}"
  local aispec_exit="${3:-0}"
  printf '%s' "$staged_content" > "$STAGED_FILES"
  printf '%s' "$aispec_version" > "$AISPEC_VERSION_FILE"
  printf '%s' "$aispec_exit" > "$AISPEC_EXIT_FILE"
  bash "$PRE_COMMIT_HOOK"
}

# ────────────────────────────────────────────────────────────
# Cenário B3-1: staged não inclui prd.md/techspec.md → bloco 3 não dispara
# ────────────────────────────────────────────────────────────
echo
echo "  B3-1: staged apenas README.md → bloco 3 não dispara, exit 0"
printf 'README.md\n' > "$STAGED_FILES"
printf 'v0.21.0' > "$AISPEC_VERSION_FILE"
printf '0' > "$AISPEC_EXIT_FILE"
mkdir -p "$TMP_REPO_ROOT/.specs/prd-b3test1"
run_hook_with_staged "README.md" > /dev/null 2>&1; rc_b31=$?
assert_exit "B3-1: staged=README exit 0" 0 "$rc_b31"

# ────────────────────────────────────────────────────────────
# Cenário B3-2: prd.md staged, ai-spec retorna exit 0 → sem drift → exit 0
# ────────────────────────────────────────────────────────────
echo
echo "  B3-2: prd.md staged, sem drift → exit 0 + msg 'spec-drift OK'"
mkdir -p "$TMP_REPO_ROOT/.specs/prd-b3test2"
touch "$TMP_REPO_ROOT/.specs/prd-b3test2/tasks.md"
run_hook_with_staged ".specs/prd-b3test2/prd.md" "v0.21.0" "0" > /dev/null 2>&1; rc_b32=$?
assert_exit "B3-2: sem drift, exit 0" 0 "$rc_b32"

# ────────────────────────────────────────────────────────────
# Cenário B3-3: prd.md staged, ai-spec retorna exit 1 → drift → exit 1 + remediação
# ────────────────────────────────────────────────────────────
echo
echo "  B3-3: prd.md staged, drift detectado → exit 1 + comando de remediação"
mkdir -p "$TMP_REPO_ROOT/.specs/prd-b3test3"
touch "$TMP_REPO_ROOT/.specs/prd-b3test3/tasks.md"
stderr_b33=$(mktemp)
printf '.specs/prd-b3test3/prd.md\n' > "$STAGED_FILES"
printf 'v0.21.0' > "$AISPEC_VERSION_FILE"
printf '1' > "$AISPEC_EXIT_FILE"
bash "$PRE_COMMIT_HOOK" 2>"$stderr_b33"; rc_b33=$?
assert_exit "B3-3: drift, exit 1" 1 "$rc_b33"
assert_stderr_contains "B3-3: mensagem de remediação" "ai-spec sync-spec-hash" "$stderr_b33"
rm -f "$stderr_b33"

# ────────────────────────────────────────────────────────────
# Cenário B3-4: ai-spec ausente do PATH → warn em stderr + exit 0
# ────────────────────────────────────────────────────────────
echo
echo "  B3-4: ai-spec ausente no PATH → warn + exit 0"
mkdir -p "$TMP_REPO_ROOT/.specs/prd-b3test4"
touch "$TMP_REPO_ROOT/.specs/prd-b3test4/tasks.md"
printf '.specs/prd-b3test4/prd.md\n' > "$STAGED_FILES"
stderr_b34=$(mktemp)
# Rodar com PATH que tem git shim mas sem ai-spec
PATH="$NOSPEC_BIN:/usr/bin:/bin" \
  bash "$PRE_COMMIT_HOOK" 2>"$stderr_b34"; rc_b34=$?
assert_exit "B3-4: sem ai-spec, exit 0" 0 "$rc_b34"
assert_stderr_contains "B3-4: warn ai-spec ausente" "ai-spec não encontrado" "$stderr_b34"
rm -f "$stderr_b34"

# ────────────────────────────────────────────────────────────
# Cenário B3-5: ai-spec versão antiga (v0.10.0 < v0.21.0) → warn + exit 0
# ────────────────────────────────────────────────────────────
echo
echo "  B3-5: ai-spec versão antiga v0.10.0 → warn + exit 0"
mkdir -p "$TMP_REPO_ROOT/.specs/prd-b3test5"
touch "$TMP_REPO_ROOT/.specs/prd-b3test5/tasks.md"
printf '.specs/prd-b3test5/prd.md\n' > "$STAGED_FILES"
printf 'v0.10.0' > "$AISPEC_VERSION_FILE"
stderr_b35=$(mktemp)
bash "$PRE_COMMIT_HOOK" 2>"$stderr_b35"; rc_b35=$?
assert_exit "B3-5: versão antiga, exit 0" 0 "$rc_b35"
assert_stderr_contains "B3-5: warn versão < MIN" "v0.10.0" "$stderr_b35"
rm -f "$stderr_b35"

# ────────────────────────────────────────────────────────────
# Limpeza dos temporários do bloco 3
# ────────────────────────────────────────────────────────────
rm -rf "$TMP_REPO_ROOT" "$STAGED_FILES" "$AISPEC_EXIT_FILE" "$AISPEC_VERSION_FILE" "$NOSPEC_BIN"

# ============================================================================
echo
echo "==============================================="
echo "Resultado: $passed asserts OK, $failed asserts FAIL"
echo "==============================================="
if [[ "$failed" -gt 0 ]]; then
  exit 1
fi
exit 0
