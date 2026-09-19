# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Resultado tipado de hook e tradução única de exit code, fail-closed para hook crítico
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório
- **Relacionados:**
  - PRD [`.specs/prd-hooks-canonicos-vendor-neutral/prd.md`](prd.md) — RF-12 (`prd.md:239`), RF-13 (`prd.md:241`), RF-14 (`prd.md:243`), RF-15 (`prd.md:246`), RF-16 (`prd.md:249`)
  - Techspec [`.specs/prd-hooks-canonicos-vendor-neutral/techspec.md`](techspec.md), que registra esta ADR em `techspec.md:541`
  - [ADR-001 — Canonical Hook Contract v1 como fonte única com projeções derivadas](adr-001-hook-contract-fonte-unica-projecoes.md), referenciada em `techspec.md:540`

## Contexto

O harness não tem hoje nenhum resultado canônico de hook. Toda decisão de governança trafega como
par implícito de exit code mais texto livre em `stderr`, e cada consumidor reconstrói o significado
por conta própria. Isso produz quatro problemas verificáveis no estado atual do repositório.

**Divergência de código de bloqueio entre scripts.** Três validadores declaram constante própria
com o mesmo valor mas nomes distintos: `.agents/scripts/git-operation-gate.sh:5`
(`readonly GIT_OPERATION_BLOCK_EXIT=2`), `.agents/hooks/validate-preload.sh:5`
(`readonly PRELOAD_BLOCK_EXIT=2`) e `.agents/scripts/validate-session-end.sh:4`
(`readonly SESSION_END_BLOCK_EXIT=2`). Um quarto validador não segue a convenção:
`.claude/hooks/validate-governance.sh:45` bloqueia com `exit 1` literal, sem constante e com
semântica diferente da dos demais. Não existe ponto no repositório que declare qual código
significa bloqueio; o significado está replicado e já divergiu.

**`NOT_APPLICABLE` e sucesso são indistinguíveis.** Um hook que não tem alvo a avaliar sai com
exit 0, exatamente como um hook que avaliou o alvo e aprovou. `git-operation-gate.sh:41` sai com
`exit 0` quando `command_text` está vazio — ausência de alvo — e `git-operation-gate.sh:133` sai com
o mesmo `exit 0` após ter avaliado o comando e não ter detectado violação. O consumidor não tem
como separar "nada a decidir" de "decidido e permitido", o que impede tanto auditoria quanto
qualquer prova de cobertura de gate.

**Erro de infraestrutura falha aberto.** Vários pontos tratam ausência de dependência como
permissão. Em `.claude/hooks/validate-governance.sh:33`, quando `parse-hook-input.sh` não é
encontrado em nenhum dos três caminhos de cascata, o hook emite `AVISO` e faz `exit 0`, ou seja, a
edição de `AGENTS.md` passa sem gate nenhum. Em `.agents/scripts/hook-prereq-gate.sh:48-51`, a
ausência de `validate-skill-prerequisites.sh` degrada explicitamente para no-op com `exit 0`. Em
`.claude/hooks/subagent-stop-wrapper.sh:34-35`, a ausência de `post-execute-task.sh` produz
`exit 0` silencioso, sem sequer um aviso em `stderr`. Em todos os três casos, um erro de instalação
se apresenta ao runtime como aprovação.

**O modo ACP orquestrado descarta o erro do hook.** Em `internal/runtime/runner.go:428` e
`internal/runtime/runner.go:466`, o despacho nos pontos de tool-call é escrito como
`_ = disp.Dispatch(...)`, enquanto os despachos de abertura de sessão e de construção de prompt
(`runner.go:362`, `runner.go:371`, `runner.go:377`) propagam o erro. Hoje nenhum hook de produção
está registrado em `PointToolCallPreDispatch` nem em `PointToolCallPostComplete`, então o descarte
é inerte na prática, mas ele torna esses dois pontos estruturalmente incapazes de bloquear.

Duas restrições delimitam o espaço de solução. A primeira é a semântica de bloqueio de cada CLI,
que não é uniforme: Claude e Codex bloqueiam com exit 2 ou com o campo JSON
`permissionDecision:"deny"`; o `preToolUse` do Copilot é fail-closed para exit 2 e para qualquer
código não zero, mas fail-open no timeout; o OpenCode não bloqueia por exit code, e sim por
`throw new Error` dentro do plugin JS — `.opencode/plugin/governance.js:279`, `:286`, `:294`,
`:301` e `:315` — e o seu `tool.execute.after` (`governance.js:304`, que delega para
`observePostTool`) nunca bloqueia: rejeição do validador vira apenas `console.warn` em
`governance.js:262`. A segunda restrição é o estado da auditoria existente:
`.aispec/governance-escapes.log` é um TSV de cinco colunas escrito por
`git-operation-gate.sh:50-52` e `validate-preload.sh:90-92`, coberto pelo `.gitignore:34` (padrão
`.aispec/`), sem nenhum leitor no repositório, sem rotação e sem validação; o campo de comando é
gravado cru, de modo que um comando contendo `\t` corrompe o registro silenciosamente.

