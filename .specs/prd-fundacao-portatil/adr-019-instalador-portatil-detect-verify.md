# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Instalador portátil: auto-detecção de agentes, escopo global e verify file-first (current/missing/drifted)
- **Data:** 2026-05-22
- **Status:** Proposta
- **Decisores:** Jailton Junior (owner), arquitetura ai-spec-harness
- **Relacionados:** [PRD](prd.md) RF-06..RF-12; [techspec](techspec.md); `internal/install/install.go`; `internal/detect/detect.go`; `internal/upgrade` (checksum); `internal/runtime/probe`; referência Compozy `internal/setup/*`; ADR-001 (go:embed)

## Contexto

A instalação atual:
- **exige `--tools`** (`install.Service.validate` falha com `len(Tools)==0`; CLI marca o flag como
  obrigatório). Não há auto-detecção de quais CLIs estão instaladas.
- detecção existente (`internal/detect.FileDetector.DetectTools`) infere ferramentas por
  **arquivos do projeto** (`CLAUDE.md`, `.claude/`, `.codex/`, `.github/copilot-instructions.md`),
  **não** por binários instalados no sistema.
- já há primitivos reaproveitáveis: `internal/runtime/probe` faz `LookPath` dos binários ACP;
  `internal/upgrade` já compara checksums com estados (`StatusOK/Outdated/Missing/ContentDivergent`);
  manifesto `.ai_spec_harness.json` guarda checksums; `LinkMode` symlink/copy já existe; idempotência
  parcial via `RemoveAll` antes de copiar.
- **não há escopo global** (só `ProjectDir`); **não há comando `verify`** com estados
  current/missing/drifted; bootstrap sempre depende de `--tools`.

O Compozy (`internal/setup`) resolve com detecção automática de 40+ agentes
(`Agent.Detected/Universal`), escopo projeto/global (`InstallScope`), `InstallMode` symlink/copy e
`Verify` com estados `current/missing/drifted`. O PRD pede paridade para as 4 CLIs alvo, file-first,
macOS+Linux.

## Decisão

1. **Auto-detecção de agentes (RF-06):** novo `detect.AgentDetector` que combina (a) presença de
   **binário ACP** no PATH — reusando a lógica de `LookPath` de `internal/runtime/probe`/`specs`
   (claude-agent-acp, codex-acp, gemini, copilot) — e (b) presença de **diretórios de config**
   conhecidos (`~/.claude`, `~/.codex`, `~/.gemini`, `~/.config/...`) e arquivos de projeto (sinal
   já coberto por `FileDetector`). `--tools` torna-se **opcional**: ausente ⇒ instala nos agentes
   detectados; presente ⇒ override explícito (precedência de flag, ADR-016).
2. **Escopo global (RF-07):** `InstallOptions` ganha `Scope` (`project` default | `global`). Escopo
   global resolve destinos sob `~/.aispec/` e/ou os diretórios globais por-agente
   (`~/.claude/`, `~/.codex/`, ...), via `os.UserHomeDir` (R-SEC-001, nunca hardcoded). Projeto
   permanece o default e o caminho atual.
3. **Verify file-first (RF-10):** novo `install.Verify(opts)` que reusa o **comparador de checksum
   do `internal/upgrade`** para reportar, por skill/agente, `current | missing | drifted`. Mapeado
   para o estado existente: `StatusOK→current`, `StatusMissing→missing`,
   `StatusOutdated|ContentDivergent→drifted`. Exposto como subcomando CLI (`verify`/`inspect`).
4. **Idempotência (RF-09):** formalizar a convergência — reexecutar `Install` produz o mesmo estado
   e um `Verify` subsequente reporta **100% `current`**. Symlink/copy (RF-08) inalterados; ADR-001
   (assets via go:embed) preservado como fonte.
5. **Modos interativo/não-interativo (RF-12):** manter o fluxo não-interativo atual como base
   (instala tudo aos agentes detectados); modo interativo de seleção é **camada de UI fina** sobre a
   mesma API de install — sem TUI pesada nesta fase (evitar dependência desproporcional vs. Compozy
   que usa `huh`).
6. **Plataformas (RF-11):** macOS+Linux; Windows fora de escopo. symlink default em Unix, copy
   opcional/forçado quando symlink não suportado (comportamento atual preservado).

## Alternativas Consideradas

- **Manter `--tools` obrigatório:** falha RF-06; mantém fricção de adoção. Rejeitada.
- **Detecção só por arquivos de projeto (status quo `FileDetector`):** num codebase **vazio** não
  detecta nada — justamente o cenário-alvo de bootstrap. Por isso a detecção por **binário no PATH**
  é necessária. Rejeitada como única fonte.
- **Novo subsistema de verify do zero:** desperdiça o comparador de checksum já testado do
  `internal/upgrade`. Rejeitada — reuso.
- **TUI interativa completa (paridade `huh`):** dependência e custo desproporcionais ao PRD.
  Adiada.

## Consequências

### Benefícios Esperados
- Bootstrap zero-fricção em qualquer codebase, sem flags obrigatórias — RF-06/RF-11.
- Defaults globais e instalação multi-projeto — RF-07.
- Auditoria de drift por skill/agente — RF-10; convergência garantida — RF-09.

### Trade-offs e Custos
- Detecção por binário acopla install ao catálogo de comandos das specs; mitigado reusando `specs`.
- Escopo global amplia a superfície de escrita em FS (home); exige normalização/validação de paths.

### Riscos e Mitigações
- **Risco:** detectar agente cujo binário existe mas é incompatível → **Mitigação:** detecção
  reporta candidatos; instalação registra o que foi instalado; `verify` valida pós-condição.
- **Risco:** escrita global indevida → **Mitigação:** `global` é opt-in; paths via `os.UserHomeDir`
  normalizados e validados (R-SEC-001); `--dry-run` mostra o plano.
- **Risco:** `$HOME` ausente em CI → **Mitigação:** escopo global degrada com erro explícito; projeto
  permanece default.

## Plano de Implementação
1. `detect.AgentDetector` (binário no PATH via specs + dirs de config + arquivos de projeto).
2. Tornar `--tools` opcional no CLI; default = detectados; flag = override.
3. `InstallOptions.Scope` + resolução de destinos globais via `os.UserHomeDir`.
4. `install.Verify` reusando comparador de checksum de `internal/upgrade`; subcomando `verify`.
5. Teste de idempotência (install 2x → verify 100% current) e de bootstrap < 30s.

## Monitoramento e Validação
- Métrica de bootstrap (tempo) em teste de aceitação (< 30s em repo vazio).
- Teste: repo vazio com binários presentes ⇒ detecção instala nos agentes corretos sem `--tools`.
- Teste: install→install→verify ⇒ 100% `current`; mutação de arquivo ⇒ `drifted`; remoção ⇒ `missing`.

## Impacto em Documentação e Operação
- Guia de Instalação Universal (procedimento de bootstrap portátil) — entregável do PRD.
- Atualizar `doctor`/`inspect` docs e `AGENTS.md` (comandos de install/verify).

## Revisão Futura
- Reavaliar TUI interativa e detecção de agentes além das 4 CLIs quando entrar o PRD de breadth.
