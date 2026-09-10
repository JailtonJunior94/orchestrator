# Documento de Requisitos do Produto (PRD)

<!-- spec-version: 3 -->

**Feature:** Memória durável de agentes (multi-tier, cross-CLI, offline)
**Slug:** `memoria-duravel-agentes`
**Origem:** análise do `ai-memory` 2.0 (akitaonrails, MIT) confrontada com o subsistema `internal/runtime/memory` do harness
**Modelo de domínio:** [`discoveries/domain-memoria-duravel-de-agentes/domain-model.md`](../../discoveries/domain-memoria-duravel-de-agentes/domain-model.md) — validado por `validate-bundle.py` (SUCCESS)
**Estado das decisões:** 12 decisões de produto + 20 decisões de modelagem, todas fechadas. Nenhuma questão em aberto.

## Visão Geral

O harness já orquestra sessões de agentes de código em quatro CLIs (Claude, Codex, Gemini, Copilot) sobre
um protocolo comum (ACP) e já persiste evidência rica de cada sessão (`events.jsonl`, `tool_calls.md`,
`execution_report.md`). O que ele **não** tem é conhecimento que sobreviva à sessão.

O subsistema de memória atual (F3-Claude) é, na prática, um bloco de notas da última sessão:

1. `MemoryPersistHook` grava `MEMORY.md` em modo `replace` com um template fixo de quatro campos
   (task, exit status, contagem de eventos, contagem de tool calls). **Cada sessão apaga a anterior.**
2. `Store.WriteTask` existe na interface e nunca é chamado em produção — o tier de task é lido em
   `internal/runtime/runner.go` e nunca escrito.
3. A recuperação é integral: o arquivo inteiro entra no prompt. Ao estourar o limite de compactação, o
   único mecanismo é anexar a frase `compact the flagged memory files before proceeding` ao prompt e
   confiar que o agente colabore.
4. O escopo é `.specs/<prd>/memory/`. Nada atravessa PRDs; nada consolida aprendizado do repositório.

O efeito é o custo que o `ai-memory` nomeia bem: **re-explicar a arquitetura a cada sessão**. Em uso real
de produção isso aparece como retrabalho silencioso — o agente redescobre convenções já decididas, refaz
diagnósticos já feitos, e repete erros já corrigidos em sessões anteriores.

Esta feature transforma o subsistema de memória num sistema de conhecimento durável de três camadas
(projeto → PRD → task), com captura automática a partir da evidência que o harness já produz,
consolidação não-destrutiva, e recuperação seletiva sob orçamento de tokens. Tudo nativo em Go, dentro do
binário único, sem daemon, sem rede e sem chamada de LLM no caminho default.

### Por que os conceitos do ai-memory 2.0, mas não sua implementação

O modelo conceitual do `ai-memory` — **captura → consolidação → recuperação → handoff**, com Markdown
versionado em git como fonte de verdade e banco derivado descartável — é diretamente aplicável e é o que
este PRD adota.

A implementação dele não é adotável aqui, e a razão é de produto, não de gosto:

| Característica do ai-memory | Conflito com o harness |
|---|---|
| Binário Rust + daemon Docker em `127.0.0.1:49374` | ADR-001 (binário único + `go:embed`) e a história de instalação portátil (`install`/`verify`) |
| SQLite/FTS5 (CGO) | Restrição de build cross-plataforma sem CGO (CI: ubuntu-24.04 + macos-15 + Windows) |
| Embeddings locais (~87 MB, candle) | Tamanho de distribuição e determinismo de evidência |
| Estado central mutável fora do repositório | Evidência do harness é file-first, auditável e re-verificável (selo `seal-evidence`, `commit_patch_sha256`) |
| Auth multiusuário + audit log | Fora do modelo de execução local do harness |

Nenhum código do `ai-memory` é copiado, portado ou vendorado. A licença MIT dele permitiria, mas a
linguagem e a arquitetura divergem — o que se reaproveita é o desenho, publicamente documentado.

## Objetivos

Cada objetivo tem critério de verificação executável. Um objetivo sem critério verificável não entra.

