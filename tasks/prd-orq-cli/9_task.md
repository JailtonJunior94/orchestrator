# Tarefa 9.0: HITL (Prompt Interativo)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o sistema de Human-in-the-Loop: prompt interativo no terminal com ações approve/edit/redo/exit e integração com editor externo.

<requirements>
- Interface Prompter conforme tech spec: Prompt(ctx, output) → (Action, error)
- Ações: Approve, Edit, Redo, Exit (conforme R-GOV-001)
- Terminal adapter com rendering do output e menu de ações
- Integração com Editor de `internal/platform/` para ação Edit
- Prompt deve aparecer em < 2s após output do provider (F6.7)
- Fake Prompter para testes do engine
</requirements>

## Subtarefas

- [ ] 9.1 Definir interface Prompter e tipo Action em `internal/hitl/prompter.go`
- [ ] 9.2 Implementar TerminalPrompter em `internal/hitl/terminal.go`: renderizar output, exibir menu `[A] Aprovar [E] Editar [R] Refazer [S] Sair`, ler input
- [ ] 9.3 Implementar ação Edit: gravar output em temp → abrir editor via platform.Editor → ler conteúdo editado → retornar como output modificado
- [ ] 9.4 Implementar FakePrompter em `internal/hitl/fake.go` para testes (sequência de ações pré-definidas)
- [ ] 9.5 Testes de TerminalPrompter com stdin simulado (cada ação)
- [ ] 9.6 Testes de integração da ação Edit com editor fake

## Detalhes de Implementação

Referir seções "Interfaces Chave" (Prompter) e "F6. Human-in-the-Loop" no PRD.

- TerminalPrompter lê de `io.Reader` (stdin injetável para testes)
- Output do step deve ser renderizado antes do prompt (separar conteúdo de prompt — R-CLI-001)
- Edit precisa retornar o conteúdo editado para que o engine substitua o output do step
- Prompter interface pode retornar `(Action, string, error)` onde string é o output editado (quando Edit)
- FakePrompter aceita slice de ações no construtor, retorna na ordem

## Critérios de Sucesso

- Prompt exibe menu correto após output
- Cada ação (A, E, R, S) é reconhecida corretamente
- Input inválido solicita nova entrada
- Edit abre editor e retorna conteúdo modificado
- FakePrompter funciona para testes do engine
- `go test ./internal/hitl/...` passa

## Testes da Tarefa

- [ ] Testes com stdin simulado: ação Approve
- [ ] Testes com stdin simulado: ação Redo
- [ ] Testes com stdin simulado: ação Exit
- [ ] Testes com stdin simulado: input inválido → re-prompt
- [ ] Teste de integração: Edit com editor fake
- [ ] Teste FakePrompter retorna ações na ordem

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/hitl/prompter.go`
- `internal/hitl/terminal.go`
- `internal/hitl/fake.go`
- `internal/hitl/*_test.go`
