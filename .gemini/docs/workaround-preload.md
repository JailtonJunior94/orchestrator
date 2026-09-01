# Workaround de Preload de Governanca — Gemini CLI

> Carregar este arquivo apenas se o hook `.gemini/hooks/validate-preload.sh` falhar ou nao estiver presente.

## Contexto

Este adaptador não declara capacidades estáticas do Gemini CLI. Os scripts em `.gemini/hooks/`
são utilitários locais; antes de qualquer operação que exija isolamento ou escrita concorrente,
consulte `ai-spec runtime-capabilities <raiz-do-worktree>` e siga o resultado fail-closed do CLI.

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

## Invocacao via adaptador instalado

```bash
ai-spec runtime-capabilities <raiz-do-worktree>
```

Use o mecanismo de hook declarado pelo JSON retornado. Se não houver capacidade declarada, execute
os scripts manualmente e não trate isso como prova de isolamento ou escrita concorrente.

## Variaveis de controle

| Variavel | Valores | Efeito |
|----------|---------|--------|
| `GEMINI_PRELOAD_MODE` | `fail` (default) / `warn` | Controla se validate-preload bloqueia ou apenas avisa |
| `GEMINI_GOVERNANCE_MODE` | `fail` (default) / `warn` | Controla se validate-governance bloqueia ou apenas avisa |
| `GOVERNANCE_PRELOAD_CONFIRMED` | `0` (default) / `1` | Bypass do bloqueio de preload quando contrato ja foi confirmado |

## Limitacao conhecida

Instruções e scripts não comprovam isolamento. Use `GOVERNANCE_PRELOAD_CONFIRMED=1` apenas para
o preload já confirmado; nunca como bypass de uma capacidade ausente no CLI.
