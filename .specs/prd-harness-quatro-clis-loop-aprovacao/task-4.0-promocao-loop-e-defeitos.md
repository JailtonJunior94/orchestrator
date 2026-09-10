# Tarefa 4.0: Promoção do loop ao agregado e correção dos defeitos de causa-raiz

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

O agregado da tarefa 2.0 existe sem consumidor e o mapa 1:1 da tarefa 3.0 existe como dado. Esta tarefa
liga os dois ao código de produção, **promovendo** o loop existente em vez de substituí-lo (RF-31), e
corrige os dois defeitos de causa-raiz sem os quais toda a entrega descreveria comportamento que o
código não tem.

**Defeito D1 — o veredito de produção é ficção.**
`internal/runtime/runner_autoreview.go:233`, `buildReviewOutputFromSummary`, devolve string sintética
derivada do `CancelReason` do `Summary`: `"erro na sessão de review: %v"`, `"sessão de review encerrada
com: %s"` ou `"review concluído sem marcadores hard"`. É essa string — não a saída real do revisor — que
alimenta `parseReviewStatus` e `extractHardIssues` em `:183-184`. Nenhuma das três contém marcador de
bloqueio, logo o veredito de produção é `ok` sempre que a sessão não erra. A função é **excluída**, não
ajustada: enquanto ela existir, existe um caminho que fabrica veredito. A saída real do revisor passa a
ser capturada da sessão e entregue ao tradutor fail-closed do agregado.

**Defeito D2 — a evidência da rodada é destruída.**
`internal/runtime/runner_autoreview.go:186-189` monta `buildReviewPointer` — um apontador de três linhas
(`# Auto-Review`, `ReviewStatus:`, `Relatório completo:`) — e o escreve em `reviewPath` com
`os.WriteFile`, **sobrescrevendo** o `review.md` produzido pela skill. Com evidência por rodada (RF-42),
esse defeito destruiria o histórico inteiro do Ciclo. O endereço de escrita passa a derivar do número da
rodada, e a escrita passa a ser **exclusiva**: reescrever endereço existente é erro, não sobrescrita
silenciosa.

Mais três correções de causa-raiz na mesma fatia, porque todas vivem no caminho que está sendo migrado:

- **`AI_REVIEW_PRIOR_SHA` é contrato órfão.** `.agents/skills/review/SKILL.md:16` e `:70` o especificam,
  e nenhum código Go, hook ou script o exporta. A revisão incremental por delta (RF-39) nunca funcionou.
  Passa a ser exportado por rodada, com o ponto de corte vindo da porta `Repositorio`.
- **O guarda de profundidade quebra a rodada 1.** `internal/invocation/invocation.go:12` fixa
  `_defaultMax = 2` e a etapa de execução já consome um nível; a correção falha de forma dura. Cada
  rodada abre com a profundidade **resetada** (RF-38), replicando o precedente já validado no
  orquestrador de PRD. O Ciclo é iterativo no mesmo nível, **não recursivo**.
- **Três caminhos, não um.** A techspec é explícita: o repositório tem duas implementações divergentes
  do mesmo ciclo (`internal/taskloop/bugfix.go:83` e `internal/runtime/runner.go:216-228`), e o loop
  existente só é alcançável por função **sem chamador de produção**. O caminho real de produção é
  `internal/taskloop/taskloop.go:244` (`Service.Execute`). Migrar apenas o runner ACP tornaria falsa a
  afirmação de comportamento idêntico entre agentes.

<requirements>
- RF-31: o loop existente é **promovido** ao agregado, não substituído. Suas quatro lacunas fecham:
  ressalvas realimentam o ciclo, detecção de não-convergência, aborto por ausência de mudança e
  exportação do ponto de corte por rodada.
- RF-34: cada rodada de revisão e de correção executa em **sessão/subagente novo**, recebendo apenas os
  achados e o delta. Contexto acumulado através de rodadas é **proibido** — é o vetor direto de
  aprovação alucinada.
- RF-38: iterativo no mesmo nível de invocação, não recursivo; profundidade resetada a cada rodada.
- RF-39: a revisão da rodada N>1 opera **somente sobre o delta**, exportando o ponto de corte pela
  variável `AI_REVIEW_PRIOR_SHA`.
- RF-40: a revisão é sempre a skill `review` do harness, idêntica nos quatro agentes; capacidades
  nativas de revisão dos CLIs **não** são usadas.
