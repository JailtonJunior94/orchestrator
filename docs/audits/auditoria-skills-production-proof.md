<!-- Generated: 2026-05-30T11:21:32Z -->
<!-- Auditor: GitHub Copilot CLI (claude-opus-4.8) -->
<!-- Escopo: análise read-only. Nenhuma skill foi alterada. -->

# Sumário Executivo

Auditoria production-proof da cadeia de 7 skills de governança do repositório
`orchestrator` (codebase-alvo): `create-prd`, `create-technical-specification`,
`create-tasks`, `execute-all-tasks`, `execute-task`, `review`, `bugfix`.

A cadeia é **madura e acima da média do mercado**: contratos canônicos (YAML, bug-schema
JSON, regex de `tasks.md`), validadores programáticos em shell, checkpoints atômicos,
spec-hash para drift e mapeamento explícito por ferramenta. Há **disciplina epistêmica
honesta** dentro das próprias skills — vários pontos declaram "best-effort" em vez de
fingir enforcement (ex.: `create-prd/SKILL.md:24`).

Porém **três bloqueadores impedem o selo "production-proof"**, todos verificados por leitura
direta:

1. **Falso positivo de conclusão por ausência de validação de critério de aceite / DoD.**
   Nem `execute-task/SKILL.md`, nem o validador `validate-task-evidence.sh`, nem o template
   `task-execution-report-template.md` confrontam os critérios de aceite da tarefa. A "prova"
   de teste é a string livre `Testes: pass` escrita pelo próprio agente
   (`.claude/scripts/validate-task-evidence.sh:61`). Isso é exatamente o falso positivo que o
   pedido quer eliminar.
2. **Protocolo de múltipla escolha inexistente.** Nenhuma das 7 skills menciona "múltipla
   escolha" ou opção "(Recomendado)" antes de decisões. `create-prd` e
   `create-technical-specification` pedem "perguntas de esclarecimento" em texto livre
   (`create-prd/SKILL.md:27-35`, `create-technical-specification/SKILL.md:23-30`), sem o formato
   exigido pelo pedido.
3. **Paridade assimétrica de enforcement.** O enforcement real por hooks é nativo **apenas no
   Claude Code**; em Codex, Gemini e Copilot CLI os mesmos `.sh` só rodam se o agente os invocar
   manualmente (`agent-governance/references/enforcement-matrix.md:19-21,36-40`). Além disso, os
   formatos de subagente de Codex/Gemini são "inferidos, não validados empiricamente"
   (`execute-all-tasks/SKILL.md:144`).

**Veredito:** **production-ready com gaps relevantes**, não production-proof. Os gaps são
endereçáveis sem reescrita — são adições cirúrgicas de gate e protocolo.

---

# Score Global

- **Nota global: 76 / 100** (média ponderada das 7 skills).
- **Classificação final: production-ready com gaps relevantes (75–89).**
- **Nível de confiança global: médio-alto.** Toda skill, validador, hook, lib e schema citado
  foi lido diretamente; a incerteza residual está na execução *runtime* em Codex/Gemini/Copilot,
  que não foi exercida nesta auditoria (apenas inspeção estática).

### 3 maiores bloqueadores

1. **DoD/critério de aceite não validado** em `execute-task`/`execute-all-tasks` → falso positivo
   de `done` (`validate-task-evidence.sh:42-103`, sem checagem de aceite).
2. **Múltipla escolha ausente** em toda a cadeia, contra requisito explícito do pedido
   (grep `"múltipla escolha"` → nenhuma ocorrência nas 7 SKILL.md).
3. **Enforcement real só nativo no Claude**; demais tools dependem de invocação manual dos hooks
   (`enforcement-matrix.md:19-21`).

### 3 maiores pontos fortes

1. **Contratos canônicos determinísticos**: YAML estrito (`execute-all-tasks/SKILL.md:99`),
   `bug-schema.json` (`references/bug-schema.json`), regex de `tasks.md`
   (`pre-execute-all-tasks.sh:175-186`).
2. **Validadores programáticos + checkpoints atômicos + spec-hash**: enforcement real onde os
   hooks são chamados (`post-execute-task.sh:139-205`, `execute-task/SKILL.md:81-90`).
3. **Handoffs fortes**: sync gate `create-tasks`↔`execute-task` (`create-tasks/SKILL.md:85-96`) e
   contrato `review`→`bugfix` via schema (`review/SKILL.md:53`, `bugfix/SKILL.md:21-24`).

---

# Scorecard por Skill

