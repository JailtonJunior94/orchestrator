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

assert_stderr_not_contains() {
  local desc="$1"
  local pattern="$2"
  local stderr_file="$3"
  if grep -qE "$pattern" "$stderr_file" 2>/dev/null; then
    echo "  ✗ stderr contem '$pattern' (nao deveria)"
    echo "    stderr: $(head -5 "$stderr_file" 2>/dev/null)"
    failed=$((failed+1))
  else
    echo "  ✓ stderr NAO contem '$pattern'"
    passed=$((passed+1))
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
{"schema_version":2,"run_id":"run-hooks","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","patch_ref":"evidence/patch.diff","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"approved"}
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
{"schema_version":2,"run_id":"run-wrapper","task_id":"2.0","attempt":1,"status":"done","base_sha":"0123456789012345678901234567890123456789","patch_sha256":"0123456789012345678901234567890123456789012345678901234567890123","patch_ref":"evidence/patch.diff","final_state_sha256":"0123456789012345678901234567890123456789012345678901234567890123","coverage_regression":false,"tests":[{"command":"go test ./...","exit_code":0,"output_sha256":"0123456789012345678901234567890123456789012345678901234567890123"}],"criteria":[{"id":"AC-01","evidence_ref":"report.md#criterion"}],"evidence":["report.md"],"review_verdict":"approved"}
EOF
wrapper_yaml="status: done
report_path: $wrapper_tasks_root/prd-wrapper/2.0_execution_report.md
summary: resultado versionado"
printf '%b\n' "$wrapper_yaml" | env -u AI_SDD_LEGACY_HOOK_CONTRACT AI_TASKS_ROOT="$wrapper_tasks_root" STRICT_HOOK_FAILURES=1 bash "$HOOKS_DIR/subagent-stop-wrapper.sh" 2>"$stderr"; rc=$?
assert_exit "F02b YAML e encaminhado ao checkpoint JSON v2" 0 "$rc"

echo "--- F02c: stop_hook_active so libera quando e campo top-level ---"
gate_root="$(mktemp -d "$REPO_ROOT/.tmp-test-hooks-gate.XXXXXX")"
gate_tasks_root="${gate_root#"$REPO_ROOT"/}"
mkdir -p "$gate_root/prd-gate"
gate_yaml="status: done
report_path: $gate_tasks_root/prd-gate/2.0_execution_report.md
summary: sem checkpoint versionado"

gate_envelope() {
  AISPEC_GATE_YAML="$gate_yaml" python3 -c '
import json
import os
import sys

envelope = {"hook_event_name": "SubagentStop", "subagent_output": os.environ["AISPEC_GATE_YAML"]}
mode = sys.argv[1]
if mode == "top_level_true":
    envelope["stop_hook_active"] = True
elif mode == "top_level_false":
    envelope["stop_hook_active"] = False
elif mode == "top_level_string":
    envelope["stop_hook_active"] = "true"
elif mode == "top_level_number":
    envelope["stop_hook_active"] = 1
elif mode == "nested":
    envelope["meta"] = {"stop_hook_active": True}
elif mode == "inside_output":
    envelope["subagent_output"] += chr(10) + "nota: \"stop_hook_active\": true"
print(json.dumps(envelope))
' "$1"
}

run_gate() {
  printf '%s' "$(gate_envelope "$1")" \
    | env -u AI_SDD_LEGACY_HOOK_CONTRACT AI_TASKS_ROOT="$gate_tasks_root" STRICT_HOOK_FAILURES=1 \
      bash "$HOOKS_DIR/subagent-stop-wrapper.sh" 2>/dev/null
}

run_gate absent; rc=$?
assert_exit "F02c envelope sem stop_hook_active bloqueia" 2 "$rc"
run_gate inside_output; rc=$?
assert_exit "F02c stop_hook_active no texto do subagente nao libera" 2 "$rc"
run_gate nested; rc=$?
assert_exit "F02c stop_hook_active aninhado nao libera" 2 "$rc"
run_gate top_level_string; rc=$?
assert_exit "F02c stop_hook_active string nao libera" 2 "$rc"
run_gate top_level_number; rc=$?
assert_exit "F02c stop_hook_active numerico nao libera" 2 "$rc"
run_gate top_level_false; rc=$?
assert_exit "F02c stop_hook_active false nao libera" 2 "$rc"
run_gate top_level_true; rc=$?
assert_exit "F02c stop_hook_active top-level true libera" 0 "$rc"
rm -rf "$gate_root"

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

# Opt-out explicito (AI_VALIDATE_GIT_HISTORY=0) deve pular F35.
# O desfecho permanece exit 1: o fixture usa relatorio trivial e o gate RF-53
# do ramo legado nao aceita done sem prova de aprovacao. O que este cenario
# comprova e que a checagem F35 especificamente deixou de disparar.
AI_VALIDATE_GIT_HISTORY=0 bash "$HOOKS_DIR/post-execute-task.sh" "revert" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F35 opt-out (=0) nao dispara F35 mas RF-53 mantem exit 1" 1 "$rc"
assert_stderr_not_contains "F35 pulado pelo opt-out" "FAIL F35" "$stderr"
assert_stderr_contains "RF-53 cobra prova de aprovacao no ramo legado" "FAIL RF-53" "$stderr"
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

# Com env override = WARN. O desfecho permanece exit 1 porque o relatorio do
# fixture nao comprova aprovacao (gate RF-53 do ramo legado).
AI_ALLOW_MISSING_CHECKPOINT=1 bash "$HOOKS_DIR/post-execute-task.sh" "nochkpt" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "F25 com AI_ALLOW_MISSING_CHECKPOINT=1 rebaixa para WARN (RF-53 mantem exit 1)" 1 "$rc"
assert_stderr_contains "WARN F25 detectado em modo back compat" "WARN F25: checkpoint ausente.*back compat" "$stderr"
assert_stderr_not_contains "F25 nao bloqueia com override" "FAIL F25" "$stderr"
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
echo "--- RF-53: AI_SDD_LEGACY_HOOK_CONTRACT nao fecha done sem prova ---"
# ============================================================================
# Regressao do escape: com a variavel em 1 o hook desviava de
# `ai-spec validate-result execution` e o ramo legado nunca cobrava veredito,
# APPROVED ou criterios. Um relatorio sem conteudo nenhum saia com exit 0.
make_prd "legacy_escape" "| 1.0 | A | done | — | — | — |"
mkdir -p "$TASKS_BASE/prd-legacy_escape/.checkpoints"
echo "status: done" > "$TASKS_BASE/prd-legacy_escape/.checkpoints/1.0.yaml"
escape_report=".test-hooks-legacy-escape-report.md"
echo "Nada aqui. Sem mapa 1:1. Sem veredito. Sem nada." > "$REPO_ROOT/$escape_report"
yaml=$(mktemp)
cat > "$yaml" <<EOF
status: done
report_path: $escape_report
summary: ok
EOF
stderr=$(mktemp)
AI_SDD_LEGACY_HOOK_CONTRACT=1 AI_VALIDATE_GIT_HISTORY=0 bash "$HOOKS_DIR/post-execute-task.sh" "legacy_escape" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "RF-53 contrato legado com relatorio trivial = exit 1" 1 "$rc"
assert_stderr_contains "RF-53 cobrado no ramo legado" "FAIL RF-53" "$stderr"

# O caminho estrito (sem a variavel) continua cobrando o mesmo desfecho.
env -u AI_SDD_LEGACY_HOOK_CONTRACT bash "$HOOKS_DIR/post-execute-task.sh" "legacy_escape" "1.0" "$yaml" 2>"$stderr"; rc=$?
assert_exit "RF-53 contrato estrito com relatorio trivial = exit 1" 1 "$rc"
rm -f "$yaml" "$stderr" "$REPO_ROOT/$escape_report"

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
# Bloco 4: git-operation-gate.sh — RF-57 (parsing sem truncamento) e RF-68
# (negacao por ausencia de alvo). Cobre a prova de bypass do ADR-003 e a
# postura correta de comando vazio (alinhada a validate-preload.sh:109).
# ============================================================================
echo
echo "Bloco 4: git-operation-gate.sh"

GIT_GATE_HOOK="$REPO_ROOT/.agents/scripts/git-operation-gate.sh"
GIT_GATE_TMP=$(mktemp -d "$TMP_BASE/git-gate.XXXXXX" 2>/dev/null || mktemp -d /tmp/git-gate.XXXXXX)

echo
echo "  G4-1: comando benigno passa (exit 0)"
printf '%s' '{"tool_input":{"command":"echo hello"}}' | bash "$GIT_GATE_HOOK" >/dev/null 2>&1
assert_exit "G4-1: comando benigno" 0 $?

echo
echo "  G4-2: git push nao solicitado bloqueia (exit 2)"
printf '%s' '{"tool_input":{"command":"git push origin main"}}' | bash "$GIT_GATE_HOOK" >/dev/null 2>&1
assert_exit "G4-2: git push bloqueia" 2 $?

echo
echo "  G4-3: git push com GOVERNANCE_GIT_OPERATION_CONFIRMED=1 passa (exit 0)"
printf '%s' '{"tool_input":{"command":"git push origin main"}}' \
  | GOVERNANCE_GIT_OPERATION_CONFIRMED=1 bash "$GIT_GATE_HOOK" >/dev/null 2>&1
assert_exit "G4-3: opt-out explicito passa" 0 $?

echo
echo "  G4-4: comando vazio nega (RF-68) — regressao do exit 0 antigo"
printf '%s' '{}' | bash "$GIT_GATE_HOOK" >/dev/null 2>&1
assert_exit "G4-4: comando ausente bloqueia" 2 $?

echo
echo "  G4-5: payload truncavel (~70KB de padding + git push) bloqueia (ADR-003)"
git_gate_exploit="$GIT_GATE_TMP/exploit.json"
python3 -c "
import json
padding = 'A' * 70000
command = padding + '; echo hi ; git push origin main'
print(json.dumps({'tool_input': {'command': command}}))
" >"$git_gate_exploit"
bash "$GIT_GATE_HOOK" <"$git_gate_exploit" >/dev/null 2>&1
assert_exit "G4-5: payload grande com git push bloqueia" 2 $?
rm -f "$git_gate_exploit"

echo
echo "  G4-6: JSON invalido bloqueia (falha fechada, sem fallback grep)"
printf 'nao e json' | bash "$GIT_GATE_HOOK" >/dev/null 2>&1
assert_exit "G4-6: JSON invalido bloqueia" 2 $?

rm -rf "$GIT_GATE_TMP"

# ============================================================================
# Bloco 4b: hook-payload.sh — RF-66 (timeout declarado, negacao no estouro) e
# RF-67 (recursao hook -> ferramenta -> hook impedida por construcao). Cobre
# as primitivas hook_timeout_watch, hook_recursion_guard e hook_measure_start
# consumidas por validate-preload.sh, git-operation-gate.sh e
# validate-governance.sh.
# ============================================================================
echo
echo "Bloco 4b: hook-payload.sh — timeout e guarda de recursao"

HOOK_LIB="$REPO_ROOT/.agents/lib/hook-payload.sh"
GUARD_TMP=$(mktemp -d "$TMP_BASE/hook-guard.XXXXXX" 2>/dev/null || mktemp -d /tmp/hook-guard.XXXXXX)

echo
echo "  P7-1: hook_timeout_watch mata filho lento e trata como negacao (RF-66)"
SLOW_CHILD="$GUARD_TMP/slow-child.sh"
cat > "$SLOW_CHILD" <<'EOF'
#!/usr/bin/env bash
sleep 5
echo "nao deveria imprimir"
exit 0
EOF
chmod +x "$SLOW_CHILD"

TIMEOUT_CALLER="$GUARD_TMP/timeout-caller.sh"
cat > "$TIMEOUT_CALLER" <<EOF
#!/usr/bin/env bash
set -euo pipefail
source "$HOOK_LIB"
bash "$SLOW_CHILD" &
child_pid=\$!
if hook_timeout_watch "fake-slow-child" 1 "\$child_pid"; then
  exit 0
else
  exit 2
fi
EOF
chmod +x "$TIMEOUT_CALLER"

p71_start=$(date +%s)
stderr_p71=$(mktemp)
bash "$TIMEOUT_CALLER" >/dev/null 2>"$stderr_p71"
rc_p71=$?
p71_end=$(date +%s)
p71_elapsed=$((p71_end - p71_start))

assert_exit "P7-1: filho lento negado apos timeout" 2 "$rc_p71"
assert_stderr_contains "P7-1: mensagem de bloqueio por timeout (RF-66)" "excedeu timeout declarado" "$stderr_p71"
if [[ "$p71_elapsed" -le 3 ]]; then
  echo "  ✓ P7-1: bloqueio ocorreu em ${p71_elapsed}s (timeout declarado de 1s), nao esperou os 5s do filho"
  passed=$((passed+1))
else
  echo "  ✗ P7-1: bloqueio demorou ${p71_elapsed}s (esperado <=3s) — timeout nao preemptou o filho"
  failed=$((failed+1))
fi
rm -f "$stderr_p71"

echo
echo "  P7-2: hook_recursion_guard impede cadeia hook -> ferramenta -> hook (RF-67)"
RECURSIVE_HOOK="$GUARD_TMP/recursive-hook.sh"
cat > "$RECURSIVE_HOOK" <<EOF
#!/usr/bin/env bash
set -euo pipefail
source "$HOOK_LIB"
hook_recursion_guard "fake-recursive-hook" 2
echo "depth-now=\$AI_HOOK_DEPTH" >&2
bash "\$0"
exit 0
EOF
chmod +x "$RECURSIVE_HOOK"

stderr_p72=$(mktemp)
AI_HOOK_MAX_DEPTH=3 bash "$RECURSIVE_HOOK" >/dev/null 2>"$stderr_p72"
rc_p72=$?

assert_exit "P7-2: recursao bloqueada com exit declarado do hook" 2 "$rc_p72"
assert_stderr_contains "P7-2: mensagem de bloqueio por recursao (RF-67)" "recursao hook -> ferramenta -> hook detectada" "$stderr_p72"
depth_entries=$(grep -c "depth-now=" "$stderr_p72" 2>/dev/null || echo 0)
if [[ "$depth_entries" -eq 3 ]]; then
  echo "  ✓ P7-2: exatamente 3 entradas permitidas antes do bloqueio (AI_HOOK_MAX_DEPTH=3)"
  passed=$((passed+1))
else
  echo "  ✗ P7-2: $depth_entries entrada(s) permitida(s) antes do bloqueio (esperado 3)"
  failed=$((failed+1))
fi
rm -f "$stderr_p72"

echo
echo "  P7-3: git-operation-gate.sh real declara timeout e profundidade sem regressao (comando benigno)"
stderr_p73=$(mktemp)
printf '%s' '{"tool_input":{"command":"echo hello"}}' | bash "$GIT_GATE_HOOK" >/dev/null 2>"$stderr_p73"
rc_p73=$?
assert_exit "P7-3: git-operation-gate.sh comando benigno ainda passa" 0 "$rc_p73"
assert_stderr_contains "P7-3: duracao mensuravel emitida (RF-66)" "hook\.duration_ms=" "$stderr_p73"
rm -f "$stderr_p73"

echo
echo "  P7-4: git-operation-gate.sh real nega quando profundidade ja excede o maximo (RF-67)"
stderr_p74=$(mktemp)
printf '%s' '{"tool_input":{"command":"echo hello"}}' \
  | AI_HOOK_DEPTH=3 AI_HOOK_MAX_DEPTH=3 bash "$GIT_GATE_HOOK" >/dev/null 2>"$stderr_p74"
rc_p74=$?
assert_exit "P7-4: profundidade pre-excedida bloqueia mesmo comando benigno" 2 "$rc_p74"
assert_stderr_contains "P7-4: mensagem de recursao no hook real" "recursao hook -> ferramenta -> hook detectada em git-operation-gate" "$stderr_p74"
rm -f "$stderr_p74"

echo
echo "  P7-5: hook_timeout_watch nao deixa processo watchdog orfao quando o filho termina antes do orcamento (regressao RF61Scenario14/HOOKTIMEOUT-2)"
FAST_CHILD="$GUARD_TMP/fast-child.sh"
cat > "$FAST_CHILD" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$FAST_CHILD"

FAST_TIMEOUT_SECONDS=137
FAST_CALLER="$GUARD_TMP/fast-timeout-caller.sh"
cat > "$FAST_CALLER" <<EOF
#!/usr/bin/env bash
set -euo pipefail
source "$HOOK_LIB"
bash "$FAST_CHILD" &
child_pid=\$!
hook_timeout_watch "fake-fast-child" $FAST_TIMEOUT_SECONDS "\$child_pid"
exit \$?
EOF
chmod +x "$FAST_CALLER"

p75_start=$(date +%s)
bash "$FAST_CALLER" >/dev/null 2>/dev/null
rc_p75=$?
p75_end=$(date +%s)
p75_elapsed=$((p75_end - p75_start))

assert_exit "P7-5: filho rapido aprovado sem esperar o orcamento" 0 "$rc_p75"
if [[ "$p75_elapsed" -le 3 ]]; then
  echo "  ✓ P7-5: retorno em ${p75_elapsed}s (orcamento declarado de ${FAST_TIMEOUT_SECONDS}s), nao bloqueou ate o fim do orcamento"
  passed=$((passed+1))
else
  echo "  ✗ P7-5: retorno demorou ${p75_elapsed}s (esperado <=3s) — hook_timeout_watch bloqueou alem do necessario"
  failed=$((failed+1))
fi

if pgrep -f "sleep $FAST_TIMEOUT_SECONDS" >/dev/null 2>&1; then
  echo "  ✗ P7-5: processo watchdog orfao (sleep $FAST_TIMEOUT_SECONDS) ainda vivo apos hook_timeout_watch retornar — fd de stdout/stderr fica aberto e trava Cmd.Wait() do chamador (RF61Scenario14)"
  failed=$((failed+1))
  pkill -f "sleep $FAST_TIMEOUT_SECONDS" >/dev/null 2>&1 || true
else
  echo "  ✓ P7-5: nenhum processo watchdog orfao (sleep $FAST_TIMEOUT_SECONDS) remanescente"
  passed=$((passed+1))
fi

rm -rf "$GUARD_TMP"

# ============================================================================
# Bloco 5: post-wave.sh — RF-40 (atomicidade), RF-41 (idempotencia) e RF-49
# (sanitizacao do YAML embutido no checkpoint parcial).
# ============================================================================
echo
echo "Bloco 5: post-wave.sh"

# Bloco 3 exporta PATH="$TMP_BASE/bin:$PATH" com um shim de `git` que responde
# `rev-parse --show-toplevel` com o TMP_REPO_ROOT (ja removido) daquele bloco,
# independente do cwd real. post-wave.sh depende de `git rev-parse
# --show-toplevel` real; usar um PATH limpo aqui evita herdar esse shim.
REAL_BIN_PATH="/usr/bin:/bin:/opt/homebrew/bin:/usr/local/bin"

POST_WAVE_HOOK="$REPO_ROOT/.agents/hooks/post-wave.sh"
POST_WAVE_TMP=$(mktemp -d "$TMP_BASE/post-wave.XXXXXX" 2>/dev/null || mktemp -d /tmp/post-wave.XXXXXX)
mkdir -p "$POST_WAVE_TMP/.specs/prd-postwave"
(cd "$POST_WAVE_TMP" && PATH="$REAL_BIN_PATH" git init -q .)
printf 'status: done\ntoken: ghp_1234567890abcdef1234567890\n' >"$POST_WAVE_TMP/results.yaml"
PARTIAL_MD_PW="$POST_WAVE_TMP/.specs/prd-postwave/_orchestration_report.partial.md"

echo
echo "  H5-1: primeira wave cria o parcial (exit 0)"
(cd "$POST_WAVE_TMP" && AI_TASKS_ROOT=.specs PATH="$REAL_BIN_PATH" bash "$POST_WAVE_HOOK" postwave 1.0 results.yaml >/dev/null 2>&1)
assert_exit "H5-1: primeira wave, exit 0" 0 $?

echo
echo "  H5-2: reexecutar a mesma wave nao duplica a secao (RF-41)"
(cd "$POST_WAVE_TMP" && AI_TASKS_ROOT=.specs PATH="$REAL_BIN_PATH" bash "$POST_WAVE_HOOK" postwave 1.0 results.yaml >/dev/null 2>&1)
assert_exit "H5-2: reexecucao idempotente, exit 0" 0 $?
wave_count=$(grep -c "### Wave 1.0" "$PARTIAL_MD_PW" 2>/dev/null || echo 0)
if [[ "$wave_count" -eq 1 ]]; then
  echo "  ✓ H5-2: secao da wave 1.0 aparece exatamente 1 vez"
  passed=$((passed+1))
else
  echo "  ✗ H5-2: secao da wave 1.0 aparece $wave_count vezes (esperado 1)"
  failed=$((failed+1))
fi

echo
echo "  H5-3: segredo do YAML anexado nunca aparece em claro (RF-49)"
if grep -q "ghp_1234567890abcdef1234567890" "$PARTIAL_MD_PW" 2>/dev/null; then
  echo "  ✗ H5-3: segredo vazou para o relatorio parcial"
  failed=$((failed+1))
else
  echo "  ✓ H5-3: segredo nao vazou"
  passed=$((passed+1))
fi
if grep -q "REDACTED:provider_token" "$PARTIAL_MD_PW" 2>/dev/null; then
  echo "  ✓ H5-3: marcador de redacao presente"
  passed=$((passed+1))
else
  echo "  ✗ H5-3: marcador de redacao ausente"
  failed=$((failed+1))
fi

echo
echo "  H5-5: URL com credencial embutida no YAML e redigida (RF-49, connection_string_credential)"
printf 'status: done\nconn: postgres://admin:hunter2@db.internal:5432/app\n' >"$POST_WAVE_TMP/results-conn.yaml"
(cd "$POST_WAVE_TMP" && AI_TASKS_ROOT=.specs PATH="$REAL_BIN_PATH" bash "$POST_WAVE_HOOK" postwave 1.1 results-conn.yaml >/dev/null 2>&1)
if grep -q "admin:hunter2@" "$PARTIAL_MD_PW" 2>/dev/null; then
  echo "  ✗ H5-5: credencial da URL vazou para o relatorio parcial"
  failed=$((failed+1))
else
  echo "  ✓ H5-5: credencial da URL nao vazou"
  passed=$((passed+1))
fi
if grep -q "REDACTED:connection_string_credential" "$PARTIAL_MD_PW" 2>/dev/null; then
  echo "  ✓ H5-5: marcador de redacao presente"
  passed=$((passed+1))
else
  echo "  ✗ H5-5: marcador de redacao ausente"
  failed=$((failed+1))
fi

echo
echo "  H5-4: nao sobra temporario .tmp-post-wave apos as execucoes"
leftover_tmp=$(find "$POST_WAVE_TMP/.specs/prd-postwave" -maxdepth 1 -name ".tmp-post-wave.*" 2>/dev/null | wc -l | tr -d ' ')
if [[ "$leftover_tmp" -eq 0 ]]; then
  echo "  ✓ H5-4: nenhum temporario remanescente"
  passed=$((passed+1))
else
  echo "  ✗ H5-4: $leftover_tmp temporario(s) remanescente(s)"
  failed=$((failed+1))
fi

rm -rf "$POST_WAVE_TMP"

# ============================================================================
# Bloco 6: extensao unica do checkpoint entre os quatro consumidores (RF-39,
# RF-43). Falha se qualquer um divergir de ".json".
# ============================================================================
echo
echo "Bloco 6: extensao do checkpoint (.checkpoints/<id>.EXT)"

CKPT_SKILL="$REPO_ROOT/.agents/skills/execute-task/SKILL.md"
CKPT_STATE_GO="$REPO_ROOT/internal/sdd/state.go"
CKPT_WRAPPER="$REPO_ROOT/.agents/hooks/subagent-stop-wrapper.sh"
CKPT_GATE="$REPO_ROOT/.agents/hooks/post-execute-task.sh"

echo
echo "  K6-1: execute-task/SKILL.md declara extensao .json"
if grep -qE '\.checkpoints/<num>\.json' "$CKPT_SKILL" && ! grep -qE '\.checkpoints/<num>\.yaml' "$CKPT_SKILL"; then
  echo "  ✓ K6-1: SKILL.md usa .json"
  passed=$((passed+1))
else
  echo "  ✗ K6-1: SKILL.md nao usa .json exclusivamente"
  failed=$((failed+1))
fi

echo
echo "  K6-2: internal/sdd/state.go le .checkpoints/<taskID>.json"
if grep -qE '\.checkpoints",\s*taskID\+"\.json"' "$CKPT_STATE_GO"; then
  echo "  ✓ K6-2: state.go le .json"
  passed=$((passed+1))
else
  echo "  ✗ K6-2: state.go nao le .json"
  failed=$((failed+1))
fi

echo
echo "  K6-3: subagent-stop-wrapper.sh procura .checkpoints/\${task_id}.json"
if grep -qE '\.checkpoints/\$\{task_id\}\.json' "$CKPT_WRAPPER"; then
  echo "  ✓ K6-3: wrapper procura .json"
  passed=$((passed+1))
else
  echo "  ✗ K6-3: wrapper nao procura .json"
  failed=$((failed+1))
fi

echo
echo "  K6-4: post-execute-task.sh (gate F25) checa .checkpoints/\${TASK_ID}.json"
if grep -qE '\.checkpoints/\$\{TASK_ID\}\.json' "$CKPT_GATE" && ! grep -qE '\.checkpoints/\$\{TASK_ID\}\.yaml' "$CKPT_GATE"; then
  echo "  ✓ K6-4: gate F25 checa .json exclusivamente"
  passed=$((passed+1))
else
  echo "  ✗ K6-4: gate F25 nao checa .json exclusivamente"
  failed=$((failed+1))
fi

# ============================================================================
echo
echo "==============================================="
echo "Resultado: $passed asserts OK, $failed asserts FAIL"
echo "==============================================="
if [[ "$failed" -gt 0 ]]; then
  exit 1
fi
exit 0
