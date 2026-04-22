# Guia de uso das skills

> Referencia completa de contratos, prompts mandatorios e criterios de aceite para cada skill do pipeline de governanca. Leia este guia quando precisar reproduzir uma etapa com fidelidade maxima.
>
> Para uma visao geral e instalacao, consulte o [README](../README.md). Para guia especifico do task-loop, consulte [Referencia do task-loop](task-loop-reference.md).

Esta secao define o contrato de uso de cada skill. Seguir estas regras nao e opcional: desvios introduzem ambiguidade, retrabalho e artefatos incompativeis entre etapas.

### Principio central

Cada skill e uma etapa com entradas obrigatorias, saidas esperadas e criterios de aceite. Nao avance para a proxima etapa sem validar a saida da etapa anterior. O agente nao deve inferir o que falta — voce deve fornecer.

### Pipeline mandatorio

```text
analyze-project (se projeto existente)
        |
        v
   create-prd
        |
        v
create-technical-specification
        |
        v
   create-tasks
        |
        v
   execute-task  <-- repete por task
        |
        v
     review
        |
        v
    bugfix (para achados criticos)
        |
        v
    refactor (se necessario)
```

Regra absoluta: nunca execute `execute-task` sem `tasks.md` aprovado. Nunca execute `create-tasks` sem tech spec aprovada. Nunca execute `create-technical-specification` sem PRD aprovado.

---

### `analyze-project` — leitura arquitetural inicial

**Quando usar:** obrigatoriamente como primeira etapa em projetos existentes, antes de criar o PRD. Nao use em repositorios vazios — sem base de codigo real nao ha arquitetura para classificar.

**Entradas obrigatorias no prompt:**
- caminho do repositorio alvo (ou indicacao de que o agente esta no repositorio)
- tipo de resultado esperado (classificacao + evidencias + mapa de dependencias)
- restricoes de escopo se houver areas que nao devem ser analisadas

**Prompt mandatorio:**

```text
Use a skill analyze-project para analisar a arquitetura atual deste repositorio.

Contexto:
- repositorio existente com base de codigo real
- objetivo: entender a arquitetura antes de criar o PRD

Saidas esperadas obrigatorias:
- classificacao do tipo de projeto: monolito, monolito modular, monorepo ou microservico
- evidencias usadas na classificacao (arquivos, estrutura de pastas, configuracoes detectadas)
- stack detectada (linguagem, frameworks, banco de dados, ferramentas de build)
- padrao arquitetural predominante (ex: handler -> service -> repository, hexagonal, event-driven)
- mapa das pastas mais importantes com responsabilidade de cada uma
- fluxo de dependencias entre camadas ou modulos
- recomendacoes de governanca para este contexto especifico
```

**Criterios de aceite do artefato — nao avance sem estes:**
- [ ] classificacao do tipo de projeto com evidencias concretas (nao inferencias vagas)
- [ ] stack detectada confere com o que existe no repositorio
- [ ] mapa de dependencias entre camadas e legivel e correto
- [ ] recomendacoes de governanca sao especificas para a arquitetura encontrada (nao genericas)
- [ ] resultado registrado ou compartilhado antes de iniciar `create-prd`

---

### `create-prd` — escopo e requisitos

**Quando usar:** toda feature nova ou qualquer mudanca que altere comportamento observavel do sistema.

**Entradas obrigatorias no prompt:**
- problema que a feature resolve (nao a solucao)
- persona ou sistema afetado
- restricoes de escopo (o que esta fora)
- restricoes tecnicas conhecidas (performance, integracao, compliance)

**Prompt mandatorio:**

```text
Use a skill create-prd para definir [nome da feature].

Entradas:
- Problema: [descricao objetiva do problema]
- Persona afetada: [quem sofre o problema]
- Restricoes de escopo: [o que nao esta incluido]
- Restricoes tecnicas: [limites conhecidos]

Saidas esperadas obrigatorias:
- problema claro e verificavel
- objetivos mensuráveis
- nao objetivos explicitos
- requisitos funcionais numerados (RF-01, RF-02...)
- requisitos nao funcionais (RNF-01, RNF-02...)
- criterios de aceite por requisito funcional
- riscos com probabilidade e impacto
```

