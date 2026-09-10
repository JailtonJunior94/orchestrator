# Tarefa 9.0: Hooks e paridade comprovada nos quatro agentes

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Hoje **nenhum teste do repositório prova disparo de hook**. Todos verificam apenas que o arquivo foi
escrito: `internal/install/install_test.go:2410` e `:2439` montam um filesystem em memória e aplicam
`strings.Contains` sobre o conteúdo gerado. Teste de escrita não é teste de disparo — e foi exatamente
essa diferença que permitiu a uma chave de evento inválida sobreviver: `internal/install/install.go:1275-1302`
registra o hook de encerramento do Copilot sob a chave `stop`, que não existe no contrato do CLI; a
chave oficial é `agentStop`. O gate de encerramento do Copilot **nunca rodou**, e a asserção em
`internal/install/install_test.go:446` — que hoje falha se a string `"agentStop"` aparecer no JSON —
protege ativamente o comportamento errado.

Esta tarefa fecha o buraco em duas frentes: corrige o defeito e substitui a matriz de paridade
declarativa por três camadas de prova, com honestidade explícita sobre o que cada camada cobre. Além
disso, promove pré-condição de enforcement a conceito verificável, para que um gate que existe porém
não pode disparar deixe de ser contabilizado como ativo.

**Corrigir a chave de evento é ATIVAR um gate que nunca rodou, não restaurar um existente.** Espere
falhas novas em fluxos do Copilot que antes encerravam sem passar por gate nenhum; elas são o sinal
esperado, não regressão.

**Restrição de ordem — crítica.** Esta tarefa precisa estar completa e **verde com o Gemini ainda
presente** no repositório. Os gates de paridade são escritos com as células ainda completas, para que
fiquem vermelhos exatamente se a remoção da tarefa 10.0 degradar a cobertura. Inverter a ordem elimina
o único sinal disponível.

<requirements>
- RF-24: a verificação detecta se os hooks do Codex têm trust persistido, usando o canal read-only
  `hooks/list` do `codex app-server` (RPC, sem conceder). Sem trust, estado reportado com a instrução
  exata de concessão. O harness **nunca** concede o trust pelo usuário e **nunca** usa a flag de bypass.
- RF-25: a verificação detecta se o diretório do projeto está na lista de pastas confiáveis do Copilot,
  lendo o arquivo de configuração. Fora dela, os hooks de projeto não disparam e o estado é reportado
  como não-ativo.
- RF-27: gate canônico de **encerramento** que bloqueia o fim da sessão quando existir tarefa ativa sem
  veredito `APPROVED` registrado, presente nos quatro agentes — cobrindo também o **uso interativo**,
  onde o orquestrador não está no caminho.
- RF-28: a matriz de paridade é verificada por gate de build que **falha** quando um agente deixa de
  cobrir um ponto canônico, aponta para script divergente, **ou não possui teste de disparo real
  associado**.
- RF-29: o espelhamento dos scripts canônicos permanece verificado pelo gate de sincronização, agora
  cobrindo o gate de encerramento e o plugin do OpenCode; as contagens fixas de espelhos nos scripts de
  verificação são atualizadas.
- RF-43: o comportamento é idêntico nos quatro agentes — mesmos vereditos, motivos, estrutura de
  evidência e teto default. Nenhum agente pode ter cobertura menor que outro.
- RF-59: a chave de evento inválida do Copilot é corrigida para o nome oficial, e a asserção de teste
  invertida é corrigida junto.
- Dois estados novos na verificação: `inert` (existe e está atualizado, mas a pré-condição não é
  satisfeita — **conta como FALHA**) e `unknown` (ausência de informação — **NÃO é sucesso**).
- A regra de segurança operacional permanece intacta: **detecção não executa binários**. A verificação
  que exige RPC pertence ao comando de diagnóstico, invocado explicitamente.
- A camada 3 (disparo verdadeiro pelo CLI) **não é gate de merge** — isso precisa estar dito
  explicitamente no código, no Makefile e no workflow.
- Consolidação de estado aplica o **pior estado**: um único hook não confiado invalida o conjunto.
</requirements>

## Subtarefas

