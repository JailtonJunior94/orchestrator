# Tarefa 12.0: Telemetria comum, ablation, timeout e guarda de recursao

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar as quatro lacunas de observabilidade e contenção do contrato de hooks: schema de telemetria
comum aos quatro provedores, ablation com baseline, timeout declarado por hook e guarda de recursão
`hook → ferramenta → hook`. Cobre RF-45, RF-46, RF-47, RF-48, RF-50, RF-64, RF-65, RF-66 e RF-67.

Estado verificado, que delimita o que é construção e o que é consumo:

- A telemetria do repositório é **opt-in** por `GOVERNANCE_TELEMETRY=1`
  (`internal/telemetry/telemetry.go:13`, `:56`, `:88`), **append-only**
  (`internal/telemetry/telemetry.go:23,66,98` abrem com `os.O_APPEND|os.O_CREATE|os.O_WRONLY`), e
  em formato `<ts> chave=valor`, por decisão registrada na ADR-006 do repositório
  (`docs/adr/006-telemetria-feedback-cycle.md`). O formato e o opt-in são **preservados**.
- `internal/telemetry/ablation.go` **já existe**: `ComponentKind` (`:5`, com `ComponentSkill`,
  `ComponentHook`, `ComponentPolicy` em `:8-10`), `AblationDecision` (`:13`, com `AblationKeep`,
  `AblationOnDemand`, `AblationRemove` em `:16-18`), `AblationBaseline` (`:21`),
  `BuildAblationBaseline` (`:26`), `AblationResult` (`:42`) e `CompareAblation` (`:55`). Esta tarefa
  **consome** essas primitivas; não as recria.
- Falta o **schema formal comum por provedor**. Existe apenas o contrato implícito em
  `internal/telemetry/parser.go`: o mapa `rf44MetricNameToProductionLogKey` (`:10-17`), que liga
  seis nomes de métrica às chaves de log de produção, e o conjunto canônico de quinze métricas em
  `RF44MetricNames()` (`:20-25`) — `provider`, `model`, `task_type`, `risk`, `tokens_in`,
  `tokens_out`, `duration`, `retries`, `tool_calls`, `context_loaded`, `skills_loaded`,
  `review_rounds`, `tests_passed`, `human_intervention`, `final_status`. `RF44MetricsWithProductionWriter()`
  (`:27-35`) expõe a lacuna: só as seis mapeadas têm escritor de produção.
- **Nenhum script shell declara timeout.** `grep -rn "timeout" .agents/` retorna apenas prosa de
  referência de skill (`.agents/skills/python-implementation/references/resilience.md:4,5,6,10,13`),
  zero declaração executável. O único timeout real do repositório é do plugin OpenCode:
  `VALIDATOR_TIMEOUT_MS` em `.opencode/plugin/governance.js:16`
  (`Number(process.env.AISPEC_OPENCODE_VALIDATOR_TIMEOUT_MS) || 8000`), aplicado em `:182`, `:212` e
  `:252`, com timeout tratado como negação em `:187` e `:217`. É comportamento de **um provedor**,
  não propriedade do contrato — RF-66 exige promovê-lo a propriedade do contrato sem quebrar o
  comportamento atual do OpenCode.

<requirements>
- RF-45: capturar métricas operacionais sem depender de o agente reportá-las — provedor, modelo,
  evento, identificador de tarefa, tipo de tarefa, risco, duração, chamadas de ferramenta,
  retentativas, skills carregadas, contexto carregado, rodadas de revisão e status final.
- RF-46 (regra dura): métrica indisponível é registrada como `unknown`. Nenhum valor é inventado ou
  estimado. Tokens e custo só aparecem quando fornecidos pelo provedor ou calculáveis por método
  explicitamente identificado no próprio registro.
- RF-47: schema comum aos quatro provedores, com extensões específicas de provedor isoladas em
  espaço próprio.
- RF-48: falha não crítica de telemetria não bloqueia a tarefa nem dispara cascata de retentativas.
- RF-50: correlação por tarefa e por sessão sem depender do conteúdo do prompt.
- RF-64: medir, por hook não crítico, latência adicionada, execuções por tarefa, falhas evitadas,
  regressões detectadas, intervenções humanas evitadas e contexto adicionado, com baseline sem o
  hook.
