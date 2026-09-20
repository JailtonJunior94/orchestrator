# Tarefa 3.0: Instrumentacao de pre-requisito: tmp em gates, exaustividade de linguagens e prova por celula

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Instalar os três gates que precisam **nascer vermelhos** antes do trabalho que protegem. Cobre
**RF-59**. Sem dependências — paralelizável com 1.0 e 2.0 — e é pré-requisito de 7.0, 9.0 e 11.0.

As três frentes são heterogêneas em objeto mas idênticas em propriedade: cada uma é um gate que
precisa falhar **antes** de existir o código que ele vai proteger. Instalar depois é instalar um gate
que nasce verde e nunca provou nada.

**Postura explícita desta tarefa:** ficar vermelho ao final é o **resultado esperado** para as
frentes 2 e 3. A tarefa fecha quando os gates **existem e falham pelo motivo certo** — motivo
registrado, reprodutível e ligado à tarefa que o resolverá. Um gate verde ao fim desta tarefa, nas
frentes 2 e 3, é sinal de que o gate está errado, não de que o trabalho acabou. A frente 1 é a
exceção: ela precisa ficar **verde**, porque é correção efetiva e não instrumentação.

<requirements>
- Frente 1 (`.tmp-*`) entrega correção e fica **verde**.
- Frente 2 (`TestAllLangsAreExhaustivelyWired`) e frente 3 (prova de dispatch por célula) entregam
  instrumentação e ficam **vermelhas por motivo declarado**, cada uma com a tarefa sucessora nomeada.
- Nenhum gate vermelho pode ficar sem registro de motivo e sem sucessor: vermelho sem dono vira
  ruído e é desligado pela equipe seguinte.
- A matriz de capabilities **não pode ficar vermelha** ao fim da tarefa — ver a Observação abaixo.
- Toda citação de código conferida em `arquivo:linha`.
- Documento em PT-BR; código em inglês, zero comentários (R-STYLE-001, hard).
- Zero regressão nas áreas já verdes: todos os gates da seção de Critérios de Sucesso passam.
</requirements>

## Subtarefas

### Frente 1 — Isenção de `.tmp-*` nos gates de isolamento (deve ficar VERDE)

- [ ] 3.1 Acrescentar o padrão `.tmp-*` ao conjunto canônico de exclusões em
      `internal/taskloop/orchestrator.go:456-513` (`buildExclusions`, assinatura em `:456`, retorno
      em `:512`). **Verificado: o padrão não consta hoje.** As exclusões existentes cobrem o backup
      de migração (`:506`) e `.checkpoints/**` (`:507-511`) — nada de temporários.
- [ ] 3.2 Acrescentar o mesmo padrão em `internal/taskloop/isolation.go:322`, no slice
      `[]string{"memory", ".checkpoints", ".partials"}` de `isHarnessManagedPRDDir`
      (`:321`). **Verificado: o padrão não consta hoje.** Os dois pontos precisam concordar — se
      divergirem, o digest do selo deixa de bater com o do fechamento por um motivo invisível, que é
      exatamente o argumento registrado no comentário de contexto de `buildExclusions`.
- [ ] 3.3 Registrar no arquivo de execução a cadeia causal completa que justifica a frente:
      `internal/fs/fs.go:181` cria o temporário com `os.CreateTemp(dir, ".tmp-*")` **no diretório de
      destino**. Quando a tarefa 11.0 migrar a escrita sob `.specs/` para `WriteFileAtomic`, esse
      temporário aparece como untracked no patch semântico, altera o `PatchSHA256` e dispara o
      fail-closed de `internal/taskloop/orchestrator.go:352-354`
      (`"taskloop: resultado diverge do estado final recomputado"`). Sem 3.1 e 3.2, a 11.0 não fecha.
- [ ] 3.4 Criar `tests/integration/atomic_write_isolation_test.go`: escrita atômica sob um PRD
      temporário não altera o `PatchSHA256` nem dispara violação de isolamento; a corrida entre a
      criação do temporário e o cálculo do digest é exercitada explicitamente, não presumida.

### Frente 2 — `TestAllLangsAreExhaustivelyWired` (nasce VERMELHO, sucessor 7.0)

