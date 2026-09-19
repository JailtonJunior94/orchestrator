# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Enumeração de linguagens com teste de exaustividade como pré-requisito do suporte multi-stack
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório
- **Relacionados:** PRD `.specs/prd-hooks-canonicos-vendor-neutral/prd.md` (RF-31, RF-32, RF-33, RF-34, RF-73); techspec `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md`

## Contexto

O objetivo declarado do repositório é servir projetos Go, Node/TypeScript, C#/.NET, Java e Python de
forma igualitária. Isso hoje não acontece. A causa raiz não é ausência de suporte a Java: é que a
enumeração de linguagens está espalhada por pontos independentes do código, **nenhum deles com
`default` que falhe**. Cada ponto degrada em silêncio quando encontra uma linguagem que não conhece,
e a degradação silenciosa é indistinguível de funcionamento correto para quem instala o harness.

Levantamento verificado por leitura:

**1. A enumeração canônica não se defende.** `internal/skills/skills.go:91` declara
`AllLangs = []Lang{LangGo, LangNode, LangPython, LangDotNet}`. `ParseLang`
(`internal/skills/skills.go:93-99`) é um `switch` sem `default` — o caso não coberto cai no
`return "", false` após o bloco. `LangSkills` (`internal/skills/skills.go:150-165`) é um `switch`
sem `default`: uma `Lang` nova simplesmente não contribui skill alguma e o loop segue adiante sem
erro.

**2. O resolvedor de toolchain ignora .NET, que já está em `AllLangs`.**
`internal/detect/toolchain.go:52-87` (`detectDefault`) resolve apenas Go, Node e Python; não há
ramo .NET. O `switch` de resolução por manifesto vencedor em
`internal/detect/toolchain.go:148-163` tem exatamente três `case` (`go`, `node`, `python`) e
**nenhum `default`**. A tabela `manifestTypes` (`internal/detect/toolchain.go:99-108`) lista
`go.mod`, `go.work`, `package.json`, `pyproject.toml` e `requirements.txt` — não lista `*.csproj`,
`*.sln` nem `global.json`. Consequência verificada: um repositório C# puro produz um
`ToolchainResult` vazio (o fallback de Makefile em `internal/detect/toolchain.go:72-76` só dispara
se existir `Makefile`), e o `AGENTS.md` gerado sai **sem nenhum comando de fmt, test ou lint**.

**3. O caminho Go e o caminho shell divergiram.**
`.agents/skills/analyze-project/scripts/generate-governance.sh:605-609` já emite os três comandos
.NET corretamente — `dotnet format --verify-no-changes`, `dotnet build --no-restore`,
`dotnet test --no-build` — sob a guarda `should_include_dotnet`
(`.agents/skills/analyze-project/scripts/generate-governance.sh:168`), e ainda aponta a skill
`dotnet-csharp-implementation` (`generate-governance.sh:481-484`). O caminho Go não emite nenhum
comando .NET. As duas implementações do mesmo gerador de governança produzem resultados diferentes
para a mesma entrada, e não há teste que compare as duas.

**4. A detecção de stack não reconhece C#/.NET e trata Java como decoração.**
`internal/detect/framework.go:125-166` (`DetectPrimaryStack`) reconhece Go (`framework.go:128-134`),
Node.js (`framework.go:136-142`), Python (`framework.go:144-150`), Java/Kotlin via `pom.xml`,
`build.gradle` e `build.gradle.kts` (`framework.go:152-156`) e Rust via `Cargo.toml`
(`framework.go:158-160`). **Não reconhece C#/.NET**, embora `internal/detect/detect.go:90-92` já
trate .NET como linguagem de primeira ordem. E "Java/Kotlin" aparece ali apenas como string
decorativa destinada ao texto do `AGENTS.md`: nunca se converte em `skills.Lang`, porque não existe
`LangJava` em `internal/skills/skills.go:84-89`.

