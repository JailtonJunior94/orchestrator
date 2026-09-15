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
grep -F 'tarefa ativa sem veredito que encerre o ciclo' "$fixture/stderr" >/dev/null

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
grep -F 'tarefa fechada como done sem veredito que encerre o ciclo' "$fixture/stderr" >/dev/null

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

run_validator_with_input() {
  local output_mode="$1"
  local payload="$2"
  ( cd "$project" && AISPEC_HOOK_DECISION_OUTPUT="$output_mode" bash "$validator" <<<"$payload" >"$fixture/stdout" 2>"$fixture/stderr" )
}

sed -i.bak 's/| 1.0 | Active task | pending |/| 1.0 | Active task | blocked |/' "$prd/tasks.md"
rm -f "$prd/tasks.md.bak"
cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=CHANGES_REQUESTED
tool=claude
```
EOF

rc=0; run_validator_with_input exit '{"hook_event_name":"Stop","stop_hook_active":false}' || rc=$?
expect_exit 2 "$rc" "blocked task without approving verdict still blocks when stop_hook_active is false"
grep -F 'GATE DE ENCERRAMENTO BLOQUEADO' "$fixture/stderr" >/dev/null
grep -F 'tarefa blocked com relatorio de execucao escrito sem veredito que encerre o ciclo' "$fixture/stderr" >/dev/null

rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "blocked task without approving verdict still blocks when stop_hook_active is absent"
grep -F 'GATE DE ENCERRAMENTO BLOQUEADO' "$fixture/stderr" >/dev/null

rc=0; run_validator_with_input exit '{"hook_event_name":"Stop","stop_hook_active":true}' || rc=$?
expect_exit 0 "$rc" "stop_hook_active true releases the turn instead of looping forever"
grep -F 'stop_hook_active=true' "$fixture/stderr" >/dev/null
grep -F 'tarefa blocked com relatorio de execucao escrito sem veredito que encerre o ciclo' "$fixture/stderr" >/dev/null

rc=0; run_validator_with_input json '{"hook_event_name":"Stop","stop_hook_active":true}' || rc=$?
expect_exit 0 "$rc" "stop_hook_active true releases the turn under json decision output"
if grep -F '"decision":"block"' "$fixture/stdout" >/dev/null 2>&1; then
  echo "validate-session-end: stop_hook_active release emitted a block decision" >&2
  exit 1
fi

sed -i.bak 's/| 1.0 | Active task | blocked |/| 1.0 | Active task | pending |/' "$prd/tasks.md"
rm -f "$prd/1.0_execution_report.md" "$prd/tasks.md.bak"
rc=0; run_validator_with_input exit '{"hook_event_name":"Stop","stop_hook_active":true}' || rc=$?
expect_exit 0 "$rc" "clean tree with stop_hook_active true exits zero"

sed -i.bak 's/| 1.0 | Active task | pending |/| 1.0 | Active task | done |/' "$prd/tasks.md"
rm -f "$prd/tasks.md.bak"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```

## Achados
- [MEDIUM] internal/foo.go:1 nomenclatura inconsistente
EOF
rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "RF-33: remarks with only medium findings close the cycle"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```

## Achados
- [HIGH] internal/foo.go:1 corrida de dados
EOF
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "RF-33: remarks with a high finding do not close the cycle"
grep -F 'GATE DE ENCERRAMENTO BLOQUEADO' "$fixture/stderr" >/dev/null

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```

## Achados
- [CRITICAL] internal/foo.go:1 vazamento de segredo
EOF
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "RF-33: remarks with a critical finding do not close the cycle"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```
EOF
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "RF-33: remarks without declared severity fail closed"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```

## Achados
- **[MEDIUM]** internal/foo.go:1 nomenclatura inconsistente
EOF
rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "bold list finding with only medium severity closes the cycle"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```

## Achados
- [HIGH] internal/foo.go:10 corrida de dados
EOF
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "high severity in finding position does not close the cycle"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```

## Revisao
Veredito `APPROVED_WITH_REMARKS`, sem tag `[CRITICAL]`/`[HIGH]` bloqueante.
- Veredito do Revisor: APPROVED_WITH_REMARKS (sem tag `[critical]`/`[blocker]`)

```text
- [CRITICAL] exemplo citado dentro de bloco de codigo
```

## Achados
- [LOW] internal/foo.go:3 renomear variavel
EOF
rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "severity mentioned only in prose, backticks or fenced block closes the cycle"

cat >"$prd/1.0_execution_result.json" <<'EOF'
{
  "task_id": "1.0",
  "status": "blocked",
  "review_verdict": "changes_requested"
}
EOF
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "structured review_verdict changes_requested overrides approving prose"
grep -F 'review_verdict=changes_requested' "$fixture/stderr" >/dev/null

cat >"$prd/1.0_execution_result.json" <<'EOF'
{
  "task_id": "1.0",
  "status": "blocked",
  "review_verdict": "needs_input"
}
EOF
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "structured review_verdict needs_input overrides approving prose"

cat >"$prd/1.0_execution_report.md" <<'EOF'
# Report 1.0

```
verdict=APPROVED_WITH_REMARKS
tool=claude
```

## Achados
- [CRITICAL] internal/foo.go:1 vazamento de segredo
EOF
cat >"$prd/1.0_execution_result.json" <<'EOF'
{
  "task_id": "1.0",
  "status": "done",
  "review_verdict": "approved"
}
EOF
rc=0; run_validator exit || rc=$?
expect_exit 0 "$rc" "structured review_verdict approved overrides blocking prose"

rm -f "$prd/1.0_execution_result.json"
rc=0; run_validator exit || rc=$?
expect_exit 2 "$rc" "without the result json the textual behaviour is preserved"

sed -i.bak 's/| 1.0 | Active task | done |/| 1.0 | Active task | pending |/' "$prd/tasks.md"
rm -f "$prd/1.0_execution_report.md" "$prd/tasks.md.bak"

echo "validate-session-end: block and release OK"
