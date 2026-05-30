# Documento de Requisitos do Produto (PRD)

<!-- spec-version: 2 -->

# Skills Production-Proof + Paridade Cross-CLI 2026

## Visão Geral

A cadeia de 7 skills de governança do harness (`create-prd` →
`create-technical-specification` → `create-tasks` →
`execute-all-tasks`/`execute-task` → `review` → `bugfix`) foi auditada em
`docs/audits/auditoria-skills-production-proof.md` e classificada como
**production-ready com gaps (76/100)** — não production-proof.

Três bloqueadores impedem o selo "production-proof":

1. **Falso positivo de conclusão**: o critério de aceite / DoD da tarefa não é
   validado por nenhum gate; a "prova" de teste é a string livre `Testes: pass`
   escrita pelo próprio agente.
2. **Protocolo de múltipla escolha inexistente** em toda a cadeia — decisões de
   escopo e arquitetura são tomadas sem oferecer opções com recomendação.
3. **Paridade de enforcement assimétrica** — a auditoria assumia hooks nativos
   apenas no Claude Code.

Pesquisa nas documentações oficiais (2026) **atualiza a premissa de paridade**:
os 4 CLIs alvo agora têm **hooks nativos de bloqueio** — Claude (`PreToolUse`),
Copilot (`preToolUse`), Gemini (`BeforeTool`) com enforcement nativo completo, e
Codex (`PreToolUse`) com lacuna documentada de route-around (a ser suplementada
por `sandbox_mode`/`approval_policy`). Isso habilita **paridade real por hooks
nativos** chamando os mesmos validadores shell tool-agnósticos.

Este PRD cobre o endurecimento da cadeia até production-proof, com foco
inegociável em **eficiência, economia, robustez** e **funcionamento igualitário
em Claude Code, Codex, Copilot e Gemini CLI**. O alvo prioritário de
portabilidade é **funcionar em OUTROS repositórios** que instalam o harness, não
apenas neste.

## Objetivos

- **Eliminar o falso positivo de `done`**: nenhuma tarefa é concluída sem que
  cada critério de aceite tenha evidência verificável e o DoD seja confrontado.
- **Decisão assistida**: introduzir protocolo de múltipla escolha (2–5 opções,
  marcação "(Recomendado)", 1 pergunta por turno) nas skills de planejamento e
  na revisão de borda.
- **Paridade real por enforcement**: a mesma tarefa produz o mesmo gate nos 4
  CLIs, via hooks nativos por-tool que invocam validadores compartilhados.
- **Portabilidade**: validadores e libs canônicos em `.agents/` resolvidos em
  cascata, funcionando em repositórios que copiam apenas `.agents/`.
- **Comportamento idêntico em QUALQUER projeto (inegociável)**: a cadeia produz
  **exatamente o mesmo comportamento** — mesmos gates, mesma paridade cross-CLI,
  mesma robustez/economia/eficiência production-proof — em projetos **pequenos,
  médios e grandes**, **novos ou existentes**, independentemente de stack,
  linguagem ou layout. Nenhum gate pode depender de detalhe específico deste
  repositório (orchestrator); tudo resolve por descoberta agnóstica e cascata.
- **Zero regressão**: zero-value de toda nova chave/flag preserva o comportamento
  atual da cadeia.

### Métricas de sucesso (todas obrigatórias)

1. `validate-task-evidence.sh` **falha** quando qualquer critério de aceite da
   task file não tem evidência correspondente no report.
2. As 4 skills alvo (`create-prd`, `create-technical-specification`,
   `create-tasks`, `review`) referenciam o protocolo de múltipla escolha em
   pontos de ambiguidade material.
3. Hooks nativos configurados e versionados para os 4 tools, todos invocando os
   mesmos validadores; matriz de enforcement atualizada para a realidade 2026.
4. A cadeia converge identicamente em um repositório que contém apenas `.agents/`.
5. **Comportamento idêntico verificado em matriz de projetos**: pequeno × médio ×
   grande, novo × existente — mesmos gates e mesma paridade cross-CLI em todos.
6. Suíte existente verde (`make test integration lint`); cobertura ≥ 75%.

## Histórias de Usuário

**Persona primária — Adotante externo do harness**
- Como adotante, quero que `execute-task` **bloqueie** a conclusão quando um
  critério de aceite não foi comprovado, para não receber um `done` falso.
- Como adotante, quero que a governança rode **igual** no CLI que eu uso (Claude,
  Codex, Copilot ou Gemini), para confiar no resultado independentemente do tool.
- Como adotante que copiou apenas `.agents/`, quero que os validadores ainda
  sejam encontrados, para não perder enforcement.

**Persona secundária — Mantenedor do harness**
- Como mantenedor, quero que decisões de escopo e arquitetura me sejam
  apresentadas como **opções com recomendação**, para decidir rápido e sem drift.
