# Tarefa 2.0: Fato, identidade, durabilidade, sentinelas e Página com round-trip lossless

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Criar o pacote de domínio da memória durável: os tipos, as dez sentinelas de erro e o leitor/serializador de Página com round-trip garantido. É a fatia mais importante do plano, porque **trava a invariante mais frágil antes de existir qualquer consumidor**: o que o domínio não gerou, o domínio não reescreve.

Tipos e Página vêm juntos por dependência de compilação — a assinatura de `Pagina` usa `Fato` e `BlocoHumano`. As sentinelas são declaradas aqui para que nenhuma fatia posterior dispute o mesmo arquivo.

Não depende de 1.0: a interface opera sobre `[]byte` nas duas direções e não toca filesystem.

<requirements>
- RF-05: Página é Markdown válido com frontmatter declarando identidade, camada, sessão de origem, data e versão de formato.
- RF-36: escrita cujo round-trip não reproduza o conteúdo de autoria humana é recusada.
- RF-37: bloco de autoria humana é preservado, contado no orçamento e sinalizado.
- RF-12: identidade por chave semântica declarada mais hash do conteúdo; regravação idêntica não duplica.
- RF-27: contradição é colisão de chave com hash divergente, detectada sem LLM.
- RF-07: durabilidade é enum fechado de três valores, obrigatória na origem; zero-value inválido.
- RF-06: ligações tipadas nas relações substitui, causa, corrige e contradiz.
- RF-01 e RF-02: durabilidade resolve camada de destino; Markdown puro como fonte de verdade.
</requirements>

## Subtarefas

- [x] 2.1 Criar o pacote com `r1_catalog.go` (`type Catalog struct{}` e `func NewCatalog() *Catalog`), conforme a regra R1.
- [x] 2.2 Definir `Fato`, `Identidade`, `ChaveSemantica`, `HashConteudo`, `Durabilidade`, `OrigemDeFato`, `Ligacao`, `EstadoFato`, `BlocoHumano`.
- [x] 2.3 Declarar as dez sentinelas de erro em arquivo próprio, com `errors.New` e prefixo de pacote.
- [x] 2.4 Implementar derivação determinística de chave semântica a partir do sinal estruturado.
- [x] 2.5 Implementar `Parse` e `Serializar` com frontmatter via `gopkg.in/yaml.v3`, já dependência direta.
- [x] 2.6 Implementar a resolução de camada a partir da durabilidade.
- [x] 2.7 Implementar detecção de contradição por colisão de chave com divergência de hash.
- [x] 2.8 Declarar `Pagina` em `mockery.yml`, rodar `make mocks`, confirmar `make check-mocks`.
- [x] 2.9 Escrever o documento do formato de Página, para que a edição manual seja segura e informada.

## Detalhes de Implementação

Ver techspec.md, seções "Interfaces Chave" (bloco `Pagina`), "Modelos de Dados" e "Abordagem de Testes". Ver MD-002 (`adr-002-fato-pagina-roundtrip-lossless.md`) integralmente — é a ADR desta tarefa.

## Critérios de Sucesso

- **Teste de round-trip alimentado por corpus de páginas reais do repositório**, não apenas fixtures sintéticas, falhando se qualquer byte de `BlocoHumano` mudar.
- Zero-value de `Durabilidade` é recusado com erro tipado distinguível por `errors.Is`.
- Fato sem chave semântica é recusado com erro tipado.
- Duas execuções com o mesmo sinal estruturado produzem a mesma chave semântica.
- Chave igual com hash igual não duplica; chave igual com hash diferente marca contradição, e ambos os fatos permanecem.
- Página gerada é Markdown válido e legível por `grep` e por editor.
- Cobertura do pacote novo acima de 70%, threshold de `scripts/check-package-coverage.sh`.

## Skills Necessárias

<!-- MANDATÓRIO: preenchido por `create-tasks` Etapa 4.1 via descoberta agnóstica em `.agents/skills/`.
     NÃO inclua aqui skills cuja categoria no frontmatter seja `governance` ou `language`:
     elas são auto-carregadas em runtime. A classificação deriva exclusivamente de `category`,
     nunca do nome da skill.
     Use o conteúdo único `Nenhuma além das auto-carregadas (governance + linguagem).` se a tarefa
     não exigir skill processual extra. -->

Nenhuma além das auto-carregadas (governance + linguagem).

## Testes da Tarefa

- [x] Testes unitários
- [x] Testes de integração

<critical>SEMPRE CRIAR E EXECUTAR TESTES DA TAREFA ANTES DE CONSIDERAR A TAREFA COMO `done`</critical>

## Arquivos Relevantes
- Pacote novo de domínio da memória durável (tipos, sentinelas, Página) — ver techspec.md, "Visão Geral dos Componentes"
- `gopkg.in/yaml.v3` — dependência direta já presente
- `mockery.yml` — declarar `Pagina`
- `internal/runtime/memory/window_policy.go` — molde de domain service stateless, apenas leitura
- `internal/runtime/memory/r1_catalog.go` — molde do agrupador stateless, apenas leitura
