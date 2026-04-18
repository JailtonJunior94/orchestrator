#!/usr/bin/env bash
# Detects the primary language(s) of a project based on marker files.
# Usage: detect-language.sh [project_root]
# Outputs: space-separated list of detected languages (go, csharp, typescript)
# Exit 1 if no supported language is detected.

set -euo pipefail

ROOT="${1:-.}"
DETECTED=()

# Go
if [[ -f "$ROOT/go.mod" ]] || compgen -G "$ROOT/**/*.go" > /dev/null 2>&1; then
  DETECTED+=("go")
fi

# C#
if compgen -G "$ROOT/**/*.csproj" > /dev/null 2>&1 || compgen -G "$ROOT/**/*.sln" > /dev/null 2>&1; then
  DETECTED+=("csharp")
fi

# TypeScript
if [[ -f "$ROOT/tsconfig.json" ]] || [[ -f "$ROOT/package.json" ]] && compgen -G "$ROOT/**/*.ts" > /dev/null 2>&1; then
  DETECTED+=("typescript")
fi

if [[ ${#DETECTED[@]} -eq 0 ]]; then
  echo "ERROR: No supported language detected (Go, C#, TypeScript)." >&2
  exit 1
fi

echo "${DETECTED[*]}"
