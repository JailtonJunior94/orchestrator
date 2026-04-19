# ai-spec-harness

CLI em Go para instalar, inspecionar, validar, atualizar e remover pacotes de governanca de IA em projetos de software.

O objetivo do `ai-spec-harness` e padronizar como ferramentas como Claude, Gemini, Codex e GitHub Copilot consomem skills, regras e contexto operacional dentro de um repositorio alvo, com suporte a linguagens como Go, Node e Python.

## Quando usar

Use este projeto quando voce precisar:

- instalar um pacote canonico de governanca em outro repositorio
- distribuir skills compartilhadas via `symlink` ou copia
- gerar adaptadores para ferramentas de IA suportadas
- validar frontmatter de `SKILL.md`
- inspecionar e diagnosticar uma instalacao existente
- medir custo estimado de contexto por baseline e fluxo
- criar scaffold para novas skills de linguagem

## O que o CLI faz

| Comando | Finalidade |
| --- | --- |
| `install` | Instala governanca de IA em um projeto alvo |
| `upgrade` | Atualiza skills e metadados da instalacao |
| `inspect` | Mostra o estado atual da instalacao |
| `doctor` | Executa checks de saude e integridade |
| `uninstall` | Remove artefatos instalados pelo CLI |
| `validate` | Valida frontmatter dos `SKILL.md` |
| `metrics` | Calcula metricas de contexto e tokens estimados |
| `scaffold` | Cria a estrutura inicial de uma nova skill |
| `version` | Exibe a versao do binario |

## Requisitos

- Go `1.26.2` ou compativel com o `go.mod`
- `git` disponivel no `PATH` para diagnosticos e fluxos que dependem do repositorio alvo
- um repositorio fonte de governanca contendo `.agents/skills`
- permissao de escrita no projeto alvo

## Instalacao

### Executar sem instalar

Para rodar diretamente a partir deste repositorio:

```bash
go run . --help
```

### Instalar o binario localmente

Para compilar e instalar o binario com o nome do modulo:

```bash
go install github.com/JailtonJunior94/ai-spec-harness@latest
```

Se estiver trabalhando neste checkout local:

```bash
go install .
```

### Compilar manualmente

```bash
go build -o ai-spec-harness .
./ai-spec-harness version
```

## Uso Rapido

O fluxo principal assume dois diretorios:

- um diretorio fonte com as skills e adaptadores canonicos
- um projeto alvo onde a governanca sera instalada

Exemplo usando este repositorio como fonte:

```bash
go run . install ../meu-servico \
  --source . \
  --tools claude,codex \
  --langs go
```

Depois da instalacao, inspecione o resultado:

```bash
go run . inspect ../meu-servico
```

E rode o diagnostico:

```bash
go run . doctor ../meu-servico
```

## Exemplos Reais

### 1. Instalar governanca para um projeto Go com Claude e Gemini

```bash
go run . install ../api-pagamentos \
  --source . \
  --tools claude,gemini \
  --langs go
```

### 2. Simular a instalacao sem alterar arquivos

```bash
go run . install ../api-pagamentos \
  --source . \
  --tools codex \
  --langs go \
  --dry-run
```

### 3. Instalar tudo em modo copia

Use este modo quando o ambiente nao lida bem com `symlink` ou quando voce quer uma copia fisica das skills no projeto alvo.

```bash
go run . install ../monorepo \
  --source . \
  --tools all \
  --langs all \
  --mode copy
```

### 4. Atualizar a instalacao existente

```bash
go run . upgrade ../api-pagamentos \
  --source . \
  --langs go
```

### 5. Verificar upgrades sem modificar arquivos

```bash
go run . upgrade ../api-pagamentos \
  --source . \
  --check
```

### 6. Validar o frontmatter de skills

```bash
go run . validate .agents/skills
```

### 7. Medir custo de contexto em JSON

```bash
go run . metrics . --format json
```

### 8. Gerar scaffold para uma nova skill de linguagem

```bash
go run . scaffold rust --root .
```

### 9. Remover uma instalacao

```bash
go run . uninstall ../api-pagamentos
```

## Estrutura Gerada no Projeto Alvo

Durante o `install`, o CLI pode criar ou atualizar estruturas como:

