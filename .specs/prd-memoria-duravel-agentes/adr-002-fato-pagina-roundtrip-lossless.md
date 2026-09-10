# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Rótulo:** MD-002
- **Título:** Fato como unidade identificada por chave semântica e hash; Página como forma de persistência com round-trip lossless
- **Data:** 2026-09-10
- **Status:** Proposta
- **Decisores:** JailtonJunior94 (mantenedor do harness)
- **Relacionados:** [PRD](prd.md) RF-02, RF-05, RF-06, RF-07, RF-12, RF-27, RF-36, RF-37 · [techspec](techspec.md) · [modelo de domínio](../../discoveries/domain-memoria-duravel-de-agentes/domain-model.md) · MD-001, MD-003, MD-005

## Contexto

O PRD estabelece duas exigências que estão em tensão direta. RF-02 determina que a memória permaneça Markdown puro versionável em git, legível e editável por humanos e por ferramentas externas. O modelo de domínio determina que a unidade de conhecimento seja o Fato, com granularidade fina o suficiente para ser promovido, arquivado e contraditado individualmente (RF-07, RF-24, RF-27).

Se o arquivo for a unidade, a promoção e a detecção de contradição operam em granularidade grossa demais. Se o Fato for um registro em formato de dados, o Markdown deixa de ser fonte de verdade e a feature recria o lock-in que existe para evitar.

Há uma segunda tensão, encontrada durante a modelagem e ainda não endereçada por nenhum requisito da versão anterior do PRD: se o harness reescreve páginas para consolidar Fatos, ele reescreve também o que uma pessoa escreveu à mão naquele arquivo. RF-02 promove ativamente a edição manual em editor ou Obsidian — logo, reescrever texto humano seria destruir o valor que o requisito cria.

O formato atual não ajuda. O único ponto de persistência de produção grava um template fixo de quatro campos em modo destrutivo (`internal/runtime/hooks/memory_persist.go:65`), e o conteúdo é string opaca — não existe no código nenhuma noção de fato, identidade, origem ou durabilidade.

## Decisão

A Página é a forma de persistência do agregado de camada; o Fato é uma seção dentro dela, com metadados próprios; e o par leitura/serialização é obrigado a preservar round-trip.

A identidade de um Fato é formada por **chave semântica declarada em conjunto com hash do conteúdo**. A chave estabelece sobre o que o Fato afirma algo; o hash estabelece se o conteúdo mudou. Dessa composição derivam duas capacidades sem qualquer chamada de modelo de linguagem: idempotência exata, quando chave e hash coincidem (RF-12), e candidatura a contradição, quando a chave coincide e o hash divergem (RF-27).

A durabilidade é um enum fechado de três valores — `Efêmero`, `DePRD`, `Durável` — declarado na origem e obrigatório. Ela mapeia diretamente para a camada de destino, sem tabela de tradução, o que torna a resolução de camada determinística por construção (RF-07).

Conteúdo de autoria humana que não corresponde a nenhum Fato válido é **preservado integralmente, contado no orçamento de contexto e sinalizado para compactação humana** — nunca reescrito automaticamente (RF-37). A serialização que não reproduza esse conteúdo é recusada com erro tipado (RF-36). Isso torna o round-trip uma invariante de escrita, não uma aspiração de qualidade.

O frontmatter usa a biblioteca de serialização YAML já presente como dependência direta, sem adicionar dependência nova.

## Alternativas Consideradas

**Página como agregado, Fato como seção sem identidade.** Vantagens: parse trivial, menos metadados, arquivo mais limpo. Desvantagens: promoção e contradição passariam a operar por arquivo inteiro, revertendo na prática a escolha do termo canônico feita na modelagem. Não escolhida porque anularia RF-24 e RF-27.

**Um arquivo por Fato.** Vantagens: identidade natural pelo caminho, parse trivial, nenhum problema de round-trip. Desvantagens: explosão de arquivos pequenos, histórico de git ruidoso, e leitura humana ruim — contra o valor de wiki navegável que RF-02 promove. Não escolhida porque o custo de usabilidade e de histórico supera a simplificação técnica.

**Página como projeção derivada de um arquivo de dados.** Vantagens: parse trivial, modelo interno livre de restrições de formato. Desvantagens: viola RF-02 diretamente — a fonte de verdade deixaria de ser Markdown puro. Não escolhida.

**Identidade apenas por hash de conteúdo.** Vantagens: idempotência trivial e exata, nenhum campo novo obrigatório. Desvantagens: qualquer reformulação cria fato novo, e a contradição fica indetectável, porque dois textos diferentes sobre o mesmo assunto parecem não relacionados. Não escolhida porque inviabilizaria RF-27 sem modelo de linguagem.

**Identidade apenas por identificador gerado na origem.** Vantagens: sempre único, rastreável à sessão. Desvantagens: regravação de conteúdo idêntico duplicaria, violando RF-12 diretamente.

**Identidade por caminho e posição na página.** Desvantagens: a identidade se rompe a cada edição manual, compactação ou reordenação — e edição manual é justamente o que RF-02 promove.

