# Tarefa 6.0: Harness contract aplicado e gate de operacao git derivado da policy

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fazer a policy declarada no contrato do harness **ter efeito real** e reconstruir a decisão do gate
de operação Git por **estrutura de comando**, não por regex sobre texto livre — eliminando os escapes
verificados sem introduzir falso positivo, conforme
[ADR-003](adr-003-parsing-estrutural-sem-truncamento.md).

Hoje a afirmação "o gate deriva da policy" é falsa e verificável como falsa: `.agents/harness.yaml`
**não existe no repositório** e `GitPolicy.AutoCommit`/`AutoPush` têm **zero call-sites de
produção**. Esta tarefa cria o artefato e constrói o elo.

Requisitos cobertos: RF-18, RF-19, RF-20, RF-21, RF-22, RF-23, RF-24. Dependências: 2.0, 4.0.
Paralelizável com 7.0.

<requirements>
- RF-18 — o escopo de operações Git sujeitas ao gate é **derivado** de `harness.Contract`, não codificado no shell.
- RF-19 — `Fingerprint()` SHA-256 do escopo derivado é gravado em `.agents/generated/git-scope.json` e um gate de CI recalcula e falha na divergência. Sem isso, "derivado da policy" seria afirmação não verificável — exatamente o estado atual.
- RF-20 — a decisão do gate passa a ser por **palavra-de-comando**: resolve `env`, `command`, caminho absoluto e nome com extensão; respeita aspas na segmentação.
- RF-21 — o escopo cobre, além de `commit` e `push`: `reset --hard`, `clean` com remoção efetiva, `checkout`/`restore` destrutivos e force push em todas as variantes.
- RF-22 — zero falso positivo sobre menção textual: `echo`, `man`, `grep` e conteúdo dentro de string literal não podem bloquear.
- RF-23 — auditoria de bypass migra para o registro canônico de decisões (`internal/hookaudit`, tarefa 4.0), sem perda do TSV legado dentro desta tarefa.
- RF-24 — invocação de interpretador com conteúdo dinâmico é **classe própria** de decisão, com `policy_id` próprio, e exige aprovação explícita.
- **ORDEM OBRIGATÓRIA:** a matriz adversarial e a matriz de falso positivo entram em `scripts/test-validators.sh` **ANTES** da correção e nascem vermelhas em **pelo menos onze casos** de escape mais **quatro** de falso positivo. Corrigir antes de instrumentar invalida a tarefa.
- Zero regressão: todo caso que **hoje** bloqueia corretamente continua bloqueando; todo caso que hoje passa corretamente continua passando. A matriz é gate dos dois sentidos.
- R-STYLE-001 (hard) — código Go em inglês, zero comentários, sem prefixo `_` em identificador.
</requirements>

## Subtarefas

- [ ] 6.1 **Instrumentar primeiro.** Acrescentar a `scripts/test-validators.sh` a matriz adversarial
      (onze casos de escape) e a matriz de falso positivo (quatro casos), ambas com os exit codes
      **atuais** registrados como `want` provisório invertido. Verificado: `test-validators.sh` tem
      hoje **zero** referências a `git-operation-gate`. Persistir a execução vermelha como evidência
      antes de tocar em qualquer linha do gate.
- [ ] 6.2 Criar `.agents/harness.yaml` com a policy declarada, incluindo `git.auto_commit` e
      `git.auto_push`, coerente com `internal/harness/contract.go`. Espelhar em
      `internal/embedded/assets/` conforme ADR-001 do repositório.
- [ ] 6.3 Criar `internal/hookpolicy/git.go` com o VO `GitOperation` (`subcommand`, `destructive`,
      `requiresApproval`), a interface `GitScope` e `NewGitScope(contract harness.Contract)
      (GitScope, error)`. Assinaturas em [`techspec.md`](techspec.md) seção "Interfaces Chave".
- [ ] 6.4 Implementar `Fingerprint()` — SHA-256 determinístico sobre a lista derivada, com ordenação
      estável — e o gerador de `.agents/generated/git-scope.json`, com espelho em
      `internal/embedded/assets/`.
- [ ] 6.5 Acrescentar ao `Makefile` e a `.github/workflows/test.yml` o gate que recalcula o
      fingerprint a partir de `.agents/harness.yaml` e falha na divergência com o artefato gerado
      (RF-19). O alvo `check-policies-sync` já existe em `Makefile:87-88` e é o ponto natural de
      ancoragem.
- [ ] 6.6 Reescrever a decisão de `.agents/scripts/git-operation-gate.sh` para palavra-de-comando:
      segmentação que respeita aspas, resolução de `env` / `command` / caminho absoluto / nome com
      extensão, e leitura do escopo a partir de `.agents/generated/git-scope.json` — nunca de
      alternação embutida no script.
