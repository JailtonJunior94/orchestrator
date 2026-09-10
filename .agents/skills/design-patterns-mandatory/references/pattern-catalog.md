# Catalogo Operacional de Design Patterns

Este catalogo cobre os 22 patterns classicos indexados no catalogo publico do Refactoring.Guru para uso operacional. Cada entrada informa quando usar, quando recusar e um exemplo curto de estrutura.

## Criacionais

### Factory Method
- Intencao: delegar a criacao de um produto a uma especializacao sem expor concrete classes ao cliente.
- Sinais fortes: uma abstracao produz exatamente um tipo variavel; subclasses ou variantes escolhem o produto.
- Sinais de exclusao: familia completa de produtos; apenas um produto concreto; criacao trivial.
- Custo estrutural: moderado.
- Ganho tipico: reduz acoplamento com implementacoes concretas.
- Exemplo: `NotifierFactory -> createNotifier() -> EmailNotifier | SmsNotifier`.

### Abstract Factory
- Intencao: criar familias consistentes de produtos relacionados.
- Sinais fortes: varias partes do sistema precisam trocar familias inteiras mantendo compatibilidade entre produtos.
- Sinais de exclusao: um unico produto; uma unica familia; variacao rara.
- Custo estrutural: alto.
- Ganho tipico: consistencia entre familias e troca coordenada de variantes.
- Exemplo: `UiFactory -> createButton() + createDialog()` para tema web ou desktop.

### Builder
- Intencao: montar objeto complexo passo a passo sem construtor telescopico.
- Sinais fortes: muitas opcoes, ordem de montagem relevante, representacoes diferentes do mesmo objeto.
- Sinais de exclusao: poucos campos; named args resolvem; objeto simples.
- Custo estrutural: medio a alto.
- Ganho tipico: clareza de construcao e validacao gradual.
- Exemplo: `QueryBuilder.select().from().where().build()`.

### Prototype
- Intencao: clonar objetos existentes sem acoplamento ao tipo concreto.
- Sinais fortes: criacao cara; configuracao base reaproveitada; copia com pequenas variacoes.
- Sinais de exclusao: construcao barata; copia rasa insegura; identidade unica obrigatoria.
- Custo estrutural: medio.
- Ganho tipico: reduz custo de inicializacao repetitiva.
- Exemplo: `baseInvoiceTemplate.clone(customizations)`.

### Singleton
- Intencao: garantir unica instancia por contexto controlado.
- Sinais fortes: recurso unico de processo com inicializacao coordenada e governanca de concorrencia.
- Sinais de exclusao: conveniencia global; testes isolados; multitenancy; necessidade de varias instancias futuras.
- Custo estrutural: alto risco.
- Ganho tipico: coordenacao explicita de recurso realmente unico.
- Exemplo: `ProcessWideClockRegistry.instance()`.

## Estruturais

### Adapter
- Intencao: compatibilizar interface existente com a interface esperada pelo cliente.
- Sinais fortes: integracao com API externa ou legado com contrato incompatível.
- Sinais de exclusao: simplificacao de subsistema inteiro; duas dimensoes variaveis independentes.
- Custo estrutural: baixo a medio.
- Ganho tipico: isolamento de integracao e troca de dependencia.
- Exemplo: `PaymentGatewayAdapter` converte `charge(request)` para `authorize + capture`.

### Bridge
- Intencao: separar abstracao e implementacao para ambas variarem independentemente.
- Sinais fortes: duas dimensoes de variacao combinatoria e estavel.
- Sinais de exclusao: apenas uma dimensao varia; adaptacao unica; composicao simples basta.
- Custo estrutural: alto.
- Ganho tipico: evita explosao combinatoria de subclasses.
- Exemplo: `Report` usa `Renderer`; `PdfRenderer` e `HtmlRenderer`.

### Composite
- Intencao: tratar objetos individuais e arvores recursivas pela mesma interface.
- Sinais fortes: estrutura hierarquica real; operacoes recursivas; agregados e folhas com contrato comum.
- Sinais de exclusao: colecao plana; estrutura nao recursiva; visitor seria exagero.
- Custo estrutural: medio.
- Ganho tipico: uniformiza travessia e operacoes em arvore.
- Exemplo: `Folder` e `File` respondem a `size()`.

### Decorator
- Intencao: adicionar responsabilidade dinamicamente preservando a interface.
- Sinais fortes: wrappers empilhaveis; comportamento opcional; combinacoes dinamicas.
- Sinais de exclusao: controle de acesso, lazy load ou fronteira remota; uma unica variacao fixa.
- Custo estrutural: medio.
- Ganho tipico: extensao composicional sem subclasses explosivas.
- Exemplo: `RetryingClient(LoggingClient(BaseClient))`.

### Facade
- Intencao: expor uma interface simples para um subsistema complexo.
- Sinais fortes: muitos passos tecnicos, APIs ruidosas, onboarding caro.
- Sinais de exclusao: so adaptar contrato; precisa preservar granularidade total.
- Custo estrutural: baixo.
- Ganho tipico: simplifica uso e reduz acoplamento periferico.
- Exemplo: `VideoConversionFacade.convert(input, format)`.

