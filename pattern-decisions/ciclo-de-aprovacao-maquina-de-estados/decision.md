# Pattern Decision Bundle

## Contexto
Problema:
O Ciclo de Aprovacao conduz rodadas de revisao e correcao entre quatro estados (EmRevisao, EmCorrecao,
Aprovado, Bloqueado), com transicoes permitidas e proibidas explicitas. O estado Aprovado so pode ser
alcancado com veredito APPROVED e mapa de criterios completo. A logica equivalente hoje e um fluxo
linear que sai cedo demais e nao modela estados.

Objetivo tecnico:
Tornar as transicoes proibidas irrepresentaveis e o comportamento por estado explicito, sem introduzir
mais estrutura do que o problema justifica.

Restricoes inegociaveis:
Preservar contrato publico dos consumidores existentes; minimizar indirecao; baixo custo cognitivo;
sem estado global; o pacote de dominio nao pode depender de ACP, CLI ou filesystem. Regras [HARD] de
`go-implementation` aplicam-se integralmente (R1 obriga metodos de struct; a orientacao geral e
preferir tipos concretos e nao introduzir abstracao sem demanda concreta).

## Diagnostico do problema
Sintomas observados:
Saida antecipada do ciclo assim que os achados criticos esvaziam, aceitando qualquer veredito
nao-critico; ausencia de contador de rodadas com semantica de dominio; ausencia de motivo de parada
canonico; impossibilidade de distinguir "nao aprovou" de "falhou".

Causa estrutural provavel:
Comportamento dirigido por estado implementado como sequencia condicional, sem tipo que carregue o
estado nem barreira que impeca a transicao ilegal.

## Evidencias
Evidencia de codigo (path:line ou greenfield):
internal/taskloop/bugfix.go:83 (loop atual), internal/taskloop/bugfix.go:131 (condicao de saida que
aceita veredito nao-critico), internal/taskloop/bugfix.go:66 (teto fixo de 3), internal/runtime/runner.go:216-228
(auto-review one-shot que nao conhece o loop), internal/runtime/runner_autoreview.go:233 (veredito
derivado de texto sintetico).

Evidencia de comportamento:
Uma tarefa fecha como concluida com achados conhecidos remanescentes; um criterio de aceite declarado
nao verificavel nao bloqueia; a primeira remediacao aborta por limite de profundidade
(.agents/lib/check-invocation-depth.sh:40 combinado com .agents/skills/bugfix/SKILL.md:20).

Evidencia de restricao:
internal/runtime/specs/driver.go:14 estabelece o padrao ja adotado no repositorio para Value Object de
conjunto fechado com construtor validante — precedente direto e barato de seguir.

## Alternativa mais simples rejeitada
Alternativa:
Nenhuma alternativa mais simples foi rejeitada. A alternativa simples **venceu** — ver "Padrao primario".
A alternativa que foi rejeitada e a formal: State classico com um tipo por estado e interface de estado.

Motivo da rejeicao:
Gate 1 e Gate 2 de `efficiency-and-cost-rules.md`. O State classico adicionaria quatro tipos concretos
mais uma interface para um fluxo unico com transicoes fixas e poucas, ou seja, adicionaria mais tipos,
indirecao e pontos de falha do que remove. O seletor pontuou 4 com apenas um sinal forte
(`state_transition_driven_behavior`) e um fraco (`preserve_public_contract`) — nao ha variacao
recorrente, nem pressao de mudanca por estado, nem duplicacao a eliminar. Some-se que a orientacao de
`go-implementation` e preferir tipos concretos e nao introduzir abstracao sem fronteira consumidora real.

## Padrao primario
Recomendar: nao aplicar padrao

Adotar a alternativa simples apontada pelo proprio seletor: **tabela de transicao declarativa mais enum
de estado com regras localizadas**, encapsulada no agregado. Concretamente: `Estado` e `MotivoDeParada`
sao Value Objects de conjunto fechado com construtor validante; a tabela de transicoes permitidas e um
dado imutavel do pacote; o agregado expoe um unico comando e aplica a tabela internamente. O ganho de
explicitude que motivaria o State e obtido pela tabela, sem hierarquia.

Padrao complementar: nenhum.

