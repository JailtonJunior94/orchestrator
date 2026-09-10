# Tarefa 7.0: OpenCode como agente oficial de primeira classe

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

O OpenCode entra no registro criado em 6.0 como agente de primeira classe, via ACP **por subcomando**
(`opencode acp`), não por flag — divergência de forma frente aos outros três agentes, resolvida **sem
caso especial no runner**.

A razão de o desenho não precisar de caso especial é que `FixedArgs` já é prefixo posicional no argv, e o
subcomando é apenas um `FixedArgs` sem hífen. Mas essa invariante é hoje **acidental**: nada no código a
garante. Ela passa a ser travada por **validação de formato** — nenhum item de `FixedArgs` pode aparecer
depois de um argumento iniciado por hífen.

A instalação tem **pegada mínima deliberada**: escreve **apenas** o bloco `permission` no
`opencode.json`. Não copia skills, não cria symlink, e **não** escreve `skills.paths` nem `instructions`
— os dois foram comprovados redundantes (skills de projeto e o arquivo de governança raiz são
auto-carregados, V-09 e V-10), e `skills.paths` seria **ativamente nocivo** por ser **aditivo**,
duplicando a árvore de skills.

**IMPORTANTE (RF-06):** esta tarefa é também o momento em que o OpenCode **assume as entradas nas células
de ocupante único** — a tabela de orçamento de janela grande (`internal/metrics/metrics.go:213-215`) e a
lista de herança comum das regras de normalização (`.agents/normalization-rules.yaml:20-21` e o gêmeo
embarcado em `internal/runtime/events/normalization-rules.yaml:20-21`). Isso precisa acontecer **antes**
da remoção do agente descontinuado (tarefa 10.0): esvaziar essas estruturas muda o comportamento sem que
nenhum teste falhe. Junto entra o **gate de COBERTURA** — não de não-vazio.

<requirements>
- **RF-06:** As estruturas que ficariam vazias com a remoção (V-28) recebem a entrada correspondente ao
  novo agente, **e** ganham gate de **cobertura**: todo agente cuja janela resolvida for **grande**
  precisa de entrada, e **nenhuma chave órfã** pode existir. Gate de não-vazio é insuficiente — aceitaria
  valor de fachada.
- **RF-10:** O OpenCode é declarado runtime ACP nativo com binário `opencode` e **subcomando** `acp`. A
  abstração de spec suporta a forma subcomando **sem caso especial no runner**. O launcher de fallback usa
  gerenciador de pacotes com versão pinada (`npx --yes <pacote oficial>@<versão pinada> acp`).
- **RF-11:** As versões são **constantes pinadas**; `@latest` é **proibido**. Alteração exige registro de
  decisão em auditoria. A versão do SDK ACP permanece sincronizada com o `go.mod` pelo mecanismo vigente.
- **RF-12:** A detecção trata o OpenCode como **1ª classe**, com os mesmos **três sinais** dos demais:
  binário no `PATH`, diretório de configuração do usuário, ou sinal de projeto. Qualquer sinal isolado
  basta. **Nenhuma política de detecção *opt-in* específica por agente permanece no harness.**
- **RF-13:** A instalação **não** copia skills, **não** cria symlink e **não** escreve `skills.paths`
  nem `instructions`. Escreve **apenas** o bloco `permission` no `opencode.json`, preservando `$schema`
  e todos os campos preexistentes, de forma idempotente.
- **RF-14:** O plugin de governança é depositado no diretório de plugins de projeto, onde é
  auto-descoberto sem entrada de configuração (V-03: `.opencode/plugin/` — adota-se a forma singular).
- **RF-15:** Os validadores canônicos de evidência são instalados pelo mesmo caminho tool-neutro dos
  demais agentes: um projeto somente-OpenCode tem **exatamente** os mesmos gates que um somente-Claude.
- **RF-16:** O modo de acesso é expresso pelo bloco `permission` e/ou pela negociação do protocolo —
  **não** por flag, que o subcomando não oferece (V-02). É **proibido** presumir flag não confirmada.
- **RF-17:** O modelo é declarado pela chave `model` do `opencode.json`; a flag de modelo do harness
  mapeia para essa chave. A janela de contexto é **derivada do modelo resolvido** por **tabela
  versionada**, com **casamento exato** e depois por **maior prefixo**; sem modelo resolvível, aplica-se
  **fallback conservador declarado e testado**.
- **RF-18:** O OpenCode alcança **paridade observacional completa**: eventos, renderização de
  tool-calls, relatório de execução, watchdog, telemetria opt-in e normalização com tabela de alias
  própria.
- Nenhuma mudança de comportamento para os três agentes existentes (RF-62, O-06): a função de resolução
  de janela é **opcional** e, quando ausente, o comportamento é byte-idêntico ao atual.
</requirements>

## Subtarefas

- [ ] 7.1 Criar a spec do OpenCode em `internal/runtime/specs/opencode.go` (planejado), com `FixedArgs`
      iniciando pelo subcomando `acp`.
