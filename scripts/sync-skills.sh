#!/usr/bin/env bash
# Sincroniza skills do diretório canônico (.agents/skills) para os mirrors
# consumidos por Claude Code (.claude/skills) e GitHub Copilot (.github/skills),
# e para o bundle embedded (internal/embedded/assets/.agents/skills) usado pelo
# binário ai-spec via go:embed e distribuído via `ai-spec upgrade`.
#
# Gemini consome o canônico via .gemini/commands/<skill>.toml e não exige cópia.
# Codex não usa skills neste formato.
#
# Estratégia: rsync com --delete para garantir que mirrors sejam idênticos ao canônico.
# Os mirrors são GERADOS — não edite-os à mão; altere o canônico (.agents/skills) e rode
# este script. A fonte de verdade é sinalizada por este comentário e garantida pelo gate
# `check-skills-sync.sh` (que falha em drift), NÃO por permissão read-only: marcar os
# arquivos como read-only quebra operações do git (`unable to unlink`/checkout/pull).

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.agents/skills"

declare -a platform_mirrors=(
  "$repo_root/.claude/skills"
  "$repo_root/.github/skills"
)

embedded_mirror="$repo_root/internal/embedded/assets/.agents/skills"

if [[ ! -d "$canonical" ]]; then
  echo "ERRO: diretório canônico não encontrado: $canonical" >&2
  exit 1
fi

# Auto-heal: limpa estado read-only herdado de versões antigas deste script no canônico,
# garantindo que git/rsync/cp consigam operar e que `cp -p` não re-propague read-only.
chmod -R u+w "$canonical" 2>/dev/null || true

for mirror in "${platform_mirrors[@]}"; do
  mkdir -p "$mirror"
  # Cura estado read-only herdado de versões antigas deste script (que aplicavam a-w),
  # para que rsync e o git consigam sobrescrever/remover os arquivos do mirror.
  if [[ -d "$mirror" ]]; then
    chmod -R u+w "$mirror" 2>/dev/null || true
  fi

  # Iterar sobre cada skill presente no canônico e copiar apenas as que
  # já existem no mirror (preservando o subset por plataforma).
  for skill_dir in "$canonical"/*/; do
    skill_name="$(basename "$skill_dir")"
    if [[ -d "$mirror/$skill_name" ]]; then
      rsync -a --delete "$skill_dir" "$mirror/$skill_name/"
      echo "synced: $skill_name -> $mirror"
    fi
  done
done

# Bundle embedded — replica skills do canônico para garantir que
# `ai-spec upgrade` distribua a mesma versão consumida localmente.
# Sem chmod a-w: o diretório é embed source consumido por go:embed.
#
# Importante: pular skills cujo canônico não tenha SKILL.md (ex.: skills com apenas
# `references/`). Isso preserva o estado histórico do embedded e evita que
# rsync --delete apague conteúdo válido do bundle por causa de um canônico parcial.
mkdir -p "$embedded_mirror"
chmod -R u+w "$embedded_mirror" 2>/dev/null || true
for skill_dir in "$canonical"/*/; do
  skill_name="$(basename "$skill_dir")"
  if [[ ! -d "$embedded_mirror/$skill_name" ]]; then
    continue
  fi
  if [[ ! -f "$skill_dir/SKILL.md" ]]; then
    echo "skipped: $skill_name (sem SKILL.md no canônico) -> $embedded_mirror"
    continue
  fi
  rsync -a --delete "$skill_dir" "$embedded_mirror/$skill_name/"
  echo "synced: $skill_name -> $embedded_mirror"
done

# Sincroniza .agents/lib/ -> scripts/lib/ (B1: vendor canônico em .agents/lib/,
# mirror legado em scripts/lib/ para compatibilidade retroativa com callers que
# resolvem `scripts/lib/check-invocation-depth.sh` antes do fallback).
# Também sincroniza para o embedded mirror para que `ai-spec install` distribua
# o vendor canônico em projetos consumidores.
agents_lib="$repo_root/.agents/lib"
legacy_lib="$repo_root/scripts/lib"
embedded_lib="$repo_root/internal/embedded/assets/.agents/lib"
if [[ -d "$agents_lib" ]]; then
  chmod -R u+w "$agents_lib" 2>/dev/null || true
  chmod -R u+w "$legacy_lib" 2>/dev/null || true
  chmod -R u+w "$embedded_lib" 2>/dev/null || true
  mkdir -p "$embedded_lib"
  for lib_file in "$agents_lib"/*.sh; do
    [[ -f "$lib_file" ]] || continue
    base="$(basename "$lib_file")"
    cp -p "$lib_file" "$legacy_lib/$base"
    echo "synced: lib/$base -> $legacy_lib"
    cp -p "$lib_file" "$embedded_lib/$base"
    echo "synced: lib/$base -> $embedded_lib"
  done
fi

# Sincroniza hooks canônicos do orquestrador (.agents/hooks/) para todos os mirrors
# por-tool (.claude/.codex/.gemini/.github e internal/embedded/assets/<tool>/).
# Os 4 hooks abaixo são idênticos em todos os tools por design (executados via
# `bash .<tool>/hooks/<hook>.sh` pelo próprio skill).
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
  "$repo_root/.gemini/hooks"
  "$repo_root/.github/hooks"
  "$repo_root/internal/embedded/assets/.agents/hooks"
  "$repo_root/internal/embedded/assets/.claude/hooks"
  "$repo_root/internal/embedded/assets/.codex/hooks"
  "$repo_root/internal/embedded/assets/.gemini/hooks"
  "$repo_root/internal/embedded/assets/.github/hooks"
)
if [[ -d "$agents_hooks" ]]; then
  for mirror in "${tool_hook_mirrors[@]}"; do
    mkdir -p "$mirror"
    for hook in "${orchestrator_hooks[@]}"; do
      src="$agents_hooks/$hook"
      [[ -f "$src" ]] || continue
      cp -p "$src" "$mirror/$hook"
      chmod +x "$mirror/$hook" 2>/dev/null || true
    done
    echo "synced: orchestrator hooks -> $mirror"
  done
fi

# NÃO reaplicar read-only: arquivos/diretórios read-only quebram operações do git
# (`unable to unlink old ...`, checkout, pull, restore). A imutabilidade da fonte é
# garantida pelo gate `check-skills-sync.sh`, não por permissão de filesystem.

echo "sync-skills: concluído"
