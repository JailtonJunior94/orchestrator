# Tarefa 2.0: Pacote de domínio do Ciclo de Aprovação, sem consumidor

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar o pacote `internal/approval` (planejado) com os quinze arquivos que a techspec enumera, **sem
nenhum consumidor**. Esta tarefa entrega domínio puro: nada em `internal/taskloop` ou
`internal/runtime` passa a chamá-lo aqui — essa migração é a tarefa 4.0.

O ponto central da entrega é uma única invariante estrutural: `ProvaDeAprovacao` tem construtor único
que exige **simultaneamente** veredito aprovado e mapa 1:1 completo, e o estado `EstadoAprovado` só é
alcançável mediante esse valor. O falso positivo que hoje existe no repositório deixa de ser um defeito
a caçar e passa a ser um estado que **não compila**. `APPROVED_WITH_REMARKS` não aprova (RF-33); mapa
com critério sem evidência ou declarado "não verificável" não aprova (RF-49); zero-value de
`ProvaDeAprovacao` nunca é válido.

Dois corolários do mesmo princípio, ambos fail-closed:

- A **tabela de transições** é a autoridade sobre movimento de estado, e ausência na tabela é
  **proibição** — não permissão por omissão. `EstadoAprovado` e `EstadoBloqueado` são terminais.
- O **tradutor** do texto do revisor para `Veredito` é camada anticorrupção: texto sem declaração
  canônica resulta em `BLOCKED`. É proibido inferir aprovação pela ausência de marcadores negativos,
  que é exatamente o comportamento atual do repositório (RF-46).

A `Fingerprint` é SHA-256 sobre o conjunto **ordenado e deduplicado** de `{severidade, arquivo, regra}`,
ignorando deliberadamente número de linha e identificador atribuído pelo agente — nenhum dos dois é
estável entre rodadas. Duas fingerprints não-calculadas **nunca** são iguais, para que ausência de
achados não seja lida como não-convergência.

<requirements>
- RF-30: agregado em pacote de domínio próprio, fonte de verdade da contagem de rodadas, do veredito
  corrente e do critério de parada. **Zero import** de ACP, CLI ou filesystem.
- RF-33: encerra como aprovado exclusivamente com veredito `APPROVED` **e** mapa 1:1 completo;
  `APPROVED_WITH_REMARKS` não encerra.
- RF-37: `PoliticaDeAprovacao.Decidir` aplica as checagens baratas (veredito bloqueado, fingerprint
  repetida, teto) **antes** da correção cara; fingerprint calculada sobre `{severidade, arquivo, regra}`.
- RF-41: motivo canônico de encerramento em conjunto fechado de cinco — aprovado, limite de rodadas,
  não convergiu, sem mudança, entrada bloqueada.
- RF-45: estados terminais são **resultados**, não erros de infraestrutura; não acionam retry. Erros
  reais usam sentinelas comparáveis por `errors.Is` e tipos por `errors.As`.
- RF-46: tradução fail-closed; sem declaração canônica → `BLOCKED`.
- RF-48: `LinhaDeEvidencia` só é válida em três formas — comando com saída registrada, `arquivo:linha`
  presente no diff revisado, ou nome de teste com resultado registrado. Qualquer outra forma é inválida.
- RF-49: critério "não verificável pelo diff" proíbe `APPROVED` e vira achado de severidade alta.
- RF-50: critério não atendido é achado de severidade **mínima alta**, implicando `REJECTED`.
- Imports permitidos, exaustivos: `context`, `crypto/sha256`, `encoding/hex`, `errors`, `fmt`, `iter`,
  `slices`, `cmp`, `strings`. Nada além disso.
- Regras Go [HARD] R0–R7 de `go-implementation` aplicam-se integralmente: sem `func init()`; toda função
  é método de struct com as exceções exaustivas (construtores `New*`, métodos de interface); enums com
  `iota+1` e zero-value reservado como inválido; sem `panic` em produção; capacidade de slices declarada;
  ordenação canônica dentro do arquivo; globais não exportados com prefixo `_`. Sem `interface{}`.
