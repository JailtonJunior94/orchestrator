# Prompt enriquecido — plano de melhorias das skills com foco production-ready / production-proof

## Prompt original

> Analisar as skills `create-prd`, `create-technical-specification`, `create-tasks`, `execute-all-tasks`, `execute-task`, `review`, `bugfix`.  
> Eu quero um score robusto, production-ready, production-proof com foco em eficiencia e sinergia das skills.  
> Eu quero que o PRD e a techspec confrontem o codebase e perguntem em formato de multiplas escolhas antes de tomar qualquer decisao.  
> Eu quero que `execute-task` e `execute-all-tasks` nao tenham falso positivo e realmente terminem a tarefa validando criterio de aceite e DoD.  
> Eu quero que `create-tasks` carregue as skills para uso de cada task, focando em eficiencia e economia sempre.  
> Eu quero que `review` e `bugfix` tenham sinergia e confrontem com o codebase e as tasks para garantir robustez no que foi desenvolvido.  
> Essa robustez, economia, eficiencia, production-ready e production-proof e inegociavel.  
> O output deve ser um plano de melhorias claro, sem falso positivo, para importar/instalar em outros projetos pequenos, medios e grandes, funcionando de forma igualitaria no Claude Code, Codex CLI e Copilot CLI.  
> Nao implemente nada. Apenas analise e produza o prompt/plano.

## Analise do prompt original

### Intencao principal

Produzir uma **auditoria profunda + plano de melhorias** para a cadeia de skills de planejamento, execucao, review e bugfix, com foco em:

- robustez real, nao cosmetica;
- eliminacao de falso positivo;
- economia de contexto/tokens/esforco operacional;
- sinergia entre skills;
- paridade entre Claude Code, Codex CLI e Copilot CLI;
- portabilidade para projetos pequenos, medios e grandes.

### Contexto ja definido

- O repositorio usa `AGENTS.md` como fonte canonica.
- Ha governanca PRD-first, spec-hash, evidencias obrigatorias e isolamento de contexto.
- As skills alvo ja existem e devem ser analisadas contra o codebase real do repositorio onde estiverem instaladas e em execucao, nao apenas deste repositorio.
- O resultado desejado e **um plano**, nao implementacao.
- O prompt deve ser **robusto para uso no GitHub Copilot CLI**, preferencialmente com o modelo **`claude-opus-4.8`**, minimizando alucinacoes e downgrade silencioso de rigor.

### Ambiguidades que precisavam ser fechadas

1. **"Score robusto"** nao define escala, pesos nem criterio de aprovacao.
2. **"Production-ready" e "production-proof"** precisavam virar criterios observaveis e auditaveis.
3. **"Confrontar com o codebase"** precisava explicitar profundidade minima de leitura e necessidade de citacoes.
4. **"Perguntar em multiplas escolhas antes de tomar qualquer decisao"** precisava virar protocolo operacional objetivo.
5. **"Sem falso positivo"** precisava ser traduzido em gates, evidencias, bloqueios e condicoes de conclusao.
6. **"Funcionar igualitariamente"** precisava virar matriz de paridade por ferramenta, e nao apenas desejo generico.

### Decisoes de enriquecimento aplicadas

1. Definir **escopo fechado**: apenas analise e plano, sem implementacao.
2. Exigir **score ponderado e rubricado**, por skill e global.
3. Exigir **citacoes de arquivos e linhas** para cada achado relevante.
4. Exigir **perguntas em multiplas escolhas** antes de qualquer recomendacao prescritiva ou decisao de desenho.
5. Exigir **matriz de gaps, riscos, sinergias, backlog priorizado e estrategia de rollout**.
6. Exigir **paridade cross-tool** e consideracao de projetos pequenos, medios e grandes.
7. Exigir **disciplina epistêmica**, separando fato verificado, inferencia e ponto inconclusivo.
8. Exigir **modo operacional compatível com Copilot CLI**, com leitura orientada por evidência e economia de contexto.

## Prompt enriquecido (recomendado)