| # | Objetivo | Critério de aceitação mensurável |
|---|---|---|
| O-1 | Continuidade real entre sessões e entre CLIs | ≥ 90% dos fatos duráveis emitidos na sessão N presentes no contexto injetado da sessão N+1, medido sobre conjunto fixo de fixtures de sessão, incluindo troca de CLI |
| O-2 | Zero regressão comprovada | Suíte atual (182 arquivos de teste) verde sem alteração de expectativa **e** teste de paridade dedicado provando prompt final byte-idêntico com a feature desativada, para insumos idênticos |
| O-3 | Custo de contexto sob controle | Bloco de memória injetado ≤ 15% do teto de tokens da `WindowClass` ativa, verificado pelo hook `token_budget` existente; violação falha o teste |
| O-4 | Determinismo e ausência de rede | Teste que falha se o caminho default abrir socket, resolver DNS ou invocar CLI externa durante captura, recuperação ou compactação |
| O-5 | Compactação independente do agente | Após qualquer sessão, nenhuma página excede seus limites — inclusive com agente fake que ignora toda diretiva textual |
| O-6 | Portabilidade sem lock-in | Toda memória permanece Markdown válido; `memory export` produz artefato autocontido; nenhum estado fora do repositório é necessário para leitura |
| O-7 | Qualidade de código mantida | Cobertura ≥ 75% (threshold de CI) nos pacotes novos e alterados; `make lint` e `make vet` limpos |
| O-8 | Recuperação imperceptível | p95 < 200 ms para 1.000 páginas de memória, medido por benchmark (`make bench`) |

## Histórias de Usuário

**Persona primária — Dev que orquestra tarefas via harness**

- Como dev, quero que o agente já saiba as convenções e decisões do repositório ao abrir uma sessão, para
  não gastar as primeiras trocas re-explicando arquitetura.
- Como dev, quero abandonar uma sessão no Claude e retomar no Codex no mesmo PRD, para escolher a CLI por
  disponibilidade e custo sem perder o fio do trabalho.
- Como dev, quero retomar um PRD parado há semanas e receber o estado real (o que foi feito, o que falhou,
  o que foi decidido), para não reabrir diagnósticos já concluídos.
- Como dev, quero que um erro diagnosticado numa task não seja repetido na task seguinte, para que o custo
  do aprendizado seja pago uma vez.
- Como dev, quero decidir explicitamente o que virou conhecimento permanente do repositório, para que a
  memória de projeto não vire lixão de detalhe efêmero.

**Persona secundária — Revisor / auditor**

- Como revisor, quero inspecionar exatamente o que o agente gravou como memória e de qual sessão cada fato
  veio, para julgar se o contexto que guiou uma decisão era legítimo.
- Como revisor, quero que memória contraditória seja sinalizada, para não aprovar uma mudança guiada por
  um fato já invalidado.
- Como revisor, quero garantia de que nenhum segredo foi persistido em arquivo versionado, porque a
  memória de projeto vai para o remoto.

**Persona terciária — Mantenedor do harness**

- Como mantenedor, quero ativar a feature por opt-in e desativá-la por flag, para que qualquer regressão
  em produção seja mitigável sem rollback de versão.
- Como mantenedor, quero que toda desativação ou degradação apareça em log, porque um gate que se desliga
  em silêncio é indistinguível de um gate que aprovou.
- Como mantenedor, quero migração explícita com backup verificável, para que a adoção do novo formato seja
  reversível por decisão e não por acidente.

**Casos de borda cobertos por requisito**

| Caso de borda | Requisito que cobre |
|---|---|
| Duas CLIs abertas no mesmo projeto (escrita concorrente) | RF-16 |
| Duas sessões disputando a continuidade do mesmo trabalho | RF-23 |
| Detentor do bastão morre por crash ou `kill -9` | RF-23 |
| Repositório sem memória alguma (primeira execução) | RF-21 |
| Página corrompida ou editada à mão de forma inválida | RF-22 |
| Fato contraditório com fato anterior | RF-27 |
| Segredo presente no conteúdo a persistir | RF-15 |
| Agente que ignora toda diretiva textual | RF-09, RF-13 |
| Sessão encerrada por timeout, cancelamento ou permissão negada | RF-08 |
| Uso ad-hoc sem PRD (`TasksDir` vazio) | RF-26 |
| Memória crescendo indefinidamente | RF-13, RF-14 |

## Funcionalidades Core

### 1. Memória em três camadas

| Camada | O que guarda | Vive em | Por quê |
|---|---|---|---|
| **Projeto** | conhecimento durável do repositório: arquitetura, convenções, armadilhas recorrentes, decisões | `.aispec/memory/`, versionado em git | atravessa PRDs, sessões e CLIs; é o que hoje não existe |
| **PRD (workflow)** | estado e aprendizado do PRD ativo | `.specs/<prd>/memory/` | camada atual, preservada sem mudança de caminho |
| **Task** | contexto específico de uma task | `.specs/<prd>/memory/` | fecha o tier hoje inerte |

