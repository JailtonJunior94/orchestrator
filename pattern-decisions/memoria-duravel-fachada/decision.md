# Pattern Decision Bundle

## Contexto
Problema:
O Runtime de Sessao consome hoje um contrato estreito de quatro metodos (`memory.Store`) para ler e gravar memoria em arquivo. A feature de memoria duravel introduz um subsistema muito maior atras desse mesmo ponto: parse de Fatos em Pagina Markdown com round-trip lossless, cinco politicas stateless (relevancia, orcamento, lease, sanitizacao, compactacao), agregado de camada com lock por plataforma, bastao de continuidade com lease, e reconciliacao de sessao orfa. O consumidor nao pode orquestrar esses passos, e o contrato existente nao pode mudar de forma que quebre chamadores e mocks gerados.

Objetivo tecnico:
Expor ao Runtime de Sessao uma unica porta estreita e estavel, que internamente orquestre o subsistema completo na ordem correta de invariantes, preservando o contrato atual e permitindo que a operacao humana via CLI continue acessando o subsistema em granularidade total.

Restricoes inegociaveis:
Preservar o contrato publico consumido em producao. Nao introduzir heranca (Go, e R1/R6 da skill `go-implementation`). Minimizar indirecao e contagem de tipos. Zero estado global (R0/R6). Caminho default byte-identico quando a feature esta desativada. Sem CGO, sem daemon, sem rede, sem chamada de modelo de linguagem no caminho default.

## Diagnostico do problema
Sintomas observados:
O unico ponto de persistencia de memoria em producao grava em modo destrutivo, apagando o estado anterior a cada sessao. O metodo de escrita do tier de task existe no contrato e nao possui nenhum chamador de producao. A compactacao e uma frase anexada ao prompt, sem garantia de execucao. O consumidor conhece o formato textual injetado e a politica de limites ao mesmo tempo, sem fronteira entre orquestracao e regra.

Causa estrutural provavel:
Nao existe fronteira entre o consumidor e o subsistema. A logica de memoria esta espalhada entre o runner (montagem de contexto, resolucao de limites, decisao de registrar hook) e um hook de persistencia que conhece o template do arquivo. Qualquer capacidade nova precisa ser costurada em dois lugares, e o consumidor cresce junto com o subsistema.

## Evidencias
Evidencia de codigo (path:line ou greenfield):
internal/runtime/hooks/memory_persist.go:65 grava com modo destrutivo, unico ponto de persistencia em producao.
internal/runtime/memory/store.go:68 declara o metodo de escrita de task no contrato; internal/runtime/memory/store.go:188 o implementa; nenhum chamador de producao existe.
internal/runtime/runner.go:470 instancia o store; internal/runtime/runner.go:481 e internal/runtime/runner.go:482 sao os unicos pontos de leitura em producao.
internal/runtime/runner.go:516 anexa a diretiva textual de compactacao ao prompt.
internal/runtime/runner.go:490 concentra no consumidor o formato exato do bloco de contexto injetado.
internal/runtime/memory/window_policy.go:37 mostra que politica stateless como domain service ja e o idioma aceito do pacote.
internal/runtime/memory/store.go:104, internal/runtime/memory/store.go:121 e internal/runtime/memory/store.go:153 usam chamadas diretas de sistema operacional, divergindo da abstracao de filesystem injetada que o restante do repositorio adota.
internal/sdd/state.go:299 ate internal/sdd/state.go:321 contem o unico padrao de escrita atomica do repositorio, disponivel para reuso.
internal/taskloop/orchestrator_lock_unix.go:17 e internal/taskloop/orchestrator_lock_windows.go:15 contem o lock por plataforma reutilizavel.
mockery.yml:58 declara a interface de memoria como origem de mock gerado, o que torna qualquer mudanca de contrato uma mudanca de artefato gerado.

Evidencia de comportamento:
Com o diretorio de tasks vazio, internal/runtime/runner.go:463 retorna store nulo e internal/runtime/runner.go:476 devolve o prompt original sem injecao, e o hook de persistencia nao e registrado em internal/runtime/runner.go:547. Esse e o comportamento que precisa continuar byte-identico.
O bloco de contexto atual e a concatenacao literal de prompt, cabecalho de secao, memoria de workflow, memoria de task e diretiva de compactacao, montada em internal/runtime/runner.go:490 ate internal/runtime/runner.go:520.

