# Documento de Requisitos do Produto (PRD) — Hooks Canônicos, Determinísticos e Vendor-Neutral

<!-- spec-version: 1 -->

**Slug:** `prd-hooks-canonicos-vendor-neutral`
**Origem:** `US-hooks-canonicos-claude-codex-opencode.md` (User Story bruta)
**Tipo:** Evolução arquitetural / Harness / Governança
**Prioridade:** Alta
**Depende de:** `.specs/prd-harness-portatil-vendor-neutral/` (13 tarefas; 6 `done`, 7 `pending`)
**Baseline de não-regressão:** `v2.0.1` (commit `a8b7721`) + tarefas 1.0, 2.0, 3.0, 7.0, 9.0 e 12.0 do PRD dependente

---

## Visão Geral

O `ai-spec-harness` já possui hooks funcionando nas quatro CLIs suportadas (Claude Code, Codex,
GitHub Copilot CLI, OpenCode). O que ele **não** possui é aquilo que a User Story pede: um
**contrato de hooks canônico, versionado, com resultado estruturado e semântica declarada por
provedor**, de modo que o mesmo invariante crítico seja comprovadamente aplicado em qualquer CLI e
em qualquer stack de linguagem — Go, Node/TypeScript, C#/.NET, Java ou Python — sem duplicar a
regra e sem depender de o LLM colaborar.

Sete fatos verificados neste repositório sustentam o escopo deste documento:

1. **Existem dois modelos de ponto canônico em paralelo, e eles não se conhecem.**
   `internal/runtime/specs/enforcement.go` declara três pontos (`pre-tool`, `post-tool`,
   `session-end`) usados para confrontar a configuração nativa das quatro CLIs;
   `internal/runtime/hooks/dispatcher.go` declara sete pontos Go (`runtime.pre_open`,
   `prompt.pre_build`, `prompt.post_build`, `tool_call.pre_dispatch`, `tool_call.post_complete`,
   `session.post_end`, `session.post_review`) usados no modo orquestrado ACP. Nenhum dos dois
   expressa os cinco eventos da User Story, nenhum versiona payload, e nenhum permite a um provedor
   **declarar** um evento como não suportado — a ausência simplesmente não aparece.

2. **Não existe resultado canônico de hook.** Toda decisão hoje é um par
   *exit code + texto em stderr*: `0` permite, `2` bloqueia (`GIT_OPERATION_BLOCK_EXIT`,
   `PRELOAD_BLOCK_EXIT`, `SESSION_END_BLOCK_EXIT`), `1` bloqueia em
   `validate-governance.sh`. Não há `ALLOW` / `BLOCK` / `WARN` / `NOT_APPLICABLE` / `ERROR`, nem
   `policy_id`, `gate_id` ou `evidence` anexáveis. Consequência direta: `NOT_APPLICABLE` e sucesso
   são indistinguíveis, e um hook que falha por erro de infraestrutura sai `0` na maior parte dos
   caminhos — isto é, **falha aberta**.

3. **O adapter do Claude nunca é confrontado.** `internal/runtime/specs/registry.go` declara
   `.claude/settings.json` e `.claude/settings.local.json` (planejado) com `required: false`, enquanto Codex
   (`.codex/config.toml`) e OpenCode (`.opencode/plugin/governance.js`) são `required: true`. Como
   o Claude grava seus hooks em `settings.local.json` — arquivo **não versionado** — o gate de
   paridade não tem o que ler e passa sem verificar nada. O provedor mais usado do harness é o
   único cuja fiação de hooks não é provada por gate.

4. **O gate Git canônico cobre menos que a policy que diz aplicar.**
   `.agents/scripts/git-operation-gate.sh` (entregue na tarefa 9.0) intercepta `git commit`,
   `git push` e `rm -rf`. Ficam de fora `git reset --hard`, `git clean`, `git checkout`/`git restore`
   destrutivos e *force push* — todos listados no escopo mínimo da User Story e todos igualmente
   capazes de destruir trabalho não commitado.

5. **Não existe quality gate disparado por evento.** Os gates de qualidade existem como alvos de
   `Makefile` e como validadores shell invocados pelas skills. Nenhum deles é acionado por um evento
   de lifecycle, nenhum tem política de seleção por risco, e nenhum deduplica execução quando o
   estado relevante não mudou.

