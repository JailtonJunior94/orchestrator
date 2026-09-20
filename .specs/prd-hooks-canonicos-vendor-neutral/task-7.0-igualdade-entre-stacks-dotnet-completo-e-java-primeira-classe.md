# Tarefa 7.0: Igualdade entre stacks: dotnet completo e java como cidadao de primeira classe

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fazer o harness tratar Go, Node/TypeScript, C#/.NET, Java e Python de forma igualitária, eliminando
a degradação silenciosa descrita em [`adr-005-exaustividade-de-linguagens.md`](adr-005-exaustividade-de-linguagens.md):
a enumeração de linguagens está espalhada por pontos independentes, nenhum deles com `default` que
falhe, e cada ponto degrada em silêncio ao encontrar uma linguagem que não conhece.

A ordem de execução é obrigatória e não é preferência estilística: **.NET primeiro, Java depois**.
.NET já consta de `internal/skills/skills.go:91` (`AllLangs = []Lang{LangGo, LangNode, LangPython, LangDotNet}`)
e ainda assim não produz toolchain nem detecção de stack — completar .NET fecha três bugs ativos com
raio de alteração menor e valida o caminho antes de Java, que exige acrescentar um valor novo à
enumeração e propagá-lo por todos os pontos.

Cobre RF-31, RF-32, RF-33, RF-34 e RF-73. Depende da tarefa 3.0 (o teste de exaustividade
`TestAllLangsAreExhaustivelyWired` nasce vermelho lá, para `LangDotNet`, antes de Java existir).
Paralelizável com 5.0 e 6.0.

<requirements>
- ORDEM OBRIGATÓRIA: completar .NET em todos os pontos e provar verde antes de introduzir `LangJava`.
- Os cinco bugs ativos de enumeração listados em "Detalhes de Implementação" devem estar corrigidos,
  cada um com teste que falhava antes da correção.
- Nenhum `switch` sobre linguagem pode permanecer sem `default` que falhe ou sem cobertura provada
  pelo teste de exaustividade da tarefa 3.0.
- Valores de toolchain .NET reusam os canônicos já emitidos pelo caminho shell
  (`.agents/skills/analyze-project/scripts/generate-governance.sh:605-609`) e por
  `.agents/skills/dotnet-csharp-implementation/SKILL.md`. Proibido inventar comandos novos.
- Marcadores de manifesto e nomes de stack novos entram SEMPRE ao FIM das listas existentes
  (ver restrições de ordem abaixo). Inserção no meio é regressão silenciosa.
- Ampliação de regex de evidência AFROUXA o gate: os casos de teste entram em
  `scripts/test-validators.sh` ANTES da ampliação, nunca depois.
- `TEST_NAME_RE` de `.agents/scripts/validate-review-evidence.sh:46` deve ficar CONDICIONADA à stack
  detectada. Ampliá-la globalmente afrouxaria o gate para todas as linguagens, Go inclusive.
- FORA DE ESCOPO desta tarefa e do PRD: criar `.agents/skills/java-implementation/`. RF-32 cobre
  detecção, enumeração e toolchain — não a skill de implementação Java.
- Código em inglês, zero comentários (R-STYLE-001, hard). Documento e relatório em PT-BR.
</requirements>

## Subtarefas

- [ ] 7.1 Confirmar que a tarefa 3.0 está `done` e que `TestAllLangsAreExhaustivelyWired` está
      vermelho para `LangDotNet`. Registrar a saída como linha de base.
- [ ] 7.2 Extrair os valores canônicos de toolchain .NET de
      `.agents/skills/analyze-project/scripts/generate-governance.sh:605-609`
      (`dotnet format --verify-no-changes`, `dotnet build --no-restore`, `dotnet test --no-build`) e
      de `.agents/skills/dotnet-csharp-implementation/SKILL.md`. Registrar a origem de cada valor.
- [ ] 7.3 Corrigir `internal/detect/toolchain.go`: acrescentar o ramo .NET ao `switch winner.lang`
      (`:148-163`) e a `detectDefault` (`:52-87`), e acrescentar um `default` que falhe em vez de
      devolver `ToolchainResult` vazio.
- [ ] 7.4 Apendar os marcadores de manifesto .NET (`*.csproj`, `*.sln`, `global.json`) ao FIM de
      `manifestTypes` (`internal/detect/toolchain.go:99-108`), nunca no meio.
