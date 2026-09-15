#!/usr/bin/env bash
# check-skills-sync.sh
# Detecta drift entre o diretório canônico (.agents/skills/) e os mirrors:
# .claude/skills/, .github/skills/, internal/embedded/assets/.agents/skills/
#
# Uso: ./scripts/check-skills-sync.sh
# Exit 0 quando todos os mirrors estão sincronizados; exit 1 quando há drift.
#
# Não modifica arquivos. Para corrigir drift, rode: ./scripts/sync-skills.sh

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.agents/skills"

if [[ ! -d "$canonical" ]]; then
  echo "ERRO: diretório canônico não encontrado: $canonical" >&2
  exit 1
fi

declare -a mirrors=(
  "$repo_root/.claude/skills"
  "$repo_root/.github/skills"
  "$repo_root/internal/embedded/assets/.agents/skills"
)

# S1: dirs sob .agents/skills/ que NAO sao skills (sem SKILL.md) e portanto nao
# devem ser espelhados. Allowlist EXPLICITA — ausencia de SKILL.md por si so nao
# isenta o dir do gate.
#   tests/ -> suite pytest (conftest.py + test_validation_scripts.py) que valida os
#             scripts das skills; nao tem SKILL.md e nao e carregavel como skill.
declare -a non_skill_dirs=(
  "tests"
)

is_non_skill_dir() {
  local candidate="$1"
  local entry
  for entry in "${non_skill_dirs[@]}"; do
    [[ "$entry" == "$candidate" ]] && return 0
  done
  return 1
}

drift_count=0
ok_count=0

for mirror in "${mirrors[@]}"; do
  if [[ ! -d "$mirror" ]]; then
    echo "DRIFT: mirror nao existe: $mirror"
    drift_count=$((drift_count + 1))
    continue
  fi

  # S1: a fonte de verdade e o CANONICO. Skill canonica ausente no mirror e DRIFT,
  # nao subset legitimo — antes dessa correcao uma skill nunca embarcada era
  # invisivel ao gate (ex.: domain-modeling-production, exigida pelo tasks.md).
  for skill_dir in "$canonical"/*/; do
    skill_name="$(basename "$skill_dir")"

    if is_non_skill_dir "$skill_name"; then
      echo "SKIP: $skill_name (allowlist: dir nao-skill em $canonical)"
      continue
    fi

    if [[ ! -f "$skill_dir/SKILL.md" ]]; then
      echo "DRIFT: $skill_name existe em $canonical sem SKILL.md e nao esta na allowlist de dirs nao-skill"
      drift_count=$((drift_count + 1))
      continue
    fi

    mirror_skill="$mirror/$skill_name"
    if [[ ! -d "$mirror_skill" ]]; then
      echo "DRIFT: $skill_name existe em $canonical mas nao em $mirror"
      drift_count=$((drift_count + 1))
      continue
    fi

    if ! diff -r "$skill_dir" "$mirror_skill" > /dev/null 2>&1; then
      echo "DRIFT: $skill_name diverge entre $canonical e $mirror"
      diff -rq "$skill_dir" "$mirror_skill" 2>&1 | sed 's/^/  /' || true
      drift_count=$((drift_count + 1))
    else
      ok_count=$((ok_count + 1))
    fi
  done

  # Orfaos: presentes no mirror sem contrapartida canonica.
  for skill_dir in "$mirror"/*/; do
    skill_name="$(basename "$skill_dir")"
    if [[ ! -d "$canonical/$skill_name" ]]; then
      echo "DRIFT: $mirror/$skill_name existe mas canonical $canonical/$skill_name nao"
      drift_count=$((drift_count + 1))
    fi
  done
done

echo
echo "Skills em sync: $ok_count"
echo "Skills com drift: $drift_count"

# B1: validar paridade .agents/lib/ <-> scripts/lib/ (vendor canônico vs mirror legado)
# E .agents/lib/ <-> internal/embedded/assets/.agents/lib/ (vendor canônico vs embedded).
agents_lib="$repo_root/.agents/lib"
legacy_lib="$repo_root/scripts/lib"
embedded_lib="$repo_root/internal/embedded/assets/.agents/lib"

