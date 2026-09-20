# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Parsing estrutural sem truncamento, decisão por palavra-de-comando e negação por ausência de alvo
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório
- **Relacionados:** PRD [`prd.md`](prd.md) (RF-18 a RF-24, RF-57, RF-68), [`techspec.md`](techspec.md), [ADR-001](adr-001-hook-contract-fonte-unica-projecoes.md), [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md), [ADR-005 do PRD dependente](../prd-harness-portatil-vendor-neutral/adr-005-gate-operacao-git-destrutividade.md), `.claude/rules/governance.md` (R-SEC-001)

## Contexto

O gate canônico de operação Git, entregue na tarefa 9.0 do PRD dependente, é o único mecanismo que
impede um agente de executar `git commit` e `git push` não solicitados nas quatro CLIs. Ele é
invocado por `.agents/hooks/validate-preload.sh:61-69`, no ponto `BeforeTool`, que a pesquisa oficial
confirmou ser o **único ponto com semântica de negação real** em Claude, Codex, Copilot e OpenCode.

Uma auditoria adversarial executou o gate contra entradas construídas. Os resultados abaixo não são
análise estática: cada um foi verificado ponta a ponta contra o script real.

**Existe um bypass determinístico e incondicional.** `.agents/lib/parse-hook-input.sh:53` aplica
`head -c 65536` ao payload antes de qualquer parsing. Um comando precedido de aproximadamente 70 KB
de conteúdo arbitrário e seguido de `; git push origin main` trunca o JSON, faz o `json.load` do
Python lançar exceção, deixa o fallback por `grep` sem aspa de fechamento, produz `command_text`
vazio — e `.agents/scripts/git-operation-gate.sh:39-41` responde a comando vazio com `exit 0`. O
resultado verificado é **EXIT=0**, contra **EXIT=2** do controle `git push origin main`. O bypass
contorna simultaneamente o critério Git e o critério destrutivo, em qualquer provedor, sem usar
nenhuma das variáveis de escape auditadas.

**A postura de comando vazio é contraditória dentro do próprio harness.**
`.agents/hooks/validate-preload.sh:109` implementa explicitamente a regra "ausência de alvo é
negação, nunca aprovação". `.agents/scripts/git-operation-gate.sh:39-41` faz exatamente o oposto.

**A regex decide por posição léxica, não por estrutura.** A expressão em
`.agents/scripts/git-operation-gate.sh:70` é
`(^|[[:space:]])git([[:space:]]+-[^[:space:]]+)*[[:space:]]+(commit|push)([[:space:]]|$)`. O grupo de
flags repetido só casa tokens que começam com `-`, então qualquer flag com argumento separado quebra
a correspondência. Verificados com EXIT=0: `git -C /repo push` e `git -c user.name=x commit -m y`.
Escapam pela mesma causa — `git` precedido de caractere que não é espaço nem início de linha —
`$(echo git) push`, `eval "git push"`, `sh -c "git push"`, `/usr/bin/git push` e `git.exe push`. E
por não estarem na alternação, escapam integralmente `git reset --hard`, `git clean -fd`,
`git checkout -- .`, `git restore .` — todos no escopo mínimo declarado pela User Story.

**Os falsos positivos são simétricos ao problema.** Como não há noção de palavra-de-comando versus
argumento, `echo git commit` e `man git commit` **bloqueiam** com EXIT=2. E como a segmentação em
`:100` é `tr ';|&' '\n'`, que ignora aspas, `echo "a|git commit -m x"` também bloqueia (exit 2
verificado). O caso `echo "a|git push"` devolve exit 0 — a aspa de fechamento cai imediatamente
após `push` e derrota a âncora `([[:space:]]|$)` da regex, o que mostra que o falso positivo depende
do que vem depois do subcomando e não de uma regra compreensível pelo usuário.