Evidencia de restricao:
internal/runtime/types.go:107 documenta que diretorio de tasks vazio desabilita o subsistema e preserva a regressao anterior. internal/runtime/types.go:123 documenta que a classe de janela e propagada internamente e nao deve ser definida pelo usuario. cmd/ai_spec_harness/cli_contract_test.go:81 e cmd/ai_spec_harness/cli_contract_test.go:189 comparam bidirecionalmente a arvore real de comandos e flags contra o schema declarado, o que torna qualquer superficie nova de CLI um contrato versionado.

## Alternativa mais simples rejeitada
Alternativa:
Duas funcoes de alto nivel no pacote de memoria, uma para montar o bloco de contexto e outra para persistir os fatos da sessao, chamadas diretamente pelo runner e pelo hook, sem porta nova e sem tipo agregador.

Motivo da rejeicao:
Perdeu por custo total, nao por estetica. Primeiro: funcao livre nao e mockavel na fronteira do consumidor, e a geracao de mock do repositorio parte de interface declarada, conforme mockery.yml:58 — o runner perderia isolamento de teste que hoje possui. Segundo: as regras estritas da skill de implementacao Go exigem que toda funcao seja metodo de tipo, logo a alternativa nao elimina o tipo, apenas o deixa sem contrato. Terceiro: existem dois clientes com necessidades opostas de granularidade, o runtime e a operacao humana via CLI, e um par de funcoes tenderia a crescer em parametros para servir os dois. Quarto: a ordem de precedencia entre invariantes precisa ser aplicada num unico lugar, e duas funcoes livres chamadas de dois lugares distintos nao garantem essa ordem.

## Padrao primario
Recomendar: Facade

Padrao complementar:
Nenhum. O primario fecha a solucao sozinho. As cinco politicas permanecem domain services stateless no idioma que o pacote ja usa em internal/runtime/memory/window_policy.go:37, sem exigir padrao adicional.

## Padroes rejeitados
- Decorator: Motivo objetivo da rejeicao — a selecao entre store simples e subsistema duravel acontece uma vez, na construcao, por chave de configuracao. Isso e substituicao de implementacao, nao empilhamento dinamico de responsabilidades opcionais. Nao existem wrappers combinaveis nem necessidade de composicao em runtime, que sao os sinais fortes do padrao.
- Adapter: Motivo objetivo da rejeicao — nao existe contrato externo incompativel a traduzir. O subsistema e novo e desenhado para caber na porta; o problema central e excesso de passos internos, que e sinal de exclusao explicito de Adapter.
- Proxy: Motivo objetivo da rejeicao — nao ha controle de acesso, carga tardia, cache, limite de taxa nem fronteira remota. A porta nao governa acesso a recurso caro; ela simplifica um subsistema.
- Strategy: Motivo objetivo da rejeicao — as politicas nao sao algoritmos equivalentes trocados em runtime por tipo. Elas sao deterministicas e parametrizadas por configuracao, com uma unica implementacao cada. Sem segunda variante plausivel, a abstracao seria vazia.
- State: Motivo objetivo da rejeicao — o Fato tem cinco estados e transicoes governadas, mas o comportamento nao varia por estado; o estado apenas condiciona elegibilidade. Uma tabela de transicoes explicita resolve com menos tipos, e a regra de economia da skill manda preferir tabela de dispatch a padrao formal.
- Command: Motivo objetivo da rejeicao — nao existe fila, desfazer nem reexecucao de acao. Os sete eventos de dominio sao registros de fato ocorrido, nao acoes transportaveis. O arquivamento e reversivel por decisao humana, o que nao e desfazer de comando.
- Memento: Motivo objetivo da rejeicao — nao ha captura e restauracao de estado interno de objeto. O backup de migracao e copia de arquivo antes de conversao, e o arquivamento e mudanca de estado de Fato, nao snapshot para restauracao futura.
- Observer: Motivo objetivo da rejeicao — o repositorio ja possui um distribuidor de eventos com pontos canonicos e fan-out sequencial em internal/runtime/hooks/dispatcher.go, que sera reutilizado. Introduzir Observer duplicaria mecanismo existente, o que a regra de economia proibe.