**Criterios de aceite do artefato — nao avance sem estes:**
- [ ] cada RF tem criterio de aceite testavel
- [ ] nao objetivos estao listados (previnem escopo rastejante)
- [ ] riscos tem mitigacao ou decisao explicita
- [ ] artefato salvo em `tasks/<prd-folder>/prd.md`

---

### `create-technical-specification` — arquitetura e contratos

**Quando usar:** apos PRD aprovado, antes de qualquer linha de codigo.

**Entradas obrigatorias no prompt:**
- caminho do `prd.md` aprovado
- contexto tecnico do repositorio (stack, arquitetura atual, fronteiras existentes)
- restricoes que o PRD impos
- referencias a carregar (DDD, arquitetura, concorrencia — conforme a feature)

**Prompt mandatorio:**

```text
Use a skill create-technical-specification com base no PRD aprovado em tasks/<prd-folder>/prd.md.

Contexto tecnico:
- Stack: [linguagem, frameworks, banco de dados]
- Arquitetura atual: [ex: handler -> service -> repository]
- Fronteiras existentes: [o que nao pode mudar]
- Referencias necessarias: [ddd, arquitetura, concorrencia, testes]

Saidas esperadas obrigatorias:
- modelagem de dominio (agregados, entidades, value objects se houver invariantes)
- contratos de interface (assinaturas, DTOs, erros tipados)
- responsabilidade de cada camada
- estrategia de erros com tipos e mensagens
- estrategia de testes (unitarios, integracao, table-driven)
- ADRs para decisoes nao obvias
- riscos tecnicos e plano de rollout
```

**Criterios de aceite do artefato — nao avance sem estes:**
- [ ] contratos de interface definidos (nao implicitos)
- [ ] estrategia de erros explicita (sem "retornar erro generico")
- [ ] estrategia de testes inclui cenarios de falha
- [ ] decisoes arquiteturais justificadas (ADR ou nota inline)
- [ ] artefato salvo em `tasks/<prd-folder>/techspec.md`

---

### `create-tasks` — decomposicao em unidades executaveis

**Quando usar:** apos tech spec aprovada, para gerar o bundle de tasks que o `execute-task` ou `task-loop` vai consumir.

**Entradas obrigatorias no prompt:**
- caminho do `prd.md` e `techspec.md` aprovados
- criterio de granularidade desejado (uma responsabilidade por task)

**Prompt mandatorio:**

```text
Use a skill create-tasks para decompor a tech spec aprovada em tasks.

Arquivos de entrada:
- PRD: tasks/<prd-folder>/prd.md
- Tech spec: tasks/<prd-folder>/techspec.md

Regras de decomposicao obrigatorias:
- uma responsabilidade por task (ex: nao misturar handler com repository na mesma task)
- ordem de execucao explicita (dependencias declaradas)
- criterio de pronto por task com validacoes executaveis
- nenhuma task com escopo ambiguo ou aberto

Saidas esperadas obrigatorias:
- tasks.md com lista ordenada e dependencias
- um arquivo por task (`task-1.0-desc.md`, `1.0-desc.md` ou `1-desc.md`) com: objetivo, arquivos afetados, criterio de pronto, validacoes
```

**Criterios de aceite do artefato — nao avance sem estes:**
- [ ] nenhuma task tem dependencia circular
- [ ] cada task tem criterio de pronto com comandos de validacao (ex: `go test ./...`)
- [ ] tamanho de cada task: implementavel em uma unica sessao de agente
- [ ] artefatos salvos em `tasks/<prd-folder>/`

---

### `execute-task` — implementacao com evidencia

**Quando usar:** para cada task do bundle, uma de cada vez. Nao execute tasks em paralelo sem garantir que nao ha dependencia entre elas.

**Entradas obrigatorias no prompt:**
- caminho exato da task a executar
- contexto arquitetural (nao confie que o agente inferira)
- referencias obrigatorias da linguagem e do dominio

