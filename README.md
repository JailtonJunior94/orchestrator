# ai-spec-harness

CLI em Go para instalar, validar, inspecionar e atualizar governanca operacional para ferramentas de IA em repositorios de software.

No repositorio e no modulo Go o nome e `ai-spec-harness`. No uso diario, este README prioriza o binario de release `ai-spec`.

O projeto padroniza como Claude, Gemini, Codex e GitHub Copilot encontram skills, agentes, comandos e contexto de execucao dentro de um repositorio alvo. O foco nao e "conversar com um modelo", e sim tornar fluxos repetiveis como PRD, especificacao tecnica, decomposicao de tasks, review e execucao de tarefas mais previsiveis e auditaveis.

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

- `create-prd`
- `create-technical-specification`
- `create-tasks`
- `execute-task`
- `review`
- `bugfix`
- `refactor`

## Requisitos

- Go `1.26.2` ou compativel com o [`go.mod`](./go.mod)
- `git` disponivel no `PATH`
- permissao de escrita no projeto alvo
- um repositorio fonte de governanca contendo `.agents/skills`

## Instalacao

O caminho principal para uso local e instalar o binario de release via Homebrew. O executavel publicado por release se chama `ai-spec`.

### macOS com Homebrew

```bash
brew install jailtonjunior94/tap/ai-spec
ai-spec version
```

> **Aviso de seguranca do macOS (Gatekeeper)**
>
> O macOS pode exibir o alerta _"Apple could not verify 'ai-spec' is free of malware"_ ao executar o binario pela primeira vez. Isso ocorre porque o binario nao esta assinado com um Apple Developer ID. Ha quatro formas de resolver:
>
> **Opcao 1 — Terminal (recomendada):** remove o atributo de quarentena do binario.
>
> ```bash
> xattr -dr com.apple.quarantine $(which ai-spec)
> ```
>
> **Opcao 2 — Interface grafica:** abra o Finder, navegue ate o binario, clique com o botao direito e selecione **Abrir**. Na janela de aviso, clique em **Abrir assim mesmo**. Isso registra uma excecao permanente no Gatekeeper para este binario.
>
> **Opcao 3 — Configuracoes do sistema:** va em **Ajustes do Sistema → Privacidade e Seguranca**, role ate a secao **Seguranca** e clique em **Abrir assim mesmo** ao lado da mensagem sobre `ai-spec`. Disponivel ate 1 hora apos a primeira tentativa bloqueada.
>
> **Opcao 4 — spctl (linha de comando alternativa):** adiciona o binario a lista de excecoes do Gatekeeper via ferramenta de politica de seguranca.
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

## Completion para bash e zsh

O comando `completion` gera scripts de autocompletion para o nome real do binario. Se voce estiver usando alias `ai-spec -> ai-spec-harness` ou o inverso, mantenha o alias no shell e gere o completion para o executavel que realmente esta instalado.

### Bash

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

### Zsh

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

### macOS com download direto

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

### Linux com download direto

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

### Windows com download direto

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

## Comandos disponiveis

| Comando | Finalidade |
| --- | --- |
| `install` | Instala governanca de IA em um projeto alvo |
| `upgrade` | Atualiza skills, adaptadores e manifesto em uma instalacao existente |
| `inspect` | Exibe skills instaladas, ferramentas detectadas e estado do manifesto |
| `doctor` | Executa checks de saude sobre git, manifesto, symlinks e permissoes |
| `lint` | Detecta placeholders nao renderizados, schema divergente e `SKILL.md` invalidos |
| `metrics` | Calcula metricas de contexto e custo estimado de tokens |
| `telemetry` | Registra e resume uso de skills e referencias |
| `validate` | Valida frontmatter YAML de `SKILL.md` |
| `validate-bugs` | Valida um array JSON de bugs contra o schema canonico |
| `prerequisites` | Verifica se uma skill pode ser executada em um projeto |
| `task-loop` | Executa todas as tasks elegiveis de um PRD folder via agente de IA |
| `wrapper` | Emite instrucoes de invocacao para Codex, Gemini e Copilot |
| `scaffold` | Cria a estrutura inicial de uma nova skill de linguagem |
| `uninstall` | Remove artefatos instalados pelo CLI |
| `completion` | Gera scripts de autocompletion para shell |
| `version` | Exibe versao, commit e data de build |

### Exemplos por comando

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

# remover a instalacao
ai-spec uninstall ../api-pagamentos --dry-run
```

## Sessao de uso completa

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

## Estrategias de instalacao

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

## Desenvolvimento local

```bash
go test ./...
go run . --help
go run . install ../sandbox --source . --tools codex --langs go --dry-run
```

## Contribuicao

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

## Roadmap curto

- melhorar a consistencia do nome do binario entre release e `go install`
- expandir exemplos por stack e por ferramenta
- adicionar mais fluxos canonicos orientados por task

## Releases

- Releases: <https://github.com/JailtonJunior94/orchestrator/releases>
- Homebrew Tap: <https://github.com/JailtonJunior94/homebrew-tap>

## Licenca

Este repositorio ainda nao expoe um arquivo `LICENSE` na raiz. Se a intencao e distribuicao open source publica, vale adicionar a licenca antes de ampliar o uso externo.
