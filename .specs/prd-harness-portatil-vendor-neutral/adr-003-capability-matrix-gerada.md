# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Capability matrix gerada dos invariantes, commitada em JSON + Markdown, com gate estendendo `ValidateParityMatrix`
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório (JailtonJunior94)
- **Relacionados:** [`.specs/prd-harness-portatil-vendor-neutral/prd.md`](prd.md) — RF-18, RF-18.1, RF-19, RF-20, RF-20.1, RF-21

## Contexto

O harness precisa publicar uma capability matrix: quais capabilities cada provedor
(`claude`, `codex`, `copilot`, `opencode`) suporta, em que nível, e qual teste sustenta cada
afirmação. Essa matriz é a superfície pública do compromisso vendor-neutral — é o que um mantenedor
consulta antes de declarar que uma capability é universal, e o que RF-21 usa para proibir que uma
capability exclusiva de um provedor seja tratada como requisito universal.

Hoje a informação existe, mas espalhada e parcialmente não confiável. O levantamento abaixo é o que
motiva esta decisão.

### Onde a informação vive hoje

- `internal/parity/parity.go` registra 26 invariantes. Os tipos estão em
  `internal/parity/parity.go:48-56` (`Common`, `ToolSpecific`, `BestEffort`), `Invariant` em
  `:60-69`, `Snapshot` em `:72-78`, `Result` em `:100-103` e `CheckResult` em `:112-121`.
- O vocabulário nativo por agente está em `internal/runtime/specs/registry.go:16-37`; o validador
  canônico único compartilhado pelos quatro agentes em `registry.go:11-14`; os artefatos por agente
  em `registry.go:39-63`.
- As pré-condições declaradas por agente estão em `internal/runtime/specs/registry.go:202-234`.
- A matriz de paridade de hooks já existente — 4 agentes × 3 pontos = 12 células — está em
  `internal/install/hooks_parity_matrix_test.go:25-30`, `:441-446` e `:504-506`.
- O gate que confronta células contra prova de disparo está em
  `internal/runtime/specs/parity_gate.go:30-72`.

### Sinais que tornam a geração ingênua perigosa

**V-25 — invariantes auto-satisfeitos.** `Checker.Generate`
(`internal/parity/parity.go:196-227`) **injeta** os stubs que `CL03..CL08` e `X03` verificam em
seguida. Esses invariantes não provam nada sobre o instalador real: eles confirmam que o gerador do
snapshot escreveu o que o gerador do snapshot escreve. Derivar dessas checagens uma célula
"suportada" seria institucionalizar um falso positivo — publicar como evidência o que é tautologia.

**V-26 — eixo de escopo ambíguo.** `Level` não é o eixo de escopo. `CL01` (`:345`), `CL02` (`:362`),
`CP01` (`:379`), `CD01` (`:415`) e `CD02` (`:432`) estão marcados `Common` tendo `AppliesTo` de
ferramenta única. Quem derivar "universal vs específico" de `Level` produz uma matriz errada por
construção. O eixo real é `AppliesTo`.

**V-27 — resultado de falha sem consumidor.** `Failures()`
(`internal/parity/parity.go:166-174`) não é consultado por nenhum código de produção. O único
consumidor em produção é `cmd/ai_spec_harness/lint.go:47-77`, que lê apenas `Warnings()`. Ou seja:
hoje uma falha de invariante não derruba nada fora de teste.

**V-24 — testes E2E de paridade não rodam em CI.** `internal/parity/e2e_parity_test.go:1` declara
`//go:build integration`, mas `.github/workflows/test.yml:34` executa `./internal/parity/...`
**sem** `-tags=integration`, e o job `integration` (`.github/workflows/test.yml:159`) não inclui o
pacote. Os testes existem, passam quando executados à mão, e nenhum job os executa. Um teste que não
roda não é evidência — é a forma mais cara de falso positivo, porque custa manutenção e paga zero de
confiança.

**Pré-condição com avaliação stub.** `EvaluateHandshake`
(`internal/runtime/precondition/precondition.go:31-33`) retorna sempre `Unknown`. O `opencode`
declara `handshake` com `requiresExec=true` (`internal/runtime/specs/registry.go:202-234`), enquanto
`claude` não declara pré-condição, `codex` declara `trusted-hash` (`requiresExec=true`) e `copilot`
declara `trusted-folder` (`requiresExec=false`). Uma matriz honesta precisa exibir `handshake` como
estado indeterminado, nunca como suportado.