**Durabilidade como escala numérica.** Vantagens: ordenável, granularidade fina. Desvantagens: número sem semântica de negócio, e a fronteira de promoção viraria limiar arbitrário impossível de justificar.

**Durabilidade em dois valores, com camada resolvida por regra separada.** Desvantagens: exigiria uma segunda regra independente para escolher entre camada de task e de PRD, duplicando a decisão que a durabilidade deveria resolver sozinha.

**Conteúdo humano fora do orçamento e da compactação.** Vantagens: respeito máximo à autoria, implementação mais simples. Desvantagens: abriria um caminho por onde a página cresce sem limite algum, furando a garantia de RF-13. Não escolhida — a alternativa adotada mantém RF-13 honesto ao reportar a violação em vez de silenciá-la.

**Converter conteúdo humano em Fato com durabilidade inferida.** Desvantagens: reescreveria texto humano e inferiria semântica que ninguém declarou.

## Consequências

### Benefícios Esperados

- Markdown permanece fonte de verdade: `grep`, editor e Obsidian continuam funcionando, e o índice é dispensável (RF-19).
- Detecção de contradição determinística, sem custo de modelo de linguagem e sem heurística de similaridade.
- Idempotência exata, sem falso positivo de duplicata por reformulação.
- Resolução de camada sem tabela de tradução, portanto sem um lugar onde a regra possa divergir do enum.
- Edição manual permanece segura, o que preserva o valor de wiki editável.

### Trade-offs e Custos

- Round-trip lossless exige teste dedicado permanente. É o custo residual mais alto desta decisão, e é permanente por natureza: é a invariante que protege conteúdo humano.
- Chave semântica é campo obrigatório que alguém precisa preencher. Fato sem chave é escrita inválida, o que significa que a captura automática precisa derivar chave a partir do sinal estruturado de forma determinística.
- A página fica mais verbosa que um bloco de texto livre, porque cada Fato carrega metadados.
- Compactação pode não alcançar o limite quando o excedente é conteúdo humano, e essa violação precisa ser reportada em vez de resolvida.

### Riscos e Mitigações

- **Risco:** parse frágil quebrar em página editada à mão de forma inesperada. **Impacto:** perda de fatos ou contaminação do contexto. **Mitigação:** RF-22 exige isolar a página inválida e reportar, com a sessão prosseguindo; o teste de falha cobre esse caminho.
- **Risco:** chave semântica derivada automaticamente ficar instável entre sessões, gerando fatos duplicados que deveriam colidir. **Impacto:** poluição do conjunto ativo e contradições espúrias. **Mitigação:** a derivação é determinística a partir do sinal estruturado, e o teste de idempotência exercita duas sessões consecutivas com o mesmo sinal.
- **Risco:** round-trip passar nos testes e falhar em conteúdo real com construções Markdown incomuns. **Mitigação:** teste de round-trip alimentado por corpus de páginas reais do repositório, não apenas por fixtures sintéticas.
- **Plano de rollback:** reverter para Página como agregado sem identidade de Fato mantém o formato legível e sacrifica RF-24 e RF-27, que passariam a fora de escopo.

## Plano de Implementação

1. Definir o formato de Página e de seção de Fato, com frontmatter usando a dependência YAML existente.
2. Implementar o leitor e serializador e **travar o round-trip com teste dedicado antes de qualquer outro consumidor existir** — esta é a primeira entrega executável da feature.
3. Implementar a derivação determinística de chave semântica a partir do sinal estruturado.
4. Implementar o enum fechado de durabilidade e a resolução de camada.
5. Implementar detecção de contradição por colisão de chave com divergência de hash.
6. Implementar a contabilização de bloco de autoria humana no orçamento e sua sinalização.

Dependências: nenhuma. Esta ADR é a base executável — MD-001 e MD-003 dependem dela, não o contrário.

Critério de adoção concluída: o teste de round-trip falha se qualquer byte de conteúdo de autoria humana for alterado, e passa em corpus de páginas reais.

## Monitoramento e Validação

- **Sinais:** número de páginas isoladas por invalidez; número de contradições abertas; número de recusas de escrita por round-trip não preservado.
- **Logs:** cada página isolada e cada recusa de round-trip devem aparecer no relatório da sessão, porque RF-30 proíbe degradação silenciosa.
- **Critério de sucesso:** editar uma página à mão e rodar uma sessão em seguida não altera nenhum byte do que a pessoa escreveu.
- **Critério para revisar:** se o volume de páginas isoladas por invalidez for recorrente, o formato é frágil demais e precisa simplificar.

## Impacto em Documentação e Operação

- Documento novo descrevendo o formato de Página e de Fato, necessário para que a edição manual seja segura e informada.
- `docs/troubleshooting.md`: seção sobre página isolada por invalidez e como corrigir à mão.
- `AGENTS.md`: registro de que memória é Markdown com frontmatter e que edição manual é suportada.

## Revisão Futura

Revisar se a derivação automática de chave semântica se mostrar instável em uso real, porque nesse caso a identidade precisa de outra composição. Revisar também se busca semântica entrar em escopo, porque embeddings mudariam a natureza da detecção de contradição e possivelmente tornariam a chave semântica redundante.