```text
.agents/skills/
.claude/skills/
.claude/agents/
.claude/rules/
.claude/scripts/
.claude/hooks/
.gemini/commands/
.github/agents/
.github/copilot-instructions.md
.codex/config.toml
.ai_spec_harness.json
```

Os caminhos efetivos variam conforme as ferramentas selecionadas em `--tools`.

## Fluxo Operacional

### `install`

O comando:

1. valida diretorio fonte, diretorio alvo e flags obrigatorias
2. resolve modo `symlink` ou `copy`
3. instala skills canonicas em `.agents/skills`
4. gera adaptadores por ferramenta selecionada
5. tenta gerar governanca contextual, a menos que `--no-context` seja usado
6. persiste o manifesto `.ai_spec_harness.json`

Observacoes importantes:

- `--source` e obrigatorio
- `--tools` e obrigatorio
- `--langs` pode ser omitido, usar `none` ou `all`
- o diretorio alvo nao pode ser o mesmo repositorio fonte

### `inspect`

Mostra:

- dados do manifesto, se existir
- lista de skills instaladas e se cada uma esta em `copy` ou `symlink`
- ferramentas detectadas no projeto
- linguagens detectadas no projeto

### `doctor`

Executa checks sobre:

- repositorio git valido
- existencia de `.agents/skills`
- estado do manifesto
- symlinks quebrados
- permissao de escrita
- disponibilidade do binario `git`

### `metrics`

Calcula metricas de contexto com base em arquivos compartilhados, skills e referencias para estimar custo de tokens por baseline e por fluxo operacional.

## Arquitetura do Projeto

O codigo esta organizado em torno de servicos pequenos e modulos internos:

- [`cmd/ai_spec_harness`](/Users/jailtonjunior/Git/orchestrator/cmd/ai_spec_harness): comandos Cobra e flags publicas do CLI
- [`internal/install`](/Users/jailtonjunior/Git/orchestrator/internal/install): fluxo principal de instalacao
- [`internal/upgrade`](/Users/jailtonjunior/Git/orchestrator/internal/upgrade): comparacao de versoes e atualizacao
- [`internal/inspect`](/Users/jailtonjunior/Git/orchestrator/internal/inspect): leitura e exibicao de estado
- [`internal/doctor`](/Users/jailtonjunior/Git/orchestrator/internal/doctor): checks de saude
- [`internal/adapters`](/Users/jailtonjunior/Git/orchestrator/internal/adapters): geracao de adaptadores por ferramenta
- [`internal/skills`](/Users/jailtonjunior/Git/orchestrator/internal/skills): parsing de skills, linguagens e ferramentas
- [`internal/manifest`](/Users/jailtonjunior/Git/orchestrator/internal/manifest): persistencia do manifesto da instalacao
- [`internal/fs`](/Users/jailtonjunior/Git/orchestrator/internal/fs): abstracao de filesystem para codigo produtivo e testes

## Desenvolvimento

### Rodar testes

```bash
go test ./...
```

### Explorar a ajuda do CLI

```bash
go run . --help
go run . install --help
go run . upgrade --help
go run . validate --help
```

### Gerar release localmente

Existe configuracao em `.goreleaser.yaml` para build multi-plataforma e publicacao. Antes de usar, revise nomes de binario, repositorio de release e tap Homebrew para garantir que correspondem ao estado atual do projeto.

## Notas e Gotchas

- `install` e `upgrade` dependem explicitamente de `--source`; eles nao inferem um repositorio padrao.
- Em plataformas sem suporte nativo a `symlink`, o fluxo faz fallback para `copy`.
- `metrics` usa estimativa simples de tokens baseada em tamanho de texto; trate o valor como aproximacao operacional, nao como contagem exata do modelo.
- `validate` exige `name`, `version` e `description` no frontmatter.
- O README documenta o nome de comando exposto pelo codigo atual: `ai-spec-harness`.

## Estado Atual

- testes executados com sucesso via `go test ./...`
- README revisado em `2026-04-18`

## Proximos Passos Sugeridos

- adicionar exemplos com saida esperada de `inspect` e `doctor`
- documentar o formato do manifesto `.ai_spec_harness.json`
- alinhar naming de release, binario e repositorio publicado se o projeto for distribuido externamente
