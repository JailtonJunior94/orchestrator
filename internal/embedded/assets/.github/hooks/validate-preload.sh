#!/usr/bin/env bash

set -euo pipefail

hook_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_root="${AGENTS_ROOT:-$(cd "$hook_dir/../.." && pwd)}"

canonical="$project_root/.agents/hooks/validate-preload.sh"
if [[ ! -f "$canonical" ]]; then
  canonical="$(cd "$hook_dir/../.." && pwd)/.agents/hooks/validate-preload.sh"
fi

if [[ ! -f "$canonical" ]]; then
  echo "ERRO: validador canonico .agents/hooks/validate-preload.sh ausente — rode 'ai-spec-harness install .'" >&2
  exit 2
fi

exec env AGENTS_ROOT="$project_root" bash "$canonical" "$@"
