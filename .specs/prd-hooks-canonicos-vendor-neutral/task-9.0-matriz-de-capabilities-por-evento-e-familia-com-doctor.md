# Tarefa 9.0: Matriz de capabilities por evento e familia com doctor de hooks

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar a **segunda** matriz de capabilities — eixo `provedor × evento canônico × família de hook` —
e estender o `doctor` com uma seção de hooks e adapters.

A matriz nova COEXISTE com a existente; não a substitui e não se funde com ela. São eixos distintos:
a matriz atual cruza provedor com invariante de paridade, a nova cruza provedor com evento e
família. Unificar produziria células sem sentido (invariante de paridade não tem evento) e quebraria
o golden test existente.

Cobre RF-59 e RF-60. Depende das tarefas 3.0 e 8.0. Paralelizável com 12.0.

<requirements>
- A matriz nova é um artefato SEPARADO: `testdata/hook-capability-matrix.json` e
  `docs/hook-capability-matrix.md`. Proibido unificar com a matriz existente.
- A matriz é GERADA, nunca editada à mão, seguindo o precedente do gate atual.
- Regra de custo: células `SupportUnsupported` NÃO exigem prova de dispatch. Exigir prova de todas
  as 100 células (5 eventos × 5 famílias × 4 provedores) faria o custo explodir sem ganho.
- A prova por célula deve DISCRIMINAR célula a célula — herdar o rigor já estabelecido em
  `internal/capability/capability.go:87`, onde `buildCell` marca `StateUnknown` com razão citando
  "V-25" para invariante auto-satisfeito por stub.
- O `doctor` é ACRESCIDO de uma seção de hooks e adapters. Proibido reescrever os 15 checks
  existentes ou alterar seus nomes, camadas e status.
- RF-60 sem chamada a LLM, com exit code apropriado.
- Código em inglês, zero comentários (R-STYLE-001, hard). Documento e relatório em PT-BR.
</requirements>

## Subtarefas

- [ ] 9.1 Confirmar que a tarefa 3.0 está `done` (prova de dispatch indexada por célula em
      `internal/capability/evidence.go:140-150`, com a closure `:147-149`) e que a tarefa 8.0 está
      `done` (os 20 pares `(provedor, evento)` declarados).
- [ ] 9.2 Confirmar que `internal/runtime/specs/enforcement.go:12-22` deixou de ter apenas três
      pontos canônicos. A matriz de 5 eventos não existe antes disso.
- [ ] 9.3 Registrar a linha de base da matriz existente: 54 células, 4 provedores
      (claude 18, codex 13, opencode 12, copilot 11), 26 capabilities distintas, estados
      `supported` 32 / `provider capability` 12 / `unknown` 10.
- [ ] 9.4 Definir a enumeração das 5 famílias de hook e confirmar os 5 eventos canônicos vindos de
      `internal/hookcontract` (tarefa 4.0 / ADR-001).
- [ ] 9.5 Implementar o gerador da matriz nova, produzindo `testdata/hook-capability-matrix.json`
      com célula `(provedor, evento, família)` e estado explícito.
- [ ] 9.6 Implementar a renderização determinística de `docs/hook-capability-matrix.md` a partir do
      JSON, nunca o inverso.
- [ ] 9.7 Implementar a regra de custo: células `SupportUnsupported` são declaradas sem exigência de
      prova de dispatch; qualquer outro estado exige prova indexada pela célula.
- [ ] 9.8 Implementar `TestDispatchProof_DiscriminatesByCell`: a prova precisa falhar quando
      aplicada a uma célula que não é a provada, seguindo o precedente anti-tautológico de
      `internal/capability/capability.go:87` (razão "V-25").
- [ ] 9.9 Criar o golden test da matriz nova, independente do golden test da matriz existente
      (`internal/capability/golden_test.go:175`), que não pode ser alterado por esta tarefa.
- [ ] 9.10 Ligar o gate da matriz nova ao `Makefile` e ao job correspondente de
      `.github/workflows/test.yml`, ao lado do gate existente — pacote novo sem job é gate órfão.
- [ ] 9.11 Verificar que `make check-capability-matrix-sync` (`Makefile:90-91`, step em
      `.github/workflows/test.yml:119-120`) continua verde e cobrindo a matriz antiga sem alteração
      de conteúdo.
- [ ] 9.12 Acrescentar a `internal/doctor/doctor.go` a seção de hooks e adapters, sem reescrever os
      15 checks existentes nem mudar seus nomes, camadas (`Layer`) ou status.
