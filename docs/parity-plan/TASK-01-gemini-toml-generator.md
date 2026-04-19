# TASK-01: Geração de comandos Gemini TOML

## Contexto

O script remoto `scripts/generate-gemini-commands.sh` itera todas as skills em `.agents/skills/`,
lê o frontmatter YAML de cada `SKILL.md`, e gera um arquivo `.gemini/commands/<skill>.toml` com
um prompt que enforce o contrato de carga base (AGENTS.md, agent-governance, skill de linguagem).

No CLI Go, `internal/adapters/adapters.go` implementa apenas `GenerateClaude`. Não existe
equivalente para Gemini.

## Objetivo

Implementar `GenerateGemini` em `internal/adapters/` para gerar `.gemini/commands/*.toml`
a partir das skills instaladas no projeto-alvo.

## Subtarefas

- [x] **1.1** Estudar o formato TOML gerado pelo bash
  - Ler `scripts/generate-gemini-commands.sh` do remoto
  - Documentar a estrutura do TOML: campos `description`, `prompt`, etc.
  - Identificar regras de skip (ex: `agent-governance` é internal, não gera TOML)

- [x] **1.2** Definir template Go para o TOML Gemini
  - Criar constante ou template em `internal/adapters/` com o formato TOML
  - Incluir placeholders: `{{.SkillName}}`, `{{.Description}}`, `{{.Prompt}}`

- [x] **1.3** Implementar `GenerateGemini` na struct `Generator`
  - Iterar skills instaladas em `.agents/skills/`
  - Ler frontmatter de cada `SKILL.md` via `skills.ParseFrontmatter`
  - Detectar assets (ex: `assets/*.md`) para incluir no prompt
  - Detectar skills que usam review (execute-task, refactor) para append de instruções
  - Escrever `.gemini/commands/<skill>.toml` no projeto-alvo
  - Pular `agent-governance` (internal)

- [x] **1.4** Integrar no fluxo de install
  - Em `internal/install/`, chamar `GenerateGemini` quando `INSTALL_GEMINI` estiver ativo
  - Garantir que `--dry-run` apenas loga sem criar arquivos

- [x] **1.5** Integrar no fluxo de upgrade
  - Em `internal/upgrade/`, re-gerar TOMLs Gemini quando skills forem atualizadas

- [x] **1.6** Integrar no fluxo de uninstall
  - Em `internal/uninstall/`, remover `.gemini/commands/*.toml` quando Gemini for desinstalado

- [x] **1.7** Testes unitários
  - Testar geração de TOML para skill processual (ex: `bugfix`)
  - Testar geração para skill de linguagem (ex: `go-implementation`)
  - Testar skip de `agent-governance`
  - Testar skill com assets vs. sem assets
  - Testar dry-run (nenhum arquivo criado)

- [x] **1.8** Teste de integração
  - Adicionar cenário no `e2e_test.go`: install com `--tools gemini` e verificar conteúdo dos TOMLs

## Arquivos afetados

- `internal/adapters/adapters.go` (novo método `GenerateGemini`)
- `internal/adapters/adapters_test.go` (novos testes)
- `internal/install/install.go` (chamada ao gerador)
- `internal/upgrade/upgrade.go` (re-geração)
- `internal/uninstall/uninstall.go` (limpeza)
- `internal/integration/e2e_test.go` (cenário Gemini)

## Critério de conclusão

1. `ai-spec install --tools gemini --source <path> <target>` gera `.gemini/commands/*.toml`
2. Conteúdo dos TOMLs equivalente ao gerado pelo bash (mesma estrutura, mesmo contrato de carga)
3. `go test ./internal/adapters/... ./internal/install/... ./internal/integration/...` passa

## Status: CONCLUÍDO

Implementado em 2026-04-19. Todos os critérios de conclusão validados:
- `GenerateGemini` gera TOMLs com `description` + `prompt` contendo referência ao SKILL.md e `{{args}}`
- Detecta `assets/*.md` e inclui linhas `Carregue` no prompt quando presentes
- Detecta skills de review loop (`execute-task`, `refactor`) e appenda instrução de validação
- Integrado em install (com dry-run), upgrade e uninstall
- `go test ./internal/adapters/... ./internal/install/... -tags integration ./internal/integration/...` passa