| Skill | Score | Classificação | Falso positivo | Eficiência | Sinergia | Paridade | Diagnóstico curto |
|---|---:|---|---|---|---|---|---|
| `create-prd` | 70 | ready, gaps | Médio | Alta | Boa | Média | Drift gate honesto mas best-effort; sem múltipla escolha; confronto de codebase é opcional. |
| `create-technical-specification` | 75 | ready, gaps | Baixo | Alta | Forte | Média | Spec-hash real; explora repo; falta múltipla escolha antes de fechar arquitetura. |
| `create-tasks` | 81 | quase proof | Baixo | Alta | Forte | Alta | Regex canônicos + sync gate Etapa 5.5; melhor handoff da cadeia. |
| `execute-task` | 72 | ready, gaps | **Alto** | Alta | Forte | Média | Muitos gates, mas DoD/aceite não validado; `Testes: pass` é texto livre. |
| `execute-all-tasks` | 77 | ready, gaps | Médio | Alta | Forte | Média | Orquestração robusta; herda gap de aceite; hooks best-effort fora do Claude. |
| `review` | 75 | ready, gaps | Médio | Alta | Forte | Média | Veredito determinístico; leitura de prd/techspec/tasks é condicional; sem validador de evidência. |
| `bugfix` | 79 | quase proof | Baixo | Alta | Forte | Média | Schema + teste de regressão obrigatório + limite de tentativa; validador com path `.claude/` rígido. |
| **Global** | **76** | **ready, gaps** | — | — | — | — | Cadeia coesa; falta fechar aceite/DoD, múltipla escolha e paridade de enforcement. |

Pesos aplicados (rubrica do pedido): Robustez/anti-falso-positivo 25 · Aceite/DoD/evidência 20 ·
Sinergia 15 · Eficiência 10 · Economia 10 · Confronto codebase 10 · Paridade 10.

---

# Achados por Skill

## 1. `create-prd` (v1.3.1) — 70

**Forte**
- Gate de drift downstream lista artefatos derivados e para com `needs_input` mandatório
  (`create-prd/SKILL.md:15-22`).
- **Honestidade epistêmica exemplar**: declara explicitamente o limite "este gate é best-effort
  enforcement — depende do agente seguir a instrução" (`create-prd/SKILL.md:24`).
- Versionamento de spec (`spec-version`) para rastrear edições (`create-prd/SKILL.md:53`).

**Frágil**
- Confronto com o codebase é **opcional**: "Ler o contexto do repositório (README, AGENTS.md)
  apenas quando a funcionalidade depender de restrições específicas" (`create-prd/SKILL.md:39`).
  O pedido quer confronto sistemático com o codebase-alvo; aqui ele é condicional e a critério do
  agente. **Falso positivo potencial**: PRD "aprovado" sem ter olhado restrições reais do repo.
- **Sem múltipla escolha**: Etapa 2 pede perguntas nas 6 categorias, mas em texto livre
  (`create-prd/SKILL.md:27-35`). Não há formato de opções nem "(Recomendado)".

**Risco de alucinação**: escopo/RF/exclusões podem ser assumidos sem confronto se o agente julgar
a feature "simples". Sem gate programático que force a leitura do repo.

**Para virar production-proof**: tornar o confronto mínimo com o codebase mandatório (ainda que
raso) e converter as 6 categorias em perguntas de múltipla escolha quando houver ambiguidade.

## 2. `create-technical-specification` (v1.2.1) — 75

**Forte**
- Explora o repositório de forma mandatória na Etapa 2 (`create-technical-specification/SKILL.md:16-19`)
  — confronto de codebase mais forte que o do PRD.
- **Spec-hash real e portátil** injetado no topo da techspec via `ai-spec hash`
  (`create-technical-specification/SKILL.md:54-56`); base concreta para detecção de drift
  downstream.
- Mapeamento requisito→decisão→teste e ADRs separadas (`:43`, `:47-51`).

**Frágil**
- **Sem múltipla escolha** antes de fechar decisões de arquitetura: Etapa 3 pede "perguntas
  técnicas de esclarecimento" em texto livre, limitadas a 2 rodadas
  (`create-technical-specification/SKILL.md:23-30`). Decisões materiais de arquitetura são
  exatamente onde o pedido exige opções com recomendação.
- Verificação de dependências externas depende de navegação disponível; sem ela, vira suposição
  (`:20`, `:65`) — aceitável, mas amplia superfície de techspec otimista.

