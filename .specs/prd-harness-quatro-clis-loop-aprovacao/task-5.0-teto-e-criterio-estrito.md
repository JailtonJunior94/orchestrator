# Tarefa 5.0: Propagação do teto de rodadas e virada do critério estrito

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Esta tarefa faz a **primeira exposição** do teto de rodadas do Ciclo de Aprovação e, só depois disso,
vira o critério de encerramento do Ciclo para o modo estrito.

O campo `Options.MaxBugfixIterations` (`internal/taskloop/taskloop.go:37`) existe desde a introdução do
loop de correção e **nunca teve escritor**: nenhum ponto de `cmd/ai_spec_harness/task_loop.go` o
preenche, então ele chega sempre com valor `0` e cai no default interno de 3 rodadas. Expor esse
parâmetro exige percorrer uma cadeia de **seis saltos**, e omitir qualquer um deles produz falha
silenciosa — não erro.

Concluída a propagação, o critério de encerramento é virado: veredito com ressalvas deixa de encerrar a
tarefa e passa a **realimentar** o ciclo de correção; teto esgotado sem `APPROVED` retorna `blocked`,
nunca `done`.

A ordem interna é obrigatória: **primeiro a cadeia de propagação, depois a virada do critério**. Virar o
critério com o teto ainda fixo em 3 remove a válvula de configuração exatamente no momento em que o
número de rodadas passa a importar.

<requirements>
- **RF-32:** No modo orquestrado, o Ciclo é ativado pela **mesma flag opt-in** que hoje ativa a
  auto-revisão (`--auto-review`), com default desligado — não é criada flag nova de ativação. O que muda
  é que, ativado, o Ciclo **itera** em vez de rodar uma vez. No fluxo de skill, onde a revisão já é
  obrigatória, o Ciclo vale sempre.
- **RF-35:** O teto de rodadas passa a ser **5** por default, configurável por flag de linha de comando
  **e** por arquivo de configuração, respeitando a precedência vigente
  (flags > workspace > global > defaults built-in, ADR-016). É a primeira exposição desse parâmetro
  (V-24), o que exige percorrer toda a cadeia de propagação **e** acrescentar a chave ao merge campo a
  campo — omitir o merge faz a configuração ser silenciosamente ignorada.
- **RF-36:** Esgotado o teto sem `APPROVED`, o resultado é **`blocked`** — nunca `done`. O retorno
  identifica o motivo canônico (`max_rounds`) e devolve a decisão ao humano.
- **RF-56:** É proibido afirmar aprovação, execução de comando ou cobertura de critério que não tenha
  ocorrido. Na ausência de evidência, o resultado é `blocked` ou `needs_input` — nunca sucesso aparente.
- A regra atual que fecha tarefa com `APPROVED_WITH_REMARKS` sem tag crítica é **removida** (RF-33, já
  modelada em 2.0): aqui ela é desligada nos consumidores.
- Nenhum default muda para os agentes existentes quando a flag opt-in não é passada (RF-62, O-06).
- Cada teste alterado entra no diff **com justificativa por requisito**. Alterar asserção sem citar o RF
  que a torna obsoleta é proibido — é o padrão exato que deixou o gate de encerramento do Copilot inerte
  (V-21).
</requirements>

## Subtarefas

- [x] 5.1 Registrar a chave do teto no esquema da linha de comando (`docs/cli-schema.json`), que é
      validado **bidirecionalmente** por `cmd/ai_spec_harness/cli_contract_test.go:189` — flag ausente
      do esquema e flag ausente da implementação falham igualmente.
- [x] 5.2 Registrar e ler a flag em `cmd/ai_spec_harness/task_loop.go`, com a **validação de presença**
      via `cmd.Flags().Changed`: valor menor que 1 falha de forma explícita e tipada, e valor negativo
      **nunca** é normalizado em silêncio.
- [x] 5.3 Propagar o valor a `taskloop.Options` (`internal/taskloop/taskloop.go:37`), preenchendo pela
      primeira vez o campo hoje sem escritor.
- [x] 5.4 Acrescentar a opção correspondente ao invoker ACP, seguindo o padrão de
      `WithACPInvokerAutoReview` (`internal/taskloop/acpinvoker.go:126-128`) e o repasse ao `Job` em
      `acpinvoker.go:243`.
- [x] 5.5 Propagar a `runtime.Job` / `RuntimeConfig` (`internal/runtime/types.go`), com o zero-value
      tratado por `ApplyDefaults` (`internal/runtime/types.go:34`) de modo a preservar comportamento
      pré-mudança quando o campo não vier preenchido.
- [x] 5.6 Acrescentar a chave ao `config.Runtime` (`internal/config/runtime.go:20-31`, com tag `yaml`)
      **e** a linha correspondente em `mergeInto` (`internal/config/resolver.go:166-199`). As duas
      edições são inseparáveis: a struct sem o merge produz uma chave que o resolver lê do arquivo e
      descarta silenciosamente.
- [x] 5.7 Documentar, junto ao campo, o **alerta de semântica**: `mergeInto` usa "não-zero vence"
      (`resolver.go:164`), o que torna `0` indistinguível de ausente — razão pela qual a validação de
      teto menor que 1 vive na camada CLI, a única que sabe distinguir presença.
- [x] 5.8 Ligar o consumo do teto ao agregado do Ciclo (`internal/approval` (planejado), tarefa 2.0),
      substituindo o teto fixo de `internal/taskloop/bugfix.go:66` pelo valor resolvido.
- [x] 5.9 Virar o critério: veredito com ressalvas realimenta o ciclo em vez de encerrar
      (`internal/taskloop/bugfix.go:83` e a saída em `:131`).
