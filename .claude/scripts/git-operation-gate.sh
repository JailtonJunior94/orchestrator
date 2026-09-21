#!/usr/bin/env bash

set -euo pipefail

readonly GIT_OPERATION_BLOCK_EXIT=2

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="${AGENTS_ROOT:-$(cd "$script_dir/../.." && pwd)}"

parse_lib=""
for candidate in \
  "$project_root/.agents/lib/hook-payload.sh" \
  "$project_root/scripts/lib/hook-payload.sh" \
  "$script_dir/../lib/hook-payload.sh" \
  "$script_dir/../../scripts/lib/hook-payload.sh"; do
  if [[ -f "$candidate" ]]; then
    parse_lib="$candidate"
    break
  fi
done

if [[ -z "$parse_lib" ]]; then
  echo "ERRO: hook-payload.sh ausente em .agents/lib/ e scripts/lib/ — rode 'ai-spec-harness install .'" >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

source "$parse_lib"

git_scope_path=""
for candidate in \
  "$project_root/.agents/generated/git-scope.json" \
  "$project_root/internal/embedded/assets/.agents/generated/git-scope.json"; do
  if [[ -f "$candidate" ]]; then
    git_scope_path="$candidate"
    break
  fi
done

payload=""
if [[ ! -t 0 ]]; then
  payload="$(cat)"
fi

if ! command_text="$(printf '%s' "$payload" | parse_command_text)"; then
  echo "GOVERNANCE BLOQUEIO: payload de hook invalido; operacao git negada." >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

if [[ -z "$command_text" ]]; then
  echo "GOVERNANCE BLOQUEIO: comando ausente ou nao extraivel; operacao git negada." >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

audit_log_line() {
  local reason="$1" command="$2"
  local log="${GOVERNANCE_ESCAPE_LOG:-$project_root/.aispec/governance-escapes.log}"
  local stamp
  stamp="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)"
  mkdir -p "$(dirname "$log")" 2>/dev/null || return 0
  printf '%s\t%s\t%s\t%s\t%s\n' \
    "$stamp" "$reason" "$command" "${AI_TOOL:-desconhecido}" "${USER:-desconhecido}" \
    >> "$log" 2>/dev/null || true
}

audit_canonical_decision() {
  local decision="$1" reason="$2" policy_id="$3" gate_id="$4"
  command -v ai-spec >/dev/null 2>&1 || return 0
  local canonical_log="${GOVERNANCE_HOOKAUDIT_LOG:-$project_root/.aispec/hook-decisions.jsonl}"
  ai-spec hookaudit append \
    --path "$canonical_log" \
    --event before_tool \
    --provider "${AI_TOOL:-desconhecido}" \
    --hook git-operation-gate \
    --decision "$decision" \
    --reason "$reason" \
    --policy-id "$policy_id" \
    --gate-id "$gate_id" \
    --duration-ms 0 \
    >/dev/null 2>&1 || true
}

audit_git_operation_escape() {
  local reason="$1" policy_id="$2" gate_id="$3"
  echo "AUDITORIA: escape de operacao git registrado ($reason) para comando: $command_text" >&2
  audit_log_line "$reason" "$command_text"
  audit_canonical_decision "ALLOW" "$reason" "$policy_id" "$gate_id"
}

audit_destructive_operation_escape() {
  local reason="$1"
  echo "AUDITORIA: escape de operacao destrutiva registrado ($reason) para comando: $command_text" >&2
  audit_log_line "$reason" "$command_text"
  audit_canonical_decision "ALLOW" "$reason" "P-RM-DESTRUCTIVE" "G-RM-DESTRUCTIVE"
}

audit_interpreter_operation_escape() {
  local reason="$1"
  echo "AUDITORIA: escape de operacao com interpretador dinamico registrado ($reason) para comando: $command_text" >&2
  audit_log_line "$reason" "$command_text"
  audit_canonical_decision "ALLOW" "$reason" "P-GIT-INTERP-DYNAMIC" "G-GIT-INTERP-DYNAMIC"
}

is_destructive_removal_segment() {
  local segment="$1"
  if ! printf '%s' "$segment" | grep -Eq '(^|[[:space:]/])rm([[:space:]]|$)'; then
    return 1
  fi
  if printf '%s' "$segment" \
    | grep -Eq -- '(^|[[:space:]])-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*([[:space:]]|$)|(^|[[:space:]])-[a-zA-Z]*f[a-zA-Z]*r[a-zA-Z]*([[:space:]]|$)'; then
    return 0
  fi
  if printf '%s' "$segment" | grep -Eq '(^|[[:space:]])(-r|-R|--recursive)([[:space:]]|$)' \
    && printf '%s' "$segment" | grep -Eq '(^|[[:space:]])(-f|--force)([[:space:]]|$)'; then
    return 0
  fi
  return 1
}

destructive_detected=0
while IFS= read -r segment || [[ -n "$segment" ]]; do
  [[ -n "$segment" ]] || continue
  if is_destructive_removal_segment "$segment"; then
    destructive_detected=1
  fi
