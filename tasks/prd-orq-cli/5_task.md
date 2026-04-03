# Tarefa 5.0: Providers (Claude CLI, Copilot CLI)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar a interface Provider e os adapters para Claude CLI e Copilot CLI, usando subprocess via CommandRunner.

<requirements>
- Interface Provider conforme tech spec: Name(), Execute(), Available()
- Adapter Claude CLI com invocação resolvida pelo adapter conforme capability e versão detectada
- Adapter Copilot CLI com invocação resolvida pelo adapter conforme capability e versão detectada
- ProviderFactory para criar provider a partir do nome no YAML
- Validação de binário no PATH via `exec.LookPath`
- Captura de stdout, stderr, exit code e duração
- Timeout configurável por step
- Testes com TestHelperProcess (sem rede, sem CLIs reais)
</requirements>

## Subtarefas

- [ ] 5.1 Definir interface Provider em `internal/providers/provider.go` com tipos ProviderInput e ProviderOutput
- [ ] 5.2 Implementar adapter Claude CLI em `internal/providers/claude.go`
- [ ] 5.3 Implementar adapter Copilot CLI em `internal/providers/copilot.go`
- [ ] 5.4 Implementar ProviderFactory em `internal/providers/factory.go`
- [ ] 5.5 Implementar Available() com `exec.LookPath` para cada provider
- [ ] 5.6 Testes de Claude adapter com TestHelperProcess (sucesso, erro, timeout)
- [ ] 5.7 Testes de Copilot adapter com TestHelperProcess (sucesso, erro, timeout)
- [ ] 5.8 Testes de ProviderFactory (provider válido, inválido)

## Detalhes de Implementação

Referir seções "Interfaces Chave" (Provider) e "Pontos de Integração" (Claude CLI, Copilot CLI) em `techspec.md`.

- Claude CLI: estratégia de invocação deve ser version-aware, com fallback para stdin, argumentos ou arquivo temporário. Timeout default 5min.
- Copilot CLI: estratégia de invocação deve ser version-aware, com capability opcional para execução no filesystem sujeita a discovery por versão/SO. Timeout default 10min.
- Providers dependem de `CommandRunner` de `internal/platform/` (injeção por construtor)
- Provider não tem autonomia para avançar steps (R-ARCH-001)
- Interface extensível para futuros providers (Gemini, Codex) sem alterar engine

## Critérios de Sucesso

- Provider factory retorna adapter correto para "claude" e "copilot"
- Provider factory retorna erro para provider desconhecido
- Available() detecta binário ausente com mensagem acionável
- Execute() captura stdout, stderr, exit code e duração
- Timeout cancela execução corretamente
- `go test ./internal/providers/...` passa

## Testes da Tarefa

- [ ] Testes unitários Claude adapter (sucesso, falha, timeout) via TestHelperProcess
- [ ] Testes unitários Copilot adapter (sucesso, falha, timeout) via TestHelperProcess
- [ ] Testes de ProviderFactory
- [ ] Testes de Available() (binário presente, ausente)

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO DONE</critical>

## Arquivos Relevantes
- `internal/providers/provider.go`
- `internal/providers/claude.go`
- `internal/providers/copilot.go`
- `internal/providers/factory.go`
- `internal/providers/*_test.go`
