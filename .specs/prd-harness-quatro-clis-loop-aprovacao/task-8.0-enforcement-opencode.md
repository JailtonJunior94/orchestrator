# Tarefa 8.0: Enforcement não-desligável do OpenCode

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

O gate de governança do OpenCode passa a existir de fato, e passa a ser **não-desligável**.

O enforcement **primário** é o hook de **pré-ferramenta lançando exceção**, delegando a decisão aos
scripts canônicos. Isso não é escolha de estilo: foi **comprovado por execução real sob o runtime ACP**,
com cliente de protocolo escrito para o teste — a ferramenta nunca executa, a chamada é reportada com
estado de falha, a sessão sobrevive e o modelo lê a mensagem e se adapta. O bloqueio é robusto a
exceções consecutivas na mesma sessão e **intercepta ferramenta executada dentro de subagente**.

O hook de **permissão é PROIBIDO no desenho**. Ele é **código morto** na versão corrente — enumerando
todos os disparos de hook do binário, ele não aparece — e um gate baseado nele **falharia em silêncio**,
que é o pior modo de falha possível para um controle de segurança.

Como **defesa em profundidade**, o bloco `permission` nega declarativamente o que é categoricamente
proibido, com **padrões finos**: o padrão total remove a ferramenta do conjunto oferecido ao modelo, o
que é útil para o que não deve existir mas inaceitável para ferramenta que precisa existir. O valor
"perguntar" é **PROIBIDO** em orquestração, por ser não-determinístico entre modos de execução.

Contra os **três interruptores** que desligam o gate por completo, **duas camadas obrigatórias**:
sanitização do ambiente do **processo filho que o harness cria** (sem tocar no ambiente do usuário) e
**handshake ativo por sentinela**, que aborta a sessão se o sinal de carga não chegar antes do primeiro
prompt. A camada de handshake valida o **efeito**, não a causa, e por isso cobre também vetores futuros
desconhecidos.

<requirements>
- **RF-19:** O enforcement primário é o hook de **pré-ferramenta que lança exceção**, delegando aos
  scripts canônicos — comprovadamente bloqueante sob ACP e dentro de subagentes (V-04, V-05). É
  **PROIBIDO** basear o gate no hook de permissão, que é código morto (V-06). A mensagem da exceção é
  redigida como **instrução corretiva**, porque o modelo a lê e reage a ela.
- **RF-20:** O bloco `permission` nega declarativamente o que é categoricamente proibido, com **padrões
  finos** — o padrão total remove a ferramenta do tool-set oferecido ao modelo (V-07). O valor
  "perguntar" é **PROIBIDO** em orquestração: seu comportamento diverge entre modos de execução.
- **RF-21:** O gate não pode ser desligável. **Duas camadas obrigatórias**: (a) o harness nunca passa a
  flag de modo puro e **sanitiza o ambiente do processo filho que ele mesmo cria**, removendo os
  interruptores conhecidos (V-08, V-11) — **sem alterar o ambiente do usuário**; (b) **handshake ativo**:
  o plugin sinaliza sua carga e a sessão é **abortada** se o sinal não chegar antes do primeiro prompt.
- **RF-23:** Se o **runtime necessário ao plugin** não estiver disponível, a **instalação e a verificação
  falham de forma ruidosa**. Degradação silenciosa é proibida.
- **Timeout do validador é NEGAÇÃO**, nunca aprovação. Tratar timeout como aprovação reproduz exatamente
  o defeito do gate inerte.
- **Validador ausente bifurca por modo:** em execução orquestrada é **falha fechada**; em uso interativo
  avisa **uma vez** e segue, porque ali o usuário está no comando.
- **Só o ponto de pré-ferramenta bloqueia** — é o único que impede algo ainda não feito. Os outros dois
  pontos canônicos são cobertos, mas observacionais.
- O agente cuja política de ambiente é **zero-value** tem o ambiente herdado **intacto**, o que preserva
  os três agentes atuais sem regressão (RF-62, O-06).
</requirements>

## Subtarefas

- [x] 8.1 Implementar o plugin de governança do OpenCode com o hook de **pré-ferramenta**, bloqueando
      por **exceção** e delegando a decisão aos scripts canônicos — sem reimplementar lógica de gate.
- [x] 8.2 Redigir a mensagem da exceção como **instrução corretiva** (o que fazer em seguida), não como
      log de erro: o modelo lê o texto e se adapta a ele.
- [x] 8.3 **Proibir explicitamente** o hook de permissão: nenhum registro dele no plugin, e gate de
      varredura que reprova sua reintrodução, com comentário citando V-06.
- [x] 8.4 Implementar as **três camadas de custo** do plugin: filtro por ferramentas que mutam o
      repositório; um único `stat` por sessão para resolver a existência do validador; memoização por
      `(ferramenta, arquivos)`. O custo assintótico fica em um shell-out por arquivo distinto tocado,
      não por chamada de ferramenta.
