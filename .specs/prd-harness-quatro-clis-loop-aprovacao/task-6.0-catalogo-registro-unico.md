# Tarefa 6.0: Catálogo de Agentes como registro único

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Os **onze** pontos independentes que hoje enumeram agentes colapsam em um **registro único**. Cada
`Agente` passa a ser Value Object imutável, portando identidade, spec de invocação, enforcement por
ponto canônico, sinais de detecção, política de ambiente, referência de ADR e orçamentos. Todos os call
sites derivam do registro.

Esta tarefa é **estrutural, não de conteúdo**: ela cria o registro e migra os call sites com o conjunto
de agentes atual intacto. A entrada do novo agente é a tarefa 7.0 e a remoção do agente descontinuado é
a 10.0 — fundir qualquer uma delas aqui produziria um commit que altera o catálogo **e** seu conteúdo ao
mesmo tempo, impedindo bissecção exatamente no ponto da armadilha de regressão mais cara.

Duas armadilhas concretas são desarmadas aqui, ambas confirmadas por leitura do código:

1. **Duas ordens canônicas coexistem** — `internal/skills/skills.go:13` (`AllTools`) e a ordem fixa de
   exibição em `cmd/ai_spec_harness/verify.go:154-155` — produzindo saída não-determinística entre
   comandos.
2. **Dois catálogos ACP espelhados** — `cmd/ai_spec_harness/task_loop.go:28-32` e
   `internal/taskloop/taskloop.go:929-934` — com **fallback silencioso** em `taskloop.go:938-940`, que
   hoje executa o **job inteiro no agente errado sem produzir erro** quando a entrada falta em apenas um
   dos dois.

<requirements>
- **RF-07:** As duas ordens canônicas coexistentes (V-29) são unificadas em **uma só**, e os dois
  catálogos ACP espelhados (V-31) passam a ter **fonte única** ou verificação de sincronia que **falha**
  na divergência. O fallback silencioso atual é **removido** e substituído por erro explícito e tipado.
- **RF-22:** Os quatro agentes cobrem os **três pontos canônicos** — pré-ferramenta, pós-ferramenta e
  encerramento de sessão — cada um pelo mecanismo nativo do seu CLI, todos apontando para os **mesmos
  scripts canônicos**. Nenhuma lógica de gate é reimplementada por agente.
- **RF-26:** A regra de **não executar binários permanece intacta na detecção**. A verificação de
  pré-condições — que exige comunicação com um CLI — pertence ao **comando de diagnóstico**, invocado
  explicitamente pelo usuário. As duas responsabilidades ficam em superfícies separadas.
- Cada `Agente` é **Value Object imutável**: sem setters, comparável por valor, zero-value inválido.
- `PontoCanonico` é **conjunto fechado** (pré-ferramenta, pós-ferramenta, encerramento) e o construtor de
  `Enforcement` **RECUSA cobertura incompleta** — a invariante "um agente só integra o catálogo se cobrir
  os três pontos" passa a ser verificada na construção do registro, não em revisão humana.
- `PreCondicaoDeEnforcement` é **conceito de primeira classe**, carregando o remédio acionável embutido.
  É o que permite distinguir "gate ausente" de "gate presente porém inerte".
- Nenhuma mudança de comportamento observável para os agentes existentes (RF-62, O-06).
</requirements>

## Subtarefas

- [ ] 6.1 Modelar os tipos do registro: `Agente` (Value Object imutável), `IdentidadeDoAgente`,
      `PontoCanonico` (conjunto fechado, `iota+1` com zero-value reservado) e `Enforcement` com
      construtor validante.
- [ ] 6.2 Implementar o construtor de `Enforcement` de modo que **cobertura incompleta dos três pontos
      canônicos seja erro de construção**, não aviso — a invariante do modelo deixa de depender de
      revisão humana.
- [ ] 6.3 Modelar `PreCondicaoDeEnforcement` com o remédio acionável embutido e os estados que a
      verificação passa a distinguir (`current`, `inert`, `unknown`), conforme
      `adr-005-precondicoes-de-enforcement.md`.
- [ ] 6.4 Criar o registro único, com resolução por inicialização única em variável de pacote (ver
      trade-off registrado em `adr-002` §Trade-offs e Custos).