A separação existe para controlar vazamento: um detalhe efêmero de task não deve poluir permanentemente o
conhecimento do repositório, e o conhecimento do repositório não deve ser reescrito por uma sessão isolada.
A passagem de uma camada para outra é sempre uma decisão explícita (RF-24), nunca um efeito colateral.

`.aispec/` foi escolhido por ser namespace próprio do harness, já reconhecido como marcador de raiz de
projeto no upward-walk de config, e por **não** ser território do instalador — memória gravada por sessão
nunca pode ser confundida com asset driftado por `ai-spec verify`.

### 2. Captura automática sem cerimônia, de fonte dupla

A memória é derivada da sessão sem que o usuário diga "lembre disso" e sem uma chamada de LLM adicional.
São duas fontes com garantias diferentes, deliberadamente:

- **Sinal estruturado (determinístico, não depende do agente):** task e status de saída, gates aprovados e
  reprovados, arquivos tocados, comandos de validação executados, referências a ADR e RF, métricas de
  token. O harness já produz tudo isso.
- **Seção declarada pela sessão (rica, depende de colaboração):** fatos que só quem executou conhece —
  diagnósticos, decisões de caminho, armadilhas encontradas.

A segunda fonte enriquece; a primeira garante. Se o agente não colaborar, a memória fica mais pobre e
continua correta — nunca vazia.

### 3. Consolidação não-destrutiva com arquivamento

Escrita de memória acumula e consolida em vez de sobrescrever. Fato novo entra; fato existente não é
apagado por uma sessão que simplesmente não o mencionou. Regravação idêntica não duplica. Fatos que saem de
circulação são **arquivados**, não deletados: continuam no repositório e fora do contexto injetado.

### 4. Recuperação seletiva sob orçamento com cota por camada

Em vez de injetar arquivos inteiros, o harness seleciona o que é relevante para a task atual sob um teto
global derivado da `WindowClass` (ADR-023), subdividido em cotas por camada. Sobra de uma camada é cedida
às outras, então cota não vira desperdício.

### 5. Busca local determinística, sem índice persistido

Busca por texto e por entidade sobre a memória, sem rede, sem chave de API e sem LLM. Não há índice
persistido: o conteúdo é lido e filtrado a cada consulta. A escolha elimina de raiz três classes de defeito
— índice obsoleto, invalidação incorreta e incompatibilidade de versão de formato de índice — ao custo de
performance que o volume alvo não cobra.

### 6. Handoff tipado com lease

Um bastão de continuidade com dono único: exatamente uma sessão o reivindica, e uma segunda reivindicação é
recusada de forma explícita em vez de sobrescrever. O bastão é um lease com prazo, renovado enquanto a
sessão vive, e reivindicável por outra sessão quando o prazo vence ou o processo dono não existe mais.
Crash e `kill -9` não deixam o trabalho travado.

### 7. Ligações tipadas e detecção de contradição

Páginas se relacionam de forma explícita — substituição, causa, correção, contradição. Relações que se
contradizem são sinalizadas sem consumir LLM, na inspeção e no contexto injetado.

### 8. Superfície de CLI própria

`ai-spec memory` com subcomandos para auditar, buscar, exportar, compactar, migrar e gerenciar o bastão de
handoff. Namespace novo: nenhum comando existente muda de contrato ou de output.

## Requisitos Funcionais

### Camadas e formato

- RF-01: A memória deve ter três camadas — projeto, PRD e task — com resolução determinística de qual
  camada recebe cada fato registrado, dado o contexto da sessão.
- RF-02: A camada de projeto deve residir em `.aispec/memory/` no repositório, ser versionável em git, e
  ser legível e editável como Markdown puro por humanos e por ferramentas externas (`grep`, editor,
  Obsidian).
- RF-03: A camada de PRD deve preservar o caminho e o nome de arquivo atuais
  (`.specs/<prd>/memory/MEMORY.md`), de modo que memória já existente permaneça válida e localizável.
- RF-04: A camada de task deve ser efetivamente escrita ao final de cada sessão vinculada a uma task,
  eliminando a lacuna atual em que `Store.WriteTask` nunca é invocado em produção.
- RF-05: Cada página de memória deve ser um documento Markdown válido com frontmatter declarando
  identidade, camada, sessão de origem, data e versão de formato.
- RF-06: O sistema deve suportar ligações tipadas entre páginas, expressando ao menos as relações
  *substitui*, *causa*, *corrige* e *contradiz*.
- RF-07: Páginas das camadas de task e PRD devem poder carregar marcação de durabilidade, que é a única
  entrada válida para o mecanismo de promoção definido em RF-24.