6. **O harness não é igualitário entre stacks.** `internal/detect/toolchain.go` resolve
   `fmt`/`test`/`lint` apenas para `go`, `node` e `python`; **C# e Java não têm entrada no
   resolver**. `internal/skills/skills.go` enumera `AllLangs = {go, node, python, dotnet}` —
   **Java não existe** como linguagem de primeira classe. `internal/detect/framework.go:152`
   reconhece `pom.xml`/`build.gradle` apenas para compor uma string de framework. E
   `.agents/scripts/validate-task-evidence.sh:180` aceita `go test`, `pytest`, `npm test`,
   `dotnet test`, `cargo test` — mas **não** `mvn test` nem `gradle test`. Um projeto Java que
   instale o harness hoje falha o gate de evidência por construção, sem nada de errado com seu
   código.

7. **Não há timeout nem proteção contra recursão nos hooks shell.** Nenhum dos scripts em
   `.agents/hooks/` ou `.agents/scripts/` declara timeout (`grep timeout` retorna vazio). O único
   timeout existente é o do plugin OpenCode ao invocar validadores
   (`AISPEC_OPENCODE_VALIDATOR_TIMEOUT_MS`, default 8000ms) — comportamento de um provedor, não
   propriedade do contrato.

Este PRD cobre **exclusivamente o delta** entre a User Story e (a) o que já está entregue e (b) o
que já está especificado e agendado no PRD dependente. Requisitos da User Story já satisfeitos ou já
alocados são registrados na seção *Cobertura da User Story* com o mecanismo que os satisfaz e **não**
são reespecificados. Reespecificar o que já funciona é o principal vetor de regressão desta entrega.

O princípio que rege o documento é o da própria User Story: **o objetivo não é ter mais hooks; é
retirar do LLM a responsabilidade por invariantes que o software garante deterministicamente.**

---

## Objetivos

1. **Contrato de hooks único, canônico e versionado.**
   Sucesso: existe uma fonte única de eventos canônicos, consumida tanto pelo confronto de
   configuração nativa (`internal/runtime/specs`) quanto pelo dispatcher do modo orquestrado
   (`internal/runtime/hooks`); um gate de CI falha se os dois modelos divergirem. Hoje divergem sem
   que nada acuse.

2. **Decisão estruturada, nunca texto livre.**
   Sucesso: todo hook crítico retorna um resultado tipado com `decision`, `reason` e
   `policy_id`/`gate_id`; `ERROR` em hook crítico produz bloqueio, e `NOT_APPLICABLE` é
   distinguível de `ALLOW` em auditoria posterior.

3. **Mesma policy nas quatro CLIs, provada e não afirmada.**
   Sucesso: a suíte de conformidade executa os mesmos 14 cenários contra claude, codex, copilot e
   opencode, com fixtures compartilhadas; a regressão de qualquer policy crítica quebra o build.
   O furo do `settings.json` do Claude é fechado: nenhum provedor fica com fonte nativa não
   confrontada.

4. **Igualdade entre stacks: Go, Node/TypeScript, C#/.NET, Java e Python.**
   Sucesso: `ai-spec install` em um projeto de qualquer uma das cinco stacks produz um quality gate
   e um gate de evidência funcionais, sem edição manual; Java deixa de ser cidadão de segunda
   classe. Métrica: a suíte de portabilidade cobre as cinco stacks com projeto-fixture mínimo cada.

5. **Menor conjunto possível de hooks, com custo medido.**
   Sucesso: existe inventário versionado com classificação `KEEP`/`MODIFY`/`REMOVE`/`ON-DEMAND`
   justificada por evidência, e um gate que falha quando surge hook sem entrada no inventário.
   Nenhum hook é removido nesta entrega sem que o invariante que ele protege já esteja coberto por
   teste no core.

6. **Overhead observável e limitado.**
   Sucesso: cada execução de hook tem duração mensurável quando a telemetria está habilitada; todo
   hook tem timeout declarado; recursão hook→ferramenta→hook é impedida por construção.

---

## Histórias de Usuário

- **Como** engenheiro que usa o harness em vários projetos, **quero** que `claude`, `codex`,
  `copilot` e `opencode` apliquem os mesmos controles críticos, **para que** minha escolha de CLI
  não altere a governança do projeto.
- **Como** engenheiro de um projeto Java ou C#, **quero** instalar o harness e ter build, testes e
  evidência funcionando, **para que** eu tenha a mesma garantia que um projeto Go tem hoje.
- **Como** revisor de um PR no repositório do harness, **quero** ler a matriz de capabilities em um
  diff, **para que** eu perceba quando uma capability passou a ser declarada suportada sem teste que
  a sustente.
- **Como** responsável por um incidente, **quero** que todo bloqueio tenha motivo e policy
  associados e que a evidência seja verificável depois, **para que** eu consiga reconstruir por que
  uma operação foi negada sem ler o histórico privado da sessão da CLI.
- **Como** mantenedor do harness, **quero** que adicionar um hook seja exceção que exige
  justificativa, **para que** o conjunto não cresça por inércia.
