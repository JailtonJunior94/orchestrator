# Tarefa 3.0: Mapa 1:1 critério-evidência como dado verificável

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

**Esta tarefa é pré-requisito bloqueante da tarefa 5.0.** A ordem não é preferência de sequenciamento:
é a mitigação de um risco declarado na techspec e na ADR-001. Ligar o critério estrito de aprovação
antes que o mapa 1:1 exista como dado converte o falso positivo atual em **falso negativo total** —
todo ciclo terminaria bloqueado, porque `MapaDeCriterios.Completo()` nunca seria verdadeiro sobre um
mapa que ninguém produz.

Hoje o mapa não existe como dado, e isso é verificável:

- `.agents/skills/review/assets/review-report-template.md` tem as seções `## Achados`,
  `## Arquivos Revisados`, `## Riscos Residuais` e `## Validações Executadas` — e **nenhuma** seção de
  critérios de aceite.
- `.agents/scripts/validate-review-evidence.sh` não cobra tal seção, porque ela não existe no template.
- `internal/evidence/evidence.go` tem `validateTask`, `validateBugfix` e `validateRefactor` — **não tem
  rotina de revisão**. A paridade entre a verificação em Go e a verificação em shell está quebrada.

A tarefa fecha as quatro lacunas e propaga aos 4 espelhos. Uma restrição de implementação atravessa
tudo: as expressões de validação **não podem** usar classes de colchetes com caracteres multibyte —
`[áéíóú]` sob `LC_ALL=C` casa bytes individuais, não caracteres, e produz casamento errado silencioso.
Usar alternação (`(não atendido|nao atendido)`), e travar a invariante por caso de teste sob
`LC_ALL=C`, no mesmo padrão do caso "a2" já existente em `scripts/test-validators.sh:144-166`.

<requirements>
- RF-47: a revisão produz, para **toda** tarefa ativa, um mapa 1:1 entre cada critério de aceite e uma
  linha de evidência verificável. O confronto é **incondicional** — não há tarefa isenta.
- RF-51: os validadores canônicos verificam o mapa de forma **fail-closed**: tarefa não resolvível,
  ausência de seção de critérios ou mapa incompleto **falham** com exit code não-zero.
- RF-52: o validador Go de evidência ganha a rotina de revisão hoje inexistente, restaurando a paridade
  entre a verificação em Go e a verificação em shell.
- RF-53: o escape de compatibilidade legado `AI_SDD_STRICT_EVIDENCE` **não** cobre o mapa 1:1 nem o
  critério `APPROVED` estrito. Não existe caminho legítimo para fechar tarefa sem prova de aprovação.
- RF-54: expressões de validação sem classes de colchete multibyte; devem usar alternação; invariante
  travada por caso de teste sob locale de bytes.
- Formato da linha do mapa, literal:
  `- [atendido|não atendido|não verificável] <critério> -> <linha de evidência>`.
- A `<linha de evidência>` só é válida nas três formas de RF-48 (comando com saída registrada,
  `arquivo:linha` presente no diff revisado, nome de teste com resultado registrado).
- Propagação obrigatória aos 4 espelhos do template e aos 4 espelhos do validador; divergência é falha
  de gate própria.
- Nenhuma regressão nos casos já cobertos por `scripts/test-validators.sh` — em particular o caso "a2",
  que existe justamente para provar comportamento correto sob `LC_ALL=C`.
</requirements>

## Subtarefas

- [x] 3.1 Acrescentar a `.agents/skills/review/assets/review-report-template.md` a seção de mapa 1:1,
      com uma linha por critério no formato
      `- [atendido|não atendido|não verificável] <critério> -> <linha de evidência>`, e instrução
      explícita de que a seção é obrigatória para toda tarefa ativa.
- [x] 3.2 Propagar o template aos 3 espelhos: `.claude/skills/review/assets/review-report-template.md`,
      `.github/skills/review/assets/review-report-template.md` e
      `internal/embedded/assets/.agents/skills/review/assets/review-report-template.md`.
- [x] 3.3 Adicionar a `.agents/scripts/validate-review-evidence.sh` a asserção fail-closed do mapa:
      ausência da seção falha; seção presente com critério sem linha de evidência falha; critério
      marcado `não verificável` falha; linha de evidência fora das três formas de RF-48 falha.
