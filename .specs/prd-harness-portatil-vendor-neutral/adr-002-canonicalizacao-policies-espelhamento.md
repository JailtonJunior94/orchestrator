# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Canonicalização de policies em `.agents/policies/` (planejado) com espelhamento por par dedicado de scripts
- **Data:** 2026-09-18
- **Status:** Proposta
- **Decisores:** dono do repositório (JailtonJunior94)
- **Relacionados:** [`.specs/prd-harness-portatil-vendor-neutral/prd.md`](prd.md) (RF-09..RF-14)

## Contexto

As regras transversais do repositório vivem hoje em `.claude/rules/`, que contém exatamente dois
arquivos: `governance.md` (R-GOV-001) e `code-style.md` (R-STYLE-001, severidade `hard`). O diretório
carrega nome de ferramenta (`.claude/`), mas o conteúdo é vendor-neutral por natureza — Codex, Copilot
e OpenCode referenciam as mesmas regras nos respectivos arquivos de governança.

A lacuna central não é estética, é de cobertura: `internal/embedded/assets/.claude/rules/` contém
**apenas** `governance.md`. `code-style.md` não é embarcado, logo nenhum projeto consumidor recebe a
política mais restritiva mantida neste repositório. Um agente operando em projeto instalado nunca vê
R-STYLE-001.

A lacuna não se fecha apenas adicionando o arquivo aos assets. O caminho de instalação é declarado por
nome, não por diretório:

- `internal/install/install.go:834-839` copia a regra com `CopyFile` de nome **hardcoded**, não
  `CopyDir` — embora o `mkdir` de `.claude/rules` já exista em `internal/install/install.go:775`.
- `internal/upgrade/upgrade.go:399-402` repete o padrão via `syncFileIfPresent`.
- `internal/uninstall/uninstall.go:594` mantém lista fixa dos arquivos removíveis.

A invariante de paridade também está ancorada em um único arquivo: CL05 em
`internal/parity/parity.go:500-511`, com o stub correspondente em `internal/parity/parity.go:201`, e o
par ID↔caminho travado por `internal/parity/parity_test.go:311`. Ou seja, mesmo que o asset e a cópia
existissem, a paridade continuaria cega para a segunda regra.

Sobre o mecanismo de espelhamento, o repositório já tem três pares — e eles **não** são genéricos nem
simétricos entre si:

| Par | Canônico | Detecção de itens | Cópia | Drift |
|-----|----------|-------------------|-------|-------|
| skills (`scripts/sync-skills.sh:18,20-25,67,72` + `scripts/check-skills-sync.sh:14,21-25,66`) | `.agents/skills` | presença de `SKILL.md`, com allowlist `non_skill_dirs=("tests")` duplicada nos dois scripts | `rsync -a --delete` | `diff -r` |
| hooks (`scripts/sync-hooks.sh:16,23-29` + `scripts/check-hooks-sync.sh:14,16-21`) | **invertido**: `.claude/hooks/` é a fonte | lista fixa | `cp` + `chmod +x` |  comparação item a item |
| scripts (`scripts/check-scripts-sync.sh:16,18-28`) | `.agents/scripts` | lista fixa `EVIDENCE_VALIDATORS` | **sem script de sync próprio** — a correção é rodar `sync-skills.sh` | comparação item a item |

Duas decisões já registradas no repositório restringem qualquer solução:

1. **Listas declaradas, nunca derivadas de glob** (`scripts/check-skills-sync.sh:108-110`): ausência do
   canônico é **drift**, nunca aprovação por vacuidade. Um gate que itera sobre glob aprova silenciosamente
   um diretório apagado.
2. **Read-only como imutabilidade é proibido** (`scripts/sync-skills.sh:10-13`): quebra o git. O espelho
   é protegido por gate, não por permissão de arquivo.

Há ainda uma armadilha operacional específica desta entrega. `TestNoOrphanValidatorTestSuites`
(`tests/integration/sync_gates_guard_test.go:370-421`) cobre **apenas** `tests/scripts/*_test.sh`, e
`TestValidatorSuitesRunInCI` (`tests/integration/sync_gates_guard_test.go:424-437`) tem lista
**hardcoded** de 3 alvos. Um novo gate de sync com alvo no `Makefile` mas ausente de
`.github/workflows/test.yml` **não seria detectado por guard nenhum** — nasceria órfão e verde.

