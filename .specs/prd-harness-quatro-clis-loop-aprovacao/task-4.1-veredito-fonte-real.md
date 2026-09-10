# Tarefa 4.1: Veredito da fonte real do revisor e adaptadores das portas (defeito D1)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Primeira das três fatias que substituem a antiga tarefa 4.0 (decomposta por raio de explosão
subestimado — ver `## Riscos de Integração` em `tasks.md`).

Liga o agregado da tarefa 2.0 (`internal/approval`) ao caminho de auto-review do runtime e corrige o
**defeito D1**: `internal/runtime/runner_autoreview.go:233` (`buildReviewOutputFromSummary`) devolve
string sintética derivada do `CancelReason` do `Summary`, e é essa string — não a saída real do
revisor — que alimenta `parseReviewStatus`/`extractHardIssues`. Enquanto essa função existir, há um
caminho que fabrica veredito. Ela é **excluída**, não ajustada. A saída real do revisor passa a ser
capturada da sessão e entregue ao tradutor fail-closed do agregado.

Escopo desta fatia: **apenas o caminho runtime** (`internal/runtime/`) e os adaptadores das três
portas. A migração dos três caminhos de produção e o fechamento das quatro lacunas do loop ficam em
4.3; a evidência por rodada, o delta e o reset de profundidade ficam em 4.2.

<requirements>
- RF-57: a revisão passa a produzir e consumir a **saída real do revisor**, eliminando o texto
  sintético. `buildReviewOutputFromSummary` e todos os seus chamadores são excluídos.
- RF-40: a revisão é sempre a skill `review` do harness; capacidades nativas de revisão dos CLIs não
  são usadas.
- RF-46 (entregue por 2.0/3.0, consumido aqui): a tradução texto→veredito é fail-closed — saída sem
  linha de veredito canônica resolve para bloqueado. É **proibido** inferir aprovação pela ausência
  de marcador negativo.
- A porta `Revisor` devolve **TEXTO BRUTO**, nunca `Veredito` — a assinatura torna impossível
  reintroduzir D1 (techspec, "Interfaces Chave").
- Linha de base da distribuição de `ReviewStatus` gravada **antes** da mudança de D1, anexada à
  evidência, sob pena de não se distinguir "gate funcionando" de "gate quebrado".
- **T-REV-01, T-REV-02, T-REV-04** verdes. **T-REV-03** é migrado (não removido): seu fixture
  "aprovado" passa a ser saída real de review com linha de veredito canônica `APPROVED`, mantendo a
  asserção `ReviewStatus == "ok"`; a mudança é registrada por requisito (RF-46/RF-57) na seção
  "Conflitos de Regra" do relatório, conforme a mitigação de "virada de comportamento pode mascarar
  regressão" da ADR-001. Justificativa: a asserção original de T-REV-03 ("texto limpo sem veredito →
  ok") codifica exatamente o comportamento que RF-46 proíbe.
- A virada do critério estrito de encerramento **não** acontece aqui — é a tarefa 5.0.
</requirements>

## Subtarefas

- [ ] 4.1.1 Gravar a linha de base: executar a suíte de auto-review e o fluxo de taskloop no estado
      atual (`fda71f1`) e registrar a distribuição de `ReviewStatus` observada.
- [ ] 4.1.2 Implementar os adaptadores das três portas do agregado: `Reviewer` (sessão de revisão
      real), `Fixer` (skill `bugfix`) e `Repository` (ponto de corte, alvo completo, delta) em
      `internal/runtime/` (e stubs em `internal/taskloop/` só se necessário para compilar; a fiação
      completa do taskloop é 4.3).
- [ ] 4.1.3 **Excluir** `buildReviewOutputFromSummary` e todos os seus chamadores;
      `spawnReviewSession` passa a devolver a saída textual **real** da sessão de revisão, capturada
      dos eventos persistidos.
- [ ] 4.1.4 Substituir `parseReviewStatus`/`extractHardIssues` sobre texto sintético pela tradução
      fail-closed do agregado, alimentada pela saída real.
- [ ] 4.1.5 Migrar o fixture de T-REV-03 para o novo contrato; adicionar caso novo: saída de review
      sem linha de veredito canônica → `ReviewStatus` bloqueado (prova RF-46).

## Detalhes de Implementação

Seguir a techspec desta pasta, seção **"Sequenciamento de Desenvolvimento" → fase `F2b — Ciclo`**
(parte de veredito sintético) e a subseção **"Interfaces Chave"** (comentário normativo da porta
`Reviewer`: "devolve TEXTO BRUTO, nunca Veredito"). O **"Resumo Executivo"** classifica D1 como o
defeito de maior impacto. A ADR-001 "Plano de Implementação" posiciona esta fatia nos itens 4–5.

## Critérios de Sucesso

- `grep -rc buildReviewOutputFromSummary internal/` retorna `0` em todos os arquivos — função e
  chamadores excluídos.
- Teste dedicado a D1: dado revisor cuja saída real contém veredito de bloqueio e `Summary` com
  `CancelReason` vazio, `ReviewStatus` resulta em bloqueado; sob o código atual o teste falha, sob o
  corrigido passa; a falha no estado anterior é registrada na evidência.
- Teste: saída de review sem veredito canônico → `ReviewStatus` bloqueado (fail-closed, RF-46).
- `go test ./internal/runtime/ -run TestAutoReview -v -count=1` verde;
  `git diff --stat internal/runtime/runner_autoreview_test.go` mostra adições de casos novos e a
  migração documentada do fixture de T-REV-03 — nenhuma outra asserção alterada.
- Linha de base do passo 4.1.1 anexada à evidência com a distribuição de `ReviewStatus` antes da
  mudança.
- Estilo R-STYLE-001 no código novo/tocado: inglês, zero comentários, sem prefixo `_` em
  identificador novo ou cuja linha de declaração for alterada.
- `make check-spec-paths check-skills-sync check-scripts-sync` verde.
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

- **D1**: saída real com veredito de bloqueio produz bloqueio, mesmo com `CancelReason` vazio;
  nenhum caminho de código fabrica texto de revisão.
- **Fail-closed**: saída sem veredito canônico → bloqueado.
- **Preservação**: T-REV-01, T-REV-02, T-REV-04 verdes e não modificados; T-REV-03 com fixture
  migrado e asserção de status preservada.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/runner_autoreview.go:233` — `buildReviewOutputFromSummary` (D1). **Excluída.**
- `internal/runtime/runner_autoreview.go:183-184` — `parseReviewStatus`/`extractHardIssues`; passam a
  operar sobre a saída real via tradutor do agregado.
- `internal/runtime/runner_autoreview.go:215-228` — `spawnReviewSession`; passa a devolver a saída
  textual real.
- `internal/runtime/runner_autoreview_test.go` — T-REV-01..04.
- `internal/approval/` — agregado e portas (tarefa 2.0); `translator.go` fail-closed.
- `internal/runtime/summary.go`, `internal/runtime/types.go`, `internal/runtime/options.go`.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2b`, "Interfaces Chave".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md`.
