# Documento de Requisitos do Produto (PRD) — Harness Portátil e Vendor-Neutral

<!-- spec-version: 7 -->

**Slug:** `prd-harness-portatil-vendor-neutral`
**Origem:** `US-harness-portatil-claude-codex-opencode.md` (User Story bruta)
**Tipo:** Evolução arquitetural / plataforma de engenharia
**Prioridade:** Alta
**Baseline de não-regressão:** `v2.0.1` (commit `a8b7721`), PRD `prd-harness-quatro-clis-loop-aprovacao` (RF-01..RF-63) concluído

---

## Visão Geral

O `ai-spec-harness` já instala governança operacional para quatro CLIs de IA e já possui, entregues e
testados, os mecanismos de enforcement, aprovação, evidência e paridade semântica. O que ele **ainda não
tem** é a propriedade que a User Story pede: ser um **harness portátil e vendor-neutral**, isto é, um
contrato de engenharia que (a) tenha uma fonte canônica única e verificável, (b) seja declarado
explicitamente em vez de emergir do código, (c) seja distribuível e atualizável em projetos consumidores
de forma segura e reversível, e (d) seja comprovadamente equivalente entre os provedores por um artefato
versionado e não por convicção.

Hoje três propriedades quebram essa promessa e são **fatos verificados** neste repositório:

1. A política universal mais restritiva do repositório (`R-STYLE-001`, código em inglês, zero
   comentários) vive em `.claude/rules/code-style.md` — um diretório *vendor-specific* — e **não é
   distribuída** a projetos consumidores: os assets embarcados contêm apenas
   `internal/embedded/assets/.claude/rules/governance.md`. Um projeto que instala o harness recebe
   governança parcial e não tem como saber disso.
2. Não existe contrato declarado. `internal/config/runtime.go` expõe treze chaves **operacionais**
   (timeouts, retries, concorrência, ferramenta default) e nenhuma chave de **política** (git, aprovação,
   testes, evidência, descoberta de skills). O parse é leniente: chave desconhecida é silenciosamente
   ignorada.
3. A equivalência entre provedores existe como invariantes em Go (`internal/parity`, níveis
   `Common` / `ToolSpecific` / `BestEffort`) e como tabelas em prosa na documentação. Não há artefato
   versionado que um revisor possa ler em um diff para saber se uma capability passou a ser declarada
   universal sem teste que a sustente.

Este PRD cobre **exclusivamente o delta** entre a User Story e o que já está entregue. Requisitos da US
já satisfeitos pelo baseline são registrados como tal na seção *Cobertura da User Story*, com o
mecanismo que os satisfaz, e **não** são reespecificados — reespecificar o que já funciona é o principal
vetor de regressão nesta entrega.

O princípio que rege o documento é o da própria US: **o agente é substituível; o contrato de engenharia
não.**

---

## Objetivos

1. **Fonte canônica única e verificável.** Toda regra universal tem exatamente uma origem em
   `.agents/`; toda cópia vendor-specific é derivada e verificada por gate que falha na divergência.
   Sucesso: um gate dedicado de sincronia de policies falha quando `.claude/rules/` diverge de
   `.agents/policies/` (planejado). O gate de skills **não** é estendido nem alterado (RF-11).
2. **Harness Contract v1 declarado, versionado e fail-closed.** Sucesso: `.agents/harness.yaml` (planejado) com
   `version: 1` é validado por schema; campo desconhecido ou versão incompatível produz erro tipado e
   exit code ≠ 0, nunca *warning*.
3. **Equivalência provável, não afirmada.** Sucesso: artefato de capability matrix gerado a partir dos
   invariantes em Go, commitado, com gate de CI que falha se (a) o commitado divergir do gerado ou
   (b) uma célula marcada como suportada não tiver teste associado.
4. **Distribuição e atualização seguras.** Sucesso: `sync` mostra o diff antes de aplicar, detecta
   conflito por checksum do manifesto, aborta o lote inteiro em conflito não resolvido e nunca deixa o
   projeto em estado parcialmente aplicado.
5. **Diagnóstico que localiza a falha.** Sucesso: `doctor` reporta blocos Core / Claude / Codex /
   Copilot / OpenCode, sem chamada paga a LLM, com exit code ≠ 0 quando invariante obrigatório falha.
6. **Conformidade cross-provider em dois níveis.** Sucesso: os cenários determinísticos da US bloqueiam
   merge em CI; os cenários que exigem julgamento do modelo rodam nightly e reportam sem bloquear.
7. **Economia de contexto medida no ponto de entrada.** Sucesso: budget de tokens do contexto de
   entrada por provedor, com baseline commitado e gate em CI — complementando o budget por skill que já
   existe e já roda.
8. **Zero regressão observável.** Sucesso: todo fluxo que não ativa capacidade nova preserva
   comportamento; `go test ./...` e todos os gates do `test.yml` permanecem verdes; nenhuma mudança de
   default sem declaração no changelog.

---

## Histórias de Usuário

**Primária — engenheiro multi-CLI**
Como engenheiro que usa Claude Code, Codex, Copilot e OpenCode em projetos diferentes, quero instalar e
sincronizar um único harness canônico em qualquer repositório, para escolher livremente a CLI sem perder
governança, skills, critérios de aceite, validações, evidências, controles de Git, budgets e requisitos
de qualidade.

**Secundária — mantenedor do harness**
Como mantenedor, quero que uma regra universal tenha uma única origem e que a divergência entre origem e
derivados falhe o build, para não manter a mesma regra em quatro lugares e descobrir a divergência em
produção.

**Secundária — revisor de PR**
Como revisor, quero ler no diff qual capability passou a ser declarada universal e qual teste a sustenta,
para não aprovar uma promessa de paridade que nenhum teste prova.

**Secundária — engenheiro que retoma trabalho**
Como engenheiro que iniciou uma tarefa no Codex e retoma no Claude, quero que os artefatos SDD
persistidos sejam suficientes para continuar, para não depender de estado privado da sessão do provedor
anterior.

**Caso de borda — projeto consumidor sem credenciais**
Como engenheiro preparando um repositório em máquina sem credenciais de nenhum provedor, quero instalar
o harness normalmente, porque a instalação é operação de arquivos e não deve exigir autenticação.

---

## Fatos Verificados

Levantados neste repositório antes da redação. São a base factual dos requisitos; cada um é falseável.