### Ativos que já funcionam e devem ser reaproveitados

- **Prova de disparo real.** `internal/runtime/specs/parity_dispatch_proof_test.go:40-61` mantém o
  `dispatchProofRegistry`, que mapeia (agente, ponto) → nome de teste de integração; `:254-268`
  valida por parsing AST que esses nomes realmente existem; `:138-184` (`parseDispatchEvidence`)
  consome **apenas** ações `pass`/`skip`/`fail` de `go test -json`, hardening deliberado contra
  prova forjada por `t.Logf`; `:291-308` e `:310-315` fecham o circuito.
- **Extração escopada por formato.** `internal/install/hooks_parity_matrix_test.go` extrai
  configuração por formato — JSON (`:51-64`), TOML (`:199-222`), JS (`:167-197`, com matching de
  chaves balanceadas) — e resolve o alvo com `resolvesToCanonicalValidator` (`:243-284`) em três
  níveis: igualdade de caminho, igualdade de **bytes** e delegação provada por **posição de
  execução** via `executedScriptTargets` (`:362-400`). O hardening G6 está documentado em `:270-274`:
  menção textual não prova delegação.
- **Padrão golden-file.** `internal/contextgen/contextgen_test.go:187-208` (`assertMatchesSnapshot`,
  com `UPDATE_SNAPSHOTS=1` regenerando), `:210-236` (`snapshotDiff`) e `:238-283`, com artefatos
  commitados em `testdata/snapshots/*.agents.md`. É o padrão do repositório para "gerado × commitado".
- **Gate do gate.** `tests/integration/sync_gates_guard_test.go` prova que os gates **falham quando
  deveriam** — por exemplo `:302` (aprovação por vacuidade) e `:332` (drift em todo mirror). É o
  padrão de verificação mais forte do repositório.

### Bug latente identificado durante o levantamento

`.github/workflows/test.yml:52-57` executa `-run TestSnapshot`, mas o teste se chama
`TestContextgen_Snapshots`. O regex de `-run` é ancorado por elemento de nome, então esse padrão
provavelmente **não casa** e o passo passa por vacuidade. O que de fato protege os snapshots hoje é
o `go test ./...` da linha `:29`. O registro fica aqui porque a matriz vai adotar o mesmo padrão
golden-file e não pode herdar a mesma armadilha.

## Decisão

A capability matrix é **gerada** a partir dos invariantes em Go — fonte única — e commitada em
**dois artefatos derivados da mesma geração**:

1. **JSON** — `testdata/capability-matrix.json` (planejado). É o artefato que o gate compara.
   Estável, ordenado deterministicamente, diff linha a linha significativo.
2. **Markdown renderizado** — `docs/capability-matrix.md` (planejado). É o artefato que o mantenedor
   lê. Derivado do mesmo modelo em memória, nunca escrito à mão.

Sobre isso incidem três gates:

- **Gate de sincronia (RF-19):** qualquer divergência entre gerado e commitado — em **qualquer** dos
  dois artefatos — falha o CI. A mensagem de falha documenta o comando de regeneração
  (`UPDATE_SNAPSHOTS=1 go test ./internal/parity/...`, planejado), no mesmo espírito de
  `internal/contextgen/contextgen_test.go:187-208`.
- **Gate de evidência (RF-20):** célula marcada como suportada sem teste associado falha. Este gate
  **estende** `specs.ValidateParityMatrix` (`internal/runtime/specs/parity_gate.go:30-72`),
  reaproveitando `requiredAgents`, `dispatchProven` e `resolve`, além da string literal de falha
  `"no dispatch proof test associated"` em `:65`. Não se cria validador paralelo. A confrontação
  contra artefato instalado (`parity_gate.go:74-93`) e a declaração de chave nativa
  (`internal/runtime/specs/native_config.go:12-54`) continuam sendo o mecanismo de resolução.
- **Gate do gate:** um teste equivalente a `tests/integration/sync_gates_guard_test.go:302,332` prova
  que os dois gates acima falham quando deveriam: matriz vazia não pode ser aprovada por vacuidade, e
  drift em **qualquer** dos dois artefatos precisa ser detectado.

### Saneamento obrigatório da fonte (RF-18.1)

A geração não consome os invariantes crus. Aplica duas regras antes:

