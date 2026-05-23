# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Instalação transparente stack-aware — seleção de skill por linguagem, probe ACP, stubs por CLI e verify ampliado
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Time ai-spec-harness (owner: JailtonJunior94)
- **Relacionados:** PRD (RI-01, RI-02, RI-03, RI-04); techspec; ADR-019 (instalador portátil); `internal/install/install.go`; `internal/detect/detect.go` (`DetectLangs`/`DetectTools`); `internal/detect/architecture.go`; `internal/runtime/probe/probe.go` (`EnsureAvailable`); `internal/runtime/specs`

## Contexto

`ai-spec install .` já auto-detecta agentes (ADR-019), é idempotente e faz bootstrap < 30s. Quatro lacunas para transparência total (P/M/G):

- **RI-01:** `detect.DetectLangs` já identifica Go/Node/Python, mas `Service.Execute` recebe `opts.Langs` e não deriva a skill de linguagem da detecção automaticamente — install não fica stack-aware sem flag. *Diferencial vs Compozy (agent-centric, não detecta linguagem).*
- **RI-02:** o probe de binário ACP (`probe.EnsureAvailable`) só roda na execução da task. Usuário pode "instalar com sucesso" e falhar depois (falha tardia).
- **RI-03:** install gera `.agents/config.yaml` unificado mas a geração de stubs por CLi (`.claude/`, `.codex/config.toml`, `.gemini/commands/*`, `.github/copilot-instructions.md`) é parcial/condicional — setup nem sempre funcional sem ajuste manual.
- **RI-04:** `verify` reporta current/missing/drifted por skill, mas não por binário ACP — o loop de transparência não fecha.

## Decisão

1. **RI-01 — `install` stack-aware por default.** Quando `opts.Langs` vazio, derivar de `detect.DetectLangs(projectDir)` (mesmo padrão do auto-detect de tools em `Execute`). Mapear `LangGo→go-implementation`, `LangNode→node-implementation`, `LangPython→python-implementation` via `skills.AllSkills(langs)` (já existente). Sem flag.
2. **RI-02 — probe não-fatal no install.** Após instalar adaptadores, para cada CLI detectada, chamar `probe.EnsureAvailable` (binário direto ou fallback npx) e reportar disponível/ausente como **warning não-fatal** (`printer.Warn`), eliminando a falha tardia. Install não aborta por binário ausente (assets ficam instalados).
3. **RI-03 — stubs por CLI determinísticos.** Garantir que cada `installClaude/installCodex/installGemini/installCopilot` gere o stub mínimo funcional da sua CLI conforme detecção, idempotente (reexecução converge).
4. **RI-04 — `verify` cobre skill **e** binário ACP.** `Verify` ganha itens de tipo `binary` (por CLI: current/missing) além dos itens de skill, reusando o probe. Saída unificada `[]VerifyItem`.

Reuso máximo: `detect`, `probe` e `skills.AllSkills` já existem — esta decisão é wiring + cobertura, não nova infra.

## Alternativas Consideradas

- **Exigir `--langs` explícito (status quo).** Vantagem: zero ambiguidade. Desvantagem: fere transparência (usuário precisa conhecer a stack/CLI). Rejeitada — RI-01 pede stack-aware sem flag.
- **Probe fatal no install (abortar se binário ausente).** Vantagem: garante ambiente pronto. Desvantagem: impede instalar assets offline/CI sem CLIs; quebra idempotência. Rejeitada — warning não-fatal é o equilíbrio.
- **Gerar stubs de todas as CLIs sempre.** Polui repositórios. Rejeitada — gerar só para CLIs detectadas/solicitadas.

## Consequências

### Benefícios Esperados

- Setup funcional sem ajuste manual em P (repo vazio), M (Go/Node/Python) e G (monorepo) — fecha critério de aceitação de transparência.
- Falha de binário ACP some no momento certo (install), não na primeira task.
- `verify` vira fonte única do estado real (skills + binários).

### Trade-offs e Custos

- Probe no install adiciona latência (lookup PATH + possível `npx`); manter < 30s (RF-11) — probe é lookup leve, sem download forçado.
- Mais caminhos no `install`/`verify` para testar (matriz P/M/G).

### Riscos e Mitigações

- **Risco:** detecção de linguagem incompleta em monorepo poliglota. **Mitigação:** `DetectLangs` retorna múltiplas; `AllSkills` instala todas as skills de linguagem detectadas.
- **Risco:** probe lento em ambiente sem rede (npx). **Mitigação:** probe usa timeout curto e degrada para warning; nunca bloqueia.
- **Rollback:** flags explícitas (`--tools`, `--langs`) sobrescrevem a auto-detecção.

## Plano de Implementação

1. Derivar `opts.Langs` de `DetectLangs` quando vazio (em `Execute`).
2. Probe não-fatal por CLI pós-instalação de adaptadores; warnings.
3. Auditar/garantir stubs por CLI idempotentes.
4. Estender `Verify`/`VerifyItem` com itens `binary` por CLI.
5. Testes de integração P/M/G (FakeFileSystem unit + `t.TempDir()` integration) convergindo a 100% `current`.

## Monitoramento e Validação

- Gate: `make test` + `make integration`; cenários P/M/G idempotentes.
- Sucesso: reexecução de `install` → `verify` 100% `current`; binário ausente aparece como warning no install e `missing` no verify.

## Impacto em Documentação e Operação

- Atualizar `docs/guia-instalacao-universal.md` e AGENTS.md (seção Instalação) com stack-aware + probe + verify de binário.

## Revisão Futura

- Revisitar ao adicionar nova linguagem/CLI ou se o probe impactar o orçamento de 30s do bootstrap.
