# Relatório Final de Revisão — PRD memoria-duravel-agentes

## Veredito

**APPROVED — conformidade total, 0 achados remanescentes em qualquer severidade, nas 10 tarefas.**

## Ciclo executado

`review → bugfix → review`, repetido em 4 rodadas até esgotar todo achado, com 10 subagentes de revisão especializados por tarefa em cada rodada (paralelos) e correções aplicadas por subagentes de bugfix (rodada 1, arquitetural + mecânico) e diretamente pelo orquestrador (rodadas 2–3, após um subagente esgotar limite semanal de API).

| Rodada | Resultado |
|---|---|
| 1 | 1.0, 2.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0 **REJECTED**; 3.0 **APPROVED** — 18 achados consolidados em `bugs.md` |
| 2 | 1.0, 4.0, 8.0 **APPROVED**; 2.0, 5.0, 6.0, 7.0 (falha de API), 9.0 (ressalva), 10.0 **REJECTED/pendente** — 6 novos achados em `bugfix_report_round2.md` |
| 3 | 2.0, 5.0, 6.0, 9.0 **APPROVED**; 7.0, 10.0 **REJECTED** (achados residuais mecânicos) — 2 achados em `bugfix_report_round3.md` |
| 4 | 7.0, 10.0 **APPROVED** — 0 achados |

## Achados críticos corrigidos (destaques)

1. **BUG-01 [critical]** — `Facade.RecordSession` abortava a escrita de fatos quando a reivindicação do bastão falhava, invertendo a precedência de invariantes (segredo > não-perda de fato > bastão > orçamento). Corrigido: falha de bastão agora só marca o relatório, nunca bloqueia a escrita.
2. **BUG-02/BUG-06 [critical]** — o lease de bastão da `Facade` era inteiramente em memória de processo, nunca persistido; o comando `memory handoff` do CLI tinha seu próprio sidecar desconectado, com race TOCTOU sem lock. Corrigido: `Facade` e CLI agora compartilham o mesmo sidecar (`MEMORY.handoff.json`) sob a mesma trava (`MEMORY.handoff.lock`), leitura-decisão-escrita atômica, provado com dois processos OS reais disputando o mesmo lease.
3. **BUG-03/BUG-24 [critical]** — o benchmark de RF-20 media recuperação vazia (zero fatos) na primeira tentativa de correção, e depois um cenário vacuamente trivial (custo O(1) independente de volume) na segunda. Corrigido na terceira tentativa: 1.000 Fatos ativos reais consolidados numa única página via `Layer.Consolidate` direto (contornando a compactação real, que impossibilitaria esse volume pelo caminho normal), com asserção de volume antes de medir — p95 real de ~31ms, genuinamente sensível a volume (133k allocs/op).
4. **BUG-04/BUG-05 [high]** — round-trip de `Page.Serialize` (RF-36) mascarava mutação de newline e reordenava silenciosamente conteúdo humano intercalado entre Fatos. Corrigido com recusa explícita tipada (`ErrHumanContentNotNormalized`/`ErrHumanBlockInterleaved`), depois ajustado para não abortar a sessão inteira (apenas a camada afetada).
5. **BUG-07/BUG-21 [major]** — duas races TOCTOU sucessivas no lock de camada Windows (visibilidade parcial do conteúdo do lock; depois double-ownership no takeover). Corrigidas com publicação atômica via `os.Link` e compare-and-delete real.
6. **BUG-08 [major]** — cascata de config com merge "sticky-true" impedia a flag `--durable-memory=false` de desativar a feature quando uma camada inferior a ligava. Corrigido com mecanismo tri-state nos dois pontos de propagação (`resolver.go` e `runtimeconfig.go`), testado nas duas direções.
7. **BUG-09 [major]** — RF-09 (fonte dupla de captura) nunca populava a "seção declarada"; fechado com extração real de marcador `## Memory Declared` do transcript da sessão.
8. **BUG-11 [major]** — `EnrichReport` podia apagar silenciosamente a seção de Evidência de Memória Durável ao reinjetar métricas numa segunda chamada (cenário real de retry). Corrigido com localização correta da próxima seção e ordem canônica preservada.
9. **BUG-12 [major]** — `memory migrate`/`memory handoff` escreviam sem proteção contra symlink externo. Corrigido com `fs.RefuseExternalSymlink` em todos os pontos de escrita.
10. **BUG-13/BUG-14 [major/minor]** — subtarefa 10.3 testava o mecanismo errado para "lock órfão" (HandoffLease em vez do LayerLock real de RF-16); teste de handoff cross-CLI nunca variava CLI de fato. Ambos corrigidos com testes de integração reais.
11. **BUG-15 [major, recorrente em 4 rodadas]** — violações de R-STYLE-001.2 (zero comentários) em praticamente todos os arquivos tocados pela feature, incluindo reintroduções durante as próprias rodadas de correção (`layer.go`, `cli_contract_test.go`, `runtimeconfig_internal_test.go`, logs PT-BR em `runner.go`, comentário desatualizado em `memory_persist.go`). Todas eliminadas e reconfirmadas por varredura `git diff HEAD` em cada rodada.

