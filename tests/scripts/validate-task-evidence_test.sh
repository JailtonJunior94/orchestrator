#!/usr/bin/env bash
# Testes table-driven para .claude/scripts/validate-task-evidence.sh (RF-05).
# Cobertura: 6 casos — válido, sem sha, sem verdict, sem tool, delta -3.0%, delta +0.5%.

set -euo pipefail

SCRIPT="${1:-.claude/scripts/validate-task-evidence.sh}"
TMPDIR_BASE=$(mktemp -d)
trap 'rm -rf "$TMPDIR_BASE"' EXIT

PASS=0
FAIL=0

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
- Arquivo: tasks/prd-portability-parity/task-5.0.md
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

sha=abc1234def5678901234567890abcdef01234567
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

run_case "TC1-valido" "$VALID_REPORT" 0 "aprovada"

# ── Sem SHA ──────────────────────────────────────────────────────────────────
NO_SHA_REPORT="${VALID_REPORT/sha=abc1234def5678901234567890abcdef01234567/sha=INVALIDO}"
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

echo ""
echo "Resultado: $PASS passaram, $FAIL falharam"
if [[ $FAIL -ne 0 ]]; then
  exit 1
fi