| ID | Fato | Onde |
|----|------|------|
| **V-01** | `R-STYLE-001` (`code-style.md`) existe no repo mas **não** está nos assets embarcados; consumidor recebe só `governance.md` | `internal/embedded/assets/.claude/rules/` |
| **V-02** | Política universal reside em diretório vendor-specific (`.claude/rules/`), violando RNF01 da US | `.claude/rules/`, `CLAUDE.md` |
| **V-03** | `config.Runtime` tem 13 chaves operacionais e **nenhuma** de política; parse YAML é leniente (chave desconhecida ignorada) | `internal/config/runtime.go` |
| **V-04** | `.agents/` contém `hooks/`, `lib/`, `scripts/`, `skills/`. Não existem `policies/`, `workflows/` nem `harness.yaml` | `.agents/` |
| **V-05** | 38 subcomandos registrados; **não** existem `init` nem `sync` | `cmd/ai_spec_harness/root.go` |
| **V-06** | `upgrade` suporta `--check`, `--source`, `--ref`, comparando versões e checksums | `cmd/ai_spec_harness/upgrade.go` |
| **V-07** | Escrita é atômica **por arquivo**; não há transação de lote nem rollback de lote | `internal/install/write_tracker.go (baseline pre-Tarefa 8.0)`, consolidado em `internal/tracking/tracker.go` |
| **V-08** | Manifesto rastreia paths individuais em `installed_files`/`merged_files` (RF-05 do PRD anterior), **sem hash associado**. O campo `checksums` é por-skill e **write-only**: escrito na instalação e no upgrade, lido por nenhum código de produção. **Não existe `path → checksum`** | `internal/manifest/manifest.go:14-35`, `internal/install/install.go:1582-1593`, `internal/upgrade/upgrade.go:638` |
| **V-08b** | Drift hoje é detectado por recomputação ao vivo contra a fonte, não por comparação com o manifesto | `internal/upgrade/upgrade.go:339-341`, `internal/install/install.go:678-689` |
| **V-08c** | Não existe a categoria **conflito** em install/upgrade/uninstall; nenhum write compara o destino antes de sobrescrever | `internal/install/write_tracker.go:104-134 (baseline pre-Tarefa 8.0)`, consolidado em `internal/tracking/tracker.go` |
| **V-09** | `doctor` verifica git, diretório de skills e manifesto; não separa core de adapters nem reporta por provedor | `internal/doctor/doctor.go` |
| **V-10** | `verify` reporta `current`/`missing`/`drifted`, com `--by-cli`, `--global`, `--check-codex-trust` | `cmd/ai_spec_harness/verify.go` |
| **V-11** | Invariantes de paridade existem em Go com três níveis de enforcement; nenhum artefato de matriz é gerado ou commitado | `internal/parity/parity.go` |
| **V-12** | Budget de tokens **por skill** existe e roda em CI (`make budget`) | `internal/integration/token_budget_skill_test.go`, `.github/workflows/test.yml` |
| **V-13** | Não existe budget do **contexto de entrada** (AGENTS.md + regras auto-carregadas + metadata de skills) por provedor | — |
| **V-14** | `evals/sdd` é corpus determinístico dos contratos SDD (`schema`, `integrity`, `proof`, `paths`, `review`); **não** é parametrizado por provedor | `evals/sdd/manifest.json` |
| **V-15** | `acp-live.yml` e `hooks-live.yml` são nightly/manual e **não** são gate de merge | `.github/workflows/` |
| **V-16** | Telemetria é `.agents/telemetry.log`, linhas `<ts> chave=valor`, append-only, opt-in por `GOVERNANCE_TELEMETRY=1` (ADR-006); leitores ignoram chaves desconhecidas | `internal/telemetry/` |
| **V-17** | Campos da US ausentes da telemetria: `provider`, `model`, `task_type`, `risk`, `tokens_in/out`, `human_intervention`, `final_status` | `internal/telemetry/acp.go` |
| **V-18** | Gates de espelhamento existentes: `check-skills-sync`, `check-hooks-sync`, `check-scripts-sync` | `Makefile`, `scripts/` |
| **V-19** | Conjunto canônico de agentes é `{claude, codex, copilot, opencode}`, com gate de contrato em CI (RF-01/RF-08 do PRD anterior) | `internal/skills/` |
| **V-20** | Ciclo de Aprovação, tradutor anticorrupção fail-closed, mapa 1:1 critério→evidência e gate de encerramento estão entregues | `internal/approval/`, `internal/evidence/` |
| **V-21** | Os scripts de espelhamento **não** são genéricos: detectam skill pela presença de `SKILL.md`, têm allowlist `non_skill_dirs` e lista fixa de mirrors. Espelhar `policies/` não os reusa como estão | `scripts/sync-skills.sh`, `scripts/check-skills-sync.sh` |
| **V-22** | Convenção vigente de baseline de budget: teto com margem de 10% sobre o valor medido, em mapa atualizado conscientemente | `internal/integration/token_budget_skill_test.go` |
| **V-23** | **Não existe enforcement de auto-commit.** Nenhum hook, script ou plugin intercepta `git commit`/`git push`; a proibição existe apenas como prosa em SKILL.md e em `governance.md`. O único teste "destrutivo" (`rm -rf /`) valida *ausência de alvo extraível*, e o comando é **liberado** sob `GOVERNANCE_PRELOAD_CONFIRMED=1` | grep vazio em `.agents/scripts/`, `.agents/hooks/`, `.opencode/plugin/`, `internal/runtime/hooks/`; `tests/integration/opencode_shell_gate_test.go:67-116` |
| **V-24** | Os testes E2E de paridade cross-project **não rodam em CI**: o job `unit` executa `./internal/parity/...` sem `-tags=integration`, e o job `integration` não inclui esse pacote | `.github/workflows/test.yml:34,159` |
| **V-25** | Invariantes `CL03..CL08` e `X03` são **auto-satisfeitos por construção**: `Checker.Generate` injeta os próprios stubs que eles depois verificam; não provam nada sobre o instalador real | `internal/parity/parity.go:196-227` |
| **V-26** | `Level` não é o eixo de escopo: `CL01`, `CL02`, `CP01`, `CD01`, `CD02` são marcados `Common` tendo `AppliesTo` de ferramenta única. O eixo real de escopo é `AppliesTo` | `internal/parity/parity.go:345,362,379,415,432` |
| **V-27** | `Failures()` não é consultado por nenhum código de produção; o único consumidor de paridade em produção é `lint`, que lê apenas `Warnings()` | `internal/parity/parity.go:166-174`, `cmd/ai_spec_harness/lint.go:47-77` |
| **V-28** | O gate RF-28 do PRD anterior vive em `specs.ValidateParityMatrix` + `dispatchProofRegistry`, com evidência de execução parseada de `go test -json` (ações `pass`/`skip`/`fail`), não de texto | `internal/runtime/specs/parity_gate.go:30-72`, `parity_dispatch_proof_test.go:40-61,138-184` |
| **V-29** | O parser de telemetria entende **apenas** `skill=` e `ref=`; todo token restante é descartado sem `default`. Chave nova é gravada mas **invisível** a `report`/`summary`/`trend` | `internal/telemetry/parser.go:50-57`, `parser.go:11-15` |
| **V-30** | OpenCode e Codex carregam `AGENTS.md` e skills **nativamente**, fora de `Job.Prompt`. Medição baseada só no prompt subestima esses dois provedores | `internal/contextgen/contextgen.go:454,474`, `internal/taskloop/agent.go:214` |
| **V-31** | Existe padrão golden-file maduro: artefato commitado + regeneração por `UPDATE_SNAPSHOTS=1` + diff legível + "gate do gate" que prova que o gate falha quando deve | `internal/contextgen/contextgen_test.go:187-208,238-283`, `tests/integration/sync_gates_guard_test.go` |
| **V-32** | `FakeFileSystem.WriteFileAtomic` é **alias literal** de `WriteFile` (corpos byte-idênticos) e **nenhuma** operação de escrita do fake pode falhar: `MkdirAll`, `WriteFile`, `WriteFileAtomic`, `Symlink`, `Remove`, `RemoveAll` retornam `nil` incondicionalmente e sequer consultam `NoWrite` | `internal/fs/fake.go:140-148`, comparar com `internal/fs/fs.go:173-204` |
| **V-33** | O passo de CI "Verificar snapshots atualizados" **nunca executou um único teste**: filtra `-run TestSnapshot`, e o teste chama-se `TestContextgen_Snapshots` — verificado empiricamente, exit 0 com zero testes | `.github/workflows/test.yml:53-57`, `internal/contextgen/contextgen_test.go:238` |
| **V-34** | **Nenhum gate recomputa os hashes de `skills-lock.json`.** `ai-spec skills --verify` não é chamado por CI, Makefile ou hook algum; os testes que tocam o lock validam presença e correspondência de nomes, nunca integridade de conteúdo | `internal/skillscheck/skillscheck.go:105,114-117`, grep vazio em `.github/workflows/`, `Makefile`, `scripts/` |
| **V-35** | O uninstall remove `.claude/rules/` por **lista fixa de um único arquivo**; não há sweep por diretório como existe para skills e agents. `code-style.md` instalado sem atualizar essa lista fica órfão e ainda impede o prune de `.claude/` | `internal/uninstall/uninstall.go:590-603,607-614,570-587` |
| **V-36** | A produção **não tem** o hardening G6 que o teste tem: o caminho recursivo de `candidateResolvesToValidator` usa regex cru sobre o texto inteiro, sem remover comentários nem exigir posição de execução — o teste exige | `internal/runtime/specs/native_config.go:222-237,260-267` vs `internal/install/hooks_parity_matrix_test.go:274-282,299-360` |
| **V-37** | `Report.Flows` é criado vazio e **nunca populado**; o JSON sempre emite `"flows": {}`. O subsistema `internal/metrics/flow.go`, que mediria isso, não tem chamador de produção | `internal/metrics/metrics.go:105`, grep vazio por `Flows[` |
| **V-38** | `DirHash` retorna `("", nil)` para não-diretório, tornando "inexistente" e "erro" indistinguíveis; inclui o path relativo no hash (rename altera o hash) e ordena deterministicamente | `internal/fs/fs.go:224-254` |
| **V-39** | `make check-mocks` detecta mock desatualizado, mas é **cego a interface nova** não declarada em `mockery.yml` — nada verifica "toda interface pública tem mock" | `scripts/check-mocks.sh`, `mockery.yml` |
| **V-41** | Divergência entre alvo e CI: `Makefile:26-27` roda cinco pacotes de integração; `.github/workflows/test.yml:159` roda apenas três. Teste de integração criado fora desses três **não roda no gate de PR** | `Makefile:26-27`, `.github/workflows/test.yml:159` |
| **V-40** | Baseline empírico de não-regressão registrado: `go build ./...` exit 0; skills drift 0; hooks 28 em sync, drift 0; validadores 29 em sync, drift 0; gate de encerramento 7/7 mirrors | execução de `make check-skills-sync check-hooks-sync check-scripts-sync` |