### Captura e consolidação

- RF-08: A captura deve ocorrer automaticamente ao final da sessão, em qualquer condição de encerramento —
  sucesso, timeout, cancelamento ou permissão negada.
- RF-09: A captura deve ter duas fontes: sinal estruturado extraído dos artefatos que o harness já produz,
  e seção declarada pela sessão. A parte estruturada não pode depender de colaboração do agente.
- RF-10: A captura, a indexação, a recuperação e a compactação não devem realizar chamada de LLM nem
  acesso de rede no caminho default.
- RF-11: A escrita de memória deve ser consolidante: fatos previamente registrados não podem ser removidos
  por uma sessão que não os contradiga explicitamente.
- RF-12: A identidade de um fato deve ser formada por chave semântica declarada em conjunto com hash do
  conteúdo. Regravação de conteúdo idêntico não deve criar duplicata nem nova versão.
- RF-13: A compactação deve ser executada de forma determinística pelo harness quando uma página excede
  seus limites, sem depender de o agente cumprir diretiva textual. Após a sessão, nenhuma página pode
  exceder seus limites.
- RF-14: Fatos que saem de circulação devem ser arquivados, não removidos: permanecem no repositório,
  ficam fora do contexto injetado, e o arquivamento é operação explícita e reversível.
- RF-15: Todo conteúdo deve ser sanitizado antes de ser persistido. Ao detectar segredo, o trecho é
  redigido, o restante do fato é preservado, e a redação é registrada na evidência da sessão. O catálogo
  mínimo obrigatório cobre chaves privadas em formato PEM, tokens de provedor com prefixo reconhecível
  (`ghp_`, `gho_`, `sk-`, `AKIA`), JWT, cabeçalhos de autorização, valores de arquivo `.env` e strings de
  conexão com credencial embutida. O catálogo deve ser extensível por configuração.
- RF-16: Escritas concorrentes de múltiplos processos no mesmo projeto não devem corromper páginas nem
  perder fatos, reutilizando o padrão de lock por plataforma já existente no orquestrador
  (`internal/taskloop/orchestrator_lock_unix.go` e `internal/taskloop/orchestrator_lock_windows.go`) e
  escrita atômica.

### Recuperação

- RF-17: A recuperação deve selecionar conteúdo relevante para a task ativa em vez de injetar páginas
  inteiras.
- RF-18: O contexto de memória injetado deve respeitar um teto global derivado da `WindowClass` ativa,
  subdividido em cotas por camada, com cessão da sobra de uma camada às demais.
- RF-19: O sistema deve oferecer busca local por texto e por entidade sobre a memória, por leitura e
  filtragem determinísticas, sem índice persistido, sem rede e sem LLM.
- RF-20: A recuperação deve manter p95 abaixo de 200 ms para 1.000 páginas de memória, e reportar quando
  o volume degradar essa garantia.
- RF-21: Quando não houver memória alguma, a sessão deve prosseguir normalmente sem falhar e sem emitir
  bloco de memória vazio no prompt.
- RF-22: Página corrompida ou inválida deve ser isolada e reportada, sem abortar a sessão e sem contaminar
  o contexto injetado.

### Handoff, promoção e paridade

- RF-23: O bastão de continuidade deve ser um lease com prazo padrão de 30 minutos, renovado enquanto a
  sessão detentora estiver ativa. Exatamente uma sessão o detém; uma segunda reivindicação é recusada de
  forma explícita. Outra sessão pode reivindicá-lo quando o prazo vence ou quando o processo dono não
  existe mais, e o takeover fica registrado na evidência. O prazo deve ser configurável.
- RF-24: A promoção de um fato das camadas de task ou PRD para a camada de projeto deve ocorrer
  exclusivamente por marcação explícita — seção declarada na sessão ou decisão humana via CLI. Promoção
  automática é proibida.
- RF-25: A memória deve ser neutra em relação à CLI: qualquer uma das quatro CLIs suportadas lê e escreve
  o mesmo conteúdo com a mesma semântica (paridade ADR-008).
- RF-26: A memória deve funcionar em uso ad-hoc, sem PRD ativo, aproveitando a camada de projeto — hoje
  `TasksDir` vazio desliga o subsistema inteiro.
- RF-27: Contradição — dois fatos de mesma chave semântica com conteúdos divergentes — deve ser detectada
  sem LLM e sinalizada tanto na inspeção via CLI quanto no contexto injetado. Ambos os fatos permanecem;
  qual prevalece é decisão de quem cura.

