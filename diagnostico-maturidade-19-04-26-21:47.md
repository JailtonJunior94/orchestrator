# Diagnostico de Maturidade — ai-spec-harness

## Metadados

| campo | valor |
|-------|-------|
| modelo | Claude Opus 4.6 (1M context) |
| data/hora | 19/04/2026 21:47 |
| arquivo | `diagnostico-maturidade-19-04-26-21:47.md` |

---

## 1. Pontos Fortes

- **Governanca multi-agente concreta.** Suporta Claude Code (hooks + agents + skills), Codex (`config.toml`), Gemini (commands + hooks), Copilot (`copilot-instructions.md`) e Cursor (`.cursorrules`). Cada ferramenta recebe artefatos especificos, nao apenas copia textual.
- **Contrato de carregamento obrigatorio.** `AGENTS.md` define ordem de leitura e pre-requisitos antes de qualquer operacao. Isso reduz hallucination e desvio de escopo em todos os agentes.
- **Lazy-loading de referencias.** O `agent-governance` carrega `ddd.md`, `security.md`, `testing.md` etc. sob demanda, evitando injecao de contexto desnecessario.
- **Self-dogfooding real.** O harness aplica a propria governanca (`.ai_spec_harness.json` manifesta instalacao no proprio repo).
- **Paridade semantica, nao textual.** `internal/parity` valida invariantes (presenca de referencias canonicas, consistencia de paths) em vez de comparacao textual fragil.
- **Controle de profundidade de invocacao.** `check-invocation-depth.sh` impede loops infinitos de skills recursivas (limite padrao: 2).
- **Hooks de pre/pos-edicao.** `validate-preload.sh` e `validate-governance.sh` interceptam Edit/Write no Claude Code para forcar conformidade.
- **Skills-lock com SHA-256.** 11 skills externas rastreadas com hash, garantindo integridade e reproducibilidade.
- **Testes com FakeFileSystem.** Abstracao de filesystem permite testes unitarios sem I/O real, com 49 arquivos de teste e 428 funcoes de teste.
- **CI robusto.** Matrix ubuntu-24.04 + macos-15, unit + integration + vet + lint, release automatizado com semver-next e GoReleaser.

---

## 2. Economia de Tokens

### Principais fontes de desperdicio

1. **Carregamento completo de AGENTS.md + governance em toda interacao.** Mesmo tarefas triviais (ex: rename de variavel) forcam leitura de contrato completo. Premissa: ~2.500 tokens por carregamento desnecessario.
2. **References redundantes entre skills.** `go-implementation` e `object-calisthenics-go` compartilham conceitos de DDD e error-handling; quando ambas sao carregadas, ha sobreposicao de ~30-40% do conteudo.
3. **SKILL.md verbosos.** Alguns skills (agent-governance, go-implementation) tem SKILL.md com 6+ estagios detalhados que sao injetados integralmente mesmo quando apenas 1-2 estagios sao relevantes.
4. **Snapshots de AGENTS.md em testdata.** 9 snapshots sao uteis para testes mas, se indexados por agente, inflam contexto sem necessidade.

### Ganhos rapidos

- **Classificacao de complexidade pre-carregamento:** Antes de carregar referencias, classificar a tarefa como trivial/media/complexa. Tarefas triviais pulariam carregamento de referencias. Reducao estimada: **15-25% por ciclo trivial**.
- **Fragmentacao de SKILL.md por estagio:** Permitir carregamento seletivo de estagios (ex: apenas "Estagio 3: Executar"). Reducao estimada: **10-20% por ciclo**.
- **Deduplicacao de referencias compartilhadas:** Consolidar `shared-*.md` como fonte unica e referenciar em vez de duplicar. Reducao estimada: **5-10%**.

### Estimativa acumulada

- **Ciclo atual estimado:** ~8.000-12.000 tokens de contexto de governanca por interacao complexa.
- **Apos evolucao basica (classificacao + fragmentacao):** Reducao de **20-35%**, resultando em ~5.500-9.000 tokens.
- **Premissa:** Baseado em contagem de caracteres dos SKILL.md e references com estimador char/4.

