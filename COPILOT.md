# ai-spec-harness — Governança para GitHub Copilot

Use `AGENTS.md` como fonte canonica das regras deste repositorio. Stack, comandos, convencoes, estrutura, CI e padroes estao documentados em `AGENTS.md` — nao duplicados aqui.

## Governança

Regras transversais, precedencia e restrições operacionais estao definidas em:

- `AGENTS.md` — instrucao de sessao e contrato de carga base
- `.agents/skills/agent-governance/SKILL.md` — governanca para analise e alteracao de codigo
- `.claude/rules/governance.md` — precedencia e politica de evidencia

Regras essenciais:
1. Ler `AGENTS.md` e `.agents/skills/agent-governance/SKILL.md` antes de editar codigo.
2. Toda alteracao deve ser justificavel pelo PRD, por regra explicita ou por necessidade tecnica demonstravel.
3. Preservar estilo, arquitetura e fronteiras existentes antes de propor mudancas.
4. Validar mudancas com comandos proporcionais ao risco.
5. Nao inventar contexto ausente.
6. Nao executar acoes destrutivas sem pedido explicito.

## Mecanismo de Carregamento de Contexto

O GitHub Copilot possui **dois modos distintos** com mecanismos diferentes:

### Copilot Chat (VS Code / GitHub.com) — Suporte Nativo

O arquivo `.github/copilot-instructions.md` e carregado automaticamente como contexto de repositorio pelo GitHub Copilot Chat na extensao VS Code (v1.143+) e no GitHub.com. Este e o mecanismo nativo para fornecer instrucoes de repositorio ao Copilot.

- **Arquivo de contexto automatico:** `.github/copilot-instructions.md` (ja existe neste repositorio)
- **Escopo:** instruido automaticamente em todas as conversas do Copilot Chat no repositorio
- **Este arquivo (`COPILOT.md`):** nao e carregado automaticamente — serve como documentacao de governanca para leitura humana e referencia via `#file:COPILOT.md` no chat

### gh copilot CLI (`gh copilot suggest` / `gh copilot explain`) — Sem Suporte Nativo

O `gh copilot` CLI **nao le** `.github/copilot-instructions.md` nem qualquer arquivo de contexto automaticamente. Cada invocacao e stateless.

**Workaround recomendado para o CLI:**
```bash
gh copilot suggest "$(cat .github/copilot-instructions.md)\n\nTarefa: <descricao>"
cat internal/install/install.go | gh copilot explain
```

## Orientacoes Especificas para Copilot

1. Copilot Chat carrega `.github/copilot-instructions.md` automaticamente.
2. `gh copilot` CLI e stateless — contexto deve ser injetado manualmente.
3. Enforcement depende do modelo seguir instrucoes — nao ha bloqueio automatico.

## Limitações Conhecidas

| Capacidade | Claude Code | Gemini CLI | Codex | Copilot Chat | gh copilot CLI |
|---|---|---|---|---|---|
| Arquivo de contexto automatico | CLAUDE.md | GEMINI.md | AGENTS.md | `.github/copilot-instructions.md` | **Nao suportado** |
| Hooks de pre/pos execucao | Sim | Sim | Sim | Nao | Nao |
| Sistema de skills/comandos | Sim | Sim | Sim | Nao | Nao |
| Estado de sessao persistente | Sim | Sim | Sim | Sim (por sessao) | **Nao — stateless** |
| Carregamento sob demanda de refs | Sim | Sim | Sim | Manual (`#file:`) | **Nao** |
