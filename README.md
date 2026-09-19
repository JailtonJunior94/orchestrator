# ai-spec-harness

CLI em Go que instala, valida, inspeciona e atualiza governança operacional para ferramentas de
IA (Claude Code, Codex, GitHub Copilot, OpenCode) em qualquer repositório de software.

O nome do módulo Go é `ai-spec-harness`. O binário publicado nas releases se chama `ai-spec`.
Este README usa `ai-spec` em todos os exemplos.

O foco não é "conversar com um modelo": é tornar fluxos repetidos, como PRD, especificação
técnica, decomposição de tarefas, revisão e execução, previsíveis e auditáveis, com as mesmas
regras valendo para Claude, Codex, Copilot e OpenCode.

## Sumário

- [Instalação do binário](#instalação-do-binário)
- [Qual é o meu cenário?](#qual-é-o-meu-cenário)
  - [Cenário 1: projeto novo, sem código ainda](#cenário-1-projeto-novo-sem-código-ainda)
  - [Cenário 2: projeto existente que já usa alguma IA](#cenário-2-projeto-existente-que-já-usa-alguma-ia)
  - [Cenário 3: projeto existente sem nenhuma IA instalada](#cenário-3-projeto-existente-sem-nenhuma-ia-instalada)
- [Depois de instalar: o fluxo de trabalho](#depois-de-instalar-o-fluxo-de-trabalho)
- [Referência de comandos](#referência-de-comandos)
- [Prompts efetivos por ferramenta](#prompts-efetivos-por-ferramenta)
- [Ciclo de vida do estado SDD](#ciclo-de-vida-do-estado-sdd)
- [Manter a instalação atualizada](#manter-a-instalação-atualizada)
- [Solução de problemas](#solução-de-problemas)
- [Harness portátil e vendor-neutral](#harness-portátil-e-vendor-neutral)
- [Runtime ACP (execução orquestrada)](#runtime-acp-execução-orquestrada)
- [Documentação de referência](#documentação-de-referência)
- [Para quem mantém este repositório](#para-quem-mantém-este-repositório)

## Instalação do binário

### Requisitos

- `git` disponível no `PATH`
- Permissão de escrita no projeto alvo
- Go `1.26.2` ou compatível, apenas se você for compilar localmente

### Homebrew (recomendado para macOS)

```bash
brew install jailtonjunior94/tap/ai-spec
ai-spec version
```

> **Aviso de segurança do macOS (Gatekeeper)**
>
> O macOS pode exibir o alerta "Apple could not verify 'ai-spec' is free of malware" na primeira
> execução, porque o binário não é assinado com um Apple Developer ID. Resolva com um destes
> caminhos:
>
> ```bash
> xattr -dr com.apple.quarantine $(which ai-spec)
> ```
>
> Ou, pela interface gráfica: Finder, botão direito no binário, **Abrir**, confirme **Abrir assim
> mesmo**. Ou em **Ajustes do Sistema > Privacidade e Segurança**, clique em **Abrir assim mesmo**
> ao lado do aviso sobre `ai-spec`. A partir do `brew upgrade ai-spec`, o `post_install` da Formula
> já executa o `xattr` automaticamente, então o alerta some para novos usuários.

Se o shell não herdar o `PATH` do Homebrew, adicione ao arquivo de inicialização (`~/.zshrc` ou
`~/.bashrc`) e mantenha um alias compatível com o nome do módulo Go:

```bash
export PATH="$(brew --prefix)/bin:$PATH"
alias ai-spec-harness="ai-spec"
```

Depois recarregue o shell:

```bash
source ~/.zshrc   # ou source ~/.bashrc
```

### Completion para bash e zsh

Bash, sessão atual:

```bash
source <(ai-spec completion bash)
```

Bash, persistente no macOS com Homebrew:

```bash
ai-spec completion bash > "$(brew --prefix)/etc/bash_completion.d/ai-spec"
```

Zsh, sessão atual (se `compinit` já estiver habilitado):

```bash
source <(ai-spec completion zsh)
```

Zsh, persistente no macOS com Homebrew:

```bash
ai-spec completion zsh > "$(brew --prefix)/share/zsh/site-functions/_ai-spec-harness"
```

### Download direto

macOS Apple Silicon:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_darwin_arm64.tar.gz
tar -xzf ai-spec_<VERSION>_darwin_arm64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

macOS Intel:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_darwin_amd64.tar.gz
tar -xzf ai-spec_<VERSION>_darwin_amd64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

Linux `amd64`:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_linux_amd64.tar.gz
tar -xzf ai-spec_<VERSION>_linux_amd64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

Linux `arm64`:

```bash
curl -LO https://github.com/JailtonJunior94/orchestrator/releases/download/v<VERSION>/ai-spec_<VERSION>_linux_arm64.tar.gz
tar -xzf ai-spec_<VERSION>_linux_arm64.tar.gz
chmod +x ai-spec
sudo mv ai-spec /usr/local/bin/ai-spec
ai-spec version
```

Windows (`amd64`/`arm64` a partir da v0.11.1), PowerShell:

```powershell
$version = "<VERSION>"
$url = "https://github.com/JailtonJunior94/orchestrator/releases/download/v$version/ai-spec_${version}_windows_amd64.zip"
Invoke-WebRequest -Uri $url -OutFile "ai-spec.zip"
Expand-Archive -Path ".\ai-spec.zip" -DestinationPath ".\ai-spec"
Move-Item ".\ai-spec\ai-spec.exe" "$env:USERPROFILE\bin\ai-spec.exe"
$env:Path += ";$env:USERPROFILE\bin"
ai-spec.exe version
```

### Instalação via Go

```bash
go install github.com/JailtonJunior94/ai-spec-harness@latest
ai-spec-harness version
```

O binário gerado por `go install` chama-se `ai-spec-harness`, não `ai-spec`. Para usar os mesmos
exemplos deste README, crie um alias:

```bash
alias ai-spec="ai-spec-harness"
```

### Executar sem instalar (a partir deste checkout)

```bash
go run . --help
```

## Qual é o meu cenário?

Escolha o cenário mais próximo do seu projeto e siga o passo a passo dele. Todos os três levam ao
mesmo lugar: um repositório com governança instalada, validada e pronta para receber um PRD.

| Cenário | Quando se aplica |
| --- | --- |
| [1. Projeto novo](#cenário-1-projeto-novo-sem-código-ainda) | Repositório vazio ou recém-criado, ainda sem código real |
| [2. Projeto existente com IA já instalada](#cenário-2-projeto-existente-que-já-usa-alguma-ia) | Já existe `.claude/`, `.codex/`, `.github/copilot-instructions.md` ou `.opencode/`, mesmo que não seja o `ai-spec` |
| [3. Projeto existente sem nenhuma IA instalada](#cenário-3-projeto-existente-sem-nenhuma-ia-instalada) | Só código, nenhum artefato de IA no repositório |

### Cenário 1: projeto novo, sem código ainda

Não existe arquitetura real para analisar, então o primeiro passo é definir escopo, não instalar
skills de linguagem específica ainda.

**Passo 1: crie o repositório e instale a governança**

```bash
mkdir meu-projeto-novo && cd meu-projeto-novo && git init
ai-spec install . --tools all --langs all
```

**Passo 2: valide a instalação**

```bash
ai-spec inspect .
ai-spec doctor .
ai-spec lint .
```

O `doctor` deve terminar com "Resultado: tudo ok" e o `lint` com "Lint aprovado". Se algo falhar,
veja [Solução de problemas](#solução-de-problemas).

**Passo 3: abra a ferramenta de IA (Claude, Codex, Copilot ou OpenCode) e peça o primeiro PRD**

```text
Use a skill create-prd para definir o primeiro escopo deste projeto novo.

Quero no resultado:
- problema
- objetivos e não objetivos
- requisitos funcionais e não funcionais
- riscos iniciais
```

**Passo 4: continue o fluxo padrão**

```text
create-prd -> create-technical-specification -> create-tasks -> execute-task
```

Detalhes de cada etapa estão em [Depois de instalar: o fluxo de trabalho](#depois-de-instalar-o-fluxo-de-trabalho).

### Cenário 2: projeto existente que já usa alguma IA

Se o repositório já tem `.claude/`, `.codex/`, `.opencode/` ou `.github/copilot-instructions.md`
(mesmo que não seja o `ai-spec`), esses arquivos podem colidir com o que o instalador gera. O
`ai-spec` protege contra isso: arquivos com marcadores `<!-- ai-spec:generated-start/end -->` são
mesclados; um arquivo sem marcadores e sem rastreio ganha um `.bak` no primeiro contato antes de
ser tocado.

**Passo 1: rode primeiro em modo simulação, para ver o que mudaria**

```bash
ai-spec install . --tools all --langs go --dry-run
```

Revise a saída. Ela mostra o que seria criado, mesclado ou preservado, sem escrever nada ainda.

**Passo 2: instale de verdade**

```bash
ai-spec install . --tools all --langs go
```

Se um arquivo já gerenciado por uma instalação anterior do `ai-spec` foi editado manualmente fora
dos marcadores (checksum em disco diferente do manifesto), o lote inteiro é abortado por padrão e
o comando nomeia cada arquivo em conflito. Revise a diferença e, só depois de decidir que quer
sobrescrever de propósito, rode:

```bash
ai-spec install . --tools all --langs go --overwrite-conflicts
```

**Passo 3: valide**

```bash
ai-spec inspect .
ai-spec doctor .
ai-spec lint .
```

**Passo 4: peça uma leitura arquitetural antes de qualquer mudança de comportamento**

```text
Use a skill analyze-project para analisar a arquitetura atual deste repositório.

Quero no resultado:
- classificação do tipo de projeto (monolito, monolito modular, monorepo ou microserviço)
- evidências usadas na classificação
- stack detectada
- padrão arquitetural predominante
- mapa das pastas mais importantes
- fluxo de dependências entre camadas ou módulos
- recomendações de governança para este contexto
```

Depois disso, siga o fluxo padrão descrito em
[Depois de instalar: o fluxo de trabalho](#depois-de-instalar-o-fluxo-de-trabalho).

### Cenário 3: projeto existente sem nenhuma IA instalada

Este é o caso mais simples: não há nada para conflitar, então a instalação é direta.

**Passo 1: instale a governança**

```bash
ai-spec install . --tools all --langs go
```

Troque `--langs go` pela linguagem real do projeto (`node`, `python`, `dotnet` ou `all` para
detectar automaticamente mais de uma). Sem `--tools`, o comando auto-detecta quais CLIs (Claude,
Codex, Copilot, OpenCode) estão presentes pelo binário no `PATH`, pelo diretório de configuração
do usuário ou por arquivos já existentes no projeto.

**Passo 2: valide**

```bash
ai-spec inspect .
ai-spec doctor .
ai-spec lint .
```

**Passo 3: peça a leitura arquitetural inicial**

```text
Use a skill analyze-project para analisar a arquitetura atual deste repositório.

Quero no resultado:
- classificação do tipo de projeto (monolito, monolito modular, monorepo ou microserviço)
- evidências usadas na classificação
- stack detectada
- padrão arquitetural predominante
- mapa das pastas mais importantes
- fluxo de dependências entre camadas ou módulos
- recomendações de governança para este contexto
```

**Passo 4: siga o fluxo padrão**

```text
create-prd -> create-technical-specification -> create-tasks -> execute-task
```

## Depois de instalar: o fluxo de trabalho

O `ai-spec-harness` não escreve PRD, especificação técnica ou código por conta própria. Ele
instala a governança para que o agente escolhido (Claude, Codex, Copilot ou OpenCode) execute cada
etapa com a skill correta, dentro do repositório alvo.

### 1. Criar o PRD

Toda mudança de comportamento, novo endpoint, nova flag de CLI ou nova regra de negócio começa
aqui. É proibido pular para especificação técnica ou implementação sem um PRD aprovado.

```text
Use a skill create-prd para criar um PRD de listagem de pagamentos.

Contexto:
- precisamos expor GET /payments
- filtros: status, página, período inicial e final
- o endpoint deve atender operação e backoffice

Quero no resultado:
- problema
- objetivos e não objetivos
- requisitos funcionais e não funcionais
- critérios de aceite
- riscos
```

O arquivo fica em `.specs/prd-<slug>/prd.md`, com requisitos funcionais numerados (`RF-01`,
`RF-02`...) para permitir rastreio de cobertura.

### 2. Criar a especificação técnica

```text
Use a skill create-technical-specification com base no PRD aprovado.

Contexto técnico:
- serviço Go existente
- arquitetura atual: handler -> service -> repository
- preservar contratos públicos existentes

Quero no resultado:
- modelagem de domínio
- fronteiras entre aplicação, domínio e infraestrutura
- estratégia de erros
- estratégia de testes
- riscos e plano de rollout
```

### 3. Decompor em tarefas

```text
Use a skill create-tasks para decompor a especificação técnica em tarefas pequenas,
executáveis e com evidências de validação.

Quero:
- ordem de execução
- dependências entre tarefas
- critério de pronto por tarefa
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

> Convenções de nome suportadas pelo `task-loop`: `1-desc.md`, `1.0-desc.md` e
> `task-1.0-desc.md`. O separador pode ser `-` ou `_`.

### 4. Executar as tarefas

Para uma tarefa isolada, com acompanhamento próximo:

```text
Use a skill execute-task para implementar a próxima tarefa elegível com validação proporcional.
```

Para um lote inteiro, o `task-loop` percorre `tasks.md`, identifica a próxima tarefa elegível e
invoca o agente com `execute-task` até concluir todas as possíveis.

```bash
ai-spec task-loop --tool codex --dry-run .specs/prd-payments-list
```

Valida ordem e elegibilidade sem gastar ciclo de agente. Depois, um lote pequeno para observar
qualidade:

```bash
ai-spec task-loop --tool codex --max-iterations 2 .specs/prd-payments-list
```

E, com confiança estabelecida, a execução completa com rastreabilidade:

```bash
ai-spec task-loop \
  --tool codex \
  --max-iterations 10 \
  --timeout 1h \
  --report-path ./task-loop-report-payments.md \
  .specs/prd-payments-list
```

Flags principais do `task-loop`:

| Flag | Função |
| --- | --- |
| `--tool` | Agente único: `claude`, `codex`, `copilot` ou `opencode` |
| `--dry-run` | Valida ordem e elegibilidade sem executar |
| `--max-iterations` | Limite de tarefas por execução (`0` = sem limite) |
| `--timeout` | Tempo limite por tarefa |
| `--report-path` | Caminho do relatório final em Markdown |
| `--executor-tool` / `--reviewer-tool` | Modo avançado: uma ferramenta implementa, outra revisa; ativa `bugfix` automático se a revisão falhar |
| `--executor-model` / `--reviewer-model` | Modelos específicos por papel |
| `--reviewer-prompt-template` | Template `.tmpl` customizado para o prompt de revisão |
| `--executor-fallback-model` / `--reviewer-fallback-model` | Modelos de fallback (suporte nativo hoje só em Claude) |
| `--allow-unknown-model` | Aceita combinações ferramenta + modelo fora do catálogo interno |

Boas práticas: comece sempre com `--dry-run`, depois um lote pequeno (`--max-iterations 2`), só
aumente quando a qualidade estiver satisfatória. Use `--report-path` em features críticas. Se a
tarefa envolver risco arquitetural real (concorrência, por exemplo), refine a `techspec.md` antes
de rodar o loop: ele é mais eficiente quando a especificação é inequívoca. Detalhes completos em
[`docs/task-loop-reference.md`](docs/task-loop-reference.md).

### 5. Validar o estado final

```bash
ai-spec lint .
go test ./...
```

### 6. Quando o pipeline completo é overhead

O fluxo `create-prd -> create-technical-specification -> create-tasks -> execute-task` é
obrigatório para qualquer mudança que altere comportamento. Para mudanças triviais, use a skill
direta, sem PRD:

| Tipo de mudança | Sinal observável | Skill direta |
| --- | --- | --- |
| Bug isolado com causa raiz reproduzível | 1 a 2 arquivos, sem RF novo, teste de regressão cabível | `bugfix` |
| Refactor delimitado, mesmo contrato | Sem mudança em API pública, schema ou flags de CLI | `refactor` |
| Renomeação local, typo, comentário | Zero impacto em comportamento | `execute-task` direto, sem PRD |
| Dependência atualizada sem breaking change | Só `go.mod`/`package.json` | `semantic-commit` direto |

O pipeline completo é mandatório sempre que a mudança adiciona, remove ou modifica um requisito
funcional; altera um contrato público (API HTTP, flags de CLI, schema de banco, formato de
arquivo); ou invalida um ADR existente. Em caso de dúvida, aplique o
[Scorecard de Qualidade e Confiança](docs/quality-scorecard.md).

### 7. Qual skill usar em cada situação

| Situação | Skill recomendada |
| --- | --- |
| Feature nova com 3 ou mais tarefas, ou novo requisito funcional | `create-prd` -> `create-technical-specification` -> `create-tasks` -> `execute-all-tasks` |
| User stories brutas como entrada | `us-to-prd` antes de `create-prd` |
| Tarefa única já planejada | `execute-task` |
| Bug isolado com sintoma reproduzível | `bugfix` |
| Refactor delimitado, sem novo comportamento | `refactor` |
| Diff pronto, falta revisar antes de merge | `review` |
| Comentários de PR para triar | `github-pr-comment-triage` |
| Projeto sem governança instalada | `analyze-project` |
| Commit, changelog ou release pendente | `semantic-commit` -> `github-diff-changelog-publisher` -> `github-release-publication-flow` |

## Referência de comandos

| Comando | Finalidade |
| --- | --- |
| `install` (alias `init`) | Instala governança de IA em um projeto alvo |
| `upgrade` (alias `sync`) | Atualiza skills, adaptadores e manifesto em uma instalação existente |
| `inspect` | Exibe skills instaladas, ferramentas detectadas e estado do manifesto |
| `doctor` | Verificações de saúde: repositório git, symlinks, permissões, manifesto, mais um bloco por provedor instalado |
| `lint` | Detecta placeholders não renderizados, schema divergente e `SKILL.md` inválidos |
| `verify` | Reporta cada skill como `current`, `missing` ou `drifted` |
| `metrics` | Calcula custo estimado de contexto e tokens |
| `telemetry` | Registra e resume uso de skills e referências |
| `skills check` / `skills --verify` | Verifica versões de skills externas contra `skills-lock.json` |
| `validate` | Valida o frontmatter YAML de `SKILL.md` |
| `validate-bugs` | Valida um array JSON de bugs contra o schema canônico |
| `prerequisites` | Verifica se uma skill pode ser executada em um projeto |
| `check-spec-drift` | Verifica cobertura de RFs e divergência de hash entre PRD, techspec e tasks |
| `sync-spec-hash` | Recalcula o SHA-256 de `prd.md` e `techspec.md` e atualiza `tasks.md` |
| `task-loop` | Executa todas as tarefas elegíveis de um PRD via agente de IA |
| `wrapper` | Emite instruções de invocação para Codex e Copilot |
| `scaffold` | Cria a estrutura inicial de uma nova skill de linguagem |
| `uninstall` | Remove artefatos instalados pelo CLI |
| `completion` | Gera scripts de autocompletion para o shell |
| `version` | Exibe versão, commit e data de build |
| `skill-bump` | Detecta skills alteradas desde a última tag e propõe bump de versão |

### Flags principais do `install` e do `upgrade`

| Flag | Valores | Para que serve |
| --- | --- | --- |
| `--source` | caminho | Repositório de governança usado como fonte. Sem ela, usa os assets embutidos no binário |
| `--tools` | `claude,codex,copilot,opencode` ou `all` | Quais adaptadores gerar. Sem a flag, auto-detecta pelo `PATH` e diretórios de configuração |
| `--langs` | `go,node,python` ou `all` | Skills de linguagem a incluir |
| `--mode` | `copy` \| `symlink` | `copy`: snapshot físico, projeto autocontido (recomendado para outros projetos). `symlink`: reflete mudanças da fonte (bom para desenvolver a própria governança) |
| `--dry-run` | | Simula sem escrever |
| `--global` | | Instala em `~/.aispec` (escopo global) |
| `--ref` | tag, branch ou SHA | Usa um ref git da fonte como base |
| `--codex-profile` | `full` \| `lean` | Perfil de skills do Codex |
| `--overwrite-conflicts` | | Sobrescreve arquivo gerenciado cujo checksum diverge do manifesto (edição manual fora dos marcadores). Sem a flag, o lote inteiro é abortado |
| `--check` (só `upgrade`) | | Apenas verifica o que mudaria, sem escrever nada |

`install` e `upgrade` aceitam os aliases `init` e `sync`, que resolvem para o mesmo comando, sem
flag ou saída diferente:

```bash
ai-spec init . --source . --tools all --langs go
ai-spec sync . --check
```

### Exemplos úteis por comando

```bash
# instalar governança em um projeto
ai-spec install ../api-pagamentos --source . --tools codex,claude --langs go

# inspecionar e diagnosticar
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec doctor ../api-pagamentos --codex-trust   # + trust dos hooks do Codex via RPC read-only

# verificar governança gerada
ai-spec lint ../api-pagamentos
ai-spec lint ../api-pagamentos --strict          # avisos de paridade viram erro

# verificar instalação, com resumo por CLI
ai-spec verify ../api-pagamentos --by-cli

# atualizar instalação e checar antes de aplicar
ai-spec upgrade ../api-pagamentos --source . --langs go --check
ai-spec upgrade ../api-pagamentos --source . --langs go

# validar todas as skills do repositório fonte
ai-spec validate .agents/skills

# validar bugs.json contra o schema canônico
ai-spec validate-bugs ./bugs.json

# checar pré-requisitos antes de rodar uma skill
ai-spec prerequisites create-tasks .

# medir custo de contexto em JSON
ai-spec metrics . --format json

# registrar e ler telemetria de skill
GOVERNANCE_TELEMETRY=1 ai-spec telemetry log create-prd
ai-spec telemetry summary
ai-spec telemetry report --trend

# emitir instrução pronta de wrapper para uma ferramenta
ai-spec wrapper codex create-tasks .

# criar scaffold para uma nova linguagem
ai-spec scaffold rust --root .

# verificar cobertura de requisitos e drift de spec
ai-spec check-spec-drift .specs/prd-payments-list/tasks.md

# sincronizar hashes após editar prd.md ou techspec.md
ai-spec sync-spec-hash .specs/prd-payments-list/tasks.md

# executar todas as tarefas elegíveis de um PRD
ai-spec task-loop --tool codex .specs/prd-payments-list

# verificar versões de skills externas contra o lock file
ai-spec skills check .
ai-spec skills --verify .

# listar versões de skills (embutidas, instaladas ou ambas)
ai-spec version --skills
ai-spec version --skills=embedded
ai-spec version --skills=installed

# detectar skills alteradas desde a última tag
ai-spec skill-bump .
ai-spec skill-bump . --dry-run

# remover a instalação
ai-spec uninstall ../api-pagamentos --dry-run
ai-spec uninstall ../api-pagamentos
```

## Prompts efetivos por ferramenta

Para reduzir desvio, todo prompt deve ter quatro blocos:

```text
1. skill explícita
2. contexto objetivo e verificável
3. restrições reais do repositório
4. saída esperada
```

```text
Template:

Use a skill <skill>.

Contexto:
- <fatos concretos>
- <arquivos, contratos ou restrições>

Quero no resultado:
- <artefatos ou decisões esperadas>
- <critérios de qualidade>
```

Evite: "faça o melhor possível", "analise tudo", "implemente completo" sem delimitar escopo, ou
pedir PRD, especificação técnica e execução no mesmo prompt.

### Codex

```bash
ai-spec wrapper codex create-prd .
ai-spec wrapper codex create-technical-specification .
ai-spec wrapper codex create-tasks .
ai-spec wrapper codex execute-task .
```

### Claude Code

Claude Code usa os artefatos instalados em `.claude/`, incluindo hooks e skills sincronizadas pelo
projeto. O melhor resultado vem de prompts curtos, com uma skill por vez.

```text
Use a skill review para revisar o diff atual.

Contexto:
- priorize regressão, risco, segurança e testes faltantes
- considere o PRD e a especificação técnica da feature atual

Quero no resultado:
- findings ordenados por severidade
- arquivos e linhas afetadas
- riscos residuais, se não houver bloqueadores
```

### GitHub Copilot

```bash
ai-spec wrapper copilot execute-task .
ai-spec wrapper copilot review .
```

```text
Use a skill execute-task para implementar a task atual.

Contexto:
- mantenha o escopo restrito ao task file selecionado
- não altere comportamento público fora do que a task exigir
- gere evidências e atualize o status somente após validação

Quero no resultado:
- código e testes da task
- resumo das validações executadas
- status final canônico: done, blocked, failed ou needs_input
```

## Ciclo de vida do estado SDD

Além dos arquivos Markdown (`prd.md`, `techspec.md`, `tasks.md`), o estado operacional de um PRD
vive em `.specs/prd-<slug>/sdd-state.json` (schema v2). O Markdown é a projeção legível por
humanos; o estado é o que as automações leem.

### Sequência canônica

```bash
# 1. Aprovar em cascata. O primeiro approve cria o sdd-state.json se ele ainda
#    não existir, não há comando de init separado. Cada artefato exige o
#    anterior aprovado e com o conteúdo intacto.
ai-spec approve prd      .specs/prd-exemplo
ai-spec approve techspec .specs/prd-exemplo
ai-spec approve tasks    .specs/prd-exemplo

# 2. Validar o contrato (schema, hashes, vínculos RF -> tarefa, DAG, ownership)
ai-spec validate-sdd .specs/prd-exemplo

# 3. Executar de forma recuperável (lock exclusivo, eventos append-only)
ai-spec orchestrate .specs/prd-exemplo --run-id <id>
```

`validate-sdd` num PRD sem estado falha, porque o estado nasce no primeiro `approve`.

### Editando um artefato já aprovado

Editar um `prd.md` aprovado deixa todo o downstream desatualizado, e `sync-spec-hash` se recusa a
rodar nesse estado, de propósito, para que a sincronização de hash nunca esconda um drift real.

```bash
# depois de editar prd.md (e/ou techspec.md)
ai-spec invalidate .specs/prd-exemplo --from prd   # marca descendentes como stale
ai-spec approve prd      .specs/prd-exemplo         # reaprova a origem alterada
ai-spec approve techspec .specs/prd-exemplo
ai-spec sync-spec-hash   .specs/prd-exemplo/tasks.md
ai-spec approve tasks    .specs/prd-exemplo         # tasks.md mudou no passo anterior
ai-spec validate-sdd     .specs/prd-exemplo
```

Reaprovar um artefato intacto é recusado: a reaprovação existe para aceitar conteúdo que mudou,
não para gerar ruído de estado.

### Selo de evidência

A prova de execução é verificada contra a árvore de trabalho viva, que deixa de existir quando o
trabalho é commitado. Sem selo, a evidência de uma tarefa não é reauditável depois do merge. O
harness nunca cria commits, então o selo é uma segunda fase, aplicada depois do commit:

```bash
ai-spec seal-evidence <result.json> --prd-dir .specs/prd-exemplo --commit HEAD
ai-spec seal-evidence <result.json> --prd-dir .specs/prd-exemplo --verify
```

O selo grava `commit_sha` e `commit_patch_sha256`, exige que o commit descenda da base registrada
e recusa reselagem. A verificação não toca a árvore de trabalho, então permanece válida
indefinidamente. O selo torna a evidência imutável e reverificável dali em diante, mas não prova
que o commit é byte-idêntico à árvore do fechamento, porque essa árvore já não existe no momento
do selo.

### Validação de resultados e migração

```bash
ai-spec validate-result execution <result.json>              # contrato JSON v2 estrito
ai-spec validate-result execution <result.json> \
  --verify-physical --prd-dir .specs/prd-exemplo             # + provas físicas

ai-spec migrate-sdd  .specs/prd-exemplo --dry-run            # nunca altera estado
ai-spec rollback-sdd .specs/prd-exemplo --run-id <id>        # remove só o estado v2 daquela execução
```

### Gate de critérios de aceite

Um relatório cuja task file não seja resolvível pelo campo `Arquivo:`, ou cuja task não declare
seção de critérios, falha. Não existe opt-out: o gate é fail-closed e não há variável de ambiente
que reabra o comportamento antigo.

## Manter a instalação atualizada

Quando uma nova versão for publicada em
<https://github.com/JailtonJunior94/orchestrator/releases>, siga esta sequência em cada
repositório instrumentado:

**1. Atualize o binário**

```bash
brew upgrade ai-spec
ai-spec version
```

Ou, via Go:

```bash
go install github.com/JailtonJunior94/ai-spec-harness@latest
ai-spec-harness version
```

Em CI GitHub Actions:

```yaml
- uses: JailtonJunior94/orchestrator/.github/actions/setup-ai-spec@setup-action-v1
  with:
    version: latest
```

Essa Action instala via Homebrew no macOS e baixa o tarball oficial no Linux, validando o
`sha256` antes de expor o binário no `PATH`.

**2. Sincronize a governança instalada em cada projeto**

```bash
ai-spec upgrade ../api-pagamentos --check
ai-spec upgrade ../api-pagamentos
ai-spec inspect ../api-pagamentos
ai-spec doctor ../api-pagamentos
ai-spec lint ../api-pagamentos
```

Use `--check` como preflight. Se você estiver sincronizando a partir deste checkout local em vez
da versão publicada, mantenha a fonte explícita:

```bash
ai-spec upgrade ../api-pagamentos --source . --check
ai-spec upgrade ../api-pagamentos --source . --langs go
```

> Sempre passe `--source` no `upgrade` quando a instalação partiu de um checkout local. Sem
> `--source`, o `upgrade` compara contra os assets embutidos no binário (a release instalada). Se
> o projeto foi instalado de uma fonte mais nova que a release, rodar `upgrade` sem `--source`
> rebaixa as skills para a versão embutida.

**3. Revalide sempre**

Toda instalação ou sincronização deve terminar com `inspect`, `doctor` e `lint`. Se `doctor`
falhar, trate a instalação como não confiável até corrigir manifesto, symlinks, permissões ou
estado do git.

**4. Se houver PRD ou especificação técnica editados intencionalmente**

```bash
ai-spec sync-spec-hash .specs/<prd>/tasks.md
```

### `verify --source` e `upgrade --check` respondem perguntas diferentes

- `ai-spec verify <projeto> --source <fonte>` audita as skills já instaladas contra a fonte.
  `0 drifted` confirma fidelidade. Esta é a porta de saúde.
- `ai-spec upgrade <projeto> --source <fonte> --check` compara o conjunto completo de skills da
  fonte com o projeto e pode listar como ausente skills que existem na fonte mas que o `install`
  não instala por padrão (skills internas de manutenção deste repositório, por exemplo). Não trate
  isso automaticamente como erro: revise a lista antes de aplicar o upgrade.

A confirmação final de saúde é `doctor` verde mais `verify --source` com `0 drifted`, não
`upgrade --check`.

### `symlink` ou `copy`

`--mode symlink` é melhor para desenvolver a própria governança, porque o projeto alvo reflete
mudanças feitas na fonte:

```bash
ai-spec install ../api-pagamentos --source . --tools all --langs all --mode symlink
```

`--mode copy` é melhor quando o ambiente não lida bem com links simbólicos ou quando você quer um
snapshot físico do baseline, autocontido:

```bash
ai-spec install ../api-pagamentos --source . --tools all --langs all --mode copy
```

### Pre-commit hook opt-in (spec-drift, skills-sync, hooks-sync)

O repositório oferece um hook `pre-commit` em `scripts/git-hooks/pre-commit` que valida três
invariantes antes de cada commit: sincronização de `.agents/skills/` com os espelhos, sincronização
de `.claude/hooks/` com os espelhos, e drift de `prd.md`/`techspec.md` via
`ai-spec check-spec-drift` quando esses arquivos estão staged.

Ativação, opt-in:

```bash
git config core.hooksPath scripts/git-hooks
```

Comportamento permissivo: se `ai-spec` não está no `PATH` ou está em versão antiga, o bloco de
spec-drift emite um aviso em `stderr` e permite o commit. Em caso de drift real, o hook bloqueia
com a remediação sugerida (`ai-spec sync-spec-hash .specs/prd-<slug>/tasks.md`). Escape hatch
responsável: `git commit --no-verify` pula o hook em casos legítimos, mas o drift fica para a
revisão pegar depois, então não normalize o bypass.

### Reset completo (uninstall + install)

Use apenas quando precisar de um baseline limpo do zero, não como rotina de atualização.

```bash
ai-spec uninstall <projeto> --dry-run     # confira a lista antes
ai-spec uninstall <projeto>
ai-spec install   <projeto> --source <fonte> --tools all --langs go --mode copy
ai-spec doctor    <projeto>
ai-spec lint      <projeto>
ai-spec verify    <projeto> --source <fonte>
```

`uninstall` remove apenas os arquivos rastreados no manifesto (`.ai_spec_harness.json`). Ele não
apaga conteúdo seu, como `.claude/settings.local.json`. Nunca use `rm -rf .claude .opencode
.agents`: esses diretórios podem conter arquivos seus que o `uninstall` preserva de propósito.

## Solução de problemas

| Sintoma | Causa provável | Solução |
| --- | --- | --- |
| `BLOQUEIO: skill obrigatória não acessível` | Skill não instalada ou `INDEX.yaml` ausente | `ai-spec install .` |
| Hook do Claude Code não dispara em Edit/Write | `.claude/settings.local.json` sem o bloco `PreToolUse` | Cole o bloco de hooks e reabra a sessão |
| "governança não carregada" em toda edição | Falta `export GOVERNANCE_PRELOAD_CONFIRMED=1` | Exporte na sessão |
| `verify` reporta `drifted` sem `--source` | Comparação contra o binário embutido, não a fonte real | Rode `verify --source <fonte>`; se ainda houver drift, `ai-spec install .` |
| `upgrade --check` lista "ausente" numa instalação recém-feita | A fonte tem skills internas de manutenção que o `install` não instala por padrão | Esperado; revise a lista antes de decidir aplicar |
| `upgrade` rebaixou (downgrade) as skills | Rodou `upgrade` sem `--source` e comparou contra o binário embutido | Sempre use `upgrade --source <fonte>`; reinstale a partir da fonte para corrigir |
| `doctor` falha em "Repositório git" | Projeto sem `git init` | Rode `git init` no projeto alvo |
| `doctor` falha em symlinks de skills | Symlinks órfãos ou quebrados | Reinstale; remova só links comprovadamente órfãos |
| Quero remover tudo | | `ai-spec uninstall .` |

### Variáveis de ambiente operacionais

| Variável | Efeito |
| --- | --- |
| `GOVERNANCE_PRELOAD_CONFIRMED=1` | Confirma que você carregou `AGENTS.md` na sessão |
| `PREREQ_MODE=warn` | O gate de pré-requisitos emite aviso em vez de bloquear |
| `GOVERNANCE_PRELOAD_MODE=warn` | O hook de preload não bloqueia por ausência de confirmação |
| `AGENTS_ROOT=<dir>` | Override do diretório raiz do projeto para resolução de skills |
| `GOVERNANCE_GIT_OPERATION_CONFIRMED=1` | Libera `git commit`/`git push` desta invocação, quando pedido explicitamente |
| `GOVERNANCE_GIT_OPERATION_MODE=warn` | O gate de operação Git vira aviso, não bloqueio |
| `GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED=1` | Libera remoção destrutiva (`rm -rf`) desta invocação |
| `GOVERNANCE_TELEMETRY=1` | Habilita a telemetria opt-in de uso de skills |

## Harness portátil e vendor-neutral

O `ai-spec-harness` é um contrato de engenharia com fonte canônica única e verificável, declarada
explicitamente em vez de emergir do código, com equivalência entre provedores provada por artefato
versionado, não por convicção.

### Harness Contract v1 (`.agents/harness.yaml`)

Contrato declarativo e opcional de política (não de operação): `version: 1` mais cinco famílias:
`git`, `approval`, `quality`, `evidence`, `skills`. O parse é estrito: campo desconhecido ou
versão incompatível produz erro tipado e falha o comando que o consome, nunca um aviso silencioso.

```yaml
version: 1
git:
  auto_commit: false
  auto_push: false
approval:
  require_for_destructive_operations: true
quality:
  require_tests: true
  require_lint: true
evidence:
  require_execution_report: true
skills:
  discovery_mode: declared
```

Se o arquivo não existir no projeto alvo, o harness aplica um default embutido equivalente ao
exemplo acima. Nenhum projeto existente muda de comportamento só por causa desta entrega.
`ai-spec doctor` já consome o contrato e relata se a instalação está usando o arquivo declarado ou
o default embutido.

### Políticas canônicas (`.agents/policies/`)

`R-GOV-001` e `R-STYLE-001` têm uma única origem editável neste repositório,
`.agents/policies/governance.md` e `.agents/policies/code-style.md`. `.claude/rules/*.md` é o
espelho derivado. O par `scripts/sync-policies.sh` / `scripts/check-policies-sync.sh` gera e
verifica esse espelho, e o gate falha se o espelho for editado direto, se a origem for apagada ou
se os dois divergirem (`make check-policies-sync`).

A distribuição de `code-style.md` para os projetos consumidores (via `install`/`upgrade`) é
incondicional desde a versão que introduziu o harness portátil: não é opt-in e não há novo padrão
configurável. Veja a seção "Unreleased" do `CHANGELOG.md` para a nota de migração, caso o
consumidor não queira aplicar a regra localmente.

### Capability matrix gerada

`docs/capability-matrix.md` e `testdata/capability-matrix.json` são gerados a partir dos
invariantes de `internal/parity` (fonte única), nunca editados à mão. Toda célula marcada como
`supported` ou `provider capability` cita o teste que a sustenta; células `unknown` trazem um
motivo explicando por que ainda não há prova, nunca por omissão silenciosa.
`make check-capability-matrix-sync` falha o CI se o artefato commitado divergir do gerado ou se
uma célula suportada não tiver teste associado. Para regenerar após alterar um invariante:

```bash
UPDATE_SNAPSHOTS=1 go test ./internal/capability/...
```

### Manifesto com checksum e conflito

O manifesto da instalação (`.ai_spec_harness.json`) registra `file_checksums` (caminho para
SHA-256) para cada arquivo gerenciado, além do rastreamento por skill que já existia. Um manifesto
anterior a essa entrega, sem o campo, nunca é tratado como divergente, apenas como "sem rastreio
de checksum", e é migrado automaticamente no próximo `upgrade`.

Com o checksum, `install`/`upgrade` detectam conflito: se um arquivo gerenciado foi editado
manualmente fora do harness, o lote inteiro é abortado por padrão, nomeando cada arquivo em
conflito e a origem canônica dele. A flag `--overwrite-conflicts` sobrescreve de propósito (ver
[Flags principais do install e do upgrade](#flags-principais-do-install-e-do-upgrade)).

A aplicação do lote é transacional: se qualquer arquivo falhar no meio do processo, a árvore volta
a ficar byte-idêntica ao estado anterior, sem arquivo temporário remanescente e sem escrita
parcial.

### Gate canônico de operação Git

`.agents/scripts/git-operation-gate.sh` é o gate compartilhado, espelhado para os quatro
provedores, que nega `git commit`/`git push` não solicitados e remoção recursiva e forçada
(`rm -rf`, `rm -fr`, `--recursive --force`, em qualquer combinação de flags), mesmo com
`GOVERNANCE_PRELOAD_CONFIRMED=1` exportado. Comando Git de leitura (`git status`, `git diff`,
`git log`) continua liberado. Os escapes exigem pedido explícito do usuário (ver
[Variáveis de ambiente operacionais](#variáveis-de-ambiente-operacionais)) e o bloqueio sai com
exit code `2`.

### Budget de contexto e conformidade cross-provider

Além do budget por skill medido por `ai-spec metrics`, o harness mede o budget de contexto de
entrada por provedor, ou seja, o que Claude, Codex, Copilot e OpenCode efetivamente carregam no
primeiro turno (`CLAUDE.md`, `AGENTS.md`, `.codex/config.toml`,
`.github/copilot-instructions.md`, entre outros), com teto de 10% de margem sobre o medido.

A suíte de conformidade cross-provider (`internal/conformance`) formaliza cenários normativos,
classificados como determinístico (roda em CI, bloqueia merge) ou live (roda nightly, nunca
bloqueia merge). Um runner único confirma que os quatro provedores retornam o mesmo exit code
para a mesma entrada.

### Telemetria comparável e doctor multi-provider

`ai-spec telemetry` registra métricas comparáveis entre execuções (provedor, modelo, duração,
chamadas de ferramenta, status final, tentativas), visíveis em `telemetry report`,
`telemetry summary` e `telemetry report --trend`.

`ai-spec doctor` reporta um bloco Core mais um bloco por provedor efetivamente instalado
(Claude/Codex/Copilot/OpenCode), sempre com verificações estáticas e sem chamada paga a LLM. O
flag `--codex-trust` verifica o trust dos hooks do Codex via RPC read-only (`hooks/list`) do
`codex app-server`:

```bash
ai-spec doctor ../api-pagamentos --codex-trust
```

## Runtime ACP (execução orquestrada)

O `ai-spec-harness` suporta um modo de execução baseado no Agent Client Protocol (ACP), ativado
pela flag `--runtime=acp` do `task-loop`. Nesse modo, o harness abre uma sessão ACP com a CLI
escolhida e consome um stream de eventos em tempo real, em vez de aguardar um processo one-shot
encerrar. O modo `--runtime=legacy` continua sendo o padrão; nenhuma tarefa existente muda de
comportamento sem mudança explícita de flag.

O runtime ACP funciona com as quatro CLIs (`claude`, `codex`, `copilot`, `opencode`) com
comportamento equivalente: mesma normalização de chamadas de ferramenta, mesmas métricas
unificadas, mesma memória em duas camadas, mesmo guard de governança em runtime e os mesmos
artefatos forenses (`evidence/<task>/events.jsonl`, `tool_calls.md`, `execution_report.md`).

O guard de governança em `runtime.pre_open` (spec-hash/PRD-first) aborta a sessão antes de abrir
se `tasks.md` tiver hash divergente ou ausente. Bypass granular com `--skip-drift-guard`. A sessão
encerra com segurança por watchdog de inatividade (`--activity-timeout`, padrão 120s) e cap
absoluto de sessão (5x o activity-timeout), com kill do grupo de processos no teardown.

### Permissão de escrita: `--access-mode full`

Por padrão (`--access-mode restricted`), o agente roda em modo restrito e pede permissão para
escrever arquivos. Sem aprovação, a sessão não altera o repositório. Para que o agente implemente
de fato, use `--access-mode full`:

| CLI | O que `--access-mode full` aplica |
| --- | --- |
| claude | `--bypass-permissions` no claude-agent-acp |
| codex | `-c approval_policy=never -c sandbox_mode=danger-full-access` |
| copilot | auto-aprovação das permissões via ACP |
| opencode | não se aplica; controlado declarativamente pelo bloco `permission` do `opencode.json` |

`--access-mode full` dá ao agente acesso pleno ao filesystem (e à rede, no caso do codex). Use
apenas em ambiente isolado e confiável.

### Como usar por ferramenta

Todos os comandos rodam dentro do repositório alvo já instrumentado (`ai-spec install`) e com um
PRD válido (`prd.md` + `techspec.md` + `tasks.md` com spec-hash sincronizado).

```bash
# Claude: requer claude-agent-acp no PATH (ou fallback via npx) e login/API key
ai-spec task-loop --tool claude --runtime acp --access-mode full .specs/prd-<slug>

# Codex: requer codex-acp no PATH (ou fallback via npx) e login do Codex
ai-spec task-loop --tool codex --runtime acp --access-mode full .specs/prd-<slug>

# Copilot: requer copilot --acp no PATH (ou fallback via npx) e gh auth login
ai-spec task-loop --tool copilot --runtime acp --access-mode full .specs/prd-<slug>

# OpenCode: requer opencode acp no PATH (ou fallback via npx) e login do OpenCode
ai-spec task-loop --tool opencode --runtime acp --access-mode full .specs/prd-<slug>
```

Parâmetros úteis, válidos para qualquer CLI:

```bash
# uma tarefa por vez, watchdog curto, modo silencioso
ai-spec task-loop --tool codex --runtime acp --access-mode full \
  --max-iterations 1 --activity-timeout 2m --quiet .specs/prd-<slug>

# bypass granular do guard de spec-drift (mantém governança/budget de contexto ativos)
ai-spec task-loop --tool claude --runtime acp --access-mode full \
  --skip-drift-guard .specs/prd-<slug>
```

Para cada CLI, o runtime resolve o binário nesta ordem: direto no `PATH` (recomendado para
produção), fallback `npx --yes <pacote>@<versão-pinada>` (requer `npx`/Node e internet no
primeiro uso), ou falha com mensagem acionável se nenhum dos dois estiver disponível. As versões
npm são pinadas em `internal/runtime/specs/*.go`, nunca `@latest`.

ADRs relacionadas: [ADR-009 (ACP via coder/acp-go-sdk)](.specs/adr/009-acp-protocol-adoption.md),
[ADR-012 (Copilot ACP)](.specs/adr/012-copilot-cli-acp-native.md),
[ADR-013 (Codex ACP)](.specs/adr/013-codex-cli-acp-native.md),
[ADR-020 (OpenCode ACP)](.specs/adr/020-opencode-acp-subcomando.md). Para migrar do modo legado,
veja o [Guia de migração legacy -> ACP](docs/migracao-legacy-acp.md).

## Documentação de referência

- [Playbook Mestre de Desenvolvimento](docs/development-playbook.md): árvore de decisão para
  escolher entre `create-prd`, `create-technical-specification`, `create-tasks`, `execute-task`,
  `task-loop` e `execute-all-tasks`.
- [Scorecard de Qualidade e Confiança](docs/quality-scorecard.md): critério objetivo para
  classificar o bundle em `pass`, `warning` ou `fail`.
- [Checklist de Preflight e Readiness](docs/preflight-checklist.md): gates bloqueantes e checks
  recomendados antes de executar.
- [Biblioteca de Prompts](docs/prompt-library.md): prompts copiáveis por etapa.
- [Matriz de Confiabilidade por Ferramenta](docs/tool-reliability-matrix.md): papel recomendado,
  risco operacional e custo de contexto por ferramenta.
- [Guia de uso das skills](docs/skills-usage-guide.md): contrato detalhado por skill, entradas
  obrigatórias, prompts mandatórios e critérios de aceite.
- [Guia do task-loop](docs/task-loop-reference.md): flags, heurísticas e comparativos.
- [Guia de resolução de problemas](docs/troubleshooting.md): problemas comuns com sintoma, causa,
  solução e verificação.
- [Ciclo de feedback por telemetria](docs/telemetry-feedback-cycle.md): como evoluir o SDD com
  dados reais (`GOVERNANCE_TELEMETRY=1`).
- [Capacidades do runtime Claude](docs/runtime-claude-capabilities.md).
- [Gates de evidência](docs/evidence-gates.md).
- [Bundle canônico de referência](.specs/prd-example-preflight-entrypoint/): exemplo auditável
  ponta a ponta, com `prd.md`, `techspec.md`, `tasks.md`, `task-*.md` e relatórios de execução.
- Releases: <https://github.com/JailtonJunior94/orchestrator/releases>
- Homebrew Tap: <https://github.com/JailtonJunior94/homebrew-tap>

## Para quem mantém este repositório

### Desenvolvimento local

```bash
go test ./...
go run . --help
go run . install ../sandbox --source . --tools codex --langs go --dry-run
```

Antes de abrir um PR:

```bash
go test ./...
go run . validate .agents/skills
go run . lint .
```

Áreas onde contribuições são bem-vindas: novas skills de linguagem, melhorias de adaptadores por
ferramenta, validações adicionais em `lint`/`doctor`/`metrics`, e exemplos de fluxos reais em
repositórios Go, Node e Python.

### Criar ou mover a tag `setup-action-v1`

A Action `setup-ai-spec` é referenciada por workflows consumidores via tag móvel
`setup-action-v1`. Depois de qualquer release que altere a Action, mova a tag para o novo SHA:

```bash
git tag -f setup-action-v1 <sha> && git push -f origin setup-action-v1
```

Substitua `<sha>` pelo commit que contém a versão da Action a ser publicada.