- [x] 3.4 Escrever todas as expressões novas com **alternação**, jamais com classe de colchete contendo
      caractere multibyte, e validar cada uma sob `LC_ALL=C`.
- [x] 3.5 Propagar o validador aos 3 espelhos: `.claude/scripts/validate-review-evidence.sh`,
      `internal/embedded/assets/.agents/scripts/validate-review-evidence.sh` e
      `internal/embedded/assets/.claude/scripts/validate-review-evidence.sh`.
- [x] 3.6 Implementar em `internal/evidence/evidence.go` a rotina `validateReview`, ao lado de
      `validateTask` (`:73`), `validateBugfix` (`:122`) e `validateRefactor` (`:247`), com a mesma
      forma de retorno (`[]Finding`) e registrada no despacho por `ReportKind` de
      `Validator.Validate` (`:32`).
- [x] 3.7 Restringir o escape legado: em `.agents/scripts/validate-task-evidence.sh:27`,
      `strict_evidence="${AI_SDD_STRICT_EVIDENCE:-1}"` e o aviso de `:41` passam a **não** alcançar as
      asserções do mapa 1:1 nem o critério `APPROVED` estrito. `AI_SDD_STRICT_EVIDENCE=0` continua
      reabrindo apenas o escopo legado pré-existente. Propagar aos espelhos do validador de tarefa.
- [x] 3.8 Acrescentar casos a `scripts/test-validators.sh`, cada um executado também sob `LC_ALL=C`,
      espelhando a estrutura do caso "a2" (`:144-166`): mapa ausente falha; mapa incompleto falha;
      critério `não verificável` falha; linha de evidência inválida falha; mapa completo e válido passa;
      e — o caso decisivo — `AI_SDD_STRICT_EVIDENCE=0` **não** faz nenhum dos anteriores passar.
- [x] 3.9 Adicionar testes em Go para `validateReview` em `internal/evidence/`, com tabela cobrindo os
      mesmos cenários, garantindo a paridade Go↔shell exigida por RF-52.
- [x] 3.10 Rodar os gates de sincronia e registrar a evidência.

## Detalhes de Implementação

Seguir a techspec desta pasta, seção **"Sequenciamento de Desenvolvimento" → fase
`F2a — Mapa 1:1 (pré-requisito bloqueante)`**, que delimita o escopo em cinco itens: seção de critérios
no template, asserção nos validadores canônicos, rotina de revisão no validador em Go, restrição do
escape legado e propagação aos espelhos.

A subseção **"Sobre o posicionamento de F2a"** da mesma techspec explica por que a fase foi movida para
antes do Ciclo e por que a ordenação anterior contradizia a própria seção de Riscos Conhecidos.

A tabela **"Riscos Conhecidos"** da techspec registra o risco na íntegra: "Ligar o critério estrito sem
isso transforma falso positivo em **falso negativo total** — todo ciclo terminaria bloqueado", com a
mitigação declarada como pré-requisito bloqueante.

A ADR-001, seção **"Riscos e Mitigações"**, repete a mesma dependência dura e a classifica como
pré-requisito bloqueante da fase.

As três formas válidas de linha de evidência estão em RF-48 do `prd.md`. A regra de que critério "não
verificável pelo diff" proíbe `APPROVED` está em RF-49. A restrição sobre classes de colchete multibyte
está em RF-54.

O modelo de casamento da definição de completude que o domínio consumirá é `MapaDeCriterios.Completo()`,
especificado na techspec, subseção **"A invariante central, garantida por tipo"**: falso quando qualquer
critério está sem evidência **ou** declarado não verificável. O formato de dado entregue aqui é o insumo
desse método — as duas definições precisam ser a mesma.

## Critérios de Sucesso

- `grep -c 'não verificável' .agents/skills/review/assets/review-report-template.md` retorna `> 0` (hoje
  retorna `0`); os 4 arquivos de template contêm a seção nova, verificável por
  `for f in .agents .claude .github; do grep -q 'atendido' "$f/skills/review/assets/review-report-template.md" || echo "FALTA $f"; done`
  sem saída, mais o espelho embutido.
- `bash .agents/scripts/validate-review-evidence.sh <relatório sem seção de mapa>` sai com código `1`
  e a mensagem nomeia a seção ausente.
