# Estratégia de Desenvolvimento de Alta Performance (SDD) — Cross-Project

> Guia agnóstico de projeto para aplicar o Software Development Design (SDD) deste `ai-spec-harness` em qualquer repositório Go, Node ou Python.
>
> Para a visão executiva específica deste repositório, consulte o [README](../README.md). Para decisão de fluxo, use o [Playbook Mestre](development-playbook.md). Para contratos de skills, use o [Guia de uso das skills](skills-usage-guide.md).

## Premissa

O `ai-spec-harness` é portátil: `ai-spec install <projeto-alvo>` replica o SDD em qualquer repositório suportado, distribuindo skills por symlink ou cópia e gerando adaptadores por ferramenta (Claude, Gemini, Codex, Copilot). O mesmo conjunto de invariantes (`spec-hash`, `execution_report.md`, regex canônicos de `tasks.md`) passa a valer no projeto alvo sem reescrita.

## Por que adotar este SDD em outros projetos

| Pilar | Benefício transferível | Mecanismo |
| :--- | :--- | :--- |
| Determinismo procedural | Elimina "requisito esquecido" e "implementação obsoleta" | `spec-hash` SHA-256 ligando PRD → TechSpec → Tasks |
| Defesa em profundidade | Código não só funciona — segue convenções do projeto | Loop mandatório `Implementação → Validação → Review → Bugfix` |
| Isolamento de contexto | Reduz alucinações e custo de tokens | Subagentes em `execute-all-tasks` carregam só governance + linguagem |
| Escalabilidade agnóstica de modelo | Mesmo fluxo em Claude, Gemini, Codex, Copilot | Adaptadores finos; lógica procedural mora nas `SKILL.md` |
| Auditoria nativa | Provar o que foi planejado, executado e validado | `execution_report.md` + `_orchestration_report.md` + `DiffSHA` |

## Matriz de adoção mínima por tipo de projeto

Comandos copy-paste por arquétipo de projeto. Ajuste `--tools` conforme as ferramentas disponíveis no time.

| Arquétipo | Comando de instalação | Skills carregadas por execute-task | Observações |
| :--- | :--- | :--- | :--- |
| **Monolito Go** | `ai-spec install <path> --source . --tools claude,codex --langs go` | `agent-governance` + `go-implementation` | Caso canônico deste repositório |
| **Monorepo Node/TS** | `ai-spec install <path> --source . --tools claude,gemini --langs node` | `agent-governance` + `node-implementation` | Carrega `node-implementation` para `.ts/.tsx/.js/.jsx/.mjs` |
| **Microserviço Python** | `ai-spec install <path> --source . --tools claude --langs python` | `agent-governance` + `python-implementation` | Skill detecta `*.py` no diff |
| **Repositório poliglota** | `ai-spec install <path> --source . --tools claude,codex,gemini --langs go,node,python` | Skill por linguagem inferida do diff | `execute-task` resolve por arquivos relevantes da task |
| **Projeto novo sem stack** | `ai-spec install <path> --source .` + rodar `analyze-project` | Detectado pela skill `analyze-project` | Gera `AGENTS.md` e governança a partir da análise |

> **Pré-requisito comum:** binário `ai-spec` no `PATH` do executor. Sem ele, os gates B2 (`execute-task` e `execute-all-tasks`) param com `needs_input` — comportamento por design, não bug.

## Fluxo canônico portátil

O fluxo é idêntico em qualquer projeto onde o `ai-spec-harness` foi instalado:

```text
analyze-project (se projeto existente)
        ↓
   create-prd ──────────────┐
        ↓                   │ spec-hash SHA-256
create-technical-spec ──────┤ liga os três
        ↓                   │ documentos
   create-tasks ────────────┘
        ↓
   execute-task  ←──── repete por task
        ↓
     review
        ↓
    bugfix (achados críticos)
        ↓
    refactor (se necessário)
```

**Regra absoluta (vale em qualquer projeto):**
- Nunca executar `execute-task` sem `tasks.md` aprovado.
- Nunca executar `create-tasks` sem TechSpec aprovada.
- Nunca executar `create-technical-specification` sem PRD aprovado.