- **Caso de borda:** uma tarefa iniciada no Claude precisa ser retomada no Codex após interrupção —
  o checkpoint tem de bastar, sem depender do histórico privado da sessão anterior.
- **Caso de borda:** o plugin do OpenCode falha ao carregar — a falha tem de ser observável, e não
  se converter silenciosamente em sessão sem governança.

---

## Funcionalidades Core

### 1. Inventário e racionalização de hooks

Um artefato versionado que lista todo hook e todo script que funciona como hook, com os onze campos
exigidos pela User Story, mais a classificação `KEEP`/`MODIFY`/`REMOVE`/`ON-DEMAND` com
justificativa. É o pré-requisito declarado da User Story: nenhum hook novo antes do inventário.
Importa porque o repositório hoje tem sete scripts em `.agents/hooks/`, dez em `.agents/scripts/`,
sete hooks Go em `internal/runtime/hooks/` e um plugin JS — e nada declara quais são equivalentes
entre si.

### 2. Canonical Hook Contract v1

Cinco eventos canônicos (`SessionStart`, `BeforeTool`, `AfterTool`, `BeforeComplete`, `SessionEnd`),
com semântica documentada por evento, payload com schema versionado, tratamento explícito de evento
desconhecido e capacidade de um provedor **declarar** um evento como não suportado. Substitui a
coexistência atual de três pontos em um modelo e sete em outro, sem quebrar nenhum dos dois
consumidores.

### 3. Resultado canônico de hook

Estrutura tipada `{decision, reason, policy_id, gate_id, evidence, metadata}` com
`ALLOW`/`BLOCK`/`WARN`/`NOT_APPLICABLE`/`ERROR`. Os hooks shell existentes continuam comunicando por
exit code; a tradução exit code → resultado canônico fica em um único ponto, preservando
comportamento observável. Importa porque hoje a decisão crítica depende de convenções numéricas não
uniformes entre scripts.

### 4. As cinco famílias de hooks canônicos

`git-policy`, `quality-gate`, `evidence-gate`, `checkpoint`, `telemetry` — nenhuma sexta família
nesta entrega. A regra vive no core; o adapter apenas traduz.

### 5. Quality gate stack-agnostic

O gate de qualidade resolve build/test/lint a partir da stack detectada, cobrindo Go, Node/TypeScript,
C#/.NET, Java e Python em pé de igualdade, e deduplica execução quando o estado relevante não mudou.
Importa porque é a única família nova de hook do escopo e porque é onde a promessa de "mesmos ganhos
em qualquer projeto" se materializa ou fracassa.

### 6. Adapters finos e matriz de capabilities

Um adapter por provedor, contendo tradução e validação de payload — nunca policy. A matriz declara,
por evento e por família, se o suporte é `supported`, `supported/adapter`, `verified` ou
`unsupported`, e cada célula afirmativa tem teste associado.

### 7. Diagnóstico e conformidade

`doctor` passa a reportar hooks e adapters, detectando adapter ausente, hook não executável,
configuração inválida, versão incompatível e divergência entre contrato e adapter — tudo sem chamar
LLM. A suíte de conformidade executa os catorze cenários da User Story contra os quatro provedores.

---

## Requisitos Funcionais

### Bloco A — Inventário e racionalização

- **RF-01:** Produzir inventário versionado de todo hook e de todo script que atue como hook,
  registrando por item: nome, localização, evento, provedor, objetivo, policy/gate associado,
  natureza bloqueante ou não bloqueante, custo aproximado, falha esperada e cobertura de testes.
  O inventário deve cobrir as quatro origens: `.agents/hooks/`, `.agents/scripts/`,
  `internal/runtime/hooks/` e `.opencode/plugin/`.
- **RF-02:** Identificar e registrar explicitamente no inventário: duplicações entre provedores,
  hooks sem teste e hooks sem policy ou gate claramente associado.
- **RF-03:** Atribuir a cada item do inventário uma classificação `KEEP`, `MODIFY`, `REMOVE` ou
  `ON-DEMAND`, com justificativa verificável que cite segurança, confiabilidade, cobertura de
  invariante, custo, latência, manutenção, duplicação, impacto em contexto, frequência de uso ou
  existência de alternativa determinística mais simples. Nenhum item pode ser classificado `KEEP`
  com a justificativa de já existir.
- **RF-04:** Nenhum hook classificado `REMOVE` pode ser removido nesta entrega antes de o invariante
  que ele protege estar coberto por teste no core canônico. A remoção efetiva é trabalho posterior
  à suíte de conformidade; até lá o item permanece ativo ou é rebaixado a `ON-DEMAND`
  desabilitado por padrão.
