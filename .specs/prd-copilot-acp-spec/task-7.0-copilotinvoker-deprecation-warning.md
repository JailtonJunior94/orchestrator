# Tarefa 7.0: Aviso único de depreciação no copilotInvoker legado

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Adicionar aviso de depreciação em `copilotInvoker.Invoke` (`internal/taskloop/agent.go:381-388`) emitido **uma única vez por execução do processo** via `sync.Once`. Texto contém referência clara ao modo ACP recomendado e ao ADR-012. Caminho legado continua funcional; warning é informativo, não bloqueante.

<requirements>
- sync.Once garante emissão única por execução do processo (não por task).
- Texto contém "modo legado", "ACP", "ADR-012".
- Caminho de execução do copilotInvoker permanece funcional e inalterado em comportamento.
- T-17 (warning na primeira invocação) e T-18 (silêncio nas demais) verdes.
</requirements>

## Subtarefas

- [ ] 7.1 Declarar `var copilotLegacyWarnOnce sync.Once` no package `taskloop` (top-level em `agent.go` ou arquivo adjacente).
- [ ] 7.2 No início de `copilotInvoker.Invoke`, antes de montar args, invocar `copilotLegacyWarnOnce.Do(...)` emitindo o warning em `stderr`.
- [ ] 7.3 Texto do warning: contém literais `"modo legado"`, `"--runtime=acp"`, `"ADR-012"`. Sugestão: `"WARNING: Copilot CLI em modo legado (sem ACP). Migrar para --runtime=acp. Modo legado será removido em vX.Y.Z. Ver ADR-012."`.
- [ ] 7.4 Estender `internal/taskloop/agent_test.go` com T-17 (warning na primeira chamada) e T-18 (sem warning na segunda chamada no mesmo processo).
- [ ] 7.5 Reset do `sync.Once` entre testes: usar nova instância de `copilotInvoker` ou variável local — não global.

## Detalhes de Implementação

Ver `techspec.md` §"Design de Implementação" → bloco `internal/taskloop/agent.go — aviso de depreciação no copilotInvoker`. Decisão D-05 (caminho legado preservado por uma versão; warning é único).

Anti-padrão: NÃO usar `log.Print` (output vai para stdout/log destino). Usar `fmt.Fprintln(os.Stderr, ...)` para garantir captura em smoke tests. NÃO emitir warning a cada invocação — gera ruído em loops longos.

Observação sobre testabilidade: como `sync.Once` no nível de package é global, testes precisam isolar via subprocess ou via wrapper que aceite a `*sync.Once` injetada. Decisão simples: emitir um T-17/T-18 dentro de um único subtest `t.Run("warns once", ...)` que invoca `Invoke` duas vezes consecutivas em um mesmo `copilotInvoker` e verifica que `stderr` contém o warning **exatamente uma vez**.

## Critérios de Sucesso

- Primeira chamada a `copilotInvoker.Invoke` por execução do processo emite warning em `stderr`.
- Chamadas subsequentes no mesmo processo não emitem warning adicional.
- Caminho de execução do CLI legado (`copilot --autopilot --yolo -p <prompt>`) permanece inalterado em args.
- T-17/T-18 verdes.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills auto-carregadas em runtime: `agent-governance`, `execute-task`, `bugfix`,
     `review`, `refactor`, nem skills `*-implementation` (linguagem, inferida pelo diff).
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] T-17: primeira `Invoke` emite warning contendo `"modo legado"`, `"ACP"`, `"ADR-012"` em stderr capturado.
- [ ] T-18: segunda `Invoke` no mesmo `copilotInvoker` instance (e mesmo processo de teste) **não** emite warning adicional. Stderr capturado mostra warning emitido apenas uma vez.
- [ ] Comportamento de `Invoke` (stdout/stderr/exitCode/err) permanece equivalente ao antes da mudança quando warning é ignorado.
- [ ] `go test ./internal/taskloop/...` → 100% verde.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Definition of Done (DoD)

- [ ] `sync.Once` declarado e usado em `copilotInvoker.Invoke`.
- [ ] Texto do warning contém os três literais obrigatórios (`"modo legado"`, `"--runtime=acp"`, `"ADR-012"`).
- [ ] Warning emitido em `stderr` (não stdout, não log).
- [ ] T-17 e T-18 verdes em `agent_test.go`.
- [ ] `copilotInvoker` continua funcional para invocações legadas (args inalterados).
- [ ] `go test ./internal/taskloop/...` → 100% verde.
- [ ] Diff zero em `internal/runtime/`.

## Arquivos Relevantes

- `internal/taskloop/agent.go:381-388` (modificar — adicionar `sync.Once` warning)
- `internal/taskloop/agent_test.go` (estender com T-17 e T-18)
- ADR-012 §"Decisão D-05" (caminho legado por uma versão)