## Decisão

Introduzir em `internal/hookcontract` um Value Object `Result` com campos não exportados e
construtor validador, acompanhado de um enum `Decision` com cinco valores: `ALLOW`, `BLOCK`,
`WARN`, `NOT_APPLICABLE` e `ERROR`. `Result` carrega `reason`, `policyID`, `gateID`, `evidence` e
`metadata`. `NewResult` recusa `DecisionBlock` sem `reason` e sem `policyID` ou `gateID`
preenchidos, de modo que nenhum bloqueio possa existir no sistema sem motivo e sem âncora de
política.

```go
package hookcontract

type Decision int

const (
	DecisionAllow Decision = iota
	DecisionBlock
	DecisionWarn
	DecisionNotApplicable
	DecisionError
)

type Result struct {
	decision Decision
	reason   string
	policyID string
	gateID   string
	evidence []string
	metadata map[string]string
}

func NewResult(decision Decision, reason, policyID, gateID string, evidence []string, metadata map[string]string) (Result, error) {
	if decision == DecisionBlock {
		if reason == "" {
			return Result{}, errors.New("block decision requires a reason")
		}
		if policyID == "" && gateID == "" {
			return Result{}, errors.New("block decision requires policy id or gate id")
		}
	}
	return Result{
		decision: decision,
		reason:   reason,
		policyID: policyID,
		gateID:   gateID,
		evidence: append([]string(nil), evidence...),
		metadata: cloneMetadata(metadata),
	}, nil
}
```

Introduzir um `ExitCodeTranslator` como ponto único de conversão entre exit code e `Decision`, nos
dois sentidos:

```go
type ExitCodeTranslator interface {
	ToDecision(code int, stderr string, critical bool) Result
	ToExitCode(result Result) int
}
```

Quando `critical == true`, qualquer código não mapeado produz `DecisionError`, e o chamador
converte `DecisionError` em bloqueio. Esse é o fecho que elimina o fail-open descrito nos três
pontos citados no contexto: erro de infraestrutura em hook crítico deixa de se apresentar como
aprovação.

Os scripts shell continuam comunicando por exit code. Nenhum validador é reescrito para emitir JSON
no stdout; a tradução acontece do lado Go, na fronteira entre o processo do hook e o runtime. O
escopo da decisão é, portanto, o contrato interno do harness, não o protocolo que as CLIs já
consomem.

Persistir as decisões em `.aispec/hook-decisions.jsonl`, em append real, uma decisão por linha, com
leitor implementado em `internal/hookaudit`. O TSV atual permanece escrito em paralelo por um
release e é removido depois, quando o leitor JSONL estiver em uso.

Deixar de descartar o erro em `internal/runtime/runner.go:428` e `internal/runtime/runner.go:466`,
alinhando esses dois pontos ao tratamento já aplicado em `runner.go:362`, `:371` e `:377`.

Partes do sistema impactadas: os pacotes novos `internal/hookcontract` e `internal/hookaudit`; o
despacho de hooks em `internal/runtime/runner.go`; os adapters por CLI, que passam a chamar o
tradutor em vez de interpretar o código por conta própria; e a documentação de gates.

## Alternativas Consideradas

**A1 — Reescrever os scripts shell para emitirem JSON estruturado no stdout.**
Descrição: cada validador passaria a imprimir um objeto com decisão, motivo e política, eliminando a
necessidade de tradução.
Vantagens: decisão rica produzida na origem, sem camada intermediária e sem perda de informação.
Desvantagens: quebra o contrato de exit code que as quatro CLIs já consomem nativamente; exigiria
reescrever dez validadores e o plugin JS de uma só vez; e não resolve o caso do Copilot, que exige
exit 2 para bloquear mesmo que o stdout declare `allow`.
Motivo da rejeição: raio de regressão desproporcional ao ganho, sobre a superfície que hoje sustenta
todos os gates de governança.