- **RF-05:** Um gate de CI deve falhar quando existir hook ou script equivalente a hook sem entrada
  correspondente no inventário, e quando uma entrada do inventário apontar para arquivo inexistente.

### Bloco B — Contrato canônico e resultado estruturado

- **RF-06:** Declarar cinco eventos canônicos — `SessionStart`, `BeforeTool`, `AfterTool`,
  `BeforeComplete`, `SessionEnd` — em fonte única, com semântica documentada por evento. A
  nomenclatura pode seguir a convenção já adotada no repositório desde que a semântica seja
  preservada.
- **RF-07:** O contrato não pode importar nem referenciar nomes específicos de Claude, Codex,
  Copilot ou OpenCode. Um gate deve falhar se um identificador de provedor aparecer no pacote do
  contrato.
- **RF-08:** Reconciliar os dois modelos de ponto existentes — os três pontos de
  `internal/runtime/specs` e os sete pontos de `internal/runtime/hooks` — em uma única fonte de
  verdade, preservando o comportamento observável de ambos os consumidores. Um gate deve falhar se
  os dois voltarem a divergir.
- **RF-09:** O payload de cada evento deve ter schema explícito e versionado. Payload que não
  satisfaça o schema é rejeitado com erro tipado, nunca aceito parcialmente.
- **RF-10:** Evento desconhecido deve ser tratado explicitamente e jamais convertido em sucesso
  silencioso.
- **RF-11:** Um provedor deve poder declarar um evento como não suportado. A ausência de suporte é
  estado próprio, distinto de sucesso e distinto de falha.
- **RF-12:** Definir resultado canônico de hook com os estados `ALLOW`, `BLOCK`, `WARN`,
  `NOT_APPLICABLE` e `ERROR`, carregando `reason`, `policy_id`, `gate_id`, `evidence` e `metadata`.
- **RF-13:** Decisões críticas não podem depender de parsing de texto livre. Todo `BLOCK` deve
  carregar motivo verificável e a identificação da policy ou gate que o originou.
- **RF-14:** `ERROR` em hook crítico deve produzir bloqueio ou estado explícito de necessidade de
  intervenção — nunca `ALLOW` automático. Hooks não críticos devem ter estratégia de falha
  declarada individualmente.
- **RF-15:** Fornecer tradução única e testada entre os exit codes dos hooks shell existentes
  (`0`, `1`, `2`) e o resultado canônico, preservando o comportamento observável de cada script
  hoje instalado.
- **RF-16:** O resultado de hook deve ser auditável após a execução, com persistência que não
  dependa do histórico privado da sessão da CLI.
- **RF-17:** O contrato deve ser versionado, com estratégia de migração declarada. Versão
  incompatível entre contrato e adapter é erro explícito, nunca degradação silenciosa.

### Bloco C — Famílias de hooks canônicos

**git-policy**

- **RF-18:** Ampliar o gate Git canônico para cobrir, além de `git commit` e `git push` já
  cobertos, as operações `git reset --hard`, `git clean`, `git checkout` e `git restore`
  destrutivos e *force push* em todas as suas variantes de escrita.
- **RF-19:** A lista de operações interceptadas deve ser derivada da policy declarada no Harness
  Contract, e não de blacklist mantida à parte. Um gate deve falhar quando a lista do script
  divergir da policy declarada.
- **RF-20:** Auto-commit e auto-push permanecem desabilitados por padrão. Operações destrutivas
  permitidas exigem aprovação explícita.
- **RF-21:** Comandos equivalentes não podem escapar por variação sintática trivial — encadeamento,
  `env`, subshell, alias de flag curta e longa, interpolação por variável.
- **RF-22:** Comandos de leitura seguros (`git status`, `git log`, `git diff`, `git show`) continuam
  permitidos, com fixture de falso positivo conhecido para cada um.
- **RF-23:** O bypass deve exigir mecanismo explícito e ser auditável, registrando quem, quando e
  qual comando. O mecanismo atual por variável de ambiente confirmada com registro em log de escapes
  é o baseline a preservar.
- **RF-24:** A família `git-policy` deve ter testes adversariais cobrindo tentativas de contorno.

**quality-gate**

- **RF-25:** O gate de qualidade deve ser disparado apenas em evento adequado, nunca a cada chamada
  de ferramenta.
- **RF-26:** Deve existir política de seleção de checks por tipo e risco da tarefa, declarada e não
  inferida pelo agente.
- **RF-27:** Falha de check obrigatório impede conclusão aprovada.
- **RF-28:** A saída do gate deve ser armazenável como evidência.
- **RF-29:** Quando houver resultado determinístico disponível, nenhuma revisão por LLM pode
  substituí-lo.
- **RF-30:** Execuções redundantes devem ser evitadas quando o estado relevante não mudou, com
  invalidação por *fingerprint*.