- [ ] 9.13 Implementar em `doctor` a detecção de **adapter ausente** (RF-60), com exit code
      apropriado.
- [ ] 9.14 Implementar a detecção de **hook não executável** (bit de execução ausente no artefato
      instalado).
- [ ] 9.15 Implementar a detecção de **configuração inválida** (fonte nativa ilegível ou com
      estrutura incompatível com o formato declarado em `internal/runtime/specs/native_config.go`).
- [ ] 9.16 Implementar a detecção de **versão incompatível** entre o contrato e o adapter.
- [ ] 9.17 Implementar a detecção de **divergência entre contrato e adapter**: par declarado no
      contrato que o adapter não realiza, ou vice-versa.
- [ ] 9.18 Garantir que nenhuma das detecções de 9.13 a 9.17 chama LLM: são todas leitura de
      artefato e comparação determinística.
- [ ] 9.19 Criar `doctor_hooks_test.go` ao lado de `internal/doctor/multiprovider_test.go`
      (16 funções de teste hoje), cobrindo os cinco modos de falha e os exit codes.
- [ ] 9.20 Executar os gates da seção "Critérios de Sucesso" e anexar as saídas ao
      `execution_report.md`.

## Detalhes de Implementação

Referência canônica: [`techspec.md`](techspec.md), seção "Rastreabilidade — RF × decisão × arquivo ×
teste" — linha `RF-59` (matriz por evento × família, prova discriminada, `internal/capability/`,
`TestDispatchProof_DiscriminatesByCell`) e linha `RF-60` (`doctor --hooks` sobre os 15 checks
existentes, `doctor_hooks_test.go`, ao lado de `multiprovider_test.go`). O racional da prova por
célula está em [`adr-004-prova-dispatch-por-celula.md`](adr-004-prova-dispatch-por-celula.md); os
eventos canônicos vêm de
[`adr-001-hook-contract-fonte-unica-projecoes.md`](adr-001-hook-contract-fonte-unica-projecoes.md).
Não duplicar nenhum dos dois aqui.

### Matriz EXISTENTE — linha de base verificada

`internal/capability/`, `testdata/capability-matrix.json` e `docs/capability-matrix.md` sustentam
**54 células** no eixo `provedor × invariante de paridade`:

| Dimensão | Valor verificado |
|---|---|
| Provedores | 4 — claude 18, codex 13, opencode 12, copilot 11 |
| Capabilities distintas | 26 |
| Estados | `supported` 32, `provider capability` 12, `unknown` 10 |
| Gate | `make check-capability-matrix-sync` — `Makefile:90-91` |
| Step de CI | `.github/workflows/test.yml:119-120` |

### Matriz NOVA — eixo distinto, coexistência obrigatória

A matriz nova cruza `provedor × evento canônico × família de hook`: 5 eventos × 5 famílias ×
4 provedores. **COEXISTE, não unifica.** Unificar produziria células sem sentido — um invariante de
paridade não tem evento associado — e quebraria o golden test existente
(`internal/capability/golden_test.go:175`).

Artefatos: `testdata/hook-capability-matrix.json` e `docs/hook-capability-matrix.md`, ambos gerados,
nunca editados à mão.

### Dependência dura

A matriz de 5 eventos só existe depois que:

1. `internal/runtime/specs/enforcement.go:12-22` deixar de ter três pontos canônicos
   (`PointPreTool`, `PointPostTool`, `PointSessionEnd`) — **tarefa 5.0**; e
2. a prova de dispatch for indexada por célula em `internal/capability/evidence.go:140-150` —
   **tarefa 3.0**. Hoje `dispatchProvenFromTests` devolve uma closure
   `func(provider, capabilityID string) bool { return resolved }` (`:147-149`) que **ignora os dois
   parâmetros** e responde o mesmo para toda célula.

### Regra de custo

Células `SupportUnsupported` NÃO exigem prova de dispatch. Sem essa regra, o custo explode até 100
células provadas, sendo que uma parcela delas descreve exatamente a ausência de suporte — não há o
que provar executando.

### Precedente de rigor a seguir

`buildCell` (`internal/capability/capability.go:77`) marca `StateUnknown` com razão textual citando
"V-25" (`:87`) para invariante auto-satisfeito por stub injetado pelo gerador, explicitando que ele
"nunca sustenta uma célula supported". A prova da matriz nova deve ter o mesmo grau de honestidade:
uma célula só é `supported` quando a prova discrimina aquela célula.

### Doctor — estado atual verificado

