#!/usr/bin/env bash
# scripts/sync-acp-sdk-version.sh
#
# Mantém a constante ClaudeSDKVersion em internal/runtime/specs/claude.go
# sincronizada com a versão do github.com/coder/acp-go-sdk declarada em go.mod.
#
# Uso: bash scripts/sync-acp-sdk-version.sh
#
# Comportamento:
#   - Lê go.mod e extrai a versão do coder/acp-go-sdk.
#   - Compara com o valor atual em claude.go.
#   - Se divergir: aplica substituição idempotente, emite mensagem em stderr, exit 0.
#   - Se igual: não altera o arquivo, exit 0.
#   - Em caso de erro (go.mod ausente, pacote não encontrado, parse falhou): exit 1.
#
# Restrições (ADR-009 + PRD §"Restrições Técnicas"):
#   - Nunca atualiza ClaudeNpmVersion automaticamente (requer processo audit/).
#   - Substituição é idempotente: rodar N vezes produz o mesmo resultado que rodar 1 vez.

set -euo pipefail

GOMOD="${GOMOD_PATH:-go.mod}"
CLAUDE_GO="${CLAUDE_GO_PATH:-internal/runtime/specs/claude.go}"
PACKAGE="github.com/coder/acp-go-sdk"

if [[ ! -f "$GOMOD" ]]; then
  echo "ERRO: $GOMOD não encontrado" >&2
  exit 1
fi

if [[ ! -f "$CLAUDE_GO" ]]; then
  echo "ERRO: $CLAUDE_GO não encontrado" >&2
  exit 1
fi

# Extrair versão do pacote em go.mod.
# Aceita tanto formato de bloco (	pkg vX.Y.Z) quanto linha direta (require pkg vX.Y.Z).
# grep -F faz match literal da string do pacote, awk extrai o campo da versão.
sdk_version="$(grep -F "$PACKAGE" "$GOMOD" | awk '{for(i=1;i<=NF;i++) if($i ~ /^v[0-9]/) {print $i; exit}}' | head -1)"

if [[ -z "$sdk_version" ]]; then
  echo "ERRO: pacote '${PACKAGE}' não encontrado em $GOMOD" >&2
  exit 1
fi

# Validar formato semântico (vX.Y.Z ou vX.Y.Z-suffix)
if ! [[ "$sdk_version" =~ ^v[0-9]+\.[0-9]+\.[0-9] ]]; then
  echo "ERRO: versão '$sdk_version' não é semântica (esperado vX.Y.Z)" >&2
  exit 1
fi

# Ler o valor atual da constante
current_version="$(grep -E 'ClaudeSDKVersion\s*=\s*"v[^"]*"' "$CLAUDE_GO" | sed -E 's/.*"(v[^"]+)".*/\1/' | head -1)"

if [[ -z "$current_version" ]]; then
  echo "ERRO: constante ClaudeSDKVersion não encontrada em $CLAUDE_GO" >&2
  exit 1
fi

if [[ "$current_version" == "$sdk_version" ]]; then
  # Já sincronizado — exit 0 sem alterações
  exit 0
fi

# Aplicar substituição idempotente (substitui apenas a linha da constante)
# Usa um delimitador alternativo para evitar problemas com barras no path
sed -i.bak -E "s|(ClaudeSDKVersion[[:space:]]*=[[:space:]]*\")v[^\"]*(\")|\1${sdk_version}\2|" "$CLAUDE_GO"
rm -f "${CLAUDE_GO}.bak"

echo "sync-acp-sdk-version: ClaudeSDKVersion atualizada de $current_version para $sdk_version em $CLAUDE_GO" >&2