**5. A detecção de linguagem é raso-apenas, e a de toolchain é recursiva — assimetria entre camadas.**
Correção a uma premissa que circulava sobre este ponto: em `internal/detect/detect.go:63-95`
(`detectLangsAt`), **todas** as linguagens são detectadas por `fs.Exists` no diretório informado —
Go (`detect.go:66-72`), Node (`detect.go:74-80`), Python (`detect.go:82-88`) e .NET
(`detect.go:90-92`). Nenhuma delas usa varredura recursiva ali; `hasDotNet`
(`internal/detect/detect.go:100-119`) usa `ReadDir` não-recursivo (`detect.go:108`) pelo mesmo
motivo dos demais. A assimetria real é entre camadas: `internal/detect/toolchain.go:423-464`
(`findManifests` → `findManifestsRecursive` → `findManifestsHelper`) varre recursivamente até
`maxDepth` 4 (`internal/detect/toolchain.go:36`) para `go.mod`, `package.json`, `pyproject.toml` e
`requirements.txt`, e `internal/detect/framework.go:31,57,83,102` faz o mesmo com profundidade 4 —
enquanto .NET não participa de nenhuma dessas varreduras. O resultado prático coincide com o que se
temia: em monorepo com `src/Api/Api.csproj`, o toolchain .NET nunca é resolvido. A compensação
parcial existente é `DetectLangsIn` (`internal/detect/detect.go:32-61`), que reaplica
`detectLangsAt` sobre `focusPaths`, mas ela depende de o chamador fornecer caminhos em foco e não
cobre o resolvedor de toolchain.

**6. Projeto C# recebe gatilhos de revisão de Go hoje — bug de correção ativo.**
`internal/triggers/triggers.go:59-66` (`normalizeLang`) tem `default: return "go"`. `DetectLang`
(`internal/triggers/triggers.go:78-98`) conta apenas extensões `.go` (`triggers.go:82-83`),
`.ts/.tsx/.js/.jsx/.mjs/.cjs` (`triggers.go:84-85`) e `.py` (`triggers.go:86-87`). Arquivos `.cs` e
`.java` não incrementam nada, `DetectLang` retorna `""`, e `""` cai no `default` para `"go"`. O
efeito é que um diff exclusivamente C# recebe os gatilhos de revisão de Go. Isso não é lacuna de
funcionalidade futura: é defeito de correção em produção, e o comentário em
`internal/triggers/triggers.go:32-33` documenta o fallback como se fosse comportamento desejado.

**7. Existe teste que assere o comportamento errado.** `internal/triggers/triggers_test.go:133`
declara `cases := []string{"rust", "java", "", "cpp", "kotlin"}` e verifica que todos caem em Go.
É o único teste que quebra de forma dura e esperada ao corrigir o item 6.

**8. Allow-list hardcoded no gerador de contexto.** `internal/contextgen/contextgen.go:340` declara
`labels := map[string]string{"go": "Go", "node": "Node", "python": "Python"}` e
`internal/contextgen/contextgen.go:341` itera `for _, lang := range []string{"go", "node", "python"}`.
Mesmo que o `ToolchainResult` contivesse as chaves `dotnet` ou `java`, o loop as pularia — a
renderização é governada pela lista literal, não pelo conteúdo detectado.

**9. A regex de prova de teste é não-determinística do ponto de vista do usuário.**
`.agents/scripts/validate-task-evidence.sh:180` contém:

```
(go test|gotestsum|pytest|unittest|npm (run )?test|yarn test|pnpm test|jest|vitest|mocha|make test|make integration|cargo test|dotnet test|ctest|rspec|phpunit|[^a-z]test[^a-z])
```

Java não tem alternativa nomeada — não há `mvn`, `verify`, `gradle` nem `gradlew`. O que existe é o
catch-all final `[^a-z]test[^a-z]`, que exige um caractere não-alfabético **antes e depois** de
"test". Consequência: `mvn test -q` passa (há espaço depois de "test"), e `mvn test` no fim da linha
falha (não há caractere seguinte). O mesmo comando aprova ou reprova conforme a presença de uma flag
irrelevante. Falhar sempre seria preferível: pelo menos seria diagnosticável.

**10. A nomenclatura de teste aceita é exclusivamente Go.**
`.agents/scripts/validate-review-evidence.sh:46` define
`TEST_NAME_RE='^(Test|Benchmark|Example)[A-Za-z0-9_/]*$'`, aplicada em
`.agents/scripts/validate-review-evidence.sh:202`. Nomes de teste JUnit, xUnit, pytest e jest são
invalidados silenciosamente por não casarem com a convenção de prefixo do `testing` de Go.