- [ ] 3.5 Criar `TestAllLangsAreExhaustivelyWired` em `internal/skills/`, iterando
      `skills.AllLangs` (`internal/skills/skills.go:91`) e exigindo, **por linguagem**, os cinco
      pontos de wiring:
      1. skill de implementação em `LangSkills` — `internal/skills/skills.go:150-166`, switch em
         `:153`, cases em `:154-161`;
      2. entrada em `langImplementationSkills` — `internal/install/install.go:1337-1343`;
      3. label e presença no loop de geração — `internal/contextgen/contextgen.go:339-340`
         (`labels := map[string]string{...}` e `for _, lang := range []string{"go","node","python"}`);
      4. arquivo de trigger em `.agents/skills/agent-governance/triggers/` — hoje existem apenas
         `go.yaml`, `node.yaml` e `python.yaml`;
      5. `case` no switch de `internal/detect/toolchain.go:148-163`.
- [ ] 3.6 Registrar o motivo do vermelho: **`LangDotNet` falha nos pontos 3, 4 e 5**. `LangDotNet`
      está declarado em `internal/skills/skills.go:88` e presente em `AllLangs` (`:91`), em
      `ParseLang` (`:95`) e em `LangSkills` (`:160`), mas **não** tem label/loop em
      `contextgen.go:339-340`, **não** tem `triggers/dotnet.yaml` e **não** tem `case` em
      `toolchain.go:148-163`. O vermelho antecede Java: a lacuna de .NET já existe hoje. Sucessor
      declarado: **tarefa 7.0**.
- [ ] 3.7 Acrescentar `default` que falha ou avisa aos pontos de decisão sem tratamento exaustivo.
      **Duas correções de premissa, verificadas:**
      - `internal/detect/toolchain.go:148` — switch sem `default`: acrescentar. ✔
      - `internal/skills/skills.go:95` — `ParseLang`, switch sem `default` (cai no `return "", false`
        de `:97`): acrescentar tratamento explícito. ✔
      - `internal/skills/skills.go:153` — `LangSkills`, switch sem `default`: acrescentar. ✔
      - `internal/contextgen/contextgen.go:339-340` — **não é switch**: é um `map[string]string` de
        labels seguido de `for _, lang := range []string{"go","node","python"}`. `default` não se
        aplica. O tratamento correto é derivar as duas listas de `skills.AllLangs` em vez de
        hardcodá-las, e o teste da frente 2 é quem trava isso.
      - `internal/triggers/triggers.go:60` — **já tem `default`** (`:63-64`), e o `default` é
        justamente o bug: `normalizeLang` faz fallback silencioso para `"go"` em qualquer linguagem
        desconhecida. Não é caso de acrescentar `default`; é caso de **trocar o fallback silencioso
        por falha explícita**. Consequência conhecida e aceita: `internal/triggers/triggers_test.go:133`
        assere hoje que `"java"` e `"kotlin"` caem no fallback para Go — é o único teste que quebra
        de propósito, e precisa ser editado no mesmo commit da correção.
- [ ] 3.8 Registrar como risco os testes de falso verde que a 7.0 herda:
      `internal/install/verify_lang_filter_test.go` lista `AllLangs` manualmente expandido — ao
      acrescentar `LangJava` ele continua compilando e passando **sem cobrir Java**, pior do que
      quebrar; e `internal/skills/skills_test.go` usa aritmética hardcoded
      (`len(BaseSkills) + len(ComplementarySkills) + 2`), que quebra ao mexer em `LangSkills`.

### Frente 3 — Indexação da prova de dispatch por célula (nasce VERMELHO, sucessor 9.0)

- [ ] 3.9 Corrigir a indexação em `internal/capability/evidence.go`: `dispatchProvenFromTests`
      (`:140`) calcula `resolved` como **um único booleano** em `:146`
      (`declared && executionProof(...)`) e devolve em `:147-149` uma closure que **ignora os
      parâmetros `provider` e `capabilityID`**. Resultado: uma única prova de execução vale para
      toda a matriz, sem discriminar célula.
- [ ] 3.10 Registrar o que **não** deve ser perdido na correção — a prova já é forte em duas
      dimensões e ambas estão travadas por `evidence_regression_test.go`:
      execução real exigida via `go test -json -count=1 -run <pattern>` em
      `internal/capability/evidence.go:128-137` (`executionProof`), e `skip`/`fail` revogando `pass`
      em `:114-117`. A correção é de **indexação**, não de força: nenhuma das duas propriedades pode
      ser afrouxada para fazer a célula caber.
- [ ] 3.11 Acrescentar `TestDispatchProof_DiscriminatesByCell` — **nasce vermelho**: duas células
      distintas `(provedor, capabilityID)` recebem hoje a mesma resposta da closure de `:147-149`.
      Sucessor declarado: **tarefa 9.0**, que consome a matriz por evento e família.
