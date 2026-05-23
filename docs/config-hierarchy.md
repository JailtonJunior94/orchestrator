# Hierarquia de Configuração — ai-spec-harness

<!-- Gerado pela Tarefa 8.0 do PRD Fundação Portátil -->
<!-- Referência canônica: ADR-016 -->

## Visão Geral

A configuração do `ai-spec-harness` é resolvida em **cascata de 4 camadas**, com precedência
determinística. Cada camada só sobrescreve campos **não-zero** da camada anterior (merge campo a
campo — não substituição de struct):

```
flags CLI  >  workspace (projeto)  >  global (~/.aispec)  >  defaults built-in
```

Implementação: `internal/config/resolver.go` — `config.DefaultResolver.Resolve(cwd, overrides)`.

---

## Camadas de Configuração

### 1. Defaults Built-in (menor precedência)

Valores compilados no binário via `config.DefaultRuntime()`. Garantem comportamento F1 (zero
regressão) quando nenhuma camada superior está presente.

```yaml
# valores default (implícitos — sem arquivo necessário)
tasks_root: .specs
prd_prefix: prd-
evidence_dir: ""       # zero-value => sem subdir fixo
coverage_threshold: 70 # porcentagem; zero-value => 70%
language_default: ""
timeout: ""           # zero-value => sem timeout adicional
max_retries: 0        # zero-value => uma tentativa (comportamento F1)
retry_backoff_multiplier: 0.0  # zero-value => sem espera entre tentativas
concurrent: 0         # zero-value => 1 (sequencial, F1)
batch_size: 0         # zero-value => 1 (F1)
default_tool: ""
```

### 2. Config Global (`~/.aispec/config.yaml`)

Defaults reutilizáveis entre projetos. Resolvido via `os.UserHomeDir()` — nunca hardcoded
(R-SEC-001). Ausência do arquivo ou do `$HOME` é **não-fatal** — o harness continua com
defaults built-in.

```yaml
# ~/.aispec/config.yaml (exemplo)
tasks_root: .specs
prd_prefix: prd-
evidence_dir: evidence
coverage_threshold: 0.80
default_tool: claude
timeout: 5m
max_retries: 2
retry_backoff_multiplier: 1.5
concurrent: 2
batch_size: 4
```

> **Nota:** O arquivo é lido mas **não criado automaticamente**. Crie manualmente para definir
> seus defaults globais.

### 3. Config de Projeto (workspace)

Arquivo encontrado por **upward-walk** a partir do diretório de trabalho atual (CWD), procurando
na ordem:

1. `.aispec/config.yaml`
2. `.claude/config.yaml`
3. `.agents/config.yaml`

O walk para no primeiro diretório que contém um dos marcadores de projeto:
`.git/`, `.aispec/`, `.claude/`, `.agents/`.

Campos presentes no arquivo do projeto sobrescrevem a config global campo a campo.

```yaml
# .claude/config.yaml (exemplo de projeto)
tasks_root: .specs
prd_prefix: prd-
evidence_dir: evidence/tasks
coverage_threshold: 0.90  # sobrescreve o global (0.80)
```

### 4. Flags CLI (maior precedência)

Flags passadas diretamente na linha de comando sobrescrevem todas as camadas. Aplicadas como
`overrides` no `Resolver.Resolve(cwd, overrides)`.

```bash
# Exemplo: sobrescrever tool e timeout em runtime
ai-spec-harness task-loop --tool gemini --timeout 10m .specs/prd-meu-prd
```

---

## Exemplo Completo de Resolução

```
built-in:            coverage_threshold=70, concurrent=1, max_retries=0
global (~/.aispec):  coverage_threshold=0.80, concurrent=2, max_retries=2
projeto (.claude/):  coverage_threshold=0.90  (apenas este campo)
flags CLI:           --timeout 3m             (adiciona campo)

Resultado final:
  coverage_threshold = 0.90   ← projeto vence global
  concurrent         = 2      ← global (projeto não declara)
  max_retries        = 2      ← global (projeto não declara)
  timeout            = 3m     ← flags (vence tudo)
```

---

## Arquivo de Configuração (`config.yaml`)

Formato **YAML**. Consistente com `.claude/config.yaml` e `.agents/config.yaml` existentes.

### Chaves disponíveis

| Chave | Tipo | Padrão | Descrição |
|-------|------|--------|-----------|
| `tasks_root` | string | `.specs` | Diretório raiz dos artefatos SDD |
| `prd_prefix` | string | `prd-` | Prefixo de diretórios PRD |
| `evidence_dir` | string | `evidence` | Diretório de evidências |
| `coverage_threshold` | float | `0.75` | Cobertura mínima de testes (0–1) |
| `language_default` | string | `""` | Linguagem padrão quando não detectada |
| `timeout` | string | `""` | Timeout por sessão ACP (ex.: `5m`, `30s`) |
| `max_retries` | int | `0` | Tentativas extras em falha transitória (0 = sem retry) |
| `retry_backoff_multiplier` | float | `0.0` | Fator de espera exponencial entre tentativas |
| `concurrent` | int | `0` | Grau de paralelismo no runloop (0 ou 1 = sequencial) |
| `batch_size` | int | `0` | Tamanho do lote de tasks por iteração (0 ou 1 = sem lote) |
| `default_tool` | string | `""` | Ferramenta padrão (claude, codex, gemini, copilot) |

### Exemplo completo

```yaml
# .aispec/config.yaml ou .claude/config.yaml
tasks_root: .specs
prd_prefix: prd-
evidence_dir: evidence
coverage_threshold: 0.80
language_default: go
timeout: 5m
max_retries: 2
retry_backoff_multiplier: 1.5
concurrent: 2
batch_size: 4
default_tool: claude
```

---

## Upward-Walk — Comportamento Detalhado

O `DefaultResolver` caminha do CWD até a raiz do sistema de arquivos:

```
/home/user/projetos/meu-repo/src/feature/
    └── não tem .claude/config.yaml → sobe
/home/user/projetos/meu-repo/src/
    └── não tem .claude/config.yaml → sobe
/home/user/projetos/meu-repo/
    └── tem .claude/config.yaml ← PARA AQUI, usa este arquivo
```

**Limites do walk:**
- Encontra um marcador de projeto (`.git/`, `.aispec/`, `.claude/`, `.agents/`).
- Chega à raiz do sistema de arquivos.

O walk nunca escapa do repositório além dos marcadores de projeto.

---

## Compatibilidade Retroativa (RF-16)

Sem arquivo `~/.aispec/config.yaml` e executando a partir da raiz do repositório:

```
Resolver.Resolve(repoRoot, Runtime{}) == LoadRuntime(repoRoot)  [byte-idêntico]
```

O comportamento é **idêntico ao atual** (zero regressão).

A função `LoadRuntime(repoRoot)` continua disponível como wrapper fino sobre o `Resolver`,
para não quebrar chamadores existentes.

---

## Referências

- [ADR-016](../.specs/prd-fundacao-portatil/adr-016-config-hierarquico-universal.md) — Decisão arquitetural da config hierárquica
- [Guia de Instalação Universal](guia-instalacao-universal.md) — Bootstrap portátil
- `internal/config/resolver.go` — Implementação do `DefaultResolver`
- `internal/config/runtime.go` — Tipo `Runtime` e `LoadRuntime` (wrapper de compat)