### Operação, compatibilidade e auditoria

- RF-28: A feature deve ser opt-in por flag em `task-loop` e por chave de configuração na cascata de
  precedência do ADR-016 (`flags > workspace > global > defaults`), com zero-value preservando o
  comportamento atual.
- RF-29: Com a feature desativada, o prompt final deve ser byte-idêntico ao produzido pela versão atual
  para os mesmos insumos.
- RF-30: Desativação da feature e qualquer degradação de funcionalidade devem ser registradas de forma
  explícita em log. Desligamento silencioso é proibido.
- RF-31: A CLI deve expor o comando `ai-spec memory` com subcomandos para inspecionar (`show`), buscar
  (`search`), exportar (`export`), compactar (`compact`), migrar (`migrate`) e gerenciar o bastão de
  handoff. Nenhum comando existente pode ter contrato ou output alterados.
- RF-32: A inspeção deve permitir rastrear cada fato até a sessão de origem.
- RF-33: A migração do formato atual deve ocorrer somente por comando explícito, preservar 100% do
  conteúdo existente, produzir backup verificável antes de qualquer conversão, e recusar migração de
  conteúdo já migrado.
- RF-34: O `execution_report.md` deve registrar evidência da operação de memória da sessão: o que foi lido,
  o que foi gravado, orçamento consumido por camada, se houve compactação, se houve redação de segredo, e
  se houve takeover de handoff.
- RF-35: Métricas de memória devem ser expostas na telemetria existente, permanecendo opt-in via
  `GOVERNANCE_TELEMETRY` e append-only (ADR-006).
- RF-36: A leitura e a serialização de uma página devem preservar integralmente o conteúdo que não foi
  gerado pelo harness. Escrita cujo round-trip não reproduza o conteúdo de autoria humana deve ser
  recusada — o que o domínio não gerou, o domínio não reescreve.
- RF-37: Conteúdo de autoria humana sem fato correspondente deve ser preservado, contado no orçamento de
  contexto e sinalizado para compactação humana. Quando ele impedir a camada de voltar aos limites, RF-13
  deve reportar a violação em vez de silenciá-la ou reescrever o conteúdo.

## Experiência do Usuário

A experiência-alvo é a ausência de cerimônia: o valor aparece sem que o usuário mude o que digita.

**Fluxo principal (invisível).** Com a feature ativada na configuração do projeto, o usuário roda
`ai-spec task-loop` como hoje. A memória é lida antes da sessão, injetada dentro do orçamento, e
consolidada ao final. Nenhum comando novo é obrigatório no dia a dia.

**Fluxo de troca de CLI.** O usuário encerra uma sessão em uma CLI e abre outra no mesmo PRD. A segunda
sessão recebe o estado da primeira. Se a primeira ainda detém o bastão, a segunda é informada de forma
explícita em vez de disputar silenciosamente; se a primeira morreu, o bastão é reivindicado e o takeover
aparece na evidência.

**Fluxo de curadoria (explícito).** Ao fim de um trabalho, o dev promove para a camada de projeto os fatos
que valem para sempre. É a única forma de escrever na camada permanente, e é uma decisão consciente.

**Fluxo de auditoria (explícito).** Antes de aprovar uma mudança, o revisor inspeciona a memória: o que
está registrado, de qual sessão veio, o que está sinalizado como contraditório, o que foi redigido por
sanitização.

**Fluxo de escape.** Uma flag desativa a feature inteira e volta ao comportamento atual, sem downgrade de
versão. As mensagens seguem PT-BR, e a desativação é ruidosa em log de propósito — um gate que se desliga
em silêncio é indistinguível de um gate que aprovou (lição de BUG-127, registrada em `AGENTS.md`).

## Restrições Técnicas de Alto Nível

**Não negociáveis de plataforma**

- Go 1.27+, binário único, assets via `go:embed` (ADR-001). Nenhum daemon, nenhum serviço de background,
  nenhum container.
- Sem CGO. O binário é distribuído para Linux, macOS e Windows via GoReleaser; a matriz de CI cobre
  ubuntu-24.04 e macos-15.
- Nenhum acesso de rede obrigatório. O caminho default deve funcionar offline e de forma determinística.
- Nenhuma chamada de LLM no caminho de captura, indexação, recuperação ou compactação.
- Novas dependências diretas exigem justificativa explícita: o projeto mantém hoje sete, e o frontmatter
  de RF-05 é servido pela dependência `yaml.v3` já presente.

**Integração com o existente**

