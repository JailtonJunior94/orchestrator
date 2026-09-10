# Tarefa 10.0: Remoção total do Gemini e desinstalação fiel

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Executar as etapas topológicas restantes da remoção do Gemini. A etapa 1 da ordem de build — desarmar o
gate auto-bloqueante de `cmd/ai_spec_harness/cli_contract_test.go:181-184`, que **exige** a string
`gemini` no esquema do CLI — já foi feita na tarefa 1.0; esta tarefa continua dali.

A ordem interna é obrigatória e cada passo é um commit que deixa o repositório verde
(`go build ./... && go vet ./... && go test ./... -count=1` mais os gates de sincronia). A ordem não é
preferência de estilo: **cinco arquivos de teste fora do pacote de specs consomem as constantes do
agente**, e removê-las antes das folhas quebra a compilação de quatro pacotes ao mesmo tempo, impedindo
bissecção exatamente na etapa de maior raio de explosão da entrega.

Duas correções de defeito pré-existente são pré-requisito da regeneração de snapshots, não trabalho
vizinho: `internal/contextgen/contextgen.go` emite a tabela de capacidades **ignorando os agentes
selecionados** — é por isso que o snapshot de projeto só-codex, `testdata/snapshots/codex-only.agents.md`,
cita outros agentes — e afirma, em `internal/contextgen/contextgen.go:493` e `:594`, que o Copilot **não
tem hooks nativos**, afirmação hoje comprovadamente falsa (a tarefa 9.0 acabou de ligar o hook de
encerramento dele). Corrigir os dois **antes** de regenerar os 9 golden files faz com que eles passem a
refletir a verdade, em vez de congelar dois defeitos.

<requirements>
- RF-01: o conjunto canônico de agentes passa a ser exatamente `{claude, codex, copilot, opencode}`;
  toda superfície que hoje enumera o Gemini deixa de aceitá-lo — seleção de ferramenta, catálogo de
  runtimes ACP, catálogo de drivers, perfis de execução, tabela de compatibilidade de modelos e tabela
  de orçamento de tokens por fluxo.
- RF-02: todos os artefatos exclusivos do agente removido são apagados do repositório e dos assets
  embarcados — diretório de configuração, arquivo de governança raiz, rotina de instalação dedicada,
  geradores de adaptadores, settings default, invoker legado, extrator de métricas dedicado, testes de
  integração dedicados e fixture de paridade.
- RF-03: invocar o agente removido por nome produz erro **tipado e explicativo**, citando o conjunto
  suportado e apontando o guia de migração — nunca o erro genérico de "valor inválido".
- RF-04: a ADR do runtime removido é marcada como **Substituída**, com o **corpo intocado**; a tabela de
  governança por ferramenta, o README, a matriz de degradação, a matriz de confiabilidade e o guia de
  instalação refletem os 4 agentes. Changelog, ADRs, PRDs anteriores, auditorias e evidências de execução
  são históricos e **não** são reescritos.
- RF-05: a desinstalação remove **exclusivamente** os arquivos que o harness instalou; o manifesto passa
  a rastrear arquivos individualmente e a desinstalação passa a consumi-lo como fonte de verdade, em vez
  de lista fixa. Arquivos do usuário preservados; diretório removido só se ficar vazio; operação
  idempotente e não-fatal quando nada existir.
- RF-09: testes e snapshots dedicados removidos ou regenerados; contagens fixas de agentes e asserções
  por índice de slice revistas; `--tools=all` resolve exatamente os 4 agentes.
- RF-60: a desinstalação passa a remover todos os arquivos que a instalação cria — os que hoje ficam
  órfãos —, condição necessária para RF-05 ser verdadeiro.
- Campo de rastreamento por arquivo no manifesto é **aditivo**, e sua **omissão** cai em caminho
  conservador **anunciado**, para manifestos antigos no disco de usuários.
- A regeneração de snapshots exige **revisão manual do diff arquivo a arquivo**: regeneração automática
  é um cheque em branco.
</requirements>

## Subtarefas

- [ ] 10.1 **Desfragilizar asserts posicionais e contagens fixas**, de modo que as etapas de remoção
      seguintes não precisem tocar nesses arquivos.