- [ ] 6.7 Ampliar o escopo para `reset --hard`, `clean` com remoção efetiva (`-f` combinado com
      `-d`/`-x`), `checkout`/`restore` destrutivos e force push em todas as variantes (`--force`,
      `--force-with-lease`, `-f`, refspec com `+`) — este último como **classificação** explícita,
      já que as variantes de force push hoje já bloqueiam por efeito colateral da alternação (ver
      "Detalhes de Implementação").
- [ ] 6.8 Introduzir a classe de decisão para invocação de interpretador com conteúdo dinâmico
      (`eval`, `sh -c`, `bash -c`, substituição de comando `$( )` e crase), com `policy_id` próprio
      e exigência de aprovação (RF-24).
- [ ] 6.9 Auditar caso a caso os invocadores internos de `bash`, registrando o veredito no
      `execution_report.md`: `.agents/hooks/subagent-stop-wrapper.sh:127` e
      `.agents/hooks/post-execute-task.sh:283` invocam `bash` com **caminho fixo em variável**, não
      com conteúdo dinâmico — não devem ser capturados pela nova classe.
- [ ] 6.10 Ligar a auditoria de bypass ao registro canônico `internal/hookaudit` da tarefa 4.0,
      mantendo o TSV legado em paralelo nesta tarefa (RF-23).
- [ ] 6.11 Inverter os `want` da matriz de 6.1 para os valores corretos e provar a suíte verde.
- [ ] 6.12 Espelhar todos os artefatos alterados e rodar
      `make check-hooks-sync check-scripts-sync check-policies-sync`.

## Detalhes de Implementação

Racional completo em [ADR-003](adr-003-parsing-estrutural-sem-truncamento.md) (parsing estrutural,
decisão por palavra-de-comando, negação por ausência de alvo) e em [`techspec.md`](techspec.md)
seções "Interfaces Chave" e "Sequenciamento de Desenvolvimento → Fase 3". **Não duplicar aqui.**

### O elo que não existe — estado verificado

- **`.agents/harness.yaml` NÃO EXISTE.** `find . -name "harness.yaml" -not -path "./.git/*"` retorna
  zero resultados. A tarefa 3.0 do PRD dependente foi marcada `done` sem entregar o artefato.
- `internal/harness/contract.go:12-14` declara `type GitPolicy struct` com `AutoCommit` (`:13`) e
  `AutoPush` (`:14`). Busca por esses dois identificadores em `internal/` e `cmd/`, excluindo
  `_test.go`, retorna **apenas as duas declarações** — zero call-sites de produção.
- O único consumidor do pacote `internal/harness` é `internal/doctor/doctor.go` (mais o mock gerado
  em `internal/harness/mocks/loader.go`). `checkContract` (`doctor.go:241`) carrega o contrato e lê
  **somente** `contract.Version` (`doctor.go:250` e `doctor.go:254`), reportando `warn` quando
  `.agents/harness.yaml` não foi declarado — que é o estado permanente hoje.

O contrato carrega e é ignorado. Esta tarefa cria o artefato **e** o elo.

### Escopo atual do gate — verificado

`.agents/scripts/git-operation-gate.sh:70`:

```
| grep -Eq '(^|[[:space:]])git([[:space:]]+-[^[:space:]]+)*[[:space:]]+(commit|push)([[:space:]]|$)'
```

Só `commit` e `push`. A segmentação é `tr ';|&' '\n'` em `git-operation-gate.sh:100` — corte cego,
que não conhece aspas. Comando vazio responde `exit 0` (`:39-41`).

### Matriz adversarial — escapes verificados por execução real

Cada linha abaixo foi executada ponta a ponta contra `.agents/scripts/git-operation-gate.sh` com
`GOVERNANCE_GIT_OPERATION_CONFIRMED=0`. **Todos devolveram `exit 0` — escapam hoje.** Controle:
`git push` devolve `exit 2`.

| # | Comando | Causa do escape |
|---|---|---|
| 1 | `git -C /repo push` | O grupo `([[:space:]]+-[^[:space:]]+)*` consome `-C` mas quebra no primeiro token que não começa com `-` (`/repo`) |
| 2 | `git -c user.name=x commit -m y` | Mesma causa: `user.name=x` interrompe o grupo de flags |
| 3 | `$(echo git) push` | Substituição de comando não é resolvida por regex |
| 4 | `eval "git push"` | Conteúdo dinâmico de interpretador |
| 5 | `sh -c "git push"` | Conteúdo dinâmico de interpretador |
| 6 | `/usr/bin/git push` | A âncora `(^\|[[:space:]])git` não casa após `/` |
| 7 | `git.exe push` | O token é `git.exe`, não `git` |
| 8 | `git reset --hard` | Ausente da alternação `(commit\|push)` |
| 9 | `git clean -fd` | Ausente da alternação |
| 10 | `git checkout -- .` | Ausente da alternação |
| 11 | `git restore .` | Ausente da alternação |