---

## 3. Fragilidades

1. **Ausencia de GEMINI.md e CODEX.md na raiz.** Gemini e Codex tem configuracoes em subdiretorios, mas nao tem documento de governanca de nivel superior equivalente ao `CLAUDE.md`. Isso cria assimetria na experiencia entre agentes.
2. **Hooks exclusivos do Claude Code.** `validate-preload.sh` e `validate-governance.sh` so executam no Claude Code. Gemini tem 1 hook basico; Codex e Copilot nao tem nenhum. A garantia de conformidade e desigual.
3. **Cobertura de testes de integracao limitada.** Apenas 4 arquivos com build tag `integration`. Pacotes criticos como `install`, `upgrade`, `parity` e `specdrift` dependem primariamente de FakeFileSystem — cenarios de filesystem real (permissoes, symlinks quebrados, paths longos) nao sao cobertos.
4. **Sem metricas de cobertura no CI.** O workflow `test.yml` nao gera relatorio de cobertura (`-coverprofile`). Nao ha threshold minimo de cobertura.
5. **Nenhum teste de mutacao ou fuzz.** A referencia `testing.md` menciona fuzz e property-based testing, mas nao ha evidencia de implementacao no repositorio.
6. **Precedencia de regras e implicitamente documentada.** A hierarquia de 5 niveis em `governance.md` exige interpretacao humana para resolver conflitos. Nao ha mecanismo programatico de resolucao.

---

## 4. Gaps para Harness

1. **Validacao de governanca pos-instalacao ausente para Codex/Copilot.** O harness instala artefatos, mas nao valida se estao sendo efetivamente consumidos por esses agentes. Claude Code tem hooks; os demais nao.
2. **Sem harness de regressao de governanca.** Nao ha teste automatizado que verifique se uma alteracao em `SKILL.md` ou `references/` quebra o contrato de carregamento ou a paridade entre agentes.
3. **Drift detection parcial.** `internal/specdrift` detecta drift entre specs, mas nao cobre drift entre artefatos instalados no projeto-alvo e o baseline embarcado.
4. **Sem validacao de schema para SKILL.md.** O frontmatter de skills nao e validado contra um JSON Schema formal. Erros de formatacao so sao detectados em runtime.
5. **Telemetria sem feedback loop.** `internal/telemetry` coleta dados, mas nao ha evidencia de dashboard ou alerta que feche o ciclo de melhoria.

---

## 5. Maturidade Spec-Driven e Evolucao

O projeto demonstra maturidade spec-driven **acima da media** para projetos Go de porte similar:

- **Presente:** SKILL.md com frontmatter padronizado, contrato de carregamento em AGENTS.md, references lazy-loaded, enforcement-matrix documentando capacidades por agente, invariantes semanticas em `parity`.
- **Parcial:** Tasks em `docs/tasks/` e `tasks/` definem criterios de aceitacao, mas nao ha evidencia de PRD formal ou tech spec completo para features ja entregues. O `PLANO-EXECUCAO.md` funciona como roadmap, nao como spec.
- **Ausente:** Nenhum PRD aprovado, nenhuma tech spec com ADRs, nenhum registro formal de decisoes arquiteturais. O fluxo `create-prd → create-technical-specification → create-tasks → execute-task` esta implementado como skills, mas nao ha evidencia de uso no proprio repositorio.

**Evolucao recomendada:** Aplicar o proprio fluxo spec-driven do harness ao harness. Criar PRD e tech spec para as proximas features usando as skills `create-prd` e `create-technical-specification`.

---

## 6. Plano de Evolucao

Priorizado por impacto/risco (maior impacto, menor risco primeiro):