**Prompt mandatorio:**

```text
Use a skill execute-task para implementar a task tasks/<prd-folder>/<N>_task.md.

Contexto obrigatorio:
- Leia o arquivo de task antes de iniciar qualquer alteracao
- Arquitetura: [descreva a camada e os contratos relevantes]
- Referencias a carregar: [go-implementation, ddd, tests — conforme a task]

Criterios de execucao nao negociaveis:
- preservar contratos publicos existentes (nenhuma assinatura publica muda sem ADR)
- nenhuma interface nova sem fronteira real justificada
- context.Context em todas as operacoes de IO
- testes table-driven para todos os cenarios do criterio de pronto
- registrar evidencia de conclusao no arquivo de task (output do teste, lint)
- nao fechar a task sem evidencia de validacao
```

**Criterios de aceite da execucao — nao marque a task como concluida sem estes:**
- [ ] todos os testes do criterio de pronto passam (`go test ./...`)
- [ ] lint sem erro (`go vet ./...` ou equivalente)
- [ ] nenhum contrato publico foi alterado sem justificativa registrada
- [ ] evidencia registrada no arquivo de task

**O que conta como evidencia valida:**

Evidencia e o output real dos comandos de validacao colado diretamente no arquivo de task. Nao e uma afirmacao como "testes passaram" — e o texto que prova isso.

Exemplos de evidencia valida:

```text
## Evidencia de conclusao

$ go test ./internal/payment/... -v
--- PASS: TestListPayments/status_filter (0.003s)
--- PASS: TestListPayments/pagination (0.002s)
--- PASS: TestListPayments/date_range (0.004s)
ok  	github.com/org/repo/internal/payment	0.012s

$ go vet ./...
(sem saida — nenhum erro encontrado)
```

O arquivo de task so pode ser marcado como `done` apos esse bloco estar registrado nele.

---

#### Como executar tasks com fidelidade maxima — ciclo por task com limpeza de contexto

Este e o padrao de execucao que produz o melhor resultado: uma task por sessao de agente, contexto limpo a cada inicio, fidelidade total ao artefato especificado.

#### Por que limpar o contexto entre tasks

O agente acumula "pressao de continuidade" ao longo de uma sessao: tende a manter decisoes anteriores, introduzir abstraocoes que emergiram na task passada e desviar do artefato atual. Limpar o contexto elimina esse risco. Cada task recebe um agente que so conhece o que voce forneceu naquela invocacao.

Regra: **uma sessao de agente por task. Ao terminar uma task, feche a sessao e abra uma nova.**

#### Fluxo de execucao por task

```text
[abrir nova sessao]
        |
        v
[fornecer contexto minimo obrigatorio: techspec + task file]
        |
        v
[executar execute-task com prompt mandatorio]
        |
        v
[validar criterio de pronto: testes + lint + evidencia]
        |
        v
[registrar evidencia no arquivo de task]
        |
        v
[fechar sessao]
        |
        v
[abrir nova sessao para a proxima task]
```

#### Contexto minimo obrigatorio por sessao

Cada nova sessao deve receber exatamente estes tres elementos — nem menos (risco de desvio), nem mais (risco de contaminacao):

1. o arquivo da task atual (`<N>_task.md`)
2. o trecho relevante da tech spec (`techspec.md`) — apenas as secoes que a task toca
3. a regra arquitetural da camada sendo implementada

Nao forneca tasks anteriores, outputs de sessoes passadas ou contexto de features paralelas. Isso contamina o contexto e aumenta a chance de o agente "lembrar" uma decisao que nao deve ser replicada.

#### Prompt mandatorio por task (template copiavel)

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/<prd-folder>/<N>_task.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio para esta task:
- Arquitetura: [camada e responsabilidade, ex: "repository — acesso a banco, sem regra de negocio"]
- Contratos relevantes da tech spec: [cole apenas as assinaturas e tipos que esta task usa]
- Invariantes que nao podem mudar: [contratos publicos, tipos de erro, comportamento esperado pelos testes]

