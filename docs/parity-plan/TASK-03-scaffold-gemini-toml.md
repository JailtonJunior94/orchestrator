# TASK-03: Scaffold com TOML Gemini

## Contexto

O script remoto `scripts/scaffold-lang-skill.sh` cria:
1. `.agents/skills/<lang>-implementation/SKILL.md` com procedimento padrão
2. 13 reference stubs em `references/`
3. `.gemini/commands/<lang>-implementation.toml`

O CLI Go `internal/scaffold/scaffold.go` implementa os itens 1 e 2 (com 14 reference stubs),
mas não gera o TOML Gemini.

## Objetivo

Estender `internal/scaffold/` para gerar o TOML Gemini correspondente à skill criada.

## Dependência

- **TASK-01** — reutilizar o template TOML e/ou a lógica de `GenerateGemini` para gerar
  um único TOML para a skill scaffoldada.

## Subtarefas

- [x] **3.1** Extrair template TOML reutilizável
  - Se TASK-01 criou template inline em `GenerateGemini`, extrair para constante/função pública
  - Garantir que o template aceite skill name, description e prompt como parâmetros

- [x] **3.2** Adicionar geração de TOML ao scaffold
  - Após criar `SKILL.md` e `references/`, gerar `.gemini/commands/<lang>-implementation.toml`
  - Criar diretório `.gemini/commands/` se não existir
  - Usar descrição padrão: "Implementação em <Lang> seguindo as regras de governança"

- [x] **3.3** Atualizar mensagem de pós-scaffold
  - O scaffold bash imprime follow-up steps manuais (parity tests, codex config, install.sh, AGENTS.md)
  - Manter prints equivalentes no Go

- [x] **3.4** Testes unitários
  - Testar que `ai-spec scaffold rust` cria 3 artefatos: SKILL.md, references/, TOML
  - Verificar conteúdo do TOML gerado
  - Testar com filesystem fake

- [x] **3.5** Testar idempotência
  - Rodar scaffold duas vezes para mesma linguagem deve sobrescrever sem erro

## Arquivos afetados

- `internal/scaffold/scaffold.go` (adicionar geração TOML)
- `internal/scaffold/scaffold_test.go` (novos testes)
- `internal/adapters/adapters.go` (expor template TOML se necessário)

## Critério de conclusão

1. `ai-spec scaffold rust --root <path>` cria `.gemini/commands/rust-implementation.toml`
2. TOML contém prompt com contrato de carga base
3. `go test ./internal/scaffold/...` passa

## Status: CONCLUÍDO

Implementado em 2026-04-19. Todos os critérios validados:
- scaffold.go já gerava TOML inline; guard de `IsDir` removido para suportar idempotência
- `TestScaffold_ThreeArtifacts`: verifica criação de SKILL.md, references/ e TOML para `rust`
- `TestScaffold_Idempotent`: executa duas vezes sem erro e confirma TOML presente
- `go test ./internal/scaffold/...` passa (4/4)
