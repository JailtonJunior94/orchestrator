#!/usr/bin/env bash
# hook-prereq-gate.sh
# Gate compartilhado invocado pelos hooks PreToolUse dos 3 CLIs inegociaveis
# (Claude, Codex, Copilot). Resolve em duas etapas:
#
#   1. validate-skill-prerequisites.sh — bloqueia se a skill da stack tocada
#      nao tiver SKILL.md + references/INDEX.yaml acessiveis (camada de
#      descoberta ausente = alucinacao garantida).
#   2. resolve-references.sh — emite em stderr a LISTA CIRURGICA de references
#      que o agente deve carregar para esta edicao especifica. Custo zero quando
#      a tarefa nao bate em nada (so 'always' refs sao listadas).
#
# Saida:
#   exit 0 — pode prosseguir; lista cirurgica em stderr como GUIDANCE
#   exit 1 — bloqueio: skill obrigatoria ausente (mensagem em stderr)
#
# Variaveis:
#   AGENTS_ROOT        raiz do projeto onde .agents/skills/ resolve.
#   PREREQ_MODE=warn   nao bloqueia mesmo com skill ausente (default: fail).
#   GATE_DIFF          conteudo de diff a passar via stdin do resolver
#                      (alternativamente, gate le seu proprio stdin se tty=false).
#
# Uso (CLI-agnostico):
#   bash .agents/scripts/hook-prereq-gate.sh <files...>
#   echo "<diff>" | bash .agents/scripts/hook-prereq-gate.sh <files...>

set -euo pipefail

AGENTS_ROOT="${AGENTS_ROOT:-$(pwd)}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

files=("$@")
if [[ ${#files[@]} -eq 0 ]]; then
  exit 0  # nada a validar (ex.: comando read-only)
fi

# Le diff de stdin uma vez para reutilizar nos resolvers das varias skills.
diff_text=""
if [[ ! -t 0 ]]; then
  diff_text="$(cat)"
fi
if [[ -n "${GATE_DIFF:-}" ]]; then
  diff_text="$GATE_DIFF"
fi

# Etapa 1: bloqueio inegociavel. Validador e tool-neutro e shell-only.
prereq_script="$SCRIPT_DIR/validate-skill-prerequisites.sh"
if [[ ! -x "$prereq_script" && ! -f "$prereq_script" ]]; then
  echo "AVISO: validate-skill-prerequisites.sh ausente em $SCRIPT_DIR" >&2
  exit 0  # degrada para no-op em ambientes sem o gate
fi
if ! AGENTS_ROOT="$AGENTS_ROOT" bash "$prereq_script" "${files[@]}"; then
  exit 1
fi

# Etapa 2: emite guidance cirurgica em stderr (nunca bloqueia por aqui).
# Mapeia extensao -> skill para sumir overhead quando arquivos misturam stacks.
resolve_script="$SCRIPT_DIR/resolve-references.sh"
if [[ ! -x "$resolve_script" && ! -f "$resolve_script" ]]; then
  exit 0
fi

# Coleta skills unicas a partir das extensoes
skills=""
for f in "${files[@]}"; do
  case "$f" in
    *.go) skills="$skills go-implementation" ;;
    *.ts|*.tsx|*.js|*.jsx|*.mjs|*.cjs) skills="$skills node-implementation" ;;
    *.cs|*.csproj) skills="$skills dotnet-csharp-implementation" ;;
    *.py) skills="$skills python-implementation" ;;
  esac
done
skills_uniq=$(printf '%s\n' $skills | awk 'NF && !seen[$0]++' | tr '\n' ' ')
skills_trim="${skills_uniq// /}"

if [[ -z "$skills_trim" ]]; then
  exit 0  # nenhuma extensao mapeada para skill — silencio (sem header vazio)
fi

echo "GUIDANCE (carga surgical): references a carregar para esta edicao:" >&2
for skill in $skills_uniq; do
  [[ -z "$skill" ]] && continue
  # Passa apenas os files relevantes a cada skill (filtragem por extensao)
  skill_files=()
  for f in "${files[@]}"; do
    case "$skill:$f" in
      go-implementation:*.go) skill_files+=("$f") ;;
      node-implementation:*.ts|node-implementation:*.tsx|node-implementation:*.js|node-implementation:*.jsx|node-implementation:*.mjs|node-implementation:*.cjs) skill_files+=("$f") ;;
      dotnet-csharp-implementation:*.cs|dotnet-csharp-implementation:*.csproj) skill_files+=("$f") ;;
      python-implementation:*.py) skill_files+=("$f") ;;
    esac
  done
  if [[ ${#skill_files[@]} -eq 0 ]]; then
    continue
  fi
  if [[ -n "$diff_text" ]]; then
    AGENTS_ROOT="$AGENTS_ROOT" bash "$resolve_script" "$skill" "${skill_files[@]}" <<<"$diff_text" >&2 2>/dev/null || true
  else
    AGENTS_ROOT="$AGENTS_ROOT" bash "$resolve_script" "$skill" "${skill_files[@]}" >&2 2>/dev/null || true
  fi
done

exit 0