- Como mantenedor, quero que `review` confronte **cada critério de aceite** da
  task ativa, para não aprovar diff que não cumpre a tarefa.
- Como mantenedor, quero **um único validador compartilhado** por gate, chamado
  igualmente pelos hooks dos 4 tools, para manter paridade sem duplicação.

**Casos de borda**
- Hook ausente num tool capaz de rodá-lo → falha bloqueante, não "modo legado"
  silencioso.
- Codex roteando por caminho de tool não interceptado → sandbox/approval cobre a
  lacuna.
- Repositório só-`.agents/` → cascata resolve o validador.
- `Testes: pass` escrito sem rodar teste → gate rejeita (sem prova física).

## Funcionalidades Core

1. **Gate de aceite/DoD** — seção obrigatória de Critérios de Aceite no template
   de evidência + validação programática que casa cada critério com prova.
   Importante porque é o principal vetor de falso positivo.

2. **Protocolo de múltipla escolha** — referência canônica única, integrada às
   skills de planejamento e à revisão de borda. Importante para reduzir suposição
   e retrabalho downstream.

3. **Hooks nativos por-tool + validadores compartilhados** — configuração de
   hooks no formato de cada CLI, todos chamando os mesmos `.sh` tool-agnósticos;
   Codex suplementado por sandbox/approval. Importante para paridade real.

4. **Validadores canônicos portáteis** — `.agents/scripts/` como fonte de verdade,
   espelhada e resolvida em cascata. Importante para portabilidade em outros repos.

5. **Sinergia de contratos** — `review` confronta aceite sempre; tabela canônica
   de severidade `review`↔`bug-schema`; cascata de path no `bugfix`. Importante
   para handoffs sem perda semântica.

6. **Economia/eficiência** — DiffSHA e rastreabilidade RF default-on; lista de
   skills derivada de metadado; orçamento de tokens por tool + kill de subagente
   no timeout quando suportado. Importante para custo previsível em lote.

## Requisitos Funcionais

**Bloco A — Gate anti-falso-positivo (P0)**
- RF-01: O template de evidência (`task-execution-report-template.md`) deve conter
  uma seção obrigatória **"Critérios de Aceite"** com um item por critério e
  status verificável, além de um item de **DoD**.
- RF-02: `validate-task-evidence.sh` deve **extrair cada critério de aceite** da
  task file e **falhar** se algum não aparecer atendido/comprovado no report.
- RF-03: O validador deve **rejeitar** evidência de teste baseada apenas na string
  livre `Testes: pass`/`Lint: pass` sem prova associada (comando/saída/arquivo).
- RF-04: A validação DiffSHA vs git (F35) deve ser **default-on**
  (`AI_VALIDATE_GIT_HISTORY` default ligado), comprovando que o diff existe no git.
- RF-05: `execute-task` e `execute-all-tasks` devem referenciar o gate de aceite e
  **falhar de forma bloqueante** quando o tool é capaz de rodar o `.sh` e ele não
  rodou — sem "modo legado" textual silencioso.

**Bloco B — Protocolo de múltipla escolha (P0)**
- RF-06: Deve existir uma **referência canônica** de protocolo de múltipla escolha
  (2–5 opções, marcação "(Recomendado)", uma pergunta por turno, gatilho =
  ambiguidade material) em `agent-governance/references/`.
- RF-07: `create-prd` e `create-technical-specification` devem aplicar o protocolo
  antes de fechar escopo/arquitetura quando houver ambiguidade material.
- RF-08: `create-tasks` (decisões de fatiamento) e `review` (severidade de borda)
  devem aplicar o protocolo.

**Bloco C — Paridade cross-CLI por hooks nativos (P0)**
- RF-09: Os validadores de evidência devem ter fonte canônica **tool-agnóstica em
  `.agents/scripts/`**, espelhada para os mirrors e resolvida em **cascata**
  (`.agents/scripts/` → `.claude/scripts/` → `scripts/`) por todas as skills.
- RF-10: Devem existir configurações de **hooks nativos por-tool** que invoquem
  automaticamente os validadores compartilhados: Claude (`.claude/settings.json`),
  Codex (`.codex/hooks.json`/`config.toml`), Copilot (`.github/hooks/*.json`),
  Gemini (`.gemini/settings.json`).
- RF-11: A configuração do Codex deve **suplementar** o hook com `sandbox_mode` e
  `approval_policy` para cobrir a lacuna documentada de interceptação.
- RF-12: A `enforcement-matrix.md` deve ser **atualizada** para refletir os hooks
  nativos 2026 dos 4 tools, incluindo o caveat do Codex.