- [ ] 7.2 **Travar a invariante de prefixo posicional** por validação de formato: nenhum item de
      `FixedArgs` pode aparecer depois de um argumento iniciado por hífen. A validação falha de modo
      visível se a forma de invocação mudar a montante.
- [ ] 7.3 Declarar o **launcher de fallback** via gerenciador de pacotes com **versão pinada em
      constante**; adicionar gate que reprova a string `@latest` em qualquer launcher.
- [ ] 7.4 Registrar o OpenCode no registro único criado em 6.0, com os três pontos canônicos cobertos
      (o construtor de `Enforcement` recusa cobertura incompleta).
- [ ] 7.5 Implementar a **detecção de primeira classe** pelos três sinais (binário no `PATH`, diretório
      de configuração do usuário, sinal de projeto), sem executar binário (RF-26, tarefa 6.0).
- [ ] 7.6 Remover qualquer política de detecção **opt-in específica por agente** que ainda exista, de
      modo que nenhum agente seja cidadão de segunda classe da própria detecção.
- [ ] 7.7 Implementar a instalação de **pegada mínima**: merge idempotente que escreve **apenas** o bloco
      `permission` no `opencode.json`, preservando `$schema` e todos os campos preexistentes e **nunca**
      removendo configuração do usuário.
- [ ] 7.8 **NÃO escrever** `skills.paths` nem `instructions`, e adicionar teste que **falha** se qualquer
      uma das duas chaves for escrita pelo instalador — a regressão aqui é silenciosa e cara
      (duplicação da árvore de skills).
- [ ] 7.9 Depositar o plugin de governança em `.opencode/plugin/` (auto-descoberto, sem entrada de
      configuração). O **conteúdo** do plugin — bloqueio por exceção — é da tarefa 8.0; aqui apenas o
      caminho de depósito e a idempotência.
- [ ] 7.10 Instalar os **validadores canônicos** pelo mesmo caminho tool-neutro dos demais agentes.
- [ ] 7.11 Implementar a **tabela versionada de janela por modelo**, com resolução por casamento exato,
      depois por maior prefixo (cobrindo sufixos de data em identificadores de modelo), e **fallback
      conservador** em qualquer caminho não resolvido.
- [ ] 7.12 Tornar a função de resolução de janela **opcional na spec**: ausente, o comportamento dos três
      agentes atuais é byte-idêntico ao de hoje.
- [ ] 7.13 Mapear a flag de modelo do harness para a chave `model` do `opencode.json`, sem presumir
      nenhuma flag do subcomando (V-02).
- [ ] 7.14 **Assumir as células de ocupante único**: entrada do OpenCode em
      `internal/metrics/metrics.go:213-215` (`ToolBudgetsLarge`) e em `inherit_common` nos **dois**
      arquivos de regras de normalização (`.agents/normalization-rules.yaml:20-21` e
      `internal/runtime/events/normalization-rules.yaml:20-21`), mantendo o espelhamento verificado pelo
      gate de sincronia.
- [ ] 7.15 Criar o **gate de COBERTURA** (não de não-vazio): todo agente cuja janela resolvida for grande
      precisa de entrada em `ToolBudgetsLarge`, e nenhuma chave órfã pode existir. O gate força decisão
      explícita em vez de aceitar valor de fachada.
- [ ] 7.16 Completar a **paridade observacional**: eventos, renderização de tool-calls, relatório de
      execução, watchdog, telemetria opt-in e tabela de alias de normalização própria do OpenCode.

## Detalhes de Implementação

Ver `techspec.md`:

- §Sequenciamento de Desenvolvimento → **Restrição de ordem descoberta na análise** — por que a fase do
  OpenCode precede a da remoção: nas células de ocupante único, esvaziar a estrutura muda o comportamento
  e **nenhum teste falha**.
- §Sequenciamento de Desenvolvimento → **ACP por flag e por subcomando, sem caso especial** — a tabela de
  argv por agente e a invariante de prefixo posicional que passa a ser travada.
- §Sequenciamento de Desenvolvimento → **Janela derivada do modelo** — casamento exato, maior prefixo e
  fallback conservador; a segurança por construção ao remover uma entrada da tabela.
- §Sequenciamento de Desenvolvimento → Fases — **F3b — OpenCode**, dependente de F3a.
- §Considerações Técnicas → Conformidade com Padrões — "ADR-023 (classe de janela): estendida, não
  violada — a janela estática permanece o caminho dos agentes atuais".
- `adr-003-opencode-acp-subcomando.md` — decisão completa, alternativas rejeitadas (integração
  não-interativa, cópia de skills, constante estática de janela) e os riscos com mitigação.
- `prd.md` §Fatos Verificados, Bloco 1 — V-01, V-02, V-03, V-09, V-10 sustentam cada proibição desta
  tarefa; nenhuma delas é preferência de estilo.

Não duplicar aqui o desenho do plugin de bloqueio, da sanitização de ambiente nem do handshake:
pertencem à tarefa 8.0.

## Critérios de Sucesso