1. **Escopo deriva de `AppliesTo`, não de `Level`.** Célula é classificada como universal apenas
   quando `AppliesTo` cobre o conjunto canônico completo de provedores. `Level` continua existindo
   como qualificador de rigor (`Common`/`ToolSpecific`/`BestEffort`), mas perde qualquer papel na
   determinação de escopo — é exatamente a confusão que V-26 documenta em `CL01`, `CL02`, `CP01`,
   `CD01` e `CD02`.
2. **Invariante auto-satisfeito não sustenta célula suportada.** Qualquer invariante cuja verificação
   seja satisfeita pelos stubs injetados por `Checker.Generate`
   (`internal/parity/parity.go:196-227`) — hoje `CL03..CL08` e `X03` — é marcado na geração como
   evidência inválida. Ele pode aparecer na matriz, mas nunca com suporte declarado: aparece como
   estado não comprovado, com a razão explícita.

### Estados honestos

A matriz distingue, no mínimo: `supported` (com teste associado resolvido), `unsupported`,
`provider capability` (RF-21 — exclusiva de um provedor, jamais promovida a requisito universal) e
`unknown`. `EvaluateHandshake` (`internal/runtime/precondition/precondition.go:31-33`), enquanto
stub que retorna sempre `Unknown`, produz célula `unknown` — nunca `supported`.

### Escopo e impacto

Impactados: `internal/parity` (geração e saneamento), `internal/runtime/specs` (extensão do gate),
`.github/workflows/test.yml` (execução em CI, incluindo RF-20.1) e os dois artefatos commitados.
Não impactados: o contrato público da CLI, `internal/install/hooks_parity_matrix_test.go` (que
permanece como está e serve de referência de mecanismo de prova) e o comportamento de
`cmd/ai_spec_harness/lint.go`.

### RF-20.1 — colocar os E2E em CI

Os testes de `internal/parity/e2e_parity_test.go` passam a rodar em CI, seja adicionando
`./internal/parity/...` ao job `integration` de `.github/workflows/test.yml:159`, seja executando o
pacote com `-tags=integration` no job correspondente. O gate do gate cobre a regressão: se o job
deixar de executá-los, a ausência de evidência precisa falhar, não passar em silêncio.

Todo código, identificador, nome de teste e mensagem de erro introduzido por esta decisão é escrito
em inglês e sem comentários, conforme R-STYLE-001 (`hard`). A documentação derivada permanece em
PT-BR.

## Alternativas Consideradas

### (a) YAML declarativo como fonte, com código validando contra ele

- **Descrição:** manter um `capability-matrix.yaml` escrito à mão como fonte de verdade; o código Go
  passaria a validar se o comportamento real corresponde ao declarado.
- **Vantagens:** legível sem executar nada; fácil de editar por quem não mexe em Go; separa
  declaração de implementação.
- **Desvantagens:** cria uma segunda fonte que pode divergir do comportamento real. O YAML seria
  editável independentemente do código, e a divergência só apareceria se alguém escrevesse — e
  mantivesse — o validador cruzado completo. Na prática, transforma o problema de "gerar a verdade"
  no problema mais difícil de "provar que a declaração bate com a verdade", em 26 invariantes e 4
  provedores.
- **Motivo da rejeição:** RF-18 exige fonte única. Duas fontes com reconciliação são exatamente o
  modo de falha que V-25 e V-26 já mostraram existir dentro de uma única fonte — multiplicar fontes
  multiplicaria o risco em vez de reduzi-lo.

### (b) Matriz renderizada sob demanda pelo CLI, nada commitado

- **Descrição:** um subcomando renderiza a matriz na hora, a partir dos invariantes; nenhum artefato
  entra no repositório.
- **Vantagens:** impossível ficar desatualizada; zero custo de regeneração; nenhum ruído de diff.
- **Desvantagens:** a matriz deixa de ser revisável em pull request. Uma mudança que rebaixe uma
  célula de `supported` para `unknown` passaria sem aparecer em nenhum diff — exatamente a classe de
  regressão silenciosa que este PRD existe para eliminar.
- **Motivo da rejeição:** falha o critério explícito de RF-18 de matriz **versionada junto ao
  código**. Sem artefato commitado não há gate de sincronia possível (RF-19), porque não há nada com
  que comparar.

### (c) Apenas Markdown commitado

- **Descrição:** commitar somente a tabela renderizada, e fazer o gate comparar essa tabela.
- **Vantagens:** um único artefato; imediatamente legível; menos arquivos para manter.
- **Desvantagens:** o diff de tabela Markdown é ruidoso — mudar a largura de uma célula reajusta o
  alinhamento e reescreve linhas inteiras que não mudaram semanticamente. E o gate passaria a
  comparar prosa formatada: qualquer reajuste de coluna viraria falso positivo, e o reflexo previsível
  da equipe seria relaxar a comparação (normalizar espaços, ignorar alinhamento) até o gate deixar de
  proteger o conteúdo.
