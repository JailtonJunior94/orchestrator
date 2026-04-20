# ai-spec-harness — Claude Code

> Use `AGENTS.md` como fonte canonica das regras deste repositorio. Stack, comandos, convencoes, estrutura, CI e padroes estao documentados em `AGENTS.md` — nao duplicados aqui.

## Instrucoes

1. Ler `AGENTS.md` no inicio da sessao.
2. `.agents/skills/` e a fonte de verdade dos fluxos procedurais.
3. Em tarefas de execucao: carregar `AGENTS.md` + `agent-governance` + skill da linguagem afetada.
4. Skills de planejamento entram apenas quando a tarefa pedir explicitamente.
5. Referencias adicionais apenas quando a tarefa exigir.
6. Preservar estilo, arquitetura e fronteiras existentes.
7. Validar mudancas com comandos proporcionais ao risco.

## Telemetria e Auditoria

- Telemetria: `GOVERNANCE_TELEMETRY=1`; ver [`docs/telemetry-feedback-cycle.md`](docs/telemetry-feedback-cycle.md); relatorio: `ai-spec-harness telemetry report`
- Auditorias salvas em `audit/`, indexadas em [`audit/README.md`](audit/README.md)

## ADRs

Consultar antes de mudancas estruturais. Template: [`tasks/adr/000-template.md`](tasks/adr/000-template.md)

- [001](tasks/adr/001-go-embed-baseline.md) — assets via `go:embed`
- [002](tasks/adr/002-fake-filesystem-testes.md) — FakeFileSystem vs afero
- [003](tasks/adr/003-paridade-semantica.md) — invariantes semanticas vs diff textual
- [004](tasks/adr/004-lazy-loading-referencias.md) — references sob demanda
- [005](tasks/adr/005-skills-lock-sha256.md) — lock file SHA-256
- [006](docs/adr/006-telemetria-feedback-cycle.md) — telemetria opt-in append-only
- [007](docs/adr/007-copilot-cli-stateless-workaround.md) — Copilot injecao manual
- [008](docs/adr/008-parity-multi-tool-invariants.md) — 29 invariantes 3 niveis