- A hierarquia de configuração vigente é mandatória: `flags CLI > workspace > global > defaults`
  (ADR-016). Toda chave nova entra nessa cascata com zero-value preservando o comportamento atual.
- A extensão deve ocorrer atrás da interface `memory.Store` e dos pontos canônicos do dispatcher de hooks
  já existentes, sem introduzir um segundo mecanismo paralelo de injeção de prompt.
- `WindowClass` (ADR-023) governa os limites; nada pode assumir janela fixa.
- Os shell hooks em `.claude/hooks/*.sh` não podem ser modificados — servem o modo interativo e coexistem
  com os hooks Go do modo orquestrado.
- O servidor MCP interno continua expondo exclusivamente `run_agent`; memória não vira tool MCP.
- O gate `make check-spec-paths` exige que todo caminho citado nos artefatos de contrato deste PRD exista
  no repositório quando o PRD entrar em gestão SDD.

**Privacidade e segurança**

- Nenhum dado de memória deixa a máquina. A telemetria permanece opt-in e append-only (ADR-006).
- Sanitização é pré-condição de persistência, não pós-processamento.
- Escrita de arquivo deve resistir a traversal de caminho: a defesa atual via `filepath.Base` é o piso,
  não o teto, agora que caminhos de projeto entram em jogo.
- A camada de projeto é versionada em git e vai para o remoto: o que é gravado nela é conteúdo público do
  repositório para todos os efeitos práticos, e é por isso que RF-15 é bloqueante.

**Performance**

- Recuperação com p95 abaixo de 200 ms para 1.000 páginas.
- A leitura de memória não pode dominar o tempo de bootstrap; o requisito de bootstrap em menos de 30 s da
  Fundação Portátil continua valendo.

**Qualidade**

- Testes table-driven; `FakeFileSystem` em testes unitários (ADR-002); `t.TempDir()` sob build tag
  `integration`.
- Cobertura mínima de 75% (threshold de CI).
- Idioma PT-BR em comentários, erros e mensagens.

## Fora de Escopo

Explicitamente **não** entram nesta feature:

- **Busca semântica e embeddings.** Exigiria empacotar modelo (~87 MB no `ai-memory`) ou depender de API
  externa; ambos violam o binário único e o determinismo offline. Busca textual e por entidade cobre o
  caso de uso primário. Reavaliável quando existir alternativa pura-Go de custo aceitável.
- **Índice persistido.** Decisão fechada: o volume alvo não justifica o custo de invalidação,
  versionamento e divergência de estado derivado. Entra apenas se medição contra RF-20 provar necessidade.
- **Servidor HTTP, interface web e memória como tool MCP.**
- **Modo multiusuário, autenticação e audit log de time.** Requer servidor compartilhado — incompatível
  com o modelo local e com a auditabilidade file-first da evidência.
- **Sincronização cross-máquina.** Para a camada de projeto, git já é o mecanismo; um segundo seria
  redundante e conflitante.
- **Camada global do usuário (`~/.aispec/memory/`).** Misturar conhecimento de repositórios distintos cria
  risco de vazamento de contexto entre projetos e clientes. Consideração futura, sob opt-in próprio.
- **Interoperabilidade com o `ai-memory`** (bridge, import/export do formato dele, conformidade formal com
  OKF). Os conceitos são adotados; o contrato de dados dele não é implementado.
- **Enriquecimento de memória por LLM.** Fora do caminho default por RF-10; qualquer variante opt-in é
  feature futura com PRD próprio.
- **Promoção automática para a camada de projeto.** Proibida por RF-24.
- **Matriz de 20+ harnesses.** O escopo é a paridade das quatro CLIs que o harness já suporta.
- **Reescrita do subsistema atual.** A evolução é aditiva atrás das interfaces existentes; substituição
  ampla é vetada pela restrição de risco de regressão.
- **Ativação por default.** Nesta feature a memória nova é opt-in. Tornar-se default é decisão de uma
  major futura, com evidência de campo acumulada.
- **Correção do defeito de `check-spec-paths`** (uso de `mapfile`, indisponível no bash 3.2 do macOS, que
  desliga o gate em silêncio). Defeito pré-existente e sem relação com memória: entra por fluxo de bugfix
  dedicado, com teste de regressão executado sob bash 3.2.

## Skills Obrigatórias (carga declarada)

Carga procedural obrigatória para os artefatos e tarefas derivados deste PRD. A declaração é vinculante:
`create-tasks` deve replicá-la em `## Skills Necessárias` de cada task, e `execute-task` deve carregá-la
antes de qualquer alteração de código.