**11. Há testes que darão falso verde ao ampliar a enumeração.**
`internal/install/verify_lang_filter_test.go:17` e `internal/install/verify_lang_filter_test.go:56`
enumeram `AllSkills([]skills.Lang{skills.LangGo, skills.LangNode, skills.LangPython, skills.LangDotNet})`
manualmente, em vez de usar `skills.AllLangs`. Ao adicionar `LangJava`, esses testes continuam
compilando e passando — e deixam de cobrir Java sem emitir sinal algum. O mesmo padrão aparece em
`internal/skills/skills_test.go:87`, com a aritmética hardcoded
`want := len(BaseSkills) + len(ComplementarySkills) + 2`, que codifica em número a suposição de que
apenas Go contribui duas skills.

**12. Existem restrições de ordem que não podem ser violadas.** Em
`internal/detect/toolchain.go:138-145` o desempate é explícito: "primeiro manifesto com melhor score
vence". Marcadores novos (`*.csproj`, `pom.xml`, `build.gradle`) **devem** ser acrescentados ao
**fim** de `manifestTypes` (`internal/detect/toolchain.go:99-108`), sob pena de mudar o vencedor em
monorepos poliglotas já instalados. E `tests/integration/portability_test.go:49` assere
`wantStackSub: "Go, Node.js, Python"` como substring **contígua** do `AGENTS.md`, verificada em
`tests/integration/portability_test.go:89` — logo `"C#/.NET"` e `"Java/Kotlin"` devem entrar ao
**fim** da lista produzida por `DetectPrimaryStack`, nunca no meio.

**13. Não existe skill de implementação para Java.** `ls .agents/skills/` retorna quatro skills de
linguagem: `go-implementation`, `node-implementation`, `python-implementation` e
`dotnet-csharp-implementation` (mais `object-calisthenics-go`, que é skill de revisão de Go). Não há
`java-implementation/`. O mapa `langImplementationSkills` em `internal/install/install.go:1337-1343`
reflete exatamente esse conjunto de cinco entradas.

**14. Não existem arquivos de gatilho para .NET nem para Java.**
`ls .agents/skills/agent-governance/triggers/` retorna somente `go.yaml`, `node.yaml` e
`python.yaml`.

O denominador comum dos catorze pontos é o mesmo: **adicionar uma constante a `AllLangs` compila,
passa toda a suíte atual e produz comportamento errado em silêncio**. É exatamente o modo de falha
que o repositório já vive com `LangDotNet`, que está em `AllLangs` desde sempre e ainda assim
carrega os defeitos dos itens 2, 4, 5 e 6.

## Decisão

A decisão **não é "adicionar Java"**. A decisão é **tornar a enumeração de linguagens verificável
antes de ampliá-la**, e só então ampliá-la. O escopo é o conjunto de pontos de produção que hoje
dependem da enumeração implícita, mais os validadores shell e seus espelhos de asset.

A aplicação se dá em cinco passos, em ordem estrita:

**Primeiro — teste de exaustividade, escrito antes de qualquer outra alteração.** Criar
`TestAllLangsAreExhaustivelyWired`, que itera `skills.AllLangs` e exige, para cada linguagem:

1. pelo menos uma skill retornada por `LangSkills` para aquela `Lang`;
2. entrada correspondente em `internal/install/install.go:1337` (`langImplementationSkills`);
3. label declarado e presença no loop de renderização de `internal/contextgen/contextgen.go:340-341`;
4. arquivo de gatilhos correspondente em `.agents/skills/agent-governance/triggers/`;
5. `case` no resolvedor de toolchain de `internal/detect/toolchain.go:148-163` e entrada em
   `manifestTypes`;
6. alternativa nomeada na regex de prova de execução de testes de
   `.agents/scripts/validate-task-evidence.sh:180`.

Esse teste nasce **vermelho para `LangDotNet`**, antes de Java sequer ser cogitado. Essa vermelhidão
é o artefato central desta ADR: é a prova mecânica de que o problema é estrutural e não uma lacuna
pontual de feature.

**Segundo — converter degradação silenciosa em erro.** Acrescentar `default` que falha (ou que emite
aviso explícito, quando falhar quebraria contrato público já publicado) aos cinco pontos
identificados: `ParseLang` e `LangSkills` em `internal/skills/skills.go`, o `switch` de resolução em
`internal/detect/toolchain.go:148-163`, o `default: return "go"` de
`internal/triggers/triggers.go:63-64`, e a allow-list literal de
`internal/contextgen/contextgen.go:340-341`.

