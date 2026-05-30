# Tarefa 6.0: Subagentes Codex/Gemini — validação empírica + agent files

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Garantir isolamento de contexto real cross-tool: validar empiricamente os formatos de subagente de
Codex e Gemini e gerar os agent files faltantes (ou documentar execução inline quando o tool não
suportar). Cobre RF-16.

<requirements>
- Validar formato de subagente Codex (`.codex/agents/*.toml`) e Gemini empiricamente quando o binário existir.
- Gerar agent files faltantes para paridade com `.claude/agents/`/`.github/agents/`, OU documentar execução inline.
- Registrar o resultado da validação (suposição vs verificado) no report de execução.
</requirements>

## Subtarefas

- [ ] 6.1 Inventariar agent files existentes por tool (`.claude/agents/`, `.github/agents/`, `.codex/agents/`, `.gemini/`).
- [ ] 6.2 Validar formato empiricamente (`codex` / `gemini` agent list) quando disponível.
- [ ] 6.3 Gerar agent files faltantes (Codex/Gemini) ou documentar fallback inline.
- [ ] 6.4 Registrar evidência (verificado/inferido) e atualizar `execute-all-tasks/SKILL.md` se necessário.

## Detalhes de Implementação

Ver techspec "Pontos de Integração" e a auditoria (gap de paridade de subagentes). Gemini CLI 2026
não documenta diretório `.gemini/agents/` dedicado — nesse caso, documentar execução via skills/MCP
inline em vez de inventar um formato.

## Critérios de Sucesso

- Inventário de subagentes por tool registrado.
- Agent files faltantes gerados quando o tool suporta; caso contrário, fallback inline documentado.
- Report registra explicitamente se a validação foi empírica ou suposição (sem fingir enforcement).

## Skills Necessárias

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Verificação: agent files novos têm formato válido para o tool alvo.
- [ ] Report contém a classificação epistêmica (verificado/inferido).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.codex/agents/`, `.gemini/`, `.claude/agents/`, `.github/agents/`
- `.agents/skills/execute-all-tasks/SKILL.md`