| Skill | Quando carrega | Gate bloqueante |
|---|---|---|
| `agent-governance` | sempre, junto de `AGENTS.md`, antes de qualquer análise ou edição | contrato de carga base do `AGENTS.md` |
| `domain-modeling-production` | já executada nesta fase | `validate-bundle.py` retornou `SUCCESS` no bundle `discoveries/domain-memoria-duravel-de-agentes/` |
| `create-technical-specification` | próximo artefato, antes de qualquer código | techspec registra spec-hash deste PRD |
| `design-patterns-mandatory` | na techspec e em toda tarefa que introduza ou altere abstração | `validate_pattern_bundle.py` deve validar o bundle de padrões; padrão sem evidência técnica é rejeitado |
| `go-implementation` | em toda tarefa que produza ou altere código Go | Regras Estritas R0–R7 são `[HARD]` bloqueantes de merge; `verify-go-mod.sh` na Etapa 1 |
| `object-calisthenics-go` | em revisão e refatoração de Go | heurística de revisão; cede a `go-implementation` em conflito (R-GOV-001) |
| `review` | antes de fechar cada tarefa | `validate-review-evidence.sh` |

Consequências de `go-implementation` que este PRD já assume como restrição, e que a techspec não pode
contrariar:

- `init()` proibida (R0) — nenhuma inicialização implícita de memória ou de catálogo de sanitização.
- Toda função deve ser método de struct, com exceções exaustivas (R1) — as políticas do modelo
  (relevância, orçamento, lease, sanitização, compactação) são domain services stateless, seguindo o
  padrão já existente em `internal/runtime/memory/window_policy.go`.
- Mocks exclusivamente via `mockery.yml` (R3) — o contrato estendido de `memory.Store` exige regeneração.
- Testes em `testify/suite` table-driven (R4) — inclui o teste de round-trip de RF-36 e o de paridade
  byte-a-byte de RF-29.
- Interface definida no pacote consumidor e injeção por construtor, sem estado global (R6).
- Recursos modernos da versão declarada em `go.mod` (R7).

Restrição de escopo desta fase: `design-patterns-mandatory` e `go-implementation` **não** foram executadas
aqui, porque não existe especificação técnica nem código Go a produzir neste artefato. Executá-las agora
geraria pattern-bundle e implementação sem contrato — exatamente o que o Protocolo PRD-First proíbe.

## Decisões Registradas

Todas as ambiguidades materiais identificadas na análise foram fechadas. Rastreabilidade decisão → requisito:

| # | Decisão | Alternativa rejeitada e por quê | RF |
|---|---|---|---|
| D-01 | Evolução nativa em Go, adotando os conceitos do ai-memory | Daemon externo via MCP/HTTP: quebra binário único, torna a evidência dependente de estado externo mutável | todos |
| D-02 | Três camadas: projeto, PRD, task | Camada global do usuário: risco de vazamento entre repositórios e clientes | RF-01 |
| D-03 | Camada de projeto em `.aispec/memory/` | `.agents/memory/`: território do instalador, memória viraria drift em `verify`. `.specs/memory/`: entraria no escopo dos gates de contrato SDD | RF-02 |
| D-04 | Comando novo `ai-spec memory` | Estender `inspect`/`doctor`: alteraria output que scripts e gates já consomem | RF-31 |
| D-05 | Segredo detectado: redigir trecho, preservar fato, registrar | Descartar fato: perda silenciosa. Abortar sessão: um falso positivo custa a memória inteira e incentiva desligar a sanitização | RF-15 |
| D-06 | Arquivamento append-only, sem poda automática | TTL por idade: apaga por tempo, não por validade. LRU: descarta o fato raro, que costuma ser o crítico | RF-14 |
| D-07 | Handoff por lease com TTL de 30 min + verificação de processo vivo | Só PID: reciclagem de PID e falha em container. Só TTL: espera desnecessária após crash conhecido. Só manual: crash noturno bloqueia o fluxo | RF-23 |
| D-08 | Teto global por `WindowClass` com cotas por camada e cessão de sobra | Teto único com prioridade: camada verbosa zera as demais em silêncio | RF-18 |
| D-09 | Scan determinístico, sem índice persistido | Índice próprio, SQLite puro-Go ou Bleve: adiciona staleness, invalidação e versionamento de formato sem necessidade medida no volume alvo | RF-19, RF-20 |
| D-10 | Promoção para a camada de projeto só por marcação explícita | Recorrência ou categoria: promovem ruído com a mesma facilidade que conhecimento | RF-24 |
| D-11 | Opt-in por flag + chave na cascata ADR-016 | Default-on: maximiza exposição a regressão. Só env var: fora da cascata e não auditável. Só config: sem escape por execução | RF-28, RF-29 |
| D-12 | Migração por comando explícito, com backup verificável | Automática na primeira execução: altera estado dentro de uma sessão e acopla sucesso da tarefa ao sucesso da migração | RF-33 |
| D-13 | Captura de fonte dupla: sinal estruturado + seção declarada | Só estruturado: perde diagnóstico e decisão. Só declarado: depende de colaboração, o modo de falha atual. Resumo por LLM: custo por sessão e não determinístico | RF-09 |
| D-14 | Defeito do `check-spec-paths` tratado fora deste PRD | Incluir como RF: mistura bugfix de gate de CI com feature de memória e polui rastreabilidade | Fora de escopo |