- [ ] 3.12 Acrescentar `TestDispatchProof_ReportsEnvironmentFailure` para o fallback mudo de
      `internal/capability/evidence.go:169-173`: quando `repoRootFromWorkingDir` falha,
      `DispatchProvenFromParityTests` devolve em `:172` uma closure `return false` **sem distinguir
      falha de ambiente de prova ausente**. Falha de ambiente deve ser reportada como tal, não
      confundida com célula não provada.
- [ ] 3.13 Mapear individualmente as **44 células afirmativas** da matriz atual para sua prova
      própria, no mesmo lote — ver a Observação abaixo. Sem esse mapeamento, a correção da indexação
      deixa a matriz vermelha, e a matriz não pode ficar vermelha ao fim da tarefa.
- [ ] 3.14 Executar todos os gates declarados em Critérios de Sucesso, persistir as saídas como
      evidência e registrar, item a item, qual gate está verde e qual está vermelho pelo motivo
      declarado.

## Observação importante — as 44 células afirmativas

Corrigir a indexação de `evidence.go:146-149` tem um efeito colateral previsto e não negociável:
**as 44 células afirmativas da matriz atual ficam sem prova** no instante em que a closure deixa de
devolver um booleano único para toda a matriz. Hoje elas são afirmativas por herdarem a mesma prova
global; com indexação por célula, cada uma precisa da sua.

Por isso o mapeamento individual (subtarefa 3.13) entra **no mesmo lote** da correção, e não em
tarefa seguinte. A regra de fechamento é assimétrica e deliberada:

- `TestDispatchProof_DiscriminatesByCell` e `TestDispatchProof_ReportsEnvironmentFailure` **ficam
  vermelhos** — é a instrumentação nascendo antes do que ela protege;
- **`make check-capability-matrix-sync` fica verde** — a matriz não pode ficar vermelha ao fim da
  tarefa.

Confundir os dois casos é o erro que essa observação existe para impedir.

## Detalhes de Implementação

Referenciar, sem duplicar:

- `adr-004-prova-dispatch-por-celula.md` desta pasta — decisão da frente 3, alternativas e o custo de
  mapear as 44 células.
- `adr-005-exaustividade-de-linguagens.md` desta pasta — decisão da frente 2, incluindo o critério de
  exaustividade e o tratamento do fallback de `normalizeLang`.
- `adr-006-atomicidade-artefatos-operacionais.md` desta pasta — contexto da frente 1: por que a
  escrita atômica da 11.0 exige a isenção de `.tmp-*` antes.
- `techspec.md` — `### Interfaces Chave` (linha 116); `### Testes Unitários` (linha 318) e
  `### Testes de Integração` (linha 377); `### Rastreabilidade — RF × decisão × arquivo × teste`
  (linha 409) para RF-59; `### Ordem de Build` (linha 453) para 3.0 → 7.0, 9.0, 11.0;
  `### Riscos Conhecidos` (linha 586).
- `tasks.md` — `## Dependências Críticas`, bloco "3.0 precede 7.0 e 11.0, e é pré-requisito de 9.0";
  `## Riscos de Integração`, armadilhas "Ordem em `manifestTypes`", "Testes de falso verde",
  "Aritmética hardcoded" e "Teste que codifica o bug".

**Armadilha de ordenação a respeitar já nesta tarefa:** em
`internal/detect/toolchain.go`, o desempate de `manifestTypes` é "primeiro candidato com score máximo
vence". Marcadores novos só podem ser **apendados ao fim**; inserir no meio muda o vencedor em
monorepo poliglota e quebra o cenário `polyglot-monorepo` de `tests/integration/portability_test.go`,
cujo `wantStackSub` é substring **contígua**. Vale igualmente para `DetectPrimaryStack` em
`internal/detect/framework.go`. Esta tarefa acrescenta `default` ao switch de `toolchain.go:148`, não
reordena `manifestTypes` — a restrição fica registrada para a 7.0.

## Critérios de Sucesso

- **Frente 1 — verde:** `.tmp-*` consta em `internal/taskloop/orchestrator.go:456-513` **e** em
  `internal/taskloop/isolation.go:322`, com os dois pontos concordando.
  `tests/integration/atomic_write_isolation_test.go` passa: escrita atômica sob `.specs/` não altera
  `PatchSHA256` nem dispara o fail-closed de `orchestrator.go:352-354`.
- **Frente 2 — vermelho pelo motivo certo:** `TestAllLangsAreExhaustivelyWired` existe, executa e
  falha para `LangDotNet` nos pontos 3, 4 e 5 — contextgen, trigger e toolchain. A saída do teste
  nomeia a linguagem e o ponto de wiring ausente; falha genérica não satisfaz o critério. Sucessor
  registrado: 7.0.
