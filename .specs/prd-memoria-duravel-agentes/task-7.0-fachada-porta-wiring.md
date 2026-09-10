# Tarefa 7.0: Fachada, porta de memória e wiring por configuração

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

**Pico de risco do plano.** Única tarefa que altera simultaneamente sete arquivos existentes consumidos por outros pacotes. Executada com a rede de 6.0 já montada.

Implementa a fachada como único ponto de contato do runtime com o subsistema, a porta estreita declarada no pacote consumidor, e o wiring por resolução de configuração. Também **decide aqui** a reordenação de `Run()` — não em 8.0 — para não reabrir o arquivo que esta tarefa acabou de estabilizar.

<requirements>
- RF-28: opt-in por flag em `task-loop` e por chave na cascata; zero-value preserva comportamento.
- RF-08, RF-09: captura ao fim da sessão em qualquer condição de saída, de fonte dupla, com a parte estruturada independente do agente.
- RF-10: nenhuma chamada de LLM nem acesso de rede no caminho default.
- RF-13, RF-18: compactação executada pela fachada; orçamento com cotas.
- RF-20: p95 abaixo de 200 ms para 1.000 páginas.
- RF-21: ausência de memória não falha e não emite bloco vazio.
- RF-24: promoção só por marcação explícita.
- RF-25: paridade entre as quatro CLIs.
- **A fachada não contém decisão de domínio** — apenas ordem de chamada, precedência de invariantes e tradução de erro. É o critério de contenção da MD-001.
- Precedência aplicada em um único lugar: segredo, depois não perda, depois dono único de bastão, depois orçamento.
</requirements>

## Subtarefas

- [ ] 7.1 Implementar a fachada orquestrando os colaboradores na ordem do workflow do modelo de domínio.
- [ ] 7.2 Declarar a porta no pacote consumidor com `var _ MemoryPort = (*durable.Facade)(nil)`.
- [ ] 7.3 Adicionar a chave de ativação à cascata nos **dois** pontos: `mergeInto` em `internal/config/resolver.go` e `optionsToConfigOverrides` em `internal/taskloop/runtimeconfig.go`.
- [ ] 7.4 Adicionar a flag em `cmd/ai_spec_harness/task_loop.go`, com detecção de override explícito via `Changed`.
- [ ] 7.5 Propagar pela cadeia: `Options` em `internal/taskloop/taskloop.go`, option de invoker em `internal/taskloop/acpinvoker.go`, campo em `internal/runtime/types.go`.
- [ ] 7.6 Ligar a fachada ao consumidor por resolução de configuração, preservando o caminho anterior quando desativada.
- [ ] 7.7 **Reordenar `Run()`**: mover `dispatchSessionPostEnd` para antes de `persistSummary` em `internal/runtime/runner.go:209` e `:213`.
- [ ] 7.8 Atualizar `docs/cli-schema.json` com a flag nova.
- [ ] 7.9 Declarar `MemoryPort` em `mockery.yml`, rodar `make mocks`, confirmar `make check-mocks`.
- [ ] 7.10 Atualizar `docs/config-hierarchy.md`, `docs/task-loop-reference.md`, `CLAUDE.md` e o índice de ADRs do `AGENTS.md`.

## Detalhes de Implementação

Ver techspec.md, seções "Interfaces Chave", "Ordem de Build" (fatia T7) e "Dependência não declarada anteriormente: reordenação de `Run()`". Ver MD-001 integralmente e MD-004, seção Decisão.

## Critérios de Sucesso

- **O golden de 6.0 passa sem nenhuma alteração de expectativa.** É este critério que converte o retrato em gate.
- A suíte existente permanece verde sem alteração de expectativa.
- Nenhum colaborador do subsistema é importado pelo pacote consumidor.
- Existe teste que falha se a sanitização for executada depois da persistência.
- A chave propaga nos dois pontos da cascata, provado por teste da flag ponta a ponta e por teste por camada.
- `cmd/ai_spec_harness/cli_contract_test.go` passa nas duas direções.
- Com a feature desativada, ativação e desativação aparecem em log de forma explícita.
- `make check-mocks` verde.
- Benchmark preliminar confirma p95 abaixo de 200 ms para 1.000 páginas.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

- `design-patterns-mandatory` — esta é a tarefa que aplica o padrão Facade decidido e validado em `pattern-decisions/memoria-duravel-fachada/`; o critério de contenção (fachada sem decisão de domínio) e o gatilho de rollback precisam ser verificados durante a implementação.

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/runtime/runner.go` — wiring e reordenação de `Run()` (linhas 149, 209, 213)
- `internal/runtime/types.go` — campo no `Job`
- `internal/config/runtime.go` e `internal/config/resolver.go` — cascata, primeiro ponto
- `internal/taskloop/runtimeconfig.go` — cascata, **segundo ponto**
- `internal/taskloop/taskloop.go` e `internal/taskloop/acpinvoker.go` — cadeia de propagação
- `cmd/ai_spec_harness/task_loop.go` — flag
- `docs/cli-schema.json` e `cmd/ai_spec_harness/cli_contract_test.go` — gate bidirecional
- `mockery.yml`
- `pattern-decisions/memoria-duravel-fachada/implementation.md` — plano de mudança do padrão
