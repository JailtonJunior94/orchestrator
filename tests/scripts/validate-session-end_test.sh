#!/usr/bin/env bash
set -euo pipefail

validator_arg="${1:-.agents/scripts/validate-session-end.sh}"
validator="$(cd "$(dirname "$validator_arg")" && pwd)/$(basename "$validator_arg")"

fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

project="$fixture/project"
mkdir -p "$project/.specs/prd-gate/"
git -C "$project" init -q .
prd="$project/.specs/prd-gate"

cat >"$prd/tasks.md" <<'EOF'
# Tasks

| ID | Description | Status |
|----|-----------|--------|
| 1.0 | Active task | in_progress |
EOF

run_validator() {
  local output_mode="$1"
  ( cd "$project" && AISPEC_HOOK_DECISION_OUTPUT="$output_mode" bash "$validator" </dev/null >"$fixture/stdout" 2>"$fixture/stderr" )
}

expect_exit() {
  local want="$1" got="$2" label="$3"
  if [[ "$got" != "$want" ]]; then
    echo "validate-session-end: $label expected exit $want, got $got" >&2
    cat "$fixture/stderr" >&2
    exit 1
  fi
}

rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "in_progress task without execution report"
grep -F 'GATE DE ENCERRAMENTO BLOQUEADO' "$fixture/stderr" >/dev/null
grep -F 'tarefa ativa sem relatorio de execucao: 1.0' "$fixture/stderr" >/dev/null

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0
Veredito: CHANGES_REQUESTED
EOF

rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "report without APPROVED verdict"
grep -F 'tarefa ativa sem veredito APPROVED registrado: 1.0' "$fixture/stderr" >/dev/null

rc=0; run_validator json || rc=$?
expect_exit 2 "$rc" "block decision with json output"
grep -F '"decision":"block"' "$fixture/stdout" >/dev/null

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0
Veredito: APPROVED
EOF

rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "report with APPROVED verdict"

rc=0; run_validator json || rc=$?
expect_exit 0 "$rc" "release decision with json output"
if grep -F '"decision":"block"' "$fixture/stdout" >/dev/null 2>&1; then
  echo "validate-session-end: release path emitted a block decision" >&2
  exit 1
fi

sed -i.bak 's/| 1.0 | Active task | in_progress |/| 1.0 | Active task | done |/' "$prd/tasks.md"
rm -f "$prd/1.0_execution_report.md" "$prd/tasks.md.bak"
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "task closed as done without execution report"
grep -F 'tarefa fechada como done sem relatorio de execucao: 1.0' "$fixture/stderr" >/dev/null

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```
EOF
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "done with APPROVED_WITH_REMARKS does not close the approval cycle"
grep -F 'tarefa fechada como done sem veredito APPROVED registrado: 1.0' "$fixture/stderr" >/dev/null

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED
tool=claude
```
EOF
rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "done with canonical verdict=APPROVED block"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

- Veredito do Revisor: APPROVED (sem achados)
EOF
rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "done with canonical reviewer verdict prose"

sed -i.bak 's/| 1.0 | Active task | done |/| 1.0 | Active task | pending |/' "$prd/tasks.md"
rm -f "$prd/1.0_execution_report.md" "$prd/tasks.md.bak"
rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "task never started without report"

echo "validate-session-end: block and release OK"