- **Motivo da rejeição:** RF-18 é explícito ao separar o artefato que o gate compara do artefato que
  o humano lê. JSON ordenado dá diff semântico e comparação byte a byte confiável; Markdown dá
  legibilidade. Os dois vêm da mesma geração, então não podem divergir entre si.

## Consequências

### Benefícios Esperados

- Fonte única real: a matriz não pode afirmar algo que o código não sustenta, porque sai do código.
- Falso positivo estrutural eliminado na origem: invariante auto-satisfeito (V-25) deixa de poder
  virar célula suportada, e escopo deixa de derivar do campo errado (V-26).
- Revisão em diff: rebaixamento de capability aparece em pull request, no JSON, com uma linha.
- Reaproveitamento em vez de duplicação: o gate estende `ValidateParityMatrix`
  (`internal/runtime/specs/parity_gate.go:30-72`) e o registry de prova de disparo
  (`internal/runtime/specs/parity_dispatch_proof_test.go:40-61`), mantendo um único ponto de verdade
  sobre o que conta como evidência.
- RF-20.1 converte testes que existiam e não rodavam em evidência efetiva.
- RF-21 ganha mecanismo: `provider capability` é um estado de primeira classe na matriz, não uma
  convenção de prosa.

### Trade-offs e Custos

- Dois artefatos commitados exigem regeneração a cada mudança de invariante; esquecer produz falha de
  CI (é o comportamento desejado, mas é atrito).
- O saneamento de RF-18.1 provavelmente **reduz** o número de células `supported` em relação ao que
  uma leitura ingênua dos 26 invariantes sugeriria. A matriz vai parecer pior antes de ficar melhor —
  isso é correção, não regressão, e precisa ser comunicado como tal.
- Colocar os E2E de paridade em CI aumenta o tempo de pipeline.
- A geração precisa ser determinística (ordenação estável de provedores, capabilities e chaves), o
  que é trabalho real e é pré-condição do gate de sincronia.

### Riscos e Mitigações

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Gate de sincronia aprova por vacuidade (matriz vazia ou geração sem células) | Alto — o gate mais caro do PRD vira decorativo | Gate do gate no padrão `tests/integration/sync_gates_guard_test.go:302`, com caso explícito de matriz vazia |
| Padrão `-run` errado no workflow faz o passo passar sem executar nada | Alto — repete o bug latente de `.github/workflows/test.yml:52-57` | Não usar `-run` com nome parcial; executar o pacote inteiro e cobrir com o gate do gate |
| Geração não determinística produz diff espúrio | Médio — erode a confiança no gate e leva a relaxá-lo | Ordenação canônica explícita, fixada por teste |
| Classificação de invariante auto-satisfeito fica desatualizada ao surgir novo stub em `Checker.Generate` | Alto — retorna o falso positivo que RF-18.1 fecha | Teste que amarra o conjunto de stubs injetados em `internal/parity/parity.go:196-227` ao conjunto marcado como evidência inválida |
| `EvaluateHandshake` sai do estado stub e ninguém atualiza a matriz | Médio — `unknown` permanente mascarando suporte real | A célula `unknown` carrega a razão; regeneração após qualquer mudança em `internal/runtime/precondition/` |
| Divergência entre JSON e Markdown por caminho de geração separado | Médio — dois artefatos contando histórias diferentes | Renderizar ambos a partir do mesmo modelo em memória, na mesma execução |

**Rollback:** os artefatos são aditivos e os gates são novos. Reverter é remover os dois arquivos
commitados e o passo de CI; `ValidateParityMatrix` volta ao comportamento atual porque a mudança é
extensão, não substituição.

## Plano de Implementação

1. **Saneamento da fonte (RF-18.1).** Em `internal/parity`, derivar escopo de `AppliesTo` e marcar
   como evidência inválida os invariantes satisfeitos pelos stubs de `Checker.Generate`
   (`internal/parity/parity.go:196-227`). Teste amarrando o conjunto de stubs ao conjunto marcado.
2. **Modelo da matriz.** Tipo de célula com provedor, capability, estado
   (`supported`/`unsupported`/`provider capability`/`unknown`), razão e referência de teste. Ordenação
   canônica fixada por teste.
