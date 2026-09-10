# Tarefa 5.0: Agregado de camada com lock por camada, consolidação e escrita atômica

<critical>Ler prd.md e techspec.md desta pasta — sua tarefa será invalidada se você pular</critical>

## Visão Geral

Implementar o agregado que é a fronteira transacional do conjunto de fatos ativos de uma camada. Aqui vivem consolidação, idempotência, marcação de contradição, arquivamento e o scan de busca. O lock é **por camada**, não global: escrita de memória de task não pode bloquear leitura de memória de projeto.

Recebe a abstração de filesystem por construtor, corrigindo o desvio de `internal/runtime/memory/store.go:104` em relação ao ADR-002 do repositório — sem isso não há teste unitário com filesystem falso.

<requirements>
- RF-01, RF-02, RF-03: três camadas; projeto em `.aispec/memory/`; PRD preserva o caminho atual derivado como em `internal/runtime/memory/store.go:88`.
- RF-04: camada de task efetivamente escrita, fechando a lacuna de `internal/runtime/memory/store.go:188` sem chamador.
- RF-11: consolidação não remove fato que não foi contradito.
- RF-14: arquivamento append-only, explícito e reversível.
- RF-16: lock por camada mais escrita atômica; concorrência não corrompe nem perde fato.
- RF-19: busca por texto e por entidade via scan determinístico, sem índice persistido.
- RF-22: página inválida isolada e reportada, sem abortar.
- RF-26: funciona sem PRD ativo, usando a camada de projeto.
</requirements>

## Subtarefas

- [ ] 5.1 Definir `Camada` e `Escopo`, com o agregado recebendo `fs.FileSystem` por construtor.
- [ ] 5.2 Implementar leitura por camada, isolando página inválida e reportando.
- [ ] 5.3 Implementar consolidação com idempotência por identidade e marcação de contradição.
- [ ] 5.4 Implementar arquivamento reversível e promoção entre camadas.
- [ ] 5.5 Implementar lock por camada reutilizando o padrão por plataforma, com tratamento de órfão via lease de 4.0.
- [ ] 5.6 Gravar sempre por escrita atômica de 1.0.
- [ ] 5.7 Chamar `fs.RefuseExternalSymlink` antes de gravar em caminho derivado de configuração.
- [ ] 5.8 Implementar o scan determinístico de busca por texto e por entidade.
- [ ] 5.9 Declarar `Camada` em `mockery.yml`, rodar `make mocks`, confirmar `make check-mocks`.

## Detalhes de Implementação

Ver techspec.md, seções "Interfaces Chave" (bloco `Camada`) e "Pontos de Integração". Ver MD-003, seção Decisão, e MD-002 para a semântica de identidade e contradição.

## Critérios de Sucesso

- Fato não mencionado por uma sessão permanece ativo.
- Regravação idêntica não cria duplicata nem versão nova.
- Arquivado sai do conjunto ativo, permanece no repositório e a reversão funciona.
- Página inválida é isolada e reportada, e a sessão prossegue com o restante.
- Caminho de projeto resolve em `.aispec/memory/`; caminho de PRD é idêntico ao atual.
- Escopo sem diretório de tasks opera apenas na camada de projeto, sem falhar.
- Todo o pacote é testável com `fs.FakeFileSystem`, sem tocar disco real.
- Nenhuma gravação ocorre por caminho não atômico.
- Cobertura do pacote acima de 70%.

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
- Arquivo do agregado de camada no pacote criado em 2.0 — ver techspec.md
- `internal/fs/fs.go` — escrita atômica de 1.0 e `RefuseExternalSymlink`
- `internal/fs/symlink_guard.go` — guarda de traversal, apenas leitura
- `internal/taskloop/orchestrator_lock_unix.go` — padrão de lock, apenas leitura
- `internal/runtime/memory/store.go` — caminho atual de PRD, apenas leitura
- `mockery.yml`