Por fim, `.claude/rules/` é referenciado por 26 call sites, incluindo Go que quebra teste
(`internal/parity/parity.go:201,502,506`, `internal/parity/parity_test.go:311`,
`internal/install/install.go:775,819,834,836`, `internal/install/install_test.go:256,525`,
`internal/upgrade/upgrade.go:400-401`, `internal/upgrade/upgrade_test.go:847,849,863`) e documentação
por ferramenta (`AGENTS.md:106`, `CLAUDE.md:10`, `CODEX.md:119`, `COPILOT.md:97`,
`.github/copilot-instructions.md:11`, `.cursorrules:5`, além do template de techspec espelhado em três
roots). Nenhum hook shell referencia `.claude/rules/` (grep vazio): o caminho de carga é **declaração
documental por ferramenta**, não wiring executável — o que reduz materialmente o risco da mudança.

## Decisão

1. **Origem canônica única.** Criar `.agents/policies/` (planejado) como origem canônica única das
   regras transversais: `R-GOV-001` (`governance.md`) e `R-STYLE-001` (`code-style.md`). Toda edição de
   regra transversal passa a ocorrer exclusivamente nesse diretório (RF-09).

2. **`.claude/rules/` vira espelho gerado.** O diretório permanece no caminho onde as ferramentas o
   declaram, mas passa a ser **derivado** — conteúdo gerado, nunca editado à mão (RF-10). Edição direta
   no espelho é drift e falha o gate.

3. **Par dedicado de scripts.** O espelhamento é implementado por `scripts/sync-policies.sh`
   (planejado) e `scripts/check-policies-sync.sh` (planejado), um par novo e independente. Os scripts de
   skills, hooks e scripts de evidência **não** são generalizados nem alterados (RF-11). O par dedicado
   herda as duas restrições já registradas: lista de policies **declarada** nos dois scripts (nunca glob),
   e ausência do canônico tratada como drift; espelho protegido por gate, nunca por read-only.

4. **Cobertura completa do caminho de distribuição.** `code-style.md` passa a ser embarcado em
   `internal/embedded/assets/` e distribuído a consumidores (RF-12). Como o caminho é declarado por nome,
   os três pontos hardcoded são tocados explicitamente: `internal/install/install.go:834-839`,
   `internal/upgrade/upgrade.go:399-402` (`syncFileIfPresent`) e a lista de
   `internal/uninstall/uninstall.go:594`. A invariante CL05 (`internal/parity/parity.go:500-511`, stub em
   `:201`, par ID↔caminho em `internal/parity/parity_test.go:311`) é estendida para cobrir as duas regras.

5. **Fechamento explícito do gate órfão.** O novo gate é registrado **simultaneamente** no `Makefile` e
   em `.github/workflows/test.yml` na mesma entrega, e a lista hardcoded de
   `TestValidatorSuitesRunInCI` (`tests/integration/sync_gates_guard_test.go:424-437`) é atualizada para
   incluí-lo. Sem esse passo o gate nasce órfão e sem detecção. O snapshot do guard cobre
   `.agents/policies/` automaticamente, porque `syncGateMirrorDirs`
   (`tests/integration/sync_gates_guard_test.go:16-24`) já inclui `.agents` inteiro.

6. **`.agents/workflows/` (planejado) não é criado** nesta entrega (RF-14).

7. **Conflito de customização local.** Consumidor que editou `.claude/rules/governance.md` à mão passa a
   ter conflito no sync. O tratamento é a **máquina de conflito comum**: abortar o lote e instruir a mover a
   customização para a origem canônica. Sem caso especial e **sem migração automática** — promover conteúdo
   local a canônico degradaria a governança em silêncio.

## Alternativas Consideradas

### (a) Generalizar os três pares existentes num motor parametrizado

- **Descrição:** um único `sync-mirrors.sh` parametrizado por manifesto, substituindo os pares de skills,
  hooks e scripts.
- **Vantagens:** menos duplicação; a allowlist `non_skill_dirs` deixaria de existir em dois arquivos.
- **Desvantagens:** os três pares não são simétricos — hooks têm canônico **invertido**, skills detectam
  item por `SKILL.md`, scripts não têm sync próprio. Unificá-los exigiria reescrever o mecanismo que hoje
  protege skills, hooks e validadores.
- **Motivo da rejeição:** concentraria risco de regressão **no próprio mecanismo de proteção**. Uma falha
  no motor unificado derruba simultaneamente todos os gates de espelhamento — exatamente o componente que
  não pode falhar em silêncio.

### (b) Espelhamento em Go, com shells apenas chamando o binário

- **Descrição:** mover a lógica de sync/check para `internal/` e reduzir os `.sh` a wrappers de `ai-spec`.
- **Vantagens:** testes Go table-driven, tipagem, reuso de `internal/fs`.
- **Desvantagens:** cria dependência circular operacional — o gate que protege o código passaria a depender
  do código compilado. Build quebrado ou binário desatualizado tornaria o gate inoperante justamente quando
  mais é necessário.
- **Motivo da rejeição:** a dependência circular. O gate precisa rodar antes e independentemente do build.

