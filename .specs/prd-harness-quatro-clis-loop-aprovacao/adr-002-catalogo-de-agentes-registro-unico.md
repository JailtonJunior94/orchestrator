# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Catálogo de Agentes como registro único, com Agente como Value Object
- **Data:** 2026-09-10
- **Status:** Aceita
- **Decisores:** Solicitante do PRD
- **Relacionados:** `prd.md` (RF-01 a RF-18), `techspec.md`, `discoveries/domain-loop-de-aprovacao-e-catalogo-de-agentes-cli/`

## Contexto

Remover um agente do harness custa cerca de duzentos e quinze arquivos. A causa mecânica foi mapeada: a
enumeração dos agentes está espalhada por **onze** pontos independentes — dois catálogos de runtime
espelhados, resolução de spec no instalador, entradas de detecção, resolução no servidor de agentes
aninhados, mapeamento de perfil, dois parsers de identidade, mapa de decisões arquiteturais, mapas de
orçamento e ordem de exibição.

Três consequências concretas foram confirmadas por leitura:

- **Duas ordens canônicas** de agentes coexistem, produzindo saída não-determinística entre comandos.
- Remover uma entrada de apenas um dos catálogos espelhados faz a resolução cair em **fallback
  silencioso**, executando o job inteiro no agente errado sem nenhum erro.
- Duas estruturas têm o agente a remover como **entrada única**; esvaziá-las muda o comportamento sem
  que nenhum teste falhe.

Some-se que a matriz de paridade afirma cobertura que não é verificada: nenhum teste do repositório
prova **disparo** de hook — todos verificam apenas escrita de arquivo. Foi assim que uma chave de evento
inválida sobreviveu, deixando o gate de encerramento de um dos agentes inerte desde sempre.

## Decisão

Os onze pontos são substituídos por um **registro único**. Cada Agente é Value Object imutável, portando
identidade, spec de invocação, enforcement por ponto canônico, sinais de detecção, política de ambiente,
referência de decisão arquitetural e orçamentos. Todos os call sites passam a derivar do registro.

Dois tipos que o modelo exige e o código não tinha:

- **Ponto canônico** como conjunto fechado, com o construtor de enforcement **recusando** cobertura
  incompleta. A invariante "um agente só integra o catálogo se cobrir os três pontos" passa a ser
  verificada na construção do registro, não em revisão humana.
- **Pré-condição de enforcement** como conceito de primeira classe, carregando o remédio acionável.

A forma de invocação — ACP por flag ou por subcomando — deixa de exigir caso especial: a lista de
argumentos fixos já é prefixo posicional, e essa invariante, hoje acidental, passa a ser travada por
validação de formato.

Adicionar ou remover agente passa a ser editar o registro, o arquivo de spec e o asset de hooks. O gate
de paridade falha se faltar qualquer um dos três.

## Alternativas Consideradas

**Manter os onze pontos e sincronizá-los por revisão.** Vantagem: nenhuma mudança estrutural.
Desvantagem: é o estado atual, e ele já produziu duas ordens divergentes, um fallback silencioso e uma
chave de evento inválida. Rejeitada por evidência de falha.

**Agentes como configuração externa, sem tipo de domínio.** Vantagem: adicionar agente sem recompilar.
Desvantagem: perde a validação do conjunto fechado em tempo de compilação — e o conjunto fechado é a
invariante que sustenta a paridade. Rejeitada.

**Cada agente como entidade autônoma com seu registro.** Vantagem: espelha a organização atual por
arquivo. Desvantagem: mantém a enumeração espalhada; nada garante que os agentes cubram os mesmos
pontos. Rejeitada.

## Consequências

### Benefícios Esperados

- O custo de adicionar ou remover agente cai de centenas de arquivos para três.
- Cobertura de enforcement incompleta deixa de compilar o registro.
- Ordem canônica única elimina saída não-determinística.
- O fallback silencioso é substituído por erro explícito.
- Os mapas de orçamento deixam de poder ficar vazios: o orçamento passa a viver no próprio agente.

### Trade-offs e Custos

- Refactor amplo, tocando quase todos os pacotes que hoje enumeram agentes.
- O registro concentra responsabilidade: um erro nele afeta tudo — mitigado por ser justamente o ponto
  mais coberto por gates.
- A resolução do registro usa inicialização única em variável de pacote, o que é a fronteira mais
  próxima que o desenho chega das regras de inicialização implícita. Alternativa estrita — reconstruir a
  cada chamada — custa alocações em caminho quente. Se a variável de pacote for considerada inaceitável
  em revisão, a mudança é local e nada mais no desenho muda.

### Riscos e Mitigações

**Regeneração de golden files congelando regressão.** Mitigação: revisão manual do diff arquivo a
arquivo; regeneração automática é cheque em branco.

**Ordem de remoção quebrando o build em ponto intermediário.** Mitigação: ordem topológica de quinze
etapas, cada uma deixando o repositório verde, começando pelo gate que exige a própria string a remover
— a única aresta sem predecessora.

**Estruturas esvaziadas sem sinal.** Mitigação: gate de **cobertura**, não de não-vacuidade: todo agente
cuja janela resolvida for grande precisa de entrada, e nenhuma chave órfã pode existir. O gate força a
decisão explícita em vez de aceitar valor de fachada.

## Plano de Implementação

Fase de catálogo precede a de remoção, porque nas células de ocupante único o novo agente precisa entrar
antes. Dentro da remoção, a ordem é topológica e cada etapa tem comando de validação próprio.

Adoção concluída quando nenhum ponto do código enumera agentes fora do registro, verificado por gate que
varre a árvore em busca de literais de lista de agentes fora da lista de exceções.

## Monitoramento e Validação

Sucesso: o gate de ordem canônica, o de sincronia de catálogo e o de cobertura de orçamento verdes; a
matriz de paridade sem célula sem teste de disparo associado. Sinal de revisão: necessidade recorrente
de exceção na lista de literais permitidos.

## Impacto em Documentação e Operação

Tabela de governança por ferramenta, guia de instalação, matriz de degradação, matriz de confiabilidade,
esquema da linha de comando e os golden files de governança gerada.

## Revisão Futura

Revisitar se o número de agentes crescer a ponto de o registro literal ficar difícil de ler, ou se
surgir demanda real de registrar agente sem recompilar.