1. **Adicionar `-coverprofile` ao CI e definir threshold.** Impacto alto, risco baixo. Visibilidade imediata sobre cobertura real.
2. **Criar GEMINI.md e CODEX.md na raiz.** Impacto medio, risco baixo. Equaliza experiencia entre agentes.
3. **Implementar classificacao de complexidade pre-carregamento.** Impacto alto, risco medio. Economia de tokens significativa em tarefas triviais.
4. **Adicionar JSON Schema para frontmatter de SKILL.md.** Impacto medio, risco baixo. Previne erros silenciosos em skills.
5. **Expandir testes de integracao para `install`, `upgrade`, `parity`.** Impacto alto, risco medio. Cobre cenarios de filesystem real.
6. **Implementar harness de regressao de governanca.** Impacto alto, risco medio. Garante que mudancas em skills nao quebrem contratos.
7. **Fragmentar SKILL.md por estagio para carregamento seletivo.** Impacto medio, risco medio. Requer mudanca na forma como agentes consomem skills.
8. **Documentar ADRs para decisoes arquiteturais existentes.** Impacto medio, risco baixo. Formaliza decisoes implicitas.

---

## 7. Scoring

| dimensao | nota | justificativa |
|----------|------|---------------|
| robustez | 7 | FakeFileSystem, 49 arquivos de teste, CI multi-plataforma. Perde pontos por ausencia de cobertura reportada, fuzz testing e testes de integracao limitados. |
| economia de tokens | 7 | Lazy-loading de referencias e carregamento sob demanda sao diferenciais. Perde por carregamento obrigatorio de contrato completo em tarefas triviais e sobreposicao entre skills. |
| eficiencia operacional | 8 | Makefile com targets claros, GoReleaser automatizado, semver-next integrado, Homebrew tap. Pipeline de release e maduro e reproducivel. |
| harness | 8 | Hooks pre/pos-edicao, controle de profundidade, skills-lock com SHA-256, self-dogfooding. Perde por assimetria entre agentes (hooks so no Claude Code) e ausencia de validacao pos-instalacao. |
| spec-driven | 6 | Skills com SKILL.md padronizado e contrato de carregamento sao solidos. Perde significativamente por ausencia de PRDs, tech specs e ADRs formais no proprio repositorio. |
| prontidao geral para agentes | 7.5 | Pronto para uso produtivo com Claude Code. Codex, Gemini e Copilot tem suporte funcional mas com gaps de enforcement. Framework de governanca e superior a maioria dos projetos open-source. |

---

## 8. Tabela de Melhorias

| melhoria | tipo | impacto | risco | custo (tokens) | motivador |
|----------|------|---------|-------|----------------|-----------|
| Adicionar `-coverprofile` e threshold ao CI | robustez | alto | baixo | baixo | Sem visibilidade de cobertura real; risco de regressao silenciosa |
| Criar GEMINI.md e CODEX.md na raiz | harness | medio | baixo | baixo (~500 tokens cada) | Assimetria de governanca entre agentes |
| Classificacao de complexidade pre-carregamento | custo | alto | medio | medio (~15-25% reducao/ciclo) | Carregamento desnecessario em tarefas triviais |
| JSON Schema para frontmatter de SKILL.md | spec-driven | medio | baixo | baixo | Erros de formatacao detectados tarde demais |
| Expandir testes de integracao (install, upgrade, parity) | robustez | alto | medio | medio | Cenarios de filesystem real nao cobertos |
| Harness de regressao de governanca | harness | alto | medio | medio | Mudancas em skills podem quebrar contratos sem deteccao |
| Fragmentar SKILL.md por estagio | custo | medio | medio | alto (~10-20% reducao/ciclo) | Injecao integral de skills quando so 1-2 estagios sao necessarios |
| Documentar ADRs para decisoes existentes | spec-driven | medio | baixo | baixo | Decisoes arquiteturais implicitas dificultam onboarding |
| Adicionar hooks de validacao para Gemini/Codex | harness | medio | medio | medio | Enforcement desigual entre agentes |
| Implementar fuzz testing em parsers criticos | robustez | medio | baixo | medio | Parsers de frontmatter e taskloop sem cobertura adversarial |
| Deduplicar referencias compartilhadas entre skills | custo | baixo | baixo | baixo (~5-10% reducao) | Sobreposicao de conteudo entre go-implementation e OC |
| Fechar feedback loop de telemetria | eficiencia | medio | medio | alto | Telemetria coletada sem consumo visivel |

---

Deseja que eu aplique as melhorias priorizadas?