- RF-65: hook sem ganho observável é removido ou convertido para `ON-DEMAND`. Segurança crítica não
  depende de retorno econômico para permanecer habilitada.
- RF-66: todo hook tem timeout declarado, e o tempo gasto por hook é mensurável.
- RF-67: recursão `hook → ferramenta → hook` impedida **por construção**, não por convenção.
- PRESERVAR o opt-in `GOVERNANCE_TELEMETRY=1` e o formato `<ts> chave=valor`. **Não** introduzir
  Prometheus nem Grafana — seria dependência sem demanda para uma CLI local.
- R-STYLE-001 (hard): código em inglês, zero comentários, sem prefixo `_` em identificador Go.
- Zero regressão: nenhum consumidor atual de telemetria muda de comportamento sem registro.
</requirements>

## Subtarefas

- [ ] 12.1 Confirmar que 4.0 (contrato canônico) e 8.0 (adapters dos quatro provedores com
      `SessionStart` e `SessionEnd` reais) estão `done`. Sem 8.0 não existem os pares
      `(provedor, evento)` declarados sobre os quais o schema por provedor se apoia.
- [ ] 12.2 Formalizar o schema comum (RF-47): promover o conjunto canônico hoje implícito em
      `internal/telemetry/parser.go:20-25` a schema declarado e validado, com espaço de nomes
      **separado** para extensões de provedor. Gate que falha se uma chave específica de provedor
      vazar para o espaço comum.
- [ ] 12.3 Implementar a regra dura de RF-46: valor indisponível é gravado literalmente como
      `unknown`. `tokens_in` e `tokens_out` só são emitidos quando fornecidos pelo provedor ou
      calculáveis por método explicitamente identificado no registro. `TestTelemetry_UnavailableIsUnknown`
      e um teste adversarial que falha se qualquer caminho estimar valor sem identificar o método.
- [ ] 12.4 Ligar as métricas de RF-45 que hoje **não** têm escritor de produção: o mapa
      `rf44MetricNameToProductionLogKey` (`internal/telemetry/parser.go:10-17`) cobre seis das
      quinze de `RF44MetricNames()` (`:20-25`). Acrescentar escritor para as demais, ou declará-las
      `unknown` por construção com justificativa registrada — nunca deixar a lacuna implícita.
- [ ] 12.5 Emitir as entradas novas de hook, no formato existente e sob o mesmo opt-in:
      `hook.duration_ms`, `hook.decision`, `hook.timeout` e `hook.ablation`, conforme
      `techspec.md:556-563`.
- [ ] 12.6 Implementar RF-50: correlação por `task_id` e `session_id` derivados do contrato
      (`hookcontract.Envelope.SessionID`, `techspec.md:200-208`), **nunca** do conteúdo do prompt.
      Teste que falha se qualquer campo de correlação for derivado de texto de prompt.
- [ ] 12.7 Implementar RF-48: falha de telemetria é não crítica por construção — não propaga erro
      ao caminho de decisão do hook e não conta como tentativa para nenhuma política de retry.
      Teste com escritor de telemetria que sempre falha, provando que a tarefa conclui e que a
      contagem de retentativas permanece inalterada.
- [ ] 12.8 Consumir `BuildAblationBaseline` (`internal/telemetry/ablation.go:26`) e
      `CompareAblation` (`:55`) para produzir o baseline **sem** o hook exigido por RF-64, e a
      decisão `KEEP` / `ON-DEMAND` / `REMOVE` de RF-65 usando as constantes já existentes em
      `internal/telemetry/ablation.go:16-18`. `TestAblation_BaselineWithoutHook` verde.
- [ ] 12.9 Implementar a exceção de RF-65: hook de segurança crítica permanece habilitado
      independentemente do resultado da ablation. Teste que prova que uma ablation `REMOVE` sobre
      hook crítico **não** o desabilita.