- [ ] 6.5 **Unificar a ordem canônica**: `internal/skills/skills.go:13` vira fonte única derivada do
      registro; `cmd/ai_spec_harness/verify.go:154-155` passa a consumi-la; `ParseTool`
      (`internal/skills/skills.go:15-20`) deriva da mesma lista, eliminando a terceira enumeração
      implícita no `switch`.
- [ ] 6.6 Adicionar **gate de ordem canônica** — verde por construção após 6.5 — que falha se qualquer
      ponto do código reintroduzir uma ordem própria.
- [ ] 6.7 **Unificar os dois catálogos ACP** (`cmd/ai_spec_harness/task_loop.go:28-32` e
      `internal/taskloop/taskloop.go:929-934`) em fonte única derivada do registro.
- [ ] 6.8 **Substituir o fallback silencioso** de `internal/taskloop/taskloop.go:938-940` por **erro
      explícito e tipado**: tool não presente no catálogo passa a falhar, e não a rodar como Claude.
- [ ] 6.9 Adicionar **gate de sincronia dos catálogos** que falha na divergência, para o caso de a fonte
      única ser reaberta no futuro.
- [ ] 6.10 Migrar os demais call sites que enumeram agentes ao registro: `specForTool` do instalador,
      `allEntries` da detecção, `resolveAgentSpec` do servidor MCP, `mapIDEToSpec` do perfil,
      `ParseDriverID`, mapa de ADRs do probe e mapas de orçamento.
- [ ] 6.11 **Separar detecção de diagnóstico**: a detecção (`internal/detect/agent.go`) permanece
      **PROIBIDA de executar binários** — apenas binário no `PATH`, diretório de configuração e sinal de
      projeto. Toda verificação que exige comunicação com CLI é movida para a superfície de diagnóstico.
- [ ] 6.12 Adicionar gate que varre a árvore em busca de literais de lista de agentes fora do registro,
      com lista de exceções explícita e justificada.

## Detalhes de Implementação

Ver `techspec.md`:

- §Sequenciamento de Desenvolvimento → **Catálogo de Agentes: um registro, tudo derivado** — a lista dos
  onze pontos de enumeração e os dois tipos novos (`PontoCanonico`, `PreCondicaoDeEnforcement`).
- §Sequenciamento de Desenvolvimento → Ordem de build da Fase 4, etapas **2 a 5** — a ordem interna
  obrigatória: unificar ordem canônica, ligar o gate de ordem, unificar catálogos, ligar o gate de
  sincronia. A etapa 4 é descrita como "a armadilha de regressão mais cara" e precisa estar armada antes
  de qualquer remoção de entrada.
- §Sequenciamento de Desenvolvimento → Fases — **F3a — Catálogo**, dependente de F1.
- §Verificação de pré-condições sem violar a regra de detecção — a tabela que separa o que é verificável
  sem executar binário do que exige RPC, e os estados `inert` e `unknown`.
- §Abordagem de Testes → Testes de Integração, camada 1 (**Matriz obrigatória**) — produto cartesiano
  agentes × pontos canônicos.
- `adr-002-catalogo-de-agentes-registro-unico.md` — decisão, alternativas rejeitadas e o trade-off da
  variável de pacote.
- `adr-005-precondicoes-de-enforcement.md` — pré-condições como conceito de primeira classe.

Não duplicar aqui a spec do OpenCode nem o desenho do plugin: pertencem às tarefas 7.0 e 8.0.

## Critérios de Sucesso

- `go build ./... && go vet ./... && go test ./... -count=1` verde ao final da tarefa.
- Teste de compilação do registro prova que **construir um `Agente` com cobertura incompleta dos três
  pontos canônicos retorna erro** — asserção por ponto faltante.
- Teste prova que o zero-value de `Agente` e de `PontoCanonico` é **inválido** e recusado pelos
  construtores.
- Teste prova imutabilidade do `Agente`: nenhum método com receptor ponteiro que mute campo; comparação
  por valor consistente.
- Existe **uma única** lista ordenada de agentes no código: o gate de ordem canônica falha se um segundo
  literal de ordem for introduzido.