- RF-42: cada rodada persiste evidência própria, numerada e imutável; o endereço deriva do número da
  rodada; reescrever endereço existente é erro.
- RF-44: recursão permanece hard-bloqueada — a sessão de revisão disparada pelo Ciclo nunca dispara um
  novo Ciclo dentro de si.
- RF-57: a revisão passa a produzir e consumir a **saída real do revisor**, eliminando o texto sintético.
- RF-58: a escrita do artefato de revisão pelo Go deixa de sobrescrever o relatório produzido pela skill.
- Os **três** caminhos consomem o mesmo agregado: `ACPRunner` (`internal/runtime/runner.go:216-228`),
  `Service.Execute` (`internal/taskloop/taskloop.go:244`) e `RunLoop`.
- **T-REV-01 a T-REV-04** de `internal/runtime/runner_autoreview_test.go` são preservados. Qualquer
  alteração neles entra com justificativa por requisito, conforme a mitigação de "virada de comportamento
  pode mascarar regressão" da ADR-001.
- A virada do critério estrito de encerramento **não** acontece aqui — é a tarefa 5.0. Esta tarefa
  entrega o estágio em que o agregado reproduz o comportamento legado com a suíte existente verde.
- Linha de base gravada **antes** da mudança de D1, sob pena de não se distinguir "gate funcionando" de
  "gate quebrado" quando a taxa de aprovação cair.
</requirements>

## Subtarefas

- [ ] 4.1 Gravar a linha de base: executar a suíte de auto-review e o fluxo de taskloop no estado atual
      e registrar a distribuição de `ReviewStatus` observada. Sem isso, o efeito de D1 é indistinguível
      de regressão.
- [ ] 4.2 Implementar os adaptadores das três portas do agregado: `Revisor` (sessão de revisão real),
      `Corretor` (skill `bugfix`) e `Repositorio` (ponto de corte, alvo completo, delta) em
      `internal/runtime/` e `internal/taskloop/` (planejados).
- [ ] 4.3 **Excluir** `buildReviewOutputFromSummary` de `internal/runtime/runner_autoreview.go:233` e
      todos os seus chamadores; `spawnReviewSession` passa a devolver a saída textual **real** da sessão
      de revisão, capturada dos eventos persistidos.
- [ ] 4.4 Substituir `parseReviewStatus`/`extractHardIssues` sobre texto sintético (`:183-184`) pela
      tradução fail-closed do agregado, alimentada pela saída real.
- [ ] 4.5 Corrigir D2: remover a sobrescrita de `reviewPath` por `buildReviewPointer` (`:186-189`);
      o endereço de evidência passa a derivar do número da rodada e a escrita passa a ser exclusiva
      (`O_CREATE|O_EXCL`), com erro explícito em endereço já existente.
- [ ] 4.6 Exportar `AI_REVIEW_PRIOR_SHA` por rodada no ambiente da sessão de revisão, com o valor vindo
      de `Repositorio.PontoDeCorte`, honrando o contrato de `.agents/skills/review/SKILL.md:16`.
- [ ] 4.7 Resetar a profundidade de invocação a cada rodada, no padrão já validado no orquestrador de
      PRD, sem elevar `_defaultMax` de `internal/invocation/invocation.go:12`. Preservar o bloqueio duro
      de recursão (RF-44): o `Job` filho da revisão continua com auto-review desligado.
- [ ] 4.8 Migrar o caminho do runner ACP: `internal/runtime/runner.go:216-228` passa a conduzir o Ciclo
      em vez de disparar `runAutoReview` uma única vez.
- [ ] 4.9 Migrar `Service.Execute` (`internal/taskloop/taskloop.go:244`) — **o caminho real de
      produção** — ao mesmo agregado.
- [ ] 4.10 Migrar `RunLoop` ao mesmo agregado, delegando a `BugfixLoop` (`internal/taskloop/bugfix.go:83`)
      apenas o que o agregado não absorve.
- [ ] 4.11 Fechar as quatro lacunas do loop: ressalvas realimentam a correção; não-convergência detectada
      por fingerprint repetida; aborto por ausência de mudança; ponto de corte exportado por rodada.
- [ ] 4.12 Verificar T-REV-01..04 intocados e verdes; escrever os testes novos de D1, D2, delta,
      profundidade e paridade entre os três caminhos.

## Detalhes de Implementação