- [ ] 12.10 Implementar timeout declarado por hook (RF-66) como propriedade do contrato, com o
       tempo gasto mensurável e emitido como `hook.duration_ms`. Timeout é tratado como **negação**,
       preservando a semântica já implementada em `.opencode/plugin/governance.js:187,217`. Não
       alterar o default de 8000 ms nem o nome da variável `AISPEC_OPENCODE_VALIDATOR_TIMEOUT_MS`
       (`.opencode/plugin/governance.js:16`) sem registro explícito — é comportamento observável de
       provedor.
- [ ] 12.11 Implementar a guarda de recursão de RF-67 **por construção**: profundidade de invocação
       propagada no envelope e verificada na entrada de cada hook, de forma que
       `hook → ferramenta → hook` seja impossível, não apenas desaconselhado. `TestRecursionGuard`
       cobrindo o caminho direto e o indireto.
- [ ] 12.12 Registrar os pacotes de teste novos em um job que o CI executa
       (`.github/workflows/test.yml`), sob pena de gate órfão.
- [ ] 12.13 Rodar os gates de não-regressão e anexar as saídas ao `execution_report.md`.

## Detalhes de Implementação

- Entradas de telemetria novas e o racional de preservar formato e opt-in: `techspec.md:550-571`.
  **Referenciar, não duplicar.**
- Rastreabilidade RF-45 a RF-50 e RF-64 a RF-67: `techspec.md:409-451`.
- Decisão de opt-in e append-only do repositório: `docs/adr/006-telemetria-feedback-cycle.md`.
- Semântica do resultado emitido em `hook.decision`:
  [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md).
- Envelope versionado que carrega `SessionID` e a profundidade de invocação: `techspec.md:190-215`.

## Critérios de Sucesso

- Schema comum declarado e validado, com extensões de provedor em espaço próprio; gate de
  vazamento verde (RF-47).
- `TestTelemetry_UnavailableIsUnknown` verde e teste adversarial de estimativa não identificada
  verde (RF-46).
- Formato `<ts> chave=valor` e opt-in `GOVERNANCE_TELEMETRY=1` **inalterados**: teste que prova
  no-op completo sem a variável, sem arquivo criado.
- Nenhuma dependência nova no `go.mod`; ausência de Prometheus e Grafana verificável no diff.
- `TestAblation_BaselineWithoutHook` verde, consumindo as primitivas já existentes em
  `internal/telemetry/ablation.go` — sem redefinir `ComponentKind`, `AblationDecision`,
  `AblationBaseline` nem `AblationResult`.
- Teste de RF-65 provando que hook de segurança crítica sobrevive a ablation `REMOVE`.
- Timeout declarado por hook, mensurável, tratado como negação; comportamento atual do OpenCode
  preservado bit a bit.
- `TestRecursionGuard` verde para recursão direta e indireta.
- Teste de RF-48 com escritor de telemetria sempre falhando: tarefa conclui, contagem de
  retentativas inalterada.
- **Não-regressão (inegociável):** `make test lint vet` verdes.
- `make coverage` respeitando 75% total e 70% por pacote crítico (`Makefile:42`, `Makefile:47`).

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

**A modificar**

- `internal/telemetry/parser.go:10-17,20-25,27-35` — contrato implícito a formalizar.
- `internal/telemetry/telemetry.go:13,23,56,66,88,98` — opt-in e escrita append-only a preservar.
- `.github/workflows/test.yml` — registro dos pacotes de teste novos.

**A consumir, sem recriar**

- `internal/telemetry/ablation.go:5,8-10,13,16-18,21,26,42,55`.
- `internal/hookcontract/` — envelope, `Result` e tradutor de exit code, entregues pela tarefa 4.0.

**A preservar, verificando paridade**

- `.opencode/plugin/governance.js:16,182,187,212,217,252` — único timeout existente.

**Documentos de referência**

- `.specs/prd-hooks-canonicos-vendor-neutral/prd.md:323-333,357-363` (RF-45 a RF-50, RF-64 a RF-67).
- `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md:190-215,409-451,550-571`.
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-002-resultado-tipado-traducao-exit-code.md`.
- `docs/adr/006-telemetria-feedback-cycle.md`.
