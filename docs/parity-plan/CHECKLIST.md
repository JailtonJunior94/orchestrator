# Plano de Equalização: ai-governance (bash) → ai-spec-harness (Go)

> Gerado em: 2026-04-19
> Fonte da verdade: `JailtonJunior94/ai-governance@main` (commit `c5545f8`)
> Implementação local: `JailtonJunior94/ai-spec-harness` (commit `c004882`)

## Resumo

6 lacunas identificadas, decompostas em 6 tarefas principais.
Prioridade ordenada por impacto funcional na paridade.

---

## Checklist de execução

### P0 — Paridade funcional direta

- [ ] **TASK-01** — Geração de comandos Gemini TOML
  - Arquivo: [TASK-01-gemini-toml-generator.md](./TASK-01-gemini-toml-generator.md)
  - Impacto: `ai-spec install --tools gemini` não gera `.gemini/commands/`
  - Pacote: `internal/adapters/`

- [ ] **TASK-02** — Geração de agentes Copilot
  - Arquivo: [TASK-02-copilot-agents-generator.md](./TASK-02-copilot-agents-generator.md)
  - Impacto: `ai-spec install --tools copilot` não gera `.github/agents/*.agent.md`
  - Pacote: `internal/adapters/`

- [ ] **TASK-03** — Scaffold com TOML Gemini
  - Arquivo: [TASK-03-scaffold-gemini-toml.md](./TASK-03-scaffold-gemini-toml.md)
  - Impacto: `ai-spec scaffold <lang>` não cria o comando Gemini correspondente
  - Pacote: `internal/scaffold/`

### P1 — Segurança e robustez

- [ ] **TASK-04** — Controle de profundidade de invocação
  - Arquivo: [TASK-04-invocation-depth-guard.md](./TASK-04-invocation-depth-guard.md)
  - Impacto: sem proteção contra encadeamento infinito de skills
  - Pacote: novo `internal/invocation/` ou guard em comandos existentes

### P2 — Autonomia e precisão

- [ ] **TASK-05** — Skills canônicas embutidas ou referenciáveis
  - Arquivo: [TASK-05-canonical-skills-strategy.md](./TASK-05-canonical-skills-strategy.md)
  - Impacto: CLI depende de clone prévio do repo remoto para `--source`
  - Pacote: `internal/install/` ou novo `internal/embed/`

- [ ] **TASK-06** — Suporte opcional a tiktoken para métricas
  - Arquivo: [TASK-06-tiktoken-metrics.md](./TASK-06-tiktoken-metrics.md)
  - Impacto: estimativas de tokens com ~15% de divergência vs. contagem real
  - Pacote: `internal/metrics/`

---

## Ordem de execução sugerida

```
TASK-01 ─┐
TASK-02 ─┤── paralelo (ambos em internal/adapters, sem dependência)
TASK-03 ─┘── depende de TASK-01 (reutiliza gerador Gemini)
TASK-04 ──── independente
TASK-05 ──── independente (decisão de design necessária)
TASK-06 ──── independente (menor prioridade)
```

## Critério de conclusão global

Todos os itens acima marcados como `[x]` e:
1. `go test ./...` passa sem falhas
2. `go vet ./...` sem warnings
3. Testes de integração cobrem os novos fluxos
4. Parity matrix atualizada sem itens `Ausente` ou `Parcial` nas P0/P1