```md
Voce vai atuar como auditor sênior de governanca operacional de IA para o repositorio onde estas skills estiverem instaladas.

## Ambiente de execucao alvo

- **Ferramenta alvo principal:** GitHub Copilot CLI
- **Modelo alvo obrigatorio quando disponivel:** `claude-opus-4.8`
- **Objetivo do modelo:** maximizar rigor analitico, criticidade, robustez e reduzir alucinacoes

Se `claude-opus-4.8` nao estiver disponivel:

1. declare explicitamente a limitacao;
2. nao faca downgrade silencioso de rigor;
3. mantenha o mesmo protocolo de evidência, incerteza e bloqueio;
4. nao simule confianca superior ao que a evidência suporta.

## Missao

Analise profundamente as skills:

- `create-prd`
- `create-technical-specification`
- `create-tasks`
- `execute-all-tasks`
- `execute-task`
- `review`
- `bugfix`

Seu objetivo e produzir **um plano de melhorias production-ready e production-proof**, sem implementar nada, com foco inegociavel em:

1. robustez real;
2. eliminacao de falso positivo;
3. eficiencia operacional;
4. economia de contexto/tokens/esforco;
5. sinergia entre skills;
6. paridade entre Claude Code, Codex CLI e Copilot CLI;
7. portabilidade para projetos pequenos, medios e grandes.

## Restricoes inegociaveis

1. **Nao implemente nada.**
2. **Nao altere arquivos.**
3. **Nao proponha respostas genericas sem confronto com o codebase.**
4. **Toda conclusao relevante deve citar arquivos e linhas do repositorio.**
5. **Nao assuma comportamento sem evidência local.**
6. **Nao marque algo como robusto, production-ready ou production-proof sem provar por criterio observavel.**
7. **Quando confrontar com o codebase, use sempre o codebase do repositorio onde a skill estiver instalada/executando. Nao use este repositorio como ancora universal, exceto quando ele for o proprio alvo analisado.**
8. **Nao use memoria, analogia ou padrao generico como substituto de leitura do codebase-alvo.**
9. **Nao degrade rigor por economia de tokens, pressa ou desejo de concluir.**
10. **Se faltar evidência para uma afirmacao estrutural, classifique como `Inconclusivo`.**

## Carga obrigatoria de contexto

Antes de analisar, leia e use como base:

1. `AGENTS.md`
2. `.agents/skills/agent-governance/SKILL.md`
3. `SKILL.md` de cada skill alvo
4. assets, templates, hooks, scripts, docs e validadores citados por essas skills quando forem relevantes para validar robustez, falso positivo, evidência, drift, sinergia e paridade

Considere como **codebase-alvo** o repositorio atual onde a skill estiver instalada. Toda analise de aderencia, viabilidade, decisao, paridade operacional e robustez deve ser confrontada com esse codebase-alvo.

## Modo operacional para Copilot CLI

Execute a analise como se estivesse no Copilot CLI, com estas regras:

1. priorize leitura orientada por hipótese, nao varredura cega do repositorio inteiro;
2. leia primeiro arquivos canonicos, `SKILL.md`, docs, scripts, templates, validadores e artefatos de handoff;
3. agrupe leituras relacionadas para economizar contexto e round-trips;
4. so amplie a exploracao quando a evidência atual nao bastar para confirmar ou refutar um ponto;
5. evite conclusoes baseadas em um unico arquivo quando o comportamento depender da cadeia entre skills;
6. prefira sempre evidência direta do codebase-alvo a analogias com outros repositorios;
7. mantenha a resposta final objetiva, auditavel e densa.

## Protocolo obrigatorio de decisao

Antes de tomar qualquer decisao prescritiva, recomendacao normativa, trade-off, priorizacao ou conclusao que nao seja puramente factual:

1. no Copilot CLI, use a ferramenta de pergunta ao usuario em **uma pergunta por vez**;
2. use **formato de multiplas escolhas**;
3. traga de **2 a 5 opcoes maximo**;
4. marque a opcao recomendada com **"(Recomendado)"**;
5. espere resposta antes de consolidar a decisao.

Se nao houver decisao aberta e o codebase ja fornecer evidência suficiente, siga sem inventar perguntas.

### Exemplo do formato esperado

**Pergunta:** Qual nivel de rigor voce quer para bloqueio de falso positivo em `execute-task`?
1. Gate estrito por evidencia + DoD + criterio de aceite + status sincronizado **(Recomendado)**
2. Gate por evidencia + criterio de aceite
3. Gate minimo por testes e report

## Protocolo anti-alucinacao e disciplina epistêmica

Classifique toda afirmacao importante em uma destas categorias:

- **Verificado:** sustentado por leitura direta do codebase-alvo, com citacao.
- **Inferido:** conclusao razoavel a partir de evidências parciais; declare a lacuna.
- **Inconclusivo:** faltam evidências para afirmar com seguranca.

Regras obrigatorias:

1. nao transforme `Inferido` em `Verificado`;
2. nao esconda lacunas de evidência;
3. nao use "provavelmente" para mascarar desconhecimento;
4. nao use tom de certeza quando a base for parcial;
5. se houver conflito entre arquivos, explicite o conflito e reduza a confianca;
6. diferencie claramente:
   - instrucao documental;
   - convencao esperada;
   - gate automatizado;
   - garantia efetiva.

## Limiar minimo de evidência

Antes de emitir score, classificacao ou recomendacao estrutural:

1. tenha evidência suficiente da skill analisada;
2. cite mais de um ponto do codebase quando o comportamento depender de fluxo entre artefatos;
3. nao use um unico exemplo feliz como prova de robustez;
4. procure sinais de enforcement, nao apenas texto instrutivo;
5. procure explicitamente caminhos de falso positivo, bypass, fallback permissivo e degradacao silenciosa.

## O que exatamente deve ser auditado

### 1. `create-prd`

Avalie se a skill:

- confronta suficientemente o pedido com o codebase-alvo e restricoes reais do repositorio onde a skill estiver instalada;
- evita drift silencioso downstream;
- transforma ambiguidade de produto em perguntas objetivas;
- deveria usar perguntas em multiplas escolhas antes de assumir escopo, restricoes, RFs, exclusoes e metricas;
- gera PRD suficientemente rastreavel e portatil para projetos de tamanhos diferentes.

### 2. `create-technical-specification`

Avalie se a skill:

- confronta PRD com arquitetura, modulos, interfaces, padroes, riscos e validacoes reais do codebase-alvo;
- pergunta em multiplas escolhas antes de fechar decisoes de arquitetura;
- reduz suposicoes perigosas;
- gera especificacao pronta para implementacao, com rastreabilidade, testes, ADRs e observabilidade adequados;
- previne techspec otimista, vaga ou desalinhada do repositorio.

### 3. `create-tasks`

Avalie se a skill:

- quebra o trabalho em fatias executaveis, auditaveis e eficientes;
- carrega/declara corretamente as skills necessarias por task;
- minimiza over-splitting e desperdicio operacional;
- reduz custo de contexto e evita tarefas ambíguas;
- sincroniza corretamente `tasks.md` com os `task-*.md`;
- cria tarefas que facilitem execucao deterministica em diferentes tools.

### 4. `execute-task`

Avalie se a skill:

- evita falso positivo de conclusao;
- so termina como `done` com evidencia fisica real;
- valida criterio de aceite, DoD, drift, sync, review e evidência;
- impede conclusao otimista com testes insuficientes, report vazio, status inconsistente ou review frouxa;
- tem fluxo robusto para remediacao, retry limitado, checkpoint e persistencia.

### 5. `execute-all-tasks`

Avalie se a skill:

- orquestra sem mascarar falhas;
- respeita dependencias, paralelismo seguro, halt-first, wait-all-then-halt e consistencia de `tasks.md`;
- evita falso positivo em lote;
- garante que cada task realmente fechou com criterio de aceite, DoD e evidência;
- e production-proof em cenarios com timeout, checkpoint, YAML ausente, drift, cross-PRD, race condition e degradacao cross-tool.

### 6. `review`

Avalie se a skill:

- confronta o diff com o codebase-alvo, com `prd.md`, `techspec.md` e tasks relevantes;
- captura bugs reais, regressao, seguranca, testes faltantes e lacunas de evidência;
- tem budget de revisao eficiente sem perder criticidade;
- entrega veredito deterministico e consumivel por `bugfix`;
- evita ruido, subjetividade e falsos negativos perigosos.

### 7. `bugfix`

Avalie se a skill:

- recebe achados do `review` de forma canônica e auditavel;
- corrige causa raiz e nao apenas sintoma;
- exige teste de regressao e evidência;
- respeita limites de tentativa, escopo e severidade;
- tem sinergia real com `review`, `execute-task` e contexto das tasks.

### 8. Sinergia entre as skills

Avalie o fluxo ponta a ponta:

`create-prd` -> `create-technical-specification` -> `create-tasks` -> `execute-all-tasks` / `execute-task` -> `review` -> `bugfix`

Identifique:

- gaps de handoff;
- duplicacoes;
- zonas cinzentas de responsabilidade;
- pontos de falso positivo;
- custos desnecessarios;
- riscos de drift;
- riscos de inconsistência cross-tool;
- oportunidades de padronizacao, economia e maior determinismo.

## Definicoes operacionais obrigatorias

Use estas definicoes, nao definicoes vagas:

- **Robustez:** resiste a erro de input, drift, ambiguidade, output malformado, estado parcial, corrida, contexto incompleto e degradacao de tool.
- **Production-ready:** suficientemente claro, validado e operacional para uso real repetivel.
- **Production-proof:** alem de pronto, possui barreiras concretas contra falso positivo, drift silencioso, evidencia fraca, handoff quebrado e divergencia entre tools.
- **Eficiência:** menor custo operacional para atingir qualidade alta.
- **Economia:** menor consumo de contexto, round-trips, retrabalho, paralelismo indevido e sobrecarga documental.
- **Sem falso positivo:** nada pode sair como concluido, aprovado ou robusto sem evidencia e gates coerentes com o risco.

## Rubrica obrigatoria de score

Atribua score **por skill** e **global**, usando escala **0 a 100**, com esta ponderacao:

| Dimensao | Peso |
|---|---:|
| Robustez / anti-falso-positivo | 25 |
| Validacao de criterio de aceite / DoD / evidências | 20 |
| Sinergia com outras skills | 15 |
| Eficiência operacional | 10 |
| Economia de contexto/esforco | 10 |
| Confronto com codebase real | 10 |
| Paridade Claude/Codex/Copilot | 10 |

### Interpretacao obrigatoria

- **90-100**: production-proof
- **75-89**: production-ready com gaps relevantes
- **50-74**: funcional, mas ainda arriscado
- **0-49**: insuficiente para operacao confiavel

Nao atribua nota sem explicar:

- por que recebeu aquela nota;
- qual evidencia sustenta a nota;
- o que impede nota maior.

## Formato obrigatorio da saida

Entregue a resposta em **Markdown** com esta estrutura:

1. `# Sumario Executivo`
2. `# Score Global`
3. `# Scorecard por Skill`
4. `# Achados por Skill`
5. `# Analise de Sinergia e Handoffs`
6. `# Principais Fontes de Falso Positivo`
7. `# Gaps de Paridade entre Claude Code, Codex CLI e Copilot CLI`
8. `# Plano de Melhorias Priorizado`
9. `# Sequencia Recomendada de Adocao`
10. `# Decisoes em Aberto`
11. `# Evidências e Citacoes`
12. `# Registro de Incertezas e Limites de Evidência`

