#!/usr/bin/env bash

parse_file_path() {
  if command -v python3 >/dev/null 2>&1; then
    python3 -c '
import json
import sys

try:
    data = json.load(sys.stdin)
except Exception as err:
    print("hook payload parse failed: %s" % err, file=sys.stderr)
    raise SystemExit(1)

tool_input = data.get("tool_input", data.get("input", data))
if isinstance(tool_input, dict):
    tool_input = tool_input.get("arguments", tool_input)
    for key in ("file_path", "filePath", "target_file", "path", "file"):
        value = tool_input.get(key)
        if isinstance(value, str) and value:
            print(value)
            raise SystemExit
    patch = tool_input.get("patch", tool_input.get("patchText", tool_input.get("command", "")))
else:
    patch = tool_input if isinstance(tool_input, str) else ""
for line in patch.splitlines():
    if line.startswith(("*** Add File: ", "*** Update File: ", "*** Delete File: ")):
        print(line.split(": ", 1)[1])
        raise SystemExit
'
    return
  fi
  if command -v jq >/dev/null 2>&1; then
    local raw_payload
    raw_payload="$(cat)"
    if ! printf '%s' "$raw_payload" | jq -e . >/dev/null 2>&1; then
      echo "hook payload parse failed: invalid JSON" >&2
      return 1
    fi
    printf '%s' "$raw_payload" | jq -r '.tool_input.arguments.file_path // .tool_input.arguments.filePath // .tool_input.arguments.target_file // .tool_input.arguments.path // .tool_input.arguments.file // .input.arguments.file_path // .input.arguments.filePath // .input.arguments.target_file // .input.arguments.path // .input.arguments.file // .tool_input.file_path // .tool_input.filePath // .tool_input.target_file // .tool_input.path // .tool_input.file // .input.file_path // .input.filePath // .input.target_file // .input.path // .input.file // .file_path // .filePath // .target_file // .path // .file // empty'
    return
  fi
  echo "hook payload parse failed: python3 or jq is required" >&2
  return 1
}

parse_command_text() {
  if command -v python3 >/dev/null 2>&1; then
    python3 -c '
import json
import sys

try:
    data = json.load(sys.stdin)
except Exception as err:
    print("hook payload parse failed: %s" % err, file=sys.stderr)
    raise SystemExit(1)

tool_input = data.get("tool_input", data.get("input", data))
if isinstance(tool_input, dict):
    tool_input = tool_input.get("arguments", tool_input)
    for key in ("command", "cmd", "script"):
        value = tool_input.get(key)
        if isinstance(value, str) and value.strip():
            print(value)
            raise SystemExit
        if isinstance(value, list) and value:
            print(" ".join(str(item) for item in value))
            raise SystemExit
'
    return
  fi
  if command -v jq >/dev/null 2>&1; then
    local raw_payload
    raw_payload="$(cat)"
    if ! printf '%s' "$raw_payload" | jq -e . >/dev/null 2>&1; then
      echo "hook payload parse failed: invalid JSON" >&2
      return 1
    fi
    printf '%s' "$raw_payload" | jq -r '(.tool_input.arguments.command // .tool_input.arguments.cmd // .tool_input.arguments.script // .input.arguments.command // .input.arguments.cmd // .input.arguments.script // .tool_input.command // .tool_input.cmd // .tool_input.script // .input.command // .input.cmd // .input.script // .command // .cmd // .script // empty) | if type == "array" then join(" ") else . end'
    return
  fi
  echo "hook payload parse failed: python3 or jq is required" >&2
  return 1
}

extract_command_source_targets() {
  local command_text="$1"
  printf '%s' "$command_text" \
    | tr '[:space:];|&()<>' '\n' \
    | sed -E 's/^["'"'"'`]+//; s/["'"'"'`]+$//' \
    | grep -E '\.(go|ts|tsx|js|jsx|mjs|cjs|py|cs|csproj)$' 2>/dev/null \
    | grep -v '^-' \
    | awk 'NF && !seen[$0]++' || true
}

hook_recursion_guard() {
  local hook_name="$1"
  local block_exit="$2"
  local max_depth="${AI_HOOK_MAX_DEPTH:-3}"
  local current_depth="${AI_HOOK_DEPTH:-0}"

  if [[ "$current_depth" -ge "$max_depth" ]]; then
    echo "GOVERNANCE BLOQUEIO: recursao hook -> ferramenta -> hook detectada em $hook_name (RF-67, profundidade atual=$current_depth, maximo=$max_depth)." >&2
    exit "$block_exit"
  fi
  export AI_HOOK_DEPTH=$((current_depth + 1))
}

hook_measure_start() {
  HOOK_MEASURE_NAME="$1"
  HOOK_MEASURE_TIMEOUT="${2:-}"
  HOOK_MEASURE_START_SECONDS="$SECONDS"
  trap 'hook_measure_on_exit' EXIT
}

hook_measure_on_exit() {
  local exit_code=$?
  local duration_ms=$(( (SECONDS - HOOK_MEASURE_START_SECONDS) * 1000 ))
  echo "hook.duration_ms=$duration_ms hook=$HOOK_MEASURE_NAME hook.timeout_s=${HOOK_MEASURE_TIMEOUT:-unknown} hook.decision_exit=$exit_code" >&2
  exit "$exit_code"
}

hook_timeout_watch() {
  local hook_name="$1"
  local timeout_seconds="$2"
  local child_pid="$3"

  (
    sleep "$timeout_seconds"
    kill -TERM "$child_pid" 2>/dev/null
  ) &
  local watchdog_pid=$!

  local rc=0
  wait "$child_pid" 2>/dev/null || rc=$?

  kill "$watchdog_pid" 2>/dev/null
  wait "$watchdog_pid" 2>/dev/null

  if [[ "$rc" -ge 128 ]]; then
    echo "GOVERNANCE BLOQUEIO: hook $hook_name excedeu timeout declarado de ${timeout_seconds}s; tratado como negacao (RF-66)." >&2
    return 124
  fi
  return "$rc"
}