- RF-13: Deve existir **gate de sync/drift** para os novos hook configs e
  validadores canônicos (à imagem de `check-skills-sync`/`check-hooks-sync`).

**Bloco D — Sinergia e contratos (P1)**
- RF-14: `review` deve **ler a task ativa e confrontar cada critério de aceite**
  sempre, independentemente de o diff tocar arquivos citados na task.
- RF-15: Deve existir **tabela canônica de mapeamento de severidade**
  `review`↔`bug-schema` (`critical/high/medium/low` → `critical/major/minor`).
- RF-16: Os formatos de subagente de Codex/Gemini devem ser **validados
  empiricamente**; agent files faltantes gerados ou a execução inline documentada,
  com registro no report.
- RF-17: O path do validador do `bugfix` deve ser resolvido em **cascata**
  (`.agents/` → `.claude/` → `scripts/`), não rígido em `.claude/scripts/`.

**Bloco E — Economia/eficiência (P2)**
- RF-18: A rastreabilidade RF do `bugfix` deve ser **default-on** (RF/tasks
  passados ao validador por padrão).
- RF-19: A lista de skills "auto-carregadas" em `create-tasks` deve ser derivada
  de **metadado** (frontmatter), não de prosa hardcoded.
- RF-20: Deve existir **validador de evidência para `review`** no modo
  `--auto-review`, com template próprio (simetria com `execute-task`/`bugfix`).
- RF-21: `execute-all-tasks` deve aplicar **orçamento de tokens por tool** e
  **matar o subagente no timeout** quando o tool suportar kill nativo.

**Transversal**
- RF-22: Zero-value de toda nova chave/flag deve **preservar o comportamento
  atual** (sem regressão) e a cadeia deve **convergir identicamente nos 4 CLIs**.
- RF-23: A cadeia deve produzir **exatamente o mesmo comportamento** (gates,
  paridade cross-CLI, robustez/economia/eficiência) em **qualquer tipo de
  projeto** — pequeno, médio ou grande; novo ou existente; qualquer stack/layout.
  Nenhum gate, validador ou hook pode depender de caminho, nome ou artefato
  específico do repositório `orchestrator`; toda resolução é por **descoberta
  agnóstica** (`ls .agents/skills/`, frontmatter, env exportadas) e **cascata**.

## Restrições Técnicas de Alto Nível

- **Canônico em `.agents/`**: skills, libs, hooks e validadores canônicos vivem em
  `.agents/`; mirrors (`.claude/`, `.codex/`, `.gemini/`, `.github/`,
  `internal/embedded/assets/`) são **gerados** via scripts de sync com gate de drift.
- **Hooks nativos não negociáveis para paridade**: enforcement por hook nativo de
  cada tool; Codex sempre suplementado por sandbox/approval (nunca só o hook).
- **Idioma PT-BR**; Conventional Commits (tipo em inglês, corpo em português).
- **Testes** table-driven com FakeFileSystem (unit) / `t.TempDir()` (integration);
  DI via construtor; zero estado global.
- **Portabilidade primeiro**: o comportamento deve valer em repositórios que
  instalam o harness, não apenas neste (self-dogfooding). **Inegociável em
  qualquer tipo de projeto** (pequeno/médio/grande, novo/existente, qualquer
  stack): nenhum gate pode acoplar a paths ou artefatos específicos deste repo.
- **Sem regressão F1** garantida por zero-value e pela suíte existente.

## Fora de Escopo

- Reescrita das skills além das adições cirúrgicas de gate e protocolo.
- Novos runtimes/CLIs além dos 4 alvo (claude, codex, gemini, copilot).
- Mudança do modelo de persistência file-first (sem banco de dados/daemon).
- Suporte a Windows como alvo primário (hooks `powershell` apenas onde o tool já
  oferece o campo cross-platform, ex.: Copilot).
- Alteração do protocolo ACP ou do núcleo de runtime.

## Suposições e Questões em Aberto

- **Suposição**: o caminho canônico portátil dos validadores será
  `.agents/scripts/`, com `.claude/scripts/` mantido como mirror para
  compatibilidade; a forma exata do gate de sync (novo `check-scripts-sync.sh`
  vs extensão do existente) será fixada na Especificação Técnica.
- **Suposição**: "tool capaz de rodar o `.sh`" será detectado pela presença do
  hook config + binário do tool; a heurística será detalhada na TechSpec.
- **Questão aberta**: a validação empírica de subagentes Codex/Gemini (RF-16)
  depende dos binários disponíveis no ambiente de execução; sem eles, registrar a
  suposição e documentar execução inline.
- **Questão aberta**: o formato exato dos hooks por-tool (camelCase vs PascalCase
  no Copilot; JSON vs TOML inline no Codex) será consolidado na TechSpec a partir
  das docs oficiais 2026 citadas na auditoria/pesquisa.
