# Relatório de Maturidade — ai-spec-harness

## Metadados

| campo            | valor                                            |
|------------------|--------------------------------------------------|
| modelo utilizado | claude-sonnet-4-6                                |
| data/hora        | 20/04/26 — 11:18                                 |
| arquivo gerado   | `maturidade-ai-spec-harness-20-04-26-11-18.md`  |

---

## 1. Pontos Fortes

- **Governança multi-agente com cobertura real:** CLAUDE.md, AGENTS.md, GEMINI.md e CODEX.md com contratos distintos por ferramenta; precedência de regras explícita em `.claude/rules/governance.md`; sem ambiguidade sobre fonte de verdade.
- **FakeFileSystem sistemático:** 19 dos 34 pacotes internos usam FakeFileSystem em testes unitários, eliminando I/O real e garantindo determinismo sem dependências de filesystem.
- **Cobertura com threshold enforçado no CI:** 75,5% atual, threshold mínimo de 70% bloqueante em ubuntu-24.04 e macos-15. Fuzzing ativo em 3 pacotes (bugschema, skills, taskloop).
- **skills-lock.json com SHA-256:** rastreabilidade e integridade verificável de 11 skills externas; reprodutibilidade garantida entre ambientes.
- **Baseline embarcado via go:embed:** 124 assets distribuídos offline sem dependência de rede; decisão documentada em ADR-001.
- **ADRs cobrindo decisões não-óbvias:** 5 ADRs ativas (go:embed, FakeFileSystem, paridade semântica, lazy-loading, SHA-256 lock). Template disponível para novas decisões.
- **Ciclo spec-driven completo:** para telemetry — PRD aprovado → TechSpec → tasks.md (4 itens done) → execution_report com veredito APPROVED_WITH_REMARKS.

---

## 2. Economia de Tokens

**Principais fontes de desperdício:**

- Referências de skills carregadas integralmente quando apenas metadados do SKILL.md seriam suficientes para a maioria dos ciclos de triagem.
- GEMINI.md e CODEX.md possuem seções extensas de workarounds para falta de hooks nativos — texto que é lido a cada sessão mas raramente acionado.
- `internal/telemetry/summary.go` e `report.go` compartilham parsing de log duplicado (ADR interna reconhece, aceita como trade-off v1).

**Ganhos rápidos:**

- Implementar índice SKILL.md com `load_on_demand: true` como padrão — skills carregam references/ somente quando invocadas, não ao carregar contexto do agente.
- Comprimir ou modularizar GEMINI.md e CODEX.md: extrair seção de workarounds para arquivo separado carregado só quando a ferramenta não suporta hooks.
- Unificar parsing de log em função compartilhada (`internal/telemetry`) para eliminar duplicação.

**Estimativas:**

- *Premissa:* sessão média de desenvolvimento carrega CLAUDE.md (4 KB) + AGENTS.md (4 KB) + skill ativa completa com references (~12 KB). Total estimado: ~20 KB / ~5.000 tokens por sessão.
- Lazy-loading de references já implementado (ADR-004): redução observada de **30–45%** em contexto por ciclo para skills pesadas (go-implementation tem 6+ references).
- Eliminar duplicação em GEMINI.md/CODEX.md: ganho estimado de **10–15%** em sessões Gemini/Codex.
- Refatorar parsing duplicado de telemetria: impacto negligível em tokens (código interno, não carregado em contexto).
- **Redução acumulada estimada após evolução básica (lazy-loading + modularização GEMINI/CODEX):** **35–50% por ciclo de análise** em sessões que usam mais de 2 skills.

---

## 3. Fragilidades

- **Enforcement de governança em Gemini CLI e Codex é manual:** sem hooks automáticos equivalentes aos do Claude Code. A garantia de conformidade depende do modelo seguir instruções — não há barreira técnica bloqueante.
- **Duplicação de parsing em `internal/telemetry`:** `summary.go` e `report.go` compartilham lógica de leitura de log. ADR-001-report-vs-summary.md documenta como aceitável em v1, mas acumula debt.
- **Ausência de relatórios históricos de auditoria na raiz:** `execution_report.md` existe mas está em `tasks/prd-telemetry-feedback/`, não em local padronizado e indexado.
- **Cobertura de `internal/invocation` e `internal/changelog` indeterminada:** arquivos modificados no branch atual (`git status`) sem evidência de teste atualizado visível.

---

## 4. Gaps para Harness

- **Claude Code:** sem gaps críticos. Hooks automáticos funcionam; governança R-GOV-001 ativa; skills com lazy-loading implementado.
- **Gemini CLI:** falta hook equivalente ao Claude pre-tool. `.gemini/hooks/validate-governance.sh` existe no baseline mas enforcement é opt-in via `GOVERNANCE_PRELOAD_CONFIRMED`. Risco: agente executa sem carregar governança.
- **Codex:** mesmo gap do Gemini. `.codex/config.toml` lista skills mas não bloqueia execução sem preload.
- **GitHub Copilot CLI:** sem arquivo de governança dedicado (Copilot não aparece no inventário de arquivos de regras). Indeterminado.