Regras de execucao nao negociaveis:
- siga o criterio de pronto definido no arquivo de task — nao adicione nem remova escopo
- nenhuma interface nova sem fronteira real justificada na tech spec
- context.Context em todas as operacoes de IO
- testes table-driven cobrindo todos os cenarios do criterio de pronto
- ao finalizar: rode os testes, rode o lint, registre o output como evidencia no arquivo de task
- nao declare a task concluida sem evidencia registrada

Nao inferir. Se o arquivo de task tiver ambiguidade, pare e aponte antes de implementar.
```

#### Exemplo concreto: bundle de 3 tasks em sessoes separadas

Estrutura do bundle:

```text
tasks/prd-payments-list/
  prd.md
  techspec.md
  tasks.md
  task-1.0-repository.md   <- acesso ao banco, query com filtros
  task-2.0-service.md      <- regras de aplicacao, orquestracao
  task-3.0-handler.md      <- parse HTTP, validacao de input, resposta
```

**Sessao 1 — task 01 (repository):**

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/prd-payments-list/task-1.0-repository.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio:
- Arquitetura: camada repository — acesso a banco, sem regra de negocio, sem HTTP
- Contratos da tech spec:
    type PaymentRepository interface {
        List(ctx context.Context, filter PaymentFilter) ([]Payment, error)
    }
    type PaymentFilter struct { Status string; Page int; From, To time.Time }
- Invariantes: nenhuma logica de negocio nesta camada; erros devem usar os tipos definidos em techspec.md

Regras: siga o criterio de pronto da task. Testes table-driven para filtros. Registre evidencia ao finalizar.
```

[fechar sessao 1]

**Sessao 2 — task 02 (service):**

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/prd-payments-list/task-2.0-service.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio:
- Arquitetura: camada service — orquestracao e regras de aplicacao, sem acesso direto ao banco
- Contratos da tech spec:
    type PaymentService interface {
        List(ctx context.Context, filter PaymentFilter) ([]PaymentDTO, error)
    }
    O service recebe PaymentRepository por injecao; nao instancia dependencias.
- Invariantes: nao duplicar validacoes do handler; erros do repository devem ser propagados com contexto

Regras: siga o criterio de pronto da task. Testes com mock do repository. Registre evidencia ao finalizar.
```

[fechar sessao 2]

**Sessao 3 — task 03 (handler):**

```text
Nova sessao. Sem contexto anterior.

Tarefa: implementar tasks/prd-payments-list/task-3.0-handler.md

Leia o arquivo de task antes de qualquer acao.

Contexto obrigatorio:
- Arquitetura: camada handler — parse HTTP, validacao de input, mapeamento para DTO de resposta
- Contratos da tech spec:
    GET /payments?status=&page=&from=&to=
    200: { data: PaymentDTO[], total: int, page: int }
    400: { error: string } para input invalido
    500: { error: string } para erro interno
    O handler recebe PaymentService por injecao.
- Invariantes: nenhuma logica de negocio no handler; status HTTP mapeados conforme techspec.md

Regras: siga o criterio de pronto da task. Testes table-driven para cenarios de input valido, invalido e erro do service. Registre evidencia ao finalizar.
```

[fechar sessao 3]

#### O que fazer se o agente desviar durante a execucao

Se o agente iniciar uma task sem ler o arquivo, introduzir escopo nao previsto, criar abstraocao nao especificada ou pular a evidencia de validacao, interrompa imediatamente. Nao corrija na mesma sessao — feche, abra uma nova e reenvie o prompt mandatorio com a correcao explicita do desvio:

```text
Na sessao anterior houve um desvio: [descreva o desvio].
Nova sessao. Sem contexto anterior.

Tarefa: reimplementar tasks/<prd-folder>/<N>_task.md do zero.
[prompt mandatorio completo]

