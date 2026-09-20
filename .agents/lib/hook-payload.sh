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