**Terceiro — completar .NET antes de introduzir Java.** `LangDotNet` já está em `AllLangs` e carrega
três defeitos ativos: ausência no resolvedor de toolchain (item 2), ausência em `DetectPrimaryStack`
(item 4) e recebimento de gatilhos de Go (item 6). Corrigir .NET valida o mecanismo com raio de
regressão menor, porque a linguagem já é instalada e já possui skill própria.

**Quarto — introduzir `LangJava`**, percorrendo os pontos de produção enumerados na techspec e os
espelhos de asset mantidos por `scripts/sync-skills.sh`, sempre respeitando as restrições de ordem
do item 12 do Contexto.

**Quinto — condicionar `TEST_NAME_RE` à stack detectada**, em vez de ampliá-lo globalmente. Ampliar
`.agents/scripts/validate-review-evidence.sh:46` de forma global afrouxaria o gate de evidência para
**todas** as linguagens, Go inclusive — o oposto do objetivo.

## Alternativas Consideradas

**A1 — Adicionar Java direto, sem teste de exaustividade.**
*Vantagem:* entrega aparentemente mais rápida; o diff inicial é pequeno.
*Desvantagem:* acrescentar `LangJava` a `internal/skills/skills.go:91` sem tocar nos demais pontos
compila, passa toda a suíte atual e produz comportamento errado em silêncio — projeto Java receberia
gatilhos de Go por `internal/triggers/triggers.go:63-64`, `AGENTS.md` sem comandos de validação por
`internal/detect/toolchain.go:148-163` e renderização vazia por
`internal/contextgen/contextgen.go:341`.
*Motivo da rejeição:* é precisamente o modo de falha que o repositório já sofre com .NET. Repetir o
procedimento que gerou o defeito não pode ser a correção do defeito.

**A2 — Centralizar tudo numa única tabela de linguagem.**
*Vantagem:* fonte única real; elimina a classe de erro na origem em vez de detectá-la.
*Desvantagem:* exigiria refatorar de uma vez todos os call-sites que hoje decidem por enumeração
implícita, incluindo os validadores shell e os assets espelhados em `internal/embedded/assets/`,
`.claude/` e `.github/` — um único lote grande e de reversão difícil, tocando simultaneamente
detecção, instalação, geração de contexto e gates de evidência.
*Motivo da rejeição:* raio de regressão desproporcional. O teste de exaustividade obtém a mesma
garantia de forma incremental e reversível, e mantém a porta aberta para a centralização depois,
quando ela puder ser feita com rede de segurança.

**A3 — Ampliar apenas a regex de prova de teste para aceitar `mvn` e `gradle`.**
*Vantagem:* destrava Java no gate de evidência com uma linha; resolve o sintoma mais visível e mais
reclamado.
*Desvantagem:* deixa intactos os gatilhos (item 6), o resolvedor de toolchain (item 2) e o gerador
de contexto (item 8). Java passaria a ter evidência aceita enquanto continuaria recebendo guidance
de revisão de Go e `AGENTS.md` sem comandos — estado pior que o atual, porque o gate verde
mascararia a governança errada.
*Motivo da rejeição:* aprovar evidência sem corrigir a orientação subjacente contraria a política de
evidência de `.claude/rules/governance.md`.

**A4 — Remover .NET de `AllLangs` para reduzir escopo.**
*Vantagem:* tornaria a enumeração internamente consistente imediatamente, sem trabalho de ampliação.
*Desvantagem:* seria regressão de funcionalidade já entregue.
*Motivo da rejeição:* a skill `dotnet-csharp-implementation` existe em `.agents/skills/`, está
registrada em `internal/install/install.go:1341` e é instalada; o gerador shell já emite comandos
.NET em `generate-governance.sh:605-609`. Remover .NET quebraria instalações existentes e
contrariaria RF-33.

## Consequências

### Benefícios Esperados

- Falhas silenciosas passam a ser falhas de teste. Os quatro defeitos de .NET e a divergência
  Go↔shell deixam de depender de inspeção manual para serem percebidos.
- Projeto C#/.NET deixa de receber gatilhos de revisão de Go — correção de defeito em produção, não
  funcionalidade nova.
- Java entra com rede de segurança: o teste de exaustividade reprova a introdução parcial antes do
  merge, em vez de deixá-la chegar ao usuário.
- A divergência entre `internal/detect/toolchain.go` e
  `.agents/skills/analyze-project/scripts/generate-governance.sh` fica visível e rastreável, criando
  base para a unificação futura dos dois geradores.