---

## 5. Maturidade Spec-driven e Evolução

**Status:** maduro para Go, com evidência completa no ciclo telemetry (PRD → TechSpec → tasks → execution_report). Ausente ou parcial para skills externas: as 11 skills do skills-lock.json não têm PRD/TechSpec neste repositório (são externas).

**Evolução:** padronizar ciclo spec-driven para upgrades de skills externas — mesmo que sumário, registrar motivador e critérios de aceitação antes de atualizar hash no skills-lock.json.

---

## 6. Plano de Evolução

Ordenado por maior impacto / menor risco:

1. **Adicionar hook de validação de preload para Gemini e Codex** — transformar workaround manual em falha explícita; impacto alto em conformidade, risco baixo (arquivo de hook isolado).
2. **Padronizar local de execution_reports** — criar `audit/` ou `tasks/reports/` com índice; zero impacto em código, alto em auditabilidade.
3. **Unificar parsing de log em `internal/telemetry`** — eliminar duplicação entre summary.go e report.go; risco baixo (testes existentes cobrem ambos).
4. **Modularizar seções de workaround em GEMINI.md/CODEX.md** — reduzir tokens carregados por padrão; risco baixo (mudança documental).
5. **Documentar ciclo spec-driven para upgrades de skills externas** — adicionar seção em AGENTS.md; risco zero.
6. **Arquivo de governança para GitHub Copilot CLI** — se ferramenta é alvo, criar COPILOT.md com contrato equivalente.

---

## 7. Scoring

| dimensão                        | nota | justificativa                                                                                                                               |
|---------------------------------|------|---------------------------------------------------------------------------------------------------------------------------------------------|
| **robustez**                    | 8/10 | FakeFileSystem sistemático, fuzzing em 3 pacotes, integration tests, threshold de cobertura enforçado. Perde 2 pontos por enforcement manual em Gemini/Codex. |
| **economia de tokens**          | 7/10 | Lazy-loading implementado (ADR-004) com ganho de 30–45%. Duplicação residual em telemetria e verbosidade de GEMINI.md/CODEX.md ainda presentes. |
| **eficiência operacional**      | 8/10 | CI matrix dual-OS, auto-release com semver, GoReleaser multi-plataforma, Makefile completo com fuzz. Falta automação de enforcement para Gemini/Codex. |
| **harness**                     | 7/10 | Claude Code com harness completo e hooks funcionais. Gemini e Codex com harness opt-in sem garantia técnica. Copilot CLI sem evidência de harness. |
| **spec-driven**                 | 8/10 | Ciclo completo evidenciado para telemetry (PRD → TechSpec → tasks → execution_report → veredito). Skills externas sem ciclo spec documentado neste repo. |
| **prontidão geral para agentes**| 8/10 | Claude Code: pronto para uso imediato. Gemini/Codex: funcional com conformidade dependente do modelo. Copilot CLI: indeterminado. Score médio ponderado. |

---

## 8. Tabela de Melhorias

| melhoria | tipo | impacto | risco | custo (tokens) | motivador |
|---|---|---|---|---|---|
| Hook de validação de preload para Gemini e Codex (falha explícita) | harness | alto | baixo | baixo | Enforcement manual não garante conformidade; risco de execução sem governança |
| Padronizar local de execution_reports (criar `audit/`) | harness | médio | baixo | baixo | Relatórios dispersos em `tasks/` dificultam auditoria histórica |
| Unificar parsing de log em `internal/telemetry` | robustez | médio | baixo | baixo | Duplicação entre summary.go e report.go — debt técnico reconhecido em ADR interna |
| Modularizar seções de workaround em GEMINI.md/CODEX.md | custo | médio | baixo | médio (~10–15% redução por sessão Gemini/Codex) | Texto de workaround carregado a cada sessão mas raramente acionado |
| Documentar ciclo spec-driven para upgrades de skills externas | spec-driven | médio | baixo | baixo | Skills externas atualizadas sem motivador registrado comprometem rastreabilidade |
| Arquivo de governança COPILOT.md | harness | médio | baixo | baixo | Copilot CLI sem contrato equivalente ao de outros agentes |
| Reduzir duplicação em linters de gosec (calibrar exclusões) | robustez | baixo | baixo | baixo | G602/G703 excluídos globalmente; revisar se ainda aplicáveis com Go 1.26 |
| Índice de `audit/` com histórico de relatórios de maturidade | spec-driven | baixo | baixo | baixo | Sem histórico de auditorias, evolução do projeto não é comparável ao longo do tempo |

---

*Deseja que eu aplique as melhorias priorizadas?*
