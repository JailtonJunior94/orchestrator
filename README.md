# ai-spec-harness

CLI em Go para instalar, validar, inspecionar e atualizar governanca operacional para ferramentas de IA em repositorios de software.

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

Existem dois caminhos praticos:

1. usar um binario de release publicado
2. instalar via `go install`

Observacao importante:

- os binarios gerados por release se chamam `ai-spec`
- a instalacao via `go install github.com/JailtonJunior94/ai-spec-harness@latest` gera o executavel `ai-spec-harness`

Se quiser padronizar o nome no seu ambiente, crie um alias ou renomeie o binario localmente.

### macOS com Homebrew

O release esta configurado para publicacao via Homebrew Cask.

```bash
brew tap JailtonJunior94/homebrew-tap
brew install --cask ai-spec
ai-spec version
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

### Executar sem instalar

```bash
go run . --help
```

## Inicio rapido

Exemplo usando este repositorio como fonte de governanca e um servico Go como destino:

```bash
ai-spec-harness install ../api-pagamentos \
  --source . \
  --tools claude,gemini,codex,copilot \
  --langs go
```

Depois valide o estado da instalacao:

```bash
ai-spec-harness inspect ../api-pagamentos
ai-spec-harness doctor ../api-pagamentos
ai-spec-harness lint ../api-pagamentos
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
| `scaffold` | Cria a estrutura inicial de uma nova skill de linguagem |
| `uninstall` | Remove artefatos instalados pelo CLI |
| `completion` | Gera scripts de autocompletion para shell |
| `version` | Exibe versao, commit e data de build |

### Exemplos por comando

```bash
# instalar governanca em um projeto
ai-spec-harness install ../api-pagamentos --source . --tools codex,claude --langs go

# inspecionar e diagnosticar
ai-spec-harness inspect ../api-pagamentos
ai-spec-harness doctor ../api-pagamentos

# verificar governanca gerada
ai-spec-harness lint ../api-pagamentos

# atualizar instalacao
ai-spec-harness upgrade ../api-pagamentos --source . --langs go

# apenas checar se existe upgrade pendente
ai-spec-harness upgrade ../api-pagamentos --source . --check

# validar todas as skills do repositorio fonte
ai-spec-harness validate .agents/skills

# validar bugs.json contra bug-schema.json
ai-spec-harness validate-bugs ./bugs.json

# medir custo de contexto em JSON
ai-spec-harness metrics . --format json

# registrar telemetria de skill
GOVERNANCE_TELEMETRY=1 ai-spec-harness telemetry log create-prd
ai-spec-harness telemetry summary

# criar scaffold para uma nova linguagem
ai-spec-harness scaffold rust --root .

# remover a instalacao
ai-spec-harness uninstall ../api-pagamentos --dry-run
```

## Fluxo completo: PRD -> Tech Spec -> Tasks -> Execucao

O `ai-spec-harness` nao gera PRD ou implementa endpoint sozinho. Ele prepara o repositorio alvo para que a ferramenta de IA escolhida execute esses fluxos com skills e adaptadores canonicos.

### 1. Instale a governanca no projeto alvo

```bash
ai-spec-harness install ../api-pagamentos \
  --source . \
  --tools codex,claude,gemini,copilot \
  --langs go
```

### 2. Entre no repositorio instrumentado

```bash
cd ../api-pagamentos
```

### 3. Gere o PRD

No agente de sua escolha, peca explicitamente o fluxo `create-prd`.

Exemplo de prompt:

```text
Use a skill create-prd para propor um PRD de listagem de pagamentos.
Objetivo: expor GET /payments com filtros por status, pagina e periodo.
O resultado deve cobrir problema, objetivos, nao objetivos, regras de negocio,
contrato de API, criterios de aceite e riscos.
```

### 4. Gere a especificacao tecnica