## Padroes rejeitados
- State: adiciona quatro tipos e uma interface para um fluxo unico com transicoes fixas; falha nos Gates 1 e 2.
- Strategy: nao ha familia de algoritmos intercambiaveis; a politica de parada e dado imutavel injetado, nao comportamento substituivel.
- Template Method: exigiria heranca ou embedding para um unico fluxo; a preferencia obrigatoria e composicao.
- Observer: os eventos de dominio sao emitidos para persistencia e telemetria por um unico consumidor conhecido; registro dinamico de observadores nao tem demanda.
- Command: `SubmeterParaAprovacao` nao precisa ser reificado como dado — nao ha undo, replay, fila nem serializacao de comando.
- Memento: nao ha requisito de snapshot e restauracao; retomada de ciclo esta explicitamente fora de escopo.

## Justificativa de economia
Custo evitado:
Quatro tipos de estado, uma interface e o custo de navegar entre eles em toda leitura futura do fluxo.
Evita tambem o custo recorrente de manter uma hierarquia sincronizada com um conjunto de estados que
nao tem pressao de crescimento.

Retorno esperado:
Uma unica implementacao compartilhada pelo runtime ACP e pelo orquestrador legado, no lugar de duas
divergentes. A regra de parada passa a ter um dono, e mudar o criterio de aprovacao passa a ser uma
mudanca localizada na tabela e na politica.

## Justificativa de eficiencia
Impacto em execucao:
Nenhum ganho alegado em hot path — nao ha mecanismo que o sustente e alegar seria falso. O ganho de
execucao real vem de outra decisao, independente de pattern: abortar por fingerprint repetida e por
ausencia de mudanca antes de gastar uma sessao de revisao.

Impacto em manutencao:
Remove branching condicional recorrente e substitui por dado declarativo. A transicao proibida deixa de
ser uma condicao que alguem precisa lembrar de escrever e passa a ser a ausencia de uma entrada na tabela.

## Justificativa de robustez
Falhas mitigadas:
Encerramento como aprovado sem veredito ou sem mapa completo; aprovacao inferida pela ausencia de
marcadores negativos; confusao entre resultado terminal de negocio e falha de infraestrutura, que hoje
faria o caminho de retry reprocessar uma reprovacao legitima.

Invariantes protegidos:
Ciclo aprovado sempre possui veredito APPROVED e mapa 1:1 completo; todo ciclo encerrado possui
exatamente um motivo canonico; o numero de rodadas nunca excede o teto; evidencia de rodada e imutavel.

## Estrutura minima
Participantes:
Agregado do ciclo (dono da iteracao e da regra de parada); Value Objects de conjunto fechado (Estado,
Veredito, MotivoDeParada); tabela de transicoes permitidas como dado imutavel; politica de aprovacao
imutavel injetada; portas minimas para revisar, corrigir e capturar mudanca, declaradas no consumidor.

Responsabilidades:
O agregado decide; as portas executam; a tabela declara o que e legal; a politica parametriza os limites.

## Fluxo
1. O ciclo abre em EmRevisao com a rodada 1 e a politica injetada.
2. A porta de revisao produz texto; a camada de traducao devolve veredito e achados, ou recusa.
3. Havendo veredito APPROVED e mapa completo, a transicao para Aprovado e consultada na tabela e aplicada.
4. Caso contrario, calcula-se a fingerprint; repeticao consecutiva transiciona para Bloqueado.
5. Havendo rodadas disponiveis, transiciona para EmCorrecao e a porta de correcao executa.
6. Ausencia de mudanca transiciona para Bloqueado; havendo mudanca, incrementa a rodada e volta ao passo 2.
7. Esgotado o teto, transiciona para Bloqueado com o motivo correspondente.