# G1: a lista e DECLARADA, nao derivada de glob. Sem ela, apagar .agents/lib/ (ou
# esvazia-lo) fazia o bloco inteiro nao executar nenhuma comparacao e o gate
# aprovava por vacuidade. Ausencia do canonico e DRIFT, nunca motivo para pular.
declare -a required_libs=(
  "check-invocation-depth.sh"
  "parse-hook-input.sh"
)
declare -a lib_mirrors=(
  "$legacy_lib"
  "$embedded_lib"
)

lib_drift=0
if [[ ! -d "$agents_lib" ]]; then
  echo "DRIFT lib: diretorio canonico ausente: $agents_lib"
  lib_drift=$((lib_drift + 1))
else
  for base in "${required_libs[@]}"; do
    lib_file="$agents_lib/$base"
    if [[ ! -f "$lib_file" ]]; then
      echo "DRIFT lib: $base declarado como obrigatorio mas ausente em .agents/lib/"
      lib_drift=$((lib_drift + 1))
      continue
    fi
    for mirror in "${lib_mirrors[@]}"; do
      if [[ ! -d "$mirror" ]]; then
        echo "DRIFT lib: mirror nao existe: $mirror"
        lib_drift=$((lib_drift + 1))
        continue
      fi
      if [[ ! -f "$mirror/$base" ]]; then
        echo "DRIFT lib: $base existe em .agents/lib/ mas nao em $mirror"
        lib_drift=$((lib_drift + 1))
      elif ! diff -q "$lib_file" "$mirror/$base" > /dev/null 2>&1; then
        echo "DRIFT lib: $base diverge entre .agents/lib/ e $mirror"
        lib_drift=$((lib_drift + 1))
      fi
    done
  done

  # Arquivo novo no canonico sem entrada na lista declarada tambem e DRIFT:
  # caso contrario a lista envelhece em silencio.
  for lib_file in "$agents_lib"/*.sh; do
    [[ -f "$lib_file" ]] || continue
    base="$(basename "$lib_file")"
    declared=0
    for entry in "${required_libs[@]}"; do
      [[ "$entry" == "$base" ]] && { declared=1; break; }
    done
    if [[ "$declared" -eq 0 ]]; then
      echo "DRIFT lib: $base existe em .agents/lib/ mas nao esta na lista declarada required_libs de $0"
      lib_drift=$((lib_drift + 1))
    fi
  done

  if [[ "$lib_drift" -eq 0 ]]; then
    echo "Libs em sync: ${#required_libs[@]} libs x ${#lib_mirrors[@]} mirrors (legacy + embedded)"
  fi
fi

# A01: validar paridade de hooks canônicos do orquestrador (.agents/hooks/) com
# todos os mirrors por-tool (incluindo embedded). Garante que execute-all-tasks
# e execute-task encontram o mesmo hook independente do CLI usado.
agents_hooks="$repo_root/.agents/hooks"
declare -a orchestrator_hooks=(
  "post-execute-task.sh"
  "post-wave.sh"
  "pre-execute-all-tasks.sh"
  "subagent-stop-wrapper.sh"
)
declare -a tool_hook_mirrors=(
  "$repo_root/.claude/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.agents/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.codex/hooks"
  "$repo_root/internal/embedded/assets/.github/hooks"
)
hook_drift=0
if [[ ! -d "$agents_hooks" ]]; then
  echo "DRIFT hook: diretorio canonico ausente: $agents_hooks"
  hook_drift=$((hook_drift + 1))
else
  # G1: hook canonico ausente e DRIFT. O `continue` anterior transformava a
  # ausencia do canonico em aprovacao silenciosa de todos os mirrors.
  for hook in "${orchestrator_hooks[@]}"; do
    src="$agents_hooks/$hook"
    if [[ ! -f "$src" ]]; then
      echo "DRIFT hook: $hook declarado como canonico mas ausente em $agents_hooks"
      hook_drift=$((hook_drift + 1))
      continue
    fi
    for mirror in "${tool_hook_mirrors[@]}"; do
      if [[ ! -d "$mirror" ]]; then
        echo "DRIFT hook: mirror nao existe: $mirror"
        hook_drift=$((hook_drift + 1))
        continue
      fi
      dst="$mirror/$hook"
      if [[ ! -f "$dst" ]]; then
        echo "DRIFT hook: $hook ausente em $mirror"
        hook_drift=$((hook_drift + 1))
        continue
      fi
      if ! diff -q "$src" "$dst" > /dev/null 2>&1; then
        echo "DRIFT hook: $hook diverge entre .agents/hooks/ e $mirror"
        hook_drift=$((hook_drift + 1))
      fi
    done
  done
  if [[ "$hook_drift" -eq 0 ]]; then
    echo "Hooks do orquestrador em sync: ${#orchestrator_hooks[@]} hooks x ${#tool_hook_mirrors[@]} mirrors"
  fi
fi

# A01: validar presenca de hooks de validacao por-tool em todos os tools.
# validate-preload.sh: gate de carga base de governanca (existe em todos).
# validate-governance.sh: aviso pos-edicao em arquivos de governanca (existe em todos).
# Conteudo difere por mecanica de invocacao do tool (stdin JSON vs env vs $1);
# checamos apenas existencia, nao diff.
declare -a validation_hooks=(
  "validate-preload.sh"
  "validate-governance.sh"
)
declare -a tool_dirs=(
  "$repo_root/.claude/hooks"
  "$repo_root/.codex/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.codex/hooks"
  "$repo_root/internal/embedded/assets/.github/hooks"
)
validation_drift=0
for tool_dir in "${tool_dirs[@]}"; do
  [[ -d "$tool_dir" ]] || { echo "DRIFT validation: dir ausente: $tool_dir"; validation_drift=$((validation_drift + 1)); continue; }
  for hook in "${validation_hooks[@]}"; do
    if [[ ! -f "$tool_dir/$hook" ]]; then
      echo "DRIFT validation: $hook ausente em $tool_dir"
      validation_drift=$((validation_drift + 1))
    fi
  done
done
if [[ "$validation_drift" -eq 0 ]]; then
  echo "Hooks de validacao por-tool em paridade: ${#validation_hooks[@]} hooks x ${#tool_dirs[@]} tools"
fi

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
  echo "DRIFT session-end gate: canonico ausente: $session_end_canonical"
  session_end_drift=$((session_end_drift + 1))
else
  for mirror in "${session_end_gate_dirs[@]}"; do
    mirror_path="$mirror/validate-session-end.sh"
    if [[ ! -f "$mirror_path" ]]; then
      echo "DRIFT session-end gate: validate-session-end.sh ausente em $mirror"
      session_end_drift=$((session_end_drift + 1))
      continue
    fi
    if ! diff -q "$session_end_canonical" "$mirror_path" > /dev/null 2>&1; then
      echo "DRIFT session-end gate: validate-session-end.sh diverge entre .agents/scripts e $mirror"
      session_end_drift=$((session_end_drift + 1))
    fi
  done
  if [[ "$session_end_drift" -eq 0 ]]; then
    echo "Gate de encerramento em paridade nos 4 agentes oficiais: ${#session_end_gate_dirs[@]} mirrors"
  fi
fi

opencode_plugin_drift=0
opencode_plugin_embedded="$repo_root/internal/embedded/assets/.opencode/plugin/governance.js"
opencode_plugin_root="$repo_root/.opencode/plugin/governance.js"
if [[ ! -f "$opencode_plugin_embedded" ]]; then
  echo "DRIFT opencode plugin: governance.js ausente em internal/embedded/assets/.opencode/plugin/"
  opencode_plugin_drift=1
elif [[ ! -f "$opencode_plugin_root" ]]; then
  echo "DRIFT opencode plugin: governance.js ausente em .opencode/plugin/"
  opencode_plugin_drift=1
elif ! diff -q "$opencode_plugin_root" "$opencode_plugin_embedded" > /dev/null 2>&1; then
  echo "DRIFT opencode plugin: governance.js diverge entre .opencode/plugin e internal/embedded/assets/.opencode/plugin"
  opencode_plugin_drift=1
fi
if [[ "$opencode_plugin_drift" -eq 0 ]]; then
  echo "Plugin do OpenCode presente e em paridade: internal/embedded/assets/.opencode/plugin/governance.js"
fi

if [[ "$drift_count" -gt 0 || "$lib_drift" -gt 0 || "$hook_drift" -gt 0 || "$validation_drift" -gt 0 || "$session_end_drift" -gt 0 || "$opencode_plugin_drift" -gt 0 ]]; then
  echo
  echo "Para corrigir: ./scripts/sync-skills.sh"
  exit 1
fi

echo "Todos os mirrors sincronizados com .agents/skills/, .agents/lib/ e .agents/hooks/"
exit 0
