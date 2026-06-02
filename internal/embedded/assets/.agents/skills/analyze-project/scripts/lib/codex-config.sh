#!/usr/bin/env bash
# build_codex_config: gera o conteudo de .codex/config.toml para o projeto alvo.
# Inclui as skills transversais sempre habilitadas mais as skills de linguagem
# detectadas pelas flags (0/1) passadas. Encerra com sandbox_mode + approval_policy
# (ADR-002, fecha lacuna de route-around do PreToolUse do Codex) e [hooks] padrao.
#
# Uso: build_codex_config <go_flag> <node_flag> <python_flag> [dotnet_flag=0]

build_codex_config() {
  local go_flag="${1:-0}"
  local node_flag="${2:-0}"
  local python_flag="${3:-0}"
  local dotnet_flag="${4:-0}"

  _codex_emit_skill() {
    printf '[[skills.config]]\npath = "%s"\nenabled = true\n\n' "$1"
  }

  # Skills transversais (sempre habilitadas em qualquer projeto)
  local transversal=(
    "create-prd"
    "create-technical-specification"
    "create-tasks"
    "execute-task"
    "execute-all-tasks"
    "refactor"
    "review"
    "analyze-project"
    "agent-governance"
    "bugfix"
    "github-diff-changelog-publisher"
    "github-pr-comment-triage"
    "github-release-publication-flow"
    "pull-request"
    "prompt-enricher"
    "semantic-commit"
    "us-to-prd"
  )

  local skill
  for skill in "${transversal[@]}"; do
    _codex_emit_skill ".agents/skills/$skill"
  done

  # Skills de linguagem (carregadas conditionalmente)
  if [[ "$go_flag" == "1" ]]; then
    _codex_emit_skill ".agents/skills/go-implementation"
    _codex_emit_skill ".agents/skills/object-calisthenics-go"
  fi
  if [[ "$node_flag" == "1" ]]; then
    _codex_emit_skill ".agents/skills/node-implementation"
  fi
  if [[ "$python_flag" == "1" ]]; then
    _codex_emit_skill ".agents/skills/python-implementation"
  fi
  if [[ "$dotnet_flag" == "1" ]]; then
    _codex_emit_skill ".agents/skills/dotnet-csharp-implementation"
  fi

  cat <<'CODEX_GOVERNANCE'

# Governanca (paridade cross-CLI): o hook PreToolUse do Codex tem lacuna de
# route-around documentada. sandbox_mode + approval_policy garantem que mudancas
# de filesystem e comandos passem por aprovacao, fechando a lacuna (ADR-002).
sandbox_mode = "workspace-write"
approval_policy = "on-request"

[[hooks.PreToolUse]]
[[hooks.PreToolUse.hooks]]
type = "command"
command = "bash .codex/hooks/validate-preload.sh"

[[hooks.PostToolUse]]
[[hooks.PostToolUse.hooks]]
type = "command"
command = "bash .codex/hooks/validate-governance.sh"
CODEX_GOVERNANCE

  unset -f _codex_emit_skill
}