> **Gate de Fase 1:** nenhum requisito deste PRD pode ser implementado antes de os fatos acima serem
> reconfirmados na execução. Um fato falseado invalida o requisito que dele depende e exige `needs_input`.

---

## Funcionalidades Core

### 1. Harness Contract v1 (`.agents/harness.yaml`)

Arquivo declarativo, versionado, com os invariantes obrigatórios do harness: política de Git, aprovação,
qualidade, evidência e descoberta de skills. Parse **estrito**: campo desconhecido ou versão incompatível
falha com erro tipado. Fica separado de `config.yaml` (operacional, leniente) porque política e *tuning*
têm regimes de falha opostos: política é fail-closed, *tuning* é best-effort. Misturá-los obrigaria a um
parser híbrido, que é justamente onde nascem falsos positivos e falsos negativos.

### 2. Canonicalização vendor-neutral (`.agents/policies/`)

Regras transversais migram de `.claude/rules/` para `.agents/policies/`. `.claude/rules/` passa a ser
**espelho gerado**, pelo mesmo mecanismo que já espelha `.agents/skills/` → `.claude/skills/`, com gate
de sincronia. Claude Code continua carregando exatamente o mesmo conteúdo do mesmo caminho — a mudança é
de origem, não de destino. `.agents/workflows/` (planejado) **não** é criado enquanto não houver conteúdo real que o
justifique.

### 3. Capability Matrix gerada e versionada

Artefato derivado dos invariantes já existentes em `internal/parity`, regenerável por comando `make`,
commitado no repositório e verificado por dois gates: divergência entre gerado e commitado falha; célula
declarada suportada sem teste associado falha. Estende o gate de paridade já entregue (RF-28 do PRD
anterior) em vez de criar um mecanismo paralelo.

### 4. Distribuição: `init` / `sync`

`init` e `sync` como aliases de `install` e `upgrade`, preservando integralmente o contrato de CLI atual.
`sync` ganha semântica de conflito explícita: arquivo gerenciado modificado localmente (detectado por
checksum do manifesto) é conflito; conflito não resolvido aborta o lote inteiro; aplicação parcial nunca
é deixada no disco.

### 5. Doctor multi-provider

`doctor` passa a agregar, em blocos Core / Claude / Codex / Copilot / OpenCode: presença de arquivos
obrigatórios, validade do contrato, integridade de skills, estado de instalação por provedor e
pré-condições de enforcement já conhecidas (trust do Codex, pastas confiáveis do Copilot). Somente
verificações estáticas; nenhuma chamada paga a LLM. Exit code ≠ 0 em falha obrigatória.

### 6. Suíte de conformidade cross-provider em dois níveis

Os cenários da US são classificados por **falseabilidade**: o que é decidível por fixture, contrato,
filesystem ou configuração roda em CI e bloqueia merge; o que depende do comportamento do modelo roda
nightly contra CLIs reais e reporta sem bloquear. Toda falha é atribuída a `core`, `adapter` ou
`provider`.

### 7. Budget de contexto de entrada

Extensão do mecanismo de budget já existente (`make budget`, que já roda em CI) para medir o contexto de
**entrada** por provedor — não apenas por skill — com baseline commitado e falha em crescimento não
declarado.

### 8. Telemetria comparável

Novas chaves no formato `chave=valor` já vigente, preservando ADR-006 e todos os leitores existentes.
Métrica indisponível é **ausente**, nunca inventada nem preenchida com zero.

---

## Requisitos Funcionais

### Bloco A — Inventário e Harness Contract

- **RF-01:** Produzir inventário explícito e versionado das regras hoje distribuídas entre `AGENTS.md`,
  `CLAUDE.md`, `CODEX.md`, `COPILOT.md`, `.claude/`, `.codex/`, `.opencode/` e `.github/`, classificando
  cada uma como `universal`, `claude-specific`, `codex-specific`, `copilot-specific` ou
  `opencode-specific`. O inventário é artefato de entrega, não rascunho descartável, e é a entrada
  obrigatória do Bloco B. Nenhuma migração pode ocorrer antes dele.
- **RF-02:** Existe `.agents/harness.yaml`, contrato declarativo com `version: 1` explícito, expressando
  no mínimo: política de Git (`auto_commit`, `auto_push`), política de aprovação (operações destrutivas),
  requisitos de qualidade (testes, lint), política de evidência e política de descoberta de skills.
- **RF-03:** O contrato é validado por schema versionado. **Campo desconhecido é erro**, não aviso;
  valor de tipo inválido é erro; versão diferente da suportada é erro tipado que nomeia a versão
  encontrada, a suportada e a ação corretiva. Configuração inválida nunca é silenciosamente ignorada.
- **RF-04:** Somente `version: 1` é aceito nesta entrega. A política de evolução — o que é mudança
  aditiva e o que é incompatível — é registrada em ADR. **Nenhuma maquinaria de migração é construída**
  enquanto não existir uma segunda versão: abstração sem caminho exercitável é código morto.
- **RF-05:** O contrato é **fonte da verdade sobre política** e não duplica chaves operacionais de
  `config.yaml`. Um valor não pode existir nos dois arquivos. O gate de build falha se um nome de chave
  aparecer em ambos os schemas.
- **RF-06:** `config.yaml` preserva integralmente o comportamento atual: mesmas chaves, mesma cascata de
  precedência (`flags > workspace > global > defaults`), mesma tolerância a campo desconhecido. Nenhuma
  configuração existente em qualquer projeto pode passar a falhar por causa desta entrega.
- **RF-07:** Ausência de `.agents/harness.yaml` em um projeto **não** quebra nenhum fluxo: o binário
  aplica o contrato v1 default embarcado e o reporta como `default` no diagnóstico. Preservação de F1 é
  obrigatória; projeto instalado antes desta entrega continua funcionando sem nenhuma ação do usuário.
- **RF-08:** A validação do contrato é verificação estática pura: sem rede, sem execução de binário de
  provedor, sem chamada a LLM.

### Bloco B — Canonicalização vendor-neutral

- **RF-09:** Existe `.agents/policies/` como origem canônica única das regras transversais. `R-GOV-001`
  e `R-STYLE-001` passam a residir lá.
- **RF-10:** `.claude/rules/` passa a ser **derivado**, gerado pelo mesmo mecanismo de espelhamento já
  usado para skills e hooks. O conteúdo e o caminho que o Claude Code carrega permanecem idênticos
  — a mudança é invisível para a ferramenta.
- **RF-11:** O espelhamento de `policies/` é implementado por um **par dedicado** de scripts — geração e
  verificação — na mesma forma dos três pares já existentes para skills, hooks e scripts. Os scripts de
  skills **não** são generalizados nem alterados (V-21): refatorar o gate que hoje protege a entrega
  inteira concentraria o risco exatamente no mecanismo de proteção. Divergência entre origem canônica e
  qualquer derivado **falha o build**; editar o derivado em vez da origem deixa de ser possível sem
  quebrar CI.
- **RF-12:** `R-STYLE-001` passa a ser distribuído a projetos consumidores (V-01). Um projeto que instala
  o harness recebe o conjunto **completo** de regras universais, não um subconjunto silencioso. Como isso
  altera a governança aplicada a projetos já instalados — sem alterar comportamento do binário —, a
  entrega é **minor** e o changelog declara nominalmente a regra nova e como removê-la localmente. A
  distribuição é incondicional: torná-la opt-in manteria V-01 de pé por default.
