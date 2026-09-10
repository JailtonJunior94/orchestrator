# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Rótulo:** MD-005
- **Título:** Evidência e métricas de memória por extensão do conjunto de métricas existente, sem seção nova no relatório
- **Data:** 2026-09-10
- **Status:** Proposta
- **Decisores:** JailtonJunior94 (mantenedor do harness)
- **Relacionados:** [PRD](prd.md) RF-32, RF-34, RF-35 · [techspec](techspec.md) · [ADR-006 do repositório](../../docs/adr/006-telemetria-feedback-cycle.md) · MD-001, MD-004

## Contexto

RF-34 exige que o relatório de execução registre a operação de memória da sessão: o que foi lido, o que foi gravado, orçamento consumido por camada, se houve compactação, se houve redação de segredo, e se houve tomada de bastão. RF-32 exige rastrear cada fato até a sessão de origem. RF-35 exige métricas na telemetria, mantendo o caráter opt-in do ADR-006 do repositório.

O levantamento do código de produção revelou três restrições que determinam a forma da solução.

Primeira: o relatório é montado por injeção de seção com expressões regulares. `internal/runtime/persistence/report.go:20` detecta a seção de runtime, e `internal/runtime/persistence/report.go:104` substitui a seção de métricas **do cabeçalho até o fim do arquivo**. Isso torna a seção de métricas obrigatoriamente a última seção do relatório, e qualquer seção nova precisa vir antes dela para não quebrar esse invariante.

Segunda: o conjunto de métricas já possui um mapa de campos extra, e a renderização em `internal/runtime/persistence/report.go:71` itera os campos de forma genérica. Um campo novo adicionado ao mapa aparece na tabela do relatório sem tocar template algum.

Terceira: o tipo de evento do runtime é um enum fechado, validado em `internal/runtime/events/kinds.go:25`. Não há mecanismo de extensão dinâmica: um tipo de evento novo exige alterar o enum e a tabela de validação. Já o distribuidor de hooks em `internal/runtime/hooks/dispatcher.go:31` exige apenas um método de identificação, e possui pontos canônicos já declarados — incluindo dois pontos de tool-call sem nenhum hook registrado hoje.

Há também uma restrição de forma, não de código: os validadores de evidência em `.agents/scripts/` capturam seções do relatório por bloco, e o `AGENTS.md` registra que expressões de bracket com caracteres multibyte nunca casam em processador de texto orientado a byte, desligando o gate em silêncio. Uma seção nova mal posicionada pode fechar prematuramente a captura de uma seção anterior.

## Decisão

Os sete eventos de domínio da memória são emitidos pelo **distribuidor de hooks existente**, não pelo enum fechado de eventos do runtime. O distribuidor exige apenas um método de identificação e já oferece fan-out sequencial com aborto no primeiro erro — reutilizá-lo evita estender um enum fechado e evita construir um segundo mecanismo de notificação.

As métricas de memória entram pelo **mapa de campos extra do conjunto de métricas existente**, e não por campos novos na estrutura de resumo. A renderização já é genérica, portanto as métricas aparecem no relatório sem alteração de template e sem risco de quebrar o invariante de posição.

A evidência qualitativa exigida por RF-34 — redações aplicadas, páginas isoladas, tomadas de bastão, omissões por orçamento — é injetada como seção **posicionada antes da seção de métricas**, respeitando o invariante de que métricas são a última seção. O nome da seção é escolhido de forma a não colidir com nenhum dos padrões de cabeçalho que os validadores de evidência já procuram.

A rastreabilidade de RF-32 vive nos metadados do próprio Fato — sessão, CLI e data — conforme MD-002 desta feature, e não em um índice externo. A inspeção via CLI lê os metadados; não há segundo lugar onde a origem possa divergir.

A telemetria segue o formato de linha com pares de chave e valor já usado no repositório, com campos adicionados de forma condicional para não poluir sessões que não usaram memória. O caráter opt-in do ADR-006 é preservado sem exceção.

## Alternativas Consideradas

**Estender o enum fechado de tipos de evento do runtime.** Vantagens: os eventos de memória apareceriam no registro de eventos da sessão, junto dos demais. Desvantagens: exige alterar o enum e a tabela de validação, ampliando o contrato do registro de eventos, que é lido por ferramentas externas; e o registro é reescrito por inteiro a cada evento, conforme `internal/runtime/persistence/jsonl.go`, o que aumenta o custo de escrita proporcionalmente ao número de eventos novos. Não escolhida.

**Construir um mecanismo próprio de notificação para os sete eventos.** Desvantagens: duplicaria o distribuidor existente, contra a regra de economia que orienta preferir desacoplamento local a framework novo.

**Campos estruturados novos na estrutura de resumo, em vez do mapa de campos extra.** Vantagens: tipagem forte, campos nomeados no código. Desvantagens: exigiria alterar o template do relatório, e o levantamento mostrou que a estrutura de resumo já cresceu ao longo de várias ondas — cada campo novo é uma alteração de contrato de um tipo consumido em vários lugares. O mapa de campos extra existe exatamente para esse caso e já é renderizado genericamente. Não escolhida para contadores; permanece como opção se algum dado de memória precisar de forma não escalar.