**Para production-proof**: protocolo de múltipla escolha para fronteiras de domínio, contratos de
interface e estratégia de teste antes de redigir.

## 3. `create-tasks` (v1.6.1) — 81 (melhor da cadeia)

**Forte**
- **Sync gate Etapa 5.5**: compara conjunto de skills em `tasks.md` (coluna `Skills`) com a seção
  `## Skills Necessárias` de cada task file e **para com `failed: skills sync drift`** se
  divergirem (`create-tasks/SKILL.md:85-96`). Fecha falso positivo entre o que o humano lê e o que
  `execute-task` carregaria.
- Carrega/declara skills por task com descoberta agnóstica em runtime (`ls .agents/skills/`,
  leitura de `description`) — atende diretamente ao pedido "carregar as skills para uso de cada
  task" (`create-tasks/SKILL.md:39-67`).
- Regex canônicos para `Dependências`/`Paralelizável`/`Status` com falha explícita em valor
  malformado (`:70-82`); spec-hash sincronizado via `ai-spec sync-spec-hash` (`:34-37`).
- Teto de tarefas (`AI_MAX_TASKS_PER_PRD`, default 10) contra over-splitting (`:24`).

**Frágil**
- A enumeração de skills "auto-carregadas" é **hardcoded em prosa** (`create-tasks/SKILL.md:48-50`)
  e o próprio texto admite que precisa ser atualizada manualmente quando surgir nova skill de
  linguagem. Risco de drift de manutenção (não de execução).
- Eficiência da declaração de skills depende de casamento semântico feito pelo agente; sem gate
  além do sync de presença.

**Para production-proof**: derivar a lista de auto-carregadas de metadado (frontmatter
`depends_on`/categoria) em vez de prosa, para eliminar drift de manutenção.

## 4. `execute-task` (v1.5.1) — 72 (maior fonte de falso positivo)

**Forte**
- Cadeia de gates densa: resolução de lib de profundidade (`SKILL.md:13-22`), **gate de binário
  `ai-spec` sem degradação silenciosa** (`:23-30`), pre-flight de drift/skills (`:31-33`),
  checkpoint YAML atômico antes de mutar `tasks.md` (`:81-90`), lock em `tasks.md` (`:86-90`).
- Mapeamento de veredito do reviewer com **escalada de remark crítico** (`APPROVED_WITH_REMARKS`
  com `[critical]` → `blocked`) (`:70-76`).

**Frágil — falso positivo ALTO**
- **Critério de aceite e DoD não são validados.** Busca direta: nenhuma ocorrência de "critério de
  aceite", "DoD" ou "definition of done" em `execute-task/SKILL.md`, no validador, ou no template
  (verificado por grep). O template de evidência não tem seção de critérios de aceite
  (`task-execution-report-template.md:1-44`).
- O validador `validate-task-evidence.sh` confere **presença de seções e a string literal**
  `Testes: pass`/`Lint: pass` (`:61-62`), não a execução real. Um agente pode escrever
  `Testes: pass` sem rodar teste e passar no validador. Mitigado parcialmente por F35 (DiffSHA vs
  git) **mas F35 é opt-in** (`post-execute-task.sh:179`, `AI_VALIDATE_GIT_HISTORY` default 0).
- A própria Etapa 4.3 ("Verificar critérios com evidência explícita", `SKILL.md:68`) depende de
  disciplina do agente — não há gate que ligue cada critério de aceite da task a uma evidência.

**Para production-proof (P0)**: adicionar seção obrigatória "Critérios de Aceite" no template, com
um item por critério e status verificável, e estender `validate-task-evidence.sh` para exigir que
cada critério da task file apareça atendido no report. Considerar tornar F35 default-on.

## 5. `execute-all-tasks` (v1.7.2) — 77

**Forte**
- **Enforcement programático real** via `pre-execute-all-tasks.sh` (regex, gaps F29, cross-PRD
  spec-hash F18, ciclo F27 — `pre-execute-all-tasks.sh:148-326`) e `post-execute-task.sh` por
  tarefa.
- **Wait-all-then-halt** contra race em waves paralelas (`SKILL.md:105-108`); checkpoint do
  orquestrador append-only com rename atômico (`:110-115`); fallback de YAML ausente via
  checkpoint (`:90-93`).
- Isolamento de contexto ≤100 tokens/tarefa por subagent fresh (`:11-14`) — economia forte.

**Frágil**
- **Herda o gap de aceite/DoD** do `execute-task`: a cadeia de validação confere evidência física,
  status canônico e consistência de `tasks.md` (`:99-102`), mas não critério de aceite.
