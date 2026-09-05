#!/usr/bin/env bash
# check-spec-paths.sh
# Gate: toda referencia de caminho citada nos artefatos de contrato de um PRD sob
# gestao SDD precisa resolver no repositorio.
#
# Motivacao: a techspec de prd-sdd-robusto declarou por meses dois pacotes
# (internal/sdd/orchestrator e internal/sdd/review) que nunca existiram. A
# divergencia sobreviveu a auditoria completa dos requisitos porque validate-sdd
# compara hashes de artefato e vinculos RF->tarefa, nao a prosa que descreve
# componentes.
#
# Escopo: apenas PRDs com sdd-state.json, ou seja, sob gestao SDD ativa. PRDs
# historicos citam legitimamente caminhos ja removidos e arquivos de projetos
# externos analisados na epoca; exigir que sejam reescritos seria errado.
#
# Uso: bash scripts/check-spec-paths.sh [diretorio-prd...]
# Sem argumentos, varre todos os PRDs sob gestao SDD.
# Exit 0 = todas as referencias resolvem; 1 = alguma nao resolve.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT" || exit 2

# Prefixos ancorados na raiz do repositorio. Um token so e tratado como caminho
# quando comeca por um destes: evita tratar pacote npm, flag de comando ou
# caminho relativo a outra raiz como referencia deste repositorio.
_PREFIXES='^(internal|cmd|scripts|docs|tests|evals|deployment|migrations|taskfiles|configs|\.agents|\.claude|\.github|\.codex|\.gemini|\.specs)/'

# Caminhos documentados como opcionais por decisao de arquitetura: existem apenas
# quando a pessoa opta por cria-los, entao ausencia nao e defeito.
_OPTIONAL='^(\.agents/config\.yaml|\.claude/config\.yaml|\.claude/settings\.json)$'

extract_paths() {
  local file="$1"
  grep -ohE '`[^`]+`' "$file" 2>/dev/null | tr -d '`' | while read -r token; do
    # Descarta o que nao e caminho literal: comando com espaco, placeholder,
    # variavel, glob, expressao de codigo, chamada de funcao.
    case "$token" in
      *' '*|*'{'*|*'}'*|*'<'*|*'>'*|*'$'*|*'*'*|*'"'*|*'='*|*'@'*|*'..'*|*'('*|*')'*) continue ;;
    esac
    token="${token%%::*}"   # arquivo.go::Simbolo
    token="${token%%:*}"    # arquivo.go:42 e arquivo.go:Simbolo
    token="${token%/}"      # diretorio/ -> diretorio
    [[ -n "$token" ]] || continue
    echo "$token" | grep -qE "$_PREFIXES" || continue
    # Notacao pacote.Simbolo do Go: ultimo segmento tem ponto seguido de maiuscula.
    case "${token##*/}" in *.[A-Z]*) continue ;; esac
    echo "$token" | grep -qE "$_OPTIONAL" && continue
    printf '%s\n' "$token"
  done
}

prd_dirs=("$@")
if [[ ${#prd_dirs[@]} -eq 0 ]]; then
  mapfile -t prd_dirs < <(find .specs -name 'sdd-state.json' -not -path '*/.checkpoints/*' 2>/dev/null | sed 's|/sdd-state.json||' | sort)
fi

if [[ ${#prd_dirs[@]} -eq 0 ]]; then
  echo "Nenhum PRD sob gestao SDD encontrado; nada a verificar."
  exit 0
fi

missing=0
checked=0
for prd in "${prd_dirs[@]}"; do
  for artifact in prd.md techspec.md tasks.md; do
    file="$prd/$artifact"
    [[ -f "$file" ]] || continue
    while read -r path; do
      [[ -n "$path" ]] || continue
      checked=$((checked+1))
      if [[ ! -e "$path" ]]; then
        echo "FALTANDO: $file cita caminho inexistente: $path"
        missing=1
      fi
    done < <(extract_paths "$file" | sort -u)
  done
done

if [[ "$missing" -ne 0 ]]; then
  echo ""
  echo "Referencias de caminho invalidas em artefato de contrato sob gestao SDD."
  echo "Corrija o caminho ou reconcilie o artefato com a estrutura real do codigo."
  exit 1
fi

echo "Referencias de caminho: $checked verificadas, todas resolvem (${#prd_dirs[@]} PRD sob gestao SDD)."
exit 0
