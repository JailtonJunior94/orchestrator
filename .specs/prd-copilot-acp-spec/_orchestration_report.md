# Orchestration Report — copilot-acp-spec

**Date:** 2026-05-21  
**Tool:** Claude Code  
**Slug:** copilot-acp-spec  
**Status final:** `done`

## Snapshot Inicial vs Final

| Métrica | Inicial | Final |
|---------|---------|-------|
| Total tarefas | 10 | 10 |
| done | 0 | 10 |
| pending | 10 | 0 |

## Execução por Wave

| Wave | Tasks | Mode | Status |
|------|-------|------|--------|
| 1 | 1.0 | solo | done |
| 2 | 2.0, 3.0, 4.0 | parallel | done |
| 3 | 5.0 | solo (Não) | done |
| 4 | 6.0, 7.0 | parallel | done |
| 5 | 8.0 | solo | done |
| 6 | 9.0 | solo | done |
| 7 | 10.0 | solo | done |

## Tabela de Tarefas

| # | Título | Status | Note |
|---|--------|--------|------|
| 1.0 | Estender Spec value object com metadata SDK/NPM | done | Contract recovery: subagent rodou review em vez de execute-task; impl verificada correta |
| 2.0 | Criar specs/copilot.go com construtor e testes | done | CopilotNpmVersion=1.0.51 confirmada via npm registry |
| 3.0 | Generalizar runtime_init em runner.go | done | T-09 (Copilot) e T-05 (regressão Claude) verdes |
| 4.0 | Generalizar probe error template + adrByID | done | Fixup: unused ctx em resolve() removido |
| 5.0 | Tabela runtimeACPCatalog em cmd/ | done | RF-06/RF-07; T-13/T-14/T-15 verdes |
| 6.0 | Wiring taskloop.Service.Execute ACP-routing | done | R-08 client/ diff pré-existente; T-16/T-22/T-23/T-24 verdes |
| 7.0 | Aviso único depreciação copilotInvoker (sync.Once) | done | Contract recovery; T-17/T-18 verdes |
| 8.0 | Sub-suite Copilot em acp_integration_test.go | done | T-10/T-11/T-12 verdes; paridade observacional confirmada |
| 9.0 | Documentação F1 cross-cutting | done | COPILOT.md, AGENTS.md, ADR-007, telemetria, cli-schema |
| 10.0 | Smoke test real copilot --acp + audit/ | done | 51 eventos ACP; probe fix FixedArgs; go test 100% verde |

## Eventos Notáveis

- **Spec drift pré-voo:** RF-11 ausente da tabela de cobertura de tasks.md → corrigido adicionando RF-11 à task 2.0 (fix documental aprovado pelo usuário).
- **Contract recovery (2×):** Tasks 1.0 e 7.0 — subagent rodou skill `review` em vez de `execute-task`; implementação verificada correta pelo orquestrador que escreveu o report de execução.
- **Fixups pós-wave:** 4 diagnostics corrigidos (probe.go unused ctx, report.go WriteString concat, taskloop_test.go fmt.Appendf, SplitSeq em 2 arquivos, probe_test.go slices.Contains).
- **Smoke T10.0:** copilot v1.0.51 em PATH, gh auth OK; smoke levou ~42 min; 3 event kinds desconhecidos Copilot-específicos (available_commands_update, config_option_update) registrados como observação.

## Invariantes Verificadas

| Invariante | Resultado |
|-----------|-----------|
| R-01: `go test ./internal/runtime/specs/...` 100% verde | PASS |
| R-08: diff zero em persistence/, watchdog.go | PASS (client/ diff pré-existente ao branch) |
| T2.0: CopilotNpmVersion confirmada via npm registry | PASS (1.0.51) |
| T10.0: copilot --acp no PATH + gh auth status | PASS |
| `go test ./...` verde | PASS (todos os packages) |

## Próximos Passos

- Preencher placeholder `vX.Y.Z` em `copilotLegacyDeprecationMsg` (agent.go) antes do merge.
- Criar PR com referência ao ADR-012.
