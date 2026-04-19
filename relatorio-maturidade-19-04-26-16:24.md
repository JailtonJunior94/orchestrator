# Relatorio de Maturidade para Agentes de IA

---
- **Modelo utilizado:** Claude Opus 4.6 (1M context)
- **Data/hora:** 19/04/2026 16:24
- **Arquivo gerado:** `relatorio-maturidade-19-04-26-16:24.md`
---

## 1. Pontos Fortes

- **Governanca estruturada e em camadas.** AGENTS.md + agent-governance SKILL.md + governance.md formam uma cadeia de precedencia explicita com resolucao de conflitos documentada (R-GOV-001). Isso e raro e reduz ambiguidade para qualquer agente.
- **25 skills embarcadas com frontmatter padronizado.** Cada skill tem `name`, `version`, `description` e procedimentos em etapas numeradas — estrutura que agentes consomem com previsibilidade.
- **Controle de profundidade de invocacao** (`AI_INVOCATION_DEPTH`) previne loops entre skills como review -> bugfix -> review. Mecanismo defensivo concreto.
- **Golden tests com snapshots.** 9 snapshots em `testdata/snapshots/` + 8 projetos-baseline cobrem go-microservice, node-api, python-api, polyglot-monorepo etc. Isso ancora regressoes de geracao.
- **Script de paridade embedded vs remoto** (`check-embedded-parity.sh`) com opcao `--fix` — garante sincronizacao com repositorio de governanca externo.
- **Cobertura de testes saudavel.** 28 de 33 pacotes com testes, todos passando. Testes de integracao separados por build tag. CI roda em Ubuntu e macOS.
- **Dependencias minimas.** Apenas cobra + jsonschema + tiktoken-go como dependencias diretas. Superficie de ataque e custo de contexto baixos.
- **Multi-plataforma real.** GoReleaser gera binarios para Linux/macOS/Windows (amd64+arm64) com Homebrew tap.

## 2. Economia de Tokens

**Principais fontes de desperdicio:**
- SKILL.md com blocos repetitivos de "contrato de carga base" duplicados em cada skill (~15-25 linhas x 25 skills). Agentes recarregam essas instrucoes toda invocacao.
- Referencias em `references/*.md` carregadas por gatilho textual sem indice resumido — agente precisa ler o SKILL.md inteiro para decidir quais referencias carregar.
- Ausencia de CLAUDE.md na raiz forca agentes a navegar AGENTS.md -> agent-governance -> references para montar contexto base. Um arquivo raiz com preload concentrado reduziria round-trips.

**Ganhos rapidos:**
- Criar CLAUDE.md raiz com preload de regras base: -15-25% de tokens no bootstrap.
- Extrair bloco repetitivo de carga base para um unico `PRELOAD.md` referenciado por ponteiro: -10-15% por skill invocada.
- Adicionar indice de referencias com gatilhos em 1 linha cada: -5-10% quando agente avalia quais refs carregar.

**Estimativa de reducao:**
- Por ciclo individual: 20-35% menos tokens no carregamento de contexto.
- Acumulada apos evolucao basica: 25-40% considerando sessoes multi-skill.

*Premissa: agente carrega 3-5 skills por sessao media, cada skill consome ~800-1500 tokens de instrucao.*

## 3. Fragilidades

- **5 pacotes sem testes:** `config`, `doctor`, `embedded`, `inspect` e o pacote raiz. Pacotes `doctor` e `inspect` sao comandos de diagnostico que poderiam ter testes de smoke basicos.
- **Ausencia de Makefile/Taskfile.** Agentes que tentam descobrir como buildar/testar/lintar precisam inferir ou ler o CI. Isso custa tokens e e nao-deterministico entre agentes.
- **Sem contratos formais de API** (OpenAPI, proto, JSON Schema de entrada/saida dos comandos). O `bug-schema.json` e o unico schema formal encontrado.
- **Nenhum arquivo Node.js no projeto.** As skills `node-implementation` e `node-monorepo` existem como templates mas nao ha codigo Node para validar que funcionam. Evidencia indeterminada para Node.
- **Python presente apenas como scripts auxiliares** (validacao, paridade). Sem testes unitarios para os scripts Python embarcados exceto `test_validate_prompt.py`.
- **Sem CLAUDE.md na raiz.** Agentes Claude Code nao recebem contexto automatico ao abrir o projeto.

## 4. Gaps para Harness

- **Falta Makefile ou Taskfile como interface padrao.** Agentes como Codex e Gemini CLI dependem de descobrir comandos de build/test/lint. Sem interface padrao, cada agente improvisa — risco de inconsistencia.
- **Falta `.ai_spec_harness.json` no proprio repositorio.** O harness distribui configs para projetos-alvo mas nao aplica a si mesmo (bootstrapping incompleto).
- **Hooks de validacao existem apenas para Claude e Gemini.** Copilot e Codex nao tem hooks equivalentes. O `copilot-instructions.md` e `.codex/config.toml` sao gerados no alvo mas nao validados por harness proprio.
- **Sem smoke test para o binario `ai-spec` compilado.** O CI roda `go test` mas nao executa `ai-spec install --dry-run` ou equivalente contra um fixture.