São **onze** casos. A matriz de 6.1 nasce vermelha em todos eles.

### Matriz de falso positivo — verificados por execução real

**Todos devolveram `exit 2` — bloqueiam hoje sem que devessem.**

| # | Comando | Causa |
|---|---|---|
| 1 | `echo git commit` | Menção textual casa a regex |
| 2 | `man git commit` | Menção textual casa a regex |
| 3 | `echo "a\|git commit -m x"` | `tr ';\|&'` (`:100`) corta **dentro da string literal**, produzindo o segmento `git commit -m x"` |
| 4 | `echo "a\|git push origin main"` | Mesma causa |

**Correção da descrição original.** O caso citado como `echo "a|git push"` devolve na verdade
`exit 0`: após o corte, o segmento vira `git push"` e a aspa final derrota a âncora
`([[:space:]]|$)` da regex. O falso positivo **existe e é real**, mas só se reproduz quando o
comando dentro da string tem argumento depois do subcomando — as duas formas da tabela acima. Pelo
mesmo motivo, `grep -r "git push" docs/` devolve `exit 0` hoje; é um caso de **guarda de
não-regressão**, não de correção.

### Casos que já se comportam corretamente — guarda de não-regressão obrigatória

Verificados por execução: `env git push` (`exit 2`), `command git push` (`exit 2`),
`git push --force` (`exit 2`), `git push --force-with-lease` (`exit 2`), `git push -f origin main`
(`exit 2`) e `git push origin +main` (`exit 2`).

Duas consequências para o desenho:

1. `env` e `command` **já** são capturados hoje, porque a âncora de espaço da regex casa o `git` que
   os segue. A resolução de palavra-de-comando de RF-20 não pode **perder** esses casos — a
   ampliação é de robustez, não de cobertura nova.
2. Force push **já** bloqueia hoje, por efeito colateral: a alternação casa `push` independentemente
   das flags que venham depois. O que RF-21 acrescenta é **classificação** — force push deixa de ser
   indistinguível de um push comum e passa a carregar `destructive: true` e `policy_id` próprio. A
   cobertura nova de force push está nas formas que escapam pelos motivos 1, 2 e 6 da matriz
   adversarial (`git -C /repo push --force`, `/usr/bin/git push -f`).

Toda a matriz de não-regressão acima entra em `scripts/test-validators.sh` junto com as outras duas.

### Bypass e aprovação — preservados

`GOVERNANCE_GIT_OPERATION_CONFIRMED` e `GOVERNANCE_GIT_OPERATION_MODE` são lidos em
`git-operation-gate.sh:114-115` e o escape é auditado em `.aispec/governance-escapes.log` via
`audit_git_operation_escape` (`:43`, log em `:45`; o segundo escritor, para operação destrutiva,
está em `:57`). O mecanismo de bypass **não muda de nome nem de semântica** nesta tarefa — o que
muda é o destino da auditoria, que passa a incluir o registro canônico de `internal/hookaudit`.

### Interpretador com conteúdo dinâmico — decisão consciente, com custo declarado

`eval`, `sh -c`, `bash -c`, `$( )` e crase **não são resolvíveis por análise estática**. Tentar
resolvê-los produziria ou falso negativo (o estado de hoje) ou falso positivo generalizado. A
decisão registrada é **trocar escape silencioso por fricção explícita**: essas formas passam a
exigir aprovação, com `policy_id` próprio que as distingue de um bloqueio de `git push`.

**Isso pode atritar fluxo interno** e a tarefa assume esse custo conscientemente. A mitigação é a
auditoria caso a caso de 6.9: `.agents/hooks/subagent-stop-wrapper.sh:127`
(`bash "$POST_EXECUTE_HOOK" ...`) e `.agents/hooks/post-execute-task.sh:283`
(`bash "$evidence_validator" ...`) invocam `bash` com **caminho fixo resolvido em variável**, não com
conteúdo dinâmico — e portanto não caem na classe nova. Qualquer invocador que caia precisa de
veredito individual registrado, nunca de isenção por categoria.

### Ordem de trabalho — não negociável

`techspec.md` risco R-02 registra o risco inverso do usual: **instrumentar depois da correção
afrouxa o gate sem que ninguém perceba**. A ordem é: (1) matriz vermelha, (2) evidência da execução
vermelha persistida, (3) correção, (4) matriz verde. Um `execution_report.md` sem a evidência do
passo 2 reprova a tarefa.