- **RF-31:** O gate de qualidade deve resolver build, teste e lint para as cinco stacks alvo — Go,
  Node/TypeScript, C#/.NET, Java e Python — a partir da detecção de stack existente, sem edição
  manual após a instalação.
- **RF-32:** Java deve passar a ser linguagem de primeira classe no harness: reconhecida pela
  detecção de stack como linguagem (não apenas como string de framework), presente na enumeração de
  linguagens suportadas e com resolução de toolchain para Maven e Gradle.
- **RF-33:** C#/.NET deve ter resolução de toolchain equivalente às demais stacks, hoje ausente do
  resolvedor.
- **RF-34:** A prova de execução de testes exigida pelo validador de evidência deve reconhecer os
  comandos canônicos das cinco stacks, incluindo `mvn test`, `mvn verify`, `gradle test` e
  `./gradlew test`, hoje não reconhecidos.

**evidence-gate**

- **RF-35:** Antes da conclusão de tarefa que exija evidência, o pacote de evidência deve ser
  validado deterministicamente; evidência obrigatória ausente ou inválida impede conclusão aprovada.
- **RF-36:** A integridade da evidência deve ser validada por *fingerprint* quando aplicável.
- **RF-37:** Declaração textual do agente não substitui evidência.
- **RF-38:** Evidência produzida sob um provedor deve ser validável sob qualquer outro; o gate
  permanece independente de provedor e de LLM.

**checkpoint**

- **RF-39:** O checkpoint deve persistir o estado mínimo necessário para retomada, sem depender do
  histórico privado da sessão do provedor.
- **RF-40:** A escrita do checkpoint deve ser atômica: escrita parcial não pode produzir checkpoint
  válido.
- **RF-41:** Checkpoints devem ser idempotentes e reexecutáveis sem efeito colateral.
- **RF-42:** Estado corrompido deve ser detectado e reportado explicitamente, nunca consumido como
  válido.
- **RF-43:** Uma tarefa iniciada em uma CLI deve poder ser retomada por outra quando o workflow
  permitir.
- **RF-44:** O checkpoint não pode persistir segredo nem contexto sensível desnecessário.

**telemetry**

- **RF-45:** Capturar métricas operacionais sem depender de o agente reportá-las: provedor, modelo,
  evento, identificador de tarefa, tipo de tarefa, risco, duração, número de chamadas de ferramenta,
  retentativas, skills carregadas, contexto carregado, rodadas de revisão e status final.
- **RF-46:** Métrica indisponível deve ser registrada como `unknown`; nenhum valor pode ser
  inventado ou estimado sem identificação explícita do método de cálculo.
- **RF-47:** O schema de telemetria deve ser comum aos quatro provedores, com extensões específicas
  de provedor isoladas em espaço próprio.
- **RF-48:** Falha não crítica de telemetria não pode bloquear a tarefa nem disparar cascata de
  retentativas.
- **RF-49:** Chaves, tokens e segredos não podem ser registrados em telemetria, logs ou evidência.
- **RF-50:** Deve existir correlação por tarefa e por sessão sem depender do conteúdo do prompt.

### Bloco D — Adapters, matriz e diagnóstico

- **RF-51:** Adapter Claude Code mapeando apenas os eventos necessários ao contrato canônico, com
  semântica bloqueante preservada onde suportada.
- **RF-52:** Fechar o furo de confronto do Claude: a fiação de hooks do Claude deve ser verificável
  por gate, hoje não verificada porque a única fonte que a contém não é versionada e está declarada
  como opcional no registro de configurações nativas.
- **RF-53:** Adapter Codex mapeando os eventos suportados, com as diferenças semânticas
  documentadas e os eventos ausentes declarados como capability não suportada.
- **RF-54:** Adapter OpenCode utilizando o mecanismo nativo de plugins e eventos, contendo apenas a
  integração necessária, com falha de carga do plugin observável.
- **RF-55:** Adapter GitHub Copilot CLI mantido em paridade com os demais, preservando o
  enforcement já entregue e testado para esse provedor.
- **RF-56:** Todo adapter contém tradução, nunca duplicação de policy. Um gate deve falhar quando
  lógica de decisão de policy aparecer dentro de um adapter.
- **RF-57:** Todo payload vindo do provedor é tratado como não confiável e validado estruturalmente
  antes de entrar no core.
- **RF-58:** Cada adapter deve ter suíte de testes própria.
- **RF-59:** Manter matriz de capabilities verificável cobrindo os cinco eventos canônicos e as
  cinco famílias de hooks para os quatro provedores. Cada célula afirmativa deve ter teste
  correspondente; limitações devem ser explícitas; capability específica de um provedor não pode ser
  promovida ao núcleo universal.
