# Tarefa 10.0: Quality gate disparado por evento com selecao por risco e deduplicacao

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar `internal/qualitygate/` — o pacote **não existe** hoje (`ls internal/qualitygate` retorna
`No such file or directory`). Esta é a única família de hook genuinamente nova do escopo do PRD.

Gates de qualidade já existem no repositório, mas exclusivamente como **alvos de `Makefile`**
(`Makefile:23` `test`, `Makefile:26` `integration`, `Makefile:32` `lint`, `Makefile:36` `vet`,
`Makefile:42` `coverage`) e como **validadores shell** invocados em prosa pelas skills
(`.agents/scripts/validate-task-evidence.sh`). Nenhum deles é acionado por evento de lifecycle,
nenhum tem política de seleção por risco declarada, e nenhum deduplica execução redundante.

Cobre RF-25 a RF-30. Detalhamento em `techspec.md:409-451` (linha de rastreabilidade
`RF-25 a RF-30 → internal/qualitygate/ → TestQualityGate_SkipsWhenFingerprintUnchanged`) e
`techspec.md:508` (Fase 6). O resultado do gate é expresso pelo `Result` tipado do contrato
canônico, conforme [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md) — não reinventar
semântica de exit code aqui.

<requirements>
- RF-25: o gate dispara apenas em evento adequado, NUNCA a cada chamada de ferramenta. A User Story
  lista explicitamente "executar todos os testes após cada tool call" como FORA de escopo.
- RF-26: a política de seleção de checks por tipo e risco da tarefa é declarada em artefato, não
  inferida pelo agente em runtime.
- RF-27: falha de check obrigatório impede conclusão aprovada (`DecisionBlock`).
- RF-28: a saída do gate é armazenável como evidência, consumível pelos validadores existentes.
- RF-29: havendo resultado determinístico disponível, nenhuma revisão por LLM pode substituí-lo.
- RF-30: execuções redundantes são evitadas quando o estado relevante não mudou, com invalidação por
  fingerprint.
- Nenhuma decisão de policy vive dentro do gate de provedor: o gate consome `hookcontract` e
  `hookpolicy`, e o adapter apenas traduz (RF-56).
- R-STYLE-001 (hard): todo código em inglês, zero comentários, sem prefixo `_` em identificador Go.
- Zero regressão: nenhum alvo de `Makefile`, validador shell ou teste existente muda de
  comportamento observável sem registro explícito.
</requirements>

## Subtarefas

- [ ] 10.1 Confirmar que as dependências 4.0 (contrato canônico `internal/hookcontract`) e 7.0
      (resolução de toolchain das cinco stacks) estão `done`. Sem 7.0, `internal/detect/toolchain.go:148-163`
      resolve `fmt`/`test`/`lint` apenas para `go`, `node` e `python` — o `switch` não tem `case`
      para .NET nem para Java, e o gate seria estruturalmente incapaz de rodar nas cinco stacks
      exigidas por RF-31.
- [ ] 10.2 **Gate de ordem, bloqueante.** Verificar que a tarefa 5.0 já corrigiu o descarte de erro
      em `internal/runtime/runner.go:428` (`_ = disp.Dispatch(ctx, hooks.PointToolCallPreDispatch, …)`)
      e `internal/runtime/runner.go:466` (`_ = disp.Dispatch(ctx, hooks.PointToolCallPostComplete, …)`).
      Se 5.0 não estiver `done`, esta tarefa **para** e reporta `blocked`. Racional em
      `techspec.md:599` (risco R-12): hoje nenhum hook de produção está registrado nesses dois
      pontos, então deixar de descartar o erro é **inerte**; esta tarefa registra o **primeiro** hook
      de produção em `tool_call.pre_dispatch`/`tool_call.post_complete`, e se a correção entrar
      depois ela passa a bloquear onde hoje não bloqueia.
- [ ] 10.3 Declarar a política de seleção em artefato versionado (RF-26): mapa
      `(task type, risk) → checks obrigatórios / opcionais`, com esquema validado no carregamento e
      erro tipado em declaração inválida. Nenhuma inferência pelo agente.
- [ ] 10.4 Implementar `internal/qualitygate/` com o resolvedor que traduz a política declarada em
      comandos concretos por stack, consumindo `internal/detect.ToolchainResult`
      (`internal/detect/toolchain.go:147`) em vez de hardcoding.
- [ ] 10.5 Implementar o binding por evento (RF-25): registrar o gate apenas em
      `EventBeforeComplete` e, quando aplicável, `EventSessionEnd`. Teste que falha se o gate for
      registrado em `EventBeforeTool` ou `EventAfterTool`.