- `go build ./... && go vet ./... && go test ./... -count=1` verde ao final da tarefa.
- Teste prova que o argv gerado para o OpenCode é `opencode acp --cwd <repo> ...` — subcomando **antes**
  de qualquer argumento iniciado por hífen.
- Teste da **validação de formato** falha para uma spec artificial em que um item de `FixedArgs` aparece
  depois de um argumento com hífen.
- Teste prova que o launcher de fallback usa **versão pinada**; gate de varredura reprova `@latest` em
  qualquer launcher do repositório.
- `ai-spec-harness install .` em repositório com `opencode` no `PATH` **detecta** o agente sem nenhuma
  flag `--tools`; o mesmo vale com apenas o diretório de configuração presente e com apenas o sinal de
  projeto presente (três testes independentes, um por sinal).
- Nenhuma política de detecção opt-in por agente permanece: gate de varredura verde.
- Após `install`, o `opencode.json` do projeto contém **apenas** o bloco `permission` acrescentado;
  `$schema` e todos os campos preexistentes preservados byte a byte (verificável por diff).
- Teste **falha** se o instalador escrever `skills.paths` ou `instructions` — asserção explícita de
  ausência das duas chaves.
- Reexecutar `install` produz `opencode.json` **idêntico** (idempotência verificável por diff vazio).
- O plugin de governança aparece em `.opencode/plugin/` e nenhuma entrada de configuração o referencia.
- Um projeto somente-OpenCode e um projeto somente-Claude têm o **mesmo conjunto de validadores
  canônicos instalados** (comparação de listas de arquivos, ignorando o mecanismo nativo de cada CLI).
- Resolução de janela: teste com tabela cobrindo casamento exato, casamento por maior prefixo (modelo
  com sufixo de data), modelo desconhecido (fallback conservador) e modelo ausente (fallback
  conservador). Nenhum caminho resolve para janela **maior** que a da entrada correspondente.
- Gate de **cobertura** de orçamento: fica **vermelho** quando um agente de janela grande fica sem
  entrada **e** quando existe chave órfã em `ToolBudgetsLarge`. Os dois casos têm teste dedicado.
- `inherit_common` contém o OpenCode nos **dois** arquivos de regras de normalização, e o gate de
  sincronia de espelhos está verde.
- Paridade observacional: execução do OpenCode sob o servidor ACP falso produz `events.jsonl`,
  `tool_calls.md` e `execution_report.md` com a mesma estrutura dos demais agentes.
- Não-regressão: para os três agentes atuais, o argv gerado e a janela resolvida são **byte-idênticos**
  aos anteriores (golden files sem diff).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
  - Construção do argv para as duas formas (flag e subcomando), com suíte em tabela.
  - Validação de formato de `FixedArgs`: prefixo posicional válido aceito; item após hífen recusado.
  - Pinagem de versão do launcher de fallback; ausência de `@latest`.
  - Detecção por cada um dos três sinais isoladamente e por combinações.
  - Merge do `opencode.json`: preservação de `$schema` e campos preexistentes; idempotência; ausência
    de `skills.paths` e `instructions`.
  - Resolução de janela: exato, maior prefixo, desconhecido, vazio; fallback nunca superestima.
  - Gate de cobertura de orçamento: agente sem entrada e chave órfã.
- [ ] Testes de integração
  - `install` + `verify` sobre repositório temporário com OpenCode: estado `current` por skill e por
    agente; reexecução converge para o mesmo estado.
  - Paridade de validadores canônicos entre projeto somente-OpenCode e somente-Claude.
  - Sessão sob o servidor ACP falso in-process: eventos, tool-calls normalizados, watchdog e relatório
    de execução equivalentes aos dos demais agentes.
  - Não-regressão dos três agentes existentes (golden files de argv e de janela).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/specs/opencode.go` (planejado) — spec ACP por subcomando, launcher de fallback com
  versão pinada, resolução de janela por modelo.
- `internal/runtime/specs/spec.go` — abstração de spec; `FixedArgs` e a validação de formato do prefixo
  posicional.
- `internal/runtime/specs/registro.go` (planejado) — registro único (tarefa 6.0) que recebe a entrada do
  OpenCode.
- `internal/runtime/specs/janela.go` (planejado) — tabela versionada de janela por modelo.
- `internal/detect/agent.go` — detecção pelos três sinais; remoção de política opt-in por agente.
- `internal/install/install.go` — instalação de pegada mínima e merge do `opencode.json`.
- `internal/metrics/metrics.go:213-215` — `ToolBudgetsLarge`, célula de ocupante único a assumir.
- `.agents/normalization-rules.yaml:20-21` — `inherit_common`, célula de ocupante único (fonte).
- `internal/runtime/events/normalization-rules.yaml:20-21` — espelho embarcado da mesma lista.
- `.opencode/plugin/` (planejado) — diretório auto-descoberto onde o plugin de governança é depositado.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — §ACP por flag e por subcomando,
  §Janela derivada do modelo, §Restrição de ordem descoberta na análise.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-003-opencode-acp-subcomando.md`
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-002-catalogo-de-agentes-registro-unico.md`
