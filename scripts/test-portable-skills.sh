#!/usr/bin/env bash
# Garante que contratos humanos permaneçam delegados ao CLI e que os templates
# não reintroduzam direção de DAG ou classificação de skills por convenção textual.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
failures=0

assert_contains() {
  local file="$1" pattern="$2"
  if ! grep -Fq -- "$pattern" "$file"; then
    echo "FAIL: esperado '$pattern' em ${file#$repo_root/}" >&2
    failures=$((failures + 1))
  fi
}

assert_absent() {
  local file="$1" pattern="$2"
  if grep -Fq -- "$pattern" "$file"; then
    echo "FAIL: não esperado '$pattern' em ${file#$repo_root/}" >&2
    failures=$((failures + 1))
  fi
}

task_template="$repo_root/.agents/skills/create-tasks/assets/task-template.md"
tasks_template="$repo_root/.agents/skills/create-tasks/assets/tasks-template.md"
orchestrator_skill="$repo_root/.agents/skills/execute-all-tasks/SKILL.md"
enforcement="$repo_root/.agents/skills/agent-governance/references/enforcement-matrix.md"
agents_template="$repo_root/.agents/skills/analyze-project/assets/agents-template.md"
codex_adapter="$repo_root/.codex/docs/workaround-preload.md"
gemini_adapter="$repo_root/.gemini/docs/workaround-preload.md"

assert_contains "$task_template" 'category'
assert_absent "$task_template" 'agent-governance`, `execute-task`, `bugfix`, `review`, `refactor`'
assert_contains "$tasks_template" 'T1 --> T2'
assert_contains "$tasks_template" 'category: governance` ou `category: language'
assert_absent "$tasks_template" '`*-implementation`'
assert_contains "$orchestrator_skill" 'ai-spec runtime-capabilities <raiz-do-worktree>'
assert_absent "$orchestrator_skill" '| Tool | Primitiva de spawn | Kill nativo no timeout? |'
assert_absent "$enforcement" '| Capacidade | Claude Code | Codex | Gemini CLI | Copilot CLI |'
assert_contains "$agents_template" 'ai-spec runtime-capabilities <raiz-do-worktree>'
assert_contains "$codex_adapter" 'ai-spec runtime-capabilities <raiz-do-worktree>'
assert_contains "$gemini_adapter" 'ai-spec runtime-capabilities <raiz-do-worktree>'
assert_absent "$gemini_adapter" 'se suportado pela versao do Gemini CLI'

if [[ "$failures" -ne 0 ]]; then
  exit 1
fi

echo 'portable skills contracts: OK'