- **RF-13:** Nenhuma regra classificada como `universal` no inventário (RF-01) permanece mantida
  manualmente em mais de um local. A remoção de uma duplicata só é permitida quando existir teste
  cobrindo o comportamento que ela governa — a ordem é: primeiro o teste, depois a remoção.
- **RF-14:** `.agents/workflows/` **não** é criado nesta entrega. Não há conteúdo identificado que o
  justifique, e diretório vazio é estrutura sem demanda concreta. A decisão e seu gatilho de reabertura
  ficam registrados no PRD.

### Bloco C — Adapters e Capability Matrix

- **RF-15:** O conjunto canônico de provedores permanece `{claude, codex, copilot, opencode}`. Este PRD
  **não** reduz esse conjunto. O Copilot participa do contrato, da capability matrix, do diagnóstico e
  dos testes adversariais como cidadão de primeira classe.
- **RF-16:** Cada adapter referencia o core canônico e **não** mantém cópia integral de regra universal.
  Recurso exclusivo de um provedor fica isolado na configuração específica dele.
- **RF-17:** Nenhum adapter pode enfraquecer política obrigatória. O gate falha quando um adapter
  declara, para uma política marcada como obrigatória no contrato, enforcement mais fraco que o do core.
- **RF-18:** Existe artefato de capability matrix versionado e commitado, **gerado** a partir dos
  invariantes em `internal/parity` — fonte única. Cada célula declara o provedor, a capability, o nível
  de suporte e o teste que a sustenta. São commitados **dois** artefatos derivados da mesma fonte: um
  JSON, que é o que o gate compara, e uma tabela Markdown renderizada dele, que é o que o mantenedor lê.
  Comparar prosa formatada produziria falso positivo a cada reajuste de coluna.
- **RF-18.1:** Nem todo invariante existente é evidência válida. Os auto-satisfeitos por construção
  (V-25) e o eixo de escopo ambíguo `Level` vs `AppliesTo` (V-26) são saneados **antes** de alimentarem
  a matriz: o escopo passa a derivar de `AppliesTo`, e invariante cuja verificação é satisfeita pelo
  próprio gerador do snapshot não pode sustentar célula marcada como suportada. Publicar uma matriz
  derivada de invariante vacuoso seria falso positivo institucionalizado — o oposto do objetivo 3.
- **RF-19:** Gate de CI falha quando **qualquer** dos dois artefatos commitados diverge do gerado. O
  comando de regeneração é documentado na mensagem de falha.
- **RF-20:** Gate de CI falha quando uma célula é marcada como suportada sem teste associado,
  **estendendo** o gate já entregue (`specs.ValidateParityMatrix` + registry de prova de disparo, com
  evidência lida de `go test -json` — V-28) em vez de criar validador paralelo. Capability universal sem
  teste correspondente é proibida.
- **RF-20.1:** Os testes E2E de paridade cross-project passam a **rodar em CI**. Hoje não rodam (V-24):
  existem, são verdes quando executados à mão, e nenhum job os executa. Um teste que não roda não é
  evidência — é a forma mais cara de falso positivo.
- **RF-21:** Capability exclusiva de um provedor é classificada explicitamente como `provider capability`
  e **nunca** como requisito universal. Invocar uma capability não suportada produz falha ou relato
  explícito, jamais degradação silenciosa.

### Bloco D — Distribuição: instalação e sincronização

- **RF-22:** `init` existe como alias de `install` e `sync` como alias de `upgrade`. O contrato de CLI
  atual é preservado integralmente: nenhum nome, flag ou comportamento existente muda. Os aliases são
  cobertos pelo teste de contrato de CLI.
- **RF-23:** A instalação preserva os comportamentos já entregues e verificados: identifica a raiz do
  repositório, é idempotente, não commita, não faz push, não exige credencial de nenhum provedor e
  instala somente os arquivos necessários.
- **RF-23.1:** **Correção factual:** o relatório de instalação hoje distingue apenas **criado** e
  **mesclado** (`installed_files`/`merged_files`); **atualizado** e **preservado** não existem como
  categorias, e o install não tem tipo de relatório estruturado — só saída textual. A User Story exige
  as quatro categorias mais conflito. Portanto o relatório passa a ser um tipo estruturado com
  **criado / atualizado / preservado / conflito / mesclado**, no molde do que `verify` já usa
  (estado tipado + remédio), em vez de string solta.
- **RF-24:** A instalação passa a reportar explicitamente a categoria **conflito**, hoje ausente do
  relatório (V-08c): arquivo gerenciado cujo checksum no disco diverge do registrado para aquele path.
  Isto tem uma **dependência dura**: o manifesto hoje não guarda `path → checksum` (V-08), e o campo
  `checksums` existente é por-skill e write-only. Registrar o checksum por path instalado é
  pré-requisito de RF-24, RF-26 e RF-27 — sem ele, "conflito" não é decidível e o requisito seria
  falso-positivo por construção. O campo é aditivo e `omitempty`, seguindo o precedente já testado de
  `installed_files`/`merged_files`.
- **RF-25:** `sync` exibe o diff das alterações **antes** de aplicá-las. O modo de pré-visualização já
  existente (`--check`) é o caminho de leitura; a aplicação é uma ação distinta e explícita.
- **RF-26:** Arquivo local **não gerenciado** pelo manifesto nunca é sobrescrito nem removido por `sync`.
- **RF-27:** Conflito (RF-24) **aborta o lote inteiro** por default. Aplicar sobre conflito exige flag
  explícita de sobrescrita, e a flag reporta nominalmente cada arquivo sobrescrito. O caso de um arquivo
  de regra editado à mão em projeto consumidor — que passa a ser derivado por RF-10 — é tratado por esta
  mesma máquina, **sem caso especial**: a mensagem nomeia o arquivo, informa que ele virou derivado e
  aponta a origem canônica onde a customização deve passar a viver. Promover conteúdo local a regra
  canônica automaticamente é proibido: uma regra enfraquecida localmente viraria governança oficial em
  silêncio.
- **RF-28:** A aplicação é transacional em lote: falha no meio da operação **reverte o lote** e reporta
  o que foi revertido. O estado final é "tudo aplicado" ou "nada aplicado" — nunca parcial e silencioso.
  Isto estende a escrita atômica por arquivo já existente (V-07), que não cobre o lote.
- **RF-29:** A versão instalada e a versão disponível são identificáveis, e existe validação pós-
  atualização que confirma que o harness ficou consistente. `sync` é determinístico: mesma origem e mesmo
  destino produzem o mesmo resultado.

### Bloco E — Diagnóstico multi-provider

- **RF-30:** `doctor` passa a reportar em blocos separados: `Core` e um bloco por provedor instalado.
  `verify` preserva integralmente seu contrato atual — nenhuma flag, saída ou código de saída muda.
- **RF-31:** O bloco `Core` verifica: presença dos arquivos canônicos obrigatórios, validade do contrato
  (RF-03), integridade de skills pelo mecanismo de lock já existente e sincronia entre origem canônica e
  derivados.
- **RF-32:** Cada bloco de provedor reporta: instruções descobríveis, skills disponíveis, políticas
  aplicadas, validadores de evidência instalados e pré-condições de enforcement conhecidas — incluindo,
  para Codex e Copilot, as pré-condições de trust já modeladas no baseline.
- **RF-33:** `doctor` detecta e distingue: arquivo obrigatório ausente, configuração divergente, versão
  de contrato incompatível e integridade de skill inválida. Cada falha é atribuída a `core`,
  `instalação`, `adapter`, `provider` ou `validação` — a atribuição é parte da saída, não inferência do
  usuário.
- **RF-34:** `doctor` retorna exit code ≠ 0 quando qualquer invariante obrigatório falha, e executa
  somente verificações estáticas — sem chamada paga a LLM.

### Bloco F — Conformidade cross-provider e progressive disclosure