## Justificativa de economia
Custo evitado:
Hoje o consumidor tem quatro pontos de acoplamento com o subsistema, em internal/runtime/runner.go:470, internal/runtime/runner.go:481, internal/runtime/runner.go:482 e internal/runtime/hooks/memory_persist.go:65. Sem fronteira, cada capacidade nova do subsistema multiplica esses pontos. Com a fachada, o subsistema pode ganhar fatos, camadas, lease, sanitizacao e compactacao sem que nenhuma linha do consumidor mude.

Retorno esperado:
A conta de mudanca cai de um consumidor acoplado a cinco colaboradores para um consumidor acoplado a uma porta. O custo estrutural adicionado e um unico tipo com metodos, classificado como custo baixo no catalogo, contra a remocao de orquestracao espalhada em dois arquivos de camadas diferentes.

## Justificativa de eficiencia
Impacto em execucao:
Nenhum ganho de execucao e alegado. A fachada nao esta em caminho quente: e chamada duas vezes por sessao de agente, cuja duracao e dominada por chamada de modelo de linguagem. A indirecao adicionada e uma chamada de metodo por operacao, irrelevante nessa escala. Alegar performance aqui seria alegacao sem mecanismo.

Impacto em manutencao:
Reduz branching no consumidor: a decisao de injetar ou nao contexto, de registrar ou nao hook, e de qual limite aplicar deixa de ser condicional no runner e passa a ser resolucao interna do subsistema. Reduz dependencia concreta: o consumidor deixa de conhecer formato textual, nomes de arquivo e limites. Reduz churn: mudanca de politica nao toca o consumidor.

## Justificativa de robustez
Falhas mitigadas:
Primeira, persistencia fora de ordem: sem uma fronteira unica, um caminho de escrita poderia persistir antes de sanitizar, publicando segredo em arquivo versionado. A fachada e o unico ponto onde a ordem de precedencia de invariantes e aplicada. Segunda, compactacao ignorada: hoje a compactacao depende de o agente obedecer texto no prompt; a fachada executa compactacao antes de encerrar a operacao, independente do agente. Terceira, escrita parcial concorrente: a fachada e o unico lugar que adquire lock de camada e usa escrita atomica, o que impede que dois processos corrompam a mesma pagina. Quarta, regressao silenciosa: com porta unica, o teste de paridade byte-a-byte tem um unico ponto de entrada para exercitar.

Invariantes protegidos:
Nenhum Fato persistido contem segredo. Fato ativo so deixa o conjunto ativo por contradicao registrada, promocao marcada ou arquivamento explicito. No maximo um dono de bastao por linha de trabalho. Contexto injetado nunca excede o teto global de orcamento. Leitura e serializacao de pagina preservam integralmente conteudo de autoria humana. Fato efemero nunca alcanca a camada de projeto.

## Estrutura minima
Participantes:
Fachada do subsistema, que e o unico ponto de entrada do consumidor de runtime. Colaboradores internos: leitor e serializador de pagina com round-trip lossless, agregado de camada com lock e escrita atomica, politica de relevancia, politica de orcamento, politica de sanitizacao, politica de compactacao, politica de lease do bastao. Cliente de runtime, que ve apenas a porta estreita. Cliente de operacao humana, que acessa os colaboradores em granularidade total.

Responsabilidades:
A fachada resolve configuracao efetiva, ordena a chamada dos colaboradores, aplica a precedencia entre invariantes, traduz erro interno em erro de fronteira, e decide entre falha fechada na escrita e degradacao explicita na leitura. Os colaboradores nao se conhecem entre si e nao conhecem a fachada. O cliente de runtime nao conhece nenhum colaborador. O cliente de operacao humana nao passa pela fachada, porque precisa de granularidade que a porta estreita deliberadamente esconde.

## Fluxo
1. O cliente de runtime pede o contexto de memoria da sessao, informando escopo e classe de janela.
2. A fachada resolve a configuracao efetiva e, se a feature estiver desativada, delega ao comportamento anterior sem alterar nada.
3. A fachada pede ao leitor de pagina os Fatos ativos de cada camada, isolando pagina ilegivel em vez de abortar.
4. A fachada pede a politica de orcamento o teto global e as cotas por camada.
5. A fachada pede a politica de relevancia a ordenacao dos candidatos e corta no orcamento, registrando omissoes.
6. A fachada devolve o bloco de contexto e a lista de sinalizacoes de contradicao e omissao.
7. Ao final da sessao, o cliente de runtime entrega o sinal estruturado e a secao declarada.
8. A fachada pede a politica de sanitizacao a redacao de trechos sensiveis, recusando a escrita quando o trecho nao for isolavel.
9. A fachada adquire o lock da camada resolvida pela durabilidade, consolida os Fatos com idempotencia e marcacao de contradicao, e persiste com escrita atomica.
10. A fachada pede a politica de compactacao o conjunto a arquivar quando a camada excede limites, e libera o lock.
11. A fachada devolve o resumo da operacao para o cliente de runtime registrar como evidencia.