`internal/doctor/doctor.go` (562 linhas) foi reescrito e JÁ tem **15 checks** estratificados por
camada (`Layer`), com blocos por provedor:

| # | Check | Linhas |
|---|---|---|
| 1 | `Repositorio git` | `:153-155` |
| 2 | `Diretorio de skills` | `:169-172` |
| 3 | `Manifesto` | `:179-183` |
| 4 | `Symlinks de skills` | `:210-217` |
| 5 | `Permissoes de escrita` | `:229-231` |
| 6 | `Git instalado` | `:236-238` |
| 7 | `Contrato do harness` | `:244-254` |
| 8 | `Integridade de skills (lock)` | `:261-281` |
| 9 | `Sincronia canonica/derivados` | `:291-318` |
| 10 | `Instrucoes descobriveis` | `:378-407` |
| 11 | `Skills disponiveis` | `:429-437` |
| 12 | `Politicas aplicadas` | `:455-463` |
| 13 | `Validadores de evidencia instalados` | `:478-485` |
| 14 | `Pre-condicoes de enforcement` | `:509-524` |
| 15 | `Trust de hooks do Codex` | `:530-560` |

Já existe `internal/doctor/multiprovider_test.go` com **16 funções de teste** (`:12` a `:257`).

Esta tarefa **ACRESCENTA** a seção de hooks e adapters — não reescreve nenhum dos 15 checks.

### RF-60 — modos de falha exigidos

`doctor` deve detectar, SEM chamar LLM e com exit code apropriado: adapter ausente; hook não
executável; configuração inválida; versão incompatível; divergência entre contrato e adapter.

## Critérios de Sucesso

- `testdata/hook-capability-matrix.json` e `docs/hook-capability-matrix.md` gerados, com célula
  `(provedor, evento, família)` e estado explícito para cada uma.
- A matriz existente permanece com 54 células, 4 provedores (claude 18, codex 13, opencode 12,
  copilot 11), 26 capabilities e estados `supported` 32 / `provider capability` 12 / `unknown` 10 —
  diferença nesses números é regressão.
- `internal/capability/golden_test.go` inalterado e verde.
- `TestDispatchProof_DiscriminatesByCell` verde, e comprovadamente vermelho quando a prova é
  aplicada a uma célula diferente da provada.
- Nenhuma célula `SupportUnsupported` exigindo prova de dispatch; nenhuma célula de outro estado sem
  prova indexada.
- Os 15 checks do `doctor` mantêm nome, camada e semântica de status; a seção nova é aditiva.
- Os cinco modos de falha de RF-60 detectados com exit code apropriado, sem nenhuma chamada a LLM no
  caminho de execução.
- **Não-regressão (inegociável):** `make test lint vet` verdes, sem exceção e sem teste marcado como
  skip para passar.
- `make check-capability-matrix-sync` verde (`Makefile:90-91`), e o gate da matriz nova ligado ao
  `Makefile` e a um job de `.github/workflows/test.yml`.
- `make integration` verde.
- `make coverage` respeita os gates de 75% total e 70% por pacote crítico.

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

- `internal/capability/capability.go` — `buildCell` (`:77`), razão "V-25" (`:87`)
- `internal/capability/evidence.go` — `dispatchProvenFromTests` (`:140-150`), closure que ignora os
  parâmetros de célula (`:147-149`)
- `internal/capability/golden_test.go` — golden da matriz existente (`:175`), não alterar
- `testdata/capability-matrix.json`, `docs/capability-matrix.md` — matriz existente, 54 células
- `testdata/hook-capability-matrix.json`, `docs/hook-capability-matrix.md` — artefatos novos
- `internal/runtime/specs/enforcement.go` — `CanonicalPoint` e `canonicalPoints` (`:12-22`)
- `internal/doctor/doctor.go` — 15 checks (`:153-560`), seção de hooks e adapters a acrescentar
- `internal/doctor/multiprovider_test.go` — 16 funções de teste (`:12-257`)
- `internal/doctor/doctor_hooks_test.go` — novo, ao lado do anterior
- `Makefile` — `check-capability-matrix-sync` (`:90-91`) e gate novo
- `.github/workflows/test.yml` — step do gate existente (`:119-120`) e job do gate novo
- [`adr-004-prova-dispatch-por-celula.md`](adr-004-prova-dispatch-por-celula.md) — prova por célula
- [`adr-001-hook-contract-fonte-unica-projecoes.md`](adr-001-hook-contract-fonte-unica-projecoes.md)
  — eventos canônicos
- [`techspec.md`](techspec.md) — rastreabilidade RF-59 e RF-60