**A2 — Manter exit codes e apenas padronizar as constantes.**
Descrição: unificar `GIT_OPERATION_BLOCK_EXIT`, `PRELOAD_BLOCK_EXIT`, `SESSION_END_BLOCK_EXIT` e o
`exit 1` de `validate-governance.sh:45` num único valor compartilhado.
Vantagens: mudança mínima, sem pacote novo e sem alteração no runtime.
Desvantagens: não distingue `NOT_APPLICABLE` de sucesso, não corrige o fail-open por erro de
infraestrutura e não fornece motivo auditável ao bloqueio.
Motivo da rejeição: RF-13, RF-14 e RF-16 ficariam integralmente sem cobertura; a alternativa resolve
apenas o sintoma cosmético da divergência de constantes.

**A3 — Tradução distribuída, cada adapter convertendo seu próprio exit code.**
Descrição: manter a conversão dentro de cada adapter de CLI, sem componente compartilhado.
Vantagens: adapters mais autônomos, sem acoplamento a um tipo comum.
Desvantagens: recria exatamente a divergência já observada entre `exit 1` e `exit 2` nos scripts, só
que em Go; e torna RF-14 não verificável por um teste único, porque a regra de fail-closed passaria
a existir em quatro implementações independentes.
Motivo da rejeição: a duplicação da decisão crítica entre provedores é precisamente o que a User
Story existe para eliminar.

**A4 — Tratar erro de hook crítico como `WARN` e deixar a operação prosseguir.**
Descrição: preservar o comportamento atual de degradação, apenas tornando-o explícito no resultado.
Vantagens: zero risco de bloqueio novo; nenhuma mudança de comportamento observável.
Desvantagens: mantém o fail-open de `validate-governance.sh:33`, `hook-prereq-gate.sh:48-51` e
`subagent-stop-wrapper.sh:34-35`.
Motivo da rejeição: contraria P06 da User Story, que exige fail-closed para invariantes críticos.

## Consequências

### Benefícios Esperados

- Toda decisão de bloqueio passa a carregar `policyID` ou `gateID` e `reason`, verificados no
  construtor. Bloqueio sem motivo deixa de ser representável no sistema.
- `ERROR` deixa de se apresentar como `ALLOW` nos hooks críticos. Falha de instalação, dependência
  ausente ou validador não executável produzem estado explícito, não aprovação silenciosa.
- `NOT_APPLICABLE` torna-se distinguível de sucesso, o que permite medir cobertura real de gate e
  separar "não havia alvo" de "havia alvo e foi aprovado".
- A regra de tradução concentra-se num ponto único, testável por tabela, em vez de estar replicada
  entre quatro scripts e quatro adapters.
- A auditoria ganha leitor. O JSONL em `internal/hookaudit` substitui um TSV que hoje ninguém lê e
  que corrompe silenciosamente quando o comando auditado contém tabulação.

### Trade-offs e Custos

- Acrescenta-se uma camada entre o script e o runtime. O caminho de uma decisão passa a ser script
  → exit code → `ExitCodeTranslator` → `Result` → chamador, em vez de script → exit code → chamador.
- Os hooks shell continuam sem poder expressar `WARN` nativamente. O canal de saída tem apenas exit
  code, e a aproximação disponível é `exit 0` acompanhado de mensagem em `stderr` — exatamente o que
  `git-operation-gate.sh:122-126` já faz no modo `warn`. A tradução de `WARN` a partir de shell é,
  portanto, heurística sobre `stderr`, não um estado declarado pelo script. Esta limitação é
  assumida conscientemente e não é resolvida por esta ADR.
- O log JSONL cresce sem limite até que a tarefa de rotação seja executada. Durante o release de
  transição, TSV e JSONL coexistem, e o custo de escrita por decisão dobra.

### Riscos e Mitigações

**Risco (a): deixar de descartar o erro em `runner.go:428` e `runner.go:466` passa a bloquear onde
hoje não bloqueia.**
Impacto: uma execução orquestrada pode abortar num ponto de tool-call que hoje nunca aborta.
Mitigação: nenhum hook de produção está registrado nesses dois pontos no estado atual, o que foi
verificado. A mudança entra antes de qualquer registro novo, e não depois, de modo que o primeiro
hook registrado já encontre a semântica correta em vez de herdar a semântica permissiva.

**Risco (b): a tabela de tradução divergir do que os scripts realmente fazem.**
Impacto: a camada nova passaria a mentir sobre a decisão do script, que é uma falha pior do que a
divergência atual porque é invisível.
Mitigação: teste `TestExitCodeTranslator_PreservesCurrentSemantics`, table-driven, usando como
fixture os códigos reais de cada script — `2` de `git-operation-gate.sh:5`,
`validate-preload.sh:5` e `validate-session-end.sh:4`, e `1` de `validate-governance.sh:45`.

