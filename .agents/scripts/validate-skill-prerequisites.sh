#!/usr/bin/env bash
# validate-skill-prerequisites.sh
# Gate tool-neutro: dado um conjunto de arquivos tocados pela tarefa, bloqueia se
# a skill de linguagem correspondente nao estiver disponivel para descoberta
# (i.e., SKILL.md + references/INDEX.yaml ausentes na arvore .agents/skills/).
#
# Politica: nao bloqueia por "referencia X nao carregada" — referencias sao opt-in
# via when_to_load. Bloqueia somente por ausencia da camada de descoberta, garantindo
# que o agente sempre tenha o mapa.
#
# Mapeamento extensao → skill obrigatoria:
#   .go             → go-implementation
#   .ts, .tsx, .js, .mjs, .cjs, .jsx → node-implementation
#   .cs, .csproj    → dotnet-csharp-implementation
#
# Uso:
#   bash .agents/scripts/validate-skill-prerequisites.sh <files...>
#
# Variaveis:
#   AGENTS_ROOT             raiz onde .agents/skills/ resolve (default: pwd).
#   PREREQ_MODE=warn        nao bloqueia (exit 0) mesmo com skill ausente. Default: fail.
#
# Exit:
#   0 = OK ou warn-mode
#   1 = bloqueio (skill obrigatoria nao instalada)

set -euo pipefail

AGENTS_ROOT="${AGENTS_ROOT:-$(pwd)}"
MODE="${PREREQ_MODE:-fail}"

needed=""
for f in "$@"; do
  case "$f" in
    *.go) needed="$needed go-implementation" ;;
    *.ts|*.tsx|*.js|*.jsx|*.mjs|*.cjs) needed="$needed node-implementation" ;;
    *.cs|*.csproj) needed="$needed dotnet-csharp-implementation" ;;
    *.py) needed="$needed python-implementation" ;;
  esac
done

# Deduplica
needed_uniq=$(printf '%s\n' $needed | awk '!seen[$0]++' | tr '\n' ' ')

missing=()
for skill in $needed_uniq; do
  [[ -z "$skill" ]] && continue
  skill_md="$AGENTS_ROOT/.agents/skills/$skill/SKILL.md"
  index="$AGENTS_ROOT/.agents/skills/$skill/references/INDEX.yaml"
  if [[ ! -f "$skill_md" || ! -f "$index" ]]; then
    missing+=("$skill")
  fi
done

if [[ ${#missing[@]} -eq 0 ]]; then
  exit 0
fi

{
  echo "BLOQUEIO: tarefa toca arquivos cuja skill obrigatoria nao esta acessivel."
  echo "Skills ausentes:"
  for s in "${missing[@]}"; do
    echo "  - $s ($AGENTS_ROOT/.agents/skills/$s/{SKILL.md,references/INDEX.yaml})"
  done
  echo
  echo "Acao: rode 'ai-spec-harness install .' ou copie .agents/skills/ do repo de governanca."
  echo "Override (nao recomendado): export PREREQ_MODE=warn"
} >&2

if [[ "$MODE" == "warn" ]]; then
  exit 0
fi
exit 1
