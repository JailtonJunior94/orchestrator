# Enforcement Matrix

<!-- TL;DR
Matriz de capacidades de enforcement por ferramenta de IA (Claude, Codex, Gemini, Copilot) na realidade 2026: os 4 CLIs têm hooks nativos de bloqueio. Indica full/partial/none por capacidade e o caveat de route-around do Codex.
Keywords: enforcement, matriz, claude, gemini, copilot, codex, hook, PreToolUse, BeforeTool, agentStop, sandbox
Load complete when: tarefa exige verificar quais regras de governança são tecnicamente impostas por qual ferramenta de IA.
-->

- Rule ID: R-ENF-001
- Severidade: informativo
- Escopo: enforcement de governança em CLIs de IA.

Tabela de capacidades de enforcement por ferramenta de IA suportada.

## Atualização 2026 (importante)

Diferente da premissa histórica ("hooks só no Claude"), em 2026 **os 4 CLIs possuem hooks
nativos de bloqueio**. A paridade real é obtida com **um único conjunto de validadores shell
tool-agnósticos** (`.agents/scripts/` + `.agents/hooks/`, resolvidos em cascata) invocados pelo
config de hook nativo de cada tool:

| Tool | Hook de bloqueio | Config gerado por `ai-spec install` |
|---|---|---|
| Claude Code | `PreToolUse` -> `permissionDecision:"deny"`; `PostToolUse`; `SubagentStop` | `.claude/settings.local.json` |
| Copilot CLI | `preToolUse`/`postToolUse`/`agentStop` (`version:1`) | `.github/hooks/governance.json` |
| Gemini CLI | `BeforeTool`/`AfterTool`/`AfterAgent` (exit 2 ou `decision:"deny"`) | `.gemini/settings.json` |
| Codex CLI | `PreToolUse`/`PostToolUse` | `.codex/hooks.json` + `[hooks]` no `config.toml` |

## Legenda

- **full**: suporte nativo com enforcement real (bloqueio ou alerta automatico)
- **partial**: suporte parcial (depende de cooperacao do agente ou configuracao manual)
- **none**: sem suporte nativo para a capacidade

## Matrix

| Capacidade | Claude Code | Codex | Gemini CLI | Copilot CLI |
|---|---|---|---|---|
| Hook pre-edicao (bloqueio) | full | full[^1] | full | full |
| Hook pos-edicao (alerta) | full | full | full | full |
| Hook de fim de agente/subagente | full | partial | full | full |
| Contrato de carga base (AGENTS.md) | full | full | full | full |
| Skills como SKILL.md | full | full | full | full |
| Subagentes dedicados | full | partial[^2] | partial[^2] | full |
| Commands/slash commands | full | none | full | none |
| Carregamento lazy de referencias | full | full | full | full |
| Budget gates (CI-time) | full | full | full | full |
| Validacao de evidencia (reports) | full | full | full | full |
| Bug schema JSON validation | full | full | full | full |
| Controle de profundidade de invocacao | full | full | full | full |
| Governanca contextual gerada | full | full | full | full |
| Sandbox / approval policy | partial | full[^3] | partial | partial |

## Notas

[^1]: **Codex - caveat de route-around**: o hook `PreToolUse` do Codex tem lacuna documentada
   (um agente pode rotear comandos para fora do escopo do hook). Por isso o enforcement do Codex
   **nunca depende so do hook**: e suplementado por `sandbox_mode = "workspace-write"` e
   `approval_policy = "on-request"` no `.codex/config.toml`. Ver ADR-002.

[^2]: **Subagentes dedicados**: Claude (`.claude/agents/`) e Copilot (`.github/agents/`) tem
   formato dedicado de subagente. Codex/Gemini executam o fluxo inline (sem diretorio dedicado de
   subagente) - registrar a execucao inline no relatorio (ver tarefa de subagentes).

[^3]: **Sandbox/approval do Codex** fecha a lacuna de route-around do hook.

- **Validadores compartilhados**: os 4 tools chamam os mesmos `.agents/scripts/validate-*.sh` e
  `.agents/hooks/*.sh` via cascata `.agents/` -> tool-especifico -> `scripts/`. A consistencia dos
  mirrors e garantida pelos gates `check-skills-sync`, `check-hooks-sync` e `check-scripts-sync`.
- **Budget gates** rodam em CI (GitHub Actions) e sao agnosticos a ferramenta.
- **partial** = capacidade existe mas depende de config adicional ou cooperacao do agente; **none**
  = ausencia de mecanismo nativo.