**A policy declarada é decorativa.** `internal/harness/contract.go:12-15` define
`GitPolicy{AutoCommit, AutoPush}`. Uma busca por esses campos em `internal/` e `cmd/` retorna
**apenas** a declaração e asserções de parsing em `contract_test.go` — **zero call-sites de
produção**. `internal/doctor/doctor.go` lê somente `contract.Version`. `.agents/harness.yaml` não
existe no repositório. Nenhum script shell lê os campos. Portanto `auto_commit: true` não libera
nada e `auto_commit: false` não impede nada: a decisão real vem só de
`GOVERNANCE_GIT_OPERATION_CONFIRMED` e `GOVERNANCE_GIT_OPERATION_MODE`
(`.agents/scripts/git-operation-gate.sh:114-115`).

**O rastro de auditoria não tem leitor.** `.aispec/governance-escapes.log` é TSV de cinco colunas
escrito em `git-operation-gate.sh:50-52` e `:62-64`, coberto pelo `.gitignore:34` via o padrão
`.aispec/`, sem rotação, sem validação e sem nenhum leitor no repositório. O campo de comando é
gravado cru: um comando contendo tabulação corrompe o TSV e nada detecta. As linhas presentes hoje
são resíduo da própria suíte de testes.

**A cobertura é quase inexistente.** `scripts/test-validators.sh` e `scripts/test-hooks.sh` não têm
**nenhuma** referência ao gate. A única cobertura são dois literais — `git commit -m x` e
`git push origin main` — em `tests/integration/hooks_matrix_dispatch_test.go`. Todos os escapes acima
são invisíveis à suíte.

Isto é violação ativa de R-SEC-001, que exige tratar input externo como não confiável e construir
subprocessos com argumentos explícitos.

## Decisão

Reconstruir a cadeia de decisão do gate em três camadas, cada uma corrigindo uma classe de falha
distinta. A ordem importa: corrigir a regex sem corrigir o truncamento deixaria o bypass intacto.

**Camada 1 — Ingestão de payload sem perda.** Substituir `.agents/lib/parse-hook-input.sh` por
`.agents/lib/hook-payload.sh`, que:

- **não trunca.** O limite de 64 KiB é removido. Payload grande é caso legítimo — um `Write` de
  arquivo extenso o produz naturalmente — e silenciosamente perder a cauda é o mecanismo do bypass;
- **não tem fallback por `grep`.** O fallback atual casa `"command":"…"` em qualquer posição do texto
  bruto, o que é tanto burlável quanto gerador de falso positivo. Se o parsing estrutural falhar, o
  resultado é erro tipado, não um palpite;
- **falha fechado.** Payload que não decodifica produz `DecisionError`, que em hook crítico vira
  bloqueio pela [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md);
- **decodifica o envelope versionado** de `internal/hookcontract`, não um formato ad hoc por
  provedor.

**Camada 2 — Decisão por palavra-de-comando.** A avaliação deixa de ser correspondência de regex
sobre um segmento de texto e passa a ser análise da linha de comando com noção de estrutura:
identificação do executável efetivo — resolvendo `env`, `command`, caminho absoluto e nome com
extensão — separação de flags globais de `git` do subcomando, e respeito a aspas na segmentação.

Com isso, `git -C /repo push` e `git -c user.name=x commit` passam a ser reconhecidos, e
`echo git commit` e `man git commit` passam a ser permitidos, porque `git` não é a palavra-de-comando.

Construções que escondem o comando dentro de uma string — `eval "git push"`, `sh -c "git push"`,
`$(echo git) push` — **não** são resolvidas por análise estática, e esta decisão **não finge
resolvê-las**. Elas passam a ser tratadas como classe própria: invocação de interpretador com
conteúdo dinâmico é marcada como operação que exige aprovação, com `policy_id` próprio, em vez de
ser silenciosamente permitida. Isso troca um escape silencioso por uma fricção explícita.

**Camada 3 — Escopo derivado da policy.** Criar `internal/hookpolicy`, que deriva a lista de
operações interceptadas a partir de `internal/harness.Contract` e a materializa em
`.agents/generated/git-scope.json` com um fingerprint SHA-256. O shell lê esse artefato em vez de
manter blacklist própria. Um gate de CI recalcula o fingerprint a partir do contrato e falha na
divergência.

Este é o elo que **hoje não existe**. Sem ele, RF-19 — "a lista deve ser derivada da policy declarada
e não de blacklist mantida à parte" — seria uma afirmação não verificável, exatamente como a
afirmação equivalente que já existe hoje e é falsa.