- `ai-spec-harness verify .` e `ai-spec-harness install .` produzem os agentes na **mesma ordem**,
  verificável por diff das duas saídas.
- Teste prova que resolver um tool **ausente do catálogo ACP** retorna erro tipado — e não `Claude()`.
  O teste falha se o fallback de `taskloop.go:938-940` for reintroduzido.
- Gate de sincronia dos catálogos fica **vermelho** quando uma entrada é removida de apenas um dos dois
  pontos (verificado por teste que simula a divergência).
- Matriz obrigatória (unitária): produto cartesiano agentes × três pontos canônicos, com verificação de
  que cada célula declara script canônico e de que **nenhum agente tem cobertura menor que outro**.
- Nenhum script canônico é reimplementado por agente: gate de espelhamento verde.
- Teste prova que `internal/detect/` **não executa nenhum binário**: nenhuma chamada a `exec.Command`
  no pacote de detecção (verificável por `go vet` customizado ou por gate de varredura).
- `git diff` dos golden files de governança gerada **vazio** — esta tarefa não muda conteúdo.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

- `domain-modeling-production` — a tarefa transforma onze enumerações espalhadas em um registro de Value Objects imutáveis com conjunto fechado (`PontoCanonico`), construtor validante que recusa cobertura incompleta e conceito de primeira classe para pré-condição de enforcement, exatamente o desenho de invariantes por tipo que a skill conduz.

## Testes da Tarefa

- [ ] Testes unitários
  - Construtor de `Enforcement`: suíte com tabela cobrindo cobertura completa (aceita) e cada
    combinação de cobertura incompleta (recusa com erro tipado).
  - `PontoCanonico`: conjunto fechado, zero-value reservado, parsing de valor desconhecido recusado.
  - `Agente`: imutabilidade, comparação por valor, zero-value inválido.
  - `PreCondicaoDeEnforcement`: cada pré-condição carrega remédio acionável não-vazio; mapeamento para
    `current` / `inert` / `unknown`.
  - Resolução de spec ACP: tool conhecido resolve; tool desconhecido retorna erro (não fallback).
  - Ordem canônica: única fonte, derivada em todos os consumidores.
- [ ] Testes de integração
  - Matriz obrigatória agentes × pontos canônicos, com verificação do arquivo de registro escrito.
  - `verify` e `install` sobre repositório temporário produzem a mesma ordem de agentes.
  - Gate de sincronia de catálogos: teste que introduz divergência artificial e asserta falha do gate.
  - Não-regressão: fluxos existentes dos agentes atuais produzem saída idêntica à anterior.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/specs/registro.go` (planejado) — registro único de agentes.
- `internal/runtime/specs/enforcement.go` (planejado) — `PontoCanonico`, `Enforcement`,
  `PreCondicaoDeEnforcement`.
- `internal/skills/skills.go:13` — `AllTools`, primeira ordem canônica; vira fonte derivada do registro.
- `internal/skills/skills.go:15-20` — `ParseTool`, terceira enumeração implícita a eliminar.
- `cmd/ai_spec_harness/verify.go:154-155` — segunda ordem canônica ("Ordem fixa: claude, codex, copilot,
  gemini, depois quaisquer outros"), a ser removida em favor da fonte única.
- `cmd/ai_spec_harness/task_loop.go:28-32` — `_runtimeACPCatalog`, primeiro catálogo ACP.
- `internal/taskloop/taskloop.go:929-934` — `_acpSpecCatalog`, catálogo ACP espelhado.
- `internal/taskloop/taskloop.go:938-940` — `resolveACPSpec`, fallback silencioso para `Claude()` a
  substituir por erro explícito.
- `internal/detect/agent.go` — detecção; permanece proibida de executar binários (RF-26).
- `internal/install/install.go` — `specForTool`, a derivar do registro.
- `internal/runtime/probe/probe.go` — mapa de ADRs por agente.
- `internal/metrics/metrics.go:213-215` — `ToolBudgetsLarge`, um dos mapas de orçamento a migrar.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — §Catálogo de Agentes,
  §Verificação de pré-condições, §Ordem de build da Fase 4 (etapas 2–5).
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-002-catalogo-de-agentes-registro-unico.md`
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-005-precondicoes-de-enforcement.md`