## Requisitos minimos de cada secao

### `# Score Global`

Inclua:

- nota global;
- classificacao final;
- resumo dos 3 maiores bloqueadores;
- resumo dos 3 maiores pontos fortes.
- nivel de confianca global: `alto`, `medio` ou `baixo`.

### `# Scorecard por Skill`

Use uma tabela com colunas:

| Skill | Score | Classificacao | Falso positivo | Eficiencia | Sinergia | Paridade | Diagnostico curto |

### `# Achados por Skill`

Para cada skill, traga:

- o que esta forte;
- o que esta fragil;
- onde ha falso positivo potencial;
- onde falta confronto com codebase;
- onde faltam multiplas escolhas antes de decisoes;
- onde ha risco de alucinacao operacional por excesso de inferencia;
- o que precisa mudar para ficar production-ready / production-proof.

### `# Analise de Sinergia e Handoffs`

Explique:

- onde o fluxo quebra;
- onde responsabilidades se sobrepoem;
- onde a skill seguinte recebe contexto incompleto;
- onde o design atual gera retrabalho ou custo excessivo;
- onde ha oportunidades de reduzir custo e aumentar determinismo.

### `# Principais Fontes de Falso Positivo`

Liste pelo menos:

- status otimista;
- evidência insuficiente;
- validação incompleta;
- review permissiva demais;
- sincronizacao fraca entre artefatos;
- drift não bloqueado;
- paridade cross-tool inconsistente.

