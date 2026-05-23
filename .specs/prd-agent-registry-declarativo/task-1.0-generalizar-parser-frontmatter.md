# Tarefa 1.0: Generalizar parser de frontmatter (ParseFrontmatterFields)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Extrair a lógica de parsing textual de frontmatter YAML hoje hardcoded em `internal/skills/frontmatter.go` para uma função genérica reutilizável `ParseFrontmatterFields(content []byte) map[string]string`. `ParseFrontmatter` (do package skills) torna-se um wrapper fino sobre essa função, garantindo zero regressão para consumidores existentes. Essa extração é precondição para o package `internal/agents/`, conforme R-01 do techspec.

<requirements>
- A nova função deve preservar todo o comportamento atual do parser de skills (semver, depends_on, triggers, max_depth, etc.).
- A suíte completa de `internal/skills/` deve passar sem alterações nos testes existentes.
- Suportar campos nested simples (ex.: `runtime.ide`) via dot-notation no mapa retornado, OU via parsing de bloco indentado — escolha documentada no commit.
- Função exportada apenas se necessária por `internal/agents/`; caso contrário, manter privada e expor via funções de mais alto nível.
</requirements>

## Subtarefas

- [ ] 1.1 Mapear todos os consumidores atuais de `ParseFrontmatter` em `internal/skills/` e em qualquer outro package.
- [ ] 1.2 Extrair a função genérica preservando assinatura atual de `ParseFrontmatter` como wrapper.
- [ ] 1.3 Adicionar suporte a campos nested necessários ao schema de `Agent` (`runtime.ide`, `runtime.model`, `runtime.reasoning_effort`, `runtime.access_mode`).
- [ ] 1.4 Rodar `go test ./internal/skills/...` e confirmar zero regressão.
- [ ] 1.5 Adicionar testes unitários novos para `ParseFrontmatterFields` cobrindo campos nested e ausência de frontmatter.

## Detalhes de Implementação

Ver techspec, seção **Design de Implementação → Interfaces Chave** (bloco `internal/skills/frontmatter.go — REFATORAÇÃO`) e seção **Sequenciamento de Desenvolvimento** passo 1.

Referência de código atual: `internal/skills/frontmatter.go:23-67`.

## Critérios de Sucesso

- `go test ./internal/skills/...` continua verde após o refator.
- `ParseFrontmatterFields` exposto (público ou interno ao package) é consumível por `internal/agents/` na tarefa 2.0.
- Nenhuma assinatura pública existente de `ParseFrontmatter`/`ParseFrontmatterName`/`ValidateFrontmatter` é alterada.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários novos para `ParseFrontmatterFields` (campos nested, frontmatter ausente, frontmatter vazio).
- [ ] Suíte completa `go test ./internal/skills/...` permanece verde (regressão).
- [ ] Cobertura do caso T-22 do techspec (frontmatter refactor — skills atuais ainda parseiam corretamente).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/skills/frontmatter.go` (modificado — extração da função genérica)
- `internal/skills/frontmatter_test.go` (modificado/adicionado — testes novos + regressão)
- `internal/skills/schema.go` (eventualmente modificado — consumir nova função se aplicável)