- [ ] 7.5 Acrescentar `"C#/.NET"` ao FIM de `DetectPrimaryStack`
      (`internal/detect/framework.go:125-166`), depois do ramo `Rust` (`:158-160`).
- [ ] 7.6 Ampliar `internal/triggers/triggers.go`: `normalizeLang` (`:59-66`) e `DetectLang`
      (`:78-98`) passam a reconhecer `.cs` e a linguagem `dotnet`, eliminando o
      `default: return "go"` como destino de linguagens conhecidas.
- [ ] 7.7 Ampliar a allow-list de `internal/contextgen/contextgen.go:340-341` (`labels` e o slice
      iterado) para incluir .NET, sem a qual nenhuma das correções acima produz saída visível no
      `AGENTS.md` gerado.
- [ ] 7.8 Rodar a suíte e provar `TestAllLangsAreExhaustivelyWired` verde para `LangDotNet`. Ponto
      de corte: Java só começa depois disso.
- [ ] 7.9 Acrescentar casos de teste a `scripts/test-validators.sh` cobrindo a regex de prova de
      teste de `.agents/scripts/validate-task-evidence.sh:180` — hoje ela não tem nenhum teste.
      Incluir o caso do catch-all: `[^a-z]test[^a-z]` exige caractere não-alfabético DEPOIS de
      "test", então `mvn test -q` passa e `mvn test` no fim da linha falha.
- [ ] 7.10 Ampliar a regex de `.agents/scripts/validate-task-evidence.sh:180` para `mvn` e `gradle`
      e corrigir a dependência do catch-all em caractere posterior, com os casos de 7.9 já no lugar.
- [ ] 7.11 Condicionar `TEST_NAME_RE` (`.agents/scripts/validate-review-evidence.sh:46`,
      `'^(Test|Benchmark|Example)[A-Za-z0-9_/]*$'`) à stack detectada, para aceitar JUnit, xUnit,
      pytest e jest sem afrouxar o gate para Go.
- [ ] 7.12 Introduzir `LangJava` em `internal/skills/skills.go`: `AllLangs` (`:91`), `ParseLang`
      (`:93-99`) e `LangSkills` (`:150-165`), cada `switch` com `default` que falhe.
- [ ] 7.13 Apendar os marcadores de manifesto Java (`pom.xml`, `build.gradle`, `build.gradle.kts`)
      ao FIM de `manifestTypes` e ligar `"Java/Kotlin"` — hoje string decorativa em
      `internal/detect/framework.go:155` que nunca vira `skills.Lang` — ao FIM de
      `DetectPrimaryStack`.
- [ ] 7.14 Propagar Java por `toolchain.go`, `triggers.go` e `contextgen.go:340-341` com os mesmos
      critérios de 7.3 a 7.7.
- [ ] 7.15 Converter `internal/install/verify_lang_filter_test.go:17,56` para usar `skills.AllLangs`
      em vez da lista manualmente expandida — do contrário continuam compilando e passando sem
      cobrir Java, que é pior do que quebrar.
- [ ] 7.16 Corrigir a aritmética hardcoded de `internal/skills/skills_test.go:87`
      (`len(BaseSkills) + len(ComplementarySkills) + 2`), que quebra ao mexer em `LangSkills`.
- [ ] 7.17 Editar `internal/triggers/triggers_test.go:133` — `TestLoad_Fallback` assere que
      `"java"` e `"kotlin"` caem no fallback Go. É o único teste que quebra de propósito e precisa
      ser editado no MESMO commit da correção.
- [ ] 7.18 Criar as fixtures `testdata/java-maven/`, `testdata/java-gradle/` e `testdata/dotnet-api/`
      ao lado das existentes (`testdata/go-monolith/`, `testdata/node-api/`,
      `testdata/polyglot-monorepo/`).
- [ ] 7.19 Acrescentar cenários ao FIM das listas de `internal/detect/stack_coverage_test.go` e de
      `tests/integration/portability_test.go:47-49`, cobrindo as cinco stacks (RF-73).
- [ ] 7.20 Registrar no relatório de execução que `.agents/skills/java-implementation/` está fora do
      escopo deste PRD e por quê.
- [ ] 7.21 Executar os gates da seção "Critérios de Sucesso" e anexar as saídas ao
      `execution_report.md`.

## Detalhes de Implementação