O escopo passa a incluir, além de `commit` e `push`: `reset --hard`, `clean` com remoção efetiva,
`checkout` e `restore` destrutivos.

Force push **já é interceptado hoje** pela alternação `push` — `git push --force`, `-f`,
`--force-with-lease` e refspec com `+` todos retornam exit 2, verificado por execução. O que RF-21
acrescenta não é interceptação e sim **classificação de risco**: force push passa a carregar
`policy_id` próprio e a exigir aprovação distinta da de um push comum. A cobertura genuinamente nova
está nas formas combinadas que hoje escapam por causa do grupo de flags — `git -C /repo push --force`
e `/usr/bin/git push -f`. Pelo mesmo motivo, `env git push` e `command git push` já bloqueiam (exit 2
verificado) e entram na matriz como guardas de **não-regressão**, não como ampliação de escopo.

**Complemento — auditoria com leitor.** O TSV é substituído por `.aispec/hook-decisions.jsonl`, com
uma decisão por linha, campos escapados por serialização JSON e um leitor em `internal/hookaudit`.
O TSV é mantido em paralelo por um release e removido depois.

## Alternativas Consideradas

**A1 — Elevar o limite de 64 KiB para um valor maior, mantendo o truncamento.**
*Vantagens:* mudança de um caractere; nenhum risco de regressão.
*Desvantagens:* qualquer limite finito é contornável por um payload maior. O bypass deixaria de ser
trivial e continuaria determinístico. Troca uma vulnerabilidade por um número.
*Rejeitada* por não resolver a classe do problema.

**A2 — Manter o truncamento e tratar payload truncado como bloqueio.**
*Vantagens:* fecha o bypass sem mudar o parser; barato.
*Desvantagens:* transforma todo `Write` de arquivo grande em bloqueio. O falso positivo seria
frequente e de alto atrito, e o caminho previsível seria o usuário desligar o gate — que é pior que
o estado atual.
*Rejeitada* por produzir um gate que ninguém mantém ligado.

**A3 — Escrever o gate inteiro em Go e invocá-lo pelo binário `ai-spec`.**
*Vantagens:* parsing estrutural nativo; testabilidade muito superior; elimina a dependência de
`python3`/`jq` no host.
*Desvantagens:* o gate roda em `BeforeTool`, isto é, antes de **cada** chamada de ferramenta. Passar
a exigir o binário nesse caminho crítico introduz dependência dura onde hoje há degradação
controlada, e `validate-task-evidence.sh:416-433` já demonstra o custo de acoplar a versão mínima do
binário a um gate. Além disso, o OpenCode executa o validador por `spawnSync` com timeout de 8000 ms
(`.opencode/plugin/governance.js:16`); o custo de inicialização passaria a competir com esse
orçamento.
*Rejeitada para esta entrega, registrada como evolução provável.* A Camada 3 já move a **definição**
do escopo para Go; mover a **avaliação** é o passo seguinte natural, quando houver medição de latência
que o justifique.

**A4 — Usar um parser de shell completo, do tipo `mvdan/sh`, para analisar a linha de comando.**
*Vantagens:* correção muito superior na separação de palavras, aspas e substituições.
*Desvantagens:* dependência nova e pesada para um problema que, no lado shell, não pode consumi-la; e
no lado Go recairia na objeção A3. Contraria a diretriz de preferir bibliotecas pequenas e de não
introduzir dependência sem demanda concreta.
*Rejeitada por ora*, e explicitamente reavaliável se a análise estrutural em shell se mostrar
insuficiente sob a matriz adversarial.

**A5 — Manter a blacklist no script e apenas documentar que ela deveria refletir a policy.**
*Vantagens:* zero trabalho.
*Desvantagens:* é literalmente o estado atual, e a auditoria mostrou que a afirmação de derivação já
é feita e é falsa. Documentar uma relação inexistente é pior que não documentá-la.
*Rejeitada.*

**A6 — Aceitar os falsos positivos `echo git commit` e `man git commit` como custo de segurança.**
*Vantagens:* nenhuma mudança na lógica de decisão.
*Desvantagens:* falso positivo em comando de leitura treina o usuário a usar a variável de escape por
hábito, o que degrada o gate justamente nos casos em que ele importa. RF-22 exige fixture para cada
falso positivo conhecido — a exigência do PRD é eliminá-los, não tolerá-los.
*Rejeitada.*