- `ai-spec install .` passa a produzir `AGENTS.md` com comandos de validação não vazios para as cinco
  stacks, atendendo RF-31 e RF-73.

### Trade-offs e Custos

- **O teste de exaustividade adiciona atrito real a cada linguagem nova.** Isso é custo deliberado e
  assumido: a sexta linguagem exigirá tocar seis pontos antes de o build ficar verde. O atrito é o
  mecanismo, não um efeito colateral dele.
- Criar `.agents/skills/java-implementation/` nos quatro espelhos é trabalho de conteúdo, não de
  código, e está **fora do escopo deste PRD**. O PRD declara isso explicitamente em Fora de Escopo
  (`.specs/prd-hooks-canonicos-vendor-neutral/prd.md:504-506`): RF-32 cobre detecção, enumeração e
  resolução de toolchain para Java; escrever a skill é trabalho de outro PRD. Esta ADR registra que o
  teste de exaustividade deve, portanto, tolerar Java sem skill própria ou ser introduzido junto com
  um placeholder explícito e registrado — a decisão de qual das duas formas pertence à techspec, mas
  a exceção precisa ser visível e datada, nunca implícita.
- Ampliar a regex de `validate-task-evidence.sh` aumenta a superfície de comandos aceitos como prova
  de teste, o que é afrouxamento do gate.

### Riscos e Mitigações

- **`internal/triggers/triggers_test.go:133` quebra de propósito.**
  *Impacto:* suíte vermelha no commit de correção do item 6.
  *Mitigação:* editar o teste no mesmo commit, removendo `"java"` e `"kotlin"` da lista de casos que
  devem cair em Go, e acrescentando asserção positiva de que essas linguagens resolvem para o próprio
  arquivo de gatilhos quando ele existir.

- **Testes de falso verde continuam passando sem cobrir Java.**
  *Impacto:* cobertura aparente sem cobertura real — o risco mais insidioso, porque produz confiança
  injustificada.
  *Mitigação:* trocar `internal/install/verify_lang_filter_test.go:17` e
  `internal/install/verify_lang_filter_test.go:56` para usar `skills.AllLangs`, e substituir a
  aritmética de `internal/skills/skills_test.go:87` por contagem derivada de
  `LangSkills(skills.AllLangs)`.

- **Ampliar a regex afrouxa o gate — risco de direção inversa ao usual.**
  *Impacto:* comandos que não são execução de teste passariam a ser aceitos como prova.
  *Mitigação:* `scripts/test-validators.sh` ganha casos **antes** da ampliação, porque hoje a regex
  de `validate-task-evidence.sh:180` não tem nenhum teste. Os casos devem cobrir, no mínimo:
  `mvn test` no fim da linha (hoje falha), `mvn test -q` (hoje passa), `./gradlew test`, e pelo menos
  um negativo que deve continuar reprovando. Só depois de os casos existirem a regex é alterada, e a
  alternativa nomeada para Java substitui a dependência do catch-all `[^a-z]test[^a-z]`.

- **Plano de rollback.** Cada linguagem é aditiva e removível isoladamente: reverter o commit de uma
  linguagem restaura o estado anterior sem afetar as demais, porque nenhum dos pontos alterados é
  compartilhado entre linguagens além da própria enumeração. O `default` que falha é a única
  alteração transversal, e é revertível em um commit.

## Plano de Implementação

Ordem estrita, cada etapa concluída antes da seguinte:

1. **Teste de exaustividade, vermelho.** Escrever `TestAllLangsAreExhaustivelyWired` com as seis
   verificações da seção Decisão. Confirmar que reprova para `LangDotNet`. Registrar a saída vermelha
   como evidência — ela é a justificativa desta ADR.
2. **`default` nos cinco pontos.** `ParseLang`, `LangSkills`, `switch` de toolchain, `normalizeLang`
   e a allow-list de `contextgen`. Falha explícita onde não houver contrato público em risco; aviso
   explícito onde houver, com o motivo registrado.
3. **.NET completo.** Ramo .NET em `detectDefault` e `case` no `switch` de
   `internal/detect/toolchain.go:148-163`; `*.csproj`, `*.sln` e `global.json` ao **fim** de
   `manifestTypes`; reconhecimento de C#/.NET ao **fim** de `DetectPrimaryStack`; `.cs` em
   `DetectLang` e `dotnet.yaml` em `.agents/skills/agent-governance/triggers/`; label e entrada no
   loop de `contextgen`.