Referência canônica: [`techspec.md`](techspec.md), seção "Rastreabilidade — RF × decisão × arquivo ×
teste" (linha `RF-31 a RF-34`: `detect/toolchain_*.go`, `validate-task-evidence.sh`, provados por
`stack_coverage_test.go` e `test-validators.sh`; linha `RF-73`: `portability_test.go`). O racional
completo da enumeração exaustiva está em
[`adr-005-exaustividade-de-linguagens.md`](adr-005-exaustividade-de-linguagens.md) — não duplicar
aqui.

### Bugs ATIVOS a corrigir (todos verificados por leitura)

1. **`internal/detect/toolchain.go:148-163`** — o `switch winner.lang` tem exatamente três `case`
   (`go`, `node`, `python`) e NENHUM `default`. `.NET` está em `skills.AllLangs` mas não tem `case`.
   Consequência: um repositório C# puro produz `ToolchainResult` vazio e o `AGENTS.md` gerado sai
   SEM nenhum comando de fmt, test ou lint. O mesmo vale para `detectDefault` (`:52-87`).

2. **Divergência Go↔shell** — `.agents/skills/analyze-project/scripts/generate-governance.sh:605-609`
   JÁ emite `dotnet format --verify-no-changes`, `dotnet build --no-restore` e
   `dotnet test --no-build` corretamente, sob `should_include_dotnet` (`:168`). O caminho Go não
   emite nada. Reusar os valores canônicos do shell e de
   `.agents/skills/dotnet-csharp-implementation/SKILL.md`; não inventar.

3. **`internal/detect/framework.go:125-166`** — `DetectPrimaryStack` NÃO detecta C#/.NET, embora o
   caminho shell detecte. `"Java/Kotlin"` aparece em `:155` como string decorativa a partir de
   `pom.xml`/`build.gradle`/`build.gradle.kts` e nunca vira uma `skills.Lang`.

4. **`internal/triggers/triggers.go:59-66`** — `normalizeLang` tem `default: return "go"`;
   `DetectLang` (`:78-98`) só conta `.go`, `.ts/.tsx/.js/.jsx/.mjs/.cjs` e `.py`. Um projeto C#
   recebe hoje gatilhos de revisão de **Go**.

5. **`internal/contextgen/contextgen.go:340-341`** — allow-list hardcoded de três strings
   (`labels := map[string]string{"go": "Go", "node": "Node", "python": "Python"}` e o
   `for _, lang := range []string{"go", "node", "python"}` na linha seguinte). Sem tocar nisso,
   nada do resto produz saída visível.

6. **`.agents/scripts/validate-task-evidence.sh:180`** — a regex de prova forte de testes aceita
   `dotnet test` mas não `mvn` nem `gradle`. O catch-all `[^a-z]test[^a-z]` exige um caractere
   não-alfabético DEPOIS de "test": `mvn test -q` passa e `mvn test` no fim da linha falha. Do ponto
   de vista de quem usa, isso é não-determinístico.

7. **`.agents/scripts/validate-review-evidence.sh:46`** — `TEST_NAME_RE='^(Test|Benchmark|Example)[A-Za-z0-9_/]*$'`
   é Go-only e invalida nomes JUnit, xUnit, pytest e jest. Deve ficar CONDICIONADA à stack
   detectada; ampliá-la globalmente afrouxaria o gate para todas as linguagens, Go inclusive.

### Restrições de ordem que causam regressão se violadas

- **Marcadores novos SEMPRE ao FIM de `manifestTypes`** (`internal/detect/toolchain.go:99-108`). O
  desempate é "primeiro candidato com score máximo vence" (`:138-145`); inserir no meio muda o
  vencedor em monorepo poliglota.
- **`"C#/.NET"` e `"Java/Kotlin"` SEMPRE ao FIM de `DetectPrimaryStack`.**
  `tests/integration/portability_test.go:49` assere `wantStackSub: "Go, Node.js, Python"` como
  substring **contígua** no cenário `polyglot-monorepo`; qualquer inserção intermediária a quebra.
- **Testes de falso verde**: `internal/install/verify_lang_filter_test.go:17,56` listam `AllLangs`
  manualmente expandido e continuariam passando sem cobrir Java. Converter para `skills.AllLangs` na
  mesma tarefa.
- **Aritmética hardcoded**: `internal/skills/skills_test.go:87` usa
  `len(BaseSkills) + len(ComplementarySkills) + 2` e quebra ao mexer em `LangSkills`.
