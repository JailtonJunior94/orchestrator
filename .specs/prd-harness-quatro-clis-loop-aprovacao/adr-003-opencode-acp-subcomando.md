# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** OpenCode via ACP por subcomando, com janela derivada do modelo
- **Data:** 2026-09-10
- **Status:** Aceita
- **Decisores:** Solicitante do PRD
- **Relacionados:** `prd.md` (RF-10 a RF-18), `techspec.md`, ADR do runtime descontinuado (marcada como Substituída)

## Contexto

O conjunto de agentes precisa refletir o ecossistema oficial. Verificação contra fonte primária, com o
binário instalado, estabeleceu os fatos que restringem o desenho:

- O modo ACP é **subcomando**, não flag — divergência de forma frente aos outros três agentes.
- O subcomando **não aceita** seleção de modelo nem flag de permissão. Modelo e política vivem no
  arquivo de configuração do projeto.
- Skills de projeto e o arquivo de governança raiz são **auto-carregados**. Declarar caminho de skills
  seria redundante e, pior, **aditivo** — duplicaria a árvore.
- A flag de modo puro e dois interruptores de ambiente **desligam** o gate por completo.

## Decisão

O OpenCode entra como agente de primeira classe via ACP por subcomando, com launcher de fallback por
gerenciador de pacotes e versões pinadas em constantes, sob o mesmo processo de auditoria dos demais.

A instalação escreve **apenas** o bloco de permissões no arquivo de configuração do projeto, preservando
o esquema e todos os campos preexistentes, e deposita o plugin de governança onde ele é auto-descoberto.
Não copia skills, não cria link simbólico, não declara caminho de skills nem instruções.

A janela de contexto é **derivada do modelo** por tabela versionada, com casamento exato e depois por
maior prefixo, caindo em fallback conservador em qualquer caminho não resolvido. Remover uma entrada da
tabela é seguro por construção: o modelo cai no conservador, nunca em janela maior.

Duas proibições registradas no bloco de permissões, ambas derivadas de comportamento observado: o padrão
total de negação **remove a ferramenta do conjunto oferecido ao modelo**, então só padrões finos são
usados para ferramentas que precisam existir; e o valor "perguntar" é **proibido** em orquestração,
porque seu comportamento diverge entre modos de execução.

## Alternativas Consideradas

**Integração apenas pelo caminho não-interativo.** Vantagem: mais simples. Desvantagem: perde eventos,
watchdog e telemetria, violando a paridade observacional. Rejeitada.

**Copiar skills para o diretório do agente.** Vantagem: simétrico ao que se faz para outro agente.
Desvantagem: duplica o que já é auto-carregado, criando ponto de drift. Rejeitada por evidência.

**Constante estática de janela, como nos irmãos.** Vantagem: determinístico e simples. Desvantagem: o
agente é agnóstico de modelo; uma constante daria orçamento grande a um modelo de janela pequena.
Rejeitada.

## Consequências

### Benefícios Esperados

- Paridade observacional completa com os demais agentes.
- Menor pegada possível no projeto do usuário: cada arquivo não escrito é um ponto de drift que não
  existe.
- A abstração de spec passa a acomodar duas formas de invocação sem caso especial no executor.

### Trade-offs e Custos

- Uma tabela de janela por modelo a manter, com o mesmo rigor de versionamento das constantes.
- Escrita em arquivo de configuração de terceiro, com todo o cuidado de merge que isso exige.

### Riscos e Mitigações

**Tabela de modelo desatualizada.** Impacto: orçamento conservador demais. Mitigação: o fallback nunca
superestima, e a atualização segue o processo de auditoria.

**Escrita destrutiva na configuração do usuário.** Mitigação: merge preservando esquema e campos
preexistentes, idempotente, sem remover configuração.

**Mudança de forma de invocação a montante.** Mitigação: a validação de formato de argumentos falha de
modo visível se a invariante de prefixo posicional deixar de valer.

## Plano de Implementação

Spec, registro e instalação vêm **antes** da remoção do agente descontinuado, porque nas estruturas de
ocupante único o novo agente precisa assumir a entrada antes de a antiga sair.

## Monitoramento e Validação

Sucesso: um projeto somente com este agente tem exatamente os mesmos gates que um projeto somente com o
agente de referência; e os eventos, a renderização de chamadas de ferramenta e o relatório de execução
são equivalentes.

## Impacto em Documentação e Operação

Guia de instalação, matriz de confiabilidade, tabela de governança por ferramenta e a tabela de
regras de normalização de chamadas de ferramenta.

## Revisão Futura

Revisitar se o agente passar a expor seleção de modelo por argumento, ou se a forma do modo ACP mudar.
