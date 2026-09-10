# Matriz de Selecao e Sinais Canonicos

Usar apenas sinais confirmados por evidencia. Cada sinal abaixo pode entrar no JSON de entrada do seletor.

## Sinais canonicos aceitos
- `single_product_variation`
- `family_of_related_products`
- `stepwise_construction`
- `clone_template`
- `single_process_shared_resource`
- `external_interface_mismatch`
- `dual_axis_variation`
- `recursive_tree_structure`
- `uniform_component_contract`
- `add_responsibilities_dynamically`
- `subsystem_too_complex`
- `high_memory_duplication`
- `access_control_or_lazy_loading`
- `sequential_conditional_handlers`
- `request_as_data`
- `custom_traversal`
- `dense_colleague_coordination`
- `snapshot_and_restore`
- `event_fanout`
- `state_transition_driven_behavior`
- `runtime_algorithm_swap`
- `fixed_workflow_with_variable_steps`
- `stable_structure_many_operations`
- `prefer_composition`
- `prefer_direct_solution`
- `low_change_frequency`
- `single_variant_only`
- `performance_hot_path`
- `memory_pressure`
- `strict_test_isolation`
- `multi_tenant_context`
- `remote_boundary`
- `undo_or_replay`
- `cross_product_consistency`
- `inheritance_already_natural`

## Restricoes canonicas aceitas
- `avoid_global_state`
- `avoid_inheritance`
- `minimize_class_count`
- `minimize_indirection`
- `preserve_public_contract`
- `tight_latency_budget`
- `tight_memory_budget`
- `high_change_frequency`
- `team_needs_low_cognitive_load`
- `must_support_runtime_switch`
- `must_support_undo`
- `must_support_broadcast`
- `must_support_checkpoints`
- `must_support_remote_access`

## Regras de desempate obrigatorias

### Strategy vs State
- Escolher `Strategy` quando o objetivo for trocar algoritmos equivalentes sem transicoes governadas por estado.
- Escolher `State` quando houver estados validos, transicoes, regras por estado e branching espalhado por status.
- Se ambos aparecerem e nao houver evidencia clara de transicao, parar como ambiguo.

### Decorator vs Proxy
- Escolher `Decorator` quando o ganho vier de empilhar responsabilidades opcionais.
- Escolher `Proxy` quando o ganho vier de governar acesso, lazy load, cache, seguranca ou fronteira remota.
- Se o wrapper so redireciona acesso a recurso caro, nao chamar de `Decorator`.

### Factory Method vs Abstract Factory
- Escolher `Factory Method` quando apenas um produto variar.
- Escolher `Abstract Factory` quando uma familia inteira de produtos precisar trocar em conjunto.
- Se nao houver familia consistente, recusar `Abstract Factory`.

### Strategy vs Template Method
- Escolher `Strategy` quando a variacao precisar ocorrer por composicao ou em runtime.
- Escolher `Template Method` quando a ordem do algoritmo for fixa e a heranca for aceitavel.
- Se `avoid_inheritance` estiver presente, favorecer `Strategy`.

### Bridge vs Strategy
- Escolher `Bridge` quando duas dimensoes variam de forma independente e combinatoria.
- Escolher `Strategy` quando apenas o algoritmo ou politica varia dentro de um mesmo contexto.
- Se apenas uma dimensao variar, recusar `Bridge`.

### Adapter vs Facade
- Escolher `Adapter` quando o problema central for incompatibilidade de interface.
- Escolher `Facade` quando o problema central for excesso de passos ou APIs ruidosas.
- Se ambos existirem, `Facade` pode usar `Adapter` internamente; recomendar um primario e um complementar apenas se ambos forem necessarios.

### Adapter vs Proxy
- Escolher `Adapter` quando o ganho vier de traduzir contrato entre interfaces.
- Escolher `Proxy` quando o ganho vier de governar acesso, latencia, cache, lazy load ou fronteira remota.
- Se a mesma evidencia suportar os dois e nao separar claramente traducao de governanca de acesso, parar como ambiguo.

### Facade vs Proxy
- Escolher `Facade` quando o problema central for simplificar um subsistema com muitos passos.
- Escolher `Proxy` quando o problema central for proteger ou controlar acesso a um recurso.
- Se a evidencia nao separar simplificacao de governanca de acesso, parar como ambiguo.

### Command vs Chain of Responsibility
- Escolher `Command` quando a acao precisar ser transportada, persistida, reexecutada, auditada ou desfeita.
- Escolher `Chain of Responsibility` quando a requisicao precisar passar sequencialmente por handlers independentes.
- Se nao houver fila, undo ou replay, nao chamar de `Command`.

### Observer vs Mediator
- Escolher `Observer` quando um emissor notificar varios assinantes de forma frouxamente acoplada.
- Escolher `Mediator` quando varios colegas precisarem de coordenacao central e regras de orquestracao.
- Se a ordem entre receptores for critica, `Observer` sozinho costuma ser fraco.

### Composite vs Visitor
- Escolher `Composite` quando o problema principal for representar e operar sobre uma arvore.
- Escolher `Visitor` apenas quando a arvore ja for estavel e varias operacoes novas surgirem fora dos tipos.
- `Visitor` nao substitui `Composite`; ele so entra se a estrutura recursiva ja existir.

### Composite vs Decorator
- Escolher `Composite` quando o foco for parte e todo em arvore.
- Escolher `Decorator` quando o foco for adicionar responsabilidade a um unico componente pela mesma interface.
- Se nao houver arvore, recusar `Composite`.

### Iterator vs Visitor
- Escolher `Iterator` quando o problema for percorrer elementos.
- Escolher `Visitor` quando o problema for adicionar operacoes sobre tipos estaveis.
- Percorrer uma arvore nao justifica `Visitor` por si so.

### Flyweight vs Singleton
- Escolher `Flyweight` quando o problema for cardinalidade alta e memoria.
- Escolher `Singleton` apenas quando o problema for unicidade controlada de recurso.
- Compartilhar instancia unica nao e `Flyweight`.

### Builder vs Abstract Factory
- Escolher `Builder` para montar um objeto complexo passo a passo.
- Escolher `Abstract Factory` para produzir familias coerentes de objetos.
- Se o problema for apenas construcao rica de um unico objeto, nao usar `Abstract Factory`.

### Memento vs Prototype
- Escolher `Memento` para snapshot e restauracao de estado.
- Escolher `Prototype` para clonagem de objetos configurados como ponto de partida.
- Se nao houver restauracao futura, `Memento` esta errado.

## Regras de recusa
- Se apenas `prefer_direct_solution`, `single_variant_only` e `low_change_frequency` estiverem presentes, retornar `reject`.
- Se nenhum sinal estrutural forte estiver presente, retornar `needs_more_evidence`.
- Se um pattern depender de heranca e `avoid_inheritance` estiver presente sem contrapeso forte, penalizar pesadamente.
- Se `Composite` nao tiver evidencia de `uniform_component_contract`, retornar `reject` ou `needs_more_evidence` em vez de presumir contrato comum.
