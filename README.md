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
- [Instalacao em outros repositorios](#instalacao-em-outros-repositorios)
- [Inicio rapido](#inicio-rapido)
- [Fluxo mandatorio: SDD + Harness](#fluxo-mandatorio-sdd--harness)
- [O que fazer depois da instalacao](#o-que-fazer-depois-da-instalacao)
  - [Cenario 1: projeto existente](#cenario-1-projeto-existente)
  - [Cenario 2: projeto novo](#cenario-2-projeto-novo)
  - [Cenario 3: projeto ja instrumentado e desatualizado](#cenario-3-projeto-ja-instrumentado-e-desatualizado)
- [Nucleo operacional](#nucleo-operacional)
- [Runtime ACP (experimental)](#runtime-acp-experimental)
- [Fluxo completo recomendado](#fluxo-completo-recomendado)
- [Estrategia recomendada para desenvolver uma task](#estrategia-recomendada-para-desenvolver-uma-task)
- [Artefatos de governanca](#artefatos-de-governanca)
- [Referencia rapida de comandos](#referencia-rapida-de-comandos)
- [Exemplos por ferramenta](#exemplos-por-ferramenta)
- [Operacao da instalacao](#operacao-da-instalacao)
  - [Playbook reutilizavel: ciclo de vida completo da governanca](#playbook-reutilizavel-ciclo-de-vida-completo-da-governanca-qualquer-projeto)
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
- `execute-all-tasks`
- `review`
- `bugfix`
- `refactor`
- `finalize-changelog-readme-push`

### Camada de descoberta cirurgica (cross-CLI)

Alem dos artefatos acima, a instalacao entrega uma camada de **descoberta
cirurgica** que opera em cada `PreToolUse` dos 3 CLIs (Claude, Codex, Copilot):

1. **`references/INDEX.yaml`** em cada skill de linguagem (Go, Node, Python,
   .NET) mapeia cada reference a `file_patterns` e `diff_signals` — define
   *quando* carregar.
2. **`.agents/scripts/hook-prereq-gate.sh`** e o gate compartilhado invocado
   pelos hooks `validate-preload.sh` dos 3 CLIs. Orquestra:
   - **`validate-skill-prerequisites.sh`** — bloqueia o `PreToolUse` se a skill
     da extensao tocada nao esta acessivel (sem alucinacao por falta de
     descoberta).
   - **`resolve-references.sh`** — emite em `stderr` a lista exata de references
     a carregar para *aquela* edicao (so as que casam com path + diff).
3. Resultado pratico: **bloqueio inegociavel** quando a governanca esta
   incompleta, **guidance economica** quando esta completa (1–3 references por
   edit de codigo, **silencio total** em edits de docs/config).

Detalhes operacionais em [Instalacao em outros repositorios](#instalacao-em-outros-repositorios).

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

## Instalacao em outros repositorios

Esta secao explica como instalar e usar o harness em qualquer repositorio terceiro
(Go, Node/TypeScript, Python ou .NET/C#), com **gates inegociaveis cross-CLI**
(Claude, Codex, Copilot) e **carga cirurgica** de references — so o estritamente
necessario para a edicao em curso entra no contexto do agente.

### Passo 1: build do binario (uma vez por maquina)

```bash
cd /caminho/para/orchestrator
git pull
go build -o ai-spec .
sudo mv ai-spec /usr/local/bin/   # ou: cp ai-spec ~/bin/
```

### Passo 2: instalar no repo destino

```bash
cd /caminho/do/repo-destino
ai-spec install . --mode copy        # self-contained (recomendado)
# ou
ai-spec install . --mode symlink     # auto-atualiza ao seguir o orchestrator
```

O comando **auto-detecta**:

- Stack (Go, Node, Python, .NET) via manifesto na raiz (`go.mod`, `package.json`,
  `pyproject.toml`, `*.csproj`).
- CLIs presentes: Claude / Codex / Copilot por sinais de binario no PATH ou
  arquivos de projeto. Gemini e **opt-in por projeto** — so entra se ja houver
  `.gemini/` ou `GEMINI.md` no destino, ou se voce passar `--tools=gemini`/`all`.

Gera dentro do repo destino:

- `AGENTS.md`, `CLAUDE.md`, `.codex/config.toml`, `.github/copilot-instructions.md`
  com marcadores `<!-- ai-spec:generated-start/end -->` para **merge inteligente**
  em reinstalacoes (customizacoes fora dos marcadores sao preservadas; criacao de
  `.bak` no primeiro contato com arquivo sem marcadores).
- `.agents/skills/` — todas as skills + `references/INDEX.yaml` da linguagem
  detectada (mapa surgical por `file_patterns` e `diff_signals`).
- `.agents/scripts/` — 4 gates (`hook-prereq-gate.sh`, `resolve-references.sh`,
  `validate-skill-prerequisites.sh`, `validate-governance-references.sh`) e os 4
  validadores de evidencia.
- `.claude/hooks/`, `.codex/hooks/`, `.github/hooks/` — hooks `PreToolUse` ja
  cabeados ao gate compartilhado para os 3 CLIs.

### Passo 3: ativar PreToolUse no Claude Code (uma vez por repo)

Edite `.claude/settings.local.json` no repo destino:

```json
{
  "hooks": {
    "PreToolUse": [{
      "matcher": "Edit|Write|MultiEdit",
      "hooks": [{
        "type": "command",
        "command": "bash .claude/hooks/validate-preload.sh"
      }]
    }]
  }
}
```

Codex e Copilot ja vem prontos via `.codex/hooks.json` e `.github/hooks/governance.json`.

### Passo 4: sessao diaria

Antes da primeira edicao na sessao, confirme que voce leu `AGENTS.md`:

```bash
export GOVERNANCE_PRELOAD_CONFIRMED=1
```

A partir dai, **cada edicao de codigo** dispara automaticamente:

1. **Bloqueio inegociavel** se a skill da extensao tocada nao esta acessivel
   (`.go` exige `go-implementation/SKILL.md` + `references/INDEX.yaml`; idem para
   Node/Python/.NET). Mensagem clara aponta a skill ausente e o comando para
   corrigir.
2. **Guidance cirurgica em stderr** — lista exata das references a carregar para
   *aquela* edicao. Exemplo real (edit em `internal/repository/user.go` com diff
   contendo `db.QueryContext`):

   ```
   GUIDANCE (carga surgical): references a carregar para esta edicao:
   go-implementation/architecture          (always)
   go-implementation/examples-infrastructure  file_pattern:**/infrastructure/**
   go-implementation/persistence           diff_signal:db.QueryContext
   ```
3. **Silencio total** em edits de `.md`, `.yaml`, etc. — zero overhead, zero
   ruido no contexto. Validado: README.md em repo terceiro produz exit 0 e 0
   chars de saida.

### Passo 5: verificar a instalacao

```bash
ai-spec verify . --by-cli
```

Saida esperada:

```text
Resumo: 96 current, 0 missing, 0 drifted

Por CLI:
  claude     current=24 missing=0 drifted=0
  codex      current=24 missing=0 drifted=0
  copilot    current=24 missing=0 drifted=0
```

A flag `--by-cli` reporta drift por CLI separadamente — voce identifica se a
divergencia esta no espelho Claude, Codex ou Copilot sem ter que adivinhar.

### Passo 6: atualizar quando o orchestrator evolui

```bash
# 1. atualizar o binario
cd /caminho/para/orchestrator && git pull && go build -o ai-spec . \
  && sudo mv ai-spec /usr/local/bin/

# 2. reaplicar no repo destino (mesmo --mode usado antes)
cd /caminho/do/repo-destino && ai-spec install . --mode <copy|symlink>
```

### Variaveis de ambiente operacionais

| Variavel | Efeito | Quando usar |
|---|---|---|
| `GOVERNANCE_PRELOAD_CONFIRMED=1` | Confirma que carregou AGENTS.md na sessao | Rotina diaria |
| `PREREQ_MODE=warn` | Gate emite aviso mas nao bloqueia | Debug / pipelines legados |
| `GOVERNANCE_PRELOAD_MODE=warn` | Hook nao bloqueia por ausencia de preload | Debug |
| `AGENTS_ROOT=<dir>` | Override do project root para resolucao de skills | Testes E2E |
| `GATE_DIFF=<texto>` | Injeta diff manualmente no resolver | Smoke isolado |

### Troubleshooting

| Sintoma | Causa provavel | Solucao |
|---|---|---|
| `BLOQUEIO: skill obrigatoria nao acessivel` | Skill nao instalada ou `INDEX.yaml` ausente | `ai-spec install .` |
| Hook nao dispara em Edit/Write | `.claude/settings.local.json` sem o JSON do Passo 3 | Cole o JSON e reabra a sessao |
| `governanca nao carregada` em toda edicao | Falta `export GOVERNANCE_PRELOAD_CONFIRMED=1` | Exporte na sessao |
| `verify` reporta `drifted` | Edicao manual dentro dos marcadores ou skills divergiram | `ai-spec install .` para resincronizar |
| Quero remover tudo | — | `ai-spec uninstall .` |

### Resumo em 3 comandos para comecar agora

```bash
# 1. build (uma vez por maquina)
cd /caminho/para/orchestrator && go build -o /usr/local/bin/ai-spec .

# 2. instalar em qualquer repo
cd /caminho/do/repo-destino && ai-spec install . --mode copy

# 3. ativar na sessao
export GOVERNANCE_PRELOAD_CONFIRMED=1
```

Pronto — Claude/Codex/Copilot passam a operar com gates inegociaveis e guidance
cirurgica em cada `PreToolUse`.

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

## Fluxo mandatorio: SDD + Harness

Se voce quer manter um projeto sempre alinhado com cada versao publicada deste repositorio, siga este ciclo sem atalhos:

1. **SDD primeiro:** qualquer feature, alteracao de comportamento, novo endpoint, nova flag CLI ou mudanca de regra de negocio **DEVE** comecar em `create-prd`.
2. **Instale o binario `ai-spec`:** sem o binario atualizado, nao existe harness confiavel para instalar, validar ou sincronizar governanca.
3. **Instale o baseline no projeto:** todo repositorio alvo **DEVE** passar por `ai-spec install` antes de usar skills.
4. **Atualize o binario a cada release publicada:** saiu nova versao no GitHub Releases, atualize o `ai-spec`.
5. **Sincronize a governanca instalada em cada projeto:** depois de atualizar o binario, rode `ai-spec upgrade` no repositorio instrumentado.
6. **Revalide sempre:** toda instalacao ou sincronizacao **DEVE** terminar com `inspect`, `doctor` e `lint`.
7. **Sem `doctor`, sem confianca operacional:** se `doctor` falhar, trate a instalacao como nao confiavel ate corrigir manifesto, symlinks, permissoes ou estado do git.

### 1. SDD obrigatorio

Para mudanca de comportamento, o fluxo padrao e mandatorio:

```text
create-prd -> create-technical-specification -> create-tasks -> execute-task
```

Se houver lote maduro e paralelo seguro, troque apenas o ultimo passo por `execute-all-tasks`.

### 2. Harness e instalacao das skills

`install` e o comando que materializa o harness no repositorio alvo: cria `.agents/skills/`, adaptadores por ferramenta e `.ai_spec_harness.json`.

```bash
ai-spec install ../api-pagamentos \
  --source . \
  --tools codex,claude,gemini,copilot \
  --langs go

ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
```

Sem esse passo, nao existe baseline confiavel para Codex, Claude, Gemini ou Copilot descobrirem skills, agentes e regras.

### 3. Atualizacao obrigatoria do binario a cada release

Quando uma nova versao for publicada em <https://github.com/JailtonJunior94/orchestrator/releases>, atualize primeiro o executavel `ai-spec`.

Homebrew no macOS:

```bash
brew upgrade ai-spec
ai-spec version
```

Go:

```bash
go install github.com/JailtonJunior94/ai-spec-harness@latest
ai-spec-harness version
```

CI GitHub Actions:

```yaml
- uses: JailtonJunior94/orchestrator/.github/actions/setup-ai-spec@setup-action-v1
  with:
    version: latest
```

Com base nos commits mais recentes da `main`, esta Action e hoje o caminho recomendado para CI: no macOS instala via Homebrew; no Linux baixa o tarball oficial e valida `sha256` antes de expor o binario no `PATH`.

### 4. Sync obrigatorio da governanca instalada

Atualizar o binario nao basta. Depois de cada release publicada, sincronize cada repositorio ja instrumentado para aplicar skills, adaptadores e manifesto da nova versao.

```bash
ai-spec upgrade ../api-pagamentos --check
ai-spec upgrade ../api-pagamentos
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
```

Use `--check` como preflight. Se houver mudanca pendente, aplique o upgrade antes de executar novas tasks no projeto instrumentado.

Se voce estiver sincronizando a partir deste checkout local, pode manter a fonte explicita:

```bash
ai-spec upgrade ../api-pagamentos --source . --check
ai-spec upgrade ../api-pagamentos --source . --langs go
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
```

### 5. Doctor e gate de saude

`doctor` nao e opcional. Ele verifica repositorio git, symlinks, permissoes e manifesto da instalacao.

```bash
ai-spec doctor ../api-pagamentos
```

Rode `doctor` em tres momentos: logo apos `install`, logo apos `upgrade` e sempre que houver suspeita de drift local na governanca.

### 6. Regra operacional para ficar sempre atualizado

Checklist minimo por versao publicada:

1. Atualize o binario `ai-spec`.
2. Rode `ai-spec version` para confirmar a nova versao.
3. Rode `ai-spec upgrade <repo> --check` em cada repositorio instrumentado.
4. Rode `ai-spec upgrade <repo>` onde houver mudanca pendente.
5. Rode `ai-spec inspect <repo>`, `ai-spec doctor <repo>` e `ai-spec lint <repo>`.
6. Se houver PRD/TechSpec editado intencionalmente no bundle, rode tambem `ai-spec sync-spec-hash .specs/<prd>/tasks.md` para manter o `spec-hash` sincronizado.

Esse e o contrato pratico: **instalar uma vez, atualizar o binario a cada release, sincronizar cada projeto com `upgrade` e revalidar sempre**.

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
ai-spec lint ../api-legado
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
ai-spec doctor .
ai-spec lint .
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

### Cenario 3: projeto ja instrumentado e desatualizado

Se o repositorio ja usa `ai-spec`, mas voce publicou uma nova versao do harness ou das skills, o fluxo mandatorio e: atualizar o binario, sincronizar a governanca instalada e revalidar tudo.

Atualize primeiro o binario:

```bash
brew upgrade ai-spec
ai-spec version
```

Depois sincronize o repositorio instrumentado:

```bash
ai-spec upgrade ../api-legado --check
ai-spec upgrade ../api-legado
ai-spec inspect ../api-legado
ai-spec doctor ../api-legado
ai-spec lint ../api-legado
```

Se voce estiver publicando a atualizacao a partir deste checkout local, mantenha a fonte explicita:

```bash
ai-spec upgrade ../api-legado --source . --check
ai-spec upgrade ../api-legado --source . --langs go
ai-spec inspect ../api-legado
ai-spec doctor ../api-legado
ai-spec lint ../api-legado
```

So depois desse ciclo o repositorio deve voltar a executar novas tasks.

## Nucleo operacional

O `README.md` continua sendo a entrada executiva. O detalhe operacional agora fica concentrado nestes documentos:

- [Playbook Mestre de Desenvolvimento](docs/development-playbook.md): arvore de decisao para escolher entre `create-prd`, `create-technical-specification`, `create-tasks`, `execute-task`, `task-loop` e `execute-all-tasks`.
- [Scorecard de Qualidade e Confianca](docs/quality-scorecard.md): criterio objetivo para classificar o bundle em `pass`, `warning` ou `fail`.
- [Checklist de Preflight e Readiness](docs/preflight-checklist.md): gates bloqueantes e checks recomendados antes de executar.
- [Biblioteca de Prompts](docs/prompt-library.md): prompts copiaveis por etapa, com variante canonica para `execute-task`.
- [Matriz de Confiabilidade por Ferramenta](docs/tool-reliability-matrix.md): papel recomendado, risco operacional e custo de contexto por ferramenta.
- [Bundle canonico de referencia](.specs/prd-example-preflight-entrypoint/): exemplo auditavel end-to-end com `prd.md`, `techspec.md`, `tasks.md`, `task-*.md` e execution reports.

Atalho de decisao:

```text
Escopo ainda aberto? -> create-prd
Arquitetura e validacoes ainda em aberto? -> create-technical-specification
Precisa decompor em tasks pequenas? -> create-tasks
Vai executar uma task isolada ou medir qualidade real? -> execute-task
Tem lote pequeno com supervisao frequente? -> task-loop
Tem bundle maduro com DAG confiavel? -> execute-all-tasks
```

## Runtime ACP (cross-CLI)

O `ai-spec-harness` suporta um modo de execucao baseado no **Agent Client Protocol (ACP)**,
ativado pela flag `--runtime=acp`. Nesse modo, o harness abre uma sessao ACP com a CLI escolhida e
consome um stream de eventos em tempo real, em vez de aguardar um processo one-shot encerrar.

O runtime ACP funciona com **quatro CLIs** — `claude`, `codex`, `copilot` e `gemini` — com
comportamento equivalente (paridade): mesma normalizacao de tool-calls, mesmas metricas unificadas,
mesma memoria 2-tier, mesmo guard de governanca em runtime e os mesmos artefatos forenses.

### O que e

Em vez de invocar a CLI via `exec.Cmd` one-shot (modo `--runtime=legacy`, padrao), o modo ACP:

- Conecta a CLI via `github.com/coder/acp-go-sdk` (JSON-RPC sobre stdio).
- Recebe eventos granulares (`agent_message`, `agent_thought`, `tool_call_start`,
  `tool_call_update`, `session_end`) em tempo real.
- **Normaliza tool-calls por driver** preservando `raw_name`/`raw_input`: a mesma operacao produz o
  mesmo `normalized_name` nas 4 CLIs (ex.: claude `bash`, codex `shell`, copilot `run`, gemini
  `bash` -> todos `bash`; campo canonico `command`). Garantido pela suite `internal/parity` (RP-03)
  como gate de CI obrigatorio.
- Persiste eventos em `evidence/<task>/events.jsonl`, `tool_calls.md` e `execution_report.md`.
- Aplica o guard de governanca em `runtime.pre_open` (spec-hash/PRD-first): aborta antes da sessao
  se `tasks.md` tiver hash divergente/ausente. Bypass granular com `--skip-drift-guard`.
- Encerra a sessao com seguranca: watchdog de inatividade (`--activity-timeout`, padrao 120s),
  **cap absoluto de sessao** (5x o activity-timeout) e kill do grupo de processos no teardown —
  garante que a sessao nunca penda indefinidamente, mesmo se a CLI emitir keep-alives ou nao
  encerrar o turn (comportamento observado no codex-acp). Cancelamento registra
  `cancel_reason=activity_timeout` no `execution_report.md`.

### Permissao de escrita: `--access-mode full`

Por padrao (`--access-mode restricted`), o agente roda em modo restrito e **pede permissao** para
escrever arquivos — sem aprovacao, a sessao nao altera o repositorio. Para que o agente implemente de
fato, use `--access-mode full`. O harness traduz isso para cada CLI:

| CLI | `--access-mode full` aplica |
| --- | --- |
| claude | `--bypass-permissions` no claude-agent-acp |
| codex | `-c approval_policy=never -c sandbox_mode=danger-full-access` |
| gemini | `--approval-mode yolo` |
| copilot | auto-aprovacao das permissoes via ACP (Copilot nao tem flag de bypass dedicada) |

> **Aviso:** `--access-mode full` da ao agente acesso pleno ao filesystem (e a rede, no codex). Use
> apenas em ambiente isolado/confiavel.

### Como usar por ferramenta

Todos os comandos rodam dentro do repositorio alvo ja instrumentado (`ai-spec install`) e com um
PRD folder valido (`prd.md` + `techspec.md` + `tasks.md` com `spec-hash` sincronizado).

**Claude**

```bash
ai-spec task-loop --tool claude --runtime acp --access-mode full .specs/prd-<slug>
```
- Binario: `claude-agent-acp` no PATH, ou fallback `npx --yes @agentclientprotocol/claude-agent-acp@<pin>`.
- Auth: login do Claude (`~/.claude`) ou `ANTHROPIC_API_KEY`.

**Codex**

```bash
ai-spec task-loop --tool codex --runtime acp --access-mode full .specs/prd-<slug>
```
- Binario: `codex-acp` no PATH, ou fallback `npx --yes @zed-industries/codex-acp@<pin>`.
- Auth: login do Codex (`~/.codex`).

**Copilot**

```bash
ai-spec task-loop --tool copilot --runtime acp --access-mode full .specs/prd-<slug>
```
- Binario: `copilot --acp` no PATH, ou fallback `npx --yes @github/copilot@<pin> --acp`.
- Auth: `gh auth login` (conta GitHub com acesso ao Copilot).

**Gemini**

```bash
ai-spec task-loop --tool gemini --runtime acp --access-mode full .specs/prd-<slug>
```
- Binario: `gemini --acp` no PATH, ou fallback `npx --yes @google/gemini-cli@<pin> --acp`.
- Auth: login do Gemini CLI.

Parametros uteis (validos para qualquer CLI):

```bash
# uma task por vez, watchdog curto, modo silencioso
ai-spec task-loop --tool codex --runtime acp --access-mode full \
  --max-iterations 1 --activity-timeout 2m --quiet .specs/prd-<slug>

# bypass granular do guard spec-drift (mantem governance/token_budget ativos)
ai-spec task-loop --tool claude --runtime acp --access-mode full \
  --skip-drift-guard .specs/prd-<slug>
```

### Resolucao do binario (fallback)

Para cada CLI, o runtime resolve o launcher nesta ordem:

1. Binario direto no `PATH` (recomendado para producao).
2. Fallback `npx --yes <pacote>@<versao-pinada>` — requer `npx`/Node e internet no primeiro uso.
3. Falha com mensagem acionavel se nenhum dos dois estiver disponivel.

As versoes npm sao **pinadas** em `internal/runtime/specs/*.go` (nunca `@latest`); atualizacao via
processo `audit/`.

### Validacao end-to-end (evidencia real)

A paridade foi validada executando o harness de verdade contra um repositorio Go real
(`devkit-go`, feature isolada `pkg/strutil`), uma task por CLI via `--runtime acp --access-mode full`:

| CLI | Task | Resultado | Duracao |
| --- | --- | --- | --- |
| claude | 1.0 (Reverse) | `pending -> done`, exit 0 | ~2m48s |
| copilot | 2.0 (IsPalindrome) | `pending -> done`, exit 0 | ~3m21s |
| codex | 3.0 (WordCount) | `pending -> done`, exit 0 | ~3m27s |

Cada sessao: implementou o codigo (rune-aware, idiomatico), rodou os testes, gerou
`execution_report.md` + checkpoint, marcou a task `done` e **terminou de forma limitada** (sem hang,
sem processos orfaos).

### Notas importantes

- O modo `--runtime=legacy` permanece o padrao. Nenhuma task existente muda de comportamento sem
  mudanca explicita de flag.
- O guard `spec_drift` so atua quando ha `tasks.md` rastreavel; `tasks.md` ausente e no-op (o
  `TasksDir` tambem alimenta a memoria 2-tier sem exigir PRD).
- ADRs relacionadas: [ADR-009 (ACP via coder/acp-go-sdk)](.specs/adr/009-acp-protocol-adoption.md),
  [ADR-012 (Copilot ACP)](.specs/adr/012-copilot-cli-acp-native.md),
  [ADR-013 (Codex ACP)](.specs/adr/013-codex-cli-acp-native.md),
  [ADR-015 (Gemini ACP)](.specs/adr/015-gemini-cli-acp-native.md).
- Para migrar do modo legado, ver o [Guia de migracao legacy -> ACP](docs/migracao-legacy-acp.md).

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
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
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
.specs/
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
ai-spec task-loop --tool codex --dry-run .specs/prd-payments-list
```

Execute o primeiro lote pequeno para observar qualidade:

```bash
ai-spec task-loop --tool codex --max-iterations 2 .specs/prd-payments-list
```

Execucao completa com rastreabilidade:

```bash
ai-spec task-loop \
  --tool codex \
  --max-iterations 10 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  .specs/prd-payments-list
```

#### Flags disponiveis

**Modo Simples (Agente Unico)**
- `--tool`: agente unico (claude, codex, gemini, copilot)
- `--dry-run`: valida ordem e elegibilidade sem executar
- `--max-iterations`: limite de tasks por execucao (0 = sem limite)
- `--timeout`: tempo limite por task
- `--report-path`: caminho para o relatorio final em markdown

**Modo Avancado (Executor + Reviewer)**
- `--executor-tool`: ferramenta para implementacao (ex: codex)
- `--reviewer-tool`: ferramenta para revisao (ex: claude). Ativa fase de bugfix automatica se o reviewer falhar
- `--executor-model` / `--reviewer-model`: modelos especificos para cada papel
- `--reviewer-prompt-template`: path de template `.tmpl` customizado para o prompt de revisao
- `--executor-fallback-model` / `--reviewer-fallback-model`: modelos de fallback (suporte nativo Claude only)
- `--allow-unknown-model`: aceita combinacoes ferramenta+modelo fora do catalogo interno

#### Melhores praticas para o task-loop

1. **Validacao previa:** garanta que as tasks em `tasks.md` estejam pequenas, ordenadas e com dependencias claras antes de iniciar o loop.
2. **Ciclo de confianca:** comece sempre com `--dry-run`, seguido de um lote pequeno (`--max-iterations 2`). Aumente o lote apenas quando a qualidade estiver satisfatoria.
3. **Relatorios:** sempre use `--report-path` em features criticas para manter a rastreabilidade e facilitar o debug de falhas.
4. **Refinamento de spec:** se a task envolver riscos arquiteturais ou alta complexidade (ex: concorrencia), refine a `techspec.md` exaustivamente. O loop e mais eficiente quando a especificacao e inequivoca.
5. **Review automatico:** em tarefas sensiveis, utilize o Modo Avancado com um `reviewer-tool` diferente do `executor-tool` para aumentar a taxa de sucesso e detectar regressoes precocemente.
6. **Isolamento:** o `task-loop` garante isolamento total entre tarefas (novo processo por task). Nao tente carregar estado entre iteracoes; cada task deve ser auto-contida.

> Consulte o [Guia do task-loop](docs/task-loop-reference.md) para detalhes sobre heuristicas, alternativas e comparativos entre execucao manual e automatica.

### 9. Validar o estado final

```bash
ai-spec lint .
go test ./...
```

> **Leitura recomendada:** o [Guia de uso das skills](docs/skills-usage-guide.md) detalha o contrato de cada skill — entradas obrigatorias, prompts mandatorios e criterios de aceite — para que voce possa reproduzir cada etapa com fidelidade maxima.

## Estratégia de Desenvolvimento de Alta Performance

Este projeto utiliza um fluxo de governança rigoroso para garantir que cada linha de código seja justificada por um requisito e validada por testes. Para obter o melhor equilíbrio entre **eficiência**, **confiança** e **economia de tokens**, siga o protocolo abaixo.

### 1. O Caminho do Sucesso (Golden Path)

Para qualquer funcionalidade nova ou alteração complexa, o fluxo ideal é:

`create-prd` ➔ `create-technical-specification` ➔ `create-tasks` ➔ `execute-task`

| Etapa | Skill | Por que usar? | Ganho de Eficiência |
| :--- | :--- | :--- | :--- |
| **Produto** | `create-prd` | Remove ambiguidade de "o que" fazer. | Evita retrabalho por requisitos mal compreendidos. |
| **Arquitetura** | `create-technical-specification` | Define "como" fazer e antecipa riscos. | Garante consistência e evita dívida técnica precoce. |
| **Plano** | `create-tasks` | Divide o problema em fatias testáveis. | Facilita revisões pequenas e paralelas. |
| **Ação** | `execute-task` | Implementa com rigor e evidências. | Máxima confiança com validação e review embutidos. |

### 2. Maximizando a Confiança com Invariantes

A confiabilidade deste sistema baseia-se em **invariantes físicas**, não apenas em instruções textuais:
- **Spec-Hash:** O PRD gera um hash SHA-256. A TechSpec registra esse hash. As Tasks registram os hashes de ambos. Se você mudar o PRD, o `execute-task` detectará o "drift" e bloqueará a execução até que a cadeia seja atualizada.
- **Isolamento de Contexto:** Cada tarefa é executada com o mínimo de contexto necessário (Governance + Skill de Linguagem), reduzindo alucinações causadas por excesso de ruído no prompt.
- **Gate de Evidências:** Uma task só é marcada como `done` após a geração de um relatório de execução (`execution_report.md`) validado.

#### Comandos operacionais do `spec-hash`

A invariante só funciona se for verificada. Os dois comandos abaixo transformam o conceito em gate executável:

```bash
# 1. Detectar drift: bloqueia execução se PRD/TechSpec divergirem dos hashes em tasks.md
ai-spec check-spec-drift .specs/prd-<slug>/tasks.md
# Exit 0 = cadeia íntegra; ≠ 0 = drift detectado, execute-task bloqueia

# 2. Sincronizar após editar PRD ou TechSpec intencionalmente
ai-spec sync-spec-hash .specs/prd-<slug>/
# Recalcula SHA-256 de prd.md e techspec.md e atualiza os comentários em tasks.md
```

**Regra de ouro:** rode `check-spec-drift` antes de qualquer `execute-task` ou `execute-all-tasks`. Rode `sync-spec-hash` **somente** após decidir conscientemente que a edição no PRD/TechSpec deve invalidar tasks downstream — sincronizar sem reavaliar tasks reintroduz drift silencioso.

### 3. Economia e Escala: Qual execução escolher?

- **Use `execute-task` (Padrão):** Para tasks individuais, correções de bugs ou quando você quer acompanhar o agente de perto. É o método mais seguro e econômico para interações curtas.
- **Use `execute-all-tasks` (Escala):** Quando você tem um bundle maduro (PRD+TechSpec+Tasks) e quer disparar a implementação de múltiplas tarefas independentes em paralelo. Ele economiza seu tempo de supervisão e utiliza subagentes para manter a sessão principal limpa.

#### Custo de contexto medido (proxy de tokens)

Medição em 2026-05-18 sobre `SKILL.md` deste repositório usando a heurística oficial `chars/3.5`. Dados primários e metodologia em [`audit/sdd-token-baseline-2026-05-18.md`](audit/sdd-token-baseline-2026-05-18.md).

| Cenário | Skills ativas | Tokens estimados |
| :--- | :--- | ---: |
| Planejamento completo (PRD + TechSpec + Tasks) | 3 skills de planejamento | ~6.366 |
| Execução de 1 task (Go) | `agent-governance` + `go-implementation` + `execute-task` | ~6.383 |
| Execução de 1 task (Node) | `agent-governance` + `node-implementation` + `execute-task` | ~5.809 |
| Execução de 1 task (Python) | `agent-governance` + `python-implementation` + `execute-task` | ~5.775 |
| **Prompt direto sem SDD** (controle) | nenhuma skill, contexto re-explicado a cada iteração | **~6.000–12.000 por feature** |
| `execute-all-tasks` orquestrador (10 tasks) | só orquestração, subagent fresh por task | **≤1.000 no orquestrador** |

**Leitura:** o custo base do SDD é equivalente ao custo de um prompt direto **na primeira iteração**, mas o SDD não paga retrabalho (não recontextualiza a cada iteração) e mantém o orquestrador limpo em PRDs grandes via isolamento por subagent.

> Para evoluir esta medição com dados reais (não estimados), habilite telemetria: `export GOVERNANCE_TELEMETRY=1` e consulte o [Ciclo de feedback por telemetria](docs/telemetry-feedback-cycle.md).

### 4. Escape Hatch: quando o pipeline completo é overhead

O fluxo `create-prd → create-technical-specification → create-tasks → execute-task` é o padrão para **mudanças que alteram comportamento**. Para mudanças triviais, ele é overhead e o desvio é legítimo — desde que regras objetivas sejam respeitadas:

| Tipo de mudança | Sinal observável | Skill direta | Pipeline completo? |
| :--- | :--- | :--- | :---: |
| Bug isolado com causa-raiz reproduzível | ≤ 1–2 arquivos, sem RF novo, teste de regressão cabível | `bugfix` | Não |
| Refactor delimitado, mesmo contrato | Sem mudança em API pública/schema/CLI flags | `refactor` | Não |
| Renomeação local, typo, comentário | Zero impacto em comportamento | `execute-task` direto sem PRD | Não |
| Dep bump sem breaking change | `go.mod`/`package.json` apenas | `semantic-commit` direto | Não |
| Novo RF, novo endpoint, novo flag | Altera contrato público ou cria comportamento | — | **Sim, mandatório** |
| Mudança em ≥ 3 arquivos com lógica acoplada | Risco de regressão cruzada | — | **Sim, mandatório** |
| Mudança que reescreve invariante existente | Quebra premissa documentada em ADR/techspec | — | **Sim, mandatório** |

**Regra dura — quando o pipeline NÃO pode ser pulado:**
1. Qualquer alteração que adicione, remova ou modifique um Requisito Funcional (RF).
2. Qualquer alteração em contrato público (API HTTP, CLI flags, schema de banco, formato de arquivo).
3. Qualquer alteração que invalide um ADR existente (consultar [`.specs/adr/`](.specs/adr/) e [`docs/adr/`](docs/adr/)).

Em caso de dúvida, aplique o [Scorecard de Qualidade e Confiança](docs/quality-scorecard.md) — se o bundle ficaria em `fail` por ausência de PRD/TechSpec, o pipeline é mandatório.

### 5. Cheat Sheet — Qual skill usar agora?

Decisão rápida quando você sabe o que precisa fazer mas não sabe por onde começar:

| Situação | Skill recomendada | Por quê |
| :--- | :--- | :--- |
| Feature nova com ≥ 3 tasks ou novo RF | `create-prd` → `create-technical-specification` → `create-tasks` → `execute-all-tasks` | Rastreabilidade ponta-a-ponta com `spec-hash` |
| User Stories brutas como entrada | `us-to-prd` antes de `create-prd` | Converte história em RF numerado |
| Task única já planejada | `execute-task` | Sem overhead de orquestração |
| Bug isolado com sintoma reproduzível | `bugfix` | Causa-raiz + teste de regressão obrigatório |
| Refactor delimitado, sem novo comportamento | `refactor` | Preserva contrato e coleta evidência de não regressão |
| Diff pronto, falta revisar antes de merge | `review` | Scan determinístico contra regras do repositório |
| Comentários de PR para triar | `github-pr-comment-triage` | Prioriza e organiza antes de `bugfix`/`execute-task` |
| Projeto sem governança instalada | `analyze-project` | Detecta stack e instala baseline |
| Commit, changelog ou release pendente | `semantic-commit` → `github-diff-changelog-publisher` → `github-release-publication-flow` | Pipeline de entrega padronizado |

### 6. Documentos de apoio

Para reproduzir o fluxo com fidelidade máxima, consulte os guias especializados:

- [Estratégia SDD Cross-Project](docs/sdd-strategy.md) — como adotar este SDD em outros repositórios Go/Node/Python
- [Playbook Mestre de Desenvolvimento](docs/development-playbook.md) — decisão entre planejamento, execução unitária e em lote
- [Checklist de Preflight e Readiness](docs/preflight-checklist.md) — gates antes de chamar `execute-task` ou `execute-all-tasks`
- [Scorecard de Qualidade e Confiança](docs/quality-scorecard.md) — critério canônico para classificar prontidão de um bundle
- [Biblioteca de Prompts](docs/prompt-library.md) — prompts copiáveis com baixo desvio
- [Matriz de Confiabilidade por Ferramenta](docs/tool-reliability-matrix.md) — qual CLI usar em cada papel (Claude, Codex, Gemini, Copilot)
- [Guia de uso das skills](docs/skills-usage-guide.md) — contrato detalhado por skill
- [Ciclo de feedback por telemetria](docs/telemetry-feedback-cycle.md) — como evoluir o SDD com dados reais (`GOVERNANCE_TELEMETRY=1`)

---

## Protocolo Mandatório: Ciclo de Vida do PRD

O PRD não é um documento opcional; é a **âncora de verdade** de todo o desenvolvimento. O descumprimento deste protocolo invalida a governança e quebra as cadeias de confiança automatizadas.

### 1. Início Único e Obrigatório
Toda e qualquer feature, alteração de comportamento ou nova funcionalidade **DEVE** começar com a skill `create-prd`. 
- **Sem atalhos:** É proibido pular para a Especificação Técnica ou Implementação sem um PRD aprovado.
- **Localização:** O arquivo deve residir obrigatoriamente em `.specs/prd-<slug>/prd.md`.

### 2. Anatomia do PRD Objetivo
O PRD deve ser estritamente focado no **O QUE** e **POR QUE**, evitando detalhes de implementação:
- **Requisitos Funcionais (RF):** Devem ser numerados (RF-01, RF-02) para permitir o rastreio de cobertura (`check-spec-drift`).
- **Critérios de Aceite:** Devem ser binários e verificáveis.
- **Exclusões:** O que NÃO será feito é tão importante quanto o que será.

### 3. Imutabilidade e Sincronização de Drift
Uma vez que o PRD avança para TechSpec e Tasks, ele torna-se a base de um **hash SHA-256**.
- **Edição Pós-Aprovação:** Se o PRD for alterado, o sistema de governança detectará o "Drift de Especificação".
- **Ação Obrigatória:** Após editar um `prd.md`, você **DEVE** rodar `ai-spec sync-spec-hash` para atualizar os hashes em `tasks.md`. O `execute-task` recusará a execução se os hashes divergirem, garantindo que você nunca implemente algo baseado em uma versão obsoleta do requisito.

### 4. Encerramento
Um PRD só é considerado `done` quando todos os seus requisitos funcionais possuem evidências de teste vinculadas em relatórios de execução.

---

## Avaliação do Modelo de Desenvolvimento (SDD)

Avaliação ancorada nas cinco dimensões do [Scorecard de Qualidade e Confiança](docs/quality-scorecard.md), com peso total 100. Cada dimensão recebe `pass` (100% do peso), `warning` (50%) ou `fail` (0%) e é justificada por evidência verificável neste repositório.

| Dimensão | Peso | Status | Pontos | Evidência objetiva |
| :--- | :---: | :---: | :---: | :--- |
| Qualidade do PRD | 25 | `pass` | 25 | `create-prd` exige RF numerado, fora de escopo explícito e gate de drift downstream (`SKILL.md` Etapa 1.4) |
| Qualidade da Tech Spec | 25 | `pass` | 25 | `create-technical-specification` registra `spec-hash-prd` no cabeçalho; impede consumo de PRD divergente |
| Qualidade das Tasks | 20 | `pass` | 20 | `create-tasks` gera regex canônicos validados por `pre-execute-all-tasks.sh` (status, dependências, paralelizável) |
| Readiness de Execução | 20 | `pass` | 20 | Action `setup-ai-spec` com canal dual brew/curl instala `ai-spec` em CI automaticamente; hook opt-in valida spec-drift pré-commit |
| Evidência e Rastreabilidade | 10 | `pass` | 10 | `execution_report.md` + `_orchestration_report.md` + `DiffSHA` validado por `git cat-file` (F35) |
| **Total** | **100** | — | **100** | Faixa `pass` (85–100) — pronto para executar com risco controlado |

### Pilares estruturais que sustentam a nota

1. **Determinismo procedural:** o `spec-hash` SHA-256 ligando PRD → TechSpec → Tasks elimina a classe de erros "requisito esquecido" ou "implementação obsoleta" — `ai-spec check-spec-drift` bloqueia execução em caso de divergência.
2. **Defesa em profundidade:** o loop mandatório `Implementação → Validação → Review → Bugfix` garante que o código não apenas funcione, mas siga convenções do projeto.
3. **Escalabilidade agnóstica:** adaptadores finos por ferramenta (Claude, Gemini, Codex, Copilot) provam a robustez da lógica procedural sobre a sensibilidade do modelo.

### Histórico de atrito resolvido

- Readiness dependia de `ai-spec` no `PATH` do executor; a Action `setup-ai-spec` (canal dual brew/curl) e o hook pre-commit opt-in eliminaram a fricção de instalação manual em CI e local.
- Telemetria (`GOVERNANCE_TELEMETRY=1`) é opt-in; o ciclo de feedback descrito em [`docs/telemetry-feedback-cycle.md`](docs/telemetry-feedback-cycle.md) só evolui se o operador habilitar — decisão intencional de privacidade.

> **Veredito:** SDD classificado em `pass` operacional (100/100). É um dos modelos de governança mais seguros para desenvolvimento assistido por IA, priorizando **correção e rastreabilidade** sobre velocidade sem controle. A instalação automática via Action e o hook opt-in fecharam os últimos pontos de atrito operacional.

### Aplicar este SDD em outros projetos

O harness é portátil: `ai-spec install <projeto-alvo>` replica skills, adaptadores e invariantes em qualquer repositório Go, Node ou Python. Veja matriz de adoção e custos por arquétipo em [`docs/sdd-strategy.md`](docs/sdd-strategy.md).

## Artefatos de governanca

Apos a instalacao, o repositorio alvo contem os seguintes artefatos que os agentes usam para carregar contexto, executar skills e registrar resultados:

| Artefato | Localizacao | Funcao |
| --- | --- | --- |
| `AGENTS.md` | raiz do repositorio alvo | fonte canonica de governanca: stack, convencoes, comandos de validacao e instrucoes para todos os agentes |
| `SKILL.md` | `.agents/skills/<skill-name>/SKILL.md` | define o contrato de uma skill: passos obrigatorios, entradas, saidas e criterios de aceite que o agente deve seguir |
| `.specs/<folder>/prd.md` | pasta de cada PRD | requisitos do produto aprovados; input obrigatorio para `create-technical-specification` |
| `.specs/<folder>/techspec.md` | pasta de cada PRD | especificacao tecnica aprovada; input obrigatorio para `create-tasks` e `execute-task` |
| `.specs/<folder>/tasks.md` | pasta de cada PRD | tabela de tasks com status e dependencias; consumida pelo `task-loop` |
| `bugs.json` | raiz ou pasta de qualidade | array JSON de bugs no schema canonico; validado por `ai-spec validate-bugs` |
| `skills-lock.json` | raiz do repositorio fonte | registra SHA-256 de cada skill externa; `ai-spec skills check` detecta mudancas de versao e `ai-spec skills --verify` e o gate de integridade (versao + hash) |
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
| `skills --verify` | Gate de integridade: falha (exit != 0) se versao ou hash SHA-256 divergirem de `skills-lock.json`; usado por `execute-task`/`execute-all-tasks` |
| `validate` | Valida frontmatter YAML de `SKILL.md` |
| `validate-bugs` | Valida um array JSON de bugs contra o schema canonico |
| `prerequisites` | Verifica se uma skill pode ser executada em um projeto |
| `check-spec-drift` | Verifica cobertura de IDs `RF-nn`/`REQ-nn` do PRD em `tasks.md` e detecta divergencia de hash entre `prd.md`/`techspec.md` e os hashes registrados |
| `sync-spec-hash` | Recalcula SHA-256 de `prd.md` e `techspec.md` e atualiza os comentarios `spec-hash` em `tasks.md`; usar apos editar o PRD para evitar bloqueio do `task-loop` |
| `task-loop` | Executa todas as tasks elegiveis de um PRD folder via agente de IA |
| `wrapper` | Emite instrucoes de invocacao para Codex, Gemini e Copilot |
| `scaffold` | Cria a estrutura inicial de uma nova skill de linguagem |
| `uninstall` | Remove artefatos instalados pelo CLI |
| `completion` | Gera scripts de autocompletion para shell |
| `version` | Exibe versao, commit e data de build; `--skills` lista versoes das skills embutidas e instaladas |
| `skill-bump` | Detecta skills alteradas desde a ultima tag e atualiza o campo `version` no frontmatter `SKILL.md` |

### Exemplos uteis por comando

```bash
# instalar governanca em um projeto
ai-spec install ../api-pagamentos --source . --tools codex,claude --langs go

# inspecionar e diagnosticar
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos

# verificar governanca gerada
ai-spec lint ../api-pagamentos

# verificar instalacao com resumo por-CLI (paridade Claude/Codex/Copilot)
ai-spec verify ../api-pagamentos --by-cli

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

# verificar cobertura de requisitos e drift de spec
ai-spec check-spec-drift .specs/prd-payments-list/tasks.md

# sincronizar hashes apos editar prd.md ou techspec.md
ai-spec sync-spec-hash .specs/prd-payments-list/tasks.md

# executar todas as tasks elegiveis de um PRD folder
ai-spec task-loop --tool codex .specs/prd-payments-list

# verificar versoes de skills externas contra o lock file
ai-spec skills check .
ai-spec skills check . --force

# gate de integridade (versao + hash SHA-256) — bloqueante
ai-spec skills --verify .

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

# listar versoes de todas as skills (embutidas e instaladas)
ai-spec version --skills

# listar apenas skills embutidas ou apenas instaladas
ai-spec version --skills=embedded
ai-spec version --skills=installed

# detectar skills alteradas desde a ultima tag e propor bump de versao
ai-spec skill-bump .
ai-spec skill-bump . --dry-run
```

## Exemplos por ferramenta

Depois que a governanca estiver instalada no repositorio alvo, cada ferramenta consome o baseline de forma um pouco diferente.

### Padrao de prompt efetivo

Para reduzir desvio, o prompt deve sempre ter 4 blocos:

```text
1. skill explicita
2. contexto objetivo e verificavel
3. restricoes reais do repositorio
4. saida esperada
```

```text
Template:

Use a skill <skill>.

Contexto:
- <fatos concretos>
- <arquivos, contratos ou restricoes>

Quero no resultado:
- <artefatos ou decisoes esperadas>
- <criterios de qualidade>
```

Evite:

- "faca o melhor possivel"
- "analise tudo"
- "implemente completo" sem delimitar escopo
- pedir PRD, tech spec e execucao no mesmo prompt

### Codex

```bash
ai-spec wrapper codex create-prd .
ai-spec wrapper codex create-technical-specification .
ai-spec wrapper codex create-tasks .
ai-spec wrapper codex execute-task .
```

Prompt efetivo para `create-prd`:

```text
Use a skill create-prd.

Contexto:
- queremos adicionar listagem de pagamentos
- endpoint desejado: GET /payments
- filtros: status, pagina, data inicial e final
- usuarios principais: operacao e backoffice
- nao decidir detalhes de implementacao nesta etapa

Quero no resultado:
- problema
- objetivos e nao objetivos
- requisitos funcionais numerados
- restricoes e criterios de sucesso
- suposicoes em aberto, se existirem
```

Prompt efetivo para `execute-task`:

```text
Use a skill execute-task para implementar a proxima task elegivel.

Contexto:
- execute apenas a task selecionada em .specs/prd-payments-list/
- preserve contratos publicos existentes
- rode validacao proporcional e review

Quero no resultado:
- implementacao concluida ou status bloqueante explicito
- testes e validacoes executadas
- caminho do execution report
```

### Claude

Claude Code usa os artefatos instalados em `.claude/`, incluindo hooks e skills sincronizadas pelo projeto. O melhor resultado vem de prompts curtos, com uma skill por vez.

Prompt efetivo para `create-technical-specification`:

```text
Use a skill create-technical-specification com base no PRD aprovado.

Contexto tecnico:
- servico Go existente
- arquitetura atual: handler -> service -> repository
- preservar contratos publicos existentes
- explicitar erros, riscos e estrategia de testes

Quero no resultado:
- desenho de implementacao
- fronteiras de dominio e interfaces
- trade-offs e ADRs necessarias
- estrategia de validacao
```

Prompt efetivo para `review`:

```text
Use a skill review para revisar o diff atual.

Contexto:
- priorize regressao, risco, seguranca e testes faltantes
- considere o PRD e a tech spec da feature atual

Quero no resultado:
- findings ordenados por severidade
- arquivos e linhas afetadas
- riscos residuais, se nao houver blockers
```

### Gemini

```bash
ai-spec wrapper gemini create-tasks .
ai-spec wrapper gemini execute-task .
```

Prompt efetivo para `create-tasks`:

```text
Use a skill create-tasks.

Contexto:
- o PRD e a tech spec da feature ja estao aprovados
- queremos tasks pequenas, independentes e testaveis
- priorizar ordem segura de entrega

Quero no resultado:
- proposta inicial com ate 10 tasks
- dependencias explicitas
- indicacao de paralelismo seguro
- arquivos finais tasks.md e task-*.md apos aprovacao
```

### GitHub Copilot

```bash
ai-spec wrapper copilot execute-task .
ai-spec wrapper copilot review .
```

Prompt efetivo para `execute-task`:

```text
Use a skill execute-task para implementar a task atual.

Contexto:
- mantenha o escopo restrito ao task file selecionado
- nao altere comportamento publico fora do que a task exigir
- gere evidencias e atualize o status somente apos validacao

Quero no resultado:
- codigo e testes da task
- resumo das validacoes executadas
- status final canonico: done, blocked, failed ou needs_input
```

### Regra pratica para qualquer ferramenta

Se a intencao for implementar **uma** task, o prompt mais eficiente costuma ser:

```text
Use a skill execute-task para implementar a proxima task elegivel com validacao proporcional.
```

Se a intencao for preparar o terreno antes de codificar, use um prompt por etapa:

```text
create-prd -> create-technical-specification -> create-tasks -> execute-task
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

### Pre-commit hook opt-in (spec-drift, skills-sync, hooks-sync)

O repositório oferece um hook `pre-commit` em `scripts/git-hooks/pre-commit` que valida três invariantes antes de cada commit:

1. **skills-sync**: garante que `.agents/skills/` e mirrors (`.claude/skills/`, etc.) estão sincronizados.
2. **hooks-sync**: garante que `.claude/hooks/` (canônico) e mirrors estão sincronizados.
3. **spec-drift**: executa `ai-spec check-spec-drift` quando `prd.md` ou `techspec.md` de algum bundle aparecem em arquivos staged.

Ativação (opt-in):

```bash
git config core.hooksPath scripts/git-hooks
```

Comportamento permissivo: se `ai-spec` não está no PATH ou está em versão antiga, o bloco spec-drift emite warning em stderr e permite o commit (exit 0). Em caso de drift real, o hook bloqueia com mensagem indicando `ai-spec sync-spec-hash .specs/prd-<slug>/tasks.md` como remediação.

Escape hatch responsável: `git commit --no-verify` pula o hook em casos legítimos (commits de docs em PRDs antigos, por exemplo). Drift fica para review pegar — não normalize o bypass.

### Caso real: atualizar `financialcontrol-api`

Em 2026-05-24, o projeto `/Users/jailtonjunior/Git/financialcontrol-api` foi atualizado a partir dos commits recentes deste repositorio. O objetivo foi aplicar o baseline atual do harness em outro projeto Go e validar a instalacao como evidencia operacional.

Commits usados como referencia:

- `7dbdc0b fix(sync): nao marcar skills/lib como read-only (quebrava o git)`
- `833bdd1 refactor(sdd): migra root de artefatos de tasks/ para .specs/`
- `45e429c fix(fs): copia/remove de assets read-only (source imutavel) + cobertura execute-all-tasks`
- `40f9537 docs(readme): runtime ACP cross-CLI — processo + uso por ferramenta (claude/codex/copilot/gemini)`

O processo atualizou skills de governanca para Claude, Codex, Copilot e Gemini, adicionou `execute-all-tasks`, trocou comandos Gemini antigos para o namespace `workspace.*`, instalou hooks compartilhados e substituiu o symlink legado de `.agents/skills` por uma copia local versionavel.

Comandos reais utilizados:

```bash
cd /Users/jailtonjunior/Git/orchestrator
git log --oneline -n 12
git show --stat --oneline --decorate -n 1 7dbdc0b
git show --stat --oneline --decorate -n 1 833bdd1
git show --stat --oneline --decorate -n 1 45e429c
go run . inspect /Users/jailtonjunior/Git/financialcontrol-api
go run . verify /Users/jailtonjunior/Git/financialcontrol-api --tools all --langs go
go run . install /Users/jailtonjunior/Git/financialcontrol-api --tools all --langs go --dry-run
go run . install /Users/jailtonjunior/Git/financialcontrol-api --tools all --langs go --mode copy
rm /Users/jailtonjunior/Git/financialcontrol-api/.agents/skills
go run . install /Users/jailtonjunior/Git/financialcontrol-api --tools all --langs go --mode copy
go run . verify /Users/jailtonjunior/Git/financialcontrol-api --tools all --langs go
```

Resultado de validacao do harness:

```text
Resumo: 96 current, 0 missing, 0 drifted
```

Observacao operacional: a primeira instalacao em `copy` manteve o symlink legado de `.agents/skills` para `/Users/jailtonjunior/Git/ai-governance/.agents/skills`. O link foi removido e a instalacao repetida para garantir que `.agents/skills` passasse a ser um diretorio local com assets atuais.

#### Validacao com `ai-spec` instalado

Depois da atualizacao pelo checkout local, a instalacao tambem foi validada com o binario instalado no sistema para confirmar o caminho operacional usado fora do desenvolvimento do harness.

Comandos reais utilizados:

```bash
cd /Users/jailtonjunior/Git/orchestrator
command -v ai-spec
ai-spec version
ai-spec inspect /Users/jailtonjunior/Git/financialcontrol-api
ai-spec verify /Users/jailtonjunior/Git/financialcontrol-api --tools all --langs go
```

Resultado:

```text
/opt/homebrew/bin/ai-spec
ai-spec-harness 0.23.2 (commit: e5a833a3c3326fefdf40458dc882031ba4daedcb, built: 2026-05-23T22:25:05Z)
Resumo: 96 current, 0 missing, 0 drifted
```

### Playbook reutilizavel: ciclo de vida completo da governanca (qualquer projeto)

Runbook ponta a ponta para **instalar, atualizar ou reinstalar** a governanca em qualquer projeto, com cada comando explicado. Validado no fluxo real de reinstalacao do `financialcontrol-api` (baseline `dev` -> `0.23.2`).

> **Convencao:** defina duas variaveis e reaproveite em todos os comandos.
>
> ```bash
> PROJETO=/caminho/do/projeto-alvo      # ex.: /Users/voce/Git/minha-api
> FONTE=.                               # repo da governanca (este). Use "." se ja estiver nele,
>                                       # ou o caminho absoluto: /Users/voce/Git/orchestrator
> ```

#### Pre-requisitos

```bash
command -v ai-spec          # confirma que o binario esta no PATH
ai-spec version             # confirma a versao (ex.: ai-spec-harness 0.23.2)
```

(macOS: se aparecer _"Apple could not verify"_, veja a secao [Instalacao](#instalacao).)

---

#### 🔄 Como ATUALIZAR (o caminho recomendado para o dia a dia)

Para manter um projeto sempre no baseline mais novo **sem perder customizacoes**, use `upgrade` — nao `uninstall`. Ha duas coisas que podem ficar desatualizadas:

**1. Atualizar o binario `ai-spec` (a ferramenta em si):**

```bash
brew update && brew upgrade ai-spec     # se instalado via Homebrew
ai-spec version                         # confirme a nova versao
```

(Outras formas de instalar/atualizar o binario estao na secao [Instalacao](#instalacao).)

**2. Atualizar a governanca dentro de um projeto:**

```bash
# a) Primeiro, SO VERIFIQUE o que mudaria (nao escreve nada):
ai-spec upgrade $PROJETO --source $FONTE --langs go --check

# b) REVISE a saida antes de aplicar (veja os avisos abaixo), depois aplique:
ai-spec upgrade $PROJETO --source $FONTE --langs go

# c) Valide imediatamente (upgrade so e confiavel apos doctor verde):
ai-spec doctor  $PROJETO                      # DEVE: "Resultado: tudo ok"
ai-spec verify  $PROJETO --source $FONTE      # DEVE: 0 drifted (skills instaladas fieis a fonte)
ai-spec lint    $PROJETO                      # DEVE: "Lint aprovado"
```

| Quando usar | Comando |
| --- | --- |
| Atualizar mantendo a instalacao existente (rotina) | `ai-spec upgrade $PROJETO --source $FONTE --langs go` |
| So checar o que mudaria (CI, pre-commit) | `ai-spec upgrade $PROJETO --source $FONTE --langs go --check` |
| Reset total / baseline do zero (corrigir instalacao corrompida) | `uninstall` + `install` (passos abaixo) |

> ⚠️ **SEMPRE passe `--source` no `upgrade`.** Sem `--source`, o `upgrade` compara contra os assets **embutidos no binario** (a release instalada). Se o projeto foi instalado de uma fonte mais nova que a release, rodar `upgrade` sem `--source` **rebaixa (downgrade)** as skills para a versao embutida. Use sempre `--source $FONTE` apontando para a mesma fonte do `install`.

> ⚠️ **`upgrade --check` e `verify` respondem perguntas DIFERENTES — nao confunda.**
> - `verify $PROJETO --source $FONTE` audita as skills **ja instaladas** vs a fonte → `0 drifted` confirma fidelidade. **Esta e a porta de saude.**
> - `upgrade --check --source $FONTE` compara o **conjunto completo de skills da fonte** com o projeto e pode listar como `AUSENTE` skills que existem na fonte mas que o `install` **nao instala** (ex.: `finalize-changelog-readme-push` — skills internas/de mantenedor deste repo). Ou seja, **`upgrade --check` pode acusar "N ausentes" mesmo numa instalacao recem-feita e saudavel.** Nao trate isso como erro automaticamente: **revise a lista** — aplicar o `upgrade` ADICIONARIA essas skills ao projeto, o que pode nao ser desejado para um projeto de aplicacao.

> ✅ **Regra de ouro:** `upgrade --source $FONTE` para o dia a dia (preserva o que ja existe); `uninstall + install` apenas quando precisar de um baseline limpo do zero. A confirmacao final de saude e **`doctor` verde + `verify --source` com `0 drifted`** — nao o `upgrade --check`.

---

#### Passo 0 — Auditoria (read-only, nao altera nada)

Comece fotografando o estado atual. Nenhum destes comandos escreve no projeto.

```bash
ai-spec inspect $PROJETO                 # manifesto, skills, toolchain detectado
ai-spec doctor  $PROJETO                 # saude: git, manifesto, symlinks, permissoes
ai-spec lint    $PROJETO                 # placeholders, schema, SKILL.md
ai-spec verify  $PROJETO                 # current / missing / drifted por skill
ai-spec uninstall $PROJETO --dry-run     # PREVE o que seria removido (nao remove)
```

| Comando | Pergunta que responde |
| --- | --- |
| `inspect` | O que esta instalado? Qual versao, modo, tools, langs? |
| `doctor` | A instalacao esta saudavel? |
| `lint` | Os arquivos de governanca sao validos? |
| `verify` | Cada skill esta `current`, `missing` ou `drifted`? |
| `uninstall --dry-run` | O que exatamente seria apagado numa desinstalacao? |

#### Passo 1 — Desinstalacao cirurgica (so para reset do zero)

`uninstall` remove **apenas** os arquivos rastreados no manifesto (`.ai_spec_harness.json`). Ele **nao** apaga conteudo seu como `.claude/settings.local.json`.

```bash
ai-spec uninstall $PROJETO --dry-run     # 1) confira a lista
ai-spec uninstall $PROJETO               # 2) execute de verdade
```

Confira o resultado e a preservacao do que e seu:

```bash
test -f $PROJETO/.ai_spec_harness.json && echo "manifesto presente" || echo "manifesto removido"
test -f $PROJETO/.claude/settings.local.json && echo "settings preservado" || echo "ATENCAO: settings sumiu"
```

> ⚠️ **Nunca use `rm -rf .claude .gemini .agents`.** Esses diretorios podem conter conteudo seu (`settings.local.json`, agents/rules proprios) que o `uninstall` preserva de proposito. Remova manualmente apenas symlinks orfaos comprovadamente da governanca (links quebrados apontando para `.agents/skills/<skill-inexistente>`), e nunca arquivos commitados que voce nao criou.

#### Passo 2 — Instalacao limpa (so para reset do zero)

```bash
ai-spec install $PROJETO \
  --source $FONTE \      # repo de governanca como fonte de verdade
  --tools all \          # claude,gemini,codex,copilot (ou subconjunto)
  --langs go \           # go,node,python ou all (skills de linguagem)
  --mode copy \          # copy (auto-contido) ou symlink (reflete a fonte)
  --dry-run              # remova esta linha para executar de verdade
```

Flags principais do `install`:

| Flag | Valores | Para que serve |
| --- | --- | --- |
| `--source` | caminho | Repo de governanca usado como fonte. Sem ele, usa os assets embutidos no binario. |
| `--tools` | `claude,gemini,codex,copilot` ou `all` | Quais adaptadores gerar. Sem a flag, **auto-detecta** por binario no PATH + dirs de config. |
| `--langs` | `go,node,python` ou `all` | Skills de linguagem a incluir. |
| `--mode` | `copy` \| `symlink` | `copy` = snapshot fisico, projeto auto-contido (recomendado p/ outros projetos). `symlink` = reflete mudancas da fonte (bom p/ desenvolver a governanca). |
| `--dry-run` | — | Simula sem escrever. |
| `--global` | — | Instala em `~/.aispec` (escopo global). |
| `--ref` | tag/branch/SHA | Usa um ref git da fonte como base. |
| `--codex-profile` | `full` \| `lean` | Perfil de skills do Codex. |

#### Passo 3 — Validacao (gate de "100% funcional")

```bash
ai-spec inspect $PROJETO
ai-spec doctor  $PROJETO                  # DEVE terminar com "Resultado: tudo ok"
ai-spec lint    $PROJETO                  # DEVE terminar com "Lint aprovado"
ai-spec verify  $PROJETO --source $FONTE  # DEVE dar 0 missing, 0 drifted
```

So considere concluido quando `doctor` estiver verde.

> 🔎 **Por que `verify` sem `--source` pode mostrar "drifted"?** Sem `--source`, o `verify` compara contra os assets **embutidos no binario** (a release instalada). Se voce instalou de uma fonte mais nova que a release (ex.: working tree a frente da tag), as skills mais recentes aparecem como `drifted` — o que e **esperado e correto**. Para validar contra a fonte real da instalacao, sempre rode `verify --source $FONTE`. Ali, `0 drifted` confirma fidelidade.

#### Passo 4 — Conectar os hooks (Claude Code)

O instalador **nao sobrescreve** um `.claude/settings.local.json` existente, entao os hooks `validate-preload` (PreToolUse) e `validate-governance` (PostToolUse) precisam ser adicionados manualmente. Faca merge do bloco `hooks`, preservando suas `permissions`:

```json
{
  "permissions": { "allow": ["..."] },
  "hooks": {
    "PreToolUse": [
      { "matcher": "Edit|Write", "hooks": [ { "type": "command", "command": "bash .claude/hooks/validate-preload.sh" } ] }
    ],
    "PostToolUse": [
      { "matcher": "Edit|Write", "hooks": [ { "type": "command", "command": "bash .claude/hooks/validate-governance.sh" } ] }
    ]
  }
}
```

Valide o JSON e faca um smoke test dos scripts:

```bash
python3 -m json.tool $PROJETO/.claude/settings.local.json >/dev/null && echo "JSON ok"
( cd $PROJETO && bash .claude/hooks/validate-preload.sh </dev/null;    echo "preload exit=$?" )
( cd $PROJETO && bash .claude/hooks/validate-governance.sh </dev/null; echo "governance exit=$?" )
```

> ⚠️ **`.claude/settings.local.json` costuma estar no `.gitignore`** (arquivo local por desenvolvedor). Nesse caso o bloco `hooks` **nao entra no commit** e **cada desenvolvedor precisa adiciona-lo na sua copia**. Se quiser hooks compartilhados via git, mova-os para um `.claude/settings.json` versionado.

#### Passo 5 — Documentar no projeto alvo

Adicione uma secao **"AI Governance"** no README do projeto alvo, registrando: data do baseline, versao do `ai-spec`, modo/tools/langs e como invocar skills (`create-prd`, `execute-task`, `review`, etc.). Isso da rastreabilidade ao time.

#### Passo 6 — Commit (sem push automatico)

```bash
git -C $PROJETO add -A
git -C $PROJETO commit -m "chore(governance): atualiza baseline ai-spec <versao>"
# push fica a seu criterio: git -C $PROJETO push
```

Nunca rode git destrutivo (`reset --hard`, `clean -fd`, force-push) durante o ciclo. Lembre: arquivos gitignored (como `settings.local.json`) nao serao incluidos.

#### Referencia rapida do ciclo (copia e cola)

```bash
PROJETO=/caminho/do/projeto-alvo
FONTE=.

# === ATUALIZAR (rotina) ===
ai-spec upgrade $PROJETO --source $FONTE --check     # ha update?
ai-spec upgrade $PROJETO --source $FONTE --langs go  # aplica
ai-spec doctor  $PROJETO                             # tudo ok
ai-spec verify  $PROJETO --source $FONTE             # 0 missing, 0 drifted

# === RESET DO ZERO (quando necessario) ===
ai-spec uninstall $PROJETO --dry-run
ai-spec uninstall $PROJETO
ai-spec install   $PROJETO --source $FONTE --tools all --langs go --mode copy
ai-spec doctor    $PROJETO
ai-spec lint      $PROJETO
ai-spec verify    $PROJETO --source $FONTE

# === COMMIT ===
git -C $PROJETO add -A && git -C $PROJETO commit -m "chore(governance): baseline ai-spec"
```

#### Armadilhas comuns

| Sintoma | Causa provavel | Acao |
| --- | --- | --- |
| `verify` mostra `drifted`, mas `--source` da `0 drifted` | Fonte a frente da release embutida | Esperado; valide com `--source` |
| `doctor` falha em "Symlinks de skills" | Symlinks orfaos/quebrados | Reinstale; remova so links comprovadamente orfaos |
| Warning "settings.local.json ja existe" no install | Arquivo preservado de proposito | Conecte os hooks manualmente (Passo 4) |
| Hooks nao entram no commit | `settings.local.json` gitignored | Use `.claude/settings.json` versionado se quiser compartilhar |
| `doctor` falha em "Repositorio git" | Projeto sem `git init` | Rode `git init` no projeto alvo |
| `upgrade --check` lista `AUSENTE` numa instalacao recem-feita | Fonte tem skills internas/de mantenedor que o `install` nao instala (ex.: `finalize-changelog-readme-push`) | Esperado; revise a lista. A saude e `doctor` verde + `verify --source 0 drifted`, nao `upgrade --check` |
| `upgrade` rebaixou (downgrade) as skills | Rodou `upgrade` sem `--source` (comparou com embedded) | Sempre use `upgrade --source $FONTE`; reinstale a partir da fonte para corrigir |

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

### Criar/mover tag `setup-action-v1`

A Action `setup-ai-spec` é referenciada por workflows consumidores via tag móvel `setup-action-v1`. Após cada release que altere a Action, mova a tag para o novo SHA:

```bash
git tag -f setup-action-v1 <sha> && git push -f origin setup-action-v1
```

Substitua `<sha>` pelo commit que contém a versão da Action a ser publicada. A tag móvel garante que consumidores apontem sempre para a versão estável mais recente sem precisar atualizar seus workflows.

## Referencias

### Releases

- Releases: <https://github.com/JailtonJunior94/orchestrator/releases>
- Homebrew Tap: <https://github.com/JailtonJunior94/homebrew-tap>

### Documentacao

- [Playbook Mestre de Desenvolvimento](docs/development-playbook.md) — arvore de decisao e fluxo recomendado entre task unica, lote curto e orquestracao completa
- [Scorecard de Qualidade e Confianca](docs/quality-scorecard.md) — criterio `pass` / `warning` / `fail` para readiness do bundle
- [Checklist de Preflight e Readiness](docs/preflight-checklist.md) — gates bloqueantes antes de `execute-task`, `task-loop` ou `execute-all-tasks`
- [Guia de uso das skills](docs/skills-usage-guide.md) — contratos, prompts mandatorios e criterios de aceite por skill
- [Biblioteca de prompts](docs/prompt-library.md) — prompts copiaveis para execucao por task e variantes curta, padrao e rigorosa
- [Matriz de Confiabilidade por Ferramenta](docs/tool-reliability-matrix.md) — papel recomendado, risco operacional e custo de contexto por ferramenta
- [Bundle canonico de referencia](.specs/prd-example-preflight-entrypoint/) — exemplo end-to-end auditavel do fluxo governado
- [Guia do task-loop](docs/task-loop-reference.md) — flags, heuristicas, alternativas e comparativos
- [Guia de resolucao de problemas](docs/troubleshooting.md) — 12 problemas comuns com sintoma, causa, solucao e verificacao
- [Telemetria e ciclo de feedback](docs/telemetry-feedback-cycle.md)
- [ADR 006 — Telemetria opt-in](docs/adr/006-telemetria-feedback-cycle.md)
- [ADR 007 — Copilot CLI stateless workaround](docs/adr/007-copilot-cli-stateless-workaround.md)
- [ADR 008 — Paridade multi-tool com invariantes](docs/adr/008-parity-multi-tool-invariants.md)
