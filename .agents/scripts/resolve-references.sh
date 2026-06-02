#!/usr/bin/env bash
# resolve-references.sh
# Resolve referencias surgical a partir de uma lista de arquivos tocados pela tarefa
# e (opcional) sinais de diff. Para cada language skill com INDEX.yaml, retorna apenas
# referencias com when_to_load casando (file_patterns OU diff_signals) ou marcadas
# como always:true.
#
# Tool-neutro: pode ser invocado por hooks do Claude, Codex ou Copilot via cascata
# (.agents/scripts/ resolve antes de mirrors).
#
# Uso:
#   bash .agents/scripts/resolve-references.sh <skill-name> <files...>
#   echo "<diff text>" | bash ... <skill-name> <files...>
#
# Variaveis:
#   AGENTS_ROOT          raiz onde .agents/skills/<skill> resolve (default: pwd).
#   RESOLVE_VERBOSE=1    imprime motivo da inclusao por referencia.
#
# Saida (stdout, uma linha por referencia incluida):
#   <skill>/<ref-id>\t<absolute-path-to-md>\t<trigger>
# Exit 0 sempre que skill+INDEX.yaml existem; exit 2 se faltar INDEX (oportunidade
# para validate-skill-prerequisites.sh bloquear).

set -euo pipefail

skill_name="${1:-}"
[[ -n "$skill_name" ]] || { echo "uso: resolve-references.sh <skill> <files...>" >&2; exit 64; }
shift

AGENTS_ROOT="${AGENTS_ROOT:-$(pwd)}"
skill_dir="$AGENTS_ROOT/.agents/skills/$skill_name"
index="$skill_dir/references/INDEX.yaml"

if [[ ! -f "$index" ]]; then
  echo "INDEX.yaml ausente em $skill_dir/references/ — skill nao instalada ou nao indexada" >&2
  exit 2
fi

# Le stdin se disponivel para diff
diff_text=""
if [[ ! -t 0 ]]; then
  diff_text="$(cat)"
fi

files=("$@")

# Converte paths absolutos para relativos ao project root quando possivel.
# Patterns no INDEX sao sempre relativos ao projeto — sem isso, diretorios
# do path absoluto podem casar incorretamente (ex: /Git/Messaging/ ativando
# **/Messaging/** para qualquer arquivo do repo).
#
# Estrategia (em ordem):
#   1. Se arquivo esta sob AGENTS_ROOT, relativiza por ai.
#   2. Senao, walk-up a partir do arquivo procurando .agents/ ou .git/ — usa
#      essa raiz como project root para a relativizacao.
#   3. Se nao achar, mantem absoluto (best-effort).
detect_project_root_for() {
  local f="$1"
  local dir
  dir=$(dirname "$f")
  while [[ "$dir" != "/" && "$dir" != "." && -n "$dir" ]]; do
    if [[ -d "$dir/.agents" || -d "$dir/.git" ]]; then
      printf '%s' "$dir"
      return 0
    fi
    dir=$(dirname "$dir")
  done
  return 1
}

rel_files=()
for f in "${files[@]}"; do
  if [[ "$AGENTS_ROOT" != "/" && "$f" == "$AGENTS_ROOT/"* ]]; then
    rel_files+=("${f#$AGENTS_ROOT/}")
    continue
  fi
  if proj_root=$(detect_project_root_for "$f"); then
    rel_files+=("${f#$proj_root/}")
    continue
  fi
  rel_files+=("$f")
done
files=("${rel_files[@]}")