3. **Renderização dupla (RF-18).** Um modelo, dois emissores: JSON
   (`testdata/capability-matrix.json`, planejado) e Markdown (`docs/capability-matrix.md`, planejado),
   na mesma execução.
4. **Gate de sincronia (RF-19).** Golden-file no padrão de
   `internal/contextgen/contextgen_test.go:187-208`, com `UPDATE_SNAPSHOTS=1` regenerando e
   `snapshotDiff` (`:210-236`) na mensagem de falha, que documenta o comando de regeneração.
5. **Extensão do gate de evidência (RF-20).** Estender
   `specs.ValidateParityMatrix` (`internal/runtime/specs/parity_gate.go:30-72`) para consumir as
   células da matriz gerada, preservando a resolução via `dispatchProven` e a literal
   `"no dispatch proof test associated"` (`:65`).
6. **E2E em CI (RF-20.1).** Incluir `./internal/parity/...` com `-tags=integration` no job
   `integration` de `.github/workflows/test.yml:159`.
7. **Gate do gate.** Casos adversariais: matriz vazia, drift só no JSON, drift só no Markdown, célula
   `supported` sem teste resolvido, invariante auto-satisfeito tentando sustentar `supported`.
8. **Correção do bug latente.** Ajustar `.github/workflows/test.yml:52-57` para não depender de um
   padrão `-run` que não casa.

**Dependências:** o passo 5 depende de 1–3; o passo 7 depende de 4–6. Os passos 1 e 8 são
independentes e podem ir primeiro.

**Critério de conclusão:** os dois artefatos estão commitados; CI falha ao adulterar qualquer um
deles; CI falha ao marcar célula `supported` sem teste resolvido; os E2E de paridade aparecem no log
do job `integration`; o gate do gate está verde.

## Monitoramento e Validação

- **Sinal primário:** o gate do gate permanece verde. Se ele ficar vermelho, o gate protegido está
  quebrado ou vacuoso.
- **Sinal de cobertura:** a contagem de células `supported` sem teste resolvido deve ser zero; a
  contagem de células `unknown` deve ser rastreada e justificada célula a célula (hoje, no mínimo,
  `handshake` do `opencode`).
- **Sinal de execução:** os testes de `internal/parity/e2e_parity_test.go` aparecem nomeadamente no
  log de CI. Ausência de nome no log é falha, não sucesso.
- **Critérios de sucesso:** nenhum pull request que altere invariante passa sem alterar a matriz
  commitada; nenhuma capability exclusiva de provedor aparece classificada como universal (RF-21).
- **Critérios de revisão:** volume de falsos positivos do gate de sincronia alto o bastante para
  gerar pressão por relaxamento, ou `EvaluateHandshake` deixando o estado stub.

## Impacto em Documentação e Operação

- `docs/capability-matrix.md` (planejado) passa a existir como documento de referência versionado.
- `AGENTS.md` — tabela de gates ganha a linha correspondente à área `internal/parity/`.
- `docs/evidence-gates.md` — registrar o gate de matriz ao lado dos demais gates fail-closed e o
  racional de RF-18.1.
- `docs/degradation-matrix.md` — alinhar com RF-21: capability não suportada produz falha ou relato
  explícito, jamais degradação silenciosa.
- `docs/troubleshooting.md` — entrada para a falha de sincronia, com o comando de regeneração.
- `.github/workflows/test.yml` — execução dos E2E e correção do padrão `-run`.
- Onboarding: a regra operacional "alterou invariante, regenere a matriz" precisa estar visível junto
  das demais rotinas de mirror (`make check-skills-sync`).

## Revisão Futura

- **Marco de revisão:** após o primeiro ciclo completo de release com os gates ativos, ou 90 dias
  após a aceitação, o que vier primeiro.
- **Eventos que invalidam premissas:** `EvaluateHandshake`
  (`internal/runtime/precondition/precondition.go:31-33`) deixar de ser stub; `Checker.Generate`
  parar de injetar stubs (V-25 resolvido na raiz, o que tornaria o saneamento de RF-18.1
  parcialmente obsoleto); `Failures()` (`internal/parity/parity.go:166-174`) ganhar consumidor em
  produção; mudança no conjunto canônico de provedores (RF-15).
- **Condições para substituição:** se o custo de regeneração dupla se mostrar maior que o ganho de
  revisibilidade, ou se surgir mecanismo de renderização que dê diff semântico confiável sobre
  Markdown, esta ADR deve ser substituída por uma nova que reavalie a alternativa (c).
