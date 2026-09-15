# Tarefa 4.3: Extrator de critérios compartilhado e adaptador de portas em `internal/taskloop`

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Primeira das seis fatias que promovem o loop ao agregado (Bloco D do PRD / fase F2b da techspec;
re-fatiamento após 3 travas de design — ver techspec, subseção **"Integração do agregado nos três
caminhos (Bloco D)"**).

Esta fatia não migra nenhum call site. Ela entrega a infraestrutura comum: um pacote-folha novo
`internal/taskcriteria` para extração de critérios de aceite (consumível por `internal/taskloop` e
`internal/runtime`, que não podem se importar) e o adaptador das três portas do agregado em
`internal/taskloop/approval_adapters.go` sobre `FinalReviewer` / `BugfixInvoker` / captura de diff,
incluindo a evidência-sentinela de paridade que permite ao `CriteriaMap` alcançar `Completo()` antes
da tarefa 5.0.

<requirements>
- RF-34: cada rodada de revisão e de correção executa em sessão/subagente novo, recebendo apenas os
  achados e o delta. Contexto acumulado através de rodadas é proibido.
- `internal/taskcriteria.Extract(content []byte) []string` extrai os itens de checklist sob
  `## Definition of Done` / `## Critérios de Sucesso` / `## Acceptance Criteria` — mesma detecção de
  seção hoje em `internal/taskloop/acceptance.go:144-150`. Imports do pacote: apenas `bufio`/`strings`.
- `internal/taskloop/acceptance.go` (`parseCriteriaFromTaskFile`) passa a delegar ao novo pacote,
  preservando o cálculo de `missing` e a suíte existente verde.
- O adaptador de `internal/taskloop` implementa `approval.Reviewer` / `approval.Fixer` /
  `approval.Repository` sem reimplementar lógica de decisão de veredito — ela permanece em
  `internal/approval`. A porta `Reviewer` devolve **texto bruto**, nunca `ReviewVerdict`.
- A evidência-sentinela de paridade fica **confinada ao adaptador**: `Cycle` e `NewApprovalProof`
  permanecem inalterados. Documentar como shim time-boxed removido em 5.0.
- Nenhum call site (`taskloop.go`, `runloop.go`, `runner.go`) é modificado nesta fatia.
</requirements>

## Subtarefas

- [ ] 4.3.1 Criar `internal/taskcriteria/` com `Extract` e testes de tabela (seções reconhecidas,
      itens marcados e não marcados, ausência de seção, múltiplas seções equivalentes).
- [ ] 4.3.2 Refatorar `internal/taskloop/acceptance.go` para delegar a extração ao novo pacote;
      suíte de `internal/taskloop` verde sem ajuste de asserção.
- [ ] 4.3.3 Criar `internal/taskloop/approval_adapters.go`: `reviewerPort` sobre `FinalReviewer`
      (texto bruto de `FinalReviewResult.RawOutput`; `Finding` traduzido de `taskloop.Finding` para
      `approval.Finding` com `file`/`rule` default quando ausentes; `CriteriaMap` com
      evidência-sentinela de paridade), `fixerPort` sobre `BugfixInvoker` (devolve só `error`; extrai
      `Fail-before`/`Pass-after` para um `*bugfixEvidenceRecorder` side-channel; fail-closed sem
      marcadores) e `repositoryPort` (`CaptureDiff` + ponto de corte via `git rev-parse`).
- [ ] 4.3.4 Testes de contrato dos adaptadores: cada porta traduz a chamada e nada mais; nenhuma lê
      o campo `Verdict`.

## Detalhes de Implementação

Seguir a techspec desta pasta, fase **`F2b — Ciclo`** (ordem interna, itens 3), a subseção
**"Integração do agregado nos três caminhos (Bloco D)"** (D-B1, D-B1-corolário, D-B3) e a subseção
**"Interfaces Chave"** (três portas declaradas no consumidor; `Reviewer` devolve texto bruto).

## Critérios de Sucesso

- `go test ./internal/taskcriteria/... -count=1` verde.
- `grep -n 'approval\.' internal/taskloop/approval_adapters.go` retorna ocorrências.
- `internal/taskloop/approval_adapters.go` não contém lógica de decisão de veredito — só tradução de
  chamada; `grep -n 'Verdict' internal/taskloop/approval_adapters.go` não retorna leitura do campo.
- `git diff --stat` mostra apenas arquivos novos + delegação em `internal/taskloop/acceptance.go`;
  `runloop.go`, `taskloop.go`, `internal/runtime/runner.go` intocados.
- `go test ./internal/taskloop/... -count=1` verde sem ajuste de asserção.
- T-REV-01/02/04 verdes e sem alteração de asserção.
- Estilo R-STYLE-001 no código novo/tocado.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
- `go build ./... && go vet ./... && go test ./... -count=1` verde.
- `go test -tags=integration ./internal/taskloop/... -count=1` verde.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória:

- `taskcriteria.Extract`: seções reconhecidas, itens marcados/não marcados, ausência de seção.
- Adaptadores de `internal/taskloop`: cada porta traduz a chamada e nada mais; nenhuma lê `Verdict`.
- `fixerPort`: fail-closed quando o retorno do `BugfixInvoker` não traz `Fail-before`/`Pass-after`.
- Preservação: T-REV-01/02/04 verdes e não modificados.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/taskcriteria/` — pacote-folha novo.
- `internal/taskloop/acceptance.go:97,144-150` — extração hoje embutida; passa a delegar.
- `internal/taskloop/approval_adapters.go` — novo; adaptador das três portas.
- `internal/taskloop/reviewer.go` — `FinalReviewer`, `FinalReviewResult`, `taskloop.Finding`.
- `internal/taskloop/bugfix.go:159,186` — `extractBugfixEvidence`, `formatBugfixOrigin` reusados.
- `internal/approval/ports.go`, `internal/approval/evidence.go:43-50,140-148`, `internal/approval/finding.go:34`.
- `internal/runtime/approval_adapters.go` — referência dos adaptadores do caminho runtime.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Integração do agregado
  nos três caminhos (Bloco D)".
