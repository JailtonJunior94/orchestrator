# Gates de evidencia — historico e racional

Conteudo movido de `AGENTS.md` (setembro/2026). O resumo operacional continua em `AGENTS.md`
(secao "Validadores de evidencia"); este documento guarda o racional e o historico de cada gate.

## Validadores canonicos (paridade cross-CLI)

Os validadores canonicos vivem em `.agents/scripts/` (tool-neutros) e sao espelhados para
`.claude/scripts/` e `internal/embedded/assets/{.claude,.agents}/scripts/` via `scripts/sync-skills.sh`
(gate: `make check-scripts-sync`). O instalador copia-os para `.agents/scripts/` do projeto destino
**sempre** (independente dos tools), garantindo que projetos so-Codex/Copilot/OpenCode tenham os
mesmos gates que Claude. As skills resolvem em cascata `.agents/scripts/` -> `.claude/scripts/` -> `scripts/`.

- `validate-task-evidence.sh` — gate anti-falso-positivo (DoD + cada criterio de aceite + prova forte de testes).
- `validate-bugfix-evidence.sh` — rastreabilidade de origem default-on (`--no-rf` para opt-out).
- `validate-refactor-evidence.sh` — evidencia de nao-regressao.
- `validate-review-evidence.sh` — evidencia do modo `--auto-review` (veredito + severidade).

Tambem em `.agents/scripts/`: `hook-prereq-gate.sh`, `resolve-references.sh`,
`validate-governance-references.sh`, `validate-session-end.sh`, `validate-skill-prerequisites.sh`.

- `git-operation-gate.sh` — gate canonico de operacao Git (RF-40.1/RF-40.2). Delegado por
  `.agents/hooks/validate-preload.sh` antes de qualquer escape de preload, para que a decisao seja
  avaliada independentemente de `GOVERNANCE_PRELOAD_CONFIRMED`. Nega `git commit`/`git push` nao
  solicitados e remocao recursiva/forcada (`rm -rf` e variantes), exit de bloqueio `2`. Escapes
  proprios e auditados no mesmo `${GOVERNANCE_ESCAPE_LOG:-.aispec/governance-escapes.log}`:
  `GOVERNANCE_GIT_OPERATION_CONFIRMED=1` / `GOVERNANCE_GIT_OPERATION_MODE=warn` para operacao Git;
  `GOVERNANCE_DESTRUCTIVE_OPERATION_CONFIRMED=1` para comando destrutivo, sem modo `warn`. Comando
  Git de leitura (`status`, `diff`, `log`, etc.) nunca e bloqueado por este gate.

## Gate de referencias de caminho

`make check-spec-paths` falha quando um artefato de contrato (`prd.md`, `techspec.md`, `tasks.md`)
de um PRD **sob gestao SDD** cita um caminho que nao existe no repositorio.

O gate nasceu de um defeito real: a techspec de `prd-sdd-robusto` declarou por meses dois pacotes
que nunca existiram, e a divergencia sobreviveu a auditoria completa dos requisitos porque
`validate-sdd` compara hashes e vinculos RF->tarefa, nao a prosa que descreve componentes.

O escopo e deliberadamente restrito a PRDs com `sdd-state.json`: specs historicas citam caminhos ja
removidos e arquivos de projetos externos analisados na epoca, e reescreve-las seria errado. Um PRD
entra no escopo quando entra em gestao SDD.

Notacao de simbolo (`arquivo.go::Func`), pacote Go (`internal/fs.FileSystem`), comando, placeholder
e caminho opcional por decisao de arquitetura nao sao tratados como referencia de caminho.

## Selo de evidencia (RF-14)

A prova de fechamento e verificada contra a arvore de trabalho viva, que deixa de existir quando o
trabalho e commitado — por isso ela nao e re-auditavel depois. `ai-spec seal-evidence` fecha essa
lacuna gravando `commit_sha` e `commit_patch_sha256` (o patch recomputado em `base..commit` com as
mesmas exclusoes do fechamento):

```bash
ai-spec seal-evidence .specs/prd-x/result.json --prd-dir .specs/prd-x   # selar apos commitar
ai-spec seal-evidence .specs/prd-x/result.json --prd-dir .specs/prd-x --verify
```

O selo exige que o commit descenda da base registrada e recusa reselagem. A verificacao nao toca a
arvore de trabalho, entao permanece valida indefinidamente. Limite: o selo torna a evidencia
imutavel e reverificavel dali em diante, mas nao prova que o commit e byte-identico a arvore do
fechamento — essa arvore ja nao existe quando o selo e aplicado.

## Gate de criterios de aceite fail-closed

O gate de criterios de aceite e **fail-closed desde a 0.31.0**: um relatorio cuja task file nao seja
resolvivel pelo campo `Arquivo:`, ou cuja task nao declare secao de criterios, falha. A janela de
compatibilidade do NFR-01 concedia warning-only por duas versoes menores a partir de `0.29.0`, e
cobriu `0.29` e `0.30`.

`AI_SDD_STRICT_EVIDENCE=0` reabre o comportamento legado apenas para migracao. O opt-out e ruidoso
de proposito: BUG-127 mostrou que o problema nunca foi o escape existir, e sim ele ser silencioso —
um gate que se desliga sozinho e indistinguivel de um gate que aprovou.

## Regex em validadores shell

As expressoes regulares desses validadores nao podem usar classes de bracket com caracteres
multibyte (`Crit[eé]rios`): em `awk` byte-oriented (mawk, padrao nos runners Linux) elas nunca
casam e o gate se desliga silenciosamente. Usar alternacao (`Crit(e|é)rios`). O caso "a2" de
`scripts/test-validators.sh` trava essa invariante executando o gate sob `LC_ALL=C`.

## Metadado `category` no frontmatter de SKILL.md

Cada SKILL.md pode declarar `category: governance|language|processual`. `governance`/`language` sao
auto-carregadas em runtime; `processual` (ou ausente) e declarada por tarefa. `create-tasks` deriva a
lista de skills auto-carregadas desse metadado, nao de prosa hardcoded.
