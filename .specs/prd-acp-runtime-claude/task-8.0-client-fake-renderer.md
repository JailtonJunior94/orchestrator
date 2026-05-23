# Tarefa 8.0: ACP Client + Fake Server + Renderer

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar três componentes da camada de infra de transport e output, coesos por afinidade de camada:

1. **`internal/runtime/client/`** — `acpClient` que envolve `coder/acp-go-sdk` para abrir sessão, enviar prompt e expor `<-chan events.Event` consumindo `acp.SessionUpdate` via `events.FromACPUpdate` (entregue em 4.0).
2. **`internal/runtime/acpfake/`** — servidor ACP fake **usando o próprio `coder/acp-go-sdk` no lado servidor** (decisão #21); aceita um script `[]ScriptedUpdate` e emite na ordem com delays opcionais.
3. **`internal/runtime/render/`** — `HumanRenderer` com `io.Writer` injetado (decisão #18), renderizando prefixos `[agent]`, `[thought]`, `[tool]`, `[tool:done]`.

<requirements>
- `Client` interface respeitada: `Open`, `Updates() <-chan events.Event`, `Err()`, `Close()`.
- Canal fechado em `session_end` ou erro do SDK; consumidor não bloqueia indefinidamente.
- Erro acumulado acessível via `Client.Err()` após canal fechar.
- `acpfake.Server` sobe in-process via `os.Pipe()`; cada teste constrói seu script.
- `HumanRenderer` injeta `io.Discard` quando `--quiet` (validado em integration na 9.0).
- Sem panic em qualquer caminho (R-ERR-001).
</requirements>

## Subtarefas

### Client

- [ ] 8.1 Criar `internal/runtime/client/client.go` com interface `Client` (assinatura da techspec §"Interfaces Chave") e impl `acpClient` que carrega `coder/acp-go-sdk` no construtor.
- [ ] 8.2 Implementar `(c *acpClient) Open(ctx context.Context, launcher specs.Launcher, prompt string) error`: spawn subprocess via `exec.Command(cmd, args...)` (sem shell — R-SEC-001); conecta stdio ao SDK; envia prompt como mensagem inicial conforme API do SDK.
- [ ] 8.3 Implementar goroutine interna que lê do `acp.SessionUpdate` do SDK, chama `events.FromACPUpdate(launcher.Kind(), update)` e empurra para o canal `c.events`. Em erro, fecha canal e armazena em `c.err`.
- [ ] 8.4 Implementar `Updates() <-chan events.Event`, `Err() error` (lê pós-fechamento), `Close() error` (idempotente; SIGTERM no subprocess se ainda vivo, depois SIGKILL após 2s).
- [ ] 8.5 Criar `ClientFactory` interface e impl default; injetada no `ACPRunner` (9.0).

### Acpfake

- [ ] 8.6 Criar `internal/runtime/acpfake/script.go` com type `ScriptedUpdate { Delay time.Duration; Update acp.SessionUpdate }`; helper `func NewScript() *Script` com `.AppendAgentMessage(text string)`, `.AppendToolCall(id, name string)`, `.AppendToolCallUpdate(id, status string)`, `.AppendSessionEnd()`, `.AppendUnknown(rawKind string, payload any)`, `.AppendRequestPermission(toolName string)`.
- [ ] 8.7 Criar `internal/runtime/acpfake/server.go` com `Server { script *Script; sdk SDKServer }` que usa o **servidor ACP do `coder/acp-go-sdk`** para emitir updates ao cliente via pipes; método `Start(ctx context.Context) (io.ReadWriter, error)` retorna o "stdio" virtual para o cliente.
- [ ] 8.8 Garantir que `acpfake` não vaza goroutines (`goleak` ou contagem manual).

### Renderer

- [ ] 8.9 Criar `internal/runtime/render/human.go` com `HumanRenderer { out io.Writer }`; construtor `NewHumanRenderer(out io.Writer) *HumanRenderer`.
- [ ] 8.10 Implementar `(r *HumanRenderer) Render(evt events.Event)` com switch por kind, gerando linhas conforme RF-11. Sem retornar erro (loga internamente se `fmt.Fprintf` falhar).

### Testes

- [ ] 8.11 Testes unitários do `acpClient` consumindo o `acpfake` (round-trip in-process); cenários: happy path, fake fica mudo, fake emite unknown, fake fecha abruptamente.
- [ ] 8.12 Testes unitários do `HumanRenderer` por kind (8 fixtures) usando `bytes.Buffer`.
- [ ] 8.13 Teste de leak: `goleak.VerifyTestMain` em `client_test.go` e `acpfake_test.go`.

## Detalhes de Implementação

Ver `techspec.md`:
- §"Visão Geral dos Componentes" (tabela)
- §"Design de Implementação" → "Interfaces Chave" (Client, ClientFactory, Renderer)
- §"Design de Implementação" → "Renderer (RF-11, OC #9)"
- §"Pontos de Integração" → "github.com/coder/acp-go-sdk"

## Critérios de Sucesso

- `go test ./internal/runtime/client/... ./internal/runtime/acpfake/... ./internal/runtime/render/... -race` ≥ 85% cobertura agregada.
- `acpClient + acpfake` ciclo round-trip in-process completa em < 1s.
- Sem leak de goroutines após `Close()`.
- `HumanRenderer.Render` produz golden lines para cada kind.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] `TestAcpClient_HappyPath`: round-trip com fake script de 5 updates; canal recebe 5 events + session_end
- [ ] `TestAcpClient_FakeMute`: fake não emite; consumidor bloqueia em receive; cancelar contexto fecha canal
- [ ] `TestAcpClient_UnknownDrift`: fake emite kind desconhecido; canal recebe `Event{Kind: KindUnknown}`
- [ ] `TestAcpClient_AbruptClose`: fake fecha pipe; canal fecha; `Err()` retorna erro wrappado
- [ ] `TestAcpfake_Scripted`: API de construção do script gera updates esperados
- [ ] `TestHumanRenderer_PerKind`: 8 fixtures cobrindo cada kind
- [ ] `TestHumanRenderer_QuietBehavior`: writer = `io.Discard` produz string vazia
- [ ] `TestNoGoroutineLeak`: `goleak.VerifyTestMain` no pacote client

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/client/client.go` + `client_test.go` (novo)
- `internal/runtime/client/factory.go` (novo)
- `internal/runtime/acpfake/script.go` (novo)
- `internal/runtime/acpfake/server.go` + `server_test.go` (novo)
- `internal/runtime/render/human.go` + `human_test.go` (novo)
- `internal/runtime/render/testdata/*.txt` (golden lines)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-8.0/execution_report.md`
- [ ] `go test ./internal/runtime/{client,acpfake,render}/... -count=1 -race -cover` ≥ 85% agregado
- [ ] `golangci-lint run ./internal/runtime/{client,acpfake,render}/...` sem violações
- [ ] Verificação manual: `go vet ./...` sem warning sobre uso indevido do SDK
- [ ] Commit semântico `feat(runtime): add ACP client, fake server and human renderer`
