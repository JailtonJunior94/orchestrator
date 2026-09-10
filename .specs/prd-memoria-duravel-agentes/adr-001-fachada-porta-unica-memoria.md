# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Rótulo:** MD-001
- **Título:** Fachada como porta única do runtime para o subsistema de memória durável
- **Data:** 2026-09-10
- **Status:** Proposta
- **Decisores:** JailtonJunior94 (mantenedor do harness)
- **Relacionados:** [PRD](prd.md) RF-01 a RF-37 · [techspec](techspec.md) · [modelo de domínio](../../discoveries/domain-memoria-duravel-de-agentes/domain-model.md) · [bundle de padrões](../../pattern-decisions/memoria-duravel-fachada/decision.md) · MD-002, MD-003, MD-004, MD-005

## Contexto

O subsistema de memória atual expõe um contrato de quatro métodos (`memory.Store`) consumido em exatamente quatro pontos de produção: `internal/runtime/runner.go:470` instancia o store, `internal/runtime/runner.go:481` e `internal/runtime/runner.go:482` leem, e `internal/runtime/hooks/memory_persist.go:65` grava. Além desses quatro pontos, o consumidor também detém conhecimento que não é dele: `internal/runtime/runner.go:490` monta o formato exato do bloco de contexto injetado, e `internal/runtime/runner.go:516` anexa a diretiva textual de compactação.

A feature de memória durável introduz, atrás desse mesmo ponto de integração, um subsistema substancialmente maior: leitor e serializador de página com round-trip lossless, agregado de camada com lock e escrita atômica, e cinco políticas stateless (relevância, orçamento, sanitização, compactação, lease). O modelo de domínio define um workflow de dez passos com ordem obrigatória, e uma ordem de precedência entre invariantes que precisa ser aplicada em um único lugar — segredo, depois não perda de fato, depois dono único de bastão, depois orçamento.

Duas restrições delimitam a decisão. A primeira: o contrato atual não pode mudar de forma que quebre os quatro chamadores nem o mock gerado declarado em `mockery.yml:58`. A segunda: existem dois clientes com necessidades opostas de granularidade — o runtime, que precisa de uma porta estreita, e a operação humana via CLI, que precisa de granularidade total para promover, arquivar, buscar e compactar.

## Decisão

Introduzir uma fachada como único ponto de contato entre o Runtime de Sessão e o subsistema de memória durável, e **não** roteá-la para o cliente de operação humana.

A fachada é um struct concreto no pacote de memória durável, com todas as dependências injetadas por construtor, satisfazendo uma interface pequena declarada no pacote consumidor. Ela não contém decisão de domínio: contém ordem de chamada, aplicação da precedência entre invariantes, e tradução de erro na fronteira. Os colaboradores não conhecem a fachada nem uns aos outros.

Os subcomandos de operação humana acessam os colaboradores diretamente, sem passar pela fachada. Isso é deliberado: Facade não impede acesso direto ao subsistema, e forçar a operação humana pela porta estreita obrigaria a fachada a expor tudo, anulando o próprio ganho.

As cinco políticas **não** viram interfaces. Permanecem structs stateless com métodos, no idioma que o pacote já adota em `internal/runtime/memory/window_policy.go:37`, porque cada uma tem implementação única e nenhuma segunda variante plausível.

O escopo da decisão é a fronteira entre consumidor e subsistema. Ela não decide formato de arquivo (MD-002), nem mecanismo de concorrência (MD-003), nem mecanismo de ativação (MD-004), nem forma da evidência (MD-005).

## Alternativas Consideradas

**Duas funções de alto nível no pacote de memória**, chamadas diretamente pelo runner e pelo hook, sem porta nova e sem tipo agregador. Vantagens: menos um tipo, menos indireção, leitura linear. Desvantagens: função livre não é mockável na fronteira do consumidor, e a geração de mock do repositório parte de interface declarada (`mockery.yml:58`), então o runner perderia isolamento de teste que hoje possui; as regras estritas de implementação Go proíbem função livre, logo o tipo existiria de qualquer forma, apenas sem contrato; dois clientes com necessidades opostas de granularidade fariam as assinaturas crescer em parâmetros; e a precedência entre invariantes ficaria replicada nos dois pontos de chamada. Não escolhida por custo total pior — não por estética. Esta é a alternativa que quase venceu, e permanece registrada como caminho de rollback.

**Decorator sobre o store existente.** Vantagens: reuso literal do contrato atual, ativação por composição. Desvantagens: a seleção entre store simples e subsistema durável acontece uma única vez, na construção, por chave de configuração — isso é substituição de implementação, não empilhamento dinâmico de responsabilidades opcionais; não existem wrappers combináveis. Não escolhida porque os sinais fortes do padrão estão ausentes, e aplicá-lo seria falso positivo.

**Adapter.** Vantagens: nomearia bem a tradução entre modelo interno e formato de arquivo. Desvantagens: não existe contrato externo incompatível a traduzir — o subsistema é novo e nasce compatível com a porta; o problema central é excesso de passos internos, que é sinal de exclusão explícito de Adapter.

**Proxy.** Desvantagens: não há controle de acesso, carga tardia, cache, limite de taxa nem fronteira remota. A porta não governa acesso a recurso caro.

