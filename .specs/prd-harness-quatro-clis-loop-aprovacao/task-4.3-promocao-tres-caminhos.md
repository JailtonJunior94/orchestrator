# Tarefa 4.3: Promoção do loop nos três caminhos de produção e fechamento das quatro lacunas

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Terceira das três fatias que substituem a antiga tarefa 4.0.

Com o veredito já vindo da fonte real (4.1) e a evidência/delta/profundidade já corrigidos (4.2),
esta fatia **promove** o loop existente (`internal/taskloop/bugfix.go:83`) ao agregado
`internal/approval` e o faz conduzir os **três** caminhos de produção — sem os três, a afirmação de
comportamento idêntico entre agentes seria falsa:

- `ACPRunner` — `internal/runtime/runner.go:216-228` (hoje dispara `runAutoReview` uma vez).
- `Service.Execute` — `internal/taskloop/taskloop.go:244`, o **caminho real de produção**.
- `RunLoop` — `internal/taskloop/runloop.go`, delegando a `BugfixLoop` apenas o que o agregado não
  absorve.

Fecha as quatro lacunas do loop (RF-31) e entrega a prova de paridade entre os três caminhos.

<requirements>
- RF-31: o loop existente é **promovido** ao agregado, não substituído. As quatro lacunas fecham:
  ressalvas realimentam o ciclo, detecção de não-convergência por fingerprint repetida, aborto por
  ausência de mudança, exportação do ponto de corte por rodada.
- RF-34: cada rodada de revisão e de correção executa em **sessão/subagente novo**, recebendo apenas
  os achados e o delta. Contexto acumulado através de rodadas é **proibido**.
- Os **três** caminhos consomem o mesmo agregado, com veredito, motivo canônico e estrutura de
  evidência idênticos.
- **T-REV-01, T-REV-02, T-REV-04** verdes e não modificados; T-REV-03 conforme migrado em 4.1.
- As asserções de veredito em `internal/taskloop/*_test.go` são ajustadas **somente** onde
  codificam o parsing leniente antigo incompatível com o agregado fail-closed; cada ajuste é
  justificado por requisito no relatório. Asserções que já usam veredito canônico permanecem
  intocadas.
- A virada do critério estrito de encerramento **não** acontece aqui — é a tarefa 5.0. Esta fatia
  entrega o estágio em que o agregado reproduz o comportamento legado com a suíte existente verde
  (gate de não-regressão mais importante do plano — ADR-001).
- Estados terminais do Ciclo (não convergir, esgotar rodadas, não produzir mudança) são
  **resultados**, não erro de infraestrutura, e não acionam retry (RF-45, já modelado em 2.0;
  respeitado na fiação).
</requirements>

## Subtarefas

- [ ] 4.3.1 Migrar o caminho do runner ACP: `internal/runtime/runner.go:216-228` passa a conduzir o
      Ciclo em vez de disparar `runAutoReview` uma única vez.
- [ ] 4.3.2 Migrar `Service.Execute` (`internal/taskloop/taskloop.go:244`) ao mesmo agregado.
- [ ] 4.3.3 Migrar `RunLoop` (`internal/taskloop/runloop.go`) ao mesmo agregado, delegando a
      `BugfixLoop` apenas o que o agregado não absorve.
- [ ] 4.3.4 Fechar as quatro lacunas do loop: ressalvas realimentam a correção; não-convergência por
      fingerprint repetida; aborto por ausência de mudança; ponto de corte exportado por rodada.
- [ ] 4.3.5 Ajustar as asserções de veredito em `internal/taskloop/*_test.go` estritamente onde
      incompatíveis com o agregado fail-closed, com justificativa por requisito.
- [ ] 4.3.6 Escrever o teste de paridade entre `ACPRunner`, `Service.Execute` e `RunLoop`, mais os
      cinco fluxos E2E com o servidor ACP falso in-process.

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`** (promoção do loop; quatro lacunas), tabela
**"Componentes modificados"** (`internal/taskloop/bugfix.go`, `internal/runtime/runner.go`), tabela
**"Riscos Conhecidos"** ("O caminho de produção não é o que parecia" → "os três caminhos migram para
o agregado"), **"Abordagem de Testes" → "Testes E2E"** (servidor ACP falso in-process) e a ADR-001
"Plano de Implementação" itens 4–7 (item 4: estágio de paridade).

## Critérios de Sucesso

- `grep -rn 'approval\.' internal/runtime/runner.go internal/taskloop/taskloop.go internal/taskloop/runloop.go`
  retorna ocorrências nos três arquivos.
- Teste de paridade: o mesmo cenário de entrada produz veredito, motivo canônico e estrutura de
  evidência idênticos nos três caminhos.
- E2E com o servidor ACP falso: ciclo que aprova na primeira rodada; ciclo que aprova na terceira
  após duas correções; ciclo que aborta por fingerprint repetida **sem gastar a rodada seguinte**;
  ciclo que aborta por ausência de mudança; ciclo cuja remediação não produz diff. Todos verdes.
- Ressalvas (`APPROVED_WITH_REMARKS`) realimentam a correção — teste dedicado.
- `go test ./internal/runtime/ -run TestAutoReview -v -count=1` verde; T-REV-01/02/04 sem alteração
  de asserção; `git diff internal/taskloop/*_test.go` mostra apenas ajustes justificados por
  requisito.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/runtime/... ./internal/taskloop/... -count=1` verde.
- Golden files de governança byte-idênticos.

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

Cobertura obrigatória:

- **Lacunas do loop**: ressalvas realimentam; fingerprint repetida aborta; ausência de mudança
  aborta; ponto de corte exportado por rodada.
- **Paridade**: `ACPRunner`, `Service.Execute` e `RunLoop` produzem veredito, motivo e evidência
  idênticos para a mesma entrada.
- **Preservação**: T-REV-01, T-REV-02, T-REV-04 verdes e não modificados.
- **Integração / E2E**: os cinco fluxos com o servidor ACP falso in-process, mais o teste de
  paridade entre os três caminhos.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/runner.go:216-228` — ponto de chamada de `runAutoReview`; passa a conduzir o
  Ciclo.
- `internal/taskloop/taskloop.go:244` — `Service.Execute`, o caminho **real** de produção.
- `internal/taskloop/runloop.go` — `RunLoop`, terceiro caminho.
- `internal/taskloop/bugfix.go:83` — `BugfixLoop.Run`, o loop promovido ao agregado.
- `internal/taskloop/reviewer.go`, `internal/taskloop/acpinvoker.go`, `internal/taskloop/acceptance.go`.
- `internal/taskloop/*_test.go` — asserções de veredito a ajustar cirurgicamente.
- `internal/approval/` — agregado e portas.
- `internal/runtime/runner_autoreview.go` — estado pós-4.1/4.2.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Componentes
  modificados", "Riscos Conhecidos", "Testes E2E".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md` — "Plano de
  Implementação", itens 4–7.