- As três portas são declaradas **no consumidor** (dentro do domínio) e trafegam valores opacos.
  `Revisor` devolve **texto bruto**, nunca `Veredito` — se o adaptador pudesse devolver `Veredito`,
  reintroduziria o defeito de `internal/runtime/runner_autoreview.go:233`.
- `Corretor` devolve apenas `error`: "houve mudança" é responsabilidade do `Repositorio`, porque o
  corretor não é fonte confiável sobre si mesmo.
</requirements>

## Subtarefas

- [ ] 2.1 Criar `internal/approval/identidades.go` (planejado): `IdentidadeDaTarefa`,
      `IdentidadeDoAgente` — dado opaco vindo do Catálogo, sem semântica de CLI.
- [ ] 2.2 Criar `internal/approval/veredito.go` (planejado): `Veredito` como conjunto fechado de 4 com
      `iota+1`, construtor validante e zero-value inválido. `Aprova()` verdadeiro **apenas** para
      `APPROVED`.
- [ ] 2.3 Criar `internal/approval/motivo.go` (planejado): `MotivoDeParada`, conjunto fechado de 5.
- [ ] 2.4 Criar `internal/approval/achado.go` (planejado): `Severidade` (`iota+1`), `Achado`, e a
      tradução determinística 4→3 níveis para o schema de bugs vigente.
- [ ] 2.5 Criar `internal/approval/evidencia.go` (planejado): `LinhaDeEvidencia` com as três formas
      válidas de RF-48, `CriterioDeAceite`, `MapaDeCriterios` com `Completo()`, `Associados()` e
      `Total()`.
- [ ] 2.6 Criar `internal/approval/prova.go` (planejado): `ProvaDeAprovacao` e `NewProvaDeAprovacao`
      exigindo as duas condições simultaneamente, conforme o bloco de código da techspec.
- [ ] 2.7 Criar `internal/approval/fingerprint.go` (planejado): `Fingerprint` e calculadora SHA-256
      sobre conjunto ordenado e deduplicado; duas não-calculadas nunca iguais.
- [ ] 2.8 Criar `internal/approval/politica.go` (planejado): `PoliticaDeAprovacao` imutável com
      Functional Options e `Decidir` na ordem das checagens baratas antes da cara. Teto `< 1` recusado
      na construção.
- [ ] 2.9 Criar `internal/approval/estado.go` (planejado): `Estado` (`iota+1`) e
      `_transicoesPermitidas`, com `EstadoAprovado` e `EstadoBloqueado` mapeados para `nil`.
- [ ] 2.10 Criar `internal/approval/rodada.go` (planejado): entidade `Rodada`, imutável após concluída.
- [ ] 2.11 Criar `internal/approval/tradutor.go` (planejado): camada anticorrupção fail-closed, com
      variantes em português e inglês, e **sem** atalho por substring solta.
- [ ] 2.12 Criar `internal/approval/portas.go` (planejado): `Revisor`, `Corretor`, `Repositorio` e os
      DTOs opacos `PedidoDeRevisao`, `PedidoDeCorrecao`, `SaidaDoRevisor`, `PontoDeCorte`,
      `AlvoDeRevisao`.
- [ ] 2.13 Criar `internal/approval/eventos.go` e `internal/approval/resultado.go` (planejados):
      eventos de domínio e resultado do comando, sem tipo de IO atravessando a fronteira.
- [ ] 2.14 Criar `internal/approval/erros.go` (planejado): sentinelas (`ErrProvaInsuficiente`,
      `ErrTransicaoProibida`, entre outras) e tipos customizados, escolhidos conforme o uso do chamador.
- [ ] 2.15 Criar `internal/approval/ciclo.go` (planejado): agregado `Ciclo` com comando único de
      início; nenhum chamador externo abre rodada — é isso que garante o teto (ADR-001).