### (c) Tratar policy como uma skill

- **Descrição:** criar `.agents/skills/policies/SKILL.md` (planejado) e reaproveitar o par de skills sem escrever nada.
- **Vantagens:** custo de implementação praticamente zero.
- **Desvantagens:** apagaria a distinção conceitual entre regra transversal (sempre ativa, declarativa) e
  skill (fluxo procedural carregado sob demanda); colidiria com o gate de bidirecionalidade de
  `skills-lock.json`, que passaria a exigir versionamento e hash de algo que não é skill externa.
- **Motivo da rejeição:** custo baixo comprado com corrupção do modelo conceitual e conflito de gate.

### (d) Criar `.agents/workflows/` junto, mesmo vazio

- **Descrição:** antecipar a estrutura de workflows universais na mesma entrega.
- **Vantagens:** evitaria uma segunda mexida na estrutura no futuro.
- **Desvantagens:** diretório sem conteúdo identificado; gate sem item para proteger; convite a
  preenchimento oportunista sem requisito.
- **Motivo da rejeição:** estrutura sem demanda concreta, contra o modo de trabalho do repositório (item 4).

## Consequências

### Benefícios Esperados

- **Fecha a lacuna de cobertura:** R-STYLE-001 passa a chegar aos projetos consumidores. Hoje a regra
  `hard` mais restritiva do repositório simplesmente não é distribuída.
- **Vendor-neutrality real:** a origem das regras deixa de morar em diretório com nome de ferramenta;
  Codex, Copilot e OpenCode passam a apontar para a mesma origem que o Claude Code.
- **Drift de policy vira erro detectável:** hoje não há gate algum sobre `.claude/rules/`; passa a haver.
- **Risco de blast radius contido:** os três pares existentes ficam intocados; uma regressão no par novo
  afeta apenas policies.
- **Custo de manutenção honesto:** a duplicação da lista declarada nos dois scripts é o preço da decisão
  G1 já registrada — ausência do canônico como drift, nunca aprovação por vacuidade.

### Trade-offs e Custos

- **Quarto par de scripts de espelhamento.** Duplicação estrutural reconhecida e aceita conscientemente,
  em troca de isolamento de risco.
- **Lista declarada em dois lugares** (`sync-policies.sh` e `check-policies-sync.sh`), como já ocorre em
  skills com `non_skill_dirs`.
- **26 call sites a revisar**, incluindo Go que quebra teste — trabalho mecânico mas obrigatório.
- **Três pontos hardcoded no caminho de instalação** (install, upgrade, uninstall) que precisam ser
  tocados em conjunto; esquecer um produz instalação parcial silenciosa.
- **Consumidor com regra customizada passa a ter conflito** onde antes tinha edição livre.

### Riscos e Mitigações

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Gate órfão: `check-policies-sync.sh` (planejado) no `Makefile` mas fora do CI | Alto — gate verde que nunca roda; nenhum guard existente detecta | Registrar no `Makefile` **e** em `.github/workflows/test.yml` na mesma entrega e atualizar a lista hardcoded de `TestValidatorSuitesRunInCI` (`tests/integration/sync_gates_guard_test.go:424-437`) |
| Instalação parcial: asset embarcado mas `CopyFile` não atualizado | Alto — consumidor recebe uma regra e não a outra, sem erro | Tocar `internal/install/install.go:834-839`, `internal/upgrade/upgrade.go:399-402` e `internal/uninstall/uninstall.go:594` no mesmo lote; cobrir por teste de instalação |
| Paridade cega: CL05 continuar validando só `governance.md` | Médio — invariante passa com cobertura incompleta | Estender `internal/parity/parity.go:500-511` e o par ID↔caminho de `internal/parity/parity_test.go:311` |
| Aprovação por vacuidade se `.agents/policies/` (planejado) sumir | Alto — gate verde sobre diretório inexistente | Lista **declarada** nos dois scripts; ausência do canônico é falha explícita (herdado de `scripts/check-skills-sync.sh:108-110`) |
| Tentação de proteger o espelho com read-only | Médio — quebra o git | Proibido no repositório (`scripts/sync-skills.sh:10-13`); proteção é por gate |
| Consumidor perder customização local no sync | Médio | Conflito comum: abortar o lote e instruir a mover para a origem canônica; sem migração automática |

**Rollback:** a mudança é reversível por reversão de commit. O espelho `.claude/rules/` mantém caminho e
conteúdo idênticos aos atuais, então reverter restaura o estado anterior sem migração de dados. O único
efeito persistente em consumidores já atualizados é a presença extra de `code-style.md`, que é inerte se
não referenciado.

## Plano de Implementação