done < <(printf '%s' "$command_text" | tr ';|&' '\n')

destructive_confirmed="${GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED:-0}"
if [[ "$destructive_detected" -eq 1 ]]; then
  if [[ "$destructive_confirmed" == "1" ]]; then
    audit_destructive_operation_escape "GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED=1"
  else
    echo "GOVERNANCE BLOQUEIO: comando destrutivo detectado (remocao recursiva e forcada) negado independentemente do estado de preload." >&2
    echo "Comando: $command_text" >&2
    echo "Para prosseguir com pedido explicito do usuario: export GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED=1" >&2
    exit "$GIT_OPERATION_BLOCK_EXIT"
  fi
fi

classify_git_command_structurally() {
  python3 -c '
import json
import re
import shlex
import sys

command_text = sys.stdin.read()
scope_path = sys.argv[1] if len(sys.argv) > 1 else ""

scope_names = set()
if scope_path:
    try:
        with open(scope_path, "r", encoding="utf-8") as handle:
            scope = json.load(handle)
        scope_names = {op.get("subcommand", "") for op in scope.get("operations", [])}
    except Exception:
        scope_names = set()

TRANSPARENT_WRAPPERS = {
    "env", "command", "exec", "time", "nohup", "nice", "ionice", "setsid",
    "timeout", "stdbuf", "sudo", "doas",
}
GIT_ARG_FLAGS = {"-C", "-c", "--git-dir", "--work-tree", "--namespace", "--super-prefix"}
INTERPRETER_SHELLS = {"sh", "bash", "zsh", "dash", "ksh"}
FORCE_FLAGS = {"--force", "--force-with-lease", "-f"}
ASSIGNMENT_RE = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")
LINE_CONTINUATION_RE = re.compile(r"\\\r?\n")


class TokenizeError(Exception):
    pass


def tokenize(text):
    normalized = LINE_CONTINUATION_RE.sub(" ", text)
    lexer = shlex.shlex(normalized, posix=True, punctuation_chars=True)
    lexer.whitespace_split = True
    try:
        return list(lexer)
    except ValueError as err:
        raise TokenizeError(str(err)) from err


PUNCTUATION_CHARS = set("();<>|&")


def is_pure_punctuation_token(token):
    return len(token) > 0 and all(char in PUNCTUATION_CHARS for char in token)


def split_segments(tokens):
    segments = []
    current = []
    for token in tokens:
        if is_pure_punctuation_token(token):
            if current:
                segments.append(current)
                current = []
            continue
        current.append(token)
    if current:
        segments.append(current)
    return segments


def normalize_base(token):
    base = token.rsplit("/", 1)[-1]
    if base.endswith(".exe"):
        base = base[:-4]
    return base


def resolve_command_word(segment):
    idx = 0
    while idx < len(segment) and ASSIGNMENT_RE.match(segment[idx]):
        idx += 1
    if idx >= len(segment):
        return None, idx
    return normalize_base(segment[idx]), idx


def resolve_candidate_positions(segment, head_word, head_idx):
    if head_word in TRANSPARENT_WRAPPERS:
        return list(range(head_idx, len(segment)))
    return [head_idx]


def resolve_git_subcommand(segment, start_idx):
    idx = start_idx + 1
    while idx < len(segment):
        token = segment[idx]
        if token.startswith("-"):
            if token in GIT_ARG_FLAGS:
                idx += 2
                continue
            idx += 1
            continue
        return token, idx
    return None, idx


def has_flag_char(rest, char):
    for token in rest:
        if token.startswith("--"):
            continue
        if token.startswith("-") and char in token[1:]:
            return True
    return False


def classify_git(segment, start_idx):
    subcommand, sub_idx = resolve_git_subcommand(segment, start_idx)
    if subcommand is None:
        return None
    rest = segment[sub_idx + 1:]
    if subcommand == "commit":
        return {"subcommand": "commit", "policy_id": "P-GIT-COMMIT"}
    if subcommand == "push":
        is_force = any(token in FORCE_FLAGS for token in rest) or any(
            token.startswith("+") for token in rest if not token.startswith("-")
        )
        if is_force:
            return {"subcommand": "push --force", "policy_id": "P-GIT-PUSH-FORCE"}
        return {"subcommand": "push", "policy_id": "P-GIT-PUSH"}
    if subcommand == "reset":
        if "--hard" in rest:
            return {"subcommand": "reset --hard", "policy_id": "P-GIT-RESET-HARD"}
        return None
    if subcommand == "clean":
        if (has_flag_char(rest, "f") or "--force" in rest) and (
            has_flag_char(rest, "d") or has_flag_char(rest, "x")
        ):
            return {"subcommand": "clean", "policy_id": "P-GIT-CLEAN"}
        return None
    if subcommand == "checkout":
        if "--" in rest or "-f" in rest or "--force" in rest:
            return {"subcommand": "checkout --", "policy_id": "P-GIT-CHECKOUT-RESTORE"}
        return None
    if subcommand == "restore":
        return {"subcommand": "restore", "policy_id": "P-GIT-CHECKOUT-RESTORE"}
    return None


matches = []
interpreter_hits = []

try:
    tokens = tokenize(command_text)
except TokenizeError as err:
    print("tokenize failed: %s" % err, file=sys.stderr)
    sys.exit(1)

for segment in split_segments(tokens):
    if not segment:
        continue
    head_word, head_idx = resolve_command_word(segment)
    if head_word is None:
        continue
    for pos in resolve_candidate_positions(segment, head_word, head_idx):
        raw_token = segment[pos]
        candidate = normalize_base(raw_token)
        if candidate == "git":
            result = classify_git(segment, pos)
            if result and (not scope_names or result["subcommand"] in scope_names):
                matches.append(result)
        if candidate == "eval":
            interpreter_hits.append("eval")
        if candidate in INTERPRETER_SHELLS and "-c" in segment[pos:]:
            interpreter_hits.append(candidate + "_-c")
        if "$" in raw_token and not ASSIGNMENT_RE.match(raw_token):
            interpreter_hits.append("variable_expansion")

if "$(" in command_text or "`" in command_text:
    interpreter_hits.append("command_substitution")

for match in matches:
    print("GIT_MATCH policy_id=%s subcommand=%s" % (match["policy_id"], match["subcommand"]))
for hit in interpreter_hits:
    print("INTERPRETER_MATCH reason=%s" % hit)
' "$1"
}

