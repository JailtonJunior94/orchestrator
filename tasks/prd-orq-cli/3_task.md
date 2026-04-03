# Tarefa 3.0: Workflow (Parser, Validator, Template)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o módulo de workflows: parser YAML, validator, template resolver e catálogo com o dev-workflow embarcado via `embed.FS`.

<requirements>
- Parser YAML usando `gopkg.in/yaml.v3`
- Validator que rejeita workflows inválidos antes da execução (fail fast)
- TemplateResolver para `{{input}}` e `{{steps.<name>.output}}`
- Catálogo de workflows built-in via `embed.FS`
- Interfaces conforme tech spec: Parser, Validator, TemplateResolver
- Testes table-driven para YAML válido/inválido e templates
</requirements>

## Subtarefas

- [ ] 3.1 Definir structs de Workflow e Step para deserialização YAML em `internal/workflows/model.go`
- [ ] 3.2 Implementar Parser em `internal/workflows/parser.go` usando yaml.v3
- [ ] 3.3 Implementar Validator em `internal/workflows/validator.go`: validar name obrigatório, steps não vazios, providers válidos, referências de steps existentes, templates resolvíveis
- [ ] 3.4 Implementar TemplateResolver em `internal/workflows/template.go`: resolver `{{input}}` e `{{steps.<name>.output}}`
- [ ] 3.5 Criar arquivo `workflows/dev-workflow.yaml` e embarcar via `embed.FS` em `internal/workflows/catalog.go`
- [ ] 3.6 Testes table-driven para parser (YAML válido, inválido, malformado)
- [ ] 3.7 Testes table-driven para validator (referências inválidas, providers inexistentes, steps duplicados)
- [ ] 3.8 Testes para template resolver (variáveis existentes, inexistentes, aninhadas)

## Detalhes de Implementação

Referir seções "Interfaces Chave" (Parser, Validator, TemplateResolver) e "Workflow YAML Schema" em `techspec.md`.

- O dev-workflow deve seguir exatamente o schema YAML definido na tech spec
- Variáveis de template: `{{input}}` para input do workflow, `{{steps.<name>.output}}` para output de steps anteriores
- Validação deve ocorrer antes de iniciar execução (conforme R-DDD-001 fail fast)
- Estrutura YAML extensível para suportar novos campos futuros sem breaking changes (F2.7)

## Critérios de Sucesso

- Parse do dev-workflow.yaml embarcado funciona
- Validação rejeita YAML com steps referenciando steps inexistentes
- Validação rejeita providers não registrados
- Template resolver interpola corretamente `{{input}}` e `{{steps.X.output}}`
- Template resolver retorna erro para variáveis inexistentes
- `go test ./internal/workflows/...` passa

## Testes da Tarefa

- [ ] Testes unitários table-driven para parser
- [ ] Testes unitários table-driven para validator
- [ ] Testes unitários para template resolver
- [ ] Teste de integração: parse do dev-workflow embarcado

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/workflows/model.go`
- `internal/workflows/parser.go`
- `internal/workflows/validator.go`
- `internal/workflows/template.go`
- `internal/workflows/catalog.go`
- `workflows/dev-workflow.yaml`
- `internal/workflows/*_test.go`