As 20 decisões de modelagem de domínio (M-01 a M-20) estão registradas em
[`discoveries/domain-memoria-duravel-de-agentes/transcript.md`](../../discoveries/domain-memoria-duravel-de-agentes/transcript.md).
Quatro delas alteraram requisitos deste PRD: M-17 (durabilidade em três níveis) refinou RF-07; M-18
(identidade por chave semântica mais hash) refinou RF-12 e RF-27; M-19 (round-trip lossless) originou
RF-36; M-20 (conteúdo de autoria humana) originou RF-37. As demais são internas ao modelo e não criam
requisito de produto.

## Premissas Verificadas

Fatos confirmados no repositório, não suposições:

1. `MemoryPersistHook` grava em `memory.WriteModeReplace` — comportamento destrutivo confirmado em
   `internal/runtime/hooks/memory_persist.go`.
2. `Store.WriteTask` não tem chamador em produção — confirmado por varredura de todo o código não-teste.
3. A compactação hoje é uma diretiva textual anexada ao prompt em `internal/runtime/runner.go`, sem
   qualquer garantia de execução.
4. `.specs/` não está em `.gitignore`, portanto a memória de PRD já é versionada em git. A camada de
   projeto herda a mesma premissa.
5. `.aispec/` já é marcador reconhecido de raiz de projeto na hierarquia de config (ADR-016).
6. `yaml.v3` já é dependência direta, cobrindo o frontmatter de RF-05 sem dependência nova.
7. O padrão de lock por plataforma existe em `internal/taskloop/orchestrator_lock_unix.go` e
   `internal/taskloop/orchestrator_lock_windows.go`, e é reutilizável para RF-16.
8. O `ai-memory` está sob licença MIT e sua arquitetura é documentada publicamente. Nenhum código dele é
   copiado, portado ou vendorado, e nenhuma obrigação de licença é assumida.
9. O modelo de domínio foi confrontado com o código de produção: 9 achados `confirmado` com evidência
   `path:linha` e 4 `ausente`. Nenhum achado foi elevado a `confirmado` com base em teste, mock, fixture
   ou documentação.

## Riscos Endereçados

Cada risco tem mitigação decidida. Nenhum fica pendente de decisão.

| Risco | Mitigação decidida |
|---|---|
| Captura sem LLM produz memória mais pobre que resumo gerado por modelo | Fonte dupla (RF-09): a parte estruturada garante conteúdo correto; a seção declarada enriquece quando o agente colabora. Aposta explícita: fato derivado de sinal real (gate, diff, falha) vale mais em produção que prosa gerada, e custa zero |
| Consolidação determinística não "entende" o conteúdo e pode manter fato obsoleto | Assimetria deliberada: RF-11 nunca apaga fato não contradito, e RF-27 sinaliza contradição. Manter fato obsoleto sinalizado é preferível a perder fato válido em silêncio — a perda silenciosa é o defeito que esta feature corrige |
| Memória de projeto vai para o git remoto e pode conter segredo | RF-15 é bloqueante e pré-condição de escrita, com catálogo mínimo obrigatório e redação registrada na evidência |
| Scan sem índice degrada em volumes muito acima do alvo | RF-20 fixa o alvo e exige que a degradação seja reportada, não silenciosa; índice persistido fica como otimização condicionada a medição |
| Camada de projeto vira lixão de detalhe efêmero | RF-24 proíbe promoção automática: só marcação explícita escreve na camada permanente |
| Feature nova introduz regressão em produção | RF-28 e RF-29 garantem opt-in com prompt byte-idêntico quando desativada; O-2 exige teste de paridade dedicado além da suíte atual |
| Degradação de funcionalidade passa despercebida | RF-30 proíbe desligamento silencioso; RF-34 exige evidência da operação de memória em cada `execution_report.md` |