### `# Gaps de Paridade entre Claude Code, Codex CLI e Copilot CLI`

Para cada gap, informe:

- impacto;
- skill afetada;
- risco de divergencia operacional;
- proposta de mitigacao.

### `# Plano de Melhorias Priorizado`

Monte uma tabela com:

| Prioridade | Mudanca proposta | Skills afetadas | Beneficio | Risco mitigado | Impacto em eficiencia/economia | Esforco | Dependencias | Evidência usada |

Use prioridades:

- `P0` = obrigatorio antes de chamar de production-ready
- `P1` = importante para robustez sustentavel
- `P2` = melhoria relevante, mas nao bloqueante

### `# Sequencia Recomendada de Adocao`

Proponha uma ordem de execucao segura em fases:

1. endurecimento anti-falso-positivo;
2. sinergia e contratos entre skills;
3. paridade cross-tool;
4. economia/eficiencia fina;
5. validacao final de portabilidade.

### `# Decisoes em Aberto`

Se houver qualquer decisao de desenho, severidade, rigor, prioridade, UX, enforcement ou compatibilidade que nao seja puramente factual:

- nao assuma;
- apresente em formato de multiplas escolhas;
- uma pergunta por vez;
- com opcao recomendada.

### `# Evidências e Citacoes`

Para cada achado importante, cite:

- arquivo;
- linha ou faixa de linhas;
- relacao com o problema descrito.

### `# Registro de Incertezas e Limites de Evidência`

Liste de forma objetiva:

- pontos que ficaram inconclusivos;
- quais evidências faltaram;
- qual impacto isso tem no score;
- quais itens exigem confirmacao humana antes de qualquer mudanca real.

## Regras adicionais de qualidade

1. Nao confunda "ter instrucoes" com "ter enforcement real".
2. Diferencie claramente:
   - best-effort
   - validacao manual
   - gate programatico
   - garantia real
3. Sempre explicite quando algo depende de disciplina do agente em vez de enforcement tecnico.
4. Identifique qualquer degradacao silenciosa ou fallback permissivo.
5. Identifique se a robustez atual depende demais do agente "fazer a coisa certa".
6. Proponha melhorias que preservem economia e nao adicionem burocracia inutil.
7. Adapte a resposta para ser confiavel no Copilot CLI: objetiva, verificavel e sem inflar certeza.
8. Se mencionar paridade entre Claude Code, Codex CLI e Copilot CLI, mostre base concreta para a comparacao.

## Criterios de aceitacao da sua resposta

Sua resposta so sera considerada aceita se:

1. analisar explicitamente todas as 7 skills;
2. trouxer score global e score por skill com rubrica;
3. confrontar as conclusoes com o codebase real;
4. apontar onde faltam perguntas em multiplas escolhas;
5. apontar fontes concretas de falso positivo;
6. produzir backlog priorizado orientado a production-ready / production-proof;
7. cobrir portabilidade para projetos pequenos, medios e grandes;
8. cobrir paridade entre Claude Code, Codex CLI e Copilot CLI;
9. nao implementar nada;
10. citar evidências objetivas;
11. explicitar incertezas sem mascarar lacunas;
12. manter rigor compativel com uso no Copilot CLI com `claude-opus-4.8`.

## Nao faca

- nao escreva uma resposta genérica;
- nao reescreva as skills;
- nao invente features inexistentes;
- nao proponha mudancas sem justificar por evidência;
- nao chame de production-proof algo que ainda depende de disciplina manual fragil;
- nao use linguagem que simule certeza sem base verificavel;
- nao esconda limitacoes do modelo, do contexto ou da evidência;
- nao finalize sem separar claramente fatos, riscos, gaps e recomendacoes.
```

## Observacoes finais

- **Variante recomendada:** usar exatamente o prompt acima no **GitHub Copilot CLI** com **`claude-opus-4.8`**, sem reduzir o rigor.
- **Sem variante "mais leve"**: pelos requisitos do pedido, reduzir rigor aumentaria risco de falso positivo e descaracterizaria o objetivo.