### Flyweight
- Intencao: compartilhar estado intrinseco para reduzir memoria em muitos objetos semelhantes.
- Sinais fortes: altissima cardinalidade de objetos equivalentes; pressao real de memoria.
- Sinais de exclusao: poucos objetos; estado extrinseco confuso; custo de complexidade maior que a memoria salva.
- Custo estrutural: alto.
- Ganho tipico: economia de memoria e cache locality.
- Exemplo: `GlyphFactory` reaproveita formas de caracteres.

### Proxy
- Intencao: controlar acesso a um objeto mantendo a mesma interface.
- Sinais fortes: lazy load, cache, controle de acesso, rate limit, remote proxy.
- Sinais de exclusao: extensao funcional arbitraria; enriquecimento empilhavel.
- Custo estrutural: medio.
- Ganho tipico: governanca de acesso e protecao de recurso caro.
- Exemplo: `CachingImageProxy` carrega a imagem sob demanda.

## Comportamentais

### Chain of Responsibility
- Intencao: passar requisicoes por uma cadeia de handlers ate um deles tratar ou encerrar.
- Sinais fortes: pipeline sequencial de verificacao, fallback ou enriquecimento condicional.
- Sinais de exclusao: pedido encapsulado para fila ou undo; broadcast de eventos.
- Custo estrutural: medio.
- Ganho tipico: organiza etapas independentes e extensaveis.
- Exemplo: `AuthHandler -> RateLimitHandler -> BusinessHandler`.

### Command
- Intencao: encapsular uma acao como objeto ou payload executavel.
- Sinais fortes: fila, retry, undo, agendamento, auditoria ou desacoplamento entre emissor e executor.
- Sinais de exclusao: pipeline de handlers; callback simples sem ciclo de vida.
- Custo estrutural: medio.
- Ganho tipico: serializacao e rastreabilidade de acoes.
- Exemplo: `CreateInvoiceCommand` enviado para worker.

### Iterator
- Intencao: percorrer uma colecao sem expor sua representacao interna.
- Sinais fortes: varias estrategias de travessia ou estrutura interna nao trivial.
- Sinais de exclusao: loop trivial sobre lista publica; API da linguagem ja resolve.
- Custo estrutural: baixo a medio.
- Ganho tipico: isolamento da travessia.
- Exemplo: `TreeIterator.depthFirst()`.

### Mediator
- Intencao: centralizar coordenacao entre objetos que se conhecem demais.
- Sinais fortes: muitos colegas trocando sinais cruzados; orquestracao central reduz acoplamento.
- Sinais de exclusao: broadcast simples; workflow linear; event bus ja suficiente.
- Custo estrutural: alto.
- Ganho tipico: corta dependencias cruzadas e sequencia a colaboracao.
- Exemplo: `DialogMediator` coordena campos e botoes.

### Memento
- Intencao: capturar e restaurar estado sem expor detalhes internos.
- Sinais fortes: undo/redo, checkpoint, rollback local de estado.
- Sinais de exclusao: event log ja resolve; clone bruto basta; estado grande demais para snapshot.
- Custo estrutural: medio.
- Ganho tipico: recuperacao previsivel de estado.
- Exemplo: `Editor.save()` gera snapshot para `undo()`.

### Observer
- Intencao: notificar assinantes sobre mudancas sem acoplamento direto.
- Sinais fortes: um emissor, multiplos ouvintes, fanout dinamico.
- Sinais de exclusao: coordenacao central forte; ordem estrita entre receptores; acoplamento temporal sensivel.
- Custo estrutural: medio.
- Ganho tipico: extensibilidade por assinaturas.
- Exemplo: `OrderPaidEvent` notifica faturamento e analytics.

### State
- Intencao: variar comportamento conforme estado interno com transicoes explicitas.
- Sinais fortes: maquina de estados, branching por status, transicoes e regras por estado.
- Sinais de exclusao: apenas algoritmos intercambiaveis; quase nenhum estado; enum simples basta.
- Custo estrutural: medio.
- Ganho tipico: remove condicionais espalhadas por estado.
- Exemplo: `DraftOrder`, `PaidOrder`, `CancelledOrder`.

### Strategy
- Intencao: trocar algoritmos ou politicas preservando o mesmo contrato.
- Sinais fortes: multiplos algoritmos equivalentes, selecao em runtime, variacao independente do contexto.
- Sinais de exclusao: comportamento guiado por transicoes de estado; heranca fixa obrigatoria.
- Custo estrutural: baixo a medio.
- Ganho tipico: desacopla politica do contexto.
- Exemplo: `PricingStrategy` para promocao, atacado ou varejo.

### Template Method
- Intencao: fixar o esqueleto de um algoritmo e variar etapas por especializacao.
- Sinais fortes: workflow fixo com passos customizaveis e ordem invariavel.
- Sinais de exclusao: composicao resolve melhor; runtime swap necessario; heranca fragil.
- Custo estrutural: medio.
- Ganho tipico: compartilha fluxo e evita duplicacao.
- Exemplo: `ImportJob.run()` chama `read`, `validate`, `persist`.

### Visitor
- Intencao: adicionar novas operacoes sobre estrutura estavel sem alterar os tipos visitados.
- Sinais fortes: arvore de tipos estavel; novas operacoes surgem com frequencia.
- Sinais de exclusao: estrutura muda com frequencia; uma ou duas operacoes; double dispatch desnecessario.
- Custo estrutural: alto.
- Ganho tipico: separa operacoes complexas de uma estrutura estavel.
- Exemplo: `TaxVisitor` e `RenderingVisitor` sobre mesma AST.