## Pseudocodigo canonico
```text
TABELA_DE_TRANSICOES := conjunto imutavel de (origem, destino) permitidos

funcao aplicar(ciclo, destino, motivo):
    se (ciclo.estado, destino) nao pertence a TABELA_DE_TRANSICOES:
        retornar erro de transicao ilegal
    se destino == Aprovado e (veredito != APPROVED ou mapa incompleto):
        retornar erro de invariante
    ciclo.estado := destino
    se destino e terminal: ciclo.motivo := motivo

funcao submeter(ciclo):
    enquanto verdadeiro:
        resultado := porta_revisao.revisar(alvo_da_rodada)
        veredito, achados := traduzir(resultado)          # fail-closed
        mapa := confrontar_criterios(achados, tarefa)
        se veredito == APPROVED e mapa.completo:
            aplicar(ciclo, Aprovado, "aprovado"); retornar
        fp := fingerprint(achados)
        se fp == ciclo.fingerprint_anterior:
            aplicar(ciclo, Bloqueado, "nao_convergiu"); retornar
        se ciclo.rodada >= politica.teto:
            aplicar(ciclo, Bloqueado, "limite_de_rodadas"); retornar
        aplicar(ciclo, EmCorrecao, "")
        mudou := porta_correcao.corrigir(achados)
        se nao mudou:
            aplicar(ciclo, Bloqueado, "sem_mudanca"); retornar
        ciclo.fingerprint_anterior := fp
        ciclo.rodada := ciclo.rodada + 1
        aplicar(ciclo, EmRevisao, "")
```

## Mapeamento por paradigma
Paradigma alvo:
Go 1.27, imperativo com tipos concretos, sem heranca, com metodos de struct obrigatorios por R1.

Adaptacao:
Estado, Veredito e MotivoDeParada como structs com campo interno nao exportado e construtor validante,
espelhando internal/runtime/specs/driver.go:14. Tabela de transicoes como valor de pacote nao exportado
com prefixo `_`, conforme R5.26, sem `init()` (R0). Politica como Value Object imutavel injetado no
construtor. Portas declaradas como interfaces no consumidor (R6), mockadas por mockery.yml (R3). Enums
com iota comecando em `iota+1` quando houver enum, mantendo o zero-value como "nao inicializado" (R5.8).
Erros terminais de negocio expressos como resultado, e falhas de infraestrutura como sentinelas
comparaveis por errors.Is (R5.10).

## Plano de implementacao ou refatoracao
Passos:
1. Criar os Value Objects de conjunto fechado com construtores validantes e testes de rejeicao.
2. Declarar a tabela de transicoes e a funcao de aplicacao, com teste por transicao proibida.
3. Implementar o agregado e o comando unico, com a politica injetada.
4. Declarar as portas minimas e gerar mocks.
5. Migrar o loop existente para consumir o agregado, preservando o comportamento observavel dos
   consumidores atuais.
6. Ligar o runtime ACP ao mesmo agregado.

## Plano de testes
Teste positivo:
Ciclo que aprova na rodada 1; ciclo que aprova na rodada 3 apos duas correcoes; motivo `aprovado`
registrado; evidencia de cada rodada em endereco proprio.

Teste negativo:
Cada transicao proibida rejeitada explicitamente; tentativa de alcancar Aprovado com veredito
APPROVED_WITH_REMARKS; tentativa de alcancar Aprovado com mapa incompleto; texto de revisor sem
veredito canonico resultando em bloqueio; politica com teto menor que 1 recusada na construcao.

Teste de regressao:
Fingerprint repetida em rodadas consecutivas encerra com `nao_convergiu` sem consumir o teto; correcao
sem mudanca encerra com `sem_mudanca`; teto esgotado encerra com `limite_de_rodadas` e nunca `done`;
cada rodada abre com profundidade de invocacao resetada; consumidores existentes preservam
comportamento observavel quando a capacidade nova esta desligada.

## Criterios de aceite
- O seletor deterministico foi executado e sua saida esta registrada em `selector-output.json`.
- A decisao registrada e `nao aplicar padrao`, com a alternativa simples adotada e justificada pelos Gates 1 e 2.
- Toda transicao proibida do modelo de dominio possui teste que a rejeita.
- Nenhum tipo novo foi criado apenas para representar estado.
- O pacote de dominio nao importa ACP, CLI nem filesystem.

## Riscos e criterios de nao uso
Riscos:
Se o conjunto de estados crescer e cada estado passar a ter comportamento proprio e volumoso, a tabela
deixa de ser suficiente e o State volta a ser candidato legitimo. O gatilho de reavaliacao e a chegada
de um quinto estado com comportamento nao trivial, ou a necessidade de variar o comportamento de um
mesmo estado por contexto.

Nao usar quando:
Nao aplicar a tabela de transicao se as transicoes passarem a depender de dados externos em tempo de
execucao, ou se for necessario compor maquinas de estado aninhadas — nesses casos a tabela plana
esconde a complexidade em vez de expo-la.
