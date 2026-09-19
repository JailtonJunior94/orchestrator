# Capability Matrix

Gerado a partir dos invariantes de `internal/parity` (RF-18). Nao editar a mao: rode
`UPDATE_SNAPSHOTS=1 go test ./internal/capability/...` para regenerar.

| Provider | Capability | Description | State | Reason | Test |
|---|---|---|---|---|---|
| claude | C01 | AGENTS.md e gerado com comentario de governance-schema version | supported |  | TestParitySuite/TestParity_AllTools |
| codex | C01 | AGENTS.md e gerado com comentario de governance-schema version | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | C01 | AGENTS.md e gerado com comentario de governance-schema version | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | C01 | AGENTS.md e gerado com comentario de governance-schema version | supported |  | TestParitySuite/TestParity_AllTools |
| claude | C02 | AGENTS.md referencia skill agent-governance como base canonica | supported |  | TestParitySuite/TestParity_AllTools |
| codex | C02 | AGENTS.md referencia skill agent-governance como base canonica | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | C02 | AGENTS.md referencia skill agent-governance como base canonica | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | C02 | AGENTS.md referencia skill agent-governance como base canonica | supported |  | TestParitySuite/TestParity_AllTools |
| claude | C03 | AGENTS.md contem matriz de enforcement por ferramenta | supported |  | TestParitySuite/TestParity_AllTools |
| codex | C03 | AGENTS.md contem matriz de enforcement por ferramenta | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | C03 | AGENTS.md contem matriz de enforcement por ferramenta | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | C03 | AGENTS.md contem matriz de enforcement por ferramenta | supported |  | TestParitySuite/TestParity_AllTools |
| claude | C04 | AGENTS.md referencia .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| codex | C04 | AGENTS.md referencia .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | C04 | AGENTS.md referencia .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | C04 | AGENTS.md referencia .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| codex | CD01 | .codex/config.toml e gerado com skill agent-governance | provider capability |  | TestParitySuite/TestParity_AllTools |
| codex | CD02 | .codex/config.toml referencia .agents/skills/ como caminho de skills | provider capability |  | TestParitySuite/TestParity_AllTools |
| claude | CL01 | CLAUDE.md e gerado e menciona AGENTS.md como fonte canonica | provider capability |  | TestParitySuite/TestParity_AllTools |
| claude | CL02 | CLAUDE.md referencia .agents/skills/ como fonte de verdade | provider capability |  | TestParitySuite/TestParity_AllTools |
| claude | CL03 | .claude/hooks/validate-governance.sh deve existir | unknown | invariant satisfied by generator-injected stub ".claude/hooks/validate-governance.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| claude | CL04 | .claude/hooks/validate-preload.sh deve existir | unknown | invariant satisfied by generator-injected stub ".claude/hooks/validate-preload.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| claude | CL05 | .claude/rules/governance.md e .claude/rules/code-style.md devem existir | unknown | invariant satisfied by generator-injected stub ".claude/rules/governance.md, .claude/rules/code-style.md"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| claude | CL06 | .claude/scripts/validate-task-evidence.sh deve existir | unknown | invariant satisfied by generator-injected stub ".claude/scripts/validate-task-evidence.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| claude | CL07 | .claude/scripts/validate-bugfix-evidence.sh deve existir | unknown | invariant satisfied by generator-injected stub ".claude/scripts/validate-bugfix-evidence.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| claude | CL08 | .claude/scripts/validate-refactor-evidence.sh deve existir | unknown | invariant satisfied by generator-injected stub ".claude/scripts/validate-refactor-evidence.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| copilot | CP01 | copilot-instructions.md e gerado e menciona AGENTS.md | provider capability |  | TestParitySuite/TestParity_AllTools |
| copilot | CP02 | copilot-instructions.md documenta a pre-condicao de pasta confiavel para os hooks nativos | provider capability |  | TestParitySuite/TestParity_AllTools |
| claude | FB01 | AGENTS.md referencia agent-governance — pre-requisito estrutural do fallback launcher (RF-19, ADR-017) | supported |  | TestParitySuite/TestParity_AllTools |
| codex | FB01 | AGENTS.md referencia agent-governance — pre-requisito estrutural do fallback launcher (RF-19, ADR-017) | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | FB01 | AGENTS.md referencia agent-governance — pre-requisito estrutural do fallback launcher (RF-19, ADR-017) | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | FB01 | AGENTS.md referencia agent-governance — pre-requisito estrutural do fallback launcher (RF-19, ADR-017) | supported |  | TestParitySuite/TestParity_AllTools |
| claude | INV-30 | tool_calls_normalized_name_invariant: mesma operação semântica em Claude e Codex produz normalized_name idêntico em events.jsonl | provider capability |  | TestParitySuite/TestParity_AllTools |
| codex | INV-30 | tool_calls_normalized_name_invariant: mesma operação semântica em Claude e Codex produz normalized_name idêntico em events.jsonl | provider capability |  | TestParitySuite/TestParity_AllTools |
| claude | INV-31 | mcp_nested_depth_never_exceeds_max: eventos nested_agent têm depth ≤ AISPEC_MAX_AGENT_DEPTH | supported |  | TestParitySuite/TestParity_AllTools |
| codex | INV-31 | mcp_nested_depth_never_exceeds_max: eventos nested_agent têm depth ≤ AISPEC_MAX_AGENT_DEPTH | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | INV-31 | mcp_nested_depth_never_exceeds_max: eventos nested_agent têm depth ≤ AISPEC_MAX_AGENT_DEPTH | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | INV-31 | mcp_nested_depth_never_exceeds_max: eventos nested_agent têm depth ≤ AISPEC_MAX_AGENT_DEPTH | supported |  | TestParitySuite/TestParity_AllTools |
| claude | INV-32 | cross_cli_tool_call_name_parity (RP-03): a mesma operação produz normalized_name idêntico nas CLIs com fixture | supported |  | TestParitySuite/TestParity_AllTools |
| codex | INV-32 | cross_cli_tool_call_name_parity (RP-03): a mesma operação produz normalized_name idêntico nas CLIs com fixture | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | INV-32 | cross_cli_tool_call_name_parity (RP-03): a mesma operação produz normalized_name idêntico nas CLIs com fixture | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | INV-32 | cross_cli_tool_call_name_parity (RP-03): a mesma operação produz normalized_name idêntico nas CLIs com fixture | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | OC01 | .opencode/plugin/governance.js instalado, declara o gate tool.execute.before e nunca usa handler de permissao | provider capability |  | TestParitySuite/TestParity_AllTools |
| opencode | OC02 | opencode.json tem bloco permission sem "ask" e preserva as ferramentas exigidas | provider capability |  | TestParitySuite/TestParity_AllTools |
| opencode | OC03 | opencode.json nao escreve skills.paths nem instructions (RF-13: auto-carregamento nativo) | provider capability |  | TestParitySuite/TestParity_AllTools |
| claude | X01 | Todos os artefatos de ferramenta referenciam .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| codex | X01 | Todos os artefatos de ferramenta referenciam .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| copilot | X01 | Todos os artefatos de ferramenta referenciam .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| opencode | X01 | Todos os artefatos de ferramenta referenciam .agents/skills/ como caminho canonico | supported |  | TestParitySuite/TestParity_AllTools |
| codex | X02 | Profile compact e aplicado em instalacao Codex-only (sem secoes verbose) | provider capability |  | TestParitySuite/TestParity_AllTools |
| claude | X03 | scripts/lib/check-invocation-depth.sh deve existir | unknown | invariant satisfied by generator-injected stub "scripts/lib/check-invocation-depth.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| codex | X03 | scripts/lib/check-invocation-depth.sh deve existir | unknown | invariant satisfied by generator-injected stub "scripts/lib/check-invocation-depth.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| copilot | X03 | scripts/lib/check-invocation-depth.sh deve existir | unknown | invariant satisfied by generator-injected stub "scripts/lib/check-invocation-depth.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
| opencode | X03 | scripts/lib/check-invocation-depth.sh deve existir | unknown | invariant satisfied by generator-injected stub "scripts/lib/check-invocation-depth.sh"; auto-satisfied by construction and never sustains a supported cell (V-25) |  |