- [ ] 10.6 Implementar fingerprint de estado relevante e o cache de decisão (RF-30):
      `TestQualityGate_SkipsWhenFingerprintUnchanged` e o teste inverso, que prova reexecução quando
      o fingerprint muda. Skip deduplicado é `DecisionNotApplicable` com razão explícita, nunca
      `DecisionAllow` silencioso — distinção introduzida por
      [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md), que registra
      `.agents/scripts/git-operation-gate.sh:41` e `:133` como exemplo do problema inverso.
- [ ] 10.7 Implementar a precedência de RF-29: quando existir resultado determinístico para o check,
      nenhum caminho de revisão por LLM pode sobrescrevê-lo. Teste adversarial que tenta a
      sobrescrita e exige falha.
- [ ] 10.8 Persistir a saída do gate em formato consumível como evidência (RF-28), compatível com a
      regex de prova de teste de `.agents/scripts/validate-task-evidence.sh:180` — sem **ampliar**
      a regex nesta tarefa (a ampliação pertence à 7.0 e carrega o risco R-02 de afrouxar o gate).
- [ ] 10.9 Implementar RF-27: check obrigatório com falha produz `DecisionBlock` com `policyID` e
      `gateID` preenchidos, traduzido para exit code pelo tradutor único do contrato.
- [ ] 10.10 Registrar o pacote de teste novo em um job que o CI executa
       (`.github/workflows/test.yml`), sob pena de gate órfão — armadilha ativa registrada em
       `tasks.md`, seção "Armadilhas verificadas".
- [ ] 10.11 Rodar os gates de não-regressão e anexar as saídas ao `execution_report.md`.

## Detalhes de Implementação

- Arquitetura e fluxo de dados: `techspec.md:76-113`. O gate fica abaixo de `hookcontract` e nunca
  importa pacote de provedor.
- Interface de `Result` e construtores validadores: `techspec.md:116-180`. O gate **não** define
  decisão própria.
- Tradução de exit code: ponto único em `hookcontract/exitcode.go`, com `critical == true` fazendo
  código não mapeado virar `DecisionError`. Ver
  [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md) — não duplicar o racional aqui.
- Sequenciamento: `techspec.md:508` (Fase 6, quality gate e telemetria).
- Estado atual do resolvedor de toolchain: `internal/detect/toolchain.go:148-163` cobre três stacks.
  A ordem de `manifestTypes` (`internal/detect/toolchain.go:99-108`) e o desempate "primeiro
  candidato com score máximo vence" (`internal/detect/toolchain.go:140-145`) são propriedade da
  tarefa 7.0; esta tarefa **consome** o resultado e não altera a ordem.

## Critérios de Sucesso

- `internal/qualitygate/` existe, é consumido por pelo menos um ponto de produção, e não importa
  nenhum pacote de provedor (gate de AST análogo a `TestHookContractHasNoProviderIdentifier`).
- Teste que **falha** se o gate for registrado em evento por chamada de ferramenta (RF-25).
- Política de seleção carregada de artefato declarado; teste que prova erro tipado em política
  inválida e ausência de qualquer inferência em runtime (RF-26).
- `TestQualityGate_SkipsWhenFingerprintUnchanged` verde, e o teste inverso de invalidação verde
  (RF-30).
- Check obrigatório falhando produz `DecisionBlock` com `policyID` e `gateID` (RF-27).
- Saída do gate persistida e aceita pelos validadores de evidência sem ampliação de regex (RF-28).
- Teste adversarial de RF-29 verde.
- **Não-regressão (inegociável):** `make test lint vet` verdes, sem alteração de comportamento
  observável em nenhum hook classificado `KEEP` pela tarefa 1.0.
- `make integration` verde, com o pacote novo efetivamente incluído no alvo (`Makefile:26`).
- `make coverage` respeitando 75% total e 70% por pacote crítico (`Makefile:42`, `Makefile:47`).
- `internal/runtime/runner.go:428` e `:466` já sem `_ =` no momento do merge desta tarefa —
  evidência anexada.

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

**A criar**

- `internal/qualitygate/` — pacote novo; não existe no repositório.
- Artefato de política de seleção por tipo e risco (RF-26).

**A modificar**

- `internal/runtime/runner.go:428,466` — verificação de pré-requisito (correção pertence à 5.0).
- `.github/workflows/test.yml` — registro do pacote de teste novo, anti-gate-órfão.

**A consumir, sem alterar**

- `internal/detect/toolchain.go:99-108,140-145,147,148-163` — resolvedor de toolchain.
- `internal/hookcontract/{result,exitcode}.go` — entregues pela tarefa 4.0.
- `.agents/scripts/validate-task-evidence.sh:180` — regex de prova de teste.
- `Makefile:23,26,32,36,42,47` — alvos existentes dos gates.

**Documentos de referência**

- `.specs/prd-hooks-canonicos-vendor-neutral/prd.md:277-286` (RF-25 a RF-30).
- `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md:76-180,409-451,508,599`.
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-002-resultado-tipado-traducao-exit-code.md`.
