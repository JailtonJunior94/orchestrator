# TASK-06: Suporte opcional a tiktoken para métricas

## Contexto

O script remoto `scripts/context-metrics.py` usa `tiktoken` (tokenizer cl100k_base) quando
disponível para contagem precisa de tokens. Quando ausente, fallback para `chars / 3.5`.

O CLI Go `internal/metrics/cost.go` usa apenas `chars / 3.5` e documenta a divergência
de ~15% em `CostNote`. Os gates de regressão (`regression_test.go`, `budget_test.go`) usam
essa estimativa.

## Objetivo

Adicionar suporte opcional a tokenização precisa via binding Go de tiktoken, ativável por
flag `--precise`.

## Análise de trade-offs

| Aspecto | chars/3.5 (atual) | tiktoken-go |
|---------|-------------------|-------------|
| Precisão | ~85% | ~99% |
| Dependência | nenhuma | `github.com/pkoukk/tiktoken-go` (~5MB download inicial de BPE) |
| Performance | O(1) por arquivo | O(n) tokenização real |
| Reprodutibilidade | determinístico | determinístico |

Para gates de regressão (comparação relativa), `chars/3.5` é suficiente.
Para reports absolutos (ex: "quanto custa uma sessão em tokens"), tiktoken é mais preciso.

## Subtarefas

- [x] **6.1** Avaliar binding Go de tiktoken
  - Testar `github.com/pkoukk/tiktoken-go` ou `github.com/tiktoken-go/tokenizer`
  - Verificar: suporte a cl100k_base, performance, tamanho do módulo
  - Decidir se vale adicionar a dependência

- [x] **6.2** Definir interface de tokenização
  - Criar interface `Tokenizer` em `internal/metrics/`
  - Implementação default: `CharEstimator` (chars/3.5)
  - Implementação precisa: `TiktokenEstimator` (cl100k_base)

- [x] **6.3** Implementar `TiktokenEstimator`
  - Usar o binding Go escolhido
  - Lazy-load do modelo BPE (só carrega se `--precise` for usado)
  - Fallback para `CharEstimator` se o modelo não puder ser carregado

- [x] **6.4** Adicionar flag `--precise` ao comando `metrics`
  - Em `cmd/ai_spec_harness/metrics.go`, adicionar flag booleana
  - Passar tokenizer correto para `metrics.Service`

- [x] **6.5** Testes
  - Testar `CharEstimator` (determinístico, sem dependência externa)
  - Testar `TiktokenEstimator` com string conhecida (se binding disponível)
  - Testar fallback quando tiktoken não está disponível

- [x] **6.6** Atualizar `CostNote` e documentação
  - Remover ou condicionalizar a nota de divergência
  - Documentar `--precise` no help do comando

## Arquivos afetados

- `go.mod` (nova dependência)
- `internal/metrics/cost.go` (interface Tokenizer, implementações)
- `internal/metrics/metrics.go` (aceitar Tokenizer injetado)
- `internal/metrics/metrics_test.go` (testes com ambos estimadores)
- `cmd/ai_spec_harness/metrics.go` (flag --precise)

## Critério de conclusão

1. `ai-spec metrics <path>` continua usando chars/3.5 por default
2. `ai-spec metrics --precise <path>` usa tiktoken cl100k_base
3. Divergência entre default e precise é documentada no output
4. `go test ./internal/metrics/...` passa
5. Gates de regressão não são afetados (continuam usando estimativa default)