## Pseudocodigo canonico
```text
FACHADA MemoriaDuravel
    dependencias: leitorPagina, agregadoCamada, politicaRelevancia,
                  politicaOrcamento, politicaSanitizacao, politicaCompactacao,
                  politicaLease, configuracaoEfetiva

    OPERACAO montarContexto(escopo, classeJanela) -> blocoContexto, sinalizacoes
        SE configuracaoEfetiva.desativada ENTAO
            RETORNAR comportamento anterior inalterado
        camadas <- leitorPagina.lerCamadas(escopo)          # pagina ilegivel e isolada
        orcamento <- politicaOrcamento.resolver(classeJanela)
        candidatos <- politicaRelevancia.ordenar(camadas, escopo)
        selecao, omissoes <- cortarNoOrcamento(candidatos, orcamento)
        RETORNAR renderizar(selecao), sinalizacoes(selecao, omissoes)

    OPERACAO registrarSessao(sinalEstruturado, secaoDeclarada) -> resumo
        candidatos <- derivarFatos(sinalEstruturado, secaoDeclarada)
        PARA CADA fato EM candidatos
            fato, redacoes <- politicaSanitizacao.aplicar(fato)
            SE trecho sensivel nao isolavel ENTAO FALHAR FECHADO
        camada <- resolverCamadaPorDurabilidade(candidatos)
        COM agregadoCamada.lock(camada)
            agregadoCamada.consolidar(candidatos)           # idempotencia e contradicao
            aArquivar <- politicaCompactacao.decidir(camada)
            agregadoCamada.arquivar(aArquivar)
            agregadoCamada.persistirAtomico(camada)
        RETORNAR resumo(escritas, redacoes, compactacao, omissoes)

CLIENTE DE RUNTIME  -> conhece somente FACHADA
CLIENTE DE OPERACAO -> conhece os colaboradores diretamente, sem passar pela FACHADA
COLABORADORES       -> nao conhecem a FACHADA nem uns aos outros
```

## Mapeamento por paradigma
Paradigma alvo:
Go, orientacao a objetos leve com tipos estruturais, composicao sobre heranca, interfaces pequenas definidas no pacote consumidor, injecao por construtor e ausencia de estado global.

Adaptacao:
A fachada e um struct concreto com dependencias injetadas por construtor, satisfazendo uma interface pequena declarada no pacote consumidor. As cinco politicas nao viram interfaces: permanecem structs stateless com metodos, no idioma que o pacote ja adota em internal/runtime/memory/window_policy.go:37, porque cada uma tem implementacao unica e nenhuma segunda variante plausivel. Nada de heranca, nada de classe abstrata, nada de fabrica de fachada. O agregado de camada e o leitor de pagina recebem a abstracao de filesystem injetada, corrigindo o desvio observado em internal/runtime/memory/store.go:104 e permitindo teste unitario com filesystem falso. A ordem de execucao permanece explicita e rastreavel no corpo dos metodos da fachada, sem despacho implicito.

## Plano de implementacao ou refatoracao
Passos:
1. Introduzir escrita atomica reutilizavel, portando o padrao de internal/sdd/state.go:299 ate internal/sdd/state.go:321 para a abstracao de filesystem, com implementacao real e implementacao falsa.
2. Introduzir o leitor e serializador de pagina com round-trip lossless, e travar a invariante com teste dedicado antes de qualquer outro consumidor existir.
3. Introduzir os tipos de dominio e as cinco politicas stateless, cada uma com teste proprio, sem tocar o consumidor.
4. Introduzir o agregado de camada com lock por plataforma e escrita atomica, reutilizando o lock de internal/taskloop/orchestrator_lock_unix.go:17 e tratando explicitamente o lock orfao de internal/taskloop/orchestrator_lock_windows.go:15.
5. Introduzir a fachada, orquestrando os colaboradores na ordem do fluxo, com a porta estreita satisfazendo a interface declarada no consumidor.
6. Ligar a fachada ao consumidor por resolucao de configuracao, mantendo o caminho anterior intacto quando a feature esta desativada, e provar a equivalencia com teste de paridade byte-a-byte.
7. Regenerar os mocks derivados de interface, conforme mockery.yml:58.
8. Expor a superficie de operacao humana acessando os colaboradores diretamente, e atualizar o contrato declarado de CLI exigido por cmd/ai_spec_harness/cli_contract_test.go:81.