## Consequências

### Benefícios Esperados

- O único bypass verificado do harness é fechado, e a correção é da classe, não do caso.
- A postura de "ausência de alvo é negação" passa a ser uniforme entre os dois gates.
- Quatro operações destrutivas do escopo mínimo da User Story passam a ser interceptadas.
- Dois falsos positivos confirmados desaparecem, e cada um ganha fixture de não regressão.
- `GitPolicy` deixa de ser campo decorativo e passa a ter efeito verificável.
- O rastro de auditoria ganha leitor, escape correto de caracteres e formato validável.

### Trade-offs e Custos

- **Remover o limite de payload aumenta o custo do parsing** em proporção ao tamanho da entrada, em
  um caminho executado a cada chamada de ferramenta. É custo real, e o orçamento de 8000 ms do
  OpenCode é o teto prático a respeitar.
- **Análise estrutural em shell é mais código e mais frágil** que uma regex de uma linha. A mitigação
  é a matriz de testes, não a elegância do script.
- **A classe `eval`/`sh -c` passa a exigir aprovação**, o que introduz fricção nova em fluxos
  legítimos que usem interpretador. É escolha deliberada: fricção explícita no lugar de escape
  silencioso.
- **`.agents/generated/git-scope.json` é artefato gerado e versionado**, com o custo de manutenção e
  de sincronia que todo artefato assim tem, mais um espelho obrigatório em
  `internal/embedded/assets/`.
- **Dois formatos de log convivem por um release.**

### Riscos e Mitigações

| Risco | Impacto | Mitigação | Rollback |
|---|---|---|---|
| A reescrita do parsing introduz um escape novo que ninguém previu | Alto | A matriz adversarial é escrita **antes** da correção, com os 20 casos verificados na auditoria, e roda em `scripts/test-validators.sh`, que hoje tem zero referências ao gate | O script antigo é preservado até a matriz ficar verde |
| Remover o fallback por `grep` quebra hosts sem `python3` nem `jq` | Alto | `.agents/lib/hook-payload.sh` declara a dependência e **falha fechado** na ausência dela, em vez de degradar. O `doctor` passa a verificar a presença do interpretador | Restaurar o fallback reintroduz o falso positivo e o burlável — deve ser registrado como dívida, não como solução |
| Sem truncamento, um payload muito grande estoura o orçamento de 8000 ms do OpenCode | Médio | Medir sob a suíte de portabilidade com payloads de 1 KiB, 64 KiB, 65 KiB e 1 MiB; o timeout do OpenCode já é fail-closed (`governance.js:187,217` tratam timeout como negação) | Reintroduzir limite, agora com bloqueio explícito em vez de `exit 0` |
| Ampliar o escopo para `reset --hard`, `clean`, `checkout` e `restore` gera falso positivo em fluxo legítimo | Médio | Cada operação nova entra com fixture de caso permitido e caso bloqueado; a variável de escape existente continua disponível e auditada | Remoção por operação, já que o escopo é lista derivada e não código |
| O fingerprint do escopo diverge do contrato sem ninguém perceber | Médio | Gate de CI recalcula e falha na divergência — é a mesma mecânica que já protege os espelhos de skills e hooks | Não aplicável |
| Tratar `eval`/`sh -c` como operação aprovável bloqueia fluxo interno do próprio harness | Médio | Auditar os scripts do repositório antes: `subagent-stop-wrapper.sh` e `post-execute-task.sh` invocam `bash` com caminho fixo, não conteúdo dinâmico — precisa ser confirmado caso a caso | Rebaixar a classe para aviso |
| A suíte de testes polui `.aispec/hook-decisions.jsonl` como já polui o TSV | Baixo | O caminho do log é redirecionável por variável; os testes passam a usar `t.TempDir()` | Não aplicável |

## Plano de Implementação

1. **Escrever a matriz adversarial e a matriz de falso positivo em `scripts/test-validators.sh`**, com
   os resultados esperados derivados do contrato. Vermelha em pelo menos onze casos.