- [x] 8.5 Implementar **timeout do validador como negação**, com teste dedicado que injeta um validador
      lento e asserta bloqueio.
- [x] 8.6 Implementar a **bifurcação por modo** para validador ausente: orquestrado falha fechado;
      interativo avisa uma vez (e apenas uma vez) e segue.
- [x] 8.7 Cobrir os três pontos canônicos no plugin, com **apenas o de pré-ferramenta bloqueando**.
- [x] 8.8 Escrever o bloco `permission` no `opencode.json` com **padrões finos**, e gate que reprova
      padrão total em ferramenta que precisa existir.
- [x] 8.9 **Proibir o valor "perguntar"** no bloco `permission` gerado, com gate de varredura sobre o
      asset e sobre a saída do instalador.
- [x] 8.10 Implementar a **política de ambiente** no registro (`EnvPolicy`), sanitizando o ambiente do
      **processo filho criado pelo harness** contra os interruptores conhecidos — flag de modo puro
      nunca passada, e as variáveis de desligamento removidas do ambiente do filho.
- [x] 8.11 Garantir que a sanitização **não toca no ambiente do usuário** e que o agente com `EnvPolicy`
      zero-value herda o ambiente **byte-idêntico** ao de hoje.
- [x] 8.12 Implementar o **waiter do sentinela** como componente próprio: caminho **único por sessão**,
      **removido antes do spawn** — sentinela de execução anterior faria o handshake passar com o plugin
      desligado.
- [x] 8.13 Encaixar o handshake no **único ponto correto do ciclo de vida**: **depois** da negociação do
      protocolo, quando o subprocesso já carregou plugins, e **antes** do primeiro prompt. Falhar ali
      mata o processo sem que o modelo veja uma única palavra.
- [x] 8.14 Implementar o ponto de extensão no cliente de protocolo seguindo o precedente já existente no
      runner, para não churnar os fakes de teste.
- [x] 8.15 Fazer **instalação e verificação falharem ruidosamente** quando o runtime necessário ao plugin
      estiver ausente (RF-23), com mensagem citando a dependência exata.
- [x] 8.16 Replicar sob o runtime ACP os **dois testes de interruptor** executados apenas no modo direto,
      e **registrar o resultado** — obrigação declarada no PRD §Suposições e no `adr-004` §Riscos.
- [x] 8.17 Registrar a métrica de **sessões recusadas por pré-condição** na telemetria opt-in existente,
      sem novo mecanismo de consentimento.

## Detalhes de Implementação

Ver `techspec.md`:

- §Sequenciamento de Desenvolvimento → **Enforcement do OpenCode** — as três decisões de desenho que vêm
  direto da evidência: custo em três camadas, timeout como negação, validador ausente bifurcando por
  modo.
- §Sequenciamento de Desenvolvimento → **Sanitização de ambiente e handshake** — por que o handshake
  encaixa depois da negociação e antes do primeiro prompt, e por que o sentinela é removido antes do
  spawn.
- §Sequenciamento de Desenvolvimento → Fases — **F3c — Enforcement**, dependente de F3b.
- §Monitoramento e Observabilidade — o bloqueio efetivo é observável **no próprio fluxo do protocolo**
  (atualização da chamada de ferramenta com estado de falha e o texto da exceção), o que permite
  registrar evidência de gate acionado **sem depender de ler o log do plugin**.
- §Abordagem de Testes → Testes de Integração — as três camadas de prova (matriz obrigatória, disparo
  simulado no CI, disparo verdadeiro pelo CLI em job noturno fora do gate de merge).
- `adr-004-enforcement-opencode-handshake.md` — a decisão completa, incluindo o registro de que o
  mecanismo foi decidido **duas vezes** e por que a reversão é a informação relevante; as alternativas
  rejeitadas (hook de permissão, permissões sem plugin, detecção estática, neutralizar variáveis no
  ambiente do usuário).
- `adr-005-precondicoes-de-enforcement.md` — os estados `inert` e `unknown` que a verificação passa a
  reportar.

Não duplicar aqui a spec, a detecção nem a instalação de pegada mínima: pertencem à tarefa 7.0.

## Critérios de Sucesso

- `go build ./... && go vet ./... && go test ./... -count=1` verde ao final da tarefa.
- **Teste de disparo real sob o runtime ACP** prova que uma ação proibida é **bloqueada**: a ferramenta
  não executa, a atualização da chamada chega com estado de falha e o texto da exceção, e a sessão
  sobrevive.
- Teste prova o bloqueio **dentro de subagente**, não apenas no nível superior.
- Teste prova robustez a **exceções consecutivas** na mesma sessão (o bloqueio não degrada após a
  primeira).