- **Hooks best-effort fora do Claude**: "Se ausente em todos os caminhos, prosseguir com validação
  textual (modo legado)" (`SKILL.md:22`) — degrada para texto se o hook não for achado/chamado.
- **Soft timeout não mata subagent**: "tools não oferecem kill nativo; subagent continua
  consumindo tokens em background" (`:72`). Honesto, mas é custo real em projetos grandes.

**Para production-proof**: tornar a falha do hook bloqueante (não "modo legado" silencioso) quando
o tool for capaz, e propagar o gate de aceite herdado.

## 6. `review` (v1.2.0) — 75

**Forte**
- **Veredito determinístico** por mapa severidade→veredito (`review/SKILL.md:56-65`); emite bugs no
  formato `bug-schema.json` para `bugfix` (`:53`) — handoff canônico.
- Budget de revisão (`AI_REVIEW_MAX_FILES` 8, `AI_REVIEW_MAX_DIFF_LINES` 400) com `BLOCKED` pedindo
  fatiamento (`:17-20`) — economia e anti-ruído.
- Revisão incremental pós-bugfix via `AI_REVIEW_PRIOR_SHA` (`:15`) — evita re-revisar o PR inteiro.

**Frágil**
- **Confronto com tasks/prd/techspec é condicional**: lê esses docs "somente quando o diff toca
  arquivo citado neles" (`review/SKILL.md:21`). O pedido quer que `review` confronte as tasks para
  garantir o que foi desenvolvido; aqui a confrontação de critério de aceite pode ser pulada se o
  diff não tocar arquivos citados. **Falso negativo perigoso**: aprovar diff que não cumpre o
  critério da task.
- **Sem persistência de evidência nem validador próprio**: diferente de `execute-task` e `bugfix`,
  não há `validate-review-evidence.sh` nem template. No modo `--auto-review`, o `review.md` é
  persistido sem validação estrutural (assimetria de garantia).

**Para production-proof**: ler a task ativa e confrontar explicitamente cada critério de aceite no
veredito, independentemente de o diff tocar os arquivos citados.

## 7. `bugfix` (v1.1.1) — 79

**Forte**
- Entrada canônica validada contra `bug-schema.json` por script, com **fallback manual quando
  `jsonschema` não está disponível** (`bugfix/SKILL.md:21-24`).
- **Teste de regressão obrigatório por bug** reproduzindo `reproduction`/`expected`
  (`:41`); limite de 2 tentativas por bug → `failed` (`:43-44`); estados canônicos
  (`fixed|blocked|skipped|failed`).
- Validador de evidência exige causa raiz, teste de regressão e validação
  (`validate-bugfix-evidence.sh:78-88`); depth-aware (não re-invoca review dentro do ciclo)
  (`bugfix/SKILL.md:67`).

**Frágil**
- Path do validador é rígido em `.claude/scripts/validate-bugfix-evidence.sh` na Etapa 5.5
  (`bugfix/SKILL.md:51`), enquanto outras skills resolvem em cascata (`.claude` → `.agents` →
  `scripts`). Em projeto que copiou só `.agents/`, esse caminho some → risco de pular validação.
- Rastreabilidade RF no validador é **opt-in** via `--rf` (`validate-bugfix-evidence.sh:99-105`);
  sem o flag, não há checagem de que o fix mapeia a um requisito.

**Para production-proof**: resolver o caminho do validador em cascata (igual `execute-task`) e
passar os RF/tasks afetados ao validador por padrão.

---

# Análise de Sinergia e Handoffs

Fluxo: `create-prd` → `create-technical-specification` → `create-tasks` →
`execute-all-tasks`/`execute-task` → `review` → `bugfix`.

**Handoffs fortes (verificados)**
- **PRD→TechSpec→Tasks por spec-hash**: techspec grava `spec-hash-prd`
  (`create-technical-specification/SKILL.md:54-56`); `create-tasks` sincroniza ambos os hashes
  (`create-tasks/SKILL.md:34-37`); `execute-task`/`execute-all-tasks` checam drift
  (`execute-task/SKILL.md:33`, `pre-execute-all-tasks.sh:221-271`). Cadeia de rastreabilidade
  coesa.
- **Tasks→Execute por skills sync**: coluna `Skills` ↔ `## Skills Necessárias` validada em ambos
  os lados (`create-tasks/SKILL.md:85-96` e `execute-task/SKILL.md:47-56`).
