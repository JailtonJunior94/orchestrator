# Tarefa 13.0: Conformidade cross-provider, nao-regressao e documentacao

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechamento do PRD. Depende das doze tarefas anteriores e não é paralelizável. Cobre RF-61, RF-62,
RF-63, RF-74 e RF-75.

Estado verificado da suíte de conformidade:

- `tests/integration/conformance_suite_test.go` **já existe e é real**. Cinco testes:
  `TestConformanceSuite_ScenariosSixAndSevenAreDeterministicPerManifest` (`:122`),
  `TestConformanceSuite_GitOperationGateBlocksAcrossFourProvidersWithSingleRunner` (`:135`),
  `TestConformanceSuite_DestructiveOperationGateBlocksAcrossFourProvidersWithSingleRunner` (`:147`),
  `TestConformanceSuite_SuiteFailsWhenCanonicalGitOperationGateIsRemoved` (`:159`) e
  `TestConformanceSuite_FourProvidersReturnSameExitCodeForSameInput` (`:184`). Quatro provedores,
  runner único e **gate-of-the-gate**: remover `.agents/scripts/git-operation-gate.sh` tem de deixar
  a suíte vermelha (`:159`).
- As fixtures atuais estão em `tests/integration/conformance_suite_test.go:192-196`:
  `unsolicited-commit` (`git commit -m x`), `unsolicited-push` (`git push origin main`),
  `destructive-removal` (`rm -rf /`), `read-only-status` (`git status`) e `confirmed-commit`
  (`git commit -m x` com `GOVERNANCE_GIT_OPERATION_CONFIRMED=1`).
- São **cinco fixtures**, das quais **quatro** mapeiam para cenários de RF-61 — comando seguro
  permitido, `git commit` não autorizado, `git push` não autorizado e comando destrutivo.
  `confirmed-commit` é um extra de caminho feliz, fora da enumeração de RF-61.
- RF-61 (`prd.md:364-368`) enumera **catorze** cenários. **Dez** faltam e são o entregável desta
  tarefa: teste obrigatório falhando; evidência ausente; evidência inválida; checkpoint válido;
  checkpoint corrompido; telemetria sem contagem de tokens disponível; evento desconhecido;
  capability não suportada; adapter retornando erro; tentativa de bypass por variação de comando.
- `internal/conformance/manifest.go:20` declara catorze cenários, porém com **nomes diferentes** dos
  da User Story: `Tarefa simples de leitura` (`:24`), `Implementação pequena` (`:31`), `Bugfix`
  (`:38`), `Tarefa que exige skill` (`:45`), `Tarefa que não deve carregar skill irrelevante`
  (`:52`), `Tentativa de commit automático` (`:59`), `Operação destrutiva sem aprovação` (`:66`),
  `Falha de testes` (`:73`), `Evidência ausente` (`:80`), `Retomada de tarefa` (`:87`),
  `SDD criado por um provider, continuado por outro` (`:94`), `Configuração inválida` (`:101`),
  `Skill adulterada` (`:108`) e `Capability não suportada` (`:115`). É **catálogo**, não execução:
  a suíte consulta o manifesto apenas para classificar determinismo (`conformance_suite_test.go:122-130`).

<requirements>
- RF-61: suíte de conformidade com os catorze cenários da User Story contra os quatro provedores.
- RF-62: fixtures reutilizadas entre provedores quando semanticamente equivalentes; o resultado
  esperado é derivado do **contrato**, NUNCA de igualdade textual entre CLIs.
- RF-63: regressão em policy crítica bloqueia o release.
- RF-74 (coração da tarefa, declarado inegociável): todos os gates e testes existentes do
  repositório continuam passando; nenhum hook classificado `KEEP` na tarefa 1.0 tem comportamento
  observável alterado sem registro explícito.
- RF-75: documentar eventos, policies, gates, limitações por provedor e por stack, e
  troubleshooting.
- R-STYLE-001 (hard): código em inglês, zero comentários. Documentação em PT-BR.
- A propriedade gate-of-the-gate existente (`conformance_suite_test.go:159`) é preservada e
  estendida aos cenários novos: cada cenário tem de conseguir ficar vermelho.
</requirements>

## Subtarefas

- [ ] 13.1 Confirmar que as doze tarefas anteriores estão `done`. Qualquer uma pendente bloqueia
      esta tarefa — o fechamento de não-regressão não pode ser antecipado nem fundido.
- [ ] 13.2 Ler `tests/integration/conformance_suite_test.go` inteiro e registrar no
      `execution_report.md` a lista dos cinco testes atuais (`:122`, `:135`, `:147`, `:159`,
      `:184`) e das cinco fixtures (`:192-196`), como linha de base antes de qualquer alteração.