- [x] 5.10 Virar o resultado de teto esgotado para `blocked` com motivo canônico `max_rounds`, e garantir
      que esse estado terminal **não** aciona o caminho de retry (RF-45, já modelado em 2.0).
- [x] 5.11 Revisar cada teste que asserta encerramento com ressalvas ou teto 3, registrando no diff a
      justificativa por requisito de cada asserção alterada.

## Detalhes de Implementação

Ver `techspec.md`:

- §Arquitetura do Sistema → Componentes modificados — linha
  `internal/config/`, `cmd/ai_spec_harness/task_loop.go`: "Primeira exposição do teto de rodadas, com
  merge campo a campo".
- §Sequenciamento de Desenvolvimento → Fases — **F2c — Critério estrito**: "Cadeia de propagação do teto
  de rodadas e virada do critério de encerramento", dependente de F2b.
- §Considerações Técnicas → Riscos Conhecidos — última linha: "A semântica de merge 'não-zero vence'
  torna zero indistinguível de ausente / A validação de teto vive na camada de linha de comando, a única
  que sabe distinguir presença; valor negativo falha explicitamente em vez de ser normalizado em
  silêncio".
- §Abordagem de Testes → Testes Unitários — "Política: teto menor que 1 recusado na construção".
- `adr-001-ciclo-de-aprovacao-agregado.md` para a autoridade do agregado sobre a regra de parada: o teto
  é um parâmetro da política, e a decisão de parar continua sendo do agregado, não do consumidor.

Não duplicar aqui o desenho do agregado: ele pertence à tarefa 2.0 e à techspec §Pacote de domínio
`internal/approval`.

## Critérios de Sucesso

- `go build ./... && go vet ./... && go test ./... -count=1` verde ao final da tarefa.
- `go test ./cmd/ai_spec_harness/ -run TestCLI_Contract -count=1` verde: o esquema em
  `docs/cli-schema.json` e a implementação Cobra concordam nos dois sentidos sobre a nova flag.
- Executar `ai-spec-harness task-loop --help` lista a flag do teto de rodadas com default **5**
  declarado no texto de ajuda.
- Executar o comando com valor `0` e com valor negativo produz **erro de saída não-zero** com mensagem
  citando o mínimo aceito; a saída não contém nenhuma normalização silenciosa.
- Teste de resolução de configuração prova a precedência completa para a nova chave: valor em
  `~/.aispec/config.yaml` é sobreposto por `.claude/config.yaml`, que é sobreposto pela flag.
- Teste de regressão prova que a chave escrita **apenas** no arquivo de configuração chega ao
  `taskloop.Options` — o teste falha se a linha de `mergeInto` for removida.
- Teste prova que ciclo com veredito `APPROVED_WITH_REMARKS` **não** encerra: a rodada seguinte é
  iniciada e o corretor é chamado.
- Teste prova que ciclo que esgota o teto retorna estado `blocked` com motivo `max_rounds`, e que
  nenhuma execução retorna `done` sem `APPROVED`.
- Teste prova que o caminho de retry de infraestrutura **não** é acionado por `max_rounds`.
- Sem a flag opt-in, o comportamento observável dos fluxos existentes é idêntico ao anterior
  (`git diff` dos golden files de execução vazio).
- Todo teste alterado tem, no corpo do diff, comentário citando o RF que justifica a alteração.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
  - Validação do teto na camada CLI: `0`, negativo e ausente produzem, respectivamente, erro, erro e
    default 5.
  - `mergeInto` com a nova chave: suíte com tabela cobrindo global-only, projeto-only, ambos e nenhum.
  - `ApplyDefaults` preserva comportamento anterior para o zero-value.
  - Encerramento estrito: `APPROVED` encerra; `APPROVED_WITH_REMARKS` realimenta; teto esgotado bloqueia.
- [x] Testes de integração
  - Cadeia de propagação ponta a ponta (flag → `Options` → invoker ACP → `Job` → agregado) com o servidor
    ACP falso in-process, sem depender de CLI real.
  - Precedência de configuração com arquivos reais em diretório temporário (global + workspace + flag).
  - Não-regressão: fluxo sem a flag opt-in produz resultado idêntico ao anterior.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `docs/cli-schema.json` — esquema da linha de comando; contrato bidirecional.
- `cmd/ai_spec_harness/cli_contract_test.go:189` — validação bidirecional flags ↔ esquema.
- `cmd/ai_spec_harness/task_loop.go` — registro e leitura da flag; única camada que distingue presença
  via `cmd.Flags().Changed`.
- `internal/taskloop/taskloop.go:37` — `Options.MaxBugfixIterations`, campo hoje sem escritor.
- `internal/taskloop/acpinvoker.go:126-128`, `:243` — padrão de opção do invoker e repasse ao `Job`.
- `internal/runtime/types.go:34` — `RuntimeConfig.ApplyDefaults`.
- `internal/runtime/types.go:128-133` — `AutoReview`, a flag opt-in reaproveitada por RF-32.
- `internal/config/runtime.go:20-31` — struct `Runtime` e tags `yaml`.
- `internal/config/resolver.go:164-199` — `mergeInto` campo a campo, semântica "não-zero vence".
- `internal/taskloop/bugfix.go:66`, `:83`, `:131` — teto atual, loop e ponto de saída.
- `internal/approval/politica.go` (planejado) — política que passa a receber o teto resolvido.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — §Fases (F2c), §Riscos Conhecidos.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md` — autoridade do
  agregado sobre a regra de parada.