- **Review→Bugfix por schema**: `bug-schema.json` é o contrato único, validado por script em
  `bugfix` (`review/SKILL.md:53`, `bugfix/SKILL.md:21-24`). Excelente determinismo.

**Quebras / zonas cinzentas**
- **Critério de aceite não atravessa o handoff**: ele é definido em `create-tasks`
  (`SKILL.md:32` "critérios de aceitação explícitos") mas **não é consumido como gate** por
  `execute-task` (validação) nem por `review` (leitura condicional, `review/SKILL.md:21`).
  É a maior zona cinzenta: todos "sabem" do critério, ninguém o **verifica** programaticamente.
- **`review` ↔ `tasks`**: como `review` só lê a task condicionalmente, o handoff
  execução→revisão pode perder o vínculo com o critério de aceite.
- **Severidades divergentes**: `review` usa `critical/high/medium/low` (`review/SKILL.md:51`),
  enquanto `bug-schema.json` exige `critical/major/minor` (`bug-schema.json:21`). O mapeamento
  `high→major`/`low→minor` está implícito; não há tabela canônica → risco de perda na conversão.
- **Custo de timeout em lote**: subagent não é morto no timeout (`execute-all-tasks/SKILL.md:72`),
  gerando consumo de tokens em background em PRDs grandes.

**Oportunidades de economia/determinismo**
- Um único "gate de aceite" reaproveitável por `execute-task`, `execute-all-tasks` e `review`
  reduz retrabalho e fecha o falso positivo de uma vez.
- Tabela canônica de mapeamento de severidade `review`↔`bug-schema` elimina ambiguidade.

---

# Principais Fontes de Falso Positivo

1. **Status otimista por evidência textual**: `Testes: pass`/`Lint: pass` são strings livres
   aceitas pelo validador (`validate-task-evidence.sh:61-62`); não provam execução.
2. **Critério de aceite / DoD não validado**: ausente em SKILL, validador e template de
   `execute-task` (grep negativo; `task-execution-report-template.md:1-44`).
3. **Validação incompleta opt-in**: F35 (DiffSHA vs git) só roda com `AI_VALIDATE_GIT_HISTORY=1`
   (`post-execute-task.sh:179`); rastreabilidade RF do bugfix só com `--rf`
   (`validate-bugfix-evidence.sh:99-105`).
4. **Review permissiva por leitura condicional**: `review` pode aprovar sem confrontar a task
   (`review/SKILL.md:21`) — falso negativo que vira falso positivo de conclusão a montante.
5. **Sincronização frágil só de presença**: o sync gate de `create-tasks` compara conjuntos de
   nomes, não a adequação semântica da skill (`create-tasks/SKILL.md:89-95`).
6. **Drift não bloqueado em modo legado**: hooks ausentes/não chamados → "validação textual (modo
   legado)" (`execute-all-tasks/SKILL.md:22`); o gate vira best-effort.
7. **Paridade inconsistente**: enforcement nativo só no Claude; nos demais tools o mesmo gate
   depende de o agente chamar o `.sh` (`enforcement-matrix.md:19-21`).

---

# Gaps de Paridade entre Claude Code, Codex CLI e Copilot CLI

Base concreta: `agent-governance/references/enforcement-matrix.md:19-40` e a tabela "Mapeamento por
Tool" em `execute-all-tasks/SKILL.md:122-144`.

| Gap | Impacto | Skill(s) afetada(s) | Risco de divergência | Mitigação proposta |
|---|---|---|---|---|
| Hooks Pre/PostToolUse nativos só no Claude (`none` em Codex/Gemini/Copilot, `enforcement-matrix.md:19-21`) | Alto | `execute-task`, `execute-all-tasks` | Mesma tarefa "validada" no Claude e "best-effort" no Copilot/Codex | Invocação explícita do `.sh` como passo obrigatório do prompt do subagent em todos os tools; falhar se o `.sh` não rodar. |
| Subagentes: Codex/Gemini só têm `task-executor`; Claude/Copilot têm 8 agent files (`.codex/agents/`, `.gemini/agents/` vs `.claude/agents/`, `.github/agents/`) | Médio | orquestração e skills de planejamento | Planejamento/revisão sem isolamento em Codex/Gemini | Gerar os agent files faltantes ou documentar que rodam no main session. |
| Formatos de agent file de Codex/Gemini "inferidos, não validados empiricamente" (`execute-all-tasks/SKILL.md:144`) | Alto | `execute-all-tasks` | Spawn falha silenciosa → degrada para inline, contexto acumula | Validação empírica (`codex agent list`, `gemini agents list`) antes do 1º uso; registrar no report. |
| Soft timeout sem kill (`execute-all-tasks/SKILL.md:72`) | Médio | `execute-all-tasks` | Custo de tokens divergente por tool | Coordenação de kill no nível do tool quando suportado; orçamento por tool. |
| Path rígido `.claude/scripts/` no validador do `bugfix` (`bugfix/SKILL.md:51`) | Médio | `bugfix` | Em repo só-`.agents/`, validação some | Resolver em cascata como `execute-task` Etapa 5.2. |
| `default_tool`/AI_TOOL e config em cascata existem, mas enforcement depende de instrução fora do Claude (`enforcement-matrix.md:36-38`) | Médio | todas | "Equal-by-instruction" ≠ "equal-by-enforcement" | Tornar os validadores shell o ponto comum de enforcement, chamado igualmente por todos os tools. |