Correcao especifica: [o que nao deve ser feito desta vez]
```

#### Resumo das regras de ouro

| Regra | Motivo |
| --- | --- |
| uma sessao por task | elimina pressao de continuidade e desvio acumulado |
| contexto minimo — apenas o que a task precisa | contaminacao de contexto e a causa mais comum de desvio |
| leia o arquivo de task antes de qualquer acao | o agente deve seguir o artefato, nao inferir |
| nao declare concluida sem evidencia | evidencia e o unico sinal objetivo de que a task foi feita corretamente |
| desvio detectado: feche e recomece | correcao dentro da sessao desviada tende a gerar mais desvio |

---

### `review` — revisao antes de merge

**Quando usar:** obrigatoriamente antes de qualquer merge ou fechamento de ciclo de tasks. Nao e opcional.

**Entradas obrigatorias no prompt:**
- diff ou branch a revisar
- contexto do que foi implementado (skill usada, task referenciada)
- areas de risco conhecidas (performance, seguranca, contratos)

**Prompt mandatorio:**

```text
Use a skill review para revisar o diff atual.

Contexto da implementacao:
- Tasks executadas: [lista de tasks do bundle]
- Skill usada na implementacao: execute-task
- Areas de risco: [performance, seguranca, contratos, concorrencia]

Focos obrigatorios da revisao:
- corretude: a implementacao atende todos os RFs e criterios de aceite do PRD?
- regressao: alguma mudanca quebra contrato publico ou comportamento existente?
- seguranca: ha injecao de dependencia insegura, dado sensivel exposto ou validacao faltando?
- testes: todos os cenarios do criterio de pronto estao cobertos?
- dívida tecnica introduzida: o que precisara de refactor futuro?

Saidas esperadas:
- lista de achados por categoria (critico, importante, sugestao)
- para cada achado critico: arquivo, linha, descricao e correcao sugerida
- veredicto final: aprovado / aprovado com ressalvas / reprovado
```

**Criterios de aceite da revisao — nao avance sem estes:**
- [ ] nenhum achado critico em aberto
- [ ] achados importantes tem plano de correcao ou decisao explicita de aceitar o risco
- [ ] veredicto registrado

---

### `bugfix` — correcao de achados criticos apos review

**Quando usar:** quando a `review` retornar achados criticos ou importantes que precisam ser corrigidos antes do merge. Nao use para melhorias esteticas ou divida tecnica sem impacto — esses casos vao para `refactor`.

**Entradas obrigatorias no prompt:**
- lista de achados criticos da `review` com arquivo, linha e descricao
- comportamento esperado antes da correcao (o que o codigo deveria fazer)
- testes de regressao obrigatorios que devem ser adicionados ou corrigidos
- invariantes que nao podem mudar durante a correcao (contratos publicos, tipos de erro)

**Prompt mandatorio:**

```text
Nova sessao. Sem contexto anterior.

Use a skill bugfix para corrigir os achados criticos da revisao.

Achados a corrigir (da saida da skill review):
- [arquivo:linha] Achado 1: [descricao do problema]
- [arquivo:linha] Achado 2: [descricao do problema]

Comportamento esperado apos a correcao:
- [descricao do comportamento correto para cada achado]

Invariantes que nao podem mudar:
- contratos publicos: [assinaturas, endpoints ou tipos que nao podem ser alterados]
- tipos de erro: [erros que o restante do sistema depende]
- comportamento de outros fluxos nao afetados por estes achados

Regras de execucao nao negociaveis:
- identifique a causa raiz de cada achado antes de escrever qualquer linha de codigo
- adicione ou corrija testes de regressao que provem que o bug nao pode regredir
- nao altere comportamento fora do escopo dos achados listados
- ao finalizar: rode os testes, rode o lint e registre o output como evidencia
- nao declare o bugfix concluido sem evidencia de que a causa raiz foi eliminada