- **Frente 2 — `default` instalado:** switches de `toolchain.go:148`, `skills.go:95` e `skills.go:153`
  passam a tratar caso não previsto. `contextgen.go:339-340` e `triggers.go:60` recebem o tratamento
  correto para o que de fato são (lista hardcoded e `default` de fallback silencioso), com a
  divergência de premissa registrada no relatório de execução.
- **Frente 3 — vermelho pelo motivo certo:** `TestDispatchProof_DiscriminatesByCell` e
  `TestDispatchProof_ReportsEnvironmentFailure` existem, executam e falham pelas causas declaradas
  (`evidence.go:146-149` e `:169-173`). Sucessor registrado: 9.0.
- **Frente 3 — prova preservada:** `evidence_regression_test.go` continua verde; nem a exigência de
  execução real (`evidence.go:128-137`) nem a revogação de `pass` por `skip`/`fail` (`:114-117`)
  foram afrouxadas.
- **As 44 células afirmativas estão mapeadas individualmente** e
  `make check-capability-matrix-sync` está **verde**.
- Cada gate vermelho tem motivo declarado e tarefa sucessora nomeada no relatório de execução.
- **Gates de não-regressão (inegociáveis):**
  - `make test lint vet` (mínimo transversal) — verde, exceto os testes desta tarefa declarados como
    vermelhos por design, listados nominalmente no relatório de execução
  - `make check-capability-matrix-sync` — verde, a tarefa toca `internal/capability/`
  - `make coverage` — 75% total e 70% por pacote crítico

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
  - `internal/skills/` — `TestAllLangsAreExhaustivelyWired`, table-driven sobre `skills.AllLangs`,
    um subteste por linguagem e por ponto de wiring. **Vermelho esperado** para `LangDotNet`.
  - `internal/capability/` — `TestDispatchProof_DiscriminatesByCell` (**vermelho esperado**) e
    `TestDispatchProof_ReportsEnvironmentFailure` (**vermelho esperado**).
  - `internal/capability/evidence_regression_test.go` — continua **verde**; guarda de não-afrouxamento.
  - `internal/taskloop/` — exclusões de `buildExclusions` e `isHarnessManagedPRDDir` com
    `FakeFileSystem`, incluindo o caso `.tmp-*` aninhado sob `.specs/<prd>/`.
- [ ] Testes de integração
  - `tests/integration/atomic_write_isolation_test.go` — build tag `integration`, `t.TempDir()`:
    `WriteFileAtomic` sob `.specs/` não altera `PatchSHA256` nem dispara violação de isolamento.
  - Asserção de que o pacote de integração novo está referenciado em `.github/workflows/test.yml`
    (armadilha de gate órfão).

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

Frente 1:
- `internal/taskloop/orchestrator.go:456-513` — `buildExclusions`; fail-closed em `:352-354`
- `internal/taskloop/isolation.go:321-322` — `isHarnessManagedPRDDir`
- `internal/fs/fs.go:181` — `os.CreateTemp(dir, ".tmp-*")`
- `tests/integration/atomic_write_isolation_test.go` (a criar)

Frente 2:
- `internal/skills/skills.go:88,91,94-97,150-166` — `LangDotNet`, `AllLangs`, `ParseLang`, `LangSkills`
- `internal/install/install.go:1337-1343` — `langImplementationSkills`
- `internal/contextgen/contextgen.go:339-340` — labels e loop hardcoded (não é switch)
- `internal/detect/toolchain.go:148-163` — switch sem `default`
- `internal/triggers/triggers.go:59-65` — `normalizeLang`, `default` de fallback silencioso em `:63-64`
- `internal/triggers/triggers_test.go:133` — teste que codifica o bug do fallback
- `.agents/skills/agent-governance/triggers/` — `go.yaml`, `node.yaml`, `python.yaml` (sem dotnet)
- `internal/install/verify_lang_filter_test.go` — risco de falso verde (registro, correção na 7.0)
- `internal/skills/skills_test.go` — aritmética hardcoded (registro, correção na 7.0)
- `internal/detect/framework.go` — `DetectPrimaryStack`, restrição de ordenação registrada
- `tests/integration/portability_test.go` — cenário `polyglot-monorepo`

Frente 3:
- `internal/capability/evidence.go:114-117,128-137,140,146,147-149,169-173`
- `internal/capability/evidence_regression_test.go` — guarda de não-afrouxamento

Decisões de referência:
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-004-prova-dispatch-por-celula.md`
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-005-exaustividade-de-linguagens.md`
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-006-atomicidade-artefatos-operacionais.md`
