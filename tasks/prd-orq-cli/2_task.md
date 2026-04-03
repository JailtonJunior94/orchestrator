# Tarefa 2.0: Domínio (Run, Step, Value Objects)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Modelar as entidades de domínio (Run, StepExecution), value objects (RunStatus, StepStatus, WorkflowName, StepName, ProviderName) e a state machine com transições explícitas. Este é o núcleo do sistema sem dependências externas.

<requirements>
- Run como aggregate root com campos não exportados e métodos de comportamento
- StepExecution com status e transições controladas
- Value objects autovalidados e imutáveis
- State machine com transições explícitas (conforme R-GOV-001)
- Construtores que protegem invariantes (fail fast)
- Testes table-driven para todas as transições válidas e inválidas
</requirements>

## Subtarefas

- [ ] 2.1 Criar value objects em `internal/runtime/domain/values.go`: RunStatus, StepStatus, WorkflowName, StepName, ProviderName com autovalidação
- [ ] 2.2 Criar entidade StepExecution em `internal/runtime/domain/step.go` com transições de status controladas
- [ ] 2.3 Criar entidade Run em `internal/runtime/domain/run.go` como aggregate root com métodos: ApproveStep, RetryStep, Pause, Resume, MarkStepFailed, MarkStepCompleted, CurrentStep
- [ ] 2.4 Implementar factory/constructor NewRun que valida invariantes e inicializa steps a partir de definição do workflow
- [ ] 2.5 Testes table-driven para transições de RunStatus e StepStatus (todas válidas e inválidas)
- [ ] 2.6 Testes para métodos de Run: approve, retry, pause, resume, mark failed, mark completed

## Detalhes de Implementação

Referir seções "Entidades de Domínio" e "Value Objects" em `techspec.md`.

- Estados de run permitidos: `pending`, `running`, `paused`, `failed`, `completed`, `cancelled`
- Estados de step permitidos: `pending`, `running`, `waiting_approval`, `approved`, `retrying`, `failed`, `skipped`
- Transições devem ser explícitas — não permitir saltos arbitrários entre estados
- Campos sensíveis do domínio não exportados (conforme R-DDD-001)
- Erros de domínio como sentinelas (conforme R-ERR-001)

## Critérios de Sucesso

- Nenhum campo público nas entidades que permita estado inválido
- Todas as transições inválidas retornam erro
- Cobertura de todas as transições de estado via table-driven tests
- `go test ./internal/runtime/domain/...` passa
- Domínio sem imports de packages externos (apenas stdlib)

## Testes da Tarefa

- [ ] Testes unitários table-driven para RunStatus transitions
- [ ] Testes unitários table-driven para StepStatus transitions
- [ ] Testes unitários para cada método do Run aggregate
- [ ] Testes de validação dos value objects (inputs válidos e inválidos)
- [ ] Teste de construtor NewRun com invariantes

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/runtime/domain/values.go`
- `internal/runtime/domain/step.go`
- `internal/runtime/domain/run.go`
- `internal/runtime/domain/errors.go`
- `internal/runtime/domain/*_test.go`
