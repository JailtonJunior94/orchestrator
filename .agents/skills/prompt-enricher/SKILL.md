---
name: prompt-enricher
description: Transforma um input bruto em um prompt pronto para LLM, otimizado para robustez, latência, economia de tokens e precisão de resposta. Extrai objetivo, restrições, contexto, critérios de aceite, proteções contra falhas e contrato de saída; depois reescreve o prompt com apenas o contexto mínimo necessário para execução confiável. Use quando precisar melhorar, endurecer, comprimir ou operacionalizar prompts para chatbots, agentes de código, fluxos automatizados ou chamadas de API. Não use para seleção de modelo, avaliação de outputs, datasets de fine-tuning ou escrita ampla de requisitos de produto.
---

# Enriquecedor de Prompt

## Procedimentos

**Etapa 1: Classificar o Input**
1. Extraia o input bruto do usuário literalmente antes de reescrever qualquer trecho.
2. Determine o modo principal do prompt:
   - `execution`: instrui um modelo ou agente a executar uma tarefa.
   - `transformation`: reescreve, resume, traduz, classifica ou extrai conteúdo.
   - `decision`: compara opções, diagnostica causas ou recomenda ações.
   - `generation`: cria código, texto, planos, assets ou artefatos estruturados.
3. Determine a pressão principal de otimização:
   - `robustness`: prioriza remoção de ambiguidade, casos de borda e proteções contra falha.
   - `performance`: prioriza menor latência, menos ramificações de raciocínio e resposta rápida.
   - `assertiveness`: prioriza saídas decididas, premissas explícitas e recomendações diretas.
   - `economy`: prioriza contexto menor, limites de saída mais rígidos e menor uso de tokens.
4. Se o usuário fornecer restrições do modelo alvo, preserve-as. Caso contrário, mantenha o prompt agnóstico ao modelo.

**Etapa 2: Extrair o Contexto Mínimo Viável**
1. Leia `references/context-rules.md` para identificar qual contexto manter, comprimir ou descartar.
2. Monte um conjunto de trabalho compacto com apenas os fatos necessários para execução correta:
   - objetivo
   - dados de entrada relevantes
   - restrições rígidas
   - formato de saída
   - critérios de aceite
3. Remova contexto complementar, exemplos repetidos, preferências especulativas e moldura narrativa, salvo quando alterarem a correção da tarefa.
4. Se faltar informação crítica, converta a lacuna em uma premissa explícita ou em um pedido de esclarecimento curto dentro do prompt enriquecido.

**Etapa 3: Aplicar as Regras de Enriquecimento**
1. Leia `references/enrichment-rules.md` e aplique apenas os blocos de regra compatíveis com a pressão de otimização definida na Etapa 1.
2. Adicione somente as seções que aumentem a confiabilidade de execução para o modo de prompt detectado.
3. Use `assets/prompt-template.md` como estrutura padrão.
4. Mantenha o prompt enriquecido compacto:
   - prefira instruções em vez de explicações
   - prefira limites explícitos em vez de orientações amplas
   - prefira esquemas, listas ou campos em vez de parágrafos longos
5. Adicione um bloco curto de "Não faça" apenas quando ele evitar falhas recorrentes.

**Etapa 4: Produzir o Prompt Final**
1. Monte o prompt final nesta ordem, salvo quando uma ordem menor for claramente melhor:
   - papel ou postura operacional
   - objetivo
   - entradas
   - restrições
   - dicas de processo
   - contrato de saída
   - tratamento de falhas ou premissas
2. Garanta que toda instrução seja acionável, testável e não contraditória.
3. Se o input de origem já for forte, aperte a redação em vez de reescrevê-lo por completo.
4. Se o usuário pedir economia máxima, colapse seções opcionais e preserve apenas:
   - objetivo
   - contexto essencial
   - restrições rígidas
   - contrato de saída

**Etapa 5: Validar o Resultado**
1. Salve o prompt enriquecido em um arquivo temporário quando a validação local for útil.
2. Execute `python3 .agents/skills/prompt-enricher/scripts/validate-prompt.py [prompt-file]` a partir da raiz do repositório para validar a qualidade estrutural.
3. Se o script reportar seções ausentes, risco de contradição, diretivas vagas ou pressão de verbosidade, revise o prompt e valide novamente.
4. Retorne:
   - o prompt enriquecido
   - uma justificativa curta com as principais otimizações aplicadas
   - premissas opcionais se o input de origem não trouxer contexto suficiente

## Tratamento de Erros
* Se o input de origem for apenas um tema ou um pedido em uma linha, produza um prompt mínimo com um bloco curto de premissas em vez de inventar fatos detalhados.
* Se o input de origem contiver instruções conflitantes, preserve a restrição de maior prioridade e exponha o conflito explicitamente no prompt enriquecido.
* Se o prompt enriquecido ficar maior que o texto de origem sem ganho de clareza, remova orientações de processo não essenciais e valide de novo.
* Se `.agents/skills/prompt-enricher/scripts/validate-prompt.py` falhar por arquivo inválido ou problema de codificação, execute novamente com um arquivo texto UTF-8 e preserve apenas ASCII quando fizer sentido.
