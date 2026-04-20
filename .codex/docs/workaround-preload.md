# Workaround de Preload de Governanca — Codex

> Carregar este arquivo apenas se o hook `.codex/hooks/validate-preload.sh` falhar ou nao estiver presente.

## Contexto

O Codex nao suporta hooks de pre/pos-edicao equivalentes ao Claude Code (PreToolUse/PostToolUse). O arquivo `.codex/config.toml` lista apenas as skills disponiveis para resolucao pelo harness — nao ha mecanismo para interceptar operacoes de edicao de arquivo.

## Capacidades do Codex

| Mecanismo | Suporte |
|-----------|---------|
| PreToolUse hooks | Nao suportado |
| PostToolUse hooks | Nao suportado |
| Instrucoes de sessao (AGENTS.md) | Suportado via system prompt |
| Skills via config.toml | Suportado |

## Workaround recomendado

Como o Codex nao suporta hooks automaticos, use as seguintes alternativas para manter compliance:

1. **Variavel de ambiente pre-sessao:** Configurar `GOVERNANCE_PRELOAD_CONFIRMED=1` antes de iniciar uma sessao Codex confirma explicitamente que o contrato de carga base sera seguido.

2. **Hook pre-commit no git:** Adicionar ao `.git/hooks/pre-commit` uma chamada a `bash .codex/hooks/validate-preload.sh` para capturar edicoes indevidas em arquivos de governanca antes do commit.

   ```bash
   # .git/hooks/pre-commit
   git diff --cached --name-only | while read file; do
     bash .codex/hooks/validate-preload.sh "$file" || exit 1
   done
   ```

3. **Instrucao explicita no AGENTS.md:** O Codex le `AGENTS.md` como system prompt — o contrato de carga base esta documentado nele e o modelo e instruido a segui-lo antes de editar codigo.

## Gap registrado

Codex nao oferece enforcement automatico de governanca. A compliance depende inteiramente do modelo seguir as instrucoes procedurais do `AGENTS.md` e das skills carregadas. Este gap esta documentado para rastreabilidade.