# Parser YAML minimal sem dependencia externa (jq/yq podem nao estar disponiveis
# em todos os ambientes). Aceita a sintaxe gerada por nos. Cada entrada exige:
#   - id:
#     file:
#     [always: true]
#     [when_to_load:]
#       [file_patterns: [...]]
#       [diff_signals: [...]]
#
# A parser produz registros separados por NUL com formato:
#   id<TAB>file<TAB>always<TAB>file_patterns_csv<TAB>diff_signals_csv
parse_index() {
  # Saida: registros separados por `|` (NAO usar tab — bash read colapsa tabs
  # consecutivos quando IFS contem whitespace, sumindo com campos vazios).
  # Dentro de fp/ds, valores sao separados por \x1f (unit separator) para
  # permitir virgulas dentro de strings YAML (ex.: "[K comparable, V any]").
  awk -v US=$'\x1f' '
    function parse_inline_array(raw,    inner, out, m, val) {
      # Recebe linha tipo `<lead>: ["a", "b, c", "d"]` e retorna valores
      # separados por US (unit separator). Strings nao-quoted toleradas.
      sub(/^[^[]*\[/, "", raw)        # strip ate [
      sub(/\][[:space:]]*$/, "", raw) # strip ]
      out = ""
      while (length(raw) > 0) {
        if (match(raw, /"[^"]*"/)) {
          val = substr(raw, RSTART+1, RLENGTH-2)
          raw = substr(raw, RSTART+RLENGTH)
          sub(/^[[:space:]]*,[[:space:]]*/, "", raw)
          out = (out == "" ? val : out US val)
        } else if (match(raw, /[^,]+/)) {
          val = substr(raw, RSTART, RLENGTH)
          raw = substr(raw, RSTART+RLENGTH)
          sub(/^[[:space:]]*,[[:space:]]*/, "", raw)
          gsub(/^[[:space:]]+|[[:space:]]+$/, "", val)
          if (val != "") out = (out == "" ? val : out US val)
        } else {
          break
        }
      }
      return out
    }
    BEGIN { in_refs = 0; in_when = 0; cur_id = ""; cur_file = ""; cur_always = "0"; cur_fp = ""; cur_ds = "" }
    /^references:/ { in_refs = 1; next }
    !in_refs { next }
    /^  - id:/ {
      if (cur_id != "") { printf "%s|%s|%s|%s|%s\n", cur_id, cur_file, cur_always, cur_fp, cur_ds }
      cur_id = $3; cur_file = ""; cur_always = "0"; cur_fp = ""; cur_ds = ""; in_when = 0; next
    }
    /^    file:/ { cur_file = $2; next }
    /^    always:[[:space:]]*true/ { cur_always = "1"; next }
    /^    when_to_load:/ { in_when = 1; next }
    in_when && /^      file_patterns:/ {
      if (index($0, "[") > 0) cur_fp = parse_inline_array($0)
      next
    }
    in_when && /^      diff_signals:/ {
      if (index($0, "[") > 0) cur_ds = parse_inline_array($0)
      next
    }
    END {
      if (cur_id != "") { printf "%s|%s|%s|%s|%s\n", cur_id, cur_file, cur_always, cur_fp, cur_ds }
    }
  ' "$index"
}

# Glob -> regex via sed (mais previsivel que parameter expansion para padroes
# multi-segmento com **). Resultado e usado com bash =~. Conversao:
#   ** -> .*    (cross-segment)
#   *  -> [^/]* (single-segment)
#   .  -> \.    (literal)
# Para outros especiais de regex no padrao (raro em globs), confiamos no fato
# de que paths normais nao os contem.
match_glob() {
  local pattern="$1" file="$2"
  local re
  re=$(printf '%s' "$pattern" | sed \
    -e 's/\./\\./g' \
    -e 's|\*\*|__GLOBSTAR__|g' \
    -e 's|\*|[^/]*|g' \
    -e 's|__GLOBSTAR__|.*|g')
  # Match em qualquer prefixo do path (padroes sao relativos).
  if [[ "$file" =~ ^.*${re}$ ]]; then
    return 0
  fi
  return 1
}

match_signal() {
  local needle="$1"
  [[ -z "$needle" ]] && return 1
  if [[ -n "$diff_text" ]]; then
    grep -F -q -- "$needle" <<<"$diff_text" 2>/dev/null && return 0
  fi
  return 1
}

included_ids=""
while IFS='|' read -r id file always fp ds; do
  [[ -z "$id" ]] && continue
  trigger=""  # reset por iteracao (do contrario vaza valor do entry anterior)
  if [[ "$always" == "1" ]]; then
    trigger="always"
  else
    # file_patterns — separador interno e \x1f (unit separator), preserva virgulas dentro de strings.
    if [[ -n "$fp" ]]; then
      IFS=$'\x1f' read -ra patterns <<<"$fp"
      for pat in "${patterns[@]}"; do
        [[ -z "$pat" ]] && continue
        for f in "${files[@]}"; do
          if match_glob "$pat" "$f"; then
            trigger="file_pattern:$pat"
            break 2
          fi
        done
      done
    fi
    # diff_signals — mesmo separador \x1f que file_patterns.
    if [[ -z "$trigger" && -n "$ds" ]]; then
      IFS=$'\x1f' read -ra signals <<<"$ds"
      for sig in "${signals[@]}"; do
        [[ -z "$sig" ]] && continue
        if match_signal "$sig"; then
          trigger="diff_signal:$sig"
          break
        fi
      done
    fi
  fi
  if [[ -n "$trigger" ]]; then
    abs="$skill_dir/references/$file"
    [[ -f "$abs" ]] || abs="(missing)"
    case " $included_ids " in
      *" $id "*) ;;
      *)
        printf '%s/%s\t%s\t%s\n' "$skill_name" "$id" "$abs" "$trigger"
        included_ids="$included_ids $id"
        [[ "${RESOLVE_VERBOSE:-0}" == "1" ]] && echo "  motivo: $trigger" >&2
        ;;
    esac
  fi
done < <(parse_index)

exit 0
