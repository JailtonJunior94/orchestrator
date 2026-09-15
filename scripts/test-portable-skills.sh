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

for root in "$repo_root/.agents/skills" "$repo_root/.claude/skills" "$repo_root/.github/skills" \
            "$repo_root/internal/embedded/assets/.agents/skills"; do
  [[ -d "$root" ]] || continue
  execute_task_skill="$root/execute-task/SKILL.md"
  if [[ -f "$execute_task_skill" ]]; then
    assert_absent "$execute_task_skill" 'Sem tag crítica → Etapa 5'
    assert_absent "$execute_task_skill" 'Cadeia review → bugfix → review é máxima'
    assert_contains "$execute_task_skill" '`APPROVED_WITH_REMARKS` → **encerra somente sem achado `[HIGH]`/`[CRITICAL]` (RF-33)**'
  fi
  bugfix_skill="$root/bugfix/SKILL.md"
  if [[ -f "$bugfix_skill" ]]; then
    assert_absent "$bugfix_skill" 'Bugfix nao deve re-invocar review se ja estiver sendo executado dentro de um ciclo review -> bugfix.'
    assert_contains "$bugfix_skill" 'quem abre a rodada seguinte de revisao e o orquestrador do Ciclo de Aprovacao (RF-38)'
  fi
  governance_skill="$root/agent-governance/SKILL.md"
  if [[ -f "$governance_skill" ]]; then
    assert_contains "$governance_skill" 'o limite aplicavel e o **teto de rodadas** (default 5, RF-35/RF-38)'
  fi
done

for depth_lib in "$repo_root/.agents/lib/check-invocation-depth.sh" \
                 "$repo_root/scripts/lib/check-invocation-depth.sh" \
                 "$repo_root/internal/embedded/assets/.agents/lib/check-invocation-depth.sh"; do
  [[ -f "$depth_lib" ]] || continue
  assert_absent "$depth_lib" 'Cadeia detectada: execute-task -> review -> bugfix -> (bloqueado).'
  assert_contains "$depth_lib" 'para pelo teto de rodadas (RF-38)'
done

if [[ "$failures" -ne 0 ]]; then
  exit 1
fi

echo 'portable skills contracts: OK'