- [ ] 9.1 Corrigir a chave `stop` → `agentStop` no gerador de hooks do Copilot em
      `internal/install/install.go:1275-1302`, mantendo o apontamento para os mesmos scripts canônicos.
- [ ] 9.2 Corrigir a asserção invertida em `internal/install/install_test.go:446`, que hoje reprova o
      JSON quando a chave oficial aparece; a asserção passa a **exigir** `agentStop` e a reprovar `stop`.
- [ ] 9.3 Criar o gate canônico de encerramento como script tool-neutro em `.agents/scripts/` (planejado),
      que bloqueia o fim da sessão com tarefa ativa sem veredito `APPROVED` registrado.
- [ ] 9.4 Registrar o gate de encerramento nos **quatro** agentes pelo mecanismo nativo de cada um,
      cobrindo também o caminho interativo (não só o orquestrado).
- [ ] 9.5 Criar o pacote de verificadores de pré-condição (planejado), um por mecanismo, com o remédio
      acionável embutido no tipo.
- [ ] 9.6 Verificador de trust do Codex por RPC read-only `hooks/list` do `codex app-server`, com timeout
      e tratamento de falha próprios; sem RPC solicitado, o estado é `unknown`.
- [ ] 9.7 Verificador de pasta confiável do Copilot por leitura do arquivo de configuração, que **não é
      JSON estrito**: remover comentários de linha **fora de strings**, sem expressão regular — regex
      sobre o marcador `//` engoliria endereços dentro de valores.
- [ ] 9.8 Introduzir os estados `inert` e `unknown` na saída de `verify`, com `inert` contando como falha
      e `unknown` impresso como ausência de informação; consolidação pelo pior estado.
- [ ] 9.9 Camada 1 — matriz obrigatória unitária: produto cartesiano agentes × pontos canônicos,
      afirmando que o arquivo de registro escrito contém a chave nativa declarada **e** invoca o
      validador declarado. É a camada que teria pego a chave inválida do Copilot.
- [ ] 9.10 Camada 2 — disparo simulado em integração: executar o script instalado com o contrato de
      entrada documentado de cada CLI e afirmar bloqueio real (entrada que viola governança produz saída
      não-zero). Não exige CLI instalado.
- [ ] 9.11 Camada 3 — disparo verdadeiro pelo CLI em job noturno com build tag dedicada, espelhando o
      padrão `acp_live` que o repositório já tem: alvo `test-acp-live` em `Makefile:106-109`, nota de
      escopo em `.github/workflows/test.yml:150-151` e workflow `.github/workflows/acp-live.yml`.
- [ ] 9.12 Fazer o job noturno **falhar se qualquer célula for pulada**, para não virar teste vazio; e
      declarar explicitamente, no alvo e no workflow, que a camada 3 não é gate de merge.
- [ ] 9.13 Gate de build de paridade que reprova agente sem cobertura de ponto canônico, com validador
      divergente, ou sem teste de disparo associado — escrito com as células ainda completas.
- [ ] 9.14 Atualizar as contagens fixas de espelhos: array `mirrors` em `scripts/check-skills-sync.sh:21-25`,
      array `tool_hook_mirrors` em `scripts/check-skills-sync.sh:122-131` e array `mirror_dirs` em
      `scripts/check-hooks-sync.sh:24-33`, de modo a cobrir o gate de encerramento e o plugin do OpenCode.

## Detalhes de Implementação

Ver techspec.md, seções:

- **"Verificação de pré-condições sem violar a regra de detecção"** — a tabela que define, por
  pré-condição, se é verificável sem executar binário e qual estado `verify` reporta por default e com
  opt-in explícito; e a nota sobre o arquivo de configuração do Copilot não ser JSON estrito.
- **"Catálogo de Agentes: um registro, tudo derivado"** — os tipos `PontoCanonico` e
  `PreCondicaoDeEnforcement`, este último com o remédio acionável embutido, que é o que permite
  distinguir "gate ausente" de "gate presente porém inerte".
- **"Testes de Integração"** — as três camadas de prova de enforcement e a honestidade explícita sobre
  o que cada uma cobre.
- **"Fases"**, linha **F5 — Hooks e paridade**, e a nota de que F5 precede F4.

