# Workaround de Preload de Governanca — Gemini CLI

> Carregar este arquivo apenas se o hook `.gemini/hooks/validate-preload.sh` falhar ou nao estiver presente.

## Contexto

O Gemini CLI nao suporta hooks automaticos (PreToolUse/PostToolUse) nativos como o Claude Code. Os scripts em `.gemini/hooks/` sao utilitarios manuais, nao acionados automaticamente.

## Scripts disponíveis

| Script | Equivalente Claude | Uso |
|--------|-------------------|-----|
| `.gemini/hooks/validate-preload.sh` | `.claude/hooks/validate-preload.sh` | Lembrete de carga base antes de editar codigo |
| `.gemini/hooks/validate-governance.sh` | `.claude/hooks/validate-governance.sh` | Aviso ao modificar arquivos de governanca |

## Como invocar manualmente

```bash
# Verificar contrato de carga antes de editar
bash .gemini/hooks/validate-preload.sh path/to/file.go

# Verificar apos editar arquivo de governanca
bash .gemini/hooks/validate-governance.sh .agents/skills/review/SKILL.md
```

## Invocacao via flag --hook (se suportado pela versao do Gemini CLI)

```bash
gemini --hook "bash .gemini/hooks/validate-preload.sh {file}" \
       --hook "bash .gemini/hooks/validate-governance.sh {file}"
```

## Variaveis de controle

| Variavel | Valores | Efeito |
|----------|---------|--------|
| `GEMINI_PRELOAD_MODE` | `fail` (default) / `warn` | Controla se validate-preload bloqueia ou apenas avisa |
| `GEMINI_GOVERNANCE_MODE` | `fail` (default) / `warn` | Controla se validate-governance bloqueia ou apenas avisa |
| `GOVERNANCE_PRELOAD_CONFIRMED` | `0` (default) / `1` | Bypass do bloqueio de preload quando contrato ja foi confirmado |

## Limitacao conhecida

Sem hooks automaticos, a compliance depende de seguir as instrucoes procedurais manualmente. Use `GOVERNANCE_PRELOAD_CONFIRMED=1` em sessoes longas onde o contrato ja foi confirmado no inicio.