- Gate de varredura reprova qualquer registro do **hook de permissão** no plugin — o teste falha se ele
  for reintroduzido.
- Teste prova que **timeout do validador resulta em negação**: validador artificialmente lento produz
  bloqueio, nunca liberação.
- Teste prova a **bifurcação por modo** do validador ausente: orquestrado falha fechado; interativo emite
  aviso exatamente **uma** vez e prossegue.
- Bloco `permission` gerado contém **apenas padrões finos** para ferramentas que precisam existir, e
  **nenhuma** ocorrência do valor "perguntar" — verificável por gate sobre o asset e sobre o arquivo
  escrito.
- Teste prova que o argv do OpenCode **nunca** contém a flag de modo puro.
- Teste prova que o ambiente do **processo filho** não contém nenhum dos interruptores conhecidos, e que
  o ambiente do **processo pai** permanece inalterado.
- Teste prova que o agente com `EnvPolicy` zero-value herda ambiente **byte-idêntico** ao anterior (os
  três agentes atuais sem regressão).
- Teste prova que, **sem o sinal do sentinela**, a sessão é **abortada antes do primeiro prompt** — e que
  nenhum prompt chega ao modelo nesse caminho.
- Teste prova que o caminho do sentinela é **único por sessão** e que um sentinela remanescente de
  execução anterior **não** faz o handshake passar (o arquivo é removido antes do spawn).
- `ai-spec-harness install .` e `ai-spec-harness verify .` **falham com saída não-zero e mensagem
  explícita** quando o runtime necessário ao plugin está ausente — nenhum caminho reporta `current`.
- Os **dois testes de interruptor** antes executados apenas no modo direto foram replicados sob ACP, com
  o resultado registrado na evidência da tarefa.
- Métrica de sessões recusadas por pré-condição aparece no relatório com `GOVERNANCE_TELEMETRY=1` e
  **não** aparece sem a variável.
- Matriz obrigatória (unitária) verde: o OpenCode cobre os três pontos canônicos e aponta para os mesmos
  scripts canônicos dos demais agentes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
  - Política de ambiente: sanitização remove cada interruptor conhecido; zero-value herda intacto;
    ambiente do pai inalterado.
  - Waiter do sentinela: caminho único por sessão; remoção antes do spawn; ausência de sinal aborta.
  - Geração do bloco `permission`: padrões finos, ausência do valor "perguntar", idempotência.
  - Memoização por `(ferramenta, arquivos)` e filtro por ferramentas que mutam o repositório.
  - Timeout como negação; validador ausente por modo.
  - Falha ruidosa por runtime do plugin ausente (instalação e verificação).
- [x] Testes de integração
  - **Disparo simulado** (roda no CI, sem CLI instalado): o script instalado, executado com o contrato
    de entrada documentado, produz saída não-zero para entrada que viola governança.
  - **Disparo real sob ACP** com cliente de protocolo mínimo: bloqueio no nível superior e dentro de
    subagente; exceções consecutivas; estado de falha e texto da exceção observados no fluxo do
    protocolo.
  - Handshake: sessão abortada sem sentinela; sessão prossegue com sentinela; sentinela residual não
    valida a sessão.
  - Replicação sob ACP dos dois testes de interruptor antes feitos no modo direto, com resultado
    registrado.
  - Não-regressão dos três agentes atuais: ambiente herdado e argv byte-idênticos.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `.opencode/plugin/governance.js` (planejado) — plugin de governança: hook de pré-ferramenta que bloqueia
  por exceção, sinaliza carga via sentinela e cobre os três pontos canônicos.
- `internal/runtime/specs/enforcement.go` (planejado) — `PontoCanonico`, `Enforcement` e
  `PreCondicaoDeEnforcement` (criados na tarefa 6.0), aqui preenchidos para o OpenCode.
- `internal/runtime/specs/envpolicy.go` (planejado) — política de ambiente; zero-value preserva os três
  agentes atuais.
- `internal/runtime/handshake/` (planejado) — waiter do sentinela como componente próprio.
- `internal/runtime/client/client.go` — ponto de extensão do cliente de protocolo, seguindo o precedente
  existente no runner para não churnar os fakes.
- `internal/install/install.go` — escrita do bloco `permission`, depósito do plugin e falha ruidosa por
  runtime ausente (RF-23).
- `cmd/ai_spec_harness/verify.go` — reporte dos estados `inert` e `unknown` das pré-condições.
- `.agents/scripts/` — validadores canônicos aos quais o plugin delega; nenhuma lógica de gate é
  reimplementada no plugin (ADR PP-001).
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — §Enforcement do OpenCode,
  §Sanitização de ambiente e handshake, §Abordagem de Testes (três camadas de prova).
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-004-enforcement-opencode-handshake.md`
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-005-precondicoes-de-enforcement.md`