- **RF-35:** Existe suíte de conformidade cross-provider com os catorze cenários da US, cada um
  classificado por falseabilidade em **determinístico** (decidível por fixture, contrato, filesystem ou
  configuração) ou **live** (depende do comportamento do modelo). A classificação de cada um dos catorze
  é **normativa** e está fixada na tabela *Classificação dos Cenários de Conformidade* deste PRD; o
  manifesto da suíte a replica e é revisável em diff. Cenário que mude de classificação exige alteração
  deste PRD.
- **RF-36:** Os cenários determinísticos rodam em CI e **bloqueiam merge**: configuração inválida, skill
  adulterada, capability não suportada, evidência ausente, retomada de tarefa, continuidade SDD entre
  provedores, carga de skill irrelevante e — pela via da invocação direta do hook/script canônico — as
  três políticas críticas (auto-commit, operação destrutiva sem aprovação, falha de testes).
- **RF-37:** Os cenários live rodam no fluxo nightly já existente contra CLIs reais, reportam resultado
  e **não** bloqueiam merge: tarefa simples de leitura, implementação pequena, bugfix, tarefa que exige
  skill e, como **confirmação** do que RF-36 já prova deterministicamente, as três políticas críticas.
- **RF-38:** O mesmo fixture avalia todos os provedores. A avaliação mede **invariantes**, nunca
  igualdade textual de resposta. Resultado é reprodutível naquilo que é determinístico.
- **RF-39:** Toda falha da suíte é atribuída a `core`, `adapter` ou `provider` quando a informação
  permitir; quando não permitir, isso é declarado em vez de adivinhado.
- **RF-40:** As políticas críticas — auto-commit, operação destrutiva sem aprovação, ausência de
  confirmação obrigatória — são cobertas em **dois níveis**, para os quatro provedores. O nível
  determinístico invoca o script canônico diretamente com a operação proibida e bloqueia merge; o nível
  live confirma, no nightly, que a CLI real roteia a operação pelo ponto instrumentado. Só o
  determinístico deixaria um provedor mudar o ponto de interceptação com o gate verde; só o live não
  bloquearia regressão em política crítica. O resultado esperado é **bloqueio seguro** em todos os casos.
- **RF-40.1:** **Premissa corrigida (V-23):** o enforcement de auto-commit **não existe hoje** — a
  proibição é prosa, não código. Portanto este PRD **cria** o gate canônico de operação Git antes de
  poder testá-lo: um ponto de decisão único, nos mesmos scripts canônicos que os quatro provedores já
  compartilham, que nega `git commit` e `git push` não solicitados. Sem isso, RF-11 da User Story
  ("commit automático permanece desabilitado por padrão") depende exclusivamente de o LLM lembrar de
  obedecer — exatamente o que o Definition of Done proíbe.
- **RF-40.2:** O gate de operação destrutiva é separado do gate de preload. Hoje eles estão fundidos:
  `rm -rf /` é negado por *não ter alvo extraível* e é **liberado** quando o preload está confirmado
  (V-23). Destrutividade passa a ser critério próprio, avaliado independentemente do estado de preload.
- **RF-41:** Existe budget de tokens do **contexto de entrada** por provedor — `AGENTS.md` + regras
  auto-carregadas + metadata de descoberta de skills — com baseline commitado. Crescimento acima do teto
  falha o CI, com mensagem que informa o valor medido, o teto e como aceitar o crescimento
  deliberadamente. O teto adota a convenção vigente: margem de 10% sobre o valor medido, atualizado
  conscientemente (V-22). O budget **por skill** já existente permanece intocado. Este gate mede o
  contexto **declarado**, que é determinístico e está sempre disponível; o contexto **observado** é
  responsabilidade de RF-44, e as duas métricas têm papéis distintos e declarados — nunca se substituem.
- **RF-41.1:** A medição do contexto declarado **não pode derivar apenas do prompt construído pelo
  harness**: OpenCode e Codex carregam `AGENTS.md` e skills nativamente, fora dele (V-30), e uma medição
  baseada só no prompt subestimaria esses dois provedores — produzindo um gate verde enquanto o contexto
  real cresce. O numerador por provedor é o conjunto de arquivos que aquele provedor efetivamente
  carrega na entrada, declarado por provedor e testado.
- **RF-42:** O arquivo de entrada de cada provedor contém somente instruções universais necessárias.
  Documentação especializada, políticas extensas e skills não relacionadas à tarefa são carregadas sob
  demanda e não entram no contexto inicial.

### Bloco G — Telemetria comparável e ablation

- **RF-43:** A telemetria preserva o formato `chave=valor` append-only e opt-in por variável de ambiente
  (ADR-006). Os campos novos são **acrescentados** como chaves adicionais; nenhum leitor existente
  quebra, nenhum arquivo muda de formato.
- **RF-43.1:** **Qualificação necessária (V-29):** escrever a chave não a torna observável. O parser
  atual reconhece apenas `skill=` e `ref=` e descarta silenciosamente todo o resto, sem `default`. Todo
  campo de RF-44 que precise ser **lido** exige estender o parser e o registro de entrada. A
  compatibilidade retroativa é preservada — chave desconhecida continua ignorada em vez de falhar —, mas
  a extensão do leitor é parte do escopo, não consequência automática.
- **RF-44:** São registrados, quando disponíveis: `provider`, `model`, `task_type`, `risk`, `tokens_in`,
  `tokens_out`, `duration`, `retries`, `tool_calls`, `context_loaded`, `skills_loaded`, `review_rounds`,
  `tests_passed`, `human_intervention`, `final_status`.
- **RF-45:** Métrica que o provedor não disponibiliza é **ausente** — nunca inventada, nunca substituída
  por zero, nunca inferida. A ausência não impede a execução. Isso vale nominalmente para
  `context_loaded`, que registra o contexto **observado** e existe apenas onde o provedor o expõe: ele
  alimenta a análise de ablation (RF-47), e **nunca** é usado como gate — um gate com força variável por
  provedor é exatamente a capability universal silenciosa que RF-21 proíbe. Métrica específica de
  provedor fica em chave de extensão própria, fora do conjunto comum.
- **RF-46:** Nenhuma chave secreta, credencial ou conteúdo sensível é persistido pela telemetria. Existe
  teste que falha se um campo sensível conhecido for emitido.
- **RF-47:** Existe mecanismo de **ablation** que permite comparar `baseline` contra
  `baseline + componente` para skill, hook ou policy, sobre as métricas de RF-44. A decisão
  `KEEP` / `ON-DEMAND` / `REMOVE` é sustentada por evidência registrada, nunca por avaliação subjetiva do
  LLM. Nesta entrega o escopo é o **mecanismo e o baseline**; a execução das campanhas de ablation é sob
  demanda e não é gate de release.

### Bloco H — Não-regressão e release

- **RF-48:** Todos os gates hoje verdes permanecem verdes: testes unitários, integração, lint, vet,
  cobertura total e por pacote crítico, sincronia de skills/hooks/scripts, teste de hooks, validadores de
  evidência, caminhos de spec, skills portáteis, budget e evals SDD.
- **RF-49:** Todo fluxo que não ativa capacidade introduzida por este PRD preserva comportamento
  observável idêntico ao baseline `v2.0.1`. Qualquer mudança de default é declarada explicitamente no
  changelog — e este PRD **não prevê nenhuma**. A entrega é publicada como **minor**: nenhuma API, flag
  ou fluxo é quebrado. A única mudança perceptível ao consumidor é a governança mais completa de RF-12,
  declarada no changelog com nota de migração.
- **RF-50:** Um projeto instalado com a versão anterior continua funcionando após a atualização do
  binário sem nenhuma ação do usuário (consequência direta de RF-07).
- **RF-51:** A documentação é reconciliada: instalação, atualização, uso multi-CLI, contrato do harness,
  capability matrix e diagnóstico. A documentação existente que descreve comportamento preservado **não**
  é reescrita.
- **RF-53:** Trocar de CLI **não exige reinstalar nem converter** o projeto. O mesmo diretório
  instalado serve os quatro provedores simultaneamente, e existe teste que instala uma vez e exerce os
  quatro sem nenhuma operação intermediária.