- [ ] 10.2 **Remover as folhas de teste primeiro** — os cinco arquivos de teste fora do pacote de specs
      que consomem as constantes do agente. Inverter a ordem quebra a compilação de quatro pacotes ao
      mesmo tempo e impede bissecção.
- [ ] 10.3 **Remover a camada de runtime/specs**, atômica com as seis linhas do esquema do CLI em
      `docs/cli-schema.json`, e ligar a direção inversa do gate de contrato (nenhum agente aposentado
      pode aparecer no esquema).
- [ ] 10.4 **Remover o valor do enum de agentes**, guiado pelo compilador — etapa grande e
      inevitavelmente atômica. Inclui a **remoção completa da política de detecção opt-in**
      (`internal/detect/agent.go:125-129` e `:181`), a única exceção de política do conjunto.
- [ ] 10.5 **Decisões de conteúdo** que os gates de não-vacuidade da tarefa 7.0 forçam a explicitar —
      `internal/metrics/metrics.go:213-216` (`ToolBudgetsLarge`) e `inherit_common` nos dois arquivos de
      regras de normalização já devem ter o ocupante novo; confirmar e explicitar.
- [ ] 10.6 **Artefatos físicos e espelhos**, com ordem interna crítica: (a) parametrizar os arrays dos
      scripts de sincronia, (b) remover a entrada do agente desses arrays, (c) **só então** apagar os
      diretórios. Inverter apaga diretórios que os scripts ainda enumeram e quebra o gate.
- [ ] 10.7 **CI e release**: remover o empacotamento do diretório do agente dos workflows.
- [ ] 10.8 **ADR do runtime removido marcada como `Status: Substituída`**, corpo intocado.
- [ ] 10.9 Erro tipado e explicativo na invocação por nome do agente removido, citando o conjunto
      suportado e apontando o guia de migração.
- [ ] 10.10 Manifesto ganha **rastreamento por arquivo**: hoje `internal/manifest/manifest.go:14-26`
      indexa por **nome de skill** (`Skills []string`, `SkillVersions`) e hasheia o arquivo de **origem**
      (`Checksums`, `SourceDir`) — não serve para desinstalar. O campo novo é aditivo, e sua omissão cai
      em caminho conservador anunciado.
- [ ] 10.11 Desinstalação passa a consumir o manifesto como fonte de verdade e a remover os arquivos que
      a instalação cria e que hoje ficam órfãos, preservando arquivos do usuário; substituir a lista fixa
      de `internal/uninstall/uninstall.go:124-137`.
- [ ] 10.12 Corrigir o bug pré-existente de `internal/contextgen/contextgen.go:497` que emite a tabela de
      capacidades **ignorando os agentes selecionados**.
- [ ] 10.13 Corrigir a afirmação factualmente falsa de que o Copilot não tem hooks nativos, em
      `internal/contextgen/contextgen.go:493` e `:594`.
- [ ] 10.14 **Só depois de 10.12 e 10.13**, regenerar os 9 golden files de `testdata/snapshots/` via
      `UPDATE_SNAPSHOTS=1` (`internal/contextgen/contextgen_test.go:190`), com **revisão manual do diff
      arquivo a arquivo**, distinguindo "mudou porque o agente saiu" de "mudou porque o gerador quebrou".

## Detalhes de Implementação

Ver techspec.md, seções:

- **"Ordem de build da Fase 4 (remoção)"** — as quinze etapas; esta tarefa executa da 7 à 15, já que a
  etapa 1 foi feita na tarefa 1.0 e as etapas 2–6 e 8 pertencem às tarefas 1.0, 7.0 e 9.0.
- **"Regeneração de snapshots"** — por que a regeneração automática é um cheque em branco.
- **"Sequenciamento de Desenvolvimento" → "Restrição de ordem descoberta na análise"** — as células de
  ocupante único que precisam receber o novo agente **antes** da remoção.
- **"Riscos Conhecidos"** — as linhas sobre golden files, sobre o bug pré-existente da geração de
  governança e sobre manifestos antigos no disco de usuários.
- **"Arquivos Relevantes e Dependentes" → "Removidos"** — o mapa de remoção.

## Critérios de Sucesso

- Cada uma das subtarefas 10.1 a 10.11 é um commit isolado após o qual
  `go build ./... && go vet ./... && go test ./... -count=1` sai 0, mais
  `bash scripts/check-skills-sync.sh`, `bash scripts/check-hooks-sync.sh`,
  `bash scripts/check-scripts-sync.sh` e `bash scripts/check-spec-paths.sh`.
