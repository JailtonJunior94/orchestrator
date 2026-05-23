# Tarefa 7.0: Persistence Layer (JSONL + Report + ToolCalls)

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar `internal/runtime/persistence/` com três responsabilidades distintas mas coesas pela camada de IO: (a) `JSONLWriter` append-only para `events.jsonl` (RF-08); (b) `ToolCallsRenderer` para `tool_calls.md` (RF-09); (c) `ReportEnricher` para adicionar seção `## Runtime ACP` idempotente em `execution_report.md` (RF-10). Tudo usa `internal/fs.FileSystem` para testabilidade.

<requirements>
- Append-only no `events.jsonl`: cada chamada `Append` escreve uma linha; falha de escrita propaga erro mas não corrompe arquivo.
- `tool_calls.md` é overwrite (gerado uma vez no fim); sem tool calls = arquivo com texto `Nenhum tool call registrado.`.
- `ReportEnricher.Enrich` é idempotente: se a seção `## Runtime ACP` já existe, substitui-a; senão, faz append.
- Paths normalizados via `filepath.Clean` (R-SEC-001 "Filesystem").
- Todas as escritas via `fs.FileSystem.WriteFile` ou similar (auditável).
- Nenhum import de `coder/acp-go-sdk` (consome apenas `events.Event` e `runtime.Summary`).
</requirements>

## Subtarefas

- [ ] 7.1 Criar `internal/runtime/persistence/jsonl.go` com struct `JSONLWriter { path string; fsys fs.FileSystem }`; construtor `NewJSONLWriter(path string, fsys fs.FileSystem) (*JSONLWriter, error)` que valida path e cria diretório pai.
- [ ] 7.2 Implementar `(w *JSONLWriter) Append(evt events.Event) error`: serializa via `evt.MarshalJSON()`, anexa `\n`, escreve em modo append (`os.O_APPEND|os.O_CREATE|os.O_WRONLY`). Erro de IO retorna wrapped com `fmt.Errorf("appending event to %s: %w", w.path, err)`.
- [ ] 7.3 Criar `internal/runtime/persistence/toolcalls.go` com `WriteToolCalls(path string, summaries []events.ToolCallSummary, fsys fs.FileSystem) error`. Formato markdown: cabeçalho `# Tool Calls`, lista numerada com `- {n}. **{name}** — status: {status} — id: `{tool_call_id}``. Sem summaries: escreve `Nenhum tool call registrado.\n`.
- [ ] 7.4 Criar `internal/runtime/persistence/report.go` com `EnrichReport(reportPath string, summary runtime.Summary, fsys fs.FileSystem) error`. Lê o arquivo existente; localiza marcador `## Runtime ACP` via regex `(?m)^## Runtime ACP$`; se encontrado, substitui o bloco até o próximo `^## ` (exclusivo); senão, faz append no final do arquivo. Conteúdo gerado por template:
  ```
  ## Runtime ACP

  - runtime: acp
  - launcher: {{.Launcher}}
  - events_count: {{.EventsCount}}
  - unknown_events_count: {{.UnknownEventsCount}}
  - cancel_reason: {{.CancelReason}}
  ```
- [ ] 7.5 Criar testes unitários para cada arquivo usando `fs.FakeFileSystem` (já existente em `internal/fs/`).
- [ ] 7.6 Definir struct `runtime.Summary` em `internal/runtime/summary.go` se ainda não existir (campos conforme techspec §"Application Service: ACPRunner"). Adiar criação se 9.0 já vai criar; coordenar para evitar duplicação.

## Detalhes de Implementação

Ver `techspec.md`:
- §"Pontos de Integração" → "Filesystem"
- §"Design de Implementação" → "Enriquecimento do execution_report.md"
- PRD RF-08, RF-09, RF-10 para formatos exatos
- §"Object Calisthenics Aplicado" regras 2, 3, 7

## Critérios de Sucesso

- `go test ./internal/runtime/persistence/...` ≥ 90% cobertura.
- Idempotência do `EnrichReport` validada: chamar duas vezes produz o mesmo arquivo final.
- `tool_calls.md` gerado bate com golden file.
- `JSONLWriter` em race test não corrompe arquivo quando duas goroutines escrevem (proteção por mutex interno).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] `TestJSONLWriter_Append`: vários eventos resultam em N linhas, cada uma JSON válido
- [ ] `TestJSONLWriter_AppendConcurrent`: 100 goroutines escrevendo; arquivo final tem 100 linhas válidas (mutex)
- [ ] `TestWriteToolCalls_WithEvents` e `TestWriteToolCalls_Empty`: golden file
- [ ] `TestEnrichReport_FreshAppend`: arquivo sem seção → append
- [ ] `TestEnrichReport_ReplaceExisting`: arquivo com seção → substituição
- [ ] `TestEnrichReport_Idempotency`: duas chamadas consecutivas produzem mesmo conteúdo

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/runtime/persistence/jsonl.go` + `jsonl_test.go` (novo)
- `internal/runtime/persistence/toolcalls.go` + `toolcalls_test.go` (novo)
- `internal/runtime/persistence/report.go` + `report_test.go` (novo)
- `internal/runtime/persistence/testdata/*.md` (golden files)
- `internal/runtime/summary.go` (novo se 9.0 ainda não criou)

## Validações (DoD)

- [ ] `make verify` executado com sucesso e capturado em `evidence/task-7.0/execution_report.md`
- [ ] `go test ./internal/runtime/persistence/... -count=1 -race -cover` ≥ 90%
- [ ] `golangci-lint run ./internal/runtime/persistence/...` sem violações
- [ ] `grep -r "coder/acp-go-sdk" internal/runtime/persistence/` retorna vazio
- [ ] Commit semântico `feat(runtime/persistence): add events.jsonl writer, tool_calls.md and report enricher`
