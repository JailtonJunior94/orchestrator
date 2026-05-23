# Tarefa 3.0: Discovery de AGENT.md em escopo global + workspace

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar descoberta recursiva de `AGENT.md` em dois escopos: global (`<HOME>/.ai-harness/agents/<name>/AGENT.md`) e workspace (`<workdir>/.ai-harness/agents/<name>/AGENT.md`). Em colisão de nome, workspace prevalece; o agente global homônimo é marcado como `shadowed` e logado em nível `info`.

<requirements>
- Operar sobre `fs.FileSystem` (ADR-002) — sem chamar `os.UserHomeDir()` diretamente. Construtor recebe `home` e `workdir` como parâmetros explícitos.
- Ignorar subdiretórios sem `AGENT.md` válido sem retornar erro.
- Falha de leitura em um agente não interrompe a descoberta dos demais (retornar erro agregado ou registrar e continuar — decisão documentada).
- Resolução de paths via `filepath.Join` (R-SEC-001, sem traversal).
</requirements>

## Subtarefas

- [ ] 3.1 Criar `internal/agents/discovery.go` com função `discoverAgents(fsys fs.FileSystem, scope Scope, root string) ([]ResolvedAgent, error)`.
- [ ] 3.2 Combinar global + workspace em uma função `mergeWithShadowing([]ResolvedAgent, []ResolvedAgent) (merged, shadowed []ResolvedAgent)`.
- [ ] 3.3 Aplicar regra RF-06 (name do frontmatter = nome do diretório pai) na descoberta.
- [ ] 3.4 Logar shadowing em nível `info` (formato padrão do projeto).
- [ ] 3.5 Adicionar testes T-01 a T-04 e T-09 usando `internal/fs/fake.go` (FakeFileSystem).

## Detalhes de Implementação

Ver techspec, seção **Arquitetura → Visão Geral dos Componentes** e **Modelos de Dados → Estrutura de descoberta**.

Padrões de consumo de FakeFileSystem: ver `internal/fs/fake_test.go` para exemplos.

## Critérios de Sucesso

- Casos T-01 (vazio), T-02 (só global), T-03 (só workspace), T-04 (colisão), T-09 (name/dir mismatch) passam.
- Tempo de descoberta + resolução < 50ms p95 em workspace com 20 agentes (verificado em teste de performance opcional).
- Shadowing produz log informativo, não erro.
- Diff zero em `internal/runtime/persistence/*` e `internal/runtime/watchdog.go`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-01: nenhum agente em nenhum escopo → `Discover()` retorna `[]`, sem erro.
- [ ] T-02: apenas global → lista apenas global, `Scope=ScopeGlobal`.
- [ ] T-03: apenas workspace → lista apenas workspace, `Scope=ScopeWorkspace`.
- [ ] T-04: colisão global+workspace → workspace prevalece; global em campo `shadowed`.
- [ ] T-09: `name` ≠ dirname → erro RF-06 propagado.
- [ ] Frontmatter inválido em um agente: descoberta dos demais continua.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/agents/discovery.go` (novo)
- `internal/agents/discovery_test.go` (novo)
- `internal/fs/fake.go` (referência; sem modificação)
