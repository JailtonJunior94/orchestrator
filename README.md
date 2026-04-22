# ai-spec-harness

CLI em Go para instalar, validar, inspecionar e atualizar governanca operacional para ferramentas de IA em repositorios de software.

O nome do modulo Go e `ai-spec-harness`. O binario publicado via release se chama `ai-spec`. Este README usa `ai-spec` em todos os exemplos.

O projeto padroniza como Claude, Gemini, Codex e GitHub Copilot encontram skills, agentes, comandos e contexto de execucao dentro de um repositorio alvo. O foco nao e "conversar com um modelo" — e tornar fluxos repetidos como PRD, especificacao tecnica, decomposicao de tasks, review e execucao de tarefas mais previsiveis e auditaveis.

## Sumario

- [O que este projeto resolve](#o-que-este-projeto-resolve)
- [Como funciona](#como-funciona)
- [Instalacao](#instalacao)
  - [Requisitos](#requisitos)
  - [Homebrew — recomendado para macOS](#homebrew--recomendado-para-macos)
  - [Completion para bash e zsh](#completion-para-bash-e-zsh)
  - [Download direto](#download-direto)
  - [Instalacao via Go](#instalacao-via-go)
  - [Executar sem instalar](#executar-sem-instalar)
- [Inicio rapido](#inicio-rapido)
- [O que fazer depois da instalacao](#o-que-fazer-depois-da-instalacao)
  - [Cenario 1: projeto existente](#cenario-1-projeto-existente)
  - [Cenario 2: projeto novo](#cenario-2-projeto-novo)
- [Fluxo completo recomendado](#fluxo-completo-recomendado)
- [Artefatos de governanca](#artefatos-de-governanca)
- [Referencia rapida de comandos](#referencia-rapida-de-comandos)
- [Exemplos por ferramenta](#exemplos-por-ferramenta)
- [Operacao da instalacao](#operacao-da-instalacao)
- [Para quem mantem este repositorio](#para-quem-mantem-este-repositorio)
- [Referencias](#referencias)

## O que este projeto resolve

Sem uma estrutura canonica, cada repositorio tende a ter prompts soltos, instrucoes duplicadas, skills divergentes e pouca rastreabilidade sobre como agentes devem operar. O `ai-spec-harness` resolve isso ao:

- instalar um baseline de governanca em um projeto alvo
- distribuir skills compartilhadas por `symlink` ou copia
- gerar adaptadores por ferramenta para Claude, Gemini, Codex e Copilot
- validar `SKILL.md`, schema de bugs e artefatos de governanca
- inspecionar e diagnosticar instalacoes existentes
- medir custo estimado de contexto por baseline e fluxo
- criar scaffold para novas skills de linguagem

## Como funciona

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
- `finalize-changelog-readme-push`

## Instalacao

### Requisitos

- Go `1.26.2` ou compativel com o [`go.mod`](./go.mod)
- `git` disponivel no `PATH`
- permissao de escrita no projeto alvo
- um repositorio fonte de governanca contendo `.agents/skills`

### Homebrew — recomendado para macOS

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

Se o seu shell nao estiver herdando o `PATH` do Homebrew corretamente, adicione o prefixo ao arquivo de inicializacao e mantenha um alias compativel com o nome do modulo Go:

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
source ~/.zshrc   # ou source ~/.bashrc
```

> **Nota sobre nomes:** exemplos de release e do README usam `ai-spec`. A instalacao via `go install` gera o executavel `ai-spec-harness`. O alias acima evita alternar mentalmente entre os dois nomes.

### Completion para bash e zsh

#### Bash

Sessao atual:

```bash
source <(ai-spec completion bash)
```

Persistente no macOS com Homebrew:

```bash
ai-spec completion bash > "$(brew --prefix)/etc/bash_completion.d/ai-spec"
```

Via `~/.bashrc`:

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

Via `~/.zshrc`:

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

Binarios para `windows_amd64` e `windows_arm64` sao gerados a partir da v0.11.1.

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

```bash
go install github.com/JailtonJunior94/ai-spec-harness@latest
ai-spec-harness version
```

Durante desenvolvimento local neste checkout:

```bash
go install .
ai-spec-harness version
```

Para padronizar o nome com o binario de release, adicione um alias:

`~/.zshrc` / `~/.bashrc`

```bash
alias ai-spec="ai-spec-harness"
```

### Executar sem instalar

```bash
go run . --help
```

## Inicio rapido

Instale a governanca em um repositorio alvo usando este repositorio como fonte, depois valide:

```bash
ai-spec install ../api-pagamentos \
  --source . \
  --tools claude,gemini,codex,copilot \
  --langs go

ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
```

O repositorio alvo estara pronto para receber skills e agentes processuais.

## O que fazer depois da instalacao

### Cenario 1: projeto existente

Se o repositorio ja tem codigo, instale a governanca, valide e peca ao agente uma leitura arquitetural com `analyze-project`:

```bash
ai-spec install ../api-legado \
  --source . \
  --tools codex,claude,gemini,copilot \
  --langs go

ai-spec inspect ../api-legado
ai-spec doctor ../api-legado
cd ../api-legado
```

Prompt inicial sugerido para o agente:

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

### Cenario 2: projeto novo

Se o repositorio ainda esta vazio, nao existe arquitetura real para classificar. Comece pelo escopo do produto:

```bash
mkdir novo-produto && cd novo-produto && git init
cd ../ai-spec-harness

ai-spec install ../novo-produto \
  --source . \
  --tools codex,claude,gemini,copilot \
  --langs go

cd ../novo-produto
ai-spec inspect .
```

Prompt inicial sugerido para o agente:

```text
Use a skill create-prd para definir o primeiro escopo deste projeto novo.

Quero no resultado:
- problema
- objetivos e nao objetivos
- requisitos funcionais e nao funcionais
- riscos iniciais
```

Depois do PRD aprovado, o fluxo natural e:

```text
create-technical-specification -> create-tasks -> execute-task
```

## Fluxo completo recomendado

O `ai-spec-harness` nao escreve PRD, tech spec ou codigo por conta propria. Ele instala a governanca para que o agente escolhido execute cada etapa com as skills corretas dentro do repositorio alvo.

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

### 6. Criar a tech spec

```text
Use a skill create-technical-specification com base no PRD aprovado.
Carregue tambem as referencias necessarias de DDD e arquitetura.

Contexto tecnico:
- servico Go existente
- arquitetura atual: handler -> service -> repository
- preservar contratos publicos existentes

Quero no resultado:
- modelagem de dominio
- fronteiras entre aplicacao, dominio e infraestrutura
- estrategia de erros
- estrategia de testes
- riscos e plano de rollout
```

### 7. Gerar o bundle de tasks

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
    task-1.0-descricao.md
    task-2.0-descricao.md
    task-3.0-descricao.md
```

> **Convencoes de nome suportadas pelo task-loop:** `1-desc.md`, `1.0-desc.md` e `task-1.0-desc.md`. O separador pode ser `-` ou `_`.

### 8. Executar as tasks

O `task-loop` percorre `tasks.md`, identifica a proxima task elegivel e invoca o agente com `execute-task` ate concluir todas as tasks possiveis.

Valide sem gastar ciclo de agente:

```bash
ai-spec task-loop --tool codex --dry-run tasks/prd-payments-list
```

Execute o primeiro lote pequeno para observar qualidade:

```bash
ai-spec task-loop --tool codex --max-iterations 2 tasks/prd-payments-list
```

Execucao completa com rastreabilidade:

```bash
ai-spec task-loop \
  --tool codex \
  --max-iterations 10 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  tasks/prd-payments-list
```

> Consulte o [Guia do task-loop](docs/task-loop-reference.md) para flags detalhadas, modo avancado com executor e reviewer independentes, heuristicas e alternativas sem o loop automatico.

### 9. Validar o estado final

```bash
ai-spec lint .
go test ./...
```

> **Leitura recomendada:** o [Guia de uso das skills](docs/skills-usage-guide.md) detalha o contrato de cada skill — entradas obrigatorias, prompts mandatorios e criterios de aceite — para que voce possa reproduzir cada etapa com fidelidade maxima.

## Artefatos de governanca

Apos a instalacao, o repositorio alvo contem os seguintes artefatos que os agentes usam para carregar contexto, executar skills e registrar resultados:

| Artefato | Localizacao | Funcao |
| --- | --- | --- |
| `AGENTS.md` | raiz do repositorio alvo | fonte canonica de governanca: stack, convencoes, comandos de validacao e instrucoes para todos os agentes |
| `SKILL.md` | `.agents/skills/<skill-name>/SKILL.md` | define o contrato de uma skill: passos obrigatorios, entradas, saidas e criterios de aceite que o agente deve seguir |
| `tasks/<folder>/prd.md` | pasta de cada PRD | requisitos do produto aprovados; input obrigatorio para `create-technical-specification` |
| `tasks/<folder>/techspec.md` | pasta de cada PRD | especificacao tecnica aprovada; input obrigatorio para `create-tasks` e `execute-task` |
| `tasks/<folder>/tasks.md` | pasta de cada PRD | tabela de tasks com status e dependencias; consumida pelo `task-loop` |
| `bugs.json` | raiz ou pasta de qualidade | array JSON de bugs no schema canonico; validado por `ai-spec validate-bugs` |
| `skills-lock.json` | raiz do repositorio fonte | registra SHA-256 de cada skill externa; `ai-spec skills check` detecta mudancas de interface |
| `.ai_spec_harness.json` | raiz do repositorio alvo | manifesto da instalacao: versao, ferramentas e modo de instalacao |

> **Regra pratica:** sempre que um prompt ou exemplo mencionar `AGENTS.md` ou `SKILL.md`, o agente deve ler esses arquivos antes de qualquer acao. Eles sao a fonte de verdade — nao inferencias do historico de conversa.

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

```bash
ai-spec wrapper codex create-tasks .
ai-spec wrapper codex execute-task .
```

```text
Use a skill create-prd para criar o PRD desta feature a partir do contexto do repositorio.
```

```text
Use a skill execute-task para implementar a proxima task elegivel com validacao proporcional.
```

### Claude

Claude Code usa os artefatos instalados em `.claude/`, incluindo hooks e skills sincronizadas pelo projeto. O fluxo operacional continua o mesmo: pedir explicitamente a skill desejada dentro do repositorio instrumentado.

```text
Use a skill create-technical-specification com base no PRD aprovado e preserve a arquitetura existente.
```

```text
Use a skill review para revisar o diff atual com foco em regressao, risco e testes faltantes.
```

### Gemini

```bash
ai-spec wrapper gemini create-tasks .
ai-spec wrapper gemini execute-task .
```

```text
Use a skill create-tasks para quebrar a tech spec em tasks pequenas, ordenadas e testaveis.
```

### GitHub Copilot

```bash
ai-spec wrapper copilot execute-task .
ai-spec wrapper copilot review .
```

```text
Use a skill execute-task para implementar a task atual sem quebrar contratos publicos existentes.
```

## Operacao da instalacao

### `symlink`

Melhor para desenvolvimento da governanca, porque o projeto alvo passa a refletir alteracoes feitas na fonte.

```bash
ai-spec install ../api-pagamentos \
  --source . \
  --tools all \
  --langs all \
  --mode symlink
```

### `copy`

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

- [Guia de uso das skills](docs/skills-usage-guide.md) — contratos, prompts mandatorios e criterios de aceite por skill
- [Guia do task-loop](docs/task-loop-reference.md) — flags, heuristicas, alternativas e comparativos
- [Guia de resolucao de problemas](docs/troubleshooting.md) — 12 problemas comuns com sintoma, causa, solucao e verificacao
- [Telemetria e ciclo de feedback](docs/telemetry-feedback-cycle.md)
- [ADR 006 — Telemetria opt-in](docs/adr/006-telemetria-feedback-cycle.md)
- [ADR 007 — Copilot CLI stateless workaround](docs/adr/007-copilot-cli-stateless-workaround.md)
- [ADR 008 — Paridade multi-tool com invariantes](docs/adr/008-parity-multi-tool-invariants.md)