Ver também `adr-005-precondicoes-de-enforcement.md`, seções "Decisão", "Riscos e Mitigações" e
"Plano de Implementação".

## Critérios de Sucesso

- `go test ./internal/install/... -run TestInstall -count=1` passa **e** a busca por `"stop"` como chave
  de evento no JSON do Copilot não retorna ocorrência; `"agentStop"` retorna exatamente uma.
- `grep -rn '"agentStop"' internal/install/install.go` retorna a chave no bloco do Copilot, e o teste em
  `internal/install/install_test.go` reprova o JSON quando ela **falta** (asserção invertida corrigida —
  verificável mutando a string e observando o teste ficar vermelho).
- A matriz obrigatória unitária cobre 4 agentes × 3 pontos canônicos = **12 células**, todas presentes;
  remover qualquer célula do registro deixa `go test ./... -count=1` vermelho.
- O gate de build de paridade fica **vermelho** quando: um agente perde um ponto canônico; uma célula
  aponta para validador divergente; ou uma célula não tem teste de disparo associado. As três condições
  são provadas por teste negativo dedicado.
- `go test -tags=integration ./tests/integration/...` executa o script de encerramento instalado com o
  contrato de entrada de cada um dos 4 CLIs e obtém **saída não-zero** para entrada que viola governança.
  Nenhum CLI precisa estar instalado.
- O alvo de nightly da camada 3 existe no `Makefile` seguindo o padrão de `test-acp-live` (`Makefile:106-109`)
  e o workflow correspondente declara, em comentário e em nome de job, que **não é gate de merge**.
- O job noturno **falha** se qualquer célula da matriz for pulada — provado por execução com uma célula
  marcada como skip.
- `ai-spec-harness verify .` imprime `inert` para pré-condição não satisfeita e `unknown` para
  pré-condição não avaliada; o código de saída é **não-zero** quando houver ao menos um `inert`, e o
  `unknown` **não** é contabilizado como sucesso no sumário.
- A detecção continua sem executar binário: `grep -rn "exec.Command\|exec.CommandContext" internal/detect/`
  não retorna ocorrência nova; o RPC do Codex vive apenas na superfície de diagnóstico.
- Nenhuma ocorrência da flag de bypass de trust do Codex no repositório.
- `bash scripts/check-skills-sync.sh` e `bash scripts/check-hooks-sync.sh` saem 0 com o gate de
  encerramento e o plugin do OpenCode incluídos nos espelhos.
- **Com o Gemini ainda presente**, `go build ./... && go vet ./... && go test ./... -count=1` mais os
  gates de sincronia estão todos verdes.

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
- `internal/install/install.go:1275-1302` — gerador de hooks do Copilot; chave `stop` inválida a corrigir
- `internal/install/install_test.go:446` — asserção invertida que protege o comportamento errado
- `internal/install/install_test.go:2410`, `:2439` — testes que só provam escrita de arquivo; molde a
  superar, não a replicar
- `.agents/scripts/` — validadores canônicos tool-neutros; destino do gate de encerramento
- `.agents/scripts/validate-review-evidence.sh` — validador de artefato de revisão, referência de estilo
- `.agents/scripts/hook-prereq-gate.sh` — gate de pré-requisito existente, referência de contrato
- `internal/runtime/specs/` — registro de agentes e tipos de enforcement (planejado)
- `internal/runtime/precondicoes/` (planejado) — verificadores de pré-condição, um por mecanismo
- `cmd/ai_spec_harness/verify.go` — saída de `verify`; recebe os estados `inert` e `unknown`
- `Makefile:106-109` — alvo `test-acp-live`, padrão a espelhar para o nightly da camada 3
- `.github/workflows/acp-live.yml` — workflow noturno existente, padrão a espelhar
- `.github/workflows/test.yml:150-151` — nota de escopo que separa gate de merge de nightly
- `scripts/check-skills-sync.sh:21-25`, `:122-131` — arrays de espelhos com contagem fixa
- `scripts/check-hooks-sync.sh:24-33` — array `mirror_dirs` com contagem fixa
- `tests/integration/` — destino da camada 2 (disparo simulado)
- `adr-005-precondicoes-de-enforcement.md` — decisão que esta tarefa implementa, apenas leitura