**Risco (c): necessidade de reverter a decisão.**
Impacto: baixo. O tradutor é aditivo.
Plano de rollback: remover `internal/hookcontract` e `internal/hookaudit` e restaurar o `_ =` em
`runner.go:428` e `:466`. Nenhum script shell precisa ser tocado, porque nenhum foi alterado.

## Plano de Implementação

1. Criar `Decision` e `Result` em `internal/hookcontract`, com construtor validador que recusa
   `DecisionBlock` sem `reason` e sem `policyID`/`gateID`. Dependência: nenhuma.
2. Implementar a tabela de tradução `ToDecision`/`ToExitCode` e o teste
   `TestExitCodeTranslator_PreservesCurrentSemantics` com os códigos reais de cada script como
   fixture. Dependência: etapa 1.
3. Criar `internal/hookaudit` com escritor e leitor do JSONL, e teste de round-trip
   escrita/leitura. Dependência: etapa 1.
4. Remover o `_ =` de `internal/runtime/runner.go:428` e `:466`, propagando o erro como nos demais
   despachos. Dependência: etapas 1 e 2.
5. Passar a escrever `.aispec/hook-decisions.jsonl` em paralelo ao TSV existente, sem remover os
   `printf` de `git-operation-gate.sh:50-52` e `validate-preload.sh:90-92`. Dependência: etapa 3.
6. Após um release com os dois formatos, remover a escrita do TSV e a referência a
   `.aispec/governance-escapes.log` na documentação. Dependência: etapa 5.

Critérios para considerar a adoção concluída: as etapas 1 a 5 entregues com teste verde; nenhum
adapter interpretando exit code fora do tradutor; e o leitor de `internal/hookaudit` em uso por pelo
menos um consumidor real.

## Monitoramento e Validação

Métricas emitidas como entradas `hook.decision` e `hook.duration_ms` em `.agents/telemetry.log`, no
formato já existente `<ts> chave=valor`, append-only, opt-in por `GOVERNANCE_TELEMETRY=1`, conforme
a ADR-006 do repositório ([`docs/adr/006-telemetria-feedback-cycle.md`](../../docs/adr/006-telemetria-feedback-cycle.md)).
Nenhuma infraestrutura externa de observabilidade é introduzida.

Critérios de sucesso:

- Zero ocorrências de `DecisionError` convertido em `ALLOW` na suíte de conformidade.
- Todo registro com `decision=BLOCK` no JSONL apresenta `policy_id` não vazio.
- A tradução de cada código real observado nos quatro scripts é coberta por caso de tabela em
  `TestExitCodeTranslator_PreservesCurrentSemantics`.

Critério de revisão: se a tradução precisar de mais de um caso especial por provedor, a premissa de
ponto único falhou e esta ADR deve ser reaberta antes de acomodar o segundo caso especial.

## Impacto em Documentação e Operação

- `docs/hooks-canonicos.md` — documento novo, descrevendo `Decision`, `Result`, a tabela de
  tradução e a regra de fail-closed para hook crítico.
- [`docs/evidence-gates.md`](../../docs/evidence-gates.md) — atualizar a referência a
  `${GOVERNANCE_ESCAPE_LOG:-.aispec/governance-escapes.log}` em `docs/evidence-gates.md:26` para
  apontar ao JSONL durante a transição e apenas ao JSONL após a etapa 6.
- [`AGENTS.md`](../../AGENTS.md) — incluir na tabela de gates a menção ao resultado canônico e ao
  ponto único de tradução, na seção de validadores de evidência.
- [`docs/troubleshooting.md`](../../docs/troubleshooting.md) — procedimento de leitura de
  `.aispec/hook-decisions.jsonl` para diagnosticar bloqueio, incluindo o fato de que `.aispec/` é
  ignorado pelo git (`.gitignore:34`) e o log não trafega em commit.

## Revisão Futura

Revisar quando uma quinta CLI entrar no harness, porque o custo de manutenção da tabela cresce com o
número de provedores e a premissa de ponto único precisa ser reavaliada nesse momento. Revisar
também quando qualquer provedor já suportado passar a aceitar decisão estruturada no lugar de exit
code — nesse cenário, a alternativa A1 deixa de ser desproporcional para aquele provedor e a
fronteira entre tradução e emissão nativa precisa ser redesenhada. Condição de substituição por nova
ADR: se a limitação de `WARN` não expressável em shell se tornar bloqueante para algum requisito
funcional futuro.
