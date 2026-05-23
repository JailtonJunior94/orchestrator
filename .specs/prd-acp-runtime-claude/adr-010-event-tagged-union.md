# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** `runtime.Event` como struct tagged union em vez de sealed interface
- **Data:** 2026-05-20
- **Status:** Aceita
- **Decisores:** Mantenedores do `ai-spec-harness`
- **Relacionados:**
  - PRD: `.specs/prd-acp-runtime-claude/prd.md`
  - Techspec: `.specs/prd-acp-runtime-claude/techspec.md`
  - ADR anterior: [009](../adr/009-acp-protocol-adoption.md)
  - Inspiração: `internal/core/agent/acp_convert.go` e `internal/core/model/session_update.go` do compozy/compozy

## Contexto

A techspec introduz um tipo único `runtime.Event` que viaja pelo canal `<-chan events.Event` entre `client.acpClient` e `ACPRunner`. Esse tipo precisa:

1. **Carregar payloads diferentes por kind** — `agent_message`, `agent_thought`, `tool_call_start`, `tool_call_update`, `session_end`, `runtime_init`, `unknown` têm estruturas internas diferentes.
2. **Serializar em JSONL** com envelope estável (`{ts, kind, tool_call_id, launcher, raw}`) — RF-08.
3. **Ser usado em `switch` por consumidores** (renderer humano, contadores, persistência) sem reflection.
4. **Não vazar tipos de `coder/acp-go-sdk`** para fora do pacote `events`.
5. **Ser estável o suficiente para golden tests** (envelope com snapshot).

Há dois desenhos idiomáticos em Go para representar variantes discriminadas:

- **Sealed interface:** tipos separados (`AgentMessage`, `ToolCallStart`, ...) implementando `Event interface { Kind() EventKind; isEvent() }`.
- **Tagged union:** struct `Event { Kind EventKind; AgentMessage *AgentMessagePayload; ToolCallStart *ToolCallStartPayload; ... }` com ponteiros opcionais (apenas um não-nil por kind).

O compozy escolheu o segundo desenho em `model.SessionUpdate`. Replicar essa escolha reduz risco arquitetural (decisão já validada em produção em projeto comparável) e elimina trabalho de tradução.

## Decisão

Representar `runtime.Event` como **struct tagged union**: um único tipo concreto, exportado, com campo discriminador `kind EventKind` e ponteiros opcionais para os payloads tipados de cada kind.

Forma canônica:

```go
type Event struct {
    ts          time.Time
    kind        EventKind
    toolCallID  ToolCallID
    launcher    specs.Launcher

    agentMessage   *AgentMessagePayload
    agentThought   *AgentThoughtPayload
    toolCallStart  *ToolCallStartPayload
    toolCallUpdate *ToolCallUpdatePayload
    sessionEnd     *SessionEndPayload
    runtimeInit    *RuntimeInitPayload
    unknown        *UnknownPayload

    raw json.RawMessage
}
```

- Apenas **um** ponteiro de payload deve ser não-nil por instância (invariante validada nos construtores `events.NewAgentMessage(...)`, etc.).
- Acesso por consumidor sempre via método (`evt.AgentMessage()`, `evt.ToolCallStart()`); retornam `nil` quando o kind não bate, sem panic.
- `MarshalJSON` produz o envelope RF-08 com `raw` inteiro de `acp.SessionUpdate`.
- Construtores são as únicas formas de criação fora de testes (R-DDD-001: "Struct literal fora de testes e factories é proibido").

## Alternativas Consideradas

### Alternativa 1 — Sealed interface

```go
type Event interface { Kind() EventKind; Timestamp() time.Time; isEvent() }
type AgentMessage struct { ... }   // implementa Event
type ToolCallStart struct { ... }  // implementa Event
// ... etc
```

- **Vantagens:** mais idiomático "Go puro"; cada variante tem só os campos que importam; impossibilidade estrutural de instância "híbrida".
- **Desvantagens:**
  - Serialização JSON exige `MarshalJSON`/`UnmarshalJSON` em cada tipo + lógica de dispatch por kind no envelope — mais código de boilerplate.
  - Type-switch fica mais verboso quando o consumidor só quer ler `tool_call_id` (precisa fazer cast para o tipo certo).
  - Diverge do desenho do compozy (referência); aumenta superfície de divergência em conversores futuros (Codex, Gemini).
- **Motivo de rejeição:** custo de manutenção da camada de serialização supera o ganho de type-safety; o invariante "um payload por kind" é facilmente garantido em construtor.

### Alternativa 2 — `map[string]any` com chave `kind`

- **Vantagens:** zero código de tipos; máxima flexibilidade.
- **Desvantagens:** sem type-safety; consumidor faz `evt["tool_call_id"].(string)`; refactor quebra silenciosamente; testes precisam validar shape manualmente; pior debugging.
- **Motivo de rejeição:** abre mão da principal vantagem do Go; transformar erros de tipo em erros de runtime é regressão.

### Alternativa 3 — `any` + type-switch em todo lugar