**Strategy para as políticas.** Desvantagens: as políticas não são algoritmos equivalentes trocados em runtime por tipo; são determinísticas e parametrizadas por configuração, com uma implementação cada. Sem segunda variante plausível, a abstração seria vazia.

**Nenhum padrão, orquestração no consumidor.** Desvantagens: é o estado atual levado ao extremo — o runner cresceria junto com o subsistema, e a ordem de invariantes ficaria distribuída entre runner e hook, exatamente a causa estrutural que a feature corrige.

O seletor determinístico da skill `design-patterns-mandatory` foi executado antes da recomendação e retornou `status: ok`, padrão primário `Facade`, score 4, sinal forte `subsystem_too_complex`, sem blockers e sem lacunas de evidência. A saída bruta está preservada em [`pattern-decisions/memoria-duravel-fachada/selector-output.json`](../../pattern-decisions/memoria-duravel-fachada/selector-output.json).

## Consequências

### Benefícios Esperados

- O consumidor passa de acoplado a cinco colaboradores para acoplado a uma porta. Capacidades novas do subsistema não tocam nenhuma linha do runner.
- A ordem de precedência entre invariantes tem um único ponto de aplicação, o que torna testável a garantia de que a sanitização precede a persistência.
- A compactação deixa de depender de o agente obedecer texto no prompt e passa a executar dentro da operação, antes de encerrar.
- O teste de paridade byte-a-byte exigido pelo PRD tem um único ponto de entrada para exercitar.
- O conhecimento de formato textual sai do consumidor e volta para o subsistema.

### Trade-offs e Custos

- Um tipo adicional, classificado como custo estrutural baixo no catálogo de padrões.
- Uma chamada de método a mais por operação. Nenhum ganho de execução é alegado: a fachada é chamada duas vezes por sessão de agente, cuja duração é dominada por chamada de modelo de linguagem. Alegar performance aqui seria alegação sem mecanismo.
- Duas superfícies coexistem — a porta estreita e o acesso direto dos subcomandos — e ambas precisam permanecer coerentes sem que nenhuma reimplemente regra.

### Riscos e Mitigações

- **Risco:** a fachada virar objeto-deus, absorvendo regra de domínio. **Impacto:** perda da fronteira que a decisão cria. **Mitigação:** critério explícito de contenção — a fachada não contém decisão, apenas ordem de chamada e tradução de erro; violações aparecem como lógica condicional de domínio dentro dela.
- **Risco:** a porta estreita esconder informação que o consumidor legitimamente precisa. **Impacto:** assinaturas crescendo em parâmetros de saída. **Mitigação:** o resultado devolvido é um valor estruturado, não uma lista de argumentos de saída.
- **Risco:** duplicação entre a fachada e a superfície de operação humana. **Mitigação:** ambas dependem dos mesmos colaboradores; nenhuma reimplementa regra.
- **Plano de rollback:** dissolver a fachada e voltar à alternativa de métodos de alto nível no agregado de camada, chamados diretamente pelo consumidor. A reversão é barata por construção, porque os colaboradores não conhecem a fachada — removê-la não exige tocar nenhum colaborador, apenas religar o consumidor.

## Plano de Implementação

1. Implementar os colaboradores antes da fachada, cada um com teste próprio, sem tocar o consumidor.
2. Implementar a fachada orquestrando os colaboradores na ordem do workflow do modelo de domínio.
3. Declarar a porta estreita no pacote consumidor, com `var _ Porta = (*fachada)(nil)` conforme a convenção sistemática do repositório.
4. Registrar a interface nova em `mockery.yml` e regenerar os mocks, satisfazendo a regra R3.
5. Ligar a fachada ao consumidor por resolução de configuração, preservando o caminho anterior intacto (MD-004).
6. Expor os subcomandos de operação humana acessando os colaboradores diretamente.

Dependências: MD-002 (formato e identidade) e MD-003 (concorrência) precedem a fachada, porque ela orquestra ambos. MD-004 é simultâneo ao passo 5.

Critério de adoção concluída: nenhum colaborador do subsistema é importado pelo pacote consumidor, e o gate de mocks passa.

## Monitoramento e Validação

- **Sinais:** número de colaboradores importados pelo pacote consumidor deve ser zero; número de pontos onde a precedência de invariantes é aplicada deve ser um.
- **Critério de sucesso:** uma capacidade nova do subsistema pode ser adicionada sem alterar nenhuma linha do consumidor.
- **Critério para revisar:** o consumidor começar a pedir operações internas uma a uma, provando que precisa de granularidade total; ou o subsistema encolher para dois ou três colaboradores sem ordem obrigatória entre eles.

## Impacto em Documentação e Operação

- `AGENTS.md` e `CLAUDE.md`: seção de memória precisa registrar a existência da porta e o fato de que a operação humana não passa por ela.
- `docs/task-loop-reference.md`: referência de flags e comportamento de memória.
- `mockery.yml`: interface nova declarada.
- `docs/cli-schema.json`: contrato dos subcomandos novos, exigido pelo gate bidirecional em `cmd/ai_spec_harness/cli_contract_test.go:81`.

## Revisão Futura

Revisar quando a memória nova deixar de ser opt-in e passar a ser default, porque nesse momento a porta estreita se torna o único caminho em produção e qualquer lacuna de informação nela deixa de ter escape. Revisar também se um terceiro cliente aparecer, porque duas superfícies são sustentáveis e três provavelmente não.