**Nota honesta**: a paridade hoje é "igualitária por instrução", não "por enforcement". Os
validadores `.sh` são tool-agnósticos e portáteis (bash 3.x, `pre-execute-all-tasks.sh:11`) — a
chave é exigir a invocação deles igualmente, já que só o Claude os dispara nativamente.

---

# Plano de Melhorias Priorizado

| Prioridade | Mudança proposta | Skills afetadas | Benefício | Risco mitigado | Impacto eficiência/economia | Esforço | Dependências | Evidência usada |
|---|---|---|---|---|---|---|---|---|
| **P0** | Seção obrigatória "Critérios de Aceite" no template + gate no `validate-task-evidence.sh` ligando cada critério da task a evidência | `execute-task`, `execute-all-tasks` | Elimina o principal falso positivo de `done` | Status otimista sem aceite | Neutro (gate barato) | Médio | — | `validate-task-evidence.sh:42-103`, `task-execution-report-template.md:1-44` |
| **P0** | Protocolo de múltipla escolha (2–5 opções, "(Recomendado)", 1 por vez) antes de decisões de escopo/arquitetura | `create-prd`, `create-technical-specification` | Atende requisito explícito; reduz suposição | Drift de produto/arquitetura | Economiza retrabalho downstream | Baixo | — | grep negativo; `create-prd/SKILL.md:27-35`, `create-technical-specification/SKILL.md:23-30` |
| **P0** | Exigir invocação dos hooks `.sh` em todos os tools (sem "modo legado" silencioso quando o tool é capaz) | `execute-all-tasks`, `execute-task` | Paridade de enforcement real | Drift não bloqueado fora do Claude | Custo marginal de 1 chamada | Médio | — | `execute-all-tasks/SKILL.md:22`, `enforcement-matrix.md:19-21` |
| **P1** | `review` lê a task ativa e confronta cada critério de aceite no veredito, sempre | `review` | Fecha falso negativo de revisão | Aprovar diff que não cumpre a task | Custo baixo (1 leitura) | Baixo | Gate de aceite (P0) | `review/SKILL.md:21` |
| **P1** | Tabela canônica de severidade `review`↔`bug-schema` (`high→major`, `low→minor`) | `review`, `bugfix` | Handoff sem perda semântica | Conversão implícita ambígua | Neutro | Baixo | — | `review/SKILL.md:51`, `bug-schema.json:21` |
| **P1** | Validar formatos de subagente Codex/Gemini empiricamente; gerar agent files faltantes | `execute-all-tasks` | Isolamento real cross-tool | Spawn silencioso → contexto acumula | Economia de contexto em lote | Médio | — | `execute-all-tasks/SKILL.md:144`; `.codex/agents/`, `.gemini/agents/` |
| **P1** | Resolver path do validador do `bugfix` em cascata `.claude`→`.agents`→`scripts` | `bugfix` | Portabilidade real | Validação some em repo só-`.agents/` | Neutro | Baixo | — | `bugfix/SKILL.md:51` vs `execute-task/SKILL.md:80` |
| **P2** | Tornar F35 (DiffSHA) e rastreabilidade RF default-on | `execute-task`, `execute-all-tasks`, `bugfix` | Prova de que o diff existe no git | Revert/rewrite mascarado | Custo baixo | Baixo | — | `post-execute-task.sh:179`, `validate-bugfix-evidence.sh:99-105` |
| **P2** | Derivar lista de skills "auto-carregadas" de metadado em vez de prosa | `create-tasks` | Elimina drift de manutenção | Lista desatualizada | Neutro | Médio | — | `create-tasks/SKILL.md:48-50` |
| **P2** | Validador de evidência para `review` no modo `--auto-review` | `review` | Simetria de garantia | `review.md` sem validação estrutural | Neutro | Baixo | — | ausência de `validate-review-evidence.sh` |