- **RF-60:** `doctor` deve consultar a matriz e reportar o estado de cada família de hook e de cada
  evento por provedor, detectando adapter ausente, hook não executável, configuração inválida,
  versão incompatível e divergência entre contrato e adapter, sem chamar LLM e retornando exit code
  apropriado.

### Bloco E — Conformidade, custo, segurança e distribuição

- **RF-61:** Executar suíte de conformidade com os catorze cenários da User Story contra os quatro
  provedores: comando seguro permitido; `git commit` não autorizado; `git push` não autorizado;
  comando destrutivo; teste obrigatório falhando; evidência ausente; evidência inválida; checkpoint
  válido; checkpoint corrompido; telemetria sem contagem de tokens disponível; evento desconhecido;
  capability não suportada; adapter retornando erro; tentativa de bypass por variação de comando.
- **RF-62:** As fixtures devem ser reutilizadas entre provedores quando semanticamente equivalentes,
  e o resultado esperado deve ser derivado do contrato — nunca de igualdade textual entre CLIs.
- **RF-63:** Regressão em policy crítica deve bloquear o release.
- **RF-64:** Medir, para cada hook não crítico: latência adicionada, execuções por tarefa, falhas
  evitadas, regressões detectadas, intervenções humanas evitadas e contexto adicionado. Deve existir
  *baseline* sem o hook.
- **RF-65:** Hook sem ganho observável deve ser removido ou convertido para `ON-DEMAND`. Segurança
  crítica não depende de retorno econômico para permanecer habilitada.
- **RF-66:** Todo hook deve ter timeout declarado, e o tempo gasto por hook deve ser mensurável.
- **RF-67:** Recursão hook → ferramenta → hook deve ser impedida por construção.
- **RF-68:** Argumentos vindos do provedor devem ser parseados estruturalmente quando possível; não
  pode haver `eval` ou execução dinâmica baseada diretamente em input externo; paths devem ser
  normalizados e validados.
- **RF-69:** Hooks não podem conceder permissões adicionais ao agente por padrão.
- **RF-70:** Alteração em hook crítico deve ser detectável por integridade ou versionamento.
- **RF-71:** A instalação deve distribuir apenas os adapters dos provedores configurados, com a
  lógica de core compartilhada e sem fork por provedor.
- **RF-72:** Nenhum daemon do harness pode ser obrigatório para o modo nativo das CLIs, salvo
  capability explicitamente documentada que realmente o exija.
- **RF-73:** A suíte de portabilidade deve cobrir projeto-fixture mínimo para cada uma das cinco
  stacks alvo, provando que a instalação produz quality gate e evidence gate funcionais sem edição
  manual.
- **RF-74:** Todos os gates e testes existentes do repositório devem continuar passando; a entrega
  não pode alterar comportamento observável de nenhum hook classificado `KEEP` sem registro
  explícito.
- **RF-75:** Documentar eventos, policies, gates, limitações por provedor, limitações por stack e
  procedimento de troubleshooting.

---

## Cobertura da User Story

Mapeamento entre os requisitos da User Story e o tratamento neste PRD. `Delegado` significa que o
requisito já está especificado e agendado no PRD dependente
`.specs/prd-harness-portatil-vendor-neutral/`, e **não** é reespecificado aqui.

