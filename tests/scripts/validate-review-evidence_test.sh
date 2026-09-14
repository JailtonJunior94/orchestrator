#!/usr/bin/env bash

set -euo pipefail

SCRIPT="${1:-.agents/scripts/validate-review-evidence.sh}"
TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

PASS=0
FAIL=0

review_body() {
  cat <<'EOF'
## Achados

Sem achados.

## Arquivos Revisados

- internal/approval/evidence.go (diff da branch)

## Riscos Residuais

- nenhum

## Validações Executadas

- go test ./internal/approval/... -> ok 0.4s
EOF
}

build_review() {
  local verdict="$1" map_block="$2" out="$3"
  {
    printf '# Relatório de Review\n\n'
    printf -- '- Veredito: %s\n\n' "$verdict"
    if [[ -n "$map_block" ]]; then
      printf '## Mapa de Critérios de Aceite\n\n%s\n\n' "$map_block"
    fi
    review_body
  } > "$out"
}

run_case() {
  local label="$1" verdict="$2" map_block="$3" want_exit="$4" want_text="$5"
  local f="$TMP_ROOT/review_$PASS$FAIL.md"
  build_review "$verdict" "$map_block" "$f"

  local actual_exit=0 actual_out
  actual_out="$(bash "$SCRIPT" "$f" 2>&1)" || actual_exit=$?

  local locale_exit=0
  LC_ALL=C bash "$SCRIPT" "$f" >/dev/null 2>&1 || locale_exit=$?

  rm -f "$f"

  if [[ "$actual_exit" -ne "$want_exit" ]]; then
    echo "FAIL [$label]: exit=$actual_exit, want=$want_exit"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi
  if [[ "$locale_exit" -ne "$want_exit" ]]; then
    echo "FAIL [$label]: exit sob LC_ALL=C=$locale_exit, want=$want_exit"
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

VALID_MAP='- [atendido] Criterio um -> go test ./internal/approval/... -> ok 0.4s
- [atendido] Criterio dois -> internal/approval/evidence.go:42'

run_case "RC1-approved-criterio-nao-atendido" "APPROVED" \
  '- [nao atendido] Criterio um -> internal/approval/evidence.go:42' \
  1 "nao atendido"

run_case "RC2-approved-criterio-nao-verificavel" "APPROVED" \
  '- [não verificável] Criterio um -> internal/approval/evidence.go:42' \
  1 "verific"

run_case "RC3-approved-evidencia-em-prosa" "APPROVED" \
  '- [atendido] Criterio um -> porque confio no autor da mudança' \
  1 "fora das tres formas"

run_case "RC4-approved-arquivo-linha-fora-do-diff" "APPROVED" \
  '- [atendido] Criterio um -> internal/ausente.go:42' \
  1 "ausente da seção"

run_case "RC5-approved-teste-sem-resultado" "APPROVED" \
  '- [atendido] Criterio um -> TestAlgoNaoRodado' \
  1 "fora das tres formas"

run_case "RC6-approved-with-remarks-mapa-ok" "APPROVED_WITH_REMARKS" \
  "$VALID_MAP" 0 "aprovada"

run_case "RC7-sem-secao-de-mapa" "APPROVED" "" 1 "Mapa de Criterios de Aceite"

run_case "RC8-mapa-valido" "APPROVED" "$VALID_MAP" 0 "aprovada"

run_case "RC9-criterio-acentuado" "APPROVED" \
  '- [atendido] Critério de aceitação é validado -> go test ./internal/approval/... -> ok 0.4s' \
  0 "aprovada"

# ── BUG-X6: "parece comando" + "->" nao e prova ──────────────────────────────
# O agregado (internal/approval/evidence.go) ja recusava registro trivial; os
# validadores canonicos aceitavam. Estas fixtures travam a paridade.

run_case "RC10-registro-trivial-done" "APPROVED" \
  '- [atendido] criterio trivial -> make it work -> done' \
  1 "registro trivial nao e prova"

run_case "RC11-registro-trivial-ok" "APPROVED" \
  '- [atendido] criterio trivial -> make build -> ok' \
  1 "registro trivial nao e prova"

run_case "RC12-registro-na" "APPROVED" \
  '- [atendido] criterio sem prova -> go test ./... -> n/a' \
  1 "registro trivial nao e prova"

run_case "RC13-nao-e-comando" "APPROVED" \
  '- [atendido] criterio inventado -> frobnicate tudo -> saida qualquer' \
  1 "nao e comando executavel"

run_case "RC14-teste-sem-resultado-canonico" "APPROVED" \
  '- [atendido] teste citado -> TestFoo -> talvez' \
  1 "sem resultado canonico"

run_case "RC15-comando-com-saida-real" "APPROVED" \
  '- [atendido] criterio comprovado -> go test ./... -> exit 0' \
  0 "aprovada"

run_case "RC16-teste-com-resultado" "APPROVED" \
  '- [atendido] criterio comprovado -> TestFoo -> pass' \
  0 "aprovada"

run_case "A1-git-diff-sem-saida-verificavel" "APPROVED" \
  '- [atendido] Criterio um -> git diff -> muitas mudancas' \
  1 "nao e prova"

run_case "A2-lint-com-saida-generica" "APPROVED" \
  '- [atendido] Criterio um -> golangci-lint run -> zero issues' \
  1 "nao e prova"

run_case "A3-pytest-sem-alvo" "APPROVED" \
  '- [atendido] Criterio um -> pytest -> 12 passed' \
  1 "nao e comando executavel"

run_case "A4-contagem-nua-sem-contexto" "APPROVED" \
  '- [atendido] Criterio um -> go test ./x -> 3' \
  1 "nao e prova"

run_case "A5-script-com-prosa" "APPROVED" \
  '- [atendido] Criterio um -> ./run.sh -> saiu bem' \
  1 "nao e prova"

run_case "A6-go-test-com-pacote-e-duracao" "APPROVED" \
  '- [atendido] Criterio um -> go test ./internal/sample -count=1 -> ok internal/sample 0.512s' \
  0 "aprovada"

run_case "A7-go-vet-com-exit-code" "APPROVED" \
  '- [atendido] Criterio um -> go vet ./... -> exit 0' \
  0 "aprovada"

run_case "A8-teste-com-resultado-canonico" "APPROVED" \
  '- [atendido] Criterio um -> TestFoo -> pass' \
  0 "aprovada"

run_case "A9-arquivo-linha-revisado" "APPROVED" \
  '- [atendido] Criterio um -> internal/approval/evidence.go:42' \
  0 "aprovada"

run_case "A10-pytest-com-alvo-e-contagem" "APPROVED" \
  '- [atendido] Criterio um -> pytest tests/unit -> 12 passed in 0.4s' \
  0 "aprovada"

run_case "A11-lint-com-contagem-de-issues" "APPROVED" \
  '- [atendido] Criterio um -> golangci-lint run -> 0 issues' \
  0 "aprovada"

run_case "A13-mesmo-basename-diretorio-diferente" "APPROVED" \
  '- [atendido] Criterio um -> internal/inventado/evidence.go:42' \
  1 "ausente da seção"

run_empty_reviewed_case() {
  local label="A12-secao-arquivos-revisados-vazia"
  local f="$TMP_ROOT/review_empty.md"
  {
    printf '# Relatório de Review\n\n'
    printf -- '- Veredito: APPROVED\n\n'
    printf '## Mapa de Critérios de Aceite\n\n'
    printf -- '- [atendido] Criterio um -> internal/inventado.go:999\n\n'
    printf '## Achados\n\nSem achados.\n\n'
    printf '## Arquivos Revisados\n\n- nenhum arquivo relevante\n\n'
    printf '## Riscos Residuais\n\n- nenhum\n\n'
    printf '## Validações Executadas\n\n- go test ./internal/approval/... -> ok 0.4s\n'
  } > "$f"

  local actual_exit=0 actual_out
  actual_out="$(bash "$SCRIPT" "$f" 2>&1)" || actual_exit=$?
  rm -f "$f"

  if [[ "$actual_exit" -ne 1 ]]; then
    echo "FAIL [$label]: exit=$actual_exit, want=1"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi
  if ! echo "$actual_out" | grep -qi "sem nenhum arquivo listado"; then
    echo "FAIL [$label]: output não acusa seção vazia"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi
  if ! echo "$actual_out" | grep -qi "ausente da seção"; then
    echo "FAIL [$label]: referência inventada aprovada com seção vazia"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi
  echo "PASS [$label]"
  PASS=$((PASS+1))
}

run_empty_reviewed_case

run_case "A14-veredito-pt-br-aprovado" "aprovado" "$VALID_MAP" 0 "aprovada"

run_case "A15-veredito-pt-br-aprovado-com-ressalvas" "aprovado com ressalvas" "$VALID_MAP" 0 "aprovada"

run_case "A16-veredito-pt-br-reprovado-sem-achado-alto" "reprovado" "$VALID_MAP" \
  1 "REJECTED exige"

run_case "A17-veredito-pt-br-aprovado-com-criterio-nao-atendido" "aprovado" \
  '- [nao atendido] Criterio um -> internal/approval/evidence.go:42' \
  1 "nao atendido"

run_case "A18-veredito-pt-br-bloqueado" "bloqueado" "$VALID_MAP" 0 "aprovada"

run_case "A19-veredito-token-invalido" "talvez" "$VALID_MAP" 1 "veredito canonico"

VERDICT_FIXTURE="${VERDICT_FIXTURE:-tests/fixtures/approval/verdict-tokens.tsv}"

if [[ ! -f "$VERDICT_FIXTURE" ]]; then
  echo "FAIL [A20-fixture-de-paridade]: fixture ausente em $VERDICT_FIXTURE"
  FAIL=$((FAIL+1))
fi

build_review_line() {
  local verdict_line="$1" findings_block="$2" out="$3"
  {
    printf '# Relatório de Review\n\n'
    printf '%s\n\n' "$verdict_line"
    printf '## Mapa de Critérios de Aceite\n\n%s\n\n' "$VALID_MAP"
    printf '## Achados\n\n%s\n\n' "$findings_block"
    printf '## Arquivos Revisados\n\n- internal/approval/evidence.go (diff da branch)\n\n'
    printf '## Riscos Residuais\n\n- nenhum\n\n'
    printf '## Validações Executadas\n\n- go test ./internal/approval/... -> ok 0.4s\n'
  } > "$out"
}

run_fixture_case() {
  local verdict_line="$1" want_verdict="$2"
  local label="A20-paridade[$verdict_line]"
  local findings_block="Sem achados."
  local want_exit=0 want_text="aprovada"

  if [[ "$want_verdict" == "REJECTED" ]]; then
    findings_block=$'- Severidade: high\n- Arquivo: internal/approval/evidence.go\n- Linha: 42\n- Impacto: achado bloqueante'
  fi
  if [[ "$want_verdict" == "NONE" ]]; then
    want_exit=1
    want_text="veredito canonico"
  fi

  local f="$TMP_ROOT/review_fixture_$PASS$FAIL.md"
  build_review_line "$verdict_line" "$findings_block" "$f"

  local actual_exit=0 actual_out
  actual_out="$(bash "$SCRIPT" "$f" 2>&1)" || actual_exit=$?
  rm -f "$f"

  if [[ "$actual_exit" -ne "$want_exit" ]]; then
    echo "FAIL [$label]: exit=$actual_exit, want=$want_exit (veredito canonico esperado: $want_verdict)"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi
  if ! echo "$actual_out" | grep -qi "$want_text"; then
    echo "FAIL [$label]: output não contém '$want_text'"
    echo "  output: $actual_out"
    FAIL=$((FAIL+1))
    return
  fi
  echo "PASS [$label]"
  PASS=$((PASS+1))
}

while IFS=$'\t' read -r fixture_line fixture_verdict; do
  [[ -z "$fixture_line" || "${fixture_line:0:1}" == "#" ]] && continue
  run_fixture_case "$fixture_line" "$fixture_verdict"
done < "$VERDICT_FIXTURE"

echo ""
echo "Resultado: $PASS passaram, $FAIL falharam"
if [[ $FAIL -ne 0 ]]; then
  exit 1
fi