git_matches=()
interpreter_matches=()
if command -v python3 >/dev/null 2>&1; then
  classification=""
  if ! classification="$(printf '%s' "$command_text" | classify_git_command_structurally "${git_scope_path:-}" 2>/dev/null)"; then
    interpreter_matches+=("INTERPRETER_MATCH reason=classifier_error_fail_closed")
  fi
  while IFS= read -r line; do
    [[ -n "$line" ]] || continue
    case "$line" in
      GIT_MATCH*) git_matches+=("$line") ;;
      INTERPRETER_MATCH*) interpreter_matches+=("$line") ;;
    esac
  done <<< "$classification"
else
  interpreter_matches+=("INTERPRETER_MATCH reason=python3_unavailable_fail_closed")
fi

interpreter_operation_confirmed="${GOVERNANCE_INTERPRETER_OPERATION_CONFIRMED:-0}"
if [[ "${#interpreter_matches[@]}" -gt 0 ]]; then
  reason="${interpreter_matches[0]#*reason=}"
  if [[ "$interpreter_operation_confirmed" == "1" ]]; then
    audit_interpreter_operation_escape "GOVERNANCE_INTERPRETER_OPERATION_CONFIRMED=1"
  else
    echo "GOVERNANCE BLOQUEIO: invocacao de interpretador com conteudo dinamico detectada ($reason); nao e resolvivel por analise estatica e exige aprovacao explicita." >&2
    echo "Comando: $command_text" >&2
    echo "Para prosseguir com pedido explicito do usuario: export GOVERNANCE_INTERPRETER_OPERATION_CONFIRMED=1" >&2
    exit "$GIT_OPERATION_BLOCK_EXIT"
  fi
fi

git_operation_confirmed="${GOVERNANCE_GIT_OPERATION_CONFIRMED:-0}"
git_operation_mode="${GOVERNANCE_GIT_OPERATION_MODE:-fail}"

if [[ "${#git_matches[@]}" -gt 0 ]]; then
  first_match="${git_matches[0]}"
  policy_id="$(printf '%s' "$first_match" | sed -n 's/.*policy_id=\([^ ]*\).*/\1/p')"
  subcommand="$(printf '%s' "$first_match" | sed -n 's/.*subcommand=\(.*\)$/\1/p')"

  if [[ "$git_operation_confirmed" == "1" ]]; then
    audit_git_operation_escape "GOVERNANCE_GIT_OPERATION_CONFIRMED=1" "$policy_id" "G-GIT-${subcommand// /-}"
    exit 0
  fi
  if [[ "$git_operation_mode" == "warn" ]]; then
    audit_git_operation_escape "GOVERNANCE_GIT_OPERATION_MODE=warn" "$policy_id" "G-GIT-${subcommand// /-}"
    echo "GOVERNANCE_GIT_OPERATION_MODE=warn: prosseguindo sem bloqueio (opt-out explicito)." >&2
    exit 0
  fi
  echo "GOVERNANCE BLOQUEIO: operacao git nao solicitada (git $subcommand, $policy_id) negada pelo gate canonico de operacao git." >&2
  echo "Comando: $command_text" >&2
  echo "Para prosseguir com pedido explicito do usuario: export GOVERNANCE_GIT_OPERATION_CONFIRMED=1" >&2
  exit "$GIT_OPERATION_BLOCK_EXIT"
fi

exit 0