- [ ] 13.3 Registrar a distinção entre catálogo e execução: `internal/conformance/manifest.go:20`
      tem catorze cenários com nomenclatura própria, que **não** coincide com a enumeração de RF-61
      em `prd.md:364-368`. Decidir e documentar se os nomes convergem ou se a suíte passa a mapear
      explicitamente `manifesto ↔ RF-61`; não deixar a divergência implícita.
- [ ] 13.4 Implementar o cenário **teste obrigatório falhando**, contra os quatro provedores.
- [ ] 13.5 Implementar o cenário **evidência ausente**.
- [ ] 13.6 Implementar o cenário **evidência inválida**.
- [ ] 13.7 Implementar o cenário **checkpoint válido**.
- [ ] 13.8 Implementar o cenário **checkpoint corrompido** — consome a detecção entregue pela
      tarefa 11.0.
- [ ] 13.9 Implementar o cenário **telemetria sem contagem de tokens disponível**, exigindo
      `unknown` explícito conforme RF-46 (tarefa 12.0).
- [ ] 13.10 Implementar o cenário **evento desconhecido** (`ErrUnknownEvent`, tarefa 4.0).
- [ ] 13.11 Implementar o cenário **capability não suportada**, exigindo que `unsupported` seja
       estado declarado e não ausência silenciosa.
- [ ] 13.12 Implementar o cenário **adapter retornando erro**.
- [ ] 13.13 Implementar o cenário **tentativa de bypass por variação de comando** — adversarial,
       cobrindo as variações que a tarefa 2.0 fechou.
- [ ] 13.14 Aplicar RF-62 a todos os cenários novos: fixture única reutilizada entre provedores
       quando semanticamente equivalente; a asserção compara o resultado **derivado do contrato**,
       jamais texto de saída entre CLIs. Gate que falha se uma asserção comparar stdout literal de
       provedores distintos.
- [ ] 13.15 Estender a propriedade gate-of-the-gate a cada cenário novo, no molde de
       `conformance_suite_test.go:159`: para cada um, existe uma mutação que **tem** de deixar a
       suíte vermelha.
- [ ] 13.16 Implementar RF-63: regressão em policy crítica bloqueia o release — o gate de
       conformidade entra na cadeia que o release consulta, e não apenas no job de teste.
- [ ] 13.17 Executar a verificação de RF-74: rodar o gate final completo e produzir o diff de
       comportamento observável de cada hook classificado `KEEP` na tarefa 1.0. Qualquer alteração
       exige registro explícito no `execution_report.md`; alteração não registrada é falha.
- [ ] 13.18 Criar `docs/hooks-canonicos.md` (novo): os cinco eventos canônicos, as cinco famílias de
       hooks, as policies, os gates, a matriz de capabilities por provedor, as limitações por
       provedor, as limitações por stack e o procedimento de troubleshooting (RF-75).
- [ ] 13.19 Atualizar `docs/evidence-gates.md` com o evidence gate e o checkpoint atômico da tarefa
       11.0.
- [ ] 13.20 Atualizar `docs/degradation-matrix.md` com os estados `unsupported` declarados por
       provedor e evento.
- [ ] 13.21 Atualizar `docs/troubleshooting.md` com os modos de falha novos: timeout de hook,
       guarda de recursão, checkpoint corrompido, evidência inválida.
- [ ] 13.22 Atualizar `docs/runtime-claude-capabilities.md` com os eventos e capabilities novos do
       provedor Claude.
- [ ] 13.23 **Corrigir afirmação falsa em `CLAUDE.md:20`.** O texto diz que, com hooks ativos,
       `validate-governance.sh` "bloqueia edicao de `AGENTS.md` e de `SKILL.md`". O script está
       registrado como hook de **pós-ferramenta** (`internal/runtime/specs/registry.go:12`,
       `scriptPostTool = ".agents/hooks/validate-governance.sh"`) e o `exit 1` de
       `.claude/hooks/validate-governance.sh:45` ocorre **depois** da edição já ter acontecido —
       `AfterTool` não bloqueia em nenhuma das quatro CLIs. Ele **informa**, não impede. Corrigir o
       texto em `CLAUDE.md` e refletir a correção em `AGENTS.md`. Exportar `GOVERNANCE_HOOK_MODE=warn`
       para editar esses dois arquivos, conforme `CLAUDE.md:20`.
- [ ] 13.24 Rodar o gate final de release completo e anexar todas as saídas ao `execution_report.md`.