1. Criar `.agents/policies/` (planejado) movendo `governance.md` e `code-style.md` de `.claude/rules/`,
   preservando conteúdo byte a byte.
2. Escrever `scripts/sync-policies.sh` (planejado) com lista de policies declarada, cópia determinística e
   espelho em `.claude/rules/` + `internal/embedded/assets/`.
3. Escrever `scripts/check-policies-sync.sh` (planejado) com a **mesma lista declarada**, falhando em
   drift e em ausência do canônico.
4. Registrar o alvo no `Makefile` **e** em `.github/workflows/test.yml`, e atualizar
   `TestValidatorSuitesRunInCI` (`tests/integration/sync_gates_guard_test.go:424-437`). Etapa indivisível —
   não fechar o passo 2/3 sem ela.
5. Embarcar `code-style.md` em `internal/embedded/assets/` e ajustar os três pontos hardcoded:
   `internal/install/install.go:834-839`, `internal/upgrade/upgrade.go:399-402`,
   `internal/uninstall/uninstall.go:594`.
6. Estender a invariante CL05: `internal/parity/parity.go:201,500-511` e
   `internal/parity/parity_test.go:311`.
7. Atualizar os 26 call sites de `.claude/rules/`, começando pelos Go que quebram teste
   (`internal/install/install_test.go:256,525`, `internal/upgrade/upgrade_test.go:847,849,863`) e
   seguindo para a documentação por ferramenta (`AGENTS.md:106`, `CLAUDE.md:10`, `CODEX.md:119`,
   `COPILOT.md:97`, `.github/copilot-instructions.md:11`, `.cursorrules:5`, template de techspec nos três
   roots).
8. Rodar `make test integration lint vet coverage` e o novo gate de policies.

**Dependências:** passo 4 depende de 2 e 3; passo 6 depende de 5; passo 7 depende de 1. Passos 5 e 6 podem
correr em paralelo a 2–4.

**Critérios de conclusão:**

- `.agents/policies/` (planejado) é a única origem editável das duas regras.
- `check-policies-sync.sh` (planejado) falha ao editar o espelho e ao apagar o canônico.
- O gate roda no CI e é reconhecido pelo guard de suítes.
- Instalação limpa em projeto consumidor entrega `governance.md` **e** `code-style.md`.
- CL05 cobre as duas regras.
- Nenhum call site aponta para `.claude/rules/` como origem editável.

## Monitoramento e Validação

- **Sinal primário:** execução do gate de policies em todo push/PR via `test.yml`; qualquer falha é drift
  real, não ruído.
- **Sinal de cobertura de distribuição:** teste de instalação que assere a presença das duas regras no
  destino, cobrindo install e upgrade.
- **Sinal de paridade:** CL05 verde com as duas regras declaradas.
- **Sinal de não-regressão dos pares existentes:** `make check-skills-sync check-hooks-sync
  check-scripts-sync` permanece verde — evidência de que a decisão de não generalizar foi respeitada.
- **Critérios de sucesso:** zero drift de policy detectado fora de PR que edita policy; zero relato de
  consumidor sem `code-style.md` após upgrade.
- **Critério de revisão/reversão:** se o par dedicado exigir manutenção desproporcional (por exemplo,
  divergência recorrente entre as duas listas declaradas), reabrir a alternativa (a) com escopo restrito.

## Impacto em Documentação e Operação

- `AGENTS.md` — seção de estrutura e tabela de gates por área tocada; incluir `.agents/policies/`
  (planejado) e o novo gate.
- `CLAUDE.md`, `CODEX.md`, `COPILOT.md`, `.github/copilot-instructions.md`, `.cursorrules` — atualizar o
  caminho declarado de carga das regras transversais.
- Template de techspec espelhado nos três roots (`.agents/`, `.claude/`, `internal/embedded/assets/`).
- `Makefile` — novo alvo documentado junto dos demais gates de sync.
- Onboarding: registrar explicitamente que `.claude/rules/` é **gerado** e não deve ser editado.
- Fluxo de upgrade em consumidores: documentar a máquina de conflito para regra customizada à mão.

## Revisão Futura

- **Marco de revisão:** ao surgir a **terceira** policy transversal, ou quando `.agents/workflows/`
  (planejado) for demandado por conteúdo concreto (RF-14 reabre aí).
- **Eventos que invalidam premissas:** os pares de skills/hooks/scripts convergirem naturalmente para o
  mesmo formato (reabre a alternativa (a)); o espelhamento passar a ser exercitado por wiring executável e
  não só por declaração documental (muda o perfil de risco assumido aqui).
- **Condição de substituição:** nova ADR que unifique os quatro pares sob um motor parametrizado, desde que
  acompanhada de plano de migração gate a gate com cobertura equivalente demonstrada antes do corte.
