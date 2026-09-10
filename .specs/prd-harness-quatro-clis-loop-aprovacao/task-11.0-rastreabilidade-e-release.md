# Tarefa 11.0: Rastreabilidade, não-regressão e release major

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Fechar a entrega provando três coisas que nenhuma tarefa anterior prova sozinha.

**Primeiro**, que a cadeia **requisito → tarefa → critério → evidência** é verificável de ponta a ponta,
e que a integridade entre PRD, especificação técnica e tarefas continua ancorada por hash — os
marcadores `spec-hash-prd` e `spec-hash-techspec` no cabeçalho de `tasks.md:1-2` estão hoje zerados, e o
`spec-hash-prd` da techspec precisa continuar batendo com o `prd.md` real.

**Segundo**, que os três agentes remanescentes preservam comportamento observável em **todos** os fluxos
que não ativam as novas capacidades. Isso não é declaração: é uma suíte de não-regressão, com destaque
para os dois vetores nomeados pela techspec — a **política de ambiente zero-value**, que precisa
preservar o ambiente herdado **byte-idêntico** para Claude, Codex e Copilot, e a **função de resolução de
janela ausente**, que precisa produzir comportamento **byte-idêntico** ao atual quando não definida.

**Terceiro**, que a entrega está preparada como **major**, com seção de mudanças incompatíveis no
changelog e guia de migração do agente removido.

**ATENÇÃO — escopo de release.** A **publicação remota da release NÃO faz parte desta tarefa**. A
governança do repositório (`.claude/rules/governance.md`, R-GOV-001, "Segurança Operacional") proíbe
publicação remota sem pedido explícito. Esta tarefa **prepara os artefatos** — changelog, guia de
migração, marcação de versão — e para aí. Criar tag remota, publicar release ou disparar workflow de
publicação é violação de escopo.

<requirements>
- RF-55: a cadeia requisito → tarefa → critério → evidência é verificável de ponta a ponta, e a
  integridade entre PRD, especificação técnica e tarefas continua ancorada por hash.
- RF-61: a entrega é preparada como **major**, com seção de mudanças incompatíveis no changelog e guia
  de migração do agente removido.
- RF-62: os três agentes remanescentes preservam comportamento observável em todos os fluxos que não
  ativam as novas capacidades; toda mudança de default é declarada explicitamente no changelog.
- A política de ambiente **zero-value** preserva o ambiente herdado **byte-idêntico** — o agente cujo
  `EnvPolicy` é zero-value não tem nenhuma variável removida nem adicionada.
- A **função de resolução de janela ausente** produz comportamento **byte-idêntico** ao atual: a janela
  estática permanece o caminho dos três agentes atuais (ADR-023 estendida, não violada).
- **Nenhuma publicação remota.** A tarefa prepara artefatos; não cria tag remota, não publica release,
  não dispara workflow de publicação.
</requirements>

## Subtarefas

- [ ] 11.1 Preencher e verificar `spec-hash-prd` e `spec-hash-techspec` em `tasks.md:1-2`, hoje zerados,
      e confirmar que o `spec-hash-prd` do cabeçalho de `techspec.md:1` bate com o sha256 atual de `prd.md`.
- [ ] 11.2 Gate que reprova drift de hash entre PRD, techspec e tasks — divergência falha explicitamente,
      em vez de degradar em silêncio.
- [ ] 11.3 Mapa de rastreabilidade requisito → tarefa → critério → evidência, derivado dos artefatos e não
      mantido à mão, cobrindo os 63 RFs do PRD.
- [ ] 11.4 Gate que reprova RF sem tarefa, tarefa sem critério e critério sem linha de evidência
      verificável, fechando a cadeia nas quatro pontas.
- [ ] 11.5 Suíte de não-regressão dos três agentes remanescentes, sobre todos os fluxos que **não** ativam
      as capacidades novas.
- [ ] 11.6 Vetor nomeado 1 — teste que captura o ambiente do processo filho para agente com política de
      ambiente zero-value e afirma igualdade **byte-idêntica** com o ambiente herdado.
- [ ] 11.7 Vetor nomeado 2 — teste que exercita a spec **sem** função de resolução de janela e afirma
      saída byte-idêntica à da janela estática atual.
- [ ] 11.8 Seção de **mudanças incompatíveis** no `CHANGELOG.md`, listando a remoção do agente, a virada do
      critério estrito de aprovação e cada default alterado.
- [ ] 11.9 Guia de migração do agente removido em `docs/`, referenciado pelo erro tipado de invocação
      criado na tarefa 10.0.
