#!/usr/bin/env bash
set -euo pipefail

validator="${1:-.agents/scripts/validate-bugfix-evidence.sh}"
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT

valid="$fixture/valid.md"
cat >"$valid" <<'EOF'
# Relatorio de Bugfix
- Total de bugs no escopo: 2
- Corrigidos: 2
- Testes de regressao adicionados: 2
- Pendentes: nenhum
- Estado final: done
## Bugs
- ID: BUG-1
- Severidade: critical
- Origem: RF-01
- Estado: fixed
- Causa raiz: primeira
- Arquivos alterados: a
- Teste de regressao: teste a
- Validacao: pass
- ID: BUG-2
- Severidade: major
- Origem: issue #2
- Estado: fixed
- Causa raiz: segunda
- Arquivos alterados: b
- Teste de regressao: teste b
- Validacao: pass
## Comandos Executados
- go test ./... -> pass
## Riscos Residuais
- nenhum
EOF
bash "$validator" "$valid" >/dev/null

partial="$fixture/partial.md"
sed '/Causa raiz: segunda/d; s/Corrigidos: 2/Corrigidos: 1/' "$valid" >"$partial"
if bash "$validator" "$partial" >"$fixture/output" 2>&1; then
  echo "validador aceitou bloco incompleto" >&2
  exit 1
fi
grep -F 'Causa raiz no bloco BUG-2' "$fixture/output" >/dev/null
grep -F 'Corrigidos=1 diverge dos blocos (2)' "$fixture/output" >/dev/null
echo "validate-bugfix-evidence: blocos e totalizadores OK"
