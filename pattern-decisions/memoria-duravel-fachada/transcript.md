# Transcript da Decisao de Design Pattern

## Contexto Inicial

- Origem: PRD `.specs/prd-memoria-duravel-agentes/prd.md` (spec-version 3, 37 requisitos funcionais) e modelo de dominio `discoveries/domain-memoria-duravel-de-agentes/domain-model.md` (validado com SUCCESS).
- Objetivo tecnico: introduzir um subsistema de memoria duravel atras do ponto de integracao que o Runtime de Sessao ja usa, sem alterar o comportamento atual quando a feature esta desativada.
- Modo de contexto: codebase existente, nao greenfield. Go 1.27.1, sete dependencias diretas, 182 arquivos de teste.
- Restricao dominante herdada do PRD: zero regressao no caminho default, com paridade byte-a-byte como gate de merge.

## Evidencias Coletadas

Levantadas por investigacao dirigida do codigo de producao, com classificacao explicita. Nenhum achado foi elevado a confirmado com base em teste, mock, fixture ou documentacao.

| Achado | Evidencia | Consequencia para a decisao |
|---|---|---|
| Persistencia destrutiva a cada sessao | internal/runtime/hooks/memory_persist.go:65 | O unico ponto de escrita atual conhece o template do arquivo; falta fronteira |
| Escrita de tier de task sem chamador de producao | internal/runtime/memory/store.go:68 e internal/runtime/memory/store.go:188 | Contrato ja tem superficie morta; ampliar o contrato agrava |
| Leitura concentrada em dois pontos | internal/runtime/runner.go:481 e internal/runtime/runner.go:482 | O consumidor tem poucos pontos de acoplamento, e devem continuar poucos |
| Compactacao como diretiva textual | internal/runtime/runner.go:516 | A garantia depende do agente; precisa migrar para dentro do subsistema |
| Formato do bloco de contexto no consumidor | internal/runtime/runner.go:490 | Conhecimento de formato esta no lado errado da fronteira |
| Politica stateless como domain service ja e idioma do pacote | internal/runtime/memory/window_policy.go:37 | As cinco politicas novas nao precisam de padrao formal |
| Chamadas diretas de sistema operacional no store | internal/runtime/memory/store.go:104, :121, :153 | Desvio da abstracao injetada; impede teste unitario com filesystem falso |
| Escrita atomica existe, fora da abstracao | internal/sdd/state.go:299 ate internal/sdd/state.go:321 | Padrao disponivel para portar, nao para reinventar |
| Lock por plataforma reutilizavel | internal/taskloop/orchestrator_lock_unix.go:17 | Granularidade de lock ja existente casa com agregado por camada |
| Lock orfao sobrevive a crash em uma das plataformas | internal/taskloop/orchestrator_lock_windows.go:15 | Exige deteccao por prazo vencido e por ausencia do processo dono |
| Mock gerado a partir de interface declarada | mockery.yml:58 | Alternativa com funcao livre perderia isolamento de teste do consumidor |
| Contrato de linha de comando validado bidirecionalmente | cmd/ai_spec_harness/cli_contract_test.go:81 e :189 | Superficie de operacao humana e contrato versionado, nao detalhe |
| Erro de despacho do ponto de fim de sessao e descartado | internal/runtime/runner.go:428 | Falha de persistencia e silenciosa hoje; a fachada deve tornar explicita |
| Secao de metricas deve permanecer a ultima do relatorio | internal/runtime/persistence/report.go:104 | Restringe onde evidencia nova pode ser injetada |
| Conjunto de metricas tem mapa de campos extra generico | internal/runtime/events/metricset.go e internal/runtime/persistence/report.go:71 | Metricas de memoria entram sem tocar template |

## Rodada 1 - Diagnostico do Problema

Pergunta: onde exatamente esta a causa estrutural, na ausencia de abstracao de armazenamento ou na ausencia de fronteira entre consumidor e subsistema?

Analise: o armazenamento nao e o problema. O contrato atual de quatro metodos e adequado para o que o consumidor precisa. O problema e que a logica de memoria esta dividida entre o consumidor, que monta formato e decide limites, e um hook, que conhece template de arquivo. Nao existe lugar onde a ordem de invariantes possa ser aplicada uma unica vez.

Conclusao: a causa estrutural e ausencia de fronteira, nao ausencia de abstracao de dados. O problema pertence ao grupo de acoplamento e excesso de passos, nao ao grupo de criacao, nem de variacao de algoritmo, nem de estado.

Registro do que NAO esta provado: nao esta provado que exista pressao de memoria ou de cardinalidade; nao esta provado que exista necessidade de troca de algoritmo em runtime; nao esta provado que exista fronteira remota; nao esta provado que exista necessidade de desfazer ou reexecutar acao. Essa lista foi usada para impedir superprescricao.

## Rodada 2 - Alternativa Mais Simples

Pergunta: duas funcoes de alto nivel no pacote de memoria resolvem com custo total menor?

