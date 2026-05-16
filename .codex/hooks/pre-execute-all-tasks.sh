#!/usr/bin/env bash
# pre-execute-all-tasks.sh
# Validacao programatica pre-orquestracao.
# Fecha as fragilidades F17 (unset PREFLIGHT_DONE), F18 (cross-PRD spec-hash),
# F27 (cross-PRD circular dependency), F29 (gaps numericos),
# alem de regex F7/F12/F20 reforcado em runtime.
#
# Uso:
#   bash .claude/hooks/pre-execute-all-tasks.sh <prd-slug>
#
# Compativel com bash 3.x (macOS default) — nao usa associative arrays.
#
# Exit codes:
#   0 — todas validacoes passaram (warnings nao bloqueiam)
#   1 — pelo menos uma validacao falhou; mensagens em stderr
#   2 — argumentos invalidos

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "Uso: $0 <prd-slug>" >&2
  exit 2
fi

PRD_SLUG="$1"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
TASKS_ROOT="${AI_TASKS_ROOT:-tasks}"
PRD_PREFIX="${AI_PRD_PREFIX:-prd-}"
PRD_DIR="$REPO_ROOT/$TASKS_ROOT/$PRD_PREFIX$PRD_SLUG"
TASKS_MD="$PRD_DIR/tasks.md"

if [[ ! -f "$TASKS_MD" ]]; then
  echo "FAIL: $TASKS_MD nao existe" >&2
  exit 1
fi

# === F17: limpar AI_PREFLIGHT_DONE herdada ===
unset AI_PREFLIGHT_DONE

errors=0
warnings=0

trim() {
  echo "$1" | xargs
}

hash_file() {
  local file="$1"
  local hash
  if command -v ai-spec >/dev/null 2>&1; then
    hash=$(ai-spec hash "$file" 2>/dev/null || true)
    if [[ -n "$hash" ]]; then
      echo "$hash"
      return 0
    fi
  fi
  if [[ -f "$REPO_ROOT/main.go" && -d "$REPO_ROOT/cmd/ai_spec_harness" ]] && command -v go >/dev/null 2>&1; then
    hash=$(cd "$REPO_ROOT" && go run . hash "$file" 2>/dev/null || true)
    if [[ -n "$hash" ]]; then
      echo "$hash"
      return 0
    fi
  fi
  echo "FAIL: comando 'ai-spec hash' indisponivel; instale/atualize ai-spec-harness para calcular hash portavel de $file" >&2
  return 1
}

normalize_status() {
  local raw
  raw=$(trim "$1")
  printf "%s" "$raw" | tr '[:upper:]' '[:lower:]'
}

normalize_deps() {
  local raw
  raw=$(trim "$1")
  case "$raw" in
    ""|"—"|"-"|"none"|"None"|"NONE"|"n/a"|"N/A"|"nenhuma"|"Nenhuma")
      echo "—"
      return 0
      ;;
  esac
  echo "$raw" | sed -E 's/[[:space:]]*,[[:space:]]*/, /g'
}

normalize_parallel() {
  local raw rest normalized
  raw=$(trim "$1")
  case "$raw" in
    ""|"—"|"-"|"none"|"None"|"NONE"|"n/a"|"N/A")
      echo "—"
      return 0
      ;;
    "Não"|"não"|"nao"|"NÃO"|"NAO"|"No"|"no"|"false"|"False"|"FALSE"|"n"|"N")
      echo "Não"
      return 0
      ;;
  esac

  case "$raw" in
    Com[[:space:]]*|com[[:space:]]*|COM[[:space:]]*)
      rest=$(echo "$raw" | sed -E 's/^[Cc][Oo][Mm][[:space:]]+//')
      normalized=$(echo "$rest" | sed -E 's/[[:space:]]*,[[:space:]]*/, /g')
      echo "Com $normalized"
      return 0
      ;;
  esac

  echo "$raw"
}

warn_if_normalized() {
  local label="$1"
  local id="$2"
  local raw="$3"
  local normalized="$4"
  if [[ "$raw" != "$normalized" ]]; then
    echo "WARN: $label normalizado em $id: '$raw' -> '$normalized'" >&2
    warnings=$((warnings+1))
  fi
}