- `bash .agents/scripts/validate-review-evidence.sh <relatório com critério sem evidência>` sai `1`.
- `bash .agents/scripts/validate-review-evidence.sh <relatório com critério "não verificável">` sai `1`.
- `bash .agents/scripts/validate-review-evidence.sh <relatório completo e válido>` sai `0`.
- Cada um dos quatro comandos acima produz **exit code idêntico** quando reexecutado com prefixo
  `LC_ALL=C` — a prova de RF-54.
- `grep -nE '\[[^]]*[^\x00-\x7F][^]]*\]' .agents/scripts/validate-review-evidence.sh` não retorna
  nenhuma classe de colchete com byte não-ASCII.
- `AI_SDD_STRICT_EVIDENCE=0 bash .agents/scripts/validate-review-evidence.sh <relatório com mapa incompleto>`
  sai `1` e a saída **não** contém `gate de aceite ignorado` — prova direta de RF-53.
- `bash scripts/test-validators.sh` passa, e a sua saída nomeia os casos novos do mapa 1:1, inclusive as
  variantes sob `LC_ALL=C` e o caso do escape legado.
- `grep -n 'func (r1 \*Validator) validateReview' internal/evidence/evidence.go` retorna a rotina nova;
  `go test ./internal/evidence/... -count=1 -v` passa e nomeia os casos de mapa.
- Paridade Go↔shell comprovada: o mesmo relatório de entrada produz o mesmo veredito no validador shell
  e em `validateReview`, para os cinco cenários, registrado na evidência.
- `make check-skills-sync check-scripts-sync` verde — prova de que os 4 espelhos do template e os 4 do
  validador foram propagados.
- `make check-spec-paths` verde.
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

- [x] Testes unitários
- [x] Testes de integração

Cobertura obrigatória:

- `scripts/test-validators.sh`: casos novos para mapa ausente, mapa incompleto, critério não verificável,
  linha de evidência fora das três formas de RF-48, e mapa válido — **cada um duplicado sob `LC_ALL=C`**,
  no padrão do caso "a2" (`:144-166`).
- `scripts/test-validators.sh`: caso dedicado provando que `AI_SDD_STRICT_EVIDENCE=0` não reabre o mapa
  1:1, contrastando com o caso "d2" (`:247-248`), que documenta o escape ainda válido para o escopo
  legado.
- `internal/evidence/`: tabela de testes para `validateReview` cobrindo os mesmos cinco cenários.
- Integração: execução do validador shell e do validador Go sobre os mesmos artefatos de entrada,
  comparando vereditos — é o que comprova a paridade exigida por RF-52.
- Gates de sincronia dos 4 espelhos como teste de integração de propagação.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `.agents/skills/review/assets/review-report-template.md` — hoje sem seção de critérios; recebe o mapa.
- `.claude/skills/review/assets/review-report-template.md` — espelho.
- `.github/skills/review/assets/review-report-template.md` — espelho.
- `internal/embedded/assets/.agents/skills/review/assets/review-report-template.md` — espelho embutido.
- `.agents/scripts/validate-review-evidence.sh` — recebe a asserção fail-closed do mapa.
- `.claude/scripts/validate-review-evidence.sh` — espelho.
- `internal/embedded/assets/.agents/scripts/validate-review-evidence.sh` — espelho embutido.
- `internal/embedded/assets/.claude/scripts/validate-review-evidence.sh` — espelho embutido.
- `internal/evidence/evidence.go:32` — `Validator.Validate`, despacho por `ReportKind`; `:73`
  `validateTask`, `:122` `validateBugfix`, `:247` `validateRefactor` — a rotina de revisão é a ausente.
- `.agents/scripts/validate-task-evidence.sh:27` — `strict_evidence="${AI_SDD_STRICT_EVIDENCE:-1}"`;
  `:41` — aviso `gate de aceite ignorado`; ambos precisam deixar de alcançar o mapa 1:1.
- `internal/embedded/assets/.agents/scripts/validate-task-evidence.sh:27` e
  `internal/embedded/assets/.claude/scripts/validate-task-evidence.sh:27` — espelhos do escape legado.
- `scripts/test-validators.sh:144-166` — caso "a2", padrão a replicar para `LC_ALL=C`; `:247-248` —
  caso "d2", que documenta o escopo legado do escape.
- `.agents/skills/review/SKILL.md` — descreve o artefato de revisão; alinhar ao template novo.
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — fase `F2a`, "Sobre o posicionamento de
  F2a" e tabela "Riscos Conhecidos".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md` — "Riscos e
  Mitigações".