## Plano de testes
Teste positivo:
Com a feature ativada e memoria presente nas tres camadas, a fachada devolve bloco de contexto contendo Fatos das tres camadas, dentro do teto global, com cotas respeitadas e omissoes declaradas. Ao registrar a sessao, os Fatos aparecem consolidados na camada correta segundo a durabilidade, e o resumo reporta escritas, redacoes e compactacao.

Teste negativo:
Registro de Fato sem chave semantica falha fechado com erro tipado e nao grava nada. Registro cujo trecho sensivel nao e isolavel falha fechado e nao persiste conteudo algum do Fato. Promocao sem marcacao explicita e recusada. Reivindicacao de bastao detido por processo vivo dentro do prazo e recusada informando o dono. Serializacao que nao reproduz conteudo de autoria humana e recusada.

Teste de regressao:
Com a feature desativada, o bloco injetado no prompt e byte-identico ao produzido pela versao atual, para insumos identicos, incluindo o caso de diretorio de tasks vazio, o caso de somente memoria de workflow, o caso de workflow e task na ordem correta, e o caso de diretiva de compactacao anexada. A suite existente de memoria, hooks e runner permanece verde sem alteracao de expectativa.

Teste de falha:
Pagina corrompida e isolada e reportada, e a sessao prossegue com o restante da memoria. Compactacao que nao alcanca o limite por conteudo de autoria humana reporta a violacao sem reescrever o conteudo. Dois processos reais competindo pela mesma camada nao corrompem a pagina nem perdem Fato, exercitado com processos separados e nao apenas com goroutines. Lock orfao em plataforma sem liberacao automatica e detectado por prazo vencido e por ausencia do processo dono, e nunca sobrescrito em silencio.

## Criterios de aceite
- A fachada e o unico ponto de contato do consumidor de runtime com o subsistema, e nenhum colaborador e importado pelo consumidor.
- Com a feature desativada, o prompt final e byte-identico ao atual para insumos identicos, provado por teste dedicado.
- A ordem de precedencia entre invariantes e aplicada em um unico lugar, e existe teste que falha se a sanitizacao for executada depois da persistencia.
- Nenhuma politica virou interface sem segunda implementacao real.
- O leitor de pagina possui teste de round-trip que falha se conteudo de autoria humana for alterado.
- O agregado de camada usa a abstracao de filesystem injetada e escrita atomica, e possui teste com dois processos reais.
- Os mocks gerados foram regenerados e o gate de mocks passa.
- O contrato declarado de CLI foi atualizado e o gate bidirecional de contrato passa.

## Riscos e criterios de nao uso
Riscos:
A fachada pode virar objeto-deus se absorver regra de dominio em vez de apenas orquestrar; o criterio de contencao e que ela nao contenha decisao, apenas ordem de chamada e traducao de erro. A porta estreita pode esconder informacao que o consumidor legitimamente precisa, o que apareceria como parametros crescendo na assinatura; o criterio e que o resumo devolvido seja um valor estruturado, nao uma lista de argumentos de saida. Existe risco de duplicacao entre a fachada e a superficie de operacao humana, mitigado por ambas dependerem dos mesmos colaboradores e nenhuma reimplementar regra.

Nao usar quando:
Nao usar fachada se o consumidor passar a precisar de granularidade total, porque nesse caso ela deixa de simplificar e passa a atrapalhar; o sinal e o consumidor comecar a pedir operacoes internas uma a uma. Nao usar se o subsistema encolher para dois ou tres colaboradores sem ordem obrigatoria entre eles, porque ai a alternativa de funcao de alto nivel volta a vencer em custo total. Nao usar como porta unica se a operacao humana for forcada a passar por ela, porque isso obrigaria a fachada a expor tudo e a anular seu proprio ganho.