```text
Use a skill create-technical-specification com base no PRD aprovado.
Considere um servico Go com arquitetura HTTP -> service -> repository.
Defina contrato, validacoes, estrategia de paginacao, observabilidade, testes,
migracoes necessarias e riscos de compatibilidade.
```

### 5. Gere o bundle de tasks

```text
Use a skill create-tasks para decompor a tech spec em tarefas implementaveis.
Quero tasks pequenas, com evidencias de validacao e ordem de execucao.
```

## Exemplo ponta a ponta: feature de listagem de pagamentos em Go

Um caso realista para exercitar o fluxo acima e criar `GET /payments` em um servico Go e mostrar o que cada etapa produz.

### 1. Exemplo de PRD

Saida esperada do fluxo `create-prd`:

```md
# PRD: Listagem de pagamentos

## Problema
Times operacionais nao conseguem consultar pagamentos por status e intervalo de datas
sem acessar diretamente o banco ou usar relatorios manuais.

## Objetivo
Expor `GET /payments` para listar pagamentos com filtros e paginacao.

## Nao objetivos
- criar pagamento
- atualizar pagamento
- exportar CSV

## Regras de negocio
- aceitar filtros opcionais por `status`, `from` e `to`
- pagina padrao `1`
- `page_size` padrao `20`, maximo `100`
- retornar itens ordenados do mais recente para o mais antigo
- responder erro `400` para query params invalidos

## Contrato HTTP
`GET /payments?status=paid&page=1&page_size=20&from=2026-04-01&to=2026-04-30`

## Criterios de aceite
- listar pagamentos paginados
- aplicar filtros informados
- retornar payload JSON consistente
- cobrir cenarios invalidos com testes
```

### 2. Exemplo de tech spec

Saida esperada do fluxo `create-technical-specification`:

```md
# Tech Spec: GET /payments

## Arquitetura
Fluxo `handler -> service -> repository`.

## Handler
- parsear query params
- validar `page`, `page_size`, `from`, `to`
- responder `400` em caso de entrada invalida

## Service
- montar `ListPaymentsInput`
- delegar busca ao repository
- garantir ordenacao por `created_at DESC`

## Repository
- executar consulta paginada
- aplicar filtros opcionais por status e intervalo de datas
- retornar lista e total

## Testes
- teste de handler para request valida
- teste de handler para `page` invalida
- teste de handler para `page_size > 100`
- teste de service para propagacao correta de filtros
- teste de repository para paginacao e ordenacao
```

### 3. Exemplo de bundle gerado por `create-tasks`

Estrutura esperada:

```text
tasks/
  prd-payments-list/
    tasks.md
    01_task.md
    02_task.md
    03_task.md
    04_task.md
```

Exemplo de `tasks.md`:

```md
# Tasks - payments list

1. Criar contrato HTTP e validacao de query params para `GET /payments`
2. Implementar service de listagem paginada
3. Implementar repository com filtros por status e periodo
4. Adicionar testes de handler e service
```

Exemplo de `01_task.md`:

```md
# Task 01 - Handler de listagem de pagamentos

## Objetivo
Implementar o endpoint `GET /payments` com parse e validacao de query params.

## Escopo
- criar handler HTTP
- validar `page`, `page_size`, `from`, `to`
- retornar `400` para entradas invalidas
- serializar resposta JSON

## Evidencias
- teste automatizado para request valida
- teste automatizado para query param invalido
```

Exemplo de `02_task.md`:

```md
# Task 02 - Service de listagem

## Objetivo
Implementar a camada de service responsavel por orquestrar a consulta paginada.

## Escopo
- definir `ListPaymentsInput`
- delegar ao repository
- manter regras de paginacao e ordenacao
```

Exemplo de `03_task.md`:

```md
# Task 03 - Repository de pagamentos

## Objetivo
Consultar pagamentos com filtros opcionais e totalizacao.

## Escopo
- aplicar filtro por `status`
- aplicar intervalo `from` e `to`
- suportar `limit` e `offset`
- retornar `items` e `total`
```

