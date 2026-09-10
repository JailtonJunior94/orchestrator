# Registro de Decisão Arquitetural (ADR)

## Metadados

- **Título:** Ciclo de Aprovação como agregado de domínio isolado
- **Data:** 2026-09-10
- **Status:** Aceita
- **Decisores:** Solicitante do PRD (decisão registrada em sessão de modelagem)
- **Relacionados:** `prd.md` (RF-30 a RF-45), `techspec.md`, `discoveries/domain-loop-de-aprovacao-e-catalogo-de-agentes-cli/`, `pattern-decisions/ciclo-de-aprovacao-maquina-de-estados/`

## Contexto

O harness fecha tarefas afirmando aprovação sem prova. Três fatos verificados no código sustentam o
diagnóstico:

- O veredito de produção é derivado de **texto sintético**: a função que monta a saída da sessão de
  revisão devolve strings fixas derivadas do motivo de cancelamento, e é essa string — não a saída real
  do revisor — que alimenta o parser de veredito. Nenhuma delas contém marcador de bloqueio, logo o
  parser retorna "ok" sempre que a sessão não erra.
- Um veredito com ressalvas fecha a tarefa como concluída, e um critério de aceite declarado "não
  verificável pelo diff" é apenas registrado como risco.
- Existe um loop de remediação no repositório, mas ele só é alcançável por uma função **sem chamador de
  produção**. O caminho real executa revisão uma única vez.

Além disso, o guarda de profundidade de invocação tem limite dois, a etapa de execução já consome um, e
a correção falha de forma dura — de modo que nenhum ciclo passa da primeira rodada hoje.

Restrição dominante declarada para o modelo: correção **fail-closed**. Diante de dúvida, bloquear.

## Decisão

O Ciclo de Aprovação passa a ser um **agregado em pacote de domínio próprio**, sem dependência de
protocolo, CLI ou filesystem, e é a autoridade única sobre a regra de parada.

Escopo:

- Um comando único inicia o ciclo; as rodadas são decisão interna do agregado. Nenhum chamador externo
  abre rodada, o que é o que garante o teto.
- O estado terminal de aprovação é alcançável **exclusivamente** mediante um valor de prova cujo único
  construtor exige, simultaneamente, veredito aprovado e mapa de critérios completo. Aprovação sem prova
  torna-se estado inconstruível por tipo, não condição a verificar.
- Encerramento — inclusive não-convergente — é **resultado**, não erro. Erros ficam reservados a falha
  real de infraestrutura, com sentinelas comparáveis pelo mecanismo idiomático da linguagem.
- Tradução do texto do revisor para veredito é camada anticorrupção **fail-closed**: sem declaração
  canônica, o resultado é bloqueio, nunca aprovação por ausência de marcadores negativos.
- Cada rodada executa em sessão nova, com a profundidade de invocação resetada.

Impacto: os **três** caminhos de execução — o runtime de sessão, o serviço de execução de tarefas e o
loop de tarefas — passam a consumir o mesmo agregado.

## Alternativas Consideradas

**Manter a lógica onde está, endurecendo as condições.** Vantagem: nenhum pacote novo, refactor mínimo.
Desvantagem: mantém duas implementações do mesmo ciclo divergindo, e deixa a regra de negócio acoplada a
orquestração de tarefas e a protocolo. Rejeitada porque o custo de manter a paridade entre as duas
implementações é exatamente a dívida que a entrega quer eliminar.

**Implementar o ciclo apenas no runtime de sessão.** Vantagem: alinhado ao caminho mais moderno.
Desvantagem: exclui o caminho legado, tornando falsa a afirmação de comportamento idêntico entre
agentes. Rejeitada por tornar aspiracional um requisito declarado.

**Enforcement por prompt, descrevendo o loop nas skills.** Vantagem: zero código. Desvantagem: é
best-effort — limitação que o próprio fluxo de criação de PRD documenta sobre seus gates. Rejeitada:
aprovação de tarefa é forte demais para depender de o modelo lembrar de iterar.

## Consequências

### Benefícios Esperados

- A regra de aprovação passa a ser testável sem subir sessão nem tocar disco.
- Uma implementação única compartilhada pelos três caminhos, eliminando drift por construção.
- O falso positivo deixa de ser um defeito a caçar e passa a ser um estado que não compila.
- Motivo de parada canônico torna auditável **por que** um ciclo não convergiu — hoje o caso
  operacional mais difícil e sem nenhum insumo.

### Trade-offs e Custos

- Um pacote novo e três portas a implementar em dois adaptadores.
- Mais resultados devolvidos ao humano, com atrito operacional real.
- Sessão nova por rodada paga custo de contexto inicial repetido — preço deliberado do isolamento, que é
  o que elimina o vetor de aprovação alucinada por contexto acumulado.
- Retomada após interrupção fica fora de escopo: um ciclo interrompido recomeça.

### Riscos e Mitigações

**O mapa de critérios não existe como dado.** O artefato de revisão não tem seção para ele e o validador
não o cobra. Ligar o critério estrito sem isso converte falso positivo em **falso negativo total**.
Mitigação: seção no template, asserção no validador e propagação aos espelhos são **pré-requisito
bloqueante** da fase; a ordem de implementação carrega essa dependência dura de forma explícita.

**Critérios de aceite não chegam ao caminho de sessão.** A extração vive numa dependência do caminho
legado. Mitigação: o campo de nome do arquivo de tarefa já existe no job e é o gancho para o plumbing.

**Virada de comportamento pode mascarar regressão.** Mitigação: a migração passa por um estágio em que o
agregado reproduz o comportamento legado com a suíte existente intocada; só depois o critério estrito é
ligado, e cada teste alterado entra com justificativa por requisito.

**Rollback:** o critério de encerramento é parametrizado pela política injetada; reverter para o
comportamento anterior é uma mudança de política, não de estrutura.

## Plano de Implementação

1. Pacote de domínio com os Value Objects e seus construtores validantes, sem consumidor.
2. Camada de tradução migrada, com o atalho por substring solta removido.
3. Agregado, portas e mocks.
4. Estágio de paridade: o loop existente delega ao agregado preservando o comportamento legado, com a
   suíte atual verde e sem alteração — é o gate de não-regressão mais importante do plano.
5. Correção do veredito sintético e do endereçamento de evidência por rodada.
6. Cadeia de propagação do teto e ambiente por rodada.
7. Ponto de corte por rodada e revisão incremental.
8. Virada do critério de encerramento.
9. Ligação dos três caminhos e remoção do estágio de paridade.

Adoção concluída quando os três caminhos consomem o agregado, a suíte está verde e as diferenças contra
a linha de base são exatamente os testes previstos.

## Monitoramento e Validação

Métricas: rodadas por ciclo; distribuição de motivos de parada; taxa de aprovação na primeira rodada;
critérios reprovados por falta de evidência. Sucesso: nenhuma tarefa concluída sem prova, e todo
encerramento com motivo canônico registrado.

Revisar a decisão se a distribuição de motivos indicar que o critério estrito está bloqueando trabalho
legítimo em volume — sinal de critérios de aceite mal escritos ou de revisor mal calibrado, ambos
tratáveis sem afrouxar a invariante.

## Impacto em Documentação e Operação

Skills de execução, revisão, correção e governança; a seção do arquivo de governança que descreve a
revisão automática como comportamento de rodada única; o esquema da linha de comando; o guia de execução.

## Revisão Futura

Revisitar se surgir necessidade de retomada de ciclo após interrupção, se o conjunto de estados crescer
com comportamento próprio não trivial, ou se um quinto caminho de execução aparecer.
