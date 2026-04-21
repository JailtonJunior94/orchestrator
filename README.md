# ai-spec-harness

CLI em Go para instalar, validar, inspecionar e atualizar governanca operacional para ferramentas de IA em repositorios de software.

No repositorio e no modulo Go o nome e `ai-spec-harness`. No uso diario, este README prioriza o binario de release `ai-spec`.

O projeto padroniza como Claude, Gemini, Codex e GitHub Copilot encontram skills, agentes, comandos e contexto de execucao dentro de um repositorio alvo. O foco nao e "conversar com um modelo", e sim tornar fluxos repetiveis como PRD, especificacao tecnica, decomposicao de tasks, review e execucao de tarefas mais previsiveis e auditaveis.

## Sumario

- [Visao geral](#visao-geral)
  - [O que este projeto resolve](#o-que-este-projeto-resolve)
  - [Como funciona](#como-funciona)
- [Instalacao](#instalacao)
  - [Requisitos](#requisitos)
  - [Opcao recomendada: macOS com Homebrew](#opcao-recomendada-macos-com-homebrew)
  - [Completion para bash e zsh](#completion-para-bash-e-zsh)
  - [Download direto](#download-direto)
  - [Instalacao via Go](#instalacao-via-go)
  - [Executar sem instalar](#executar-sem-instalar)
- [Inicio rapido](#inicio-rapido)
- [O que fazer depois da instalacao](#o-que-fazer-depois-da-instalacao)
  - [Cenario 1: projeto existente que voce quer mapear](#cenario-1-projeto-existente-que-voce-quer-mapear)
  - [Cenario 2: projeto novo que ainda so tem `.git`](#cenario-2-projeto-novo-que-ainda-so-tem-git)
- [Fluxo completo recomendado](#fluxo-completo-recomendado)
  - [1. Instalar a governanca no projeto alvo](#1-instalar-a-governanca-no-projeto-alvo)
  - [2. Validar a instalacao](#2-validar-a-instalacao)
  - [3. Fazer upgrade quando houver nova versao de governanca](#3-fazer-upgrade-quando-houver-nova-versao-de-governanca)
  - [4. Entrar no repositorio instrumentado](#4-entrar-no-repositorio-instrumentado)
  - [5. Criar o PRD](#5-criar-o-prd)
  - [6. Criar a tech spec com foco em DDD e arquitetura](#6-criar-a-tech-spec-com-foco-em-ddd-e-arquitetura)
  - [7. Gerar o bundle de tasks](#7-gerar-o-bundle-de-tasks)
  - [8. Executar todas as tasks com o looper do CLI](#8-executar-todas-as-tasks-com-o-looper-do-cli)
  - [9. Validar o estado final](#9-validar-o-estado-final)
- [Como obter o melhor das skills — uso mandatorio e rigoroso](#como-obter-o-melhor-das-skills--uso-mandatorio-e-rigoroso)
  - [Principio central](#principio-central)
  - [Pipeline mandatorio](#pipeline-mandatorio)
  - [`create-prd` — escopo e requisitos](#create-prd--escopo-e-requisitos)
  - [`create-technical-specification` — arquitetura e contratos](#create-technical-specification--arquitetura-e-contratos)
  - [`create-tasks` — decomposicao em unidades executaveis](#create-tasks--decomposicao-em-unidades-executaveis)
  - [`execute-task` — implementacao com evidencia](#execute-task--implementacao-com-evidencia)
  - [Como executar tasks com fidelidade maxima — ciclo por task com limpeza de contexto](#como-executar-tasks-com-fidelidade-maxima--ciclo-por-task-com-limpeza-de-contexto)
  - [`review` — revisao antes de merge](#review--revisao-antes-de-merge)
  - [`refactor` — melhoria sem mudanca de comportamento](#refactor--melhoria-sem-mudanca-de-comportamento)
  - [Checklist de entrada por skill](#checklist-de-entrada-por-skill)
  - [Sinais de desvio — quando parar e corrigir](#sinais-de-desvio--quando-parar-e-corrigir)
- [SDD + harness para features Go](#sdd--harness-para-features-go)
  - [Exemplo 1: feature Go mais complexa com DDD, goroutines e performance](#exemplo-1-feature-go-mais-complexa-com-ddd-goroutines-e-performance)
  - [Exemplo 2: feature Go simples de API de listagem](#exemplo-2-feature-go-simples-de-api-de-listagem)
  - [Quando usar cada estilo](#quando-usar-cada-estilo)
  - [Wrapper para iniciar mais rapido](#wrapper-para-iniciar-mais-rapido)
- [Task-loop de forma eficiente](#task-loop-de-forma-eficiente)
  - [Fluxo recomendado](#fluxo-recomendado)
  - [Comandos recomendados](#comandos-recomendados)
  - [Quando cada flag ajuda](#quando-cada-flag-ajuda)
  - [Heuristicas praticas](#heuristicas-praticas)
- [Executar uma task ou varias tasks](#executar-uma-task-ou-varias-tasks)
  - [Sem task-loop: executar uma task especifica](#sem-task-loop-executar-uma-task-especifica)
  - [Com task-loop: executar todas as tasks elegiveis](#com-task-loop-executar-todas-as-tasks-elegiveis)
  - [Quando usar cada abordagem](#quando-usar-cada-abordagem)
- [Referencia rapida de comandos](#referencia-rapida-de-comandos)
  - [Exemplos uteis por comando](#exemplos-uteis-por-comando)
- [Exemplos por ferramenta](#exemplos-por-ferramenta)
  - [Codex](#codex)
  - [Claude](#claude)
  - [Gemini](#gemini)
  - [GitHub Copilot](#github-copilot)
- [Operacao da instalacao](#operacao-da-instalacao)
  - [Estrategias de instalacao](#estrategias-de-instalacao)
- [Para quem mantem este repositorio](#para-quem-mantem-este-repositorio)
  - [Desenvolvimento local](#desenvolvimento-local)
  - [Contribuicao](#contribuicao)
  - [Roadmap curto](#roadmap-curto)
- [Referencias](#referencias)
  - [Releases](#releases)
  - [Documentacao](#documentacao)

## Visao geral

### O que este projeto resolve

Sem uma estrutura canonica, cada repositorio tende a ter prompts soltos, instrucoes duplicadas, skills divergentes e pouca rastreabilidade sobre como agentes devem operar. O `ai-spec-harness` resolve isso ao:

- instalar um baseline de governanca em um projeto alvo
- distribuir skills compartilhadas por `symlink` ou copia
- gerar adaptadores por ferramenta para Claude, Gemini, Codex e Copilot
- validar `SKILL.md`, schema de bugs e artefatos de governanca
- inspecionar e diagnosticar instalacoes existentes
- medir custo estimado de contexto por baseline e fluxo
- criar scaffold para novas skills de linguagem

### Como funciona

O CLI usa este repositorio como fonte de governanca e instala os artefatos necessarios no projeto alvo. Dependendo das ferramentas selecionadas, ele cria estruturas como:

```text
.agents/skills/
.claude/agents/
.gemini/commands/
.github/agents/
.github/copilot-instructions.md
.codex/config.toml
.ai_spec_harness.json
```

Depois da instalacao, o repositorio alvo passa a expor skills e agentes processuais para fluxos como:

- `analyze-project`
- `create-prd`
- `create-technical-specification`
- `create-tasks`
- `execute-task`
- `review`
- `bugfix`
- `refactor`

## Instalacao

### Requisitos

- Go `1.26.2` ou compativel com o [`go.mod`](./go.mod)
- `git` disponivel no `PATH`
- permissao de escrita no projeto alvo
- um repositorio fonte de governanca contendo `.agents/skills`

### Opcao recomendada: macOS com Homebrew

O caminho principal para uso local e instalar o binario de release via Homebrew. O executavel publicado por release se chama `ai-spec`.

```bash
brew install jailtonjunior94/tap/ai-spec
ai-spec version
```

> **Aviso de seguranca do macOS (Gatekeeper)**
>
> O macOS pode exibir o alerta _"Apple could not verify 'ai-spec' is free of malware"_ ao executar o binario pela primeira vez. Isso ocorre porque o binario nao esta assinado com um Apple Developer ID. Ha quatro formas de resolver:
>
> **Opcao 1 — Terminal (recomendada):**
>
> ```bash
> xattr -dr com.apple.quarantine $(which ai-spec)
> ```
>
> **Opcao 2 — Interface grafica:** abra o Finder, navegue ate o binario, clique com o botao direito e selecione **Abrir**. Na janela de aviso, clique em **Abrir assim mesmo**.
>
> **Opcao 3 — Configuracoes do sistema:** va em **Ajustes do Sistema -> Privacidade e Seguranca**, role ate a secao **Seguranca** e clique em **Abrir assim mesmo** ao lado da mensagem sobre `ai-spec`.
>
> **Opcao 4 — `spctl`:**
>
> ```bash
> sudo spctl --add --label "ai-spec" $(which ai-spec)
> ```
>
> Versoes futuras instaladas via `brew upgrade ai-spec` executam o `xattr` automaticamente no `post_install` da Formula, eliminando o alerta para novos usuarios.

Se o seu shell nao estiver herdando o `PATH` do Homebrew corretamente, adicione o prefixo do Homebrew ao arquivo de inicializacao e mantenha um alias compativel com o nome do modulo Go:

`~/.zshrc`

```bash
export PATH="$(brew --prefix)/bin:$PATH"
alias ai-spec-harness="ai-spec"
```

`~/.bashrc`

```bash
export PATH="$(brew --prefix)/bin:$PATH"
alias ai-spec-harness="ai-spec"
```

Depois recarregue o shell:

```bash
source ~/.zshrc
# ou
source ~/.bashrc
```

Observacao importante:

- exemplos de release e do README usam `ai-spec`
- a instalacao via `go install github.com/JailtonJunior94/ai-spec-harness@latest` gera o executavel `ai-spec-harness`
- o alias acima evita alternar mentalmente entre os dois nomes

### Completion para bash e zsh

O comando `completion` gera scripts de autocompletion para o nome real do binario. Se voce estiver usando alias `ai-spec -> ai-spec-harness` ou o inverso, mantenha o alias no shell e gere o completion para o executavel que realmente esta instalado.

#### Bash

Sessao atual:

```bash
source <(ai-spec completion bash)
```

Persistente no macOS com Homebrew:

```bash
ai-spec completion bash > "$(brew --prefix)/etc/bash_completion.d/ai-spec"
```

Se voce preferir declarar isso no `~/.bashrc`, use:

```bash
if command -v ai-spec >/dev/null 2>&1; then
  source <(ai-spec completion bash)
fi
```

#### Zsh

Se o shell ainda nao tiver `compinit` habilitado, adicione ao `~/.zshrc`:

```bash
autoload -U compinit
compinit
```

Sessao atual:

```bash
source <(ai-spec completion zsh)
```

Persistente no macOS com Homebrew:

```bash
ai-spec completion zsh > "$(brew --prefix)/share/zsh/site-functions/_ai-spec-harness"
```

Se voce preferir inicializar via `~/.zshrc`, use:

```bash
if command -v ai-spec >/dev/null 2>&1; then
  source <(ai-spec completion zsh)
fi
```

### Download direto

#### macOS

Apple Silicon:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_darwin_arm64.tar.gz
tar -xzf ai-spec_<VERSION>_darwin_arm64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

Intel:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_darwin_amd64.tar.gz
tar -xzf ai-spec_<VERSION>_darwin_amd64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

#### Linux

`amd64`:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_linux_amd64.tar.gz
tar -xzf ai-spec_<VERSION>_linux_amd64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

`arm64`:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_linux_arm64.tar.gz
tar -xzf ai-spec_<VERSION>_linux_arm64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

#### Windows

PowerShell:

```powershell
$version = "<VERSION>"
$url = "https://github.com/JailtonJunior94/orchestrator/releases/download/v$version/ai-spec_${version}_windows_amd64.zip"
Invoke-WebRequest -Uri $url -OutFile "ai-spec.zip"
Expand-Archive -Path ".\\ai-spec.zip" -DestinationPath ".\\ai-spec"
Move-Item ".\\ai-spec\\ai-spec.exe" "$env:USERPROFILE\\bin\\ai-spec.exe"
$env:Path += ";$env:USERPROFILE\\bin"
ai-spec.exe version
```

### Instalacao via Go

Se voce prefere instalar a partir do modulo:

```bash
go install github.com/JailtonJunior94/ai-spec-harness@latest
ai-spec-harness version
```

Durante desenvolvimento local neste checkout:

```bash
go install .
ai-spec-harness version
```

Se quiser padronizar a experiencia local com o mesmo nome do binario publicado via release, adicione um alias ao shell:

`~/.zshrc`

```bash
alias ai-spec="ai-spec-harness"
```

`~/.bashrc`

```bash
alias ai-spec="ai-spec-harness"
```

### Executar sem instalar

```bash
go run . --help
```

## Inicio rapido

Exemplo usando este repositorio como fonte de governanca e um servico Go como destino:

```bash
ai-spec install ../api-pagamentos \
  --source . \
  --tools claude,gemini,codex,copilot \
  --langs go
```

Depois valide o estado da instalacao:

```bash
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
```

## O que fazer depois da instalacao

O melhor primeiro passo depende do estado do repositorio alvo.

### Cenario 1: projeto existente que voce quer mapear

Se o repositorio ja tem codigo, a melhor entrada costuma ser instalar a governanca, validar a base e pedir ao agente uma leitura arquitetural inicial com `analyze-project`.

1. Instale a governanca no projeto alvo:

```bash
ai-spec install ../api-legado \
  --source . \
  --tools codex,claude,gemini,copilot \
  --langs go
```

2. Valide a instalacao e entre no repositorio:

```bash
ai-spec inspect ../api-legado
ai-spec doctor ../api-legado
cd ../api-legado
```

3. Se estiver usando Codex, Gemini ou Copilot, gere uma instrucao pronta com o wrapper:

```bash
ai-spec wrapper codex analyze-project .
```

4. No agente, use um pedido objetivo como este:

```text
Use a skill analyze-project para analisar a arquitetura atual deste repositorio.

Quero no resultado:
- classificacao do tipo de projeto (monolito, monolito modular, monorepo ou microservico)
- evidencias usadas na classificacao
- stack detectada
- padrao arquitetural predominante
- mapa das pastas mais importantes
- fluxo de dependencias entre camadas ou modulos
- recomendacoes de governanca para este contexto
```

Esse fluxo e o melhor quando ja existe base de codigo porque `analyze-project` foi desenhada para ler estrutura real, detectar stack e adaptar a governanca ao contexto encontrado.

### Cenario 2: projeto novo que ainda so tem `.git`

Se o repositorio ainda esta vazio, nao comeca por `analyze-project`. Nesse caso ainda nao existe arquitetura real para classificar, entao o caminho mais util e instalar a governanca e iniciar pelo escopo do produto.

Exemplo de um repositorio novo:

```bash
mkdir novo-produto
cd novo-produto
git init
```

Supondo que `novo-produto` esteja ao lado deste checkout do `ai-spec-harness`, volte ao repositorio fonte de governanca e instale a base:

```bash
cd ../ai-spec-harness
ai-spec install ../novo-produto \
  --source . \
  --tools codex,claude,gemini,copilot \
  --langs go
```

Depois entre no projeto e comece por `create-prd`:

```bash
cd ../novo-produto
ai-spec inspect .
ai-spec wrapper codex create-prd .
```

Prompt inicial sugerido:

```text
Use a skill create-prd para definir o primeiro escopo deste projeto novo.

Contexto:
- repositorio recem-criado
- ainda nao existe arquitetura implementada
- queremos uma base pequena e evolutiva

Quero no resultado:
- problema
- objetivos
- nao objetivos
- requisitos funcionais
- requisitos nao funcionais
- riscos iniciais
```

Depois do PRD aprovado, o fluxo natural costuma ser:

```text
create-technical-specification -> create-tasks -> execute-task
```

## Fluxo completo recomendado

O `ai-spec-harness` nao escreve PRD, tech spec ou codigo por conta propria. Ele instala a governanca e os adaptadores para que o agente escolhido execute cada etapa com as skills corretas dentro do repositorio alvo.

### 1. Instalar a governanca no projeto alvo

```bash
ai-spec install ../api-pagamentos \
  --source . \
  --tools codex,claude,gemini,copilot \
  --langs go
```

### 2. Validar a instalacao

```bash
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
```

### 3. Fazer upgrade quando houver nova versao de governanca

```bash
ai-spec upgrade ../api-pagamentos --source . --check
ai-spec upgrade ../api-pagamentos --source . --langs go
```

### 4. Entrar no repositorio instrumentado

```bash
cd ../api-pagamentos
```

### 5. Criar o PRD

Exemplo de prompt para o agente:

```text
Use a skill create-prd para criar um PRD de listagem de pagamentos.

Contexto:
- precisamos expor GET /payments
- filtros: status, pagina, periodo inicial e final
- o endpoint deve atender operacao e backoffice

Quero no resultado:
- problema
- objetivos e nao objetivos
- requisitos funcionais e nao funcionais
- criterios de aceite
- riscos
```

### 6. Criar a tech spec com foco em DDD e arquitetura

Exemplo de prompt para o agente:

```text
Use a skill create-technical-specification com base no PRD aprovado.
Carregue tambem as referencias necessarias de DDD e arquitetura.

Contexto tecnico:
- servico Go existente
- arquitetura atual: handler -> service -> repository
- preservar contratos publicos existentes

Quero no resultado:
- modelagem de dominio
- agregados, entidades e value objects se fizer sentido
- fronteiras entre aplicacao, dominio e infraestrutura
- estrategia de erros
- estrategia de testes
- riscos e plano de rollout
```

### 7. Gerar o bundle de tasks

Exemplo de prompt para o agente:

```text
Use a skill create-tasks para decompor a tech spec em tasks pequenas,
executaveis e com evidencias de validacao.

Quero:
- ordem de execucao
- dependencias entre tasks
- criterio de pronto por task
- arquivos esperados: tasks.md e uma task por arquivo
```

Estrutura esperada:

```text
tasks/
  prd-payments-list/
    prd.md
    techspec.md
    tasks.md
    01_task.md
    02_task.md
    03_task.md
```

### 8. Executar todas as tasks com o looper do CLI

O looper do projeto e o comando `task-loop`. Ele percorre `tasks.md`, identifica a proxima task elegivel e invoca o agente com a skill `execute-task` ate concluir todas as tasks possiveis.

```bash
ai-spec task-loop --tool codex tasks/prd-payments-list
```

Exemplos uteis:

```bash
# simular sem invocar o agente
ai-spec task-loop --tool codex --dry-run tasks/prd-payments-list

# limitar iteracoes para um lote inicial
ai-spec task-loop --tool codex --max-iterations 3 tasks/prd-payments-list

# aumentar timeout por task e salvar relatorio final em caminho explicito
ai-spec task-loop --tool codex --timeout 1h --report-path ./task-loop-report.md tasks/prd-payments-list
```

### 9. Validar o estado final

```bash
ai-spec lint .
go test ./...
```

## Como obter o melhor das skills — uso mandatorio e rigoroso

Esta secao define o contrato de uso de cada skill. Seguir estas regras nao e opcional: desvios introduzem ambiguidade, retrabalho e artefatos incompativeis entre etapas.

### Principio central

Cada skill e uma etapa com entradas obrigatorias, saidas esperadas e criterios de aceite. Nao avance para a proxima etapa sem validar a saida da etapa anterior. O agente nao deve inferir o que falta — voce deve fornecer.

### Pipeline mandatorio

```text
analyze-project (se projeto existente)
        |
        v
   create-prd
        |
        v
create-technical-specification
        |
        v
   create-tasks
        |
        v
   execute-task  <-- repete por task
        |
        v
     review
        |
        v
    refactor (se necessario)
```

Regra absoluta: nunca execute `execute-task` sem `tasks.md` aprovado. Nunca execute `create-tasks` sem tech spec aprovada. Nunca execute `create-technical-specification` sem PRD aprovado.

---

### `create-prd` — escopo e requisitos

**Quando usar:** toda feature nova ou qualquer mudanca que altere comportamento observavel do sistema.

**Entradas obrigatorias no prompt:**
- problema que a feature resolve (nao a solucao)
- persona ou sistema afetado
- restricoes de escopo (o que esta fora)
- restricoes tecnicas conhecidas (performance, integracao, compliance)

**Prompt mandatorio:**

```text
Use a skill create-prd para definir [nome da feature].

Entradas:
- Problema: [descricao objetiva do problema]
- Persona afetada: [quem sofre o problema]
- Restricoes de escopo: [o que nao esta incluido]
- Restricoes tecnicas: [limites conhecidos]

Saidas esperadas obrigatorias:
- problema claro e verificavel
- objetivos mensuráveis
- nao objetivos explicitos
- requisitos funcionais numerados (RF-01, RF-02...)
- requisitos nao funcionais (RNF-01, RNF-02...)
- criterios de aceite por requisito funcional
- riscos com probabilidade e impacto
```

**Criterios de aceite do artefato — nao avance sem estes:**
- [ ] cada RF tem criterio de aceite testavel
- [ ] nao objetivos estao listados (previnem escopo rastejante)
- [ ] riscos tem mitigacao ou decisao explicita
- [ ] artefato salvo em `tasks/<prd-folder>/prd.md`

---

### `create-technical-specification` — arquitetura e contratos

**Quando usar:** apos PRD aprovado, antes de qualquer linha de codigo.

**Entradas obrigatorias no prompt:**
- caminho do `prd.md` aprovado
- contexto tecnico do repositorio (stack, arquitetura atual, fronteiras existentes)
- restricoes que o PRD impos
- referencias a carregar (DDD, arquitetura, concorrencia — conforme a feature)

**Prompt mandatorio:**

```text
Use a skill create-technical-specification com base no PRD aprovado em tasks/<prd-folder>/prd.md.

Contexto tecnico:
- Stack: [linguagem, frameworks, banco de dados]
- Arquitetura atual: [ex: handler -> service -> repository]
- Fronteiras existentes: [o que nao pode mudar]
- Referencias necessarias: [ddd, arquitetura, concorrencia, testes]

Saidas esperadas obrigatorias:
- modelagem de dominio (agregados, entidades, value objects se houver invariantes)
- contratos de interface (assinaturas, DTOs, erros tipados)
- responsabilidade de cada camada
- estrategia de erros com tipos e mensagens
- estrategia de testes (unitarios, integracao, table-driven)
- ADRs para decisoes nao obvias
- riscos tecnicos e plano de rollout
```

**Criterios de aceite do artefato — nao avance sem estes:**
- [ ] contratos de interface definidos (nao implicitos)
- [ ] estrategia de erros explicita (sem "retornar erro generico")
- [ ] estrategia de testes inclui cenarios de falha
- [ ] decisoes arquiteturais justificadas (ADR ou nota inline)
- [ ] artefato salvo em `tasks/<prd-folder>/techspec.md`

---

### `create-tasks` — decomposicao em unidades executaveis

**Quando usar:** apos tech spec aprovada, para gerar o bundle de tasks que o `execute-task` ou `task-loop` vai consumir.

**Entradas obrigatorias no prompt:**
- caminho do `prd.md` e `techspec.md` aprovados
- criterio de granularidade desejado (uma responsabilidade por task)

**Prompt mandatorio:**

```text
Use a skill create-tasks para decompor a tech spec aprovada em tasks.

Arquivos de entrada:
- PRD: tasks/<prd-folder>/prd.md
- Tech spec: tasks/<prd-folder>/techspec.md

Regras de decomposicao obrigatorias:
- uma responsabilidade por task (ex: nao misturar handler com repository na mesma task)
- ordem de execucao explicita (dependencias declaradas)
- criterio de pronto por task com validacoes executaveis
- nenhuma task com escopo ambiguo ou aberto

Saidas esperadas obrigatorias:
- tasks.md com lista ordenada e dependencias
- um arquivo por task (01_task.md, 02_task.md...) com: objetivo, arquivos afetados, criterio de pronto, validacoes
```

**Criterios de aceite do artefato — nao avance sem estes:**
- [ ] nenhuma task tem dependencia circular
- [ ] cada task tem criterio de pronto com comandos de validacao (ex: `go test ./...`)
- [ ] tamanho de cada task: implementavel em uma unica sessao de agente
- [ ] artefatos salvos em `tasks/<prd-folder>/`

---

### `execute-task` — implementacao com evidencia

**Quando usar:** para cada task do bundle, uma de cada vez. Nao execute tasks em paralelo sem garantir que nao ha dependencia entre elas.

**Entradas obrigatorias no prompt:**
- caminho exato da task a executar
- contexto arquitetural (nao confie que o agente inferira)
- referencias obrigatorias da linguagem e do dominio

**Prompt mandatorio:**

```text
Use a skill execute-task para implementar a task tasks/<prd-folder>/<N>_task.md.

Contexto obrigatorio:
- Leia o arquivo de task antes de iniciar qualquer alteracao
- Arquitetura: [descreva a camada e os contratos relevantes]
- Referencias a carregar: [go-implementation, ddd, tests — conforme a task]

Criterios de execucao nao negociaveis:
- preservar contratos publicos existentes (nenhuma assinatura publica muda sem ADR)
- nenhuma interface nova sem fronteira real justificada
- context.Context em todas as operacoes de IO
- testes table-driven para todos os cenarios do criterio de pronto
- registrar evidencia de conclusao no arquivo de task (output do teste, lint)
- nao fechar a task sem evidencia de validacao
```

**Criterios de aceite da execucao — nao marque a task como concluida sem estes:**
- [ ] todos os testes do criterio de pronto passam (`go test ./...`)
- [ ] lint sem erro (`go vet ./...` ou equivalente)
- [ ] nenhum contrato publico foi alterado sem justificativa registrada
- [ ] evidencia registrada no arquivo de task

---

### Como executar tasks com fidelidade maxima — ciclo por task com limpeza de contexto

Este e o padrao de execucao que produz o melhor resultado: uma task por sessao de agente, contexto limpo a cada inicio, fidelidade total ao artefato especificado.

#### Por que limpar o contexto entre tasks

O agente acumula "pressao de continuidade" ao longo de uma sessao: tende a manter decisoes anteriores, introduzir abstraocoes que emergiram na task passada e desviar do artefato atual. Limpar o contexto elimina esse risco. Cada task recebe um agente que so conhece o que voce forneceu naquela invocacao.

Regra: **uma sessao de agente por task. Ao terminar uma task, feche a sessao e abra uma nova.**

#### Fluxo de execucao por task

```text
[abrir nova sessao]
        |
        v
[fornecer contexto minimo obrigatorio: techspec + task file]
        |
        v
[executar execute-task com prompt mandatorio]
        |
        v
[validar criterio de pronto: testes + lint + evidencia]
        |
        v
[registrar evidencia no arquivo de task]
        |
        v
[fechar sessao]
        |
        v
[abrir nova sessao para a proxima task]
```

#### Contexto minimo obrigatorio por sessao

Cada nova sessao deve receber exatamente estes tres elementos — nem menos (risco de desvio), nem mais (risco de contaminacao):

1. o arquivo da task atual (`<N>_task.md`)
2. o trecho relevante da tech spec (`techspec.md`) — apenas as secoes que a task toca
3. a regra arquitetural da camada sendo implementada

Nao forneca tasks anteriores, outputs de sessoes passadas ou contexto de features paralelas. Isso contamina o contexto e aumenta a chance de o agente "lembrar" uma decisao que nao deve ser replicada.

#### Prompt mandatorio por task (template copiavel)

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/<prd-folder>/<N>_task.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio para esta task:
- Arquitetura: [camada e responsabilidade, ex: "repository — acesso a banco, sem regra de negocio"]
- Contratos relevantes da tech spec: [cole apenas as assinaturas e tipos que esta task usa]
- Invariantes que nao podem mudar: [contratos publicos, tipos de erro, comportamento esperado pelos testes]

Regras de execucao nao negociaveis:
- siga o criterio de pronto definido no arquivo de task — nao adicione nem remova escopo
- nenhuma interface nova sem fronteira real justificada na tech spec
- context.Context em todas as operacoes de IO
- testes table-driven cobrindo todos os cenarios do criterio de pronto
- ao finalizar: rode os testes, rode o lint, registre o output como evidencia no arquivo de task
- nao declare a task concluida sem evidencia registrada

Nao inferir. Se o arquivo de task tiver ambiguidade, pare e aponte antes de implementar.
```

#### Exemplo concreto: bundle de 3 tasks em sessoes separadas

Estrutura do bundle:

```text
tasks/prd-payments-list/
  prd.md
  techspec.md
  tasks.md
  01_repository.md   <- acesso ao banco, query com filtros
  02_service.md      <- regras de aplicacao, orquestracao
  03_handler.md      <- parse HTTP, validacao de input, resposta
```

**Sessao 1 — task 01 (repository):**

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/prd-payments-list/01_repository.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio:
- Arquitetura: camada repository — acesso a banco, sem regra de negocio, sem HTTP
- Contratos da tech spec:
    type PaymentRepository interface {
        List(ctx context.Context, filter PaymentFilter) ([]Payment, error)
    }
    type PaymentFilter struct { Status string; Page int; From, To time.Time }
- Invariantes: nenhuma logica de negocio nesta camada; erros devem usar os tipos definidos em techspec.md

Regras: siga o criterio de pronto da task. Testes table-driven para filtros. Registre evidencia ao finalizar.
```

[fechar sessao 1]

**Sessao 2 — task 02 (service):**

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/prd-payments-list/02_service.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio:
- Arquitetura: camada service — orquestracao e regras de aplicacao, sem acesso direto ao banco
- Contratos da tech spec:
    type PaymentService interface {
        List(ctx context.Context, filter PaymentFilter) ([]PaymentDTO, error)
    }
    O service recebe PaymentRepository por injecao; nao instancia dependencias.
- Invariantes: nao duplicar validacoes do handler; erros do repository devem ser propagados com contexto

Regras: siga o criterio de pronto da task. Testes com mock do repository. Registre evidencia ao finalizar.
```

[fechar sessao 2]

**Sessao 3 — task 03 (handler):**

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/prd-payments-list/03_handler.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio:
- Arquitetura: camada handler — parse HTTP, validacao de input, mapeamento para DTO de resposta
- Contratos da tech spec:
    GET /payments?status=&page=&from=&to=
    200: { data: PaymentDTO[], total: int, page: int }
    400: { error: string } para input invalido
    500: { error: string } para erro interno
    O handler recebe PaymentService por injecao.
- Invariantes: nenhuma logica de negocio no handler; status HTTP mapeados conforme techspec.md

Regras: siga o criterio de pronto da task. Testes table-driven para cenarios de input valido, invalido e erro do service. Registre evidencia ao finalizar.
```

[fechar sessao 3]

#### O que fazer se o agente desviar durante a execucao

Se o agente iniciar uma task sem ler o arquivo, introduzir escopo nao previsto, criar abstraocao nao especificada ou pular a evidencia de validacao, interrompa imediatamente. Nao corrija na mesma sessao — feche, abra uma nova e reenvie o prompt mandatorio com a correcao explicita do desvio:

```text
Na sessao anterior houve um desvio: [descreva o desvio].
Nova sessao. Sem contexto anterior.

Tarefa: reimplementar tasks/<prd-folder>/<N>_task.md do zero.
[prompt mandatorio completo]

Correcao especifica: [o que nao deve ser feito desta vez]
```

#### Resumo das regras de ouro

| Regra | Motivo |
| --- | --- |
| uma sessao por task | elimina pressao de continuidade e desvio acumulado |
| contexto minimo — apenas o que a task precisa | contaminacao de contexto e a causa mais comum de desvio |
| leia o arquivo de task antes de qualquer acao | o agente deve seguir o artefato, nao inferir |
| nao declare concluida sem evidencia | evidencia e o unico sinal objetivo de que a task foi feita corretamente |
| desvio detectado: feche e recomece | correcao dentro da sessao desviada tende a gerar mais desvio |

---

#### Execucao automatica do mesmo comportamento com task-loop

O `task-loop` ja implementa automaticamente todo o ciclo descrito acima. Nao e necessario abrir e fechar sessoes manualmente.

**Por que o task-loop garante isolamento de contexto:**

Cada iteracao do `task-loop` invoca o agente como um processo completamente novo via `exec.CommandContext` com a flag `--print -p <prompt>`. Isso significa:

- nenhum estado da sessao anterior e carregado — o processo inicia do zero
- o contexto injetado e exatamente o minimo obrigatorio: arquivo da task + pasta do PRD (que contem `prd.md` e `techspec.md`)
- a variavel `AI_INVOCATION_DEPTH` e resetada para `0` a cada invocacao, prevenindo aninhamento
- ao terminar ou falhar, o processo e encerrado com `SIGKILL` no grupo inteiro — sem estado residual

**Equivalencia entre o ciclo manual e o task-loop:**

| Ciclo manual | O que task-loop faz automaticamente |
| --- | --- |
| abrir nova sessao | spawn de novo processo por task |
| fornecer task file + trecho da techspec | passa `task file` + `prd folder` no prompt; agente instrui a carregar `prd.md` e `techspec.md` |
| executar execute-task com prompt mandatorio | prompt gerado por `BuildPrompt` com instrucao de seguir `SKILL.md` |
| fechar sessao apos evidencia registrada | processo encerrado; status relido de `tasks.md` e do arquivo da task |
| abrir nova sessao para proxima task | proxima iteracao spawn novo processo |

**Comando para executar o ciclo automatico com fidelidade maxima:**

Primeiro valide sem gastar ciclo de agente:

```bash
ai-spec task-loop --tool claude --dry-run tasks/prd-payments-list
```

Execute uma task de cada vez, observando qualidade antes de ampliar o lote:

```bash
ai-spec task-loop --tool claude --max-iterations 1 tasks/prd-payments-list
```

Quando a qualidade do primeiro lote estiver boa, aumente gradualmente:

```bash
ai-spec task-loop --tool claude --max-iterations 3 --timeout 30m tasks/prd-payments-list
```

Execucao completa com rastreabilidade:

```bash
ai-spec task-loop \
  --tool claude \
  --max-iterations 10 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  tasks/prd-payments-list
```

**Quando preferir o ciclo manual em vez do task-loop:**

| Situacao | Abordagem |
| --- | --- |
| task com ambiguidade na spec — precisa de input antes de implementar | ciclo manual: leia a task, resolva a ambiguidade, depois execute |
| task que toca fronteira arquitetural nao documentada na techspec | ciclo manual: adicione o contexto faltante no prompt antes de invocar |
| bundle ainda instavel — tasks sem criterio de pronto claro | ciclo manual com `--max-iterations 1` ate estabilizar |
| bundle maduro, criterios de pronto claros, techspec completa | `task-loop` automatico |

**Sinal de que o task-loop pode ser usado com seguranca:**

- `tasks.md` tem dependencias explicitas e nenhuma task com escopo aberto
- cada task file tem secao `Criterio de pronto` com comandos de validacao concretos
- `techspec.md` define contratos, tipos de erro e responsabilidade de cada camada
- `dry-run` nao aponta task com status invalido ou dependencia circular

Se qualquer um desses itens estiver faltando, resolva antes de rodar o loop automatico. O task-loop e tao bom quanto os artefatos que consome.

---

#### Execucao sem task-loop — alternativas com isolamento de sessao

Sim, e possivel executar com isolamento sem o `task-loop`. Ha tres abordagens, cada uma com tradeoffs diferentes.

**O que o task-loop faz sob o hood (base para as alternativas):**

Cada iteracao do `task-loop` executa exatamente este comando por tool:

```bash
# Claude
claude --dangerously-skip-permissions --print --bare -p "<prompt>"

# Codex
codex exec --dangerously-bypass-approvals-and-sandbox -p "<prompt>"

# Gemini
gemini --yolo -p "<prompt>"

# Copilot
copilot --autopilot --yolo -p "<prompt>"
```

O prompt e gerado por `BuildPrompt(taskFilePath, prdFolder)` e contem:

```text
You are executing the "execute-task" skill.

First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/execute-task/SKILL.md

Target task file: <task-file>
PRD folder: <prd-folder>

Execute ONLY this task. Follow all skill steps:
1. Validate eligibility
2. Load context (prd.md, techspec.md)
3. Implement
4. Validate (tests, lint)
5. Review
6. Update task status in task file and tasks.md
7. Generate execution report
```

Qualquer alternativa que replique esse comportamento tem isolamento garantido.

---

**Alternativa 1 — invocar o agente diretamente por task (mais simples)**

Use quando quiser controle total sobre qual task executar e quando, sem depender do parser de `tasks.md`.

Construa o prompt manualmente seguindo o mesmo padrao do `BuildPrompt` e invoque o binario do agente diretamente:

```bash
# Claude — uma task
claude --dangerously-skip-permissions --print --bare -p \
  "You are executing the \"execute-task\" skill.

First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/execute-task/SKILL.md

Target task file: tasks/prd-payments-list/01_repository.md
PRD folder: tasks/prd-payments-list

Execute ONLY this task. Follow all skill steps:
1. Validate eligibility
2. Load context (prd.md, techspec.md)
3. Implement
4. Validate (tests, lint)
5. Review
6. Update task status in task file and tasks.md
7. Generate execution report

Update **Status:** in tasks/prd-payments-list/01_repository.md and the corresponding row in tasks/prd-payments-list/tasks.md to reflect the final state."
```

Troque o `Target task file` para cada task e execute um comando por vez. Cada invocacao e um processo novo — isolamento identico ao `task-loop`.

Para Codex ou Gemini, substitua o binario e as flags:

```bash
# Codex
codex exec --dangerously-bypass-approvals-and-sandbox -p "<mesmo prompt>"

# Gemini
gemini --yolo -p "<mesmo prompt>"
```

---

**Alternativa 2 — script shell iterando tasks.md**

Use quando quiser automatizar o ciclo sem o `task-loop` mas com controle do shell (logs customizados, notificacoes, condicoes de parada proprias).

O `tasks.md` usa formato de tabela markdown com as colunas `ID`, `Title`, `Status` e `Dependencies`. Uma task e elegivel quando `Status == pending` e todas as dependencias estao com `Status == done`.

Script minimo para iterar e invocar uma task por vez com Claude:

```bash
#!/usr/bin/env bash
set -euo pipefail

PRD_FOLDER="${1:?informe o prd folder}"
TOOL="${2:-claude}"
MAX="${3:-99}"
count=0

while [ "$count" -lt "$MAX" ]; do
  # encontrar proxima task pendente (sem dependencia bloqueando)
  TASK_FILE=$(ls "${PRD_FOLDER}"/[0-9]*_*.md 2>/dev/null \
    | while read -r f; do
        status=$(grep -m1 "^\*\*Status:\*\*" "$f" | awk '{print $2}' | tr -d '[:space:]')
        [ "$status" = "pending" ] && echo "$f" && break
      done)

  [ -z "$TASK_FILE" ] && echo "nenhuma task pendente" && break

  PROMPT="You are executing the \"execute-task\" skill.

First, read AGENTS.md at the repository root to load governance rules and conventions.

Then read and follow the instructions in: .agents/skills/execute-task/SKILL.md

Target task file: ${TASK_FILE}
PRD folder: ${PRD_FOLDER}

Execute ONLY this task. Follow all skill steps:
1. Validate eligibility
2. Load context (prd.md, techspec.md)
3. Implement
4. Validate (tests, lint)
5. Review
6. Update task status in task file and tasks.md
7. Generate execution report

Update **Status:** in ${TASK_FILE} and the corresponding row in ${PRD_FOLDER}/tasks.md to reflect the final state."

  echo "executando: ${TASK_FILE}"
  case "$TOOL" in
    claude)  claude --dangerously-skip-permissions --print --bare -p "$PROMPT" ;;
    codex)   codex exec --dangerously-bypass-approvals-and-sandbox -p "$PROMPT" ;;
    gemini)  gemini --yolo -p "$PROMPT" ;;
    copilot) copilot --autopilot --yolo -p "$PROMPT" ;;
  esac

  count=$((count + 1))
done
```

Uso:

```bash
chmod +x run-tasks.sh
./run-tasks.sh tasks/prd-payments-list claude 3
```

Limitacoes deste script em relacao ao `task-loop`:
- nao valida dependencias entre tasks (so verifica `pending`)
- nao gera relatorio estruturado
- nao lida com `blocked`, `failed` ou `needs_input`
- nao tem timeout por task

Para essas garantias, use o `task-loop`. O script e util para casos simples ou para entender o mecanismo.

---

**Alternativa 3 — `--max-iterations 1` como substituto do ciclo manual**

Se o objetivo e executar uma task de cada vez com pausa para revisao entre elas, `--max-iterations 1` e a opcao mais segura. Ela usa exatamente o mesmo isolamento do loop completo, mas para apos a primeira task elegivel.

```bash
# executar uma task, revisar, depois rodar novamente
ai-spec task-loop --tool claude --max-iterations 1 tasks/prd-payments-list
# revisar o resultado
ai-spec task-loop --tool claude --max-iterations 1 tasks/prd-payments-list
# continuar ate concluir
```

---

**Comparativo das abordagens**

| Abordagem | Isolamento de sessao | Dependencias entre tasks | Timeout | Relatorio | Quando usar |
| --- | --- | --- | --- | --- | --- |
| `task-loop` completo | automatico | sim | sim | sim | bundle maduro, execucao sem supervisao |
| `task-loop --max-iterations 1` | automatico | sim | sim | sim | execucao task a task com pausa para revisao |
| invocacao direta por task | manual (um processo por chamada) | nao (voce controla a ordem) | nao | nao | task isolada, ambiguidade na spec, fronteira nao documentada |
| script shell | manual (um processo por iteracao) | parcial (so status pending) | nao nativo | nao | automacao leve sem dependencias complexas |

**Conclusao:** o `task-loop` e a implementacao de referencia e a mais robusta. As alternativas existem e funcionam para casos especificos, mas replicam apenas parte do comportamento. Para isolamento garantido com gestao de dependencias, timeout e rastreabilidade, use o `task-loop`.

---

### `review` — revisao antes de merge

**Quando usar:** obrigatoriamente antes de qualquer merge ou fechamento de ciclo de tasks. Nao e opcional.

**Entradas obrigatorias no prompt:**
- diff ou branch a revisar
- contexto do que foi implementado (skill usada, task referenciada)
- areas de risco conhecidas (performance, seguranca, contratos)

**Prompt mandatorio:**

```text
Use a skill review para revisar o diff atual.

Contexto da implementacao:
- Tasks executadas: [lista de tasks do bundle]
- Skill usada na implementacao: execute-task
- Areas de risco: [performance, seguranca, contratos, concorrencia]

Focos obrigatorios da revisao:
- corretude: a implementacao atende todos os RFs e criterios de aceite do PRD?
- regressao: alguma mudanca quebra contrato publico ou comportamento existente?
- seguranca: ha injecao de dependencia insegura, dado sensivel exposto ou validacao faltando?
- testes: todos os cenarios do criterio de pronto estao cobertos?
- dívida tecnica introduzida: o que precisara de refactor futuro?

Saidas esperadas:
- lista de achados por categoria (critico, importante, sugestao)
- para cada achado critico: arquivo, linha, descricao e correcao sugerida
- veredicto final: aprovado / aprovado com ressalvas / reprovado
```

**Criterios de aceite da revisao — nao avance sem estes:**
- [ ] nenhum achado critico em aberto
- [ ] achados importantes tem plano de correcao ou decisao explicita de aceitar o risco
- [ ] veredicto registrado

---

### `refactor` — melhoria sem mudanca de comportamento

**Quando usar:** quando a `review` apontar divida tecnica relevante ou quando uma area do codigo estiver impedindo evolucao segura. Nao use como substituto para implementacao correta na primeira vez.

**Entradas obrigatorias no prompt:**
- achados da `review` que motivam o refactor
- escopo delimitado (nao refatore fora do escopo dos achados)
- invariantes que nao podem mudar (contratos publicos, comportamento observavel)

**Prompt mandatorio:**

```text
Use a skill refactor para corrigir os achados de refatoracao da revisao.

Achados que motivam este refactor:
- [lista dos achados da review com arquivo e linha]

Escopo obrigatorio:
- apenas os arquivos e funcoes apontados pelos achados
- nenhuma mudanca de comportamento observavel

Invariantes que nao podem mudar:
- contratos publicos: [lista de assinaturas e endpoints]
- comportamento de erro: [tipos de erro e mensagens esperados pelos testes]

Criterios de conclusao:
- todos os testes existentes continuam passando
- lint sem erro
- diff gerado e revisado antes de commitar
```

**Criterios de aceite do refactor — nao commite sem estes:**
- [ ] nenhum teste regressou
- [ ] nenhum contrato publico alterado
- [ ] diff menor que o esperado (sem escopo rastejante)
- [ ] segunda passagem de `review` no diff do refactor se o escopo foi grande

---

### Checklist de entrada por skill

Use esta tabela antes de invocar qualquer skill. Se um item obrigatorio estiver faltando, providencie antes de invocar.

| Skill | Obrigatorio antes de invocar | Artefato de saida esperado |
| --- | --- | --- |
| `create-prd` | problema definido, persona, restricoes de escopo | `tasks/<folder>/prd.md` |
| `create-technical-specification` | `prd.md` aprovado, contexto tecnico, referencias | `tasks/<folder>/techspec.md` |
| `create-tasks` | `prd.md` + `techspec.md` aprovados | `tasks/<folder>/tasks.md` + `<N>_task.md` |
| `execute-task` | task file com criterio de pronto, contexto arquitetural | evidencia no task file, testes passando |
| `review` | diff ou branch, contexto da implementacao, areas de risco | lista de achados com veredicto |
| `refactor` | achados de `review`, escopo delimitado, invariantes | diff revisado, testes passando |

### Sinais de desvio — quando parar e corrigir

Se qualquer um dos seguintes acontecer, pare e corrija antes de continuar:

- agente gerou codigo sem ler o arquivo de task
- tech spec nao tem estrategia de erros explicita
- task executada sem evidencia registrada
- review pulada por "a implementacao parece correta"
- refactor alterou comportamento observavel
- PRD sem criterios de aceite testáveis
- tasks.md com tasks que misturam responsabilidades

Esses sao os desvios mais comuns e os que mais causam retrabalho.

---

## SDD + harness para features Go

Se voce quiser tirar o melhor do repositorio em features Go, o fluxo mais forte costuma ser usar o harness de forma guiada por especificacao:

```text
create-prd -> create-technical-specification -> create-tasks -> execute-task
```

Essa abordagem funciona bem porque separa quatro preocupacoes:

- produto e escopo
- desenho tecnico e fronteiras arquiteturais
- decomposicao em tasks pequenas
- implementacao com validacao e revisao

### Exemplo 1: feature Go mais complexa com DDD, goroutines e performance

Bom para casos como processamento paralelo, pipelines internos, consolidacao de dados, importacao assincrona ou qualquer feature em que concorrencia e modelagem importam tanto quanto o endpoint.

Prompt para o PRD:

```text
Use a skill create-prd para definir a feature de conciliacao de recebimentos.

Contexto:
- servico Go existente
- precisamos processar lotes grandes sem degradar a latencia do sistema
- a feature vai combinar regras de negocio e processamento concorrente

Quero no resultado:
- problema
- objetivos e nao objetivos
- requisitos funcionais
- requisitos nao funcionais
- restricoes de performance e observabilidade
- riscos operacionais
```

Prompt para a tech spec:

```text
Use a skill create-technical-specification com base no PRD aprovado.
Carregue referencias de DDD, arquitetura Go, concorrencia e testes quando necessario.

Contexto tecnico:
- servico Go existente
- preservar fronteiras de dominio
- evitar goroutines soltas e vazamentos
- usar context.Context nas fronteiras cancelaveis
- garantir backpressure ou limite de concorrencia quando fizer sentido

Quero no resultado:
- agregados, entidades e value objects se fizer sentido
- separacao entre handler, aplicacao, dominio e infraestrutura
- desenho do fluxo concorrente
- estrategia para worker pool, channels ou sincronizacao apenas se houver ganho real
- riscos de race, starvation, leak e contencao
- estrategia de testes e observabilidade
```

Prompt para a execucao:

```text
Use a skill execute-task para implementar a proxima task elegivel desta feature Go.

Criterios obrigatorios:
- preservar DDD e fronteiras arquiteturais existentes
- preferir tipos concretos por padrao
- introduzir interface apenas em fronteiras reais
- tratar concorrencia com cancelamento, limites e ownership claro das goroutines
- explicitar trade-offs de performance quando houver fan-out, batching ou worker pool
- manter erros com contexto e testes de regressao
```

### Exemplo 2: feature Go simples de API de listagem

Bom para um caso mais comum: criar um endpoint de listagem seguindo o fluxo `handler -> service -> repository`, com DDD leve e sem sofisticacao desnecessaria.

Prompt para a tech spec:

```text
Use a skill create-technical-specification para uma nova API GET /payments.
Carregue referencias de DDD e arquitetura Go quando necessario.

Contexto tecnico:
- servico Go existente
- arquitetura atual: handler -> service -> repository
- queremos manter a separacao atual sem overengineering

Quero no resultado:
- contrato do endpoint
- DTOs de entrada e saida
- regras de filtragem, paginacao e ordenacao
- responsabilidade de cada camada
- estrategia de erros
- estrategia de testes
```

Prompt para a execucao:

```text
Use a skill execute-task para implementar a API de listagem de pagamentos.

Quero que a implementacao siga:
- handler: parse, validacao de input e resposta HTTP
- service: regras de aplicacao e orquestracao
- repository: consulta e persistencia

Requisitos:
- manter nomes claros e funcoes pequenas
- usar context.Context nas operacoes de IO
- evitar abstrair alem do necessario
- adicionar testes table-driven para filtros e cenarios principais
```

### Quando usar cada estilo

| Cenario | Melhor ponto de partida |
| --- | --- |
| feature nova ainda ambigua | `create-prd` |
| feature definida mas com risco de arquitetura | `create-technical-specification` |
| implementacao aprovada em artefatos `tasks/` | `execute-task` |
| endpoint simples com desenho ja conhecido | tech spec curta + `create-tasks` |
| feature com concorrencia, throughput ou latencia critica | PRD + tech spec detalhada antes de implementar |

### Wrapper para iniciar mais rapido

Se estiver usando Codex, Gemini ou Copilot, voce pode emitir a instrucao pronta antes de abrir o prompt:

```bash
ai-spec wrapper codex create-technical-specification .
ai-spec wrapper codex execute-task .
```

## Task-loop de forma eficiente

Hoje o README ja mostra o `task-loop` dentro do fluxo completo, mas vale um workflow objetivo para usa-lo com menos retrabalho.

### Fluxo recomendado

1. Primeiro valide se as tasks estao pequenas, ordenadas e com dependencias claras.
2. Rode um `dry-run` para confirmar qual task sera escolhida primeiro.
3. Comece com poucas iteracoes para observar a qualidade do lote inicial.
4. So depois aumente `max-iterations` e `timeout` para execucao mais longa.
5. Sempre salve o relatorio final em caminho explicito quando estiver trabalhando em uma feature relevante.

### Comandos recomendados

Inspecao inicial:

```bash
ai-spec task-loop --tool codex --dry-run tasks/prd-payments-list
```

Primeiro lote pequeno:

```bash
ai-spec task-loop --tool codex --max-iterations 2 tasks/prd-payments-list
```

Execucao mais longa com relatorio salvo:

```bash
ai-spec task-loop \
  --tool codex \
  --max-iterations 8 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  tasks/prd-payments-list
```

### Quando cada flag ajuda

| Flag | Quando usar |
| --- | --- |
| `--dry-run` | validar ordem e elegibilidade das tasks antes de gastar ciclo de agente |
| `--max-iterations` | controlar lote inicial, reduzir risco e evitar rodar tasks demais de uma vez |
| `--timeout` | dar mais tempo para tasks grandes ou com validacoes demoradas |
| `--report-path` | manter rastreabilidade e facilitar revisao posterior |

### Heuristicas praticas

- prefira `max-iterations` baixo no inicio de uma feature nova
- aumente o lote apenas quando as primeiras tasks estiverem saindo com boa qualidade
- use `task-loop` quando o pacote de tasks ja estiver maduro; para task isolada ou ainda instavel, prefira `execute-task`
- se a feature for Go e envolver DDD, concorrencia ou risco arquitetural, refine antes em `techspec.md` em vez de delegar ambiguidade ao loop

## Executar uma task ou varias tasks

Existem dois modos principais de executar tasks de um PRD folder: executar uma unica task diretamente ou deixar o `task-loop` iterar pelo pacote inteiro. A escolha depende de quao maduro e o pacote de tasks e do quanto de controle voce quer manter por iteracao.

### Sem task-loop: executar uma task especifica

Use esse modo quando quiser revisar ou executar uma task isolada sem passar pelo orchestrador. Funciona bem para tasks experimentais, ajustes pontuais ou quando o pacote ainda nao esta maduro o suficiente para execucao em lote.

Inspecione o pacote e escolha a task manualmente:

```bash
ai-spec inspect ../api-pagamentos
```

Emita a instrucao pronta para o agente:

```bash
ai-spec wrapper codex execute-task .
```

Prompt direto para o agente executar uma unica task:

```text
Use a skill execute-task para implementar a task 01_task.md localizada em tasks/prd-payments-list.

Criterios obrigatorios:
- ler o arquivo de task antes de iniciar qualquer alteracao
- preservar a arquitetura e os contratos publicos existentes
- executar os testes e validacoes definidos na propria task
- registrar evidencias de conclusao no arquivo de task
```

Para uma task mais complexa, com contexto arquitetural explicito:

```text
Use a skill execute-task para implementar a task 02_task.md em tasks/prd-payments-list.
Carregue tambem as referencias de DDD e arquitetura Go.

Contexto:
- servico Go existente com arquitetura handler -> service -> repository
- preservar fronteiras de camada
- usar context.Context nas operacoes de IO
- adicionar testes table-driven para os cenarios principais
```

### Com task-loop: executar todas as tasks elegiveis

Use esse modo quando o pacote de tasks ja estiver maduro, ordenado e com dependencias claras. O `task-loop` percorre `tasks.md`, identifica a proxima task elegivel e invoca o agente com `execute-task` ate concluir todas as tasks possiveis.

Validar ordem e elegibilidade antes de gastar ciclo de agente:

```bash
ai-spec task-loop --tool codex --dry-run tasks/prd-payments-list
```

Executar as primeiras tasks com lote pequeno para observar qualidade:

```bash
ai-spec task-loop --tool codex --max-iterations 2 tasks/prd-payments-list
```

Execucao completa do pacote com relatorio salvo:

```bash
ai-spec task-loop \
  --tool codex \
  --max-iterations 8 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  tasks/prd-payments-list
```

### Quando usar cada abordagem

| Situacao | Abordagem recomendada |
| --- | --- |
| task isolada, experimental ou instavel | `execute-task` direto (sem task-loop) |
| revisar uma unica task antes de continuar o lote | `execute-task` direto |
| pacote maduro, ordenado e com dependencias claras | `task-loop` |
| primeiro lote de uma feature nova — ainda incerto | `task-loop` com `--max-iterations 2` |
| execucao longa com rastreabilidade necessaria | `task-loop` com `--report-path` |
| feature com concorrencia ou risco arquitetural alto | refinar `techspec.md` antes de qualquer execucao |

## Referencia rapida de comandos

| Comando | Finalidade |
| --- | --- |
| `install` | Instala governanca de IA em um projeto alvo |
| `upgrade` | Atualiza skills, adaptadores e manifesto em uma instalacao existente |
| `inspect` | Exibe skills instaladas, ferramentas detectadas e estado do manifesto |
| `doctor` | Executa checks de saude sobre git, manifesto, symlinks e permissoes |
| `lint` | Detecta placeholders nao renderizados, schema divergente e `SKILL.md` invalidos; `--strict` trata avisos de paridade como erros |
| `metrics` | Calcula metricas de contexto e custo estimado de tokens |
| `telemetry` | Registra e resume uso de skills e referencias; suporta `--trend`, `--budget-check` e `--top-skills` |
| `skills check` | Verifica versoes de skills externas contra `skills-lock.json` e detecta mudancas de interface |
| `validate` | Valida frontmatter YAML de `SKILL.md` |
| `validate-bugs` | Valida um array JSON de bugs contra o schema canonico |
| `prerequisites` | Verifica se uma skill pode ser executada em um projeto |
| `task-loop` | Executa todas as tasks elegiveis de um PRD folder via agente de IA |
| `wrapper` | Emite instrucoes de invocacao para Codex, Gemini e Copilot |
| `scaffold` | Cria a estrutura inicial de uma nova skill de linguagem |
| `uninstall` | Remove artefatos instalados pelo CLI |
| `completion` | Gera scripts de autocompletion para shell |
| `version` | Exibe versao, commit e data de build |

### Exemplos uteis por comando

```bash
# instalar governanca em um projeto
ai-spec install ../api-pagamentos --source . --tools codex,claude --langs go

# inspecionar e diagnosticar
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos

# verificar governanca gerada
ai-spec lint ../api-pagamentos

# atualizar instalacao
ai-spec upgrade ../api-pagamentos --source . --langs go

# apenas checar se existe upgrade pendente
ai-spec upgrade ../api-pagamentos --source . --check

# validar todas as skills do repositorio fonte
ai-spec validate .agents/skills

# validar bugs.json contra bug-schema.json
ai-spec validate-bugs ./bugs.json

# checar pre-requisitos antes de rodar uma skill
ai-spec prerequisites create-tasks .

# medir custo de contexto em JSON
ai-spec metrics . --format json

# registrar telemetria de skill
GOVERNANCE_TELEMETRY=1 ai-spec telemetry log create-prd
ai-spec telemetry summary

# emitir instrucao pronta de wrapper para uma ferramenta
ai-spec wrapper codex create-tasks .

# criar scaffold para uma nova linguagem
ai-spec scaffold rust --root .

# executar todas as tasks elegiveis de um PRD folder
ai-spec task-loop --tool codex tasks/prd-payments-list

# verificar versoes de skills externas contra o lock file
ai-spec skills check .
ai-spec skills check . --force

# ver tendencia semanal de invocacoes de telemetria
ai-spec telemetry report --trend
ai-spec telemetry report --trend --format json

# inspecionar referencias carregadas por nivel de complexidade
ai-spec inspect . --brief
ai-spec inspect . --complexity=standard

# lint com verificacao estrita de invariantes de paridade
ai-spec lint . --strict

# remover a instalacao
ai-spec uninstall ../api-pagamentos --dry-run
```

## Exemplos por ferramenta

Depois que a governanca estiver instalada no repositorio alvo, cada ferramenta consome o baseline de forma um pouco diferente.

### Codex

Para validar pre-condicoes e emitir uma instrucao objetiva de uso:

```bash
ai-spec wrapper codex create-tasks .
ai-spec wrapper codex execute-task .
```

Exemplos de pedidos ao agente:

```text
Use a skill create-prd para criar o PRD desta feature a partir do contexto do repositorio.
```

```text
Use a skill execute-task para implementar a proxima task elegivel com validacao proporcional.
```

### Claude

Claude Code usa os artefatos instalados em `.claude/`, incluindo hooks e skills sincronizadas pelo projeto. O fluxo operacional continua o mesmo: pedir explicitamente a skill desejada dentro do repositorio instrumentado.

Exemplos de pedidos ao agente:

```text
Use a skill create-technical-specification com base no PRD aprovado e preserve a arquitetura existente.
```

```text
Use a skill review para revisar o diff atual com foco em regressao, risco e testes faltantes.
```

### Gemini

Gemini pode usar os comandos gerados em `.gemini/commands/`. Se quiser validar antes a governanca e os pre-requisitos:

```bash
ai-spec wrapper gemini create-tasks .
ai-spec wrapper gemini execute-task .
```

Exemplos de pedidos ao agente:

```text
Use a skill create-tasks para quebrar a tech spec em tasks pequenas, ordenadas e testaveis.
```

### GitHub Copilot

Copilot consome as instrucoes geradas em `.github/copilot-instructions.md` e os artefatos sob `.github/skills/`. O wrapper tambem pode ser usado para validar contexto antes da execucao:

```bash
ai-spec wrapper copilot execute-task .
ai-spec wrapper copilot review .
```

Exemplos de pedidos ao agente:

```text
Use a skill execute-task para implementar a task atual sem quebrar contratos publicos existentes.
```

## Operacao da instalacao

### Estrategias de instalacao

#### `symlink`

Melhor para desenvolvimento da governanca, porque o projeto alvo passa a refletir alteracoes feitas na fonte.

```bash
ai-spec install ../api-pagamentos \
  --source . \
  --tools all \
  --langs all \
  --mode symlink
```

#### `copy`

Melhor quando o ambiente nao lida bem com links simbolicos ou quando voce quer snapshot fisico do baseline.

```bash
ai-spec install ../api-pagamentos \
  --source . \
  --tools all \
  --langs all \
  --mode copy
```

## Para quem mantem este repositorio

### Desenvolvimento local

```bash
go test ./...
go run . --help
go run . install ../sandbox --source . --tools codex --langs go --dry-run
```

### Contribuicao

Issues e pull requests sao bem-vindos, especialmente para:

- novas skills de linguagem
- melhorias de adaptadores por ferramenta
- validacoes adicionais em `lint`, `doctor` e `metrics`
- exemplos de fluxos reais em repositorios Go, Node e Python

Antes de abrir PR, rode:

```bash
go test ./...
go run . validate .agents/skills
go run . lint .
```

### Roadmap curto

- melhorar a consistencia do nome do binario entre release e `go install`
- expandir exemplos por stack e por ferramenta
- adicionar mais fluxos canonicos orientados por task

## Referencias

### Releases

- Releases: <https://github.com/JailtonJunior94/orchestrator/releases>
- Homebrew Tap: <https://github.com/JailtonJunior94/homebrew-tap>

### Documentacao

- [Guia de resolucao de problemas](docs/troubleshooting.md) — 12 problemas comuns com sintoma, causa, solucao e verificacao
- [Telemetria e ciclo de feedback](docs/telemetry-feedback-cycle.md)
- [ADR 006 — Telemetria opt-in](docs/adr/006-telemetria-feedback-cycle.md)
- [ADR 007 — Copilot CLI stateless workaround](docs/adr/007-copilot-cli-stateless-workaround.md)
- [ADR 008 — Paridade multi-tool com invariantes](docs/adr/008-parity-multi-tool-invariants.md)

### Licenca

Este repositorio ainda nao expoe um arquivo `LICENSE` na raiz. Se a intencao e distribuicao open source publica, vale adicionar a licenca antes de ampliar o uso externo.
