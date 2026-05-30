# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Paridade de enforcement por hooks nativos por-tool chamando validador compartilhado
- **Data:** 2026-05-30
- **Status:** Proposta
- **Decisores:** Mantenedor do harness
- **Relacionados:** `.specs/prd-skills-production-proof/prd.md` (RF-05, RF-10, RF-11, RF-12),
  `.specs/prd-skills-production-proof/techspec.md`, ADR-001 (deste PRD),
  `.agents/skills/agent-governance/references/enforcement-matrix.md`

## Contexto

A auditoria assumia que hooks de pre/post-tool eram nativos apenas no Claude Code, tornando a
paridade "igualitária por instrução", não "por enforcement". Pesquisa nas documentações oficiais
(2026) mostra que **os 4 CLIs agora têm hooks nativos de bloqueio**:

- Claude Code — `PreToolUse` → `permissionDecision: "deny"` (nativo completo).
- GitHub Copilot CLI — `preToolUse`/`agentStop` → `permissionDecision: "deny"` / `decision: "block"` (nativo completo).
- Gemini CLI — `BeforeTool` → `decision: "deny"` ou exit `2` (nativo completo, mais granular).
- OpenAI Codex CLI — `PreToolUse` → `continue: false`, porém as docs alertam que **não é uma
  fronteira completa** (route-around por caminho de tool alternativo; `unified_exec` incompleto).

Isso habilita transformar paridade em enforcement real.

## Decisão

Configurar **hooks nativos no formato de cada tool**, todos invocando os **mesmos validadores
shell tool-agnósticos** (canônico em `.agents/scripts/`, ver ADR-001) como ponto comum de
enforcement. O **Codex é sempre suplementado** com `sandbox_mode` + `approval_policy` para cobrir
a lacuna documentada — o hook nunca é a única barreira no Codex. Quando o tool ativo é capaz de
rodar o `.sh` e ele não roda/falha, o estado é `failed`/`blocked` (sem "modo legado" silencioso).

## Alternativas Consideradas

- **Só validadores invocados pelas skills** (sem hooks nativos): mantém dependência de o agente
  seguir instrução — rejeitada por não atingir enforcement real agora possível.
- **Confiar apenas nos hooks nativos do Codex**: ignora a lacuna documentada — rejeitada;
  sandbox/approval é mandatório no Codex.

## Consequências

### Benefícios Esperados

- A mesma tarefa produz o mesmo gate nos 4 CLIs, independentemente do agente cooperar.
- Manutenção única do validador; configs de hook finas por-tool.

### Trade-offs e Custos

- Quatro formatos de config de hook para manter e cobrir por gate de sync/fixtures.
- Custo marginal de 1 chamada de validador por encerramento/edição.

### Riscos e Mitigações

- **Risco:** divergência de formato entre tools. **Mitigação:** fixtures por-tool + `check-scripts-sync`.
- **Risco:** Codex contorna o hook. **Mitigação:** sandbox/approval obrigatório.
- **Rollback:** desabilitar os hooks por-tool e voltar à invocação por skill.

## Plano de Implementação

1. Criar configs: `.claude/settings.json`, `.codex/hooks.json`+`config.toml`,
   `.github/hooks/governance.json`, `.gemini/settings.json`.
2. Apontar todos para os validadores via cascata (ADR-001).
3. Reescrever `enforcement-matrix.md` para a realidade 2026 (incl. caveat Codex).
4. Ajustar `execute-task`/`execute-all-tasks` para falha bloqueante quando o tool é capaz.
5. Concluído quando os 4 tools bloqueiam um report sem critério de aceite comprovado.

## Monitoramento e Validação

- Teste de portabilidade nos 4 CLIs; entries de telemetria opt-in registrando gate disparado.

## Impacto em Documentação e Operação

- `enforcement-matrix.md`, `AGENTS.md`/`CLAUDE.md`/`GEMINI.md` (seção hooks por-tool), guias.

## Revisão Futura

- Revisitar quando o Codex fechar a lacuna de `unified_exec` ou quando os formatos de hook
  convergirem para um schema único cross-tool.