Seguir a techspec desta pasta, seção **"Sequenciamento de Desenvolvimento" → fase `F2b — Ciclo`**, que
delimita o escopo: "promoção do loop existente; correção dos defeitos de veredito sintético e de
sobrescrita de evidência; evidência numerada por rodada; reset de profundidade; revisão por delta".

A tabela **"Componentes modificados"** da techspec nomeia as três mudanças estruturais desta fatia:
`internal/taskloop/bugfix.go` ("o loop é promovido: passa a delegar ao agregado, fechando as quatro
lacunas"), `internal/runtime/runner_autoreview.go` ("deixa de sintetizar o texto do revisor e de
sobrescrever a evidência; passa a alimentar o agregado com a saída real") e `internal/runtime/runner.go`
("o ponto de chamada da revisão passa a conduzir o Ciclo").

A justificativa de desenho para D1 está na subseção **"Interfaces Chave"**, no comentário normativo da
porta `Revisor`: "devolve TEXTO BRUTO, nunca Veredito. Se o adaptador pudesse devolver Veredito,
reintroduziria o defeito atual (runner_autoreview.go:233)". A correção não é apenas remover a função —
é tornar impossível reintroduzi-la pela assinatura da porta.

O **"Resumo Executivo"** da techspec classifica D1 como o defeito de maior impacto: "tornando todo
veredito de produção ficção".

A tabela **"Riscos Conhecidos"** registra "O caminho de produção não é o que parecia" com a mitigação
"os três caminhos migram para o agregado. Sem isso a paridade declarada seria falsa", e registra
"Critérios de aceite não chegam ao caminho ACP" com a mitigação "o campo de nome do arquivo de tarefa já
existe no job e é o gancho natural para o plumbing".

A ADR-001, **"Plano de Implementação"**, posiciona esta tarefa nos itens 4 a 7, com destaque para o item
4: "estágio de paridade: o loop existente delega ao agregado preservando o comportamento legado, com a
suíte atual verde e sem alteração — é o gate de não-regressão mais importante do plano".

A ADR-001, **"Contexto"**, documenta que "o guarda de profundidade de invocação tem limite dois, a etapa
de execução já consome um, e a correção falha de forma dura — de modo que nenhum ciclo passa da primeira
rodada hoje", e que "cada rodada executa em sessão nova, com a profundidade de invocação resetada".

A cobertura E2E exigida está em **"Abordagem de Testes" → "Testes E2E"**, com o servidor ACP falso
in-process já existente no repositório.

## Critérios de Sucesso

- `grep -c buildReviewOutputFromSummary internal/runtime/` sobre todo o pacote retorna `0` — a função e
  todos os seus chamadores foram excluídos, não apenas contornados.
- `grep -rn 'buildReviewPointer' internal/runtime/` não retorna nenhuma escrita sobre o `review.md`
  produzido pela skill.
- Teste dedicado a D1: dado um revisor cuja saída real contém veredito de bloqueio e um `Summary` com
  `CancelReason` vazio, `ReviewStatus` resulta em bloqueado. Sob o código atual esse teste falha; sob o
  código corrigido passa. A demonstração da falha no estado anterior é registrada na evidência.
- Teste dedicado a D2: duas rodadas gravam evidências em endereços distintos, **ambas legíveis ao final**;
  tentar escrever em endereço já existente retorna erro. Verificável por
  `go test ./internal/runtime/ -run TestEvidenciaPorRodada -v -count=1`.
- `AI_REVIEW_PRIOR_SHA` presente no ambiente da sessão de revisão da rodada N>1 e ausente na rodada 1,
  com o valor igual ao ponto de corte devolvido por `Repositorio.PontoDeCorte` — verificável por teste
  que inspeciona o ambiente passado ao subprocesso.
- Rodada 1 **completa** com o guarda de profundidade em `_defaultMax = 2` inalterado
  (`grep -n '_defaultMax = 2' internal/invocation/invocation.go` continua retornando a linha 12), provado
  por teste E2E de ciclo que aprova na terceira rodada após duas correções.
- RF-44 preservado: teste comprova que o `Job` filho da sessão de revisão não abre novo Ciclo.
- **T-REV-01, T-REV-02, T-REV-03 e T-REV-04 verdes e não modificados**:
  `go test ./internal/runtime/ -run TestAutoReview -v -count=1` passa, e
  `git diff --stat internal/runtime/runner_autoreview_test.go` mostra apenas adições de casos novos, sem
  alteração das quatro asserções existentes.
- Os três caminhos consomem o agregado, verificável por
  `grep -rn 'approval\.' internal/runtime/runner.go internal/taskloop/taskloop.go internal/taskloop/runloop.go`
  retornando ocorrências nos três arquivos.
- Teste de paridade: o mesmo cenário de entrada produz veredito, motivo canônico e estrutura de evidência
  idênticos nos três caminhos.
- E2E com o servidor ACP falso: ciclo que aprova na primeira rodada; ciclo que aprova na terceira após
  duas correções; ciclo que aborta por fingerprint repetida **sem gastar a rodada seguinte**; ciclo que
  aborta por ausência de mudança. Todos verdes.
- Linha de base do passo 4.1 anexada à evidência, com a comparação antes/depois da distribuição de
  `ReviewStatus` e justificativa de cada divergência.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.

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

- **D1**: saída real com veredito de bloqueio produz bloqueio, mesmo com `CancelReason` vazio; nenhum
  caminho de código fabrica texto de revisão.
- **D2**: evidências de rodadas distintas coexistem; escrita em endereço existente é erro.
- **Delta**: `AI_REVIEW_PRIOR_SHA` exportado apenas em N>1 e com o ponto de corte correto.
- **Profundidade**: ciclo de três rodadas completa sem elevar `_defaultMax`; recursão continua bloqueada.
- **Lacunas do loop**: ressalvas realimentam; fingerprint repetida aborta; ausência de mudança aborta.
- **Preservação**: T-REV-01..04 verdes e não modificados.
- **Integração / E2E**: os cinco fluxos com o servidor ACP falso in-process, mais o teste de paridade
  entre `ACPRunner`, `Service.Execute` e `RunLoop` — é a única prova de que os três caminhos têm o mesmo
  comportamento, e a techspec registra que o repositório já sofreu caso de unitário verde com integração
  real falhando.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/runner_autoreview.go:233` — `buildReviewOutputFromSummary`, D1: texto sintético que
  alimenta o parser. **Excluída** nesta tarefa.
- `internal/runtime/runner_autoreview.go:183-184` — `parseReviewStatus` e `extractHardIssues` sobre a
  saída sintética; passam a operar sobre a saída real via tradutor do agregado.
- `internal/runtime/runner_autoreview.go:186-189` — `buildReviewPointer` + `os.WriteFile` sobre
  `reviewPath`, D2: sobrescreve o `review.md` da skill com apontador de 3 linhas.
- `internal/runtime/runner_autoreview.go:215-228` — `spawnReviewSession`, que passa a devolver a saída
  textual real da sessão.
- `internal/runtime/runner_autoreview_test.go` — T-REV-01 (`:101-145`), T-REV-02 (`:148-188`),
  T-REV-03 (`:190-226`), T-REV-04 (`:228-272`): preservados.
- `internal/runtime/runner.go:216-228` — ponto de chamada de `runAutoReview`; passa a conduzir o Ciclo.
- `internal/taskloop/taskloop.go:244` — `Service.Execute`, o caminho **real** de produção.
- `internal/taskloop/runloop.go` — `RunLoop`, terceiro caminho a migrar.
- `internal/taskloop/bugfix.go:83` — `BugfixLoop.Run`, o loop promovido ao agregado.
- `internal/taskloop/reviewer.go`, `internal/taskloop/acpinvoker.go`, `internal/taskloop/acceptance.go` —
  adaptadores e extração de critérios de aceite.
- `internal/invocation/invocation.go:12` — `_defaultMax = 2`; `:25` `CheckDepth`; `:36` `IncrementDepth`.
  O reset por rodada não altera o limite.
- `.agents/skills/review/SKILL.md:16` e `:70` — contrato órfão de `AI_REVIEW_PRIOR_SHA`, mais os 3
  espelhos em `.claude/skills/`, `.github/skills/` e `internal/embedded/assets/.agents/skills/`.
- `internal/approval/` — agregado e portas consumidos aqui (entregues pela tarefa 2.0).
- `internal/runtime/summary.go`, `internal/runtime/types.go`, `internal/runtime/options.go` — campos de
  resultado e opções do Ciclo.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Componentes modificados",
  "Interfaces Chave", "Riscos Conhecidos" e "Testes E2E".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md` — "Contexto" e
  "Plano de Implementação", itens 4 a 7.
