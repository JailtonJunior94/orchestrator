#!/usr/bin/env bash
# check-embedded-parity.sh
# Verifica paridade entre internal/embedded/assets/.agents/skills/
# e o repositório remoto JailtonJunior94/ai-governance@e83a1db
#
# Uso: ./scripts/check-embedded-parity.sh [--fix]
# --fix: baixa e sobrescreve arquivos com drift do remoto

set -euo pipefail

REMOTE_REPO="JailtonJunior94/ai-governance"
REMOTE_REF="e83a1db"
EMBEDDED_BASE="internal/embedded/assets/.agents/skills"
REMOTE_BASE=".agents/skills"
TREE_JSON_FILE=$(mktemp /tmp/parity-tree.XXXXXX.json)

FIX=false
if [[ "${1:-}" == "--fix" ]]; then
  FIX=true
fi

MISSING=0
EXTRA=0
DRIFT=0
OK=0

# Lookup helper: get remote SHA for a given path
remote_sha() {
  python3 -c "
import json, sys
path = sys.argv[1]
with open('${TREE_JSON_FILE}') as f:
    tree = json.load(f)
for item in tree:
    if item['path'] == path:
        print(item['sha'])
        sys.exit(0)
" "$1"
}

# Download file content from remote
download_remote() {
  local remote_path="$1"
  local local_path="$2"
  mkdir -p "$(dirname "${local_path}")"
  gh api "repos/${REMOTE_REPO}/contents/${remote_path}?ref=${REMOTE_REF}" \
    --jq '.content' | python3 -c "
import sys, base64
content = sys.stdin.read().replace('\n', '')
sys.stdout.buffer.write(base64.b64decode(content))
" > "${local_path}"
}

trap 'rm -f "${TREE_JSON_FILE}"' EXIT

echo "=== Parity Check: embedded vs ${REMOTE_REPO}@${REMOTE_REF} ==="
echo ""

# Obter tree completa do remoto em uma única chamada
echo "Buscando tree remota..."
gh api "repos/${REMOTE_REPO}/git/trees/${REMOTE_REF}?recursive=1" \
  --jq '[.tree[] | select(.type=="blob") | select(.path | startswith(".agents/skills")) | {path: .path, sha: .sha}]' \
  > "${TREE_JSON_FILE}"

# Extrair paths remotos
REMOTE_PATHS=$(python3 -c "
import json
with open('${TREE_JSON_FILE}') as f:
    tree = json.load(f)
for item in tree:
    print(item['path'])
")

# Extrair paths locais (relativizados)
LOCAL_PATHS=$(find "${EMBEDDED_BASE}" -type f | sed "s|${EMBEDDED_BASE}/||" | sort)

echo "### Verificando arquivos do remoto contra embedded ###"
echo ""

while IFS= read -r REMOTE_PATH; do
  REL_PATH="${REMOTE_PATH#${REMOTE_BASE}/}"
  LOCAL_PATH="${EMBEDDED_BASE}/${REL_PATH}"

  if [[ ! -f "${LOCAL_PATH}" ]]; then
    echo "[MISSING] ${REL_PATH}"
    MISSING=$((MISSING + 1))
    if [[ "${FIX}" == "true" ]]; then
      printf "          -> Baixando... "
      download_remote "${REMOTE_PATH}" "${LOCAL_PATH}"
      echo "OK"
    fi
    continue
  fi

  # git hash-object calcula SHA1("blob {size}\0{content}") — mesmo que o GitHub usa
  LOCAL_SHA=$(git hash-object "${LOCAL_PATH}")
  REMOTE_SHA=$(remote_sha "${REMOTE_PATH}")

  if [[ "${LOCAL_SHA}" != "${REMOTE_SHA}" ]]; then
    echo "[DRIFT]   ${REL_PATH}"
    echo "          local:  ${LOCAL_SHA}"
    echo "          remote: ${REMOTE_SHA}"
    DRIFT=$((DRIFT + 1))
    if [[ "${FIX}" == "true" ]]; then
      printf "          -> Sincronizando... "
      download_remote "${REMOTE_PATH}" "${LOCAL_PATH}"
      echo "OK"
    fi
  else
    echo "[OK]      ${REL_PATH}"
    OK=$((OK + 1))
  fi
done <<< "${REMOTE_PATHS}"

echo ""
echo "### Verificando arquivos extras no embedded (ausentes no remoto) ###"
echo ""

while IFS= read -r LOCAL_REL; do
  REMOTE_PATH="${REMOTE_BASE}/${LOCAL_REL}"
  EXISTS=$(python3 -c "
import json, sys
path = sys.argv[1]
with open('${TREE_JSON_FILE}') as f:
    tree = json.load(f)
for item in tree:
    if item['path'] == path:
        print('yes')
        sys.exit(0)
print('no')
" "${REMOTE_PATH}")

  if [[ "${EXISTS}" == "no" ]]; then
    echo "[EXTRA]   ${LOCAL_REL}  (local only)"
    EXTRA=$((EXTRA + 1))
  fi
done <<< "${LOCAL_PATHS}"

echo ""
echo "=== Relatório Final ==="
printf "OK:      %d\n" "${OK}"
printf "MISSING: %d  (existe no remoto, ausente no embedded)\n" "${MISSING}"
printf "DRIFT:   %d  (conteudo divergente)\n" "${DRIFT}"
printf "EXTRA:   %d  (apenas no embedded — adaptacoes locais)\n" "${EXTRA}"
echo ""

if [[ "${DRIFT}" -gt 0 || "${MISSING}" -gt 0 ]] && [[ "${FIX}" == "false" ]]; then
  echo "Dica: execute com --fix para sincronizar drift e baixar arquivos ausentes."
fi

if [[ "${MISSING}" -eq 0 && "${DRIFT}" -eq 0 ]]; then
  echo "Resultado: ZERO divergencias nao documentadas. Paridade OK."
  exit 0
else
  echo "Resultado: existem divergencias. Revise os itens acima."
  exit 1
fi
