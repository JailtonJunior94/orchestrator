# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Generalização de fallback launchers (de npx-only para cadeia genérica ordenada)
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Jailton Junior (owner), arquitetura ai-spec-harness
- **Relacionados:** [PRD](prd.md) RF-01/RF-19; [techspec](techspec.md); `internal/runtime/probe/probe.go`; `internal/runtime/specs/spec.go`; ADR-009/012/013/015

## Contexto

`specs.Spec` já possui o campo `Fallbacks []FallbackLauncher` e `probe.EnsureAvailable` já tenta os
fallbacks após o binário canônico. **Porém a resolução é npx-only**: em `probe.resolve`, o fallback
é materializado via `specs.NewNpxLauncher(extractPackage(fb), extractVersion(fb))`, e
`extractPackage`/`extractVersion` assumem a forma exata de `FixedArgs = ["--yes", "@pkg@ver"]`
(`probe.go:99-138`). Isso impede:
- múltiplos fallbacks heterogêneos por runtime;
- fallbacks que **não** sejam `npx` (ex.: caminho alternativo de binário, `bunx`, wrapper local);
- preservação fiel de `FixedArgs` arbitrários do fallback.

O PRD (RF-01) exige uma **cadeia ordenada de launchers de fallback genéricos**, e o Compozy modela
isso como `Fallbacks []Launcher` (`{Command, FixedArgs, ProbeArgs}`) sem acoplamento a npx.

## Decisão

Generalizar a resolução de fallback para tratar cada `FallbackLauncher` como um **launcher de
comando genérico**, preservando seus `FixedArgs` literalmente:

1. Em `probe.resolve`, ao encontrar `fb.Command` no PATH, materializar via
   `specs.NewBinaryLauncher(path, fb.FixedArgs...)` — **sem** assumir semântica npx.
2. Tratar **toda a cadeia** `spec.Fallbacks` em ordem; o primeiro cujo `Command` exista no PATH vence.
3. **Aposentar** `extractPackage`/`extractVersion`/`NewNpxLauncher` como caminho obrigatório; o
   launcher npx vira apenas um caso particular expresso por `FallbackLauncher{Command:"npx",
   FixedArgs:["--yes","@pkg@ver"]}` — idêntico em comportamento ao atual, mas sem código especial.
4. Manter `sdkVersion/npmVersion/npmPackage` no `Spec` apenas como **metadado** para a mensagem de
   erro de indisponibilidade (`formatLauncherUnavailable`), não como driver da resolução.
5. **Paridade byte-equivalente (RF-05):** para as specs atuais (cujo único fallback é npx no formato
   esperado), o argv resolvido permanece idêntico ao atual.

## Alternativas Consideradas

- **Manter npx-only e só permitir múltiplos pacotes npx:** insuficiente — não cobre wrappers
  locais nem binários alternativos exigidos por ambientes reais.
- **Mover a resolução para fora do probe (no runner):** espalha responsabilidade e quebra o cache
  por spec.ID já existente em `probe`. Rejeitada.
- **Probe ativo (executar `--version` de cada candidato):** mais robusto, porém mais lento e com
  efeitos colaterais; conflita com a meta de bootstrap rápido. Rejeitada nesta fase (LookPath basta).

## Consequências

### Benefícios Esperados
- Resiliência real de runtime: qualquer launcher alternativo declarável por spec — RF-01.
- Código mais simples (remove parsing frágil de `@scope/pkg@ver`).
- Habilita o invariante de paridade de fallback (RF-19).

### Trade-offs e Custos
- Mudança de assinatura interna em `probe` e remoção de helpers; exige atualizar testes do probe.
- `NewNpxLauncher` deixa de ser caminho privilegiado (pode ser mantido como helper de conveniência).

### Riscos e Mitigações
- **Risco:** regressão no argv do npx atual → **Mitigação:** teste de paridade comparando argv
  resolvido antes/depois para cada spec (claude/codex/gemini/copilot).
- **Risco:** fallback aponta para binário incompatível → **Mitigação:** ordem declarada por spec
  coloca o canônico primeiro; fallback é último recurso e registrado em `runtime_init`.

## Plano de Implementação
1. Reescrever `probe.resolve` para iterar `spec.Fallbacks` usando `NewBinaryLauncher(path, fb.FixedArgs...)`.
2. Remover/depreciar `extractPackage`/`extractVersion`; converter fallbacks npx das specs para o
   formato genérico (mantendo `FixedArgs` atuais).
3. Atualizar testes de `probe` e adicionar teste de paridade de argv por spec.

## Monitoramento e Validação
- Teste: binário canônico ausente + fallback presente → launcher == fallback com `FixedArgs` exatos.
- Teste: argv resolvido idêntico ao baseline para as 4 specs (RF-05).
- Invariante de paridade de fallback no `internal/parity` (RF-19).

## Impacto em Documentação e Operação
- Atualizar comentário do pacote `probe` (cabeçalho ainda menciona claude-agent-acp/npx específico).
- Documentar como declarar fallbacks por runtime.

## Revisão Futura
- Reavaliar se probe ativo (`ProbeArgs`) passa a ser necessário ao expandir runtimes (PRD futuro de breadth).