task_status_in_file() {
  local tasks_md="$1"
  local wanted_id="$2"
  local line tid status
  while IFS= read -r line; do
    case "$line" in
      "|"*) ;;
      *) continue ;;
    esac
    IFS='|' read -ra cells <<< "$line"
    [[ ${#cells[@]} -lt 6 ]] && continue
    tid=$(trim "${cells[1]}")
    [[ "$tid" == "$wanted_id" ]] || continue
    status=$(normalize_status "${cells[3]}")
    echo "$status"
    return 0
  done < "$tasks_md"
  return 1
}

# Arrays paralelos (indice i): IDS[i], DEPS[i], STATUSES[i]
IDS=()
DEPS=()
STATUSES=()

# === Coletar tarefas e validar regex ===
while IFS= read -r line; do
  case "$line" in
    "|"*) ;;
    *) continue ;;
  esac

  IFS='|' read -ra cells <<< "$line"
  [[ ${#cells[@]} -lt 6 ]] && continue
  tid=$(trim "${cells[1]}")
  raw_status=$(trim "${cells[3]}")
  raw_deps=$(trim "${cells[4]}")
  raw_parallel=$(trim "${cells[5]}")
  status=$(normalize_status "$raw_status")
  deps=$(normalize_deps "$raw_deps")
  parallel=$(normalize_parallel "$raw_parallel")

  [[ "$tid" =~ ^[0-9]+\.[0-9]+$ ]] || continue

  IDS+=("$tid")
  DEPS+=("$deps")
  STATUSES+=("$status")

  warn_if_normalized "status" "$tid" "$raw_status" "$status"
  warn_if_normalized "dependencias" "$tid" "$raw_deps" "$deps"
  warn_if_normalized "paralelizavel" "$tid" "$raw_parallel" "$parallel"

  if ! [[ "$status" =~ ^(pending|in_progress|needs_input|blocked|failed|done)$ ]]; then
    echo "FAIL: status malformed em $tid: '$status'" >&2
    errors=$((errors+1))
  fi
  if ! [[ "$deps" =~ ^(—|([a-z][a-z0-9_-]*/)?[0-9]+\.[0-9]+(,[[:space:]]*([a-z][a-z0-9_-]*/)?[0-9]+\.[0-9]+)*)$ ]]; then
    echo "FAIL: dependencias malformed em $tid: '$deps'" >&2
    errors=$((errors+1))
  fi
  if ! [[ "$parallel" =~ ^(—|Não|Com[[:space:]]+[0-9]+\.[0-9]+(,[[:space:]]*[0-9]+\.[0-9]+)*)$ ]]; then
    echo "FAIL: paralelizavel malformed em $tid: '$parallel'" >&2
    errors=$((errors+1))
  fi
done < "$TASKS_MD"

# === F29: gaps numericos no major id ===
if [[ ${#IDS[@]} -gt 1 ]]; then
  MAJORS=()
  for id in "${IDS[@]}"; do
    MAJORS+=("${id%%.*}")
  done
  SORTED=($(printf "%s\n" "${MAJORS[@]}" | sort -n -u))
  if [[ ${#SORTED[@]} -gt 1 ]]; then
    min=${SORTED[0]}
    max=${SORTED[$((${#SORTED[@]}-1))]}
    expected=$((max - min + 1))
    actual=${#SORTED[@]}
    if [[ $actual -lt $expected ]]; then
      MISSING=()
      for ((i=min; i<=max; i++)); do
        present=0
        for s in "${SORTED[@]}"; do
          if [[ "$s" -eq "$i" ]]; then present=1; break; fi
        done
        [[ "$present" -eq 0 ]] && MISSING+=("$i.0")
      done
      if [[ "${AI_ALLOW_TASK_ID_GAPS:-0}" == "1" ]]; then
        echo "WARN F29: gaps aceitos por AI_ALLOW_TASK_ID_GAPS=1: ${MISSING[*]}" >&2
        warnings=$((warnings+1))
      else
        echo "FAIL F29: gaps na numeracao: ${MISSING[*]}. Confirme intencional corrigindo tasks.md ou execute com AI_ALLOW_TASK_ID_GAPS=1." >&2
        errors=$((errors+1))
      fi
    fi
  fi
fi

# === F18: validar spec-hash de cross-PRD deps ===
for i in "${!IDS[@]}"; do
  tid="${IDS[$i]}"
  deps="${DEPS[$i]}"
  IFS=',' read -ra DEP_LIST <<< "$deps"
  for dep in "${DEP_LIST[@]}"; do
    dep=$(echo "$dep" | xargs)
    if [[ "$dep" =~ ^([a-z][a-z0-9_-]*)/([0-9]+\.[0-9]+)$ ]]; then
      ext_slug="${BASH_REMATCH[1]}"
      ext_dir="$REPO_ROOT/$TASKS_ROOT/$PRD_PREFIX$ext_slug"
        ext_task_id="${BASH_REMATCH[2]}"
        ext_tasks_md="$ext_dir/tasks.md"
        ext_prd_md="$ext_dir/prd.md"

        if [[ ! -f "$ext_tasks_md" ]]; then
          echo "FAIL F18: cross-PRD target nao encontrado: $ext_tasks_md (referenciado por $tid)" >&2
          errors=$((errors+1))
          continue
        fi

        if [[ ! -f "$ext_prd_md" ]]; then
          echo "FAIL F18: cross-PRD PRD ausente: $ext_prd_md (referenciado por $tid)" >&2
          errors=$((errors+1))
          continue
        fi

        ext_status=$(task_status_in_file "$ext_tasks_md" "$ext_task_id" || true)
        if [[ -z "$ext_status" ]]; then
          echo "FAIL F18: cross-PRD task not found: $ext_slug/$ext_task_id (referenciado por $tid)" >&2
          errors=$((errors+1))
        elif [[ "$ext_status" != "done" ]]; then
          echo "FAIL F18: cross-PRD task not done: $ext_slug/$ext_task_id status=$ext_status (referenciado por $tid)" >&2
          errors=$((errors+1))
        fi

        ext_hash=$(grep -E "^<!-- spec-hash-prd:" "$ext_tasks_md" | head -1 | sed -E 's/^<!-- spec-hash-prd:[[:space:]]*([a-f0-9]+).*/\1/')
        if [[ -z "$ext_hash" ]]; then
          echo "FAIL F18: cross-PRD '$ext_slug' sem spec-hash-prd em $ext_tasks_md" >&2
          errors=$((errors+1))
        else
          current_hash=$(hash_file "$ext_prd_md" || true)
          if [[ -z "$current_hash" ]]; then
            errors=$((errors+1))
          elif [[ "$ext_hash" != "$current_hash" ]]; then
            echo "FAIL F18: cross-PRD '$ext_slug' tem spec drift (tasks.md=$ext_hash, prd.md atual=$current_hash). Re-execute aquele PRD primeiro." >&2
            errors=$((errors+1))
          fi
        fi
      fi
  done
done

# === F27: detectar ciclo cross-PRD via DFS limitada ===
detect_cycle() {
  local origin="$1"
  local current="$2"
  local depth="$3"
  local seen="$4"

  [[ "$depth" -gt 3 ]] && return 0
  local target_md="$REPO_ROOT/$TASKS_ROOT/$PRD_PREFIX$current/tasks.md"
  [[ ! -f "$target_md" ]] && return 0

  while IFS= read -r line; do
    case "$line" in
      "|"*) ;;
      *) continue ;;
    esac
    IFS='|' read -ra c <<< "$line"
    [[ ${#c[@]} -lt 6 ]] && continue
    local d
    d=$(echo "${c[4]}" | xargs)
    IFS=',' read -ra dlist <<< "$d"
    for dep in "${dlist[@]}"; do
      dep=$(echo "$dep" | xargs)
      if [[ "$dep" =~ ^([a-z][a-z0-9_-]*)/[0-9]+\.[0-9]+$ ]]; then
        local next="${BASH_REMATCH[1]}"
        if [[ "$next" == "$origin" ]]; then
          echo "FAIL F27: ciclo cross-PRD detectado: $seen -> $current -> $origin" >&2
          return 1
        fi
        if [[ "$seen" != *"$next"* ]]; then
          if ! detect_cycle "$origin" "$next" $((depth+1)) "$seen -> $next"; then
            return 1
          fi
        fi
      fi
    done
  done < "$target_md"
  return 0
}

for i in "${!IDS[@]}"; do
  deps="${DEPS[$i]}"
  IFS=',' read -ra DEP_LIST <<< "$deps"
  for dep in "${DEP_LIST[@]}"; do
    dep=$(echo "$dep" | xargs)
    if [[ "$dep" =~ ^([a-z][a-z0-9_-]*)/[0-9]+\.[0-9]+$ ]]; then
      ext_slug="${BASH_REMATCH[1]}"
      if ! detect_cycle "$PRD_SLUG" "$ext_slug" 1 "$PRD_SLUG"; then
        errors=$((errors+1))
        break
      fi
    fi
  done
done

# === Resumo ===
if [[ "$errors" -gt 0 ]]; then
  echo "pre-execute-all-tasks: $errors erro(s), $warnings warning(s) para PRD $PRD_SLUG" >&2
  exit 1
fi

if [[ "$warnings" -gt 0 ]]; then
  echo "pre-execute-all-tasks: OK com $warnings warning(s) para PRD $PRD_SLUG" >&2
else
  echo "pre-execute-all-tasks: OK (PRD $PRD_SLUG, ${#IDS[@]} tarefas validadas)" >&2
fi
exit 0
