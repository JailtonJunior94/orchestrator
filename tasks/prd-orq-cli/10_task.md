# Tarefa 10.0: CLI Cobra + Bootstrap

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar os comandos Cobra (run, continue, list), o composition root (bootstrap) e o rendering de progresso no terminal. Esta é a camada final que expõe tudo ao usuário.

<requirements>
- Comando `orq run <workflow>` com flags `--input` e `-f, --file` (F1.1-F1.3)
- Comando `orq continue` para retomar workflow pausado (F1.4)
- Comando `orq list` para listar workflows disponíveis (F1.5)
- Composition root em `internal/bootstrap/` que faz wiring de todas as dependências
- Rendering de progresso: step atual, provider, duração, status (F1.6)
- Mensagens de erro acionáveis na camada de terminal (R-ERR-001)
- Comandos delegam para application services, sem lógica de negócio
</requirements>

## Subtarefas

- [ ] 10.1 Implementar composition root em `internal/bootstrap/bootstrap.go`: criar e conectar todas as dependências (engine, providers, state, hitl, workflows, platform, logger)
- [ ] 10.2 Implementar comando `run` em `internal/cli/run.go`: flags --input e -f, validação de input, delegação para engine.Run()
- [ ] 10.3 Implementar comando `continue` em `internal/cli/continue.go`: delegação para engine.Continue()
- [ ] 10.4 Implementar comando `list` em `internal/cli/list.go`: listar workflows do catálogo
- [ ] 10.5 Implementar rendering de progresso em `internal/cli/renderer.go`: indicação de step, provider, duração, status com formatação legível
- [ ] 10.6 Registrar comandos no root command em `cmd/orq/main.go`
- [ ] 10.7 Implementar tradução de erros técnicos para mensagens acionáveis na camada CLI
- [ ] 10.8 Testes de integração: comando run com doubles (provider fake, HITL fake)
- [ ] 10.9 Testes de integração: comando continue
- [ ] 10.10 Testes: comando list exibe workflows
- [ ] 10.11 Testes: flags inválidas retornam erro

## Detalhes de Implementação

Referir seções "Visão Geral dos Componentes", "Fluxo de dados principal" e "Experiência do Usuário" em `techspec.md`.

- Composition root instancia tudo: logger (slog), clock, filesystem, command runner, providers (factory), state store, output processor, workflow parser/validator/resolver, engine, HITL prompter
- Comandos Cobra apenas: validar flags → montar request → delegar para service → renderizar resultado
- Run command: `--input` e `-f` são mutuamente exclusivos; pelo menos um é obrigatório
- Rendering: `[1/4] PRD (claude) ⏳ Gerando...` → `✅ Concluído (1.2s)`
- Erros de provider: informar binário, timeout, exit code (R-ERR-001)
- Fluxo: `cmd/orq/ → internal/cli/ → internal/bootstrap/ → engine`

## Critérios de Sucesso

- `orq run dev-workflow --input "teste"` executa pipeline completo com provider fake
- `orq continue` retoma run pausado
- `orq list` exibe "dev-workflow"
- Rendering mostra progresso claro por step
- Erros exibem mensagem acionável
- `go test ./internal/cli/... ./internal/bootstrap/...` passa
- `task build && ./orq --help` funciona

## Testes da Tarefa

- [ ] Teste de integração: run command com doubles
- [ ] Teste de integração: continue command
- [ ] Teste: list command
- [ ] Teste: --input e -f mutuamente exclusivos
- [ ] Teste: input obrigatório (erro se ausente)
- [ ] Teste: workflow não encontrado → mensagem acionável
- [ ] Teste: provider não encontrado no PATH → mensagem acionável

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `cmd/orq/main.go`
- `internal/bootstrap/bootstrap.go`
- `internal/cli/run.go`
- `internal/cli/continue.go`
- `internal/cli/list.go`
- `internal/cli/renderer.go`
- `internal/cli/*_test.go`
- `internal/bootstrap/*_test.go`