2. **Criar `.agents/lib/hook-payload.sh`** sem truncamento e sem fallback, com falha fechada, e o
   teste de fronteira de payload em `tests/integration/hook_payload_boundary_test.go`.
3. **Alinhar a postura de comando vazio** em `git-operation-gate.sh:39-41` com
   `validate-preload.sh:109`. Este passo isolado já fecha o bypass.
4. **Substituir a decisão por regex pela análise de palavra-de-comando**, até a matriz do passo 1
   ficar verde.
5. **Criar `internal/hookpolicy`** com `GitScope` derivado do contrato e o teste
   `TestGitScope_DerivesFromContract`, que nasce vermelho por não existir o elo.
6. **Gerar `.agents/generated/git-scope.json`** com fingerprint, espelhar em
   `internal/embedded/assets/` e ligar o gate de CI.
7. **Ampliar o escopo** para as quatro operações destrutivas faltantes, uma por vez, cada uma com seu
   par de fixtures.
8. **Introduzir `internal/hookaudit`** e o JSONL, em paralelo ao TSV.
9. **Remover o TSV** e o `parse-hook-input.sh` antigo, depois de um release.

A adoção está concluída quando as duas matrizes estiverem verdes, o gate de fingerprint estiver
ativo, e `scripts/test-validators.sh` cobrir o gate — que hoje não cobre.

## Monitoramento e Validação

- **Sinal primário:** matriz adversarial verde no CI. É a única evidência que distingue esta decisão
  de uma afirmação.
- **Sinal de regressão:** qualquer caso da matriz de falso positivo que volte a bloquear.
- **Telemetria:** entradas `hook.decision` com `policy_id` e `hook.duration_ms` em
  `.agents/telemetry.log`, no formato `<ts> chave=valor` já estabelecido, opt-in por
  `GOVERNANCE_TELEMETRY=1`. A latência do gate em `BeforeTool` é a métrica de custo a acompanhar,
  porque é o caminho mais quente do harness.
- **Auditoria:** contagem de escapes por motivo, lida de `.aispec/hook-decisions.jsonl` — leitura que
  hoje é impossível.
- **Critério de sucesso:** nenhum dos vinte casos adversariais produz EXIT=0; nenhum dos cinco casos
  de falso positivo produz EXIT=2; `auto_commit: true` no contrato altera demonstravelmente o escopo.
- **Critério para revisar ou reverter:** se a análise estrutural em shell exigir mais de um caso
  especial por classe de escape, a alternativa A3 — avaliação em Go — deve ser reconsiderada com a
  medição de latência em mãos. Se a latência em `BeforeTool` passar a ser percebida, o mesmo.

## Impacto em Documentação e Operação

- `docs/evidence-gates.md` — o escopo real do gate e o mecanismo de derivação.
- `docs/troubleshooting.md` — o que fazer quando uma operação legítima é bloqueada, e por que a
  variável de escape não deve virar hábito.
- `docs/hooks-canonicos.md` — a classe de comandos por interpretador e o `policy_id` correspondente.
- `AGENTS.md` — a seção de validadores de evidência menciona os gates; a lista de operações
  interceptadas passa a apontar para o artefato gerado em vez de ser transcrita.
- `.claude/rules/governance.md` — registrar que o gate passa a ser a materialização de R-SEC-001 no
  caminho de execução de comandos.
- Runbook: o rastro de auditoria muda de arquivo e de formato; quem hoje lê o TSV manualmente precisa
  ser avisado antes da remoção.

## Revisão Futura

Revisar quando ocorrer qualquer um destes eventos:

- a latência do gate em `BeforeTool` passar a ser perceptível, o que reabre a alternativa A3;
- surgir um escape que a análise estrutural em shell não cubra sem caso especial;
- o `PermissionRequest` de Claude, Codex e Copilot passar a ser usado pelo harness — ele é bloqueante
  e hoje está inteiramente inexplorado, e mudaria o desenho de onde a aprovação é solicitada;
- o Codex corrigir o pulo silencioso de hooks untrusted (openai/codex#46210), que hoje é uma via de
  desabilitação do gate fora do alcance desta decisão.
