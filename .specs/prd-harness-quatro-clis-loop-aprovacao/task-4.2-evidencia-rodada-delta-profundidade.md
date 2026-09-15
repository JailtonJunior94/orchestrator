# Tarefa 4.2: Evidência por rodada, revisão por delta e reset de profundidade (defeito D2 + contratos órfãos)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Segunda das três fatias que substituem a antiga tarefa 4.0.

Corrige o **defeito D2** e dois contratos órfãos, todos no caminho que a 4.1 começou a migrar:

- **D2 — a evidência da rodada é destruída.** `internal/runtime/runner_autoreview.go:186-189` monta
  `buildReviewPointer` (apontador de 3 linhas) e o escreve em `reviewPath` com `os.WriteFile`,
  **sobrescrevendo** o `review.md` da skill. Com evidência por rodada (RF-42), isso destruiria o
  histórico do Ciclo. O endereço de escrita passa a derivar do número da rodada e a escrita passa a
  ser **exclusiva** (`O_CREATE|O_EXCL`): reescrever endereço existente é erro.
- **`AI_REVIEW_PRIOR_SHA` é contrato órfão.** `.agents/skills/review/SKILL.md:16` e `:70` o
  especificam; nenhum código Go, hook ou script o exporta (RF-39 nunca funcionou). Passa a ser
  exportado por rodada, com o ponto de corte vindo da porta `Repository`.
- **O guarda de profundidade quebra a rodada 1.** `internal/invocation/invocation.go:12` fixa
  `_defaultMax = 2` e a etapa de execução já consome um nível. Cada rodada abre com a profundidade
  **resetada** (RF-38), replicando o precedente do orquestrador de PRD. O Ciclo é iterativo no mesmo
  nível, **não recursivo**; a recursão permanece hard-bloqueada (RF-44).

Escopo: `internal/runtime/` e os espelhos da skill `review`. A fiação dos três caminhos e as quatro
lacunas do loop ficam em 4.3.

<requirements>
- RF-58: a escrita do artefato de revisão pelo Go deixa de sobrescrever o relatório da skill.
- RF-42: cada rodada persiste evidência própria, numerada e imutável; o endereço deriva do número da
  rodada; reescrever endereço existente é erro (`O_CREATE|O_EXCL`).
- RF-39: a revisão da rodada N>1 opera somente sobre o delta, exportando o ponto de corte pela
  variável `AI_REVIEW_PRIOR_SHA`; ausente na rodada 1.
- RF-38: iterativo no mesmo nível de invocação, não recursivo; profundidade resetada a cada rodada,
  sem elevar `_defaultMax`.
- RF-44: recursão hard-bloqueada — o `Job` filho da sessão de revisão continua com auto-review
  desligado e nunca abre novo Ciclo.
- O contrato de `.agents/skills/review/SKILL.md:16` e `:70` é honrado; os 3 espelhos
  (`.claude/skills/`, `.github/skills/`, `internal/embedded/assets/.agents/skills/`) permanecem
  sincronizados.
- A virada do critério estrito **não** acontece aqui — é a tarefa 5.0.
</requirements>

## Subtarefas

- [ ] 4.2.1 Corrigir D2: remover a sobrescrita de `reviewPath` por `buildReviewPointer`; o endereço
      de evidência passa a derivar do número da rodada; escrita exclusiva com erro explícito em
      endereço existente.
- [ ] 4.2.2 Exportar `AI_REVIEW_PRIOR_SHA` por rodada no ambiente da sessão de revisão, com o valor
      vindo de `Repository.CutPoint` (nome em inglês conforme R-STYLE-001), honrando o contrato da
      skill `review`.
- [ ] 4.2.3 Resetar a profundidade de invocação a cada rodada no padrão do orquestrador de PRD, sem
      elevar `_defaultMax`. Preservar o bloqueio duro de recursão (RF-44).

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`** (partes de sobrescrita de evidência, evidência
numerada por rodada, reset de profundidade, revisão por delta), tabela **"Componentes modificados"**
(`internal/runtime/runner_autoreview.go`) e a ADR-001 "Contexto" (guarda de profundidade) e "Plano de
Implementação" itens 6–7.

## Critérios de Sucesso

- `grep -rn 'buildReviewPointer' internal/runtime/` não retorna escrita sobre o `review.md` da skill.
- Teste D2: duas rodadas gravam evidências em endereços distintos, ambas legíveis ao final; escrever
  em endereço já existente retorna erro. `go test ./internal/runtime/ -run TestEvidence -v -count=1`.
- `AI_REVIEW_PRIOR_SHA` presente no ambiente da sessão de revisão da rodada N>1 e ausente na rodada
  1, valor igual ao ponto de corte de `Repository.CutPoint` — verificável por teste que inspeciona o
  ambiente passado ao subprocesso.
- Rodada 1 completa com `grep -n '_defaultMax = 2' internal/invocation/invocation.go` ainda na linha
  12 — provado por teste E2E de ciclo que aprova na terceira rodada após duas correções.
- RF-44: teste comprova que o `Job` filho da sessão de revisão não abre novo Ciclo.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde (espelhos da skill `review` em
  sync).
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/runtime/... -count=1` verde.

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

- **D2**: evidências de rodadas distintas coexistem; escrita em endereço existente é erro.
- **Delta**: `AI_REVIEW_PRIOR_SHA` exportado apenas em N>1 e com o ponto de corte correto.
- **Profundidade**: ciclo de três rodadas completa sem elevar `_defaultMax`; recursão continua
  bloqueada.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/runner_autoreview.go:186-189` — `buildReviewPointer` + `os.WriteFile` (D2).
- `internal/invocation/invocation.go:12` — `_defaultMax = 2`; `:25` `CheckDepth`; `:36`
  `IncrementDepth`. O reset por rodada não altera o limite.
- `.agents/skills/review/SKILL.md:16` e `:70` — contrato de `AI_REVIEW_PRIOR_SHA`, mais os 3
  espelhos.
- `internal/approval/` — porta `Repository` (ponto de corte).
- `internal/runtime/runner_autoreview.go`, `internal/runtime/summary.go`.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Componentes
  modificados".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md`.