## 5. Maturidade Spec-Driven e Evolucao

O projeto demonstra maturidade spec-driven **parcial mas acima da media**:
- Skills com frontmatter padronizado servem como especificacoes operacionais para agentes.
- Governanca com precedencia e severidade (`hard` vs `guideline`) e incomum e positiva.
- Snapshots como golden tests ancoram o comportamento esperado de geracao.
- Porem, **nao ha especificacao formal dos comandos CLI** (flags, inputs, outputs, exit codes). A "spec" e o codigo + testes.
- Nao ha spec de contrato entre o harness e os projetos-alvo (o que e gerado, onde, com que formato).

**Evolucao recomendada:** Adicionar um `COMMANDS.md` ou schema JSON descrevendo cada comando, suas flags e seus contratos de saida. Isso fecha o loop spec-driven para o proprio CLI.

## 6. Plano de Evolucao

Priorizado por maior impacto / menor risco:

1. **Criar CLAUDE.md na raiz** com preload de AGENTS.md + agent-governance. Impacto imediato em economia de tokens e UX para Claude Code.
2. **Criar Makefile com targets padrao** (`build`, `test`, `test-integration`, `lint`, `vet`). Normaliza interface para todos os agentes.
3. **Adicionar smoke test do binario** no CI: `ai-spec install --dry-run` contra fixture `testdata/fixtures/minimal-project`.
4. **Extrair bloco de carga base repetitivo** para um `PRELOAD.md` referenciado por ponteiro em cada SKILL.md.
5. **Adicionar testes para `doctor` e `inspect`** — sao comandos de diagnostico mas sem cobertura.
6. **Documentar contratos CLI** em schema JSON ou COMMANDS.md com flags, inputs e exit codes.
7. **Adicionar testes para scripts Python embarcados** (validation scripts usados em hooks).
8. **Aplicar harness ao proprio repositorio** (dogfooding do `.ai_spec_harness.json`).

## 7. Scoring

| Criterio | Nota | Justificativa |
|---|---|---|
| **Robustez** | 7/10 | 28/33 pacotes testados, CI multi-OS, golden tests com snapshots. Perde pontos pela ausencia de smoke test do binario e 5 pacotes sem testes. |
| **Economia de tokens** | 5/10 | Blocos repetitivos de carga base em 25 skills, ausencia de CLAUDE.md raiz, referencias sem indice resumido. Estrutura funcional mas com desperdicio mensuravel por sessao. |
| **Eficiencia operacional** | 6/10 | Sem Makefile/Taskfile forca agentes a inferir comandos. CI e claro mas nao ha interface padrao local. Dependencias minimas e build automatizado compensam parcialmente. |
| **Harness** | 7/10 | Harness maduro para distribuicao em projetos-alvo com suporte a 4 agentes. Perde pontos por nao se aplicar a si mesmo e por falta de smoke test do binario. |
| **Spec-driven** | 6/10 | Skills com frontmatter padronizado, governanca com precedencia, snapshots como golden tests. Falta especificacao formal dos comandos CLI e contrato harness-alvo. |
| **Prontidao geral para agentes** | 6/10 | Funcional e acima da media para projetos Go. Governanca real, skills bem estruturadas, testes solidos. Gaps em economia de tokens e interface padrao de build impedem nota mais alta. |

## 8. Tabela de Melhorias

| Melhoria | Tipo | Impacto | Risco | Custo (tokens) | Motivador |
|---|---|---|---|---|---|
| Criar CLAUDE.md na raiz com preload | custo | alto | baixo | baixo (~15-25% reducao bootstrap) | Agentes Claude Code nao recebem contexto automatico |
| Criar Makefile com targets padrao | eficiencia | alto | baixo | baixo | Agentes inferem comandos de build/test com custo variavel |
| Smoke test do binario no CI | robustez | alto | baixo | baixo | Nenhuma validacao do artefato final compilado |
| Extrair bloco de carga base para PRELOAD.md | custo | medio | baixo | medio (~10-15% reducao por skill) | 25 skills com bloco repetitivo de 15-25 linhas |
| Testes para doctor e inspect | robustez | medio | baixo | baixo | 2 pacotes de diagnostico sem cobertura |
| Schema JSON dos comandos CLI | spec-driven | medio | baixo | medio | Sem especificacao formal de flags/inputs/outputs |
| Testes para scripts Python embarcados | robustez | medio | baixo | baixo | Scripts de validacao sem cobertura exceto 1 |
| Dogfooding: harness aplicado ao proprio repo | harness | medio | medio | medio | Harness nao valida a si mesmo |
| Indice resumido de referencias em skills | custo | medio | baixo | baixo (~5-10% reducao) | Agente le SKILL.md inteiro para decidir refs |
| Hooks de validacao para Copilot e Codex | harness | baixo | medio | alto | Apenas Claude e Gemini tem hooks de validacao |

---

Deseja que eu aplique as melhorias priorizadas?
