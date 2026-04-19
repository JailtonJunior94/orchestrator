# TASK-02: Geração de agentes Copilot

## Contexto

O script remoto `scripts/generate-adapters.sh` gera wrappers tanto para Claude Code
(`.claude/agents/*.md`) quanto para GitHub Copilot (`.github/agents/*.agent.md`).

No CLI Go, `internal/adapters/adapters.go` implementa `GenerateClaude` com um `skillRegistry`
que mapeia 8 skills processuais para nomes de arquivo e instruções. Não existe `GenerateCopilot`.

O formato Copilot usa sufixo `.agent.md` e estrutura ligeiramente diferente do Claude.

## Objetivo

Implementar `GenerateCopilot` em `internal/adapters/` para gerar `.github/agents/*.agent.md`.

## Subtarefas

- [x] **2.1** Estudar o formato dos agentes Copilot no remoto
  - Ler `.github/agents/*.agent.md` do remoto via `gh api`
  - Documentar a estrutura: headings, referências a skills, instrução de carga
  - Comparar com o formato Claude para identificar diferenças

- [x] **2.2** Estender `skillRegistry` com dados Copilot
  - Adicionar campo `CopilotFileName` ao registro (ex: `bugfix.agent.md`)
  - Verificar se as instruções diferem entre Claude e Copilot

- [x] **2.3** Implementar `GenerateCopilot` na struct `Generator`
  - Iterar `ProcessualSkills` do registry
  - Para cada skill, gerar `.github/agents/<name>.agent.md`
  - Usar template com formato Copilot (heading, instrução de carga, referência ao SKILL.md)

- [x] **2.4** Integrar no fluxo de install
  - Chamar `GenerateCopilot` quando `INSTALL_COPILOT` estiver ativo
  - Criar diretório `.github/agents/` se não existir
  - Respeitar `--dry-run`

- [x] **2.5** Integrar no fluxo de upgrade
  - Re-gerar agentes Copilot quando skills forem atualizadas

- [x] **2.6** Integrar no fluxo de uninstall
  - Remover `.github/agents/*.agent.md` na desinstalação do Copilot

- [x] **2.7** Testes unitários
  - Testar geração dos 8 agentes
  - Verificar sufixo `.agent.md`
  - Verificar conteúdo do template (referência correta ao SKILL.md)
  - Testar dry-run

- [x] **2.8** Teste de integração
  - Adicionar cenário: install com `--tools copilot` e verificar que `.github/agents/` existe com 8 arquivos

## Arquivos afetados

- `internal/adapters/adapters.go` (novo método `GenerateCopilot`, extensão de `skillRegistry`)
- `internal/adapters/adapters_test.go` (novos testes)
- `internal/install/install.go` (chamada ao gerador)
- `internal/upgrade/upgrade.go` (re-geração)
- `internal/uninstall/uninstall.go` (limpeza)
- `internal/integration/e2e_test.go` (cenário Copilot)

## Critério de conclusão

1. `ai-spec install --tools copilot --source <path> <target>` gera 8 arquivos `.github/agents/*.agent.md`
2. Formato equivalente ao gerado pelo bash
3. `go test ./internal/adapters/... ./internal/install/...` passa