Variante de #2 onde cada payload é seu próprio tipo, mas o canal carrega `<-chan any`.

- **Vantagens:** flexível; consumidor escolhe tipo concreto.
- **Desvantagens:** o canal perde a documentação do contrato; novos kinds não aparecem na assinatura.
- **Motivo de rejeição:** mesma classe de problema que sealed interface, sem os benefícios.

## Consequências

### Benefícios Esperados

- Envelope JSONL implementado em **um único** `MarshalJSON` no tipo `Event`.
- Consumidor faz `switch evt.Kind()` e acessa `evt.AgentMessage()` quando relevante — caminho linear, sem casts.
- Paridade com `compozy/compozy` `model.SessionUpdate`: padrão validado em produção em projeto análogo; reduz risco de design inédito.
- Construtor (`events.NewAgentMessage`, etc.) protege a invariante "exatamente um payload não-nil"; testes ficam triviais.
- Adicionar um novo kind exige: (a) adicionar `EventKind` constante; (b) adicionar campo ponteiro; (c) adicionar construtor; (d) adicionar caso no `MarshalJSON`. Mudança localizada.

### Trade-offs e Custos

- O struct fica visivelmente maior em LOC do que cada tipo separado da Alternativa 1 — mas concentrado num só arquivo (`event.go`), facilitando navegação.
- Quem ler o tipo precisa saber que **apenas um ponteiro** será não-nil por kind. Documentado no comment do tipo + enforcement em construtor. Risco médio para leitor casual; baixo para mantenedor.
- "Um payload por kind" é invariante de construção, não de tipo. Quebrável se alguém montar `Event{}` literal fora dos construtores. Mitigação: regra de lint local (proibir struct literal de `Event{}` fora de `events_test.go`, `events/*.go`) — mesma regra que `R-DDD-001` já exige.

### Riscos e Mitigações

- **Risco:** mantenedor cria `Event{Kind: KindAgentMessage}` sem setar `agentMessage` ⇒ consumidor pega nil.
  - **Mitigação:** lint rule (ou simplesmente revisar PRs) impedindo literal fora dos construtores. Métodos de acesso retornam `nil` em vez de panic; consumidores que dependem do payload têm assertion no teste.
  - **Rollback:** ajustar construtor ou método de acesso conforme aparecer.

- **Risco:** `MarshalJSON` único cresce a cada novo kind até virar function-of-doom.
  - **Mitigação:** delegar para `evt.payload().MarshalJSON()` via interface `payload { json.Marshaler }` interna ao pacote — adiar até existir pressão real.
  - **Rollback:** refactor é local ao pacote `events`.

- **Risco:** divergir de uma futura recomendação idiomática do ecossistema Go (ex.: tipos genéricos com `~union` se a linguagem ganhar).
  - **Mitigação:** decisão é local; migração futura seria mecânica.

## Plano de Implementação

1. Definir `EventKind` (enum string) + `ParseEventKind`.
2. Definir payloads tipados em `payloads.go` (todos com campos privados + getters de intenção).
3. Definir `Event` struct + construtores `New*` que validam invariantes.
4. Implementar `MarshalJSON` produzindo o envelope RF-08.
5. Adicionar `events_test.go` cobrindo cada construtor + golden file de envelope.
6. Documentar a invariante "apenas um payload não-nil" no comentário do tipo.
7. Critério de adoção: todos os consumidores do canal de eventos (`HumanRenderer`, `JSONLWriter`, `ToolCallCounters`, `ReportEnricher`) usam apenas o `Event` exportado, sem importar payloads internamente.

## Monitoramento e Validação

- **Sinais a observar:**
  - Aumento de `Event{...}` literal fora dos construtores em PRs (revisão manual).
  - Crescimento do `MarshalJSON` além de ~80 LOC ⇒ revisar para delegação.
- **Critério de sucesso:** a Iteração 1 de rollout (techspec, Plano de Rollout) entrega o tipo sem necessidade de redesenho; consumidores cabem em ~10 LOC cada.
- **Critério para revisão:** se a quantidade de kinds passar de ~12 ou se um payload precisar variar dinamicamente (ex.: `tool_call_update` com schemas diferentes por tool), reavaliar contra sealed interface.

## Impacto em Documentação e Operação

- Techspec já cita este ADR.
- README — sem alteração (decisão interna).
- AGENTS.md — adicionar entrada nas ADRs locais do PRD.
- Sem impacto em onboarding, runbook ou observabilidade.

## Revisão Futura

- **Marco de revisão:** quando o segundo runtime (provavelmente Codex) for adicionado via ACP e introduzir kinds novos. Verificar se o desenho ainda comporta sem ginástica.
- **Eventos que invalidam premissas:**
  - Go ganhar union types nativos com paridade ergonômica.
  - O `coder/acp-go-sdk` introduzir variantes que tornem "um payload por kind" insuficiente.
- **Condição para substituição:** decisão de mover para sealed interface (novo ADR) ou para representação tipada por linguagem futura.