Analise: quase. A alternativa foi levada a serio porque o proprio seletor a nomeia como concorrente. Ela perde por quatro razoes concretas, nao por estetica. Primeira, funcao livre nao e mockavel na fronteira do consumidor e a geracao de mock do repositorio parte de interface declarada, conforme mockery.yml:58. Segunda, as regras estritas de implementacao Go proibem funcao livre, entao o tipo existiria de qualquer forma, apenas sem contrato. Terceira, existem dois clientes com necessidades opostas de granularidade e um par de funcoes cresceria em parametros para servir os dois. Quarta, a precedencia entre invariantes precisa ser aplicada num unico lugar.

Conclusao: alternativa simples registrada e rejeitada por custo total pior. O gate de alternativa simples da skill esta satisfeito sem cair em `nao aplicar padrao`.

## Rodada 3 - Selecao e Conflitos

Entrada do seletor: sinais `subsystem_too_complex` e `prefer_composition`; restricoes `preserve_public_contract`, `avoid_inheritance`, `minimize_indirection`, `avoid_global_state`, `team_needs_low_cognitive_load`. Nenhum pattern em recusa forcada.

Sinais deliberadamente NAO declarados, para evitar falso positivo: `add_responsibilities_dynamically` foi descartado porque a selecao entre implementacoes acontece uma unica vez na construcao, o que e substituicao e nao empilhamento; `state_transition_driven_behavior` foi descartado porque o estado do Fato condiciona elegibilidade e nao altera comportamento polimorfico; `runtime_algorithm_swap` foi descartado porque cada politica tem implementacao unica; `external_interface_mismatch` foi descartado porque o subsistema e novo e nasce compativel com a porta; `access_control_or_lazy_loading` foi descartado porque nao ha governanca de acesso a recurso caro; `event_fanout` foi descartado porque o distribuidor de eventos ja existe e sera reutilizado.

Saida do seletor: status ok, pattern primario Facade, score 4, sinal forte `subsystem_too_complex`, sinal fraco `preserve_public_contract`, sem blockers, sem lacunas de evidencia. Saida registrada sem reescrita manual em `selector-output.json`.

Validacao contra o catalogo: a intencao do pattern bate com o problema; o sinal forte de muitos passos tecnicos esta presente; os sinais de exclusao nao estao presentes, porque nao se trata de apenas adaptar contrato e porque o cliente de runtime nao precisa de granularidade total; o custo estrutural e classificado como baixo; o pattern pertence ao catalogo classico fechado e nao esta na lista de barra alta para overengineering.

Conflito examinado explicitamente: o sinal de exclusao "precisa preservar granularidade total" exigiria recusar Facade se todos os clientes precisassem de granularidade. Nao e o caso: o cliente de runtime nao precisa, e o cliente de operacao humana acessa os colaboradores diretamente, sem passar pela fachada. Fachada nao impede acesso direto ao subsistema, e essa e a leitura correta do catalogo.

Pattern complementar: nenhum. O primario fecha a solucao sozinho e as politicas permanecem domain services stateless no idioma existente do pacote.

## Rodada 4 - Implementacao e Testes

Mapeamento por paradigma: struct concreta com dependencias injetadas por construtor, satisfazendo interface pequena declarada no pacote consumidor. Politicas como structs stateless sem interface, por ausencia de segunda variante plausivel. Erros de dominio como valores sentinela combinados com envelopamento por contexto, para permitir distincao entre falha fechada e degradacao. Nenhuma heranca, nenhuma classe abstrata, nenhuma fabrica de fachada.

Correcao de desvio aproveitada: o agregado de camada e o leitor de pagina recebem a abstracao de filesystem por construtor, corrigindo o desvio de internal/runtime/memory/store.go:104 e habilitando teste unitario com filesystem falso, conforme a decisao registrada no ADR de filesystem falso.

Plano de testes: cobre positivo, negativo, regressao e falha operacional. A cobertura de regressao inclui paridade byte-a-byte do bloco injetado com a feature desativada. A cobertura de falha inclui dois processos reais competindo pela mesma camada, porque o unico molde concorrente existente no repositorio usa goroutines no mesmo processo e nao prova exclusao inter-processo.

## Decisoes Registradas

| # | Decisao | Justificativa |
|---|---|---|
| P-01 | Pattern primario Facade | Sinal forte de subsistema com muitos passos, sinais de exclusao ausentes, custo estrutural baixo, fora da lista de barra alta |
| P-02 | Nenhum pattern complementar | O primario fecha a solucao; complementar duplicaria responsabilidade |
| P-03 | Politicas como structs stateless, sem interface | Implementacao unica por politica; abstracao sem segunda variante seria vazia |
| P-04 | Contrato atual de quatro metodos preservado | Chamadores de producao e mock gerado dependem dele |
| P-05 | Cliente de operacao humana nao passa pela fachada | Precisa de granularidade total; forcar a fachada anularia o ganho |
| P-06 | Abstracao de filesystem injetada nos colaboradores | Corrige desvio existente e habilita teste unitario com filesystem falso |
| P-07 | Escrita atomica portada do padrao existente | Reuso em vez de reinvencao; evita arquivo parcial visivel |
| P-08 | Lock por camada reutilizando o lock por plataforma existente | Granularidade casa com o agregado; trata lock orfao explicitamente |
| P-09 | Metricas de memoria pelo mapa de campos extra | Renderizacao do relatorio ja itera campos genericamente |
| P-10 | Falha de persistencia deixa de ser silenciosa | Erro de despacho e descartado hoje, o que contraria a proibicao de degradacao silenciosa |