4. **Verde.** `TestAllLangsAreExhaustivelyWired` passa para as quatro linguagens existentes. Esta é a
   porta de entrada para a etapa seguinte — Java não começa antes disso.
5. **Java.** `LangJava` na enumeração e em cada um dos seis pontos verificados pelo teste,
   respeitando as restrições de ordem do item 12 do Contexto: marcadores `pom.xml`, `build.gradle` e
   `build.gradle.kts` ao **fim** de `manifestTypes`; `"Java/Kotlin"` já está ao fim de
   `DetectPrimaryStack` (`internal/detect/framework.go:152-156`) e deve ali permanecer, passando a
   derivar de `skills.Lang` em vez de string literal. Espelhos de asset regenerados por
   `scripts/sync-skills.sh` e validados por `make check-skills-sync check-scripts-sync`.
6. **Fixtures.** `testdata/java-maven`, `testdata/java-gradle` e `testdata/dotnet-api`.
7. **Cenários.** Casos novos em `internal/detect/stack_coverage_test.go` e em
   `tests/integration/portability_test.go`, sempre acrescentados ao **fim** das listas, para não
   alterar a substring contígua asserida em `tests/integration/portability_test.go:49`.

Dependências: a etapa 5 depende do verde da etapa 4; a alteração da regex dentro da etapa 5 depende
dos casos novos em `scripts/test-validators.sh`. A adoção considera-se concluída quando o teste de
exaustividade está verde para as cinco linguagens sem nenhuma exceção declarada além da registrada
para a skill Java.

## Monitoramento e Validação

- **Critério de sucesso primário:** `TestAllLangsAreExhaustivelyWired` verde para as cinco
  linguagens, sem `t.Skip` e sem lista de exceções além da documentada nesta ADR.
- **Critério de sucesso funcional:** `ai-spec install .` executado sobre a fixture de cada stack
  produz `AGENTS.md` com comandos de fmt, test e lint **não vazios**. A ausência de comandos é
  exatamente o sintoma observável hoje em repositório C# puro, e é o que deve deixar de ocorrer.
- **Sinais a acompanhar:** `make check-skills-sync check-scripts-sync` para os espelhos de asset;
  `scripts/test-validators.sh` para a regex de prova de teste; `make coverage` contra os gates de 75%
  total e 70% por pacote crítico, com atenção a `internal/detect`, `internal/triggers` e
  `internal/contextgen`.
- **Critério de revisão da decisão:** se uma linguagem nova exigir exceção ao teste de exaustividade,
  a abstração está errada — o sinal deve disparar reconsideração de A2 (tabela centralizada), não a
  concessão da exceção.

## Impacto em Documentação e Operação

- `AGENTS.md`: tabela "Regras por Linguagem" — acrescentar Java quando a skill existir, e registrar
  no intervalo a lacuna deliberada.
- `docs/guia-instalacao-universal.md`: seção de linguagens suportadas e exemplos de `--langs`.
- Help e exemplos da flag `--langs`: `cmd/ai_spec_harness/install.go:43` descreve hoje
  `"Linguagens: go,node,python ou all"`, **omitindo `dotnet`**, que já é aceito por `ParseLang`; os
  exemplos em `cmd/ai_spec_harness/install.go:32-37` repetem a omissão. A validação ocorre em
  `cmd/ai_spec_harness/flags.go:80-89`, que delega a `ParseLang` e portanto já aceita `dotnet` — a
  divergência é puramente de documentação da flag e induz o usuário a erro.
- `docs/troubleshooting.md`: entrada nova para o sintoma "AGENTS.md sem comandos de validação",
  apontando a stack não resolvida como causa.
- `.agents/skills/analyze-project/scripts/generate-governance.sh`: registrar que o bloco .NET
  (linhas 605-609) é hoje a referência de comportamento correto para o caminho Go, até a unificação.

## Revisão Futura

Revisar esta ADR na introdução da **sexta linguagem**, ou antes disso caso a divergência Go↔shell do
gerador de governança seja eliminada por unificação dos dois caminhos — nesse cenário, o item 3 do
Contexto deixa de valer e a alternativa A2 volta a ser avaliável com raio de regressão aceitável.
Invalida as premissas desta ADR: a criação de `.agents/skills/java-implementation/` por outro PRD, ou
qualquer alteração que torne a enumeração de linguagens dado de configuração em vez de constante de
código.