**Seção nova no fim do relatório.** Desvantagens: quebraria o invariante de `internal/runtime/persistence/report.go:104`, que substitui do cabeçalho de métricas até o fim do arquivo — a seção nova seria apagada na próxima injeção de métricas.

**Índice externo de rastreabilidade.** Vantagens: consulta mais rápida por origem. Desvantagens: cria um segundo lugar onde a origem pode divergir do Fato, e contraria a decisão de não manter índice persistido.

**Emitir métricas de memória sem respeitar o opt-in.** Rejeitada de imediato: contraria o ADR-006 do repositório e a garantia de privacidade declarada no PRD.

## Consequências

### Benefícios Esperados

- Nenhuma alteração no enum fechado de eventos nem no contrato do registro de eventos da sessão.
- Métricas aparecem no relatório sem tocar template, porque a renderização já itera campos genericamente.
- O invariante de posição da seção de métricas é preservado por construção.
- Rastreabilidade sem índice: a origem vive no Fato, em um único lugar.
- Reuso do distribuidor de hooks, incluindo os dois pontos de tool-call hoje declarados e sem hook registrado, caso a memória precise reagir por tool-call no futuro.

### Trade-offs e Custos

- Os eventos de memória não aparecem no registro de eventos da sessão, apenas na evidência e na telemetria. Quem procurar memória no registro de eventos não vai encontrar — isso precisa estar documentado.
- Métricas no mapa de campos extra são contadores sem tipagem forte: um erro de digitação no nome da chave produz uma métrica silenciosamente separada. Mitigado por constantes nomeadas para as chaves.
- A seção de evidência precisa ser posicionada com cuidado em relação aos validadores, e essa restrição é frágil por natureza, porque depende de captura por expressão regular.

### Riscos e Mitigações

- **Risco:** a seção nova fechar prematuramente a captura de uma seção anterior por um validador de evidência. **Impacto:** gate de evidência desligado em silêncio, que é o padrão de defeito que o `AGENTS.md` documenta. **Mitigação:** o teste de validadores do repositório precisa ser estendido com um relatório contendo a seção nova, provando que os gates continuam capturando corretamente. Nenhuma expressão nova pode usar classe de bracket com caractere multibyte.
- **Risco:** chave de métrica digitada de forma inconsistente entre pontos de emissão. **Mitigação:** constantes nomeadas, e teste que enumera as chaves esperadas.
- **Risco:** falha de um hook de memória abortar o fan-out e impedir hooks subsequentes, dado o aborto no primeiro erro do distribuidor. **Impacto:** hook não relacionado deixa de executar. **Mitigação:** os hooks de memória tratam o próprio erro e o compõem na evidência, em vez de propagá-lo — coerente com a decisão de MD-004 de registrar sem abortar.
- **Plano de rollback:** desativar a feature elimina a emissão. A seção de evidência deixa de ser injetada, e o relatório volta ao formato atual.

## Plano de Implementação

1. Definir os sete eventos de domínio como tipos que satisfazem o contrato de identificação do distribuidor existente.
2. Definir constantes nomeadas para as chaves de métrica de memória.
3. Emitir as métricas pelo mapa de campos extra do conjunto de métricas.
4. Injetar a seção de evidência antes da seção de métricas, com função de injeção seguindo o padrão existente.
5. Estender o teste de validadores de evidência do repositório com um relatório que contenha a seção nova.
6. Adicionar os campos de telemetria de forma condicional, preservando o opt-in.

Dependências: MD-001 precede, porque a fachada é quem produz o resumo da operação. MD-004 precede o passo 4, porque a evidência inclui falhas que hoje são descartadas.

Critério de adoção concluída: o relatório contém a evidência exigida por RF-34, a seção de métricas permanece a última, e os gates de validação de evidência continuam capturando corretamente.

## Monitoramento e Validação

- **Sinais:** presença da seção de evidência de memória nos relatórios de sessões com a feature ativada; contagem de chaves de métrica distintas emitidas, que deve ser estável.
- **Logs:** cada evento de domínio emitido, quando o registro estiver habilitado.
- **Critério de sucesso:** o teste de validadores passa com o relatório estendido, e a seção de métricas permanece a última em todos os relatórios gerados.
- **Critério para revisar:** se algum dado de memória exigir forma não escalar, o mapa de campos extra deixa de servir e a estrutura de resumo passa a ser o lugar certo.

## Impacto em Documentação e Operação

- `docs/telemetry-feedback-cycle.md`: campos novos de telemetria.
- `.agents/scripts/` e seus espelhos: se algum validador precisar reconhecer a seção nova, a alteração exige o gate de sincronização de scripts do repositório.
- `docs/task-loop-reference.md`: formato da evidência de memória no relatório.
- Documentação da feature: registro explícito de que eventos de memória não aparecem no registro de eventos da sessão.

## Revisão Futura

Revisar se a memória passar a precisar reagir por tool-call, porque nesse caso os dois pontos de tool-call hoje sem hook entram em uso e o volume de eventos muda de ordem de magnitude. Revisar também se o registro de eventos da sessão ganhar escrita incremental real, porque aí o custo de estender o enum de tipos de evento cai e a alternativa rejeitada volta a ser viável.