- `grep -rni "gemini" --include="*.go" internal/ cmd/` retorna **zero** ocorrências em código de produção.
- `grep -rni "gemini" docs/cli-schema.json` retorna zero ocorrências, e o gate de contrato passa a
  **reprovar** a presença da string (direção invertida) — provado por teste negativo.
- `--tools=all` resolve exatamente 4 agentes; asserção por contagem explícita, não por índice de slice.
- Invocar `--tool gemini` produz erro tipado citando `{claude, codex, copilot, opencode}` e o guia de
  migração; o teste distingue esse erro do erro genérico de valor inválido via `errors.As`.
- `internal/metrics/metrics.go` mantém `ToolBudgetsLarge` **não vazio**; o gate de não-vacuidade fica
  vermelho se a entrada for removida sem substituto.
- Nenhum diretório `.gemini/` permanece no repositório nem em `internal/embedded/assets/`, e nenhum array
  de espelho em `scripts/check-hooks-sync.sh:24-33` ou `scripts/check-skills-sync.sh:122-131` referencia
  caminho inexistente.
- A ADR do runtime removido tem `Status: Substituída` e `git diff` sobre ela mostra **exclusivamente** a
  linha de status alterada.
- Desinstalação: teste que instala em diretório temporário, cria um arquivo do usuário no mesmo diretório,
  desinstala, e afirma que **todo** arquivo criado pela instalação sumiu e o arquivo do usuário
  permaneceu. Reexecutar a desinstalação é idempotente e sai 0.
- Manifesto sem o campo novo (fixture simulando manifesto antigo no disco) faz a desinstalação cair em
  caminho conservador e **anunciar** isso na saída — nunca apagar arquivo não rastreado.
- `go test ./internal/contextgen/... -count=1` passa e `testdata/snapshots/codex-only.agents.md` **não**
  cita agente fora do conjunto selecionado.
- Nenhum dos 9 arquivos de `testdata/snapshots/` afirma que o Copilot não tem hooks nativos.
- O diff dos 9 golden files é revisado item a item e cada alteração é justificada por escrito na evidência
  da tarefa — mudança não justificada bloqueia a conclusão.

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
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `internal/manifest/manifest.go:14-26` — struct `Manifest`, hoje indexada por nome de skill e com hash
  do arquivo de origem; recebe o campo aditivo de rastreamento por arquivo
- `internal/uninstall/uninstall.go:124-137` — lista fixa de remoção, com o bloco dedicado ao agente
  removido; passa a consumir o manifesto
- `internal/install/install.go:816-878` — rotina que cria os arquivos hoje órfãos na desinstalação
- `internal/detect/agent.go:125-129`, `:181` — política de detecção opt-in a remover por completo
- `internal/skills/skills.go:13` — ordem canônica única, fonte da enumeração de agentes
- `internal/metrics/metrics.go:213-216` — `ToolBudgetsLarge`, célula de ocupante único
- `.agents/normalization-rules.yaml:20-21` e o gêmeo embarcado — `inherit_common`, célula de ocupante único
- `internal/contextgen/contextgen.go:493`, `:497`, `:594` — afirmação falsa sobre hooks do Copilot e
  tabela de capacidades que ignora os agentes selecionados
- `internal/contextgen/contextgen_test.go:190`, `:239-253` — gancho `UPDATE_SNAPSHOTS=1` e a tabela de casos
- `testdata/snapshots/` — os 9 golden files a regenerar com revisão manual
- `cmd/ai_spec_harness/cli_contract_test.go:181-184` — gate de contrato; direção inversa ligada na etapa 10.3
- `docs/cli-schema.json` — as seis linhas do esquema do CLI, atômicas com a camada de runtime/specs
- `scripts/check-hooks-sync.sh:24-33`, `scripts/check-skills-sync.sh:21-25`, `:122-131` — arrays a
  parametrizar antes de apagar diretórios
- `.github/workflows/release.yml`, `.github/workflows/release-dry-run.yml` — empacotamento do diretório do agente
- `.specs/adr/` e `docs/adr/` — ADR do runtime removido, apenas a linha de status alterada