### 4. Exemplo de execucao com `execute-task`

Pedidos tipicos ao agente:

```text
Use a skill execute-task para implementar a task 01 do bundle tasks/prd-payments-list.
Rode validacao proporcional e gere um resumo objetivo da alteracao.
```

```text
Use a skill execute-task para implementar a task 02 do bundle tasks/prd-payments-list.
Mantenha a arquitetura handler -> service -> repository e nao quebre endpoints existentes.
```

```text
Use a skill execute-task para implementar a task 03 do bundle tasks/prd-payments-list.
Inclua testes proporcionais para repository e service.
```

### 5. Resultado esperado apos executar as tasks

#### Contrato HTTP

```http
GET /payments?status=paid&page=1&page_size=20&from=2026-04-01&to=2026-04-30
```

#### Exemplo de response

```json
{
  "items": [
    {
      "id": "pay_123",
      "amount": 12500,
      "currency": "BRL",
      "status": "paid",
      "created_at": "2026-04-18T15:04:05Z"
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1
}
```

#### Exemplo de implementacao em Go

```go
package payments

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type Payment struct {
	ID        string    `json:"id"`
	Amount    int64     `json:"amount"`
	Currency  string    `json:"currency"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ListPaymentsInput struct {
	Status   string
	Page     int
	PageSize int
	From     *time.Time
	To       *time.Time
}

type ListPaymentsOutput struct {
	Items    []Payment `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int       `json:"total"`
}

type Repository interface {
	ListPayments(ctx context.Context, input ListPaymentsInput) ([]Payment, int, error)
}

type Service struct {
	Repo Repository
}

func (s Service) ListPayments(ctx context.Context, input ListPaymentsInput) (ListPaymentsOutput, error) {
	items, total, err := s.Repo.ListPayments(ctx, input)
	if err != nil {
		return ListPaymentsOutput{}, err
	}

	return ListPaymentsOutput{
		Items:    items,
		Page:     input.Page,
		PageSize: input.PageSize,
		Total:    total,
	}, nil
}

type Handler struct {
	Service Service
}

func (h Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	input, err := parseListPaymentsInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out, err := h.Service.ListPayments(r.Context(), input)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func parseListPaymentsInput(r *http.Request) (ListPaymentsInput, error) {
	q := r.URL.Query()

	page := 1
	if raw := q.Get("page"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			return ListPaymentsInput{}, errInvalidQuery("page")
		}
		page = v
	}

	pageSize := 20
	if raw := q.Get("page_size"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > 100 {
			return ListPaymentsInput{}, errInvalidQuery("page_size")
		}
		pageSize = v
	}

	var from *time.Time
	if raw := q.Get("from"); raw != "" {
		v, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return ListPaymentsInput{}, errInvalidQuery("from")
		}
		from = &v
	}

	var to *time.Time
	if raw := q.Get("to"); raw != "" {
		v, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return ListPaymentsInput{}, errInvalidQuery("to")
		}
		to = &v
	}

	return ListPaymentsInput{
		Status:   q.Get("status"),
		Page:     page,
		PageSize: pageSize,
		From:     from,
		To:       to,
	}, nil
}

type queryError struct {
	Field string
}

func (e queryError) Error() string {
	return "invalid query parameter: " + e.Field
}

func errInvalidQuery(field string) error {
	return queryError{Field: field}
}
```

### 6. Verifique o estado final

```bash
ai-spec-harness lint .
go test ./...
```

## Estrategias de instalacao

### `symlink`

Melhor para desenvolvimento da governanca, porque o projeto alvo passa a refletir alteracoes feitas na fonte.

```bash
ai-spec-harness install ../api-pagamentos \
  --source . \
  --tools all \
  --langs all \
  --mode symlink
```

### `copy`

Melhor quando o ambiente nao lida bem com links simbolicos ou quando voce quer snapshot fisico do baseline.

```bash
ai-spec-harness install ../api-pagamentos \
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
