#!/usr/bin/env bash
# check-hooks-sync.sh
# Detecta drift entre o diretorio canonico (.claude/hooks/) dos hooks do orquestrador
# e os mirrors (.agents/hooks/, .codex/hooks/, .github/hooks/,
# internal/embedded/assets/{.claude,.agents,.codex,.github}/hooks/).
#
# Uso: ./scripts/check-hooks-sync.sh
# Exit 0 = sincronizado; exit 1 = drift detectado.
# Para corrigir: ./scripts/sync-hooks.sh

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.claude/hooks"

ORCHESTRATOR_HOOKS=(
  "post-execute-task.sh"
  "pre-execute-all-tasks.sh"
  "post-wave.sh"
  "subagent-stop-wrapper.sh"
)

declare -a mirror_dirs=(
  "$repo_root/.agents/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.agents/hooks"
  "$repo_root/internal/embedded/assets/.codex/hooks"
  "$repo_root/internal/embedded/assets/.github/hooks"
)

drift_count=0
ok_count=0

for hook in "${ORCHESTRATOR_HOOKS[@]}"; do
  canon_path="$canonical/$hook"
  if [[ ! -f "$canon_path" ]]; then
    echo "MISSING: $canon_path (canonico ausente)"
    drift_count=$((drift_count + 1))
    continue
  fi

  for mirror in "${mirror_dirs[@]}"; do
    mirror_path="$mirror/$hook"
    if [[ ! -f "$mirror_path" ]]; then
      echo "MISSING: $mirror_path"
      drift_count=$((drift_count + 1))
      continue
    fi
    if ! diff -q "$canon_path" "$mirror_path" >/dev/null 2>&1; then
      echo "DRIFT: $hook diverge entre $canonical e $mirror"
      drift_count=$((drift_count + 1))
    else
      ok_count=$((ok_count + 1))
    fi
  done
done

echo
echo "Hooks em sync: $ok_count"
echo "Drift / missing: $drift_count"

session_end_canonical="$repo_root/.agents/scripts/validate-session-end.sh"
declare -a session_end_gate_dirs=(
  "$repo_root/.claude/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.codex/hooks"
  "$repo_root/internal/embedded/assets/.github/hooks"
  "$repo_root/internal/embedded/assets/.agents/scripts"
)
session_end_drift=0
if [[ ! -f "$session_end_canonical" ]]; then
  echo "MISSING: $session_end_canonical (gate de encerramento canonico)"
  session_end_drift=$((session_end_drift + 1))
else
  for mirror in "${session_end_gate_dirs[@]}"; do
    mirror_path="$mirror/validate-session-end.sh"
    if [[ ! -f "$mirror_path" ]]; then
      echo "MISSING: $mirror_path (gate de encerramento)"
      session_end_drift=$((session_end_drift + 1))
      continue
    fi
    if ! diff -q "$session_end_canonical" "$mirror_path" >/dev/null 2>&1; then
      echo "DRIFT: validate-session-end.sh diverge entre .agents/scripts e $mirror"
      session_end_drift=$((session_end_drift + 1))
    fi
  done
  echo "Gate de encerramento em sync: $((${#session_end_gate_dirs[@]} - session_end_drift))/${#session_end_gate_dirs[@]} mirrors"
fi

tool_validation_drift=0
declare -a TOOL_VALIDATION_HOOKS=(
  "validate-preload.sh"
  "validate-governance.sh"
  "validate-session-end.sh"
)
declare -a tool_hook_roots=(
  ".claude"
  ".codex"
  ".github"
)
for root in "${tool_hook_roots[@]}"; do
  for hook in "${TOOL_VALIDATION_HOOKS[@]}"; do
    canon_path="$repo_root/$root/hooks/$hook"
    mirror_path="$repo_root/internal/embedded/assets/$root/hooks/$hook"
    if [[ ! -f "$canon_path" ]]; then
      echo "MISSING: $canon_path (validador canonico por ferramenta)"
      tool_validation_drift=$((tool_validation_drift + 1))
      continue
    fi
    if [[ ! -f "$mirror_path" ]]; then
      echo "MISSING: $mirror_path (mirror embarcado)"
      tool_validation_drift=$((tool_validation_drift + 1))
      continue
    fi
    if ! diff -q "$canon_path" "$mirror_path" >/dev/null 2>&1; then
      echo "DRIFT: $root/hooks/$hook diverge do asset embarcado"
      tool_validation_drift=$((tool_validation_drift + 1))
    fi
  done
done
echo "Validadores por ferramenta com drift: $tool_validation_drift"

