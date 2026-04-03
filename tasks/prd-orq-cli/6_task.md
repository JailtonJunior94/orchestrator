# Tarefa 6.0: Output (Extração, Correção, Validação)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar processamento de output dos providers: extração de JSON de Markdown, correção automática de JSON malformado e validação.

<requirements>
- Extrair blocos JSON embutidos em output Markdown dos providers
- Correção automática de JSON com problemas comuns (trailing commas, aspas incorretas)
- Validação de JSON via parse (sintaxe válida)
- Validação via JSON Schema quando schema estiver definido para o step
- Separar output Markdown (legível) do JSON (estruturado)
- Indicar se output é corrigível para permitir retry automático
</requirements>

## Subtarefas

- [ ] 6.1 Implementar extrator de JSON de Markdown em `internal/output/extractor.go` (detectar blocos ```json``` e JSON inline)
- [ ] 6.2 Implementar corretor automático em `internal/output/fixer.go` (trailing commas, aspas, formato)
- [ ] 6.3 Implementar validador em `internal/output/validator.go` (parse JSON + JSON Schema opcional)
- [ ] 6.4 Implementar OutputProcessor que compõe extração → correção → validação em `internal/output/processor.go`
- [ ] 6.5 Testes table-driven para extrator (JSON em code block, inline, sem JSON, múltiplos blocos)
- [ ] 6.6 Testes table-driven para corretor (trailing commas, aspas, já válido, incorrigível)
- [ ] 6.7 Testes para validador (JSON válido, inválido, com schema, sem schema)
- [ ] 6.8 Testes de integração do OutputProcessor completo

## Detalhes de Implementação

Referir seção "F5. Output Híbrido e Validação" no PRD e "OutputProcessor" na tech spec.

- Extração: buscar blocos ````json ... ```` primeiro, depois tentar detectar JSON inline `{...}`
- Correção automática (F5.5): trailing commas, aspas simples → duplas, newlines em strings
- Se correção falhar, sinalizar para que engine faça retry com provider (máx 2 retries — F5.6)
- Se retry falhar, pausar para HITL (F5.7) — essa lógica fica no engine, não aqui
- `{{steps.<name>.output}}` resolve para Markdown completo (decisão técnica #9 da tech spec)

## Critérios de Sucesso

- Extrai JSON corretamente de Markdown com code blocks
- Corrige JSON com trailing commas
- Valida JSON sintaticamente
- OutputProcessor compõe todo o fluxo
- `go test ./internal/output/...` passa

## Testes da Tarefa

- [ ] Testes unitários table-driven para extrator
- [ ] Testes unitários table-driven para corretor
- [ ] Testes unitários para validador
- [ ] Testes de integração do OutputProcessor

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/output/extractor.go`
- `internal/output/fixer.go`
- `internal/output/validator.go`
- `internal/output/processor.go`
- `internal/output/*_test.go`