- [ ] 2.16 Registrar as três portas em `mockery.yml` e gerar `internal/approval/mocks/` (planejado) com
      `make mocks`; `make check-mocks` verde.
- [ ] 2.17 Escrever a suíte de invariantes com tabela, incluindo os testes de caixa-branca das
      transições proibidas e o fuzzing de fingerprint e tradutor.

## Detalhes de Implementação

Seguir a techspec desta pasta, seção **"Design de Implementação" → "Pacote de domínio
`internal/approval`"**, cuja tabela lista os quinze arquivos e a responsabilidade de cada um, e declara
a lista exaustiva de imports permitidos.

As assinaturas das três portas estão na subseção **"Interfaces Chave"** da mesma techspec, com os
comentários normativos que justificam por que `Revisor` devolve texto bruto e por que `Corretor` devolve
apenas `error`. Reproduzir a semântica, não reinterpretá-la.

O construtor de `ProvaDeAprovacao` está integralmente especificado na subseção **"A invariante central,
garantida por tipo"**, incluindo as mensagens de erro envolvendo `ErrProvaInsuficiente`. A subseção
fecha com a regra de que "`aprovar` só aceita `ProvaDeAprovacao` e recusa o zero-value" e que "não
existe sobrecarga sem prova".

A tabela de transições está na subseção **"Tabela de transições"**; a semântica de "ausência é
proibição" e os terminais `nil` vêm literalmente de lá.

O critério de estabilidade da fingerprint e a justificativa de por que hash do texto integral e contagem
por severidade foram ambos rejeitados estão em **"Fingerprint estável"**.

A ordem das checagens de `Decidir` está em **"Ordem deliberada das checagens"**, que também registra que
"a aprovação **não** passa por ali — só a prova abre o estado Aprovado".

A decisão arquitetural completa, com alternativas rejeitadas e consequências, está em
`adr-001-ciclo-de-aprovacao-agregado.md`, itens 1 e 3 do seu "Plano de Implementação".

A cobertura de teste obrigatória está em **"Abordagem de Testes" → "Testes Unitários"** da techspec,
que exige caixa-branca porque "as transições proibidas só são exercitáveis com acesso ao método de
transição".

## Critérios de Sucesso

- `go build ./internal/approval/... && go vet ./internal/approval/...` verde.
- Isolamento do domínio comprovado por comando:
  `go list -deps ./internal/approval | grep -E 'internal/(runtime|taskloop|detect|install|manifest)|acp|os$|path/filepath|net/http'`
  retorna **vazio**. Complementarmente, `go list -f '{{join .Imports "\n"}}' ./internal/approval`
  produz apenas o conjunto autorizado (`context`, `crypto/sha256`, `encoding/hex`, `errors`, `fmt`,
  `iter`, `slices`, `cmp`, `strings`).
- Os 15 arquivos existem: `ls internal/approval/*.go | wc -l` retorna `>= 15` (excluindo `_test.go`).
- **Uma asserção de teste por transição proibida**, nomeadas individualmente e visíveis em
  `go test ./internal/approval/ -run TestTransicao -v -count=1`: correção não pode aprovar; aprovado é
  terminal; bloqueado não retoma; não há auto-transição.
- **Uma asserção de teste por forma de recusa da prova**, visíveis em
  `go test ./internal/approval/ -run TestProva -v -count=1`: veredito `APPROVED_WITH_REMARKS` recusado;
  veredito `REJECTED` recusado; veredito `BLOCKED` recusado; veredito zero-value recusado; mapa
  incompleto recusado; mapa com critério "não verificável" recusado; zero-value de `ProvaDeAprovacao`
  recusado por `aprovar`. Toda recusa satisfaz `errors.Is(err, ErrProvaInsuficiente)`.
- Fingerprint: teste comprova estabilidade sob reordenação, duplicata, mudança de número de linha e
  mudança de identificador atribuído pelo agente; e instabilidade sob mudança de severidade, arquivo ou
  regra; e que duas fingerprints não-calculadas não são iguais.