declare -a AGENTS_VALIDATION_HOOKS=(
  "validate-preload.sh"
  "validate-governance.sh"
  "validate-session-end.sh"
)
for hook in "${AGENTS_VALIDATION_HOOKS[@]}"; do
  canon_path="$repo_root/.agents/hooks/$hook"
  mirror_path="$repo_root/internal/embedded/assets/.agents/hooks/$hook"
  if [[ ! -f "$canon_path" || ! -f "$mirror_path" ]]; then
    echo "MISSING: validador tool-neutro .agents/hooks/$hook"
    tool_validation_drift=$((tool_validation_drift + 1))
    continue
  fi
  if ! diff -q "$canon_path" "$mirror_path" >/dev/null 2>&1; then
    echo "DRIFT: .agents/hooks/$hook diverge do asset embarcado"
    tool_validation_drift=$((tool_validation_drift + 1))
  fi
done

# S4: hooks de .claude/hooks/ deliberadamente FORA de qualquer mirror e da
# distribuicao do `ai-spec install`. Allowlist EXPLICITA — a exclusao e uma
# decisao registrada, nao silencio do gate.
#   validate-token-budget.sh -> aviso opcional e nao-bloqueante de budget de contexto,
#     nunca registrado em .claude/settings*.json (ou seja, inerte ate wiring manual).
#     O caminho orquestrado ja e coberto pelo hook Go equivalente
#     internal/runtime/hooks/token_budget.go (ADR-014), que e quem roda sob --runtime acp.
#     Distribui-lo duplicaria a mesma semantica nos projetos consumidores.
declare -a CLAUDE_LOCAL_ONLY_HOOKS=(
  "validate-token-budget.sh"
)
local_only_drift=0
for hook in "${CLAUDE_LOCAL_ONLY_HOOKS[@]}"; do
  if [[ ! -f "$repo_root/.claude/hooks/$hook" ]]; then
    echo "MISSING: .claude/hooks/$hook (declarado local-only mas ausente)"
    local_only_drift=$((local_only_drift + 1))
    continue
  fi
  if [[ -f "$repo_root/internal/embedded/assets/.claude/hooks/$hook" ]]; then
    echo "DRIFT: $hook e declarado local-only mas foi embarcado em internal/embedded/assets/.claude/hooks/"
    local_only_drift=$((local_only_drift + 1))
    continue
  fi
  echo "LOCAL-ONLY: .claude/hooks/$hook fora dos mirrors por decisao explicita (ver allowlist neste script)"
done

# S3: o Codex carrega hooks de projeto de UMA unica representacao por camada.
# `.codex/hooks.json` legado coexistindo com `.codex/config.toml` faz o CLI emitir
# "loading hooks from both ...; prefer a single representation for this layer" —
# e internal/install proibe essa coexistencia nos projetos instalados.
codex_dual_drift=0
if [[ -f "$repo_root/.codex/config.toml" && -f "$repo_root/.codex/hooks.json" ]]; then
  echo "DRIFT: .codex/hooks.json legado coexiste com .codex/config.toml (representacao dupla de hooks do Codex)"
  codex_dual_drift=1
fi

opencode_plugin_drift=0
opencode_plugin_embedded="$repo_root/internal/embedded/assets/.opencode/plugin/governance.js"
opencode_plugin_mirror="$repo_root/.opencode/plugin/governance.js"
if [[ ! -f "$opencode_plugin_embedded" ]]; then
  echo "MISSING: $opencode_plugin_embedded (plugin do OpenCode)"
  opencode_plugin_drift=1
elif [[ ! -f "$opencode_plugin_mirror" ]]; then
  echo "MISSING: $opencode_plugin_mirror (espelho de raiz do plugin do OpenCode)"
  opencode_plugin_drift=1
elif ! diff -q "$opencode_plugin_mirror" "$opencode_plugin_embedded" >/dev/null 2>&1; then
  echo "DRIFT: governance.js diverge entre .opencode/plugin e internal/embedded/assets/.opencode/plugin"
  opencode_plugin_drift=1
fi

if [[ "$drift_count" -gt 0 || "$session_end_drift" -gt 0 || "$tool_validation_drift" -gt 0 || "$opencode_plugin_drift" -gt 0 || "$local_only_drift" -gt 0 || "$codex_dual_drift" -gt 0 ]]; then
  echo
  echo "Para corrigir: ./scripts/sync-hooks.sh"
  exit 1
fi

echo "Todos os hooks orquestrador sincronizados."
exit 0