| US | Tratamento | Onde |
|---|---|---|
| RF01 Inventário | Especificado | RF-01, RF-02, RF-05 |
| RF02 Classificação | Especificado | RF-03, RF-04 |
| RF03 Canonical Contract | Especificado | RF-06 a RF-11, RF-17 |
| RF04 Resultado canônico | Especificado | RF-12 a RF-16 |
| RF05 Git Policy | Parcialmente entregue + delta | Baseline: tarefa 9.0 `done` (`git-operation-gate.sh`). Delta: RF-18 a RF-24 |
| RF06 Quality Gate | Especificado | RF-25 a RF-34 |
| RF07 Evidence Gate | Parcialmente entregue + delta | Baseline: `validate-task-evidence.sh` fail-closed, `seal-evidence`. Delta: RF-35 a RF-38 |
| RF08 Checkpoint | Parcialmente entregue + delta | Baseline: `post-wave.sh`, retomada do `task-loop`. Delta: RF-39 a RF-44 |
| RF09 Telemetry | Parcialmente delegado + delta | Delegado: telemetria comparável — tarefa 11.0. Delta de schema e política `unknown`: RF-45 a RF-50 |
| RF10 Claude Adapter | Especificado | RF-51, RF-52 |
| RF11 Codex Adapter | Especificado | RF-53 |
| RF12 OpenCode Adapter | Especificado | RF-54 |
| — Copilot Adapter | Acrescentado ao escopo | RF-55 (decisão registrada em *Suposições*) |
| RF13 Capability Matrix | Delegado + extensão | Delegado: geração da matriz — tarefa 6.0. Extensão aos cinco eventos e cinco famílias: RF-59 |
| RF14 Hook Doctor | Delegado + extensão | Delegado: `doctor` multi-provedor — tarefa 11.0. Extensão a hooks e adapters: RF-60 |
| RF15 Conformance | Delegado + extensão | Delegado: suíte cross-provider — tarefa 10.0. Extensão aos catorze cenários: RF-61 a RF-63 |
| RF16 Ablation e custo | Delegado + extensão | Delegado: ablation — tarefa 11.0. Extensão: RF-64, RF-65 |
| RF17 Performance e dedup | Especificado | RF-30, RF-66, RF-67 |
| RF18 Segurança | Especificado | RF-49, RF-57, RF-68, RF-69, RF-70 |
| RF19 Instalação e sync | Delegado + delta | Delegado: aliases `init`/`sync`, conflito transacional — tarefa 8.0; fundação de checksum — tarefa 7.0 `done`. Delta: RF-71, RF-72 |
| RNF01 Determinismo | Transversal | RF-12, RF-13, RF-29 |
| RNF02 Vendor neutrality | Transversal | RF-07, RF-56 |
| RNF03 Baixo overhead | Transversal | RF-30, RF-64, RF-66 |
| RNF04 Observabilidade | Transversal | RF-45, RF-66 |
| RNF05 Auditabilidade | Transversal | RF-13, RF-16, RF-23 |
| RNF06 Testabilidade | Transversal | RF-58, RF-61 |
| RNF07 Portabilidade | Transversal + ampliado | RF-71, RF-73 — ampliado às cinco stacks |
| RNF08 Idempotência | Transversal | RF-41 |
| RNF09 Compatibilidade evolutiva | Transversal | RF-17 |
| RNF10 Simplicidade | Transversal | RF-03, RF-05, RF-65 |

**Avaliação de cobertura:** os dezenove RFs e os dez RNFs da User Story estão cobertos — nove
integralmente especificados aqui, seis com delta sobre baseline já entregue e quatro com delegação
explícita a tarefas já agendadas no PRD dependente. A User Story é atendível em 100%, condicionada
às duas premissas registradas em *Suposições e Questões em Aberto*.

---

## Restrições Técnicas de Alto Nível

- **Ordem de execução com o PRD dependente.** Sete tarefas de
  `.specs/prd-harness-portatil-vendor-neutral/` permanecem `pending`, quatro delas cobrindo
  requisitos que este PRD delega (6.0 matriz, 8.0 install/sync, 10.0 conformidade, 11.0 telemetria e
  ablation). As tarefas deste PRD que estendem esses itens não podem ser executadas antes das
  tarefas delegadas correspondentes, sob pena de conflito de arquivo e de `spec-hash` divergente.
- **Nenhum dos dois modelos de ponto pode quebrar durante a reconciliação.**
  `internal/runtime/specs` serve ao gate de paridade de configuração nativa;
  `internal/runtime/hooks` serve ao runtime orquestrado ACP. Ambos têm testes e consumidores vivos.
- **O harness nunca altera `.claude/hooks/*.sh` em modo orquestrado.** Hooks Go servem o modo
  orquestrado, hooks shell servem o modo interativo, e essa separação é invariante existente.
- **Quatro provedores, não três.** Copilot está no registro de agentes, tem enforcement entregue,
  gate de paridade e ADR-012 aceita. Reduzir a três provedores seria regressão.
- **Binário `ai-spec`.** A User Story usa `orchestrator init` e `orchestrator sync` como nomes
  genéricos. O binário publicado é `ai-spec`, distribuído por GoReleaser e Homebrew; renomeá-lo
  quebraria instalações existentes. Os aliases `init`/`sync` são escopo da tarefa 8.0 do PRD
  dependente.
- **Idioma e comentários.** `R-STYLE-001` é regra `hard`: código em inglês, zero comentários. Os
  termos de domínio deste documento em português não podem migrar literalmente para identificadores.
- **Protocolo PRD-First e âncora de confiança.** Alterações de comportamento exigem RF mapeado;
  edições em PRD exigem ressincronização de `spec-hash`.
- **Compatibilidade de plataforma.** Os hooks shell precisam funcionar em bash 3.x (padrão do
  macOS) e sob `mawk` no Linux; o CI executa a matriz `ubuntu-24.04` e `macos-15`.
