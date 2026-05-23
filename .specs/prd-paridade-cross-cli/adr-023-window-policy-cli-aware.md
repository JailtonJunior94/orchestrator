# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Política de janela CLI-aware via campo estático na `Spec` (token-budget + memória sensíveis à janela)
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Time ai-spec-harness (owner: JailtonJunior94)
- **Relacionados:** PRD (RIN-04); techspec; ADR-018 (RuntimeConfig); `internal/runtime/specs/spec.go`; `internal/runtime/hooks/token_budget.go`; `internal/runtime/memory/store.go`; `internal/metrics/metrics.go` (`ToolBudgets`, `CheckBudget`)

## Contexto

Nem Compozy nem o ai-spec-harness gerenciam contexto por janela de token: a compactação é por linha/byte (`memory/store.go`: 150/12KB · 200/16KB) e o `token_budget` usa um teto fixo por ferramenta (`ToolBudgets["claude"]=70000`). CLIs com janelas muito maiores (ex.: Gemini ≥ 1M tokens) são subutilizadas — compactamos cedo demais e o teto de 70k não reflete a CLI ativa. O PRD trata isso como **oportunidade de liderança** (escala G), exigindo zero-value = comportamento F1.

Decisão de origem (clarificação com o usuário): o sinal de janela deve ser **estático por CLI na `Spec`**, não detectado do payload ACP em runtime — determinístico, testável sem rede e desacoplado do schema do provider.

## Decisão

1. Adicionar à `specs.Spec` um Value Object **`ContextWindow`** (campo estático, definido nos construtores de catálogo `Claude()/Codex()/Copilot()/Gemini()`):
   - `ContextWindow{ MaxTokens int }` com método `Class() WindowClass` retornando `WindowStandard` (< 1M) ou `WindowLarge` (≥ 1M).
   - Zero-value (`MaxTokens == 0`) ⇒ `WindowStandard` ⇒ comportamento F1 exato.
2. **token_budget CLI-aware:** `TokenBudgetHook` passa a resolver o limite por `Spec` (não só por string `Tool`). `metrics.ToolBudgets` ganha entradas por driver; quando `ContextWindow.Class()==WindowLarge`, aplica um teto generoso derivado da janela em vez do default fixo. Driver sem entrada cai no default atual (sem regressão).
3. **memória sensível à janela:** introduzir `memory.WindowPolicy` (domain service stateless) que ajusta os limites de compactação por `WindowClass`. `WindowStandard` ⇒ limites atuais (150/12KB · 200/16KB). `WindowLarge` ⇒ limites ampliados configuráveis. Zero-value preserva F1.
4. A `WindowClass` é propagada do `Spec` → `Job`/runner → hooks/memória; nenhuma leitura de runtime/handshake.

## Alternativas Consideradas

- **Detectar janela do payload ACP em runtime.** Vantagem: preciso por sessão/modelo. Desvantagem: não-determinístico, acoplado ao schema `usage` do provider, difícil de testar (proibido teste com rede real — R-TEST-001). Rejeitada (decisão do usuário).
- **Config-only (sem campo na Spec).** Vantagem: flexível. Desvantagem: usuário precisaria configurar janela por CLI manualmente — fere transparência. Rejeitada; config pode *sobrescrever* o default da Spec, mas o default vem da Spec.
- **Tornar memória/token sempre agressivos para janelas grandes.** Risco de custo (mais tokens faturados). Mantido opt-in via classe + zero-value F1.

## Consequências

### Benefícios Esperados

- Aproveita janelas grandes (Gemini) sem compactar cedo — diferencial vs Compozy.
- Determinístico e testável (tabela estática), sem dependência de rede.
- Zero regressão: zero-value ⇒ F1.

### Trade-offs e Custos

- `Spec` ganha um campo (DTO de catálogo — aceitável por OC #8, que isenta DTOs/config).
- Limites grandes podem aumentar custo de tokens; mitigado por serem opt-in por classe e sobreponíveis por config.

### Riscos e Mitigações

- **Risco:** classificar mal a janela de uma CLI (valor estático desatualizado quando o provider muda). **Mitigação:** valor é dado de catálogo versionado; config de projeto pode sobrescrever; revisão ao bump de SDK/CLI.
- **Rollback:** zerar `ContextWindow` na Spec ⇒ volta a `WindowStandard`/F1.

## Plano de Implementação

1. `ContextWindow` VO + `WindowClass` em `specs`.
2. Popular janelas nos construtores de catálogo das 4 CLIs.
3. `TokenBudgetHook` resolve limite por Spec/classe; estender `ToolBudgets`.
4. `memory.WindowPolicy` + wiring no `prepareMemoryStore`.
5. Testes table-driven por classe (standard/large) + regressão F1 (zero-value).

## Monitoramento e Validação

- Gate: `make test`; testes de `token_budget` e `memory` por `WindowClass`.
- Sucesso: sessão Gemini (large) não compacta nos limites F1; sessões Claude/Codex/Copilot inalteradas com zero-value.

## Impacto em Documentação e Operação

- Atualizar `CLAUDE.md`/`docs/config-hierarchy.md` com a chave de override de janela e o comportamento por classe.

## Revisão Futura

- Revisitar ao mudar a janela oficial de qualquer CLI ou se surgir necessidade de detecção por modelo.
