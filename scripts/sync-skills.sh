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
# Mirrors de plataforma (.claude/.github) recebem read-only para sinalizar fonte de
# verdade; o mirror embedded permanece gravável por convenção (é embed source de Go).

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
canonical="$repo_root/.agents/skills"

declare -a mirrors_readonly=(
  "$repo_root/.claude/skills"
  "$repo_root/.github/skills"
)

embedded_mirror="$repo_root/internal/embedded/assets/.agents/skills"

if [[ ! -d "$canonical" ]]; then
  echo "ERRO: diretório canônico não encontrado: $canonical" >&2
  exit 1
fi

for mirror in "${mirrors_readonly[@]}"; do
  mkdir -p "$mirror"
  # Tornar arquivos do mirror graváveis antes do rsync para evitar erro em read-only.
  if [[ -d "$mirror" ]]; then
    chmod -R u+w "$mirror" 2>/dev/null || true
  fi

  # Iterar sobre cada skill presente no canônico e copiar apenas as que
  # já existem no mirror (preservando o subset por plataforma).
  for skill_dir in "$canonical"/*/; do
    skill_name="$(basename "$skill_dir")"
    if [[ -d "$mirror/$skill_name" ]]; then
      rsync -a --delete "$skill_dir" "$mirror/$skill_name/"
      chmod -R a-w "$mirror/$skill_name"
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

# Reaplica read-only no canônico para reforçar a fonte de verdade.
chmod -R a-w "$canonical"

echo "sync-skills: concluído"