## Detalhes de Implementação

- Estratégia de testes por bloco e cenário de aceite ponta a ponta: `techspec.md:308-408`.
  **Referenciar, não duplicar.**
- Rastreabilidade RF-61 a RF-63 e RF-74/RF-75: `techspec.md:409-451`.
- Fase 8 do sequenciamento: `techspec.md:519`.
- Estado parcial herdado do PRD dependente (5 de 14 cenários): `techspec.md:537-545`.
- Riscos R-07 (Codex pula hooks untrusted em silêncio) e R-08 (Copilot fail-open no timeout de
  `preToolUse`): `techspec.md:594-596`. Ambos são **limitações a documentar**, não defeitos a
  corrigir nesta tarefa.
- Semântica de resultado e de exit code que as asserções derivam:
  [ADR-002](adr-002-resultado-tipado-traducao-exit-code.md).
- Atomicidade e detecção de corrupção que os cenários de checkpoint exercitam:
  [ADR-006](adr-006-atomicidade-artefatos-operacionais.md).

## Critérios de Sucesso

- Os catorze cenários de RF-61 executam contra os quatro provedores, com os dez novos implementados
  e os quatro existentes preservados.
- Cada cenário novo tem mutação que o deixa vermelho (gate-of-the-gate estendido).
- `TestConformanceSuite_SuiteFailsWhenCanonicalGitOperationGateIsRemoved` continua verde e continua
  falhando quando `.agents/scripts/git-operation-gate.sh` é removido.
- Gate de RF-62 verde: nenhuma asserção compara saída textual entre CLIs.
- RF-63 ligado: regressão em policy crítica interrompe a cadeia de release, com prova.
- **RF-74, inegociável:** o gate final completo verde, e o diff de comportamento observável dos
  hooks `KEEP` da tarefa 1.0 anexado — vazio, ou com cada alteração registrada explicitamente.
- `docs/hooks-canonicos.md` criado e cobrindo eventos, policies, gates, limitações por provedor,
  limitações por stack e troubleshooting.
- `docs/evidence-gates.md`, `docs/degradation-matrix.md`, `docs/troubleshooting.md` e
  `docs/runtime-claude-capabilities.md` atualizados.
- `CLAUDE.md:20` corrigido e `AGENTS.md` consistente com a correção.

### Gate final de release

Todos verdes, com saída anexada:

```
make test integration lint vet coverage \
     check-skills-sync check-hooks-sync check-scripts-sync \
     check-policies-sync check-capability-matrix-sync \
     check-spec-paths check-mocks test-hooks test-validators
```

Alvos verificados no `Makefile`: `test:23`, `integration:26`, `lint:32`, `vet:36`, `coverage:42`,
`check-skills-sync:78`, `check-hooks-sync:81`, `check-scripts-sync:84`, `check-policies-sync:87`,
`check-capability-matrix-sync:90`, `test-hooks:93`, `check-spec-paths:105`, `test-validators:115`,
`check-mocks:17`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

**Conformidade**

- `tests/integration/conformance_suite_test.go:122,135,147,159,184,192-196` — suíte a estender.
- `internal/conformance/manifest.go:20,24,31,38,45,52,59,66,73,80,87,94,101,108,115` — catálogo dos
  catorze cenários.

**Documentação**

- `docs/hooks-canonicos.md` — **a criar**.
- `docs/evidence-gates.md`, `docs/degradation-matrix.md`, `docs/troubleshooting.md`,
  `docs/runtime-claude-capabilities.md` — a atualizar.
- `CLAUDE.md:20` — afirmação falsa a corrigir; `AGENTS.md` — consistência.

**Evidência da afirmação falsa**

- `internal/runtime/specs/registry.go:12` — `scriptPostTool = ".agents/hooks/validate-governance.sh"`.
- `.claude/hooks/validate-governance.sh:17-18,41,45,50` — modos `fail`/`warn`, padrão de arquivos
  interceptados e `exit 1` pós-edição.

**Gate de release**

- `Makefile:17,23,26,32,36,42,47,78,81,84,87,90,93,105,115`.
- `.github/workflows/test.yml`.

**Documentos de referência**

- `.specs/prd-hooks-canonicos-vendor-neutral/prd.md:364-372,398-404` (RF-61 a RF-63, RF-74, RF-75).
- `.specs/prd-hooks-canonicos-vendor-neutral/techspec.md:308-451,519,537-545,594-596`.
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-002-resultado-tipado-traducao-exit-code.md`.
- `.specs/prd-hooks-canonicos-vendor-neutral/adr-006-atomicidade-artefatos-operacionais.md`.