## Critérios de Sucesso

- [ ] `.agents/harness.yaml` existe, é carregado, e `doctor` deixa de reportar `warn` de contrato
      não declarado.
- [ ] `GitPolicy.AutoCommit`/`AutoPush` têm call-site de produção real em `internal/hookpolicy`
      (RF-18). A busca que hoje retorna só as duas declarações passa a retornar consumo.
- [ ] `Fingerprint()` é determinístico, gravado em `.agents/generated/git-scope.json`, espelhado em
      `internal/embedded/assets/`, e o gate de CI **falha comprovadamente** quando o artefato é
      editado à mão — demonstrar a execução vermelha (RF-19).
- [ ] Os **onze** casos da matriz adversarial passam de `exit 0` para bloqueio ou aprovação
      explícita (RF-20, RF-21, RF-24).
- [ ] Os **quatro** casos da matriz de falso positivo passam de `exit 2` para `exit 0` (RF-22).
- [ ] Os casos de guarda de não-regressão (`env git push`, `command git push`, as quatro variantes
      de force push, `git push` simples, `grep -r "git push" docs/`) mantêm o comportamento atual.
- [ ] Invocação de interpretador com conteúdo dinâmico tem `policy_id` próprio, distinto do de
      `git push` (RF-24).
- [ ] O veredito individual de cada invocador interno de `bash` está no `execution_report.md`
      (6.9).
- [ ] A auditoria de bypass alcança `internal/hookaudit` sem perda do TSV legado (RF-23).
- [ ] A evidência inclui a execução **vermelha** das duas matrizes anterior à correção.

### Gates de não-regressão — obrigatórios, requisito inegociável

- [ ] `make test` — verde.
- [ ] `make lint` — verde.
- [ ] `make vet` — verde.
- [ ] `make check-hooks-sync` — verde (área tocada: `.agents/hooks/`).
- [ ] `make check-scripts-sync` — verde (área tocada: `.agents/scripts/`).
- [ ] `make check-policies-sync` — verde (área tocada: `.agents/policies/`, `.agents/generated/`).
- [ ] `make test-validators` — verde, com as três matrizes novas.
- [ ] `make test-hooks` — verde (área tocada: `.claude/hooks/`, `.agents/hooks/`).
- [ ] `make check-mocks` — verde (interfaces novas em `internal/hookpolicy`).
- [ ] `make coverage` — 75% total e 70% por pacote crítico preservados.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários — `internal/hookpolicy`: derivação de `GitScope` a partir de
      `harness.Contract`, determinismo e estabilidade de `Fingerprint()`, classificação de
      `GitOperation` (`destructive`, `requiresApproval`), construtor validador recusando operação
      sem subcomando. `FakeFileSystem` para a geração do artefato.
- [ ] Testes de integração — `scripts/test-validators.sh` com as três matrizes (adversarial,
      falso positivo e guarda de não-regressão) executadas contra o script real; `scripts/test-hooks.sh`
      para o dispatch do gate no ponto `BeforeTool`; gate de fingerprint no CI.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

**A criar:**
- `.agents/harness.yaml` (+ espelho em `internal/embedded/assets/.agents/harness.yaml`)
- `internal/hookpolicy/git.go` e testes
- `.agents/generated/git-scope.json` (+ espelho em `internal/embedded/assets/`)

**A modificar:**
- `.agents/scripts/git-operation-gate.sh:5,39-41,45,57,62-63,67-70,100,114-115`
  (+ espelhos `.claude/scripts/git-operation-gate.sh`,
  `internal/embedded/assets/.agents/scripts/`, `internal/embedded/assets/.claude/scripts/`)
- `scripts/test-validators.sh` — hoje com zero referências ao gate Git
- `scripts/test-hooks.sh`
- `Makefile` — alvo do gate de fingerprint, ancorado em `check-policies-sync` (`Makefile:87-88`)
- `.github/workflows/test.yml` — step do gate de fingerprint
- `internal/doctor/doctor.go:241-254` — apenas se o veredito de contrato declarado mudar de estado

**Consumido, não modificado:**
- `internal/harness/contract.go:12-14`
- `internal/hookcontract/`, `internal/hookaudit/` (entregues na tarefa 4.0)

**A auditar caso a caso (6.9), não modificar sem veredito:**
- `.agents/hooks/subagent-stop-wrapper.sh:127`
- `.agents/hooks/post-execute-task.sh:283`

**Leitura obrigatória:**
- `.specs/prd-hooks-canonicos-vendor-neutral/prd.md` (RF-18 a RF-24)
- `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md` (Interfaces Chave; Fase 3; risco R-02)
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-003-parsing-estrutural-sem-truncamento.md`
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-002-resultado-tipado-traducao-exit-code.md`
