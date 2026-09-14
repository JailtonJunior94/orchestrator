#!/usr/bin/env bash
# Extrai file_path do JSON de hook input (stdin).
# Uso: file_path="$(echo "$input" | parse_file_path)"
# Tenta python3, depois jq, depois grep/sed como fallback.
#
# Fonte compartilhada entre validate-governance.sh e validate-preload.sh.

parse_file_path() {
  local input
  input="$(head -c 65536)"

  local file_path=""
  if command -v python3 >/dev/null 2>&1; then
    file_path="$(printf '%s' "$input" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    tool_input = data.get('tool_input', data.get('input', data))
    if isinstance(tool_input, dict):
        tool_input = tool_input.get('arguments', tool_input)
        for key in ('file_path', 'filePath', 'target_file', 'path', 'file'):
            value = tool_input.get(key)
            if isinstance(value, str) and value:
                print(value)
                raise SystemExit
        patch = tool_input.get('patch', tool_input.get('patchText', ''))
    else:
        patch = tool_input if isinstance(tool_input, str) else ''
    for line in patch.splitlines():
        if line.startswith('*** Add File: ') or line.startswith('*** Update File: ') or line.startswith('*** Delete File: '):
            print(line.split(': ', 1)[1])
            raise SystemExit
except Exception:
    pass
" 2>/dev/null || true)"
  elif command -v jq >/dev/null 2>&1; then
    file_path="$(printf '%s' "$input" | jq -r '.tool_input.arguments.file_path // .tool_input.arguments.filePath // .tool_input.arguments.target_file // .tool_input.arguments.path // .tool_input.arguments.file // .input.arguments.file_path // .input.arguments.filePath // .input.arguments.target_file // .input.arguments.path // .input.arguments.file // .tool_input.file_path // .tool_input.filePath // .tool_input.target_file // .tool_input.path // .tool_input.file // .input.file_path // .input.filePath // .input.target_file // .input.path // .input.file // .file_path // .filePath // .target_file // .path // .file // empty' 2>/dev/null || true)"
  fi

  if [[ -z "$file_path" ]]; then
    file_path="$(printf '%s' "$input" | grep -o '"file_path":"[^"]*"' 2>/dev/null | head -1 | sed 's/"file_path":"//;s/"//' || true)"
  fi

  if [[ -z "$file_path" ]]; then
    file_path="$(printf '%s' "$input" | grep -Eo '\*\*\* (Add|Update|Delete) File: [^[:space:]]+' 2>/dev/null | head -1 | sed -E 's/^\*\*\* (Add|Update|Delete) File: //' || true)"
  fi

  printf '%s' "$file_path"
}

parse_command_text() {
  local input
  input="$(head -c 65536)"

  local command_text=""
  if command -v python3 >/dev/null 2>&1; then
    command_text="$(printf '%s' "$input" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    tool_input = data.get('tool_input', data.get('input', data))
    if isinstance(tool_input, dict):
        tool_input = tool_input.get('arguments', tool_input)
        for key in ('command', 'cmd', 'script'):
            value = tool_input.get(key)
            if isinstance(value, str) and value.strip():
                print(value)
                raise SystemExit
            if isinstance(value, list) and value:
                print(' '.join(str(item) for item in value))
                raise SystemExit
except Exception:
    pass
" 2>/dev/null || true)"
  elif command -v jq >/dev/null 2>&1; then
    command_text="$(printf '%s' "$input" | jq -r '.tool_input.arguments.command // .tool_input.arguments.cmd // .tool_input.arguments.script // .input.arguments.command // .tool_input.command // .tool_input.cmd // .tool_input.script // .input.command // .command // empty' 2>/dev/null || true)"
  fi

  if [[ -z "$command_text" ]]; then
    command_text="$(printf '%s' "$input" | grep -o '"command":"[^"]*"' 2>/dev/null | head -1 | sed 's/"command":"//;s/"$//' || true)"
  fi

  printf '%s' "$command_text"
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