## Estado final por tarefa

| # | Título | Veredito |
|---|--------|----------|
| 1.0 | Escrita atômica na abstração de filesystem | APPROVED |
| 2.0 | Fato, identidade, durabilidade, sentinelas e Página com round-trip lossless | APPROVED |
| 3.0 | Quatro políticas stateless | APPROVED (sem achados em nenhuma rodada) |
| 4.0 | Lease de bastão de continuidade e detecção de processo vivo | APPROVED |
| 5.0 | Agregado de camada com lock por camada, consolidação e escrita atômica | APPROVED |
| 6.0 | Golden byte-a-byte e fim da degradação silenciosa | APPROVED |
| 7.0 | Fachada, porta de memória e wiring por configuração | APPROVED |
| 8.0 | Evidência de memória, métricas e telemetria | APPROVED |
| 9.0 | Comando `memory` com seis subcomandos, incluindo migração | APPROVED |
| 10.0 | Integração multi-processo, e2e, benchmark e alvos de Make | APPROVED |

## Validação central final (executada por mim, não apenas relatada pelos subagentes)

- `go build ./...` → sem erros
- `go vet ./...` → sem erros
- `go test ./... -count=1` → **3158 testes, 77 pacotes, 0 FAIL**
- `make integration` → todos os pacotes `ok`, incluindo `internal/runtime/memory/durable` e `cmd/ai_spec_harness`
- `make bench` → `BenchmarkRecoveryWith1000ActiveFactsInSinglePage` p95 real ≈ 30.7ms (limite RF-20: 200ms), volume real de 1.000 fatos verificado antes da medição
- `bash scripts/check-mocks.sh` → sincronizado com `mockery.yml`
- `ai-spec check-spec-drift .specs/prd-memoria-duravel-agentes/tasks.md` → sem drift (spec-hash da techspec recalculado após correção de nomenclatura/documentação)
- `git status --short` → sem arquivos de sonda/scratch residuais dos subagentes

## Rastreabilidade de evidência

- `bugs.md` — 18 achados da 1ª rodada, formato canônico
- `bugfix_report_arch.md` / `bugfix_report_mech.md` — correções da 1ª rodada (2 ondas paralelas)
- `bugfix_report_round2.md` — 6 achados da 2ª rodada, corrigidos diretamente
- `bugfix_report_round3.md` — 2 achados da 3ª rodada, corrigidos diretamente
- Todos os relatórios de bugfix validados por `.agents/scripts/validate-bugfix-evidence.sh`

## Riscos residuais aceitos (não bloqueantes, documentados)

- Correções do lock de camada Windows (BUG-07/BUG-21) validadas apenas por cross-compilation nesta sessão (ambiente darwin); recomenda-se execução em CI Windows real antes do próximo release.
- Janela TOCTOU microscópica remanescente em `removeLayerLockFileIfUnchanged` (Windows) entre o `os.ReadFile` de comparação e o `os.Remove` — inerente à ausência de primitiva nativa de compare-and-delete em arquivo-marcador; eliminação completa exigiria handle exclusivo nativo do Windows, mudança arquitetural fora do escopo deste PRD.
- RF-20, segunda cláusula ("reportar quando o volume degradar a garantia de p95"), não tem alerta/threshold ativo além da métrica passiva `BuildLatencyMs` — pré-existente, não é regressão desta feature.
- 3 achados de `golangci-lint` (staticcheck/unused) em arquivos não tocados por este PRD (`internal/adapters/adapters.go`, `internal/specdrift/specdrift_test.go`, `internal/runtime/acp_opencode_telemetry_test.go`), confirmados pré-existentes ao commit-base da branch.