Legenda: **P0** obrigatório antes de "production-ready"; **P1** robustez sustentável; **P2**
relevante, não bloqueante.

---

# Sequência Recomendada de Adoção

1. **Endurecimento anti-falso-positivo (P0)**: gate de aceite/DoD no `execute-task` +
   template/validador; tornar invocação de hooks obrigatória.
2. **Sinergia e contratos entre skills (P1)**: `review` confronta task; tabela de severidade
   `review`↔`bug-schema`; path do validador do `bugfix` em cascata.
3. **Paridade cross-tool (P0/P1)**: validar empiricamente Codex/Gemini; uniformizar enforcement
   via os `.sh` chamados por todos os tools.
4. **Economia/eficiência fina (P2)**: F35/RF default-on; metadado de skills; budget por tool.
5. **Validação final de portabilidade**: rodar a cadeia completa em repo pequeno (só `.agents/`),
   médio e grande, confirmando convergência idêntica nos 3 CLIs.

---

# Decisões em Aberto

As decisões abaixo são prescritivas e dependem de preferência do usuário; **consolidadas aqui sem
pausar**, conforme combinado. Recomendação marcada com **(Recomendado)**.

**1. Rigor do gate de falso positivo em `execute-task`/`execute-all-tasks`:**
1. Gate estrito: evidência física + DoD + cada critério de aceite + status sincronizado + DiffSHA
   default-on **(Recomendado)**
2. Gate por evidência + critério de aceite (sem DiffSHA default-on)
3. Gate mínimo por testes e report (status quo)

**2. Escopo do protocolo de múltipla escolha:**
1. Em `create-prd` e `create-technical-specification`, sempre que houver ambiguidade material
   **(Recomendado)**
2. Também em `create-tasks` (decisões de fatiamento) e `review` (severidade de borda)
3. Apenas quando o usuário pedir explicitamente

**3. Comportamento de hook ausente fora do Claude:**
1. Bloquear (`failed`) se o tool é capaz de rodar o `.sh` e ele não rodou **(Recomendado)**
2. Manter "modo legado" textual com warning visível
3. Manter status quo (degrada silenciosamente)

**4. Tratamento do timeout de subagent em lote:**
1. Orçamento de tokens por tool + kill no nível do tool quando suportado **(Recomendado)**
2. Apenas registrar e seguir (status quo)
3. Reduzir o default de `AI_TASK_TIMEOUT_SECONDS`

**5. Onde versionar este relatório e o backlog:**
1. Manter em `docs/audits/` e abrir PRD `prd-skills-production-proof` para os P0 **(Recomendado)**
2. Só o relatório, sem PRD derivado
3. Converter cada P0/P1 em issue separada

---

# Evidências e Citações