- **`internal/triggers/triggers_test.go:133`** (`TestLoad_Fallback`) assere que `"java"` e
  `"kotlin"` caem no fallback Go. É o único teste que quebra de propósito; editar no mesmo commit.
- **Ampliar a regex de evidência AFROUXA o gate** — risco inverso do usual. `scripts/test-validators.sh`
  ganha os casos ANTES da ampliação, porque hoje a regex de `validate-task-evidence.sh:180` não tem
  nenhum teste automatizado.

### Fora de escopo

Criar `.agents/skills/java-implementation/` não faz parte desta tarefa nem deste PRD. RF-32 cobre
detecção, enumeração e toolchain de Java — não a skill de implementação. O registro dessa exclusão é
entregável (subtarefa 7.20).

## Critérios de Sucesso

- Um repositório C# puro e um repositório Java (Maven e Gradle) geram `AGENTS.md` com comandos de
  fmt, test e lint corretos, provados pelas fixtures `testdata/dotnet-api/`, `testdata/java-maven/`
  e `testdata/java-gradle/`.
- `TestAllLangsAreExhaustivelyWired` (tarefa 3.0) verde para as cinco linguagens.
- Nenhum `switch` sobre linguagem sem `default` que falhe remanescente nos arquivos tocados.
- `scripts/test-validators.sh` cobre a regex de `validate-task-evidence.sh:180`, com caso explícito
  para `mvn test` no fim da linha e para `mvn test -q`.
- `TEST_NAME_RE` condicionada à stack: um relatório de revisão Go com nome de teste não-Go continua
  sendo rejeitado (prova de não-afrouxamento).
- `tests/integration/portability_test.go` mantém `wantStackSub: "Go, Node.js, Python"` do cenário
  `polyglot-monorepo` intacto e passando.
- **Não-regressão (inegociável):** `make test lint vet` verdes, sem exceção e sem teste marcado como
  skip para passar.
- Gates da área tocada verdes: `make check-skills-sync check-scripts-sync test-validators`.
- `make integration` verde.
- `make coverage` respeita os gates de 75% total e 70% por pacote crítico.
- Todo teste que passou a falhar está explicado no `execution_report.md` como quebra intencional
  (`triggers_test.go:133`) ou corrigido — nenhum teste removido para silenciar falha.

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

- `internal/detect/toolchain.go` — `manifestTypes` (`:99-108`), desempate (`:138-145`),
  `switch winner.lang` (`:148-163`), `detectDefault` (`:52-87`)
- `internal/detect/framework.go` — `DetectPrimaryStack` (`:125-166`), `"Java/Kotlin"` (`:155`)
- `internal/triggers/triggers.go` — `normalizeLang` (`:59-66`), `DetectLang` (`:78-98`)
- `internal/triggers/triggers_test.go` — `TestLoad_Fallback` (`:133`)
- `internal/contextgen/contextgen.go` — allow-list de linguagens (`:340-341`)
- `internal/skills/skills.go` — `AllLangs` (`:91`), `ParseLang` (`:93-99`), `LangSkills` (`:150-165`)
- `internal/skills/skills_test.go` — aritmética hardcoded (`:87`)
- `internal/install/verify_lang_filter_test.go` — listas manuais (`:17`, `:56`)
- `internal/detect/stack_coverage_test.go` — `TestStackCoverageSuite` (`:19`)
- `tests/integration/portability_test.go` — cenários e `wantStackSub` (`:45-49`, `:89`)
- `.agents/scripts/validate-task-evidence.sh` — regex de prova de testes (`:180`)
- `.agents/scripts/validate-review-evidence.sh` — `TEST_NAME_RE` (`:46`)
- `.agents/skills/analyze-project/scripts/generate-governance.sh` — comandos .NET canônicos
  (`:605-609`), `should_include_dotnet` (`:168`)
- `.agents/skills/dotnet-csharp-implementation/SKILL.md` — origem dos valores de toolchain .NET
- `scripts/test-validators.sh` — casos novos, antes da ampliação de regex
- `testdata/java-maven/`, `testdata/java-gradle/`, `testdata/dotnet-api/` — fixtures novas
- [`adr-005-exaustividade-de-linguagens.md`](adr-005-exaustividade-de-linguagens.md) — decisão
- [`techspec.md`](techspec.md) — rastreabilidade RF-31 a RF-34, RF-73