- **RF-54:** O item "workflows universais são compartilhados" do Definition of Done da User Story é
  respondido explicitamente: **não existe workflow universal identificado** que não caiba em skill ou
  policy (RF-14). O item é declarado **satisfeito por ausência de objeto**, não por implementação. Se um
  workflow universal for identificado durante a execução, `.agents/workflows/` é criado com o mesmo par
  dedicado de espelhamento de RF-11 — e isso reabre este PRD em vez de ser decidido em silêncio.
- **RF-55:** Nenhum commit ou push ocorre implicitamente em **nenhum** dos fluxos — instalação,
  sincronização, diagnóstico ou execução de tarefa. Isto é verificado por teste, não apenas afirmado.
- **RF-56:** A política de release é declarada: regressão em cenário determinístico da suíte de
  conformidade **bloqueia merge**; regressão em qualquer gate do baseline **bloqueia release**; falha em
  cenário live **não** bloqueia nenhum dos dois, mas é reportada no relatório de release. Sem essa
  declaração, "regressões bloqueiam release" é interpretável de três formas distintas.
- **RF-58:** **Testes de transação e de conflito não podem usar `FakeFileSystem` como prova** (V-32): o
  fake não distingue escrita atômica de escrita simples e nenhuma de suas operações de escrita pode
  falhar, o que torna o valor probatório sobre atomicidade **nulo** e quase nulo sobre conflito. Ou o
  fake ganha injeção de falha explícita, ou a prova de RF-28 vive em teste de integração com filesystem
  real em diretório temporário. Escolher o fake sem essa correção seria um teste verde que não prova
  nada — o falso positivo mais caro desta entrega.
- **RF-59:** O passo de CI que verifica snapshots é corrigido (V-33): hoje filtra um nome de teste que
  não existe e passa com zero testes executados desde que foi escrito. Todo gate introduzido por este
  PRD nasce com **mutation test** que prova que ele falha quando deveria — o padrão já praticado em
  `cmd/ai_spec_harness/catalog_sync_test.go`, que injeta divergência artificial e exige detecção.
- **RF-60:** A integridade de skills passa a ser **verificada automaticamente** (V-34). Hoje o mecanismo
  existe, é correto e **ninguém o executa**: um `SKILL.md` editado sem atualizar `skills-lock.json` passa
  em `make test`, em `go test ./...` e na CI inteira. Sem isto, o critério da User Story
  "`skills-lock.json` permanece verificável" é satisfeito apenas no papel, e RF-31 reportaria integridade
  que nada garante.
- **RF-61:** A instalação e a desinstalação de regras transversais compartilham **uma única fonte de
  verdade de lista** (V-35). Hoje são duas listas fixas distantes uma da outra, sem acoplamento e sem
  teste de round-trip; distribuir `code-style.md` (RF-12) sem tocar a lista do uninstall deixaria o
  arquivo órfão e impediria o prune do diretório. Existe teste de round-trip instala→desinstala que
  exige árvore limpa.
- **RF-62:** A consolidação dos helpers de delegação test↔produção (V-36) é tratada como **mudança de
  comportamento**, não limpeza: a produção não tem o hardening que exige caminho em posição de execução,
  e consolidar na direção correta endurece o gate e **pode revelar violações hoje mascaradas**. A
  direção é promover a lógica do teste para produção — nunca o contrário, que afrouxaria o gate.
- **RF-63:** Nenhuma afirmação deste PRD ou da especificação técnica pode declarar como verificado o que
  V-33, V-34 e V-37 provaram não ser: snapshots não eram validados em CI, integridade de skills não é
  verificada automaticamente, e métricas por fluxo nunca foram medidas (`flows` sempre vazio). Onde a
  capacidade não existe, ela é criada por requisito explícito — nunca assumida como herdada.
- **RF-57:** As evidências desta implementação são persistidas no padrão vigente do repositório
  (relatório de execução por tarefa com prova física de validação), conforme a invariante de evidência
  obrigatória de `AGENTS.md`. Entrega sem evidência persistida não é `done`.
- **RF-52:** O modo nativo funciona sem processo permanente do `orchestrator`: após `init`, o
  desenvolvedor entra no repositório e usa qualquer CLI suportada diretamente. Existe teste que verifica
  que nenhum artefato instalado exige daemon.

---

## Experiência do Usuário

O usuário primário interage por linha de comando. Fluxos afetados:

**Preparar um projeto novo**
```bash
cd financialcontrol-api
ai-spec init .          # alias de install; relatório: criados / atualizados / preservados / conflitos
claude                  # ou codex, copilot, opencode — mesmos invariantes
```

**Atualizar um projeto existente**
```bash
ai-spec sync . --check  # diff das alterações, sem escrever
ai-spec sync .          # aplicação transacional; aborta o lote em conflito
```

**Diagnosticar**
```bash
ai-spec doctor .
# Core: contract / skills / policies / integrity
# Claude | Codex | Copilot | OpenCode: instructions / skills / policies / evidence
# exit != 0 quando invariante obrigatório falha
```

Princípios de saída: toda falha nomeia a camada responsável (core, instalação, adapter, provider,
validação); toda ação destrutiva ou ambígua para e pede decisão explícita; nenhuma operação de Git é
implícita.

---

## Restrições Técnicas de Alto Nível

1. **Go 1.27**, CLI Cobra, distribuição por GoReleaser + Homebrew. Assets embarcados via `go:embed`
   (ADR-001).
2. **Nenhuma dependência nova sem demanda concreta.** O contrato usa o parser YAML já presente; a
   medição de tokens usa a biblioteca de tokenização já presente.
3. **Não-regressão é restrição, não objetivo.** O baseline é `v2.0.1`. Toda entrega deste PRD é aditiva
   ou substitutiva com comportamento equivalente comprovado por teste.
4. **Fail-closed em política; best-effort em tuning.** Contrato, integridade, aprovação e evidência
   falham de forma segura sob ambiguidade. Preferências operacionais preservam a tolerância atual.
5. **Determinismo sobre julgamento de LLM.** Tudo que é validável por código não usa LLM. Verificação
   estática não executa binário de provedor nem acessa rede.
6. **Entrada externa é não confiável.** Arquivos de projeto, configurações e saídas de ferramentas são
   validados antes de uso.
7. **Telemetria é opt-in e append-only** (ADR-006); nenhum dado sensível é persistido.
8. **Custo de CI é restrição.** Cenários que exigem CLI real ficam fora do caminho crítico de merge.
9. **Conjunto canônico de provedores é `{claude, codex, copilot, opencode}`** e não é alterado aqui.
10. **Convenções do repositório são invioláveis:** código em inglês, zero comentários
    (`R-STYLE-001`, hard); artefatos `.md` em PT-BR; Conventional Commits.
11. **Nenhuma restrição externa incide sobre esta entrega** — confirmado pelo dono do repositório
    (D-20). Não há exigência regulatória, de compliance, de licenciamento de terceiro ou de política
    corporativa a atender. As únicas restrições são as internas listadas acima.

---

## Fora de Escopo

Excluídos explicitamente desta entrega. Cada um exige história própria com justificativa mensurável:

- Execução via OpenRouter; novo *agent swarm*; *multi-model voting*; vector database; RAG obrigatório;
  graph database; MCP customizado apenas para leitura de arquivos.
- Novo planner sobre o SDD existente; reimplementação de qualquer uma das CLIs.
- Auto-commit irrestrito, auto-push, auto-merge.
- **Maquinaria de migração de contrato** entre versões (RF-04): só existe `version: 1`.
- **`.agents/workflows/`** (RF-14): reabre quando houver conteúdo de workflow universal identificado que
  não caiba em skill ou policy.
- **Renomeação de `install`/`upgrade`**: `init`/`sync` são aliases aditivos; a depreciação dos nomes
  atuais não está em escopo.
- **Distribuição opt-in de `R-STYLE-001`** (D-15): a regra é distribuída incondicionalmente; torná-la
  ativável por chave manteria o defeito V-01 como comportamento default.
- **Generalização dos scripts de espelhamento de skills/hooks/scripts** (D-13): `policies/` ganha par
  próprio; refatorar os gates existentes é risco concentrado sem ganho nesta entrega.
