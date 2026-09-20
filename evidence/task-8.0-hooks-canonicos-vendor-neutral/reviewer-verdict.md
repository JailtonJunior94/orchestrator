# Parecer do revisor independente — Tarefa 8.0

Subagente `reviewer`, contexto isolado (prd.md, techspec.md, adr-001-hook-contract-fonte-unica-projecoes.md), diff restrito aos arquivos da tarefa 8.0.

## Veredito: APPROVED_WITH_REMARKS

**Achados:**

- **[LOW]** `.claude/settings.json` (novo, self-dogfood) grava `env GOVERNANCE_PRELOAD_MODE=warn` diretamente no comando `PreToolUse`. Confirmado em `.agents/hooks/validate-preload.sh` que isso afeta *apenas* o fallback final ("governança não carregada", sem alvo extraível) — `hook-prereq-gate.sh` e `git-operation-gate.sh` continuam ativos e bloqueantes incondicionalmente (verificado na prática: uma chamada Bash sem alvo de arquivo foi bloqueada por `git-operation-gate`/`hook-prereq-gate` durante a revisão, mesmo com o `warn` embutido). O trade-off é real mas está corretamente escopado e auditado (`audit_escape` grava em `.aispec/governance-escapes.log`). Ressalva: por estar em `.claude/settings.json` **versionado** (não em `settings.local.json`), a relaxação vale para qualquer sessão interativa neste repo, não só para o executor automatizado que motivou a mudança. Recomenda-se documentar essa abrangência no PRD/ADR ou considerar escopar via variável específica do task-loop — não bloqueante, pois a mitigação é honesta, auditada e não desarma os dois gates duros.
- **[LOW]** `CopilotSessionEndHookKey = "agentStop"` (em `internal/upgrade/copilot_governance.go`, não tocado nesta tarefa) ficou com nome semanticamente incorreto agora que `agentStop` corresponde a `PointBeforeComplete` e existe um `SessionEnd` real e distinto. Não é regressão desta tarefa (constante preexistente, fora do diff), mas vale registrar como dívida para renomear quando esse arquivo for tocado de novo.

**Verificações que confirmaram a qualidade do diff (sem achados):**

1. Os 20 pares `(provedor, evento)` estão declarados explicitamente e com prova real: `TestTwentyProviderEventPairsHaveExplicitState` (`internal/runtime/specs/registry_test.go`) confronta os 20 estados esperados linha a linha, batendo com a tabela do ADR-001 (Claude/Codex/Copilot = `SupportVerified` nos 5 pontos; OpenCode = adapter/adapter/unsupported para SessionStart/BeforeComplete/SessionEnd).
2. RF-56 é prova real, não tautologia: `TestAdapterContainsNoPolicyDecision` faz AST scan de `registry.go`/`native_config.go`, e `TestAdapterPolicyDecisionGateDetectsAPlantedViolation` planta uma violação em arquivo temporário e confirma que o scanner a detecta.
3. Nenhum `SupportVerified` declarado sem confirmação: `NewObservedPointCoverage`/`NewAdapterPointCoverage`/`NewPointCoverage` todos validam a `nativeKey` contra `RecognizedNativeKeys` (vocabulário declarado por CLI), e a alegação de suporte nativo real (SessionStart/SessionEnd das 3 CLIs) está documentada e justificada no ADR-001 com evidência textual.
4. A remoção de `sessionEnd`/`SessionEnd` de `ObsoleteCopilotHookKeys` está coberta por dois testes novos que comprovam a não-regressão: `TestInstallCopilotMigratesObsoleteStopKeysInSettings` (só migra `stop`/`Stop`) e `TestInstallCopilotPreservesSessionEndAsADistinctEvent` (garante que `sessionEnd`/`SessionEnd` sobrevivem ao merge sem vazar para `agentStop`).
5. Zero comentários nos arquivos Go novos/tocados inspecionados (`registry.go`, `enforcement.go`, `provider_limitations.go`, `adapter_policy_gate_test.go`); identificadores em inglês.
6. Nenhuma regressão óbvia — `go build`, `go vet` e `go test` limpos nos pacotes tocados (`specs`, `install`, `uninstall`, `upgrade`, `doctor`, `contextgen`: 452 testes passando). `.claude/settings.json` é corretamente rastreado por `tracking.Tracker.WriteFile` (registra `created`/`merged`), então tanto a remoção via `InstalledFiles` quanto o reverse-merge via `claudeVersionedSettingsRelPath` funcionam corretamente.
7. A mitigação do item de `.claude/settings.json` é aceitável como dívida documentada; não bloqueia a aprovação.

**Critérios de aceite (task 8.0) — confronto do revisor:**
- 20 pares com estado explícito, sem omissão → atendido.
- Nenhum par suportado sem hook nativo/limitation → atendido.
- `.claude/settings.json` versionado com fiação canônica, gate falha se ausente/incompleto → atendido.
- `.github/copilot/settings.json` e 4 fontes do Codex reconhecidas → atendido.
- Gate RF-56 verde + contraprova vermelha → atendido.
- `doctor --codex-trust` no gate de release → atendido.
- `make test lint vet` verdes → atendido nos pacotes tocados verificados diretamente.
- `make check-hooks-sync`, `make integration`, `make coverage` → não re-executados pelo revisor (fora do escopo de arquivos desta tarefa e/ou dependentes de tasks concorrentes 10.0/11.0); risco residual, não bloqueante.
- Nenhum panic no init do catálogo → atendido.

**Riscos residuais apontados pelo revisor:** a pesquisa em documentação oficial das 4 CLIs que justifica `SupportVerified` em SessionStart/SessionEnd não foi re-verificada contra a documentação upstream nesta rodada de revisão — aceita com base no ADR-001, que registra a pesquisa como já realizada em tarefa anterior.