| # | Achado | Arquivo:linha | Relação |
|---|---|---|---|
| E1 | `Testes: pass`/`Lint: pass` são string livre | `.claude/scripts/validate-task-evidence.sh:61-62` | Falso positivo de execução |
| E2 | Validador não checa critério de aceite/DoD | `.claude/scripts/validate-task-evidence.sh:42-103` (grep negativo) | Gap de aceite |
| E3 | Template sem seção de critérios de aceite | `.agents/skills/execute-task/assets/task-execution-report-template.md:1-44` | Gap de aceite |
| E4 | Nenhuma skill menciona múltipla escolha | grep negativo nas 7 `SKILL.md` | Requisito não atendido |
| E5 | PRD pede esclarecimento em texto livre | `.agents/skills/create-prd/SKILL.md:27-35` | Falta múltipla escolha |
| E6 | TechSpec idem | `.agents/skills/create-technical-specification/SKILL.md:23-30` | Falta múltipla escolha |
| E7 | Drift gate do PRD é best-effort (auto-declarado) | `.agents/skills/create-prd/SKILL.md:24` | Enforcement frágil mas honesto |
| E8 | Confronto de codebase no PRD é opcional | `.agents/skills/create-prd/SKILL.md:39` | Confronto fraco |
| E9 | Hooks nativos só `full` no Claude | `.agents/skills/agent-governance/references/enforcement-matrix.md:19-21,36-40` | Paridade |
| E10 | Formatos Codex/Gemini inferidos, não validados | `.agents/skills/execute-all-tasks/SKILL.md:144` | Paridade/risco de spawn |
| E11 | "Modo legado" textual se hook ausente | `.agents/skills/execute-all-tasks/SKILL.md:22` | Drift não bloqueado |
| E12 | Soft timeout não mata subagent | `.agents/skills/execute-all-tasks/SKILL.md:72` | Custo em lote |
| E13 | Sync gate skills `create-tasks`↔`execute-task` | `.agents/skills/create-tasks/SKILL.md:85-96` | Handoff forte |
| E14 | Cadeia de validação YAML + evidência física | `.claude/hooks/post-execute-task.sh:139-205` | Enforcement real (quando chamado) |
| E15 | Spec-hash injetado pela techspec | `.agents/skills/create-technical-specification/SKILL.md:54-56` | Rastreabilidade |
| E16 | `review` lê task/prd/techspec condicionalmente | `.agents/skills/review/SKILL.md:21` | Falso negativo de revisão |
| E17 | Severidades divergentes review vs schema | `.agents/skills/review/SKILL.md:51` vs `.agents/skills/agent-governance/references/bug-schema.json:21` | Perda no handoff |
| E18 | F35 (DiffSHA) é opt-in | `.claude/hooks/post-execute-task.sh:179` | Validação incompleta default |
| E19 | Path rígido `.claude/scripts` no bugfix | `.agents/skills/bugfix/SKILL.md:51` | Portabilidade |
| E20 | Rastreabilidade RF do bugfix é opt-in | `.claude/scripts/validate-bugfix-evidence.sh:99-105` | Validação incompleta default |
| E21 | Checkpoint atômico antes de mutar tasks.md | `.agents/skills/execute-task/SKILL.md:81-90` | Robustez forte |
| E22 | Codex/Gemini só têm `task-executor` | `.codex/agents/`, `.gemini/agents/` (vs `.claude/agents/`, `.github/agents/`) | Paridade de subagentes |
| E23 | Teto de tarefas anti over-split | `.agents/skills/create-tasks/SKILL.md:24` | Eficiência |
| E24 | Budget de revisão | `.agents/skills/review/SKILL.md:17-20` | Economia |
| E25 | Teste de regressão obrigatório no bugfix | `.agents/skills/bugfix/SKILL.md:41` | Robustez |

---

# Registro de Incertezas e Limites de Evidência

| Ponto | Classificação | Evidência faltante | Impacto no score |
|---|---|---|---|
| Comportamento real de spawn de subagente em Codex/Gemini/Copilot | **Inconclusivo** | Não houve execução runtime; apenas inspeção de agent files | Limita confiança da seção Paridade; score de paridade conservador |
| Eficácia prática do "modo legado" textual quando o `.sh` falta | **Inferido** | Não testado em repo sem hooks | Reforça P0 de invocação obrigatória, mas não medido |
| Se `Testes: pass` é de fato escrito sem rodar testes na prática | **Inferido** | Possibilidade estrutural comprovada; frequência real desconhecida | Sustenta o gap como vetor, não como incidência medida |
| Convergência idêntica nos 3 CLIs em projeto P/M/G | **Inconclusivo** | Auditoria foi só neste repo (orchestrator) | Portabilidade avaliada por design, não por execução cross-repo |
| `ai-spec` disponível no PATH dos repos-alvo | **Inferido** | Gate B2 existe (`execute-task/SKILL.md:23-30`); instalação no destino não verificada | Sem o binário, vários gates param com `needs_input` (bom), mas reduz automação |
| Adequação semântica das skills declaradas por task | **Inferido** | Sync gate valida presença, não pertinência (`create-tasks/SKILL.md:89-95`) | Falso positivo residual de baixa severidade |

**Itens que exigem confirmação humana antes de qualquer mudança real**: as 5 Decisões em Aberto
acima, em especial o nível de rigor do gate de aceite (Decisão 1) e o comportamento de hook ausente
(Decisão 3), por afetarem diretamente o trade-off robustez × portabilidade.

---

## Nota de ambiente

Auditoria executada no GitHub Copilot CLI com o modelo `claude-opus-4.8` (alvo do pedido).
Rigor mantido conforme o protocolo: classificação epistêmica explícita, citações `arquivo:linha`,
distinção entre best-effort, validação manual, gate programático e garantia real. Nenhuma skill foi
modificada — entrega exclusivamente analítica, conforme a restrição "Não implemente nada".
