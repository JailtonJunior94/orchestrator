# Transcript da Decisao de Design Pattern

## Contexto Inicial
Decisao estrutural do Ciclo de Aprovacao definido em
.specs/prd-harness-quatro-clis-loop-aprovacao/prd.md (Bloco D) e modelado em
discoveries/domain-loop-de-aprovacao-e-catalogo-de-agentes-cli/domain-model.md. O fluxo tem quatro
estados com transicoes permitidas e proibidas explicitas, e um estado terminal so alcancavel sob duas
condicoes simultaneas. A pergunta e se isso justifica um pattern formal em Go.

## Evidencias Coletadas
Codigo: internal/taskloop/bugfix.go:83 (fluxo linear atual), :131 (saida antecipada que aceita veredito
nao-critico), :66 (teto fixo de 3); internal/runtime/runner.go:216-228 (segunda implementacao, one-shot);
internal/runtime/runner_autoreview.go:233 (veredito derivado de texto sintetico);
internal/runtime/specs/driver.go:14 (padrao ja adotado de Value Object de conjunto fechado com
construtor validante).

Comportamento: tarefa fecha como concluida com achados remanescentes; criterio declarado nao verificavel
nao bloqueia; a primeira remediacao aborta por limite de profundidade.

Restricao: regras [HARD] de go-implementation, com orientacao de preferir tipos concretos e nao
introduzir abstracao sem fronteira consumidora real.

## Rodada 1 - Diagnostico do Problema
Comportamento dirigido por estado implementado como sequencia condicional, sem tipo que carregue o
estado nem barreira que impeca a transicao ilegal. Duas implementacoes divergentes do mesmo fluxo.

## Rodada 2 - Alternativa Mais Simples
Alternativa avaliada: tabela de transicao declarativa mais Value Objects de conjunto fechado com
construtor validante, encapsulados no agregado. Ela entrega a mesma explicitude de transicao e a mesma
protecao de invariante, sem hierarquia e sem interface de estado.

## Rodada 3 - Selecao e Conflitos
Execucao 1 do seletor deterministico: `status = ok`, pattern primario **State**, score 4, com um unico
sinal forte (`state_transition_driven_behavior`) e um fraco (`preserve_public_contract`). O proprio
seletor apontou como alternativa simples "Usar tabela de transicao ou enum com regras localizadas".

Confronto com efficiency-and-cost-rules.md:
- Gate 1 falha: variante unica e sem pressao de mudanca por estado; a regra manda preferir tabela de
  dispatch antes de pattern formal.
- Gate 2 falha: em Go o State classico exigiria quatro tipos mais uma interface, adicionando mais tipos,
  indirecao e pontos de falha do que remove; nao ha duplicacao, variacao ou branching recorrente por estado.
- Gate 3 neutro: sem ganho de execucao alegavel; o ganho de manutencao e obtido igualmente pela tabela.
- Gate 4 neutro: a tabela protege as mesmas invariantes com rastreamento de execucao mais direto.

Conflito com a governanca da linguagem: nao existe consumidor que precise substituir o comportamento de
um estado, logo nao ha fronteira consumidora real que justifique a interface.

Execucao 2 do seletor, com o pattern reprovado declarado em `force_reject` conforme a Etapa 2 do
procedimento: `status = reject`, alternativa simples "Usar solucao direta, modulo pequeno ou refactor local".

## Rodada 4 - Implementacao e Testes
Implementacao registrada em implementation.md: tabela de transicoes como valor de pacote nao exportado,
sem funcao de inicializacao implicita; Value Objects com campo interno nao exportado e construtor
validante; politica imutavel injetada; portas declaradas no consumidor e mockadas pela configuracao
existente. Testes obrigatorios cobrem cada transicao proibida, as duas condicoes simultaneas do estado
terminal de aprovacao, os dois abortos antecipados e a distincao entre resultado terminal de negocio e
falha de infraestrutura.

## Decisoes Registradas
1. Decisao final: **nao aplicar padrao**. Adotar tabela de transicao declarativa mais Value Objects de
   conjunto fechado.
2. State rejeitado por Gates 1 e 2, apesar de ter vencido a pontuacao bruta do seletor.
3. Strategy, Template Method, Observer, Command e Memento rejeitados por ausencia de sinal forte
   correspondente, com motivo objetivo registrado em decision.md.
4. Gatilho de reavaliacao: quinto estado com comportamento nao trivial, transicoes dependentes de dados
   externos em tempo de execucao, ou necessidade de maquinas de estado aninhadas.
