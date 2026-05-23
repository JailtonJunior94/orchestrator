# Tarefa 2.0: Domínio e JSON Schema do package internal/agents/

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar o package `internal/agents/` com os value objects (`ResolvedAgent`, `Metadata`, `RuntimeDefaults`, `Scope`), o JSON Schema embarcado de `AGENT.md` (`agent-frontmatter.schema.json`) e a função de validação. Espelha intencionalmente `internal/core/agents/` do Compozy mas se acopla ao catálogo `specs` existente.

<requirements>
- Value objects imutáveis com construtores validantes (R-DDD-001).
- JSON Schema embarcado via `go:embed`, espelhando o padrão de `internal/skills/schema.go:13-33`.
- Schema valida campos obrigatórios (`name`, `description`, `version`) e opcionais (`runtime.ide`, `runtime.model`, `runtime.reasoning_effort`, `runtime.access_mode`) — RF-05.
- `runtime.ide` aceita enum `claude|codex|gemini|copilot` (RF-08).
- `version` valida SemVer (RF-07).
- `name` valida regex `^[a-z0-9][a-z0-9-]*$`.
- Erros sentinela compostos com `fmt.Errorf("%w: ...")` para `errors.Is`.
</requirements>

## Subtarefas

- [ ] 2.1 Criar `internal/agents/agent.go` com `ResolvedAgent`, `Metadata`, `RuntimeDefaults`, `Scope` (enum `ScopeGlobal|ScopeWorkspace`).
- [ ] 2.2 Criar `internal/agents/agent-frontmatter.schema.json` (JSON Schema 2020-12) conforme bloco em techspec → Modelos de Dados.
- [ ] 2.3 Criar `internal/agents/schema.go` com `go:embed` e função `ValidateAgentFrontmatter(content []byte, dirName string) (ResolvedAgent, error)`.
- [ ] 2.4 Adicionar erros sentinela: `ErrFrontmatterInvalid`, `ErrNameDirMismatch`, `ErrVersionInvalid`, `ErrIDEUnsupported`.
- [ ] 2.5 Consumir `ParseFrontmatterFields` (da tarefa 1.0) para extrair campos antes da validação JSON Schema.

## Detalhes de Implementação

Ver techspec, seção **Design de Implementação → Interfaces Chave** (bloco `internal/agents/agent.go`) e **Modelos de Dados** (bloco JSON Schema completo).

Padrão de schema a espelhar: `internal/skills/schema.go:13-33` (init + compilação + validação).

Mapping `runtime.ide → specs.<Tool>()`: na tarefa 6.0; aqui apenas armazenar o valor string validado.

## Critérios de Sucesso

- `internal/agents/` compila isoladamente sem dependências de `taskloop` ou `runtime/specs`.
- `ValidateAgentFrontmatter` retorna erro acionável com campo problemático para os casos T-05, T-06, T-07, T-08, T-09 do techspec.
- `ResolvedAgent.Name` coincide com o diretório pai quando ambos fornecidos (RF-06).
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-05: frontmatter sem `description` → erro citando campo obrigatório.
- [ ] T-06: `version` não-SemVer → erro citando campo `version`.
- [ ] T-07: `runtime.ide` fora do enum → erro citando opções válidas.
- [ ] T-08: `runtime.reasoning_effort` inválido → erro citando enum.
- [ ] T-09: `name` no frontmatter ≠ nome do diretório → erro RF-06.
- [ ] Caso positivo: frontmatter válido produz `ResolvedAgent` com todos os campos preenchidos.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/agents/agent.go` (novo)
- `internal/agents/schema.go` (novo)
- `internal/agents/agent-frontmatter.schema.json` (novo)
- `internal/agents/errors.go` (novo — erros sentinela)
- `internal/agents/schema_test.go` (novo)
- `internal/agents/agent_test.go` (novo)