- Tradutor: teste comprova que `APPROVED_WITH_REMARKS` **não** é lido como aprovado (o prefixo comum
  torna a ordem de casamento significativa), que texto sem declaração canônica resulta em `BLOCKED`, e
  que as variantes em português e inglês são cobertas.
- Política: teto `< 1` recusado na construção com erro tipado; ordem das checagens verificada pela
  **ausência de chamada ao mock de `Corretor`** quando uma checagem barata já decide.
- Fuzzing presente e executável para fingerprint e tradutor:
  `go test ./internal/approval/ -run Fuzz -fuzz FuzzTradutor -fuzztime 30s` sem crash.
- `make mocks && make check-mocks` verde, com `internal/approval/mocks/` gerado para as três portas.
- `go test ./internal/approval/... -count=1 -cover` verde; cobertura do pacote registrada na evidência.
- `go build ./... && go vet ./... && go test ./... -count=1` verde — nenhuma regressão, já que o pacote
  ainda não tem consumidor.
- `make check-spec-paths` continua verde com os caminhos `(planejado)` desta tarefa agora reais.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

- `domain-modeling-production` — a tarefa materializa um agregado com Value Objects de conjunto fechado, tabela de transições, invariante garantida por tipo e portas declaradas no consumidor, exatamente o escopo de modelagem de domínio orientada a produção exigido por RF-30 e pela ADR-001.

## Testes da Tarefa

- [ ] Testes unitários
- [ ] Testes de integração

Cobertura obrigatória, conforme a techspec:

- **Invariante central**: prova recusada com veredito de ressalvas; com mapa incompleto; com critério
  não verificável; com veredito zero-value. Zero-value da prova nunca válido.
- **Toda transição proibida do modelo**, uma asserção por linha da tabela.
- **Fingerprint**: estabilidade e instabilidade nos eixos declarados; não-calculadas nunca iguais.
- **Tradução fail-closed**: ressalvas não lidas como aprovação; texto sem declaração canônica → bloqueio;
  variantes PT/EN.
- **Política**: teto `< 1` recusado; ordem das checagens verificada pela ausência de chamada ao corretor.
- **Fuzzing** de fingerprint e tradutor, seguindo o padrão já usado no repositório para parsers e
  validadores.
- Integração aqui é limitada por construção: o pacote não toca IO. A prova de integração é o gate de
  isolamento (`go list -deps`) mais `make check-mocks`.

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes

- `internal/approval/` (planejado) — pacote novo, 15 arquivos: `identidades.go`, `veredito.go`,
  `motivo.go`, `achado.go`, `evidencia.go`, `prova.go`, `fingerprint.go`, `politica.go`, `estado.go`,
  `rodada.go`, `tradutor.go`, `portas.go`, `eventos.go`, `resultado.go`, `erros.go`, `ciclo.go`.
- `internal/approval/portas.go` (planejado) — `Revisor`, `Corretor`, `Repositorio` e DTOs opacos.
- `internal/approval/mocks/` (planejado) — mocks das três portas, gerados por `mockery.yml`.
- `mockery.yml` — recebe as três portas novas.
- `internal/runtime/runner_autoreview.go:233` — `buildReviewOutputFromSummary`, o defeito que justifica
  a assinatura de `Revisor` devolver texto bruto. **Não é alterado nesta tarefa** (é da 4.0); citado
  apenas como referência de desenho.
- `internal/taskloop/bugfix.go:83` — `BugfixLoop.Run`, o loop existente que será promovido na 4.0.
  **Não é alterado nesta tarefa.**
- `.specs/prd-harness-quatro-clis-loop-aprovacao/techspec.md` — seções "Design de Implementação",
  "Interfaces Chave", "A invariante central, garantida por tipo", "Tabela de transições",
  "Fingerprint estável", "Ordem deliberada das checagens" e "Abordagem de Testes".
- `.specs/prd-harness-quatro-clis-loop-aprovacao/adr-001-ciclo-de-aprovacao-agregado.md` — decisão,
  alternativas rejeitadas e plano de implementação.
