#!/usr/bin/env bash
# Prova o gate de referencias de caminho nos dois sentidos, como exige a regra
# de gates do repositorio: precisa aprovar spec integra e reprovar spec que cita
# caminho inexistente.
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GATE="$REPO_ROOT/scripts/check-spec-paths.sh"
TMP="$REPO_ROOT/.tmp-spec-paths-test"
rm -rf "$TMP"; mkdir -p "$TMP/prd-fixture"
trap 'rm -rf "$TMP"' EXIT

pass=0; fail=0
assert() {
  local desc="$1" want="$2" got="$3"
  if [[ "$got" -eq "$want" ]]; then echo "  OK   $desc (exit=$got)"; pass=$((pass+1));
  else echo "  FAIL $desc (esperado=$want obtido=$got)"; fail=$((fail+1)); fi
}

printf '{"schema_version":2}\n' > "$TMP/prd-fixture/sdd-state.json"
printf '# PRD\n' > "$TMP/prd-fixture/prd.md"

# Caso 1: caminho existente -> aprova
printf '# TechSpec\n\nO gate vive em `scripts/check-spec-paths.sh`.\n' > "$TMP/prd-fixture/techspec.md"
bash "$GATE" "$TMP/prd-fixture" >/dev/null 2>&1
assert "caminho existente aprova" 0 $?

# Caso 2: caminho inexistente -> reprova (o defeito original)
printf '# TechSpec\n\nComponente `internal/sdd/orchestrator` cuida do plano.\n' > "$TMP/prd-fixture/techspec.md"
out=$(bash "$GATE" "$TMP/prd-fixture" 2>&1); code=$?
assert "caminho inexistente reprova" 1 $code
if echo "$out" | grep -q "internal/sdd/orchestrator"; then
  echo "  OK   diagnostico nomeia o caminho invalido"; pass=$((pass+1))
else
  echo "  FAIL diagnostico nao nomeia o caminho"; fail=$((fail+1))
fi

# Caso 3: nao confunde notacao de simbolo, comando nem placeholder com caminho
printf '# TechSpec\n\nVer `internal/runtime/runner.go::Run`, `internal/fs.FileSystem`,\n`go test ./...`, `.specs/prd-<slug>/prd.md` e `internal/{a,b}`.\n' > "$TMP/prd-fixture/techspec.md"
bash "$GATE" "$TMP/prd-fixture" >/dev/null 2>&1
assert "simbolo, comando e placeholder nao viram caminho" 0 $?

# Caso 4: caminho opcional por decisao de arquitetura nao reprova
printf '# TechSpec\n\nConfig opcional em `.agents/config.yaml`.\n' > "$TMP/prd-fixture/techspec.md"
bash "$GATE" "$TMP/prd-fixture" >/dev/null 2>&1
assert "caminho opcional documentado nao reprova" 0 $?

# Caso 5: caminho inexistente marcado como (planejado) -> aprova
printf '# TechSpec\n\nO pacote `internal/sdd/orchestrator` (planejado) ainda nao existe.\n' > "$TMP/prd-fixture/techspec.md"
bash "$GATE" "$TMP/prd-fixture" >/dev/null 2>&1
assert "caminho inexistente com marcador (planejado) aprova" 0 $?

# Caso 6: (planejado) nao mascara outro caminho inexistente sem marcador na mesma spec
printf '# TechSpec\n\n`internal/sdd/orchestrator` (planejado) e `internal/sdd/review` (quebrado).\n' > "$TMP/prd-fixture/techspec.md"
out=$(bash "$GATE" "$TMP/prd-fixture" 2>&1); code=$?
assert "marcador (planejado) nao mascara caminho quebrado vizinho" 1 $code
if echo "$out" | grep -q "internal/sdd/review" && ! echo "$out" | grep -q "internal/sdd/orchestrator"; then
  echo "  OK   diagnostico acusa so o caminho sem marcador"; pass=$((pass+1))
else
  echo "  FAIL diagnostico deveria acusar apenas internal/sdd/review"; fail=$((fail+1))
fi

# Caso 7: PRD historico fora do escopo, com PRD ativo presente
rm -f "$TMP/prd-fixture/sdd-state.json"
printf '# TechSpec\n\nComponente `internal/sdd/orchestrator`.\n' > "$TMP/prd-fixture/techspec.md"
mkdir -p "$TMP/prd-ativo"
printf '{"schema_version":2}\n' > "$TMP/prd-ativo/sdd-state.json"
printf '# PRD\n\nO gate vive em `scripts/check-spec-paths.sh`.\n' > "$TMP/prd-ativo/prd.md"
out=$(SPEC_PATHS_ROOT="$TMP" bash "$GATE" 2>&1); code=$?
assert "PRD historico sem estado SDD fica fora do escopo" 0 $code
if echo "$out" | grep -q "1 PRD sob gestao SDD"; then
  echo "  OK   a varredura selecionou exatamente o PRD ativo"; pass=$((pass+1))
else
  echo "  FAIL a varredura nao confirmou escopo nao-vazio: $out"; fail=$((fail+1))
fi

# Caso 8: escopo vazio reprova
mkdir -p "$TMP/vazio"
out=$(SPEC_PATHS_ROOT="$TMP/vazio" bash "$GATE" 2>&1); code=$?
assert "escopo vazio reprova em vez de aprovar por vacuidade" 1 $code
if echo "$out" | grep -q "ESCOPO VAZIO"; then
  echo "  OK   diagnostico nomeia o escopo vazio"; pass=$((pass+1))
else
  echo "  FAIL diagnostico nao nomeia o escopo vazio"; fail=$((fail+1))
fi

echo ""
echo "Resultado: $pass OK, $fail FAIL"
[[ "$fail" -eq 0 ]] || exit 1