## Custo de contexto por skill (proxy de tokens)

Tamanho real das `SKILL.md` neste repositório, convertido em tokens estimados via heurística oficial `chars/3.5`. Use estes números para dimensionar orçamento de IA ao instalar em outro projeto.

| Skill | Tamanho (chars) | Tokens estimados |
| :--- | ---: | ---: |
| `create-prd` | 5.692 | ~1.626 |
| `create-technical-specification` | 4.802 | ~1.372 |
| `create-tasks` | 11.790 | ~3.368 |
| `execute-task` | 8.707 | ~2.487 |
| `execute-all-tasks` | 14.024 | ~4.007 |
| `agent-governance` | 6.027 | ~1.722 |
| `go-implementation` | 7.609 | ~2.174 |
| `node-implementation` | 5.600 | ~1.600 |
| `python-implementation` | 5.481 | ~1.566 |

**Carga típica por execução** (governance + linguagem + skill ativa):

| Cenário | Skills carregadas | Tokens estimados |
| :--- | :--- | ---: |
| Planejamento (PRD + TechSpec + Tasks) | 3 skills de planejamento | ~6.366 |
| Execução de 1 task Go | `agent-governance` + `go-implementation` + `execute-task` | ~6.383 |
| Execução de 1 task Node | `agent-governance` + `node-implementation` + `execute-task` | ~5.809 |
| Execução de 1 task Python | `agent-governance` + `python-implementation` + `execute-task` | ~5.775 |
| `execute-all-tasks` orquestrador (sem subagent) | só orquestração | ~5.730 |
| `execute-all-tasks` por subagent | governance + linguagem + execute-task isolados | ~6.383 |

> **Insight de economia:** sem isolamento por subagent, executar 10 tasks acumularia ~63.830 tokens. Com `execute-all-tasks`, o orquestrador mantém ≤100 tokens/task após o YAML compacto — economia de ordem de magnitude em PRDs grandes.

## Escolha da ferramenta por papel

Consulte a [Matriz de Confiabilidade por Ferramenta](tool-reliability-matrix.md) para decidir qual CLI usar em cada etapa. Resumo operacional:

| Papel | Recomendação inicial | Por quê |
| :--- | :--- | :--- |
| Planejamento conversacional | Claude | Melhor rastreamento de invariantes longas |
| Execução em lote autônoma | Codex / Claude | Maturidade em loop não interativo |
| Revisão e segunda opinião | Gemini | Boa diversidade de viés do modelo |
| Sugestão inline no editor | Copilot | Latência baixa para refactors pontuais |

## Como ativar o ciclo de feedback contínuo

A telemetria é **opt-in** — sem ela, o SDD não evolui automaticamente. Em qualquer projeto que adote este harness:

```bash
export GOVERNANCE_TELEMETRY=1
# executar normalmente; eventos são acumulados em append-only
ai-spec telemetry report --top-skills 10
ai-spec telemetry report --trend 30d
ai-spec telemetry report --budget-check
```

Detalhes em [`docs/telemetry-feedback-cycle.md`](telemetry-feedback-cycle.md) e ADR [`docs/adr/006-telemetria-feedback-cycle.md`](adr/006-telemetria-feedback-cycle.md).

## Limites deliberados desta estratégia

- O SDD prioriza **correção e rastreabilidade** sobre velocidade. Para protótipos descartáveis, o overhead pode não compensar — use o [Escape Hatch documentado no README](../README.md#4-escape-hatch-quando-o-pipeline-completo-é-overhead).
- O harness só replica skills empacotadas. Skills internas específicas do projeto alvo precisam ser criadas via `ai-spec scaffold`.
- A nota SDD (90/100) refletida no [Scorecard](quality-scorecard.md) foi medida neste repositório; o projeto alvo herda o mesmo teto **apenas se** mantiver a disciplina de PRD + TechSpec + Tasks. Sem isso, a nota cai para a base de readiness do bundle local.