- **Gates de cobertura.** 75% total e 70% por pacote crítico permanecem vigentes.
- **Sem dependências novas sem demanda concreta.** As dependências diretas atuais são o SDK ACP,
  tiktoken-go, jsonschema/v6, yaml.v3 e testify.

---

## Fora de Escopo

Herdado da User Story e mantido:

- criar novos agentes especialistas;
- substituir o fluxo SDD por hooks;
- executar revisão semântica de código dentro de hooks;
- *agent swarm*, votação multi-modelo, OpenRouter;
- barramento de eventos distribuído;
- banco de dados introduzido apenas para hooks;
- MCP criado apenas para transportar eventos de hook;
- executar toda a suíte de testes após cada chamada de ferramenta;
- injetar prompts extensos em hooks;
- replicar integralmente os hooks por provedor;
- habilitar auto-commit, auto-push ou auto-merge.

Acrescentado pelas decisões desta sessão:

- **remover efetivamente qualquer hook nesta entrega.** A classificação `REMOVE` produz o
  diagnóstico; a remoção é trabalho posterior, condicionada à cobertura do invariante no core
  (RF-04);
- **renomear o binário `ai-spec`** ou publicar `orchestrator` como segundo nome de binário;
- **descontinuar o adapter Copilot**;
- **reespecificar os requisitos delegados** ao PRD dependente (matriz gerada, aliases `init`/`sync`,
  suíte cross-provider, telemetria comparável e ablation) — este PRD apenas os estende;
- **uma sexta família de hook** além de `git-policy`, `quality-gate`, `evidence-gate`, `checkpoint`
  e `telemetry`;
- **suporte a stacks além de Go, Node/TypeScript, C#/.NET, Java e Python** — Rust, PHP, Ruby e
  outras ficam fora, ainda que o validador de evidência já tolere alguns de seus comandos de teste;
- **skill de implementação para Java.** RF-32 torna Java cidadão de primeira classe na detecção,
  na enumeração de linguagens e na resolução de toolchain; escrever
  `.agents/skills/java-implementation/` (planejado) é trabalho de outro PRD.

---

## Suposições e Questões em Aberto

**Suposições assumidas** (decididas nesta sessão, registradas para auditoria):

1. **Sequenciamento.** Este PRD cobre apenas o delta; os requisitos já agendados permanecem no PRD
   dependente. Nenhuma edição é feita em `.specs/prd-harness-portatil-vendor-neutral/prd.md`, e o
   `spec-hash` das seis tarefas já concluídas lá permanece íntegro.
2. **Copilot incluído** como quarto provedor em todos os requisitos de adapter, matriz e
   conformidade.
3. **Binário mantido como `ai-spec`**; `orchestrator` é lido como nome genérico na User Story.
4. **`REMOVE` não executa nesta entrega** sem cobertura prévia do invariante por teste no core.
5. **Cinco stacks alvo** — Go, Node/TypeScript, C#/.NET, Java e Python. Python entra porque já é
   cidadão de primeira classe no repositório; excluí-lo seria regressão.

**Questões em aberto:**

1. **Dependência de execução.** As tarefas deste PRD que estendem itens delegados dependem da
   conclusão das tarefas 6.0, 8.0, 10.0 e 11.0 do PRD dependente. Essa dependência precisa ser
   resolvida no planejamento de tarefas: ou este PRD é executado depois, ou as tarefas são
   intercaladas com acoplamento explícito. A decisão pertence à etapa de `create-tasks` e não altera
   os requisitos.
2. **Semântica de `BeforeComplete` no OpenCode.** O plugin do OpenCode hoje usa `session.idle` como
   ponto de fim de sessão. Não está verificado se existe evento nativo com semântica de
   "antes de concluir" distinta de fim de sessão. A matriz de capabilities deve registrar
   `adapter/limitation` até que a documentação oficial e um teste confirmem o contrário —
   nunca presumir equivalência (princípio P07 da User Story).
3. **`SessionStart` nas quatro CLIs.** O vocabulário de chaves nativas hoje declarado no registro
   cobre apenas os três pontos atuais. O suporte real a um evento de início de sessão em cada CLI
   precisa ser confirmado contra documentação oficial e teste antes de a matriz marcar
   `supported`.
4. **Origem da fiação de hooks do Claude.** Fechar RF-52 exige decidir entre versionar
   `.claude/settings.json` no repositório consumidor, tornar a fonte obrigatória no registro, ou
   verificar a fiação por outro meio. As três opções têm efeitos distintos sobre projetos
   consumidores e a escolha pertence à especificação técnica.
5. **Retomada cross-CLI (RF-43).** Não está verificado quanto do estado necessário à retomada hoje
   vive apenas no histórico privado de cada CLI. A viabilidade plena de RF-43 depende desse
   levantamento, que é parte do inventário (RF-01).