- [ ] 11.10 Marcar a versão como **major** em `VERSION` (hoje `1.1.0`), **sem** criar tag remota nem
      publicar.
- [ ] 11.11 Registrar na evidência da tarefa, de forma explícita, que a publicação remota foi deixada
      pendente de pedido do usuário, com os artefatos prontos e o comando exato que o usuário executaria.

## Detalhes de Implementação

Ver techspec.md, seções:

- **"Fases"**, linha **F6 — Fechamento**: rastreabilidade ancorada por hash, release major e suíte de
  não-regressão dos agentes remanescentes; depende de F2c e F4.
- **"Sanitização de ambiente e handshake"** — a frase que fixa o vetor 1: o agente cujo `EnvPolicy` é
  zero-value tem o ambiente herdado intacto, o que mantém os três agentes atuais sem regressão.
- **"Janela derivada do modelo"** — a frase que fixa o vetor 2: quando a função de resolução está
  ausente, caso dos três agentes atuais, o comportamento é byte-idêntico ao de hoje.
- **"Conformidade com Padrões"** — a linha de segurança operacional (sem publicação remota sem pedido
  explícito) e a nota de que a ADR-023 é estendida, não violada.

Ver também `prd.md`, Bloco G (RF-61, RF-62, RF-63) e objetivo **O-06 — Zero regressão nos agentes
remanescentes**.

## Critérios de Sucesso

- `tasks.md:1-2` contém os dois hashes preenchidos, e o gate de drift falha quando qualquer um deles é
  mutado — provado por teste negativo.
- O sha256 de `prd.md` bate com o `spec-hash-prd` de `techspec.md:1`; o comando de verificação sai 0.
- O mapa de rastreabilidade cobre os **63 RFs**; RF sem tarefa associada, tarefa sem critério ou critério
  sem linha de evidência verificável deixam o gate **vermelho**, cada condição provada por caso negativo.
- Suíte de não-regressão: para Claude, Codex e Copilot, todo fluxo que não ativa capacidade nova produz
  saída idêntica à linha de base gravada antes da entrega. Qualquer diferença falha o teste.
- Vetor 1: o ambiente do processo filho de agente com `EnvPolicy` zero-value é **byte-idêntico** ao
  ambiente herdado — comparação de conjunto completo de pares chave/valor, não amostragem.
- Vetor 2: a saída da spec sem função de resolução de janela é **byte-idêntica** à da janela estática
  atual, verificada por comparação exata de bytes.
- `CHANGELOG.md` contém seção de mudanças incompatíveis nomeando a remoção do agente e **cada** default
  alterado; nenhum default alterado fica fora dela.
- `docs/` contém o guia de migração, e o erro tipado de invocação do agente removido aponta para ele — o
  caminho citado existe (`bash scripts/check-spec-paths.sh` sai 0).
- `VERSION` reflete a versão major; `git tag --list` **não** contém tag nova e nenhum workflow de
  publicação foi disparado.
- `go build ./... && go vet ./... && go test ./... -count=1` verde; `bash scripts/check-spec-paths.sh`
  sai 0 (RF-63 permanece verde após a entrega).

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

- `github-diff-changelog-publisher` — RF-61 exige changelog major com seção de mudanças incompatíveis gerado a partir do diff entre refs; a skill é dona desse fluxo e para no artefato, sem publicar a release.

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- `.specs/prd-harness-quatro-clis-loop-aprovacao/tasks.md:1-2` — marcadores de hash hoje zerados
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md:1` — `spec-hash-prd` do PRD consumido
- `.specs/prd-harness-quatro-clis-loop-aprovacao/prd.md` — os 63 RFs a rastrear, apenas leitura
- `CHANGELOG.md` — seção de mudanças incompatíveis
- `VERSION` — hoje `1.1.0`; passa a major, sem tag remota
- `docs/migracao-legacy-acp.md` — precedente de guia de migração, referência de formato
- `docs/guia-migracao-agente-removido.md` (planejado) — guia referenciado pelo erro tipado da tarefa 10.0
- `internal/runtime/specs/spec.go` — política de ambiente e função de resolução de janela; os dois vetores
- `scripts/check-spec-paths.sh` — gate de referências de caminho; precisa permanecer verde (RF-63)
- `.claude/rules/governance.md` — R-GOV-001, "Segurança Operacional": proíbe publicação remota sem pedido
  explícito; apenas leitura
- `.github/workflows/release.yml` — **não** disparado por esta tarefa; apenas leitura