Saidas esperadas:
- diff com a correcao minima necessaria
- testes de regressao adicionados ou corrigidos
- evidencia de validacao (output de go test ./... e go vet ./...)
- descricao da causa raiz de cada achado corrigido
```

**Criterios de aceite do bugfix — nao feche o ciclo sem estes:**
- [ ] causa raiz identificada e documentada para cada achado
- [ ] testes de regressao adicionados que falhariam antes da correcao
- [ ] todos os testes existentes continuam passando (`go test ./...`)
- [ ] lint sem erro (`go vet ./...` ou equivalente)
- [ ] nenhum contrato publico alterado
- [ ] diff restrito ao escopo dos achados (sem correcoes oportunistas)
- [ ] segunda passagem de `review` no diff do bugfix se houver mais de dois achados criticos

---

### `refactor` — melhoria sem mudanca de comportamento

**Quando usar:** quando a `review` apontar divida tecnica relevante ou quando uma area do codigo estiver impedindo evolucao segura. Nao use como substituto para implementacao correta na primeira vez.

**Entradas obrigatorias no prompt:**
- achados da `review` que motivam o refactor
- escopo delimitado (nao refatore fora do escopo dos achados)
- invariantes que nao podem mudar (contratos publicos, comportamento observavel)

**Prompt mandatorio:**

```text
Use a skill refactor para corrigir os achados de refatoracao da revisao.

Achados que motivam este refactor:
- [lista dos achados da review com arquivo e linha]

Escopo obrigatorio:
- apenas os arquivos e funcoes apontados pelos achados
- nenhuma mudanca de comportamento observavel

Invariantes que nao podem mudar:
- contratos publicos: [lista de assinaturas e endpoints]
- comportamento de erro: [tipos de erro e mensagens esperados pelos testes]

Criterios de conclusao:
- todos os testes existentes continuam passando
- lint sem erro
- diff gerado e revisado antes de commitar
```

**Criterios de aceite do refactor — nao commite sem estes:**
- [ ] nenhum teste regressou
- [ ] nenhum contrato publico alterado
- [ ] diff menor que o esperado (sem escopo rastejante)
- [ ] segunda passagem de `review` no diff do refactor se o escopo foi grande

---

### Checklist de entrada por skill

Use esta tabela antes de invocar qualquer skill. Se um item obrigatorio estiver faltando, providencie antes de invocar.

| Skill | Obrigatorio antes de invocar | Artefato de saida esperado |
| --- | --- | --- |
| `analyze-project` | repositorio com base de codigo real | classificacao + evidencias + mapa de dependencias (registrado antes do PRD) |
| `create-prd` | problema definido, persona, restricoes de escopo | `tasks/<folder>/prd.md` |
| `create-technical-specification` | `prd.md` aprovado, contexto tecnico, referencias | `tasks/<folder>/techspec.md` |
| `create-tasks` | `prd.md` + `techspec.md` aprovados | `tasks/<folder>/tasks.md` + `<N>_task.md` |
| `execute-task` | task file com criterio de pronto, contexto arquitetural | evidencia no task file, testes passando |
| `review` | diff ou branch, contexto da implementacao, areas de risco | lista de achados com veredicto |
| `bugfix` | achados criticos da `review`, causa raiz suspeita, invariantes | causa raiz documentada, testes de regressao, testes passando |
| `refactor` | achados de `review`, escopo delimitado, invariantes | diff revisado, testes passando |

### Sinais de desvio — quando parar e corrigir

Se qualquer um dos seguintes acontecer, pare e corrija antes de continuar:

- analyze-project classificou o tipo de projeto sem evidencias concretas (ex: "parece um monolito")
- analyze-project nao detectou a stack real ou listou ferramentas que nao existem no repositorio
- recomendacoes de governanca sao genericas e poderiam se aplicar a qualquer projeto
- agente gerou codigo sem ler o arquivo de task
- tech spec nao tem estrategia de erros explicita
- task executada sem evidencia registrada
- review pulada por "a implementacao parece correta"
- bugfix executado sem identificar a causa raiz do achado
- bugfix sem testes de regressao que provem a correcao
- bugfix alterou escopo alem dos achados listados
- refactor alterou comportamento observavel
- PRD sem criterios de aceite testáveis
- tasks.md com tasks que misturam responsabilidades

Esses sao os desvios mais comuns e os que mais causam retrabalho.