- **Unificação de `doctor` e `verify`**: as duas superfícies coexistem; `verify` não muda.
- **Execução das campanhas de ablation** (RF-47): entrega-se o mecanismo e o baseline, não as campanhas.
- **Reespecificação do que o baseline já entrega**: Ciclo de Aprovação, tradutor anticorrupção, mapa 1:1
  critério→evidência, gate de encerramento, enforcement por provedor, remoção do Gemini, rastreio de
  arquivos no manifesto. Alterar esses mecanismos aqui é risco de regressão sem ganho.

---

## Cobertura da User Story

Mapeamento explícito entre os RFs da US e este PRD. Nenhum item da US fica sem destino — essa é a
condição de "zero lacuna".

| RF da US | Situação | Destino |
|----------|----------|---------|
| RF01 — Fonte canônica única | Parcial (V-01, V-02, V-18) | RF-01, RF-09..RF-13 |
| RF02 — Contrato formal | **Ausente** (V-03, V-04) | RF-02..RF-08 |
| RF03 — Instalação | Entregue; falta categoria conflito e alias | RF-22..RF-24 |
| RF04 — Sincronização | Parcial (V-06, V-07) | RF-25..RF-29 |
| RF05 — Skills compartilhadas | **Entregue** (skills-lock, ADR-005, espelhamento) | Preservado por RF-48 |
| RF06 — Progressive disclosure | Parcial (V-12 sim, V-13 não) | RF-41, RF-42 |
| RF07 — Adapter Claude | **Entregue** | Preservado por RF-16, RF-48 |
| RF08 — Adapter Codex | **Entregue** | Preservado por RF-16, RF-48 |
| RF09 — Adapter OpenCode | **Entregue** (ADR-020, RF-10..RF-18 anteriores) | Preservado por RF-16, RF-48 |
| — Adapter Copilot | **Entregue**; ausente da US | RF-15 (inclusão explícita) |
| RF10 — Capability Matrix | Parcial (V-11) | RF-18..RF-21 |
| RF11 — Política de Git | Entregue; falta cobertura adversarial dos 4 | RF-40 |
| RF12 — Approval Gates | **Entregue** (V-20) | Preservado por RF-48 |
| RF13 — Evidence | **Entregue** (V-20) | Preservado por RF-48 |
| RF14 — Deterministic Gates | **Entregue** | Preservado por RF-48 |
| RF15 — SDD compartilhado | Parcial: formato é neutro; continuidade cross-provider não é testada | RF-36 |
| RF16 — Doctor multi-provider | Parcial (V-09, V-10) | RF-30..RF-34 |
| RF17 — Conformidade cross-provider | **Ausente** (V-14, V-15) | RF-35..RF-39 |
| RF18 — Ablation | **Ausente** | RF-47 (mecanismo) |
| RF19 — Métricas comparáveis | Parcial (V-16, V-17) | RF-43..RF-46 |
| RNF01 — Vendor neutrality | Violado (V-02) | RF-09..RF-12 |
| RNF02 — Fail-closed | Entregue; estendido ao contrato | RF-03, RF-27, RF-28 |
| RNF03 — Idempotência | **Entregue** | RF-23, preservado por RF-48 |
| RNF04 — Observabilidade | Parcial | RF-33 |
| RNF05 — Economia de contexto | Parcial | RF-41, RF-42 |
| RNF06 — Compatibilidade | **Ausente** | RF-04 |
| RNF07 — Segurança | **Entregue** | Restrição 6 |
| RNF08 — Determinismo | **Entregue** | Restrições 5 e 8 |
| RNF09 — Manutenibilidade | Parcial | RF-13, RF-16 |
| RNF10 — Portabilidade | Entregue; falta teste | RF-52 |

---

## Classificação dos Cenários de Conformidade

Normativa. Fecha a classificação dos catorze cenários da User Story; RF-35 a referencia e o manifesto da
suíte a replica. "Det." = roda em CI e bloqueia merge. "Live" = roda no nightly e reporta sem bloquear.

| # | Cenário da US | Det. | Live | Como é decidido sem julgamento de LLM |
|---|---------------|:----:|:----:|----------------------------------------|
| 1 | Tarefa simples de leitura | — | ✓ | Depende da resposta do modelo; nada a decidir por contrato |
| 2 | Implementação pequena | — | ✓ | Idem |
| 3 | Bugfix | — | ✓ | Idem |
| 4 | Tarefa que exige skill | — | ✓ | Idem |
| 5 | Tarefa que **não** deve carregar skill irrelevante | ✓ | ✓ | Det.: contexto declarado do provedor não referencia a skill (RF-41/RF-42). Live: confirma a carga real |
| 6 | Tentativa de commit automático | ✓ | ✓ | Det.: invoca o hook canônico com a operação proibida e exige bloqueio. Live: confirma o roteamento |
| 7 | Operação destrutiva sem aprovação | ✓ | ✓ | Idem |
| 8 | Falha de testes | ✓ | ✓ | Det.: gate determinístico reprova e impede aprovação. Live: confirma o roteamento |
| 9 | Evidência ausente | ✓ | — | Validadores canônicos de evidência, fail-closed, sobre fixture |
| 10 | Retomada de tarefa | ✓ | — | Estado persistido em disco; retomada é função pura do artefato |
| 11 | SDD criado por um provider, continuado por outro | ✓ | — | Artefatos SDD são arquivos; a continuidade é verificável por fixture cruzado |
| 12 | Configuração inválida | ✓ | — | Validação de schema do contrato (RF-03) |
| 13 | Skill adulterada | ✓ | — | Verificação de integridade por hash, mecanismo já existente |
| 14 | Capability não suportada | ✓ | — | Consulta à capability matrix gerada (RF-18) |

Cenários 6, 7 e 8 aparecem nos dois níveis por decisão explícita (D-14): o bloqueio é determinístico e
por isso bloqueia merge; a *tentativa* vem do modelo e por isso o nightly confirma que a CLI real ainda
passa pelo ponto instrumentado.

---

## Faseamento e Gates

Cada fase só abre quando o gate da anterior fecha. O gate é bloqueante.

| Fase | Blocos | Gate de saída |
|------|--------|---------------|
| 1 — Inventário e contrato | A | Inventário completo (RF-01) e contrato validando com `version: 1`, sem nenhuma migração de arquivo feita |
| 2 — Canonicalização | B | Gate de sincronia de `policies/` verde; conteúdo carregado pelo Claude Code byte-idêntico ao anterior |
| 3 — Adapters e matriz | C | Matriz gerada, commitada, sem célula suportada sem teste |
| 4 — Distribuição | D | `sync` transacional com conflito coberto por teste; `init`/`sync` no teste de contrato de CLI |
| 5 — Doctor | E | `doctor` reporta os 4 provedores e retorna exit code ≠ 0 em falha obrigatória |
| 6 — Conformidade | F | Cenários determinísticos verdes em CI; cenários live executando e reportando |
| 7 — Telemetria e ablation | G | Chaves novas emitidas sem quebrar leitor existente; baseline de ablation registrado |
| 8 — Fechamento | H | Todos os gates do baseline verdes; documentação reconciliada; changelog sem mudança de default |

---

## Critérios de Sucesso Mensuráveis

1. Regra universal com mais de uma origem manual: **zero** (verificado por gate).
2. Célula de capability matrix declarada suportada sem teste: **zero** (verificado por gate).
3. Gates do baseline que regridem: **zero**.
4. Mudanças de default não declaradas no changelog: **zero**.
5. Projetos instalados na versão anterior que exigem ação manual após a atualização: **zero**.
6. Cenários determinísticos da suíte de conformidade cobertos em CI: **100%** dos classificados como
   determinísticos.
7. Crescimento do contexto de entrada acima do baseline sem aceite explícito: **detectado e bloqueado**.
8. Métricas de telemetria inventadas ou preenchidas por inferência: **zero**.

---

## Suposições e Questões em Aberto

### Questões em aberto

**Nenhuma.** As vinte decisões materiais levantadas na elaboração foram resolvidas pelo dono do
repositório e estão registradas em *Histórico de Decisões* (D-01..D-20).

### Suposições

**Nenhuma suposição permanece aberta.** As seis suposições da primeira versão deste PRD foram encerradas:

| ID | Suposição original | Encerramento |
|----|--------------------|--------------|
| S-01 | A US lista três CLIs por uso pessoal, não por intenção de excluir o Copilot | **Confirmada** pelo dono do repositório → decisão D-01; RF-15 fixa os quatro provedores |
| S-02 | `orchestrator init` / `sync` descrevem intenção, não o nome literal do binário | **Confirmada** → decisão D-02; `init`/`sync` entram como aliases de `install`/`upgrade` |
| S-03 | O espelhamento de `policies/` reutiliza o mecanismo de skills sem alteração | **Falseada por verificação** no repositório → fato V-21; resolvida por D-13 (par dedicado de scripts) |
| S-04 | Contexto observado não é uniformemente exposto, logo RF-41 mede o declarado | **Convertida em desenho explícito** → D-18: declarado é o gate, observado é `context_loaded` em telemetria e nunca é gate |
| S-05 | Os catorze cenários são classificáveis por falseabilidade | **Resolvida** → classificação normativa fixada na tabela *Classificação dos Cenários de Conformidade*; D-14 decide os três ambíguos |
| S-06 | Nenhuma restrição externa incide sobre a entrega | **Confirmada** pelo dono do repositório → D-20; registrada como Restrição Técnica 11 |

### Limites honestos declarados

Não são lacunas nem ressalvas: são propriedades conhecidas do desenho, decididas e documentadas.

- O gate de contexto (RF-41) mede o **limite superior** declarado, não o consumo real. O consumo real é
  registrado por RF-44/RF-45 onde o provedor o expõe. A separação é deliberada (D-18).
- Os cenários live (RF-37) não bloqueiam merge por decisão de custo e determinismo (D-04). O que eles
  cobrem que é crítico já está coberto deterministicamente por RF-36/RF-40 (D-14).
- RF-47 entrega o **mecanismo e o baseline** de ablation, não campanhas executadas (D-16). A ausência de
  campanha é escopo declarado, não requisito não cumprido.

---

## Histórico de Decisões

| ID | Decisão | Alternativas descartadas e por quê |
|----|---------|------------------------------------|
| **D-01** | Incluir Copilot como 4º provedor em contrato, matriz, doctor e testes adversariais | Escopo estrito de 3 criaria capability universal não verificada para um provedor suportado, contradizendo RF-28/RF-43 já entregues. Tier reduzido enfraqueceria a matriz sem ganho. |
| **D-02** | Manter `install`/`upgrade` e adicionar `init`/`sync` como aliases | Renomear com depreciação é mudança de contrato público de CLI logo após um `v2.0`. Não ter aliases faria o usuário que segue a US literalmente receber "unknown command". |
| **D-03** | Contrato em `.agents/harness.yaml` separado e estrito | Estender `config.yaml` exigiria parser híbrido (leniente fora do bloco, estrito dentro) — origem clássica de falso positivo/negativo. Contrato só embutido reduziria configurabilidade sem necessidade. |
| **D-04** | Conformidade em dois níveis: determinístico é gate, live é nightly | "Tudo live" põe custo de LLM e flakiness no caminho crítico, contra RNF08 e contra o critério de não exigir chamada paga para verificação estática. "Tudo determinístico" deixaria RF11/RF12 simulados. |
| **D-05** | Criar `.agents/policies/`, mover as rules, manter `.claude/rules/` como espelho gerado | Criar `workflows/` vazio adiciona estrutura sem demanda (AGENTS.md, modo de trabalho #4). Tratar policy como skill apagaria a distinção conceitual que a US faz. |
| **D-06** | Telemetria: manter `chave=valor` e só acrescentar chaves | JSONL paralelo duplicaria escrita e formato; migração completa quebraria leitores e reabriria ADR-006. |
| **D-07** | Budget de contexto de entrada por provedor, com baseline versionado, bloqueando CI | Medir sem bloquear tornaria "detectar crescimento" best-effort. Medir em sessão live depende de dado que nem todo provedor expõe. |
| **D-08** | PRD único com RFs em blocos faseados e gate entre fases | Dois PRDs acoplariam capability matrix (depende do contrato) através de dois spec-hashes. Cortar telemetria/ablation deixaria a US com lacuna declarada. |
| **D-09** | Estender `doctor` como agregador; `verify` permanece intocado | Unificar quebraria scripts e docs que usam `verify --global` e `--check-codex-trust`. Flag `--providers` esconderia o relatório que a US desenha atrás de conhecimento prévio. |
| **D-10** | Só `version: 1`, com falha explícita em versão incompatível e política de evolução em ADR | Migrador automático seria abstração sem caminho exercitável. Janela N-1 só faz sentido quando existir v2. |
| **D-11** | `sync` com dry-run no diff e aplicação transacional com abort-on-conflict | Best-effort com backup `.orig` contraria RNF02 e o critério de não sobrescrever sem política explícita. Merge 3-way textual em arquivo de governança pode gerar artefato inválido e aceito em silêncio. |
| **D-12** | Capability matrix gerada do Go, commitada, com gate de sincronia e de teste-por-célula | YAML como fonte pode divergir do comportamento real. Matriz só renderizada sob demanda impede revisão em diff e falha o critério "versionada junto ao código". |
| **D-13** | Par dedicado de scripts para espelhar `policies/`, sem tocar nos de skills | Generalizar os três gates existentes concentraria risco de regressão no próprio mecanismo de proteção. Mover o espelhamento para Go criaria dependência circular: o gate que protege o código passaria a depender do código compilado. |
| **D-14** | Cenários 6, 7 e 8 nos dois níveis: gate determinístico no hook + live confirmatório | Só determinístico deixa um provedor mudar o ponto de interceptação com o gate verde. Só live faz regressão em política crítica aparecer no dia seguinte, sem bloquear merge — inaceitável sob RNF02. |
| **D-15** | Entrega publicada como **minor**, com `R-STYLE-001` distribuída incondicionalmente e declarada no changelog | Major seria `v3.0.0` logo após o `v2.0.0` para uma entrega que não quebra API nem fluxo. Distribuição opt-in manteria V-01 de pé por default — o consumidor continuaria recebendo governança parcial sem saber. |
| **D-16** | Ablation: mecanismo e baseline nesta entrega; campanhas sob demanda, sem gate | Campanha piloto obrigatória traria execução live repetida ao escopo de entrega. Remover RF-47 deixaria RF18 da US sem destino — lacuna declarada. |
| **D-17** | Capability matrix: JSON como fonte de comparação + Markdown renderizado, ambos commitados | Só Markdown faria o gate comparar prosa formatada, com falso positivo a cada reajuste de coluna. Só JSON perderia a leitura humana que a própria US desenha como tabela. |
| **D-18** | Contexto declarado é o gate; contexto observado é telemetria quando o provedor expuser | Só declarado removeria `context_loaded` e enfraqueceria RF-47. Observado como gate daria força diferente ao gate por provedor — a capability universal silenciosa que RF-21 proíbe. |
| **D-19** | Regra editada à mão em consumidor é conflito comum: aborta o lote e instrui a mover a customização para a origem canônica | Migração automática promoveria conteúdo local não revisado a regra canônica — regra enfraquecida viraria oficial em silêncio, contra RNF02. Preservar sem espelhar reintroduziria o drift que RF-11 existe para matar. |
| **D-20** | Confirmado: nenhuma restrição externa (regulatória, compliance, licenciamento, política corporativa) | — (confirmação do dono do repositório; encerra S-06) |

---

## Rastreabilidade

- **Baseline:** `v2.0.1` / commit `a8b7721`.
- **PRD antecessor:** `.specs/prd-harness-quatro-clis-loop-aprovacao/prd.md` (RF-01..RF-63) — concluído.
  Os RFs deste PRD **não** renumeram nem substituem os daquele; a numeração é local a este documento.
- **ADRs incidentes:** ADR-001 (go:embed), ADR-002 (FakeFileSystem), ADR-003 (paridade semântica),
  ADR-005 (skills-lock SHA-256), ADR-006 (telemetria opt-in), ADR-008 (invariantes de paridade),
  ADR-020 (OpenCode ACP).
- **ADRs previstos por esta entrega:** contrato do harness e sua política de evolução (RF-04);
  canonicalização de policies e espelhamento (RF-09..RF-11).
- **Ancoragem:** após a aprovação deste PRD, rodar `ai-spec sync-spec-hash` antes de gerar a
  especificação técnica.
